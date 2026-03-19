package gcal

import (
	"time"

	"ondapile/internal/model"

	"github.com/google/uuid"
)

// normalizeCalendar converts a Google Calendar API response to a Calendar model.
func normalizeCalendar(raw map[string]interface{}, accountID string) *model.Calendar {
	calendar := &model.Calendar{
		Object:    "calendar",
		ID:        "cal_" + uuid.New().String(),
		AccountID: accountID,
		Provider:  "GOOGLE_CALENDAR",
		Metadata:  raw,
	}

	// Extract calendar ID
	if id, ok := raw["id"].(string); ok {
		calendar.ProviderID = id
	}

	// Extract name/summary
	if summary, ok := raw["summary"].(string); ok {
		calendar.Name = summary
	}

	// Extract description
	if desc, ok := raw["description"].(string); ok {
		// Store in metadata if needed
		_ = desc
	}

	// Extract color
	if colorId, ok := raw["colorId"].(string); ok {
		calendar.Color = colorId
	}

	// Check if primary
	if primary, ok := raw["primary"].(bool); ok {
		calendar.IsPrimary = primary
	}

	// Check access role for read-only
	if accessRole, ok := raw["accessRole"].(string); ok {
		calendar.IsReadOnly = accessRole == "reader" || accessRole == "freeBusyReader"
	}

	// Extract timezone
	if tz, ok := raw["timeZone"].(string); ok {
		calendar.TimeZone = tz
	}

	// Extract timestamps
	calendar.CreatedAt = time.Now()
	calendar.UpdatedAt = time.Now()

	return calendar
}

// normalizeCalendarList converts a Google Calendar API calendar list response to a slice of Calendar models.
func normalizeCalendarList(raw map[string]interface{}, accountID string) ([]model.Calendar, string) {
	var calendars []model.Calendar
	nextPageToken := ""

	if items, ok := raw["items"].([]interface{}); ok {
		for _, item := range items {
			if itemMap, ok := item.(map[string]interface{}); ok {
				calendar := normalizeCalendar(itemMap, accountID)
				calendars = append(calendars, *calendar)
			}
		}
	}

	// Extract next page token
	if token, ok := raw["nextPageToken"].(string); ok {
		nextPageToken = token
	}

	return calendars, nextPageToken
}

// normalizeEvent converts a Google Calendar API event response to a CalendarEvent model.
func normalizeEvent(raw map[string]interface{}, accountID string, calendarID string) *model.CalendarEvent {
	event := &model.CalendarEvent{
		Object:     "calendar_event",
		ID:         "evt_" + uuid.New().String(),
		AccountID:  accountID,
		CalendarID: calendarID,
		Provider:   "GOOGLE_CALENDAR",
		Metadata:   raw,
		Attendees:  []model.CalendarAttendee{},
		Reminders:  []model.Reminder{},
	}

	// Extract event ID
	if id, ok := raw["id"].(string); ok {
		event.ProviderID = id
	}

	// Extract title/summary
	if summary, ok := raw["summary"].(string); ok {
		event.Title = summary
	}

	// Extract description
	if desc, ok := raw["description"].(string); ok {
		event.Description = desc
	}

	// Extract location
	if loc, ok := raw["location"].(string); ok {
		event.Location = loc
	}

	// Extract status
	if status, ok := raw["status"].(string); ok {
		switch status {
		case "confirmed":
			event.Status = model.EventStatusConfirmed
		case "tentative":
			event.Status = model.EventStatusTentative
		case "cancelled":
			event.Status = model.EventStatusCancelled
		default:
			event.Status = model.EventStatusConfirmed
		}
	} else {
		event.Status = model.EventStatusConfirmed
	}

	// Extract start/end times
	event.StartAt, event.EndAt, event.AllDay = parseEventTimes(raw)

	// Extract attendees
	if attendees, ok := raw["attendees"].([]interface{}); ok {
		for _, att := range attendees {
			if attMap, ok := att.(map[string]interface{}); ok {
				attendee := parseAttendee(attMap)
				event.Attendees = append(event.Attendees, attendee)
			}
		}
	}

	// Extract conference URL (Google Meet)
	if conferenceData, ok := raw["conferenceData"].(map[string]interface{}); ok {
		if entryPoints, ok := conferenceData["entryPoints"].([]interface{}); ok && len(entryPoints) > 0 {
			if firstEntry, ok := entryPoints[0].(map[string]interface{}); ok {
				if uri, ok := firstEntry["uri"].(string); ok {
					event.ConferenceURL = uri
				}
			}
		}
	}

	// Extract recurrence
	if recurrence, ok := raw["recurrence"].([]interface{}); ok && len(recurrence) > 0 {
		if rrule, ok := recurrence[0].(string); ok {
			event.Recurrence = &rrule
		}
	}

	// Extract reminders
	if reminders, ok := raw["reminders"].(map[string]interface{}); ok {
		if overrides, ok := reminders["overrides"].([]interface{}); ok {
			for _, override := range overrides {
				if overrideMap, ok := override.(map[string]interface{}); ok {
					method, _ := overrideMap["method"].(string)
					minutes, _ := overrideMap["minutes"].(float64)

					event.Reminders = append(event.Reminders, model.Reminder{
						Method:        method,
						MinutesBefore: int(minutes),
					})
				}
			}
		}
	}

	// Extract timestamps
	event.CreatedAt = parseTime(raw["created"])
	event.UpdatedAt = parseTime(raw["updated"])

	return event
}

// normalizeEventList converts a Google Calendar API events list response to a slice of CalendarEvent models.
func normalizeEventList(raw map[string]interface{}, accountID string, calendarID string) ([]model.CalendarEvent, string) {
	var events []model.CalendarEvent
	nextPageToken := ""

	if items, ok := raw["items"].([]interface{}); ok {
		for _, item := range items {
			if itemMap, ok := item.(map[string]interface{}); ok {
				event := normalizeEvent(itemMap, accountID, calendarID)
				events = append(events, *event)
			}
		}
	}

	// Extract next page token
	if token, ok := raw["nextPageToken"].(string); ok {
		nextPageToken = token
	}

	return events, nextPageToken
}

// parseEventTimes extracts start/end times from an event.
func parseEventTimes(raw map[string]interface{}) (start, end time.Time, allDay bool) {
	// Check for all-day event (date field)
	if startData, ok := raw["start"].(map[string]interface{}); ok {
		if date, ok := startData["date"].(string); ok {
			// All-day event
			allDay = true
			start, _ = time.Parse("2006-01-02", date)

			if endData, ok := raw["end"].(map[string]interface{}); ok {
				if endDate, ok := endData["date"].(string); ok {
					end, _ = time.Parse("2006-01-02", endDate)
				}
			}
		} else if dateTime, ok := startData["dateTime"].(string); ok {
			// Timed event
			start = parseTimeString(dateTime)

			if endData, ok := raw["end"].(map[string]interface{}); ok {
				if endDateTime, ok := endData["dateTime"].(string); ok {
					end = parseTimeString(endDateTime)
				}
			}
		}
	}

	return start, end, allDay
}

// parseAttendee converts a Google Calendar attendee to a CalendarAttendee model.
func parseAttendee(raw map[string]interface{}) model.CalendarAttendee {
	attendee := model.CalendarAttendee{}

	if email, ok := raw["email"].(string); ok {
		attendee.Identifier = email
	}

	if displayName, ok := raw["displayName"].(string); ok {
		attendee.DisplayName = displayName
	}

	if responseStatus, ok := raw["responseStatus"].(string); ok {
		switch responseStatus {
		case "accepted":
			attendee.RSVP = "ACCEPTED"
		case "declined":
			attendee.RSVP = "DECLINED"
		case "tentative":
			attendee.RSVP = "TENTATIVE"
		default:
			attendee.RSVP = "NEEDS_ACTION"
		}
	}

	if organizer, ok := raw["organizer"].(bool); ok {
		attendee.Organizer = organizer
	}

	return attendee
}

// parseTime parses a time string from the API.
func parseTime(value interface{}) time.Time {
	if str, ok := value.(string); ok {
		return parseTimeString(str)
	}
	return time.Now()
}

// parseTimeString parses various time formats.
func parseTimeString(str string) time.Time {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, str); err == nil {
			return t
		}
	}

	return time.Now()
}

// getString safely extracts a string value from a map.
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
