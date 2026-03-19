package outlook

import (
	"context"
	"fmt"
	"time"

	"ondapile/internal/adapter"
	"ondapile/internal/model"
)

// ListCalendars returns calendars from Outlook.
func (a *OutlookAdapter) ListCalendars(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Calendar], error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	path := "/me/calendars"

	// Add query parameters
	params := ""
	limit := opts.Limit
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	params = fmt.Sprintf("$top=%d", limit)

	if opts.Cursor != "" {
		params = fmt.Sprintf("%s&$skiptoken=%s", params, opts.Cursor)
	}

	if params != "" {
		path = path + "?" + params
	}

	var result map[string]interface{}
	if err := graphGet(client, path, &result); err != nil {
		return nil, fmt.Errorf("failed to list calendars: %w", err)
	}

	calendars, nextLink := normalizeGraphCalendarList(result, accountID)
	hasMore := nextLink != ""

	return model.NewPaginatedList(calendars, nextLink, hasMore), nil
}

// GetCalendar gets a specific calendar from Outlook.
func (a *OutlookAdapter) GetCalendar(ctx context.Context, accountID string, calendarID string) (*model.Calendar, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	path := fmt.Sprintf("/me/calendars/%s", calendarID)

	var result map[string]interface{}
	if err := graphGet(client, path, &result); err != nil {
		return nil, fmt.Errorf("failed to get calendar: %w", err)
	}

	calendar := normalizeGraphCalendar(result, accountID)
	return calendar, nil
}

// ListEvents returns events from a specific calendar.
func (a *OutlookAdapter) ListEvents(ctx context.Context, accountID string, calendarID string, opts adapter.ListOpts) (*model.PaginatedList[model.CalendarEvent], error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	var path string
	if calendarID == "" || calendarID == "primary" {
		path = "/me/calendar/events"
	} else {
		path = fmt.Sprintf("/me/calendars/%s/events", calendarID)
	}

	// Build query parameters
	params := ""
	limit := opts.Limit
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	params = fmt.Sprintf("$top=%d", limit)

	// Add date filters
	if opts.After != nil {
		params = fmt.Sprintf("%s&startDateTime=%s", params, opts.After.Format(time.RFC3339))
	}
	if opts.Before != nil {
		params = fmt.Sprintf("%s&endDateTime=%s", params, opts.Before.Format(time.RFC3339))
	}

	if opts.Cursor != "" {
		params = fmt.Sprintf("%s&$skiptoken=%s", params, opts.Cursor)
	}

	// Add orderby
	params = fmt.Sprintf("%s&$orderby=start/dateTime asc", params)

	if params != "" {
		path = path + "?" + params
	}

	var result map[string]interface{}
	if err := graphGet(client, path, &result); err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	events, nextLink := normalizeGraphEventList(result, accountID, calendarID)
	hasMore := nextLink != ""

	return model.NewPaginatedList(events, nextLink, hasMore), nil
}

// GetEvent gets a specific event from Outlook.
func (a *OutlookAdapter) GetEvent(ctx context.Context, accountID string, calendarID string, eventID string) (*model.CalendarEvent, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	var path string
	if calendarID == "" || calendarID == "primary" {
		path = fmt.Sprintf("/me/calendar/events/%s", eventID)
	} else {
		path = fmt.Sprintf("/me/calendars/%s/events/%s", calendarID, eventID)
	}

	var result map[string]interface{}
	if err := graphGet(client, path, &result); err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	event := normalizeGraphEvent(result, accountID, calendarID)
	return event, nil
}

// CreateEvent creates a new event in Outlook.
func (a *OutlookAdapter) CreateEvent(ctx context.Context, accountID string, calendarID string, req adapter.CreateEventRequest) (*model.CalendarEvent, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	body := buildGraphEventBody(req)

	var path string
	if calendarID == "" || calendarID == "primary" {
		path = "/me/calendar/events"
	} else {
		path = fmt.Sprintf("/me/calendars/%s/events", calendarID)
	}

	var result map[string]interface{}
	if err := graphPost(client, path, body, &result); err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	event := normalizeGraphEvent(result, accountID, calendarID)

	if a.dispatcher != nil {
		a.dispatcher.Dispatch(ctx, model.EventCalendarEventCreated, event)
	}

	return event, nil
}

// UpdateEvent updates an existing event in Outlook.
func (a *OutlookAdapter) UpdateEvent(ctx context.Context, accountID string, calendarID string, eventID string, req adapter.UpdateEventRequest) (*model.CalendarEvent, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	body := buildGraphEventUpdateBody(req)

	var path string
	if calendarID == "" || calendarID == "primary" {
		path = fmt.Sprintf("/me/calendar/events/%s", eventID)
	} else {
		path = fmt.Sprintf("/me/calendars/%s/events/%s", calendarID, eventID)
	}

	var result map[string]interface{}
	if err := graphPatch(client, path, body, &result); err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	event := normalizeGraphEvent(result, accountID, calendarID)

	if a.dispatcher != nil {
		a.dispatcher.Dispatch(ctx, model.EventCalendarEventUpdated, event)
	}

	return event, nil
}

// DeleteEvent deletes an event from Outlook.
func (a *OutlookAdapter) DeleteEvent(ctx context.Context, accountID string, calendarID string, eventID string) error {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	var path string
	if calendarID == "" || calendarID == "primary" {
		path = fmt.Sprintf("/me/calendar/events/%s", eventID)
	} else {
		path = fmt.Sprintf("/me/calendars/%s/events/%s", calendarID, eventID)
	}

	if err := graphDelete(client, path); err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	if a.dispatcher != nil {
		a.dispatcher.Dispatch(ctx, model.EventCalendarEventDeleted, map[string]string{
			"account_id":  accountID,
			"calendar_id": calendarID,
			"event_id":    eventID,
		})
	}

	return nil
}

// Helper functions

func buildGraphEventBody(req adapter.CreateEventRequest) map[string]interface{} {
	body := map[string]interface{}{
		"subject": req.Title,
		"body": map[string]interface{}{
			"contentType": "text",
			"content":     req.Description,
		},
		"location": map[string]interface{}{
			"displayName": req.Location,
		},
	}

	// Set start time
	if req.AllDay {
		body["isAllDay"] = true
		body["start"] = map[string]interface{}{
			"dateTime": req.StartAt.Format("2006-01-02T00:00:00"),
			"timeZone": "UTC",
		}
		body["end"] = map[string]interface{}{
			"dateTime": req.EndAt.Format("2006-01-02T00:00:00"),
			"timeZone": "UTC",
		}
	} else {
		body["start"] = map[string]interface{}{
			"dateTime": req.StartAt.Format(time.RFC3339),
			"timeZone": "UTC",
		}
		body["end"] = map[string]interface{}{
			"dateTime": req.EndAt.Format(time.RFC3339),
			"timeZone": "UTC",
		}
	}

	// Add attendees
	if len(req.Attendees) > 0 {
		attendees := make([]map[string]interface{}, len(req.Attendees))
		for i, att := range req.Attendees {
			attendee := map[string]interface{}{
				"emailAddress": map[string]interface{}{
					"address": att.Identifier,
				},
			}
			if att.DisplayName != "" {
				attendee["emailAddress"].(map[string]interface{})["name"] = att.DisplayName
			}
			attendees[i] = attendee
		}
		body["attendees"] = attendees
	}

	// Add online meeting if requested
	if req.Conference != nil && req.Conference.AutoCreate {
		body["isOnlineMeeting"] = true
		body["onlineMeetingProvider"] = "teamsForBusiness"
	}

	return body
}

func buildGraphEventUpdateBody(req adapter.UpdateEventRequest) map[string]interface{} {
	body := make(map[string]interface{})

	if req.Title != nil {
		body["subject"] = *req.Title
	}
	if req.Description != nil {
		body["body"] = map[string]interface{}{
			"contentType": "text",
			"content":     *req.Description,
		}
	}
	if req.Location != nil {
		body["location"] = map[string]interface{}{
			"displayName": *req.Location,
		}
	}
	if req.StartAt != nil {
		body["start"] = map[string]interface{}{
			"dateTime": req.StartAt.Format(time.RFC3339),
			"timeZone": "UTC",
		}
	}
	if req.EndAt != nil {
		body["end"] = map[string]interface{}{
			"dateTime": req.EndAt.Format(time.RFC3339),
			"timeZone": "UTC",
		}
	}
	if req.AllDay != nil {
		body["isAllDay"] = *req.AllDay
	}

	if len(req.Attendees) > 0 {
		attendees := make([]map[string]interface{}, len(req.Attendees))
		for i, att := range req.Attendees {
			attendee := map[string]interface{}{
				"emailAddress": map[string]interface{}{
					"address": att.Identifier,
				},
			}
			if att.DisplayName != "" {
				attendee["emailAddress"].(map[string]interface{})["name"] = att.DisplayName
			}
			attendees[i] = attendee
		}
		body["attendees"] = attendees
	}

	return body
}
