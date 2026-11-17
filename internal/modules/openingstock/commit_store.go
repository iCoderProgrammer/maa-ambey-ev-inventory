package openingstock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/stock"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/vehicles"
)

// commitFault, when set (tests only), runs after the units are created and
// before the transaction commits; an error rolls everything back.
var commitFault func(ctx context.Context, tx pgx.Tx) error

// SaveApproval implements Repository.
func (s *PGStore) SaveApproval(ctx context.Context, a NewApproval) error {
	var note *string
	if a.Input.Note != nil {
		n := strings.TrimSpace(*a.Input.Note)
		note = &n
	}
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var id string
		err := tx.QueryRow(ctx, `INSERT INTO opening_stock_import_approvals (batch_id, file_sha256, approved_by,
				physical_count_reference, physical_count_date, mappings_reviewed, duplicates_reviewed, received_dates_approved,
				note, request_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id::text`,
			a.BatchID, a.FileSHA256, a.Actor.UserID, strings.TrimSpace(a.Input.PhysicalCountReference), a.Input.PhysicalCountDate,
			*a.Input.MappingsReviewed, *a.Input.DuplicatesReviewed, *a.Input.ReceivedDatesApproved, note,
			nullText(a.RequestID)).Scan(&id)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "opening_stock_import_approvals_batch_key" {
				return ErrAlreadyApproved
			}
			if errors.As(err, &pgErr) && pgErr.ConstraintName == "opening_stock_import_batch_previewed" {
				return ErrNotCommittable
			}
			return fmt.Errorf("insert import approval: %w", err)
		}
		return audit.Record(ctx, tx, audit.Event{Actor: a.Actor, Action: "opening_stock_import.approved",
			EntityType: "opening_stock_import", EntityID: a.BatchID, RequestID: a.RequestID, Summary: audit.Summary{
				"approvalId": id, "sha256": a.FileSHA256, "physicalCountReference": strings.TrimSpace(a.Input.PhysicalCountReference),
				"physicalCountDate": a.Input.PhysicalCountDate, "receivedDatesApproved": *a.Input.ReceivedDatesApproved}})
	})
	return err
}

func nullText(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// SourceFile implements Repository: the batch's stored file.
func (s *PGStore) SourceFile(ctx context.Context, batchID string) (SourceFile, error) {
	var f SourceFile
	err := s.pool.QueryRow(ctx, `SELECT a.object_key, encode(a.sha256, 'hex'), a.size_bytes
		FROM opening_stock_import_batches b JOIN attachments a ON a.id = b.file_attachment_id
		WHERE b.id = $1 AND a.owner_type = 'opening_stock_import' AND a.owner_id = b.id::text`, batchID).Scan(&f.ObjectKey, &f.SHA256, &f.SizeBytes)
	if errors.Is(err, pgx.ErrNoRows) {
		return f, ErrSourceInvalid
	}
	return f, err
}

// StoredRows implements Repository: the staged rows in file order.
func (s *PGStore) StoredRows(ctx context.Context, batchID string) ([]StoredRow, error) {
	return storedRows(ctx, s.pool, batchID)
}

func storedRows(ctx context.Context, db dbQuerier, batchID string) ([]StoredRow, error) {
	rows, err := db.Query(ctx, `SELECT row_number, can_proceed, source, brand_id::text, model_id::text, variant_id::text,
			location_id::text, chassis_normalised, stock_status
		FROM opening_stock_import_rows WHERE batch_id = $1 ORDER BY row_number`, batchID)
	if err != nil {
		return nil, fmt.Errorf("read staged rows: %w", err)
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (StoredRow, error) {
		var x StoredRow
		var src []byte
		if err := r.Scan(&x.Line, &x.CanProceed, &src, &x.BrandID, &x.ModelID, &x.VariantID, &x.LocationID, &x.ChassisNormalised,
			&x.StockStatus); err != nil {
			return x, err
		}
		return x, json.Unmarshal(src, &x.Source)
	})
}

// Commit implements Repository. One transaction:
//  1. lock the batch row (FOR UPDATE): concurrent commits of the batch queue here;
//  2. refuse if it was committed, is not PREVIEWED, has no approval for its file or has expired;
//  3. re-read the catalogue, locations, suppliers and identifiers under share
//     locks and re-validate every row (Validate + rowGate): any refusal → nothing written;
//  4. insert the commit (unique per batch and per committer+key), then for each
//     row the vehicle unit, its history, its OPENING_BALANCE movement and audit;
//  5. the audit event of the commit. At COMMIT the database checks the ledger
//     (every unit has its entry movement) and that units = rows.
func (s *PGStore) Commit(ctx context.Context, in CommitInput) (CommitResult, error) {
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		// The lock is its own statement: under READ COMMITTED a request that waited
		// here must read the state the winner committed, which a later statement's
		// fresh snapshot sees (a subquery of the locking statement would not).
		var locked string
		err := tx.QueryRow(ctx, `SELECT id::text FROM opening_stock_import_batches WHERE id = $1 AND file_sha256 = $2 FOR UPDATE`,
			in.BatchID, in.FileSHA256).Scan(&locked)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return ErrNotFound
		case err != nil:
			return fmt.Errorf("lock import batch: %w", err)
		}
		var status string
		var expired, committed, approved bool
		if err := tx.QueryRow(ctx, `SELECT b.status, b.expires_at <= $2,
				EXISTS (SELECT 1 FROM opening_stock_import_commits c WHERE c.batch_id = b.id),
				EXISTS (SELECT 1 FROM opening_stock_import_approvals a WHERE a.batch_id = b.id AND a.file_sha256 = b.file_sha256)
			FROM opening_stock_import_batches b WHERE b.id = $1`, in.BatchID, in.Now).Scan(&status, &expired, &committed, &approved); err != nil {
			return fmt.Errorf("read import batch: %w", err)
		}
		switch {
		case committed:
			return &AlreadyCommittedError{}
		case status != BatchPreviewed:
			return ErrNotCommittable
		case !approved:
			return ErrApprovalRequired
		case expired:
			return ErrPreviewExpired
		case len(in.ApprovedStatuses) == 0:
			return ErrStatusesNotApproved
		}
		stored, err := storedRows(ctx, tx, in.BatchID)
		if err != nil {
			return err
		}
		if !sameSource(in.Rows, stored) {
			return ErrSourceInvalid
		}
		lookups, err := readLookups(ctx, tx, CollectRefs(in.Rows), " FOR SHARE")
		if err != nil {
			return err
		}
		results := Validate(in.Rows, lookups, in.Scope, in.BusinessDate)
		if refused := rowGate(results, stored, in.ApprovedStatuses); len(refused) > 0 {
			return &CommitBlockedError{Rows: refused}
		}

		if _, err := tx.Exec(ctx, `INSERT INTO opening_stock_import_commits (id, batch_id, file_sha256, approval_id, committed_by,
				idempotency_key, units_created, request_id, committed_at)
			SELECT $1, $2, $3, a.id, $4, $5, $6, $7, $8 FROM opening_stock_import_approvals a WHERE a.batch_id = $2`,
			in.CommitID, in.BatchID, in.FileSHA256, in.Actor.UserID, in.IdempotencyKey, len(results), nullText(in.RequestID),
			in.Now); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				switch pgErr.ConstraintName {
				case "opening_stock_import_commits_idempotency_key":
					return ErrKeyTaken
				case "opening_stock_import_commits_batch_key":
					return &AlreadyCommittedError{}
				}
			}
			return fmt.Errorf("insert import commit: %w", err)
		}
		for _, r := range results {
			src := func(col string) *string {
				if v := strings.TrimSpace(r.Source[col]); v != "" {
					return &v
				}
				return nil
			}
			_, err := vehicles.RegisterOpeningStockUnit(ctx, tx, in.Actor, in.RequestID, vehicles.OpeningStockUnit{
				BatchID: in.BatchID, BatchReference: in.BatchReference, RowNumber: r.Line, BrandID: *r.BrandID, ModelID: *r.ModelID,
				VariantID: *r.VariantID, LocationID: *r.LocationID, SupplierID: r.SupplierID, ChassisNumber: *r.ChassisNumber,
				MotorSerial: src(ColMotor), BatterySerial: src(ColBattery), ChargerSerial: src(ColCharger), Status: *r.StockStatus,
				ReceivedDate: *r.ReceivedDate, UnitCost: *r.AcquisitionCost})
			var dup *vehicles.DuplicateIdentifierError
			if errors.As(err, &dup) {
				return &DuplicateAtCommitError{RowNumber: r.Line}
			}
			if err != nil {
				return fmt.Errorf("create opening stock unit (line %d): %w", r.Line, err)
			}
		}
		if commitFault != nil {
			if err := commitFault(ctx, tx); err != nil {
				return err
			}
		}
		return audit.Record(ctx, tx, audit.Event{Actor: in.Actor, Action: "opening_stock_import.committed",
			EntityType: "opening_stock_import", EntityID: in.BatchID, RequestID: in.RequestID, Summary: audit.Summary{
				"commitId": in.CommitID, "batchReference": in.BatchReference, "sha256": in.FileSHA256, "unitsCreated": len(results),
				"movementType": stock.TypeOpeningBalance}})
	})
	if err != nil {
		return CommitResult{}, commitError(err)
	}
	c, ok, err := s.CommitOf(ctx, in.BatchID)
	if err == nil && !ok {
		err = errors.New("commit not stored")
	}
	return c, err
}

// commitError maps commit-time database refusals; anything else stays internal.
func commitError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch {
	case pgErr.Code == "23505" && pgErr.ConstraintName == "opening_stock_import_commits_batch_key":
		return &AlreadyCommittedError{}
	case pgErr.Code == "23505" && pgErr.ConstraintName == "vehicle_units_chassis_key":
		return &DuplicateAtCommitError{}
	}
	return err
}

const commitColumns = `c.id::text, c.reference, c.batch_id::text, b.reference, c.units_created, c.committed_by::text, u.display_name,
	c.committed_at, c.approval_id::text, c.file_sha256, coalesce(b.template_version, '')`

func (s *PGStore) readCommit(ctx context.Context, where string, args ...any) (CommitResult, bool, error) {
	var c CommitResult
	err := s.pool.QueryRow(ctx, `SELECT `+commitColumns+` FROM opening_stock_import_commits c
		JOIN opening_stock_import_batches b ON b.id = c.batch_id LEFT JOIN users u ON u.id = c.committed_by WHERE `+where, args...).
		Scan(&c.CommitID, &c.Reference, &c.BatchID, &c.BatchReference, &c.UnitsCreated, &c.CommittedByID, &c.CommittedByName,
			&c.CommittedAt, &c.ApprovalID, &c.FileSHA256, &c.TemplateVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, false, nil
	}
	if err != nil {
		return c, false, fmt.Errorf("read import commit: %w", err)
	}
	c.BatchStatus, c.SourceType, c.MovementType = BatchCommitted, vehicles.SourceOpeningStock, stock.TypeOpeningBalance
	rows, err := s.pool.Query(ctx, `SELECT v.opening_stock_row_number, v.id::text, v.unit_number, v.status, m.id::text
		FROM vehicle_units v JOIN stock_movements m ON m.vehicle_unit_id = v.id AND m.movement_type = 'OPENING_BALANCE'
		WHERE v.opening_stock_batch_id = $1 ORDER BY v.opening_stock_row_number`, c.BatchID)
	if err != nil {
		return c, false, fmt.Errorf("read committed units: %w", err)
	}
	c.Units, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (CommittedUnit, error) {
		var u CommittedUnit
		return u, r.Scan(&u.RowNumber, &u.VehicleUnitID, &u.UnitNumber, &u.Status, &u.MovementID)
	})
	return c, err == nil, err
}

// CommitByKey implements Repository.
func (s *PGStore) CommitByKey(ctx context.Context, committer, key string) (CommitResult, bool, error) {
	return s.readCommit(ctx, `c.committed_by = $1 AND c.idempotency_key = $2`, committer, key)
}

// CommitOf implements Repository.
func (s *PGStore) CommitOf(ctx context.Context, batchID string) (CommitResult, bool, error) {
	return s.readCommit(ctx, `c.batch_id = $1`, batchID)
}
