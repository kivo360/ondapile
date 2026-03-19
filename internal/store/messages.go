package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ondapile/internal/model"

	"github.com/jackc/pgx/v5"
)

type MessageStore struct {
	s *Store
}

func NewMessageStore(s *Store) *MessageStore {
	return &MessageStore{s: s}
}

func (ms *MessageStore) Create(ctx context.Context, msg *model.Message) (*model.Message, error) {
	attachJSON, _ := json.Marshal(msg.Attachments)
	reactJSON, _ := json.Marshal(msg.Reactions)
	var quotedJSON []byte
	if msg.Quoted != nil {
		quotedJSON, _ = json.Marshal(msg.Quoted)
	}
	metaJSON, _ := json.Marshal(msg.Metadata)

	q := `INSERT INTO messages (chat_id, account_id, provider, provider_id, text, sender_id, is_sender,
	            timestamp, attachments, reactions, quoted, seen, delivered, edited, deleted, hidden,
	            is_event, event_type, metadata)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	      RETURNING id, created_at`

	var createdID string
	var createdAt time.Time
	err := ms.s.Pool.QueryRow(ctx, q,
		msg.ChatID, msg.AccountID, msg.Provider, msg.ProviderID, msg.Text,
		msg.SenderID, msg.IsSender, msg.Timestamp, attachJSON, reactJSON,
		quotedJSON, msg.Seen, msg.Delivered, msg.Edited, msg.Deleted,
		msg.Hidden, msg.IsEvent, msg.EventType, metaJSON,
	).Scan(&createdID, &createdAt)

	if err != nil {
		return nil, err
	}
	msg.ID = createdID
	msg.Object = "message"
	return msg, nil
}

func (ms *MessageStore) GetByID(ctx context.Context, id string) (*model.Message, error) {
	q := `SELECT id, chat_id, account_id, provider, provider_id, text, sender_id, is_sender,
	            timestamp, attachments, reactions, quoted, seen, delivered, edited, deleted,
	            hidden, is_event, event_type, metadata
	      FROM messages WHERE id = $1`

	var m model.Message
	var attachJSON, reactJSON []byte
	err := ms.s.Pool.QueryRow(ctx, q, id).Scan(
		&m.ID, &m.ChatID, &m.AccountID, &m.Provider, &m.ProviderID, &m.Text,
		&m.SenderID, &m.IsSender, &m.Timestamp, &attachJSON, &reactJSON,
		&m.Quoted, &m.Seen, &m.Delivered, &m.Edited, &m.Deleted,
		&m.Hidden, &m.IsEvent, &m.EventType, &m.Metadata,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(attachJSON, &m.Attachments)
	json.Unmarshal(reactJSON, &m.Reactions)
	m.Object = "message"
	return &m, nil
}

func (ms *MessageStore) GetByProviderID(ctx context.Context, accountID, providerID string) (*model.Message, error) {
	q := `SELECT id, chat_id, account_id, provider, provider_id, text, sender_id, is_sender,
	            timestamp, attachments, reactions, quoted, seen, delivered, edited, deleted,
	            hidden, is_event, event_type, metadata
	      FROM messages WHERE account_id = $1 AND provider_id = $2`

	var m model.Message
	var attachJSON, reactJSON []byte
	err := ms.s.Pool.QueryRow(ctx, q, accountID, providerID).Scan(
		&m.ID, &m.ChatID, &m.AccountID, &m.Provider, &m.ProviderID, &m.Text,
		&m.SenderID, &m.IsSender, &m.Timestamp, &attachJSON, &reactJSON,
		&m.Quoted, &m.Seen, &m.Delivered, &m.Edited, &m.Deleted,
		&m.Hidden, &m.IsEvent, &m.EventType, &m.Metadata,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(attachJSON, &m.Attachments)
	json.Unmarshal(reactJSON, &m.Reactions)
	m.Object = "message"
	return &m, nil
}

func (ms *MessageStore) ListByChat(ctx context.Context, chatID string, cursor string, limit int) ([]*model.Message, string, bool, error) {
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	where := "WHERE chat_id = $1"
	args := []any{chatID}
	argN := 2

	if cursor != "" {
		where += " AND timestamp < $" + itoa(argN)
		args = append(args, cursor)
		argN++
	}

	q := "SELECT id, chat_id, account_id, provider, provider_id, text, sender_id, is_sender, " +
		"timestamp, attachments, reactions, quoted, seen, delivered, edited, deleted, " +
		"hidden, is_event, event_type, metadata " +
		"FROM messages " + where + " ORDER BY timestamp DESC LIMIT $" + itoa(argN)
	args = append(args, limit+1)

	rows, err := ms.s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", false, err
	}
	defer rows.Close()

	var messages []*model.Message
	for rows.Next() {
		var m model.Message
		var attachJSON, reactJSON []byte
		if err := rows.Scan(&m.ID, &m.ChatID, &m.AccountID, &m.Provider, &m.ProviderID, &m.Text,
			&m.SenderID, &m.IsSender, &m.Timestamp, &attachJSON, &reactJSON,
			&m.Quoted, &m.Seen, &m.Delivered, &m.Edited, &m.Deleted,
			&m.Hidden, &m.IsEvent, &m.EventType, &m.Metadata); err != nil {
			return nil, "", false, err
		}
		json.Unmarshal(attachJSON, &m.Attachments)
		json.Unmarshal(reactJSON, &m.Reactions)
		m.Object = "message"
		messages = append(messages, &m)
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	var nextCursor string
	if hasMore && len(messages) > 0 {
		nextCursor = messages[len(messages)-1].Timestamp.Format(time.RFC3339Nano)
	}

	return messages, nextCursor, hasMore, nil
}

func (ms *MessageStore) List(ctx context.Context, accountID *string, cursor string, limit int) ([]*model.Message, string, bool, error) {
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	where := "WHERE hidden = false"
	args := []any{}
	argN := 1

	if accountID != nil && *accountID != "" {
		where += " AND account_id = $" + itoa(argN)
		args = append(args, *accountID)
		argN++
	}
	if cursor != "" {
		where += " AND timestamp < $" + itoa(argN)
		args = append(args, cursor)
		argN++
	}

	q := "SELECT id, chat_id, account_id, provider, provider_id, text, sender_id, is_sender, " +
		"timestamp, attachments, reactions, quoted, seen, delivered, edited, deleted, " +
		"hidden, is_event, event_type, metadata " +
		"FROM messages " + where + " ORDER BY timestamp DESC LIMIT $" + itoa(argN)
	args = append(args, limit+1)

	rows, err := ms.s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", false, err
	}
	defer rows.Close()

	var messages []*model.Message
	for rows.Next() {
		var m model.Message
		var attachJSON, reactJSON []byte
		if err := rows.Scan(&m.ID, &m.ChatID, &m.AccountID, &m.Provider, &m.ProviderID, &m.Text,
			&m.SenderID, &m.IsSender, &m.Timestamp, &attachJSON, &reactJSON,
			&m.Quoted, &m.Seen, &m.Delivered, &m.Edited, &m.Deleted,
			&m.Hidden, &m.IsEvent, &m.EventType, &m.Metadata); err != nil {
			return nil, "", false, err
		}
		json.Unmarshal(attachJSON, &m.Attachments)
		json.Unmarshal(reactJSON, &m.Reactions)
		m.Object = "message"
		messages = append(messages, &m)
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	var nextCursor string
	if hasMore && len(messages) > 0 {
		nextCursor = messages[len(messages)-1].Timestamp.Format(time.RFC3339Nano)
	}

	return messages, nextCursor, hasMore, nil
}

func (ms *MessageStore) UpdateReadStatus(ctx context.Context, messageID string, seen bool) error {
q := `UPDATE messages SET seen = $1 WHERE id = $2`
_, err := ms.s.Pool.Exec(ctx, q, seen, messageID)
return err
}

// Delete deletes a message from the database.
func (ms *MessageStore) Delete(ctx context.Context, id string) error {
	q := `DELETE FROM messages WHERE id = $1`
	_, err := ms.s.Pool.Exec(ctx, q, id)
	return err
}

// ListBySender lists messages from a specific sender.
func (ms *MessageStore) ListBySender(ctx context.Context, accountID, senderID string, cursor string, limit int) ([]*model.Message, string, bool, error) {
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	where := "WHERE account_id = $1 AND sender_id = $2 AND hidden = false"
	args := []any{accountID, senderID}
	argN := 3

	if cursor != "" {
		where += " AND timestamp < $" + itoa(argN)
		args = append(args, cursor)
		argN++
	}

	q := "SELECT id, chat_id, account_id, provider, provider_id, text, sender_id, is_sender, " +
		"timestamp, attachments, reactions, quoted, seen, delivered, edited, deleted, " +
		"hidden, is_event, event_type, metadata " +
		"FROM messages " + where + " ORDER BY timestamp DESC LIMIT $" + itoa(argN)
	args = append(args, limit+1)

	rows, err := ms.s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", false, err
	}
	defer rows.Close()

	var messages []*model.Message
	for rows.Next() {
		var m model.Message
		var attachJSON, reactJSON []byte
		if err := rows.Scan(&m.ID, &m.ChatID, &m.AccountID, &m.Provider, &m.ProviderID, &m.Text,
			&m.SenderID, &m.IsSender, &m.Timestamp, &attachJSON, &reactJSON,
			&m.Quoted, &m.Seen, &m.Delivered, &m.Edited, &m.Deleted,
			&m.Hidden, &m.IsEvent, &m.EventType, &m.Metadata); err != nil {
			return nil, "", false, err
		}
		json.Unmarshal(attachJSON, &m.Attachments)
		json.Unmarshal(reactJSON, &m.Reactions)
		m.Object = "message"
		messages = append(messages, &m)
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	var nextCursor string
	if hasMore && len(messages) > 0 {
		nextCursor = messages[len(messages)-1].Timestamp.Format(time.RFC3339Nano)
	}

	return messages, nextCursor, hasMore, nil
}

var _ = fmt.Sprintf
