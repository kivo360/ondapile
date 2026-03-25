package integration

import (
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

// TestOutlookAdapterMethods tests the 5 new Outlook adapter email methods
// using the MockProvider with custom Func implementations.

// Test ReplyEmail endpoint
func TestOutlookAdapterReplyEmail(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "reply-test", "name": "Reply Test Account"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create an email via email store
	emailStr := emailStore.NewEmailStore(s)
	email := &model.Email{
		ID:        "eml_reply_test_001",
		AccountID: account.ID,
		Provider:  "MOCK",
		Subject:   "Original Email",
		Body:      "Original body",
		Role:      model.FolderInbox,
		Folders:   []string{model.FolderInbox},
		Read:      true,
		Date:      time.Now(),
		Metadata:  map[string]any{},
	}
	err = emailStr.StoreEmail(t.Context(), email)
	require.NoError(t, err)

	// Reply to the email
	replyBody := fmt.Sprintf(`{"account_id": "%s", "to": [{"identifier": "reply@example.com", "display_name": "Reply User"}], "subject": "Original Email", "body_html": "This is a reply"}`, account.ID)
	resp = apiRequest(t, router, "POST", "/api/v1/emails/"+email.ID+"/reply", []byte(replyBody), testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.Email
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "email", result.Object)
	assert.Contains(t, result.Subject, "Re:")
	assert.Equal(t, "This is a reply", result.Body)
	assert.Equal(t, account.ID, result.AccountID)
}

// Test ForwardEmail endpoint
func TestOutlookAdapterForwardEmail(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "forward-test", "name": "Forward Test Account"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create an email via email store
	emailStr := emailStore.NewEmailStore(s)
	email := &model.Email{
		ID:        "eml_fwd_test_001",
		AccountID: account.ID,
		Provider:  "MOCK",
		Subject:   "Email to Forward",
		Body:      "Body to forward",
		Role:      model.FolderInbox,
		Folders:   []string{model.FolderInbox},
		Read:      true,
		Date:      time.Now(),
		Metadata:  map[string]any{},
	}
	err = emailStr.StoreEmail(t.Context(), email)
	require.NoError(t, err)

	// Forward the email
	forwardBody := fmt.Sprintf(`{"account_id": "%s", "to": [{"identifier": "forward@example.com", "display_name": "Forward User"}], "subject": "Email to Forward", "body_html": "Forwarding this email"}`, account.ID)
	resp = apiRequest(t, router, "POST", "/api/v1/emails/"+email.ID+"/forward", []byte(forwardBody), testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.Email
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "email", result.Object)
	assert.Contains(t, result.Subject, "Fwd:")
	assert.Equal(t, "Forwarding this email", result.Body)
	assert.Equal(t, account.ID, result.AccountID)
}

// Test UpdateEmailProvider with Read status
func TestOutlookAdapterUpdateEmailRead(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "update-read-test", "name": "Update Read Test Account"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create an unread email via email store
	emailStr := emailStore.NewEmailStore(s)
	email := &model.Email{
		ID:        "eml_update_test_001",
		AccountID: account.ID,
		Provider:  "MOCK",
		Subject:   "Email to Update",
		Body:      "Test body",
		Role:      model.FolderInbox,
		Folders:   []string{model.FolderInbox},
		Read:      false,
		Date:      time.Now(),
		Metadata:  map[string]any{},
	}
	err = emailStr.StoreEmail(t.Context(), email)
	require.NoError(t, err)

	// Update email as read
	updateBody := `{"read": true}`
	resp = apiRequest(t, router, "PUT", "/api/v1/emails/"+email.ID, []byte(updateBody), testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.Email
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "email", result.Object)
	assert.True(t, result.Read)
}

// Test UpdateEmailProvider with Folder move
func TestOutlookAdapterUpdateEmailFolder(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "update-folder-test", "name": "Update Folder Test Account"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create an email in inbox via email store
	emailStr := emailStore.NewEmailStore(s)
	email := &model.Email{
		ID:        "eml_update_folder_001",
		AccountID: account.ID,
		Provider:  "MOCK",
		Subject:   "Email to Move",
		Body:      "Test body",
		Role:      model.FolderInbox,
		Folders:   []string{model.FolderInbox},
		Read:      false,
		Date:      time.Now(),
		Metadata:  map[string]any{},
	}
	err = emailStr.StoreEmail(t.Context(), email)
	require.NoError(t, err)

	// Move email to archive folder
	updateBody := `{"folder": "ARCHIVE"}`
	resp = apiRequest(t, router, "PUT", "/api/v1/emails/"+email.ID, []byte(updateBody), testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result model.Email
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "email", result.Object)
	assert.Equal(t, model.FolderArchive, result.Role)
}

// Test DeleteEmailProvider endpoint
func TestOutlookAdapterDeleteEmail(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "delete-test", "name": "Delete Test Account"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create an email via email store
	emailStr := emailStore.NewEmailStore(s)
	email := &model.Email{
		ID:        "eml_delete_test_001",
		AccountID: account.ID,
		Provider:  "MOCK",
		Subject:   "Email to Delete",
		Body:      "Test body",
		Role:      model.FolderInbox,
		Folders:   []string{model.FolderInbox},
		Read:      false,
		Date:      time.Now(),
		Metadata:  map[string]any{},
	}
	err = emailStr.StoreEmail(t.Context(), email)
	require.NoError(t, err)

	// Delete the email
	resp = apiRequest(t, router, "DELETE", "/api/v1/emails/"+email.ID, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "email", result["object"])
	assert.Equal(t, email.ID, result["id"])
	assert.Equal(t, true, result["deleted"])

	// Verify email is deleted from store
	emailStoreCheck := emailStore.NewEmailStore(s)
	deletedEmail, err := emailStoreCheck.GetEmail(t.Context(), email.ID)
	require.NoError(t, err)
	assert.Nil(t, deletedEmail, "Email should be deleted from store")
}

// Test ListFolders endpoint
func TestOutlookAdapterListFolders(t *testing.T) {
	router, s := setupTest(t)

	// Create an account
	accountBody := `{"provider": "MOCK", "identifier": "folders-test", "name": "Folders Test Account"}`
	resp := apiRequest(t, router, "POST", "/api/v1/accounts", []byte(accountBody), testAPIKey)
	requireStatus(t, resp, http.StatusCreated)

	var account model.Account
	err := json.Unmarshal(resp.Body.Bytes(), &account)
	require.NoError(t, err)

	// Create emails in different folders
	emailStr := emailStore.NewEmailStore(s)

	// Email in inbox
	email1 := &model.Email{
		ID:        "eml_folder_inbox_001",
		AccountID: account.ID,
		Provider:  "MOCK",
		Subject:   "Inbox Email",
		Body:      "Test body",
		Role:      model.FolderInbox,
		Folders:   []string{model.FolderInbox},
		Read:      false,
		Date:      time.Now(),
		Metadata:  map[string]any{},
	}
	err = emailStr.StoreEmail(t.Context(), email1)
	require.NoError(t, err)

	// Email in archive
	email2 := &model.Email{
		ID:        "eml_folder_archive_001",
		AccountID: account.ID,
		Provider:  "MOCK",
		Subject:   "Archived Email",
		Body:      "Test body",
		Role:      model.FolderArchive,
		Folders:   []string{model.FolderArchive},
		Read:      true,
		Date:      time.Now(),
		Metadata:  map[string]any{},
	}
	err = emailStr.StoreEmail(t.Context(), email2)
	require.NoError(t, err)

	// List folders
	resp = apiRequest(t, router, "GET", "/api/v1/emails/folders?account_id="+account.ID, nil, testAPIKey)
	requireStatus(t, resp, http.StatusOK)

	var result []map[string]interface{}
	err = json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	// Should have multiple folders
	assert.GreaterOrEqual(t, len(result), 2)

	// Find inbox and archive folders
	var inboxFound, archiveFound bool
	for _, folder := range result {
		role := folder["role"].(string)
		if role == model.FolderInbox {
			inboxFound = true
			assert.GreaterOrEqual(t, int(folder["total"].(float64)), 0)
		}
		if role == model.FolderArchive {
			archiveFound = true
			assert.GreaterOrEqual(t, int(folder["total"].(float64)), 0)
		}
	}
	assert.True(t, inboxFound, "INBOX folder should be present")
	assert.True(t, archiveFound, "ARCHIVE folder should be present")
}
