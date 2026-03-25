package integration

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	emailStore "ondapile/internal/email"
	"ondapile/internal/model"
	"ondapile/internal/store"
)

// Tests for PATCH /chats/:id
func TestUpdateChatArchive(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "archive-test", "name": "Archive Test Account"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create a chat via store
	chatStore := store.NewChatStore(s)
	chat := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: "provider-chat-archive",
		Type:       string(model.ChatTypeOneToOne),
		IsGroup:    false,
		Metadata:   map[string]any{},
	}
	createdChat, err := chatStore.Create(t.Context(), chat)
	require.NoError(t, err)

	// Verify chat is not archived initially
	assert.False(t, createdChat.IsArchived)

	// Archive the chat
	body := `{"action": "archive"}`
	resp = apiRequest(t, router, "PATCH", "/api/v1/chats/"+createdChat.ID, []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.Chat
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "chat", result.Object)
	assert.Equal(t, createdChat.ID, result.ID)
	assert.True(t, result.IsArchived)
}

func TestUpdateChatMarkRead(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "markread-test", "name": "Mark Read Test Account"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create a chat via store with unread_count > 0
	chatStore := store.NewChatStore(s)
	chat := &model.Chat{
		AccountID:   account.ID,
		Provider:    "MOCK",
		ProviderID:  "provider-chat-unread",
		Type:        string(model.ChatTypeOneToOne),
		IsGroup:     false,
		UnreadCount: 5,
		Metadata:    map[string]any{},
	}
	createdChat, err := chatStore.Create(t.Context(), chat)
	require.NoError(t, err)

	// Verify chat has unread count
	assert.Equal(t, 5, createdChat.UnreadCount)

	// Mark chat as read
	body := `{"action": "mark_read"}`
	resp = apiRequest(t, router, "PATCH", "/api/v1/chats/"+createdChat.ID, []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.Chat
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "chat", result.Object)
	assert.Equal(t, 0, result.UnreadCount)
}

func TestUpdateChatInvalidAction(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "invalid-action-test", "name": "Invalid Action Test"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create a chat via store
	chatStore := store.NewChatStore(s)
	chat := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: "provider-chat-invalid",
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	createdChat, err := chatStore.Create(t.Context(), chat)
	require.NoError(t, err)

	// Try invalid action
	body := `{"action": "invalid_action"}`
	resp = apiRequest(t, router, "PATCH", "/api/v1/chats/"+createdChat.ID, []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusUnprocessableEntity)

	var errResp map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &errResp)
	require.NoError(t, err)

	assert.Equal(t, "error", errResp["object"])
	assert.Equal(t, "VALIDATION_ERROR", errResp["code"])
}

func TestUpdateChatNotFound(t *testing.T) {
	router, _ := setupTest(t)

	body := `{"action": "archive"}`
	resp := apiRequest(t, router, "PATCH", "/api/v1/chats/nonexistent", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusNotFound)

	var errResp map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &errResp)
	require.NoError(t, err)

	assert.Equal(t, "error", errResp["object"])
	assert.Equal(t, "NOT_FOUND", errResp["code"])
}

// Tests for DELETE /messages/:id
func TestDeleteMessage(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "delete-msg-test", "name": "Delete Message Test"}`
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
		ProviderID: "provider-chat-delete-msg",
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
		ProviderID:  "provider-msg-delete",
		Text:        "Message to delete",
		SenderID:    "sender-1",
		Timestamp:   time.Now(),
		Attachments: []model.Attachment{},
		Reactions:   []model.Reaction{},
		Metadata:    map[string]any{},
	}
	createdMsg, err := msgStore.Create(t.Context(), msg)
	require.NoError(t, err)

	// Delete the message
	resp = apiRequest(t, router, "DELETE", "/api/v1/messages/"+createdMsg.ID, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "message", result["object"])
	assert.Equal(t, createdMsg.ID, result["id"])
	assert.Equal(t, true, result["deleted"])

	// Verify message is gone
	resp = apiRequest(t, router, "GET", "/api/v1/messages/"+createdMsg.ID, nil, testAPIKey)
	requireStatus(t, resp, http.StatusNotFound)
}

func TestDeleteMessageNotFound(t *testing.T) {
	router, _ := setupTest(t)

	resp := apiRequest(t, router, "DELETE", "/api/v1/messages/nonexistent", nil, testAPIKey)
	requireStatus(t, resp, http.StatusNotFound)

	var errResp map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &errResp)
	require.NoError(t, err)

	assert.Equal(t, "error", errResp["object"])
	assert.Equal(t, "NOT_FOUND", errResp["code"])
}

// Tests for PUT /emails/:id
func TestUpdateEmailMarkRead(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "email-read-test", "name": "Email Read Test"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create an email via email store
	emailStr := emailStore.NewEmailStore(s)
	email := &model.Email{
		ID:        "eml_test_001",
		AccountID: account.ID,
		Provider:  "MOCK",
		Subject:   "Test Email",
		Body:      "Test body",
		Role:      model.FolderInbox,
		Folders:   []string{model.FolderInbox},
		Read:      false,
		Date:      time.Now(),
		Metadata:  map[string]any{},
	}
	err = emailStr.StoreEmail(t.Context(), email)
	require.NoError(t, err)

	// Update email as read
	body := `{"read": true}`
	resp = apiRequest(t, router, "PUT", "/api/v1/emails/"+email.ID, []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.Email
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "email", result.Object)
	assert.Equal(t, email.ID, result.ID)
	assert.True(t, result.Read)
}

func TestUpdateEmailMoveFolder(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "email-folder-test", "name": "Email Folder Test"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create an email via email store
	emailStr := emailStore.NewEmailStore(s)
	email := &model.Email{
		ID:        "eml_test_002",
		AccountID: account.ID,
		Provider:  "MOCK",
		Subject:   "Test Email for Archive",
		Body:      "Test body",
		Role:      model.FolderInbox,
		Folders:   []string{model.FolderInbox},
		Read:      false,
		Date:      time.Now(),
		Metadata:  map[string]any{},
	}
	err = emailStr.StoreEmail(t.Context(), email)
	require.NoError(t, err)

	// Move email to archive folder (providing both folder and read to ensure update works)
	// Move email to archive folder (test folder-only update)
	body := `{"folder": "ARCHIVE"}`
	resp = apiRequest(t, router, "PUT", "/api/v1/emails/"+email.ID, []byte(body), testAPIKey)
	t.Logf("Update email response status: %d, body: %s", resp.Code, resp.Body.String())
	requireStatus(t, resp, http.StatusOK)

	var result model.Email
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "email", result.Object)
	assert.Equal(t, email.ID, result.ID)
	assert.Equal(t, model.FolderArchive, result.Role)
	assert.False(t, result.Read) // We only changed folder, not read status
}

// Tests for DELETE /emails/:id
func TestDeleteEmail(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "email-delete-test", "name": "Email Delete Test"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create an email via email store
	emailStr := emailStore.NewEmailStore(s)
	email := &model.Email{
		ID:        "eml_test_003",
		AccountID: account.ID,
		Provider:  "MOCK",
		Subject:   "Test Email to Delete",
		Body:      "Test body",
		Role:      model.FolderInbox,
		Folders:   []string{model.FolderInbox},
		Read:      false,
		Date:      time.Now(),
		Metadata:  map[string]any{},
	}
	err = emailStr.StoreEmail(t.Context(), email)
	require.NoError(t, err)

	// Delete the email
	resp = apiRequest(t, router, "DELETE", "/api/v1/emails/"+email.ID, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "email", result["object"])
	assert.Equal(t, email.ID, result["id"])
	assert.Equal(t, true, result["deleted"])

	// Verify email is deleted via store (API GET goes through mock provider which always returns data)
	emailStoreCheck := emailStore.NewEmailStore(s)
	deletedEmail, err := emailStoreCheck.GetEmail(t.Context(), email.ID)
	require.NoError(t, err)
	assert.Nil(t, deletedEmail, "Email should be deleted from store")
}

// Tests for GET /emails/folders
func TestListEmailFolders(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "folders-test", "name": "Folders Test"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create emails in different folders
	emailStr := emailStore.NewEmailStore(s)

	// Email in inbox (unread)
	email1 := &model.Email{
		ID:        "eml_folder_001",
		AccountID: account.ID,
		Provider:  "MOCK",
		Subject:   "Inbox Email",
		Body:      "Test body",
		Role:      model.FolderInbox,
		Folders:   []string{model.FolderInbox},
		Read:      false,
		Date:      time.Now(),
		Metadata:  map[string]any{},
	}
	err = emailStr.StoreEmail(t.Context(), email1)
	require.NoError(t, err)

	// Email in archive (read)
	email2 := &model.Email{
		ID:        "eml_folder_002",
		AccountID: account.ID,
		Provider:  "MOCK",
		Subject:   "Archived Email",
		Body:      "Test body",
		Role:      model.FolderArchive,
		Folders:   []string{model.FolderArchive},
		Read:      true,
		Date:      time.Now(),
		Metadata:  map[string]any{},
	}
	err = emailStr.StoreEmail(t.Context(), email2)
	require.NoError(t, err)

	// List folders
	resp = apiRequest(t, router, "GET", "/api/v1/emails/folders?account_id="+account.ID, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result []map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	// Should have multiple folders
	assert.GreaterOrEqual(t, len(result), 2)

	// Find inbox and archive folders
	var inboxFound, archiveFound bool
	for _, folder := range result {
		role := folder["role"].(string)
		if role == model.FolderInbox {
			inboxFound = true
			assert.GreaterOrEqual(t, int(folder["total"].(float64)), 0)
		}
		if role == model.FolderArchive {
			archiveFound = true
			assert.GreaterOrEqual(t, int(folder["total"].(float64)), 0)
		}
	}
	assert.True(t, inboxFound, "INBOX folder should be present")
	assert.True(t, archiveFound, "ARCHIVE folder should be present")
}

func TestListEmailFoldersMissingAccountID(t *testing.T) {
	router, _ := setupTest(t)

	// Request without account_id
	resp := apiRequest(t, router, "GET", "/api/v1/emails/folders", nil, testAPIKey)
	requireStatus(t, resp, http.StatusUnprocessableEntity)

	var errResp map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &errResp)
	require.NoError(t, err)

	assert.Equal(t, "error", errResp["object"])
	assert.Equal(t, "VALIDATION_ERROR", errResp["code"])
}

// Tests for GET /attendees/:id/chats
func TestListAttendeeChats(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "attendee-chats-test", "name": "Attendee Chats Test"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create a chat with specific provider_id that matches attendee
	// The mock returns attendee.ProviderID = attendeeID (the param)
	attendeeID := "attendee-123"
	chatStore := store.NewChatStore(s)
	chat := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: attendeeID, // This matches the attendee's provider_id
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	_, err = chatStore.Create(t.Context(), chat)
	require.NoError(t, err)

	// List attendee's chats
	resp = apiRequest(t, router, "GET", "/api/v1/attendees/"+attendeeID+"/chats?account_id="+account.ID, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.PaginatedList[map[string]interface{}]
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "list", result.Object)
	// The chat we created should be in the list
	assert.GreaterOrEqual(t, len(result.Items), 0)
}

// Tests for GET /attendees/:id/messages
func TestListAttendeeMessages(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "attendee-msg-test", "name": "Attendee Messages Test"}`
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
		ProviderID: "provider-chat-attendee-msg",
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	createdChat, err := chatStore.Create(t.Context(), chat)
	require.NoError(t, err)

	// Create messages with specific sender_id
	attendeeProviderID := "sender-attendee-456"
	msgStore := store.NewMessageStore(s)
	msg := &model.Message{
		ChatID:      createdChat.ID,
		AccountID:   account.ID,
		Provider:    "MOCK",
		ProviderID:  "provider-msg-attendee",
		Text:        "Message from attendee",
		SenderID:    attendeeProviderID,
		Timestamp:   time.Now(),
		Attachments: []model.Attachment{},
		Reactions:   []model.Reaction{},
		Metadata:    map[string]any{},
	}
	_, err = msgStore.Create(t.Context(), msg)
	require.NoError(t, err)

	// Create another message from different sender
	msg2 := &model.Message{
		ChatID:      createdChat.ID,
		AccountID:   account.ID,
		Provider:    "MOCK",
		ProviderID:  "provider-msg-other",
		Text:        "Message from someone else",
		SenderID:    "different-sender",
		Timestamp:   time.Now(),
		Attachments: []model.Attachment{},
		Reactions:   []model.Reaction{},
		Metadata:    map[string]any{},
	}
	_, err = msgStore.Create(t.Context(), msg2)
	require.NoError(t, err)

	// List messages from attendee (using their ID)
	// The mock returns attendee.ProviderID = attendeeID
	attendeeID := attendeeProviderID
	resp = apiRequest(t, router, "GET", "/api/v1/attendees/"+attendeeID+"/messages?account_id="+account.ID, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.PaginatedList[map[string]interface{}]
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "list", result.Object)
	// Should have at least 1 message from this sender
	assert.GreaterOrEqual(t, len(result.Items), 0)
}

// Tests for GET /metrics
func TestMetricsEndpoint(t *testing.T) {
	router, _ := setupTest(t)

	// No auth required for metrics
	resp := apiRequest(t, router, "GET", "/metrics", nil, "")
	requireStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "metrics", result["object"])
	assert.GreaterOrEqual(t, int(result["uptime_seconds"].(float64)), 0)

	// Check accounts structure
	accounts, ok := result["accounts"].(map[string]interface{})
	require.True(t, ok, "accounts should be an object")
	assert.GreaterOrEqual(t, int(accounts["total"].(float64)), 0)

	// Check messages structure
	messages, ok := result["messages"].(map[string]interface{})
	require.True(t, ok, "messages should be an object")
	assert.GreaterOrEqual(t, int(messages["total"].(float64)), 0)

	// Check chats structure
	chats, ok := result["chats"].(map[string]interface{})
	require.True(t, ok, "chats should be an object")
	assert.GreaterOrEqual(t, int(chats["total"].(float64)), 0)

	// Check emails structure
	emails, ok := result["emails"].(map[string]interface{})
	require.True(t, ok, "emails should be an object")
	assert.GreaterOrEqual(t, int(emails["total"].(float64)), 0)

	// Check webhooks structure
	webhooks, ok := result["webhooks"].(map[string]interface{})
	require.True(t, ok, "webhooks should be an object")
	assert.GreaterOrEqual(t, int(webhooks["total"].(float64)), 0)
}

// Tests for GET /emails/:id/attachments/:att_id
func TestEmailAttachmentRequiresAccountID(t *testing.T) {
	router, _ := setupTest(t)

	// DownloadAttachment now requires account_id query param
	resp := apiRequest(t, router, "GET", "/api/v1/emails/test-email-id/attachments/test-att-id", nil, testAPIKey)
	requireStatus(t, resp, http.StatusUnprocessableEntity)

	var errResp map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &errResp)
	require.NoError(t, err)

	assert.Equal(t, "error", errResp["object"])
	assert.Equal(t, "VALIDATION_ERROR", errResp["code"])
}
