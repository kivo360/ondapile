package adapter

import (
	"context"
	"fmt"
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

	// Calendar (only for calendar providers)
	ListCalendars(ctx context.Context, accountID string, opts ListOpts) (*model.PaginatedList[model.Calendar], error)
	GetCalendar(ctx context.Context, accountID string, calendarID string) (*model.Calendar, error)
	ListEvents(ctx context.Context, accountID string, calendarID string, opts ListOpts) (*model.PaginatedList[model.CalendarEvent], error)
	GetEvent(ctx context.Context, accountID string, calendarID string, eventID string) (*model.CalendarEvent, error)
	CreateEvent(ctx context.Context, accountID string, calendarID string, req CreateEventRequest) (*model.CalendarEvent, error)
	UpdateEvent(ctx context.Context, accountID string, calendarID string, eventID string, req UpdateEventRequest) (*model.CalendarEvent, error)
	DeleteEvent(ctx context.Context, accountID string, calendarID string, eventID string) error

	// OAuth (for providers that support hosted OAuth flow)
	SupportsOAuth() bool
	GetOAuthURL(ctx context.Context, state string) (string, error)
	HandleOAuthCallback(ctx context.Context, code string) (map[string]string, error)
}

// ErrNotSupported is returned by adapter methods that are not applicable for the provider.
var ErrNotSupported = fmt.Errorf("operation not supported by this provider")

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

type CreateEventRequest struct {
	Title       string                   `json:"title"`
	Description string                   `json:"description,omitempty"`
	Location    string                   `json:"location,omitempty"`
	StartAt     time.Time                `json:"start_at"`
	EndAt       time.Time                `json:"end_at"`
	AllDay      bool                     `json:"all_day"`
	Attendees   []model.CalendarAttendee `json:"attendees,omitempty"`
	Reminders   []model.Reminder         `json:"reminders,omitempty"`
	Conference  *ConferenceRequest       `json:"conference,omitempty"`
}

type UpdateEventRequest struct {
	Title       *string                  `json:"title,omitempty"`
	Description *string                  `json:"description,omitempty"`
	Location    *string                  `json:"location,omitempty"`
	StartAt     *time.Time               `json:"start_at,omitempty"`
	EndAt       *time.Time               `json:"end_at,omitempty"`
	AllDay      *bool                    `json:"all_day,omitempty"`
	Attendees   []model.CalendarAttendee `json:"attendees,omitempty"`
	Reminders   []model.Reminder         `json:"reminders,omitempty"`
}

type ConferenceRequest struct {
	Type       string `json:"type"`
	AutoCreate bool   `json:"auto_create"`
}
