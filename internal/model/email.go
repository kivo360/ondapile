package model

import "time"

type Email struct {
	Object           string            `json:"object"`
	ID               string            `json:"id"`
	AccountID        string            `json:"account_id"`
	Provider         string            `json:"provider"`
	ProviderID       *EmailProviderID  `json:"provider_id"`
	Subject          string            `json:"subject"`
	Body             string            `json:"body"`
	BodyPlain        string            `json:"body_plain"`
	FromAttendee     *EmailAttendee    `json:"from_attendee"`
	ToAttendees      []EmailAttendee   `json:"to_attendees"`
	CCAttendees      []EmailAttendee   `json:"cc_attendees"`
	BCCAttendees     []EmailAttendee   `json:"bcc_attendees"`
	ReplyToAttendees []EmailAttendee   `json:"reply_to_attendees"`
	Date             time.Time         `json:"date"`
	HasAttachments   bool              `json:"has_attachments"`
	Attachments      []EmailAttachment `json:"attachments"`
	Folders          []string          `json:"folders"`
	Role             string            `json:"role"`
	Read             bool              `json:"read"`
	ReadDate         *time.Time        `json:"read_date,omitempty"`
	IsComplete       bool              `json:"is_complete"`
	Headers          []EmailHeader     `json:"headers"`
	Tracking         *EmailTracking    `json:"tracking,omitempty"`
	Metadata         map[string]any    `json:"metadata"`
}

type EmailProviderID struct {
	MessageID string `json:"message_id"`
	ThreadID  string `json:"thread_id"`
}

type EmailAttendee struct {
	DisplayName    string `json:"display_name,omitempty"`
	Identifier     string `json:"identifier"`
	IdentifierType string `json:"identifier_type"`
}

type EmailAttachment struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
}

type EmailHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type EmailTracking struct {
	Opens         int           `json:"opens"`
	FirstOpenedAt *time.Time    `json:"first_opened_at,omitempty"`
	Clicks        int           `json:"clicks"`
	LinksClicked  []ClickedLink `json:"links_clicked,omitempty"`
}

type ClickedLink struct {
	URL       string    `json:"url"`
	ClickedAt time.Time `json:"clicked_at"`
}

// Folder role constants
const (
	FolderInbox   = "INBOX"
	FolderSent    = "SENT"
	FolderDrafts  = "DRAFTS"
	FolderTrash   = "TRASH"
	FolderSpam    = "SPAM"
	FolderArchive = "ARCHIVE"
	FolderCustom  = "CUSTOM"
)
