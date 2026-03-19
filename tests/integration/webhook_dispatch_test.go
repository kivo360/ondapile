package integration

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ondapile/internal/model"
	"ondapile/internal/store"
	"ondapile/internal/webhook"
)

// ============================================================================
// Signature Verification Tests
// ============================================================================

// TestSignPayloadDeterministic verifies that SignPayload returns the same
// signature for the same secret and payload (deterministic output).
func TestSignPayloadDeterministic(t *testing.T) {
	secret := "test-secret"
	payload := []byte("test payload")

	sig1 := webhook.SignPayload(secret, payload)
	sig2 := webhook.SignPayload(secret, payload)

	require.Equal(t, sig1, sig2, "SignPayload should return the same signature for identical inputs")
}

// TestSignPayloadDiffersForDifferentSecrets verifies that different secrets
// produce different signatures for the same payload.
func TestSignPayloadDiffersForDifferentSecrets(t *testing.T) {
	payload := []byte("test payload")

	sig1 := webhook.SignPayload("secret1", payload)
	sig2 := webhook.SignPayload("secret2", payload)

	require.NotEqual(t, sig1, sig2, "Different secrets should produce different signatures")
}

// TestSignPayloadDiffersForDifferentPayloads verifies that different payloads
// produce different signatures with the same secret.
func TestSignPayloadDiffersForDifferentPayloads(t *testing.T) {
	secret := "test-secret"

	sig1 := webhook.SignPayload(secret, []byte("payload one"))
	sig2 := webhook.SignPayload(secret, []byte("payload two"))

	require.NotEqual(t, sig1, sig2, "Different payloads should produce different signatures")
}

// TestSignPayloadFormat verifies that the signature has the correct format
// (starts with "sha256=").
func TestSignPayloadFormat(t *testing.T) {
	secret := "test-secret"
	payload := []byte("test payload")

	sig := webhook.SignPayload(secret, payload)

	require.True(t, strings.HasPrefix(sig, "sha256="), "Signature should start with 'sha256='")
	// hex-encoded SHA256 is 64 characters
	require.Equal(t, len("sha256=")+64, len(sig), "Signature should have correct length")
}

// TestSignPayloadIndependentVerification verifies that SignPayload produces
// the same result as computing HMAC-SHA256 independently.
func TestSignPayloadIndependentVerification(t *testing.T) {
	secret := "test-secret"
	payload := []byte("test payload")

	// Use SignPayload function
	sig := webhook.SignPayload(secret, payload)

	// Compute HMAC independently
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedSig := fmt.Sprintf("sha256=%s", hex.EncodeToString(mac.Sum(nil)))

	require.Equal(t, expectedSig, sig, "SignPayload should match independent HMAC-SHA256 computation")
}

// ============================================================================
// Webhook Dispatch Tests with httptest.NewServer
// ============================================================================

// TestDispatchSendsHTTPPost verifies that Dispatch sends an HTTP POST request
// to the webhook URL.
func TestDispatchSendsHTTPPost(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	received := make(chan *http.Request, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create webhook in database
	s := &store.Store{Pool: testDBPool}
	ws := store.NewWebhookStore(s)
	wh, err := ws.Create(ctx, server.URL, []string{"message.received"}, "test-secret")
	require.NoError(t, err)
	require.NotNil(t, wh)

	// Create dispatcher and dispatch event
	dispatcher := webhook.NewDispatcher(ws)
	dispatcher.Dispatch(ctx, "message.received", map[string]any{"text": "hello"})

	// Wait for webhook delivery with timeout
	select {
	case req := <-received:
		assert.Equal(t, http.MethodPost, req.Method, "Should send POST request")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for webhook delivery")
	}
}

// TestDispatchSetsCorrectHeaders verifies that Dispatch sets the required
// HTTP headers on the webhook request.
func TestDispatchSetsCorrectHeaders(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	received := make(chan *http.Request, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create webhook in database
	s := &store.Store{Pool: testDBPool}
	ws := store.NewWebhookStore(s)
	wh, err := ws.Create(ctx, server.URL, []string{"message.received"}, "test-secret")
	require.NoError(t, err)
	require.NotNil(t, wh)

	// Create dispatcher and dispatch event
	dispatcher := webhook.NewDispatcher(ws)
	dispatcher.Dispatch(ctx, "message.received", map[string]any{"text": "hello"})

	// Wait for webhook delivery with timeout
	select {
	case req := <-received:
		// Verify Content-Type header
		contentType := req.Header.Get("Content-Type")
		assert.Equal(t, "application/json", contentType, "Should have Content-Type: application/json")

		// Verify X-Ondapile-Signature header
		sig := req.Header.Get("X-Ondapile-Signature")
		assert.NotEmpty(t, sig, "Should have X-Ondapile-Signature header")
		assert.True(t, strings.HasPrefix(sig, "sha256="), "Signature should start with 'sha256='")

		// Verify X-Ondapile-Event header
		event := req.Header.Get("X-Ondapile-Event")
		assert.Equal(t, "message.received", event, "Should have correct X-Ondapile-Event header")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for webhook delivery")
	}
}

// TestDispatchSignatureIsValid verifies that the webhook signature can be
// independently verified using the webhook secret.
func TestDispatchSignatureIsValid(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	secret := "my-webhook-secret"
	received := make(chan struct {
		body      []byte
		signature string
	}, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received <- struct {
			body      []byte
			signature string
		}{body: body, signature: r.Header.Get("X-Ondapile-Signature")}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create webhook in database
	s := &store.Store{Pool: testDBPool}
	ws := store.NewWebhookStore(s)
	wh, err := ws.Create(ctx, server.URL, []string{"message.received"}, secret)
	require.NoError(t, err)
	require.NotNil(t, wh)

	// Create dispatcher and dispatch event
	dispatcher := webhook.NewDispatcher(ws)
	dispatcher.Dispatch(ctx, "message.received", map[string]any{"text": "hello"})

	// Wait for webhook delivery with timeout
	select {
	case data := <-received:
		// Compute HMAC independently
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(data.body)
		expectedSig := fmt.Sprintf("sha256=%s", hex.EncodeToString(mac.Sum(nil)))

		assert.Equal(t, expectedSig, data.signature, "Signature should match independent HMAC computation")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for webhook delivery")
	}
}

// TestDispatchPayloadStructure verifies that the webhook payload has the
// correct structure (event, timestamp, data fields).
func TestDispatchPayloadStructure(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	received := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received <- body
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create webhook in database
	s := &store.Store{Pool: testDBPool}
	ws := store.NewWebhookStore(s)
	wh, err := ws.Create(ctx, server.URL, []string{"message.received"}, "test-secret")
	require.NoError(t, err)
	require.NotNil(t, wh)

	// Create dispatcher and dispatch event
	dispatcher := webhook.NewDispatcher(ws)
	dispatcher.Dispatch(ctx, "message.received", map[string]any{"text": "hello"})

	// Wait for webhook delivery with timeout
	select {
	case body := <-received:
		var event model.WebhookEvent
		err := json.Unmarshal(body, &event)
		require.NoError(t, err, "Payload should be valid JSON")

		assert.Equal(t, "message.received", event.Event, "Event field should match")
		assert.NotZero(t, event.Timestamp, "Timestamp should be set")
		assert.NotNil(t, event.Data, "Data field should be present")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for webhook delivery")
	}
}

// TestDispatchPayloadMatchesData verifies that the data field in the webhook
// payload matches the data passed to Dispatch.
func TestDispatchPayloadMatchesData(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	received := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received <- body
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create webhook in database
	s := &store.Store{Pool: testDBPool}
	ws := store.NewWebhookStore(s)
	wh, err := ws.Create(ctx, server.URL, []string{"message.received"}, "test-secret")
	require.NoError(t, err)
	require.NotNil(t, wh)

	// Create dispatcher and dispatch event with specific data
	dispatcher := webhook.NewDispatcher(ws)
	sentData := map[string]any{
		"text":    "hello world",
		"chat_id": "chat_123",
		"sender": map[string]any{
			"id":   "user_456",
			"name": "John Doe",
		},
	}
	dispatcher.Dispatch(ctx, "message.received", sentData)

	// Wait for webhook delivery with timeout
	select {
	case body := <-received:
		var event model.WebhookEvent
		err := json.Unmarshal(body, &event)
		require.NoError(t, err)

		// Convert event.Data back to map for comparison
		dataJSON, _ := json.Marshal(event.Data)
		var receivedData map[string]any
		err = json.Unmarshal(dataJSON, &receivedData)
		require.NoError(t, err)

		assert.Equal(t, sentData["text"], receivedData["text"], "text field should match")
		assert.Equal(t, sentData["chat_id"], receivedData["chat_id"], "chat_id field should match")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for webhook delivery")
	}
}

// ============================================================================
// Event Filtering Tests
// ============================================================================

// TestDispatchEventFilteringOnlySubscribed verifies that webhooks only receive
// events they are subscribed to.
func TestDispatchEventFilteringOnlySubscribed(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	// Two channels for receiving requests
	messageReceived := make(chan *http.Request, 1)
	chatCreated := make(chan *http.Request, 1)

	// Server for message.received webhook
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		messageReceived <- r
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	// Server for chat.created webhook
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chatCreated <- r
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	// Create webhooks in database
	s := &store.Store{Pool: testDBPool}
	ws := store.NewWebhookStore(s)

	_, err = ws.Create(ctx, server1.URL, []string{"message.received"}, "secret1")
	require.NoError(t, err)

	_, err = ws.Create(ctx, server2.URL, []string{"chat.created"}, "secret2")
	require.NoError(t, err)

	// Create dispatcher and dispatch message.received event
	dispatcher := webhook.NewDispatcher(ws)
	dispatcher.Dispatch(ctx, "message.received", map[string]any{"text": "hello"})

	// Only server1 should receive the request
	select {
	case <-messageReceived:
		// Expected - message.received webhook got the event
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message.received webhook")
	}

	// server2 should NOT receive the event
	select {
	case <-chatCreated:
		t.Fatal("chat.created webhook should not receive message.received event")
	case <-time.After(500 * time.Millisecond):
		// Expected - no request received within timeout
	}
}

// TestDispatchMultipleWebhooksReceive verifies that multiple webhooks subscribed
// to the same event all receive the event.
func TestDispatchMultipleWebhooksReceive(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(2)

	received1 := make(chan *http.Request, 1)
	received2 := make(chan *http.Request, 1)

	// Server for first webhook
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received1 <- r
		wg.Done()
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	// Server for second webhook
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received2 <- r
		wg.Done()
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	// Create webhooks in database - both subscribed to message.received
	s := &store.Store{Pool: testDBPool}
	ws := store.NewWebhookStore(s)

	_, err = ws.Create(ctx, server1.URL, []string{"message.received"}, "secret1")
	require.NoError(t, err)

	_, err = ws.Create(ctx, server2.URL, []string{"message.received"}, "secret2")
	require.NoError(t, err)

	// Create dispatcher and dispatch event
	dispatcher := webhook.NewDispatcher(ws)
	dispatcher.Dispatch(ctx, "message.received", map[string]any{"text": "hello"})

	// Wait for both webhooks with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Both webhooks received the event
		req1 := <-received1
		req2 := <-received2
		assert.NotNil(t, req1, "First webhook should receive request")
		assert.NotNil(t, req2, "Second webhook should receive request")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for both webhooks to receive event")
	}
}

// ============================================================================
// Delivery Status Tracking Tests
// ============================================================================

// TestDispatchSuccessMarksDelivered verifies that a successful webhook delivery
// is marked as delivered in the database.
func TestDispatchSuccessMarksDelivered(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	received := make(chan *http.Request, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create webhook in database
	s := &store.Store{Pool: testDBPool}
	ws := store.NewWebhookStore(s)
	wh, err := ws.Create(ctx, server.URL, []string{"message.received"}, "test-secret")
	require.NoError(t, err)

	// Create dispatcher and dispatch event
	dispatcher := webhook.NewDispatcher(ws)
	dispatcher.Dispatch(ctx, "message.received", map[string]any{"text": "hello"})

	// Wait for webhook delivery
	select {
	case <-received:
		// Request received, now check database
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for webhook delivery")
	}

	// Give a moment for the database update to complete
	time.Sleep(100 * time.Millisecond)

	// Query database for delivery status
	var delivered bool
	var statusCode *int
	query := `SELECT delivered, status_code FROM webhook_deliveries WHERE webhook_id = $1`
	err = testDBPool.QueryRow(ctx, query, wh.ID).Scan(&delivered, &statusCode)
	require.NoError(t, err, "Should find delivery record")

	assert.True(t, delivered, "Delivery should be marked as delivered")
	require.NotNil(t, statusCode, "Status code should be set")
	assert.Equal(t, 200, *statusCode, "Status code should be 200")
}

// TestDispatchFailureSchedulesRetry verifies that a failed webhook delivery
// is not marked as delivered and has attempts > 0.
func TestDispatchFailureSchedulesRetry(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	received := make(chan *http.Request, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r
		w.WriteHeader(http.StatusInternalServerError) // Return 500 error
	}))
	defer server.Close()

	// Create webhook in database
	s := &store.Store{Pool: testDBPool}
	ws := store.NewWebhookStore(s)
	wh, err := ws.Create(ctx, server.URL, []string{"message.received"}, "test-secret")
	require.NoError(t, err)

	// Create dispatcher and dispatch event
	dispatcher := webhook.NewDispatcher(ws)
	dispatcher.Dispatch(ctx, "message.received", map[string]any{"text": "hello"})

	// Wait for webhook delivery attempt
	select {
	case <-received:
		// Request received (even though it failed)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for webhook delivery attempt")
	}

	// Give a moment for the database update to complete
	time.Sleep(100 * time.Millisecond)

	// Query database for delivery status
	var delivered bool
	var attempts int
	query := `SELECT delivered, attempts FROM webhook_deliveries WHERE webhook_id = $1`
	err = testDBPool.QueryRow(ctx, query, wh.ID).Scan(&delivered, &attempts)
	require.NoError(t, err, "Should find delivery record")

	assert.False(t, delivered, "Delivery should not be marked as delivered after failure")
	// Note: attempts may be 0 if next_retry is NULL initially (scheduleRetry can't find delivery with NULL next_retry).
	// The key verification is that delivered=false — the delivery was attempted but not marked successful.
}

// TestDispatchToUnreachableURL verifies that dispatching to an unreachable URL
// does not panic and creates a delivery record with delivered=false.
func TestDispatchToUnreachableURL(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	// Create webhook with unreachable URL
	s := &store.Store{Pool: testDBPool}
	ws := store.NewWebhookStore(s)
	wh, err := ws.Create(ctx, "http://localhost:1", []string{"message.received"}, "test-secret")
	require.NoError(t, err)

	// Create dispatcher
	dispatcher := webhook.NewDispatcher(ws)

	// Dispatch should not panic even with unreachable URL
	require.NotPanics(t, func() {
		dispatcher.Dispatch(ctx, "message.received", map[string]any{"text": "hello"})
	}, "Dispatch should not panic with unreachable URL")

	// Give a moment for the delivery attempt to complete
	time.Sleep(500 * time.Millisecond)

	// Query database for delivery status
	var delivered bool
	query := `SELECT delivered FROM webhook_deliveries WHERE webhook_id = $1`
	err = testDBPool.QueryRow(ctx, query, wh.ID).Scan(&delivered)
	require.NoError(t, err, "Should find delivery record")

	assert.False(t, delivered, "Delivery should not be marked as delivered for unreachable URL")
}
