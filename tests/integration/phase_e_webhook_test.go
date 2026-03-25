package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ondapile/internal/email"
	"ondapile/internal/model"
	"ondapile/internal/store"
)

// TestWebhookCoverageEmailSent verifies that email.sent webhook fires on reply and forward.
func TestWebhookCoverageEmailSent(t *testing.T) {
	router, s := setupTest(t)

	// Create a webhook receiver to capture events
	var mu sync.Mutex
	var receivedEvents []string

	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		event := r.Header.Get("X-Ondapile-Event")
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer webhookServer.Close()

	// Register webhook for email.sent events
	whBody := fmt.Sprintf(`{"url":"%s","events":["email.sent"],"secret":"test-secret"}`, webhookServer.URL)
	resp := apiRequest(t, router, "POST", "/api/v1/webhooks", []byte(whBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	// Create account
	acctBody := `{"provider":"MOCK","identifier":"webhook-test@test.com","credentials":{}}`
	resp = apiRequest(t, router, "POST", "/api/v1/accounts", []byte(acctBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)
	var acct map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &acct)
	accountID := acct["id"].(string)

	// Store a test email for reply/forward
	emailStore := email.NewEmailStore(s)
	testEmail := &model.Email{
		ID:        "eml_wh_test_001",
		AccountID: accountID,
		Provider:  "MOCK",
		Subject:   "Webhook Test",
		Body:      "<p>Hello</p>",
		BodyPlain: "Hello",
		FromAttendee: &model.EmailAttendee{
			DisplayName:    "Sender",
			Identifier:     "sender@example.com",
			IdentifierType: "EMAIL_ADDRESS",
		},
		ToAttendees: []model.EmailAttendee{{
			DisplayName:    "Recipient",
			Identifier:     "recipient@example.com",
			IdentifierType: "EMAIL_ADDRESS",
		}},
		Date:       time.Now(),
		Role:       model.FolderInbox,
		Folders:    []string{model.FolderInbox},
		ProviderID: &model.EmailProviderID{MessageID: "msg_wh_001", ThreadID: "thread_wh_001"},
		Metadata:   map[string]any{},
	}
	err := emailStore.StoreEmail(context.Background(), testEmail)
	require.NoError(t, err)

	t.Run("ReplyTriggersEmailSentWebhook", func(t *testing.T) {
		body := fmt.Sprintf(`{"account_id":"%s","body_html":"<p>Reply</p>"}`, accountID)
		resp := apiRequest(t, router, "POST", "/api/v1/emails/eml_wh_test_001/reply", []byte(body), testAPIKey)
		requireStatus(t, resp, http.StatusOK)

		// Give webhook time to deliver
		time.Sleep(500 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		assert.Contains(t, receivedEvents, "email.sent", "Reply should trigger email.sent webhook")
	})

	t.Run("ForwardTriggersEmailSentWebhook", func(t *testing.T) {
		mu.Lock()
		receivedEvents = nil
		mu.Unlock()

		body := fmt.Sprintf(`{"account_id":"%s","to":[{"identifier":"fwd@test.com","identifier_type":"EMAIL_ADDRESS"}],"body_html":"<p>FYI</p>"}`, accountID)
		resp := apiRequest(t, router, "POST", "/api/v1/emails/eml_wh_test_001/forward", []byte(body), testAPIKey)
		requireStatus(t, resp, http.StatusOK)

		time.Sleep(500 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		assert.Contains(t, receivedEvents, "email.sent", "Forward should trigger email.sent webhook")
	})
}

// TestWebhookCoverageTrackingEvents verifies email.opened and email.clicked webhooks fire from tracking.
func TestWebhookCoverageTrackingEvents(t *testing.T) {
	ctx := context.Background()
	s := setupTestDB(t)
	err := truncateTables(ctx, s.Pool)
	require.NoError(t, err)

	// Create a webhook receiver
	var mu sync.Mutex
	var receivedEvents []string

	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		event := r.Header.Get("X-Ondapile-Event")
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer webhookServer.Close()

	// Register webhooks for tracking events
	webhookStore := store.NewWebhookStore(s)
	_, err = webhookStore.Create(ctx, webhookServer.URL, []string{"email.opened", "email.clicked"}, "test-secret")
	require.NoError(t, err)

	// Create account and email
	accountStore := store.NewAccountStore(s)
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Tracking Webhook Test",
		Identifier:   "tracking-wh@test.com",
		Status:       string(model.StatusOperational),
		Capabilities: []string{"email"},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	emailStore := email.NewEmailStore(s)
	testEmail := &model.Email{
		ID:        "eml_wh_track_001",
		AccountID: account.ID,
		Provider:  "MOCK",
		Subject:   "Tracking Webhook Test",
		Body:      "<p>Hello</p>",
		Metadata:  map[string]any{},
	}
	err = emailStore.StoreEmail(ctx, testEmail)
	require.NoError(t, err)

	router := setupTestRouter(t)

	t.Run("PixelTriggersEmailOpenedWebhook", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/t/eml_wh_track_001", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		time.Sleep(500 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		assert.Contains(t, receivedEvents, "email.opened", "Pixel should trigger email.opened webhook")
	})

	t.Run("LinkClickTriggersEmailClickedWebhook", func(t *testing.T) {
		mu.Lock()
		receivedEvents = nil
		mu.Unlock()

		req := httptest.NewRequest("GET", "/l/eml_wh_track_001?url=https://example.com", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusFound, w.Code)

		time.Sleep(500 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		assert.Contains(t, receivedEvents, "email.clicked", "Link click should trigger email.clicked webhook")
	})
}
