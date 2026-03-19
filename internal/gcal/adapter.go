package gcal

import (
	"context"
	"fmt"
	"time"

	"ondapile/internal/adapter"
	"ondapile/internal/model"
	"ondapile/internal/oauth"
	"ondapile/internal/store"
	"ondapile/internal/webhook"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// GCalAdapter implements the adapter.Provider interface for Google Calendar.
type GCalAdapter struct {
	oauthCfg   *oauth2.Config
	tokenStore *oauth.TokenStore
	store      *store.Store
	dispatcher *webhook.Dispatcher
}

// NewAdapter creates a new Google Calendar adapter.
func NewAdapter(oauthCfg *oauth2.Config, tokenStore *oauth.TokenStore, s *store.Store, d *webhook.Dispatcher) *GCalAdapter {
	return &GCalAdapter{
		oauthCfg:   oauthCfg,
		tokenStore: tokenStore,
		store:      s,
		dispatcher: d,
	}
}

// Name returns the provider identifier.
func (a *GCalAdapter) Name() string {
	return "GOOGLE_CALENDAR"
}

// SupportsOAuth returns true as Google Calendar uses OAuth2.
func (a *GCalAdapter) SupportsOAuth() bool {
	return true
}

// GetOAuthURL generates the OAuth URL for Google Calendar authentication.
func (a *GCalAdapter) GetOAuthURL(ctx context.Context, state string) (string, error) {
	if a.oauthCfg == nil {
		return "", fmt.Errorf("OAuth config not initialized")
	}

	return a.oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

// HandleOAuthCallback exchanges the authorization code for a token.
func (a *GCalAdapter) HandleOAuthCallback(ctx context.Context, code string) (map[string]string, error) {
	if a.oauthCfg == nil {
		return nil, fmt.Errorf("OAuth config not initialized")
	}

	token, err := a.oauthCfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	creds := map[string]string{
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
		"token_type":    token.TokenType,
		"expiry":        token.Expiry.String(),
	}

	return creds, nil
}

// Connect establishes a Google Calendar connection using stored OAuth tokens.
func (a *GCalAdapter) Connect(ctx context.Context, accountID string, creds map[string]string) (*model.Account, error) {
	if creds["access_token"] == "" {
		return nil, fmt.Errorf("no access token provided")
	}

	token := &oauth2.Token{
		AccessToken:  creds["access_token"],
		RefreshToken: creds["refresh_token"],
		TokenType:    creds["token_type"],
	}

	if err := a.tokenStore.Save(ctx, accountID, a.Name(), token); err != nil {
		return nil, fmt.Errorf("failed to save token: %w", err)
	}

	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	var calList map[string]interface{}
	if err := gcalGet(client, "/users/me/calendarList", &calList); err != nil {
		return nil, fmt.Errorf("failed to verify Google Calendar connection: %w", err)
	}

	return &model.Account{
		Object:       "account",
		ID:           accountID,
		Provider:     a.Name(),
		Name:         "Google Calendar",
		Identifier:   accountID,
		Status:       model.StatusOperational,
		Capabilities: []string{"calendar", "events", "create", "update", "delete"},
	}, nil
}

// Disconnect closes the Google Calendar connection.
func (a *GCalAdapter) Disconnect(ctx context.Context, accountID string) error {
	return nil
}

// Reconnect re-establishes the Google Calendar connection.
func (a *GCalAdapter) Reconnect(ctx context.Context, accountID string) (*model.Account, error) {
	token, err := a.tokenStore.Load(ctx, accountID, a.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to get stored token: %w", err)
	}

	creds := map[string]string{
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
		"token_type":    token.TokenType,
	}

	return a.Connect(ctx, accountID, creds)
}

// Status returns the current connection status.
func (a *GCalAdapter) Status(ctx context.Context, accountID string) (model.AccountStatus, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return model.StatusInterrupted, nil
	}

	var calList map[string]interface{}
	if err := gcalGet(client, "/users/me/calendarList", &calList); err != nil {
		return model.StatusInterrupted, nil
	}

	return model.StatusOperational, nil
}

// GetAuthChallenge returns nil as Google Calendar uses OAuth.
func (a *GCalAdapter) GetAuthChallenge(ctx context.Context, accountID string) (*adapter.AuthChallenge, error) {
	return nil, nil
}

// SolveCheckpoint is not applicable for Google Calendar.
func (a *GCalAdapter) SolveCheckpoint(ctx context.Context, accountID string, solution string) error {
	return adapter.ErrNotSupported
}

// ListChats is not supported by Google Calendar.
func (a *GCalAdapter) ListChats(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Chat], error) {
	return nil, adapter.ErrNotSupported
}

// GetChat is not supported by Google Calendar.
func (a *GCalAdapter) GetChat(ctx context.Context, accountID string, chatID string) (*model.Chat, error) {
	return nil, adapter.ErrNotSupported
}

// ListMessages is not supported by Google Calendar.
func (a *GCalAdapter) ListMessages(ctx context.Context, accountID string, chatID string, opts adapter.ListOpts) (*model.PaginatedList[model.Message], error) {
	return nil, adapter.ErrNotSupported
}

// SendMessage is not supported by Google Calendar.
func (a *GCalAdapter) SendMessage(ctx context.Context, accountID string, chatID string, msg adapter.SendMessageRequest) (*model.Message, error) {
	return nil, adapter.ErrNotSupported
}

// StartChat is not supported by Google Calendar.
func (a *GCalAdapter) StartChat(ctx context.Context, accountID string, req adapter.StartChatRequest) (*model.Chat, error) {
	return nil, adapter.ErrNotSupported
}

// ListAttendees is not supported by Google Calendar.
func (a *GCalAdapter) ListAttendees(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Attendee], error) {
	return nil, adapter.ErrNotSupported
}

// GetAttendee is not supported by Google Calendar.
func (a *GCalAdapter) GetAttendee(ctx context.Context, accountID string, attendeeID string) (*model.Attendee, error) {
	return nil, adapter.ErrNotSupported
}

// DownloadAttachment is not supported by Google Calendar.
func (a *GCalAdapter) DownloadAttachment(ctx context.Context, accountID string, messageID string, attachmentID string) ([]byte, string, error) {
	return nil, "", adapter.ErrNotSupported
}

// SendEmail is not supported by Google Calendar.
func (a *GCalAdapter) SendEmail(ctx context.Context, accountID string, req adapter.SendEmailRequest) (*model.Email, error) {
	return nil, adapter.ErrNotSupported
}

// ListEmails is not supported by Google Calendar.
func (a *GCalAdapter) ListEmails(ctx context.Context, accountID string, opts adapter.ListEmailOpts) (*model.PaginatedList[model.Email], error) {
	return nil, adapter.ErrNotSupported
}

// GetEmail is not supported by Google Calendar.
func (a *GCalAdapter) GetEmail(ctx context.Context, accountID string, emailID string) (*model.Email, error) {
	return nil, adapter.ErrNotSupported
}

// ListCalendars returns calendars from Google Calendar.
func (a *GCalAdapter) ListCalendars(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Calendar], error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	path := "/users/me/calendarList"
	if opts.Cursor != "" {
		path = fmt.Sprintf("%s?pageToken=%s", path, opts.Cursor)
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	path = fmt.Sprintf("%s?maxResults=%d", path, limit)

	var result map[string]interface{}
	if err := gcalGet(client, path, &result); err != nil {
		return nil, fmt.Errorf("failed to list calendars: %w", err)
	}

	calendars, nextPageToken := normalizeCalendarList(result, accountID)
	hasMore := nextPageToken != ""

	return model.NewPaginatedList(calendars, nextPageToken, hasMore), nil
}

// GetCalendar gets a specific calendar from Google Calendar.
func (a *GCalAdapter) GetCalendar(ctx context.Context, accountID string, calendarID string) (*model.Calendar, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	path := fmt.Sprintf("/calendars/%s", calendarID)

	var result map[string]interface{}
	if err := gcalGet(client, path, &result); err != nil {
		return nil, fmt.Errorf("failed to get calendar: %w", err)
	}

	calendar := normalizeCalendar(result, accountID)
	return calendar, nil
}

// ListEvents returns events from a specific calendar.
func (a *GCalAdapter) ListEvents(ctx context.Context, accountID string, calendarID string, opts adapter.ListOpts) (*model.PaginatedList[model.CalendarEvent], error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	path := fmt.Sprintf("/calendars/%s/events", calendarID)

	params := ""
	if opts.Cursor != "" {
		params = fmt.Sprintf("%s&pageToken=%s", params, opts.Cursor)
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	params = fmt.Sprintf("%s&maxResults=%d", params, limit)

	if opts.After != nil {
		params = fmt.Sprintf("%s&timeMin=%s", params, opts.After.Format(time.RFC3339))
	}
	if opts.Before != nil {
		params = fmt.Sprintf("%s&timeMax=%s", params, opts.Before.Format(time.RFC3339))
	}

	if params != "" {
		path = path + "?" + params[1:]
	}

	var result map[string]interface{}
	if err := gcalGet(client, path, &result); err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	events, nextPageToken := normalizeEventList(result, accountID, calendarID)
	hasMore := nextPageToken != ""

	return model.NewPaginatedList(events, nextPageToken, hasMore), nil
}

// GetEvent gets a specific event from Google Calendar.
func (a *GCalAdapter) GetEvent(ctx context.Context, accountID string, calendarID string, eventID string) (*model.CalendarEvent, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	path := fmt.Sprintf("/calendars/%s/events/%s", calendarID, eventID)

	var result map[string]interface{}
	if err := gcalGet(client, path, &result); err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	event := normalizeEvent(result, accountID, calendarID)
	return event, nil
}

// CreateEvent creates a new event in Google Calendar.
func (a *GCalAdapter) CreateEvent(ctx context.Context, accountID string, calendarID string, req adapter.CreateEventRequest) (*model.CalendarEvent, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	body := buildEventBody(req)

	path := fmt.Sprintf("/calendars/%s/events", calendarID)

	var result map[string]interface{}
	if err := gcalPost(client, path, body, &result); err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	event := normalizeEvent(result, accountID, calendarID)

	if a.dispatcher != nil {
		a.dispatcher.Dispatch(ctx, model.EventCalendarEventCreated, event)
	}

	return event, nil
}

// UpdateEvent updates an existing event in Google Calendar.
func (a *GCalAdapter) UpdateEvent(ctx context.Context, accountID string, calendarID string, eventID string, req adapter.UpdateEventRequest) (*model.CalendarEvent, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	body := buildEventUpdateBody(req)

	path := fmt.Sprintf("/calendars/%s/events/%s", calendarID, eventID)

	var result map[string]interface{}
	if err := gcalPatch(client, path, body, &result); err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	event := normalizeEvent(result, accountID, calendarID)

	if a.dispatcher != nil {
		a.dispatcher.Dispatch(ctx, model.EventCalendarEventUpdated, event)
	}

	return event, nil
}

// DeleteEvent deletes an event from Google Calendar.
func (a *GCalAdapter) DeleteEvent(ctx context.Context, accountID string, calendarID string, eventID string) error {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	path := fmt.Sprintf("/calendars/%s/events/%s", calendarID, eventID)

	if err := gcalDelete(client, path); err != nil {
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

func buildEventBody(req adapter.CreateEventRequest) map[string]interface{} {
	body := map[string]interface{}{
		"summary":     req.Title,
		"description": req.Description,
		"location":    req.Location,
	}

	if req.AllDay {
		body["start"] = map[string]interface{}{
			"date": req.StartAt.Format("2006-01-02"),
		}
		body["end"] = map[string]interface{}{
			"date": req.EndAt.Format("2006-01-02"),
		}
	} else {
		body["start"] = map[string]interface{}{
			"dateTime": req.StartAt.Format(time.RFC3339),
		}
		body["end"] = map[string]interface{}{
			"dateTime": req.EndAt.Format(time.RFC3339),
		}
	}

	if len(req.Attendees) > 0 {
		attendees := make([]map[string]interface{}, len(req.Attendees))
		for i, att := range req.Attendees {
			attendees[i] = map[string]interface{}{
				"email":       att.Identifier,
				"displayName": att.DisplayName,
			}
		}
		body["attendees"] = attendees
	}

	if len(req.Reminders) > 0 {
		overrides := make([]map[string]interface{}, len(req.Reminders))
		for i, rem := range req.Reminders {
			overrides[i] = map[string]interface{}{
				"method":  rem.Method,
				"minutes": rem.MinutesBefore,
			}
		}
		body["reminders"] = map[string]interface{}{
			"useDefault": false,
			"overrides":  overrides,
		}
	}

	return body
}

func buildEventUpdateBody(req adapter.UpdateEventRequest) map[string]interface{} {
	body := make(map[string]interface{})

	if req.Title != nil {
		body["summary"] = *req.Title
	}
	if req.Description != nil {
		body["description"] = *req.Description
	}
	if req.Location != nil {
		body["location"] = *req.Location
	}
	if req.StartAt != nil {
		body["start"] = map[string]interface{}{
			"dateTime": req.StartAt.Format(time.RFC3339),
		}
	}
	if req.EndAt != nil {
		body["end"] = map[string]interface{}{
			"dateTime": req.EndAt.Format(time.RFC3339),
		}
	}
	if req.AllDay != nil && req.StartAt != nil {
		if *req.AllDay {
			body["start"] = map[string]interface{}{
				"date": req.StartAt.Format("2006-01-02"),
			}
			if req.EndAt != nil {
				body["end"] = map[string]interface{}{
					"date": req.EndAt.Format("2006-01-02"),
				}
			}
		}
	}

	if len(req.Attendees) > 0 {
		attendees := make([]map[string]interface{}, len(req.Attendees))
		for i, att := range req.Attendees {
			attendees[i] = map[string]interface{}{
				"email":       att.Identifier,
				"displayName": att.DisplayName,
			}
		}
		body["attendees"] = attendees
	}

	return body
}

func generateID() string {
	return "evt_" + uuid.New().String()
}

// Ensure GCalAdapter implements adapter.Provider.
var _ adapter.Provider = (*GCalAdapter)(nil)
