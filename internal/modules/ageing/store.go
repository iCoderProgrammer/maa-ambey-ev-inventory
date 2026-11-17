package ageing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
)

// Errors.
var (
	ErrNotConfigured   = errors.New("no ageing band configuration exists")
	ErrVersionConflict = errors.New("the band configuration changed")
	ErrNotProposed     = errors.New("the version is not the current proposed configuration")
	ErrSelfApproval    = errors.New("the proposer cannot approve their own configuration")
	ErrForbidden       = errors.New("forbidden")
)

// ValidationError lists invalid fields.
type ValidationError struct{ Fields []Problem }

func (e *ValidationError) Error() string { return "invalid band configuration" }

// Repository reads and records band configurations.
type Repository interface {
	// Current returns the latest version (ErrNotConfigured when none).
	Current(ctx context.Context) (BandSet, error)
	// Propose records a new version after expectedVersion (0 when none exists yet).
	Propose(ctx context.Context, actor audit.Actor, requestID string, expectedVersion int, source string, bands []Band) (BandSet, error)
	// Approve approves the current PROPOSED version.
	Approve(ctx context.Context, actor audit.Actor, requestID string, version int, reference *string) (BandSet, error)
	// RecordRefusal audits a refused approval.
	RecordRefusal(ctx context.Context, actor audit.Actor, requestID string, version int, reason string) error
}

// PGStore is the PostgreSQL Repository.
type PGStore struct{ pool *pgxpool.Pool }

// NewPGStore returns a Repository backed by pool.
func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

type querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func current(ctx context.Context, q querier, lock bool) (BandSet, error) {
	var s BandSet
	suffix := ""
	if lock {
		suffix = " FOR UPDATE OF s"
	}
	err := q.QueryRow(ctx, `SELECT s.version, s.status, s.source, s.created_by::text, cu.display_name, s.created_at,
			s.approved_by::text, au.display_name, s.approved_at, s.approval_reference
		FROM stock_ageing_band_sets s
		LEFT JOIN users cu ON cu.id = s.created_by
		LEFT JOIN users au ON au.id = s.approved_by
		ORDER BY s.version DESC LIMIT 1`+suffix).Scan(&s.Version, &s.Status, &s.Source, &s.CreatedByID, &s.CreatedByName,
		&s.CreatedAt, &s.ApprovedByID, &s.ApprovedByName, &s.ApprovedAt, &s.ApprovalReference)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, ErrNotConfigured
	}
	if err != nil {
		return s, fmt.Errorf("read ageing band configuration: %w", err)
	}
	rows, err := q.Query(ctx, `SELECT key, label, min_days, max_days FROM stock_ageing_bands WHERE set_version = $1 ORDER BY position`,
		s.Version)
	if err != nil {
		return s, fmt.Errorf("read ageing bands: %w", err)
	}
	var b Band
	s.Bands = []Band{}
	if _, err := pgx.ForEachRow(rows, []any{&b.Key, &b.Label, &b.MinDays, &b.MaxDays}, func() error {
		band := b
		if b.MaxDays != nil {
			v := *b.MaxDays
			band.MaxDays = &v
		}
		s.Bands = append(s.Bands, band)
		return nil
	}); err != nil {
		return s, fmt.Errorf("read ageing bands: %w", err)
	}
	return s, nil
}

// Current implements Repository.
func (s *PGStore) Current(ctx context.Context) (BandSet, error) { return current(ctx, s.pool, false) }

// Propose implements Repository: the next version, its bands and the audit
// event in one transaction; the current version is locked so two proposals
// cannot both follow it.
func (s *PGStore) Propose(ctx context.Context, actor audit.Actor, requestID string, expectedVersion int, source string,
	bands []Band) (BandSet, error) {
	var out BandSet
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		cur, err := current(ctx, tx, true)
		switch {
		case errors.Is(err, ErrNotConfigured):
			cur.Version = 0
		case err != nil:
			return err
		}
		if cur.Version != expectedVersion {
			return ErrVersionConflict
		}
		next := cur.Version + 1
		if _, err := tx.Exec(ctx, `INSERT INTO stock_ageing_band_sets (version, source, created_by) VALUES ($1, $2, $3)`,
			next, source, actor.UserID); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return ErrVersionConflict
			}
			return fmt.Errorf("record ageing band configuration: %w", err)
		}
		for i, b := range bands {
			if _, err := tx.Exec(ctx, `INSERT INTO stock_ageing_bands (set_version, position, key, label, min_days, max_days)
				VALUES ($1, $2, $3, $4, $5, $6)`, next, i+1, b.Key, b.Label, b.MinDays, b.MaxDays); err != nil {
				return fmt.Errorf("record ageing band: %w", err)
			}
		}
		if err := audit.Record(ctx, tx, audit.Event{Actor: actor, Action: "stock_ageing.bands_proposed", EntityType: "stock_ageing_band_set",
			EntityID: fmt.Sprint(next), RequestID: requestID,
			Summary: audit.Summary{"version": audit.Change{From: cur.Version, To: next}, "bands": len(bands), "source": source}}); err != nil {
			return err
		}
		out, err = current(ctx, tx, false)
		return err
	})
	return out, err
}

// Approve implements Repository: only the current version, only while PROPOSED,
// never by its proposer (also a database CHECK).
func (s *PGStore) Approve(ctx context.Context, actor audit.Actor, requestID string, version int, reference *string) (BandSet, error) {
	var out BandSet
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		cur, err := current(ctx, tx, true)
		if err != nil {
			return err
		}
		if cur.Version != version || cur.Status != SetProposed {
			return ErrNotProposed
		}
		if cur.CreatedByID != nil && *cur.CreatedByID == actor.UserID {
			return ErrSelfApproval
		}
		if _, err := tx.Exec(ctx, `UPDATE stock_ageing_band_sets SET status = 'APPROVED', approved_by = $2, approved_at = $3,
			approval_reference = $4 WHERE version = $1`, version, actor.UserID, time.Now(), reference); err != nil {
			return fmt.Errorf("approve ageing band configuration: %w", err)
		}
		summary := audit.Summary{"status": audit.Change{From: SetProposed, To: SetApproved}}
		if reference != nil {
			summary["reference"] = *reference
		}
		if err := audit.Record(ctx, tx, audit.Event{Actor: actor, Action: "stock_ageing.bands_approved",
			EntityType: "stock_ageing_band_set", EntityID: fmt.Sprint(version), RequestID: requestID, Summary: summary}); err != nil {
			return err
		}
		out, err = current(ctx, tx, false)
		return err
	})
	return out, err
}

// RecordRefusal audits a refused approval (self-approval) outside the refused transaction.
func (s *PGStore) RecordRefusal(ctx context.Context, actor audit.Actor, requestID string, version int, reason string) error {
	return audit.Record(ctx, s.pool, audit.Event{Actor: actor, Action: "stock_ageing.bands_approved", Outcome: audit.OutcomeDenied,
		EntityType: "stock_ageing_band_set", EntityID: fmt.Sprint(version), RequestID: requestID, Summary: audit.Summary{"reason": reason}})
}
