package openingstock

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// preflightProbe runs inside the preflight transaction (tests only: proves the
// transaction is read only).
var preflightProbe func(ctx context.Context, tx pgx.Tx) error

// PreflightSnapshot implements Repository (H-08): the staged rows, their
// preview statuses and the current catalogue, location, supplier and
// identifier lookups, read in one READ ONLY, REPEATABLE READ transaction
// without row locks, so the preflight sees one consistent state and cannot
// write or block a commit.
func (s *PGStore) PreflightSnapshot(ctx context.Context, batchID string, refs Refs) (PreflightSnapshot, error) {
	var snap PreflightSnapshot
	err := pgx.BeginTxFunc(ctx, s.pool, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}, func(tx pgx.Tx) error {
		var err error
		if snap.Stored, err = storedRows(ctx, tx, batchID); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT row_number, status FROM opening_stock_import_rows WHERE batch_id = $1`, batchID)
		if err != nil {
			return fmt.Errorf("read staged statuses: %w", err)
		}
		snap.StagedStatus = map[int]string{}
		var line int
		var status string
		if _, err := pgx.ForEachRow(rows, []any{&line, &status}, func() error { snap.StagedStatus[line] = status; return nil }); err != nil {
			return fmt.Errorf("read staged statuses: %w", err)
		}
		if snap.Lookups, err = readLookups(ctx, tx, refs, ""); err != nil {
			return err
		}
		if preflightProbe != nil {
			return preflightProbe(ctx, tx)
		}
		return nil
	})
	return snap, err
}
