package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ondapile/internal/adapter"
	"ondapile/internal/model"
)

// TestCreateAccount tests POST /api/v1/accounts
func TestCreateAccount(t *testing.T) {
	router, _ := setupTest(t)

	body := `{
		"provider": "MOCK",
		"identifier": "test-user-1",
		"name": "Test Account",
		"credentials": {"token": "test-token"}
	}`

	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var result model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "account", result.Object)
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "MOCK", result.Provider)
	assert.Equal(t, "Mock Account", result.Name)
	assert.Equal(t, model.StatusOperational, result.Status)
}

// TestCreateAccountDuplicate tests that creating an account with the same provider+identifier returns 409
func TestCreateAccountDuplicate(t *testing.T) {
	router, _ := setupTest(t)

	body := `{
		"provider": "MOCK",
		"identifier": "duplicate-user",
		"name": "First Account"
	}`

	// First creation should succeed
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	// Second creation with same provider+identifier should fail
	resp = apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusConflict)

	var errResp map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &errResp)
	require.NoError(t, err)

	assert.Equal(t, "error", errResp["object"])
	assert.Equal(t, float64(http.StatusConflict), errResp["status"])
	assert.Equal(t, "CONFLICT", errResp["code"])
}

// TestCreateAccountMissingFields tests validation errors for missing required fields
func TestCreateAccountMissingFields(t *testing.T) {
	router, _ := setupTest(t)

	tests := []struct {
		name       string
		body       string
		expectCode string
	}{
		{
			name:       "missing provider",
			body:       `{"identifier": "test", "name": "Test"}`,
			expectCode: "VALIDATION_ERROR",
		},
		{
			name:       "missing identifier",
			body:       `{"provider": "MOCK", "name": "Test"}`,
			expectCode: "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(tt.body), testAPIKey)
			requireStatus(t, resp, http.StatusUnprocessableEntity)

			var errResp map[string]interface{}
			err := json.Unmarshal(resp.Body.Bytes(), &errResp)
			require.NoError(t, err)

			assert.Equal(t, "error", errResp["object"])
			assert.Equal(t, tt.expectCode, errResp["code"])
		})
	}
}

// TestGetAccount tests GET /api/v1/accounts/:id
func TestGetAccount(t *testing.T) {
	router, _ := setupTest(t)

	// First create an account
	body := `{"provider": "MOCK", "identifier": "get-test", "name": "Get Test Account"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var created model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &created)
	require.NoError(t, err)

	// Now get the account
	resp = apiRequest(t, router, "GET", "/api/v1/accounts/"+created.ID, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.Account
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "account", result.Object)
	assert.Equal(t, created.ID, result.ID)
	assert.Equal(t, "MOCK", result.Provider)
	assert.Equal(t, "Get Test Account", result.Name)

	assert.Equal(t, "account", created.Object)
}

// TestGetAccountNotFound tests GET /api/v1/accounts/:id with non-existent ID
func TestGetAccountNotFound(t *testing.T) {
	router, _ := setupTest(t)

	resp := apiRequest(t, router, "GET", "/api/v1/accounts/nonexistent", nil, testAPIKey)
	requireStatus(t, resp, http.StatusNotFound)

	var errResp map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &errResp)
	require.NoError(t, err)

	assert.Equal(t, "error", errResp["object"])
	assert.Equal(t, float64(http.StatusNotFound), errResp["status"])
	assert.Equal(t, "NOT_FOUND", errResp["code"])
}

// TestListAccounts tests GET /api/v1/accounts
func TestListAccounts(t *testing.T) {
	router, _ := setupTest(t)

	// Create multiple accounts
	accounts := []string{"user-1", "user-2", "user-3"}
	for _, identifier := range accounts {
		body := `{"provider": "MOCK", "identifier": "` + identifier + `", "name": "Account ` + identifier + `"}`
		resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
		requireStatus(t, resp, http.StatusCreated)
	}

	// List accounts
	resp := apiRequest(t, router, "GET", "/api/v1/accounts", nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.PaginatedList[map[string]interface{}]
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "list", result.Object)
	assert.Len(t, result.Items, 3)
	assert.False(t, result.HasMore)
}

// TestListAccountsFilterByProvider tests filtering accounts by provider
func TestListAccountsFilterByProvider(t *testing.T) {
	router, _ := setupTest(t)

	// Create MOCK accounts
	for i := 1; i <= 2; i++ {
		body := `{"provider": "MOCK", "identifier": "mock-user-` + string(rune('0'+i)) + `", "name": "Mock Account"}`
		resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
		requireStatus(t, resp, http.StatusCreated)
	}

	// List with provider filter
	resp := apiRequest(t, router, "GET", "/api/v1/accounts?provider=MOCK", nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.PaginatedList[map[string]interface{}]
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "list", result.Object)
	assert.Len(t, result.Items, 2)

	// Verify all returned items are MOCK provider
	for _, item := range result.Items {
		assert.Equal(t, "MOCK", item["provider"])
	}
}

// TestDeleteAccount tests DELETE /api/v1/accounts/:id
func TestDeleteAccount(t *testing.T) {
	router, _ := setupTest(t)

	// Create an account
	body := `{"provider": "MOCK", "identifier": "delete-test", "name": "Delete Test Account"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var created model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &created)
	require.NoError(t, err)

	// Delete the account
	resp = apiRequest(t, router, "DELETE", "/api/v1/accounts/"+created.ID, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "account", result["object"])
	assert.Equal(t, created.ID, result["id"])
	assert.Equal(t, true, result["deleted"])

	// Verify account is gone
	resp = apiRequest(t, router, "GET", "/api/v1/accounts/"+created.ID, nil, testAPIKey)
	requireStatus(t, resp, http.StatusNotFound)
}

// TestGetAuthChallenge tests GET /api/v1/accounts/:id/auth-challenge
func TestGetAuthChallenge(t *testing.T) {
	router, _ := setupTest(t)

	// Create an account
	body := `{"provider": "MOCK", "identifier": "auth-test", "name": "Auth Test Account"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var created model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &created)
	require.NoError(t, err)

	// Get auth challenge
	resp = apiRequest(t, router, "GET", "/api/v1/accounts/"+created.ID+"/auth-challenge", nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result adapter.AuthChallenge
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "QR_CODE", result.Type)
	assert.Equal(t, "test-qr-data", result.Payload)
}

// TestGetAuthChallengeNotFound tests auth challenge for non-existent account
func TestGetAuthChallengeNotFound(t *testing.T) {
	router, _ := setupTest(t)

	resp := apiRequest(t, router, "GET", "/api/v1/accounts/nonexistent/auth-challenge", nil, testAPIKey)
	requireStatus(t, resp, http.StatusNotFound)

	var errResp map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &errResp)
	require.NoError(t, err)

	assert.Equal(t, "error", errResp["object"])
	assert.Equal(t, "NOT_FOUND", errResp["code"])
}
