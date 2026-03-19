package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ondapile/internal/model"
)

// TestCreateWebhook tests POST /api/v1/webhooks
func TestCreateWebhook(t *testing.T) {
	router, _ := setupTest(t)

	body := `{
		"url": "https://example.com/webhook",
		"events": ["message.received", "message.sent"],
		"secret": "my-webhook-secret"
	}`

	resp := apiRequest(t, router, "POST", "/api/v1/webhooks", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var result model.Webhook
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "webhook", result.Object)
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "https://example.com/webhook", result.URL)
	assert.Equal(t, []string{"message.received", "message.sent"}, result.Events)
	assert.Equal(t, "my-webhook-secret", result.Secret)
	assert.True(t, result.Active)
	assert.NotZero(t, result.CreatedAt)
}

// TestCreateWebhookMissingFields tests validation errors
func TestCreateWebhookMissingFields(t *testing.T) {
	router, _ := setupTest(t)

	tests := []struct {
		name       string
		body       string
		expectCode string
	}{
		{
			name:       "missing url",
			body:       `{"events": ["message.received"]}`,
			expectCode: "VALIDATION_ERROR",
		},
		{
			name:       "missing events",
			body:       `{"url": "https://example.com/webhook"}`,
			expectCode: "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := apiRequest(t, router, "POST", "/api/v1/webhooks", []byte(tt.body), testAPIKey)
			requireStatus(t, resp, http.StatusUnprocessableEntity)

			var errResp map[string]interface{}
			err := json.Unmarshal(resp.Body.Bytes(), &errResp)
			require.NoError(t, err)

			assert.Equal(t, "error", errResp["object"])
			assert.Equal(t, tt.expectCode, errResp["code"])
		})
	}
}

// TestCreateWebhookAutoGeneratesSecret tests that secret is auto-generated if not provided
func TestCreateWebhookAutoGeneratesSecret(t *testing.T) {
	router, _ := setupTest(t)

	body := `{
		"url": "https://example.com/webhook",
		"events": ["message.received"]
	}`

	resp := apiRequest(t, router, "POST", "/api/v1/webhooks", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var result model.Webhook
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "webhook", result.Object)
	assert.NotEmpty(t, result.Secret)
	assert.True(t, len(result.Secret) > 10) // Secret should be reasonably long
}

// TestListWebhooks tests GET /api/v1/webhooks
func TestListWebhooks(t *testing.T) {
	router, _ := setupTest(t)

	// Create multiple webhooks
	webhooks := []struct {
		url    string
		events []string
	}{
		{"https://example.com/webhook1", []string{"message.received"}},
		{"https://example.com/webhook2", []string{"message.sent"}},
		{"https://example.com/webhook3", []string{"account.connected", "account.disconnected"}},
	}

	for _, wh := range webhooks {
		body := `{
			"url": "` + wh.url + `",
			"events": ["` + wh.events[0] + `"]
		}`
		if len(wh.events) > 1 {
			body = `{
				"url": "` + wh.url + `",
				"events": ["` + wh.events[0] + `", "` + wh.events[1] + `"]
			}`
		}
		resp := apiRequest(t, router, "POST", "/api/v1/webhooks", []byte(body), testAPIKey)
		requireStatus(t, resp, http.StatusCreated)
	}

	// List webhooks
	resp := apiRequest(t, router, "GET", "/api/v1/webhooks", nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.PaginatedList[map[string]interface{}]
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "list", result.Object)
	assert.Len(t, result.Items, 3)
}

// TestListWebhooksReturnsEmptyList tests that listing with no webhooks returns empty list
func TestListWebhooksReturnsEmptyList(t *testing.T) {
	router, _ := setupTest(t)

	resp := apiRequest(t, router, "GET", "/api/v1/webhooks", nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.PaginatedList[map[string]interface{}]
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "list", result.Object)
	assert.Len(t, result.Items, 0)
	assert.False(t, result.HasMore)
}

// TestDeleteWebhook tests DELETE /api/v1/webhooks/:id
func TestDeleteWebhook(t *testing.T) {
	router, _ := setupTest(t)

	// Create a webhook
	body := `{
		"url": "https://example.com/webhook-to-delete",
		"events": ["message.received"]
	}`
	resp := apiRequest(t, router, "POST", "/api/v1/webhooks", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var created model.Webhook
	err := json.Unmarshal(resp.Body.Bytes(), &created)
	require.NoError(t, err)

	// Delete the webhook
	resp = apiRequest(t, router, "DELETE", "/api/v1/webhooks/"+created.ID, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "webhook", result["object"])
	assert.Equal(t, created.ID, result["id"])
	assert.Equal(t, true, result["deleted"])

	// Verify webhook is gone (list should be empty)
	resp = apiRequest(t, router, "GET", "/api/v1/webhooks", nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var listResult model.PaginatedList[map[string]interface{}]
	err = json.Unmarshal(resp.Body.Bytes(), &listResult)
	require.NoError(t, err)
	assert.Len(t, listResult.Items, 0)
}

// TestDeleteWebhookNotFound tests deleting a non-existent webhook (idempotent delete)
func TestDeleteWebhookNotFound(t *testing.T) {
	router, _ := setupTest(t)

	resp := apiRequest(t, router, "DELETE", "/api/v1/webhooks/nonexistent", nil, testAPIKey)
	// Our API uses idempotent deletes — returns 200 even for nonexistent
	requireStatus(t, resp, http.StatusOK)
}

// TestWebhookResponseStructure tests that webhook response has correct structure
func TestWebhookResponseStructure(t *testing.T) {
	router, _ := setupTest(t)

	body := `{
		"url": "https://hooks.example.com/endpoint",
		"events": ["message.received", "message.sent", "account.connected"],
		"secret": "whsec_test_secret"
	}`

	resp := apiRequest(t, router, "POST", "/api/v1/webhooks", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var result map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	// Check all expected fields
	assert.Equal(t, "webhook", result["object"])
	assert.NotNil(t, result["id"])
	assert.Equal(t, "https://hooks.example.com/endpoint", result["url"])
	assert.NotNil(t, result["events"])
	assert.NotNil(t, result["secret"])
	assert.NotNil(t, result["active"])
	assert.NotNil(t, result["created_at"])

	// Check types
	events, ok := result["events"].([]interface{})
	require.True(t, ok, "events should be an array")
	assert.Len(t, events, 3)

	active, ok := result["active"].(bool)
	require.True(t, ok, "active should be a boolean")
	assert.True(t, active)
}
