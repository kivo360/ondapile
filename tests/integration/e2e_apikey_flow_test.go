package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"ondapile/internal/api"
	"ondapile/internal/store"
)

// TestE2E_ApiKeyFullFlow simulates the full stack:
//
//	Better Auth creates user + org + apikey in PostgreSQL
//	→ Go middleware reads apikey table, verifies key, extracts org_id
//	→ Handler filters data by org_id
//	→ User only sees their org's data
//
// This is the single test a looping agent runs to know the whole system works.
func TestE2E_ApiKeyFullFlow(t *testing.T) {
	ctx := context.Background()
	truncateTables(ctx, testDBPool)

	// ─── STEP 1: Simulate Better Auth creating a user + org ───
	// In production, Better Auth writes these rows. We insert directly.
	orgAID := "org_e2e_alpha"
	orgBID := "org_e2e_beta"

	// ─── STEP 2: Simulate Better Auth creating API keys ───
	rawKeyA := "sk_live_e2e_alpha_key_12345"
	rawKeyB := "sk_live_e2e_beta_key_67890"
	hashA := sha256.Sum256([]byte(rawKeyA))
	hashB := sha256.Sum256([]byte(rawKeyB))

	now := time.Now()
	// Insert API key for org A
	_, err := testDBPool.Exec(ctx,
		`INSERT INTO apikey (id, "configId", "referenceId", key, prefix, enabled, "createdAt", "updatedAt")
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		"key_e2e_a", "default", orgAID, hex.EncodeToString(hashA[:]), "sk_live_", true, now, now)
	if err != nil {
		t.Fatalf("insert apikey A: %v", err)
	}

	// Insert API key for org B
	_, err = testDBPool.Exec(ctx,
		`INSERT INTO apikey (id, "configId", "referenceId", key, prefix, enabled, "createdAt", "updatedAt")
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		"key_e2e_b", "default", orgBID, hex.EncodeToString(hashB[:]), "sk_live_", true, now, now)
	if err != nil {
		t.Fatalf("insert apikey B: %v", err)
	}

	// ─── STEP 3: Create test data for each org ───
	_, err = testDBPool.Exec(ctx,
		`INSERT INTO accounts (id, provider, name, identifier, status, organization_id)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		"acc_e2e_a1", "google", "Alpha Account", "alpha@test.com", "OPERATIONAL", orgAID)
	if err != nil {
		t.Fatalf("insert account A: %v", err)
	}

	_, err = testDBPool.Exec(ctx,
		`INSERT INTO accounts (id, provider, name, identifier, status, organization_id)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		"acc_e2e_b1", "microsoft", "Beta Account", "beta@test.com", "OPERATIONAL", orgBID)
	if err != nil {
		t.Fatalf("insert account B: %v", err)
	}

	_, err = testDBPool.Exec(ctx,
		`INSERT INTO webhooks (id, url, events, secret, active, organization_id)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		"whk_e2e_a1", "https://alpha.com/hook", "[]", "secret_a", true, orgAID)
	if err != nil {
		t.Fatalf("insert webhook A: %v", err)
	}

	_, err = testDBPool.Exec(ctx,
		`INSERT INTO webhooks (id, url, events, secret, active, organization_id)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		"whk_e2e_b1", "https://beta.com/hook", "[]", "secret_b", true, orgBID)
	if err != nil {
		t.Fatalf("insert webhook B: %v", err)
	}

	// ─── STEP 4: Set up Go backend with ApiKeyMiddleware ───
	s := setupTestDB(t)
	apiKeyStore := store.NewApiKeyStore(s)
	accountH := api.NewAccountHandler(s, testEncryptionKey)

	r := gin.New()
	v1 := r.Group("/api/v1", api.ApiKeyMiddleware(apiKeyStore))
	v1.GET("/accounts", accountH.List)

	// ─── STEP 5: Org A's key only sees Org A's data ───
	t.Run("org_A_sees_only_own_accounts", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/accounts", nil)
		req.Header.Set("Authorization", "Bearer "+rawKeyA)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["items"].([]interface{})
		if len(data) != 1 {
			t.Fatalf("org A should see 1 account, got %d", len(data))
		}
		acc := data[0].(map[string]interface{})
		if acc["identifier"] != "alpha@test.com" {
			t.Errorf("expected alpha@test.com, got %v", acc["identifier"])
		}
	})

	// ─── STEP 6: Org B's key only sees Org B's data ───
	t.Run("org_B_sees_only_own_accounts", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/accounts", nil)
		req.Header.Set("Authorization", "Bearer "+rawKeyB)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		data := resp["items"].([]interface{})
		if len(data) != 1 {
			t.Fatalf("org B should see 1 account, got %d", len(data))
		}
		acc := data[0].(map[string]interface{})
		if acc["identifier"] != "beta@test.com" {
			t.Errorf("expected beta@test.com, got %v", acc["identifier"])
		}
	})

	// ─── STEP 7: No key → 401 ───
	t.Run("no_key_returns_401", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/accounts", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 401 {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	// ─── STEP 8: Disable Org A's key → 401 ───
	t.Run("disabled_key_returns_401", func(t *testing.T) {
		testDBPool.Exec(ctx, `UPDATE apikey SET enabled = false WHERE id = $1`, "key_e2e_a")

		req, _ := http.NewRequest("GET", "/api/v1/accounts", nil)
		req.Header.Set("Authorization", "Bearer "+rawKeyA)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 401 {
			t.Fatalf("expected 401 for disabled key, got %d", w.Code)
		}

		// Re-enable for subsequent tests
		testDBPool.Exec(ctx, `UPDATE apikey SET enabled = true WHERE id = $1`, "key_e2e_a")
	})

	// ─── STEP 9: Verify lastRequest was updated ───
	t.Run("last_request_updated", func(t *testing.T) {
		// Make a request to update lastRequest
		req, _ := http.NewRequest("GET", "/api/v1/accounts", nil)
		req.Header.Set("Authorization", "Bearer "+rawKeyA)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		// Wait for async update
		time.Sleep(200 * time.Millisecond)

		var lastRequest *time.Time
		err := testDBPool.QueryRow(ctx,
			`SELECT "lastRequest" FROM apikey WHERE id = $1`, "key_e2e_a").Scan(&lastRequest)
		if err != nil {
			t.Fatalf("query lastRequest: %v", err)
		}
		if lastRequest == nil {
			t.Fatal("lastRequest should be set after a successful request")
		}
		if time.Since(*lastRequest) > 5*time.Second {
			t.Errorf("lastRequest should be recent, got %v", lastRequest)
		}
	})

	// ─── STEP 10: Expired key → 401 ───
	t.Run("expired_key_returns_401", func(t *testing.T) {
		past := time.Now().Add(-1 * time.Hour)
		testDBPool.Exec(ctx, `UPDATE apikey SET "expiresAt" = $1 WHERE id = $2`, past, "key_e2e_a")

		req, _ := http.NewRequest("GET", "/api/v1/accounts", nil)
		req.Header.Set("Authorization", "Bearer "+rawKeyA)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != 401 {
			t.Fatalf("expected 401 for expired key, got %d", w.Code)
		}
	})
}
