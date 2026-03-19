package model

import "time"

type Attendee struct {
	Object         string         `json:"object"`
	ID             string         `json:"id"`
	AccountID      string         `json:"account_id"`
	Provider       string         `json:"provider"`
	ProviderID     string         `json:"provider_id"`
	Name           string         `json:"name,omitempty"`
	Identifier     string         `json:"identifier"`
	IdentifierType string         `json:"identifier_type"`
	AvatarURL      string         `json:"avatar_url,omitempty"`
	IsSelf         bool           `json:"is_self,omitempty"`
	Metadata       map[string]any `json:"metadata"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type IdentifierType string

const (
	IdentifierTypePhoneNumber  IdentifierType = "PHONE_NUMBER"
	IdentifierTypeEmailAddress IdentifierType = "EMAIL_ADDRESS"
	IdentifierTypeUsername     IdentifierType = "USERNAME"
	IdentifierTypeProfileURL   IdentifierType = "PROFILE_URL"
	IdentifierTypeProviderID   IdentifierType = "PROVIDER_ID"
)

type RelationStatus string

const (
	RelationConnected       RelationStatus = "CONNECTED"
	RelationPendingSent     RelationStatus = "PENDING_SENT"
	RelationPendingReceived RelationStatus = "PENDING_RECEIVED"
	RelationNotConnected    RelationStatus = "NOT_CONNECTED"
	RelationFollowing       RelationStatus = "FOLLOWING"
	RelationBlocked         RelationStatus = "BLOCKED"
)
