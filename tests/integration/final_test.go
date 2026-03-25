package integration

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"ondapile/internal/email"
	"ondapile/internal/model"
	"ondapile/internal/store"
	"ondapile/internal/tracking"
)

func TestFinalTracking(t *testing.T) {
	ctx := context.Background()
	router, s := setupTest(t)

	// Create account
	accountStore := store.NewAccountStore(s)
	account, err := accountStore.Create(ctx, store.CreateAccountParams{
		Provider:     "MOCK",
		Name:         "Final Test Account",
		Identifier:   "final@test.com",
		Status:       string(model.StatusOperational),
		Capabilities: []string{"email"},
		Metadata:     map[string]any{},
	})
	require.NoError(t, err)

	// Store email
	emailStore := email.NewEmailStore(s)
	testEmail := &model.Email{
		ID:        "eml_final",
		AccountID: account.ID,
		Provider:  "MOCK",
		Subject:   "Final Test",
		Body:      "<html><body>Test</body></html>",
		Metadata:  map[string]any{},
	}
	err = emailStore.StoreEmail(ctx, testEmail)
	require.NoError(t, err)
	fmt.Println("Email stored")

	// Check before
	before, _ := emailStore.GetEmail(ctx, "eml_final")
	fmt.Printf("Before: Tracking=%v\n", before.Tracking)

	// Use tracking store directly
	trackingStore := tracking.NewTrackingStore(emailStore)
	err = trackingStore.RecordOpen(ctx, "eml_final")
	if err != nil {
		fmt.Printf("RecordOpen error: %v\n", err)
	} else {
		fmt.Println("RecordOpen succeeded")
	}

	// Check after direct
	after, _ := emailStore.GetEmail(ctx, "eml_final")
	fmt.Printf("After direct: Tracking=%v\n", after.Tracking)
	if after.Tracking != nil {
		fmt.Printf("Opens: %d\n", after.Tracking.Opens)
	}

	// Now test via HTTP
	req := httptest.NewRequest("GET", "/t/eml_final.gif", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	fmt.Printf("HTTP status: %d\n", w.Code)

	// Check after HTTP
	after2, _ := emailStore.GetEmail(ctx, "eml_final")
	fmt.Printf("After HTTP: Tracking=%v\n", after2.Tracking)
	if after2.Tracking != nil {
		fmt.Printf("Opens: %d\n", after2.Tracking.Opens)
	}

	// Verify
	require.NotNil(t, after2.Tracking, "Tracking should not be nil after HTTP request")
	require.GreaterOrEqual(t, after2.Tracking.Opens, 1, "Should have at least 1 open")
}
