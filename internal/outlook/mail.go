package outlook

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"ondapile/internal/adapter"
	"ondapile/internal/model"

	"github.com/google/uuid"
)

// SendEmail sends an email via Microsoft Graph API.
func (a *OutlookAdapter) SendEmail(ctx context.Context, accountID string, req adapter.SendEmailRequest) (*model.Email, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Build message body
	body := buildGraphMessage(req)

	var result map[string]interface{}
	if err := graphPost(client, "/me/sendMail", map[string]interface{}{"message": body}, &result); err != nil {
		return nil, fmt.Errorf("failed to send email: %w", err)
	}

	// Create email model
	emailID := uuid.New().String()
	now := now()

	email := &model.Email{
		Object:    "email",
		ID:        emailID,
		AccountID: accountID,
		Provider:  a.Name(),
		ProviderID: &model.EmailProviderID{
			MessageID: generateID(),
		},
		Subject:    req.Subject,
		Body:       req.BodyHTML,
		BodyPlain:  req.BodyPlain,
		Date:       now,
		Folders:    []string{model.FolderSent},
		Role:       model.FolderSent,
		Read:       true,
		IsComplete: true,
	}

	// Add recipients
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

// ListEmails lists emails from Outlook.
func (a *OutlookAdapter) ListEmails(ctx context.Context, accountID string, opts adapter.ListEmailOpts) (*model.PaginatedList[model.Email], error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Build query parameters
	params := buildGraphEmailQuery(opts)

	path := "/me/messages"
	if params != "" {
		path = path + "?" + params
	}

	var result map[string]interface{}
	if err := graphGet(client, path, &result); err != nil {
		return nil, fmt.Errorf("failed to list emails: %w", err)
	}

	emails, nextLink := normalizeGraphEmailList(result, accountID)
	hasMore := nextLink != ""

	return model.NewPaginatedList(emails, nextLink, hasMore), nil
}

// GetEmail gets a single email by ID from Outlook.
func (a *OutlookAdapter) GetEmail(ctx context.Context, accountID string, emailID string) (*model.Email, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	path := fmt.Sprintf("/me/messages/%s", emailID)

	var result map[string]interface{}
	if err := graphGet(client, path, &result); err != nil {
		return nil, fmt.Errorf("failed to get email: %w", err)
	}

	email := normalizeGraphEmail(result, accountID)
	return email, nil
}

// downloadEmailAttachment downloads an attachment from Outlook.
func (a *OutlookAdapter) downloadEmailAttachment(ctx context.Context, accountID string, messageID string, attachmentID string) ([]byte, string, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create HTTP client: %w", err)
	}

	path := fmt.Sprintf("/me/messages/%s/attachments/%s", messageID, attachmentID)

	var result map[string]interface{}
	if err := graphGet(client, path, &result); err != nil {
		return nil, "", fmt.Errorf("failed to download attachment: %w", err)
	}

	// Outlook returns base64 encoded content
	contentBytes, _ := result["contentBytes"].(string)
	name, _ := result["name"].(string)
	contentType, _ := result["contentType"].(string)

	decoded, err := base64.StdEncoding.DecodeString(contentBytes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode attachment: %w", err)
	}

	_ = name // Use filename if needed
	return decoded, contentType, nil
}

// Helper functions

func buildGraphMessage(req adapter.SendEmailRequest) map[string]interface{} {
	message := map[string]interface{}{
		"subject": req.Subject,
		"body": map[string]interface{}{
			"contentType": "html",
			"content":     req.BodyHTML,
		},
	}

	// Add recipients
	if len(req.To) > 0 {
		message["toRecipients"] = convertEmailAttendeesToRecipients(req.To)
	}
	if len(req.CC) > 0 {
		message["ccRecipients"] = convertEmailAttendeesToRecipients(req.CC)
	}
	if len(req.BCC) > 0 {
		message["bccRecipients"] = convertEmailAttendeesToRecipients(req.BCC)
	}

	return message
}

func convertEmailAttendeesToRecipients(attendees []model.EmailAttendee) []map[string]interface{} {
	recipients := make([]map[string]interface{}, len(attendees))
	for i, att := range attendees {
		recipient := map[string]interface{}{
			"emailAddress": map[string]interface{}{
				"address": att.Identifier,
			},
		}
		if att.DisplayName != "" {
			recipient["emailAddress"].(map[string]interface{})["name"] = att.DisplayName
		}
		recipients[i] = recipient
	}
	return recipients
}

func buildGraphEmailQuery(opts adapter.ListEmailOpts) string {
	var parts []string

	// Add $top for limit
	limit := opts.Limit
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	parts = append(parts, fmt.Sprintf("$top=%d", limit))

	// Build $filter
	var filters []string

	if opts.From != "" {
		filters = append(filters, fmt.Sprintf("from/emailAddress/address eq '%s'", opts.From))
	}

	if opts.IsRead != nil {
		if *opts.IsRead {
			filters = append(filters, "isRead eq true")
		} else {
			filters = append(filters, "isRead eq false")
		}
	}

	if opts.Before != nil {
		filters = append(filters, fmt.Sprintf("receivedDateTime lt %s", opts.Before.Format(time.RFC3339)))
	}
	if opts.After != nil {
		filters = append(filters, fmt.Sprintf("receivedDateTime gt %s", opts.After.Format(time.RFC3339)))
	}

	if len(filters) > 0 {
		parts = append(parts, "$filter="+strings.Join(filters, " and "))
	}

	// Add $orderby
	parts = append(parts, "$orderby=receivedDateTime desc")

	return strings.Join(parts, "&")
}
