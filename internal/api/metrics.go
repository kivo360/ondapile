package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

var startTime = time.Now()

func MetricsHandler(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Count accounts by status
		var totalAccounts, operationalAccounts int
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM accounts").Scan(&totalAccounts)
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM accounts WHERE status = 'OPERATIONAL'").Scan(&operationalAccounts)

		// Count messages
		var totalMessages int
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM messages").Scan(&totalMessages)

		// Count chats
		var totalChats int
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM chats").Scan(&totalChats)

		// Count emails
		var totalEmails int
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM emails").Scan(&totalEmails)

		// Count webhooks
		var totalWebhooks, activeWebhooks int
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM webhooks").Scan(&totalWebhooks)
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM webhooks WHERE active = true").Scan(&activeWebhooks)

		// Webhook delivery stats
		var deliveredWebhooks, failedWebhooks int
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM webhook_deliveries WHERE delivered = true").Scan(&deliveredWebhooks)
		pool.QueryRow(ctx, "SELECT COUNT(*) FROM webhook_deliveries WHERE delivered = false AND attempts >= 3").Scan(&failedWebhooks)

		uptime := time.Since(startTime)

		c.JSON(http.StatusOK, gin.H{
			"object":         "metrics",
			"uptime_seconds": int(uptime.Seconds()),
			"uptime_human":   uptime.Round(time.Second).String(),
			"accounts": gin.H{
				"total":       totalAccounts,
				"operational": operationalAccounts,
			},
			"messages": gin.H{
				"total": totalMessages,
			},
			"chats": gin.H{
				"total": totalChats,
			},
			"emails": gin.H{
				"total": totalEmails,
			},
			"webhooks": gin.H{
				"total":              totalWebhooks,
				"active":             activeWebhooks,
				"deliveries_success": deliveredWebhooks,
				"deliveries_failed":  failedWebhooks,
			},
		})
	}
}
