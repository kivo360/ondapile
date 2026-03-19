package outlook

import (
	"time"

	"ondapile/internal/model"

	"github.com/google/uuid"
)

// normalizeGraphEmail converts a Microsoft Graph API message response to an Email model.
func normalizeGraphEmail(raw map[string]interface{}, accountID string) *model.Email {
	email := &model.Email{
		Object:      "email",
		ID:          "email_" + uuid.New().String(),
		AccountID:   accountID,
		Provider:    "OUTLOOK",
		Metadata:    raw,
		Attachments: []model.EmailAttachment{},
	}

	// Extract message ID
	if id, ok := raw["id"].(string); ok {
		email.ProviderID = &model.EmailProviderID{
			MessageID: id,
		}
	}

	// Extract subject
	if subject, ok := raw["subject"].(string); ok {
		email.Subject = subject
	}

	// Extract body
	if body, ok := raw["body"].(map[string]interface{}); ok {
		if contentType, ok := body["contentType"].(string); ok {
			if content, ok := body["content"].(string); ok {
				if contentType == "html" {
					email.Body = content
				} else {
					email.BodyPlain = content
				}
			}
		}
	}

	// Extract sender (from)
	if from, ok := raw["from"].(map[string]interface{}); ok {
		email.FromAttendee = parseGraphEmailAddress(from)
	}

	// Extract recipients (to)
	if toRecipients, ok := raw["toRecipients"].([]interface{}); ok {
		email.ToAttendees = parseGraphRecipients(toRecipients)
	}

	// Extract CC recipients
	if ccRecipients, ok := raw["ccRecipients"].([]interface{}); ok {
		email.CCAttendees = parseGraphRecipients(ccRecipients)
	}

	// Extract BCC recipients
	if bccRecipients, ok := raw["bccRecipients"].([]interface{}); ok {
		email.BCCAttendees = parseGraphRecipients(bccRecipients)
	}

	// Extract date
	if receivedDateTime, ok := raw["receivedDateTime"].(string); ok {
		email.Date = parseGraphTime(receivedDateTime)
	} else if createdDateTime, ok := raw["createdDateTime"].(string); ok {
		email.Date = parseGraphTime(createdDateTime)
	}

	// Extract read status
	if isRead, ok := raw["isRead"].(bool); ok {
		email.Read = isRead
	}

	// Extract has attachments
	if hasAttachments, ok := raw["hasAttachments"].(bool); ok {
		email.HasAttachments = hasAttachments
	}

	// Extract attachments
	if attachments, ok := raw["attachments"].([]interface{}); ok {
		for _, att := range attachments {
			if attMap, ok := att.(map[string]interface{}); ok {
				attachment := parseGraphAttachment(attMap)
				if attachment.ID != "" {
					email.Attachments = append(email.Attachments, attachment)
				}
			}
		}
	}

	// Extract conversation ID as thread ID
	if convId, ok := raw["conversationId"].(string); ok {
		if email.ProviderID != nil {
			email.ProviderID.ThreadID = convId
		}
	}

	// Extract internet message ID
	if internetMessageId, ok := raw["internetMessageId"].(string); ok {
		// Store in headers
		email.Headers = append(email.Headers, model.EmailHeader{
			Key:   "Message-ID",
			Value: internetMessageId,
		})
	}

	return email
}

// normalizeGraphEmailList converts a Microsoft Graph API messages list response to a slice of Email models.
func normalizeGraphEmailList(raw map[string]interface{}, accountID string) ([]model.Email, string) {
	var emails []model.Email
	nextLink := ""

	if value, ok := raw["value"].([]interface{}); ok {
		for _, item := range value {
			if itemMap, ok := item.(map[string]interface{}); ok {
				email := normalizeGraphEmail(itemMap, accountID)
				emails = append(emails, *email)
			}
		}
	}

	// Extract next link
	if link, ok := raw["@odata.nextLink"].(string); ok {
		nextLink = link
	}

	return emails, nextLink
}

// normalizeGraphCalendar converts a Microsoft Graph API calendar response to a Calendar model.
func normalizeGraphCalendar(raw map[string]interface{}, accountID string) *model.Calendar {
	calendar := &model.Calendar{
		Object:    "calendar",
		ID:        "cal_" + uuid.New().String(),
		AccountID: accountID,
		Provider:  "OUTLOOK",
		Metadata:  raw,
	}

	// Extract calendar ID
	if id, ok := raw["id"].(string); ok {
		calendar.ProviderID = id
	}

	// Extract name
	if name, ok := raw["name"].(string); ok {
		calendar.Name = name
	}

	// Check if default calendar
	if isDefaultCalendar, ok := raw["isDefaultCalendar"].(bool); ok {
		calendar.IsPrimary = isDefaultCalendar
	}

	// Extract color
	if color, ok := raw["color"].(string); ok {
		calendar.Color = color
	}

	// Check canShare for read-only indication
	if canShare, ok := raw["canShare"].(bool); ok {
		calendar.IsReadOnly = !canShare
	}

	// Extract timestamps
	calendar.CreatedAt = time.Now()
	calendar.UpdatedAt = time.Now()

	return calendar
}

// normalizeGraphCalendarList converts a Microsoft Graph API calendar list response to a slice of Calendar models.
func normalizeGraphCalendarList(raw map[string]interface{}, accountID string) ([]model.Calendar, string) {
	var calendars []model.Calendar
	nextLink := ""

	if value, ok := raw["value"].([]interface{}); ok {
		for _, item := range value {
			if itemMap, ok := item.(map[string]interface{}); ok {
				calendar := normalizeGraphCalendar(itemMap, accountID)
				calendars = append(calendars, *calendar)
			}
		}
	}

	// Extract next link
	if link, ok := raw["@odata.nextLink"].(string); ok {
		nextLink = link
	}

	return calendars, nextLink
}

// normalizeGraphEvent converts a Microsoft Graph API event response to a CalendarEvent model.
func normalizeGraphEvent(raw map[string]interface{}, accountID string, calendarID string) *model.CalendarEvent {
	event := &model.CalendarEvent{
		Object:     "calendar_event",
		ID:         "evt_" + uuid.New().String(),
		AccountID:  accountID,
		CalendarID: calendarID,
		Provider:   "OUTLOOK",
		Metadata:   raw,
		Attendees:  []model.CalendarAttendee{},
		Reminders:  []model.Reminder{},
	}

	// Extract event ID
	if id, ok := raw["id"].(string); ok {
		event.ProviderID = id
	}

	// Extract subject/title
	if subject, ok := raw["subject"].(string); ok {
		event.Title = subject
	}

	// Extract body/description
	if body, ok := raw["body"].(map[string]interface{}); ok {
		if content, ok := body["content"].(string); ok {
			event.Description = content
		}
	}

	// Extract location
	if location, ok := raw["location"].(map[string]interface{}); ok {
		if displayName, ok := location["displayName"].(string); ok {
			event.Location = displayName
		}
	}

	// Extract times
	if start, ok := raw["start"].(map[string]interface{}); ok {
		event.StartAt = parseGraphEventTime(start)
		if _, hasDate := start["date"]; hasDate {
			event.AllDay = true
		}
	}
	if end, ok := raw["end"].(map[string]interface{}); ok {
		event.EndAt = parseGraphEventTime(end)
	}

	// Extract isAllDay
	if isAllDay, ok := raw["isAllDay"].(bool); ok {
		event.AllDay = isAllDay
	}

	// Extract status
	if responseStatus, ok := raw["responseStatus"].(map[string]interface{}); ok {
		if response, ok := responseStatus["response"].(string); ok {
			switch response {
			case "accepted":
				event.Status = model.EventStatusConfirmed
			case "tentativelyAccepted":
				event.Status = model.EventStatusTentative
			case "declined":
				event.Status = model.EventStatusCancelled
			default:
				event.Status = model.EventStatusConfirmed
			}
		}
	} else {
		event.Status = model.EventStatusConfirmed
	}

	// Extract attendees
	if attendees, ok := raw["attendees"].([]interface{}); ok {
		for _, att := range attendees {
			if attMap, ok := att.(map[string]interface{}); ok {
				attendee := parseGraphAttendee(attMap)
				event.Attendees = append(event.Attendees, attendee)
			}
		}
	}

	// Extract online meeting URL
	if onlineMeeting, ok := raw["onlineMeeting"].(map[string]interface{}); ok {
		if joinUrl, ok := onlineMeeting["joinUrl"].(string); ok {
			event.ConferenceURL = joinUrl
		}
	}

	// Extract recurrence
	if recurrence, ok := raw["recurrence"].(map[string]interface{}); ok {
		// Serialize recurrence to string (simplified)
		if pattern, ok := recurrence["pattern"].(map[string]interface{}); ok {
			if rrule := buildRecurrenceRule(pattern); rrule != "" {
				event.Recurrence = &rrule
			}
		}
	}

	// Extract created/modified times
	if createdDateTime, ok := raw["createdDateTime"].(string); ok {
		event.CreatedAt = parseGraphTime(createdDateTime)
	}
	if lastModifiedDateTime, ok := raw["lastModifiedDateTime"].(string); ok {
		event.UpdatedAt = parseGraphTime(lastModifiedDateTime)
	}

	return event
}

// normalizeGraphEventList converts a Microsoft Graph API events list response to a slice of CalendarEvent models.
func normalizeGraphEventList(raw map[string]interface{}, accountID string, calendarID string) ([]model.CalendarEvent, string) {
	var events []model.CalendarEvent
	nextLink := ""

	if value, ok := raw["value"].([]interface{}); ok {
		for _, item := range value {
			if itemMap, ok := item.(map[string]interface{}); ok {
				event := normalizeGraphEvent(itemMap, accountID, calendarID)
				events = append(events, *event)
			}
		}
	}

	// Extract next link
	if link, ok := raw["@odata.nextLink"].(string); ok {
		nextLink = link
	}

	return events, nextLink
}

// Helper functions

func parseGraphEmailAddress(raw map[string]interface{}) *model.EmailAttendee {
	if emailAddress, ok := raw["emailAddress"].(map[string]interface{}); ok {
		attendee := &model.EmailAttendee{
			IdentifierType: string(model.IdentifierTypeEmailAddress),
		}
		if addr, ok := emailAddress["address"].(string); ok {
			attendee.Identifier = addr
		}
		if name, ok := emailAddress["name"].(string); ok {
			attendee.DisplayName = name
		}
		return attendee
	}
	return nil
}

func parseGraphRecipients(recipients []interface{}) []model.EmailAttendee {
	var attendees []model.EmailAttendee
	for _, r := range recipients {
		if rMap, ok := r.(map[string]interface{}); ok {
			if attendee := parseGraphEmailAddress(rMap); attendee != nil {
				attendees = append(attendees, *attendee)
			}
		}
	}
	return attendees
}

func parseGraphAttachment(raw map[string]interface{}) model.EmailAttachment {
	attachment := model.EmailAttachment{}

	if id, ok := raw["id"].(string); ok {
		attachment.ID = id
	}
	if name, ok := raw["name"].(string); ok {
		attachment.Filename = name
	}
	if contentType, ok := raw["contentType"].(string); ok {
		attachment.MimeType = contentType
	}
	if size, ok := raw["size"].(float64); ok {
		attachment.Size = int64(size)
	}

	return attachment
}

func parseGraphAttendee(raw map[string]interface{}) model.CalendarAttendee {
	attendee := model.CalendarAttendee{}

	if emailAddress, ok := raw["emailAddress"].(map[string]interface{}); ok {
		if addr, ok := emailAddress["address"].(string); ok {
			attendee.Identifier = addr
		}
		if name, ok := emailAddress["name"].(string); ok {
			attendee.DisplayName = name
		}
	}

	if status, ok := raw["status"].(map[string]interface{}); ok {
		if response, ok := status["response"].(string); ok {
			switch response {
			case "accepted":
				attendee.RSVP = "ACCEPTED"
			case "declined":
				attendee.RSVP = "DECLINED"
			case "tentativelyAccepted":
				attendee.RSVP = "TENTATIVE"
			default:
				attendee.RSVP = "NEEDS_ACTION"
			}
		}
	}

	if t, ok := raw["type"].(string); ok && t == "required" {
		// Required attendee
	}

	return attendee
}

func parseGraphEventTime(raw map[string]interface{}) time.Time {
	// Check for date-only (all-day events)
	if date, ok := raw["date"].(string); ok {
		t, _ := time.Parse("2006-01-02", date)
		return t
	}
	// Check for dateTime
	if dateTime, ok := raw["dateTime"].(string); ok {
		return parseGraphTime(dateTime)
	}
	return time.Now()
}

func parseGraphTime(str string) time.Time {
	// Try RFC3339 format first
	if t, err := time.Parse(time.RFC3339, str); err == nil {
		return t
	}
	// Try without timezone
	if t, err := time.Parse("2006-01-02T15:04:05", str); err == nil {
		return t
	}
	return time.Now()
}

func buildRecurrenceRule(pattern map[string]interface{}) string {
	// Simplified recurrence rule builder
	// In production, this would build proper RRULE strings
	type_, _ := pattern["type"].(string)
	interval, _ := pattern["interval"].(float64)

	if type_ != "" {
		return type_ + ";INTERVAL=" + string(rune(int(interval)))
	}
	return ""
}
