package email

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"ondapile/internal/adapter"
	"ondapile/internal/model"
	"ondapile/internal/store"
	"ondapile/internal/webhook"

	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/google/uuid"
	mail "github.com/wneessen/go-mail"
)

// Adapter implements the adapter.Provider interface for IMAP/SMTP email.
type Adapter struct {
	store      *store.Store
	dispatcher *webhook.Dispatcher

	// Account ID -> IMAP client mapping
	imapClients sync.Map // map[string]*imapclient.Client
	// Account ID -> SMTP client mapping
	smtpClients sync.Map // map[string]*mail.Client
	// Account ID -> cancel function for IDLE goroutines
	idleCancels sync.Map // map[string]context.CancelFunc
	// Account ID -> credentials cache for reconnection
	credentials sync.Map // map[string]map[string]string
}

// NewAdapter creates a new email adapter.
func NewAdapter(s *store.Store, d *webhook.Dispatcher) *Adapter {
	return &Adapter{
		store:      s,
		dispatcher: d,
	}
}

// Name returns the provider name.
func (a *Adapter) Name() string {
	return "IMAP"
}

// Connect establishes IMAP and SMTP connections for an account.
// Credentials format:
//
//	creds["imap_host"] = "aaven.coreviewspace.com"
//	creds["imap_port"] = "993"
//	creds["imap_username"] = "ryan@usesenseiiwyze.com"
//	creds["imap_password"] = "!278Tq58Sn99Wd41Iy"
//	creds["smtp_host"] = "aaven.coreviewspace.com"
//	creds["smtp_port"] = "587"
//	creds["smtp_username"] = "ryan@usesenseiiwyze.com"
//	creds["smtp_password"] = "!278Tq58Sn99Wd41Iy"
func (a *Adapter) Connect(ctx context.Context, accountID string, creds map[string]string) (*model.Account, error) {
	imapHost := creds["imap_host"]
	imapPort := creds["imap_port"]
	imapUsername := creds["imap_username"]
	imapPassword := creds["imap_password"]

	smtpHost := creds["smtp_host"]
	smtpPort := creds["smtp_port"]
	smtpUsername := creds["smtp_username"]
	smtpPassword := creds["smtp_password"]

	// Validate required credentials
	if imapHost == "" || imapPort == "" || imapUsername == "" || imapPassword == "" {
		return nil, fmt.Errorf("missing required IMAP credentials")
	}
	if smtpHost == "" || smtpPort == "" || smtpUsername == "" || smtpPassword == "" {
		return nil, fmt.Errorf("missing required SMTP credentials")
	}

	// Connect to IMAP
	imapClient, err := ConnectIMAP(imapHost, imapPort, imapUsername, imapPassword)
	if err != nil {
		slog.Error("failed to connect to IMAP", "account_id", accountID, "error", err)
		return nil, fmt.Errorf("imap connection failed: %w", err)
	}

	// Connect to SMTP
	smtpClient, err := ConnectSMTP(smtpHost, smtpPort, smtpUsername, smtpPassword)
	if err != nil {
		slog.Error("failed to connect to SMTP", "account_id", accountID, "error", err)
		imapClient.Close()
		return nil, fmt.Errorf("smtp connection failed: %w", err)
	}

	// Store clients
	a.imapClients.Store(accountID, imapClient)
	a.smtpClients.Store(accountID, smtpClient)
	a.credentials.Store(accountID, creds)

	// Start IDLE goroutine for new email notifications
	idleCtx, cancel := context.WithCancel(context.Background())
	a.idleCancels.Store(accountID, cancel)
	go a.idleLoop(idleCtx, accountID, imapClient)

	// Update account status to OPERATIONAL and return the DB record
	accountStore := store.NewAccountStore(a.store)
	_ = accountStore.UpdateStatus(ctx, accountID, model.StatusOperational, nil)

	// Try to load the full account from DB
	account, err := accountStore.GetByID(ctx, accountID)
	if err != nil || account == nil {
		// Fallback: return a constructed account
		return &model.Account{
			Object:       "account",
			ID:           accountID,
			Provider:     "IMAP",
			Name:         imapUsername,
			Identifier:   imapUsername,
			Status:       model.StatusOperational,
			Capabilities: []string{"email", "receive", "send", "attachments"},
			CreatedAt:    time.Now(),
			Metadata: map[string]any{
				"imap_host": imapHost,
				"imap_port": imapPort,
				"smtp_host": smtpHost,
				"smtp_port": smtpPort,
			},
		}, nil
	}

	return account, nil
}

// Disconnect closes IMAP and SMTP connections for an account.
func (a *Adapter) Disconnect(ctx context.Context, accountID string) error {
	// Stop IDLE goroutine
	if cancel, ok := a.idleCancels.Load(accountID); ok {
		cancel.(context.CancelFunc)()
		a.idleCancels.Delete(accountID)
	}

	// Close IMAP connection
	if client, ok := a.imapClients.Load(accountID); ok {
		client.(*imapclient.Client).Close()
		a.imapClients.Delete(accountID)
	}

	// Close SMTP connection
	if client, ok := a.smtpClients.Load(accountID); ok {
		client.(*mail.Client).Close()
		a.smtpClients.Delete(accountID)
	}

	// Clear credentials
	a.credentials.Delete(accountID)

	return nil
}

// Reconnect re-establishes IMAP and SMTP connections for an account.
func (a *Adapter) Reconnect(ctx context.Context, accountID string) (*model.Account, error) {
	// Disconnect first
	if err := a.Disconnect(ctx, accountID); err != nil {
		slog.Warn("error during disconnect in reconnect", "account_id", accountID, "error", err)
	}

	// Get cached credentials
	credsVal, ok := a.credentials.Load(accountID)
	if !ok {
		return nil, fmt.Errorf("no cached credentials for account %s", accountID)
	}
	creds := credsVal.(map[string]string)

	// Reconnect
	return a.Connect(ctx, accountID, creds)
}

// Status returns the current connection status for an account.
func (a *Adapter) Status(ctx context.Context, accountID string) (model.AccountStatus, error) {
	// Check if IMAP client exists and is connected
	if client, ok := a.imapClients.Load(accountID); ok {
		imapClient := client.(*imapclient.Client)
		// Try a NOOP to check if connection is alive
		if err := imapClient.Noop(); err != nil {
			return model.StatusInterrupted, nil
		}
		return model.StatusOperational, nil
	}

	return model.StatusInterrupted, nil
}

// GetAuthChallenge returns nil for email (uses credentials, not QR).
func (a *Adapter) GetAuthChallenge(ctx context.Context, accountID string) (*adapter.AuthChallenge, error) {
	// Email uses credentials, not QR codes or OAuth
	return nil, nil
}

// SolveCheckpoint is not applicable for email.
func (a *Adapter) SolveCheckpoint(ctx context.Context, accountID string, solution string) error {
	return fmt.Errorf("checkpoint solving not applicable for email provider")
}

// ListChats returns email threads as chats.
// Each unique From/To combination is treated as a chat thread.
func (a *Adapter) ListChats(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Chat], error) {
	// For email, we create chats based on email threads
	// First, list emails from INBOX to build chat list
	emailStore := NewEmailStore(a.store)

	limit := opts.Limit
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	emails, cursor, hasMore, err := emailStore.ListEmails(ctx, accountID, model.FolderInbox, opts.Cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list emails for chats: %w", err)
	}

	// Group emails by thread ID
	threadMap := make(map[string]*model.Chat)
	for _, email := range emails {
		threadID := ""
		if email.ProviderID != nil && email.ProviderID.ThreadID != "" {
			threadID = email.ProviderID.ThreadID
		} else {
			// Use from attendee as thread identifier for one-to-one emails
			if email.FromAttendee != nil {
				threadID = email.FromAttendee.Identifier
			}
		}

		if threadID == "" {
			continue
		}

		if _, exists := threadMap[threadID]; !exists {
			// Create new chat for this thread
			chatName := email.Subject
			if chatName == "" {
				chatName = "No Subject"
			}

			chat := &model.Chat{
				Object:     "chat",
				ID:         uuid.New().String(),
				AccountID:  accountID,
				Provider:   "IMAP",
				ProviderID: threadID,
				Type:       string(model.ChatTypeOneToOne),
				Name:       &chatName,
				Attendees:  []model.Attendee{},
				IsGroup:    false,
				CreatedAt:  email.Date,
				UpdatedAt:  email.Date,
			}

			// Add attendees from email
			if email.FromAttendee != nil {
				chat.Attendees = append(chat.Attendees, model.Attendee{
					Object:         "attendee",
					ID:             uuid.New().String(),
					AccountID:      accountID,
					Provider:       "IMAP",
					ProviderID:     email.FromAttendee.Identifier,
					Name:           email.FromAttendee.DisplayName,
					Identifier:     email.FromAttendee.Identifier,
					IdentifierType: string(model.IdentifierTypeEmailAddress),
					CreatedAt:      email.Date,
					UpdatedAt:      email.Date,
				})
			}

			for _, to := range email.ToAttendees {
				chat.Attendees = append(chat.Attendees, model.Attendee{
					Object:         "attendee",
					ID:             uuid.New().String(),
					AccountID:      accountID,
					Provider:       "IMAP",
					ProviderID:     to.Identifier,
					Name:           to.DisplayName,
					Identifier:     to.Identifier,
					IdentifierType: string(model.IdentifierTypeEmailAddress),
					CreatedAt:      email.Date,
					UpdatedAt:      email.Date,
				})
			}

			threadMap[threadID] = chat
		}

		// Update unread count
		if !email.Read {
			threadMap[threadID].UnreadCount++
		}

		// Update last message preview
		if threadMap[threadID].LastMessage == nil || email.Date.After(threadMap[threadID].UpdatedAt) {
			preview := email.Subject
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			threadMap[threadID].LastMessage = &model.MessagePreview{
				Text:      preview,
				Timestamp: email.Date,
			}
			threadMap[threadID].UpdatedAt = email.Date
		}
	}

	// Convert map to slice
	chats := make([]model.Chat, 0, len(threadMap))
	for _, chat := range threadMap {
		chats = append(chats, *chat)
	}

	return model.NewPaginatedList(chats, cursor, hasMore), nil
}

// GetChat returns a chat by ID (for email, this is a thread identifier).
func (a *Adapter) GetChat(ctx context.Context, accountID string, chatID string) (*model.Chat, error) {
	// For email, chatID is treated as a thread identifier or email address
	// We fetch emails related to this thread/contact
	emailStore := NewEmailStore(a.store)

	emails, _, _, err := emailStore.ListEmails(ctx, accountID, model.FolderInbox, "", 1)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}

	// Find the first email matching the chat ID
	for _, email := range emails {
		threadID := ""
		if email.ProviderID != nil {
			threadID = email.ProviderID.ThreadID
		}
		if threadID == "" && email.FromAttendee != nil {
			threadID = email.FromAttendee.Identifier
		}

		if threadID == chatID {
			chatName := email.Subject
			if chatName == "" {
				chatName = "No Subject"
			}

			return &model.Chat{
				Object:     "chat",
				ID:         chatID,
				AccountID:  accountID,
				Provider:   "IMAP",
				ProviderID: chatID,
				Type:       string(model.ChatTypeOneToOne),
				Name:       &chatName,
				CreatedAt:  email.Date,
				UpdatedAt:  email.Date,
			}, nil
		}
	}

	return nil, fmt.Errorf("chat not found")
}

// ListMessages returns messages in a chat.
// For email, this returns emails from the thread as messages.
func (a *Adapter) ListMessages(ctx context.Context, accountID string, chatID string, opts adapter.ListOpts) (*model.PaginatedList[model.Message], error) {
	// For email, we list emails that belong to this thread/chat
	emailStore := NewEmailStore(a.store)

	limit := opts.Limit
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	emails, cursor, hasMore, err := emailStore.ListEmails(ctx, accountID, model.FolderInbox, opts.Cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	messages := make([]model.Message, 0)
	for _, email := range emails {
		// Check if email belongs to this chat (thread)
		threadID := ""
		if email.ProviderID != nil {
			threadID = email.ProviderID.ThreadID
		}
		if threadID == "" && email.FromAttendee != nil {
			threadID = email.FromAttendee.Identifier
		}

		if threadID != chatID {
			continue
		}

		msg := a.emailToMessage(email, chatID)
		messages = append(messages, msg)
	}

	return model.NewPaginatedList(messages, cursor, hasMore), nil
}

// SendMessage sends a message in a chat (sends an email reply).
func (a *Adapter) SendMessage(ctx context.Context, accountID string, chatID string, msg adapter.SendMessageRequest) (*model.Message, error) {
	// Get the chat to find recipients
	chat, err := a.GetChat(ctx, accountID, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}

	// Build recipients from chat attendees
	var to []model.EmailAttendee
	for _, attendee := range chat.Attendees {
		if !attendee.IsSelf {
			to = append(to, model.EmailAttendee{
				DisplayName:    attendee.Name,
				Identifier:     attendee.Identifier,
				IdentifierType: attendee.IdentifierType,
			})
		}
	}

	if len(to) == 0 {
		return nil, fmt.Errorf("no recipients found in chat")
	}

	// Send email
	req := adapter.SendEmailRequest{
		To:          to,
		Subject:     *chat.Name,
		BodyPlain:   msg.Text,
		Attachments: msg.Attachments,
	}

	email, err := a.SendEmail(ctx, accountID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Convert to message
	message := a.emailToMessage(*email, chatID)
	return &message, nil
}

// StartChat creates a new email thread by sending the first message.
func (a *Adapter) StartChat(ctx context.Context, accountID string, req adapter.StartChatRequest) (*model.Chat, error) {
	// StartChat for email means sending an email to a new recipient
	emailStore := NewEmailStore(a.store)

	// Create the chat first
	chatID := uuid.New().String()
	now := time.Now()

	chat := &model.Chat{
		Object:     "chat",
		ID:         chatID,
		AccountID:  accountID,
		Provider:   "IMAP",
		ProviderID: req.AttendeeIdentifier,
		Type:       string(model.ChatTypeOneToOne),
		Name:       &req.Text, // Use first message as subject
		Attendees: []model.Attendee{
			{
				Object:         "attendee",
				ID:             uuid.New().String(),
				AccountID:      accountID,
				Provider:       "IMAP",
				ProviderID:     req.AttendeeIdentifier,
				Identifier:     req.AttendeeIdentifier,
				IdentifierType: string(model.IdentifierTypeEmailAddress),
				CreatedAt:      now,
				UpdatedAt:      now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Send the initial email
	sendReq := adapter.SendEmailRequest{
		To: []model.EmailAttendee{
			{
				Identifier:     req.AttendeeIdentifier,
				IdentifierType: string(model.IdentifierTypeEmailAddress),
			},
		},
		Subject:   req.Text,
		BodyPlain: req.Text,
	}

	_, err := a.SendEmail(ctx, accountID, sendReq)
	if err != nil {
		return nil, fmt.Errorf("failed to start chat: %w", err)
	}

	// Store the chat reference (in a real implementation, we'd store this in DB)
	_ = emailStore

	return chat, nil
}

// SendEmail sends an email with full control over recipients, subject, body, and attachments.
func (a *Adapter) SendEmail(ctx context.Context, accountID string, req adapter.SendEmailRequest) (*model.Email, error) {
	// Get SMTP client
	clientVal, ok := a.smtpClients.Load(accountID)
	if !ok {
		return nil, fmt.Errorf("SMTP client not connected for account %s", accountID)
	}
	smtpClient := clientVal.(*mail.Client)

	// Send the email via SMTP
	if err := SendEmail(smtpClient, req); err != nil {
		return nil, fmt.Errorf("failed to send email: %w", err)
	}

	// Create email model
	emailID := uuid.New().String()
	now := time.Now()

	email := &model.Email{
		Object:    "email",
		ID:        emailID,
		AccountID: accountID,
		Provider:  "IMAP",
		ProviderID: &model.EmailProviderID{
			MessageID: emailID,
			ThreadID:  req.Subject, // Use subject as thread identifier for sent emails
		},
		Subject:   req.Subject,
		Body:      req.BodyHTML,
		BodyPlain: req.BodyPlain,
		ToAttendees: func() []model.EmailAttendee {
			result := make([]model.EmailAttendee, len(req.To))
			copy(result, req.To)
			return result
		}(),
		CCAttendees: func() []model.EmailAttendee {
			result := make([]model.EmailAttendee, len(req.CC))
			copy(result, req.CC)
			return result
		}(),
		BCCAttendees: func() []model.EmailAttendee {
			result := make([]model.EmailAttendee, len(req.BCC))
			copy(result, req.BCC)
			return result
		}(),
		Date:           now,
		HasAttachments: len(req.Attachments) > 0,
		Attachments: func() []model.EmailAttachment {
			result := make([]model.EmailAttachment, len(req.Attachments))
			for i, att := range req.Attachments {
				result[i] = model.EmailAttachment{
					ID:       uuid.New().String(),
					Filename: att.Filename,
					MimeType: att.MimeType,
					Size:     int64(len(att.Content)),
				}
			}
			return result
		}(),
		Folders:    []string{model.FolderSent},
		Role:       model.FolderSent,
		Read:       true,
		ReadDate:   &now,
		IsComplete: true,
		Metadata:   map[string]any{},
	}

	// Store the sent email
	emailStore := NewEmailStore(a.store)
	if err := emailStore.StoreEmail(ctx, email); err != nil {
		slog.Error("failed to store sent email", "error", err)
		// Don't fail the send if storage fails
	}

	// Dispatch webhook event
	if a.dispatcher != nil {
		a.dispatcher.Dispatch(ctx, model.EventEmailSent, email)
	}

	return email, nil
}

// ListEmails lists emails from a specific folder with optional filters.
func (a *Adapter) ListEmails(ctx context.Context, accountID string, opts adapter.ListEmailOpts) (*model.PaginatedList[model.Email], error) {
	emailStore := NewEmailStore(a.store)

	limit := opts.Limit
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	folder := opts.Folder
	if folder == "" {
		folder = model.FolderInbox
	}

	emails, cursor, hasMore, err := emailStore.ListEmails(ctx, accountID, folder, opts.Cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list emails: %w", err)
	}

	// Apply filters
	filtered := make([]model.Email, 0)
	for _, email := range emails {
		// From filter
		if opts.From != "" && (email.FromAttendee == nil || !strings.Contains(email.FromAttendee.Identifier, opts.From)) {
			continue
		}
		// To filter
		if opts.To != "" {
			found := false
			for _, to := range email.ToAttendees {
				if strings.Contains(to.Identifier, opts.To) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		// Subject filter
		if opts.Subject != "" && !strings.Contains(email.Subject, opts.Subject) {
			continue
		}
		// Date filters
		if opts.Before != nil && email.Date.After(*opts.Before) {
			continue
		}
		if opts.After != nil && email.Date.Before(*opts.After) {
			continue
		}
		// Has attachments filter
		if opts.HasAttach != nil && email.HasAttachments != *opts.HasAttach {
			continue
		}
		// Is read filter
		if opts.IsRead != nil && email.Read != *opts.IsRead {
			continue
		}

		filtered = append(filtered, email)
	}

	return model.NewPaginatedList(filtered, cursor, hasMore), nil
}

// GetEmail gets a single email by ID.
func (a *Adapter) GetEmail(ctx context.Context, accountID string, emailID string) (*model.Email, error) {
	emailStore := NewEmailStore(a.store)
	return emailStore.GetEmail(ctx, emailID)
}

// ReplyEmail replies to an existing email, setting threading headers.
func (a *Adapter) ReplyEmail(ctx context.Context, accountID string, emailID string, req adapter.SendEmailRequest) (*model.Email, error) {
	// Get original email for threading
	original, err := a.GetEmail(ctx, accountID, emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get original email: %w", err)
	}

	// Set threading headers from original
	if original.ProviderID != nil && original.ProviderID.MessageID != "" {
		req.InReplyTo = original.ProviderID.MessageID
		req.References = original.ProviderID.MessageID
	}

	// Default subject to Re: original subject
	if req.Subject == "" {
		req.Subject = "Re: " + original.Subject
	}

	// Default To to original sender
	if len(req.To) == 0 && original.FromAttendee != nil {
		req.To = []model.EmailAttendee{*original.FromAttendee}
	}

	return a.SendEmail(ctx, accountID, req)
}

// ForwardEmail forwards an existing email to new recipients.
func (a *Adapter) ForwardEmail(ctx context.Context, accountID string, emailID string, req adapter.SendEmailRequest) (*model.Email, error) {
	// Get original email
	original, err := a.GetEmail(ctx, accountID, emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get original email: %w", err)
	}

	// Default subject to Fwd: original subject
	if req.Subject == "" {
		req.Subject = "Fwd: " + original.Subject
	}

	// Include original body as quoted content
	fwdHeader := fmt.Sprintf("\n\n---------- Forwarded message ----------\nFrom: %s\nSubject: %s\nDate: %s\n\n",
		original.FromAttendee.DisplayName+" <"+original.FromAttendee.Identifier+">",
		original.Subject,
		original.Date.Format(time.RFC1123),
	)
	if req.BodyHTML != "" {
		req.BodyHTML = req.BodyHTML + fwdHeader + original.Body
	} else {
		req.BodyHTML = fwdHeader + original.Body
	}

	return a.SendEmail(ctx, accountID, req)
}

// UpdateEmailProvider syncs email changes (read status, folder) to the IMAP server.
func (a *Adapter) UpdateEmailProvider(ctx context.Context, accountID string, emailID string, opts adapter.UpdateEmailOpts) error {
	clientVal, ok := a.imapClients.Load(accountID)
	if !ok {
		return fmt.Errorf("IMAP client not connected for account %s", accountID)
	}
	client := clientVal.(*imapclient.Client)

	// Get the email to find UID and folder
	email, err := a.GetEmail(ctx, accountID, emailID)
	if err != nil {
		return fmt.Errorf("failed to get email: %w", err)
	}

	// Extract UID from metadata
	uid, _ := email.Metadata["uid"].(float64)
	if uid == 0 {
		return fmt.Errorf("email has no IMAP UID")
	}

	currentFolder := model.FolderInbox
	if len(email.Folders) > 0 {
		currentFolder = email.Folders[0]
	}

	// Update read status
	if opts.Read != nil && *opts.Read {
		if err := MarkAsSeen(client, currentFolder, uint32(uid)); err != nil {
			slog.Error("failed to mark as seen", "error", err)
		}
	}

	// Move to folder
	if opts.Folder != nil && *opts.Folder != currentFolder {
		if err := MoveMessage(client, currentFolder, *opts.Folder, uint32(uid)); err != nil {
			return fmt.Errorf("failed to move message: %w", err)
		}
	}

	return nil
}

// DeleteEmailProvider deletes an email from the IMAP server.
func (a *Adapter) DeleteEmailProvider(ctx context.Context, accountID string, emailID string) error {
	clientVal, ok := a.imapClients.Load(accountID)
	if !ok {
		return fmt.Errorf("IMAP client not connected for account %s", accountID)
	}
	client := clientVal.(*imapclient.Client)

	// Get the email to find UID and folder
	email, err := a.GetEmail(ctx, accountID, emailID)
	if err != nil {
		return fmt.Errorf("failed to get email: %w", err)
	}

	// Extract UID from metadata
	uid, _ := email.Metadata["uid"].(float64)
	if uid == 0 {
		return fmt.Errorf("email has no IMAP UID")
	}

	currentFolder := model.FolderInbox
	if len(email.Folders) > 0 {
		currentFolder = email.Folders[0]
	}

	return DeleteMessage(client, currentFolder, uint32(uid))
}

// ListFolders lists all mailbox folders from the IMAP server.
func (a *Adapter) ListFolders(ctx context.Context, accountID string) ([]string, error) {
	clientVal, ok := a.imapClients.Load(accountID)
	if !ok {
		return nil, fmt.Errorf("IMAP client not connected for account %s", accountID)
	}
	client := clientVal.(*imapclient.Client)

	return ListMailboxes(client)
}

// ListAttendees lists all attendees (email contacts) for an account.
func (a *Adapter) ListAttendees(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Attendee], error) {
	// For email, we extract attendees from stored emails
	emailStore := NewEmailStore(a.store)

	limit := opts.Limit
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	// Get recent emails to extract contacts
	emails, _, _, err := emailStore.ListEmails(ctx, accountID, model.FolderInbox, opts.Cursor, limit*2)
	if err != nil {
		return nil, fmt.Errorf("failed to list attendees: %w", err)
	}

	// Build unique attendee map
	attendeeMap := make(map[string]*model.Attendee)
	for _, email := range emails {
		if email.FromAttendee != nil {
			id := email.FromAttendee.Identifier
			if _, exists := attendeeMap[id]; !exists {
				attendeeMap[id] = &model.Attendee{
					Object:         "attendee",
					ID:             uuid.New().String(),
					AccountID:      accountID,
					Provider:       "IMAP",
					ProviderID:     id,
					Name:           email.FromAttendee.DisplayName,
					Identifier:     id,
					IdentifierType: string(model.IdentifierTypeEmailAddress),
					CreatedAt:      email.Date,
					UpdatedAt:      email.Date,
				}
			}
		}
		for _, to := range email.ToAttendees {
			id := to.Identifier
			if _, exists := attendeeMap[id]; !exists {
				attendeeMap[id] = &model.Attendee{
					Object:         "attendee",
					ID:             uuid.New().String(),
					AccountID:      accountID,
					Provider:       "IMAP",
					ProviderID:     id,
					Name:           to.DisplayName,
					Identifier:     id,
					IdentifierType: string(model.IdentifierTypeEmailAddress),
					CreatedAt:      email.Date,
					UpdatedAt:      email.Date,
				}
			}
		}
	}

	// Convert to slice
	attendees := make([]model.Attendee, 0, len(attendeeMap))
	for _, att := range attendeeMap {
		attendees = append(attendees, *att)
	}

	return model.NewPaginatedList(attendees, "", false), nil
}

// GetAttendee gets a single attendee by ID.
func (a *Adapter) GetAttendee(ctx context.Context, accountID string, attendeeID string) (*model.Attendee, error) {
	// Try to find attendee by ID in stored emails
	attendees, err := a.ListAttendees(ctx, accountID, adapter.ListOpts{Limit: 100})
	if err != nil {
		return nil, err
	}

	for _, att := range attendees.Items {
		if att.ID == attendeeID {
			return &att, nil
		}
	}

	return nil, fmt.Errorf("attendee not found")
}

// DownloadAttachment downloads an attachment from an email.
func (a *Adapter) DownloadAttachment(ctx context.Context, accountID string, messageID string, attachmentID string) ([]byte, string, error) {
	// Get the email
	email, err := a.GetEmail(ctx, accountID, messageID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get email: %w", err)
	}

	// Find the attachment
	for _, att := range email.Attachments {
		if att.ID == attachmentID {
			// For IMAP, we'd need to fetch the attachment from the server
			// This requires implementing FETCH for body parts
			// For now, return an error indicating this needs implementation
			return nil, att.Filename, fmt.Errorf("attachment download not yet implemented for IMAP")
		}
	}

	return nil, "", fmt.Errorf("attachment not found")
}

// emailToMessage converts an Email model to a Message model.
func (a *Adapter) emailToMessage(email model.Email, chatID string) model.Message {
	senderID := ""
	if email.FromAttendee != nil {
		senderID = email.FromAttendee.Identifier
	}

	// Convert attachments
	attachments := make([]model.Attachment, len(email.Attachments))
	for i, att := range email.Attachments {
		attachments[i] = model.Attachment{
			ID:       att.ID,
			Filename: att.Filename,
			MimeType: att.MimeType,
			Size:     att.Size,
		}
	}

	return model.Message{
		Object:      "message",
		ID:          email.ID,
		ChatID:      chatID,
		AccountID:   email.AccountID,
		Provider:    "IMAP",
		ProviderID:  email.ID,
		Text:        email.BodyPlain,
		SenderID:    senderID,
		IsSender:    email.Role == model.FolderSent,
		Timestamp:   email.Date,
		Attachments: attachments,
		Seen:        email.Read,
		Delivered:   true,
		Edited:      false,
		Deleted:     false,
		Hidden:      false,
		IsEvent:     false,
		Metadata:    email.Metadata,
	}
}

// idleLoop watches for new emails using IMAP IDLE and dispatches webhooks.
func (a *Adapter) idleLoop(ctx context.Context, accountID string, client *imapclient.Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Poll for new messages (IMAP IDLE support varies by server)
			if err := a.pollNewMessages(ctx, accountID, client); err != nil {
				slog.Error("failed to poll for new messages", "account_id", accountID, "error", err)
			}
		}
	}
}

// pollNewMessages checks for and processes new messages in the INBOX.
func (a *Adapter) pollNewMessages(ctx context.Context, accountID string, client *imapclient.Client) error {
	// Fetch recent messages from INBOX
	emails, err := FetchMessages(client, model.FolderInbox, 10, 0)
	if err != nil {
		return fmt.Errorf("failed to fetch messages: %w", err)
	}

	emailStore := NewEmailStore(a.store)

	for _, email := range emails {
		// Check if we already have this email
		if email.ProviderID == nil {
			// No message ID, skip
			continue
		}
		existing, err := emailStore.GetEmailByProviderID(ctx, accountID, email.ProviderID.MessageID)
		if err != nil {
			slog.Error("failed to check existing email", "error", err)
			continue
		}
		if existing != nil {
			// Already processed
			continue
		}

		// Store the new email
		email.AccountID = accountID
		if err := emailStore.StoreEmail(ctx, &email); err != nil {
			slog.Error("failed to store new email", "error", err)
			continue
		}

		// Dispatch email.received webhook
		if a.dispatcher != nil {
			a.dispatcher.Dispatch(ctx, model.EventEmailReceived, &email)
		}

		// Get or create chat for this email
		threadID := ""
		if email.ProviderID != nil {
			threadID = email.ProviderID.ThreadID
		}
		if threadID == "" && email.FromAttendee != nil {
			threadID = email.FromAttendee.Identifier
		}

		chatStore := store.NewChatStore(a.store)
		chat, err := chatStore.GetByProviderID(ctx, accountID, threadID)
		if err != nil {
			slog.Error("failed to get chat", "error", err)
		}

		if chat == nil {
			// Create new chat
			chatName := email.Subject
			if chatName == "" {
				chatName = "No Subject"
			}
			chat = &model.Chat{
				Object:      "chat",
				ID:          uuid.New().String(),
				AccountID:   accountID,
				Provider:    "IMAP",
				ProviderID:  threadID,
				Type:        string(model.ChatTypeOneToOne),
				Name:        &chatName,
				UnreadCount: 1,
			}
			chat, err = chatStore.Create(ctx, chat)
			if err != nil {
				slog.Error("failed to create chat", "error", err)
			}
		} else {
			// Increment unread count
			if err := chatStore.IncrementUnread(ctx, chat.ID); err != nil {
				slog.Error("failed to increment unread", "error", err)
			}
		}

		// Update last message preview
		if chat != nil {
			preview := email.Subject
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			if err := chatStore.UpdateLastMessage(ctx, chat.ID, &preview); err != nil {
				slog.Error("failed to update last message", "error", err)
			}
		}

		// Dispatch webhook events
		if a.dispatcher != nil {
			a.dispatcher.Dispatch(ctx, model.EventEmailReceived, &email)

			// Also dispatch as message received for chat compatibility
			if chat != nil {
				message := a.emailToMessage(email, chat.ID)
				a.dispatcher.Dispatch(ctx, model.EventMessageReceived, &message)
			}
		}
	}

	return nil
}

// ListCalendars is not supported by IMAP email.
func (a *Adapter) ListCalendars(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Calendar], error) {
	return nil, adapter.ErrNotSupported
}

// GetCalendar is not supported by IMAP email.
func (a *Adapter) GetCalendar(ctx context.Context, accountID string, calendarID string) (*model.Calendar, error) {
	return nil, adapter.ErrNotSupported
}

// ListEvents is not supported by IMAP email.
func (a *Adapter) ListEvents(ctx context.Context, accountID string, calendarID string, opts adapter.ListOpts) (*model.PaginatedList[model.CalendarEvent], error) {
	return nil, adapter.ErrNotSupported
}

// GetEvent is not supported by IMAP email.
func (a *Adapter) GetEvent(ctx context.Context, accountID string, calendarID string, eventID string) (*model.CalendarEvent, error) {
	return nil, adapter.ErrNotSupported
}

// CreateEvent is not supported by IMAP email.
func (a *Adapter) CreateEvent(ctx context.Context, accountID string, calendarID string, req adapter.CreateEventRequest) (*model.CalendarEvent, error) {
	return nil, adapter.ErrNotSupported
}

// UpdateEvent is not supported by IMAP email.
func (a *Adapter) UpdateEvent(ctx context.Context, accountID string, calendarID string, eventID string, req adapter.UpdateEventRequest) (*model.CalendarEvent, error) {
	return nil, adapter.ErrNotSupported
}

// DeleteEvent is not supported by IMAP email.
func (a *Adapter) DeleteEvent(ctx context.Context, accountID string, calendarID string, eventID string) error {
	return adapter.ErrNotSupported
}

// SupportsOAuth returns false as IMAP email uses credentials-based authentication.
func (a *Adapter) SupportsOAuth() bool {
	return false
}

// GetOAuthURL is not supported by IMAP email.
func (a *Adapter) GetOAuthURL(ctx context.Context, state string) (string, error) {
	return "", adapter.ErrNotSupported
}

// HandleOAuthCallback is not supported by IMAP email.
func (a *Adapter) HandleOAuthCallback(ctx context.Context, code string) (map[string]string, error) {
	return nil, adapter.ErrNotSupported
}

// Ensure Adapter implements adapter.Provider.
var _ adapter.Provider = (*Adapter)(nil)
