package search

import (
	"context"
	"encoding/json"
	"fmt"

	"ondapile/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

// SearchService provides semantic search over messages.
type SearchService struct {
	pool *pgxpool.Pool
}

// NewSearchService creates a new SearchService.
func NewSearchService(pool *pgxpool.Pool) *SearchService {
	return &SearchService{pool: pool}
}

// SearchMessages performs semantic search using cosine similarity.
// Returns messages ordered by similarity to the query embedding.
func (s *SearchService) SearchMessages(ctx context.Context, queryEmbedding []float32, accountID *string, limit int) ([]*model.Message, error) {
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	vector := pgvector.NewVector(queryEmbedding)

	// Build query with optional account filter
	whereClause := "WHERE hidden = false AND embedding IS NOT NULL"
	args := []any{vector, limit + 1}
	argN := 3

	if accountID != nil && *accountID != "" {
		whereClause += " AND account_id = $" + itoa(argN)
		args = append(args, *accountID)
		argN++
	}

	q := `SELECT id, chat_id, account_id, provider, provider_id, text, sender_id, is_sender,
	            timestamp, attachments, reactions, quoted, seen, delivered, edited, deleted,
	            hidden, is_event, event_type, metadata
	      FROM messages ` + whereClause + `
	      ORDER BY embedding <=> $1
	      LIMIT $2`

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		json.Unmarshal(attachJSON, &m.Attachments)
		json.Unmarshal(reactJSON, &m.Reactions)
		m.Object = "message"
		messages = append(messages, &m)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// GetByID retrieves a message by ID with embedding support.
func (s *SearchService) GetByID(ctx context.Context, id string) (*model.Message, error) {
	q := `SELECT id, chat_id, account_id, provider, provider_id, text, sender_id, is_sender,
	            timestamp, attachments, reactions, quoted, seen, delivered, edited, deleted,
	            hidden, is_event, event_type, metadata
	      FROM messages WHERE id = $1`

	var m model.Message
	var attachJSON, reactJSON []byte
	err := s.pool.QueryRow(ctx, q, id).Scan(
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

// UpdateEmbedding stores the embedding vector for a message.
func (s *SearchService) UpdateEmbedding(ctx context.Context, messageID string, embedding []float32) error {
	vector := pgvector.NewVector(embedding)
	q := `UPDATE messages SET embedding = $1 WHERE id = $2`
	_, err := s.pool.Exec(ctx, q, vector, messageID)
	return err
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
