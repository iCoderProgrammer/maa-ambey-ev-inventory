package receiving

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/vehicles"
)

// Receipt posting (D-07; ADR-025). One transaction: lock the receipt, replay a
// repeated key, lock the purchase order, re-check every line against the PO
// as it is now and every captured unit, register each ACCEPTED unit as a
// vehicle unit (vehicles.RegisterFromReceiptUnit: unit, history, RECEIPT
// movement, audit) at the chosen location, add the received quantities to the
// PO lines, move the PO to PARTIALLY_RECEIVED or RECEIVED, mark the receipt
// POSTED and audit. Any error rolls everything back. Units enter
// RECEIVED_PENDING_PDI: nothing becomes saleable before PDI (D-06).

// ActionPosted is the receipt history action of a posting.
const ActionPosted = "POSTED"

// POEventGoodsReceived is the purchase order history action of a posting that
// changes the PO status.
const POEventGoodsReceived = "GOODS_RECEIVED"

// PostInput posts a draft. version is the receipt's current version; locationId
// is where the goods were received (an active location in the caller's scope).
type PostInput struct {
	Version    *int   `json:"version" example:"3"`
	LocationID string `json:"locationId" example:"7c9e6679-7425-40de-944b-e07fc1f90ae7"`
} //@name GoodsReceiptPostInput

// PostedUnit is a vehicle unit registered by the posting (as it is now).
type PostedUnit struct {
	VehicleUnitID      string  `json:"vehicleUnitId"`
	UnitNumber         string  `json:"unitNumber" example:"MAE-VEH-000001"`
	GoodsReceiptUnitID string  `json:"goodsReceiptUnitId"`
	ChassisNumber      *string `json:"chassisNumber" extensions:"x-nullable"`
	Status             string  `json:"status" example:"RECEIVED_PENDING_PDI"`
} //@name GoodsReceiptPostedUnit

// PostResult is the posted receipt and its vehicle units.
type PostResult struct {
	Receipt      Detail       `json:"receipt"`
	VehicleUnits []PostedUnit `json:"vehicleUnits"`
} //@name GoodsReceiptPostResult

// PostBlockedError lists what stops a draft from being posted (409 POSTING_BLOCKED).
type PostBlockedError struct{ Fields []FieldProblem }

func (e *PostBlockedError) Error() string { return "goods receipt cannot be posted" }

// IdentifierConflictError: a captured chassis/frame number is already a
// registered vehicle (409 DUPLICATE_IDENTIFIER, D-05). Field is the path of the
// unit in the receipt detail (lines[i].units[j].chassisNumber).
type IdentifierConflictError struct{ Field string }

func (e *IdentifierConflictError) Error() string { return "identifier already registered: " + e.Field }

// ErrIdempotencyKeyReused: the poster used the key for a different request.
var ErrIdempotencyKeyReused = errors.New("idempotency key already used for a different request")

var postKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,100}$`)

// postKey is a validated posting request.
type postKey struct {
	Version    int
	LocationID string
	Key        string
	Hash       string
}

func validatePost(receiptID string, in PostInput, key string) (postKey, error) {
	var probs []FieldProblem
	p := postKey{Key: key, LocationID: strings.ToLower(strings.TrimSpace(in.LocationID))}
	if !postKeyPattern.MatchString(key) {
		probs = append(probs, FieldProblem{"Idempotency-Key", "must be 8-100 letters, digits, '-' or '_'"})
	}
	switch {
	case in.Version == nil:
		probs = append(probs, FieldProblem{"version", "is required"})
	case *in.Version < 1:
		probs = append(probs, FieldProblem{"version", "must be 1 or more"})
	default:
		p.Version = *in.Version
	}
	if _, err := uuid.Parse(p.LocationID); err != nil || len(p.LocationID) != 36 {
		probs = append(probs, FieldProblem{"locationId", "must be the location where the goods were received"})
	}
	if len(probs) > 0 {
		return p, &ValidationError{Fields: probs}
	}
	sum := sha256.Sum256([]byte(strings.Join([]string{strings.ToLower(receiptID), strconv.Itoa(p.Version), p.LocationID}, "\x1f")))
	p.Hash = hex.EncodeToString(sum[:])
	return p, nil
}

// postLine is one receipt line with its PO line as it is now (both locked).
type postLine struct {
	Index                              int // position in the receipt detail (PO line order)
	ID, POLineID                       string
	LineNo, Ordered, Posted            int
	Received, Accepted, Rejected       int
	OverReceipt                        *int
	CapturedAccepted, CapturedRejected int
}

// postBlockers checks every line against the PO as it is now (PO-003) and
// requires every received unit to be captured individually (PO-004, INV-001).
// Identifiers stay as captured: a missing chassis number is allowed until the
// unit leaves PDI (INV-002); documents are not required (OQ-25).
func postBlockers(lines []postLine) []FieldProblem {
	var out []FieldProblem
	if len(lines) == 0 {
		return []FieldProblem{{"lines", "nothing has been received on this receipt"}}
	}
	for _, l := range lines {
		path := "lines[" + strconv.Itoa(l.Index) + "]"
		if l.CapturedAccepted != l.Accepted || l.CapturedRejected != l.Rejected {
			out = append(out, FieldProblem{path + ".units", fmt.Sprintf(
				"capture every received unit of PO line %d: %d accepted and %d rejected expected, %d and %d captured",
				l.LineNo, l.Accepted, l.Rejected, l.CapturedAccepted, l.CapturedRejected)})
		}
		left := max(l.Ordered-l.Posted, 0)
		if l.Received > left && (l.OverReceipt == nil || *l.OverReceipt < l.Received) {
			out = append(out, FieldProblem{path + ".receivedQuantity", fmt.Sprintf(
				"only %d left on PO line %d (ordered %d, posted %d); receiving %d needs an authorised over-receipt",
				left, l.LineNo, l.Ordered, l.Posted, l.Received)})
		}
	}
	return out
}

// newPOStatus is the PO status after a posting: RECEIVED when every line is
// fully received, PARTIALLY_RECEIVED otherwise (PO-002).
func newPOStatus(allReceived bool) string {
	if allReceived {
		return "RECEIVED"
	}
	return "PARTIALLY_RECEIVED"
}

// ── Repository ──────────────────────────────────────────────────────────────

func postedUnits(ctx context.Context, q querier, receiptID string) ([]PostedUnit, error) {
	return queryAll(ctx, q, func(r pgx.Row) (PostedUnit, error) {
		var u PostedUnit
		err := r.Scan(&u.VehicleUnitID, &u.UnitNumber, &u.GoodsReceiptUnitID, &u.ChassisNumber, &u.Status)
		return u, err
	}, `SELECT v.id::text, v.unit_number, v.goods_receipt_unit_id::text, v.chassis_number, v.status
		FROM vehicle_units v JOIN goods_receipt_units gu ON gu.id = v.goods_receipt_unit_id
		JOIN goods_receipt_lines gl ON gl.id = gu.goods_receipt_line_id
		JOIN purchase_order_lines pl ON pl.id = gl.purchase_order_line_id
		WHERE v.goods_receipt_id = $1 ORDER BY pl.line_no, gu.unit_no`, receiptID)
}

func postResult(ctx context.Context, q querier, receiptID string) (PostResult, error) {
	d, err := loadDetail(ctx, q, receiptID)
	if err != nil {
		return PostResult{}, err
	}
	units, err := postedUnits(ctx, q, receiptID)
	return PostResult{Receipt: d, VehicleUnits: units}, err
}

func findPostKey(ctx context.Context, q querier, actorID, key string) (id, hash string, found bool, err error) {
	err = q.QueryRow(ctx, `SELECT id::text, post_request_hash FROM goods_receipts WHERE posted_by = $1 AND post_idempotency_key = $2`,
		actorID, key).Scan(&id, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, nil
	}
	return id, hash, err == nil, err
}

var errPostKeyRace = errors.New("posting key taken concurrently")

// Post implements Repository.
func (s *PGStore) Post(ctx context.Context, actor Actor, scope access.Scope, id string, p postKey) (PostResult, bool, error) {
	var res PostResult
	var replayed bool
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		res, replayed, err = post(ctx, tx, actor, scope, id, p)
		return err
	})
	if errors.Is(err, errPostKeyRace) {
		rid, hash, found, ferr := findPostKey(ctx, s.pool, actor.UserID, p.Key)
		switch {
		case ferr != nil:
			return PostResult{}, false, ferr
		case !found:
			return PostResult{}, false, errors.New("posting key conflict without a committed posting")
		case rid != id || hash != p.Hash:
			return PostResult{}, false, ErrIdempotencyKeyReused
		}
		res, err = postResult(ctx, s.pool, id)
		return res, err == nil, err
	}
	if err != nil {
		return PostResult{}, false, postError(err)
	}
	return res, replayed, nil
}

// postError maps commit-time and constraint errors; domain errors pass through.
func postError(err error) error {
	err = vehicles.CommitError(err)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42501" {
		return &StateError{Message: "This goods receipt is no longer a draft and cannot be posted. Reload it."}
	}
	return err
}

func post(ctx context.Context, tx pgx.Tx, actor Actor, scope access.Scope, id string, p postKey) (PostResult, bool, error) {
	r, err := lockReceipt(ctx, tx, id, 0)
	if err != nil {
		return PostResult{}, false, err
	}
	// After the lock: a concurrent repeat with the same key has committed by now.
	if rid, hash, found, err := findPostKey(ctx, tx, actor.UserID, p.Key); err != nil {
		return PostResult{}, false, err
	} else if found {
		if rid != id || hash != p.Hash {
			return PostResult{}, false, ErrIdempotencyKeyReused
		}
		res, err := postResult(ctx, tx, id)
		return res, err == nil, err
	}
	switch r.Status {
	case StatusPosted:
		return PostResult{}, false, &StateError{Message: "This goods receipt is already posted; its vehicles are registered. Nothing was changed."}
	case StatusCancelled:
		return PostResult{}, false, &StateError{Message: "This goods receipt is cancelled and cannot be posted."}
	}
	if r.Version != p.Version {
		return PostResult{}, false, ErrVersionConflict
	}
	poNumber, poStatus, err := lockPO(ctx, tx, r.PurchaseOrderID)
	if err != nil {
		return PostResult{}, false, err
	}
	if !poReceivable(poStatus) {
		return PostResult{}, false, &StateError{Message: "The purchase order is no longer open for receipts (" + poStatus + "). Cancel this draft instead."}
	}
	var active bool
	err = tx.QueryRow(ctx, `SELECT active FROM locations WHERE id = $1`, p.LocationID).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && (!active || !scope.Allows(p.LocationID))) {
		return PostResult{}, false, invalid("locationId", "must be an active location in your scope")
	}
	if err != nil {
		return PostResult{}, false, err
	}

	// The receipt's lines with their PO lines as they are now (PO locked above).
	lines, err := queryAll(ctx, tx, func(row pgx.Row) (postLine, error) {
		var l postLine
		err := row.Scan(&l.Index, &l.ID, &l.POLineID, &l.LineNo, &l.Ordered, &l.Posted, &l.Received, &l.Accepted, &l.Rejected,
			&l.OverReceipt, &l.CapturedAccepted, &l.CapturedRejected)
		return l, err
	}, `SELECT (SELECT count(*)::int FROM purchase_order_lines x WHERE x.purchase_order_id = pl.purchase_order_id AND x.line_no < pl.line_no),
			gl.id::text, pl.id::text, pl.line_no, pl.quantity, pl.received_quantity, gl.received_quantity, gl.accepted_quantity,
			gl.rejected_quantity, gl.over_receipt_quantity,
			(SELECT count(*)::int FROM goods_receipt_units u WHERE u.goods_receipt_line_id = gl.id AND u.condition = 'ACCEPTED'),
			(SELECT count(*)::int FROM goods_receipt_units u WHERE u.goods_receipt_line_id = gl.id AND u.condition = 'REJECTED')
		FROM goods_receipt_lines gl JOIN purchase_order_lines pl ON pl.id = gl.purchase_order_line_id
		WHERE gl.goods_receipt_id = $1 ORDER BY pl.line_no`, id)
	if err != nil {
		return PostResult{}, false, fmt.Errorf("read receipt lines: %w", err)
	}
	if blockers := postBlockers(lines); len(blockers) > 0 {
		return PostResult{}, false, &PostBlockedError{Fields: blockers}
	}

	// Captured units in detail order; their chassis numbers must be free on every other receipt.
	type captured struct {
		ID, Condition string
		Chassis       *string
		LineIndex     int
		UnitIndex     int
	}
	units, err := queryAll(ctx, tx, func(row pgx.Row) (captured, error) {
		var c captured
		err := row.Scan(&c.ID, &c.Condition, &c.Chassis, &c.LineIndex, &c.UnitIndex)
		return c, err
	}, `SELECT u.id::text, u.condition, u.chassis_number,
			(SELECT count(*)::int FROM purchase_order_lines x WHERE x.purchase_order_id = pl.purchase_order_id AND x.line_no < pl.line_no),
			(row_number() OVER (PARTITION BY u.goods_receipt_line_id ORDER BY u.unit_no) - 1)::int
		FROM goods_receipt_units u JOIN goods_receipt_lines gl ON gl.id = u.goods_receipt_line_id
		JOIN purchase_order_lines pl ON pl.id = gl.purchase_order_line_id
		WHERE u.goods_receipt_id = $1 ORDER BY pl.line_no, u.unit_no`, id)
	if err != nil {
		return PostResult{}, false, fmt.Errorf("read captured units: %w", err)
	}
	decisions := make([]lineDecision, 0, len(lines))
	byLine := map[int]*lineDecision{}
	for _, l := range lines {
		decisions = append(decisions, lineDecision{indexedLine: indexedLine{Index: l.Index}})
		byLine[l.Index] = &decisions[len(decisions)-1]
	}
	for _, u := range units {
		if d := byLine[u.LineIndex]; d != nil {
			d.Units = append(d.Units, UnitInput{ChassisNumber: u.Chassis})
		}
	}
	if err := checkChassisElsewhere(ctx, tx, id, decisions); err != nil {
		return PostResult{}, false, err
	}

	auditActor := audit.UserActor(actor.UserID, actor.Username)
	registered := 0
	for _, u := range units {
		if u.Condition != "ACCEPTED" {
			continue
		}
		loc := p.LocationID
		if _, err := vehicles.RegisterFromReceiptUnit(ctx, tx, auditActor, actor.RequestID, u.ID, &loc); err != nil {
			var dup *vehicles.DuplicateIdentifierError
			if errors.As(err, &dup) {
				return PostResult{}, false, &IdentifierConflictError{Field: fmt.Sprintf("lines[%d].units[%d].chassisNumber", u.LineIndex, u.UnitIndex)}
			}
			return PostResult{}, false, fmt.Errorf("register vehicle unit: %w", err)
		}
		registered++
	}

	if _, err := tx.Exec(ctx, `UPDATE purchase_order_lines pl SET received_quantity = pl.received_quantity + gl.received_quantity
		FROM goods_receipt_lines gl WHERE gl.goods_receipt_id = $1 AND gl.purchase_order_line_id = pl.id`, id); err != nil {
		return PostResult{}, false, fmt.Errorf("update PO received quantities: %w", err)
	}
	var allReceived bool
	if err := tx.QueryRow(ctx, `SELECT bool_and(received_quantity >= quantity) FROM purchase_order_lines WHERE purchase_order_id = $1`,
		r.PurchaseOrderID).Scan(&allReceived); err != nil {
		return PostResult{}, false, err
	}
	toPO := newPOStatus(allReceived)
	if toPO != poStatus {
		if _, err := tx.Exec(ctx, `UPDATE purchase_orders SET status = $2, version = version + 1 WHERE id = $1`, r.PurchaseOrderID, toPO); err != nil {
			return PostResult{}, false, fmt.Errorf("update PO status: %w", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO purchase_order_events (purchase_order_id, action, from_status, to_status, actor_id, note)
			VALUES ($1, $2, $3, $4, $5, $6)`, r.PurchaseOrderID, POEventGoodsReceived, poStatus, toPO, actor.UserID, r.Number); err != nil {
			return PostResult{}, false, fmt.Errorf("record PO history: %w", err)
		}
	}
	_, err = tx.Exec(ctx, `UPDATE goods_receipts SET status = 'POSTED', posted_by = $2, posted_at = now(), location_id = $3,
			post_idempotency_key = $4, post_request_hash = $5, version = version + 1 WHERE id = $1`,
		id, actor.UserID, p.LocationID, p.Key, p.Hash)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "goods_receipts_post_idempotency_key" {
		return PostResult{}, false, errPostKeyRace
	}
	if err != nil {
		return PostResult{}, false, fmt.Errorf("mark receipt posted: %w", err)
	}
	from := StatusDraft
	if err := addEvent(ctx, tx, id, ActionPosted, &from, StatusPosted, actor.UserID, nil); err != nil {
		return PostResult{}, false, err
	}
	received, accepted, rejected := 0, 0, 0
	for _, l := range lines {
		received += l.Received
		accepted += l.Accepted
		rejected += l.Rejected
	}
	ev := event(actor, "goods_receipt.posted", id)
	ev.Reference = r.Number
	ev.Summary = audit.Summary{"number": r.Number, "purchaseOrderId": r.PurchaseOrderID, "purchaseOrderNumber": poNumber,
		"purchaseOrderStatus": audit.Change{From: poStatus, To: toPO}, "status": audit.Change{From: StatusDraft, To: StatusPosted},
		"locationId": p.LocationID, "receivedQuantity": received, "acceptedQuantity": accepted, "rejectedQuantity": rejected,
		"vehicleUnitsRegistered": registered}
	if err := audit.Record(ctx, tx, ev); err != nil {
		return PostResult{}, false, err
	}
	res, err := postResult(ctx, tx, id)
	return res, false, err
}
