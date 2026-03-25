package telegram

import (
	"context"
	"fmt"

	"ondapile/internal/adapter"
	"ondapile/internal/model"
	"ondapile/internal/store"
	"ondapile/internal/webhook"
)

// TelegramAdapter implements the adapter.Provider interface for Telegram.
// Uses the Telegram Bot API (not MTProto).
type TelegramAdapter struct {
	store      *store.Store
	dispatcher *webhook.Dispatcher
}

// NewAdapter creates a new Telegram adapter.
func NewAdapter(s *store.Store, d *webhook.Dispatcher) *TelegramAdapter {
	return &TelegramAdapter{
		store:      s,
		dispatcher: d,
	}
}

// Name returns the provider identifier.
func (a *TelegramAdapter) Name() string { return "TELEGRAM" }

// SupportsOAuth returns false — Telegram uses Bot API tokens (not OAuth).
func (a *TelegramAdapter) SupportsOAuth() bool { return false }

// GetOAuthURL is not applicable for Telegram.
func (a *TelegramAdapter) GetOAuthURL(ctx context.Context, state string) (string, error) {
	return "", adapter.ErrNotSupported
}

// HandleOAuthCallback is not applicable for Telegram.
func (a *TelegramAdapter) HandleOAuthCallback(ctx context.Context, code string) (map[string]string, error) {
	return nil, adapter.ErrNotSupported
}

// Connect establishes a Telegram connection using a bot token.
func (a *TelegramAdapter) Connect(ctx context.Context, accountID string, creds map[string]string) (*model.Account, error) {
	botToken := creds["bot_token"]
	if botToken == "" {
		return nil, fmt.Errorf("bot_token is required")
	}

	// Validate the bot token by calling getMe
	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			ID        int64  `json:"id"`
			IsBot     bool   `json:"is_bot"`
			FirstName string `json:"first_name"`
			Username  string `json:"username"`
		} `json:"result"`
	}

	if err := telegramCall(botToken, "getMe", nil, &result); err != nil {
		return nil, fmt.Errorf("failed to validate Telegram bot token: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("invalid Telegram bot token")
	}

	name := result.Result.FirstName
	if result.Result.Username != "" {
		name = result.Result.Username
	}

	return &model.Account{
		Object:       "account",
		ID:           accountID,
		Provider:     a.Name(),
		Name:         name,
		Identifier:   result.Result.Username,
		Status:       model.StatusOperational,
		Capabilities: []string{"messaging", "bot"},
	}, nil
}

// Disconnect closes the Telegram connection.
func (a *TelegramAdapter) Disconnect(ctx context.Context, accountID string) error {
	// Nothing to clean up for bot API
	return nil
}

// Reconnect re-establishes the Telegram connection.
func (a *TelegramAdapter) Reconnect(ctx context.Context, accountID string) (*model.Account, error) {
	// For Telegram, reconnection would require the bot token again
	return nil, fmt.Errorf("Telegram reconnect requires bot token")
}

// Status returns the current connection status.
func (a *TelegramAdapter) Status(ctx context.Context, accountID string) (model.AccountStatus, error) {
	// Since we don't store tokens in a persistent way for Telegram bots,
	// we return OPERATIONAL by default. The actual status would be checked
	// when making API calls.
	return model.StatusOperational, nil
}

// GetAuthChallenge returns nil for Telegram (bot tokens are provided directly).
func (a *TelegramAdapter) GetAuthChallenge(ctx context.Context, accountID string) (*adapter.AuthChallenge, error) {
	// Telegram bots use direct token authentication, no challenge needed
	return nil, nil
}

// SolveCheckpoint is not applicable for Telegram.
func (a *TelegramAdapter) SolveCheckpoint(ctx context.Context, accountID string, solution string) error {
	return adapter.ErrNotSupported
}

// --- Messaging methods ---

func (a *TelegramAdapter) ListChats(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Chat], error) {
	// Get bot token from credentials store or account
	// For now, this is a limitation - we'd need to store the bot token
	return nil, adapter.ErrNotSupported
}

func (a *TelegramAdapter) GetChat(ctx context.Context, accountID string, chatID string) (*model.Chat, error) {
	// Telegram chat ID is the chat_id used in Bot API
	return nil, adapter.ErrNotSupported
}

func (a *TelegramAdapter) ListMessages(ctx context.Context, accountID string, chatID string, opts adapter.ListOpts) (*model.PaginatedList[model.Message], error) {
	// Bot API has limited access to message history
	// Bots can only see messages sent to them after they were added
	return nil, adapter.ErrNotSupported
}

func (a *TelegramAdapter) SendMessage(ctx context.Context, accountID string, chatID string, msg adapter.SendMessageRequest) (*model.Message, error) {
	// Get bot token - in production this would be retrieved from secure storage
	return nil, adapter.ErrNotSupported
}

func (a *TelegramAdapter) StartChat(ctx context.Context, accountID string, req adapter.StartChatRequest) (*model.Chat, error) {
	// Telegram bots cannot initiate conversations with users
	// Users must message the bot first
	return nil, adapter.ErrNotSupported
}

func (a *TelegramAdapter) ListAttendees(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Attendee], error) {
	// Get chat members - requires knowing which chats the bot is in
	return nil, adapter.ErrNotSupported
}

func (a *TelegramAdapter) GetAttendee(ctx context.Context, accountID string, attendeeID string) (*model.Attendee, error) {
	// Get user info from Telegram
	return nil, adapter.ErrNotSupported
}

func (a *TelegramAdapter) DownloadAttachment(ctx context.Context, accountID string, messageID string, attachmentID string) ([]byte, string, error) {
	return nil, "", adapter.ErrNotSupported
}

// --- Email stubs ---

func (a *TelegramAdapter) SendEmail(ctx context.Context, accountID string, req adapter.SendEmailRequest) (*model.Email, error) {
	return nil, adapter.ErrNotSupported
}

func (a *TelegramAdapter) ListEmails(ctx context.Context, accountID string, opts adapter.ListEmailOpts) (*model.PaginatedList[model.Email], error) {
	return nil, adapter.ErrNotSupported
}

func (a *TelegramAdapter) GetEmail(ctx context.Context, accountID string, emailID string) (*model.Email, error) {
	return nil, adapter.ErrNotSupported
}

// ReplyEmail is not supported by Telegram.
func (a *TelegramAdapter) ReplyEmail(ctx context.Context, accountID string, emailID string, req adapter.SendEmailRequest) (*model.Email, error) {
	return nil, adapter.ErrNotSupported
}

// ForwardEmail is not supported by Telegram.
func (a *TelegramAdapter) ForwardEmail(ctx context.Context, accountID string, emailID string, req adapter.SendEmailRequest) (*model.Email, error) {
	return nil, adapter.ErrNotSupported
}

// UpdateEmailProvider is not supported by Telegram.
func (a *TelegramAdapter) UpdateEmailProvider(ctx context.Context, accountID string, emailID string, opts adapter.UpdateEmailOpts) error {
	return adapter.ErrNotSupported
}

// DeleteEmailProvider is not supported by Telegram.
func (a *TelegramAdapter) DeleteEmailProvider(ctx context.Context, accountID string, emailID string) error {
	return adapter.ErrNotSupported
}

// ListFolders is not supported by Telegram.
func (a *TelegramAdapter) ListFolders(ctx context.Context, accountID string) ([]string, error) {
	return nil, adapter.ErrNotSupported
}

// --- Calendar stubs ---

func (a *TelegramAdapter) ListCalendars(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Calendar], error) {
	return nil, adapter.ErrNotSupported
}

func (a *TelegramAdapter) GetCalendar(ctx context.Context, accountID string, calendarID string) (*model.Calendar, error) {
	return nil, adapter.ErrNotSupported
}

func (a *TelegramAdapter) ListEvents(ctx context.Context, accountID string, calendarID string, opts adapter.ListOpts) (*model.PaginatedList[model.CalendarEvent], error) {
	return nil, adapter.ErrNotSupported
}

func (a *TelegramAdapter) GetEvent(ctx context.Context, accountID string, calendarID string, eventID string) (*model.CalendarEvent, error) {
	return nil, adapter.ErrNotSupported
}

func (a *TelegramAdapter) CreateEvent(ctx context.Context, accountID string, calendarID string, req adapter.CreateEventRequest) (*model.CalendarEvent, error) {
	return nil, adapter.ErrNotSupported
}

func (a *TelegramAdapter) UpdateEvent(ctx context.Context, accountID string, calendarID string, eventID string, req adapter.UpdateEventRequest) (*model.CalendarEvent, error) {
	return nil, adapter.ErrNotSupported
}

func (a *TelegramAdapter) DeleteEvent(ctx context.Context, accountID string, calendarID string, eventID string) error {
	return adapter.ErrNotSupported
}

// Ensure TelegramAdapter implements adapter.Provider.
var _ adapter.Provider = (*TelegramAdapter)(nil)
