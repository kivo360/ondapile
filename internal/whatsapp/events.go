package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"ondapile/internal/model"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// handleMessageEvent processes incoming message events.
func (p *WhatsAppProvider) handleMessageEvent(accountID string, evt *events.Message) {
	ctx := context.Background()

	// Normalize the message
	message := p.normalizeMessage(evt, accountID)
	if message == nil {
		return
	}

	// Get or create chat
	chatJID := evt.Info.Chat.String()
	chat, err := p.chatStore.GetByProviderID(ctx, accountID, chatJID)
	if err != nil {
		slog.Error("failed to get chat", "error", err)
		return
	}

	if chat == nil {
		// Create new chat
		chat = p.createChatFromEvent(evt, accountID)
		chat, err = p.chatStore.Create(ctx, chat)
		if err != nil {
			slog.Error("failed to create chat", "error", err)
			return
		}

		// Dispatch chat created event
		p.dispatcher.Dispatch(ctx, model.EventChatCreated, chat)
	}

	// Set chat ID on message
	message.ChatID = chat.ID

	// Check if message already exists (duplicate handling)
	existing, err := p.messageStore.GetByProviderID(ctx, accountID, message.ProviderID)
	if err != nil {
		slog.Error("failed to check existing message", "error", err)
		return
	}
	if existing != nil {
		// Message already exists, skip
		return
	}

	// Store message
	message, err = p.messageStore.Create(ctx, message)
	if err != nil {
		slog.Error("failed to store message", "error", err)
		return
	}

	// Update chat last message and unread count
	preview := message.Text
	if preview == "" && len(message.Attachments) > 0 {
		preview = fmt.Sprintf("[%s]", message.Attachments[0].MimeType)
	}
	_ = p.chatStore.UpdateLastMessage(ctx, chat.ID, &preview)

	// Only increment unread if not from self
	if !message.IsSender {
		_ = p.chatStore.IncrementUnread(ctx, chat.ID)
	}

	// Dispatch webhook based on message type
	if message.Deleted {
		p.dispatcher.Dispatch(ctx, model.EventMessageDeleted, map[string]any{
			"message_id": message.ID,
			"chat_id":    chat.ID,
		})
	} else if message.IsEvent && message.EventType != nil && *message.EventType == model.EventTypeReaction {
		p.dispatcher.Dispatch(ctx, model.EventMessageReaction, map[string]any{
			"message_id": message.ID,
			"chat_id":    chat.ID,
			"reaction":   message.Text,
			"sender_id":  message.SenderID,
			"action":     "added",
		})
	} else {
		p.dispatcher.Dispatch(ctx, model.EventMessageReceived, message)
	}
}

// handleReceiptEvent processes delivery/read receipt events.
func (p *WhatsAppProvider) handleReceiptEvent(accountID string, evt *events.Receipt) {
	ctx := context.Background()

	// Find message by provider ID (use first message ID from receipt)
	if len(evt.MessageIDs) == 0 {
		return
	}
	message, err := p.messageStore.GetByProviderID(ctx, accountID, string(evt.MessageIDs[0]))
	if err != nil {
		slog.Error("failed to get message for receipt", "error", err)
		return
	}
	if message == nil {
		// Message not found, might be from before we started tracking
		return
	}

	// Update message status based on receipt type
	switch evt.Type {
	case events.ReceiptTypeDelivered:
		// Update delivered status
		// Note: messageStore doesn't have UpdateDeliveredStatus, so we use metadata
		if message.Metadata == nil {
			message.Metadata = make(map[string]any)
		}
		message.Metadata["delivered_at"] = time.Now().UTC()
		message.Delivered = true

		// Dispatch delivered event
		p.dispatcher.Dispatch(ctx, model.EventMessageRead, map[string]any{
			"message_id": message.ID,
			"account_id": accountID,
			"status":     "delivered",
		})

	case events.ReceiptTypeRead, events.ReceiptTypePlayed:
		// Update read status
		if err := p.messageStore.UpdateReadStatus(ctx, message.ID, true); err != nil {
			slog.Error("failed to update read status", "error", err)
			return
		}

		// Update seen_by
		if message.SeenBy == nil {
			message.SeenBy = make(map[string]bool)
		}
		message.SeenBy[evt.Sender.String()] = true

		// Dispatch read event
		p.dispatcher.Dispatch(ctx, model.EventMessageRead, map[string]any{
			"message_id": message.ID,
			"account_id": accountID,
			"status":     "read",
			"seen_by":    evt.Sender.String(),
		})
	}
}

// handleGroupInfoEvent processes group info update events.
func (p *WhatsAppProvider) handleGroupInfoEvent(accountID string, evt *events.GroupInfo) {
	ctx := context.Background()

	// Get chat by provider ID
	chatJID := evt.JID.String()
	chat, err := p.chatStore.GetByProviderID(ctx, accountID, chatJID)
	if err != nil {
		slog.Error("failed to get chat for group info", "error", err)
		return
	}
	if chat == nil {
		// Group not tracked yet
		return
	}

	// Update chat name if changed
	if evt.Name != nil {
		name := evt.Name.Name
		if chat.Name == nil || *chat.Name != name {
			chat.Name = &name
			p.dispatcher.Dispatch(ctx, model.EventAccountStatusChange, map[string]any{
				"chat_id":     chat.ID,
				"account_id":  accountID,
				"change_type": "group_name_updated",
				"new_name":    name,
			})
		}
	}

	// Handle participant changes (users joining)
	if len(evt.Join) > 0 {
		for _, participant := range evt.Join {
			p.dispatcher.Dispatch(ctx, model.EventAccountStatusChange, map[string]any{
				"chat_id":         chat.ID,
				"account_id":      accountID,
				"change_type":     "participant_added",
				"participant_jid": participant.String(),
			})
		}
	}

	if len(evt.Leave) > 0 {
		for _, participant := range evt.Leave {
			p.dispatcher.Dispatch(ctx, model.EventAccountStatusChange, map[string]any{
				"chat_id":         chat.ID,
				"account_id":      accountID,
				"change_type":     "participant_removed",
				"participant_jid": participant.String(),
			})
		}
	}
}

// createChatFromEvent creates a Chat model from a message event.
func (p *WhatsAppProvider) createChatFromEvent(evt *events.Message, accountID string) *model.Chat {
	chatJID := evt.Info.Chat
	senderJID := evt.Info.Sender

	// Determine if group
	isGroup := isGroupJID(chatJID)

	chatType := model.ChatTypeOneToOne
	if isGroup {
		chatType = model.ChatTypeGroup
	}

	// Build attendees list
	attendees := []model.Attendee{
		{
			Object:         "attendee",
			AccountID:      accountID,
			Provider:       p.Name(),
			ProviderID:     senderJID.String(),
			Identifier:     senderJID.User,
			IdentifierType: string(model.IdentifierTypePhoneNumber),
		},
	}

	// For groups, add the group itself as an attendee context
	if isGroup {
		attendees = append(attendees, model.Attendee{
			Object:         "attendee",
			AccountID:      accountID,
			Provider:       p.Name(),
			ProviderID:     chatJID.String(),
			Identifier:     chatJID.String(),
			IdentifierType: string(model.IdentifierTypeProviderID),
		})
	}

	chat := &model.Chat{
		Object:      "chat",
		AccountID:   accountID,
		Provider:    p.Name(),
		ProviderID:  chatJID.String(),
		Type:        string(chatType),
		IsGroup:     isGroup,
		Attendees:   attendees,
		UnreadCount: 0,
	}

	return chat
}

// handleHistorySyncEvent processes bulk history sync from WhatsApp.
func (p *WhatsAppProvider) handleHistorySyncEvent(accountID string, evt *events.HistorySync) {
	ctx := context.Background()
	if evt.Data == nil {
		return
	}

	conversations := evt.Data.GetConversations()
	slog.Info("processing history sync", "account_id", accountID, "conversations", len(conversations))

	for _, conv := range conversations {
		chatJID := conv.GetID()
		if chatJID == "" {
			continue
		}

		// Get or create chat
		chat, err := p.chatStore.GetByProviderID(ctx, accountID, chatJID)
		if err != nil {
			slog.Error("failed to get chat for history sync", "error", err)
			continue
		}

		if chat == nil {
			// Create the chat
			isGroup := len(chatJID) > 0 && (len(chatJID) > 15) // rough heuristic for group JIDs
			chatType := model.ChatTypeOneToOne
			if isGroup {
				chatType = model.ChatTypeGroup
			}
			newChat := &model.Chat{
				Object:     "chat",
				AccountID:  accountID,
				Provider:   p.Name(),
				ProviderID: chatJID,
				Type:       string(chatType),
				IsGroup:    isGroup,
			}
			chat, err = p.chatStore.Create(ctx, newChat)
			if err != nil {
				slog.Error("failed to create chat from history sync", "error", err)
				continue
			}
		}

		// Process messages in this conversation
		for _, histMsg := range conv.GetMessages() {
			wMsg := histMsg.GetMessage()
			if wMsg == nil || wMsg.GetMessage() == nil {
				continue
			}

			// Check for duplicate
			msgKey := wMsg.GetKey()
			if msgKey == nil {
				continue
			}
			providerID := msgKey.GetID()
			existing, _ := p.messageStore.GetByProviderID(ctx, accountID, providerID)
			if existing != nil {
				continue
			}

			// Build message
			senderID := ""
			if msgKey.GetParticipant() != "" {
				senderID = msgKey.GetParticipant()
			} else if msgKey.GetRemoteJID() != "" {
				senderID = msgKey.GetRemoteJID()
			}

			msgText := ""
			if wMsg.GetMessage().GetConversation() != "" {
				msgText = wMsg.GetMessage().GetConversation()
			} else if wMsg.GetMessage().GetExtendedTextMessage() != nil {
				msgText = wMsg.GetMessage().GetExtendedTextMessage().GetText()
			}

			message := &model.Message{
				Object:     "message",
				ChatID:     chat.ID,
				AccountID:  accountID,
				Provider:   p.Name(),
				ProviderID: providerID,
				Text:       msgText,
				SenderID:   senderID,
				IsSender:   msgKey.GetFromMe(),
				Timestamp:   time.Unix(int64(wMsg.GetMessageTimestamp()), 0),
				Seen:       true,
				Delivered:  true,
			}

			if _, err := p.messageStore.Create(ctx, message); err != nil {
				slog.Error("failed to store history message", "error", err)
			}
		}

		// Update chat with last message info if available
		if msgs := conv.GetMessages(); len(msgs) > 0 {
			last := msgs[len(msgs)-1].GetMessage()
			if last != nil && last.GetMessage() != nil {
				preview := last.GetMessage().GetConversation()
				_ = p.chatStore.UpdateLastMessage(ctx, chat.ID, &preview)
			}
		}
	}

	slog.Info("history sync complete", "account_id", accountID, "conversations", len(conversations))
}

// getClient retrieves the whatsmeow client for an account.
func (p *WhatsAppProvider) getClient(accountID string) (*whatsmeow.Client, bool) {
	clientVal, ok := p.clients.Load(accountID)
	if !ok {
		return nil, false
	}
	return clientVal.(*whatsmeow.Client), true
}
