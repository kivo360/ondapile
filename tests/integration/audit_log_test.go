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
	"ondapile/internal/store"
)

func TestAuditLogStore_CreateAndList(t *testing.T) {
	ctx := context.Background()
	truncateTables(ctx, testDBPool)
	s := setupTestDB(t)
	auditStore := store.NewAuditLogStore(s)

	// Create entries for two orgs
	auditStore.Create(ctx, store.AuditEntry{
		OrganizationID: "org_A", ActorID: "user_1", ActorName: "Kevin",
		Action: "account.connected", ResourceType: "account", ResourceID: "acc_1",
		Detail: json.RawMessage(`{"provider":"google"}`),
	})
	auditStore.Create(ctx, store.AuditEntry{
		OrganizationID: "org_B", ActorID: "user_2", ActorName: "Sarah",
		Action: "webhook.created", ResourceType: "webhook", ResourceID: "whk_1",
		Detail: json.RawMessage(`{"url":"https://example.com"}`),
	})

	// List for org_A
	entries, err := auditStore.List(ctx, "org_A", 50)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1, got %d", len(entries))
	}
	if entries[0].Action != "account.connected" {
		t.Errorf("wrong action: %s", entries[0].Action)
	}
}

func TestAuditLogAPI_OrgScoped(t *testing.T) {
	ctx := context.Background()
	truncateTables(ctx, testDBPool)
	s := setupTestDB(t)

	// Insert API key + audit entries
	rawKey := "sk_live_audit_test"
	hash := sha256.Sum256([]byte(rawKey))
	now := time.Now()
	testDBPool.Exec(ctx, `INSERT INTO apikey (id, "configId", "referenceId", key, prefix, enabled, "createdAt", "updatedAt") VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		"key_audit", "default", "org_audit", hex.EncodeToString(hash[:]), "sk_live_", true, now, now)

	auditStore := store.NewAuditLogStore(s)
	auditStore.Create(ctx, store.AuditEntry{
		OrganizationID: "org_audit", ActorID: "user_1", ActorName: "Kevin",
		Action: "api_key.created", ResourceType: "apikey", ResourceID: "key_1",
		Detail: json.RawMessage(`{"name":"Production"}`),
	})
	auditStore.Create(ctx, store.AuditEntry{
		OrganizationID: "org_other", ActorID: "user_2", ActorName: "Other",
		Action: "account.connected", ResourceType: "account", ResourceID: "acc_1",
		Detail: json.RawMessage(`{}`),
	})

	router := api.Router(s, nil, testAPIKey, testEncryptionKey)

	req, _ := http.NewRequest("GET", "/api/v1/audit-log", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	items := resp["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("expected 1 org-scoped entry, got %d", len(items))
	}
	entry := items[0].(map[string]interface{})
	if entry["action"] != "api_key.created" {
		t.Errorf("wrong action: %v", entry["action"])
	}
}

func TestAuditLogAPI_EmptyWithStaticKey(t *testing.T) {
	ctx := context.Background()
	truncateTables(ctx, testDBPool)
	s := setupTestDB(t)
	router := api.Router(s, nil, testAPIKey, testEncryptionKey)

	req, _ := http.NewRequest("GET", "/api/v1/audit-log", nil)
	req.Header.Set("X-API-KEY", testAPIKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	items := resp["items"].([]interface{})
	if len(items) != 0 {
		t.Fatalf("static key should return empty audit log, got %d", len(items))
	}
}
