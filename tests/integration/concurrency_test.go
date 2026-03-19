package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ondapile/internal/model"
	"ondapile/internal/store"
)

// TestConcurrentAccountCreation tests that 10 concurrent account creations via store all succeed
func TestConcurrentAccountCreation(t *testing.T) {
	router, s := setupTest(t)
	_ = router

	ctx := context.Background()
	accountStore := store.NewAccountStore(s)

	var wg sync.WaitGroup
	errs := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = accountStore.Create(ctx, store.CreateAccountParams{
				Provider:   "MOCK",
				Name:       fmt.Sprintf("Concurrent-%d", idx),
				Identifier: fmt.Sprintf("concurrent-%d", idx),
				Status:     string(model.StatusOperational),
			})
		}(i)
	}
	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}

	accounts, _, _, err := accountStore.List(ctx, nil, nil, "", 100)
	require.NoError(t, err)
	assert.Len(t, accounts, 10)
}

// TestConcurrentMessageSending tests that 10 concurrent message creations all succeed
func TestConcurrentMessageSending(t *testing.T) {
	router, s := setupTest(t)
	_ = router

	ctx := context.Background()
	accountStore := store.NewAccountStore(s)
	chatStore := store.NewChatStore(s)
	msgStore := store.NewMessageStore(s)

	// Create an account
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:   "MOCK",
		Name:       "Message Test Account",
		Identifier: "msg-test-account",
		Status:     string(model.StatusOperational),
	})
	require.NoError(t, err)

	// Create a chat
	chat := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: "chat-provider-id",
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	createdChat, err := chatStore.Create(ctx, chat)
	require.NoError(t, err)

	// Create 10 messages concurrently
	var wg sync.WaitGroup
	errs := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			msg := &model.Message{
				ChatID:      createdChat.ID,
				AccountID:   account.ID,
				Provider:    "MOCK",
				ProviderID:  fmt.Sprintf("provider-msg-%d", idx),
				Text:        fmt.Sprintf("Message %d", idx),
				SenderID:    "sender-1",
				Timestamp:   time.Now().Add(time.Duration(idx) * time.Millisecond),
				Attachments: []model.Attachment{},
				Reactions:   []model.Reaction{},
				Metadata:    map[string]any{},
			}
			_, errs[idx] = msgStore.Create(ctx, msg)
		}(i)
	}
	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}

	// Verify exactly 10 messages exist
	messages, _, _, err := msgStore.ListByChat(ctx, createdChat.ID, "", 100)
	require.NoError(t, err)
	assert.Len(t, messages, 10)
}

// TestConcurrentChatCreationSameProviderID tests that only 1 of 5 concurrent chat creations succeeds with same provider_id
func TestConcurrentChatCreationSameProviderID(t *testing.T) {
	router, s := setupTest(t)
	_ = router

	ctx := context.Background()
	accountStore := store.NewAccountStore(s)
	chatStore := store.NewChatStore(s)

	// Create an account
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:   "MOCK",
		Name:       "Chat Test Account",
		Identifier: "chat-test-account",
		Status:     string(model.StatusOperational),
	})
	require.NoError(t, err)

	// Try to create 5 chats with the same provider_id concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	successes := 0

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			chat := &model.Chat{
				AccountID:  account.ID,
				Provider:   "MOCK",
				ProviderID: "same-pid",
				Type:       string(model.ChatTypeOneToOne),
				Metadata:   map[string]any{},
			}
			_, err := chatStore.Create(ctx, chat)
			if err == nil {
				mu.Lock()
				successes++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, 1, successes, "Exactly one chat creation should succeed")
}

// TestConcurrentChatReads tests that 50 concurrent chat reads all return consistent data
func TestConcurrentChatReads(t *testing.T) {
	router, s := setupTest(t)
	_ = router

	ctx := context.Background()
	accountStore := store.NewAccountStore(s)
	chatStore := store.NewChatStore(s)

	// Create an account
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:   "MOCK",
		Name:       "Read Test Account",
		Identifier: "read-test-account",
		Status:     string(model.StatusOperational),
	})
	require.NoError(t, err)

	// Create 3 chats
	for i := 0; i < 3; i++ {
		chat := &model.Chat{
			AccountID:  account.ID,
			Provider:   "MOCK",
			ProviderID: fmt.Sprintf("provider-chat-%d", i),
			Type:       string(model.ChatTypeOneToOne),
			Metadata:   map[string]any{},
		}
		_, err := chatStore.Create(ctx, chat)
		require.NoError(t, err)
	}

	// Run 50 concurrent reads
	var wg sync.WaitGroup
	errs := make([]error, 50)
	counts := make([]int, 50)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			chats, _, _, err := chatStore.List(ctx, &account.ID, nil, "", 100)
			errs[idx] = err
			if err == nil {
				counts[idx] = len(chats)
			}
		}(i)
	}
	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}

	for _, count := range counts {
		assert.Equal(t, 3, count, "All reads should return 3 chats")
	}
}

// TestConcurrentAccountCreationViaAPI tests that 10 concurrent API account creations all succeed
func TestConcurrentAccountCreationViaAPI(t *testing.T) {
	router, _ := setupTest(t)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var createdIDs []string

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			body := fmt.Sprintf(`{"provider":"MOCK","identifier":"api-concurrent-%d","credentials":{}}`, idx)
			resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
			if resp.Code == http.StatusCreated {
				var acc map[string]any
				json.Unmarshal(resp.Body.Bytes(), &acc)
				mu.Lock()
				createdIDs = append(createdIDs, acc["id"].(string))
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	assert.Len(t, createdIDs, 10, "All 10 accounts should be created")

	// Verify all accounts exist via API
	resp := apiRequest(t, router, "GET", "/api/v1/accounts", nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.PaginatedList[map[string]interface{}]
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Len(t, result.Items, 10, "API should return all 10 accounts")
}

// TestConcurrentIncrementUnread tests that 20 concurrent IncrementUnread calls result in count of 20
func TestConcurrentIncrementUnread(t *testing.T) {
	router, s := setupTest(t)
	_ = router

	ctx := context.Background()
	accountStore := store.NewAccountStore(s)
	chatStore := store.NewChatStore(s)

	// Create an account
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:   "MOCK",
		Name:       "Unread Test Account",
		Identifier: "unread-test-account",
		Status:     string(model.StatusOperational),
	})
	require.NoError(t, err)

	// Create a chat
	chat := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: "unread-chat-pid",
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	createdChat, err := chatStore.Create(ctx, chat)
	require.NoError(t, err)

	// Run 20 concurrent IncrementUnread calls
	var wg sync.WaitGroup
	errs := make([]error, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = chatStore.IncrementUnread(ctx, createdChat.ID)
		}(i)
	}
	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}

	// Verify unread_count = 20
	updatedChat, err := chatStore.GetByID(ctx, createdChat.ID)
	require.NoError(t, err)
	assert.Equal(t, 20, updatedChat.UnreadCount, "Unread count should be 20 after 20 increments")
}

// TestConcurrentMixedReadWrite tests that concurrent readers and writers don't cause issues
func TestConcurrentMixedReadWrite(t *testing.T) {
	router, s := setupTest(t)
	_ = router

	ctx := context.Background()
	accountStore := store.NewAccountStore(s)
	chatStore := store.NewChatStore(s)
	msgStore := store.NewMessageStore(s)

	// Create an account
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:   "MOCK",
		Name:       "Mixed Test Account",
		Identifier: "mixed-test-account",
		Status:     string(model.StatusOperational),
	})
	require.NoError(t, err)

	// Create 3 chats
	chatIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		chat := &model.Chat{
			AccountID:  account.ID,
			Provider:   "MOCK",
			ProviderID: fmt.Sprintf("mixed-chat-%d", i),
			Type:       string(model.ChatTypeOneToOne),
			Metadata:   map[string]any{},
		}
		createdChat, err := chatStore.Create(ctx, chat)
		require.NoError(t, err)
		chatIDs[i] = createdChat.ID
	}

	var wg sync.WaitGroup
	errs := make([]error, 10)

	// 5 writers creating messages
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			chatID := chatIDs[idx%3]
			msg := &model.Message{
				ChatID:      chatID,
				AccountID:   account.ID,
				Provider:    "MOCK",
				ProviderID:  fmt.Sprintf("writer-msg-%d", idx),
				Text:        fmt.Sprintf("Writer message %d", idx),
				SenderID:    "sender-1",
				Timestamp:   time.Now(),
				Attachments: []model.Attachment{},
				Reactions:   []model.Reaction{},
				Metadata:    map[string]any{},
			}
			_, errs[idx] = msgStore.Create(ctx, msg)
		}(i)
	}

	// 5 readers listing messages
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			chatID := chatIDs[idx%3]
			_, _, _, errs[5+idx] = msgStore.ListByChat(ctx, chatID, "", 100)
		}(i)
	}

	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}
}

// TestRateLimitBurst tests that rapid sequential requests trigger rate limiting after burst
func TestRateLimitBurst(t *testing.T) {
	router, _ := setupTest(t)

	ok := 0
	limited := 0

	for i := 0; i < 150; i++ {
		resp := apiRequest(t, router, "GET", "/api/v1/accounts", nil, testAPIKey)
		if resp.Code == http.StatusOK {
			ok++
		} else if resp.Code == http.StatusTooManyRequests {
			limited++
		}
	}

	assert.Greater(t, ok, 50, "should have some successful requests")
	assert.Greater(t, limited, 0, "should have some rate-limited requests")
}
