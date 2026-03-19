//go:build phase2
// +build phase2

package integration

// ============================================================================
// PHASE 2 MOCK PROVIDER EXTENSIONS
//
// When Phase 2 merges, add these fields to MockProvider in mock_provider_test.go
// and uncomment the method implementations below.
//
// Fields to add to MockProvider struct:
//
//   // Calendar (Phase 2)
//   ListCalendarsFunc  func(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Calendar], error)
//   GetCalendarFunc    func(ctx context.Context, accountID string, calendarID string) (*model.Calendar, error)
//   ListEventsFunc     func(ctx context.Context, accountID string, calendarID string, opts adapter.ListOpts) (*model.PaginatedList[model.CalendarEvent], error)
//   GetEventFunc       func(ctx context.Context, accountID string, calendarID string, eventID string) (*model.CalendarEvent, error)
//   CreateEventFunc    func(ctx context.Context, accountID string, calendarID string, req adapter.CreateEventRequest) (*model.CalendarEvent, error)
//   UpdateEventFunc    func(ctx context.Context, accountID string, calendarID string, eventID string, req adapter.UpdateEventRequest) (*model.CalendarEvent, error)
//   DeleteEventFunc    func(ctx context.Context, accountID string, calendarID string, eventID string) error
//
//   // OAuth (Phase 2)
//   SupportsOAuthFunc       func() bool
//   GetOAuthURLFunc         func(ctx context.Context, accountID string, redirectURI string) (string, error)
//   HandleOAuthCallbackFunc func(ctx context.Context, accountID string, code string, redirectURI string) (*model.Account, error)
//
// ============================================================================

/*
import (
	"context"
	"errors"
	"time"

	"ondapile/internal/adapter"
	"ondapile/internal/model"
)

var ErrNotSupported = errors.New("operation not supported by this provider")

// ==================== Calendar Methods ====================

func (m *MockProvider) ListCalendars(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Calendar], error) {
	if m.ListCalendarsFunc != nil {
		return m.ListCalendarsFunc(ctx, accountID, opts)
	}
	return &model.PaginatedList[model.Calendar]{
		Object: "list", Items: []model.Calendar{}, HasMore: false,
	}, nil
}

func (m *MockProvider) GetCalendar(ctx context.Context, accountID string, calendarID string) (*model.Calendar, error) {
	if m.GetCalendarFunc != nil {
		return m.GetCalendarFunc(ctx, accountID, calendarID)
	}
	return nil, ErrNotSupported
}

func (m *MockProvider) ListEvents(ctx context.Context, accountID string, calendarID string, opts adapter.ListOpts) (*model.PaginatedList[model.CalendarEvent], error) {
	if m.ListEventsFunc != nil {
		return m.ListEventsFunc(ctx, accountID, calendarID, opts)
	}
	return &model.PaginatedList[model.CalendarEvent]{
		Object: "list", Items: []model.CalendarEvent{}, HasMore: false,
	}, nil
}

func (m *MockProvider) GetEvent(ctx context.Context, accountID string, calendarID string, eventID string) (*model.CalendarEvent, error) {
	if m.GetEventFunc != nil {
		return m.GetEventFunc(ctx, accountID, calendarID, eventID)
	}
	return nil, ErrNotSupported
}

func (m *MockProvider) CreateEvent(ctx context.Context, accountID string, calendarID string, req adapter.CreateEventRequest) (*model.CalendarEvent, error) {
	if m.CreateEventFunc != nil {
		return m.CreateEventFunc(ctx, accountID, calendarID, req)
	}
	return &model.CalendarEvent{
		Object:     "calendar_event",
		ID:         "evt_mock_" + time.Now().Format("20060102150405"),
		AccountID:  accountID,
		CalendarID: calendarID,
		Provider:   m.Name(),
		Title:      req.Title,
		StartAt:    req.StartAt,
		EndAt:      req.EndAt,
	}, nil
}

func (m *MockProvider) UpdateEvent(ctx context.Context, accountID string, calendarID string, eventID string, req adapter.UpdateEventRequest) (*model.CalendarEvent, error) {
	if m.UpdateEventFunc != nil {
		return m.UpdateEventFunc(ctx, accountID, calendarID, eventID, req)
	}
	title := "Updated Event"
	if req.Title != nil {
		title = *req.Title
	}
	return &model.CalendarEvent{
		Object:     "calendar_event",
		ID:         eventID,
		AccountID:  accountID,
		CalendarID: calendarID,
		Provider:   m.Name(),
		Title:      title,
	}, nil
}

func (m *MockProvider) DeleteEvent(ctx context.Context, accountID string, calendarID string, eventID string) error {
	if m.DeleteEventFunc != nil {
		return m.DeleteEventFunc(ctx, accountID, calendarID, eventID)
	}
	return nil
}

// ==================== OAuth Methods ====================

func (m *MockProvider) SupportsOAuth() bool {
	if m.SupportsOAuthFunc != nil {
		return m.SupportsOAuthFunc()
	}
	return false
}

func (m *MockProvider) GetOAuthURL(ctx context.Context, accountID string, redirectURI string) (string, error) {
	if m.GetOAuthURLFunc != nil {
		return m.GetOAuthURLFunc(ctx, accountID, redirectURI)
	}
	return "", ErrNotSupported
}

func (m *MockProvider) HandleOAuthCallback(ctx context.Context, accountID string, code string, redirectURI string) (*model.Account, error) {
	if m.HandleOAuthCallbackFunc != nil {
		return m.HandleOAuthCallbackFunc(ctx, accountID, code, redirectURI)
	}
	return nil, ErrNotSupported
}
*/
