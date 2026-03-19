package linkedin

import (
	"context"
	"fmt"

	"ondapile/internal/adapter"
	"ondapile/internal/model"
)

// listConversations returns a list of LinkedIn messaging conversations.
func (a *LinkedInAdapter) listConversations(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Chat], error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	path := "/messagingConversations?q=webCommunicationView"
	if opts.Limit > 0 {
		path += fmt.Sprintf("&count=%d", opts.Limit)
	}

	var result map[string]interface{}
	if err := linkedinGet(client, path, &result); err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}

	chats := normalizeConversationList(result, accountID, a.Name())
	hasMore := false
	var nextCursor string

	return model.NewPaginatedList(chats, nextCursor, hasMore), nil
}

// getConversation returns a specific LinkedIn conversation.
func (a *LinkedInAdapter) getConversation(ctx context.Context, accountID string, conversationID string) (*model.Chat, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	path := fmt.Sprintf("/messagingConversations/%s", conversationID)

	var result map[string]interface{}
	if err := linkedinGet(client, path, &result); err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	return normalizeConversation(result, accountID, a.Name()), nil
}

// listMessages returns messages for a LinkedIn conversation.
func (a *LinkedInAdapter) listMessages(ctx context.Context, accountID string, conversationID string, opts adapter.ListOpts) (*model.PaginatedList[model.Message], error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	path := fmt.Sprintf("/messagingConversations/%s/messages", conversationID)
	if opts.Limit > 0 {
		path += fmt.Sprintf("?count=%d", opts.Limit)
	}

	var result map[string]interface{}
	if err := linkedinGet(client, path, &result); err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	messages := normalizeMessageList(result, accountID, conversationID, a.Name())
	hasMore := false
	var nextCursor string

	return model.NewPaginatedList(messages, nextCursor, hasMore), nil
}

// sendMessage sends a message via LinkedIn.
func (a *LinkedInAdapter) sendMessage(ctx context.Context, accountID string, conversationID string, msg adapter.SendMessageRequest) (*model.Message, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	body := map[string]interface{}{
		"conversationUrn": conversationID,
		"text":            msg.Text,
	}

	var result map[string]interface{}
	if err := linkedinPost(client, "/messagingMessages", body, &result); err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	message := normalizeMessage(result, accountID, conversationID, a.Name())
	if a.dispatcher != nil {
		a.dispatcher.Dispatch(ctx, model.EventMessageSent, message)
	}

	return message, nil
}

// createConversation creates a new LinkedIn conversation.
func (a *LinkedInAdapter) createConversation(ctx context.Context, accountID string, req adapter.StartChatRequest) (*model.Chat, error) {
	client, err := a.httpClient(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	body := map[string]interface{}{
		"recipients": []string{req.AttendeeIdentifier},
		"message": map[string]interface{}{
			"text": req.Text,
		},
	}

	var result map[string]interface{}
	if err := linkedinPost(client, "/messagingConversations", body, &result); err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	conversationID, _ := result["id"].(string)

	return &model.Chat{
		Object:     "chat",
		ID:         conversationID,
		AccountID:  accountID,
		Provider:   a.Name(),
		ProviderID: conversationID,
		Type:       string(model.ChatTypeOneToOne),
		IsGroup:    false,
	}, nil
}
