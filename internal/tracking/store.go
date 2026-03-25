package tracking

import (
	"context"
	"fmt"
	"time"

	"ondapile/internal/email"
	"ondapile/internal/model"
)

// TrackingStore wraps email store for tracking data updates.
type TrackingStore struct {
	emailStore *email.EmailStore
}

// NewTrackingStore creates a new tracking store.
func NewTrackingStore(emailStore *email.EmailStore) *TrackingStore {
	return &TrackingStore{
		emailStore: emailStore,
	}
}

// RecordOpen records an email open event.
// It increments the opens count and sets first_opened_at if not already set.
func (s *TrackingStore) RecordOpen(ctx context.Context, emailID string) error {
	// Get the current email to check existing tracking data
	email, err := s.emailStore.GetEmail(ctx, emailID)
	if err != nil {
		return fmt.Errorf("failed to get email for tracking: %w", err)
	}
	if email == nil {
		return fmt.Errorf("email not found: %s", emailID)
	}

	// Initialize tracking if nil
	if email.Tracking == nil {
		email.Tracking = &model.EmailTracking{
			Opens:        0,
			Clicks:       0,
			LinksClicked: []model.ClickedLink{},
		}
	}

	// Increment opens
	email.Tracking.Opens++

	// Set first_opened_at if not already set
	if email.Tracking.FirstOpenedAt == nil {
		now := time.Now().UTC()
		email.Tracking.FirstOpenedAt = &now
	}

	// Update the email in the database
	return s.updateTracking(ctx, email)
}

// RecordClick records an email link click event.
// It increments the clicks count and appends to links_clicked array.
func (s *TrackingStore) RecordClick(ctx context.Context, emailID string, url string) error {
	// Get the current email to check existing tracking data
	email, err := s.emailStore.GetEmail(ctx, emailID)
	if err != nil {
		return fmt.Errorf("failed to get email for tracking: %w", err)
	}
	if email == nil {
		return fmt.Errorf("email not found: %s", emailID)
	}

	// Initialize tracking if nil
	if email.Tracking == nil {
		email.Tracking = &model.EmailTracking{
			Opens:        0,
			Clicks:       0,
			LinksClicked: []model.ClickedLink{},
		}
	}

	// Increment clicks
	email.Tracking.Clicks++

	// Append to links_clicked
	email.Tracking.LinksClicked = append(email.Tracking.LinksClicked, model.ClickedLink{
		URL:       url,
		ClickedAt: time.Now().UTC(),
	})

	// Update the email in the database
	return s.updateTracking(ctx, email)
}

// updateTracking updates the tracking column for an email.
func (s *TrackingStore) updateTracking(ctx context.Context, email *model.Email) error {
	// StoreEmail does an UPSERT, so it will update the existing record
	return s.emailStore.StoreEmail(ctx, email)
}

// GetEmailTracking retrieves the tracking data for an email.
func (s *TrackingStore) GetEmailTracking(ctx context.Context, emailID string) (*model.EmailTracking, error) {
	email, err := s.emailStore.GetEmail(ctx, emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get email: %w", err)
	}
	if email == nil {
		return nil, fmt.Errorf("email not found: %s", emailID)
	}

	if email.Tracking == nil {
		return &model.EmailTracking{
			Opens:        0,
			Clicks:       0,
			LinksClicked: []model.ClickedLink{},
		}, nil
	}

	return email.Tracking, nil
}
