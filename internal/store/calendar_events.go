package store

import (
	"context"
	"encoding/json"
	"fmt"

	"ondapile/internal/model"

	"github.com/jackc/pgx/v5"
)

type CalendarEventStore struct {
	s *Store
}

func NewCalendarEventStore(s *Store) *CalendarEventStore {
	return &CalendarEventStore{s: s}
}

func (es *CalendarEventStore) Create(ctx context.Context, event *model.CalendarEvent) (*model.CalendarEvent, error) {
	attendees, _ := json.Marshal(event.Attendees)
	reminders, _ := json.Marshal(event.Reminders)
	meta, _ := json.Marshal(event.Metadata)

	q := `INSERT INTO calendar_events (calendar_id, account_id, provider, provider_id, title, description, location, start_at, end_at, all_day, status, attendees, reminders, conference_url, recurrence, metadata)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	      RETURNING id, calendar_id, account_id, provider, provider_id, title, description, location, start_at, end_at, all_day, status, attendees, reminders, conference_url, recurrence, metadata, created_at, updated_at`

	var e model.CalendarEvent
	var attendeesData, remindersData []byte
	err := es.s.Pool.QueryRow(ctx, q,
		event.CalendarID, event.AccountID, event.Provider, event.ProviderID, event.Title,
		event.Description, event.Location, event.StartAt, event.EndAt, event.AllDay, event.Status,
		attendees, reminders, event.ConferenceURL, event.Recurrence, meta,
	).Scan(&e.ID, &e.CalendarID, &e.AccountID, &e.Provider, &e.ProviderID, &e.Title, &e.Description,
		&e.Location, &e.StartAt, &e.EndAt, &e.AllDay, &e.Status, &attendeesData, &remindersData,
		&e.ConferenceURL, &e.Recurrence, &e.Metadata, &e.CreatedAt, &e.UpdatedAt)

	if err != nil {
		return nil, err
	}

	if len(attendeesData) > 0 {
		json.Unmarshal(attendeesData, &e.Attendees)
	}
	if len(remindersData) > 0 {
		json.Unmarshal(remindersData, &e.Reminders)
	}
	e.Object = "calendar_event"
	return &e, nil
}

func (es *CalendarEventStore) GetByID(ctx context.Context, id string) (*model.CalendarEvent, error) {
	q := `SELECT id, calendar_id, account_id, provider, provider_id, title, description, location, start_at, end_at, all_day, status, attendees, reminders, conference_url, recurrence, metadata, created_at, updated_at
	      FROM calendar_events WHERE id = $1`

	var e model.CalendarEvent
	var attendeesData, remindersData []byte
	err := es.s.Pool.QueryRow(ctx, q, id).Scan(
		&e.ID, &e.CalendarID, &e.AccountID, &e.Provider, &e.ProviderID, &e.Title, &e.Description,
		&e.Location, &e.StartAt, &e.EndAt, &e.AllDay, &e.Status, &attendeesData, &remindersData,
		&e.ConferenceURL, &e.Recurrence, &e.Metadata, &e.CreatedAt, &e.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if len(attendeesData) > 0 {
		json.Unmarshal(attendeesData, &e.Attendees)
	}
	if len(remindersData) > 0 {
		json.Unmarshal(remindersData, &e.Reminders)
	}
	e.Object = "calendar_event"
	return &e, nil
}

func (es *CalendarEventStore) ListByCalendar(ctx context.Context, calendarID string, cursor string, limit int) ([]*model.CalendarEvent, string, bool, error) {
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	baseWhere := "WHERE calendar_id = $1"
	args := []any{calendarID}
	argN := 2

	if cursor != "" {
		baseWhere += fmt.Sprintf(" AND id > $%d", argN)
		args = append(args, cursor)
		argN++
	}

	q := fmt.Sprintf(`SELECT id, calendar_id, account_id, provider, provider_id, title, description, location, start_at, end_at, all_day, status, attendees, reminders, conference_url, recurrence, metadata, created_at, updated_at
	                  FROM calendar_events %s ORDER BY start_at, id LIMIT $%d`, baseWhere, argN)
	args = append(args, limit+1)

	rows, err := es.s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", false, err
	}
	defer rows.Close()

	var events []*model.CalendarEvent
	for rows.Next() {
		var e model.CalendarEvent
		var attendeesData, remindersData []byte
		if err := rows.Scan(&e.ID, &e.CalendarID, &e.AccountID, &e.Provider, &e.ProviderID, &e.Title, &e.Description,
			&e.Location, &e.StartAt, &e.EndAt, &e.AllDay, &e.Status, &attendeesData, &remindersData,
			&e.ConferenceURL, &e.Recurrence, &e.Metadata, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, "", false, err
		}
		if len(attendeesData) > 0 {
			json.Unmarshal(attendeesData, &e.Attendees)
		}
		if len(remindersData) > 0 {
			json.Unmarshal(remindersData, &e.Reminders)
		}
		e.Object = "calendar_event"
		events = append(events, &e)
	}

	hasMore := len(events) > limit
	if hasMore {
		events = events[:limit]
	}

	var nextCursor string
	if hasMore && len(events) > 0 {
		nextCursor = events[len(events)-1].ID
	}

	return events, nextCursor, hasMore, nil
}

func (es *CalendarEventStore) GetByProviderID(ctx context.Context, accountID, providerID string) (*model.CalendarEvent, error) {
	q := `SELECT id, calendar_id, account_id, provider, provider_id, title, description, location, start_at, end_at, all_day, status, attendees, reminders, conference_url, recurrence, metadata, created_at, updated_at
	      FROM calendar_events WHERE account_id = $1 AND provider_id = $2`

	var e model.CalendarEvent
	var attendeesData, remindersData []byte
	err := es.s.Pool.QueryRow(ctx, q, accountID, providerID).Scan(
		&e.ID, &e.CalendarID, &e.AccountID, &e.Provider, &e.ProviderID, &e.Title, &e.Description,
		&e.Location, &e.StartAt, &e.EndAt, &e.AllDay, &e.Status, &attendeesData, &remindersData,
		&e.ConferenceURL, &e.Recurrence, &e.Metadata, &e.CreatedAt, &e.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if len(attendeesData) > 0 {
		json.Unmarshal(attendeesData, &e.Attendees)
	}
	if len(remindersData) > 0 {
		json.Unmarshal(remindersData, &e.Reminders)
	}
	e.Object = "calendar_event"
	return &e, nil
}

func (es *CalendarEventStore) Update(ctx context.Context, event *model.CalendarEvent) error {
	attendees, _ := json.Marshal(event.Attendees)
	reminders, _ := json.Marshal(event.Reminders)
	meta, _ := json.Marshal(event.Metadata)

	q := `UPDATE calendar_events SET title = $1, description = $2, location = $3, start_at = $4, end_at = $5, all_day = $6, status = $7, attendees = $8, reminders = $9, conference_url = $10, recurrence = $11, metadata = $12, updated_at = NOW()
	      WHERE id = $13`
	_, err := es.s.Pool.Exec(ctx, q, event.Title, event.Description, event.Location, event.StartAt, event.EndAt,
		event.AllDay, event.Status, attendees, reminders, event.ConferenceURL, event.Recurrence, meta, event.ID)
	return err
}

func (es *CalendarEventStore) Delete(ctx context.Context, id string) error {
	q := `DELETE FROM calendar_events WHERE id = $1`
	_, err := es.s.Pool.Exec(ctx, q, id)
	return err
}
