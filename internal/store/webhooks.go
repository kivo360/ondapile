package store

import (
	"context"
	"encoding/json"
	"time"

	"ondapile/internal/model"

	"github.com/jackc/pgx/v5"
)

type WebhookStore struct {
	s *Store
}

func NewWebhookStore(s *Store) *WebhookStore {
	return &WebhookStore{s: s}
}

func (ws *WebhookStore) Create(ctx context.Context, url string, events []string, secret string) (*model.Webhook, error) {
	eventsJSON, _ := json.Marshal(events)

	q := `INSERT INTO webhooks (url, events, secret)
	      VALUES ($1, $2, $3)
	      RETURNING id, url, events, secret, active, created_at`

	var w model.Webhook
	err := ws.s.Pool.QueryRow(ctx, q, url, eventsJSON, secret).Scan(
		&w.ID, &w.URL, &w.Events, &w.Secret, &w.Active, &w.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	w.Object = "webhook"
	return &w, nil
}

// CreateWithOrg creates a webhook with an associated organization_id.
func (ws *WebhookStore) CreateWithOrg(ctx context.Context, url string, events []string, secret, organizationID string) (*model.Webhook, error) {
	eventsJSON, _ := json.Marshal(events)

	q := `INSERT INTO webhooks (url, events, secret, organization_id)
          VALUES ($1, $2, $3, $4)
          RETURNING id, url, events, secret, active, created_at`

	var w model.Webhook
	err := ws.s.Pool.QueryRow(ctx, q, url, eventsJSON, secret, organizationID).Scan(
		&w.ID, &w.URL, &w.Events, &w.Secret, &w.Active, &w.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	w.Object = "webhook"
	return &w, nil
}

func (ws *WebhookStore) GetByID(ctx context.Context, id string) (*model.Webhook, error) {
	q := `SELECT id, url, events, secret, active, created_at FROM webhooks WHERE id = $1`

	var w model.Webhook
	err := ws.s.Pool.QueryRow(ctx, q, id).Scan(
		&w.ID, &w.URL, &w.Events, &w.Secret, &w.Active, &w.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	w.Object = "webhook"
	return &w, nil
}

func (ws *WebhookStore) List(ctx context.Context) ([]*model.Webhook, error) {
	q := `SELECT id, url, events, secret, active, created_at FROM webhooks ORDER BY created_at DESC`

	rows, err := ws.s.Pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []*model.Webhook
	for rows.Next() {
		var w model.Webhook
		if err := rows.Scan(&w.ID, &w.URL, &w.Events, &w.Secret, &w.Active, &w.CreatedAt); err != nil {
			return nil, err
		}
		w.Object = "webhook"
		webhooks = append(webhooks, &w)
	}

	return webhooks, nil
}

func (ws *WebhookStore) ListActiveForEvent(ctx context.Context, event string) ([]*model.Webhook, error) {
	q := `SELECT id, url, events, secret, active, created_at
	      FROM webhooks
	      WHERE active = true AND events @> $1::jsonb`

	rows, err := ws.s.Pool.Query(ctx, q, `["`+event+`"]`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []*model.Webhook
	for rows.Next() {
		var w model.Webhook
		if err := rows.Scan(&w.ID, &w.URL, &w.Events, &w.Secret, &w.Active, &w.CreatedAt); err != nil {
			return nil, err
		}
		w.Object = "webhook"
		webhooks = append(webhooks, &w)
	}

	return webhooks, nil
}

// ListByOrganization returns webhooks filtered by organization_id.
func (ws *WebhookStore) ListByOrganization(ctx context.Context, organizationID string) ([]*model.Webhook, error) {
	q := `SELECT id, url, events, secret, active, created_at FROM webhooks WHERE organization_id = $1 ORDER BY created_at DESC`

	rows, err := ws.s.Pool.Query(ctx, q, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []*model.Webhook
	for rows.Next() {
		var w model.Webhook
		if err := rows.Scan(&w.ID, &w.URL, &w.Events, &w.Secret, &w.Active, &w.CreatedAt); err != nil {
			return nil, err
		}
		w.Object = "webhook"
		webhooks = append(webhooks, &w)
	}

	return webhooks, nil
}

func (ws *WebhookStore) Delete(ctx context.Context, id string) error {
	q := `DELETE FROM webhooks WHERE id = $1`
	_, err := ws.s.Pool.Exec(ctx, q, id)
	return err
}

func (ws *WebhookStore) CreateDelivery(ctx context.Context, webhookID, event string, payload any) (int64, error) {
	payloadJSON, _ := json.Marshal(payload)

	q := `INSERT INTO webhook_deliveries (webhook_id, event, payload)
	      VALUES ($1, $2, $3)
	      RETURNING id`

	var id int64
	err := ws.s.Pool.QueryRow(ctx, q, webhookID, event, payloadJSON).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (ws *WebhookStore) MarkDelivered(ctx context.Context, id int64, statusCode int) error {
	q := `UPDATE webhook_deliveries SET status_code = $1, delivered = true WHERE id = $2`
	_, err := ws.s.Pool.Exec(ctx, q, statusCode, id)
	return err
}

func (ws *WebhookStore) ScheduleRetry(ctx context.Context, id int64, nextRetry time.Time) error {
	q := `UPDATE webhook_deliveries SET attempts = attempts + 1, next_retry = $1 WHERE id = $2`
	_, err := ws.s.Pool.Exec(ctx, q, nextRetry, id)
	return err
}

func (ws *WebhookStore) ListPendingDeliveries(ctx context.Context) ([]*model.WebhookDelivery, error) {
	q := `SELECT id, webhook_id, event, payload, status_code, attempts, next_retry, delivered, created_at
	      FROM webhook_deliveries
	      WHERE delivered = false AND next_retry <= NOW()
	      ORDER BY next_retry ASC
	      LIMIT 100`

	rows, err := ws.s.Pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []*model.WebhookDelivery
	for rows.Next() {
		var d model.WebhookDelivery
		if err := rows.Scan(&d.ID, &d.WebhookID, &d.Event, &d.Payload, &d.StatusCode,
			&d.Attempts, &d.NextRetry, &d.Delivered, &d.CreatedAt); err != nil {
			return nil, err
		}
		deliveries = append(deliveries, &d)
	}

	return deliveries, nil
}

var _ = pgx.ErrNoRows
