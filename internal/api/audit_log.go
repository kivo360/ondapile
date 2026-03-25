package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ondapile/internal/model"
	"ondapile/internal/store"
)

// AuditLogHandler handles audit log API endpoints
type AuditLogHandler struct {
	auditLog *store.AuditLogStore
}

// NewAuditLogHandler creates a new AuditLogHandler
func NewAuditLogHandler(s *store.Store) *AuditLogHandler {
	return &AuditLogHandler{
		auditLog: store.NewAuditLogStore(s),
	}
}

// List handles GET /api/v1/audit-log
func (h *AuditLogHandler) List(c *gin.Context) {
	orgID := c.GetString("organization_id")
	if orgID == "" {
		// Fallback for static key - return empty list
		c.JSON(http.StatusOK, model.NewPaginatedList([]any{}, "", false))
		return
	}

	entries, err := h.auditLog.List(c.Request.Context(), orgID, 100)
	if err != nil {
		Internal(c, "Failed to list audit log")
		return
	}

	items := make([]any, len(entries))
	for i, e := range entries {
		items[i] = e
	}

	c.JSON(http.StatusOK, model.NewPaginatedList(items, "", false))
}
