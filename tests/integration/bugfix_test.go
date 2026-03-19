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

// TestBugfix_ChatLastMessageScan verifies that last_message_at and last_message_preview
// are properly scanned into separate fields (Bug 1 fix).
func TestBugfix_ChatLastMessageScan(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "lastmsg-test", "name": "Last Message Test"}`
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
		ProviderID: "provider-chat-lastmsg",
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	createdChat, err := chatStore.Create(t.Context(), chat)
	require.NoError(t, err)

	// Update the last message using the store method
	preview := "This is the last message preview"
	err = chatStore.UpdateLastMessage(t.Context(), createdChat.ID, &preview)
	require.NoError(t, err)

	// Fetch the chat again - this should NOT produce a scan error
	// (Bug was that both columns scanned into the same field causing errors)
	updatedChat, err := chatStore.GetByID(t.Context(), createdChat.ID)
	require.NoError(t, err, "GetByID should not produce a scan error")
	require.NotNil(t, updatedChat)

	// Verify LastMessage is populated correctly
	require.NotNil(t, updatedChat.LastMessage, "LastMessage should be populated")
	assert.Equal(t, preview, updatedChat.LastMessage.Text)
	assert.False(t, updatedChat.LastMessage.Timestamp.IsZero(), "LastMessage timestamp should be set")
}

// TestBugfix_ChatPaginationCursor verifies that chat cursor pagination works correctly
// using timestamp-based cursors instead of UUID comparison (Bug 3 fix).
func TestBugfix_ChatPaginationCursor(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "chat-cursor-test", "name": "Chat Cursor Test"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create 5 chats with different updated_at timestamps
	chatStore := store.NewChatStore(s)
	var chatIDs []string

	for i := 0; i < 5; i++ {
		chat := &model.Chat{
			AccountID:  account.ID,
			Provider:   "MOCK",
			ProviderID: "provider-chat-cursor-" + string(rune('a'+i)),
			Type:       string(model.ChatTypeOneToOne),
			Metadata:   map[string]any{},
		}
		createdChat, err := chatStore.Create(t.Context(), chat)
		require.NoError(t, err)
		chatIDs = append(chatIDs, createdChat.ID)

		// Small delay to ensure different updated_at timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// Get first page with limit 2
	resp = apiRequest(t, router, "GET", "/api/v1/chats?account_id="+account.ID+"&limit=2", nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var page1 model.PaginatedList[map[string]interface{}]
	err = json.Unmarshal(resp.Body.Bytes(), &page1)
	require.NoError(t, err)

	assert.Equal(t, "list", page1.Object)
	assert.Len(t, page1.Items, 2, "First page should have 2 items")
	assert.True(t, page1.HasMore, "Should have more pages with 5 chats and limit 2")
	assert.NotEmpty(t, page1.Cursor, "Cursor should be present for pagination")

	page1IDs := []string{
		page1.Items[0]["id"].(string),
		page1.Items[1]["id"].(string),
	}

	// Get second page using cursor
	resp = apiRequest(t, router, "GET", "/api/v1/chats?account_id="+account.ID+"&limit=2&cursor="+page1.Cursor, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var page2 model.PaginatedList[map[string]interface{}]
	err = json.Unmarshal(resp.Body.Bytes(), &page2)
	require.NoError(t, err)

	assert.Equal(t, "list", page2.Object)
	assert.Len(t, page2.Items, 2, "Second page should have 2 items")
	assert.True(t, page2.HasMore, "Should still have more pages")

	page2IDs := []string{
		page2.Items[0]["id"].(string),
		page2.Items[1]["id"].(string),
	}

	// Verify pages have different items (no overlap)
	for _, id1 := range page1IDs {
		for _, id2 := range page2IDs {
			assert.NotEqual(t, id1, id2, "Page 1 and Page 2 should have different chats")
		}
	}

	// Get third page
	resp = apiRequest(t, router, "GET", "/api/v1/chats?account_id="+account.ID+"&limit=2&cursor="+page2.Cursor, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var page3 model.PaginatedList[map[string]interface{}]
	err = json.Unmarshal(resp.Body.Bytes(), &page3)
	require.NoError(t, err)

	assert.Equal(t, "list", page3.Object)
	assert.Len(t, page3.Items, 1, "Third page should have 1 item (5 total - 4 already seen)")
	assert.False(t, page3.HasMore, "Should not have more pages after seeing all 5")

	page3ID := page3.Items[0]["id"].(string)

	// Verify page 3 item is different from page 1 and 2
	for _, id1 := range page1IDs {
		assert.NotEqual(t, id1, page3ID, "Page 3 should have different chats from Page 1")
	}
	for _, id2 := range page2IDs {
		assert.NotEqual(t, id2, page3ID, "Page 3 should have different chats from Page 2")
	}
}

// TestBugfix_MessagePaginationCursor verifies that message cursor pagination
// works correctly using timestamp-based cursors.
func TestBugfix_MessagePaginationCursor(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "msg-cursor-test", "name": "Message Cursor Test"}`
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
		ProviderID: "provider-chat-msg-cursor",
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	createdChat, err := chatStore.Create(t.Context(), chat)
	require.NoError(t, err)

	// Create 5 messages with different timestamps (ascending order)
	msgStore := store.NewMessageStore(s)
	baseTime := time.Now().Add(-5 * time.Minute)

	for i := 0; i < 5; i++ {
		msg := &model.Message{
			ChatID:      createdChat.ID,
			AccountID:   account.ID,
			Provider:    "MOCK",
			ProviderID:  "provider-msg-cursor-" + string(rune('a'+i)),
			Text:        "Message " + string(rune('1'+i)),
			SenderID:    "sender-1",
			Timestamp:   baseTime.Add(time.Duration(i) * time.Second),
			Attachments: []model.Attachment{},
			Reactions:   []model.Reaction{},
			Metadata:    map[string]any{},
		}
		_, err := msgStore.Create(t.Context(), msg)
		require.NoError(t, err)
	}

	// Get first page with limit 2
	resp = apiRequest(t, router, "GET", "/api/v1/chats/"+createdChat.ID+"/messages?limit=2", nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var page1 model.PaginatedList[map[string]interface{}]
	err = json.Unmarshal(resp.Body.Bytes(), &page1)
	require.NoError(t, err)

	assert.Equal(t, "list", page1.Object)
	assert.Len(t, page1.Items, 2, "First page should have 2 items")
	assert.True(t, page1.HasMore, "Should have more pages with 5 messages and limit 2")
	assert.NotEmpty(t, page1.Cursor, "Cursor should be present")

	page1IDs := []string{
		page1.Items[0]["id"].(string),
		page1.Items[1]["id"].(string),
	}

	// Get second page using cursor
	resp = apiRequest(t, router, "GET", "/api/v1/chats/"+createdChat.ID+"/messages?limit=2&cursor="+page1.Cursor, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var page2 model.PaginatedList[map[string]interface{}]
	err = json.Unmarshal(resp.Body.Bytes(), &page2)
	require.NoError(t, err)

	assert.Equal(t, "list", page2.Object)
	assert.Len(t, page2.Items, 2, "Second page should have 2 items")
	assert.True(t, page2.HasMore, "Should still have more pages")

	page2IDs := []string{
		page2.Items[0]["id"].(string),
		page2.Items[1]["id"].(string),
	}

	// Verify pages have different items
	for _, id1 := range page1IDs {
		for _, id2 := range page2IDs {
			assert.NotEqual(t, id1, id2, "Page 1 and Page 2 should have different messages")
		}
	}

	// Get third page
	resp = apiRequest(t, router, "GET", "/api/v1/chats/"+createdChat.ID+"/messages?limit=2&cursor="+page2.Cursor, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var page3 model.PaginatedList[map[string]interface{}]
	err = json.Unmarshal(resp.Body.Bytes(), &page3)
	require.NoError(t, err)

	assert.Equal(t, "list", page3.Object)
	assert.Len(t, page3.Items, 1, "Third page should have 1 item")
	assert.False(t, page3.HasMore, "Should not have more pages after seeing all 5")

	page3ID := page3.Items[0]["id"].(string)

	// Verify page 3 item is different from previous pages
	for _, id1 := range page1IDs {
		assert.NotEqual(t, id1, page3ID, "Page 3 should have different messages from Page 1")
	}
	for _, id2 := range page2IDs {
		assert.NotEqual(t, id2, page3ID, "Page 3 should have different messages from Page 2")
	}

	// Test cross-chat message pagination (GET /api/v1/messages)
	resp = apiRequest(t, router, "GET", "/api/v1/messages?account_id="+account.ID+"&limit=2", nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var crossChatPage1 model.PaginatedList[map[string]interface{}]
	err = json.Unmarshal(resp.Body.Bytes(), &crossChatPage1)
	require.NoError(t, err)

	assert.Equal(t, "list", crossChatPage1.Object)
	assert.Len(t, crossChatPage1.Items, 2, "Cross-chat first page should have 2 items")

	if crossChatPage1.HasMore {
		// Get second page
		resp = apiRequest(t, router, "GET", "/api/v1/messages?account_id="+account.ID+"&limit=2&cursor="+crossChatPage1.Cursor, nil, testAPIKey)
		requireStatus(t, resp, http.StatusOK)

		var crossChatPage2 model.PaginatedList[map[string]interface{}]
		err = json.Unmarshal(resp.Body.Bytes(), &crossChatPage2)
		require.NoError(t, err)

		assert.Equal(t, "list", crossChatPage2.Object)
		// Verify different items
		assert.NotEqual(t, crossChatPage1.Items[0]["id"], crossChatPage2.Items[0]["id"])
	}
}
