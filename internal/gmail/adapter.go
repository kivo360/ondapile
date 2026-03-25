package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"ondapile/internal/adapter"
	"ondapile/internal/model"
	"ondapile/internal/oauth"
	"ondapile/internal/store"
	"ondapile/internal/webhook"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// GmailAdapter implements the adapter.Provider interface for Gmail.
type GmailAdapter struct {
	oauthCfg   *oauth2.Config
	tokenStore *oauth.TokenStore
	store      *store.Store
	dispatcher *webhook.Dispatcher
}

// NewAdapter creates a new Gmail adapter.
func NewAdapter(oauthCfg *oauth2.Config, tokenStore *oauth.TokenStore, s *store.Store, d *webhook.Dispatcher) *GmailAdapter {
	return &GmailAdapter{
		oauthCfg:   oauthCfg,
		tokenStore: tokenStore,
		store:      s,
		dispatcher: d,
	}
}

// Name returns the provider identifier.
func (a *GmailAdapter) Name() string {
	return "GMAIL"
}

// SupportsOAuth returns true as Gmail uses OAuth2.
func (a *GmailAdapter) SupportsOAuth() bool {
	return true
}

// GetOAuthURL generates the OAuth URL for Gmail authentication.
func (a *GmailAdapter) GetOAuthURL(ctx context.Context, state string) (string, error) {
	if a.oauthCfg == nil {
		return "", fmt.Errorf("OAuth config not initialized")
	}

	return a.oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent")), nil
}

// HandleOAuthCallback exchanges the authorization code for a token.
func (a *GmailAdapter) HandleOAuthCallback(ctx context.Context, code string) (map[string]string, error) {
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

// Connect establishes a Gmail connection using stored OAuth tokens.
func (a *GmailAdapter) Connect(ctx context.Context, accountID string, creds map[string]string) (*model.Account, error) {
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
	if err := gmailGet(client, "/profile", &profile); err != nil {
		return nil, fmt.Errorf("failed to verify Gmail connection: %w", err)
	}

	emailAddress, _ := profile["emailAddress"].(string)

	return &model.Account{
		Object:       "account",
		ID:           accountID,
		Provider:     a.Name(),
		Name:         emailAddress,
		Identifier:   emailAddress,
		Status:       model.StatusOperational,
		Capabilities: []string{"email", "send", "receive", "attachments"},
	}, nil
}

// Disconnect closes the Gmail connection.
func (a *GmailAdapter) Disconnect(ctx context.Context, accountID string) error {
	return nil
}

// Reconnect re-establishes the Gmail connection.
func (a *GmailAdapter) Reconnect(ctx context.Context, accountID string) (*model.Account, error) {
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
func (a *GmailAdapter) Status(ctx context.Context, accountID string) (model.AccountStatus, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return model.StatusInterrupted, nil
	}

	var profile map[string]interface{}
	if err := gmailGet(client, "/profile", &profile); err != nil {
		return model.StatusInterrupted, nil
	}

	return model.StatusOperational, nil
}

// GetAuthChallenge returns nil as Gmail uses OAuth.
func (a *GmailAdapter) GetAuthChallenge(ctx context.Context, accountID string) (*adapter.AuthChallenge, error) {
	return nil, nil
}

// SolveCheckpoint is not applicable for Gmail.
func (a *GmailAdapter) SolveCheckpoint(ctx context.Context, accountID string, solution string) error {
	return adapter.ErrNotSupported
}

// ListChats is not supported by Gmail.
func (a *GmailAdapter) ListChats(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Chat], error) {
	return nil, adapter.ErrNotSupported
}

// GetChat is not supported by Gmail.
func (a *GmailAdapter) GetChat(ctx context.Context, accountID string, chatID string) (*model.Chat, error) {
	return nil, adapter.ErrNotSupported
}

// ListMessages is not supported by Gmail.
func (a *GmailAdapter) ListMessages(ctx context.Context, accountID string, chatID string, opts adapter.ListOpts) (*model.PaginatedList[model.Message], error) {
	return nil, adapter.ErrNotSupported
}

// SendMessage is not supported by Gmail.
func (a *GmailAdapter) SendMessage(ctx context.Context, accountID string, chatID string, msg adapter.SendMessageRequest) (*model.Message, error) {
	return nil, adapter.ErrNotSupported
}

// StartChat is not supported by Gmail.
func (a *GmailAdapter) StartChat(ctx context.Context, accountID string, req adapter.StartChatRequest) (*model.Chat, error) {
	return nil, adapter.ErrNotSupported
}

// ListAttendees is not supported by Gmail.
func (a *GmailAdapter) ListAttendees(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Attendee], error) {
	return nil, adapter.ErrNotSupported
}

// GetAttendee is not supported by Gmail.
func (a *GmailAdapter) GetAttendee(ctx context.Context, accountID string, attendeeID string) (*model.Attendee, error) {
	return nil, adapter.ErrNotSupported
}

// DownloadAttachment downloads an attachment from Gmail.
func (a *GmailAdapter) DownloadAttachment(ctx context.Context, accountID string, messageID string, attachmentID string) ([]byte, string, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create HTTP client: %w", err)
	}

	path := fmt.Sprintf("/messages/%s/attachments/%s", messageID, attachmentID)

	var result map[string]interface{}
	if err := gmailGet(client, path, &result); err != nil {
		return nil, "", fmt.Errorf("failed to download attachment: %w", err)
	}

	data, _ := result["data"].(string)
	decoded, err := base64.URLEncoding.DecodeString(data)
	if err != nil {
		decoded, err = base64.StdEncoding.DecodeString(data)
		if err != nil {
			return nil, "", fmt.Errorf("failed to decode attachment: %w", err)
		}
	}

	return decoded, "application/octet-stream", nil
}

// SendEmail sends an email via Gmail.
func (a *GmailAdapter) SendEmail(ctx context.Context, accountID string, req adapter.SendEmailRequest) (*model.Email, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	var msg strings.Builder

	msg.WriteString("To: ")
	for i, to := range req.To {
		if i > 0 {
			msg.WriteString(", ")
		}
		if to.DisplayName != "" {
			msg.WriteString(fmt.Sprintf("\"%s\" <%s>", to.DisplayName, to.Identifier))
		} else {
			msg.WriteString(to.Identifier)
		}
	}
	msg.WriteString("\r\n")

	if len(req.CC) > 0 {
		msg.WriteString("Cc: ")
		for i, cc := range req.CC {
			if i > 0 {
				msg.WriteString(", ")
			}
			if cc.DisplayName != "" {
				msg.WriteString(fmt.Sprintf("\"%s\" <%s>", cc.DisplayName, cc.Identifier))
			} else {
				msg.WriteString(cc.Identifier)
			}
		}
		msg.WriteString("\r\n")
	}

	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", req.Subject))
	msg.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	msg.WriteString("\r\n")

	if req.BodyHTML != "" {
		msg.WriteString(req.BodyHTML)
	} else {
		msg.WriteString(req.BodyPlain)
	}

	raw := base64.URLEncoding.EncodeToString([]byte(msg.String()))

	body := map[string]interface{}{
		"raw": raw,
	}

	var result map[string]interface{}
	if err := gmailPost(client, "/messages/send", body, &result); err != nil {
		return nil, fmt.Errorf("failed to send email: %w", err)
	}

	emailID := uuid.New().String()
	messageID, _ := result["id"].(string)
	threadID, _ := result["threadId"].(string)

	email := &model.Email{
		Object:    "email",
		ID:        emailID,
		AccountID: accountID,
		Provider:  a.Name(),
		ProviderID: &model.EmailProviderID{
			MessageID: messageID,
			ThreadID:  threadID,
		},
		Subject:    req.Subject,
		Body:       req.BodyHTML,
		BodyPlain:  req.BodyPlain,
		Date:       time.Now(),
		Folders:    []string{model.FolderSent},
		Role:       model.FolderSent,
		Read:       true,
		IsComplete: true,
	}

	for _, to := range req.To {
		email.ToAttendees = append(email.ToAttendees, model.EmailAttendee{
			DisplayName:    to.DisplayName,
			Identifier:     to.Identifier,
			IdentifierType: string(model.IdentifierTypeEmailAddress),
		})
	}

	if a.dispatcher != nil {
		a.dispatcher.Dispatch(ctx, model.EventEmailSent, email)
	}

	return email, nil
}

// ListEmails lists emails from Gmail.
func (a *GmailAdapter) ListEmails(ctx context.Context, accountID string, opts adapter.ListEmailOpts) (*model.PaginatedList[model.Email], error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	query := buildGmailQuery(opts)

	params := []string{}
	if query != "" {
		params = append(params, fmt.Sprintf("q=%s", query))
	}
	if opts.Cursor != "" {
		params = append(params, fmt.Sprintf("pageToken=%s", opts.Cursor))
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	params = append(params, fmt.Sprintf("maxResults=%d", limit))
	path := fmt.Sprintf("/messages?%s", strings.Join(params, "&"))

	var result map[string]interface{}
	if err := gmailGet(client, path, &result); err != nil {
		return nil, fmt.Errorf("failed to list emails: %w", err)
	}

	emails, nextPageToken := normalizeEmailList(result, accountID)

	var fullEmails []model.Email
	for _, email := range emails {
		fullEmail, err := a.GetEmail(ctx, accountID, email.ProviderID.MessageID)
		if err != nil {
			continue
		}
		if fullEmail != nil {
			fullEmails = append(fullEmails, *fullEmail)
		}
	}

	hasMore := nextPageToken != ""
	return model.NewPaginatedList(fullEmails, nextPageToken, hasMore), nil
}

// GetEmail gets a single email by ID from Gmail.
func (a *GmailAdapter) GetEmail(ctx context.Context, accountID string, emailID string) (*model.Email, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	path := fmt.Sprintf("/messages/%s", emailID)

	var result map[string]interface{}
	if err := gmailGet(client, path, &result); err != nil {
		return nil, fmt.Errorf("failed to get email: %w", err)
	}

	email := normalizeEmail(result, accountID)
	return email, nil
}

// ListFolders returns the list of Gmail labels.
func (a *GmailAdapter) ListFolders(ctx context.Context, accountID string) ([]string, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	var result map[string]interface{}
	if err := gmailGet(client, "/labels", &result); err != nil {
		return nil, fmt.Errorf("failed to list labels: %w", err)
	}

	var folders []string
	if labels, ok := result["labels"].([]interface{}); ok {
		for _, l := range labels {
			if label, ok := l.(map[string]interface{}); ok {
				if name, ok := label["name"].(string); ok {
					folders = append(folders, name)
				}
			}
		}
	}

	return folders, nil
}

// DeleteEmailProvider moves an email to trash in Gmail.
func (a *GmailAdapter) DeleteEmailProvider(ctx context.Context, accountID string, emailID string) error {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	if err := gmailPost(client, "/messages/"+emailID+"/trash", nil, nil); err != nil {
		return fmt.Errorf("failed to move email to trash: %w", err)
	}

	return nil
}

// UpdateEmailProvider updates email labels in Gmail (read, starred, folder).
func (a *GmailAdapter) UpdateEmailProvider(ctx context.Context, accountID string, emailID string, opts adapter.UpdateEmailOpts) error {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	var addLabels, removeLabels []string

	if opts.Read != nil {
		if *opts.Read {
			removeLabels = append(removeLabels, "UNREAD")
		} else {
			addLabels = append(addLabels, "UNREAD")
		}
	}

	if opts.Starred != nil {
		if *opts.Starred {
			addLabels = append(addLabels, "STARRED")
		} else {
			removeLabels = append(removeLabels, "STARRED")
		}
	}

	if opts.Folder != nil {
		addLabels = append(addLabels, *opts.Folder)
		removeLabels = append(removeLabels, "INBOX")
	}

	body := map[string]interface{}{}
	if len(addLabels) > 0 {
		body["addLabelIds"] = addLabels
	}
	if len(removeLabels) > 0 {
		body["removeLabelIds"] = removeLabels
	}

	if err := gmailPost(client, "/messages/"+emailID+"/modify", body, nil); err != nil {
		return fmt.Errorf("failed to update email: %w", err)
	}

	return nil
}

// ReplyEmail sends a reply to an existing email in Gmail.
func (a *GmailAdapter) ReplyEmail(ctx context.Context, accountID string, emailID string, req adapter.SendEmailRequest) (*model.Email, error) {
	// Fetch the original email to get threadId and headers
	original, err := a.GetEmail(ctx, accountID, emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get original email: %w", err)
	}

	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Build the reply message
	var msg strings.Builder

	// Set To header - default to original sender if not provided
	msg.WriteString("To: ")
	if len(req.To) > 0 {
		for i, to := range req.To {
			if i > 0 {
				msg.WriteString(", ")
			}
			if to.DisplayName != "" {
				msg.WriteString(fmt.Sprintf("\"%s\" <%s>", to.DisplayName, to.Identifier))
			} else {
				msg.WriteString(to.Identifier)
			}
		}
	} else if original.FromAttendee != nil {
		if original.FromAttendee.DisplayName != "" {
			msg.WriteString(fmt.Sprintf("\"%s\" <%s>", original.FromAttendee.DisplayName, original.FromAttendee.Identifier))
		} else {
			msg.WriteString(original.FromAttendee.Identifier)
		}
	}
	msg.WriteString("\r\n")

	// Add CC if provided
	if len(req.CC) > 0 {
		msg.WriteString("Cc: ")
		for i, cc := range req.CC {
			if i > 0 {
				msg.WriteString(", ")
			}
			if cc.DisplayName != "" {
				msg.WriteString(fmt.Sprintf("\"%s\" <%s>", cc.DisplayName, cc.Identifier))
			} else {
				msg.WriteString(cc.Identifier)
			}
		}
		msg.WriteString("\r\n")
	}

	// Set Subject - prepend "Re: " if not already present
	subject := req.Subject
	if subject == "" {
		subject = original.Subject
		if !strings.HasPrefix(strings.ToLower(subject), "re: ") {
			subject = "Re: " + subject
		}
	}
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))

	// Set In-Reply-To and References headers
	if original.ProviderID != nil && original.ProviderID.MessageID != "" {
		msg.WriteString(fmt.Sprintf("In-Reply-To: <%s>\r\n", original.ProviderID.MessageID))
		msg.WriteString(fmt.Sprintf("References: <%s>\r\n", original.ProviderID.MessageID))
	}

	msg.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	msg.WriteString("\r\n")

	if req.BodyHTML != "" {
		msg.WriteString(req.BodyHTML)
	} else {
		msg.WriteString(req.BodyPlain)
	}

	raw := base64.URLEncoding.EncodeToString([]byte(msg.String()))

	body := map[string]interface{}{
		"raw": raw,
	}
	if original.ProviderID != nil && original.ProviderID.ThreadID != "" {
		body["threadId"] = original.ProviderID.ThreadID
	}

	var result map[string]interface{}
	if err := gmailPost(client, "/messages/send", body, &result); err != nil {
		return nil, fmt.Errorf("failed to send reply: %w", err)
	}

	newEmailID := uuid.New().String()
	messageID, _ := result["id"].(string)
	threadID, _ := result["threadId"].(string)

	email := &model.Email{
		Object:    "email",
		ID:        newEmailID,
		AccountID: accountID,
		Provider:  a.Name(),
		ProviderID: &model.EmailProviderID{
			MessageID: messageID,
			ThreadID:  threadID,
		},
		Subject:    subject,
		Body:       req.BodyHTML,
		BodyPlain:  req.BodyPlain,
		Date:       time.Now(),
		Folders:    []string{model.FolderSent},
		Role:       model.FolderSent,
		Read:       true,
		IsComplete: true,
	}

	for _, to := range req.To {
		email.ToAttendees = append(email.ToAttendees, model.EmailAttendee{
			DisplayName:    to.DisplayName,
			Identifier:     to.Identifier,
			IdentifierType: string(model.IdentifierTypeEmailAddress),
		})
	}

	if a.dispatcher != nil {
		a.dispatcher.Dispatch(ctx, model.EventEmailSent, email)
	}

	return email, nil
}

// ForwardEmail forwards an existing email in Gmail.
func (a *GmailAdapter) ForwardEmail(ctx context.Context, accountID string, emailID string, req adapter.SendEmailRequest) (*model.Email, error) {
	// Fetch the original email
	original, err := a.GetEmail(ctx, accountID, emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get original email: %w", err)
	}

	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Build the forwarded message
	var msg strings.Builder

	// Set To header (required for forward)
	msg.WriteString("To: ")
	for i, to := range req.To {
		if i > 0 {
			msg.WriteString(", ")
		}
		if to.DisplayName != "" {
			msg.WriteString(fmt.Sprintf("\"%s\" <%s>", to.DisplayName, to.Identifier))
		} else {
			msg.WriteString(to.Identifier)
		}
	}
	msg.WriteString("\r\n")

	// Add CC if provided
	if len(req.CC) > 0 {
		msg.WriteString("Cc: ")
		for i, cc := range req.CC {
			if i > 0 {
				msg.WriteString(", ")
			}
			if cc.DisplayName != "" {
				msg.WriteString(fmt.Sprintf("\"%s\" <%s>", cc.DisplayName, cc.Identifier))
			} else {
				msg.WriteString(cc.Identifier)
			}
		}
		msg.WriteString("\r\n")
	}

	// Set Subject - prepend "Fwd: " if not already present
	subject := req.Subject
	if subject == "" {
		subject = original.Subject
		if !strings.HasPrefix(strings.ToLower(subject), "fwd: ") {
			subject = "Fwd: " + subject
		}
	}
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("Content-Type: text/html; charset=utf-8\r\n")
	msg.WriteString("\r\n")

	// Prepend forwarded message header
	msg.WriteString("<div style=\"margin-bottom: 20px;\">")
	msg.WriteString("---------- Forwarded message ----------<br>\r\n")

	// Add original sender info
	if original.FromAttendee != nil {
		msg.WriteString(fmt.Sprintf("From: %s &lt;%s&gt;<br>\r\n", original.FromAttendee.DisplayName, original.FromAttendee.Identifier))
	}
	msg.WriteString(fmt.Sprintf("Date: %s<br>\r\n", original.Date.Format(time.RFC1123)))
	msg.WriteString(fmt.Sprintf("Subject: %s<br>\r\n", original.Subject))
	msg.WriteString("</div>")
	msg.WriteString("<div style=\"border-left: 2px solid #ccc; padding-left: 10px; margin-left: 10px;\">\r\n")

	// Add the forwarded content
	if req.BodyHTML != "" {
		msg.WriteString(req.BodyHTML)
	} else if req.BodyPlain != "" {
		msg.WriteString(req.BodyPlain)
	}
	msg.WriteString("<br><br>\r\n")

	// Append original body
	if original.Body != "" {
		msg.WriteString(original.Body)
	} else {
		msg.WriteString(original.BodyPlain)
	}
	msg.WriteString("</div>")

	raw := base64.URLEncoding.EncodeToString([]byte(msg.String()))

	body := map[string]interface{}{
		"raw": raw,
	}
	if original.ProviderID != nil && original.ProviderID.ThreadID != "" {
		body["threadId"] = original.ProviderID.ThreadID
	}

	var result map[string]interface{}
	if err := gmailPost(client, "/messages/send", body, &result); err != nil {
		return nil, fmt.Errorf("failed to send forward: %w", err)
	}

	newEmailID := uuid.New().String()
	messageID, _ := result["id"].(string)
	threadID, _ := result["threadId"].(string)

	email := &model.Email{
		Object:    "email",
		ID:        newEmailID,
		AccountID: accountID,
		Provider:  a.Name(),
		ProviderID: &model.EmailProviderID{
			MessageID: messageID,
			ThreadID:  threadID,
		},
		Subject:    subject,
		Body:       msg.String(),
		BodyPlain:  "", // Forward is HTML only
		Date:       time.Now(),
		Folders:    []string{model.FolderSent},
		Role:       model.FolderSent,
		Read:       true,
		IsComplete: true,
	}

	for _, to := range req.To {
		email.ToAttendees = append(email.ToAttendees, model.EmailAttendee{
			DisplayName:    to.DisplayName,
			Identifier:     to.Identifier,
			IdentifierType: string(model.IdentifierTypeEmailAddress),
		})
	}

	if a.dispatcher != nil {
		a.dispatcher.Dispatch(ctx, model.EventEmailSent, email)
	}

	return email, nil
}

// ListCalendars is not supported by Gmail.
func (a *GmailAdapter) ListCalendars(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Calendar], error) {
	return nil, adapter.ErrNotSupported
}

// GetCalendar is not supported by Gmail.
func (a *GmailAdapter) GetCalendar(ctx context.Context, accountID string, calendarID string) (*model.Calendar, error) {
	return nil, adapter.ErrNotSupported
}

// ListEvents is not supported by Gmail.
func (a *GmailAdapter) ListEvents(ctx context.Context, accountID string, calendarID string, opts adapter.ListOpts) (*model.PaginatedList[model.CalendarEvent], error) {
	return nil, adapter.ErrNotSupported
}

// GetEvent is not supported by Gmail.
func (a *GmailAdapter) GetEvent(ctx context.Context, accountID string, calendarID string, eventID string) (*model.CalendarEvent, error) {
	return nil, adapter.ErrNotSupported
}

// CreateEvent is not supported by Gmail.
func (a *GmailAdapter) CreateEvent(ctx context.Context, accountID string, calendarID string, req adapter.CreateEventRequest) (*model.CalendarEvent, error) {
	return nil, adapter.ErrNotSupported
}

// UpdateEvent is not supported by Gmail.
func (a *GmailAdapter) UpdateEvent(ctx context.Context, accountID string, calendarID string, eventID string, req adapter.UpdateEventRequest) (*model.CalendarEvent, error) {
	return nil, adapter.ErrNotSupported
}

// DeleteEvent is not supported by Gmail.
func (a *GmailAdapter) DeleteEvent(ctx context.Context, accountID string, calendarID string, eventID string) error {
	return adapter.ErrNotSupported
}

// Helper functions

func buildGmailQuery(opts adapter.ListEmailOpts) string {
	var parts []string

	if opts.Folder != "" && opts.Folder != model.FolderInbox {
		switch opts.Folder {
		case model.FolderSent:
			parts = append(parts, "in:sent")
		case model.FolderDrafts:
			parts = append(parts, "in:drafts")
		case model.FolderTrash:
			parts = append(parts, "in:trash")
		case model.FolderSpam:
			parts = append(parts, "in:spam")
		}
	}

	if opts.From != "" {
		parts = append(parts, fmt.Sprintf("from:%s", opts.From))
	}

	if opts.To != "" {
		parts = append(parts, fmt.Sprintf("to:%s", opts.To))
	}

	if opts.Subject != "" {
		parts = append(parts, fmt.Sprintf("subject:\"%s\"", opts.Subject))
	}

	if opts.HasAttach != nil && *opts.HasAttach {
		parts = append(parts, "has:attachment")
	}

	if opts.IsRead != nil {
		if *opts.IsRead {
			parts = append(parts, "is:read")
		} else {
			parts = append(parts, "is:unread")
		}
	}

	if opts.Before != nil {
		parts = append(parts, fmt.Sprintf("before:%s", opts.Before.Format("2006/01/02")))
	}

	if opts.After != nil {
		parts = append(parts, fmt.Sprintf("after:%s", opts.After.Format("2006/01/02")))
	}

	return strings.Join(parts, " ")
}

// Ensure GmailAdapter implements adapter.Provider.
var _ adapter.Provider = (*GmailAdapter)(nil)
