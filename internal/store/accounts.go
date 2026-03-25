package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ondapile/internal/model"

	"github.com/jackc/pgx/v5"
)

type AccountStore struct {
	s *Store
}

func NewAccountStore(s *Store) *AccountStore {
	return &AccountStore{s: s}
}

type CreateAccountParams struct {
	Provider       string
	Name           string
	Identifier     string
	Status         string
	Capabilities   []string
	Proxy          *model.ProxyConfig
	Metadata       map[string]any
	OrganizationID string
}

func (as *AccountStore) Create(ctx context.Context, p CreateAccountParams) (*model.Account, error) {
	caps, _ := json.Marshal(p.Capabilities)
	meta, _ := json.Marshal(p.Metadata)
	var proxyJSON []byte
	if p.Proxy != nil {
		proxyJSON, _ = json.Marshal(p.Proxy)
	}

	q := `INSERT INTO accounts (provider, name, identifier, status, capabilities, proxy_config, metadata, organization_id)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	      RETURNING id, provider, name, identifier, status, status_detail, capabilities, created_at, updated_at, last_synced_at, metadata, proxy_config`

	var a model.Account
	var proxyData []byte
	err := as.s.Pool.QueryRow(ctx, q,
		p.Provider, p.Name, p.Identifier, p.Status, caps, proxyJSON, meta, p.OrganizationID,
	).Scan(&a.ID, &a.Provider, &a.Name, &a.Identifier, &a.Status,
		&a.StatusDetail, &a.Capabilities, &a.CreatedAt, &a.UpdatedAt,
		&a.LastSyncedAt, &a.Metadata, &proxyData)

	if err != nil {
		return nil, err
	}

	if len(proxyData) > 0 {
		json.Unmarshal(proxyData, &a.Proxy)
	}
	a.Object = "account"
	return &a, nil
}

func (as *AccountStore) GetByID(ctx context.Context, id string) (*model.Account, error) {
	q := `SELECT id, provider, name, identifier, status, status_detail, capabilities,
	            created_at, updated_at, last_synced_at, metadata, proxy_config
	      FROM accounts WHERE id = $1`

	var a model.Account
	var proxyData []byte
	err := as.s.Pool.QueryRow(ctx, q, id).Scan(
		&a.ID, &a.Provider, &a.Name, &a.Identifier, &a.Status,
		&a.StatusDetail, &a.Capabilities, &a.CreatedAt, &a.UpdatedAt,
		&a.LastSyncedAt, &a.Metadata, &proxyData,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if len(proxyData) > 0 {
		json.Unmarshal(proxyData, &a.Proxy)
	}
	a.Object = "account"
	return &a, nil
}

func (as *AccountStore) GetByIDAndOrg(ctx context.Context, id, organizationID string) (*model.Account, error) {
	q := `SELECT id, provider, name, identifier, status, status_detail, capabilities,
	            created_at, updated_at, last_synced_at, metadata, proxy_config
	      FROM accounts WHERE id = $1 AND organization_id = $2`

	var a model.Account
	var proxyData []byte
	err := as.s.Pool.QueryRow(ctx, q, id, organizationID).Scan(
		&a.ID, &a.Provider, &a.Name, &a.Identifier, &a.Status,
		&a.StatusDetail, &a.Capabilities, &a.CreatedAt, &a.UpdatedAt,
		&a.LastSyncedAt, &a.Metadata, &proxyData,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(proxyData) > 0 {
		json.Unmarshal(proxyData, &a.Proxy)
	}
	a.Object = "account"
	return &a, nil
}

func (as *AccountStore) List(ctx context.Context, provider *string, status *string, cursor string, limit int) ([]*model.Account, string, bool, error) {
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	baseWhere := "WHERE 1=1"
	args := []any{}
	argN := 1

	if provider != nil && *provider != "" {
		baseWhere += fmt.Sprintf(" AND provider = $%d", argN)
		args = append(args, *provider)
		argN++
	}
	if status != nil && *status != "" {
		baseWhere += fmt.Sprintf(" AND status = $%d", argN)
		args = append(args, *status)
		argN++
	}
	if cursor != "" {
		baseWhere += fmt.Sprintf(" AND id > $%d", argN)
		args = append(args, cursor)
		argN++
	}

	q := fmt.Sprintf(`SELECT id, provider, name, identifier, status, status_detail, capabilities,
	                         created_at, updated_at, last_synced_at, metadata, proxy_config
	                  FROM accounts %s ORDER BY id LIMIT $%d`, baseWhere, argN)
	args = append(args, limit+1)

	rows, err := as.s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", false, err
	}
	defer rows.Close()

	var accounts []*model.Account
	for rows.Next() {
		var a model.Account
		var proxyData []byte
		if err := rows.Scan(&a.ID, &a.Provider, &a.Name, &a.Identifier, &a.Status,
			&a.StatusDetail, &a.Capabilities, &a.CreatedAt, &a.UpdatedAt,
			&a.LastSyncedAt, &a.Metadata, &proxyData); err != nil {
			return nil, "", false, err
		}
		if len(proxyData) > 0 {
			json.Unmarshal(proxyData, &a.Proxy)
		}
		a.Object = "account"
		accounts = append(accounts, &a)
	}

	hasMore := len(accounts) > limit
	if hasMore {
		accounts = accounts[:limit]
	}

	var nextCursor string
	if hasMore && len(accounts) > 0 {
		nextCursor = accounts[len(accounts)-1].ID
	}

	return accounts, nextCursor, hasMore, nil
}

func (as *AccountStore) UpdateStatus(ctx context.Context, id string, status model.AccountStatus, detail *string) error {
	q := `UPDATE accounts SET status = $1, status_detail = $2, updated_at = NOW() WHERE id = $3`
	_, err := as.s.Pool.Exec(ctx, q, status, detail, id)
	return err
}

func (as *AccountStore) UpdateSyncedAt(ctx context.Context, id string) error {
	q := `UPDATE accounts SET last_synced_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := as.s.Pool.Exec(ctx, q, id)
	return err
}

func (as *AccountStore) Delete(ctx context.Context, id string) error {
	q := `DELETE FROM accounts WHERE id = $1`
	_, err := as.s.Pool.Exec(ctx, q, id)
	return err
}

func (as *AccountStore) DeleteByIDAndOrg(ctx context.Context, id, organizationID string) error {
	q := `DELETE FROM accounts WHERE id = $1 AND organization_id = $2`
	_, err := as.s.Pool.Exec(ctx, q, id, organizationID)
	return err
}

func (as *AccountStore) ListByStatus(ctx context.Context, status model.AccountStatus) ([]*model.Account, error) {
	q := `SELECT id, provider, name, identifier, status, status_detail, capabilities,
	            created_at, updated_at, last_synced_at, metadata, proxy_config
	      FROM accounts WHERE status = $1`

	rows, err := as.s.Pool.Query(ctx, q, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*model.Account
	for rows.Next() {
		var a model.Account
		var proxyData []byte
		if err := rows.Scan(&a.ID, &a.Provider, &a.Name, &a.Identifier, &a.Status,
			&a.StatusDetail, &a.Capabilities, &a.CreatedAt, &a.UpdatedAt,
			&a.LastSyncedAt, &a.Metadata, &proxyData); err != nil {
			return nil, err
		}
		a.Object = "account"
		if len(proxyData) > 0 {
			json.Unmarshal(proxyData, &a.Proxy)
		}
		accounts = append(accounts, &a)
	}

	return accounts, nil
}

func (as *AccountStore) GetByProviderIdentifier(ctx context.Context, provider, identifier string) (*model.Account, error) {
	q := `SELECT id, provider, name, identifier, status, status_detail, capabilities,
	            created_at, updated_at, last_synced_at, metadata, proxy_config
	      FROM accounts WHERE provider = $1 AND identifier = $2`

	var a model.Account
	var proxyData []byte
	err := as.s.Pool.QueryRow(ctx, q, provider, identifier).Scan(
		&a.ID, &a.Provider, &a.Name, &a.Identifier, &a.Status,
		&a.StatusDetail, &a.Capabilities, &a.CreatedAt, &a.UpdatedAt,
		&a.LastSyncedAt, &a.Metadata, &proxyData,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.Object = "account"

	if len(proxyData) > 0 {
		json.Unmarshal(proxyData, &a.Proxy)
	}
	return &a, nil
}

func (as *AccountStore) UpdateMetadata(ctx context.Context, id string, meta map[string]any) error {
	metaJSON, _ := json.Marshal(meta)
	q := `UPDATE accounts SET metadata = $1, updated_at = NOW() WHERE id = $2`
	_, err := as.s.Pool.Exec(ctx, q, metaJSON, id)
	return err
}

func (as *AccountStore) UpdateCredentials(ctx context.Context, id string, credsEnc []byte) error {
	q := `UPDATE accounts SET credentials_enc = $1, updated_at = NOW() WHERE id = $2`
	_, err := as.s.Pool.Exec(ctx, q, credsEnc, id)
	return err
}

// ListForReconnect returns all OPERATIONAL accounts for reconnection on server restart.
func (as *AccountStore) ListForReconnect(ctx context.Context) ([]*model.Account, error) {
	return as.ListByStatus(ctx, model.StatusOperational)
}

// GetCredentialsEnc returns the encrypted credentials for an account.
func (as *AccountStore) GetCredentialsEnc(ctx context.Context, id string) ([]byte, error) {
	q := `SELECT credentials_enc FROM accounts WHERE id = $1`
	var creds []byte
	err := as.s.Pool.QueryRow(ctx, q, id).Scan(&creds)
	if err != nil {
		return nil, err
	}
	return creds, nil
}

// ListByOrganization returns accounts filtered by organization_id.
func (as *AccountStore) ListByOrganization(ctx context.Context, organizationID string) ([]*model.Account, error) {
	q := `SELECT id, provider, name, identifier, status, status_detail, capabilities,
            created_at, updated_at, last_synced_at, metadata, proxy_config
      FROM accounts WHERE organization_id = $1 ORDER BY created_at DESC`

	rows, err := as.s.Pool.Query(ctx, q, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*model.Account
	for rows.Next() {
		var a model.Account
		var proxyData []byte
		if err := rows.Scan(&a.ID, &a.Provider, &a.Name, &a.Identifier, &a.Status,
			&a.StatusDetail, &a.Capabilities, &a.CreatedAt, &a.UpdatedAt,
			&a.LastSyncedAt, &a.Metadata, &proxyData); err != nil {
			return nil, err
		}
		if len(proxyData) > 0 {
			json.Unmarshal(proxyData, &a.Proxy)
		}
		a.Object = "account"
		accounts = append(accounts, &a)
	}

	return accounts, nil
}

// mock time usage
var _ = time.Now
