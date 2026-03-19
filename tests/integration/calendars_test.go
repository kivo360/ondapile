//go:build phase2
// +build phase2

package integration

// ============================================================================
// PHASE 2 TESTS — Calendar endpoints
// These tests will NOT compile until the Phase 2 branch merges, which adds:
//   - internal/calendar package
//   - internal/api/calendars.go
//   - Calendar-related model types
//   - Calendar-related adapter interface methods
//   - Calendar-related MockProvider fields
//
// Once Phase 2 merges, remove the "// +build phase2" directive above to
// enable these tests.
// ============================================================================

/*
import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ondapile/internal/adapter"
	"ondapile/internal/model"
	"ondapile/internal/store"
)

// ==================== Calendar CRUD via API ====================

func TestCalendarListCalendars(t *testing.T) {
	router, _ := setupTest(t)

	// Create account
	body := `{"provider":"MOCK","identifier":"cal-list","credentials":{}}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)
	var acc map[string]any
	json.Unmarshal(resp.Body.Bytes(), &acc)
	accountID := acc["id"].(string)

	// Register mock with calendar funcs
	mock := &MockProvider{
		ListCalendarsFunc: func(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Calendar], error) {
			return &model.PaginatedList[model.Calendar]{
				Object: "list",
				Items: []model.Calendar{
					{Object: "calendar", ID: "cal_1", AccountID: accountID, Provider: "MOCK", Name: "Work Calendar"},
					{Object: "calendar", ID: "cal_2", AccountID: accountID, Provider: "MOCK", Name: "Personal Calendar"},
				},
				HasMore: false,
			}, nil
		},
	}
	adapter.Register(mock)

	resp = apiRequest(t, router, "GET", "/api/v1/calendars?account_id="+accountID, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.PaginatedList[map[string]any]
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Len(t, result.Items, 2)
}

func TestCalendarGetCalendar(t *testing.T) {
	router, _ := setupTest(t)

	body := `{"provider":"MOCK","identifier":"cal-get","credentials":{}}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)
	var acc map[string]any
	json.Unmarshal(resp.Body.Bytes(), &acc)
	accountID := acc["id"].(string)

	mock := &MockProvider{
		GetCalendarFunc: func(ctx context.Context, accID string, calID string) (*model.Calendar, error) {
			return &model.Calendar{
				Object:    "calendar",
				ID:        calID,
				AccountID: accID,
				Provider:  "MOCK",
				Name:      "Test Calendar",
			}, nil
		},
	}
	adapter.Register(mock)

	resp = apiRequest(t, router, "GET", "/api/v1/calendars/cal_1?account_id="+accountID, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var cal map[string]any
	json.Unmarshal(resp.Body.Bytes(), &cal)
	assert.Equal(t, "calendar", cal["object"])
	assert.Equal(t, "Test Calendar", cal["name"])
}

func TestCalendarGetCalendarNotFound(t *testing.T) {
	router, _ := setupTest(t)

	body := `{"provider":"MOCK","identifier":"cal-404","credentials":{}}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)
	var acc map[string]any
	json.Unmarshal(resp.Body.Bytes(), &acc)
	accountID := acc["id"].(string)

	mock := &MockProvider{
		GetCalendarFunc: func(ctx context.Context, accID string, calID string) (*model.Calendar, error) {
			return nil, nil
		},
	}
	adapter.Register(mock)

	resp = apiRequest(t, router, "GET", "/api/v1/calendars/nonexistent?account_id="+accountID, nil, testAPIKey)
	requireStatus(t, resp, http.StatusNotFound)
}

// ==================== Calendar Event CRUD ====================

func TestCalendarListEvents(t *testing.T) {
	router, _ := setupTest(t)

	body := `{"provider":"MOCK","identifier":"evt-list","credentials":{}}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)
	var acc map[string]any
	json.Unmarshal(resp.Body.Bytes(), &acc)
	accountID := acc["id"].(string)

	now := time.Now().UTC()
	mock := &MockProvider{
		ListEventsFunc: func(ctx context.Context, accID string, calID string, opts adapter.ListOpts) (*model.PaginatedList[model.CalendarEvent], error) {
			return &model.PaginatedList[model.CalendarEvent]{
				Object: "list",
				Items: []model.CalendarEvent{
					{Object: "calendar_event", ID: "evt_1", Title: "Meeting A", StartAt: now, EndAt: now.Add(time.Hour)},
					{Object: "calendar_event", ID: "evt_2", Title: "Meeting B", StartAt: now.Add(2 * time.Hour), EndAt: now.Add(3 * time.Hour)},
				},
				HasMore: false,
			}, nil
		},
	}
	adapter.Register(mock)

	resp = apiRequest(t, router, "GET", "/api/v1/calendars/cal_1/events?account_id="+accountID, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.PaginatedList[map[string]any]
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Len(t, result.Items, 2)
}

func TestCalendarListEventsWithBeforeAfterFilters(t *testing.T) {
	router, _ := setupTest(t)

	body := `{"provider":"MOCK","identifier":"evt-filter","credentials":{}}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)
	var acc map[string]any
	json.Unmarshal(resp.Body.Bytes(), &acc)
	accountID := acc["id"].(string)

	var capturedOpts adapter.ListOpts
	mock := &MockProvider{
		ListEventsFunc: func(ctx context.Context, accID string, calID string, opts adapter.ListOpts) (*model.PaginatedList[model.CalendarEvent], error) {
			capturedOpts = opts
			return &model.PaginatedList[model.CalendarEvent]{Object: "list", Items: []model.CalendarEvent{}, HasMore: false}, nil
		},
	}
	adapter.Register(mock)

	before := "2026-04-01T00:00:00Z"
	after := "2026-03-01T00:00:00Z"
	resp = apiRequest(t, router, "GET",
		fmt.Sprintf("/api/v1/calendars/cal_1/events?account_id=%s&before=%s&after=%s", accountID, before, after),
		nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	require.NotNil(t, capturedOpts.Before, "before filter should be passed to provider")
	require.NotNil(t, capturedOpts.After, "after filter should be passed to provider")
}

func TestCalendarListEventsPagination(t *testing.T) {
	router, _ := setupTest(t)

	body := `{"provider":"MOCK","identifier":"evt-page","credentials":{}}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)
	var acc map[string]any
	json.Unmarshal(resp.Body.Bytes(), &acc)
	accountID := acc["id"].(string)

	mock := &MockProvider{
		ListEventsFunc: func(ctx context.Context, accID string, calID string, opts adapter.ListOpts) (*model.PaginatedList[model.CalendarEvent], error) {
			assert.Equal(t, 5, opts.Limit)
			return &model.PaginatedList[model.CalendarEvent]{
				Object:  "list",
				Items:   []model.CalendarEvent{},
				Cursor:  "next-page-cursor",
				HasMore: true,
			}, nil
		},
	}
	adapter.Register(mock)

	resp = apiRequest(t, router, "GET",
		fmt.Sprintf("/api/v1/calendars/cal_1/events?account_id=%s&limit=5", accountID),
		nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.PaginatedList[map[string]any]
	json.Unmarshal(resp.Body.Bytes(), &result)
	assert.True(t, result.HasMore)
	assert.NotEmpty(t, result.Cursor)
}

func TestCalendarGetEvent(t *testing.T) {
	router, _ := setupTest(t)

	body := `{"provider":"MOCK","identifier":"evt-get","credentials":{}}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)
	var acc map[string]any
	json.Unmarshal(resp.Body.Bytes(), &acc)
	accountID := acc["id"].(string)

	now := time.Now().UTC()
	mock := &MockProvider{
		GetEventFunc: func(ctx context.Context, accID string, calID string, eventID string) (*model.CalendarEvent, error) {
			return &model.CalendarEvent{
				Object:      "calendar_event",
				ID:          eventID,
				AccountID:   accID,
				CalendarID:  calID,
				Provider:    "MOCK",
				Title:       "Strategy Meeting",
				Description: "Quarterly review",
				Location:    "Room B",
				StartAt:     now,
				EndAt:       now.Add(time.Hour),
				AllDay:      false,
				Status:      "CONFIRMED",
			}, nil
		},
	}
	adapter.Register(mock)

	resp = apiRequest(t, router, "GET",
		fmt.Sprintf("/api/v1/calendars/cal_1/events/evt_1?account_id=%s", accountID),
		nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var evt map[string]any
	json.Unmarshal(resp.Body.Bytes(), &evt)
	assert.Equal(t, "calendar_event", evt["object"])
	assert.Equal(t, "Strategy Meeting", evt["title"])
	assert.Equal(t, "Quarterly review", evt["description"])
}

func TestCalendarGetEventNotFound(t *testing.T) {
	router, _ := setupTest(t)

	body := `{"provider":"MOCK","identifier":"evt-get-404","credentials":{}}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)
	var acc map[string]any
	json.Unmarshal(resp.Body.Bytes(), &acc)
	accountID := acc["id"].(string)

	mock := &MockProvider{
		GetEventFunc: func(ctx context.Context, accID string, calID string, eventID string) (*model.CalendarEvent, error) {
			return nil, nil
		},
	}
	adapter.Register(mock)

	resp = apiRequest(t, router, "GET",
		fmt.Sprintf("/api/v1/calendars/cal_1/events/nonexistent?account_id=%s", accountID),
		nil, testAPIKey)
	requireStatus(t, resp, http.StatusNotFound)
}

func TestCalendarCreateEvent(t *testing.T) {
	router, _ := setupTest(t)

	body := `{"provider":"MOCK","identifier":"evt-create","credentials":{}}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)
	var acc map[string]any
	json.Unmarshal(resp.Body.Bytes(), &acc)
	accountID := acc["id"].(string)

	now := time.Now().UTC()
	mock := &MockProvider{
		CreateEventFunc: func(ctx context.Context, accID string, calID string, evt adapter.CreateEventRequest) (*model.CalendarEvent, error) {
			return &model.CalendarEvent{
				Object:     "calendar_event",
				ID:         "evt_new",
				AccountID:  accID,
				CalendarID: calID,
				Provider:   "MOCK",
				Title:      evt.Title,
				StartAt:    now,
				EndAt:      now.Add(time.Hour),
			}, nil
		},
	}
	adapter.Register(mock)

	eventBody := fmt.Sprintf(`{
		"account_id": "%s",
		"title": "New Meeting",
		"start_at": "%s",
		"end_at": "%s"
	}`, accountID, now.Format(time.RFC3339), now.Add(time.Hour).Format(time.RFC3339))

	resp = apiRequest(t, router, "POST", "/api/v1/calendars/cal_1/events", []byte(eventBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var evt map[string]any
	json.Unmarshal(resp.Body.Bytes(), &evt)
	assert.Equal(t, "calendar_event", evt["object"])
	assert.Equal(t, "New Meeting", evt["title"])
}

func TestCalendarCreateEventValidation(t *testing.T) {
	router, _ := setupTest(t)

	body := `{"provider":"MOCK","identifier":"evt-validate","credentials":{}}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	// Missing required fields
	resp = apiRequest(t, router, "POST", "/api/v1/calendars/cal_1/events", []byte(`{}`), testAPIKey)
	requireStatus(t, resp, http.StatusUnprocessableEntity)
}

func TestCalendarUpdateEvent(t *testing.T) {
	router, _ := setupTest(t)

	body := `{"provider":"MOCK","identifier":"evt-update","credentials":{}}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)
	var acc map[string]any
	json.Unmarshal(resp.Body.Bytes(), &acc)
	accountID := acc["id"].(string)

	now := time.Now().UTC()
	mock := &MockProvider{
		UpdateEventFunc: func(ctx context.Context, accID string, calID string, eventID string, evt adapter.UpdateEventRequest) (*model.CalendarEvent, error) {
			return &model.CalendarEvent{
				Object:     "calendar_event",
				ID:         eventID,
				AccountID:  accID,
				CalendarID: calID,
				Provider:   "MOCK",
				Title:      *evt.Title,
				StartAt:    now,
				EndAt:      now.Add(2 * time.Hour),
			}, nil
		},
	}
	adapter.Register(mock)

	updateBody := fmt.Sprintf(`{"account_id":"%s","title":"Updated Meeting"}`, accountID)
	resp = apiRequest(t, router, "PATCH",
		"/api/v1/calendars/cal_1/events/evt_1", []byte(updateBody), testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var evt map[string]any
	json.Unmarshal(resp.Body.Bytes(), &evt)
	assert.Equal(t, "Updated Meeting", evt["title"])
}

func TestCalendarDeleteEvent(t *testing.T) {
	router, _ := setupTest(t)

	body := `{"provider":"MOCK","identifier":"evt-delete","credentials":{}}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)
	var acc map[string]any
	json.Unmarshal(resp.Body.Bytes(), &acc)
	accountID := acc["id"].(string)

	mock := &MockProvider{
		DeleteEventFunc: func(ctx context.Context, accID string, calID string, eventID string) error {
			return nil
		},
	}
	adapter.Register(mock)

	resp = apiRequest(t, router, "DELETE",
		fmt.Sprintf("/api/v1/calendars/cal_1/events/evt_1?account_id=%s", accountID),
		nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)
}

// ==================== Calendar Store Direct Tests ====================

func TestCalendarStoreCreateAndGet(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)

	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider: "MOCK", Name: "Calendar Store Test", Identifier: "cal-store",
		Status: string(model.StatusOperational),
	})
	require.NoError(t, err)

	// Direct calendar store tests would go here once the calendar store package exists
	// For now, verify the account was created properly as a baseline
	assert.NotEmpty(t, account.ID)
}

func TestCalendarStoreListByAccount(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	// Once calendar store exists:
	// Create account, store multiple calendars, list them, verify count and ordering
	_ = ctx
}

func TestCalendarStoreEventBeforeAfterFilter(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	// Once calendar event store exists:
	// Create events at different times, filter by before/after, verify correct subset
	_ = ctx
}

func TestCalendarStoreEventPagination(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	// Verify cursor-based pagination for calendar events
	_ = ctx
}

func TestCalendarStoreEventCRUD(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	// Create, Read, Update, Delete a calendar event via store layer
	_ = ctx
}

func TestCalendarRequiresAccountID(t *testing.T) {
	router, _ := setupTest(t)

	resp := apiRequest(t, router, "GET", "/api/v1/calendars", nil, testAPIKey)
	requireStatus(t, resp, http.StatusUnprocessableEntity)
}
*/
