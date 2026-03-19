package gmail

import (
	"encoding/base64"
	"strings"
	"time"

	"ondapile/internal/model"

	"github.com/google/uuid"
)

// normalizeEmail converts a Gmail API message response to an Email model.
func normalizeEmail(raw map[string]interface{}, accountID string) *model.Email {
	email := &model.Email{
		Object:      "email",
		ID:          "email_" + uuid.New().String(),
		AccountID:   accountID,
		Provider:    "GMAIL",
		Metadata:    raw,
		Attachments: []model.EmailAttachment{},
	}

	if id, ok := raw["id"].(string); ok {
		email.ProviderID = &model.EmailProviderID{
			MessageID: id,
		}
		if threadID, ok := raw["threadId"].(string); ok {
			email.ProviderID.ThreadID = threadID
		}
	}

	if payload, ok := raw["payload"].(map[string]interface{}); ok {
		if headers, ok := payload["headers"].([]interface{}); ok {
			for _, h := range headers {
				if header, ok := h.(map[string]interface{}); ok {
					name, _ := header["name"].(string)
					value, _ := header["value"].(string)

					switch strings.ToLower(name) {
					case "subject":
						email.Subject = value
					case "from":
						email.FromAttendee = parseEmailAttendee(value)
					case "to":
						email.ToAttendees = parseEmailAttendees(value)
					case "cc":
						email.CCAttendees = parseEmailAttendees(value)
					case "bcc":
						email.BCCAttendees = parseEmailAttendees(value)
					case "date":
						email.Date = parseEmailDate(value)
					}
				}
			}
		}

		bodyText, bodyHTML := extractBody(payload)
		email.Body = bodyHTML
		email.BodyPlain = bodyText

		if parts, ok := payload["parts"].([]interface{}); ok {
			for _, p := range parts {
				if part, ok := p.(map[string]interface{}); ok {
					if isAttachment(part) {
						filename, _ := part["filename"].(string)
						bodyData, _ := part["body"].(map[string]interface{})
						attachmentID, _ := bodyData["attachmentId"].(string)
						size, _ := bodyData["size"].(float64)
						mimeType, _ := part["mimeType"].(string)

						email.Attachments = append(email.Attachments, model.EmailAttachment{
							ID:       attachmentID,
							Filename: filename,
							MimeType: mimeType,
							Size:     int64(size),
						})
						email.HasAttachments = true
					}
				}
			}
		}
	}

	if labelIds, ok := raw["labelIds"].([]interface{}); ok {
		for _, lid := range labelIds {
			if labelID, ok := lid.(string); ok {
				folder := normalizeLabelToFolder(labelID)
				if folder != "" {
					email.Folders = append(email.Folders, folder)
				}
			}
		}
	}

	email.Read = !hasLabel(raw["labelIds"], "UNREAD")

	return email
}

// normalizeEmailList converts a Gmail API messages list response to a slice of Email models.
func normalizeEmailList(raw map[string]interface{}, accountID string) ([]*model.Email, string) {
	var emails []*model.Email
	nextPageToken := ""

	if messages, ok := raw["messages"].([]interface{}); ok {
		for _, m := range messages {
			if msgMap, ok := m.(map[string]interface{}); ok {
				email := &model.Email{
					Object:    "email",
					ID:        "email_" + uuid.New().String(),
					AccountID: accountID,
					Provider:  "GMAIL",
					ProviderID: &model.EmailProviderID{
						MessageID: getString(msgMap, "id"),
						ThreadID:  getString(msgMap, "threadId"),
					},
				}
				emails = append(emails, email)
			}
		}
	}

	if token, ok := raw["nextPageToken"].(string); ok {
		nextPageToken = token
	}

	return emails, nextPageToken
}

func parseEmailAttendee(addr string) *model.EmailAttendee {
	attendee := &model.EmailAttendee{
		IdentifierType: string(model.IdentifierTypeEmailAddress),
	}

	if idx := strings.LastIndex(addr, "<"); idx != -1 {
		if endIdx := strings.LastIndex(addr, ">"); endIdx != -1 && endIdx > idx {
			attendee.Identifier = strings.TrimSpace(addr[idx+1 : endIdx])
			attendee.DisplayName = strings.Trim(strings.TrimSpace(addr[:idx]), "\"")
		}
	} else {
		attendee.Identifier = strings.TrimSpace(addr)
	}

	return attendee
}

func parseEmailAttendees(addrs string) []model.EmailAttendee {
	var attendees []model.EmailAttendee
	parts := strings.Split(addrs, ",")
	for _, part := range parts {
		if attendee := parseEmailAttendee(part); attendee.Identifier != "" {
			attendees = append(attendees, *attendee)
		}
	}
	return attendees
}

func parseEmailDate(dateStr string) time.Time {
	formats := []string{
		time.RFC1123,
		time.RFC1123Z,
		time.RFC822,
		time.RFC822Z,
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t
		}
	}

	return time.Now()
}

func extractBody(payload map[string]interface{}) (text, html string) {
	mimeType, _ := payload["mimeType"].(string)

	if strings.HasPrefix(mimeType, "multipart/") {
		if parts, ok := payload["parts"].([]interface{}); ok {
			for _, p := range parts {
				if part, ok := p.(map[string]interface{}); ok {
					partMimeType, _ := part["mimeType"].(string)

					if partMimeType == "text/plain" && text == "" {
						text = extractBodyData(part)
					}
					if partMimeType == "text/html" && html == "" {
						html = extractBodyData(part)
					}

					if strings.HasPrefix(partMimeType, "multipart/") {
						nestedText, nestedHTML := extractBody(part)
						if text == "" {
							text = nestedText
						}
						if html == "" {
							html = nestedHTML
						}
					}
				}
			}
		}
	} else if mimeType == "text/plain" {
		text = extractBodyData(payload)
		html = text
	} else if mimeType == "text/html" {
		html = extractBodyData(payload)
	}

	return text, html
}

func extractBodyData(part map[string]interface{}) string {
	if body, ok := part["body"].(map[string]interface{}); ok {
		if data, ok := body["data"].(string); ok {
			decoded, err := base64.URLEncoding.DecodeString(data)
			if err != nil {
				decoded, err = base64.StdEncoding.DecodeString(data)
				if err != nil {
					return data
				}
			}
			return string(decoded)
		}
	}
	return ""
}

func isAttachment(part map[string]interface{}) bool {
	if filename, ok := part["filename"].(string); ok && filename != "" {
		return true
	}
	return false
}

func normalizeLabelToFolder(labelID string) string {
	switch labelID {
	case "INBOX":
		return model.FolderInbox
	case "SENT":
		return model.FolderSent
	case "DRAFT":
		return model.FolderDrafts
	case "TRASH":
		return model.FolderTrash
	case "SPAM":
		return model.FolderSpam
	case "CATEGORY_PERSONAL", "CATEGORY_SOCIAL", "CATEGORY_PROMOTIONS", "CATEGORY_UPDATES", "CATEGORY_FORUMS":
		return model.FolderInbox
	default:
		if strings.HasPrefix(labelID, "Label_") {
			return model.FolderCustom
		}
		return ""
	}
}

func hasLabel(labelIds interface{}, target string) bool {
	if ids, ok := labelIds.([]interface{}); ok {
		for _, id := range ids {
			if idStr, ok := id.(string); ok && idStr == target {
				return true
			}
		}
	}
	return false
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
