package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"

	"ondapile/internal/model"
	"ondapile/internal/store"
)

type WebhookHandler struct {
	webhooks *store.WebhookStore
}

func NewWebhookHandler(s *store.Store) *WebhookHandler {
	return &WebhookHandler{
		webhooks: store.NewWebhookStore(s),
	}
}

// GET /webhooks
func (h *WebhookHandler) List(c *gin.Context) {
	// Check for organization_id in context (set by DualAuthMiddleware)
	orgID := c.GetString("organization_id")
	var webhooks []*model.Webhook
	var err error

	if orgID != "" {
		// Use organization-scoped query
		webhooks, err = h.webhooks.ListByOrganization(c.Request.Context(), orgID)
	} else {
		// Fall back to existing behavior for backward compatibility
		webhooks, err = h.webhooks.List(c.Request.Context())
	}

	if err != nil {
		Internal(c, "Failed to list webhooks")
		return
	}

	items := make([]any, len(webhooks))
	for i, w := range webhooks {
		items[i] = w
	}

	c.JSON(http.StatusOK, model.NewPaginatedList(items, "", false))
}

// POST /webhooks
func (h *WebhookHandler) Create(c *gin.Context) {
	var req struct {
		URL    string   `json:"url" binding:"required"`
		Events []string `json:"events" binding:"required"`
		Secret string   `json:"secret"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Validation(c, err.Error())
		return
	}

	if req.Secret == "" {
		req.Secret = generateWebhookSecret()
	}

	// Check for organization_id in context (set by DualAuthMiddleware)
	orgID := c.GetString("organization_id")
	var webhook *model.Webhook
	var err error

	if orgID != "" {
		// Use organization-scoped creation
		webhook, err = h.webhooks.CreateWithOrg(c.Request.Context(), req.URL, req.Events, req.Secret, orgID)
	} else {
		// Fall back to existing behavior for backward compatibility
		webhook, err = h.webhooks.Create(c.Request.Context(), req.URL, req.Events, req.Secret)
	}

	if err != nil {
		Internal(c, "Failed to create webhook")
		return
	}

	c.JSON(http.StatusCreated, webhook)
}

// DELETE /webhooks/:id
func (h *WebhookHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	orgID := c.GetString("organization_id")
	var err error
	if orgID != "" {
		err = h.webhooks.DeleteByIDAndOrg(c.Request.Context(), id, orgID)
	} else {
		err = h.webhooks.Delete(c.Request.Context(), id)
	}
	if err != nil {
		NotFound(c, "Webhook not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"object": "webhook", "id": id, "deleted": true})
}

func generateWebhookSecret() string {
	b := make([]byte, 24)
	rand.Read(b)
	return "whsec_" + hex.EncodeToString(b)
}
