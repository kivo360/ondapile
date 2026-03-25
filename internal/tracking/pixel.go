package tracking

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ondapile/internal/model"
)

// transparentGIFBytes is a pre-computed 1x1 transparent GIF (43 bytes).
// This is the standard GIF89a format for a 1x1 transparent pixel.
var transparentGIFBytes = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00,
	0x01, 0x00, 0x80, 0x00, 0x00, 0xff, 0xff, 0xff,
	0x00, 0x00, 0x00, 0x21, 0xf9, 0x04, 0x01, 0x00,
	0x00, 0x00, 0x00, 0x2c, 0x00, 0x00, 0x00, 0x00,
	0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02, 0x44,
	0x01, 0x00, 0x3b,
}

// HandlePixel handles GET /t/:id for open tracking.
// It records the open event, dispatches a webhook, and returns a 1x1 transparent GIF.
func (t *Tracker) HandlePixel(c *gin.Context) {
	emailID := c.Param("id")

	// Record the open
	ctx := c.Request.Context()
	err := t.store.RecordOpen(ctx, emailID)
	_ = err // Silently ignore - don't leak info to email clients

	// Dispatch webhook event if dispatcher is available
	if t.dispatcher != nil {
		t.dispatcher.Dispatch(ctx, model.EventEmailOpened, map[string]interface{}{
			"email_id":  emailID,
			"opened_at": time.Now().UTC(),
		})
	}

	// Serve 1x1 transparent GIF
	c.Data(http.StatusOK, "image/gif", transparentGIFBytes)
}
