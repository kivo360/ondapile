package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ondapile/internal/adapter"
	"ondapile/internal/model"
	"ondapile/internal/store"
)

// TestAttendeesList tests GET /api/v1/attendees with account_id query param
func TestAttendeesList(t *testing.T) {
	router, _ := setupTest(t)

	// Create an account via API
	body := `{"provider": "MOCK", "identifier": "test-attendees-list", "name": "Test Account"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Register mock with ListAttendeesFunc returning 2 attendees
	mock := &MockProvider{
		ListAttendeesFunc: func(ctx context.Context, accountID string, opts adapter.ListOpts) (*model.PaginatedList[model.Attendee], error) {
			return &model.PaginatedList[model.Attendee]{
				Object: "list",
				Items: []model.Attendee{
					{Object: "attendee", ID: "att_1", AccountID: accountID, Provider: "MOCK", ProviderID: "prov-att-1", Name: "Alice", Identifier: "alice@test.com", IdentifierType: "EMAIL_ADDRESS", Metadata: map[string]any{}},
					{Object: "attendee", ID: "att_2", AccountID: accountID, Provider: "MOCK", ProviderID: "prov-att-2", Name: "Bob", Identifier: "bob@test.com", IdentifierType: "EMAIL_ADDRESS", Metadata: map[string]any{}},
				},
				HasMore: false,
			}, nil
		},
	}
	adapter.Register(mock)

	// GET /api/v1/attendees?account_id=ACCOUNT_ID
	resp = apiRequest(t, router, "GET", fmt.Sprintf("/api/v1/attendees?account_id=%s", account.ID), nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.PaginatedList[model.Attendee]
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "list", result.Object)
	assert.Len(t, result.Items, 2)
	assert.False(t, result.HasMore)
}

// TestAttendeesGet tests GET /api/v1/attendees/:id
func TestAttendeesGet(t *testing.T) {
	router, _ := setupTest(t)

	// Create an account via API
	body := `{"provider": "MOCK", "identifier": "test-attendees-get", "name": "Test Account"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Register mock with GetAttendeeFunc returning a specific attendee
	mock := &MockProvider{
		GetAttendeeFunc: func(ctx context.Context, accountID string, attendeeID string) (*model.Attendee, error) {
			return &model.Attendee{
				Object:         "attendee",
				ID:             attendeeID,
				AccountID:      accountID,
				Provider:       "MOCK",
				ProviderID:     "prov-att-test-1",
				Name:           "Test Attendee",
				Identifier:     "test@example.com",
				IdentifierType: "EMAIL_ADDRESS",
				AvatarURL:      "https://example.com/avatar.png",
				IsSelf:         false,
				Metadata:       map[string]any{"custom": "value"},
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}, nil
		},
	}
	adapter.Register(mock)

	// GET /api/v1/attendees/att_test_1?account_id=ACCOUNT_ID
	resp = apiRequest(t, router, "GET", fmt.Sprintf("/api/v1/attendees/att_test_1?account_id=%s", account.ID), nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.Attendee
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "attendee", result.Object)
	assert.Equal(t, "att_test_1", result.ID)
	assert.Equal(t, account.ID, result.AccountID)
	assert.Equal(t, "MOCK", result.Provider)
	assert.Equal(t, "Test Attendee", result.Name)
	assert.Equal(t, "test@example.com", result.Identifier)
	assert.Equal(t, "EMAIL_ADDRESS", result.IdentifierType)
	assert.Equal(t, "prov-att-test-1", result.ProviderID)
}

// TestAttendeesGetAvatar tests GET /api/v1/attendees/:id/avatar
func TestAttendeesGetAvatar(t *testing.T) {
	router, _ := setupTest(t)

	// Create an account via API
	body := `{"provider": "MOCK", "identifier": "test-avatar", "name": "Test Account"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Register mock with DownloadAttachmentFunc returning PNG magic bytes
	mock := &MockProvider{
		DownloadAttachmentFunc: func(ctx context.Context, accountID string, messageID string, attachmentID string) ([]byte, string, error) {
			return []byte{0x89, 0x50, 0x4E, 0x47}, "image/png", nil
		},
	}
	adapter.Register(mock)

	// GET /api/v1/attendees/att_1/avatar?account_id=ACCOUNT_ID
	resp = apiRequest(t, router, "GET", fmt.Sprintf("/api/v1/attendees/att_1/avatar?account_id=%s", account.ID), nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	// Verify Content-Type is "image/png"
	assert.Equal(t, "image/png", resp.Header().Get("Content-Type"))

	// Verify body bytes match
	expectedBytes := []byte{0x89, 0x50, 0x4E, 0x47}
	assert.Equal(t, expectedBytes, resp.Body.Bytes())
}

// TestAttendeesListChats tests GET /api/v1/attendees/:id/chats
func TestAttendeesListChats(t *testing.T) {
	router, s := setupTest(t)
	ctx := context.Background()

	// Create an account via API
	body := `{"provider": "MOCK", "identifier": "test-chats", "name": "Test Account"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create a chat in DB where provider_id matches the attendee's provider_id
	chatStore := store.NewChatStore(s)
	testChat := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: "attendee-prov-id-123",
		Type:       string(model.ChatTypeOneToOne),
		IsGroup:    false,
		Attendees:  []model.Attendee{},
		Metadata:   map[string]any{},
	}
	_, err = chatStore.Create(ctx, testChat)
	require.NoError(t, err)

	// Register mock with GetAttendeeFunc that returns an attendee with ProviderID matching the chat's provider_id
	mock := &MockProvider{
		GetAttendeeFunc: func(ctx context.Context, accountID string, attendeeID string) (*model.Attendee, error) {
			return &model.Attendee{
				Object:         "attendee",
				ID:             attendeeID,
				AccountID:      accountID,
				Provider:       "MOCK",
				ProviderID:     "attendee-prov-id-123", // Matches the chat's provider_id
				Name:           "Test Attendee",
				Identifier:     "test@example.com",
				IdentifierType: "EMAIL_ADDRESS",
				Metadata:       map[string]any{},
			}, nil
		},
	}
	adapter.Register(mock)

	// GET /api/v1/attendees/att_1/chats?account_id=ACCOUNT_ID
	resp = apiRequest(t, router, "GET", fmt.Sprintf("/api/v1/attendees/att_1/chats?account_id=%s", account.ID), nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.PaginatedList[map[string]interface{}]
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "list", result.Object)
	assert.Len(t, result.Items, 1)
	assert.False(t, result.HasMore)
}

// TestAttendeesListMessages tests GET /api/v1/attendees/:id/messages
func TestAttendeesListMessages(t *testing.T) {
	router, s := setupTest(t)
	ctx := context.Background()

	// Create an account via API
	body := `{"provider": "MOCK", "identifier": "test-messages", "name": "Test Account"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create a chat in DB
	chatStore := store.NewChatStore(s)
	testChat := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: "chat-prov-id-456",
		Type:       string(model.ChatTypeOneToOne),
		IsGroup:    false,
		Attendees:  []model.Attendee{},
		Metadata:   map[string]any{},
	}
	createdChat, err := chatStore.Create(ctx, testChat)
	require.NoError(t, err)

	// Create a message in DB where sender_id matches the attendee's provider_id
	messageStore := store.NewMessageStore(s)
	testMessage := &model.Message{
		ChatID:      createdChat.ID,
		AccountID:   account.ID,
		Provider:    "MOCK",
		ProviderID:  "msg-prov-id-789",
		Text:        "Hello from attendee",
		SenderID:    "attendee-sender-id-456", // This matches the attendee's provider_id
		IsSender:    false,
		Timestamp:   time.Now(),
		Attachments: []model.Attachment{},
		Reactions:   []model.Reaction{},
		Metadata:    map[string]any{},
	}
	_, err = messageStore.Create(ctx, testMessage)
	require.NoError(t, err)

	// Register mock GetAttendeeFunc returning attendee with ProviderID = sender_id of messages
	mock := &MockProvider{
		GetAttendeeFunc: func(ctx context.Context, accountID string, attendeeID string) (*model.Attendee, error) {
			return &model.Attendee{
				Object:         "attendee",
				ID:             attendeeID,
				AccountID:      accountID,
				Provider:       "MOCK",
				ProviderID:     "attendee-sender-id-456", // Matches the message's sender_id
				Name:           "Test Attendee",
				Identifier:     "test@example.com",
				IdentifierType: "EMAIL_ADDRESS",
				Metadata:       map[string]any{},
			}, nil
		},
	}
	adapter.Register(mock)

	// GET /api/v1/attendees/att_1/messages?account_id=ACCOUNT_ID
	resp = apiRequest(t, router, "GET", fmt.Sprintf("/api/v1/attendees/att_1/messages?account_id=%s", account.ID), nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.PaginatedList[map[string]interface{}]
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "list", result.Object)
	assert.Len(t, result.Items, 1)
	assert.False(t, result.HasMore)

	// Verify the message appears with correct text
	msg := result.Items[0]
	assert.Equal(t, "Hello from attendee", msg["text"])
}

// TestAttendeesGetNonexistent tests GET /api/v1/attendees/:id with non-existent ID
func TestAttendeesGetNonexistent(t *testing.T) {
	router, _ := setupTest(t)

	// Create an account via API
	body := `{"provider": "MOCK", "identifier": "test-nonexistent", "name": "Test Account"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Register mock with GetAttendeeFunc returning nil, nil
	mock := &MockProvider{
		GetAttendeeFunc: func(ctx context.Context, accountID string, attendeeID string) (*model.Attendee, error) {
			return nil, nil
		},
	}
	adapter.Register(mock)

	// GET /api/v1/attendees/nonexistent?account_id=ACCOUNT_ID
	resp = apiRequest(t, router, "GET", fmt.Sprintf("/api/v1/attendees/nonexistent?account_id=%s", account.ID), nil, testAPIKey)
	requireStatus(t, resp, http.StatusNotFound)

	var errResp map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &errResp)
	require.NoError(t, err)

	assert.Equal(t, "error", errResp["object"])
	assert.Equal(t, "NOT_FOUND", errResp["code"])
}

// TestAttendeesListRequiresAccountID tests that GET /api/v1/attendees without account_id returns 422
func TestAttendeesListRequiresAccountID(t *testing.T) {
	router, _ := setupTest(t)

	// GET /api/v1/attendees without account_id
	resp := apiRequest(t, router, "GET", "/api/v1/attendees", nil, testAPIKey)
	requireStatus(t, resp, http.StatusUnprocessableEntity)

	var errResp map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &errResp)
	require.NoError(t, err)

	assert.Equal(t, "error", errResp["object"])
	assert.Equal(t, "VALIDATION_ERROR", errResp["code"])
}

// TestAttendeesGetRequiresAccountID tests that GET /api/v1/attendees/:id without account_id returns 422
func TestAttendeesGetRequiresAccountID(t *testing.T) {
	router, _ := setupTest(t)

	// GET /api/v1/attendees/some-id without account_id
	resp := apiRequest(t, router, "GET", "/api/v1/attendees/some-id", nil, testAPIKey)
	requireStatus(t, resp, http.StatusUnprocessableEntity)

	var errResp map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &errResp)
	require.NoError(t, err)

	assert.Equal(t, "error", errResp["object"])
	assert.Equal(t, "VALIDATION_ERROR", errResp["code"])
}
