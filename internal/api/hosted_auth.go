package api

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ondapile/internal/adapter"
	"ondapile/internal/store"
)

type HostedAuthHandler struct {
	store         *store.Store
	accounts      *store.AccountStore
	encryptionKey []byte
}

func NewHostedAuthHandler(s *store.Store, encryptionKey []byte) *HostedAuthHandler {
	return &HostedAuthHandler{
		store:         s,
		accounts:      store.NewAccountStore(s),
		encryptionKey: encryptionKey,
	}
}

type hostedAuthRequest struct {
	Provider    string     `json:"provider" binding:"required"`
	RedirectURL string     `json:"redirect_url" binding:"required"`
	Name        string     `json:"name,omitempty"`
	ExpiresOn   *time.Time `json:"expires_on,omitempty"`
}

type hostedAuthResponse struct {
	Object    string `json:"object"`
	URL       string `json:"url"`
	ExpiresAt int64  `json:"expires_at"`
}

// POST /accounts/hosted-auth
func (h *HostedAuthHandler) Create(c *gin.Context) {
	var req hostedAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Validation(c, err.Error())
		return
	}

	// Get provider adapter
	prov, err := adapter.Get(req.Provider)
	if err != nil {
		ProviderError(c, "Provider not available: "+req.Provider)
		return
	}

	// Check if provider supports OAuth
	if !prov.SupportsOAuth() {
		BadRequest(c, "Provider does not support OAuth flow")
		return
	}

	// Generate state token
	state, err := generateStateToken()
	if err != nil {
		Internal(c, "Failed to generate state token")
		return
	}

	// Get OAuth URL from provider
	oauthURL, err := prov.GetOAuthURL(c.Request.Context(), state)
	if err != nil {
		ProviderError(c, "Failed to get OAuth URL: "+err.Error())
		return
	}

	// Calculate expiry (default 10 minutes)
	expiresAt := time.Now().Add(10 * time.Minute)
	if req.ExpiresOn != nil {
		expiresAt = *req.ExpiresOn
	}

	c.JSON(http.StatusOK, hostedAuthResponse{
		Object:    "oauth_url",
		URL:       oauthURL,
		ExpiresAt: expiresAt.Unix(),
	})
}

func generateStateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
