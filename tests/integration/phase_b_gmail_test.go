package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	emailStore "ondapile/internal/email"
	"ondapile/internal/model"
)

func TestGmailAdapterMethods(t *testing.T) {
	router, s := setupTest(t)

	// Create a test account
	body := `{"provider":"MOCK","identifier":"gmail-test@gmail.com","credentials":{}}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(body), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)
	var acct map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &acct)
	accountID := acct["id"].(string)

	// Store a test email in DB for reply/forward/update/delete tests
	emailStr := emailStore.NewEmailStore(s)
	testEmail := &model.Email{
		ID:        "eml_gmail_test_001",
		AccountID: accountID,
		Provider:  "MOCK",
		Subject:   "Test Email",
		Body:      "<p>Hello</p>",
		BodyPlain: "Hello",
		FromAttendee: &model.EmailAttendee{
			DisplayName:    "Sender",
			Identifier:     "sender@example.com",
			IdentifierType: "EMAIL_ADDRESS",
		},
		ToAttendees: []model.EmailAttendee{{
			DisplayName:    "Recipient",
			Identifier:     "recipient@example.com",
			IdentifierType: "EMAIL_ADDRESS",
		}},
		Date:       time.Now(),
		Role:       model.FolderInbox,
		Read:       false,
		Folders:    []string{model.FolderInbox},
		ProviderID: &model.EmailProviderID{MessageID: "msg123", ThreadID: "thread123"},
		Metadata:   map[string]any{},
	}
	emailStr.StoreEmail(context.Background(), testEmail)

	t.Run("ListFolders", func(t *testing.T) {
		// GET /api/v1/emails/folders?account_id=X
		resp := apiRequest(t, router, "GET", "/api/v1/emails/folders?account_id="+accountID, nil, testAPIKey)
		requireStatus(t, resp, http.StatusOK)

		var result []map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result), 1)
	})

	t.Run("ReplyEmail", func(t *testing.T) {
		body := fmt.Sprintf(`{"account_id":"%s","body_html":"<p>Reply</p>"}`, accountID)
		resp := apiRequest(t, router, "POST", "/api/v1/emails/eml_gmail_test_001/reply", []byte(body), testAPIKey)
		requireStatus(t, resp, http.StatusOK)

		var result model.Email
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "email", result.Object)
		assert.Contains(t, result.Subject, "Re:")
	})

	t.Run("ForwardEmail", func(t *testing.T) {
		body := fmt.Sprintf(`{"account_id":"%s","to":[{"identifier":"fwd@example.com","identifier_type":"EMAIL_ADDRESS"}],"body_html":"<p>FYI</p>"}`, accountID)
		resp := apiRequest(t, router, "POST", "/api/v1/emails/eml_gmail_test_001/forward", []byte(body), testAPIKey)
		requireStatus(t, resp, http.StatusOK)

		var result model.Email
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "email", result.Object)
		assert.Contains(t, result.Subject, "Fwd:")
	})

	t.Run("UpdateEmail_MarkRead", func(t *testing.T) {
		body := fmt.Sprintf(`{"account_id":"%s","read":true}`, accountID)
		resp := apiRequest(t, router, "PUT", "/api/v1/emails/eml_gmail_test_001", []byte(body), testAPIKey)
		requireStatus(t, resp, http.StatusOK)

		var result model.Email
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "email", result.Object)
		assert.True(t, result.Read)
	})

	t.Run("DeleteEmail", func(t *testing.T) {
		resp := apiRequest(t, router, "DELETE", "/api/v1/emails/eml_gmail_test_001?account_id="+accountID, nil, testAPIKey)
		requireStatus(t, resp, http.StatusOK)

		var result map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "email", result["object"])
		assert.Equal(t, true, result["deleted"])
	})
}
