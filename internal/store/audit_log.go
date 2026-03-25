package store

import (
	"context"
	"encoding/json"
	"time"
)

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	ID             int64           `json:"id"`
	OrganizationID string          `json:"organization_id"`
	ActorID        string          `json:"actor_id"`
	ActorName      string          `json:"actor_name,omitempty"`
	Action         string          `json:"action"`
	ResourceType   string          `json:"resource_type,omitempty"`
	ResourceID     string          `json:"resource_id,omitempty"`
	Detail         json.RawMessage `json:"detail"`
	CreatedAt      time.Time       `json:"created_at"`
}

// AuditLogStore provides access to the audit_log table
type AuditLogStore struct {
	s *Store
}

// NewAuditLogStore creates a new AuditLogStore
func NewAuditLogStore(s *Store) *AuditLogStore {
	return &AuditLogStore{s: s}
}

// List returns audit log entries for an organization, ordered by creation time (newest first)
func (a *AuditLogStore) List(ctx context.Context, orgID string, limit int) ([]AuditEntry, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := a.s.Pool.Query(ctx,
		`SELECT id, organization_id, actor_id, actor_name, action, resource_type, resource_id, detail, created_at
		 FROM audit_log 
		 WHERE organization_id = $1 
		 ORDER BY created_at DESC 
		 LIMIT $2`,
		orgID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(
			&e.ID, &e.OrganizationID, &e.ActorID, &e.ActorName, &e.Action,
			&e.ResourceType, &e.ResourceID, &e.Detail, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}

	return entries, nil
}

// Create inserts a new audit log entry
func (a *AuditLogStore) Create(ctx context.Context, entry AuditEntry) error {
	_, err := a.s.Pool.Exec(ctx,
		`INSERT INTO audit_log (organization_id, actor_id, actor_name, action, resource_type, resource_id, detail)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		entry.OrganizationID, entry.ActorID, entry.ActorName, entry.Action,
		entry.ResourceType, entry.ResourceID, entry.Detail)
	return err
}
