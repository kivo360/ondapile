package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ondapile/internal/model"

	"github.com/jackc/pgx/v5"
)

type ChatStore struct {
	s *Store
}

func NewChatStore(s *Store) *ChatStore {
	return &ChatStore{s: s}
}

func (cs *ChatStore) Create(ctx context.Context, chat *model.Chat) (*model.Chat, error) {
	meta, _ := json.Marshal(chat.Metadata)

	q := `INSERT INTO chats (account_id, provider, provider_id, type, name, is_group, is_archived, unread_count, metadata)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	      RETURNING id, account_id, provider, provider_id, type, name, is_group, is_archived, unread_count,
	                last_message_at, last_message_preview, created_at, updated_at, metadata`

	var lastMsgAt *time.Time
	var lastMsgPreview *string
	err := cs.s.Pool.QueryRow(ctx, q,
		chat.AccountID, chat.Provider, chat.ProviderID, chat.Type, chat.Name,
		chat.IsGroup, chat.IsArchived, chat.UnreadCount, meta,
	).Scan(&chat.ID, &chat.AccountID, &chat.Provider, &chat.ProviderID, &chat.Type, &chat.Name,
		&chat.IsGroup, &chat.IsArchived, &chat.UnreadCount,
		&lastMsgAt, &lastMsgPreview, &chat.CreatedAt, &chat.UpdatedAt, &chat.Metadata)

	if err != nil {
		return nil, err
	}
	setChatLastMessage(chat, lastMsgAt, lastMsgPreview)
	chat.Object = "chat"
	return chat, nil
}

func (cs *ChatStore) GetByID(ctx context.Context, id string) (*model.Chat, error) {
	q := `SELECT id, account_id, provider, provider_id, type, name, is_group, is_archived, unread_count,
	            last_message_at, last_message_preview, created_at, updated_at, metadata
	      FROM chats WHERE id = $1`

	var c model.Chat
	var lastMsgAt *time.Time
	var lastMsgPreview *string
	err := cs.s.Pool.QueryRow(ctx, q, id).Scan(
		&c.ID, &c.AccountID, &c.Provider, &c.ProviderID, &c.Type, &c.Name,
		&c.IsGroup, &c.IsArchived, &c.UnreadCount,
		&lastMsgAt, &lastMsgPreview, &c.CreatedAt, &c.UpdatedAt, &c.Metadata,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	setChatLastMessage(&c, lastMsgAt, lastMsgPreview)
	c.Object = "chat"
	return &c, nil
}

func (cs *ChatStore) GetByProviderID(ctx context.Context, accountID, providerID string) (*model.Chat, error) {
	q := `SELECT id, account_id, provider, provider_id, type, name, is_group, is_archived, unread_count,
	            last_message_at, last_message_preview, created_at, updated_at, metadata
	      FROM chats WHERE account_id = $1 AND provider_id = $2`

	var c model.Chat
	var lastMsgAt *time.Time
	var lastMsgPreview *string
	err := cs.s.Pool.QueryRow(ctx, q, accountID, providerID).Scan(
		&c.ID, &c.AccountID, &c.Provider, &c.ProviderID, &c.Type, &c.Name,
		&c.IsGroup, &c.IsArchived, &c.UnreadCount,
		&lastMsgAt, &lastMsgPreview, &c.CreatedAt, &c.UpdatedAt, &c.Metadata,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	setChatLastMessage(&c, lastMsgAt, lastMsgPreview)
	c.Object = "chat"
	return &c, nil
}

func (cs *ChatStore) List(ctx context.Context, accountID *string, isGroup *bool, cursor string, limit int) ([]*model.Chat, string, bool, error) {
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	where := "WHERE 1=1"
	args := []any{}
	argN := 1

	if accountID != nil && *accountID != "" {
		where += " AND account_id = $" + itoa(argN)
		args = append(args, *accountID)
		argN++
	}
	if isGroup != nil {
		where += " AND is_group = $" + itoa(argN)
		args = append(args, *isGroup)
		argN++
	}
	if cursor != "" {
		where += " AND updated_at < $" + itoa(argN)
		args = append(args, cursor)
		argN++
	}

	q := "SELECT id, account_id, provider, provider_id, type, name, is_group, is_archived, unread_count, " +
		"last_message_at, last_message_preview, created_at, updated_at, metadata " +
		"FROM chats " + where + " ORDER BY updated_at DESC LIMIT $" + itoa(argN)
	args = append(args, limit+1)

	rows, err := cs.s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", false, err
	}
	defer rows.Close()

	var chats []*model.Chat
	for rows.Next() {
		var c model.Chat
		var lastMsgAt *time.Time
		var lastMsgPreview *string
		if err := rows.Scan(&c.ID, &c.AccountID, &c.Provider, &c.ProviderID, &c.Type, &c.Name,
			&c.IsGroup, &c.IsArchived, &c.UnreadCount,
			&lastMsgAt, &lastMsgPreview, &c.CreatedAt, &c.UpdatedAt, &c.Metadata); err != nil {
			return nil, "", false, err
		}
		setChatLastMessage(&c, lastMsgAt, lastMsgPreview)
		c.Object = "chat"
		chats = append(chats, &c)
	}

	hasMore := len(chats) > limit
	if hasMore {
		chats = chats[:limit]
	}

	var nextCursor string
	if hasMore && len(chats) > 0 {
		nextCursor = chats[len(chats)-1].UpdatedAt.Format(time.RFC3339Nano)
	}

	return chats, nextCursor, hasMore, nil
}

func (cs *ChatStore) UpdateLastMessage(ctx context.Context, chatID string, preview *string) error {
	q := `UPDATE chats SET last_message_at = NOW(), last_message_preview = $1, updated_at = NOW() WHERE id = $2`
	_, err := cs.s.Pool.Exec(ctx, q, preview, chatID)
	return err
}

func (cs *ChatStore) IncrementUnread(ctx context.Context, chatID string) error {
	q := `UPDATE chats SET unread_count = unread_count + 1, updated_at = NOW() WHERE id = $1`
	_, err := cs.s.Pool.Exec(ctx, q, chatID)
	return err
}

func (cs *ChatStore) ResetUnread(ctx context.Context, chatID string) error {
	q := `UPDATE chats SET unread_count = 0, updated_at = NOW() WHERE id = $1`
	_, err := cs.s.Pool.Exec(ctx, q, chatID)
	return err
}

func (cs *ChatStore) Delete(ctx context.Context, id string) error {
q := `DELETE FROM chats WHERE id = $1`
_, err := cs.s.Pool.Exec(ctx, q, id)
	return err
}

// Archive updates the archived status of a chat.
func (cs *ChatStore) Archive(ctx context.Context, chatID string, archived bool) error {
	q := `UPDATE chats SET is_archived = $1, updated_at = NOW() WHERE id = $2`
	_, err := cs.s.Pool.Exec(ctx, q, archived, chatID)
	return err
}

// ListByAttendee lists 1:1 chats where the chat's provider_id matches the attendee's provider_id.
func (cs *ChatStore) ListByAttendee(ctx context.Context, accountID, attendeeProviderID string, cursor string, limit int) ([]*model.Chat, string, bool, error) {
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	where := "WHERE account_id = $1 AND provider_id = $2 AND type = 'ONE_TO_ONE'"
	args := []any{accountID, attendeeProviderID}
	argN := 3

	if cursor != "" {
		where += " AND updated_at < $" + itoa(argN)
		args = append(args, cursor)
		argN++
	}

	q := "SELECT id, account_id, provider, provider_id, type, name, is_group, is_archived, unread_count, " +
		"last_message_at, last_message_preview, created_at, updated_at, metadata " +
		"FROM chats " + where + " ORDER BY updated_at DESC LIMIT $" + itoa(argN)
	args = append(args, limit+1)

	rows, err := cs.s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", false, err
	}
	defer rows.Close()

	var chats []*model.Chat
	for rows.Next() {
		var c model.Chat
		var lastMsgAt *time.Time
		var lastMsgPreview *string
		if err := rows.Scan(&c.ID, &c.AccountID, &c.Provider, &c.ProviderID, &c.Type, &c.Name,
			&c.IsGroup, &c.IsArchived, &c.UnreadCount,
			&lastMsgAt, &lastMsgPreview, &c.CreatedAt, &c.UpdatedAt, &c.Metadata); err != nil {
			return nil, "", false, err
		}
		setChatLastMessage(&c, lastMsgAt, lastMsgPreview)
		c.Object = "chat"
		chats = append(chats, &c)
	}

	hasMore := len(chats) > limit
	if hasMore {
		chats = chats[:limit]
	}

	var nextCursor string
	if hasMore && len(chats) > 0 {
		nextCursor = chats[len(chats)-1].UpdatedAt.Format(time.RFC3339Nano)
	}

	return chats, nextCursor, hasMore, nil
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}

var _ = json.Marshal
var _ = (*model.Chat)(nil)

// setChatLastMessage constructs a MessagePreview from separate DB columns.
func setChatLastMessage(c *model.Chat, lastMsgAt *time.Time, lastMsgPreview *string) {
	if lastMsgAt != nil || lastMsgPreview != nil {
		c.LastMessage = &model.MessagePreview{}
		if lastMsgAt != nil {
			c.LastMessage.Timestamp = *lastMsgAt
		}
		if lastMsgPreview != nil {
			c.LastMessage.Text = *lastMsgPreview
		}
	}
}
