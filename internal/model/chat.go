package model

import "time"

type Chat struct {
	Object      string          `json:"object"`
	ID          string          `json:"id"`
	AccountID   string          `json:"account_id"`
	Provider    string          `json:"provider"`
	ProviderID  string          `json:"provider_id"`
	Type        string          `json:"type"`
	Name        *string         `json:"name,omitempty"`
	Attendees   []Attendee      `json:"attendees"`
	LastMessage *MessagePreview `json:"last_message,omitempty"`
	UnreadCount int             `json:"unread_count"`
	IsGroup     bool            `json:"is_group"`
	IsArchived  bool            `json:"is_archived"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	Metadata    map[string]any  `json:"metadata"`
}

type MessagePreview struct {
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

type ChatType string

const (
	ChatTypeOneToOne  ChatType = "ONE_TO_ONE"
	ChatTypeGroup     ChatType = "GROUP"
	ChatTypeChannel   ChatType = "CHANNEL"
	ChatTypeBroadcast ChatType = "BROADCAST"
)
