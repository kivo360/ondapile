package outlook

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

// OutlookAdapter implements the adapter.Provider interface for Outlook (Microsoft Graph API).
type OutlookAdapter struct {
	oauthCfg   *oauth2.Config
	tokenStore *oauth.TokenStore
	store      *store.Store
	dispatcher *webhook.Dispatcher
}

// NewAdapter creates a new Outlook adapter.
func NewAdapter(oauthCfg *oauth2.Config, tokenStore *oauth.TokenStore, s *store.Store, d *webhook.Dispatcher) *OutlookAdapter {
	return &OutlookAdapter{
		oauthCfg:   oauthCfg,
		tokenStore: tokenStore,
		store:      s,
		dispatcher: d,
	}
}

// Name returns the provider identifier.
func (a *OutlookAdapter) Name() string {
	return "OUTLOOK"
}

// SupportsOAuth returns true as Outlook uses OAuth2.
func (a *OutlookAdapter) SupportsOAuth() bool {
	return true
}

// GetOAuthURL generates the OAuth URL for Outlook authentication.
func (a *OutlookAdapter) GetOAuthURL(ctx context.Context, state string) (string, error) {
	if a.oauthCfg == nil {
		return "", fmt.Errorf("OAuth config not initialized")
	}

	return a.oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent")), nil
}

// HandleOAuthCallback exchanges the authorization code for a token.
func (a *OutlookAdapter) HandleOAuthCallback(ctx context.Context, code string) (map[string]string, error) {
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

// Connect establishes an Outlook connection using stored OAuth tokens.
func (a *OutlookAdapter) Connect(ctx context.Context, accountID string, creds map[string]string) (*model.Account, error) {
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

	var profile map[string]interface{}
	if err := graphGet(client, "/me", &profile); err != nil {
		return nil, fmt.Errorf("failed to verify Outlook connection: %w", err)
	}

	email, _ := profile["mail"].(string)
	if email == "" {
		email, _ = profile["userPrincipalName"].(string)
	}
	name, _ := profile["displayName"].(string)

	return &model.Account{
		Object:       "account",
		ID:           accountID,
		Provider:     a.Name(),
		Name:         name,
		Identifier:   email,
		Status:       model.StatusOperational,
		Capabilities: []string{"email", "calendar", "send", "receive", "events"},
	}, nil
}

// Disconnect closes the Outlook connection.
func (a *OutlookAdapter) Disconnect(ctx context.Context, accountID string) error {
	return nil
}

// Reconnect re-establishes the Outlook connection.
func (a *OutlookAdapter) Reconnect(ctx context.Context, accountID string) (*model.Account, error) {
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
func (a *OutlookAdapter) Status(ctx context.Context, accountID string) (model.AccountStatus, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return model.StatusInterrupted, nil
	}

	var profile map[string]interface{}
	if err := graphGet(client, "/me", &profile); err != nil {
		return model.StatusInterrupted, nil
	}

	return model.StatusOperational, nil
}

// GetAuthChallenge returns nil as Outlook uses OAuth.
func (a *OutlookAdapter) GetAuthChallenge(ctx context.Context, accountID string) (*adapter.AuthChallenge, error) {
	return nil, nil
}

// SolveCheckpoint is not applicable for Outlook.
func (a *OutlookAdapter) SolveCheckpoint(ctx context.Context, accountID string, solution string) error {
	return adapter.ErrNotSupported
}

// ListChats is not supported by Outlook.
func (a *OutlookAdapter) ListChats(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Chat], error) {
	return nil, adapter.ErrNotSupported
}

// GetChat is not supported by Outlook.
func (a *OutlookAdapter) GetChat(ctx context.Context, accountID string, chatID string) (*model.Chat, error) {
	return nil, adapter.ErrNotSupported
}

// ListMessages is not supported by Outlook.
func (a *OutlookAdapter) ListMessages(ctx context.Context, accountID string, chatID string, opts adapter.ListOpts) (*model.PaginatedList[model.Message], error) {
	return nil, adapter.ErrNotSupported
}

// SendMessage is not supported by Outlook.
func (a *OutlookAdapter) SendMessage(ctx context.Context, accountID string, chatID string, msg adapter.SendMessageRequest) (*model.Message, error) {
	return nil, adapter.ErrNotSupported
}

// StartChat is not supported by Outlook.
func (a *OutlookAdapter) StartChat(ctx context.Context, accountID string, req adapter.StartChatRequest) (*model.Chat, error) {
	return nil, adapter.ErrNotSupported
}

// ListAttendees is not supported by Outlook.
func (a *OutlookAdapter) ListAttendees(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Attendee], error) {
	return nil, adapter.ErrNotSupported
}

// GetAttendee is not supported by Outlook.
func (a *OutlookAdapter) GetAttendee(ctx context.Context, accountID string, attendeeID string) (*model.Attendee, error) {
	return nil, adapter.ErrNotSupported
}

// ReplyEmail is not supported by Outlook.
func (a *OutlookAdapter) ReplyEmail(ctx context.Context, accountID string, emailID string, req adapter.SendEmailRequest) (*model.Email, error) {
	return nil, adapter.ErrNotSupported
}

// ForwardEmail is not supported by Outlook.
func (a *OutlookAdapter) ForwardEmail(ctx context.Context, accountID string, emailID string, req adapter.SendEmailRequest) (*model.Email, error) {
	return nil, adapter.ErrNotSupported
}

// UpdateEmailProvider is not supported by Outlook.
func (a *OutlookAdapter) UpdateEmailProvider(ctx context.Context, accountID string, emailID string, opts adapter.UpdateEmailOpts) error {
	return adapter.ErrNotSupported
}

// DeleteEmailProvider is not supported by Outlook.
func (a *OutlookAdapter) DeleteEmailProvider(ctx context.Context, accountID string, emailID string) error {
	return adapter.ErrNotSupported
}

// ListFolders is not supported by Outlook.
func (a *OutlookAdapter) ListFolders(ctx context.Context, accountID string) ([]string, error) {
	return nil, adapter.ErrNotSupported
}

// DownloadAttachment downloads an attachment from Outlook.
func (a *OutlookAdapter) DownloadAttachment(ctx context.Context, accountID string, messageID string, attachmentID string) ([]byte, string, error) {
	return a.downloadEmailAttachment(ctx, accountID, messageID, attachmentID)
}

// Generate ID helper
func generateID() string {
	return uuid.New().String()
}

// now returns current time
func now() time.Time {
	return time.Now()
}

// Ensure OutlookAdapter implements adapter.Provider.
var _ adapter.Provider = (*OutlookAdapter)(nil)
