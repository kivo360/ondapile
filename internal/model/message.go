package model

import "time"

type Message struct {
	Object      string          `json:"object"`
	ID          string          `json:"id"`
	ChatID      string          `json:"chat_id"`
	AccountID   string          `json:"account_id"`
	Provider    string          `json:"provider"`
	ProviderID  string          `json:"provider_id"`
	Text        string          `json:"text"`
	SenderID    string          `json:"sender_id"`
	IsSender    bool            `json:"is_sender"`
	Timestamp   time.Time       `json:"timestamp"`
	Attachments []Attachment    `json:"attachments"`
	Reactions   []Reaction      `json:"reactions"`
	Quoted      *QuotedMessage  `json:"quoted,omitempty"`
	Seen        bool            `json:"seen"`
	SeenBy      map[string]bool `json:"seen_by,omitempty"`
	Delivered   bool            `json:"delivered"`
	Edited      bool            `json:"edited"`
	Deleted     bool            `json:"deleted"`
	Hidden      bool            `json:"hidden"`
	IsEvent     bool            `json:"is_event"`
	EventType   *int            `json:"event_type,omitempty"`
	Metadata    map[string]any  `json:"metadata"`
}

type Attachment struct {
	ID       string `json:"id"`
	Filename string `json:"filename,omitempty"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
	URL      string `json:"url,omitempty"`
}

type Reaction struct {
	Value    string `json:"value"`
	SenderID string `json:"sender_id"`
	IsSender bool   `json:"is_sender"`
}

type QuotedMessage struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// Event type constants
const (
	EventTypeUnknown            = 0
	EventTypeReaction           = 1
	EventTypeOwnerReaction      = 2
	EventTypeGroupCreated       = 3
	EventTypeGroupTitleChanged  = 4
	EventTypeParticipantAdded   = 5
	EventTypeParticipantRemoved = 6
	EventTypeParticipantLeft    = 7
	EventTypeMissedVoiceCall    = 8
	EventTypeMissedVideoCall    = 9
)
