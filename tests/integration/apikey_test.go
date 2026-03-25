package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"ondapile/internal/api"
	"ondapile/internal/store"
)

// setupApiKeyTable creates the apikey table for tests (Better Auth creates it in production)
func setupApiKeyTable(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS apikey (
			id TEXT PRIMARY KEY,
			"configId" TEXT NOT NULL,
			name TEXT,
			start TEXT,
			"referenceId" TEXT NOT NULL,
			prefix TEXT,
			key TEXT NOT NULL,
			"refillInterval" INTEGER,
			"refillAmount" INTEGER,
			"lastRefillAt" TIMESTAMPTZ,
			enabled BOOLEAN,
			"rateLimitEnabled" BOOLEAN,
			"rateLimitTimeWindow" INTEGER,
			"rateLimitMax" INTEGER,
			"requestCount" INTEGER,
			remaining INTEGER,
			"lastRequest" TIMESTAMPTZ,
			"expiresAt" TIMESTAMPTZ,
			"createdAt" TIMESTAMPTZ NOT NULL,
			"updatedAt" TIMESTAMPTZ NOT NULL,
			permissions TEXT,
			metadata TEXT
		)
	`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS apikey_configId_idx ON apikey("configId")`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS apikey_key_idx ON apikey(key)`)
	require.NoError(t, err)
_, err = pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS apikey_referenceId_idx ON apikey("referenceId")`)
	require.NoError(t, err)

	// Truncate table to ensure clean state for each test
	_, err = pool.Exec(ctx, `TRUNCATE TABLE apikey CASCADE`)
	require.NoError(t, err)
}

func TestApiKeyLookupByHash(t *testing.T) {
	ctx := context.Background()
	s := setupTestDB(t)
	setupApiKeyTable(ctx, t, s.Pool)
	apiKeyStore := store.NewApiKeyStore(s)

	// Compute SHA-256 hash of a known raw key
	rawKey := "sk_live_test123456"
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	// Insert test API key directly into database
	orgID := "org_test_123"
	configID := "cfg_test_456"
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO apikey (id, "configId", name, "referenceId", key, enabled, "createdAt", "updatedAt")
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`, "key_test_1", configID, "Test Key", orgID, keyHash, true)
	require.NoError(t, err)

	// Look up by hash
	apiKey, err := apiKeyStore.LookupByKeyHash(ctx, keyHash)
	require.NoError(t, err)
	require.NotNil(t, apiKey)
	require.Equal(t, "key_test_1", apiKey.ID)
	require.Equal(t, configID, apiKey.ConfigID)
	require.Equal(t, orgID, apiKey.ReferenceID)
	require.NotNil(t, apiKey.Name)
	require.Equal(t, "Test Key", *apiKey.Name)
	require.NotNil(t, apiKey.Enabled)
	require.True(t, *apiKey.Enabled)
}

func TestApiKeyLookupNotFound(t *testing.T) {
	ctx := context.Background()
	s := setupTestDB(t)
	setupApiKeyTable(ctx, t, s.Pool)
	apiKeyStore := store.NewApiKeyStore(s)

	// Look up nonexistent hash
	nonexistentHash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	apiKey, err := apiKeyStore.LookupByKeyHash(ctx, nonexistentHash)
	require.NoError(t, err)
	require.Nil(t, apiKey)
}

func TestApiKeyMiddleware_NoKey(t *testing.T) {
	s := setupTestDB(t)
	setupApiKeyTable(context.Background(), t, s.Pool)
	apiKeyStore := store.NewApiKeyStore(s)

	r := gin.New()
	r.GET("/test", api.ApiKeyMiddleware(apiKeyStore), func(c *gin.Context) {
		c.JSON(200, gin.H{"organization_id": c.GetString("organization_id")})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestApiKeyMiddleware_InvalidKey(t *testing.T) {
	ctx := context.Background()
	s := setupTestDB(t)
	setupApiKeyTable(ctx, t, s.Pool)
	apiKeyStore := store.NewApiKeyStore(s)

	// Insert a valid key
	rawKey := "sk_live_validkey"
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO apikey (id, "configId", "referenceId", key, enabled, "createdAt", "updatedAt")
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`, "key_valid", "cfg_1", "org_1", keyHash, true)
	require.NoError(t, err)

	r := gin.New()
	r.GET("/test", api.ApiKeyMiddleware(apiKeyStore), func(c *gin.Context) {
		c.JSON(200, gin.H{"organization_id": c.GetString("organization_id")})
	})

	// Use wrong key
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer sk_live_wrongkey")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestApiKeyMiddleware_ValidKey(t *testing.T) {
	ctx := context.Background()
	s := setupTestDB(t)
	setupApiKeyTable(ctx, t, s.Pool)
	apiKeyStore := store.NewApiKeyStore(s)

	// Insert a valid key
	rawKey := "sk_live_validkey"
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])
	orgID := "org_test_abc"
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO apikey (id, "configId", "referenceId", key, enabled, "createdAt", "updatedAt")
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`, "key_valid_2", "cfg_1", orgID, keyHash, true)
	require.NoError(t, err)

	r := gin.New()
	r.GET("/test", api.ApiKeyMiddleware(apiKeyStore), func(c *gin.Context) {
		c.JSON(200, gin.H{"organization_id": c.GetString("organization_id")})
	})

	// Use correct key
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), orgID)
}

func TestApiKeyMiddleware_DisabledKey(t *testing.T) {
	ctx := context.Background()
	s := setupTestDB(t)
	setupApiKeyTable(ctx, t, s.Pool)
	apiKeyStore := store.NewApiKeyStore(s)

	// Insert a disabled key
	rawKey := "sk_live_disabled"
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO apikey (id, "configId", "referenceId", key, enabled, "createdAt", "updatedAt")
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`, "key_disabled", "cfg_1", "org_1", keyHash, false)
	require.NoError(t, err)

	r := gin.New()
	r.GET("/test", api.ApiKeyMiddleware(apiKeyStore), func(c *gin.Context) {
		c.JSON(200, gin.H{"organization_id": c.GetString("organization_id")})
	})

	// Try to use disabled key
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestApiKeyMiddleware_ExpiredKey(t *testing.T) {
	ctx := context.Background()
	s := setupTestDB(t)
	setupApiKeyTable(ctx, t, s.Pool)
	apiKeyStore := store.NewApiKeyStore(s)

	// Insert an expired key
	rawKey := "sk_live_expired"
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])
	expiredTime := time.Now().Add(-24 * time.Hour) // Yesterday
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO apikey (id, "configId", "referenceId", key, enabled, "expiresAt", "createdAt", "updatedAt")
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`, "key_expired", "cfg_1", "org_1", keyHash, true, expiredTime)
	require.NoError(t, err)

	r := gin.New()
	r.GET("/test", api.ApiKeyMiddleware(apiKeyStore), func(c *gin.Context) {
		c.JSON(200, gin.H{"organization_id": c.GetString("organization_id")})
	})

	// Try to use expired key
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestApiKeyMiddleware_UpdatesLastUsed(t *testing.T) {
	ctx := context.Background()
	s := setupTestDB(t)
	setupApiKeyTable(ctx, t, s.Pool)
	apiKeyStore := store.NewApiKeyStore(s)

	// Insert a valid key
	rawKey := "sk_live_tracking"
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])
	keyID := "key_tracking"
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO apikey (id, "configId", "referenceId", key, enabled, "lastRequest", "createdAt", "updatedAt")
		VALUES ($1, $2, $3, $4, $5, NULL, NOW(), NOW())
	`, keyID, "cfg_1", "org_1", keyHash, true)
	require.NoError(t, err)

	r := gin.New()
	r.GET("/test", api.ApiKeyMiddleware(apiKeyStore), func(c *gin.Context) {
		c.JSON(200, gin.H{"organization_id": c.GetString("organization_id")})
	})

	// Use the key
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	// Give the async update time to complete
	time.Sleep(100 * time.Millisecond)

	// Verify lastRequest was updated
	var lastRequest *time.Time
	err = s.Pool.QueryRow(ctx, `SELECT "lastRequest" FROM apikey WHERE id = $1`, keyID).Scan(&lastRequest)
	require.NoError(t, err)
	require.NotNil(t, lastRequest)
	require.WithinDuration(t, time.Now(), *lastRequest, 5*time.Second)
}
