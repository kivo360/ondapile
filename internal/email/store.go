package email

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ondapile/internal/model"
	"ondapile/internal/store"

	"github.com/jackc/pgx/v5"
)

// EmailStore handles persistence for email data.
type EmailStore struct {
	s *store.Store
}

// NewEmailStore creates a new email store.
func NewEmailStore(s *store.Store) *EmailStore {
	return &EmailStore{s: s}
}

// StoreEmail stores an email in the database.
func (es *EmailStore) StoreEmail(ctx context.Context, email *model.Email) error {
	// Serialize complex fields
	providerIDJSON, _ := json.Marshal(email.ProviderID)
	fromJSON, _ := json.Marshal(email.FromAttendee)
	toJSON, _ := json.Marshal(email.ToAttendees)
	ccJSON, _ := json.Marshal(email.CCAttendees)
	bccJSON, _ := json.Marshal(email.BCCAttendees)
	replyToJSON, _ := json.Marshal(email.ReplyToAttendees)
	attachmentsJSON, _ := json.Marshal(email.Attachments)
	headersJSON, _ := json.Marshal(email.Headers)
	trackingJSON, _ := json.Marshal(email.Tracking)
	metadataJSON, _ := json.Marshal(email.Metadata)

	// Build folders array
	foldersJSON, _ := json.Marshal(email.Folders)

	q := `INSERT INTO emails (
			id, account_id, provider, provider_id, subject, body, body_plain,
			from_attendee, to_attendees, cc_attendees, bcc_attendees, reply_to_attendees,
			date_sent, has_attachments, attachments, folders, role, is_read, read_date,
			is_complete, headers, tracking, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25
		)
		ON CONFLICT (id) DO UPDATE SET
			subject = EXCLUDED.subject,
			body = EXCLUDED.body,
			body_plain = EXCLUDED.body_plain,
			from_attendee = EXCLUDED.from_attendee,
			to_attendees = EXCLUDED.to_attendees,
			cc_attendees = EXCLUDED.cc_attendees,
			bcc_attendees = EXCLUDED.bcc_attendees,
			reply_to_attendees = EXCLUDED.reply_to_attendees,
			has_attachments = EXCLUDED.has_attachments,
			attachments = EXCLUDED.attachments,
			folders = EXCLUDED.folders,
			role = EXCLUDED.role,
			is_read = EXCLUDED.is_read,
			read_date = EXCLUDED.read_date,
			is_complete = EXCLUDED.is_complete,
			headers = EXCLUDED.headers,
			tracking = EXCLUDED.tracking,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING created_at, updated_at`

	var createdAt, updatedAt time.Time
	err := es.s.Pool.QueryRow(ctx, q,
		email.ID, email.AccountID, email.Provider, providerIDJSON, email.Subject,
		email.Body, email.BodyPlain, fromJSON, toJSON, ccJSON, bccJSON, replyToJSON,
		email.Date, email.HasAttachments, attachmentsJSON, foldersJSON, email.Role,
		email.Read, email.ReadDate, email.IsComplete, headersJSON, trackingJSON,
		metadataJSON, time.Now(), time.Now(),
	).Scan(&createdAt, &updatedAt)

	if err != nil {
		return fmt.Errorf("failed to store email: %w", err)
	}

	return nil
}

// GetEmail retrieves an email by ID.
func (es *EmailStore) GetEmail(ctx context.Context, id string) (*model.Email, error) {
	q := `SELECT 
			id, account_id, provider, provider_id, subject, body, body_plain,
			from_attendee, to_attendees, cc_attendees, bcc_attendees, reply_to_attendees,
			date_sent, has_attachments, attachments, folders, role, is_read, read_date,
			is_complete, headers, tracking, metadata
		FROM emails WHERE id = $1`

	var email model.Email
	var providerIDJSON, fromJSON, toJSON, ccJSON, bccJSON, replyToJSON []byte
	var attachmentsJSON, foldersJSON, headersJSON, trackingJSON, metadataJSON []byte

	err := es.s.Pool.QueryRow(ctx, q, id).Scan(
		&email.ID, &email.AccountID, &email.Provider, &providerIDJSON, &email.Subject,
		&email.Body, &email.BodyPlain, &fromJSON, &toJSON, &ccJSON, &bccJSON, &replyToJSON,
		&email.Date, &email.HasAttachments, &attachmentsJSON, &foldersJSON, &email.Role,
		&email.Read, &email.ReadDate, &email.IsComplete, &headersJSON, &trackingJSON,
		&metadataJSON,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get email: %w", err)
	}

	// Deserialize JSON fields
	json.Unmarshal(providerIDJSON, &email.ProviderID)
	json.Unmarshal(fromJSON, &email.FromAttendee)
	json.Unmarshal(toJSON, &email.ToAttendees)
	json.Unmarshal(ccJSON, &email.CCAttendees)
	json.Unmarshal(bccJSON, &email.BCCAttendees)
	json.Unmarshal(replyToJSON, &email.ReplyToAttendees)
	json.Unmarshal(attachmentsJSON, &email.Attachments)
	json.Unmarshal(foldersJSON, &email.Folders)
	json.Unmarshal(headersJSON, &email.Headers)
	json.Unmarshal(trackingJSON, &email.Tracking)
	json.Unmarshal(metadataJSON, &email.Metadata)

	email.Object = "email"
	return &email, nil
}

// GetEmailByProviderID retrieves an email by provider message ID.
func (es *EmailStore) GetEmailByProviderID(ctx context.Context, accountID, messageID string) (*model.Email, error) {
	q := `SELECT 
			id, account_id, provider, provider_id, subject, body, body_plain,
			from_attendee, to_attendees, cc_attendees, bcc_attendees, reply_to_attendees,
			date_sent, has_attachments, attachments, folders, role, is_read, read_date,
			is_complete, headers, tracking, metadata
		FROM emails 
		WHERE account_id = $1 AND provider_id->>'message_id' = $2`

	var email model.Email
	var providerIDJSON, fromJSON, toJSON, ccJSON, bccJSON, replyToJSON []byte
	var attachmentsJSON, foldersJSON, headersJSON, trackingJSON, metadataJSON []byte

	err := es.s.Pool.QueryRow(ctx, q, accountID, messageID).Scan(
		&email.ID, &email.AccountID, &email.Provider, &providerIDJSON, &email.Subject,
		&email.Body, &email.BodyPlain, &fromJSON, &toJSON, &ccJSON, &bccJSON, &replyToJSON,
		&email.Date, &email.HasAttachments, &attachmentsJSON, &foldersJSON, &email.Role,
		&email.Read, &email.ReadDate, &email.IsComplete, &headersJSON, &trackingJSON,
		&metadataJSON,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get email by provider ID: %w", err)
	}

	// Deserialize JSON fields
	json.Unmarshal(providerIDJSON, &email.ProviderID)
	json.Unmarshal(fromJSON, &email.FromAttendee)
	json.Unmarshal(toJSON, &email.ToAttendees)
	json.Unmarshal(ccJSON, &email.CCAttendees)
	json.Unmarshal(bccJSON, &email.BCCAttendees)
	json.Unmarshal(replyToJSON, &email.ReplyToAttendees)
	json.Unmarshal(attachmentsJSON, &email.Attachments)
	json.Unmarshal(foldersJSON, &email.Folders)
	json.Unmarshal(headersJSON, &email.Headers)
	json.Unmarshal(trackingJSON, &email.Tracking)
	json.Unmarshal(metadataJSON, &email.Metadata)

	email.Object = "email"
	return &email, nil
}

// ListEmails retrieves a paginated list of emails for an account and folder.
func (es *EmailStore) ListEmails(ctx context.Context, accountID, folder, cursor string, limit int) ([]model.Email, string, bool, error) {
	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}

	// Build query
	where := "WHERE account_id = $1"
	args := []any{accountID}
	argN := 2

	// Folder filter
	if folder != "" {
		where += fmt.Sprintf(" AND folders @> to_jsonb($%d::text)", argN)
		args = append(args, folder)
		argN++
	}

	// Cursor filter (date-based pagination)
	if cursor != "" {
		where += fmt.Sprintf(" AND date_sent < (SELECT date_sent FROM emails WHERE id = $%d)", argN)
		args = append(args, cursor)
		argN++
	}

	q := fmt.Sprintf(`SELECT 
			id, account_id, provider, provider_id, subject, body, body_plain,
			from_attendee, to_attendees, cc_attendees, bcc_attendees, reply_to_attendees,
			date_sent, has_attachments, attachments, folders, role, is_read, read_date,
			is_complete, headers, tracking, metadata
		FROM emails %s ORDER BY date_sent DESC LIMIT $%d`, where, argN)
	args = append(args, limit+1)

	rows, err := es.s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, "", false, fmt.Errorf("failed to list emails: %w", err)
	}
	defer rows.Close()

	var emails []model.Email
	for rows.Next() {
		var email model.Email
		var providerIDJSON, fromJSON, toJSON, ccJSON, bccJSON, replyToJSON []byte
		var attachmentsJSON, foldersJSON, headersJSON, trackingJSON, metadataJSON []byte

		err := rows.Scan(
			&email.ID, &email.AccountID, &email.Provider, &providerIDJSON, &email.Subject,
			&email.Body, &email.BodyPlain, &fromJSON, &toJSON, &ccJSON, &bccJSON, &replyToJSON,
			&email.Date, &email.HasAttachments, &attachmentsJSON, &foldersJSON, &email.Role,
			&email.Read, &email.ReadDate, &email.IsComplete, &headersJSON, &trackingJSON,
			&metadataJSON,
		)
		if err != nil {
			return nil, "", false, fmt.Errorf("failed to scan email: %w", err)
		}

		// Deserialize JSON fields
		json.Unmarshal(providerIDJSON, &email.ProviderID)
		json.Unmarshal(fromJSON, &email.FromAttendee)
		json.Unmarshal(toJSON, &email.ToAttendees)
		json.Unmarshal(ccJSON, &email.CCAttendees)
		json.Unmarshal(bccJSON, &email.BCCAttendees)
		json.Unmarshal(replyToJSON, &email.ReplyToAttendees)
		json.Unmarshal(attachmentsJSON, &email.Attachments)
		json.Unmarshal(foldersJSON, &email.Folders)
		json.Unmarshal(headersJSON, &email.Headers)
		json.Unmarshal(trackingJSON, &email.Tracking)
		json.Unmarshal(metadataJSON, &email.Metadata)

		email.Object = "email"
		emails = append(emails, email)
	}

	hasMore := len(emails) > limit
	if hasMore {
		emails = emails[:limit]
	}

	var nextCursor string
	if hasMore && len(emails) > 0 {
		nextCursor = emails[len(emails)-1].ID
	}

	return emails, nextCursor, hasMore, nil
}

// UpdateEmailReadStatus updates the read status of an email.
func (es *EmailStore) UpdateEmailReadStatus(ctx context.Context, id string, isRead bool) error {
	var readDate *time.Time
	if isRead {
		now := time.Now()
		readDate = &now
	}

	q := `UPDATE emails SET is_read = $1, read_date = $2, updated_at = NOW() WHERE id = $3`
	_, err := es.s.Pool.Exec(ctx, q, isRead, readDate, id)
	if err != nil {
		return fmt.Errorf("failed to update email read status: %w", err)
	}
	return nil
}

// UpdateEmailFolder moves an email to a different folder.
func (es *EmailStore) UpdateEmailFolder(ctx context.Context, id string, folders []string) error {
	foldersJSON, _ := json.Marshal(folders)

	q := `UPDATE emails SET folders = $1, updated_at = NOW() WHERE id = $2`
	_, err := es.s.Pool.Exec(ctx, q, foldersJSON, id)
	if err != nil {
		return fmt.Errorf("failed to update email folder: %w", err)
	}
	return nil
}

// DeleteEmail permanently deletes an email.
func (es *EmailStore) DeleteEmail(ctx context.Context, id string) error {
	q := `DELETE FROM emails WHERE id = $1`
	_, err := es.s.Pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("failed to delete email: %w", err)
	}
	return nil
}

// GetUnreadCount returns the number of unread emails in a folder.
func (es *EmailStore) GetUnreadCount(ctx context.Context, accountID, folder string) (int, error) {
	q := `SELECT COUNT(*) FROM emails WHERE account_id = $1 AND folders @> to_jsonb($2::text) AND is_read = false`

	var count int
	err := es.s.Pool.QueryRow(ctx, q, accountID, folder).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get unread count: %w", err)
	}
	return count, nil
}

// UpdateEmail updates an email's folder and/or read status.
func (es *EmailStore) UpdateEmail(ctx context.Context, id string, folder *string, read *bool) error {
	// Start with the base query
	q := `UPDATE emails SET updated_at = NOW()`
	args := []any{}
	argN := 1

	if folder != nil {
		q += fmt.Sprintf(", role = $%d, folders = jsonb_build_array($%d::text)", argN, argN)
		args = append(args, *folder)
		argN++
	}

	if read != nil {
		q += fmt.Sprintf(", is_read = $%d, read_date = CASE WHEN $%d::boolean THEN NOW() ELSE NULL END", argN, argN)
		args = append(args, *read)
		argN++
	}

	q += fmt.Sprintf(" WHERE id = $%d", argN)
	args = append(args, id)

	_, err := es.s.Pool.Exec(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("failed to update email: %w", err)
	}
	return nil
}

// FolderCount represents the count info for a folder.
type FolderCount struct {
	Role   string `json:"role"`
	Total  int    `json:"total"`
	Unread int    `json:"unread"`
}

// GetFolderCounts returns the total and unread counts for all folders in an account.
func (es *EmailStore) GetFolderCounts(ctx context.Context, accountID string) (map[string]*FolderCount, error) {
	q := `SELECT role, COUNT(*) as total, COUNT(*) FILTER (WHERE is_read = false) as unread 
	      FROM emails WHERE account_id = $1 GROUP BY role`

	rows, err := es.s.Pool.Query(ctx, q, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get folder counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]*FolderCount)
	for rows.Next() {
		var fc FolderCount
		if err := rows.Scan(&fc.Role, &fc.Total, &fc.Unread); err != nil {
			return nil, fmt.Errorf("failed to scan folder count: %w", err)
		}
		counts[fc.Role] = &fc
	}

	return counts, nil
}

