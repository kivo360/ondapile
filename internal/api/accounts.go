package api

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"ondapile/internal/adapter"
	"ondapile/internal/config"
	"ondapile/internal/model"
	"ondapile/internal/store"

	"github.com/skip2/go-qrcode"
)

type AccountHandler struct {
	store         *store.Store
	accounts      *store.AccountStore
	encryptionKey []byte
}

func NewAccountHandler(s *store.Store, encryptionKey []byte) *AccountHandler {
	return &AccountHandler{
		store:         s,
		accounts:      store.NewAccountStore(s),
		encryptionKey: encryptionKey,
	}
}

type connectAccountRequest struct {
	Provider    string             `json:"provider" binding:"required"`
	Identifier  string             `json:"identifier" binding:"required"`
	Name        string             `json:"name"`
	Credentials map[string]string  `json:"credentials"`
	Proxy       *model.ProxyConfig `json:"proxy,omitempty"`
}

// POST /accounts
func (h *AccountHandler) Create(c *gin.Context) {
	var req connectAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Validation(c, err.Error())
		return
	}

	if req.Name == "" {
		req.Name = req.Identifier
	}

	// Check for duplicate
	existing, _ := h.accounts.GetByProviderIdentifier(c.Request.Context(), req.Provider, req.Identifier)
	if existing != nil {
		Conflict(c, "Account already connected for this provider and identifier")
		return
	}

	// Create account record
	caps := getProviderCapabilities(req.Provider)
	orgID := c.GetString("organization_id")
	account, err := h.accounts.Create(c.Request.Context(), store.CreateAccountParams{
		Provider:       req.Provider,
		Name:           req.Name,
		Identifier:     req.Identifier,
		Status:         string(model.StatusConnecting),
		Capabilities:   caps,
		Proxy:          req.Proxy,
		Metadata:       map[string]any{},
		OrganizationID: orgID,
	})
	if err != nil {
		Internal(c, "Failed to create account")
		return
	}

	// Connect to provider
	prov, err := adapter.Get(req.Provider)
	if err != nil {
		Internal(c, "Provider not available: "+req.Provider)
		return
	}

	connected, err := prov.Connect(c.Request.Context(), account.ID, req.Credentials)
	if err != nil {
		h.accounts.UpdateStatus(c.Request.Context(), account.ID, model.StatusInterrupted, strPtr(err.Error()))
		ProviderError(c, "Failed to connect to provider: "+err.Error())
		return
	}

	// Persist encrypted credentials for reconnection
	if len(req.Credentials) > 0 && len(h.encryptionKey) > 0 {
		encCreds, encErr := config.EncryptCredentials(req.Credentials, h.encryptionKey)
		if encErr != nil {
			slog.Warn("failed to encrypt credentials", "error", encErr)
		} else {
			if storeErr := h.accounts.UpdateCredentials(c.Request.Context(), account.ID, encCreds); storeErr != nil {
				slog.Warn("failed to store encrypted credentials", "error", storeErr)
			}
		}
	}

	c.JSON(http.StatusCreated, connected)
}

// GET /accounts
func (h *AccountHandler) List(c *gin.Context) {
	// Check for organization_id in context (set by ApiKeyMiddleware)
	orgID := c.GetString("organization_id")
	if orgID != "" {
		// Use organization-scoped query
		accounts, err := h.accounts.ListByOrganization(c.Request.Context(), orgID)
		if err != nil {
			Internal(c, "Failed to list accounts")
			return
		}

		// Convert to []any for paginated list
		items := make([]any, len(accounts))
		for i, a := range accounts {
			items[i] = a
		}

		c.JSON(http.StatusOK, model.NewPaginatedList(items, "", false))
		return
	}

	// Fall back to existing behavior for backward compatibility (AuthMiddleware path)
	f := GetProviderFilter(c)
	p := GetPagination(c)

	accounts, nextCursor, hasMore, err := h.accounts.List(c.Request.Context(), f.Provider, f.Status, p.Cursor, p.Limit)
	if err != nil {
		Internal(c, "Failed to list accounts")
		return
	}

	// Convert to []any for paginated list
	items := make([]any, len(accounts))
	for i, a := range accounts {
		items[i] = a
	}

	c.JSON(http.StatusOK, model.NewPaginatedList(items, nextCursor, hasMore))
}

func (h *AccountHandler) getAccount(c *gin.Context) (*model.Account, bool) {
	id := c.Param("id")
	orgID := c.GetString("organization_id")
	var account *model.Account
	var err error
	if orgID != "" {
		account, err = h.accounts.GetByIDAndOrg(c.Request.Context(), id, orgID)
	} else {
		account, err = h.accounts.GetByID(c.Request.Context(), id)
	}
	if err != nil || account == nil {
		NotFound(c, "Account not found")
		return nil, false
	}
	return account, true
}

// GET /accounts/:id
func (h *AccountHandler) Get(c *gin.Context) {
	account, ok := h.getAccount(c)
	if !ok {
		return
	}

	// Get live status from provider if available
	prov, err := adapter.Get(account.Provider)
	if err == nil {
		status, _ := prov.Status(c.Request.Context(), account.ID)
		if status != "" {
			account.Status = status
		}
	}

	c.JSON(http.StatusOK, account)
}

// DELETE /accounts/:id
func (h *AccountHandler) Delete(c *gin.Context) {
	account, ok := h.getAccount(c)
	if !ok {
		return
	}

	prov, err := adapter.Get(account.Provider)
	if err == nil {
		_ = prov.Disconnect(c.Request.Context(), account.ID)
	}

	orgID := c.GetString("organization_id")
	if orgID != "" {
		if err := h.accounts.DeleteByIDAndOrg(c.Request.Context(), account.ID, orgID); err != nil {
			Internal(c, "Failed to delete account")
			return
		}
	} else {
		if err := h.accounts.Delete(c.Request.Context(), account.ID); err != nil {
			Internal(c, "Failed to delete account")
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"object": "account", "id": account.ID, "deleted": true})
}

// POST /accounts/:id/reconnect
func (h *AccountHandler) Reconnect(c *gin.Context) {
	account, ok := h.getAccount(c)
	if !ok {
		return
	}

	prov, err := adapter.Get(account.Provider)
	if err != nil {
		ProviderError(c, "Provider not available")
		return
	}
	// Try to load persisted credentials for reconnection
	encCreds, credErr := h.accounts.GetCredentialsEnc(c.Request.Context(), account.ID)
	if credErr == nil && len(encCreds) > 0 && len(h.encryptionKey) > 0 {
		creds, decErr := config.DecryptCredentials(encCreds, h.encryptionKey)
		if decErr == nil {
			// Disconnect first, then reconnect with stored creds
			_ = prov.Disconnect(c.Request.Context(), account.ID)
			_ = h.accounts.UpdateStatus(c.Request.Context(), account.ID, model.StatusConnecting, nil)
			reconnected, err := prov.Connect(c.Request.Context(), account.ID, creds)
			if err != nil {
				h.accounts.UpdateStatus(c.Request.Context(), account.ID, model.StatusInterrupted, strPtr(err.Error()))
				ProviderError(c, "Failed to reconnect: "+err.Error())
				return
			}
			c.JSON(http.StatusOK, reconnected)
			return
		}
	}

	// Fall back to adapter's own reconnect (e.g., WhatsApp with existing session)
	_ = h.accounts.UpdateStatus(c.Request.Context(), account.ID, model.StatusConnecting, nil)
	reconnected, err := prov.Reconnect(c.Request.Context(), account.ID)
	if err != nil {
		h.accounts.UpdateStatus(c.Request.Context(), account.ID, model.StatusInterrupted, strPtr(err.Error()))
		ProviderError(c, "Failed to reconnect: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, reconnected)
}

// GET /accounts/:id/auth-challenge
func (h *AccountHandler) GetAuthChallenge(c *gin.Context) {
	account, ok := h.getAccount(c)
	if !ok {
		return
	}

	prov, err := adapter.Get(account.Provider)
	if err != nil {
		ProviderError(c, "Provider not available")
		return
	}

	challenge, err := prov.GetAuthChallenge(c.Request.Context(), account.ID)
	if err != nil {
		ProviderError(c, "Failed to get auth challenge: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, challenge)
}

// GET /accounts/:id/qr returns the QR code as a PNG image.
func (h *AccountHandler) GetQRCode(c *gin.Context) {
	account, ok := h.getAccount(c)
	if !ok {
		return
	}

	prov, err := adapter.Get(account.Provider)
	if err != nil {
		ProviderError(c, "Provider not available")
		return
	}

	challenge, err := prov.GetAuthChallenge(c.Request.Context(), account.ID)
	if err != nil {
		ProviderError(c, "Failed to get auth challenge: "+err.Error())
		return
	}

	// Generate QR code PNG from payload
	png, err := qrcode.Encode(challenge.Payload, qrcode.Medium, 256)
	if err != nil {
		ProviderError(c, "Failed to generate QR code: "+err.Error())
		return
	}

	c.Data(http.StatusOK, "image/png", png)
}

// GET /wa/qr/:id — browser-friendly QR page with auto-refresh.
func (h *AccountHandler) QRPage(c *gin.Context) {
	id := c.Param("id")
	html := `<!DOCTYPE html>
<html><head><title>WhatsApp QR</title>
<meta http-equiv="refresh" content="5">
<style>
  body { font-family: -apple-system, sans-serif; display: flex; flex-direction: column;
    align-items: center; justify-content: center; min-height: 100vh; margin: 0;
    background: #111; color: #eee; }
  img { border: 16px solid #fff; border-radius: 8px; }
  h2 { margin-bottom: 24px; }
  p { color: #999; font-size: 14px; }
</style></head><body>
  <h2>Scan with WhatsApp</h2>
  <img src="/api/v1/accounts/` + id + `/qr?key=` + c.Query("key") + `" width="300" height="300" />
  <p>Page refreshes every 5 seconds. Open WhatsApp → Linked Devices → Link a Device.</p>
</body></html>`
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// POST /accounts/:id/checkpoint
func (h *AccountHandler) SolveCheckpoint(c *gin.Context) {
	account, ok := h.getAccount(c)
	if !ok {
		return
	}

	var req struct {
		Solution string `json:"solution" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Validation(c, err.Error())
		return
	}

	prov, provErr := adapter.Get(account.Provider)
	if provErr != nil {
		ProviderError(c, "Provider not available")
		return
	}

	if err := prov.SolveCheckpoint(c.Request.Context(), account.ID, req.Solution); err != nil {
		ProviderError(c, "Failed to solve checkpoint: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"object": "account", "id": account.ID, "status": "CHECKPOINT_SOLVED"})
}

func getProviderCapabilities(provider string) []string {
	switch provider {
	case "WHATSAPP":
		return []string{"messaging", "media", "groups", "read_receipts"}
	case "IMAP", "GMAIL", "OUTLOOK":
		return []string{"email", "folders", "attachments"}
	default:
		return []string{"messaging"}
	}
}

func strPtr(s string) *string {
	return &s
}

var _ = strconv.Atoi
