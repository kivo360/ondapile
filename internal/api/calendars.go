package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"ondapile/internal/adapter"
	"ondapile/internal/model"
	"ondapile/internal/store"
)

type CalendarHandler struct {
	store     *store.Store
	calendars *store.CalendarStore
	events    *store.CalendarEventStore
}

func NewCalendarHandler(s *store.Store) *CalendarHandler {
	return &CalendarHandler{
		store:     s,
		calendars: store.NewCalendarStore(s),
		events:    store.NewCalendarEventStore(s),
	}
}

// GET /calendars
func (h *CalendarHandler) List(c *gin.Context) {
	accountID := c.Query("account_id")
	var accountIDPtr *string
	if accountID != "" {
		accountIDPtr = &accountID
	}

	cursor := c.Query("cursor")
	limit := parseLimit(c.DefaultQuery("limit", "25"))

	calendars, nextCursor, hasMore, err := h.calendars.List(c.Request.Context(), accountIDPtr, cursor, limit)
	if err != nil {
		Internal(c, "Failed to list calendars")
		return
	}

	// If DB is empty and account_id specified, try fetching from provider
	if len(calendars) == 0 && accountID != "" {
		accounts := store.NewAccountStore(h.store)
		account, aErr := accounts.GetByID(c.Request.Context(), accountID)
		if aErr == nil && account != nil {
			// Try calendar-capable providers for this account
			for _, provName := range []string{"GOOGLE_CALENDAR", "OUTLOOK", account.Provider} {
				prov, pErr := adapter.Get(provName)
				if pErr != nil {
					continue
				}
				result, lErr := prov.ListCalendars(c.Request.Context(), accountID, adapter.ListOpts{Limit: limit})
				if lErr != nil {
					slog.Warn("failed to fetch calendars from provider", "provider", provName, "error", lErr)
					continue
				}
				if result != nil && len(result.Items) > 0 {
					items := make([]any, len(result.Items))
					for i, cal := range result.Items {
						items[i] = cal
					}
					c.JSON(http.StatusOK, model.NewPaginatedList(items, result.Cursor, result.HasMore))
					return
				}
			}
		}
	}

	items := make([]any, len(calendars))
	for i, cal := range calendars {
		items[i] = cal
	}

	c.JSON(http.StatusOK, model.NewPaginatedList(items, nextCursor, hasMore))
}

// GET /calendars/:id
func (h *CalendarHandler) Get(c *gin.Context) {
	id := c.Param("id")
	calendar, err := h.calendars.GetByID(c.Request.Context(), id)
	if err != nil || calendar == nil {
		NotFound(c, "Calendar not found")
		return
	}

	c.JSON(http.StatusOK, calendar)
}

// GET /calendars/:id/events
func (h *CalendarHandler) ListEvents(c *gin.Context) {
	calendarID := c.Param("id")

	// Verify calendar exists
	calendar, err := h.calendars.GetByID(c.Request.Context(), calendarID)
	if err != nil || calendar == nil {
		NotFound(c, "Calendar not found")
		return
	}

	cursor := c.Query("cursor")
	limit := parseLimit(c.DefaultQuery("limit", "25"))

	events, nextCursor, hasMore, err := h.events.ListByCalendar(c.Request.Context(), calendarID, cursor, limit)
	if err != nil {
		Internal(c, "Failed to list events")
		return
	}

	items := make([]any, len(events))
	for i, evt := range events {
		items[i] = evt
	}

	c.JSON(http.StatusOK, model.NewPaginatedList(items, nextCursor, hasMore))
}

// GET /calendars/:id/events/:event_id
func (h *CalendarHandler) GetEvent(c *gin.Context) {
	eventID := c.Param("event_id")
	event, err := h.events.GetByID(c.Request.Context(), eventID)
	if err != nil || event == nil {
		NotFound(c, "Event not found")
		return
	}

	c.JSON(http.StatusOK, event)
}

type createEventRequest struct {
	Title       string                   `json:"title" binding:"required"`
	Description string                   `json:"description,omitempty"`
	Location    string                   `json:"location,omitempty"`
	StartAt     string                   `json:"start_at" binding:"required"`
	EndAt       string                   `json:"end_at" binding:"required"`
	AllDay      bool                     `json:"all_day"`
	Attendees   []model.CalendarAttendee `json:"attendees,omitempty"`
	Reminders   []model.Reminder         `json:"reminders,omitempty"`
}

// POST /calendars/:id/events
func (h *CalendarHandler) CreateEvent(c *gin.Context) {
	calendarID := c.Param("id")

	// Get calendar to find account and provider
	calendar, err := h.calendars.GetByID(c.Request.Context(), calendarID)
	if err != nil || calendar == nil {
		NotFound(c, "Calendar not found")
		return
	}

	var req createEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Validation(c, err.Error())
		return
	}

	// Get provider adapter
	prov, err := adapter.Get(calendar.Provider)
	if err != nil {
		ProviderError(c, "Provider not available: "+calendar.Provider)
		return
	}

	// Parse times
	startAt, err := parseTime(req.StartAt)
	if err != nil {
		Validation(c, "Invalid start_at format")
		return
	}
	endAt, err := parseTime(req.EndAt)
	if err != nil {
		Validation(c, "Invalid end_at format")
		return
	}

	// Call provider to create event
	createReq := adapter.CreateEventRequest{
		Title:       req.Title,
		Description: req.Description,
		Location:    req.Location,
		StartAt:     startAt,
		EndAt:       endAt,
		AllDay:      req.AllDay,
		Attendees:   req.Attendees,
		Reminders:   req.Reminders,
	}

	providerEvent, err := prov.CreateEvent(c.Request.Context(), calendar.AccountID, calendarID, createReq)
	if err != nil {
		ProviderError(c, "Failed to create event: "+err.Error())
		return
	}

	// Sync to local DB
	event := &model.CalendarEvent{
		Object:        "calendar_event",
		AccountID:     calendar.AccountID,
		CalendarID:    calendarID,
		Provider:      calendar.Provider,
		ProviderID:    providerEvent.ProviderID,
		Title:         providerEvent.Title,
		Description:   providerEvent.Description,
		Location:      providerEvent.Location,
		StartAt:       providerEvent.StartAt,
		EndAt:         providerEvent.EndAt,
		AllDay:        providerEvent.AllDay,
		Status:        providerEvent.Status,
		Attendees:     providerEvent.Attendees,
		Reminders:     providerEvent.Reminders,
		ConferenceURL: providerEvent.ConferenceURL,
		Recurrence:    providerEvent.Recurrence,
		Metadata:      providerEvent.Metadata,
	}

	event, err = h.events.Create(c.Request.Context(), event)
	if err != nil {
		slog.Error("failed to store event in DB", "error", err)
		// Return provider event anyway
		c.JSON(http.StatusCreated, providerEvent)
		return
	}

	c.JSON(http.StatusCreated, event)
}

type updateEventRequest struct {
	Title       *string                  `json:"title,omitempty"`
	Description *string                  `json:"description,omitempty"`
	Location    *string                  `json:"location,omitempty"`
	StartAt     *string                  `json:"start_at,omitempty"`
	EndAt       *string                  `json:"end_at,omitempty"`
	AllDay      *bool                    `json:"all_day,omitempty"`
	Attendees   []model.CalendarAttendee `json:"attendees,omitempty"`
	Reminders   []model.Reminder         `json:"reminders,omitempty"`
}

// PATCH /calendars/:id/events/:event_id
func (h *CalendarHandler) UpdateEvent(c *gin.Context) {
	calendarID := c.Param("id")
	eventID := c.Param("event_id")

	// Get calendar
	calendar, err := h.calendars.GetByID(c.Request.Context(), calendarID)
	if err != nil || calendar == nil {
		NotFound(c, "Calendar not found")
		return
	}

	// Get event
	event, err := h.events.GetByID(c.Request.Context(), eventID)
	if err != nil || event == nil {
		NotFound(c, "Event not found")
		return
	}

	var req updateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Validation(c, err.Error())
		return
	}

	// Get provider adapter
	prov, err := adapter.Get(calendar.Provider)
	if err != nil {
		ProviderError(c, "Provider not available: "+calendar.Provider)
		return
	}

	// Build update request
	updateReq := adapter.UpdateEventRequest{
		Title:       req.Title,
		Description: req.Description,
		Location:    req.Location,
		Attendees:   req.Attendees,
		Reminders:   req.Reminders,
	}

	if req.StartAt != nil {
		t, err := parseTime(*req.StartAt)
		if err != nil {
			Validation(c, "Invalid start_at format")
			return
		}
		updateReq.StartAt = &t
	}
	if req.EndAt != nil {
		t, err := parseTime(*req.EndAt)
		if err != nil {
			Validation(c, "Invalid end_at format")
			return
		}
		updateReq.EndAt = &t
	}
	updateReq.AllDay = req.AllDay

	// Call provider to update
	providerEvent, err := prov.UpdateEvent(c.Request.Context(), calendar.AccountID, calendarID, event.ProviderID, updateReq)
	if err != nil {
		ProviderError(c, "Failed to update event: "+err.Error())
		return
	}

	// Sync to local DB
	event.Title = providerEvent.Title
	event.Description = providerEvent.Description
	event.Location = providerEvent.Location
	event.StartAt = providerEvent.StartAt
	event.EndAt = providerEvent.EndAt
	event.AllDay = providerEvent.AllDay
	event.Status = providerEvent.Status
	event.Attendees = providerEvent.Attendees
	event.Reminders = providerEvent.Reminders
	event.ConferenceURL = providerEvent.ConferenceURL
	event.Recurrence = providerEvent.Recurrence
	event.Metadata = providerEvent.Metadata

	if err := h.events.Update(c.Request.Context(), event); err != nil {
		slog.Error("failed to update event in DB", "error", err)
		c.JSON(http.StatusOK, providerEvent)
		return
	}

	c.JSON(http.StatusOK, event)
}

// DELETE /calendars/:id/events/:event_id
func (h *CalendarHandler) DeleteEvent(c *gin.Context) {
	calendarID := c.Param("id")
	eventID := c.Param("event_id")

	// Get calendar
	calendar, err := h.calendars.GetByID(c.Request.Context(), calendarID)
	if err != nil || calendar == nil {
		NotFound(c, "Calendar not found")
		return
	}

	// Get event to find provider ID
	event, err := h.events.GetByID(c.Request.Context(), eventID)
	if err != nil || event == nil {
		NotFound(c, "Event not found")
		return
	}

	// Get provider adapter
	prov, err := adapter.Get(calendar.Provider)
	if err != nil {
		ProviderError(c, "Provider not available: "+calendar.Provider)
		return
	}

	// Call provider to delete
	if err := prov.DeleteEvent(c.Request.Context(), calendar.AccountID, calendarID, event.ProviderID); err != nil {
		ProviderError(c, "Failed to delete event: "+err.Error())
		return
	}

	// Delete from local DB
	if err := h.events.Delete(c.Request.Context(), eventID); err != nil {
		slog.Error("failed to delete event from DB", "error", err)
	}

	c.JSON(http.StatusOK, gin.H{"object": "calendar_event", "id": eventID, "deleted": true})
}

func parseLimit(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 25
	}
	if n <= 0 {
		return 25
	}
	if n > 100 {
		return 100
	}
	return n
}

func parseTime(s string) (time.Time, error) {
	// Try multiple formats
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time: %s", s)
}
