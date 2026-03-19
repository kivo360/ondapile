package store

import (
	"context"
	"encoding/json"
	"fmt"

	"ondapile/internal/model"

	"github.com/jackc/pgx/v5"
)

type CalendarStore struct {
	s *Store
}

func NewCalendarStore(s *Store) *CalendarStore {
	return &CalendarStore{s: s}
}

func (cs *CalendarStore) Create(ctx context.Context, calendar *model.Calendar) (*model.Calendar, error) {
	meta, _ := json.Marshal(calendar.Metadata)

	q := `INSERT INTO calendars (account_id, provider, provider_id, name, color, is_primary, is_read_only, timezone, metadata)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	      RETURNING id, account_id, provider, provider_id, name, color, is_primary, is_read_only, timezone, metadata, created_at, updated_at`

	var c model.Calendar
	err := cs.s.Pool.QueryRow(ctx, q,
		calendar.AccountID, calendar.Provider, calendar.ProviderID, calendar.Name,
		calendar.Color, calendar.IsPrimary, calendar.IsReadOnly, calendar.TimeZone, meta,
	).Scan(&c.ID, &c.AccountID, &c.Provider, &c.ProviderID, &c.Name, &c.Color,
		&c.IsPrimary, &c.IsReadOnly, &c.TimeZone, &c.Metadata, &c.CreatedAt, &c.UpdatedAt)

	if err != nil {
		return nil, err
	}
	c.Object = "calendar"
	return &c, nil
}

func (cs *CalendarStore) GetByID(ctx context.Context, id string) (*model.Calendar, error) {
	q := `SELECT id, account_id, provider, provider_id, name, color, is_primary, is_read_only, timezone, metadata, created_at, updated_at
	      FROM calendars WHERE id = $1`

	var c model.Calendar
	err := cs.s.Pool.QueryRow(ctx, q, id).Scan(
		&c.ID, &c.AccountID, &c.Provider, &c.ProviderID, &c.Name, &c.Color,
		&c.IsPrimary, &c.IsReadOnly, &c.TimeZone, &c.Metadata, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.Object = "calendar"
	return &c, nil
}

func (cs *CalendarStore) List(ctx context.Context, accountID *string, cursor string, limit int) ([]*model.Calendar, string, bool, error) {
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	baseWhere := "WHERE 1=1"
	args := []any{}
	argN := 1

	if accountID != nil && *accountID != "" {
		baseWhere += fmt.Sprintf(" AND account_id = $%d", argN)
		args = append(args, *accountID)
		argN++
	}
	if cursor != "" {
		baseWhere += fmt.Sprintf(" AND id > $%d", argN)
		args = append(args, cursor)
		argN++
	}

	q := fmt.Sprintf(`SELECT id, account_id, provider, provider_id, name, color, is_primary, is_read_only, timezone, metadata, created_at, updated_at
	                  FROM calendars %s ORDER BY id LIMIT $%d`, baseWhere, argN)
	args = append(args, limit+1)

	rows, err := cs.s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", false, err
	}
	defer rows.Close()

	var calendars []*model.Calendar
	for rows.Next() {
		var c model.Calendar
		if err := rows.Scan(&c.ID, &c.AccountID, &c.Provider, &c.ProviderID, &c.Name, &c.Color,
			&c.IsPrimary, &c.IsReadOnly, &c.TimeZone, &c.Metadata, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, "", false, err
		}
		c.Object = "calendar"
		calendars = append(calendars, &c)
	}

	hasMore := len(calendars) > limit
	if hasMore {
		calendars = calendars[:limit]
	}

	var nextCursor string
	if hasMore && len(calendars) > 0 {
		nextCursor = calendars[len(calendars)-1].ID
	}

	return calendars, nextCursor, hasMore, nil
}

func (cs *CalendarStore) GetByProviderID(ctx context.Context, accountID, providerID string) (*model.Calendar, error) {
	q := `SELECT id, account_id, provider, provider_id, name, color, is_primary, is_read_only, timezone, metadata, created_at, updated_at
	      FROM calendars WHERE account_id = $1 AND provider_id = $2`

	var c model.Calendar
	err := cs.s.Pool.QueryRow(ctx, q, accountID, providerID).Scan(
		&c.ID, &c.AccountID, &c.Provider, &c.ProviderID, &c.Name, &c.Color,
		&c.IsPrimary, &c.IsReadOnly, &c.TimeZone, &c.Metadata, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.Object = "calendar"
	return &c, nil
}

func (cs *CalendarStore) Update(ctx context.Context, calendar *model.Calendar) error {
	meta, _ := json.Marshal(calendar.Metadata)
	q := `UPDATE calendars SET name = $1, color = $2, is_primary = $3, is_read_only = $4, timezone = $5, metadata = $6, updated_at = NOW()
	      WHERE id = $7`
	_, err := cs.s.Pool.Exec(ctx, q, calendar.Name, calendar.Color, calendar.IsPrimary, calendar.IsReadOnly, calendar.TimeZone, meta, calendar.ID)
	return err
}

func (cs *CalendarStore) Delete(ctx context.Context, id string) error {
	q := `DELETE FROM calendars WHERE id = $1`
	_, err := cs.s.Pool.Exec(ctx, q, id)
	return err
}
