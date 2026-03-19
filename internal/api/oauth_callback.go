package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"ondapile/internal/adapter"
	"ondapile/internal/config"
	"ondapile/internal/model"
	"ondapile/internal/store"
)

// mapCallbackProvider maps OAuth callback URL provider names to registered adapter names.
var callbackProviderMap = map[string]string{
	"google":    "GMAIL",
	"gmail":     "GMAIL",
	"microsoft": "OUTLOOK",
	"outlook":   "OUTLOOK",
	"linkedin":  "LINKEDIN",
	"instagram": "INSTAGRAM",
	"telegram":  "TELEGRAM",
}

type OAuthCallbackHandler struct {
	store         *store.Store
	accounts      *store.AccountStore
	encryptionKey []byte
}

func NewOAuthCallbackHandler(s *store.Store, encryptionKey []byte) *OAuthCallbackHandler {
	return &OAuthCallbackHandler{
		store:         s,
		accounts:      store.NewAccountStore(s),
		encryptionKey: encryptionKey,
	}
}

// GET /api/v1/oauth/callback/:provider
func (h *OAuthCallbackHandler) Callback(c *gin.Context) {
	provider := c.Param("provider")
	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		BadRequest(c, "Missing authorization code")
		return
	}

	// Map URL provider name to registered adapter name
	adapterName := provider
	if mapped, ok := callbackProviderMap[provider]; ok {
		adapterName = mapped
	}
	prov, err := adapter.Get(adapterName)
	if err != nil {
		ProviderError(c, "Provider not available: "+provider)
		return
	}

	// Handle OAuth callback
	creds, err := prov.HandleOAuthCallback(c.Request.Context(), code)
	if err != nil {
		ProviderError(c, "OAuth callback failed: "+err.Error())
		return
	}

	// Extract account identifier from credentials (email or user ID)
	identifier := creds["email"]
	if identifier == "" {
		identifier = creds["user_id"]
	}
	if identifier == "" {
		identifier = creds["identifier"]
	}
	if identifier == "" {
		identifier = provider + "_user"
	}

	// Check if account already exists
	existing, _ := h.accounts.GetByProviderIdentifier(c.Request.Context(), adapterName, identifier)
	if existing != nil {
		// Update existing account
		if err := h.accounts.UpdateStatus(c.Request.Context(), existing.ID, model.StatusConnecting, nil); err != nil {
			slog.Warn("failed to update account status", "error", err)
		}
	} else {
		// Create new account
		name := creds["name"]
		if name == "" {
			name = identifier
		}

		caps := getProviderCapabilities(adapterName)
		account, err := h.accounts.Create(c.Request.Context(), store.CreateAccountParams{
			Provider:     adapterName,
			Name:         name,
			Identifier:   identifier,
			Status:       string(model.StatusOperational),
			Capabilities: caps,
			Metadata:     map[string]any{},
		})
		if err != nil {
			Internal(c, "Failed to create account")
			return
		}
		existing = account
	}

	// Store credentials
	if len(creds) > 0 && len(h.encryptionKey) > 0 {
		// Encrypt and store credentials
		encCreds, encErr := config.EncryptCredentials(creds, h.encryptionKey)
		if encErr != nil {
			slog.Warn("failed to encrypt credentials", "error", encErr)
		} else {
			if storeErr := h.accounts.UpdateCredentials(c.Request.Context(), existing.ID, encCreds); storeErr != nil {
				slog.Warn("failed to store encrypted credentials", "error", storeErr)
			}
		}
	}

	// Get redirect URL from state or use default
	redirectURL := "/"
	if state != "" {
		// In a real implementation, we'd look up the stored redirect URL by state
		// For now, just use a generic success page
		redirectURL = "/oauth/success"
	}

	c.Redirect(http.StatusFound, redirectURL)
}
