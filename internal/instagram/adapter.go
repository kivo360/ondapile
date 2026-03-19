package instagram

import (
	"context"
	"fmt"
	"time"

	"ondapile/internal/adapter"
	"ondapile/internal/model"
	"ondapile/internal/oauth"
	"ondapile/internal/store"
	"ondapile/internal/webhook"

	oauth2lib "golang.org/x/oauth2"
)

// InstagramAdapter implements the adapter.Provider interface for Instagram.
type InstagramAdapter struct {
	oauthCfg   *oauth2lib.Config
	tokenStore *oauth.TokenStore
	store      *store.Store
	dispatcher *webhook.Dispatcher
}

// NewAdapter creates a new Instagram adapter.
func NewAdapter(oauthCfg *oauth2lib.Config, tokenStore *oauth.TokenStore, s *store.Store, d *webhook.Dispatcher) *InstagramAdapter {
	return &InstagramAdapter{
		oauthCfg:   oauthCfg,
		tokenStore: tokenStore,
		store:      s,
		dispatcher: d,
	}
}

// Name returns the provider identifier.
func (a *InstagramAdapter) Name() string { return "INSTAGRAM" }

// SupportsOAuth returns true — Instagram uses OAuth 2.0 via Meta/Facebook.
func (a *InstagramAdapter) SupportsOAuth() bool { return true }

// GetOAuthURL generates the Instagram OAuth authorization URL.
func (a *InstagramAdapter) GetOAuthURL(ctx context.Context, state string) (string, error) {
	return a.oauthCfg.AuthCodeURL(state), nil
}

// HandleOAuthCallback exchanges the authorization code for tokens.
func (a *InstagramAdapter) HandleOAuthCallback(ctx context.Context, code string) (map[string]string, error) {
	token, err := a.oauthCfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	return map[string]string{
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
		"token_type":    token.TokenType,
	}, nil
}

// Connect establishes an Instagram connection using stored OAuth tokens.
func (a *InstagramAdapter) Connect(ctx context.Context, accountID string, creds map[string]string) (*model.Account, error) {
	token := &oauth2lib.Token{
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

	// Verify token and get user info via Instagram Graph API
	var profile map[string]interface{}
	if err := instagramGet(client, "/me", &profile); err != nil {
		return nil, fmt.Errorf("failed to verify Instagram connection: %w", err)
	}

	userID, _ := profile["id"].(string)
	name, _ := profile["name"].(string)
	if name == "" {
		name = userID
	}

	return &model.Account{
		Object:       "account",
		ID:           accountID,
		Provider:     a.Name(),
		Name:         name,
		Identifier:   userID,
		Status:       model.StatusOperational,
		Capabilities: []string{"social", "messaging"},
	}, nil
}

// Disconnect closes the Instagram connection.
func (a *InstagramAdapter) Disconnect(ctx context.Context, accountID string) error {
	return a.tokenStore.Delete(ctx, accountID, a.Name())
}

// Reconnect re-establishes the Instagram connection.
func (a *InstagramAdapter) Reconnect(ctx context.Context, accountID string) (*model.Account, error) {
	_ = a.Disconnect(ctx, accountID)
	return nil, fmt.Errorf("Instagram reconnect requires re-authentication")
}

// Status returns the current connection status.
func (a *InstagramAdapter) Status(ctx context.Context, accountID string) (model.AccountStatus, error) {
	_, err := a.tokenStore.Load(ctx, accountID, a.Name())
	if err != nil {
		return model.StatusAuthRequired, nil
	}
	return model.StatusOperational, nil
}

// GetAuthChallenge returns an OAuth URL for Instagram authentication.
func (a *InstagramAdapter) GetAuthChallenge(ctx context.Context, accountID string) (*adapter.AuthChallenge, error) {
	url, err := a.GetOAuthURL(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return &adapter.AuthChallenge{
		Type:    "OAUTH_URL",
		Payload: url,
	}, nil
}

// SolveCheckpoint is not applicable for Instagram OAuth.
func (a *InstagramAdapter) SolveCheckpoint(ctx context.Context, accountID string, solution string) error {
	return adapter.ErrNotSupported
}

// --- Messaging methods ---

func (a *InstagramAdapter) ListChats(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Chat], error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Get Instagram Business account conversations
	var result struct {
		Conversations struct {
			Data []struct {
				ID           string `json:"id"`
				Participants struct {
					Data []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"data"`
				} `json:"participants"`
				UpdatedTime string `json:"updated_time"`
			} `json:"data"`
			Paging struct {
				Cursors struct {
					After string `json:"after"`
				} `json:"cursors"`
				Next string `json:"next"`
			} `json:"paging"`
		} `json:"conversations"`
	}

	path := "/me/conversations?fields=participants,updated_time"
	if opts.Limit > 0 {
		path += fmt.Sprintf("&limit=%d", opts.Limit)
	}
	if opts.Cursor != "" {
		path += fmt.Sprintf("&after=%s", opts.Cursor)
	}

	if err := instagramGet(client, path, &result); err != nil {
		return nil, fmt.Errorf("failed to list chats: %w", err)
	}

	chats := make([]model.Chat, 0, len(result.Conversations.Data))
	for _, conv := range result.Conversations.Data {
		chat := normalizeChat(&conv, accountID, a.Name())
		chats = append(chats, *chat)
	}

	cursor := ""
	hasMore := result.Conversations.Paging.Next != ""
	if hasMore {
		cursor = result.Conversations.Paging.Cursors.After
	}

	return model.NewPaginatedList(chats, cursor, hasMore), nil
}

func (a *InstagramAdapter) GetChat(ctx context.Context, accountID string, chatID string) (*model.Chat, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	var result struct {
		ID           string `json:"id"`
		Participants struct {
			Data []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"data"`
		} `json:"participants"`
		UpdatedTime string `json:"updated_time"`
	}

	path := fmt.Sprintf("/%s?fields=participants,updated_time", chatID)
	if err := instagramGet(client, path, &result); err != nil {
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}

	return normalizeChat(&result, accountID, a.Name()), nil
}

func (a *InstagramAdapter) ListMessages(ctx context.Context, accountID string, chatID string, opts adapter.ListOpts) (*model.PaginatedList[model.Message], error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Get messages from conversation
	var result struct {
		Messages struct {
			Data []struct {
				ID   string `json:"id"`
				From struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"from"`
				To struct {
					Data []struct {
						ID   string `json:"id"`
						Name string `json:"name"`
					} `json:"data"`
				} `json:"to"`
				Message     string `json:"message"`
				CreatedTime string `json:"created_time"`
			} `json:"data"`
			Paging struct {
				Cursors struct {
					After string `json:"after"`
				} `json:"cursors"`
				Next string `json:"next"`
			} `json:"paging"`
		} `json:"messages"`
	}

	path := fmt.Sprintf("/%s/messages?fields=from,to,message,created_time", chatID)
	if opts.Limit > 0 {
		path += fmt.Sprintf("&limit=%d", opts.Limit)
	}
	if opts.Cursor != "" {
		path += fmt.Sprintf("&after=%s", opts.Cursor)
	}

	if err := instagramGet(client, path, &result); err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	messages := make([]model.Message, 0, len(result.Messages.Data))
	for _, msg := range result.Messages.Data {
		message := normalizeMessage(&msg, chatID, accountID, a.Name())
		messages = append(messages, *message)
	}

	cursor := ""
	hasMore := result.Messages.Paging.Next != ""
	if hasMore {
		cursor = result.Messages.Paging.Cursors.After
	}

	return model.NewPaginatedList(messages, cursor, hasMore), nil
}

func (a *InstagramAdapter) SendMessage(ctx context.Context, accountID string, chatID string, msg adapter.SendMessageRequest) (*model.Message, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Send message via Instagram Graph API
	body := map[string]string{
		"recipient": fmt.Sprintf(`{"id":"%s"}`, chatID),
		"message":   fmt.Sprintf(`{"text":"%s"}`, msg.Text),
	}

	var result struct {
		ID string `json:"id"`
	}

	if err := instagramPost(client, "/me/messages", body, &result); err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return &model.Message{
		Object:     "message",
		ID:         result.ID,
		ChatID:     chatID,
		AccountID:  accountID,
		Provider:   a.Name(),
		ProviderID: result.ID,
		Text:       msg.Text,
		Timestamp:  time.Now(),
	}, nil
}

func (a *InstagramAdapter) StartChat(ctx context.Context, accountID string, req adapter.StartChatRequest) (*model.Chat, error) {
	// Instagram doesn't support starting chats directly via API for business accounts
	// Users can only reply to existing conversations
	return nil, adapter.ErrNotSupported
}

func (a *InstagramAdapter) ListAttendees(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Attendee], error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Get followers/people who have messaged the account
	var result struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
		Paging struct {
			Cursors struct {
				After string `json:"after"`
			} `json:"cursors"`
			Next string `json:"next"`
		} `json:"paging"`
	}

	path := "/me/followers?fields=id,name"
	if opts.Limit > 0 {
		path += fmt.Sprintf("&limit=%d", opts.Limit)
	}
	if opts.Cursor != "" {
		path += fmt.Sprintf("&after=%s", opts.Cursor)
	}

	if err := instagramGet(client, path, &result); err != nil {
		return nil, fmt.Errorf("failed to list attendees: %w", err)
	}

	attendees := make([]model.Attendee, 0, len(result.Data))
	for _, user := range result.Data {
		attendee := normalizeAttendee(&user, accountID, a.Name())
		attendees = append(attendees, *attendee)
	}

	cursor := ""
	hasMore := result.Paging.Next != ""
	if hasMore {
		cursor = result.Paging.Cursors.After
	}

	return model.NewPaginatedList(attendees, cursor, hasMore), nil
}

func (a *InstagramAdapter) GetAttendee(ctx context.Context, accountID string, attendeeID string) (*model.Attendee, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	var result struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	path := fmt.Sprintf("/%s?fields=id,name", attendeeID)
	if err := instagramGet(client, path, &result); err != nil {
		return nil, fmt.Errorf("failed to get attendee: %w", err)
	}

	return normalizeAttendee(&result, accountID, a.Name()), nil
}

func (a *InstagramAdapter) DownloadAttachment(ctx context.Context, accountID string, messageID string, attachmentID string) ([]byte, string, error) {
	return nil, "", adapter.ErrNotSupported
}

// --- Email stubs ---

func (a *InstagramAdapter) SendEmail(ctx context.Context, accountID string, req adapter.SendEmailRequest) (*model.Email, error) {
	return nil, adapter.ErrNotSupported
}

func (a *InstagramAdapter) ListEmails(ctx context.Context, accountID string, opts adapter.ListEmailOpts) (*model.PaginatedList[model.Email], error) {
	return nil, adapter.ErrNotSupported
}

func (a *InstagramAdapter) GetEmail(ctx context.Context, accountID string, emailID string) (*model.Email, error) {
	return nil, adapter.ErrNotSupported
}

// --- Calendar stubs ---

func (a *InstagramAdapter) ListCalendars(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Calendar], error) {
	return nil, adapter.ErrNotSupported
}

func (a *InstagramAdapter) GetCalendar(ctx context.Context, accountID string, calendarID string) (*model.Calendar, error) {
	return nil, adapter.ErrNotSupported
}

func (a *InstagramAdapter) ListEvents(ctx context.Context, accountID string, calendarID string, opts adapter.ListOpts) (*model.PaginatedList[model.CalendarEvent], error) {
	return nil, adapter.ErrNotSupported
}

func (a *InstagramAdapter) GetEvent(ctx context.Context, accountID string, calendarID string, eventID string) (*model.CalendarEvent, error) {
	return nil, adapter.ErrNotSupported
}

func (a *InstagramAdapter) CreateEvent(ctx context.Context, accountID string, calendarID string, req adapter.CreateEventRequest) (*model.CalendarEvent, error) {
	return nil, adapter.ErrNotSupported
}

func (a *InstagramAdapter) UpdateEvent(ctx context.Context, accountID string, calendarID string, eventID string, req adapter.UpdateEventRequest) (*model.CalendarEvent, error) {
	return nil, adapter.ErrNotSupported
}

func (a *InstagramAdapter) DeleteEvent(ctx context.Context, accountID string, calendarID string, eventID string) error {
	return adapter.ErrNotSupported
}

// Ensure InstagramAdapter implements adapter.Provider.
var _ adapter.Provider = (*InstagramAdapter)(nil)
