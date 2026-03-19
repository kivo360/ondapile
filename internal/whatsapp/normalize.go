package whatsapp

import (
	"fmt"
	"strings"
	"time"

	"ondapile/internal/model"

	"github.com/google/uuid"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

// normalizeMessage converts a whatsmeow message event to an ondapile Message model.
func (p *WhatsAppProvider) normalizeMessage(evt *events.Message, accountID string) *model.Message {
	// Skip protocol messages (system messages)
	if evt.Message.GetProtocolMessage() != nil {
		return p.normalizeProtocolMessage(evt, accountID)
	}

	// Skip ephemeral messages (they're just wrappers)
	if evt.Message.GetEphemeralMessage() != nil {
		// Extract the actual message from ephemeral wrapper
		return p.normalizeEphemeralMessage(evt, accountID)
	}

	// Handle view-once messages
	if evt.Message.GetViewOnceMessage() != nil {
		return p.normalizeViewOnceMessage(evt, accountID)
	}

	// Handle reaction messages
	if evt.Message.GetReactionMessage() != nil {
		return p.normalizeReactionMessage(evt, accountID)
	}

	// Regular message
	return p.normalizeRegularMessage(evt, accountID)
}

// normalizeRegularMessage handles normal text/media messages.
func (p *WhatsAppProvider) normalizeRegularMessage(evt *events.Message, accountID string) *model.Message {
	msg := evt.Message
	info := evt.Info

	// Extract text content
	text := extractText(msg)

	// Generate ondapile IDs
	conduitMsgID := generateConduitID("msg_")

	// Extract sender info
	senderID := info.Sender.String()
	isSender := info.IsFromMe

	// Build message model
	message := &model.Message{
		Object:      "message",
		ID:          conduitMsgID,
		AccountID:   accountID,
		Provider:    p.Name(),
		ProviderID:  info.ID,
		Text:        text,
		SenderID:    senderID,
		IsSender:    isSender,
		Timestamp:   info.Timestamp,
		Seen:        info.IsFromMe, // Messages from self are already seen
		Delivered:   false,
		Edited:      false,
		Deleted:     false,
		Hidden:      false,
		IsEvent:     false,
		Attachments: []model.Attachment{},
		Reactions:   []model.Reaction{},
		Metadata:    make(map[string]any),
	}

	// Extract quoted message if present
	if context := getContextInfo(msg); context != nil {
		if context.StanzaID != nil {
			quotedText := ""
			if context.QuotedMessage != nil {
				quotedText = extractText(context.QuotedMessage)
			}
			message.Quoted = &model.QuotedMessage{
				ID:   generateConduitID("msg_"), // We don't have the original ID, create a placeholder
				Text: quotedText,
			}
			// Store the provider message ID in metadata for reference
			message.Metadata["quoted_provider_id"] = *context.StanzaID
		}
	}

	// Extract attachments if media message
	if HasMedia(msg) {
		attachment := p.extractAttachment(msg, conduitMsgID)
		if attachment != nil {
			message.Attachments = []model.Attachment{*attachment}
		}
	}

	return message
}

// normalizeReactionMessage handles reaction/emoji reactions.
func (p *WhatsAppProvider) normalizeReactionMessage(evt *events.Message, accountID string) *model.Message {
	reactionMsg := evt.Message.GetReactionMessage()
	if reactionMsg == nil {
		return nil
	}

	info := evt.Info
	conduitMsgID := generateConduitID("msg_")

	// Extract reaction emoji
	emoji := ""
	if reactionMsg.Text != nil {
		emoji = *reactionMsg.Text
	}

	message := &model.Message{
		Object:      "message",
		ID:          conduitMsgID,
		AccountID:   accountID,
		Provider:    p.Name(),
		ProviderID:  info.ID,
		Text:        emoji,
		SenderID:    info.Sender.String(),
		IsSender:    info.IsFromMe,
		Timestamp:   info.Timestamp,
		Seen:        true,
		Delivered:   true,
		Edited:      false,
		Deleted:     false,
		Hidden:      false,
		IsEvent:     true,
		EventType:   intPtr(model.EventTypeReaction),
		Attachments: []model.Attachment{},
		Reactions: []model.Reaction{
			{
				Value:    emoji,
				SenderID: info.Sender.String(),
				IsSender: info.IsFromMe,
			},
		},
		Metadata: make(map[string]any),
	}

	// Store the reacted message ID
	if reactionMsg.Key != nil && reactionMsg.Key.ID != nil {
		message.Metadata["reacted_to_message_id"] = *reactionMsg.Key.ID
	}

	return message
}

// normalizeProtocolMessage handles system/protocol messages.
func (p *WhatsAppProvider) normalizeProtocolMessage(evt *events.Message, accountID string) *model.Message {
	protoMsg := evt.Message.GetProtocolMessage()
	if protoMsg == nil {
		return nil
	}

	info := evt.Info

	// Check for message deletion
	if protoMsg.GetType() == waE2E.ProtocolMessage_REVOKE {
		// This is a message deletion
		conduitMsgID := generateConduitID("msg_")

		message := &model.Message{
			Object:     "message",
			ID:         conduitMsgID,
			AccountID:  accountID,
			Provider:   p.Name(),
			ProviderID: info.ID,
			Text:       "This message was deleted",
			SenderID:   info.Sender.String(),
			IsSender:   info.IsFromMe,
			Timestamp:  info.Timestamp,
			Seen:       true,
			Deleted:    true,
			Hidden:     true, // Hide deleted messages from normal view
			IsEvent:    true,
			EventType:  intPtr(model.EventTypeUnknown),
			Metadata:   make(map[string]any),
		}

		// Store the deleted message ID
		if protoMsg.Key != nil && protoMsg.Key.ID != nil {
			message.Metadata["deleted_message_id"] = *protoMsg.Key.ID
		}

		return message
	}

	// Other protocol messages are hidden
	return nil
}

// normalizeEphemeralMessage extracts content from ephemeral message wrappers.
func (p *WhatsAppProvider) normalizeEphemeralMessage(evt *events.Message, accountID string) *model.Message {
	ephemeral := evt.Message.GetEphemeralMessage()
	if ephemeral == nil || ephemeral.Message == nil {
		return nil
	}

	// Replace the message content and re-normalize
	evt.Message = ephemeral.Message
	return p.normalizeRegularMessage(evt, accountID)
}

// normalizeViewOnceMessage extracts content from view-once messages.
func (p *WhatsAppProvider) normalizeViewOnceMessage(evt *events.Message, accountID string) *model.Message {
	viewOnce := evt.Message.GetViewOnceMessage()
	if viewOnce == nil || viewOnce.Message == nil {
		return nil
	}

	// Replace the message content and re-normalize
	evt.Message = viewOnce.Message

	// Mark as view-once in metadata
	message := p.normalizeRegularMessage(evt, accountID)
	if message != nil {
		message.Metadata["view_once"] = true
	}

	return message
}

// extractText extracts text content from various message types.
func extractText(msg *waE2E.Message) string {
	switch {
	case msg.Conversation != nil:
		return msg.GetConversation()

	case msg.ExtendedTextMessage != nil:
		if msg.ExtendedTextMessage.Text != nil {
			return *msg.ExtendedTextMessage.Text
		}

	case msg.ImageMessage != nil:
		if msg.ImageMessage.Caption != nil {
			return *msg.ImageMessage.Caption
		}
		return ""

	case msg.VideoMessage != nil:
		if msg.VideoMessage.Caption != nil {
			return *msg.VideoMessage.Caption
		}
		return ""

	case msg.DocumentMessage != nil:
		if msg.DocumentMessage.Caption != nil {
			return *msg.DocumentMessage.Caption
		}
		return ""

	case msg.AudioMessage != nil:
		return ""

	case msg.StickerMessage != nil:
		return ""

	case msg.ContactMessage != nil:
		if msg.ContactMessage.DisplayName != nil {
			return fmt.Sprintf("Contact: %s", *msg.ContactMessage.DisplayName)
		}
		return "Contact shared"

	case msg.LocationMessage != nil:
		return "Location shared"

	case msg.LiveLocationMessage != nil:
		return "Live location shared"

	case msg.PollCreationMessage != nil:
		if msg.PollCreationMessage.Name != nil {
			return fmt.Sprintf("Poll: %s", *msg.PollCreationMessage.Name)
		}
		return "Poll"

	default:
		return ""
	}

	return ""
}

// extractAttachment extracts attachment information from a media message.
func (p *WhatsAppProvider) extractAttachment(msg *waE2E.Message, messageID string) *model.Attachment {
	if !HasMedia(msg) {
		return nil
	}

	filename, _, mimeType, fileSize := GetMediaInfo(msg)

	// Generate attachment ID
	attachmentID := generateConduitID("att_")

	attachment := &model.Attachment{
		ID:       attachmentID,
		Filename: filename,
		MimeType: mimeType,
		Size:     int64(fileSize),
		// URL will be generated on-demand when downloading
	}

	// Store media metadata for later download
	// This would be stored in the message's metadata field
	// The actual download happens via DownloadAttachment

	return attachment
}

// getContextInfo extracts context info (for quoted messages) from various message types.
func getContextInfo(msg *waE2E.Message) *waE2E.ContextInfo {
	switch {
	case msg.ExtendedTextMessage != nil:
		return msg.ExtendedTextMessage.ContextInfo
	case msg.ImageMessage != nil:
		return msg.ImageMessage.ContextInfo
	case msg.VideoMessage != nil:
		return msg.VideoMessage.ContextInfo
	case msg.AudioMessage != nil:
		return msg.AudioMessage.ContextInfo
	case msg.DocumentMessage != nil:
		return msg.DocumentMessage.ContextInfo
	default:
		return nil
	}
}

// generateConduitID generates an ondapile-style ID with the given prefix.
func generateConduitID(prefix string) string {
	return prefix + strings.ReplaceAll(uuid.New().String(), "-", "")
}

// intPtr returns a pointer to an int.
func intPtr(i int) *int {
	return &i
}

// parseTimestamp parses a WhatsApp timestamp (Unix seconds) to time.Time.
func parseTimestamp(unixTime int64) time.Time {
	return time.Unix(unixTime, 0).UTC()
}

// formatPhoneNumber formats a phone number for display.
func formatPhoneNumber(phone string) string {
	// Remove any non-digit characters
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)

	// Add + prefix if missing
	if !strings.HasPrefix(digits, "+") {
		digits = "+" + digits
	}

	return digits
}
