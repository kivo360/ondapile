package model

import "time"

type Webhook struct {
	Object    string    `json:"object"`
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"`
	Secret    string    `json:"secret"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

type WebhookDelivery struct {
	ID         int64      `json:"id"`
	WebhookID  string     `json:"webhook_id"`
	Event      string     `json:"event"`
	Payload    any        `json:"payload"`
	StatusCode *int       `json:"status_code,omitempty"`
	Attempts   int        `json:"attempts"`
	NextRetry  *time.Time `json:"next_retry,omitempty"`
	Delivered  bool       `json:"delivered"`
	CreatedAt  time.Time  `json:"created_at"`
}

type WebhookEvent struct {
	Event     string    `json:"event"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}

// Webhook event type constants
const (
	EventAccountConnected    = "account.connected"
	EventAccountDisconnected = "account.disconnected"
	EventAccountStatusChange = "account.status_changed"
	EventAccountCheckpoint   = "account.checkpoint"

	EventMessageReceived = "message.received"
	EventMessageSent     = "message.sent"
	EventMessageRead     = "message.read"
	EventMessageReaction = "message.reaction"
	EventMessageDeleted  = "message.deleted"
	EventChatCreated     = "chat.created"

	EventEmailReceived = "email.received"
	EventEmailSent     = "email.sent"
	EventEmailOpened   = "email.opened"
	EventEmailClicked  = "email.clicked"
	EventEmailBounced  = "email.bounced"
)
