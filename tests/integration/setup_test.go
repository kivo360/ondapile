package integration

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"ondapile/internal/adapter"
	"ondapile/internal/api"
	"ondapile/internal/config"
	"ondapile/internal/store"
	"ondapile/internal/webhook"
)

const (
	testDBName = "ondapile_test"
	testAPIKey = "test-api-key"
)

var (
	testDBPool        *pgxpool.Pool
	testEncryptionKey = config.DeriveKey("test-encryption-key-for-testing")
)

// TestMain is the entry point for all integration tests.
func TestMain(m *testing.M) {
	ctx := context.Background()

	// Connect to test database
	dsn := fmt.Sprintf("host=localhost port=5432 user=kevinhill dbname=%s sslmode=disable", testDBName)
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to test database: %v\n", err)
		os.Exit(1)
	}
	testDBPool = pool

	// Run migrations
	if err := runMigrations(ctx, pool); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run migrations: %v\n", err)
		pool.Close()
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	pool.Close()
	os.Exit(code)
}

// runMigrations executes all SQL migration files from the migrations directory.
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// Migration files in order
	migrationFiles := []string{
		"001_create_accounts.sql",
		"002_create_chats.sql",
		"003_create_messages.sql",
		"004_create_webhooks.sql",
		"005_create_attendees_and_emails.sql",
		"010_add_organization_id.sql",
		"011_create_audit_log.sql",
	}

	for _, filename := range migrationFiles {
		path := filepath.Join("..", "..", "migrations", filename)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", filename, err)
		}

		_, err = pool.Exec(ctx, string(content))
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", filename, err)
		}
	}

	return nil
}

// setupTestDB returns a Store connected to the test database.
func setupTestDB(t *testing.T) *store.Store {
	t.Helper()
	return &store.Store{Pool: testDBPool}
}

// setupTestRouter returns a fully-wired Gin router with mock provider, test DB, and API key.
func setupTestRouter(t *testing.T) *gin.Engine {
	t.Helper()

	// Register mock provider
	mockProvider := &MockProvider{}
	adapter.Register(mockProvider)
	t.Cleanup(func() {
		// Note: The registry doesn't support unregistering, but for tests
		// we'll just overwrite with a new mock each time
	})

	// Create store
	s := setupTestDB(t)

	// Create webhook dispatcher
	dispatcher := webhook.NewDispatcher(store.NewWebhookStore(s))

	// Create router with test base URL for tracking
	r := api.Router(s, dispatcher, testAPIKey, testEncryptionKey, "http://localhost:8080")
	return r
}

// truncateTables truncates all tables in the database.
func truncateTables(ctx context.Context, pool *pgxpool.Pool) error {
	tables := []string{
		"webhook_deliveries",
		"webhooks",
		"messages",
		"chats",
		"emails",
		"attendees",
		"accounts",
		"apikey",
		"audit_log",
	}

	for _, table := range tables {
		_, err := pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			// Ignore errors for tables that might not exist
			if !strings.Contains(err.Error(), "does not exist") {
				return fmt.Errorf("failed to truncate %s: %w", table, err)
			}
		}
	}

	return nil
}

// apiRequest makes an HTTP request to the test router and returns the response.
func apiRequest(t *testing.T, router *gin.Engine, method, path string, body []byte, apiKey string) *httptest.ResponseRecorder {
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
		req.Header.Set("X-API-KEY", apiKey)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

// requireStatus asserts that the response has the expected status code.
func requireStatus(t *testing.T, resp *httptest.ResponseRecorder, expected int) {
	t.Helper()
	require.Equal(t, expected, resp.Code, "Expected status %d, got %d. Body: %s", expected, resp.Code, resp.Body.String())
}

// setupTest creates a fresh test environment with truncated tables.
func setupTest(t *testing.T) (*gin.Engine, *store.Store) {
	t.Helper()

	// Truncate tables before test
	err := truncateTables(context.Background(), testDBPool)
	require.NoError(t, err, "Failed to truncate tables")

	// Register mock provider
	mockProvider := &MockProvider{}
	adapter.Register(mockProvider)

	// Create store and router
	s := setupTestDB(t)
	dispatcher := webhook.NewDispatcher(store.NewWebhookStore(s))
	r := api.Router(s, dispatcher, testAPIKey, testEncryptionKey, "http://localhost:8080")

	// Cleanup after test
	t.Cleanup(func() {
		err := truncateTables(context.Background(), testDBPool)
		require.NoError(t, err, "Failed to truncate tables in cleanup")
	})

	return r, s
}

// Helper to convert sql.NullString to *string
func nullStringToPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}
