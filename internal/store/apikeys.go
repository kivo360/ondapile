package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// ApiKey represents an API key from the Better Auth apikey table.
type ApiKey struct {
	ID          string
	ConfigID    string
	Name        *string
	Start       *string
	ReferenceID string // organizationId
	Prefix      *string
	Key         string // SHA-256 hash
	Enabled     *bool
	ExpiresAt   *time.Time
	LastRequest *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ApiKeyStore provides database operations for API keys.
type ApiKeyStore struct {
	s *Store
}

// NewApiKeyStore creates a new ApiKeyStore.
func NewApiKeyStore(s *Store) *ApiKeyStore {
	return &ApiKeyStore{s: s}
}

// LookupByKeyHash looks up an API key by its SHA-256 hash.
func (a *ApiKeyStore) LookupByKeyHash(ctx context.Context, keyHash string) (*ApiKey, error) {
	q := `SELECT id, "configId", name, start, "referenceId", prefix, key, enabled, "expiresAt", "lastRequest", "createdAt", "updatedAt"
	      FROM apikey WHERE key = $1`

	var apiKey ApiKey
	err := a.s.Pool.QueryRow(ctx, q, keyHash).Scan(
		&apiKey.ID,
		&apiKey.ConfigID,
		&apiKey.Name,
		&apiKey.Start,
		&apiKey.ReferenceID,
		&apiKey.Prefix,
		&apiKey.Key,
		&apiKey.Enabled,
		&apiKey.ExpiresAt,
		&apiKey.LastRequest,
		&apiKey.CreatedAt,
		&apiKey.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &apiKey, nil
}

// UpdateLastUsed updates the lastRequest timestamp for usage tracking.
func (a *ApiKeyStore) UpdateLastUsed(ctx context.Context, id string) error {
	q := `UPDATE apikey SET "lastRequest" = NOW(), "updatedAt" = NOW() WHERE id = $1`
	_, err := a.s.Pool.Exec(ctx, q, id)
	return err
}
