package linkedin

import (
	"context"
	"fmt"

	"ondapile/internal/adapter"
	"ondapile/internal/model"
	"ondapile/internal/oauth"
	"ondapile/internal/store"
	"ondapile/internal/webhook"

	oauth2lib "golang.org/x/oauth2"
)

// LinkedInAdapter implements the adapter.Provider interface for LinkedIn.
type LinkedInAdapter struct {
	oauthCfg   *oauth2lib.Config
	tokenStore *oauth.TokenStore
	store      *store.Store
	dispatcher *webhook.Dispatcher
}

// NewAdapter creates a new LinkedIn adapter.
func NewAdapter(oauthCfg *oauth2lib.Config, tokenStore *oauth.TokenStore, s *store.Store, d *webhook.Dispatcher) *LinkedInAdapter {
	return &LinkedInAdapter{
		oauthCfg:   oauthCfg,
		tokenStore: tokenStore,
		store:      s,
		dispatcher: d,
	}
}

// Name returns the provider identifier.
func (a *LinkedInAdapter) Name() string { return "LINKEDIN" }

// SupportsOAuth returns true — LinkedIn uses OAuth 2.0.
func (a *LinkedInAdapter) SupportsOAuth() bool { return true }

// GetOAuthURL generates the LinkedIn OAuth authorization URL.
func (a *LinkedInAdapter) GetOAuthURL(ctx context.Context, state string) (string, error) {
	return a.oauthCfg.AuthCodeURL(state), nil
}

// HandleOAuthCallback exchanges the authorization code for tokens.
func (a *LinkedInAdapter) HandleOAuthCallback(ctx context.Context, code string) (map[string]string, error) {
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

// Connect establishes a LinkedIn connection using stored OAuth tokens.
func (a *LinkedInAdapter) Connect(ctx context.Context, accountID string, creds map[string]string) (*model.Account, error) {
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

	var profile map[string]interface{}
	if err := linkedinGet(client, "/userinfo", &profile); err != nil {
		return nil, fmt.Errorf("failed to verify LinkedIn connection: %w", err)
	}

	emailAddr, _ := profile["email"].(string)
	name, _ := profile["name"].(string)
	if name == "" {
		givenName, _ := profile["given_name"].(string)
		familyName, _ := profile["family_name"].(string)
		name = givenName + " " + familyName
	}

	return &model.Account{
		Object:       "account",
		ID:           accountID,
		Provider:     a.Name(),
		Name:         name,
		Identifier:   emailAddr,
		Status:       model.StatusOperational,
		Capabilities: []string{"social", "profile"},
	}, nil
}

// Disconnect closes the LinkedIn connection.
func (a *LinkedInAdapter) Disconnect(ctx context.Context, accountID string) error {
	return a.tokenStore.Delete(ctx, accountID, a.Name())
}

// Reconnect re-establishes the LinkedIn connection.
func (a *LinkedInAdapter) Reconnect(ctx context.Context, accountID string) (*model.Account, error) {
	_ = a.Disconnect(ctx, accountID)
	return nil, fmt.Errorf("LinkedIn reconnect requires re-authentication")
}

// Status returns the current connection status.
func (a *LinkedInAdapter) Status(ctx context.Context, accountID string) (model.AccountStatus, error) {
	_, err := a.tokenStore.Load(ctx, accountID, a.Name())
	if err != nil {
		return model.StatusAuthRequired, nil
	}
	return model.StatusOperational, nil
}

// GetAuthChallenge returns an OAuth URL for LinkedIn authentication.
func (a *LinkedInAdapter) GetAuthChallenge(ctx context.Context, accountID string) (*adapter.AuthChallenge, error) {
	url, err := a.GetOAuthURL(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return &adapter.AuthChallenge{
		Type:    "OAUTH_URL",
		Payload: url,
	}, nil
}

// SolveCheckpoint is not applicable for LinkedIn OAuth.
func (a *LinkedInAdapter) SolveCheckpoint(ctx context.Context, accountID string, solution string) error {
	return adapter.ErrNotSupported
}

// --- Messaging stubs (LinkedIn messaging requires partner-level API access) ---

func (a *LinkedInAdapter) ListChats(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Chat], error) {
	return nil, adapter.ErrNotSupported
}

func (a *LinkedInAdapter) GetChat(ctx context.Context, accountID string, chatID string) (*model.Chat, error) {
	return nil, adapter.ErrNotSupported
}

func (a *LinkedInAdapter) ListMessages(ctx context.Context, accountID string, chatID string, opts adapter.ListOpts) (*model.PaginatedList[model.Message], error) {
	return nil, adapter.ErrNotSupported
}

func (a *LinkedInAdapter) SendMessage(ctx context.Context, accountID string, chatID string, msg adapter.SendMessageRequest) (*model.Message, error) {
	return nil, adapter.ErrNotSupported
}

func (a *LinkedInAdapter) StartChat(ctx context.Context, accountID string, req adapter.StartChatRequest) (*model.Chat, error) {
	return nil, adapter.ErrNotSupported
}

func (a *LinkedInAdapter) ListAttendees(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Attendee], error) {
	return nil, adapter.ErrNotSupported
}

func (a *LinkedInAdapter) GetAttendee(ctx context.Context, accountID string, attendeeID string) (*model.Attendee, error) {
	return nil, adapter.ErrNotSupported
}

func (a *LinkedInAdapter) DownloadAttachment(ctx context.Context, accountID string, messageID string, attachmentID string) ([]byte, string, error) {
	return nil, "", adapter.ErrNotSupported
}

// --- Email stubs ---

func (a *LinkedInAdapter) SendEmail(ctx context.Context, accountID string, req adapter.SendEmailRequest) (*model.Email, error) {
	return nil, adapter.ErrNotSupported
}

func (a *LinkedInAdapter) ListEmails(ctx context.Context, accountID string, opts adapter.ListEmailOpts) (*model.PaginatedList[model.Email], error) {
	return nil, adapter.ErrNotSupported
}

func (a *LinkedInAdapter) GetEmail(ctx context.Context, accountID string, emailID string) (*model.Email, error) {
	return nil, adapter.ErrNotSupported
}

// --- Calendar stubs ---

func (a *LinkedInAdapter) ListCalendars(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Calendar], error) {
	return nil, adapter.ErrNotSupported
}

func (a *LinkedInAdapter) GetCalendar(ctx context.Context, accountID string, calendarID string) (*model.Calendar, error) {
	return nil, adapter.ErrNotSupported
}

func (a *LinkedInAdapter) ListEvents(ctx context.Context, accountID string, calendarID string, opts adapter.ListOpts) (*model.PaginatedList[model.CalendarEvent], error) {
	return nil, adapter.ErrNotSupported
}

func (a *LinkedInAdapter) GetEvent(ctx context.Context, accountID string, calendarID string, eventID string) (*model.CalendarEvent, error) {
	return nil, adapter.ErrNotSupported
}

func (a *LinkedInAdapter) CreateEvent(ctx context.Context, accountID string, calendarID string, req adapter.CreateEventRequest) (*model.CalendarEvent, error) {
	return nil, adapter.ErrNotSupported
}

func (a *LinkedInAdapter) UpdateEvent(ctx context.Context, accountID string, calendarID string, eventID string, req adapter.UpdateEventRequest) (*model.CalendarEvent, error) {
	return nil, adapter.ErrNotSupported
}

func (a *LinkedInAdapter) DeleteEvent(ctx context.Context, accountID string, calendarID string, eventID string) error {
	return adapter.ErrNotSupported
}

// Ensure LinkedInAdapter implements adapter.Provider.
var _ adapter.Provider = (*LinkedInAdapter)(nil)
