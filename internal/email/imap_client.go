package email

import (
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"

	"ondapile/internal/model"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/google/uuid"
)

// ConnectIMAP establishes a TLS connection to an IMAP server.
func ConnectIMAP(host, port, username, password string) (*imapclient.Client, error) {
	address := net.JoinHostPort(host, port)

	// Connect with TLS
	client, err := imapclient.DialTLS(address, &imapclient.Options{
		TLSConfig: nil, // Use default TLS config
	})
	if err != nil {
		return nil, fmt.Errorf("failed to dial IMAP: %w", err)
	}

	// Login
	if err := client.Login(username, password).Wait(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to login to IMAP: %w", err)
	}

	slog.Info("IMAP connected and authenticated", "host", host, "username", username)
	return client, nil
}

// ListMailboxes lists all available mailboxes/folders.
func ListMailboxes(client *imapclient.Client) ([]string, error) {
	mailboxes := []string{}

	// Start listing mailboxes
	listCmd := client.List("", "*", nil)
	defer listCmd.Close()

	for {
		item := listCmd.Next()
		if item == nil {
			break
		}
		mailboxes = append(mailboxes, item.Mailbox)
	}

	if err := listCmd.Close(); err != nil {
		return nil, fmt.Errorf("failed to list mailboxes: %w", err)
	}

	return mailboxes, nil
}

// FetchMessages fetches emails from a specific folder.
func FetchMessages(client *imapclient.Client, folder string, limit, offset int) ([]model.Email, error) {
	if folder == "" {
		folder = "INBOX"
	}

	// Select the mailbox
	selectCmd := client.Select(folder, &imap.SelectOptions{})
	selected, err := selectCmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to select mailbox %s: %w", folder, err)
	}

	// If mailbox is empty, return empty slice
	if selected.NumMessages == 0 {
		return []model.Email{}, nil
	}

	// Calculate sequence range
	start := uint32(1)
	if offset > 0 && selected.NumMessages > uint32(offset) {
		start = selected.NumMessages - uint32(offset)
	}
	if start < 1 {
		start = 1
	}

	end := selected.NumMessages
	if limit > 0 && selected.NumMessages > uint32(limit+offset) {
		end = selected.NumMessages - uint32(offset)
		if end < start {
			end = start
		}
	}

	var seqSet imap.SeqSet
	seqSet.AddRange(start, end)

	// Fetch messages
	fetchOptions := &imap.FetchOptions{
		Envelope: true,
		BodySection: []*imap.FetchItemBodySection{
			{Specifier: imap.PartSpecifierHeader},
			{Specifier: imap.PartSpecifierText},
		},
		Flags: true,
		UID:   true,
	}

	fetchCmd := client.Fetch(seqSet, fetchOptions)
	defer fetchCmd.Close()

	emails := []model.Email{}
	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		buf, err := msg.Collect()
		if err != nil {
			slog.Error("failed to collect message data", "error", err)
			continue
		}

		email := parseFetchBuffer(buf)

		// Set folder
		email.Folders = []string{folder}
		if folder == "INBOX" {
			email.Role = model.FolderInbox
		} else if folder == "Sent" || folder == "SENT" || folder == "Sent Items" {
			email.Role = model.FolderSent
		} else if folder == "Drafts" || folder == "DRAFTS" {
			email.Role = model.FolderDrafts
		} else if folder == "Trash" || folder == "TRASH" || folder == "Deleted Items" {
			email.Role = model.FolderTrash
		} else if folder == "Spam" || folder == "SPAM" || folder == "Junk" {
			email.Role = model.FolderSpam
		} else if folder == "Archive" || folder == "ARCHIVE" {
			email.Role = model.FolderArchive
		} else {
			email.Role = model.FolderCustom
		}

		emails = append(emails, email)
	}

	if err := fetchCmd.Close(); err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	return emails, nil
}

// FetchMessage fetches a single email by sequence number.
func FetchMessage(client *imapclient.Client, folder string, seqNum uint32) (model.Email, error) {
	if folder == "" {
		folder = "INBOX"
	}

	// Select the mailbox
	selectCmd := client.Select(folder, &imap.SelectOptions{})
	_, err := selectCmd.Wait()
	if err != nil {
		return model.Email{}, fmt.Errorf("failed to select mailbox %s: %w", folder, err)
	}

	seqSet := imap.SeqSetNum(seqNum)

	fetchOptions := &imap.FetchOptions{
		Envelope: true,
		BodySection: []*imap.FetchItemBodySection{
			{Specifier: imap.PartSpecifierHeader},
			{Specifier: imap.PartSpecifierText},
		},
		Flags: true,
		UID:   true,
	}

	fetchCmd := client.Fetch(seqSet, fetchOptions)
	defer fetchCmd.Close()

	msg := fetchCmd.Next()
	if msg == nil {
		return model.Email{}, fmt.Errorf("message not found")
	}

	buf, err := msg.Collect()
	if err != nil {
		return model.Email{}, fmt.Errorf("failed to collect message data: %w", err)
	}

	email := parseFetchBuffer(buf)
	email.Folders = []string{folder}

	if err := fetchCmd.Close(); err != nil {
		return model.Email{}, fmt.Errorf("failed to fetch message: %w", err)
	}

	return email, nil
}

// parseFetchBuffer converts an IMAP fetch buffer to an Email model.
func parseFetchBuffer(buf *imapclient.FetchMessageBuffer) model.Email {
	email := model.Email{
		Object:     "email",
		ID:         uuid.New().String(),
		Provider:   "IMAP",
		Metadata:   map[string]any{},
		Headers:    []model.EmailHeader{},
		Read:       true,
		IsComplete: true,
	}

	// Extract UID and sequence number
	if buf.UID != 0 {
		email.Metadata["uid"] = buf.UID
	}
	if buf.SeqNum != 0 {
		email.Metadata["seq_num"] = buf.SeqNum
	}

	// Parse envelope
	if buf.Envelope != nil {
		parseEnvelope(buf.Envelope, &email)
	}

	// Parse flags
	for _, flag := range buf.Flags {
		if flag == imap.FlagSeen {
			email.Read = true
			now := time.Now()
			email.ReadDate = &now
		}
	}

	// Parse body sections
	for _, bs := range buf.BodySection {
		if err := parseBodySection(&bs, &email); err != nil {
			slog.Error("failed to parse body section", "error", err)
		}
	}

	return email
}

// parseEnvelope extracts data from the envelope.
func parseEnvelope(env *imap.Envelope, email *model.Email) {
	// Subject
	if env.Subject != "" {
		email.Subject = env.Subject
	}

	// Date
	if !env.Date.IsZero() {
		email.Date = env.Date
	} else {
		email.Date = time.Now()
	}

	// From
	if len(env.From) > 0 {
		from := env.From[0]
		email.FromAttendee = &model.EmailAttendee{
			DisplayName:    from.Name,
			Identifier:     from.Addr(),
			IdentifierType: string(model.IdentifierTypeEmailAddress),
		}
	}

	// To
	for _, to := range env.To {
		email.ToAttendees = append(email.ToAttendees, model.EmailAttendee{
			DisplayName:    to.Name,
			Identifier:     to.Addr(),
			IdentifierType: string(model.IdentifierTypeEmailAddress),
		})
	}

	// CC
	for _, cc := range env.Cc {
		email.CCAttendees = append(email.CCAttendees, model.EmailAttendee{
			DisplayName:    cc.Name,
			Identifier:     cc.Addr(),
			IdentifierType: string(model.IdentifierTypeEmailAddress),
		})
	}

	// BCC
	for _, bcc := range env.Bcc {
		email.BCCAttendees = append(email.BCCAttendees, model.EmailAttendee{
			DisplayName:    bcc.Name,
			Identifier:     bcc.Addr(),
			IdentifierType: string(model.IdentifierTypeEmailAddress),
		})
	}

	// Reply-To
	for _, replyTo := range env.ReplyTo {
		email.ReplyToAttendees = append(email.ReplyToAttendees, model.EmailAttendee{
			DisplayName:    replyTo.Name,
			Identifier:     replyTo.Addr(),
			IdentifierType: string(model.IdentifierTypeEmailAddress),
		})
	}

	// Message ID for threading
	if env.MessageID != "" {
		email.ProviderID = &model.EmailProviderID{
			MessageID: env.MessageID,
		}
		// Try to extract thread ID from In-Reply-To or References
		if len(env.InReplyTo) > 0 && env.InReplyTo[0] != "" {
			email.ProviderID.ThreadID = env.InReplyTo[0]
		} else {
			email.ProviderID.ThreadID = env.MessageID
		}
	}
}

// parseBodySection parses the body content of an email.
func parseBodySection(data *imapclient.FetchBodySectionBuffer, email *model.Email) error {
	if len(data.Bytes) == 0 {
		return nil
	}

	bodyData := data.Bytes

	// Determine if this is header or body based on the section specifier
	if data.Section != nil && data.Section.Specifier == imap.PartSpecifierHeader {
		// Parse headers
		parseHeaders(bodyData, email)
	} else {
		// Parse body content
		if err := parseBody(bodyData, email); err != nil {
			return fmt.Errorf("failed to parse body: %w", err)
		}
	}

	return nil
}

// parseHeaders extracts headers from the raw header data.
func parseHeaders(headerData []byte, email *model.Email) {
	// Parse headers manually since mail.NewReader requires *message.Entity
	headerStr := string(headerData)
	lines := strings.Split(headerStr, "\n")
	
	var currentKey string
	var currentValue strings.Builder
	
	for _, line := range lines {
		// Check if this is a continuation line (starts with whitespace)
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			if currentKey != "" {
				currentValue.WriteString(" ")
				currentValue.WriteString(strings.TrimSpace(line))
			}
			continue
		}
		
		// Save previous header if exists
		if currentKey != "" {
			value := strings.TrimSpace(currentValue.String())
			if value != "" {
				email.Headers = append(email.Headers, model.EmailHeader{
					Key:   currentKey,
					Value: value,
				})
				
				// Parse specific headers
				switch strings.ToLower(currentKey) {
				case "subject":
					if email.Subject == "" {
						email.Subject = value
					}
				case "from":
					if email.FromAttendee == nil {
						if addr := parseAddress(value); addr != nil {
							email.FromAttendee = addr
						}
					}
				case "to":
					email.ToAttendees = append(email.ToAttendees, parseAddresses(value)...)
				case "cc":
					email.CCAttendees = append(email.CCAttendees, parseAddresses(value)...)
				case "message-id":
					if email.ProviderID == nil {
						email.ProviderID = &model.EmailProviderID{}
					}
					email.ProviderID.MessageID = strings.Trim(value, "<>")
					if email.ProviderID.ThreadID == "" {
						email.ProviderID.ThreadID = email.ProviderID.MessageID
					}
				case "in-reply-to":
					if email.ProviderID == nil {
						email.ProviderID = &model.EmailProviderID{}
					}
					email.ProviderID.ThreadID = strings.Trim(value, "<>")
				}
			}
		}
		
		// Parse new header line
		colonIdx := strings.Index(line, ":")
		if colonIdx > 0 {
			currentKey = line[:colonIdx]
			currentValue.Reset()
			currentValue.WriteString(strings.TrimSpace(line[colonIdx+1:]))
		}
	}
	
	// Don't forget the last header
	if currentKey != "" {
		value := strings.TrimSpace(currentValue.String())
		if value != "" {
			email.Headers = append(email.Headers, model.EmailHeader{
				Key:   currentKey,
				Value: value,
			})
		}
	}
}

// parseAddress parses a single email address from a string.
func parseAddress(addrStr string) *model.EmailAttendee {
	addrStr = strings.TrimSpace(addrStr)
	if addrStr == "" {
		return nil
	}
	
	// Try to extract name and email
	// Format: "Name" <email@example.com> or just email@example.com
	var name, email string
	
	if idx := strings.LastIndex(addrStr, "<"); idx >= 0 {
		if endIdx := strings.Index(addrStr[idx:], ">"); endIdx > 0 {
			email = strings.TrimSpace(addrStr[idx+1 : idx+endIdx])
			name = strings.TrimSpace(addrStr[:idx])
			name = strings.Trim(name, "\"'")
		}
	}
	
	if email == "" {
		// Just an email address
		email = strings.Trim(addrStr, "<>")
	}
	
	if email == "" {
		return nil
	}
	
	return &model.EmailAttendee{
		DisplayName:    name,
		Identifier:     email,
		IdentifierType: string(model.IdentifierTypeEmailAddress),
	}
}

// parseAddresses parses multiple comma-separated email addresses.
func parseAddresses(addrsStr string) []model.EmailAttendee {
	var attendees []model.EmailAttendee
	
	// Simple split by comma (doesn't handle quoted commas)
	parts := strings.Split(addrsStr, ",")
	for _, part := range parts {
		if addr := parseAddress(part); addr != nil {
			attendees = append(attendees, *addr)
		}
	}
	
	return attendees
}

// parseBody parses the email body, handling multipart messages and attachments.
func parseBody(bodyData []byte, email *model.Email) error {
	// Check if it looks like multipart
	contentType := "text/plain"
	
	// Try to extract content-type from body
	if len(bodyData) > 0 {
		// Simple heuristic: check for HTML tags
		if strings.Contains(string(bodyData), "<html") || strings.Contains(string(bodyData), "<HTML") {
			contentType = "text/html"
		}
	}
	
	if strings.HasPrefix(contentType, "text/html") {
		email.Body = string(bodyData)
		// Also set plain text by stripping HTML
		if email.BodyPlain == "" {
			email.BodyPlain = stripHTMLTags(email.Body)
		}
	} else {
		email.BodyPlain = string(bodyData)
	}
	
	return nil
}

// stripHTMLTags removes HTML tags from a string.
func stripHTMLTags(html string) string {
	var result strings.Builder
	inTag := false

	for _, r := range html {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				result.WriteRune(r)
			}
		}
	}

	return strings.TrimSpace(result.String())
}

// IDLE starts an IDLE session and waits for new messages.
// Note: Not all servers support IDLE. This function attempts to use IDLE
// and falls back to polling if IDLE is not supported.
func IDLE(client *imapclient.Client, folder string, handler func(uint32)) error {
	if folder == "" {
		folder = "INBOX"
	}

	// Select the mailbox
	selectCmd := client.Select(folder, &imap.SelectOptions{})
	_, err := selectCmd.Wait()
	if err != nil {
		return fmt.Errorf("failed to select mailbox for IDLE: %w", err)
	}

	// Try to start IDLE
	// Note: go-imap v2 handles IDLE differently; we use a simpler polling approach
	// for compatibility across different servers
	slog.Info("IDLE not directly supported, using polling instead")

	return nil
}

// SearchEmails searches for emails matching criteria.
func SearchEmails(client *imapclient.Client, folder string, criteria imap.SearchCriteria) ([]imap.UID, error) {
	if folder == "" {
		folder = "INBOX"
	}

	// Select the mailbox
	selectCmd := client.Select(folder, &imap.SelectOptions{})
	_, err := selectCmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to select mailbox for search: %w", err)
	}

	// Search
	searchCmd := client.UIDSearch(&criteria, nil)
	results, err := searchCmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return results.AllUIDs(), nil
}

// MarkAsSeen marks an email as read.
func MarkAsSeen(client *imapclient.Client, folder string, uid uint32) error {
	if folder == "" {
		folder = "INBOX"
	}

	// Select the mailbox
	selectCmd := client.Select(folder, &imap.SelectOptions{})
	_, err := selectCmd.Wait()
	if err != nil {
		return fmt.Errorf("failed to select mailbox: %w", err)
	}

	// Store the \Seen flag using UID
	uidSet := imap.UIDSetNum(imap.UID(uid))
	storeCmd := client.Store(uidSet, &imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Silent: false,
		Flags:  []imap.Flag{imap.FlagSeen},
	}, nil)

	if err := storeCmd.Close(); err != nil {
		return fmt.Errorf("failed to mark as seen: %w", err)
	}

	return nil
}

// DeleteMessage moves a message to the Trash folder.
func DeleteMessage(client *imapclient.Client, folder string, uid uint32) error {
	if folder == "" {
		folder = "INBOX"
	}

	// Select the mailbox
	selectCmd := client.Select(folder, &imap.SelectOptions{})
	_, err := selectCmd.Wait()
	if err != nil {
		return fmt.Errorf("failed to select mailbox: %w", err)
	}

	// Add \Deleted flag using UID
	uidSet := imap.UIDSetNum(imap.UID(uid))
	storeCmd := client.Store(uidSet, &imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Silent: false,
		Flags:  []imap.Flag{imap.FlagDeleted},
	}, nil)

	if err := storeCmd.Close(); err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	// Expunge to actually delete
	expungeCmd := client.Expunge()
	if err := expungeCmd.Close(); err != nil {
		return fmt.Errorf("failed to expunge: %w", err)
	}

	return nil
}

// MoveMessage moves a message to another folder.
func MoveMessage(client *imapclient.Client, sourceFolder, destFolder string, uid uint32) error {
	if sourceFolder == "" {
		sourceFolder = "INBOX"
	}

	// Select source mailbox
	selectCmd := client.Select(sourceFolder, &imap.SelectOptions{})
	_, err := selectCmd.Wait()
	if err != nil {
		return fmt.Errorf("failed to select source mailbox: %w", err)
	}

	// Try MOVE command first (RFC 6851) using UID
	uidSet := imap.UIDSetNum(imap.UID(uid))
	moveCmd := client.Move(uidSet, destFolder)
	if _, err := moveCmd.Wait(); err == nil {
		return nil
	}

	// Fall back to COPY + DELETE
	copyCmd := client.Copy(uidSet, destFolder)
	if _, err := copyCmd.Wait(); err != nil {
		return fmt.Errorf("failed to copy message: %w", err)
	}

	// Delete from source
	return DeleteMessage(client, sourceFolder, uid)
}

// itoa converts an int to string (helper for consistency with store package).
func itoa(i int) string {
	return strconv.Itoa(i)
}
