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
	"ondapile/internal/email"
	"ondapile/internal/model"
	"ondapile/internal/store"
)

// makeTestEmail creates a test email with the given ID, accountID, and folder.
func makeTestEmail(id, accountID, folder string) *model.Email {
	return &model.Email{
		ID:         id,
		AccountID:  accountID,
		Provider:   "MOCK",
		ProviderID: &model.EmailProviderID{MessageID: "<" + id + "@test.com>", ThreadID: "thread_" + id},
		Subject:    "Test Subject " + id,
		Body:       "<html>Body for " + id + "</html>",
		BodyPlain:  "Body for " + id,
		FromAttendee: &model.EmailAttendee{
			DisplayName:    "Sender",
			Identifier:     "sender@test.com",
			IdentifierType: "EMAIL_ADDRESS",
		},
		ToAttendees: []model.EmailAttendee{
			{DisplayName: "Recipient", Identifier: "recipient@test.com", IdentifierType: "EMAIL_ADDRESS"},
		},
		CCAttendees:      []model.EmailAttendee{},
		BCCAttendees:     []model.EmailAttendee{},
		ReplyToAttendees: []model.EmailAttendee{},
		Date:             time.Now(),
		Folders:          []string{folder},
		Role:             folder,
		Read:             false,
		IsComplete:       true,
		Headers:          []model.EmailHeader{},
		Attachments:      []model.EmailAttachment{},
		Metadata:         map[string]any{},
	}
}

func TestEmailStoreRoundtrip(t *testing.T) {
	ctx := context.Background()
	s := setupTestDB(t)
	err := truncateTables(ctx, s.Pool)
	require.NoError(t, err)
	emailStore := email.NewEmailStore(s)
	accountStore := store.NewAccountStore(s)
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Test Account",
		Identifier:   "email-roundtrip@test.com",
		Status:       string(model.StatusOperational),
		Capabilities: []string{"email"},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Store an email
	testEmail := makeTestEmail("eml_test_001", account.ID, "INBOX")
	err = emailStore.StoreEmail(ctx, testEmail)
	require.NoError(t, err)

	// Retrieve the email
	retrieved, err := emailStore.GetEmail(ctx, testEmail.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	// Verify all fields match
	assert.Equal(t, testEmail.ID, retrieved.ID)
	assert.Equal(t, testEmail.AccountID, retrieved.AccountID)
	assert.Equal(t, testEmail.Provider, retrieved.Provider)
	assert.Equal(t, testEmail.Subject, retrieved.Subject)
	assert.Equal(t, testEmail.Body, retrieved.Body)
	assert.Equal(t, testEmail.BodyPlain, retrieved.BodyPlain)
	assert.Equal(t, testEmail.Read, retrieved.Read)
	assert.Equal(t, testEmail.Role, retrieved.Role)
	assert.Equal(t, "email", retrieved.Object)

	// Verify attendee fields
	require.NotNil(t, retrieved.FromAttendee)
	assert.Equal(t, testEmail.FromAttendee.DisplayName, retrieved.FromAttendee.DisplayName)
	assert.Equal(t, testEmail.FromAttendee.Identifier, retrieved.FromAttendee.Identifier)

	require.Len(t, retrieved.ToAttendees, 1)
	assert.Equal(t, testEmail.ToAttendees[0].DisplayName, retrieved.ToAttendees[0].DisplayName)
	assert.Equal(t, testEmail.ToAttendees[0].Identifier, retrieved.ToAttendees[0].Identifier)

	// Verify folders
	assert.Equal(t, testEmail.Folders, retrieved.Folders)

	// Verify ProviderID
	require.NotNil(t, retrieved.ProviderID)
	assert.Equal(t, testEmail.ProviderID.MessageID, retrieved.ProviderID.MessageID)
	assert.Equal(t, testEmail.ProviderID.ThreadID, retrieved.ProviderID.ThreadID)
}

func TestEmailListWithFolderFilter(t *testing.T) {
	ctx := context.Background()
	s := setupTestDB(t)
	err := truncateTables(ctx, s.Pool)
	require.NoError(t, err)
	emailStore := email.NewEmailStore(s)
	accountStore := store.NewAccountStore(s)
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Test Account",
		Identifier:   "email-folder-filter@test.com",
		Status:       string(model.StatusOperational),
		Capabilities: []string{"email"},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Store 3 emails: 2 in INBOX, 1 in SENT
	email1 := makeTestEmail("eml_inbox_1", account.ID, "INBOX")
	email2 := makeTestEmail("eml_inbox_2", account.ID, "INBOX")
	email3 := makeTestEmail("eml_sent_1", account.ID, "SENT")

	err = emailStore.StoreEmail(ctx, email1)
	require.NoError(t, err)
	err = emailStore.StoreEmail(ctx, email2)
	require.NoError(t, err)
	err = emailStore.StoreEmail(ctx, email3)
	require.NoError(t, err)

	// List INBOX emails
	inboxEmails, _, hasMore, err := emailStore.ListEmails(ctx, account.ID, "INBOX", "", 25)
	require.NoError(t, err)
	assert.False(t, hasMore)
	assert.Len(t, inboxEmails, 2)

	// List SENT emails
	sentEmails, _, hasMore, err := emailStore.ListEmails(ctx, account.ID, "SENT", "", 25)
	require.NoError(t, err)
	assert.False(t, hasMore)
	assert.Len(t, sentEmails, 1)
}

func TestEmailGetByProviderID(t *testing.T) {
	ctx := context.Background()
	s := setupTestDB(t)
	err := truncateTables(ctx, s.Pool)
	require.NoError(t, err)
	emailStore := email.NewEmailStore(s)
	accountStore := store.NewAccountStore(s)
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Test Account",
		Identifier:   "email-provider-id@test.com",
		Status:       string(model.StatusOperational),
		Capabilities: []string{"email"},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Store an email
	testEmail := makeTestEmail("eml_provider_test", account.ID, "INBOX")
	err = emailStore.StoreEmail(ctx, testEmail)
	require.NoError(t, err)

	// Retrieve by provider ID
	retrieved, err := emailStore.GetEmailByProviderID(ctx, account.ID, "<eml_provider_test@test.com>")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, testEmail.ID, retrieved.ID)
	assert.Equal(t, testEmail.Subject, retrieved.Subject)
}

func TestEmailGetByProviderIDNotFound(t *testing.T) {
	ctx := context.Background()
	s := setupTestDB(t)
	err := truncateTables(ctx, s.Pool)
	require.NoError(t, err)
	emailStore := email.NewEmailStore(s)
	accountStore := store.NewAccountStore(s)
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Test Account",
		Identifier:   "email-pid-notfound@test.com",
		Status:       string(model.StatusOperational),
		Capabilities: []string{"email"},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Try to retrieve non-existent email
	retrieved, err := emailStore.GetEmailByProviderID(ctx, account.ID, "<nonexistent@test.com>")
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestEmailUpdateReadStatus(t *testing.T) {
	ctx := context.Background()
	s := setupTestDB(t)
	err := truncateTables(ctx, s.Pool)
	require.NoError(t, err)
	emailStore := email.NewEmailStore(s)
	accountStore := store.NewAccountStore(s)
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Test Account",
		Identifier:   "email-read-status@test.com",
		Status:       string(model.StatusOperational),
		Capabilities: []string{"email"},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Store an email with Read=false
	testEmail := makeTestEmail("eml_read_test", account.ID, "INBOX")
	err = emailStore.StoreEmail(ctx, testEmail)
	require.NoError(t, err)

	// Verify initial state
	retrieved, err := emailStore.GetEmail(ctx, testEmail.ID)
	require.NoError(t, err)
	assert.False(t, retrieved.Read)
	assert.Nil(t, retrieved.ReadDate)

	// Update to read=true
	err = emailStore.UpdateEmailReadStatus(ctx, testEmail.ID, true)
	require.NoError(t, err)

	// Verify read status and date
	retrieved, err = emailStore.GetEmail(ctx, testEmail.ID)
	require.NoError(t, err)
	assert.True(t, retrieved.Read)
	assert.NotNil(t, retrieved.ReadDate)

	// Update to read=false
	err = emailStore.UpdateEmailReadStatus(ctx, testEmail.ID, false)
	require.NoError(t, err)

	// Verify read status is false
	retrieved, err = emailStore.GetEmail(ctx, testEmail.ID)
	require.NoError(t, err)
	assert.False(t, retrieved.Read)
}

func TestEmailUpdateFolder(t *testing.T) {
	ctx := context.Background()
	s := setupTestDB(t)
	err := truncateTables(ctx, s.Pool)
	require.NoError(t, err)
	emailStore := email.NewEmailStore(s)
	accountStore := store.NewAccountStore(s)
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Test Account",
		Identifier:   "email-folder@test.com",
		Status:       string(model.StatusOperational),
		Capabilities: []string{"email"},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Store an email in INBOX
	testEmail := makeTestEmail("eml_folder_test", account.ID, "INBOX")
	err = emailStore.StoreEmail(ctx, testEmail)
	require.NoError(t, err)

	// Update folder to ARCHIVE
	err = emailStore.UpdateEmailFolder(ctx, testEmail.ID, []string{"ARCHIVE"})
	require.NoError(t, err)

	// Verify folder changed
	retrieved, err := emailStore.GetEmail(ctx, testEmail.ID)
	require.NoError(t, err)
	assert.Contains(t, retrieved.Folders, "ARCHIVE")
}

func TestEmailDelete(t *testing.T) {
	ctx := context.Background()
	s := setupTestDB(t)
	err := truncateTables(ctx, s.Pool)
	require.NoError(t, err)
	emailStore := email.NewEmailStore(s)
	accountStore := store.NewAccountStore(s)
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Test Account",
		Identifier:   "email-delete@test.com",
		Status:       string(model.StatusOperational),
		Capabilities: []string{"email"},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Store an email
	testEmail := makeTestEmail("eml_delete_test", account.ID, "INBOX")
	err = emailStore.StoreEmail(ctx, testEmail)
	require.NoError(t, err)

	// Verify email exists
	retrieved, err := emailStore.GetEmail(ctx, testEmail.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved)

	// Delete the email
	err = emailStore.DeleteEmail(ctx, testEmail.ID)
	require.NoError(t, err)

	// Verify email is deleted
	retrieved, err = emailStore.GetEmail(ctx, testEmail.ID)
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestEmailGetUnreadCount(t *testing.T) {
	ctx := context.Background()
	s := setupTestDB(t)
	err := truncateTables(ctx, s.Pool)
	require.NoError(t, err)
	emailStore := email.NewEmailStore(s)
	accountStore := store.NewAccountStore(s)
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Test Account",
		Identifier:   "email-unread@test.com",
		Status:       string(model.StatusOperational),
		Capabilities: []string{"email"},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Store 3 emails in INBOX: 2 unread, 1 read
	email1 := makeTestEmail("eml_unread_1", account.ID, "INBOX")
	email2 := makeTestEmail("eml_unread_2", account.ID, "INBOX")
	email3 := makeTestEmail("eml_read_1", account.ID, "INBOX")
	email3.Read = true

	err = emailStore.StoreEmail(ctx, email1)
	require.NoError(t, err)
	err = emailStore.StoreEmail(ctx, email2)
	require.NoError(t, err)
	err = emailStore.StoreEmail(ctx, email3)
	require.NoError(t, err)

	// Get unread count
	count, err := emailStore.GetUnreadCount(ctx, account.ID, "INBOX")
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

// TestEmailGetFolderCounts tests getting counts for all folders.
func TestEmailGetFolderCounts(t *testing.T) {
	ctx := context.Background()
	s := setupTestDB(t)
	err := truncateTables(ctx, s.Pool)
	require.NoError(t, err)
	emailStore := email.NewEmailStore(s)
	accountStore := store.NewAccountStore(s)

	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Test Account",
		Identifier:   "email-folder-counts@test.com",
		Status:       string(model.StatusOperational),
		Capabilities: []string{"email"},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Store emails: 3 in INBOX (1 read, 2 unread), 2 in SENT (both read)
	inbox1 := makeTestEmail("eml_inbox_read", account.ID, "INBOX")
	inbox1.Read = true
	inbox2 := makeTestEmail("eml_inbox_unread_1", account.ID, "INBOX")
	inbox3 := makeTestEmail("eml_inbox_unread_2", account.ID, "INBOX")
	sent1 := makeTestEmail("eml_sent_1", account.ID, "SENT")
	sent1.Read = true
	sent2 := makeTestEmail("eml_sent_2", account.ID, "SENT")
	sent2.Read = true

	for _, email := range []*model.Email{inbox1, inbox2, inbox3, sent1, sent2} {
		err := emailStore.StoreEmail(ctx, email)
		require.NoError(t, err)
	}

	// Get folder counts
	counts, err := emailStore.GetFolderCounts(ctx, account.ID)
	require.NoError(t, err)

	// Verify INBOX: total=3, unread=2
	inboxCount := counts["INBOX"]
	require.NotNil(t, inboxCount)
	assert.Equal(t, "INBOX", inboxCount.Role)
	assert.Equal(t, 3, inboxCount.Total)
	assert.Equal(t, 2, inboxCount.Unread)

	// Verify SENT: total=2, unread=0
	sentCount := counts["SENT"]
	require.NotNil(t, sentCount)
	assert.Equal(t, "SENT", sentCount.Role)
	assert.Equal(t, 2, sentCount.Total)
	assert.Equal(t, 0, sentCount.Unread)
}

// TestEmailAPIListWithFilters tests the email list API endpoint with filters.
func TestEmailAPIListWithFilters(t *testing.T) {
	router, s := setupTest(t)
	ctx := context.Background()

	// Create an account via API
	body := `{
		"provider": "MOCK",
		"identifier": "email-api-test",
		"name": "Email API Test Account"
	}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Register mock provider with ListEmailsFunc returning test emails
	mockProvider := &MockProvider{
		ListEmailsFunc: func(ctx context.Context, accountID string, opts adapter.ListEmailOpts) (*model.PaginatedList[model.Email], error) {
			return &model.PaginatedList[model.Email]{
				Object: "list",
				Items: []model.Email{
					{
						Object:    "email",
						ID:        "eml_api_1",
						AccountID: accountID,
						Provider:  "MOCK",
						Subject:   "Test Email 1",
						Folders:   []string{"INBOX"},
						Date:      time.Now(),
					},
					{
						Object:    "email",
						ID:        "eml_api_2",
						AccountID: accountID,
						Provider:  "MOCK",
						Subject:   "Test Email 2",
						Folders:   []string{"INBOX"},
						Date:      time.Now(),
					},
				},
				Cursor:  "",
				HasMore: false,
			}, nil
		},
	}
	adapter.Register(mockProvider)

	// Store the emails in the database so they can be retrieved
	emailStore := email.NewEmailStore(s)
	testEmail1 := makeTestEmail("eml_api_1", account.ID, "INBOX")
	testEmail1.Subject = "Test Email 1"
	testEmail2 := makeTestEmail("eml_api_2", account.ID, "INBOX")
	testEmail2.Subject = "Test Email 2"

	err = emailStore.StoreEmail(ctx, testEmail1)
	require.NoError(t, err)
	err = emailStore.StoreEmail(ctx, testEmail2)
	require.NoError(t, err)

	// GET /api/v1/emails?account_id=ACCOUNT_ID&folder=INBOX
	path := fmt.Sprintf("/api/v1/emails?account_id=%s&folder=INBOX", account.ID)
	resp = apiRequest(t, router, "GET", path, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.PaginatedList[model.Email]
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "list", result.Object)
	assert.Len(t, result.Items, 2)
	assert.False(t, result.HasMore)

	// Verify the emails are the expected ones
	subjects := make(map[string]bool)
	for _, e := range result.Items {
		subjects[e.Subject] = true
	}
	assert.True(t, subjects["Test Email 1"])
	assert.True(t, subjects["Test Email 2"])
}
