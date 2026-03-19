package instagram

import (
	"time"

	"ondapile/internal/model"
)

// normalizeChat converts Instagram conversation data to model.Chat
func normalizeChat(conv interface{}, accountID, provider string) *model.Chat {
	// Type assertion based on the expected structure
	// This is a simplified version - in production, you'd handle the actual API response structure
	chat := &model.Chat{
		Object:      "chat",
		AccountID:   accountID,
		Provider:    provider,
		Type:        string(model.ChatTypeOneToOne),
		Attendees:   []model.Attendee{},
		UnreadCount: 0,
		IsGroup:     false,
		IsArchived:  false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    map[string]any{},
	}

	// The actual implementation would extract data from the conversation object
	// based on the Instagram Graph API response structure

	return chat
}

// normalizeMessage converts Instagram message data to model.Message
func normalizeMessage(msg interface{}, chatID, accountID, provider string) *model.Message {
	message := &model.Message{
		Object:      "message",
		ChatID:      chatID,
		AccountID:   accountID,
		Provider:    provider,
		Attachments: []model.Attachment{},
		Reactions:   []model.Reaction{},
		Seen:        true,
		Delivered:   true,
		Edited:      false,
		Deleted:     false,
		Hidden:      false,
		IsEvent:     false,
		Metadata:    map[string]any{},
	}

	// The actual implementation would extract data from the message object
	// based on the Instagram Graph API response structure

	return message
}

// normalizeAttendee converts Instagram user data to model.Attendee
func normalizeAttendee(user interface{}, accountID, provider string) *model.Attendee {
	attendee := &model.Attendee{
		Object:         "attendee",
		AccountID:      accountID,
		Provider:       provider,
		IdentifierType: string(model.IdentifierTypeProviderID),
		IsSelf:         false,
		Metadata:       map[string]any{},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// The actual implementation would extract data from the user object
	// based on the Instagram Graph API response structure

	return attendee
}

// Now returns the current time - helper function
var Now = time.Now
