package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ondapile/internal/email"
	"ondapile/internal/model"
	"ondapile/internal/store"
)

// ==================== Account Cascade Delete Tests ====================

// TestAccountDeleteCascadesToChats verifies that deleting an account removes associated chats
func TestAccountDeleteCascadesToChats(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)
	chatStore := store.NewChatStore(s)

	// Create account
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Cascade Test Account",
		Identifier:   "cascade-test-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Status:       string(model.StatusOperational),
		Capabilities: []string{},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)
	require.NotNil(t, account)

	// Create 2 chats
	chat1 := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: "cascade-chat-1-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	chat2 := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: "cascade-chat-2-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}

	createdChat1, err := chatStore.Create(ctx, chat1)
	require.NoError(t, err)
	createdChat2, err := chatStore.Create(ctx, chat2)
	require.NoError(t, err)

	// Delete account
	err = accountStore.Delete(ctx, account.ID)
	require.NoError(t, err)

	// Verify both chats are gone
	retrieved1, err := chatStore.GetByID(ctx, createdChat1.ID)
	require.NoError(t, err)
	assert.Nil(t, retrieved1)

	retrieved2, err := chatStore.GetByID(ctx, createdChat2.ID)
	require.NoError(t, err)
	assert.Nil(t, retrieved2)
}

// TestAccountDeleteCascadesToMessages verifies that deleting an account removes associated messages
func TestAccountDeleteCascadesToMessages(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)
	chatStore := store.NewChatStore(s)
	msgStore := store.NewMessageStore(s)

	// Create account
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Message Cascade Test",
		Identifier:   "msg-cascade-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Status:       string(model.StatusOperational),
		Capabilities: []string{},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Create chat
	chat := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: "msg-cascade-chat-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	createdChat, err := chatStore.Create(ctx, chat)
	require.NoError(t, err)

	// Create 3 messages
	for i := 0; i < 3; i++ {
		msg := &model.Message{
			ChatID:      createdChat.ID,
			AccountID:   account.ID,
			Provider:    "MOCK",
			ProviderID:  fmt.Sprintf("msg-cascade-%d-%d", i, time.Now().UnixNano()),
			Text:        fmt.Sprintf("Message %d", i),
			SenderID:    "sender-1",
			Timestamp:   time.Now(),
			Attachments: []model.Attachment{},
			Reactions:   []model.Reaction{},
			Metadata:    map[string]any{},
		}
		_, err := msgStore.Create(ctx, msg)
		require.NoError(t, err)
	}

	// Delete account
	err = accountStore.Delete(ctx, account.ID)
	require.NoError(t, err)

	// Verify messages are gone by listing (chat is also gone, so we verify account deletion cascades)
	messages, _, _, err := msgStore.List(ctx, &account.ID, "", 10)
	require.NoError(t, err)
	assert.Len(t, messages, 0)
}

// TestAccountDeleteCascadesToAttendees verifies that deleting an account removes associated attendees
func TestAccountDeleteCascadesToAttendees(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)

	// Create account
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Attendee Cascade Test",
		Identifier:   "attendee-cascade-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Status:       string(model.StatusOperational),
		Capabilities: []string{},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Insert attendee directly via SQL
	attendeeID := "attendee-" + fmt.Sprintf("%d", time.Now().UnixNano())
	_, err = testDBPool.Exec(ctx, `
		INSERT INTO attendees (id, account_id, provider, provider_id, name, identifier, identifier_type, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, attendeeID, account.ID, "MOCK", "provider-attendee-id", "Test Attendee", "test@example.com", "EMAIL_ADDRESS", `{}`)
	require.NoError(t, err)

	// Verify attendee exists
	var count int
	err = testDBPool.QueryRow(ctx, "SELECT COUNT(*) FROM attendees WHERE id = $1", attendeeID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Delete account
	err = accountStore.Delete(ctx, account.ID)
	require.NoError(t, err)

	// Verify attendee is gone
	err = testDBPool.QueryRow(ctx, "SELECT COUNT(*) FROM attendees WHERE id = $1", attendeeID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

// ==================== Duplicate Constraint Tests ====================

// TestChatDuplicateProviderIDConstraint verifies unique constraint on account_id + provider_id for chats
func TestChatDuplicateProviderIDConstraint(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)
	chatStore := store.NewChatStore(s)

	// Create account
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Duplicate Constraint Test",
		Identifier:   "dup-constraint-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Status:       string(model.StatusOperational),
		Capabilities: []string{},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Create first chat with provider_id "dup-123"
	chat1 := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: "dup-123",
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	_, err = chatStore.Create(ctx, chat1)
	require.NoError(t, err)

	// Try creating another chat with same account_id + provider_id
	chat2 := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: "dup-123",
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	_, err = chatStore.Create(ctx, chat2)
	require.Error(t, err) // Expect unique constraint violation
}

// TestMessageDuplicateProviderIDConstraint verifies unique constraint on account_id + provider_id for messages
func TestMessageDuplicateProviderIDConstraint(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)
	chatStore := store.NewChatStore(s)
	msgStore := store.NewMessageStore(s)

	// Create account
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Message Dup Test",
		Identifier:   "msg-dup-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Status:       string(model.StatusOperational),
		Capabilities: []string{},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Create chat
	chat := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: "msg-dup-chat-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	createdChat, err := chatStore.Create(ctx, chat)
	require.NoError(t, err)

	// Create first message with provider_id "msg-dup"
	msg1 := &model.Message{
		ChatID:      createdChat.ID,
		AccountID:   account.ID,
		Provider:    "MOCK",
		ProviderID:  "msg-dup",
		Text:        "First message",
		SenderID:    "sender-1",
		Timestamp:   time.Now(),
		Attachments: []model.Attachment{},
		Reactions:   []model.Reaction{},
		Metadata:    map[string]any{},
	}
	_, err = msgStore.Create(ctx, msg1)
	require.NoError(t, err)

	// Try creating another message with same account_id + provider_id
	msg2 := &model.Message{
		ChatID:      createdChat.ID,
		AccountID:   account.ID,
		Provider:    "MOCK",
		ProviderID:  "msg-dup",
		Text:        "Second message",
		SenderID:    "sender-1",
		Timestamp:   time.Now(),
		Attachments: []model.Attachment{},
		Reactions:   []model.Reaction{},
		Metadata:    map[string]any{},
	}
	_, err = msgStore.Create(ctx, msg2)
	require.Error(t, err) // Expect unique constraint violation
}

// ==================== Chat ListByAttendee Tests ====================

// TestChatListByAttendee verifies ListByAttendee returns matching ONE_TO_ONE chats
func TestChatListByAttendee(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)
	chatStore := store.NewChatStore(s)

	// Create account
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "ListByAttendee Test",
		Identifier:   "list-attendee-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Status:       string(model.StatusOperational),
		Capabilities: []string{},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	attendeeProviderID := "attendee-" + fmt.Sprintf("%d", time.Now().UnixNano())

	// Create 2 chats with matching provider_id (ONE_TO_ONE) - but we need unique provider_ids
	// The ListByAttendee matches on provider_id, so we need chats where provider_id equals attendeeProviderID
	// Due to unique constraint (account_id, provider_id), we can only have 1 chat per attendee per account
	chat1 := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: attendeeProviderID,
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	_, err = chatStore.Create(ctx, chat1)
	require.NoError(t, err)

	// Create another chat for same attendee with a second account (to avoid unique constraint)
	account2, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "ListByAttendee Test Account 2",
		Identifier:   "list-attendee-2-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Status:       string(model.StatusOperational),
		Capabilities: []string{},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	chat2 := &model.Chat{
		AccountID:  account2.ID,
		Provider:   "MOCK",
		ProviderID: attendeeProviderID,
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	_, err = chatStore.Create(ctx, chat2)
	require.NoError(t, err)

	// Create 1 chat with different provider_id (different attendee)
	chat3 := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: "different-provider-id",
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	_, err = chatStore.Create(ctx, chat3)
	require.NoError(t, err)

	// ListByAttendee for account1 should return 1 chat
	chats, _, _, err := chatStore.ListByAttendee(ctx, account.ID, attendeeProviderID, "", 25)
	require.NoError(t, err)
	assert.Len(t, chats, 1)

	// ListByAttendee for account2 should also return 1 chat
	chats, _, _, err = chatStore.ListByAttendee(ctx, account2.ID, attendeeProviderID, "", 25)
	require.NoError(t, err)
	assert.Len(t, chats, 1)
}

// TestChatListByAttendeeEmpty verifies ListByAttendee returns empty for non-matching provider_id
func TestChatListByAttendeeEmpty(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)
	chatStore := store.NewChatStore(s)

	// Create account
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "ListByAttendee Empty Test",
		Identifier:   "list-attendee-empty-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Status:       string(model.StatusOperational),
		Capabilities: []string{},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Create a chat
	chat := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: "existing-provider-id",
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	_, err = chatStore.Create(ctx, chat)
	require.NoError(t, err)

	// ListByAttendee with non-matching provider_id should return 0
	chats, _, _, err := chatStore.ListByAttendee(ctx, account.ID, "non-matching-id", "", 25)
	require.NoError(t, err)
	assert.Len(t, chats, 0)
}

// ==================== Pagination Edge Cases ====================

// TestPaginationLimitZeroDefaults verifies limit=0 defaults to 25
func TestPaginationLimitZeroDefaults(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)
	chatStore := store.NewChatStore(s)

	// Create account
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Pagination Test",
		Identifier:   "pagination-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Status:       string(model.StatusOperational),
		Capabilities: []string{},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Create 3 chats
	for i := 0; i < 3; i++ {
		chat := &model.Chat{
			AccountID:  account.ID,
			Provider:   "MOCK",
			ProviderID: fmt.Sprintf("pagination-chat-%d-%d", i, time.Now().UnixNano()),
			Type:       string(model.ChatTypeOneToOne),
			Metadata:   map[string]any{},
		}
		_, err := chatStore.Create(ctx, chat)
		require.NoError(t, err)
	}

	// Call with limit=0, should return all 3 (default is 25)
	chats, _, _, err := chatStore.List(ctx, &account.ID, nil, "", 0)
	require.NoError(t, err)
	assert.Len(t, chats, 3)
}

// TestPaginationEmptyResults verifies listing on empty table returns empty slice
func TestPaginationEmptyResults(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)

	// Create account (no chats)
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Empty Pagination Test",
		Identifier:   "empty-pagination-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Status:       string(model.StatusOperational),
		Capabilities: []string{},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	chatStore := store.NewChatStore(s)

	// List on empty table
	chats, nextCursor, hasMore, err := chatStore.List(ctx, &account.ID, nil, "", 10)
	require.NoError(t, err)
	assert.Len(t, chats, 0)
	assert.Empty(t, nextCursor)
	assert.False(t, hasMore)
}

// TestPaginationSinglePage verifies hasMore is false when items == limit
func TestPaginationSinglePage(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)
	chatStore := store.NewChatStore(s)

	// Create account
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Single Page Test",
		Identifier:   "single-page-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Status:       string(model.StatusOperational),
		Capabilities: []string{},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Create exactly 5 chats
	limit := 5
	for i := 0; i < limit; i++ {
		chat := &model.Chat{
			AccountID:  account.ID,
			Provider:   "MOCK",
			ProviderID: fmt.Sprintf("single-page-chat-%d-%d", i, time.Now().UnixNano()),
			Type:       string(model.ChatTypeOneToOne),
			Metadata:   map[string]any{},
		}
		_, err := chatStore.Create(ctx, chat)
		require.NoError(t, err)
	}

	// List with limit=5, hasMore should be false
	chats, _, hasMore, err := chatStore.List(ctx, &account.ID, nil, "", limit)
	require.NoError(t, err)
	assert.Len(t, chats, limit)
	assert.False(t, hasMore)
}

// TestPaginationExceedsLimit verifies hasMore is true when items > limit
func TestPaginationExceedsLimit(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)
	chatStore := store.NewChatStore(s)

	// Create account
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Exceeds Limit Test",
		Identifier:   "exceeds-limit-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Status:       string(model.StatusOperational),
		Capabilities: []string{},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Create limit+1 chats
	limit := 5
	for i := 0; i < limit+1; i++ {
		chat := &model.Chat{
			AccountID:  account.ID,
			Provider:   "MOCK",
			ProviderID: fmt.Sprintf("exceeds-chat-%d-%d", i, time.Now().UnixNano()),
			Type:       string(model.ChatTypeOneToOne),
			Metadata:   map[string]any{},
		}
		_, err := chatStore.Create(ctx, chat)
		require.NoError(t, err)
	}

	// List with limit=5, hasMore should be true, len=5
	chats, _, hasMore, err := chatStore.List(ctx, &account.ID, nil, "", limit)
	require.NoError(t, err)
	assert.Len(t, chats, limit)
	assert.True(t, hasMore)
}

// ==================== Unread Cycle ====================

// TestIncrementAndResetUnreadCycle verifies increment and reset of unread count
func TestIncrementAndResetUnreadCycle(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)
	chatStore := store.NewChatStore(s)

	// Create account
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Unread Cycle Test",
		Identifier:   "unread-cycle-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Status:       string(model.StatusOperational),
		Capabilities: []string{},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Create chat (unread should be 0)
	chat := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: "unread-cycle-chat-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	createdChat, err := chatStore.Create(ctx, chat)
	require.NoError(t, err)
	assert.Equal(t, 0, createdChat.UnreadCount)

	// Increment unread 3 times
	for i := 0; i < 3; i++ {
		err = chatStore.IncrementUnread(ctx, createdChat.ID)
		require.NoError(t, err)
	}

	// Verify unread=3
	retrieved, err := chatStore.GetByID(ctx, createdChat.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, retrieved.UnreadCount)

	// Reset unread
	err = chatStore.ResetUnread(ctx, createdChat.ID)
	require.NoError(t, err)

	// Verify unread=0
	retrieved, err = chatStore.GetByID(ctx, createdChat.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, retrieved.UnreadCount)
}

// ==================== Archive Toggle ====================

// TestArchiveUnarchiveToggle verifies archive/unarchive functionality
func TestArchiveUnarchiveToggle(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)
	chatStore := store.NewChatStore(s)

	// Create account
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Archive Toggle Test",
		Identifier:   "archive-toggle-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Status:       string(model.StatusOperational),
		Capabilities: []string{},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Create chat (is_archived should be false)
	chat := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: "archive-chat-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	createdChat, err := chatStore.Create(ctx, chat)
	require.NoError(t, err)
	assert.False(t, createdChat.IsArchived)

	// Archive chat
	err = chatStore.Archive(ctx, createdChat.ID, true)
	require.NoError(t, err)

	// Verify is_archived=true
	retrieved, err := chatStore.GetByID(ctx, createdChat.ID)
	require.NoError(t, err)
	assert.True(t, retrieved.IsArchived)

	// Unarchive chat
	err = chatStore.Archive(ctx, createdChat.ID, false)
	require.NoError(t, err)

	// Verify is_archived=false
	retrieved, err = chatStore.GetByID(ctx, createdChat.ID)
	require.NoError(t, err)
	assert.False(t, retrieved.IsArchived)
}

// ==================== Webhook Delivery Lifecycle ====================

// TestWebhookListActiveForEventFiltersCorrectly verifies ListActiveForEvent returns correct webhooks
func TestWebhookListActiveForEventFiltersCorrectly(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	webhookStore := store.NewWebhookStore(s)

	// Create webhook subscribed to ["message.received"]
	_, err = webhookStore.Create(ctx, "https://example.com/webhook1", []string{"message.received"}, "secret1")
	require.NoError(t, err)

	// Create webhook subscribed to ["chat.created"]
	_, err = webhookStore.Create(ctx, "https://example.com/webhook2", []string{"chat.created"}, "secret2")
	require.NoError(t, err)

	// Create webhook subscribed to ["message.received", "chat.created"]
	_, err = webhookStore.Create(ctx, "https://example.com/webhook3", []string{"message.received", "chat.created"}, "secret3")
	require.NoError(t, err)

	// ListActiveForEvent("message.received") should return 2 webhooks
	webhooks, err := webhookStore.ListActiveForEvent(ctx, "message.received")
	require.NoError(t, err)
	assert.Len(t, webhooks, 2)
}

// TestWebhookListActiveForEventExcludesInactive verifies inactive webhooks are excluded
func TestWebhookListActiveForEventExcludesInactive(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	webhookStore := store.NewWebhookStore(s)

	// Create webhook
	webhook, err := webhookStore.Create(ctx, "https://example.com/webhook-inactive", []string{"message.received"}, "secret")
	require.NoError(t, err)

	// Set active=false via raw SQL
	_, err = testDBPool.Exec(ctx, "UPDATE webhooks SET active = false WHERE id = $1", webhook.ID)
	require.NoError(t, err)

	// ListActiveForEvent should return 0 results
	webhooks, err := webhookStore.ListActiveForEvent(ctx, "message.received")
	require.NoError(t, err)
	assert.Len(t, webhooks, 0)
}

// TestWebhookCreateDeliveryAndMarkDelivered verifies delivery creation and marking as delivered
func TestWebhookCreateDeliveryAndMarkDelivered(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	webhookStore := store.NewWebhookStore(s)

	// Create webhook
	webhook, err := webhookStore.Create(ctx, "https://example.com/webhook-delivery", []string{"message.received"}, "secret")
	require.NoError(t, err)

	// Create delivery
	payload := map[string]string{"message": "test"}
	deliveryID, err := webhookStore.CreateDelivery(ctx, webhook.ID, "message.received", payload)
	require.NoError(t, err)
	assert.Greater(t, deliveryID, int64(0))

	// Mark as delivered
	err = webhookStore.MarkDelivered(ctx, deliveryID, 200)
	require.NoError(t, err)

	// ListPendingDeliveries should return empty (delivered ones are excluded)
	deliveries, err := webhookStore.ListPendingDeliveries(ctx)
	require.NoError(t, err)
	assert.Len(t, deliveries, 0)
}

// TestWebhookScheduleRetryAndListPending verifies scheduling a retry in the past is returned as pending
func TestWebhookScheduleRetryAndListPending(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	webhookStore := store.NewWebhookStore(s)

	// Create webhook
	webhook, err := webhookStore.Create(ctx, "https://example.com/webhook-retry", []string{"message.received"}, "secret")
	require.NoError(t, err)

	// Create delivery
	payload := map[string]string{"message": "test"}
	deliveryID, err := webhookStore.CreateDelivery(ctx, webhook.ID, "message.received", payload)
	require.NoError(t, err)

	// Schedule retry with next_retry in the past
	nextRetry := time.Now().Add(-1 * time.Minute)
	err = webhookStore.ScheduleRetry(ctx, deliveryID, nextRetry)
	require.NoError(t, err)

	// ListPendingDeliveries should return 1 result
	deliveries, err := webhookStore.ListPendingDeliveries(ctx)
	require.NoError(t, err)
	assert.Len(t, deliveries, 1)
	assert.Equal(t, deliveryID, deliveries[0].ID)
}

// TestWebhookScheduleRetryFuture verifies scheduling a retry in the future is not returned as pending
func TestWebhookScheduleRetryFuture(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	webhookStore := store.NewWebhookStore(s)

	// Create webhook
	webhook, err := webhookStore.Create(ctx, "https://example.com/webhook-future", []string{"message.received"}, "secret")
	require.NoError(t, err)

	// Create delivery
	payload := map[string]string{"message": "test"}
	deliveryID, err := webhookStore.CreateDelivery(ctx, webhook.ID, "message.received", payload)
	require.NoError(t, err)

	// Schedule retry with next_retry in the future
	nextRetry := time.Now().Add(1 * time.Hour)
	err = webhookStore.ScheduleRetry(ctx, deliveryID, nextRetry)
	require.NoError(t, err)

	// ListPendingDeliveries should return 0 (future retries not returned)
	deliveries, err := webhookStore.ListPendingDeliveries(ctx)
	require.NoError(t, err)
	assert.Len(t, deliveries, 0)
}

// ==================== Chat Delete ====================

// TestChatDeleteCascadesToMessages verifies deleting a chat removes associated messages
func TestChatDeleteCascadesToMessages(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)
	chatStore := store.NewChatStore(s)
	msgStore := store.NewMessageStore(s)

	// Create account
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Chat Delete Cascade Test",
		Identifier:   "chat-del-cascade-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Status:       string(model.StatusOperational),
		Capabilities: []string{},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Create chat
	chat := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: "chat-del-cascade-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	createdChat, err := chatStore.Create(ctx, chat)
	require.NoError(t, err)

	// Create 3 messages
	messageIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		msg := &model.Message{
			ChatID:      createdChat.ID,
			AccountID:   account.ID,
			Provider:    "MOCK",
			ProviderID:  fmt.Sprintf("chat-del-msg-%d-%d", i, time.Now().UnixNano()),
			Text:        fmt.Sprintf("Message %d", i),
			SenderID:    "sender-1",
			Timestamp:   time.Now(),
			Attachments: []model.Attachment{},
			Reactions:   []model.Reaction{},
			Metadata:    map[string]any{},
		}
		created, err := msgStore.Create(ctx, msg)
		require.NoError(t, err)
		messageIDs[i] = created.ID
	}

	// Delete chat
	err = chatStore.Delete(ctx, createdChat.ID)
	require.NoError(t, err)

	// Verify messages are gone
	for _, msgID := range messageIDs {
		retrieved, err := msgStore.GetByID(ctx, msgID)
		require.NoError(t, err)
		assert.Nil(t, retrieved)
	}
}

// TestChatDeleteVerification verifies GetByID returns nil after deletion
func TestChatDeleteVerification(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)
	chatStore := store.NewChatStore(s)

	// Create account
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Chat Delete Verification Test",
		Identifier:   "chat-del-verify-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Status:       string(model.StatusOperational),
		Capabilities: []string{},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Create chat
	chat := &model.Chat{
		AccountID:  account.ID,
		Provider:   "MOCK",
		ProviderID: "chat-del-verify-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Type:       string(model.ChatTypeOneToOne),
		Metadata:   map[string]any{},
	}
	createdChat, err := chatStore.Create(ctx, chat)
	require.NoError(t, err)

	// Verify chat exists
	retrieved, err := chatStore.GetByID(ctx, createdChat.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	// Delete chat
	err = chatStore.Delete(ctx, createdChat.ID)
	require.NoError(t, err)

	// Verify GetByID returns nil
	retrieved, err = chatStore.GetByID(ctx, createdChat.ID)
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

// ==================== Email Store Tests (Bonus) ====================

// TestEmailStoreCreateAndGet verifies email creation and retrieval
func TestEmailStoreCreateAndGet(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)
	emailStore := email.NewEmailStore(s)

	// Create account
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "IMAP",
		Name:         "Email Store Test",
		Identifier:   "email-store-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Status:       string(model.StatusOperational),
		Capabilities: []string{},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Create email
	testEmail := &model.Email{
		ID:        "email-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		AccountID: account.ID,
		Provider:  "IMAP",
		ProviderID: &model.EmailProviderID{
			MessageID: "test-message-id",
		},
		Subject:     "Test Email",
		Body:        "<html><body>Test body</body></html>",
		BodyPlain:   "Test body",
		FromAttendee: &model.EmailAttendee{Identifier: "from@example.com", IdentifierType: "EMAIL"},
		ToAttendees:  []model.EmailAttendee{{Identifier: "to@example.com", IdentifierType: "EMAIL"}},
		Folders:     []string{"INBOX"},
		Role:        "inbox",
		Read:        false,
		IsComplete:  true,
		Attachments: []model.EmailAttachment{},
		Headers:     []model.EmailHeader{},
		Tracking:    &model.EmailTracking{},
		Metadata:    map[string]any{},
	}

	err = emailStore.StoreEmail(ctx, testEmail)
	require.NoError(t, err)

	// Retrieve email
	retrieved, err := emailStore.GetEmail(ctx, testEmail.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, testEmail.ID, retrieved.ID)
	assert.Equal(t, testEmail.Subject, retrieved.Subject)
	assert.Equal(t, account.ID, retrieved.AccountID)
}
