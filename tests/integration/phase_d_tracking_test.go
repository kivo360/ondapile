package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ondapile/internal/email"
	"ondapile/internal/model"
	"ondapile/internal/store"
	"ondapile/internal/tracking"
)

func TestEmailTracking(t *testing.T) {
	ctx := context.Background()

	// Use setupTest to get both router and store (they share the same DB connection)
	router, s := setupTest(t)

	// Create account
	accountStore := store.NewAccountStore(s)
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Tracking Test Account",
		Identifier:   "tracking-test@test.com",
		Status:       string(model.StatusOperational),
		Capabilities: []string{"email"},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)
	accountID := account.ID

	// Store a test email in DB
	emailStore := email.NewEmailStore(s)
	testEmail := &model.Email{
		ID:        "eml_track_001",
		AccountID: accountID,
		Provider:  "MOCK",
		Subject:   "Tracked Email",
		Body:      "<html><body><p>Hello</p><a href=\"https://example.com\">Click</a></body></html>",
		Metadata:  map[string]any{},
	}
	err = emailStore.StoreEmail(ctx, testEmail)
	require.NoError(t, err)

	t.Run("TrackingPixelServesGIF", func(t *testing.T) {
		// GET /t/eml_track_001 — no auth required
		req := httptest.NewRequest("GET", "/t/eml_track_001", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "image/gif", w.Header().Get("Content-Type"))
		require.True(t, len(w.Body.Bytes()) > 0)
	})

	t.Run("TrackingPixelIncrementsOpens", func(t *testing.T) {
		// Hit pixel twice
		req1 := httptest.NewRequest("GET", "/t/eml_track_001", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		require.Equal(t, http.StatusOK, w1.Code)

		req2 := httptest.NewRequest("GET", "/t/eml_track_001", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		require.Equal(t, http.StatusOK, w2.Code)

		// Verify tracking updated in DB
		updatedEmail, err := emailStore.GetEmail(ctx, "eml_track_001")
		require.NoError(t, err)
		require.NotNil(t, updatedEmail)
		require.NotNil(t, updatedEmail.Tracking)
		assert.GreaterOrEqual(t, updatedEmail.Tracking.Opens, 2)
	})

	t.Run("LinkRedirectWorks", func(t *testing.T) {
		// GET /l/eml_track_001?url=https://example.com — should redirect
		req := httptest.NewRequest("GET", "/l/eml_track_001?url=https://example.com", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusFound, w.Code)
		require.Equal(t, "https://example.com", w.Header().Get("Location"))
	})

	t.Run("LinkClickRecorded", func(t *testing.T) {
		// Click a link
		req := httptest.NewRequest("GET", "/l/eml_track_001?url=https://clicked.com", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusFound, w.Code)

		// Verify click recorded
		updatedEmail, err := emailStore.GetEmail(ctx, "eml_track_001")
		require.NoError(t, err)
		require.NotNil(t, updatedEmail)
		require.NotNil(t, updatedEmail.Tracking)
		assert.GreaterOrEqual(t, updatedEmail.Tracking.Clicks, 1)
	})

	t.Run("InjectTrackingModifiesHTML", func(t *testing.T) {
		tracker := tracking.NewTracker(emailStore, nil, "http://localhost:8080")
		modified, err := tracker.InjectTracking("eml_test", "<html><body><a href=\"https://example.com\">Link</a></body></html>")
		require.NoError(t, err)
		assert.Contains(t, modified, "/t/eml_test")
		assert.Contains(t, modified, "/l/eml_test?url=")
	})

	t.Run("InjectTrackingPreservesNonTrackableLinks", func(t *testing.T) {
		tracker := tracking.NewTracker(emailStore, nil, "http://localhost:8080")
		html := `<html><body>
			<a href="mailto:test@example.com">Email</a>
			<a href="tel:+1234567890">Call</a>
			<a href="javascript:void(0)">Script</a>
			<a href="#anchor">Anchor</a>
			<a href="https://example.com">Normal</a>
		</body></html>`
		modified, err := tracker.InjectTracking("eml_test", html)
		require.NoError(t, err)

		// Non-trackable links should remain unchanged
		assert.Contains(t, modified, `href="mailto:test@example.com"`)
		assert.Contains(t, modified, `href="tel:+1234567890"`)
		assert.Contains(t, modified, `href="javascript:void(0)"`)
		assert.Contains(t, modified, `href="#anchor"`)

		// Normal link should be wrapped
		assert.Contains(t, modified, "/l/eml_test?url=")
	})

	t.Run("TrackingPixelWithNonExistentEmail", func(t *testing.T) {
		// Should still serve pixel even if email doesn't exist
		req := httptest.NewRequest("GET", "/t/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should still return the pixel (don't leak errors to email client)
		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "image/gif", w.Header().Get("Content-Type"))
	})

	t.Run("LinkRedirectMissingURL", func(t *testing.T) {
		// Should return 400 if url parameter is missing
		req := httptest.NewRequest("GET", "/l/eml_track_001", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestEmailTrackingFirstOpen(t *testing.T) {
	ctx := context.Background()

	// Use setupTest to get both router and store
	router, s := setupTest(t)

	// Create account
	accountStore := store.NewAccountStore(s)
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "First Open Test Account",
		Identifier:   "first-open@test.com",
		Status:       string(model.StatusOperational),
		Capabilities: []string{"email"},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Store a test email in DB
	emailStore := email.NewEmailStore(s)
	testEmail := &model.Email{
		ID:        "eml_first_open",
		AccountID: account.ID,
		Provider:  "MOCK",
		Subject:   "First Open Test",
		Body:      "<html><body>Test</body></html>",
		Metadata:  map[string]any{},
	}
	err = emailStore.StoreEmail(ctx, testEmail)
	require.NoError(t, err)

	// First open
	req1 := httptest.NewRequest("GET", "/t/eml_first_open", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	require.Equal(t, http.StatusOK, w1.Code)

	// Verify first_opened_at is set
	updatedEmail, err := emailStore.GetEmail(ctx, "eml_first_open")
	require.NoError(t, err)
	require.NotNil(t, updatedEmail)
	require.NotNil(t, updatedEmail.Tracking)
	require.NotNil(t, updatedEmail.Tracking.FirstOpenedAt)
	assert.Equal(t, 1, updatedEmail.Tracking.Opens)

	firstOpenedAt := *updatedEmail.Tracking.FirstOpenedAt

	// Second open
	req2 := httptest.NewRequest("GET", "/t/eml_first_open", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)

	// Verify first_opened_at hasn't changed
	updatedEmail2, err := emailStore.GetEmail(ctx, "eml_first_open")
	require.NoError(t, err)
	require.NotNil(t, updatedEmail2)
	require.NotNil(t, updatedEmail2.Tracking)
	assert.Equal(t, 2, updatedEmail2.Tracking.Opens)
	assert.Equal(t, firstOpenedAt, *updatedEmail2.Tracking.FirstOpenedAt)
}

func TestEmailTrackingMultipleLinks(t *testing.T) {
	ctx := context.Background()

	// Use setupTest to get both router and store
	router, s := setupTest(t)

	// Create account
	accountStore := store.NewAccountStore(s)
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Multi Link Test Account",
		Identifier:   "multi-link@test.com",
		Status:       string(model.StatusOperational),
		Capabilities: []string{"email"},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Store a test email in DB
	emailStore := email.NewEmailStore(s)
	testEmail := &model.Email{
		ID:        "eml_multi_link",
		AccountID: account.ID,
		Provider:  "MOCK",
		Subject:   "Multi Link Test",
		Body:      "<html><body>Test</body></html>",
		Metadata:  map[string]any{},
	}
	err = emailStore.StoreEmail(ctx, testEmail)
	require.NoError(t, err)

	// Click multiple different links
	links := []string{
		"https://example.com/page1",
		"https://example.com/page2",
		"https://example.com/page1", // Duplicate
	}

	for _, link := range links {
		req := httptest.NewRequest("GET", "/l/eml_multi_link?url="+link, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusFound, w.Code)
	}

	// Verify all clicks recorded
	updatedEmail, err := emailStore.GetEmail(ctx, "eml_multi_link")
	require.NoError(t, err)
	require.NotNil(t, updatedEmail)
	require.NotNil(t, updatedEmail.Tracking)
	assert.Equal(t, 3, updatedEmail.Tracking.Clicks)
	assert.Len(t, updatedEmail.Tracking.LinksClicked, 3)

	// Verify URLs are recorded correctly
	urls := make([]string, len(updatedEmail.Tracking.LinksClicked))
	for i, link := range updatedEmail.Tracking.LinksClicked {
		urls[i] = link.URL
	}
	assert.Contains(t, urls, "https://example.com/page1")
	assert.Contains(t, urls, "https://example.com/page2")
}
