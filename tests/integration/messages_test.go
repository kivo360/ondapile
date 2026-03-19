package integration

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ondapile/internal/model"
	"ondapile/internal/store"
)

// TestListMessages tests GET /api/v1/messages
func TestListMessages(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "msg-list-test", "name": "Message List Test"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create a chat
	chatStore := store.NewChatStore(s)
	chat := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: "provider-chat-msg-list",
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	createdChat, err := chatStore.Create(t.Context(), chat)
	require.NoError(t, err)

	// Create messages
	msgStore := store.NewMessageStore(s)
	for i := 0; i < 5; i++ {
		msg := &model.Message{
			ChatID:      createdChat.ID,
			AccountID:   account.ID,
			Provider:    "MOCK",
			ProviderID:  "provider-msg-list-" + string(rune('a'+i)),
			Text:        "Cross-chat message " + string(rune('1'+i)),
			SenderID:    "sender-1",
			Timestamp:   time.Now().Add(time.Duration(i) * time.Minute),
			Attachments: []model.Attachment{},
			Reactions:   []model.Reaction{},
			Metadata:    map[string]any{},
		}
		_, err := msgStore.Create(t.Context(), msg)
		require.NoError(t, err)
	}

	// List all messages (cross-chat)
	resp = apiRequest(t, router, "GET", "/api/v1/messages", nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.PaginatedList[map[string]interface{}]
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "list", result.Object)
	assert.Len(t, result.Items, 5)
}

// TestListMessagesFilterByAccount tests filtering messages by account_id
func TestListMessagesFilterByAccount(t *testing.T) {
	router, s := setupTest(t)

	// Create two accounts
	var accounts []model.Account
	for i := 1; i <= 2; i++ {
		accountBody := `{"provider": "MOCK", "identifier": "msg-filter-` + string(rune('0'+i)) + `", "name": "Account ` + string(rune('0'+i)) + `"}`
		resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
		requireStatus(t, resp, http.StatusCreated)

		var account model.Account
		err := json.Unmarshal(resp.Body.Bytes(), &account)
		require.NoError(t, err)
		accounts = append(accounts, account)
	}

	// Create chats and messages for each account
	chatStore := store.NewChatStore(s)
	msgStore := store.NewMessageStore(s)

	for i, account := range accounts {
		chat := &model.Chat{
			AccountID:  account.ID,
			Provider:   "MOCK",
			ProviderID: "provider-chat-filter-" + string(rune('a'+i)),
			Type:       string(model.ChatTypeOneToOne),
			Metadata:   map[string]any{},
		}
		createdChat, err := chatStore.Create(t.Context(), chat)
		require.NoError(t, err)

		// Create 2 messages per account
		for j := 0; j < 2; j++ {
			msg := &model.Message{
				ChatID:      createdChat.ID,
				AccountID:   account.ID,
				Provider:    "MOCK",
				ProviderID:  "provider-msg-filter-" + string(rune('a'+i)) + "-" + string(rune('0'+j)),
				Text:        "Message " + string(rune('1'+j)),
				SenderID:    "sender-1",
				Timestamp:   time.Now().Add(time.Duration(j) * time.Minute),
				Attachments: []model.Attachment{},
				Reactions:   []model.Reaction{},
				Metadata:    map[string]any{},
			}
			_, err := msgStore.Create(t.Context(), msg)
			require.NoError(t, err)
		}
	}

	// Filter by first account
	resp := apiRequest(t, router, "GET", "/api/v1/messages?account_id="+accounts[0].ID, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.PaginatedList[map[string]interface{}]
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "list", result.Object)
	assert.Len(t, result.Items, 2)

	// Verify all messages belong to the correct account
	for _, item := range result.Items {
		assert.Equal(t, accounts[0].ID, item["account_id"])
	}
}

// TestGetMessage tests GET /api/v1/messages/:id
func TestGetMessage(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "msg-get-test", "name": "Message Get Test"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create a chat
	chatStore := store.NewChatStore(s)
	chat := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: "provider-chat-msg-get",
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	createdChat, err := chatStore.Create(t.Context(), chat)
	require.NoError(t, err)

	// Create a message
	msgStore := store.NewMessageStore(s)
	msg := &model.Message{
		ChatID:      createdChat.ID,
		AccountID:   account.ID,
		Provider:    "MOCK",
		ProviderID:  "provider-msg-get-123",
		Text:        "Test message for get",
		SenderID:    "sender-1",
		Timestamp:   time.Now(),
		Attachments: []model.Attachment{},
		Reactions:   []model.Reaction{},
		Metadata:    map[string]any{"test": "value"},
	}
	createdMsg, err := msgStore.Create(t.Context(), msg)
	require.NoError(t, err)

	// Get the message
	resp = apiRequest(t, router, "GET", "/api/v1/messages/"+createdMsg.ID, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.Message
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "message", result.Object)
	assert.Equal(t, createdMsg.ID, result.ID)
	assert.Equal(t, createdChat.ID, result.ChatID)
	assert.Equal(t, account.ID, result.AccountID)
	assert.Equal(t, "Test message for get", result.Text)
}

// TestGetMessageNotFound tests GET /api/v1/messages/:id with non-existent ID
func TestGetMessageNotFound(t *testing.T) {
	router, _ := setupTest(t)

	resp := apiRequest(t, router, "GET", "/api/v1/messages/nonexistent", nil, testAPIKey)
	requireStatus(t, resp, http.StatusNotFound)

	var errResp map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &errResp)
	require.NoError(t, err)

	assert.Equal(t, "error", errResp["object"])
	assert.Equal(t, "NOT_FOUND", errResp["code"])
}

// TestAddReaction tests POST /api/v1/messages/:id/reactions
func TestAddReaction(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "reaction-test", "name": "Reaction Test"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create a chat
	chatStore := store.NewChatStore(s)
	chat := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: "provider-chat-reaction",
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	createdChat, err := chatStore.Create(t.Context(), chat)
	require.NoError(t, err)

	// Create a message
	msgStore := store.NewMessageStore(s)
	msg := &model.Message{
		ChatID:      createdChat.ID,
		AccountID:   account.ID,
		Provider:    "MOCK",
		ProviderID:  "provider-msg-reaction-123",
		Text:        "Message to react to",
		SenderID:    "sender-1",
		Timestamp:   time.Now(),
		Attachments: []model.Attachment{},
		Reactions:   []model.Reaction{},
		Metadata:    map[string]any{},
	}
	createdMsg, err := msgStore.Create(t.Context(), msg)
	require.NoError(t, err)

	// Add reaction
	body := `{"emoji": "👍"}`
	resp = apiRequest(t, router, "POST", "/api/v1/messages/"+createdMsg.ID+"/reactions", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "message", result["object"])
	assert.Equal(t, createdMsg.ID, result["id"])
	assert.Equal(t, "👍", result["reaction_added"])
}

// TestAddReactionNotFound tests adding reaction to non-existent message
func TestAddReactionNotFound(t *testing.T) {
	router, _ := setupTest(t)

	body := `{"emoji": "👍"}`
	resp := apiRequest(t, router, "POST", "/api/v1/messages/nonexistent/reactions", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusNotFound)

	var errResp map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &errResp)
	require.NoError(t, err)

	assert.Equal(t, "error", errResp["object"])
	assert.Equal(t, "NOT_FOUND", errResp["code"])
}

// TestAddReactionMissingEmoji tests validation for missing emoji field
func TestAddReactionMissingEmoji(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "reaction-missing-test", "name": "Reaction Missing Test"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create a chat and message
	chatStore := store.NewChatStore(s)
	chat := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: "provider-chat-reaction-missing",
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	createdChat, err := chatStore.Create(t.Context(), chat)
	require.NoError(t, err)

	msgStore := store.NewMessageStore(s)
	msg := &model.Message{
		ChatID:      createdChat.ID,
		AccountID:   account.ID,
		Provider:    "MOCK",
		ProviderID:  "provider-msg-reaction-missing-123",
		Text:        "Message to react to",
		SenderID:    "sender-1",
		Timestamp:   time.Now(),
		Attachments: []model.Attachment{},
		Reactions:   []model.Reaction{},
		Metadata:    map[string]any{},
	}
	createdMsg, err := msgStore.Create(t.Context(), msg)
	require.NoError(t, err)

	// Try to add reaction without emoji
	body := `{}`
	resp = apiRequest(t, router, "POST", "/api/v1/messages/"+createdMsg.ID+"/reactions", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusUnprocessableEntity)

	var errResp map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &errResp)
	require.NoError(t, err)

	assert.Equal(t, "error", errResp["object"])
	assert.Equal(t, "VALIDATION_ERROR", errResp["code"])
}

// TestListMessagesPagination tests pagination with cursor and limit
func TestListMessagesPagination(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "msg-page-test", "name": "Message Pagination Test"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create a chat
	chatStore := store.NewChatStore(s)
	chat := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: "provider-chat-page",
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	createdChat, err := chatStore.Create(t.Context(), chat)
	require.NoError(t, err)

	// Create 10 messages
	msgStore := store.NewMessageStore(s)
	var messageIDs []string
	for i := 0; i < 10; i++ {
		msg := &model.Message{
			ChatID:      createdChat.ID,
			AccountID:   account.ID,
			Provider:    "MOCK",
			ProviderID:  "provider-msg-page-" + string(rune('0'+i)),
			Text:        "Message " + string(rune('0'+i)),
			SenderID:    "sender-1",
			Timestamp:   time.Now().Add(time.Duration(i) * time.Second),
			Attachments: []model.Attachment{},
			Reactions:   []model.Reaction{},
			Metadata:    map[string]any{},
		}
		createdMsg, err := msgStore.Create(t.Context(), msg)
		require.NoError(t, err)
		messageIDs = append(messageIDs, createdMsg.ID)
	}

	// Get first page with limit 3
	resp = apiRequest(t, router, "GET", "/api/v1/messages?limit=3", nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var page1 model.PaginatedList[map[string]interface{}]
	err = json.Unmarshal(resp.Body.Bytes(), &page1)
	require.NoError(t, err)

	assert.Equal(t, "list", page1.Object)
	assert.Len(t, page1.Items, 3)
	assert.True(t, page1.HasMore, "Should have more pages with 10 messages and limit 3")
	assert.NotEmpty(t, page1.Cursor)

	// Get second page using cursor
	resp = apiRequest(t, router, "GET", "/api/v1/messages?limit=3&cursor="+page1.Cursor, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var page2 model.PaginatedList[map[string]interface{}]
	err = json.Unmarshal(resp.Body.Bytes(), &page2)
	require.NoError(t, err)

	assert.Equal(t, "list", page2.Object)
	assert.Len(t, page2.Items, 3)
	assert.True(t, page2.HasMore, "Should still have more pages")

	// Verify pages have different items
	assert.NotEqual(t, page1.Items[0]["id"], page2.Items[0]["id"])
}
