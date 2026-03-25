package tracking

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ondapile/internal/model"
)

// HandleLink handles GET /l/:id for click tracking.
// It records the click event, dispatches a webhook, and redirects to the original URL.
func (t *Tracker) HandleLink(c *gin.Context) {
	emailID := c.Param("id")
	targetURL := c.Query("url")

	if targetURL == "" {
		c.String(http.StatusBadRequest, "missing url parameter")
		return
	}

	// Record the click
	ctx := c.Request.Context()
	err := t.store.RecordClick(ctx, emailID, targetURL)
	if err != nil {
		// Log error but continue with redirect
		_ = err
	}

	// Dispatch webhook event if dispatcher is available
	if t.dispatcher != nil {
		t.dispatcher.Dispatch(ctx, model.EventEmailClicked, map[string]interface{}{
			"email_id":   emailID,
			"url":        targetURL,
			"clicked_at": time.Now().UTC(),
		})
	}

	// Redirect to original URL
	c.Redirect(http.StatusFound, targetURL)
}
