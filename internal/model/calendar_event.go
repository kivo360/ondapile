package model

import "time"

// CalendarEvent represents an event in a calendar.
type CalendarEvent struct {
	Object        string             `json:"object"` // always "calendar_event"
	ID            string             `json:"id"`     // "evt_xxxx"
	AccountID     string             `json:"account_id"`
	CalendarID    string             `json:"calendar_id"`
	Provider      string             `json:"provider"`
	ProviderID    string             `json:"provider_id"`
	Title         string             `json:"title"`
	Description   string             `json:"description,omitempty"`
	Location      string             `json:"location,omitempty"`
	StartAt       time.Time          `json:"start_at"`
	EndAt         time.Time          `json:"end_at"`
	AllDay        bool               `json:"all_day"`
	Status        string             `json:"status"` // CONFIRMED, TENTATIVE, CANCELLED
	Attendees     []CalendarAttendee `json:"attendees"`
	Reminders     []Reminder         `json:"reminders"`
	ConferenceURL string             `json:"conference_url,omitempty"`
	Recurrence    *string            `json:"recurrence,omitempty"`
	Metadata      map[string]any     `json:"metadata"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

// CalendarAttendee represents an attendee of a calendar event.
type CalendarAttendee struct {
	DisplayName string `json:"display_name,omitempty"`
	Identifier  string `json:"identifier"`
	RSVP        string `json:"rsvp,omitempty"` // ACCEPTED, DECLINED, TENTATIVE, NEEDS_ACTION
	Organizer   bool   `json:"organizer,omitempty"`
}

// Reminder represents a reminder for a calendar event.
type Reminder struct {
	Method        string `json:"method"` // popup, email
	MinutesBefore int    `json:"minutes_before"`
}

// Calendar event status constants.
const (
	EventStatusConfirmed = "CONFIRMED"
	EventStatusTentative = "TENTATIVE"
	EventStatusCancelled = "CANCELLED"
)

// Calendar webhook event constants.
const (
	EventCalendarEventCreated = "calendar.event_created"
	EventCalendarEventUpdated = "calendar.event_updated"
	EventCalendarEventDeleted = "calendar.event_deleted"
	EventCalendarEventRSVP    = "calendar.event_rsvp"
)
