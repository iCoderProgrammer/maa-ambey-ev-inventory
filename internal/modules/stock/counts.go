package stock

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
)

// Stock counts and adjustment approval (E-05; ADR-023; PRD §5.3, LED-005,
// LED-006). A count is an adjustment request at one location: staff record
// what they found, one line per SKU or vehicle unit; the server records what
// the ledger expected (and the balance/unit version it read) and the variance.
// Recording and submitting change no stock. A different person approves or
// rejects the submitted count. Approval, in one transaction, re-checks every
// line against the ledger, posts one COUNT_ADJUSTMENT movement per parts line
// with a variance (the database applies it to the balance) and records the
// decision; a rejection posts nothing. Vehicle discrepancies are findings only:
// no approved movement corrects a missing or found unit (OQ-33, OQ-39).
//
// No approval threshold is approved (LED-005, A-05): every count needs an
// approver (PolicyApproverRequired); nothing is approved automatically.

// Count statuses, policy, ledger type and source (DB CHECKs, migration 000018).
const (
	CountDraft     = "DRAFT"
	CountSubmitted = "SUBMITTED"
	CountApproved  = "APPROVED"
	CountRejected  = "REJECTED"

	// PolicyApproverRequired: every submitted count needs a decision by someone
	// who took no part in it. The only policy until thresholds are approved.
	PolicyApproverRequired = "APPROVER_REQUIRED"

	TypeCountAdjustment = "COUNT_ADJUSTMENT"
	SourceStockCount    = "STOCK_COUNT"
)

// Line outcomes (derived, not stored).
const (
	OutcomePending             = "PENDING"              // the count is not decided
	OutcomeNoChange            = "NO_CHANGE"            // approved, no variance
	OutcomePosted              = "POSTED"               // approved, parts adjustment posted
	OutcomeDiscrepancyRecorded = "DISCREPANCY_RECORDED" // approved, vehicle finding (no movement)
	OutcomeNotPosted           = "NOT_POSTED"           // rejected
)

// Errors of count operations (besides ErrNotFound, ErrItemArchived,
// ErrSerialNotSupported, ErrIdempotencyKeyReused, *ValidationError).
var (
	// ErrSelfDecision: the caller created, submitted or recorded a line of the count (LED-006).
	ErrSelfDecision = errors.New("own stock count")
	// ErrCountVersion: the count changed since the caller read it.
	ErrCountVersion = errors.New("stock count changed since it was read")
)

// CountStateError: the count is not in a status that allows the action.
type CountStateError struct{ Message string }

func (e *CountStateError) Error() string { return e.Message }

// StockChangedError lists the lines whose stock moved after they were counted.
type StockChangedError struct{ LineIDs []string }

func (e *StockChangedError) Error() string { return "stock changed since it was counted" }

// CountLine is one counted item.
type CountLine struct {
	ID       string `json:"id"`
	ItemType string `json:"itemType" enums:"STOCK_ITEM,VEHICLE_UNIT"`
	// Parts lines.
	StockItemID *string `json:"stockItemId" extensions:"x-nullable"`
	SKU         *string `json:"sku" extensions:"x-nullable" example:"BRK-PAD-01"`
	ItemName    *string `json:"itemName" extensions:"x-nullable"`
	UOM         *string `json:"uom" extensions:"x-nullable" example:"PCS"`
	// Vehicle lines.
	VehicleUnitID  *string `json:"vehicleUnitId" extensions:"x-nullable"`
	UnitNumber     *string `json:"unitNumber" extensions:"x-nullable" example:"MAE-VEH-000001"`
	ExpectedStatus *string `json:"expectedStatus" extensions:"x-nullable" example:"AVAILABLE"`
	// Found: vehicle lines only (observed 1 or 0).
	Found *bool `json:"found" extensions:"x-nullable"`
	// Quantities as decimal strings; expected is what the ledger held when the
	// line was recorded, variance = observed - expected (server-computed).
	ExpectedQuantity string  `json:"expectedQuantity" example:"10.000"`
	ObservedQuantity string  `json:"observedQuantity" example:"8.000"`
	Variance         string  `json:"variance" example:"-2.000"`
	Reason           *string `json:"reason" extensions:"x-nullable"`
	// StockChanged: the balance or unit changed after the line was recorded
	// (undecided counts only); such a count cannot be submitted or approved.
	StockChanged     bool      `json:"stockChanged"`
	Outcome          string    `json:"outcome" enums:"PENDING,NO_CHANGE,POSTED,DISCREPANCY_RECORDED,NOT_POSTED"`
	PostedMovementID *string   `json:"postedMovementId" extensions:"x-nullable"`
	RecordedBy       string    `json:"recordedBy"`
	RecordedByName   *string   `json:"recordedByName" extensions:"x-nullable"`
	RecordedAt       time.Time `json:"recordedAt"`
} //@name StockCountLine

// StockCount is a count (adjustment request). Lines are filled on the detail only.
type StockCount struct {
	ID             string     `json:"id"`
	CountNumber    string     `json:"countNumber" example:"SC-000001"`
	LocationID     string     `json:"locationId"`
	LocationCode   string     `json:"locationCode" example:"SR1"`
	LocationName   string     `json:"locationName"`
	Status         string     `json:"status" enums:"DRAFT,SUBMITTED,APPROVED,REJECTED"`
	ApprovalPolicy string     `json:"approvalPolicy" enums:"APPROVER_REQUIRED"`
	Note           *string    `json:"note" extensions:"x-nullable"`
	CreatedBy      string     `json:"createdBy"`
	CreatedByName  *string    `json:"createdByName" extensions:"x-nullable"`
	CreatedAt      time.Time  `json:"createdAt"`
	SubmittedBy    *string    `json:"submittedBy" extensions:"x-nullable"`
	SubmittedName  *string    `json:"submittedByName" extensions:"x-nullable"`
	SubmittedAt    *time.Time `json:"submittedAt" extensions:"x-nullable"`
	DecidedBy      *string    `json:"decidedBy" extensions:"x-nullable"`
	DecidedName    *string    `json:"decidedByName" extensions:"x-nullable"`
	DecidedAt      *time.Time `json:"decidedAt" extensions:"x-nullable"`
	// DecisionNote: the approval note or the rejection reason.
	DecisionNote  *string   `json:"decisionNote" extensions:"x-nullable"`
	LineCount     int       `json:"lineCount" example:"3"`
	VarianceLines int       `json:"varianceLines" example:"1"`
	Version       int       `json:"version" example:"4"`
	UpdatedAt     time.Time `json:"updatedAt"`
	// Lines: detail only (empty in lists).
	Lines []CountLine `json:"lines"`
} //@name StockCount

// CountPage is a page of counts, newest first.
type CountPage struct {
	Items []StockCount `json:"items"`
	Page  int          `json:"page" example:"1"`
	Limit int          `json:"limit" example:"25"`
	Total int          `json:"total" example:"0"`
} //@name StockCountPage

// CountCreateInput starts a count at a location.
type CountCreateInput struct {
	LocationID string  `json:"locationId"`
	Note       *string `json:"note"`
} //@name StockCountCreateInput

// CountLineInput records (or re-records) one counted item on a draft. Give
// stockItemId with observedQuantity and uom, or vehicleUnitId with found.
type CountLineInput struct {
	// Version is the count's current version (required).
	Version     *int    `json:"version" example:"1"`
	StockItemID *string `json:"stockItemId"`
	// ObservedQuantity: decimal string >= 0, up to 3 decimals, in the SKU's unit.
	ObservedQuantity *string `json:"observedQuantity" example:"8"`
	UOM              *string `json:"uom" example:"PCS"`
	VehicleUnitID    *string `json:"vehicleUnitId"`
	Found            *bool   `json:"found"`
	// Reason: required when the observed value differs from the expected one.
	Reason *string `json:"reason" example:"Two pieces damaged and discarded"`
} //@name StockCountLineInput

// CountVersionInput is a submit body.
type CountVersionInput struct {
	Version *int `json:"version" example:"3"`
} //@name StockCountVersionInput

// CountApproveInput is an approval body.
type CountApproveInput struct {
	Version *int    `json:"version" example:"4"`
	Note    *string `json:"note"`
} //@name StockCountApproveInput

// CountRejectInput is a rejection body; the reason is required.
type CountRejectInput struct {
	Version *int   `json:"version" example:"4"`
	Reason  string `json:"reason" example:"Recount the shelf: the count sheet is incomplete"`
} //@name StockCountRejectInput

// CountListFilter selects counts.
type CountListFilter struct {
	Status, LocationID string
	ScopeLocationIDs   []string
	AllLocations       bool
	Page, Limit        int
}

func textProblem(field string, p *string, max int, required bool) *FieldProblem {
	switch {
	case p == nil && required:
		return &FieldProblem{field, "is required"}
	case p != nil && (utf8.RuneCountInString(*p) > max || strings.ContainsFunc(*p, func(r rune) bool { return r < 0x20 })):
		return &FieldProblem{field, "must be at most " + strconv.Itoa(max) + " characters"}
	}
	return nil
}

func problems(ps ...*FieldProblem) error {
	var out []FieldProblem
	for _, p := range ps {
		if p != nil {
			out = append(out, *p)
		}
	}
	if len(out) > 0 {
		return &ValidationError{Fields: out}
	}
	return nil
}

func keyProblem(key string) *FieldProblem {
	if !idempotencyPattern.MatchString(key) {
		return &FieldProblem{"Idempotency-Key", "must be 8-100 letters, digits, '-' or '_'"}
	}
	return nil
}

func versionProblem(v *int) *FieldProblem {
	if v == nil || *v < 1 {
		return &FieldProblem{"version", "is required: send the version you last read"}
	}
	return nil
}

func validateCreate(in *CountCreateInput, key string) error {
	in.LocationID = strings.TrimSpace(in.LocationID)
	in.Note = trimOpt(in.Note)
	var loc *FieldProblem
	if _, err := uuid.Parse(in.LocationID); err != nil {
		loc = &FieldProblem{"locationId", "must be a valid ID"}
	}
	return problems(keyProblem(key), loc, textProblem("note", in.Note, 500, false))
}

func validateLine(in *CountLineInput) error {
	in.StockItemID, in.VehicleUnitID, in.Reason, in.ObservedQuantity = trimOpt(in.StockItemID), trimOpt(in.VehicleUnitID),
		trimOpt(in.Reason), trimOpt(in.ObservedQuantity)
	if in.UOM != nil {
		u := strings.ToUpper(strings.TrimSpace(*in.UOM))
		in.UOM = &u
	}
	ps := []*FieldProblem{versionProblem(in.Version), textProblem("reason", in.Reason, 300, false)}
	id := func(field string, p *string) {
		if _, err := uuid.Parse(*p); err != nil {
			ps = append(ps, &FieldProblem{field, "must be a valid ID"})
		}
	}
	switch {
	case (in.StockItemID == nil) == (in.VehicleUnitID == nil):
		ps = append(ps, &FieldProblem{"stockItemId", "give either stockItemId or vehicleUnitId"})
	case in.StockItemID != nil:
		id("stockItemId", in.StockItemID)
		switch {
		case in.ObservedQuantity == nil:
			ps = append(ps, &FieldProblem{"observedQuantity", "is required"})
		case !quantityPattern.MatchString(*in.ObservedQuantity):
			ps = append(ps, &FieldProblem{"observedQuantity", "must be a number from 0 with up to 3 decimals (e.g. 8 or 2.5)"})
		}
		if in.UOM == nil || !uomPattern.MatchString(*in.UOM) {
			ps = append(ps, &FieldProblem{"uom", "is required: the SKU's unit of measure"})
		}
		if in.Found != nil {
			ps = append(ps, &FieldProblem{"found", "is for vehicle units"})
		}
	default:
		id("vehicleUnitId", in.VehicleUnitID)
		if in.Found == nil {
			ps = append(ps, &FieldProblem{"found", "is required: true if the unit was found at the location"})
		}
		if in.ObservedQuantity != nil || in.UOM != nil {
			ps = append(ps, &FieldProblem{"observedQuantity", "is for parts; give found for a vehicle unit"})
		}
	}
	return problems(ps...)
}

func validateDecision(version *int, key string, text *string, field string, required bool) error {
	return problems(keyProblem(key), versionProblem(version), textProblem(field, text, 500, required))
}

// ── Reads ────────────────────────────────────────────────────────────────────

const countColumns = `c.id::text, c.count_number, c.location_id::text, l.code, l.name, c.status, c.approval_policy, c.note,
	c.created_by::text, cu.display_name, c.created_at, c.submitted_by::text, su.display_name, c.submitted_at,
	c.decided_by::text, du.display_name, c.decided_at, c.decision_note, c.version, c.updated_at,
	(SELECT count(*) FROM stock_count_lines x WHERE x.stock_count_id = c.id)::int,
	(SELECT count(*) FROM stock_count_lines x WHERE x.stock_count_id = c.id AND x.variance <> 0)::int`

const countFrom = ` FROM stock_counts c
	JOIN locations l ON l.id = c.location_id
	LEFT JOIN users cu ON cu.id = c.created_by
	LEFT JOIN users su ON su.id = c.submitted_by
	LEFT JOIN users du ON du.id = c.decided_by`

func scanCount(row pgx.Row) (StockCount, error) {
	var c StockCount
	err := row.Scan(&c.ID, &c.CountNumber, &c.LocationID, &c.LocationCode, &c.LocationName, &c.Status, &c.ApprovalPolicy,
		&c.Note, &c.CreatedBy, &c.CreatedByName, &c.CreatedAt, &c.SubmittedBy, &c.SubmittedName, &c.SubmittedAt,
		&c.DecidedBy, &c.DecidedName, &c.DecidedAt, &c.DecisionNote, &c.Version, &c.UpdatedAt, &c.LineCount, &c.VarianceLines)
	c.Lines = []CountLine{}
	return c, err
}

type rowsQuerier interface {
	querier
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

// readCount reads a count with its lines; a count outside scope is ErrNotFound.
func readCount(ctx context.Context, q rowsQuerier, scope access.Scope, id string) (StockCount, error) {
	c, err := scanCount(q.QueryRow(ctx, `SELECT `+countColumns+countFrom+` WHERE c.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !scope.Allows(c.LocationID)) {
		return StockCount{}, ErrNotFound
	}
	if err != nil {
		return StockCount{}, fmt.Errorf("read stock count: %w", err)
	}
	rows, err := q.Query(ctx, `SELECT ln.id::text, ln.item_type, ln.stock_item_id::text, i.sku, i.name, ln.uom,
			ln.vehicle_unit_id::text, v.unit_number, ln.expected_status, ln.expected_quantity::text, ln.observed_quantity::text,
			ln.variance::numeric(13,3)::text, ln.reason, ln.posted_movement_id::text, ln.recorded_by::text, u.display_name,
			ln.recorded_at,
			c.status IN ('DRAFT', 'SUBMITTED') AND ln.expected_version <> CASE WHEN ln.item_type = 'STOCK_ITEM'
				THEN CASE WHEN coalesce(b.version, 1) = 1 THEN 0 ELSE b.version END ELSE v.version END
		FROM stock_count_lines ln
		JOIN stock_counts c ON c.id = ln.stock_count_id
		LEFT JOIN stock_items i ON i.id = ln.stock_item_id
		LEFT JOIN stock_item_balances b ON b.stock_item_id = ln.stock_item_id AND b.location_id = c.location_id
		LEFT JOIN vehicle_units v ON v.id = ln.vehicle_unit_id
		LEFT JOIN users u ON u.id = ln.recorded_by
		WHERE ln.stock_count_id = $1 ORDER BY ln.item_type, coalesce(i.sku, v.unit_number), ln.id`, id)
	if err != nil {
		return StockCount{}, fmt.Errorf("read stock count lines: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var l CountLine
		if err := rows.Scan(&l.ID, &l.ItemType, &l.StockItemID, &l.SKU, &l.ItemName, &l.UOM, &l.VehicleUnitID, &l.UnitNumber,
			&l.ExpectedStatus, &l.ExpectedQuantity, &l.ObservedQuantity, &l.Variance, &l.Reason, &l.PostedMovementID,
			&l.RecordedBy, &l.RecordedByName, &l.RecordedAt, &l.StockChanged); err != nil {
			return StockCount{}, err
		}
		if l.ItemType == ItemVehicleUnit {
			found := l.ObservedQuantity != "0.000"
			l.Found = &found
		}
		l.Outcome = lineOutcome(c.Status, l)
		c.Lines = append(c.Lines, l)
	}
	return c, rows.Err()
}

func lineOutcome(status string, l CountLine) string {
	switch {
	case status == CountRejected:
		return OutcomeNotPosted
	case status != CountApproved:
		return OutcomePending
	case l.PostedMovementID != nil:
		return OutcomePosted
	case l.Variance == "0.000":
		return OutcomeNoChange
	}
	return OutcomeDiscrepancyRecorded
}

// GetCount implements Repository.
func (s *PGStore) GetCount(ctx context.Context, scope access.Scope, id string) (StockCount, error) {
	return readCount(ctx, s.pool, scope, id)
}

// ListCounts implements Repository: newest first.
func (s *PGStore) ListCounts(ctx context.Context, f CountListFilter) (CountPage, error) {
	where := ` WHERE ($1 OR c.location_id = ANY($2::uuid[])) AND ($3 = '' OR c.location_id::text = $3) AND ($4 = '' OR c.status = $4)`
	args := []any{f.AllLocations, f.ScopeLocationIDs, f.LocationID, f.Status}
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM stock_counts c`+where, args...).Scan(&total); err != nil {
		return CountPage{}, fmt.Errorf("count stock counts: %w", err)
	}
	rows, err := s.pool.Query(ctx, `SELECT `+countColumns+countFrom+where+` ORDER BY c.created_at DESC, c.count_number DESC
		LIMIT $5 OFFSET $6`, append(args, f.Limit, (f.Page-1)*f.Limit)...)
	if err != nil {
		return CountPage{}, fmt.Errorf("list stock counts: %w", err)
	}
	defer rows.Close()
	items := []StockCount{}
	for rows.Next() {
		c, err := scanCount(rows)
		if err != nil {
			return CountPage{}, err
		}
		items = append(items, c)
	}
	return CountPage{Items: items, Page: f.Page, Limit: f.Limit, Total: total}, rows.Err()
}

// ── Writes ───────────────────────────────────────────────────────────────────

type lockedCount struct {
	ID, Number, LocationID, Status, Policy, CreatedBy string
	SubmittedBy, DecidedBy, DecisionKey, DecisionHash *string
	Version                                           int
}

// lockCount locks a count row; outside scope is ErrNotFound.
func lockCount(ctx context.Context, tx pgx.Tx, scope access.Scope, id string) (lockedCount, error) {
	var c lockedCount
	err := tx.QueryRow(ctx, `SELECT id::text, count_number, location_id::text, status, approval_policy, created_by::text,
			submitted_by::text, decided_by::text, decision_idempotency_key, decision_request_hash, version
		FROM stock_counts WHERE id = $1 FOR UPDATE`, id).
		Scan(&c.ID, &c.Number, &c.LocationID, &c.Status, &c.Policy, &c.CreatedBy, &c.SubmittedBy, &c.DecidedBy,
			&c.DecisionKey, &c.DecisionHash, &c.Version)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !scope.Allows(c.LocationID)) {
		return c, ErrNotFound
	}
	if err != nil {
		return c, fmt.Errorf("lock stock count: %w", err)
	}
	return c, nil
}

func userOnly(actor audit.Actor) error {
	if actor.Type != audit.ActorUser || actor.UserID == "" {
		return errors.New("stock counts are recorded for a signed-in user")
	}
	return nil
}

func countEvent(actor audit.Actor, action, id, number, requestID string, sum audit.Summary) audit.Event {
	return audit.Event{Actor: actor, Action: action, EntityType: "stock_count", EntityID: id, Reference: number,
		RequestID: requestID, Summary: sum}
}

// CreateCount implements Repository: a DRAFT count at a location in scope. A
// repeat of the creator's key returns the first count (replayed = true).
func (s *PGStore) CreateCount(ctx context.Context, actor audit.Actor, scope access.Scope, requestID string, in CountCreateInput,
	key string) (StockCount, bool, error) {
	if err := userOnly(actor); err != nil {
		return StockCount{}, false, err
	}
	if err := validateCreate(&in, key); err != nil {
		return StockCount{}, false, err
	}
	if !scope.Allows(in.LocationID) {
		return StockCount{}, false, &ValidationError{Fields: []FieldProblem{{"locationId", "is not one of your locations"}}}
	}
	note := ""
	if in.Note != nil {
		note = *in.Note
	}
	hash := requestHash("stock-count-create", in.LocationID, note)
	replay := func(q rowsQuerier) (StockCount, bool, error) {
		var id, stored string
		err := q.QueryRow(ctx, `SELECT id::text, create_request_hash FROM stock_counts WHERE created_by = $1 AND create_idempotency_key = $2`,
			actor.UserID, key).Scan(&id, &stored)
		if errors.Is(err, pgx.ErrNoRows) {
			return StockCount{}, false, nil
		}
		if err != nil {
			return StockCount{}, false, fmt.Errorf("look up idempotency key: %w", err)
		}
		if stored != hash {
			return StockCount{}, false, ErrIdempotencyKeyReused
		}
		c, err := readCount(ctx, q, scope, id)
		return c, err == nil, err
	}
	if c, found, err := replay(s.pool); found || err != nil {
		return c, found, err
	}
	var out StockCount
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var active bool
		err := tx.QueryRow(ctx, `SELECT active FROM locations WHERE id = $1 FOR SHARE`, in.LocationID).Scan(&active)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return &ValidationError{Fields: []FieldProblem{{"locationId", "does not exist"}}}
		case err != nil:
			return fmt.Errorf("check location: %w", err)
		case !active:
			return &ValidationError{Fields: []FieldProblem{{"locationId", "is not active"}}}
		}
		var id, number string
		if err := tx.QueryRow(ctx, `INSERT INTO stock_counts (location_id, note, created_by, create_idempotency_key, create_request_hash)
			VALUES ($1, $2, $3, $4, $5) RETURNING id::text, count_number`, in.LocationID, in.Note, actor.UserID, key, hash).
			Scan(&id, &number); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.ConstraintName == "stock_counts_create_key" {
				return errKeyRace
			}
			return fmt.Errorf("create stock count: %w", err)
		}
		sum := audit.Summary{"locationId": in.LocationID, "status": CountDraft, "approvalPolicy": PolicyApproverRequired}
		if err := audit.Record(ctx, tx, countEvent(actor, "stock_count.created", id, number, requestID, sum)); err != nil {
			return err
		}
		out, err = readCount(ctx, tx, scope, id)
		return err
	})
	if IsKeyRace(err) {
		c, found, err := replay(s.pool)
		if err == nil && !found {
			err = errors.New("idempotency key conflict without a committed count")
		}
		return c, found, err
	}
	return out, false, err
}

// baseline is the ledger state a line is compared with.
type baseline struct {
	quantity string // decimal string, 3 decimals
	status   *string
	location *string
	version  int
}

// partsBaseline reads (lock: "FOR SHARE"/"FOR UPDATE") a SKU's balance at a
// location. No row and an empty row (version 1: created, never moved) are both
// 0 at version 0, so creating the row before posting is not a stock change.
func partsBaseline(ctx context.Context, tx pgx.Tx, itemID, locationID, lock string) (baseline, error) {
	b := baseline{quantity: "0.000"}
	err := tx.QueryRow(ctx, `SELECT quantity::text, version FROM stock_item_balances WHERE stock_item_id = $1 AND location_id = $2 `+lock,
		itemID, locationID).Scan(&b.quantity, &b.version)
	if errors.Is(err, pgx.ErrNoRows) {
		return b, nil
	}
	if err != nil {
		return b, fmt.Errorf("read balance: %w", err)
	}
	if b.version == 1 {
		b.version = 0
	}
	return b, nil
}

func unitBaseline(ctx context.Context, tx pgx.Tx, unitID, lock string) (baseline, bool, error) {
	b := baseline{quantity: "1.000"}
	err := tx.QueryRow(ctx, `SELECT status, location_id::text, version FROM vehicle_units WHERE id = $1 `+lock, unitID).
		Scan(&b.status, &b.location, &b.version)
	if errors.Is(err, pgx.ErrNoRows) {
		return b, false, nil
	}
	if err != nil {
		return b, false, fmt.Errorf("read vehicle unit: %w", err)
	}
	return b, true, nil
}

func checkCountVersion(c lockedCount, v *int) error {
	if v == nil || *v != c.Version {
		return ErrCountVersion
	}
	return nil
}

// RecordLine implements Repository: records or re-records one item on a DRAFT
// count. The expected value is read from the ledger now; nothing moves.
func (s *PGStore) RecordLine(ctx context.Context, actor audit.Actor, scope access.Scope, requestID, countID string,
	in CountLineInput) (StockCount, error) {
	if err := userOnly(actor); err != nil {
		return StockCount{}, err
	}
	if err := validateLine(&in); err != nil {
		return StockCount{}, err
	}
	var out StockCount
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		c, err := lockCount(ctx, tx, scope, countID)
		if err != nil {
			return err
		}
		if c.Status != CountDraft {
			return &CountStateError{Message: "Lines can be recorded only on a draft count."}
		}
		if err := checkCountVersion(c, in.Version); err != nil {
			return err
		}
		var itemType, ref, uom, observed string
		var expStatus *string
		var base baseline
		if in.StockItemID != nil {
			itemType, ref, observed = ItemStockItem, *in.StockItemID, *in.ObservedQuantity
			var status, mode string
			err := tx.QueryRow(ctx, `SELECT status, tracking_mode, uom FROM stock_items WHERE id = $1 FOR SHARE`, ref).
				Scan(&status, &mode, &uom)
			switch {
			case errors.Is(err, pgx.ErrNoRows):
				return &ValidationError{Fields: []FieldProblem{{"stockItemId", "does not exist"}}}
			case err != nil:
				return fmt.Errorf("check stock item: %w", err)
			case status != "ACTIVE":
				return ErrItemArchived
			case mode != "QUANTITY":
				return ErrSerialNotSupported
			case uom != *in.UOM:
				return &ValidationError{Fields: []FieldProblem{{"uom", "must be the SKU's unit of measure, " + uom}}}
			}
			if base, err = partsBaseline(ctx, tx, ref, c.LocationID, "FOR SHARE"); err != nil {
				return err
			}
		} else {
			itemType, ref, observed = ItemVehicleUnit, *in.VehicleUnitID, "0"
			if *in.Found {
				observed = "1"
			}
			var ok bool
			if base, ok, err = unitBaseline(ctx, tx, ref, "FOR SHARE"); err != nil {
				return err
			}
			if !ok {
				return &ValidationError{Fields: []FieldProblem{{"vehicleUnitId", "does not exist"}}}
			}
			if base.location == nil || *base.location != c.LocationID {
				return &ValidationError{Fields: []FieldProblem{{"vehicleUnitId", "is not recorded at this count's location"}}}
			}
			expStatus = base.status
		}
		var variance bool
		if err := tx.QueryRow(ctx, `SELECT $1::numeric(12,3) <> $2::numeric(12,3)`, observed, base.quantity).Scan(&variance); err != nil {
			return fmt.Errorf("compute variance: %w", err)
		}
		if variance && in.Reason == nil {
			return &ValidationError{Fields: []FieldProblem{{"reason", "is required when the count differs from the stock record"}}}
		}
		var uomArg any
		if itemType == ItemStockItem {
			uomArg = uom
		}
		col := "stock_item_id"
		if itemType == ItemVehicleUnit {
			col = "vehicle_unit_id"
		}
		var lineID string
		err = tx.QueryRow(ctx, `SELECT id::text FROM stock_count_lines WHERE stock_count_id = $1 AND `+col+` = $2`, c.ID, ref).Scan(&lineID)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			err = tx.QueryRow(ctx, `INSERT INTO stock_count_lines (stock_count_id, item_type, `+col+`, uom, expected_quantity,
					expected_status, expected_version, observed_quantity, reason, recorded_by)
				VALUES ($1, $2, $3, $4, $5::numeric, $6, $7, $8::numeric, $9, $10) RETURNING id::text`,
				c.ID, itemType, ref, uomArg, base.quantity, expStatus, base.version, observed, in.Reason, actor.UserID).Scan(&lineID)
			if err != nil {
				return fmt.Errorf("record count line: %w", err)
			}
		case err != nil:
			return fmt.Errorf("find count line: %w", err)
		default:
			if _, err := tx.Exec(ctx, `UPDATE stock_count_lines SET uom = $2, expected_quantity = $3::numeric, expected_status = $4,
					expected_version = $5, observed_quantity = $6::numeric, reason = $7, recorded_by = $8, recorded_at = now()
				WHERE id = $1`, lineID, uomArg, base.quantity, expStatus, base.version, observed, in.Reason, actor.UserID); err != nil {
				return fmt.Errorf("re-record count line: %w", err)
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE stock_counts SET version = version + 1, updated_at = now() WHERE id = $1`, c.ID); err != nil {
			return fmt.Errorf("update stock count: %w", err)
		}
		sum := audit.Summary{"lineId": lineID, "itemType": itemType, "itemId": ref, "expected": base.quantity,
			"observed": canonicalQuantity(observed), "expectedVersion": base.version}
		if in.Reason != nil {
			sum["reason"] = *in.Reason
		}
		if err := audit.Record(ctx, tx, countEvent(actor, "stock_count.line_recorded", c.ID, c.Number, requestID, sum)); err != nil {
			return err
		}
		out, err = readCount(ctx, tx, scope, c.ID)
		return err
	})
	return out, err
}

// staleLines checks every line of a count against the ledger (locking what it
// reads) and returns the IDs of lines whose balance or unit changed since they
// were recorded. lock is "FOR SHARE" (submit) or "FOR UPDATE" (approval).
func staleLines(ctx context.Context, tx pgx.Tx, c lockedCount, lock string) ([]countLineRow, []string, error) {
	rows, err := tx.Query(ctx, `SELECT id::text, item_type, coalesce(stock_item_id, vehicle_unit_id)::text, coalesce(uom, ''),
			expected_quantity::text, observed_quantity::text, expected_version, variance <> 0, coalesce(reason, '')
		FROM stock_count_lines WHERE stock_count_id = $1 ORDER BY item_type, coalesce(stock_item_id, vehicle_unit_id)`, c.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("read count lines: %w", err)
	}
	var lines []countLineRow
	for rows.Next() {
		var l countLineRow
		if err := rows.Scan(&l.ID, &l.ItemType, &l.ItemID, &l.UOM, &l.Expected, &l.Observed, &l.ExpectedVersion, &l.HasVariance,
			&l.Reason); err != nil {
			rows.Close()
			return nil, nil, err
		}
		lines = append(lines, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	var stale []string
	for _, l := range lines {
		var b baseline
		var still bool
		if l.ItemType == ItemStockItem {
			if lock == "FOR UPDATE" {
				// Approval: create the row empty if missing so it can be locked
				// (a concurrent first receipt waits here instead of racing).
				if _, err := tx.Exec(ctx, `INSERT INTO stock_item_balances (stock_item_id, location_id) VALUES ($1, $2)
					ON CONFLICT DO NOTHING`, l.ItemID, c.LocationID); err != nil {
					return nil, nil, fmt.Errorf("create balance: %w", err)
				}
			}
			if b, err = partsBaseline(ctx, tx, l.ItemID, c.LocationID, lock); err != nil {
				return nil, nil, err
			}
			still = b.version == l.ExpectedVersion
		} else {
			var ok bool
			if b, ok, err = unitBaseline(ctx, tx, l.ItemID, lock); err != nil {
				return nil, nil, err
			}
			still = ok && b.version == l.ExpectedVersion && b.location != nil && *b.location == c.LocationID
		}
		if !still {
			stale = append(stale, l.ID)
		}
	}
	return lines, stale, nil
}

type countLineRow struct {
	ID, ItemType, ItemID, UOM, Expected, Observed, Reason string
	ExpectedVersion                                       int
	HasVariance                                           bool
}

// SubmitCount implements Repository: DRAFT -> SUBMITTED. Every line must still
// match the ledger. Nothing moves.
func (s *PGStore) SubmitCount(ctx context.Context, actor audit.Actor, scope access.Scope, requestID, countID string,
	in CountVersionInput) (StockCount, error) {
	if err := userOnly(actor); err != nil {
		return StockCount{}, err
	}
	if err := problems(versionProblem(in.Version)); err != nil {
		return StockCount{}, err
	}
	var out StockCount
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		c, err := lockCount(ctx, tx, scope, countID)
		if err != nil {
			return err
		}
		if c.Status != CountDraft {
			return &CountStateError{Message: "Only a draft count can be submitted."}
		}
		if err := checkCountVersion(c, in.Version); err != nil {
			return err
		}
		lines, stale, err := staleLines(ctx, tx, c, "FOR SHARE")
		if err != nil {
			return err
		}
		if len(lines) == 0 {
			return &CountStateError{Message: "Record at least one counted item before submitting."}
		}
		if len(stale) > 0 {
			return &StockChangedError{LineIDs: stale}
		}
		if _, err := tx.Exec(ctx, `UPDATE stock_counts SET status = 'SUBMITTED', submitted_by = $2, submitted_at = now(),
			version = version + 1, updated_at = now() WHERE id = $1`, c.ID, actor.UserID); err != nil {
			return fmt.Errorf("submit stock count: %w", err)
		}
		variances := 0
		for _, l := range lines {
			if l.HasVariance {
				variances++
			}
		}
		sum := audit.Summary{"status": audit.Change{From: CountDraft, To: CountSubmitted}, "lines": len(lines),
			"varianceLines": variances, "approvalPolicy": c.Policy}
		if err := audit.Record(ctx, tx, countEvent(actor, "stock_count.submitted", c.ID, c.Number, requestID, sum)); err != nil {
			return err
		}
		out, err = readCount(ctx, tx, scope, c.ID)
		return err
	})
	return out, err
}

// tookPart reports whether the user created, submitted or recorded a line of the count (LED-006).
func tookPart(ctx context.Context, tx pgx.Tx, c lockedCount, userID string) (bool, error) {
	if c.CreatedBy == userID || (c.SubmittedBy != nil && *c.SubmittedBy == userID) {
		return true, nil
	}
	var took bool
	err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM stock_count_lines WHERE stock_count_id = $1 AND recorded_by = $2)
		OR EXISTS (SELECT 1 FROM audit_events WHERE entity_type = 'stock_count' AND entity_id = $1::text AND actor_user_id = $2
			AND action IN ('stock_count.created', 'stock_count.line_recorded', 'stock_count.submitted') AND outcome = 'SUCCESS')`,
		c.ID, userID).Scan(&took)
	if err != nil {
		return false, fmt.Errorf("check participants: %w", err)
	}
	return took, nil
}

// DecideCount implements Repository: approve (post adjustments) or reject a
// SUBMITTED count, in one transaction:
//
//  1. lock the count; a repeat of this approver's decision with the same key is
//     replayed (same body) or refused as reused; any other decided count is
//     INVALID_STATE;
//  2. version, policy and separation of duties (no one who took part);
//  3. approve: lock every line's balance/unit and re-check it against the
//     ledger (StockChangedError if anything moved), then insert one
//     COUNT_ADJUSTMENT per parts line with a variance — the database applies it
//     — and link it to the line; audit stock_item.adjusted per movement;
//  4. record the decision and audit it.
//
// The database checks again at commit that an APPROVED count has exactly these
// movements and any other count none.
func (s *PGStore) DecideCount(ctx context.Context, actor audit.Actor, scope access.Scope, requestID, countID string, approve bool,
	version *int, text *string, key string) (StockCount, bool, error) {
	if err := userOnly(actor); err != nil {
		return StockCount{}, false, err
	}
	action, status, verb := "approve", CountApproved, "approved"
	if !approve {
		action, status, verb = "reject", CountRejected, "rejected"
	}
	t := ""
	if text != nil {
		t = *text
	}
	hash := requestHash("stock-count-"+action, countID, t)
	var out StockCount
	var replayed bool
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		c, err := lockCount(ctx, tx, scope, countID)
		if err != nil {
			return err
		}
		if c.Status == CountApproved || c.Status == CountRejected {
			if c.DecidedBy != nil && *c.DecidedBy == actor.UserID && c.DecisionKey != nil && *c.DecisionKey == key {
				if c.DecisionHash == nil || *c.DecisionHash != hash {
					return ErrIdempotencyKeyReused
				}
				replayed = true
				out, err = readCount(ctx, tx, scope, c.ID)
				return err
			}
			return &CountStateError{Message: "This count was already " + strings.ToLower(c.Status) + "."}
		}
		if c.Status != CountSubmitted {
			return &CountStateError{Message: "Only a submitted count can be " + verb + "."}
		}
		if err := checkCountVersion(c, version); err != nil {
			return err
		}
		if c.Policy != PolicyApproverRequired {
			return errors.New("unknown stock count approval policy: refusing to decide")
		}
		took, err := tookPart(ctx, tx, c, actor.UserID)
		if err != nil {
			return err
		}
		if took {
			return ErrSelfDecision
		}
		var movements []string
		if approve {
			lines, stale, err := staleLines(ctx, tx, c, "FOR UPDATE")
			if err != nil {
				return err
			}
			if len(stale) > 0 {
				return &StockChangedError{LineIDs: stale}
			}
			for _, l := range lines {
				if l.ItemType != ItemStockItem || !l.HasVariance {
					continue
				}
				id, err := postCountAdjustment(ctx, tx, actor, requestID, c, l)
				if err != nil {
					return err
				}
				movements = append(movements, id)
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE stock_counts SET status = $2, decided_by = $3, decided_at = now(), decision_note = $4,
				decision_idempotency_key = $5, decision_request_hash = $6, version = version + 1, updated_at = now()
			WHERE id = $1`, c.ID, status, actor.UserID, text, key, hash); err != nil {
			return fmt.Errorf("decide stock count: %w", err)
		}
		sum := audit.Summary{"status": audit.Change{From: CountSubmitted, To: status}, "movementIds": movements,
			"approvalPolicy": c.Policy}
		if text != nil {
			if approve {
				sum["note"] = *text
			} else {
				sum["reason"] = *text
			}
		}
		if err := audit.Record(ctx, tx, countEvent(actor, "stock_count."+verb, c.ID, c.Number, requestID, sum)); err != nil {
			return err
		}
		out, err = readCount(ctx, tx, scope, c.ID)
		return err
	})
	if err != nil {
		return StockCount{}, false, err
	}
	return out, replayed, nil
}

// postCountAdjustment inserts the line's COUNT_ADJUSTMENT (expected -> observed;
// the trigger applies it to the balance row staleLines created and locked) and links it.
func postCountAdjustment(ctx context.Context, tx pgx.Tx, actor audit.Actor, requestID string, c lockedCount,
	l countLineRow) (string, error) {
	key := "count-line-" + l.ID
	var id string
	err := tx.QueryRow(ctx, `INSERT INTO stock_movements (item_type, stock_item_id, movement_type, quantity_before, quantity_after,
			from_location_id, to_location_id, uom, source_type, source_id, source_reference, reason, actor_id, request_id,
			idempotency_key, request_hash)
		VALUES ($1, $2, $3, $4::numeric, $5::numeric, $6, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15) RETURNING id::text`,
		ItemStockItem, l.ItemID, TypeCountAdjustment, l.Expected, l.Observed, c.LocationID, l.UOM, SourceStockCount, c.ID, c.Number,
		l.Reason, actor.UserID, nullable(requestID), key, requestHash(TypeCountAdjustment, l.ID, l.Expected, l.Observed)).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.ConstraintName == "stock_movements_idempotency_key" {
			return "", ErrIdempotencyKeyReused
		}
		return "", partsInsertError(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE stock_count_lines SET posted_movement_id = $2 WHERE id = $1`, l.ID, id); err != nil {
		return "", fmt.Errorf("link count adjustment: %w", err)
	}
	var sku string
	if err := tx.QueryRow(ctx, `SELECT sku FROM stock_items WHERE id = $1`, l.ItemID).Scan(&sku); err != nil {
		return "", fmt.Errorf("read sku: %w", err)
	}
	sum := audit.Summary{"movementId": id, "movementType": TypeCountAdjustment, "locationId": c.LocationID, "uom": l.UOM,
		"balance": audit.Change{From: l.Expected, To: l.Observed}, "stockCountId": c.ID, "countNumber": c.Number, "reason": l.Reason}
	if err := audit.Record(ctx, tx, audit.Event{Actor: actor, Action: "stock_item.adjusted", EntityType: "stock_item",
		EntityID: l.ItemID, Reference: sku, RequestID: requestID, Summary: sum}); err != nil {
		return "", err
	}
	return id, nil
}

// RecordCountRefusal implements Repository: audits a refused decision (DENIED).
func (s *PGStore) RecordCountRefusal(ctx context.Context, actor audit.Actor, requestID, countID, action, reason string) error {
	return audit.Record(ctx, s.pool, audit.Event{Actor: actor, Action: "stock_count." + action, Outcome: audit.OutcomeDenied,
		EntityType: "stock_count", EntityID: countID, RequestID: requestID, Summary: audit.Summary{"reason": reason}})
}
