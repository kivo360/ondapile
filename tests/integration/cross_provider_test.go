//go:build phase2
// +build phase2

package integration

// ============================================================================
// PHASE 2 TESTS — Cross-provider tests
// These tests will NOT compile until the Phase 2 branch merges, which adds
// the ability to register multiple providers with different names.
//
// Once Phase 2 merges, remove the "// +build phase2" directive above.
// ============================================================================

/*
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

// mockProviderWithName creates a MockProvider that returns a custom name.
func mockProviderWithName(name string) *MockProvider {
	return &MockProvider{
		NameFunc: func() string { return name },
	}
}

// ==================== Cross-Provider Registration ====================

func TestCrossProviderRegisterMultiple(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	// Register two mock providers with different names
	mockA := mockProviderWithName("PROVIDER_A")
	mockB := mockProviderWithName("PROVIDER_B")
	adapter.Register(mockA)
	adapter.Register(mockB)

	// Both should be retrievable
	provA, err := adapter.Get("PROVIDER_A")
	require.NoError(t, err)
	require.NotNil(t, provA)
	assert.Equal(t, "PROVIDER_A", provA.Name())

	provB, err := adapter.Get("PROVIDER_B")
	require.NoError(t, err)
	require.NotNil(t, provB)
	assert.Equal(t, "PROVIDER_B", provB.Name())
}

func TestCrossProviderListAll(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	adapter.Register(mockProviderWithName("PROVIDER_A"))
	adapter.Register(mockProviderWithName("PROVIDER_B"))

	names := adapter.List()
	assert.Contains(t, names, "PROVIDER_A")
	assert.Contains(t, names, "PROVIDER_B")
}

// ==================== Cross-Provider Chat Listing ====================

func TestCrossProviderGetChatsReturnsAllProviders(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)
	chatStore := store.NewChatStore(s)

	// Create accounts for different providers
	accA, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider: "PROVIDER_A", Name: "Account A", Identifier: "id-a",
		Status: string(model.StatusOperational),
	})
	require.NoError(t, err)

	accB, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider: "PROVIDER_B", Name: "Account B", Identifier: "id-b",
		Status: string(model.StatusOperational),
	})
	require.NoError(t, err)

	// Create chats for each account
	_, err = chatStore.Create(ctx, &model.Chat{
		AccountID: accA.ID, Provider: "PROVIDER_A", ProviderID: "chat-a-1",
		Type: "ONE_TO_ONE", Metadata: map[string]any{},
	})
	require.NoError(t, err)

	_, err = chatStore.Create(ctx, &model.Chat{
		AccountID: accB.ID, Provider: "PROVIDER_B", ProviderID: "chat-b-1",
		Type: "ONE_TO_ONE", Metadata: map[string]any{},
	})
	require.NoError(t, err)

	// List all chats (no account filter)
	chats, _, _, err := chatStore.List(ctx, nil, nil, "", 25)
	require.NoError(t, err)
	assert.Len(t, chats, 2, "Should return chats from all providers")

	providers := map[string]bool{}
	for _, c := range chats {
		providers[c.Provider] = true
	}
	assert.True(t, providers["PROVIDER_A"])
	assert.True(t, providers["PROVIDER_B"])
}

// ==================== Cross-Provider Message Listing ====================

func TestCrossProviderGetMessagesReturnsAllProviders(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)
	chatStore := store.NewChatStore(s)
	msgStore := store.NewMessageStore(s)

	// Create accounts + chats + messages for two providers
	accA, _ := accountStore.Create(ctx, store.CreateAccountParams{
		Provider: "PROVIDER_A", Name: "Msg Account A", Identifier: "msg-a",
		Status: string(model.StatusOperational),
	})
	accB, _ := accountStore.Create(ctx, store.CreateAccountParams{
		Provider: "PROVIDER_B", Name: "Msg Account B", Identifier: "msg-b",
		Status: string(model.StatusOperational),
	})

	chatA, _ := chatStore.Create(ctx, &model.Chat{
		AccountID: accA.ID, Provider: "PROVIDER_A", ProviderID: "msg-chat-a",
		Type: "ONE_TO_ONE", Metadata: map[string]any{},
	})
	chatB, _ := chatStore.Create(ctx, &model.Chat{
		AccountID: accB.ID, Provider: "PROVIDER_B", ProviderID: "msg-chat-b",
		Type: "ONE_TO_ONE", Metadata: map[string]any{},
	})

	_, err = msgStore.Create(ctx, &model.Message{
		ChatID: chatA.ID, AccountID: accA.ID, Provider: "PROVIDER_A",
		ProviderID: "msg-a-1", Text: "From A", SenderID: "sender-a",
		Timestamp: time.Now(), Attachments: []model.Attachment{},
		Reactions: []model.Reaction{}, Metadata: map[string]any{},
	})
	require.NoError(t, err)

	_, err = msgStore.Create(ctx, &model.Message{
		ChatID: chatB.ID, AccountID: accB.ID, Provider: "PROVIDER_B",
		ProviderID: "msg-b-1", Text: "From B", SenderID: "sender-b",
		Timestamp: time.Now(), Attachments: []model.Attachment{},
		Reactions: []model.Reaction{}, Metadata: map[string]any{},
	})
	require.NoError(t, err)

	// List all messages (no account filter)
	messages, _, _, err := msgStore.List(ctx, nil, "", 25)
	require.NoError(t, err)
	assert.Len(t, messages, 2, "Should return messages from all providers")

	providers := map[string]bool{}
	for _, m := range messages {
		providers[m.Provider] = true
	}
	assert.True(t, providers["PROVIDER_A"])
	assert.True(t, providers["PROVIDER_B"])
}

// ==================== Cross-Provider Account Filtering ====================

func TestCrossProviderFilterAccountsByProvider(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)

	_, err = accountStore.Create(ctx, store.CreateAccountParams{
		Provider: "PROVIDER_A", Name: "Filter A", Identifier: "filter-a",
		Status: string(model.StatusOperational),
	})
	require.NoError(t, err)

	_, err = accountStore.Create(ctx, store.CreateAccountParams{
		Provider: "PROVIDER_B", Name: "Filter B", Identifier: "filter-b",
		Status: string(model.StatusOperational),
	})
	require.NoError(t, err)

	_, err = accountStore.Create(ctx, store.CreateAccountParams{
		Provider: "PROVIDER_A", Name: "Filter A2", Identifier: "filter-a2",
		Status: string(model.StatusOperational),
	})
	require.NoError(t, err)

	// Filter by PROVIDER_A
	provA := "PROVIDER_A"
	accounts, _, _, err := accountStore.List(ctx, &provA, nil, "", 25)
	require.NoError(t, err)
	assert.Len(t, accounts, 2)
	for _, acc := range accounts {
		assert.Equal(t, "PROVIDER_A", acc.Provider)
	}

	// Filter by PROVIDER_B
	provB := "PROVIDER_B"
	accounts, _, _, err = accountStore.List(ctx, &provB, nil, "", 25)
	require.NoError(t, err)
	assert.Len(t, accounts, 1)
	assert.Equal(t, "PROVIDER_B", accounts[0].Provider)
}

func TestCrossProviderFilterAccountsViaAPI(t *testing.T) {
	router, _ := setupTest(t)

	// Register both providers
	adapter.Register(mockProviderWithName("PROV_X"))
	adapter.Register(mockProviderWithName("PROV_Y"))

	// Create accounts via API for each provider
	bodyX := `{"provider":"PROV_X","identifier":"api-x","credentials":{}}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(bodyX), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	bodyY := `{"provider":"PROV_Y","identifier":"api-y","credentials":{}}`
	resp = apiRequest(t, router, "POST", "/api/v1/accounts", []byte(bodyY), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	// List all — should get both
	resp = apiRequest(t, router, "GET", "/api/v1/accounts", nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)
	var allResult map[string]any
	json.Unmarshal(resp.Body.Bytes(), &allResult)
	items := allResult["items"].([]any)
	assert.Len(t, items, 2)

	// Filter by provider=PROV_X
	resp = apiRequest(t, router, "GET", "/api/v1/accounts?provider=PROV_X", nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)
	var xResult map[string]any
	json.Unmarshal(resp.Body.Bytes(), &xResult)
	xItems := xResult["items"].([]any)
	assert.Len(t, xItems, 1)
	assert.Equal(t, "PROV_X", xItems[0].(map[string]any)["provider"])
}

// ==================== Cross-Provider Isolation ====================

func TestCrossProviderChatIsolation(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)
	chatStore := store.NewChatStore(s)

	accA, _ := accountStore.Create(ctx, store.CreateAccountParams{
		Provider: "PROVIDER_A", Name: "Isolation A", Identifier: "iso-a",
		Status: string(model.StatusOperational),
	})
	accB, _ := accountStore.Create(ctx, store.CreateAccountParams{
		Provider: "PROVIDER_B", Name: "Isolation B", Identifier: "iso-b",
		Status: string(model.StatusOperational),
	})

	// Both providers can use the same provider_id without conflict
	// since the unique constraint is (account_id, provider_id)
	_, err = chatStore.Create(ctx, &model.Chat{
		AccountID: accA.ID, Provider: "PROVIDER_A", ProviderID: "same-chat-id",
		Type: "ONE_TO_ONE", Metadata: map[string]any{},
	})
	require.NoError(t, err)

	_, err = chatStore.Create(ctx, &model.Chat{
		AccountID: accB.ID, Provider: "PROVIDER_B", ProviderID: "same-chat-id",
		Type: "ONE_TO_ONE", Metadata: map[string]any{},
	})
	require.NoError(t, err, "Different accounts can have same provider_id")

	// Verify each can be retrieved independently
	chatA, err := chatStore.GetByProviderID(ctx, accA.ID, "same-chat-id")
	require.NoError(t, err)
	assert.Equal(t, "PROVIDER_A", chatA.Provider)

	chatB, err := chatStore.GetByProviderID(ctx, accB.ID, "same-chat-id")
	require.NoError(t, err)
	assert.Equal(t, "PROVIDER_B", chatB.Provider)

	assert.NotEqual(t, chatA.ID, chatB.ID, "Different internal IDs")
}

func TestCrossProviderMessageIsolation(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)
	chatStore := store.NewChatStore(s)
	msgStore := store.NewMessageStore(s)

	accA, _ := accountStore.Create(ctx, store.CreateAccountParams{
		Provider: "PROVIDER_A", Name: "Msg Iso A", Identifier: "msg-iso-a",
		Status: string(model.StatusOperational),
	})
	accB, _ := accountStore.Create(ctx, store.CreateAccountParams{
		Provider: "PROVIDER_B", Name: "Msg Iso B", Identifier: "msg-iso-b",
		Status: string(model.StatusOperational),
	})

	chatA, _ := chatStore.Create(ctx, &model.Chat{
		AccountID: accA.ID, Provider: "PROVIDER_A", ProviderID: "iso-chat",
		Type: "ONE_TO_ONE", Metadata: map[string]any{},
	})
	chatB, _ := chatStore.Create(ctx, &model.Chat{
		AccountID: accB.ID, Provider: "PROVIDER_B", ProviderID: "iso-chat",
		Type: "ONE_TO_ONE", Metadata: map[string]any{},
	})

	// Same provider message ID across different accounts should work
	_, err = msgStore.Create(ctx, &model.Message{
		ChatID: chatA.ID, AccountID: accA.ID, Provider: "PROVIDER_A",
		ProviderID: "same-msg-pid", Text: "Message A", SenderID: "sender-a",
		Timestamp: time.Now(), Attachments: []model.Attachment{},
		Reactions: []model.Reaction{}, Metadata: map[string]any{},
	})
	require.NoError(t, err)

	_, err = msgStore.Create(ctx, &model.Message{
		ChatID: chatB.ID, AccountID: accB.ID, Provider: "PROVIDER_B",
		ProviderID: "same-msg-pid", Text: "Message B", SenderID: "sender-b",
		Timestamp: time.Now(), Attachments: []model.Attachment{},
		Reactions: []model.Reaction{}, Metadata: map[string]any{},
	})
	require.NoError(t, err, "Different accounts can have same message provider_id")

	// Filter by account
	accAID := accA.ID
	msgsA, _, _, err := msgStore.List(ctx, &accAID, "", 25)
	require.NoError(t, err)
	assert.Len(t, msgsA, 1)
	assert.Equal(t, "Message A", msgsA[0].Text)
}
*/
