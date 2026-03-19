package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ondapile/internal/model"
	"ondapile/internal/store"
)

// TestListChats tests GET /api/v1/chats
func TestListChats(t *testing.T) {
	router, s := setupTest(t)

	// Create an account first
	accountBody := `{"provider": "MOCK", "identifier": "chat-list-test", "name": "Chat List Test Account"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create some chats directly in the database
	chatStore := store.NewChatStore(s)
	for i := 0; i < 3; i++ {
		chat := &model.Chat{
			AccountID:  account.ID,
			Provider:   "MOCK",
			ProviderID: "provider-chat-" + string(rune('a'+i)),
			Type:       string(model.ChatTypeOneToOne),
			IsGroup:    false,
			Metadata:   map[string]any{},
		}
		_, err := chatStore.Create(t.Context(), chat)
		require.NoError(t, err)
	}

	// List chats
	resp = apiRequest(t, router, "GET", "/api/v1/chats", nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.PaginatedList[map[string]interface{}]
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "list", result.Object)
	assert.Len(t, result.Items, 3)
}

// TestListChatsFilterByAccount tests filtering chats by account_id
func TestListChatsFilterByAccount(t *testing.T) {
	router, s := setupTest(t)

	// Create two accounts
	for i := 1; i <= 2; i++ {
		accountBody := `{"provider": "MOCK", "identifier": "chat-filter-` + string(rune('0'+i)) + `", "name": "Account ` + string(rune('0'+i)) + `"}`
		resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
		requireStatus(t, resp, http.StatusCreated)
	}

	// Get account list to find IDs
	resp := apiRequest(t, router, "GET", "/api/v1/accounts", nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var accountList model.PaginatedList[model.Account]
	err := json.Unmarshal(resp.Body.Bytes(), &accountList)
	require.NoError(t, err)
	require.Len(t, accountList.Items, 2)

	account1ID := accountList.Items[0].ID

	// Create chats for first account
	chatStore := store.NewChatStore(s)
	for i := 0; i < 2; i++ {
		chat := &model.Chat{
			AccountID:  account1ID,
			Provider:   "MOCK",
			ProviderID: "provider-chat-" + string(rune('a'+i)),
			Type:       string(model.ChatTypeOneToOne),
			Metadata:   map[string]any{},
		}
		_, err := chatStore.Create(t.Context(), chat)
		require.NoError(t, err)
	}

	// List chats filtered by account
	resp = apiRequest(t, router, "GET", "/api/v1/chats?account_id="+account1ID, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.PaginatedList[map[string]interface{}]
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "list", result.Object)
	assert.Len(t, result.Items, 2)
}

// TestGetChat tests GET /api/v1/chats/:id
func TestGetChat(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "chat-get-test", "name": "Chat Get Test"}`
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
		ProviderID: "provider-chat-123",
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{"test": "value"},
	}
	createdChat, err := chatStore.Create(t.Context(), chat)
	require.NoError(t, err)

	// Get the chat
	resp = apiRequest(t, router, "GET", "/api/v1/chats/"+createdChat.ID, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.Chat
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "chat", result.Object)
	assert.Equal(t, createdChat.ID, result.ID)
	assert.Equal(t, account.ID, result.AccountID)
	assert.Equal(t, "MOCK", result.Provider)
}

// TestGetChatNotFound tests GET /api/v1/chats/:id with non-existent ID
func TestGetChatNotFound(t *testing.T) {
	router, _ := setupTest(t)

	resp := apiRequest(t, router, "GET", "/api/v1/chats/nonexistent", nil, testAPIKey)
	requireStatus(t, resp, http.StatusNotFound)

	var errResp map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &errResp)
	require.NoError(t, err)

	assert.Equal(t, "error", errResp["object"])
	assert.Equal(t, "NOT_FOUND", errResp["code"])
}

// TestListChatMessages tests GET /api/v1/chats/:id/messages
func TestListChatMessages(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "chat-msg-test", "name": "Chat Messages Test"}`
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
		ProviderID: "provider-chat-msg",
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
			ProviderID:  "provider-msg-" + string(rune('a'+i)),
			Text:        "Message " + string(rune('1'+i)),
			SenderID:    "sender-1",
			Timestamp:   chat.CreatedAt.AddDate(0, 0, i),
			Attachments: []model.Attachment{},
			Reactions:   []model.Reaction{},
			Metadata:    map[string]any{},
		}
		_, err := msgStore.Create(t.Context(), msg)
		require.NoError(t, err)
	}

	// List messages
	resp = apiRequest(t, router, "GET", "/api/v1/chats/"+createdChat.ID+"/messages", nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.PaginatedList[map[string]interface{}]
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "list", result.Object)
	assert.Len(t, result.Items, 5)
}

// TestSendMessage tests POST /api/v1/chats/:id/messages
func TestSendMessage(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "send-msg-test", "name": "Send Message Test"}`
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
		ProviderID: "provider-chat-send",
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	createdChat, err := chatStore.Create(t.Context(), chat)
	require.NoError(t, err)

	// Send a message
	body := `{"text": "Hello from test!"}`
	resp = apiRequest(t, router, "POST", "/api/v1/chats/"+createdChat.ID+"/messages", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var result model.Message
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "message", result.Object)
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, createdChat.ID, result.ChatID)
	assert.Equal(t, "Hello from test!", result.Text)
}

// TestCreateChat tests POST /api/v1/chats
func TestCreateChat(t *testing.T) {
	router, _ := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "create-chat-test", "name": "Create Chat Test"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create a chat
	body := `{
		"account_id": "` + account.ID + `",
		"attendee_identifier": "user@example.com",
		"text": "Hello there!"
	}`
	resp = apiRequest(t, router, "POST", "/api/v1/chats", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var result model.Chat
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "chat", result.Object)
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, account.ID, result.AccountID)
	assert.Equal(t, "MOCK", result.Provider)
}

// TestCreateChatMissingFields tests validation for missing required fields
func TestCreateChatMissingFields(t *testing.T) {
	router, _ := setupTest(t)

	tests := []struct {
		name string
		body string
	}{
		{
			name: "missing account_id",
			body: `{"attendee_identifier": "user@example.com"}`,
		},
		{
			name: "missing attendee_identifier",
			body: `{"account_id": "test-id"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := apiRequest(t, router, "POST", "/api/v1/chats", []byte(tt.body), testAPIKey)
			requireStatus(t, resp, http.StatusUnprocessableEntity)

			var errResp map[string]interface{}
			err := json.Unmarshal(resp.Body.Bytes(), &errResp)
			require.NoError(t, err)

			assert.Equal(t, "error", errResp["object"])
			assert.Equal(t, "VALIDATION_ERROR", errResp["code"])
		})
	}
}

// TestDeleteChat tests DELETE /api/v1/chats/:id
func TestDeleteChat(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "delete-chat-test", "name": "Delete Chat Test"}`
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
		ProviderID: "provider-chat-delete",
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	createdChat, err := chatStore.Create(t.Context(), chat)
	require.NoError(t, err)

	// Delete the chat
	resp = apiRequest(t, router, "DELETE", "/api/v1/chats/"+createdChat.ID, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "chat", result["object"])
	assert.Equal(t, createdChat.ID, result["id"])
	assert.Equal(t, true, result["deleted"])

	// Verify chat is gone
	resp = apiRequest(t, router, "GET", "/api/v1/chats/"+createdChat.ID, nil, testAPIKey)
	requireStatus(t, resp, http.StatusNotFound)
}
