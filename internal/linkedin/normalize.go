package linkedin

import (
	"time"

	"ondapile/internal/model"

	"github.com/google/uuid"
)

// normalizeConversation converts a LinkedIn conversation response to a Chat model.
func normalizeConversation(raw map[string]interface{}, accountID string, provider string) *model.Chat {
	chat := &model.Chat{
		Object:    "chat",
		ID:        "chat_" + uuid.New().String(),
		AccountID: accountID,
		Provider:  provider,
		Type:      string(model.ChatTypeOneToOne),
		IsGroup:   false,
		Metadata:  raw,
	}

	if id, ok := raw["id"].(string); ok {
		chat.ID = id
		chat.ProviderID = id
	}

	if name, ok := raw["subject"].(string); ok {
		chat.Name = &name
	}

	if from, ok := raw["from"].(map[string]interface{}); ok {
		if participant, ok := from["participant"].(map[string]interface{}); ok {
			if person, ok := participant["person"].(map[string]interface{}); ok {
				firstName, _ := person["localizedFirstName"].(string)
				lastName, _ := person["localizedLastName"].(string)
				fullName := firstName + " " + lastName
				chat.Name = &fullName

				id, _ := person["id"].(string)
				if id != "" {
					chat.Attendees = []model.Attendee{
						{
							Object:         "attendee",
							AccountID:      accountID,
							Provider:       provider,
							ProviderID:     id,
							Name:           fullName,
							Identifier:     id,
							IdentifierType: string(model.IdentifierTypeProfileURL),
						},
					}
				}
			}
		}
	}

	chat.CreatedAt = time.Now()
	chat.UpdatedAt = time.Now()

	return chat
}

// normalizeConversationList converts a LinkedIn conversation list response to a slice of Chat models.
func normalizeConversationList(raw map[string]interface{}, accountID string, provider string) []model.Chat {
	var chats []model.Chat

	if elements, ok := raw["elements"].([]interface{}); ok {
		for _, elem := range elements {
			if elemMap, ok := elem.(map[string]interface{}); ok {
				chat := normalizeConversation(elemMap, accountID, provider)
				chats = append(chats, *chat)
			}
		}
	}

	return chats
}

// normalizeMessage converts a LinkedIn message response to a Message model.
func normalizeMessage(raw map[string]interface{}, accountID string, conversationID string, provider string) *model.Message {
	message := &model.Message{
		Object:    "message",
		ID:        "msg_" + uuid.New().String(),
		ChatID:    conversationID,
		AccountID: accountID,
		Provider:  provider,
		Metadata:  raw,
	}

	if id, ok := raw["id"].(string); ok {
		message.ID = id
		message.ProviderID = id
	}

	if text, ok := raw["text"].(string); ok {
		message.Text = text
	}

	if from, ok := raw["from"].(map[string]interface{}); ok {
		if person, ok := from["person"].(map[string]interface{}); ok {
			if id, ok := person["id"].(string); ok {
				message.SenderID = id
			}
		}
	}

	message.Timestamp = time.Now()
	message.Seen = true
	message.Delivered = true

	return message
}

// normalizeMessageList converts a LinkedIn message list response to a slice of Message models.
func normalizeMessageList(raw map[string]interface{}, accountID string, conversationID string, provider string) []model.Message {
	var messages []model.Message

	if elements, ok := raw["elements"].([]interface{}); ok {
		for _, elem := range elements {
			if elemMap, ok := elem.(map[string]interface{}); ok {
				message := normalizeMessage(elemMap, accountID, conversationID, provider)
				messages = append(messages, *message)
			}
		}
	}

	return messages
}
