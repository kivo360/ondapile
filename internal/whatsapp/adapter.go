package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"ondapile/internal/adapter"
	"ondapile/internal/model"
	"ondapile/internal/store"
	"ondapile/internal/webhook"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// WhatsAppProvider implements adapter.Provider for WhatsApp using whatsmeow.
type WhatsAppProvider struct {
	store           *store.Store
	accountStore    *store.AccountStore
	chatStore       *store.ChatStore
	messageStore    *store.MessageStore
	dispatcher      *webhook.Dispatcher
	deviceStorePath string

	// Multi-client management: accountID -> client
	clients sync.Map

	// QR code data cache: accountID -> QR string
	qrCodes sync.Map
}

// NewProvider creates a new WhatsApp provider instance.
func NewProvider(
	store *store.Store,
	accountStore *store.AccountStore,
	chatStore *store.ChatStore,
	messageStore *store.MessageStore,
	dispatcher *webhook.Dispatcher,
	deviceStorePath string,
) *WhatsAppProvider {
	return &WhatsAppProvider{
		store:           store,
		accountStore:    accountStore,
		chatStore:       chatStore,
		messageStore:    messageStore,
		dispatcher:      dispatcher,
		deviceStorePath: deviceStorePath,
	}
}

// Name returns the provider identifier.
func (p *WhatsAppProvider) Name() string {
	return "WHATSAPP"
}

// Connect establishes a WhatsApp connection for the given account.
// If no existing session, starts QR code flow. If session exists, connects directly.
func (p *WhatsAppProvider) Connect(ctx context.Context, accountID string, creds map[string]string) (*model.Account, error) {
	// Check if client already exists
	if _, loaded := p.clients.Load(accountID); loaded {
		return nil, fmt.Errorf("client already connected for account %s", accountID)
	}

	// Get account details
	account, err := p.accountStore.GetByID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	if account == nil {
		return nil, fmt.Errorf("account not found: %s", accountID)
	}

	// Create client with device store
	client, err := p.createClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Set up event handler
	client.AddEventHandler(p.eventHandler(accountID))

	// Check if we have an existing session
	if client.Store.ID != nil {
		// Existing session - connect directly
		if err := p.accountStore.UpdateStatus(ctx, accountID, model.StatusConnecting, nil); err != nil {
			slog.Warn("failed to update account status", "error", err)
		}

		if err := client.Connect(); err != nil {
			_ = p.accountStore.UpdateStatus(ctx, accountID, model.StatusInterrupted, strPtr("Connection failed"))
			return nil, fmt.Errorf("failed to connect: %w", err)
		}

		// Store client reference
		p.clients.Store(accountID, client)

		// Update status to operational
		if err := p.accountStore.UpdateStatus(ctx, accountID, model.StatusOperational, nil); err != nil {
			slog.Warn("failed to update account status", "error", err)
		}

		return account, nil
	}

	// No existing session - need QR authentication
	// Generate QR code
	qrChan, _ := client.GetQRChannel(ctx)

	// Store client (not connected yet)
	p.clients.Store(accountID, client)

	// Update status to auth required
	if err := p.accountStore.UpdateStatus(ctx, accountID, model.StatusAuthRequired, strPtr("QR code authentication required")); err != nil {
		slog.Warn("failed to update account status", "error", err)
	}

	// Start QR handling in background
	go p.handleQRFlow(accountID, client, qrChan)

	// Connect to trigger the QR code generation
	if err := client.Connect(); err != nil {
		p.clients.Delete(accountID)
		return nil, fmt.Errorf("failed to connect for QR auth: %w", err)
	}

	return account, nil
}

// handleQRFlow processes QR code events in the background.
func (p *WhatsAppProvider) handleQRFlow(accountID string, client *whatsmeow.Client, qrChan <-chan whatsmeow.QRChannelItem) {
	for evt := range qrChan {
		switch evt.Event {
		case "code":
			// Store QR code for retrieval
			p.qrCodes.Store(accountID, evt.Code)
		case "success":
			// QR scanned successfully
			p.qrCodes.Delete(accountID)
			slog.Info("WhatsApp QR scanned successfully", "account_id", accountID)
			p.dispatcher.Dispatch(context.Background(), model.EventAccountConnected, map[string]any{
				"account_id": accountID,
				"provider":   p.Name(),
			})
		case "timeout":
			// QR code expired
			p.qrCodes.Delete(accountID)
			slog.Warn("WhatsApp QR code expired", "account_id", accountID)
			_ = p.accountStore.UpdateStatus(context.Background(), accountID, model.StatusAuthRequired, strPtr("QR code expired"))
			p.dispatcher.Dispatch(context.Background(), model.EventAccountCheckpoint, map[string]any{
				"account_id":      accountID,
				"checkpoint_type": "qr_expired",
			})
		}
	}
}

// Disconnect closes the WhatsApp connection for the given account.
func (p *WhatsAppProvider) Disconnect(ctx context.Context, accountID string) error {
	clientVal, loaded := p.clients.LoadAndDelete(accountID)
	if !loaded {
		return fmt.Errorf("client not found for account %s", accountID)
	}

	client := clientVal.(*whatsmeow.Client)
	client.Disconnect()

	// Clear QR code if present
	p.qrCodes.Delete(accountID)

	// Update status
	if err := p.accountStore.UpdateStatus(ctx, accountID, model.StatusPaused, nil); err != nil {
		slog.Warn("failed to update account status on disconnect", "error", err)
	}

	return nil
}

// Reconnect attempts to reconnect an existing WhatsApp session.
func (p *WhatsAppProvider) Reconnect(ctx context.Context, accountID string) (*model.Account, error) {
	// Disconnect if connected
	p.Disconnect(ctx, accountID)

	// Get account
	account, err := p.accountStore.GetByID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	if account == nil {
		return nil, fmt.Errorf("account not found: %s", accountID)
	}

	// Create new client
	client, err := p.createClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Check if session exists
	if client.Store.ID == nil {
		return nil, fmt.Errorf("no existing session for account %s, re-authentication required", accountID)
	}

	// Set up event handler
	client.AddEventHandler(p.eventHandler(accountID))

	// Connect
	if err := p.accountStore.UpdateStatus(ctx, accountID, model.StatusConnecting, nil); err != nil {
		slog.Warn("failed to update account status", "error", err)
	}

	if err := client.Connect(); err != nil {
		_ = p.accountStore.UpdateStatus(ctx, accountID, model.StatusInterrupted, strPtr("Reconnection failed"))
		return nil, fmt.Errorf("failed to reconnect: %w", err)
	}

	// Store client
	p.clients.Store(accountID, client)

	// Update status
	if err := p.accountStore.UpdateStatus(ctx, accountID, model.StatusOperational, nil); err != nil {
		slog.Warn("failed to update account status", "error", err)
	}

	return account, nil
}

// Status returns the current connection status for the account.
func (p *WhatsAppProvider) Status(ctx context.Context, accountID string) (model.AccountStatus, error) {
	account, err := p.accountStore.GetByID(ctx, accountID)
	if err != nil {
		return "", fmt.Errorf("failed to get account: %w", err)
	}
	if account == nil {
		return "", fmt.Errorf("account not found: %s", accountID)
	}

	// Check if client is connected
	if clientVal, ok := p.clients.Load(accountID); ok {
		client := clientVal.(*whatsmeow.Client)
		if client.IsConnected() {
			return model.StatusOperational, nil
		}
		return model.StatusInterrupted, nil
	}

	return account.Status, nil
}

// GetAuthChallenge returns the current QR code for authentication.
func (p *WhatsAppProvider) GetAuthChallenge(ctx context.Context, accountID string) (*adapter.AuthChallenge, error) {
	qrData, ok := p.qrCodes.Load(accountID)
	if !ok {
		return nil, fmt.Errorf("no QR code available for account %s", accountID)
	}

	return &adapter.AuthChallenge{
		Type:    "QR_CODE",
		Payload: qrData.(string),
		Expiry:  0, // QR codes have their own expiry mechanism
	}, nil
}

// SolveCheckpoint is not applicable for WhatsApp QR auth.
func (p *WhatsAppProvider) SolveCheckpoint(ctx context.Context, accountID string, solution string) error {
	return fmt.Errorf("checkpoint solving not applicable for WhatsApp")
}

// ListChats returns chats from PostgreSQL store.
func (p *WhatsAppProvider) ListChats(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Chat], error) {
	chats, nextCursor, hasMore, err := p.chatStore.List(ctx, &accountID, nil, opts.Cursor, opts.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list chats: %w", err)
	}
	// Convert []*model.Chat to []model.Chat
	chatItems := make([]model.Chat, len(chats))
	for i, chat := range chats {
		chatItems[i] = *chat
	}

	return &model.PaginatedList[model.Chat]{
		Object:  "list",
		Items:   chatItems,
		Cursor:  nextCursor,
		HasMore: hasMore,
	}, nil
}


// GetChat returns a specific chat from PostgreSQL store.
func (p *WhatsAppProvider) GetChat(ctx context.Context, accountID string, chatID string) (*model.Chat, error) {
	chat, err := p.chatStore.GetByID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}
	if chat == nil || chat.AccountID != accountID {
		return nil, fmt.Errorf("chat not found: %s", chatID)
	}
	return chat, nil
}

// ListMessages returns messages for a chat from PostgreSQL store.
func (p *WhatsAppProvider) ListMessages(ctx context.Context, accountID string, chatID string, opts adapter.ListOpts) (*model.PaginatedList[model.Message], error) {
	// Verify chat belongs to account
	chat, err := p.chatStore.GetByID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}
	if chat == nil || chat.AccountID != accountID {
		return nil, fmt.Errorf("chat not found: %s", chatID)
	}

	messages, nextCursor, hasMore, err := p.messageStore.ListByChat(ctx, chatID, opts.Cursor, opts.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}
	// Convert []*model.Message to []model.Message
	messageItems := make([]model.Message, len(messages))
	for i, msg := range messages {
		messageItems[i] = *msg
	}

	return &model.PaginatedList[model.Message]{
		Object:  "list",
		Items:   messageItems,
		Cursor:  nextCursor,
		HasMore: hasMore,
	}, nil
}


// SendMessage sends a message via WhatsApp.
func (p *WhatsAppProvider) SendMessage(ctx context.Context, accountID string, chatID string, msg adapter.SendMessageRequest) (*model.Message, error) {
	// Get client
	clientVal, ok := p.clients.Load(accountID)
	if !ok {
		return nil, fmt.Errorf("client not connected for account %s", accountID)
	}
	client := clientVal.(*whatsmeow.Client)

	// Get chat
	chat, err := p.chatStore.GetByID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}
	if chat == nil || chat.AccountID != accountID {
		return nil, fmt.Errorf("chat not found: %s", chatID)
	}

	// Parse chat JID
	chatJID, err := parseJID(chat.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("invalid chat JID: %w", err)
	}

	// Send message
	var providerMsgID string
	if len(msg.Attachments) > 0 {
		// Send media
		resp, err := p.sendMedia(ctx, client, chatJID, msg.Attachments[0], msg.Text)
		if err != nil {
			return nil, fmt.Errorf("failed to send media: %w", err)
		}
		providerMsgID = resp.ID
	} else {
		// Send text
		resp, err := p.sendText(ctx, client, chatJID, msg.Text)
		if err != nil {
			return nil, fmt.Errorf("failed to send text: %w", err)
		}
		providerMsgID = resp.ID
	}

	// Get self JID for is_sender
	selfJID := client.Store.ID
	isSender := selfJID != nil

	// Create message model
	message := &model.Message{
		Object:     "message",
		ChatID:     chatID,
		AccountID:  accountID,
		Provider:   p.Name(),
		ProviderID: providerMsgID,
		Text:       msg.Text,
		SenderID:   selfJID.String(),
		IsSender:   isSender,
		Seen:       true,  // Sent messages are seen by sender
		Delivered:  false, // Will be updated by receipt events
	}

	// Store in database
	message, err = p.messageStore.Create(ctx, message)
	if err != nil {
		return nil, fmt.Errorf("failed to store message: %w", err)
	}

	// Update chat last message
	_ = p.chatStore.UpdateLastMessage(ctx, chatID, &msg.Text)

	// Dispatch webhook
	p.dispatcher.Dispatch(ctx, model.EventMessageSent, message)
	return message, nil
	}

// StartChat creates a new chat by phone number.
func (p *WhatsAppProvider) StartChat(ctx context.Context, accountID string, req adapter.StartChatRequest) (*model.Chat, error) {
	// Get client
	clientVal, ok := p.clients.Load(accountID)
	if !ok {
		return nil, fmt.Errorf("client not connected for account %s", accountID)
	}
	_ = clientVal.(*whatsmeow.Client)

	// Create JID from phone number
	jid := newJIDFromPhone(req.AttendeeIdentifier)

	// Check if chat already exists
	existingChat, err := p.chatStore.GetByProviderID(ctx, accountID, jid.String())
	if err != nil {
		return nil, fmt.Errorf("failed to check existing chat: %w", err)
	}
	if existingChat != nil {
		return existingChat, nil
	}

	// Create chat in database
	chat := &model.Chat{
		Object:     "chat",
		AccountID:  accountID,
		Provider:   p.Name(),
		ProviderID: jid.String(),
		Type:       string(model.ChatTypeOneToOne),
		IsGroup:    false,
		Attendees: []model.Attendee{
			{
				Object:         "attendee",
				AccountID:      accountID,
				Provider:       p.Name(),
				ProviderID:     jid.String(),
				Identifier:     req.AttendeeIdentifier,
				IdentifierType: string(model.IdentifierTypePhoneNumber),
			},
		},
	}

	chat, err = p.chatStore.Create(ctx, chat)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat: %w", err)
	}

	// Send initial message if provided
	if req.Text != "" {
		_, err = p.SendMessage(ctx, accountID, chat.ID, adapter.SendMessageRequest{Text: req.Text})
		if err != nil {
			slog.Warn("failed to send initial message", "error", err)
		}
	}

	// Dispatch webhook
	p.dispatcher.Dispatch(ctx, model.EventChatCreated, chat)

	return chat, nil
}

// ListAttendees returns attendees from PostgreSQL store.
func (p *WhatsAppProvider) ListAttendees(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Attendee], error) {
	return nil, fmt.Errorf("attendee listing not implemented for WhatsApp")
}

func (p *WhatsAppProvider) GetAttendee(ctx context.Context, accountID string, attendeeID string) (*model.Attendee, error) {
	return nil, fmt.Errorf("attendee retrieval not implemented for WhatsApp")
}

// DownloadAttachment downloads media from WhatsApp.
func (p *WhatsAppProvider) DownloadAttachment(ctx context.Context, accountID string, messageID string, attachmentID string) ([]byte, string, error) {
	// Get client
	clientVal, ok := p.clients.Load(accountID)
	if !ok {
		return nil, "", fmt.Errorf("client not connected for account %s", accountID)
	}
	client := clientVal.(*whatsmeow.Client)

	// Get message from store
	message, err := p.messageStore.GetByID(ctx, messageID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get message: %w", err)
	}
	if message == nil || message.AccountID != accountID {
		return nil, "", fmt.Errorf("message not found: %s", messageID)
	}

	// Find attachment
	var attachment *model.Attachment
	for i := range message.Attachments {
		if message.Attachments[i].ID == attachmentID {
			attachment = &message.Attachments[i]
			break
		}
	}
	if attachment == nil {
		return nil, "", fmt.Errorf("attachment not found: %s", attachmentID)
	}

	// Download media using the stored direct path info from metadata
	directPath, ok := message.Metadata["direct_path"].(string)
	if !ok || directPath == "" {
		return nil, "", fmt.Errorf("media direct path not available")
	}

	data, err := p.downloadMedia(ctx, client, directPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download media: %w", err)
	}

	return data, attachment.MimeType, nil
}

// SendEmail is not applicable for WhatsApp.
func (p *WhatsAppProvider) SendEmail(ctx context.Context, accountID string, req adapter.SendEmailRequest) (*model.Email, error) {
	return nil, fmt.Errorf("email sending not supported by WhatsApp provider")
}

// ListEmails is not applicable for WhatsApp.
func (p *WhatsAppProvider) ListEmails(ctx context.Context, accountID string, opts adapter.ListEmailOpts) (*model.PaginatedList[model.Email], error) {
	return nil, fmt.Errorf("email listing not supported by WhatsApp provider")
}

// GetEmail is not applicable for WhatsApp.
func (p *WhatsAppProvider) GetEmail(ctx context.Context, accountID string, emailID string) (*model.Email, error) {
	return nil, fmt.Errorf("email retrieval not supported by WhatsApp provider")
}

// createClient creates a new whatsmeow client for the given account.
func (p *WhatsAppProvider) createClient(ctx context.Context, accountID string) (*whatsmeow.Client, error) {
	return CreateClient(ctx, accountID, p.deviceStorePath)
}

// eventHandler returns an event handler function for the given account.
func (p *WhatsAppProvider) eventHandler(accountID string) func(interface{}) {
	return func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			p.handleMessageEvent(accountID, v)
		case *events.Receipt:
			p.handleReceiptEvent(accountID, v)
		case *events.Connected:
			p.handleConnectedEvent(accountID)
		case *events.Disconnected:
			p.handleDisconnectedEvent(accountID)
		case *events.GroupInfo:
			p.handleGroupInfoEvent(accountID, v)
		case *events.LoggedOut:
			p.handleLoggedOutEvent(accountID)
		case *events.HistorySync:
			p.handleHistorySyncEvent(accountID, v)
		}
	}
}

// handleConnectedEvent updates account status when connected.
func (p *WhatsAppProvider) handleConnectedEvent(accountID string) {
	slog.Info("WhatsApp connected", "account_id", accountID)
	_ = p.accountStore.UpdateStatus(context.Background(), accountID, model.StatusOperational, nil)
	p.dispatcher.Dispatch(context.Background(), model.EventAccountConnected, map[string]any{
		"account_id": accountID,
		"provider":   p.Name(),
	})
}

// handleDisconnectedEvent updates account status when disconnected.
func (p *WhatsAppProvider) handleDisconnectedEvent(accountID string) {
	slog.Warn("WhatsApp disconnected", "account_id", accountID)
	_ = p.accountStore.UpdateStatus(context.Background(), accountID, model.StatusInterrupted, strPtr("Disconnected"))
	p.dispatcher.Dispatch(context.Background(), model.EventAccountDisconnected, map[string]any{
		"account_id": accountID,
		"provider":   p.Name(),
	})
}

// handleLoggedOutEvent handles when the device is logged out.
func (p *WhatsAppProvider) handleLoggedOutEvent(accountID string) {
	slog.Warn("WhatsApp logged out", "account_id", accountID)
	p.clients.Delete(accountID)
	_ = p.accountStore.UpdateStatus(context.Background(), accountID, model.StatusAuthRequired, strPtr("Device logged out"))
}

// strPtr returns a pointer to a string.
func strPtr(s string) *string {
	return &s
}

// Ensure WhatsAppProvider implements adapter.Provider.
var _ adapter.Provider = (*WhatsAppProvider)(nil)
