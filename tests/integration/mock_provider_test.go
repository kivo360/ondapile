package integration

import (
	"context"
	"time"

	"ondapile/internal/adapter"
	"ondapile/internal/model"
)

// MockProvider implements the adapter.Provider interface for testing.
type MockProvider struct {
	NameFunc               func() string
	ConnectFunc            func(ctx context.Context, accountID string, creds map[string]string) (*model.Account, error)
	DisconnectFunc         func(ctx context.Context, accountID string) error
	ReconnectFunc          func(ctx context.Context, accountID string) (*model.Account, error)
	StatusFunc             func(ctx context.Context, accountID string) (model.AccountStatus, error)
	GetAuthChallengeFunc   func(ctx context.Context, accountID string) (*adapter.AuthChallenge, error)
	SolveCheckpointFunc    func(ctx context.Context, accountID string, solution string) error
	ListChatsFunc          func(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Chat], error)
	GetChatFunc            func(ctx context.Context, accountID string, chatID string) (*model.Chat, error)
	ListMessagesFunc       func(ctx context.Context, accountID string, chatID string, opts adapter.ListOpts) (*model.PaginatedList[model.Message], error)
	SendMessageFunc        func(ctx context.Context, accountID string, chatID string, msg adapter.SendMessageRequest) (*model.Message, error)
	StartChatFunc          func(ctx context.Context, accountID string, req adapter.StartChatRequest) (*model.Chat, error)
	ListAttendeesFunc      func(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Attendee], error)
	GetAttendeeFunc        func(ctx context.Context, accountID string, attendeeID string) (*model.Attendee, error)
	DownloadAttachmentFunc func(ctx context.Context, accountID string, messageID string, attachmentID string) ([]byte, string, error)
	SendEmailFunc          func(ctx context.Context, accountID string, req adapter.SendEmailRequest) (*model.Email, error)
	ListEmailsFunc         func(ctx context.Context, accountID string, opts adapter.ListEmailOpts) (*model.PaginatedList[model.Email], error)
	GetEmailFunc           func(ctx context.Context, accountID string, emailID string) (*model.Email, error)

	// Store connected account info for test assertions
	ConnectedAccountID string
	ConnectedCreds     map[string]string
}

// Name returns the provider name.
func (m *MockProvider) Name() string {
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return "MOCK"
}

// Connect simulates connecting to the provider.
func (m *MockProvider) Connect(ctx context.Context, accountID string, creds map[string]string) (*model.Account, error) {
	if m.ConnectFunc != nil {
		return m.ConnectFunc(ctx, accountID, creds)
	}

	// Default behavior: return operational account
	m.ConnectedAccountID = accountID
	m.ConnectedCreds = creds

	name := "Mock Account"
	if identifier, ok := creds["identifier"]; ok && identifier != "" {
		name = identifier
	}

	return &model.Account{
		Object:       "account",
		ID:           accountID,
		Provider:     m.Name(),
		Name:         name,
		Identifier:   creds["identifier"],
		Status:       model.StatusOperational,
		Capabilities: []string{"messaging", "media"},
		Metadata:     map[string]any{},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

// Disconnect simulates disconnecting from the provider.
func (m *MockProvider) Disconnect(ctx context.Context, accountID string) error {
	if m.DisconnectFunc != nil {
		return m.DisconnectFunc(ctx, accountID)
	}
	return nil
}

// Reconnect simulates reconnecting to the provider.
func (m *MockProvider) Reconnect(ctx context.Context, accountID string) (*model.Account, error) {
	if m.ReconnectFunc != nil {
		return m.ReconnectFunc(ctx, accountID)
	}

	return &model.Account{
		Object:       "account",
		ID:           accountID,
		Provider:     m.Name(),
		Name:         "Mock Account",
		Status:       model.StatusOperational,
		Capabilities: []string{"messaging", "media"},
		Metadata:     map[string]any{},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}, nil
}

// Status returns the provider status.
func (m *MockProvider) Status(ctx context.Context, accountID string) (model.AccountStatus, error) {
	if m.StatusFunc != nil {
		return m.StatusFunc(ctx, accountID)
	}
	return model.StatusOperational, nil
}

// GetAuthChallenge returns an auth challenge for testing.
func (m *MockProvider) GetAuthChallenge(ctx context.Context, accountID string) (*adapter.AuthChallenge, error) {
	if m.GetAuthChallengeFunc != nil {
		return m.GetAuthChallengeFunc(ctx, accountID)
	}

	return &adapter.AuthChallenge{
		Type:    "QR_CODE",
		Payload: "test-qr-data",
		Expiry:  0,
	}, nil
}

// SolveCheckpoint simulates solving a checkpoint.
func (m *MockProvider) SolveCheckpoint(ctx context.Context, accountID string, solution string) error {
	if m.SolveCheckpointFunc != nil {
		return m.SolveCheckpointFunc(ctx, accountID, solution)
	}
	return nil
}

// ListChats returns a paginated list of chats.
func (m *MockProvider) ListChats(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Chat], error) {
	if m.ListChatsFunc != nil {
		return m.ListChatsFunc(ctx, accountID, opts)
	}

	return &model.PaginatedList[model.Chat]{
		Object:  "list",
		Items:   []model.Chat{},
		Cursor:  "",
		HasMore: false,
	}, nil
}

// GetChat returns a chat by ID.
func (m *MockProvider) GetChat(ctx context.Context, accountID string, chatID string) (*model.Chat, error) {
	if m.GetChatFunc != nil {
		return m.GetChatFunc(ctx, accountID, chatID)
	}

	return &model.Chat{
		Object:     "chat",
		ID:         chatID,
		AccountID:  accountID,
		Provider:   m.Name(),
		ProviderID: "mock-chat-" + chatID,
		Type:       string(model.ChatTypeOneToOne),
		Attendees:  []model.Attendee{},
		Metadata:   map[string]any{},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}, nil
}

// ListMessages returns a paginated list of messages.
func (m *MockProvider) ListMessages(ctx context.Context, accountID string, chatID string, opts adapter.ListOpts) (*model.PaginatedList[model.Message], error) {
	if m.ListMessagesFunc != nil {
		return m.ListMessagesFunc(ctx, accountID, chatID, opts)
	}

	return &model.PaginatedList[model.Message]{
		Object:  "list",
		Items:   []model.Message{},
		Cursor:  "",
		HasMore: false,
	}, nil
}

// SendMessage simulates sending a message.
func (m *MockProvider) SendMessage(ctx context.Context, accountID string, chatID string, msg adapter.SendMessageRequest) (*model.Message, error) {
	if m.SendMessageFunc != nil {
		return m.SendMessageFunc(ctx, accountID, chatID, msg)
	}

	return &model.Message{
		Object:      "message",
		ID:          "msg_test_" + time.Now().Format("20060102150405"),
		ChatID:      chatID,
		AccountID:   accountID,
		Provider:    m.Name(),
		ProviderID:  "mock-msg-" + time.Now().Format("20060102150405"),
		Text:        msg.Text,
		SenderID:    accountID,
		IsSender:    true,
		Timestamp:   time.Now(),
		Attachments: []model.Attachment{},
		Reactions:   []model.Reaction{},
		Metadata:    map[string]any{},
	}, nil
}

// StartChat simulates starting a new chat.
func (m *MockProvider) StartChat(ctx context.Context, accountID string, req adapter.StartChatRequest) (*model.Chat, error) {
	if m.StartChatFunc != nil {
		return m.StartChatFunc(ctx, accountID, req)
	}

	return &model.Chat{
		Object:     "chat",
		ID:         "chat_test_" + time.Now().Format("20060102150405"),
		AccountID:  accountID,
		Provider:   m.Name(),
		ProviderID: "mock-chat-" + req.AttendeeIdentifier,
		Type:       string(model.ChatTypeOneToOne),
		Attendees:  []model.Attendee{},
		Metadata:   map[string]any{},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}, nil
}

// ListAttendees returns a paginated list of attendees.
func (m *MockProvider) ListAttendees(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Attendee], error) {
	if m.ListAttendeesFunc != nil {
		return m.ListAttendeesFunc(ctx, accountID, opts)
	}

	return &model.PaginatedList[model.Attendee]{
		Object:  "list",
		Items:   []model.Attendee{},
		Cursor:  "",
		HasMore: false,
	}, nil
}

// GetAttendee returns an attendee by ID.
func (m *MockProvider) GetAttendee(ctx context.Context, accountID string, attendeeID string) (*model.Attendee, error) {
	if m.GetAttendeeFunc != nil {
		return m.GetAttendeeFunc(ctx, accountID, attendeeID)
	}

	return &model.Attendee{
		Object:         "attendee",
		ID:             attendeeID,
		AccountID:      accountID,
		Provider:       m.Name(),
		ProviderID:     attendeeID,
		Name:           "Mock Attendee",
		Identifier:     attendeeID,
		IdentifierType: string(model.IdentifierTypeProviderID),
		Metadata:       map[string]any{},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}, nil
}

// DownloadAttachment simulates downloading an attachment.
func (m *MockProvider) DownloadAttachment(ctx context.Context, accountID string, messageID string, attachmentID string) ([]byte, string, error) {
	if m.DownloadAttachmentFunc != nil {
		return m.DownloadAttachmentFunc(ctx, accountID, messageID, attachmentID)
	}

	// Return a simple PNG placeholder
	return []byte("mock attachment data"), "application/octet-stream", nil
}

// SendEmail simulates sending an email.
func (m *MockProvider) SendEmail(ctx context.Context, accountID string, req adapter.SendEmailRequest) (*model.Email, error) {
	if m.SendEmailFunc != nil {
		return m.SendEmailFunc(ctx, accountID, req)
	}

	return &model.Email{
		Object:    "email",
		ID:        "eml_test_" + time.Now().Format("20060102150405"),
		AccountID: accountID,
		Provider:  m.Name(),
		Subject:   req.Subject,
		Body:      req.BodyHTML,
		BodyPlain: req.BodyPlain,
		Date:      time.Now(),
		Metadata:  map[string]any{},
	}, nil
}

// ListEmails returns a paginated list of emails.
func (m *MockProvider) ListEmails(ctx context.Context, accountID string, opts adapter.ListEmailOpts) (*model.PaginatedList[model.Email], error) {
	if m.ListEmailsFunc != nil {
		return m.ListEmailsFunc(ctx, accountID, opts)
	}

	return &model.PaginatedList[model.Email]{
		Object:  "list",
		Items:   []model.Email{},
		Cursor:  "",
		HasMore: false,
	}, nil
}

// GetEmail returns an email by ID.
func (m *MockProvider) GetEmail(ctx context.Context, accountID string, emailID string) (*model.Email, error) {
	if m.GetEmailFunc != nil {
		return m.GetEmailFunc(ctx, accountID, emailID)
	}

	return &model.Email{
		Object:    "email",
		ID:        emailID,
		AccountID: accountID,
		Provider:  m.Name(),
		Subject:   "Mock Email",
		Date:      time.Now(),
		Metadata:  map[string]any{},
	}, nil
}

// Email action stubs
func (m *MockProvider) ReplyEmail(ctx context.Context, accountID string, emailID string, req adapter.SendEmailRequest) (*model.Email, error) {
	return nil, adapter.ErrNotSupported
}
func (m *MockProvider) ForwardEmail(ctx context.Context, accountID string, emailID string, req adapter.SendEmailRequest) (*model.Email, error) {
	return nil, adapter.ErrNotSupported
}
func (m *MockProvider) UpdateEmailProvider(ctx context.Context, accountID string, emailID string, opts adapter.UpdateEmailOpts) error {
	return adapter.ErrNotSupported
}
func (m *MockProvider) DeleteEmailProvider(ctx context.Context, accountID string, emailID string) error {
	return adapter.ErrNotSupported
}
func (m *MockProvider) ListFolders(ctx context.Context, accountID string) ([]string, error) {
	return nil, adapter.ErrNotSupported
}

// Calendar stubs — not supported by mock provider.
func (m *MockProvider) ListCalendars(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Calendar], error) {
	return nil, adapter.ErrNotSupported
}
func (m *MockProvider) GetCalendar(ctx context.Context, accountID string, calendarID string) (*model.Calendar, error) {
	return nil, adapter.ErrNotSupported
}
func (m *MockProvider) ListEvents(ctx context.Context, accountID string, calendarID string, opts adapter.ListOpts) (*model.PaginatedList[model.CalendarEvent], error) {
	return nil, adapter.ErrNotSupported
}
func (m *MockProvider) GetEvent(ctx context.Context, accountID string, calendarID string, eventID string) (*model.CalendarEvent, error) {
	return nil, adapter.ErrNotSupported
}
func (m *MockProvider) CreateEvent(ctx context.Context, accountID string, calendarID string, req adapter.CreateEventRequest) (*model.CalendarEvent, error) {
	return nil, adapter.ErrNotSupported
}
func (m *MockProvider) UpdateEvent(ctx context.Context, accountID string, calendarID string, eventID string, req adapter.UpdateEventRequest) (*model.CalendarEvent, error) {
	return nil, adapter.ErrNotSupported
}
func (m *MockProvider) DeleteEvent(ctx context.Context, accountID string, calendarID string, eventID string) error {
	return adapter.ErrNotSupported
}

// OAuth stubs — not supported by mock provider.
func (m *MockProvider) SupportsOAuth() bool { return false }
func (m *MockProvider) GetOAuthURL(ctx context.Context, state string) (string, error) {
	return "", adapter.ErrNotSupported
}
func (m *MockProvider) HandleOAuthCallback(ctx context.Context, code string) (map[string]string, error) {
	return nil, adapter.ErrNotSupported
}

// Ensure MockProvider implements Provider interface.
var _ adapter.Provider = (*MockProvider)(nil)
