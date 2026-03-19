package model

import "time"

// Calendar represents a calendar belonging to an account.
type Calendar struct {
	Object     string         `json:"object"` // always "calendar"
	ID         string         `json:"id"`     // "cal_xxxx"
	AccountID  string         `json:"account_id"`
	Provider   string         `json:"provider"` // "GOOGLE_CALENDAR", "OUTLOOK_CALENDAR"
	ProviderID string         `json:"provider_id"`
	Name       string         `json:"name"`
	Color      string         `json:"color,omitempty"`
	IsPrimary  bool           `json:"is_primary"`
	IsReadOnly bool           `json:"is_read_only"`
	TimeZone   string         `json:"timezone,omitempty"`
	Metadata   map[string]any `json:"metadata"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}
