package stock

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
)

var idempotencyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,100}$`)

const maxNote = 500

// requestHash fingerprints an operation: the same idempotency key must come
// with the same hash to be a replay. RECEIPT uses "RECEIPT|<captured unit id>",
// the formula migration 000015 used for units registered before E-01.
func requestHash(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

func receiptKey(receiptUnitID string) string { return "receipt-unit-" + receiptUnitID }

const movementColumns = `m.id::text, m.item_type, m.vehicle_unit_id::text, m.movement_type, m.quantity_before, m.quantity_after,
	m.from_status, m.to_status, m.from_location_id::text, m.to_location_id::text, m.source_type, m.source_id::text,
	m.source_reference, m.actor_id::text, m.note, m.created_at`

func scanMovement(row pgx.Row, extra ...any) (Movement, error) {
	var m Movement
	err := row.Scan(append([]any{&m.ID, &m.ItemType, &m.VehicleUnitID, &m.MovementType, &m.QuantityBefore, &m.QuantityAfter,
		&m.FromStatus, &m.ToStatus, &m.FromLocationID, &m.ToLocationID, &m.SourceType, &m.SourceID, &m.SourceReference,
		&m.ActorID, &m.Note, &m.CreatedAt}, extra...)...)
	return m, err
}

// Receipt is the registration of one unit, for RecordReceipt.
type Receipt struct {
	UnitID             string
	GoodsReceiptID     string
	GoodsReceiptNumber string
	GoodsReceiptUnitID string
	LocationID         *string
	RequestID          string
}

// RecordReceipt writes a newly registered unit's RECEIPT movement (0 -> 1, none
// -> RECEIVED_PENDING_PDI, at its location) inside the registering transaction.
// vehicles.RegisterFromReceiptUnit calls it right after inserting the unit; it
// is not a separate step for receipt posting (D-07). Its idempotency key is
// derived from the captured goods receipt unit, so a unit is received once; the
// database also checks the movement is the unit's first, from its own receipt.
// No separate audit event: the registration's audit event names the movement.
func RecordReceipt(ctx context.Context, tx pgx.Tx, actor audit.Actor, r Receipt) (string, error) {
	if actor.Type != audit.ActorUser || actor.UserID == "" {
		return "", errors.New("stock movements are recorded for a signed-in user")
	}
	var id string
	err := tx.QueryRow(ctx, `INSERT INTO stock_movements (vehicle_unit_id, movement_type, quantity_before, quantity_after,
			from_status, to_status, to_location_id, source_type, source_id, source_reference, actor_id, request_id,
			idempotency_key, request_hash)
		VALUES ($1, $2, 0, 1, NULL, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id::text`,
		r.UnitID, TypeReceipt, ReceivedStatus, r.LocationID, SourceGoodsReceipt, r.GoodsReceiptID, r.GoodsReceiptNumber,
		actor.UserID, nullable(r.RequestID), receiptKey(r.GoodsReceiptUnitID), requestHash(TypeReceipt, r.GoodsReceiptUnitID)).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("record receipt movement: %w", err)
	}
	return id, nil
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// validateMove checks a MoveRequest before the database is touched.
func validateMove(req MoveRequest) error {
	var probs []FieldProblem
	if _, ok := Transitions[req.Type]; !ok {
		probs = append(probs, FieldProblem{"type", "is not a supported movement"})
	}
	if !idempotencyPattern.MatchString(req.IdempotencyKey) {
		probs = append(probs, FieldProblem{"Idempotency-Key", "must be 8-100 letters, digits, '-' or '_'"})
	}
	if req.ExpectedVersion != nil && *req.ExpectedVersion < 1 {
		probs = append(probs, FieldProblem{"version", "must be 1 or more"})
	}
	if req.Note != nil {
		n := strings.TrimSpace(*req.Note)
		switch {
		case n == "":
			probs = append(probs, FieldProblem{"note", "must not be blank"})
		case utf8.RuneCountInString(n) > maxNote:
			probs = append(probs, FieldProblem{"note", "must be at most 500 characters"})
		}
	}
	if len(probs) > 0 {
		return &ValidationError{Fields: probs}
	}
	return nil
}

func moveHash(req MoveRequest) string {
	version, note := "", ""
	if req.ExpectedVersion != nil {
		version = strconv.Itoa(*req.ExpectedVersion)
	}
	if req.Note != nil {
		note = strings.TrimSpace(*req.Note)
	}
	return requestHash(req.Type, req.UnitID, version, note)
}

// MoveUnit changes a vehicle unit's status by recording one movement, inside
// the caller's transaction:
//
//  1. lock the unit (concealed as ErrNotFound when it is outside scope);
//  2. if the actor already used the idempotency key: the same operation returns
//     the first movement (replayed = true, nothing written), another one is
//     ErrIdempotencyKeyReused;
//  3. check ExpectedVersion (ErrVersionConflict) and that the unit is in the
//     movement's starting status (*InvalidStateError);
//  4. insert the movement — the database applies it to the unit (status,
//     version + 1) — and record the audit event.
//
// The caller decides who may make which movement (permissions are per task) and
// commits. Any error leaves the transaction unusable: roll it back. On
// errKeyRace (another transaction committed the same key first) roll back and
// call Replay.
func MoveUnit(ctx context.Context, tx pgx.Tx, actor audit.Actor, scope access.Scope, req MoveRequest) (MoveResult, bool, error) {
	if actor.Type != audit.ActorUser || actor.UserID == "" {
		return MoveResult{}, false, errors.New("stock movements are recorded for a signed-in user")
	}
	if err := validateMove(req); err != nil {
		return MoveResult{}, false, err
	}
	if _, err := uuid.Parse(req.UnitID); err != nil {
		return MoveResult{}, false, ErrNotFound
	}
	if req.Note != nil {
		n := strings.TrimSpace(*req.Note)
		req.Note = &n
	}
	var u UnitState
	err := tx.QueryRow(ctx, `SELECT id::text, unit_number, status, location_id::text, version FROM vehicle_units
		WHERE id = $1 FOR UPDATE`, req.UnitID).Scan(&u.ID, &u.UnitNumber, &u.Status, &u.LocationID, &u.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return MoveResult{}, false, ErrNotFound
	}
	if err != nil {
		return MoveResult{}, false, fmt.Errorf("lock vehicle unit: %w", err)
	}
	if u.LocationID != nil && !scope.Allows(*u.LocationID) {
		return MoveResult{}, false, ErrNotFound
	}
	hash := moveHash(req)
	// After the lock: a concurrent request with the same key on this unit has committed by now.
	if res, found, err := replay(ctx, tx, actor.UserID, req.IdempotencyKey, hash); found || err != nil {
		return res, found, err
	}
	if req.ExpectedVersion != nil && *req.ExpectedVersion != u.Version {
		return MoveResult{}, false, ErrVersionConflict
	}
	edge := Transitions[req.Type]
	if u.Status != edge.From {
		return MoveResult{}, false, &InvalidStateError{Current: u.Status, Required: edge.From}
	}
	m, err := scanMovement(tx.QueryRow(ctx, `INSERT INTO stock_movements AS m (vehicle_unit_id, movement_type, quantity_before,
			quantity_after, from_status, to_status, from_location_id, to_location_id, source_type, actor_id, request_id,
			idempotency_key, request_hash, note)
		VALUES ($1, $2, 1, 1, $3, $4, $5, $5, $6, $7, $8, $9, $10, $11) RETURNING `+movementColumns,
		u.ID, req.Type, edge.From, edge.To, u.LocationID, SourceStaffAction, actor.UserID, nullable(req.RequestID),
		req.IdempotencyKey, hash, req.Note))
	if err != nil {
		return MoveResult{}, false, insertError(err, u.Status, edge.From)
	}
	if err := tx.QueryRow(ctx, `SELECT status, version FROM vehicle_units WHERE id = $1`, u.ID).Scan(&u.Status, &u.Version); err != nil {
		return MoveResult{}, false, fmt.Errorf("read moved unit: %w", err)
	}
	var location any
	if u.LocationID != nil {
		location = *u.LocationID
	}
	if err := audit.Record(ctx, tx, audit.Event{Actor: actor, Action: "vehicle_unit.moved", EntityType: "vehicle_unit",
		EntityID: u.ID, Reference: u.UnitNumber, RequestID: req.RequestID, Summary: audit.Summary{
			"movementId": m.ID, "movementType": m.MovementType,
			"status": audit.Change{From: edge.From, To: edge.To}, "locationId": location, "version": u.Version,
		}}); err != nil {
		return MoveResult{}, false, err
	}
	return MoveResult{Movement: m, Unit: u}, false, nil
}

// insertError maps the named constraints of a movement insert.
func insertError(err error, current, required string) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return fmt.Errorf("record movement: %w", err)
	}
	switch {
	case pgErr.Code == "23505" && pgErr.ConstraintName == "stock_movements_idempotency_key":
		return errKeyRace
	case pgErr.Code == "23514" && pgErr.ConstraintName == "stock_movements_from_matches_unit":
		return &InvalidStateError{Current: current, Required: required}
	case pgErr.Code == "23514" && pgErr.ConstraintName == "vehicle_units_chassis_after_receipt":
		return ErrChassisRequired
	}
	return fmt.Errorf("record movement: %w", err)
}

// replay looks up the actor's earlier movement with key: the same operation
// returns it with the unit as it is now; a different one is ErrIdempotencyKeyReused.
func replay(ctx context.Context, q querier, actorID, key, hash string) (MoveResult, bool, error) {
	var stored string
	var u UnitState
	m, err := scanMovement(q.QueryRow(ctx, `SELECT `+movementColumns+`, m.request_hash,
			v.id::text, v.unit_number, v.status, v.location_id::text, v.version
		FROM stock_movements m JOIN vehicle_units v ON v.id = m.vehicle_unit_id
		WHERE m.actor_id = $1 AND m.idempotency_key = $2`, actorID, key),
		&stored, &u.ID, &u.UnitNumber, &u.Status, &u.LocationID, &u.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return MoveResult{}, false, nil
	}
	if err != nil {
		return MoveResult{}, false, fmt.Errorf("look up idempotency key: %w", err)
	}
	if stored != hash {
		return MoveResult{}, false, ErrIdempotencyKeyReused
	}
	return MoveResult{Movement: m, Unit: u}, true, nil
}

// Replay resolves errKeyRace after the caller rolled back: the committed
// movement with the key is returned (replayed) or the key was used for another
// operation. q is the pool or a new transaction.
func Replay(ctx context.Context, q querier, actorID string, req MoveRequest) (MoveResult, error) {
	if req.Note != nil {
		n := strings.TrimSpace(*req.Note)
		req.Note = &n
	}
	res, found, err := replay(ctx, q, actorID, req.IdempotencyKey, moveHash(req))
	if err == nil && !found {
		err = errors.New("idempotency key conflict without a committed movement")
	}
	return res, err
}

// IsKeyRace reports whether MoveUnit failed because the key was committed concurrently.
func IsKeyRace(err error) bool { return errors.Is(err, errKeyRace) }

type querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}
