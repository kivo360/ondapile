package api

import (
	"github.com/gin-gonic/gin"

	"ondapile/internal/store"
	"ondapile/internal/webhook"
)

// Router sets up all API routes.
func Router(s *store.Store, w *webhook.Dispatcher, apiKey string, encryptionKey []byte) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(CORSMiddleware())
	r.Use(RateLimitMiddleware(10, 100)) // 10 req/s sustained, 100 burst

	// Health check (no auth)
	r.GET("/health", HealthHandler)

	// Metrics (no auth)
	r.GET("/metrics", MetricsHandler(s.Pool))

	// API v1 — all require auth
	v1 := r.Group("/api/v1", AuthMiddleware(apiKey))

	// Register handlers
	accountH := NewAccountHandler(s, encryptionKey)
	chatH := NewChatHandler(s)
	msgH := NewMessageHandler(s)
	whH := NewWebhookHandler(s)
	emailH := NewEmailHandler(s)
	attendeeH := NewAttendeeHandler(s)

	// Accounts
	v1.GET("/accounts", accountH.List)
	v1.POST("/accounts", accountH.Create)
	v1.GET("/accounts/:id", accountH.Get)
	v1.DELETE("/accounts/:id", accountH.Delete)
	v1.POST("/accounts/:id/reconnect", accountH.Reconnect)
	v1.GET("/accounts/:id/auth-challenge", accountH.GetAuthChallenge)
	v1.GET("/accounts/:id/qr", accountH.GetQRCode)
	v1.POST("/accounts/:id/checkpoint", accountH.SolveCheckpoint)

	// WhatsApp QR page (no auth — just open in browser)
	r.GET("/wa/qr/:id", accountH.QRPage)

	// Chats
	v1.GET("/chats", chatH.List)
	v1.POST("/chats", chatH.Create)
	v1.GET("/chats/:id", chatH.Get)
	v1.PATCH("/chats/:id", chatH.Update)
	v1.DELETE("/chats/:id", chatH.Delete)
	v1.GET("/chats/:id/messages", chatH.ListMessages)
	v1.POST("/chats/:id/messages", chatH.SendMessage)
	v1.GET("/chats/:id/attendees", chatH.ListAttendees)

	// Messages (cross-chat)
	v1.GET("/messages", msgH.List)
	v1.GET("/messages/:id", msgH.Get)
	v1.DELETE("/messages/:id", msgH.Delete)
	v1.GET("/messages/:id/attachments/:att_id", msgH.DownloadAttachment)
	v1.POST("/messages/:id/reactions", msgH.AddReaction)

	// Attendees (cross-chat)
	v1.GET("/attendees", attendeeH.List)
	v1.GET("/attendees/:id", attendeeH.Get)
	v1.GET("/attendees/:id/avatar", attendeeH.GetAvatar)
	v1.GET("/attendees/:id/chats", attendeeH.ListChats)
	v1.GET("/attendees/:id/messages", attendeeH.ListMessages)

	// Webhooks
	v1.GET("/webhooks", whH.List)
	v1.POST("/webhooks", whH.Create)
	v1.DELETE("/webhooks/:id", whH.Delete)

	// Emails
	v1.GET("/emails", emailH.List)
	v1.POST("/emails", emailH.Send)
	v1.GET("/emails/:id", emailH.Get)
	v1.PUT("/emails/:id", emailH.Update)
	v1.DELETE("/emails/:id", emailH.Delete)
	v1.GET("/emails/:id/attachments/:att_id", emailH.DownloadAttachment)
	v1.GET("/emails/folders", emailH.ListFolders)

	// Calendar routes
	calH := NewCalendarHandler(s)
	v1.GET("/calendars", calH.List)
	v1.GET("/calendars/:id", calH.Get)
	v1.GET("/calendars/:id/events", calH.ListEvents)
	v1.POST("/calendars/:id/events", calH.CreateEvent)
	v1.GET("/calendars/:id/events/:event_id", calH.GetEvent)
	v1.PATCH("/calendars/:id/events/:event_id", calH.UpdateEvent)
	v1.DELETE("/calendars/:id/events/:event_id", calH.DeleteEvent)

	// OAuth routes
	hostedAuthH := NewHostedAuthHandler(s, encryptionKey)
	v1.POST("/accounts/hosted-auth", hostedAuthH.Create)

	oauthH := NewOAuthCallbackHandler(s, encryptionKey)
	v1.GET("/oauth/callback/:provider", oauthH.Callback)

	return r
	v1.GET("/emails", emailH.List)
	v1.POST("/emails", emailH.Send)
	v1.GET("/emails/:id", emailH.Get)
	v1.PUT("/emails/:id", emailH.Update)
	v1.DELETE("/emails/:id", emailH.Delete)
	v1.GET("/emails/:id/attachments/:att_id", emailH.DownloadAttachment)
	v1.GET("/emails/folders", emailH.ListFolders)

	return r
}
