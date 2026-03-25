package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"


	"ondapile/internal/adapter"
	"ondapile/internal/api"
	"ondapile/internal/model"
	"ondapile/internal/store"
	"ondapile/internal/webhook"
)


// setupOrgScopedTest creates a test router with ApiKeyMiddleware for org-scoped testing
func setupOrgScopedTest(t *testing.T) (*gin.Engine, *store.Store) {
	t.Helper()

	ctx := context.Background()

	// Truncate tables before test
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err, "Failed to truncate tables")

	// Setup apikey table
	setupApiKeyTable(ctx, t, testDBPool)

	// Register mock provider
	mockProvider := &MockProvider{}
	adapter.Register(mockProvider)

	// Create store
	s := setupTestDB(t)

	// Create a router with ApiKeyMiddleware (not AuthMiddleware)
	r := gin.New()
	apiKeyStore := store.NewApiKeyStore(s)

	// Account handler
	accountH := api.NewAccountHandler(s, testEncryptionKey)
	webhookH := api.NewWebhookHandler(s)

	// Routes with ApiKeyMiddleware
	r.GET("/api/v1/accounts", api.ApiKeyMiddleware(apiKeyStore), accountH.List)
	r.POST("/api/v1/accounts", api.ApiKeyMiddleware(apiKeyStore), accountH.Create)
	r.GET("/api/v1/webhooks", api.ApiKeyMiddleware(apiKeyStore), webhookH.List)
	r.POST("/api/v1/webhooks", api.ApiKeyMiddleware(apiKeyStore), webhookH.Create)

	// Cleanup after test
	t.Cleanup(func() {
		err := truncateTables(ctx, testDBPool)
		require.NoError(t, err, "Failed to truncate tables in cleanup")
	})

	return r, s
}

// insertTestAPIKey inserts an API key for testing with the given org ID
func insertTestAPIKey(ctx context.Context, t *testing.T, pool *pgxpool.Pool, rawKey, orgID string) {
	t.Helper()

	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	_, err := pool.Exec(ctx, `
		INSERT INTO apikey (id, "configId", name, "referenceId", key, enabled, "createdAt", "updatedAt")
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`, "key_"+orgID, "cfg_"+orgID, "Test Key "+orgID, orgID, keyHash, true)
	require.NoError(t, err)
}

// insertTestAccount inserts an account directly into the database with the given org ID
func insertTestAccount(ctx context.Context, t *testing.T, pool *pgxpool.Pool, provider, identifier, orgID string) string {
	t.Helper()

	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO accounts (provider, name, identifier, status, capabilities, organization_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id
	`, provider, identifier, identifier, "OPERATIONAL", `["messaging"]`, orgID).Scan(&id)
	require.NoError(t, err)
	return id
}

// insertTestWebhook inserts a webhook directly into the database with the given org ID
func insertTestWebhook(ctx context.Context, t *testing.T, pool *pgxpool.Pool, url, orgID string) string {
	t.Helper()

	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO webhooks (url, events, secret, active, organization_id, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id
	`, url, `["message.received"]`, "whsec_test", true, orgID).Scan(&id)
	require.NoError(t, err)
	return id
}

// apiRequestWithKey makes an HTTP request with an API key in the Authorization header
func apiRequestWithKey(t *testing.T, method, path string, body []byte, apiKey string) *http.Request {
	t.Helper()

	var bodyReader *bytes.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	req := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	return req
}

func TestAccountsListFilteredByOrg(t *testing.T) {
	ctx := context.Background()
	router, s := setupOrgScopedTest(t)

	// Setup: Insert API keys for org_A and org_B
	rawKeyA := "sk_live_org_a_key"
	rawKeyB := "sk_live_org_b_key"
	insertTestAPIKey(ctx, t, s.Pool, rawKeyA, "org_A")
	insertTestAPIKey(ctx, t, s.Pool, rawKeyB, "org_B")

	// Setup: Insert accounts for each org
	accAID := insertTestAccount(ctx, t, s.Pool, "MOCK", "acc_a_phone", "org_A")
	accBID := insertTestAccount(ctx, t, s.Pool, "MOCK", "acc_b_phone", "org_B")
	_ = accAID
	_ = accBID

	// Act: GET /api/v1/accounts with org_A's bearer token
	req := apiRequestWithKey(t, "GET", "/api/v1/accounts", nil, rawKeyA)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert: Response should be 200 OK
	require.Equal(t, http.StatusOK, w.Code, "Expected 200 OK, got %d: %s", w.Code, w.Body.String())

	// Parse response
	var resp model.PaginatedList[map[string]interface{}]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	// Assert: Only org_A's account should be returned
	require.Equal(t, 1, len(resp.Items), "Expected 1 account for org_A, got %d", len(resp.Items))

	// Verify account A is in the response by querying DB to check
	var count int
	err = s.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM accounts WHERE organization_id = $1`, "org_A").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestWebhooksListFilteredByOrg(t *testing.T) {
	ctx := context.Background()
	router, s := setupOrgScopedTest(t)

	// Setup: Insert API keys for org_A and org_B
	rawKeyA := "sk_live_org_a_webhook"
	rawKeyB := "sk_live_org_b_webhook"
	insertTestAPIKey(ctx, t, s.Pool, rawKeyA, "org_A")
	insertTestAPIKey(ctx, t, s.Pool, rawKeyB, "org_B")

	// Setup: Insert webhooks for each org
	webhookAID := insertTestWebhook(ctx, t, s.Pool, "https://org-a.example.com/webhook", "org_A")
	webhookBID := insertTestWebhook(ctx, t, s.Pool, "https://org-b.example.com/webhook", "org_B")
	_ = webhookAID
	_ = webhookBID

	// Act: GET /api/v1/webhooks with org_A's bearer token
	req := apiRequestWithKey(t, "GET", "/api/v1/webhooks", nil, rawKeyA)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert: Response should be 200 OK
	require.Equal(t, http.StatusOK, w.Code, "Expected 200 OK, got %d: %s", w.Code, w.Body.String())

	// Parse response
	var resp model.PaginatedList[map[string]interface{}]
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	// Assert: Only org_A's webhook should be returned
	require.Equal(t, 1, len(resp.Items), "Expected 1 webhook for org_A, got %d", len(resp.Items))

	// Verify org_B's webhook still exists (wasn't deleted)
	var count int
	err = s.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM webhooks WHERE organization_id = $1`, "org_B").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count, "org_B's webhook should still exist")
}

func TestAccountCreateAssignsOrgId(t *testing.T) {
	ctx := context.Background()
	router, s := setupOrgScopedTest(t)

	// Setup: Insert API key for org_A
	rawKeyA := "sk_live_org_a_create"
	insertTestAPIKey(ctx, t, s.Pool, rawKeyA, "org_A")

	// Act: POST /api/v1/accounts with org_A's bearer token
	payload := []byte(`{
		"provider": "MOCK",
		"identifier": "test_phone_123",
		"name": "Test Account"
	}`)
	req := apiRequestWithKey(t, "POST", "/api/v1/accounts", payload, rawKeyA)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert: Response should be 201 Created (account creation returns 201)
	require.Equal(t, http.StatusCreated, w.Code, "Expected 201 Created, got %d: %s", w.Code, w.Body.String())


	// Parse response to get account ID
	var resp model.Account
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.NotEmpty(t, resp.ID)

	// Verify by querying DB directly - account should have organization_id = org_A
	var orgID *string
	err = s.Pool.QueryRow(ctx, `SELECT organization_id FROM accounts WHERE id = $1`, resp.ID).Scan(&orgID)
	require.NoError(t, err)
	require.NotNil(t, orgID)
	require.Equal(t, "org_A", *orgID, "Created account should have organization_id = org_A")
}

func TestWebhookCreateAssignsOrgId(t *testing.T) {
	ctx := context.Background()
	router, s := setupOrgScopedTest(t)

	// Setup: Insert API key for org_A
	rawKeyA := "sk_live_org_a_wh_create"
	insertTestAPIKey(ctx, t, s.Pool, rawKeyA, "org_A")

	// Act: POST /api/v1/webhooks with org_A's bearer token
	payload := []byte(`{
		"url": "https://org-a.example.com/webhook",
		"events": ["message.received"],
		"secret": "test-secret"
	}`)
	req := apiRequestWithKey(t, "POST", "/api/v1/webhooks", payload, rawKeyA)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert: Response should be 201 Created
	require.Equal(t, http.StatusCreated, w.Code, "Expected 201 Created, got %d: %s", w.Code, w.Body.String())

	// Parse response to get webhook ID
	var resp model.Webhook
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.NotEmpty(t, resp.ID)

	// Verify by querying DB directly - webhook should have organization_id = org_A
	var orgID *string
	err = s.Pool.QueryRow(ctx, `SELECT organization_id FROM webhooks WHERE id = $1`, resp.ID).Scan(&orgID)
	require.NoError(t, err)
	require.NotNil(t, orgID)
	require.Equal(t, "org_A", *orgID, "Created webhook should have organization_id = org_A")
}

// TestAccountsListWithLegacyAuth tests backward compatibility - accounts list works without org_id
func TestAccountsListWithLegacyAuth(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	// Register mock provider
	mockProvider := &MockProvider{}
	adapter.Register(mockProvider)

	s := setupTestDB(t)
	dispatcher := webhook.NewDispatcher(store.NewWebhookStore(s))

	// Use the standard router with AuthMiddleware (not ApiKeyMiddleware)
	router := api.Router(s, dispatcher, testAPIKey, testEncryptionKey, "")

	// Create an account using the legacy auth
	body := `{"provider": "MOCK", "identifier": "legacy-test", "name": "Legacy Test Account"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	// List accounts - should work and return all accounts (no org filtering)
	resp = apiRequest(t, router, "GET", "/api/v1/accounts", nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.PaginatedList[map[string]interface{}]
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
}

// TestWebhooksListWithLegacyAuth tests backward compatibility - webhooks list works without org_id
func TestWebhooksListWithLegacyAuth(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	// Register mock provider
	mockProvider := &MockProvider{}
	adapter.Register(mockProvider)

	s := setupTestDB(t)
	dispatcher := webhook.NewDispatcher(store.NewWebhookStore(s))

	// Use the standard router with AuthMiddleware (not ApiKeyMiddleware)
	router := api.Router(s, dispatcher, testAPIKey, testEncryptionKey, "")

	// Create a webhook using the legacy auth
	body := `{"url": "https://example.com/webhook", "events": ["message.received"]}`
	resp := apiRequest(t, router, "POST", "/api/v1/webhooks", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	// List webhooks - should work and return all webhooks (no org filtering)
	resp = apiRequest(t, router, "GET", "/api/v1/webhooks", nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.PaginatedList[map[string]interface{}]
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
}

// TestAccountsListDifferentOrgsIsolation tests complete isolation between organizations
func TestAccountsListDifferentOrgsIsolation(t *testing.T) {
	ctx := context.Background()
	router, s := setupOrgScopedTest(t)

	// Setup: Insert accounts and API keys for three different orgs
	orgs := []string{"org_A", "org_B", "org_C"}
	for _, org := range orgs {
		insertTestAccount(ctx, t, s.Pool, "MOCK", org+"_phone", org)
		insertTestAPIKey(ctx, t, s.Pool, "sk_live_"+org, org)
	}

	// Test each org can only see their own accounts
	for _, org := range orgs {
		req := apiRequestWithKey(t, "GET", "/api/v1/accounts", nil, "sk_live_"+org)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code, "Expected 200 for %s, got %d: %s", org, w.Code, w.Body.String())

		var resp model.PaginatedList[map[string]interface{}]
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		require.Len(t, resp.Items, 1, "expected 1 account for %s, got %d", org, len(resp.Items))
		require.Len(t, resp.Items, 1, "expected 1 account for %s, got %d", org, len(resp.Items))
	}
}

// TestWebhooksListDifferentOrgsIsolation tests complete isolation between organizations for webhooks
func TestWebhooksListDifferentOrgsIsolation(t *testing.T) {
	ctx := context.Background()
	router, s := setupOrgScopedTest(t)

	// Setup: Insert webhooks and API keys for three different orgs
	orgs := []string{"org_A", "org_B", "org_C"}
	for _, org := range orgs {
		insertTestWebhook(ctx, t, s.Pool, "https://"+org+".com/webhook", org)
		insertTestAPIKey(ctx, t, s.Pool, "sk_live_whk_"+org, org)
	}

	// Test each org can only see their own webhooks
	for _, org := range orgs {
		req := apiRequestWithKey(t, "GET", "/api/v1/webhooks", nil, "sk_live_whk_"+org)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code, "Expected 200 for %s, got %d: %s", org, w.Code, w.Body.String())

		var resp model.PaginatedList[map[string]interface{}]
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		require.Len(t, resp.Items, 1, "expected 1 webhook for %s, got %d", org, len(resp.Items))
		require.Len(t, resp.Items, 1, "expected 1 webhook for %s, got %d", org, len(resp.Items))
	}
}
