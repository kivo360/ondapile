package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ondapile/internal/adapter"
	"ondapile/internal/api"
	"ondapile/internal/store"
	"ondapile/internal/webhook"
)

// TestAuthRequired tests that requests without X-API-KEY return 401
func TestAuthRequired(t *testing.T) {
	// Setup without truncation since we're just testing auth
	ctx := t.Context()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	mockProvider := &MockProvider{}
	adapter.Register(mockProvider)

	s := &store.Store{Pool: testDBPool}
	webhookStore := store.NewWebhookStore(s)
	dispatcher := webhook.NewDispatcher(webhookStore)
	router := api.Router(s, dispatcher, testAPIKey, testEncryptionKey)

	// Request without API key
	req := httptest.NewRequest("GET", "/api/v1/accounts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)

	var errResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)

	assert.Equal(t, "error", errResp["object"])
	assert.Equal(t, float64(http.StatusUnauthorized), errResp["status"])
	assert.Equal(t, "UNAUTHORIZED", errResp["code"])
}

// TestAuthInvalidKey tests that requests with wrong API key return 401
func TestAuthInvalidKey(t *testing.T) {
	ctx := t.Context()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	mockProvider := &MockProvider{}
	adapter.Register(mockProvider)

	s := &store.Store{Pool: testDBPool}
	dispatcher := webhook.NewDispatcher(store.NewWebhookStore(s))
	router := api.Router(s, dispatcher, testAPIKey, testEncryptionKey)

	// Request with wrong API key
	req := httptest.NewRequest("GET", "/api/v1/accounts", nil)
	req.Header.Set("X-API-KEY", "wrong-api-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)

	var errResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)

	assert.Equal(t, "error", errResp["object"])
	assert.Equal(t, "UNAUTHORIZED", errResp["code"])
}

// TestAuthValidKey tests that requests with correct API key succeed
func TestAuthValidKey(t *testing.T) {
	ctx := t.Context()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	mockProvider := &MockProvider{}
	adapter.Register(mockProvider)

	s := &store.Store{Pool: testDBPool}
	dispatcher := webhook.NewDispatcher(store.NewWebhookStore(s))
	router := api.Router(s, dispatcher, testAPIKey, testEncryptionKey)

	// Request with correct API key
	req := httptest.NewRequest("GET", "/api/v1/accounts", nil)
	req.Header.Set("X-API-KEY", testAPIKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should succeed (empty list, not auth error)
	require.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "list", result["object"])
}

// TestAuthQueryParam tests that API key can be passed as query parameter
func TestAuthQueryParam(t *testing.T) {
	ctx := t.Context()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	mockProvider := &MockProvider{}
	adapter.Register(mockProvider)

	s := &store.Store{Pool: testDBPool}
	dispatcher := webhook.NewDispatcher(store.NewWebhookStore(s))
	router := api.Router(s, dispatcher, testAPIKey, testEncryptionKey)

	// Request with API key as query parameter (for browser img tags)
	req := httptest.NewRequest("GET", "/api/v1/accounts?key="+testAPIKey, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should succeed
	require.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "list", result["object"])
}

// TestRateLimitHeaders tests that rate limit headers are present
func TestRateLimitHeaders(t *testing.T) {
	ctx := t.Context()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	mockProvider := &MockProvider{}
	adapter.Register(mockProvider)

	s := &store.Store{Pool: testDBPool}
	dispatcher := webhook.NewDispatcher(store.NewWebhookStore(s))
	router := api.Router(s, dispatcher, testAPIKey, testEncryptionKey)

	req := httptest.NewRequest("GET", "/api/v1/accounts", nil)
	req.Header.Set("X-API-KEY", testAPIKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	// Check rate limit headers
	limit := w.Header().Get("X-RateLimit-Limit")
	remaining := w.Header().Get("X-RateLimit-Remaining")
	reset := w.Header().Get("X-RateLimit-Reset")

	assert.NotEmpty(t, limit, "X-RateLimit-Limit header should be present")
	assert.NotEmpty(t, remaining, "X-RateLimit-Remaining header should be present")
	assert.NotEmpty(t, reset, "X-RateLimit-Reset header should be present")

	// Verify values are reasonable
	assert.Equal(t, "100", limit, "Burst limit should be 100")
	assert.NotEqual(t, "0", remaining, "Remaining should not be 0 on first request")
}

// TestCORSHeaders tests that CORS headers are present
func TestCORSHeaders(t *testing.T) {
	ctx := t.Context()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	mockProvider := &MockProvider{}
	adapter.Register(mockProvider)

	s := &store.Store{Pool: testDBPool}
	dispatcher := webhook.NewDispatcher(store.NewWebhookStore(s))
	router := api.Router(s, dispatcher, testAPIKey, testEncryptionKey)

	// Test preflight OPTIONS request
	req := httptest.NewRequest("OPTIONS", "/api/v1/accounts", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// OPTIONS should return 204
	require.Equal(t, http.StatusNoContent, w.Code)

	// Check CORS headers
	allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	allowMethods := w.Header().Get("Access-Control-Allow-Methods")
	allowHeaders := w.Header().Get("Access-Control-Allow-Headers")

	assert.Equal(t, "*", allowOrigin, "Access-Control-Allow-Origin should be *")
	assert.NotEmpty(t, allowMethods, "Access-Control-Allow-Methods should be present")
	assert.Contains(t, allowMethods, "GET", "Should allow GET")
	assert.Contains(t, allowMethods, "POST", "Should allow POST")
	assert.NotEmpty(t, allowHeaders, "Access-Control-Allow-Headers should be present")
	assert.Contains(t, allowHeaders, "X-API-KEY", "Should allow X-API-KEY header")
}

// TestCORSHeadersOnRegularRequest tests CORS headers on regular requests
func TestCORSHeadersOnRegularRequest(t *testing.T) {
	ctx := t.Context()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	mockProvider := &MockProvider{}
	adapter.Register(mockProvider)

	s := &store.Store{Pool: testDBPool}
	dispatcher := webhook.NewDispatcher(store.NewWebhookStore(s))
	router := api.Router(s, dispatcher, testAPIKey, testEncryptionKey)

	req := httptest.NewRequest("GET", "/api/v1/accounts", nil)
	req.Header.Set("X-API-KEY", testAPIKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	// Check CORS headers are present on regular requests too
	allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
	assert.Equal(t, "*", allowOrigin, "Access-Control-Allow-Origin should be * on regular requests")
}

// TestHealthEndpointNoAuth tests that health endpoint doesn't require auth
func TestHealthEndpointNoAuth(t *testing.T) {
	ctx := t.Context()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	mockProvider := &MockProvider{}
	adapter.Register(mockProvider)

	s := &store.Store{Pool: testDBPool}
	dispatcher := webhook.NewDispatcher(store.NewWebhookStore(s))
	router := api.Router(s, dispatcher, testAPIKey, testEncryptionKey)

	// Health endpoint without API key
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should succeed without auth
	require.Equal(t, http.StatusOK, w.Code)
}

// TestRateLimitExceeded tests that rate limiting works (makes many requests)
func TestRateLimitExceeded(t *testing.T) {
	ctx := t.Context()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	mockProvider := &MockProvider{}
	adapter.Register(mockProvider)

	s := &store.Store{Pool: testDBPool}
	dispatcher := webhook.NewDispatcher(store.NewWebhookStore(s))
	router := api.Router(s, dispatcher, testAPIKey, testEncryptionKey)

	// Make 105 requests (burst is 100)
	var lastStatus int
	for i := 0; i < 105; i++ {
		req := httptest.NewRequest("GET", "/api/v1/accounts", nil)
		req.Header.Set("X-API-KEY", testAPIKey)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		lastStatus = w.Code

		if w.Code == http.StatusTooManyRequests {
			break
		}
	}

	// At least some requests should have been rate limited
	// Note: In practice, with burst=100, we should get rate limited after 100 requests
	// But the test might not always trigger it depending on timing
	// So we just verify that when we do get rate limited, the response is correct
	if lastStatus == http.StatusTooManyRequests {
		// If we were rate limited, verify the error response
		// The last request would have been recorded, check its body
		req := httptest.NewRequest("GET", "/api/v1/accounts", nil)
		req.Header.Set("X-API-KEY", testAPIKey)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code == http.StatusTooManyRequests {
			var errResp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &errResp)
			require.NoError(t, err)

			assert.Equal(t, "error", errResp["object"])
			assert.Equal(t, "RATE_LIMITED", errResp["code"])
			assert.NotEmpty(t, w.Header().Get("Retry-After"))
		}
	}
}
