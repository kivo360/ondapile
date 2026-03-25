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

	"ondapile/internal/api"
)

func TestDualAuth_StaticKeyStillWorks(t *testing.T) {
	truncateTables(context.Background(), testDBPool)
	s := setupTestDB(t)
	router := api.Router(s, nil, testAPIKey, testEncryptionKey, "")

	req, _ := http.NewRequest("GET", "/api/v1/accounts", nil)
	req.Header.Set("X-API-KEY", testAPIKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("static key should still work, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDualAuth_BetterAuthKeyWorks(t *testing.T) {
	ctx := context.Background()
	truncateTables(ctx, testDBPool)
	s := setupTestDB(t)

	rawKey := "sk_live_dual_auth_test"
	hash := sha256.Sum256([]byte(rawKey))
	now := time.Now()
	testDBPool.Exec(ctx,
		`INSERT INTO apikey (id, "configId", "referenceId", key, prefix, enabled, "createdAt", "updatedAt")
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		"key_dual", "default", "org_dual", hex.EncodeToString(hash[:]), "sk_live_", true, now, now)

	testDBPool.Exec(ctx,
		`INSERT INTO accounts (id, provider, name, identifier, status, organization_id)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		"acc_dual", "google", "Dual Test", "dual@test.com", "OPERATIONAL", "org_dual")

	router := api.Router(s, nil, testAPIKey, testEncryptionKey, "")

	req, _ := http.NewRequest("GET", "/api/v1/accounts", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Better Auth key should work, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	items := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("expected 1 org-scoped account, got %d", len(items))
	}
}

func TestDualAuth_InvalidKeyRejected(t *testing.T) {
	truncateTables(context.Background(), testDBPool)
	s := setupTestDB(t)
	router := api.Router(s, nil, testAPIKey, testEncryptionKey, "")

	req, _ := http.NewRequest("GET", "/api/v1/accounts", nil)
	req.Header.Set("Authorization", "Bearer sk_live_invalid")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("invalid key should be 401, got %d", w.Code)
	}
}

func TestDualAuth_BearerTokenWithStaticKey(t *testing.T) {
	truncateTables(context.Background(), testDBPool)
	s := setupTestDB(t)
	router := api.Router(s, nil, testAPIKey, testEncryptionKey, "")

	// Static key sent via Bearer token should also work
	req, _ := http.NewRequest("GET", "/api/v1/accounts", nil)
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("static key via Bearer should work, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDualAuth_NoKeyRejected(t *testing.T) {
	truncateTables(context.Background(), testDBPool)
	s := setupTestDB(t)
	router := api.Router(s, nil, testAPIKey, testEncryptionKey, "")

	req, _ := http.NewRequest("GET", "/api/v1/accounts", nil)
	// No key at all
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("no key should be 401, got %d", w.Code)
	}
}
