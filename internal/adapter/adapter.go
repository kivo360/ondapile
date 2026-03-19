package adapter

import (
	"context"
	"time"

	"ondapile/internal/model"
)

// Provider is the core abstraction. Every messaging/email/calendar provider implements this.
type Provider interface {
	// Identity
	Name() string // "WHATSAPP", "IMAP", "GMAIL", etc.

	// Lifecycle
	Connect(ctx context.Context, accountID string, creds map[string]string) (*model.Account, error)
	Disconnect(ctx context.Context, accountID string) error
	Reconnect(ctx context.Context, accountID string) (*model.Account, error)
	Status(ctx context.Context, accountID string) (model.AccountStatus, error)

	// Auth flow
	GetAuthChallenge(ctx context.Context, accountID string) (*AuthChallenge, error)
	SolveCheckpoint(ctx context.Context, accountID string, solution string) error

	// Messaging
	ListChats(ctx context.Context, accountID string, opts ListOpts) (*model.PaginatedList[model.Chat], error)
	GetChat(ctx context.Context, accountID string, chatID string) (*model.Chat, error)
	ListMessages(ctx context.Context, accountID string, chatID string, opts ListOpts) (*model.PaginatedList[model.Message], error)
	SendMessage(ctx context.Context, accountID string, chatID string, msg SendMessageRequest) (*model.Message, error)
	StartChat(ctx context.Context, accountID string, req StartChatRequest) (*model.Chat, error)

	// Attendees
	ListAttendees(ctx context.Context, accountID string, opts ListOpts) (*model.PaginatedList[model.Attendee], error)
	GetAttendee(ctx context.Context, accountID string, attendeeID string) (*model.Attendee, error)

	// Media
	DownloadAttachment(ctx context.Context, accountID string, messageID string, attachmentID string) ([]byte, string, error)

	// Email-specific (only for email providers)
	SendEmail(ctx context.Context, accountID string, req SendEmailRequest) (*model.Email, error)
	ListEmails(ctx context.Context, accountID string, opts ListEmailOpts) (*model.PaginatedList[model.Email], error)
	GetEmail(ctx context.Context, accountID string, emailID string) (*model.Email, error)
}

type AuthChallenge struct {
	Type    string `json:"type"`    // "QR_CODE", "PAIRING_CODE", "OAUTH_URL", "CREDENTIALS"
	Payload string `json:"payload"` // QR data, URL, etc.
	Expiry  int64  `json:"expiry"`  // Unix timestamp
}

type ListOpts struct {
	Cursor string
	Limit  int
	Before *time.Time
	After  *time.Time
}

type SendMessageRequest struct {
	Text        string             `json:"text"`
	Attachments []AttachmentUpload `json:"attachments,omitempty"`
	QuotedMsgID *string            `json:"quoted_message_id,omitempty"`
}

type AttachmentUpload struct {
	Filename string `json:"filename"`
	Content  []byte `json:"content"`
	MimeType string `json:"mime_type"`
}

type StartChatRequest struct {
	AttendeeIdentifier string `json:"attendee_identifier"`
	Text               string `json:"text"`
}

type SendEmailRequest struct {
	To          []model.EmailAttendee `json:"to"`
	CC          []model.EmailAttendee `json:"cc,omitempty"`
	BCC         []model.EmailAttendee `json:"bcc,omitempty"`
	Subject     string                `json:"subject"`
	BodyHTML    string                `json:"body_html"`
	BodyPlain   string                `json:"body_plain"`
	ReplyToID   *string               `json:"reply_to_email_id,omitempty"`
	Attachments []AttachmentUpload    `json:"attachments,omitempty"`
}

type ListEmailOpts struct {
	Cursor    string
	Limit     int
	Folder    string
	From      string
	To        string
	Subject   string
	Before    *time.Time
	After     *time.Time
	HasAttach *bool
	IsRead    *bool
}
