//go:build phase2
// +build phase2

package integration

// ============================================================================
// PHASE 2 TESTS — OAuth flow
// These tests will NOT compile until the Phase 2 branch merges, which adds:
//   - internal/oauth package
//   - OAuth token storage and encryption
//   - SupportsOAuth, GetOAuthURL, HandleOAuthCallback on the Provider interface
//   - OAuth-related MockProvider fields
//
// Once Phase 2 merges, remove the "// +build phase2" directive above.
// ============================================================================

/*
import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ondapile/internal/adapter"
	"ondapile/internal/config"
	"ondapile/internal/model"
	"ondapile/internal/oauth"
	"ondapile/internal/store"
)

// ==================== OAuth URL Generation ====================

func TestOAuthAuthorizeURLGeneration(t *testing.T) {
	router, _ := setupTest(t)

	mock := &MockProvider{
		SupportsOAuthFunc: func() bool { return true },
		GetOAuthURLFunc: func(ctx context.Context, accountID string, redirectURI string) (string, error) {
			u, _ := url.Parse("https://accounts.google.com/o/oauth2/v2/auth")
			q := u.Query()
			q.Set("client_id", "test-client-id")
			q.Set("redirect_uri", redirectURI)
			q.Set("scope", "https://www.googleapis.com/auth/gmail.readonly")
			q.Set("state", accountID)
			q.Set("response_type", "code")
			u.RawQuery = q.Encode()
			return u.String(), nil
		},
	}
	adapter.Register(mock)

	body := `{"provider":"MOCK","identifier":"oauth-test","credentials":{}}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)
	var acc map[string]any
	json.Unmarshal(resp.Body.Bytes(), &acc)
	accountID := acc["id"].(string)

	resp = apiRequest(t, router, "GET",
		fmt.Sprintf("/api/v1/accounts/%s/oauth/authorize?redirect_uri=https://myapp.com/callback", accountID),
		nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result map[string]any
	json.Unmarshal(resp.Body.Bytes(), &result)
	authURL := result["url"].(string)

	// Verify URL contains required OAuth params
	parsed, err := url.Parse(authURL)
	require.NoError(t, err)
	assert.Equal(t, "accounts.google.com", parsed.Host)
	assert.Equal(t, "test-client-id", parsed.Query().Get("client_id"))
	assert.Equal(t, "https://myapp.com/callback", parsed.Query().Get("redirect_uri"))
	assert.NotEmpty(t, parsed.Query().Get("scope"))
	assert.Equal(t, accountID, parsed.Query().Get("state"))
	assert.Equal(t, "code", parsed.Query().Get("response_type"))
}

func TestOAuthAuthorizeURLNonOAuthProvider(t *testing.T) {
	router, _ := setupTest(t)

	mock := &MockProvider{
		SupportsOAuthFunc: func() bool { return false },
	}
	adapter.Register(mock)

	body := `{"provider":"MOCK","identifier":"no-oauth","credentials":{}}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)
	var acc map[string]any
	json.Unmarshal(resp.Body.Bytes(), &acc)

	resp = apiRequest(t, router, "GET",
		fmt.Sprintf("/api/v1/accounts/%s/oauth/authorize", acc["id"]),
		nil, testAPIKey)
	requireStatus(t, resp, http.StatusBadRequest)
}

// ==================== OAuth Callback ====================

func TestOAuthCallbackWithMockTokenEndpoint(t *testing.T) {
	// Create a mock token endpoint
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "POST", r.Method)
		require.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		r.ParseForm()
		assert.Equal(t, "authorization_code", r.FormValue("grant_type"))
		assert.NotEmpty(t, r.FormValue("code"))
		assert.NotEmpty(t, r.FormValue("redirect_uri"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "mock-access-token-12345",
			"refresh_token": "mock-refresh-token-67890",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"scope":         "https://www.googleapis.com/auth/gmail.readonly",
		})
	}))
	defer tokenServer.Close()

	router, _ := setupTest(t)

	mock := &MockProvider{
		SupportsOAuthFunc: func() bool { return true },
		HandleOAuthCallbackFunc: func(ctx context.Context, accountID string, code string, redirectURI string) (*model.Account, error) {
			return &model.Account{
				Object:   "account",
				ID:       accountID,
				Provider: "MOCK",
				Status:   model.StatusOperational,
			}, nil
		},
	}
	adapter.Register(mock)

	body := `{"provider":"MOCK","identifier":"oauth-cb","credentials":{}}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)
	var acc map[string]any
	json.Unmarshal(resp.Body.Bytes(), &acc)
	accountID := acc["id"].(string)

	cbBody := fmt.Sprintf(`{"code":"auth-code-from-google","redirect_uri":"https://myapp.com/callback"}`)
	resp = apiRequest(t, router, "POST",
		fmt.Sprintf("/api/v1/accounts/%s/oauth/callback", accountID),
		[]byte(cbBody), testAPIKey)
	requireStatus(t, resp, http.StatusOK)
}

func TestOAuthCallbackMissingCode(t *testing.T) {
	router, _ := setupTest(t)

	mock := &MockProvider{
		SupportsOAuthFunc: func() bool { return true },
	}
	adapter.Register(mock)

	body := `{"provider":"MOCK","identifier":"oauth-nocode","credentials":{}}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)
	var acc map[string]any
	json.Unmarshal(resp.Body.Bytes(), &acc)

	resp = apiRequest(t, router, "POST",
		fmt.Sprintf("/api/v1/accounts/%s/oauth/callback", acc["id"]),
		[]byte(`{}`), testAPIKey)
	requireStatus(t, resp, http.StatusUnprocessableEntity)
}

// ==================== Token Storage ====================

func TestOAuthTokenStorageEncryptedRoundtrip(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)

	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider: "MOCK", Name: "Token Test", Identifier: "token-test",
		Status: string(model.StatusOperational),
	})
	require.NoError(t, err)

	// Store credentials using encryption
	creds := map[string]string{
		"access_token":  "mock-access-token",
		"refresh_token": "mock-refresh-token",
		"token_type":    "Bearer",
	}

	encrypted, err := config.EncryptCredentials(creds, testEncryptionKey)
	require.NoError(t, err)

	err = accountStore.UpdateCredentials(ctx, account.ID, encrypted)
	require.NoError(t, err)

	// Retrieve and decrypt
	encBytes, err := accountStore.GetCredentialsEnc(ctx, account.ID)
	require.NoError(t, err)
	require.NotNil(t, encBytes)

	decrypted, err := config.DecryptCredentials(encBytes, testEncryptionKey)
	require.NoError(t, err)

	assert.Equal(t, "mock-access-token", decrypted["access_token"])
	assert.Equal(t, "mock-refresh-token", decrypted["refresh_token"])
	assert.Equal(t, "Bearer", decrypted["token_type"])
}

func TestOAuthTokenEncryptionDifferentKeys(t *testing.T) {
	creds := map[string]string{"token": "secret-value"}

	key1 := config.DeriveKey("key-one")
	key2 := config.DeriveKey("key-two")

	encrypted, err := config.EncryptCredentials(creds, key1)
	require.NoError(t, err)

	// Decrypting with wrong key should fail
	_, err = config.DecryptCredentials(encrypted, key2)
	assert.Error(t, err, "Decrypting with wrong key should fail")
}

// ==================== Token Refresh Flow ====================

func TestOAuthTokenRefreshFlow(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)

	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider: "MOCK", Name: "Refresh Test", Identifier: "refresh-test",
		Status: string(model.StatusOperational),
	})
	require.NoError(t, err)

	// Store initial token
	creds := map[string]string{
		"access_token":  "old-access-token",
		"refresh_token": "refresh-token-123",
	}
	encrypted, err := config.EncryptCredentials(creds, testEncryptionKey)
	require.NoError(t, err)
	err = accountStore.UpdateCredentials(ctx, account.ID, encrypted)
	require.NoError(t, err)

	// Simulate refresh: update with new access token
	newCreds := map[string]string{
		"access_token":  "new-access-token",
		"refresh_token": "refresh-token-123", // refresh token usually stays the same
	}
	newEncrypted, err := config.EncryptCredentials(newCreds, testEncryptionKey)
	require.NoError(t, err)
	err = accountStore.UpdateCredentials(ctx, account.ID, newEncrypted)
	require.NoError(t, err)

	// Verify the refreshed token
	encBytes, err := accountStore.GetCredentialsEnc(ctx, account.ID)
	require.NoError(t, err)
	decrypted, err := config.DecryptCredentials(encBytes, testEncryptionKey)
	require.NoError(t, err)

	assert.Equal(t, "new-access-token", decrypted["access_token"])
	assert.Equal(t, "refresh-token-123", decrypted["refresh_token"])
}

func TestOAuthTokenRefreshPreservesAccountStatus(t *testing.T) {
	ctx := context.Background()
	err := truncateTables(ctx, testDBPool)
	require.NoError(t, err)

	s := &store.Store{Pool: testDBPool}
	accountStore := store.NewAccountStore(s)

	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider: "MOCK", Name: "Refresh Status Test", Identifier: "refresh-status",
		Status: string(model.StatusOperational),
	})
	require.NoError(t, err)

	// After token refresh, account status should remain OPERATIONAL
	creds := map[string]string{"access_token": "refreshed-token"}
	encrypted, _ := config.EncryptCredentials(creds, testEncryptionKey)
	err = accountStore.UpdateCredentials(ctx, account.ID, encrypted)
	require.NoError(t, err)

	refreshedAccount, err := accountStore.GetByID(ctx, account.ID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusOperational, refreshedAccount.Status)
}
*/
