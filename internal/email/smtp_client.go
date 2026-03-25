package email

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"ondapile/internal/adapter"
	"ondapile/internal/model"

	mail "github.com/wneessen/go-mail"
)

// ConnectSMTP establishes a STARTTLS connection to an SMTP server.
func ConnectSMTP(host, port, username, password string) (*mail.Client, error) {
	// Create client with STARTTLS
	client, err := mail.NewClient(
		host,
		mail.WithPort(parsePort(port)),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(username),
		mail.WithPassword(password),
		mail.WithTLSPolicy(mail.TLSMandatory),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create SMTP client: %w", err)
	}

	slog.Info("SMTP client created", "host", host, "port", port, "username", username)
	return client, nil
}

// SendEmail sends an email via SMTP.
func SendEmail(client *mail.Client, req adapter.SendEmailRequest) error {
	// Create message
	msg := mail.NewMsg()

	// Set from address (use first To address's domain for From if needed)
	// In a real implementation, this would come from account settings
	from := getFromAddress(req)
	if err := msg.From(from); err != nil {
		return fmt.Errorf("failed to set from address: %w", err)
	}

	// Set recipients
	if len(req.To) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	for _, to := range req.To {
		addr := formatEmailAddress(to)
		if err := msg.AddTo(addr); err != nil {
			return fmt.Errorf("failed to add To recipient %s: %w", addr, err)
		}
	}

	// Add CC recipients
	for _, cc := range req.CC {
		addr := formatEmailAddress(cc)
		if err := msg.AddCc(addr); err != nil {
			return fmt.Errorf("failed to add Cc recipient %s: %w", addr, err)
		}
	}

	// Add BCC recipients
	for _, bcc := range req.BCC {
		addr := formatEmailAddress(bcc)
		if err := msg.AddBcc(addr); err != nil {
			return fmt.Errorf("failed to add Bcc recipient %s: %w", addr, err)
		}
	}

	// Set subject
	if req.Subject != "" {
		msg.Subject(req.Subject)
	}

	// Set body content
	if req.BodyHTML != "" && req.BodyPlain != "" {
		// Both HTML and plain text
		msg.SetBodyString(mail.TypeTextHTML, req.BodyHTML)
		msg.AddAlternativeString(mail.TypeTextPlain, req.BodyPlain)
	} else if req.BodyHTML != "" {
		// HTML only
		msg.SetBodyString(mail.TypeTextHTML, req.BodyHTML)
	} else if req.BodyPlain != "" {
		// Plain text only
		msg.SetBodyString(mail.TypeTextPlain, req.BodyPlain)
	} else {
		// Empty body
		msg.SetBodyString(mail.TypeTextPlain, "")
	}

	// Set threading headers for replies/forwards
	if req.InReplyTo != "" {
		msg.SetGenHeader("In-Reply-To", fmt.Sprintf("<%s>", req.InReplyTo))
	}
	if req.References != "" {
		msg.SetGenHeader("References", req.References)
	}

	// Add attachments
	for _, att := range req.Attachments {
		if err := msg.AttachReader(att.Filename, &bytesReader{content: att.Content}, mail.WithFileContentType(mail.ContentType(att.MimeType))); err != nil {
			return fmt.Errorf("failed to attach file %s: %w", att.Filename, err)
		}
	}

	// Add headers
	msg.SetDate()
	msg.SetMessageIDWithValue(generateMessageID(from))

	// Send the message
	if err := client.DialAndSend(msg); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	slog.Info("Email sent successfully", "to_count", len(req.To), "subject", req.Subject)
	return nil
}

// bytesReader wraps []byte to implement io.Reader for attachments.
type bytesReader struct {
	content []byte
	pos     int
}

func (br *bytesReader) Read(p []byte) (n int, err error) {
	if br.pos >= len(br.content) {
		return 0, io.EOF
	}
	n = copy(p, br.content[br.pos:])
	br.pos += n
	if br.pos >= len(br.content) {
		return n, io.EOF
	}
	return n, nil
}

// formatEmailAddress formats an EmailAttendee as a full email address.
func formatEmailAddress(att model.EmailAttendee) string {
	if att.DisplayName != "" {
		return fmt.Sprintf("%s <%s>", att.DisplayName, att.Identifier)
	}
	return att.Identifier
}

// getFromAddress extracts or constructs a From address.
// This is a helper that uses the first To address's domain as a fallback.
func getFromAddress(req adapter.SendEmailRequest) string {
	// In a real implementation, this would come from the account settings
	// For now, we construct a generic from address
	if len(req.To) > 0 {
		// Try to extract domain from first To address
		to := req.To[0].Identifier
		if idx := strings.LastIndex(to, "@"); idx != -1 {
			domain := to[idx+1:]
			return fmt.Sprintf("noreply@%s", domain)
		}
	}
	return "noreply@ondapile.local"
}

// generateMessageID creates a unique Message-ID for sent emails.
func generateMessageID(from string) string {
	timestamp := time.Now().UnixNano()
	random := fmt.Sprintf("%x", timestamp)

	// Extract domain from from address
	domain := "ondapile.local"
	if idx := strings.LastIndex(from, "@"); idx != -1 {
		domain = from[idx+1:]
	}

	return fmt.Sprintf("<%s@%s>", random, domain)
}

// parsePort converts a string port to int.
func parsePort(port string) int {
	var p int
	fmt.Sscanf(port, "%d", &p)
	if p == 0 {
		p = 587 // Default to submission port
	}
	return p
}

// CloseSMTP closes the SMTP client connection.
func CloseSMTP(client *mail.Client) error {
	if client != nil {
		return client.Close()
	}
	return nil
}
