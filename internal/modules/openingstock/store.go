package openingstock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
)

// Batch statuses.
const (
	BatchPreviewed = "PREVIEWED"
	BatchRejected  = "REJECTED"
)

// Summary counts a batch's rows.
type Summary struct {
	TotalRows      int `json:"totalRows" example:"120"`
	ValidRows      int `json:"validRows" example:"100"`
	WarningRows    int `json:"warningRows" example:"12"`
	ErrorRows      int `json:"errorRows" example:"5"`
	BlockedRows    int `json:"blockedRows" example:"3"`
	CanProceedRows int `json:"canProceedRows" example:"112"`
	DuplicateRows  int `json:"duplicateRows" example:"4"`
	UnresolvedRows int `json:"unresolvedRows" example:"3"`
	// SkippedBlankRows: lines with no value at all (ignored).
	SkippedBlankRows int `json:"skippedBlankRows" example:"0"`
} //@name OpeningStockImportSummary

// Batch is one uploaded file and its preview.
type Batch struct {
	ID               string    `json:"id"`
	Reference        string    `json:"reference" example:"OSI-000001"`
	Status           string    `json:"status" enums:"PREVIEWED,REJECTED" example:"PREVIEWED"`
	TemplateVersion  *string   `json:"templateVersion" extensions:"x-nullable" example:"OS-IMPORT-1"`
	FileName         string    `json:"fileName" example:"opening-stock.csv"`
	FileSHA256       string    `json:"fileSha256"`
	FileSizeBytes    int64     `json:"fileSizeBytes" example:"2048"`
	FileAttachmentID string    `json:"fileAttachmentId"`
	UploadedByID     string    `json:"uploadedById"`
	UploadedByName   *string   `json:"uploadedByName" extensions:"x-nullable"`
	UploadedAt       time.Time `json:"uploadedAt"`
	BusinessDate     string    `json:"businessDate" example:"2026-11-16"`
	ExpiresAt        time.Time `json:"expiresAt"`
	// Expired: the preview is history only; G-07 must refuse to commit it.
	Expired    bool        `json:"expired"`
	Summary    Summary     `json:"summary"`
	FileIssues []FileIssue `json:"fileIssues"`
	// CommitAvailable stays false: the G-06 screens never offer a commit. The
	// commit is POST /opening-stock-imports/{id}/commit (G-07); see approval/commit.
	CommitAvailable bool `json:"commitAvailable"`
	// Approval: the recorded physical-count approval (G-07); null until approved.
	Approval *Approval `json:"approval" extensions:"x-nullable"`
	// Commit: the batch's commit (G-07); null until committed. Once set the batch
	// is COMMITTED for good (status stays the staging status PREVIEWED).
	Commit *CommitSummary `json:"commit" extensions:"x-nullable"`
} //@name OpeningStockImportBatch

// Row is one staged row as the API returns it.
type Row struct {
	RowNumber  int               `json:"rowNumber" example:"3"`
	Status     string            `json:"status" enums:"VALID,WARNING,ERROR,BLOCKED_UNVERIFIED_MAPPING" example:"VALID"`
	CanProceed bool              `json:"canProceed"`
	Duplicate  bool              `json:"duplicate"`
	Unresolved bool              `json:"unresolved"`
	Source     map[string]string `json:"source"`
	Normalised NormalisedRow     `json:"normalised"`
	Resolved   map[string]any    `json:"resolved"`
	Issues     []Issue           `json:"issues"`
} //@name OpeningStockImportRow

// NormalisedRow holds the values G-07 will use (null when not valid).
type NormalisedRow struct {
	BrandID           *string `json:"brandId" extensions:"x-nullable"`
	ModelID           *string `json:"modelId" extensions:"x-nullable"`
	VariantID         *string `json:"variantId" extensions:"x-nullable"`
	ColourID          *string `json:"colourId" extensions:"x-nullable"`
	LocationID        *string `json:"locationId" extensions:"x-nullable"`
	SupplierID        *string `json:"supplierId" extensions:"x-nullable"`
	ChassisNumber     *string `json:"chassisNumber" extensions:"x-nullable"`
	ChassisNormalised *string `json:"chassisNormalised" extensions:"x-nullable"`
	MotorNormalised   *string `json:"motorNormalised" extensions:"x-nullable"`
	BatteryNormalised *string `json:"batteryNormalised" extensions:"x-nullable"`
	ChargerNormalised *string `json:"chargerNormalised" extensions:"x-nullable"`
	StockStatus       *string `json:"stockStatus" extensions:"x-nullable"`
	ReceivedDate      *string `json:"receivedDate" extensions:"x-nullable"`
	AcquisitionCost   *string `json:"acquisitionCost" extensions:"x-nullable"`
} //@name OpeningStockNormalisedRow

// NewBatch is a batch to stage with its rows and its file's attachment metadata.
type NewBatch struct {
	ID, AttachmentID, ObjectKey, StorageDriver string
	FileName, SHA256Hex                        string
	SHA256                                     []byte
	SizeBytes                                  int64
	UploadedBy, IdempotencyKey, RequestID      string
	Actor                                      audit.Actor
	TemplateVersion                            *string
	Status                                     string
	BusinessDate                               time.Time
	UploadedAt, ExpiresAt                      time.Time
	Summary                                    Summary
	FileIssues                                 []FileIssue
	Rows                                       []RowResult
}

// RowFilter selects staged rows.
type RowFilter struct {
	BatchID                   string
	Status                    string
	DuplicateOnly, Unresolved bool
	Page, Limit               int
}

// Errors.
var (
	ErrNotFound         = errors.New("import batch not found")
	ErrKeyTaken         = errors.New("idempotency key already used")
	ErrForbidden        = errors.New("forbidden")
	ErrEmptyFile        = errors.New("empty file")
	ErrTooLarge         = errors.New("file too large")
	ErrUnsupportedType  = errors.New("unsupported file type")
	ErrKeyReused        = errors.New("idempotency key used for another file")
	ErrIdempotencyKey   = errors.New("idempotency key missing or invalid")
	ErrFileFieldMissing = errors.New("file field missing")
)

// Repository stages and reads batches.
type Repository interface {
	Lookups(ctx context.Context, refs Refs) (Lookups, error)
	// Save stores the attachment metadata, batch, rows and audit events in one
	// transaction; ErrKeyTaken when the uploader's idempotency key was used meanwhile.
	Save(ctx context.Context, b NewBatch) error
	Get(ctx context.Context, id string) (Batch, error)
	FindByKey(ctx context.Context, uploader, key string) (Batch, bool, error)
	List(ctx context.Context, uploader string, all bool, page, limit int) ([]Batch, int, error)
	Rows(ctx context.Context, f RowFilter) ([]Row, int, error)

	// G-07.
	// SaveApproval stores an approval with its audit event; ErrAlreadyApproved
	// when the batch was approved meanwhile.
	SaveApproval(ctx context.Context, a NewApproval) error
	SourceFile(ctx context.Context, batchID string) (SourceFile, error)
	StoredRows(ctx context.Context, batchID string) ([]StoredRow, error)
	// Commit re-validates and commits in one transaction (see PGStore.Commit).
	Commit(ctx context.Context, in CommitInput) (CommitResult, error)
	CommitByKey(ctx context.Context, committer, key string) (CommitResult, bool, error)
	CommitOf(ctx context.Context, batchID string) (CommitResult, bool, error)
}

// PGStore is the PostgreSQL Repository.
type PGStore struct{ pool *pgxpool.Pool }

// NewPGStore returns a Repository backed by pool.
func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

// Lookups implements Repository: every referenced brand, model, variant, colour,
// location and supplier, and which identifiers already exist. Read only.
func (s *PGStore) Lookups(ctx context.Context, f Refs) (Lookups, error) {
	return readLookups(ctx, s.pool, f, "")
}

// dbQuerier is a pool or a transaction.
type dbQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// readLookups reads the lookups through q; lock (" FOR SHARE" in the commit)
// is appended to the catalogue, location and supplier reads so they cannot
// change until the transaction ends.
func readLookups(ctx context.Context, db dbQuerier, f Refs, lock string) (Lookups, error) {
	l := Lookups{ExistingChassis: map[string]bool{}, OpenReceiptChassis: map[string]bool{}, ExistingMotor: map[string]bool{},
		ExistingBattery: map[string]bool{}, ExistingCharger: map[string]bool{}}
	q := func(sql string, args []any, dest func() []any, each func()) error {
		rows, err := db.Query(ctx, sql, args...)
		if err != nil {
			return err
		}
		_, err = pgx.ForEachRow(rows, dest(), func() error { each(); return nil })
		return err
	}
	var b BrandRef
	if err := q(`SELECT id::text, name, lifecycle_status, authorisation_status FROM brands WHERE lower(name) = ANY($1)`+lock,
		[]any{f.BrandNames}, func() []any { return []any{&b.ID, &b.Name, &b.Lifecycle, &b.Authorisation} },
		func() { l.Brands = append(l.Brands, b) }); err != nil {
		return l, fmt.Errorf("read brands: %w", err)
	}
	var m ModelRef
	if err := q(`SELECT id::text, brand_id::text, internal_code, name, lifecycle_status FROM models WHERE upper(internal_code) = ANY($1)`+lock,
		[]any{f.ModelCodes}, func() []any { return []any{&m.ID, &m.BrandID, &m.Code, &m.Name, &m.Lifecycle} },
		func() { l.Models = append(l.Models, m) }); err != nil {
		return l, fmt.Errorf("read models: %w", err)
	}
	var v VariantRef
	if err := q(`SELECT id::text, brand_id::text, model_id::text, variant_code, name, lifecycle_status FROM variants
		WHERE upper(variant_code) = ANY($1)`+lock, []any{f.VariantCodes},
		func() []any { return []any{&v.ID, &v.BrandID, &v.ModelID, &v.Code, &v.Name, &v.Lifecycle} },
		func() { l.Variants = append(l.Variants, v) }); err != nil {
		return l, fmt.Errorf("read variants: %w", err)
	}
	variantIDs := make([]string, len(l.Variants))
	for i, x := range l.Variants {
		variantIDs[i] = x.ID
	}
	var c ColourRef
	if err := q(`SELECT id::text, variant_id::text, name, colour_code, lifecycle_status FROM variant_colours WHERE variant_id = ANY($1::uuid[])`+lock,
		[]any{variantIDs}, func() []any { return []any{&c.ID, &c.VariantID, &c.Name, &c.Code, &c.Lifecycle} },
		func() {
			x := c
			if c.Code != nil {
				code := *c.Code
				x.Code = &code
			}
			l.Colours = append(l.Colours, x)
		}); err != nil {
		return l, fmt.Errorf("read colours: %w", err)
	}
	var loc LocationRef
	if err := q(`SELECT id::text, code, name, active FROM locations WHERE upper(code) = ANY($1)`+lock, []any{f.LocationCodes},
		func() []any { return []any{&loc.ID, &loc.Code, &loc.Name, &loc.Active} },
		func() { l.Locations = append(l.Locations, loc) }); err != nil {
		return l, fmt.Errorf("read locations: %w", err)
	}
	var sp SupplierRef
	if err := q(`SELECT id::text, legal_name, trade_name, status FROM suppliers WHERE lower(legal_name) = ANY($1) OR lower(trade_name) = ANY($1)`+lock,
		[]any{f.SupplierNames}, func() []any { return []any{&sp.ID, &sp.LegalName, &sp.TradeName, &sp.Status} },
		func() {
			x := sp
			if sp.TradeName != nil {
				t := *sp.TradeName
				x.TradeName = &t
			}
			l.Suppliers = append(l.Suppliers, x)
		}); err != nil {
		return l, fmt.Errorf("read suppliers: %w", err)
	}
	var id string
	for _, x := range []struct {
		sql  string
		vals []string
		into map[string]bool
	}{
		{`SELECT chassis_normalised FROM vehicle_units WHERE chassis_normalised = ANY($1)`, f.Chassis, l.ExistingChassis},
		{`SELECT u.chassis_normalised FROM goods_receipt_units u JOIN goods_receipts g ON g.id = u.goods_receipt_id
			WHERE g.status = 'DRAFT' AND u.chassis_normalised = ANY($1)`, f.Chassis, l.OpenReceiptChassis},
		{`SELECT motor_normalised FROM vehicle_units WHERE motor_normalised = ANY($1)`, f.Motors, l.ExistingMotor},
		{`SELECT battery_normalised FROM vehicle_units WHERE battery_normalised = ANY($1)`, f.Batteries, l.ExistingBattery},
		{`SELECT charger_normalised FROM vehicle_units WHERE charger_normalised = ANY($1)`, f.Chargers, l.ExistingCharger},
	} {
		into := x.into
		if err := q(x.sql, []any{x.vals}, func() []any { return []any{&id} }, func() { into[id] = true }); err != nil {
			return l, fmt.Errorf("read identifiers: %w", err)
		}
	}
	return l, nil
}

// Save implements Repository.
func (s *PGStore) Save(ctx context.Context, b NewBatch) error {
	if b.FileIssues == nil {
		b.FileIssues = []FileIssue{}
	}
	issues, err := json.Marshal(b.FileIssues)
	if err != nil {
		return err
	}
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO attachments (id, object_key, storage_driver, content_type, size_bytes, sha256, original_name,
				uploaded_by, owner_type, owner_id) VALUES ($1, $2, $3, 'text/csv', $4, $5, $6, $7, 'opening_stock_import', $8)`,
			b.AttachmentID, b.ObjectKey, b.StorageDriver, b.SizeBytes, b.SHA256, b.FileName, b.UploadedBy, b.ID); err != nil {
			return fmt.Errorf("insert import file attachment: %w", err)
		}
		sm := b.Summary
		if _, err := tx.Exec(ctx, `INSERT INTO opening_stock_import_batches (id, template_version, file_attachment_id, file_name,
				file_sha256, file_size_bytes, uploaded_by, idempotency_key, status, business_date, expires_at, total_rows, valid_rows,
				warning_rows, error_rows, blocked_rows, duplicate_rows, unresolved_rows, skipped_blank_rows, file_issues, uploaded_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)`,
			b.ID, b.TemplateVersion, b.AttachmentID, b.FileName, b.SHA256Hex, b.SizeBytes, b.UploadedBy, b.IdempotencyKey, b.Status,
			b.BusinessDate, b.ExpiresAt, sm.TotalRows, sm.ValidRows, sm.WarningRows, sm.ErrorRows, sm.BlockedRows, sm.DuplicateRows,
			sm.UnresolvedRows, sm.SkippedBlankRows, issues, b.UploadedAt); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "opening_stock_import_batches_idempotency_key" {
				return ErrKeyTaken
			}
			return fmt.Errorf("insert import batch: %w", err)
		}
		rows := make([][]any, len(b.Rows))
		for i, r := range b.Rows {
			src, _ := json.Marshal(r.Source)
			res, _ := json.Marshal(r.Resolved)
			iss, _ := json.Marshal(r.Issues)
			rows[i] = []any{b.ID, r.Line, r.Status, r.CanProceed, r.Duplicate, r.Unresolved, src, r.BrandID, r.ModelID, r.VariantID,
				r.ColourID, r.LocationID, r.SupplierID, r.ChassisNumber, r.ChassisNormalised, r.MotorNormalised, r.BatteryNormalised,
				r.ChargerNormalised, r.StockStatus, r.ReceivedDate, r.AcquisitionCost, res, iss}
		}
		if len(rows) > 0 {
			if _, err := tx.CopyFrom(ctx, pgx.Identifier{"opening_stock_import_rows"}, []string{"batch_id", "row_number", "status",
				"can_proceed", "duplicate", "unresolved", "source", "brand_id", "model_id", "variant_id", "colour_id", "location_id",
				"supplier_id", "chassis_number", "chassis_normalised", "motor_normalised", "battery_normalised", "charger_normalised",
				"stock_status", "received_date", "acquisition_cost", "resolved", "issues"}, pgx.CopyFromRows(rows)); err != nil {
				return fmt.Errorf("stage import rows: %w", err)
			}
		}
		if err := audit.Record(ctx, tx, audit.Event{Actor: b.Actor, Action: "attachment.uploaded", EntityType: "attachment",
			EntityID: b.AttachmentID, RequestID: b.RequestID, Summary: audit.Summary{"contentType": "text/csv", "sizeBytes": b.SizeBytes,
				"owner": "opening_stock_import"}}); err != nil {
			return err
		}
		return audit.Record(ctx, tx, audit.Event{Actor: b.Actor, Action: "opening_stock_import.previewed", EntityType: "opening_stock_import",
			EntityID: b.ID, RequestID: b.RequestID, Summary: audit.Summary{"status": b.Status, "totalRows": sm.TotalRows,
				"validRows": sm.ValidRows, "warningRows": sm.WarningRows, "errorRows": sm.ErrorRows, "blockedRows": sm.BlockedRows,
				"sha256": b.SHA256Hex}})
	})
	if errors.Is(err, ErrKeyTaken) {
		return ErrKeyTaken
	}
	return err
}

const batchColumns = `b.id::text, b.reference, b.status, b.template_version, b.file_name, b.file_sha256, b.file_size_bytes,
	b.file_attachment_id::text, b.uploaded_by::text, u.display_name, b.uploaded_at, to_char(b.business_date, 'YYYY-MM-DD'), b.expires_at,
	b.expires_at <= now(), b.total_rows, b.valid_rows, b.warning_rows, b.error_rows, b.blocked_rows, b.duplicate_rows,
	b.unresolved_rows, b.skipped_blank_rows, b.file_issues,
	ap.id::text, ap.approved_by::text, ua.display_name, ap.approved_at, ap.physical_count_reference,
	to_char(ap.physical_count_date, 'YYYY-MM-DD'), ap.mappings_reviewed, ap.duplicates_reviewed, ap.received_dates_approved, ap.note,
	c.id::text, c.reference, c.committed_by::text, uc.display_name, c.committed_at, c.units_created`

const batchFrom = ` FROM opening_stock_import_batches b LEFT JOIN users u ON u.id = b.uploaded_by
	LEFT JOIN opening_stock_import_approvals ap ON ap.batch_id = b.id LEFT JOIN users ua ON ua.id = ap.approved_by
	LEFT JOIN opening_stock_import_commits c ON c.batch_id = b.id LEFT JOIN users uc ON uc.id = c.committed_by`

func scanBatch(r pgx.Row) (Batch, error) {
	var b Batch
	var issues []byte
	var ap struct {
		id, by, ref, date           *string
		name, note                  *string
		at                          *time.Time
		mappings, duplicates, dates *bool
	}
	var cm struct {
		id, ref, by, name *string
		at                *time.Time
		units             *int
	}
	s := &b.Summary
	err := r.Scan(&b.ID, &b.Reference, &b.Status, &b.TemplateVersion, &b.FileName, &b.FileSHA256, &b.FileSizeBytes, &b.FileAttachmentID,
		&b.UploadedByID, &b.UploadedByName, &b.UploadedAt, &b.BusinessDate, &b.ExpiresAt, &b.Expired, &s.TotalRows, &s.ValidRows,
		&s.WarningRows, &s.ErrorRows, &s.BlockedRows, &s.DuplicateRows, &s.UnresolvedRows, &s.SkippedBlankRows, &issues,
		&ap.id, &ap.by, &ap.name, &ap.at, &ap.ref, &ap.date, &ap.mappings, &ap.duplicates, &ap.dates, &ap.note,
		&cm.id, &cm.ref, &cm.by, &cm.name, &cm.at, &cm.units)
	if err != nil {
		return b, err
	}
	if ap.id != nil {
		b.Approval = &Approval{ID: *ap.id, ApprovedByID: *ap.by, ApprovedByName: ap.name, ApprovedAt: *ap.at,
			PhysicalCountReference: *ap.ref, PhysicalCountDate: *ap.date, MappingsReviewed: *ap.mappings,
			DuplicatesReviewed: *ap.duplicates, ReceivedDatesApproved: *ap.dates, Note: ap.note}
	}
	if cm.id != nil {
		b.Commit = &CommitSummary{ID: *cm.id, Reference: *cm.ref, CommittedByID: *cm.by, CommittedByName: cm.name,
			CommittedAt: *cm.at, UnitsCreated: *cm.units}
	}
	s.CanProceedRows = s.ValidRows + s.WarningRows
	b.FileIssues = []FileIssue{}
	err = json.Unmarshal(issues, &b.FileIssues)
	return b, err
}

// Get implements Repository.
func (s *PGStore) Get(ctx context.Context, id string) (Batch, error) {
	b, err := scanBatch(s.pool.QueryRow(ctx, `SELECT `+batchColumns+batchFrom+` WHERE b.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return b, ErrNotFound
	}
	return b, err
}

// FindByKey implements Repository.
func (s *PGStore) FindByKey(ctx context.Context, uploader, key string) (Batch, bool, error) {
	b, err := scanBatch(s.pool.QueryRow(ctx, `SELECT `+batchColumns+batchFrom+` WHERE b.uploaded_by = $1 AND b.idempotency_key = $2`,
		uploader, key))
	if errors.Is(err, pgx.ErrNoRows) {
		return b, false, nil
	}
	return b, err == nil, err
}

// List implements Repository: newest first; the uploader's own unless all.
func (s *PGStore) List(ctx context.Context, uploader string, all bool, page, limit int) ([]Batch, int, error) {
	where, args := " WHERE b.uploaded_by = $1", []any{uploader}
	if all {
		where, args = "", nil
	}
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM opening_stock_import_batches b`+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count import batches: %w", err)
	}
	n := len(args)
	rows, err := s.pool.Query(ctx, `SELECT `+batchColumns+batchFrom+where+` ORDER BY b.uploaded_at DESC, b.id LIMIT $`+
		strconv.Itoa(n+1)+` OFFSET $`+strconv.Itoa(n+2), append(args, limit, (page-1)*limit)...)
	if err != nil {
		return nil, 0, fmt.Errorf("list import batches: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Batch, error) { return scanBatch(r) })
	return out, total, err
}

// Rows implements Repository: file order.
func (s *PGStore) Rows(ctx context.Context, f RowFilter) ([]Row, int, error) {
	conds, args := []string{"batch_id = $1"}, []any{f.BatchID}
	if f.Status != "" {
		args = append(args, f.Status)
		conds = append(conds, "status = $"+strconv.Itoa(len(args)))
	}
	if f.DuplicateOnly {
		conds = append(conds, "duplicate")
	}
	if f.Unresolved {
		conds = append(conds, "unresolved")
	}
	where := " WHERE " + strings.Join(conds, " AND ")
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM opening_stock_import_rows`+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count import rows: %w", err)
	}
	n := len(args)
	rows, err := s.pool.Query(ctx, `SELECT row_number, status, can_proceed, duplicate, unresolved, source, brand_id::text, model_id::text,
			variant_id::text, colour_id::text, location_id::text, supplier_id::text, chassis_number, chassis_normalised, motor_normalised,
			battery_normalised, charger_normalised, stock_status, to_char(received_date, 'YYYY-MM-DD'), acquisition_cost::text, resolved, issues
		FROM opening_stock_import_rows`+where+` ORDER BY row_number LIMIT $`+strconv.Itoa(n+1)+` OFFSET $`+strconv.Itoa(n+2),
		append(args, f.Limit, (f.Page-1)*f.Limit)...)
	if err != nil {
		return nil, 0, fmt.Errorf("list import rows: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Row, error) {
		var x Row
		var src, res, iss []byte
		n := &x.Normalised
		if err := r.Scan(&x.RowNumber, &x.Status, &x.CanProceed, &x.Duplicate, &x.Unresolved, &src, &n.BrandID, &n.ModelID, &n.VariantID,
			&n.ColourID, &n.LocationID, &n.SupplierID, &n.ChassisNumber, &n.ChassisNormalised, &n.MotorNormalised, &n.BatteryNormalised,
			&n.ChargerNormalised, &n.StockStatus, &n.ReceivedDate, &n.AcquisitionCost, &res, &iss); err != nil {
			return x, err
		}
		if err := json.Unmarshal(src, &x.Source); err != nil {
			return x, err
		}
		if err := json.Unmarshal(res, &x.Resolved); err != nil {
			return x, err
		}
		return x, json.Unmarshal(iss, &x.Issues)
	})
	return out, total, err
}
