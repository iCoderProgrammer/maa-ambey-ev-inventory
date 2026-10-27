package attachments

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
)

// PGStore is the PostgreSQL Repository.
type PGStore struct {
	pool *pgxpool.Pool
}

// NewPGStore returns a Repository backed by pool.
func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

const attachmentColumns = `id::text, object_key, storage_driver, content_type, size_bytes, sha256, original_name,
	uploaded_by::text, COALESCE(location_id::text, ''), COALESCE(owner_type, ''), COALESCE(owner_id, ''), created_at`

func scanAttachment(row pgx.Row) (Attachment, error) {
	var a Attachment
	err := row.Scan(&a.ID, &a.ObjectKey, &a.StorageDriver, &a.ContentType, &a.SizeBytes, &a.SHA256, &a.OriginalName,
		&a.UploadedBy, &a.LocationID, &a.OwnerType, &a.OwnerID, &a.CreatedAt)
	return a, err
}

// Insert implements Repository.
func (s *PGStore) Insert(ctx context.Context, a Attachment, ev audit.Event) (Attachment, error) {
	var saved Attachment
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		saved, err = scanAttachment(tx.QueryRow(ctx, `
			INSERT INTO attachments (id, object_key, storage_driver, content_type, size_bytes, sha256, original_name,
			                         uploaded_by, location_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, '')::uuid)
			RETURNING `+attachmentColumns,
			a.ID, a.ObjectKey, a.StorageDriver, a.ContentType, a.SizeBytes, a.SHA256, a.OriginalName, a.UploadedBy, a.LocationID))
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" && pgErr.ConstraintName == "attachments_location_id_fkey" {
				return ErrInvalidLocation
			}
			return fmt.Errorf("insert attachment: %w", err)
		}
		return audit.Record(ctx, tx, ev)
	})
	return saved, err
}

// Get implements Repository.
func (s *PGStore) Get(ctx context.Context, id string) (Attachment, error) {
	a, err := scanAttachment(s.pool.QueryRow(ctx, `SELECT `+attachmentColumns+` FROM attachments WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Attachment{}, ErrNotFound
	}
	if err != nil {
		return Attachment{}, fmt.Errorf("get attachment: %w", err)
	}
	return a, nil
}

// ListUnlinked implements Repository.
func (s *PGStore) ListUnlinked(ctx context.Context, uploaderID string, limit int) ([]Attachment, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+attachmentColumns+` FROM attachments
		WHERE uploaded_by = $1 AND owner_type IS NULL ORDER BY created_at DESC, id LIMIT $2`, uploaderID, limit)
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}
	defer rows.Close()
	out := []Attachment{}
	for rows.Next() {
		a, err := scanAttachment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan attachment: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// RecordAudit implements Repository.
func (s *PGStore) RecordAudit(ctx context.Context, ev audit.Event) error {
	return audit.Record(ctx, s.pool, ev)
}
