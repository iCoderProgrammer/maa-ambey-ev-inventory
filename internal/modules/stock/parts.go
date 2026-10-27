package stock

import (
	"context"
	"errors"
	"fmt"
	"regexp"
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

// Parts stock (E-04; ADR-022). Receipts and issues of QUANTITY-tracked spare
// parts and accessories are rows of the same ledger (item_type STOCK_ITEM).
// Inserting one is what changes stock_item_balances (database trigger), the
// balance can never go below zero, and at commit every balance equals its
// latest movement. SERIAL SKUs are refused: no serial register exists yet.

// Parts movement types and sources (DB CHECKs, migration 000017).
const (
	TypePartsReceipt       = "PARTS_RECEIPT"
	TypePartsIssue         = "PARTS_ISSUE"
	SourceSupplierDelivery = "SUPPLIER_DELIVERY"
	ItemStockItem          = "STOCK_ITEM"
)

// Errors of parts operations (besides ErrNotFound, ErrIdempotencyKeyReused, *ValidationError).
var (
	// ErrInsufficientStock: an issue larger than the balance at that location.
	ErrInsufficientStock = errors.New("not enough stock at this location")
	// ErrSerialNotSupported: SERIAL SKUs have no serial register yet (OQ-36).
	ErrSerialNotSupported = errors.New("serial-tracked SKUs cannot be received or issued yet")
	// ErrItemArchived: archived SKUs take no new movement.
	ErrItemArchived = errors.New("the SKU is archived")
)

// PartsMovement is one parts receipt or issue.
type PartsMovement struct {
	ID           string `json:"id"`
	StockItemID  string `json:"stockItemId"`
	SKU          string `json:"sku" example:"BRK-PAD-01"`
	MovementType string `json:"movementType" enums:"PARTS_RECEIPT,PARTS_ISSUE" example:"PARTS_RECEIPT"`
	LocationID   string `json:"locationId"`
	LocationCode string `json:"locationCode" example:"SR1"`
	LocationName string `json:"locationName"`
	UOM          string `json:"uom" example:"PCS"`
	// Quantity moved (positive), and the location's balance before and after; decimal strings.
	Quantity       string `json:"quantity" example:"5.000"`
	QuantityBefore string `json:"quantityBefore" example:"10.000"`
	QuantityAfter  string `json:"quantityAfter" example:"15.000"`
	SourceType     string `json:"sourceType" enums:"SUPPLIER_DELIVERY,STAFF_ACTION"`
	// SupplierID/SupplierName: receipts; null without suppliers.view.
	SupplierID      *string   `json:"supplierId" extensions:"x-nullable"`
	SupplierName    *string   `json:"supplierName" extensions:"x-nullable"`
	SourceReference *string   `json:"sourceReference" extensions:"x-nullable" example:"DC-1234"`
	Reason          *string   `json:"reason" extensions:"x-nullable"`
	Note            *string   `json:"note" extensions:"x-nullable"`
	ActorID         string    `json:"actorId"`
	ActorName       *string   `json:"actorName" extensions:"x-nullable"`
	CreatedAt       time.Time `json:"createdAt"`
} //@name PartsMovement

// PartsBalance is the quantity of a SKU at a location.
type PartsBalance struct {
	StockItemID  string    `json:"stockItemId"`
	SKU          string    `json:"sku" example:"BRK-PAD-01"`
	LocationID   string    `json:"locationId"`
	LocationCode string    `json:"locationCode" example:"SR1"`
	LocationName string    `json:"locationName"`
	UOM          string    `json:"uom" example:"PCS"`
	Quantity     string    `json:"quantity" example:"15.000"`
	Version      int       `json:"version" example:"3"`
	UpdatedAt    time.Time `json:"updatedAt"`
} //@name PartsBalance

// PartsResult is a recorded receipt or issue and the balance it left.
type PartsResult struct {
	Movement PartsMovement `json:"movement"`
	Balance  PartsBalance  `json:"balance"`
} //@name PartsResult

// PartsReceiptInput is the body of a receipt.
type PartsReceiptInput struct {
	LocationID string `json:"locationId"`
	// Quantity received: decimal string > 0, up to 3 decimals.
	Quantity string `json:"quantity" example:"5"`
	// UOM must be the SKU's unit (no conversion).
	UOM        string `json:"uom" example:"PCS"`
	SupplierID string `json:"supplierId"`
	// The supplier's delivery challan or invoice number.
	SourceReference string  `json:"sourceReference" example:"DC-1234"`
	Note            *string `json:"note"`
} //@name PartsReceiptInput

// PartsIssueInput is the body of an issue.
type PartsIssueInput struct {
	LocationID string `json:"locationId"`
	// Quantity issued: decimal string > 0, up to 3 decimals; not more than the balance.
	Quantity string `json:"quantity" example:"2"`
	UOM      string `json:"uom" example:"PCS"`
	// Why the stock leaves (e.g. used in a repair); required.
	Reason          string  `json:"reason" example:"Fitted to customer vehicle"`
	SourceReference *string `json:"sourceReference" example:"JOB-0042"`
	Note            *string `json:"note"`
} //@name PartsIssueInput

// PartsRequest is one receipt or issue, for PostParts.
type PartsRequest struct {
	Type            string // TypePartsReceipt or TypePartsIssue
	ItemID          string
	LocationID      string
	Quantity        string
	UOM             string
	SupplierID      *string
	SourceReference *string
	Reason          *string
	Note            *string
	IdempotencyKey  string
	RequestID       string
}

var (
	quantityPattern = regexp.MustCompile(`^[0-9]{1,9}(\.[0-9]{1,3})?$`)
	uomPattern      = regexp.MustCompile(`^[A-Z][A-Z0-9]{0,9}$`)
)

// canonicalQuantity is the decimal without leading or trailing zeros ("05.50" -> "5.5").
func canonicalQuantity(q string) string {
	if i := strings.IndexByte(q, '.'); i >= 0 {
		q = strings.TrimRight(strings.TrimRight(q, "0"), ".")
	}
	q = strings.TrimLeft(q, "0")
	if q == "" || q[0] == '.' {
		q = "0" + q
	}
	return q
}

func trimOpt(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	return &v
}

// validateParts checks a PartsRequest before the database is touched and normalises it.
func validateParts(req *PartsRequest) error {
	var probs []FieldProblem
	add := func(f, m string) { probs = append(probs, FieldProblem{f, m}) }
	text := func(field string, p *string, max int, required bool) {
		switch {
		case p == nil && required:
			add(field, "is required")
		case p != nil && (utf8.RuneCountInString(*p) > max || strings.ContainsFunc(*p, func(r rune) bool { return r < 0x20 })):
			add(field, "must be at most "+strconv.Itoa(max)+" characters")
		}
	}
	if !idempotencyPattern.MatchString(req.IdempotencyKey) {
		add("Idempotency-Key", "must be 8-100 letters, digits, '-' or '_'")
	}
	req.LocationID = strings.TrimSpace(req.LocationID)
	if _, err := uuid.Parse(req.LocationID); err != nil {
		add("locationId", "must be a valid ID")
	}
	req.Quantity = strings.TrimSpace(req.Quantity)
	switch {
	case req.Quantity == "":
		add("quantity", "is required")
	case !quantityPattern.MatchString(req.Quantity):
		add("quantity", "must be a number with up to 3 decimals (e.g. 5 or 2.5)")
	case canonicalQuantity(req.Quantity) == "0":
		add("quantity", "must be more than 0")
	}
	req.UOM = strings.ToUpper(strings.TrimSpace(req.UOM))
	if !uomPattern.MatchString(req.UOM) {
		add("uom", "is required: the SKU's unit of measure")
	}
	req.SupplierID, req.SourceReference, req.Reason, req.Note =
		trimOpt(req.SupplierID), trimOpt(req.SourceReference), trimOpt(req.Reason), trimOpt(req.Note)
	switch req.Type {
	case TypePartsReceipt:
		if req.SupplierID == nil {
			add("supplierId", "is required")
		} else if _, err := uuid.Parse(*req.SupplierID); err != nil {
			add("supplierId", "must be a valid ID")
		}
		text("sourceReference", req.SourceReference, 100, true)
		if req.Reason != nil {
			add("reason", "cannot be set on a receipt")
		}
	case TypePartsIssue:
		if req.SupplierID != nil {
			add("supplierId", "cannot be set on an issue")
		}
		text("reason", req.Reason, 300, true)
		text("sourceReference", req.SourceReference, 100, false)
	default:
		return errors.New("unknown parts movement type")
	}
	text("note", req.Note, 500, false)
	if len(probs) > 0 {
		return &ValidationError{Fields: probs}
	}
	return nil
}

func partsHash(req PartsRequest) string {
	s := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	return requestHash(req.Type, req.ItemID, req.LocationID, canonicalQuantity(req.Quantity), req.UOM, s(req.SupplierID),
		s(req.SourceReference), s(req.Reason), s(req.Note))
}

const partsMovementColumns = `m.id::text, m.stock_item_id::text, i.sku, m.movement_type, m.to_location_id::text, l.code, l.name,
	m.uom, abs(m.quantity_after - m.quantity_before)::text, m.quantity_before::text, m.quantity_after::text, m.source_type,
	m.supplier_id::text, s.legal_name, m.source_reference, m.reason, m.note, m.actor_id::text, u.display_name, m.created_at`

const partsMovementFrom = ` FROM stock_movements m
	JOIN stock_items i ON i.id = m.stock_item_id
	JOIN locations l ON l.id = m.to_location_id
	LEFT JOIN suppliers s ON s.id = m.supplier_id
	LEFT JOIN users u ON u.id = m.actor_id`

func scanPartsMovement(row pgx.Row, extra ...any) (PartsMovement, error) {
	var m PartsMovement
	err := row.Scan(append([]any{&m.ID, &m.StockItemID, &m.SKU, &m.MovementType, &m.LocationID, &m.LocationCode,
		&m.LocationName, &m.UOM, &m.Quantity, &m.QuantityBefore, &m.QuantityAfter, &m.SourceType, &m.SupplierID,
		&m.SupplierName, &m.SourceReference, &m.Reason, &m.Note, &m.ActorID, &m.ActorName, &m.CreatedAt}, extra...)...)
	return m, err
}

const partsBalanceColumns = `b.stock_item_id::text, i.sku, b.location_id::text, l.code, l.name, i.uom, b.quantity::text,
	b.version, b.updated_at`

const partsBalanceFrom = ` FROM stock_item_balances b
	JOIN stock_items i ON i.id = b.stock_item_id
	JOIN locations l ON l.id = b.location_id`

func scanPartsBalance(row pgx.Row) (PartsBalance, error) {
	var b PartsBalance
	err := row.Scan(&b.StockItemID, &b.SKU, &b.LocationID, &b.LocationCode, &b.LocationName, &b.UOM, &b.Quantity,
		&b.Version, &b.UpdatedAt)
	return b, err
}

func readPartsResult(ctx context.Context, q querier, movementID string) (PartsResult, error) {
	var r PartsResult
	m, err := scanPartsMovement(q.QueryRow(ctx, `SELECT `+partsMovementColumns+partsMovementFrom+` WHERE m.id = $1`, movementID))
	if err != nil {
		return r, fmt.Errorf("read parts movement: %w", err)
	}
	b, err := scanPartsBalance(q.QueryRow(ctx, `SELECT `+partsBalanceColumns+partsBalanceFrom+`
		WHERE b.stock_item_id = $1 AND b.location_id = $2`, m.StockItemID, m.LocationID))
	if err != nil {
		return r, fmt.Errorf("read parts balance: %w", err)
	}
	return PartsResult{Movement: m, Balance: b}, nil
}

// partsReplay looks up the actor's earlier movement with key: the same parts
// operation returns it with the balance as it is now; anything else is reused.
func partsReplay(ctx context.Context, q querier, actorID, key, hash string) (PartsResult, bool, error) {
	var id, itemType, stored string
	err := q.QueryRow(ctx, `SELECT id::text, item_type, request_hash FROM stock_movements WHERE actor_id = $1 AND idempotency_key = $2`,
		actorID, key).Scan(&id, &itemType, &stored)
	if errors.Is(err, pgx.ErrNoRows) {
		return PartsResult{}, false, nil
	}
	if err != nil {
		return PartsResult{}, false, fmt.Errorf("look up idempotency key: %w", err)
	}
	if itemType != ItemStockItem || stored != hash {
		return PartsResult{}, false, ErrIdempotencyKeyReused
	}
	r, err := readPartsResult(ctx, q, id)
	return r, err == nil, err
}

// PostParts records one parts receipt or issue inside the caller's transaction:
//
//  1. check the input; the location must be in the caller's scope;
//  2. share-lock the SKU (ACTIVE, QUANTITY-tracked, the request's unit is its
//     unit), check the location (active) and a receipt's supplier (not archived);
//  3. create the balance row empty if missing (a concurrent first receipt waits
//     here) and lock it;
//  4. a key the actor already used: the same operation is replayed, another is
//     ErrIdempotencyKeyReused;
//  5. compute the new quantity in the database (no floating point); an issue
//     larger than the balance is ErrInsufficientStock;
//  6. insert the movement — the database applies it to the balance — and audit.
//
// Any error leaves the transaction unusable: roll it back. On IsKeyRace(err)
// roll back and call ReplayParts.
func PostParts(ctx context.Context, tx pgx.Tx, actor audit.Actor, scope access.Scope, req PartsRequest) (PartsResult, bool, error) {
	if actor.Type != audit.ActorUser || actor.UserID == "" {
		return PartsResult{}, false, errors.New("stock movements are recorded for a signed-in user")
	}
	if _, err := uuid.Parse(req.ItemID); err != nil {
		return PartsResult{}, false, ErrNotFound
	}
	if err := validateParts(&req); err != nil {
		return PartsResult{}, false, err
	}
	if !scope.Allows(req.LocationID) {
		return PartsResult{}, false, &ValidationError{Fields: []FieldProblem{{"locationId", "is not one of your locations"}}}
	}
	hash := partsHash(req)
	// A retry of a committed operation returns it even if the SKU, location or
	// supplier changed since; the check is repeated under the balance lock below.
	if res, found, err := partsReplay(ctx, tx, actor.UserID, req.IdempotencyKey, hash); found || err != nil {
		return res, found, err
	}
	var sku, status, mode, uom string
	err := tx.QueryRow(ctx, `SELECT sku, status, tracking_mode, uom FROM stock_items WHERE id = $1 FOR SHARE`, req.ItemID).
		Scan(&sku, &status, &mode, &uom)
	if errors.Is(err, pgx.ErrNoRows) {
		return PartsResult{}, false, ErrNotFound
	}
	if err != nil {
		return PartsResult{}, false, fmt.Errorf("lock stock item: %w", err)
	}
	switch {
	case status != "ACTIVE":
		return PartsResult{}, false, ErrItemArchived
	case mode != "QUANTITY":
		return PartsResult{}, false, ErrSerialNotSupported
	case uom != req.UOM:
		return PartsResult{}, false, &ValidationError{Fields: []FieldProblem{{"uom", "must be the SKU's unit of measure, " + uom}}}
	}
	var active bool
	err = tx.QueryRow(ctx, `SELECT active FROM locations WHERE id = $1 FOR SHARE`, req.LocationID).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) {
		return PartsResult{}, false, &ValidationError{Fields: []FieldProblem{{"locationId", "does not exist"}}}
	}
	if err != nil {
		return PartsResult{}, false, fmt.Errorf("check location: %w", err)
	}
	if !active {
		return PartsResult{}, false, &ValidationError{Fields: []FieldProblem{{"locationId", "is not active"}}}
	}
	if req.SupplierID != nil {
		var supStatus string
		err = tx.QueryRow(ctx, `SELECT status FROM suppliers WHERE id = $1 FOR SHARE`, *req.SupplierID).Scan(&supStatus)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return PartsResult{}, false, &ValidationError{Fields: []FieldProblem{{"supplierId", "does not exist"}}}
		case err != nil:
			return PartsResult{}, false, fmt.Errorf("check supplier: %w", err)
		case supStatus == "ARCHIVED":
			return PartsResult{}, false, &ValidationError{Fields: []FieldProblem{{"supplierId", "is archived"}}}
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO stock_item_balances (stock_item_id, location_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		req.ItemID, req.LocationID); err != nil {
		return PartsResult{}, false, fmt.Errorf("create balance: %w", err)
	}
	var before string
	if err := tx.QueryRow(ctx, `SELECT quantity::text FROM stock_item_balances WHERE stock_item_id = $1 AND location_id = $2 FOR UPDATE`,
		req.ItemID, req.LocationID).Scan(&before); err != nil {
		return PartsResult{}, false, fmt.Errorf("lock balance: %w", err)
	}
	// After the lock: a concurrent request with the same key here has committed by now.
	if res, found, err := partsReplay(ctx, tx, actor.UserID, req.IdempotencyKey, hash); found || err != nil {
		return res, found, err
	}
	sign := "+"
	sourceType := SourceSupplierDelivery
	if req.Type == TypePartsIssue {
		sign, sourceType = "-", SourceStaffAction
	}
	var after string
	var enough bool
	if err := tx.QueryRow(ctx, `SELECT ($1::numeric `+sign+` $2::numeric)::numeric(12,3)::text, ($1::numeric `+sign+` $2::numeric) >= 0`,
		before, req.Quantity).Scan(&after, &enough); err != nil {
		return PartsResult{}, false, fmt.Errorf("compute balance: %w", err)
	}
	if !enough {
		return PartsResult{}, false, ErrInsufficientStock
	}
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO stock_movements (item_type, stock_item_id, movement_type, quantity_before, quantity_after,
			from_location_id, to_location_id, uom, source_type, supplier_id, source_reference, reason, note, actor_id, request_id,
			idempotency_key, request_hash)
		VALUES ($1, $2, $3, $4::numeric, $5::numeric, $6, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16) RETURNING id::text`,
		ItemStockItem, req.ItemID, req.Type, before, after, req.LocationID, req.UOM, sourceType, req.SupplierID,
		req.SourceReference, req.Reason, req.Note, actor.UserID, nullable(req.RequestID), req.IdempotencyKey, hash).Scan(&id)
	if err != nil {
		return PartsResult{}, false, partsInsertError(err)
	}
	action := "stock_item.received"
	if req.Type == TypePartsIssue {
		action = "stock_item.issued"
	}
	sum := audit.Summary{"movementId": id, "movementType": req.Type, "locationId": req.LocationID, "uom": req.UOM,
		"quantity": canonicalQuantity(req.Quantity), "balance": audit.Change{From: before, To: after}}
	if req.SupplierID != nil {
		sum["supplierId"] = *req.SupplierID
	}
	if req.SourceReference != nil {
		sum["sourceReference"] = *req.SourceReference
	}
	if req.Reason != nil {
		sum["reason"] = *req.Reason
	}
	if err := audit.Record(ctx, tx, audit.Event{Actor: actor, Action: action, EntityType: "stock_item", EntityID: req.ItemID,
		Reference: sku, RequestID: req.RequestID, Summary: sum}); err != nil {
		return PartsResult{}, false, err
	}
	res, err := readPartsResult(ctx, tx, id)
	return res, false, err
}

// partsInsertError maps the named rules the database checks again under concurrency.
func partsInsertError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return fmt.Errorf("record parts movement: %w", err)
	}
	switch pgErr.ConstraintName {
	case "stock_movements_idempotency_key":
		return errKeyRace
	case "stock_movements_item_active":
		return ErrItemArchived
	case "stock_movements_item_quantity_tracked":
		return ErrSerialNotSupported
	case "stock_item_balances_not_negative", "stock_movements_quantity_valid":
		return ErrInsufficientStock
	case "stock_movements_uom_matches_item":
		return &ValidationError{Fields: []FieldProblem{{"uom", "must be the SKU's unit of measure"}}}
	case "stock_movements_location_active":
		return &ValidationError{Fields: []FieldProblem{{"locationId", "is not active"}}}
	case "stock_movements_supplier_active":
		return &ValidationError{Fields: []FieldProblem{{"supplierId", "is archived"}}}
	}
	return fmt.Errorf("record parts movement: %w", err)
}

// ReplayParts resolves a key race after the caller rolled back.
func ReplayParts(ctx context.Context, q querier, actorID string, req PartsRequest) (PartsResult, error) {
	if err := validateParts(&req); err != nil {
		return PartsResult{}, err
	}
	res, found, err := partsReplay(ctx, q, actorID, req.IdempotencyKey, partsHash(req))
	if err == nil && !found {
		err = errors.New("idempotency key conflict without a committed movement")
	}
	return res, err
}

// PartsHistoryFilter selects a SKU's parts movements.
type PartsHistoryFilter struct {
	ItemID, LocationID, Type string
	// ScopeLocationIDs limits the rows unless AllLocations.
	ScopeLocationIDs []string
	AllLocations     bool
	Page, Limit      int
}

// PartsPage is a page of parts movements.
type PartsPage struct {
	Items []PartsMovement `json:"items"`
	Page  int             `json:"page" example:"1"`
	Limit int             `json:"limit" example:"25"`
	Total int             `json:"total" example:"0"`
} //@name PartsMovementPage

func itemExists(ctx context.Context, q querier, id string) error {
	var ok bool
	if err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM stock_items WHERE id = $1)`, id).Scan(&ok); err != nil {
		return fmt.Errorf("find stock item: %w", err)
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

// ── Store ────────────────────────────────────────────────────────────────────

// PostParts implements Repository: PostParts in one transaction; a concurrent
// key collision is rolled back and replayed (or refused as reused).
func (s *PGStore) PostParts(ctx context.Context, actor audit.Actor, scope access.Scope, req PartsRequest) (PartsResult, bool, error) {
	var res PartsResult
	var replayed bool
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		res, replayed, err = PostParts(ctx, tx, actor, scope, req)
		return err
	})
	if IsKeyRace(err) {
		res, err = ReplayParts(ctx, s.pool, actor.UserID, req)
		if err != nil {
			return PartsResult{}, false, err
		}
		return res, true, nil
	}
	if err != nil {
		return PartsResult{}, false, err
	}
	return res, replayed, nil
}

// PartsBalances implements Repository: the SKU's balances at the locations in scope.
func (s *PGStore) PartsBalances(ctx context.Context, itemID string, scopeIDs []string, all bool) ([]PartsBalance, error) {
	if err := itemExists(ctx, s.pool, itemID); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT `+partsBalanceColumns+partsBalanceFrom+`
		WHERE b.stock_item_id = $1 AND ($2 OR b.location_id = ANY($3::uuid[])) ORDER BY l.code`, itemID, all, scopeIDs)
	if err != nil {
		return nil, fmt.Errorf("list balances: %w", err)
	}
	defer rows.Close()
	out := []PartsBalance{}
	for rows.Next() {
		b, err := scanPartsBalance(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// PartsHistory implements Repository: newest first.
func (s *PGStore) PartsHistory(ctx context.Context, f PartsHistoryFilter) (PartsPage, error) {
	if err := itemExists(ctx, s.pool, f.ItemID); err != nil {
		return PartsPage{}, err
	}
	where := ` WHERE m.stock_item_id = $1 AND ($2 OR m.to_location_id = ANY($3::uuid[]))
		AND ($4 = '' OR m.to_location_id::text = $4) AND ($5 = '' OR m.movement_type = $5)`
	args := []any{f.ItemID, f.AllLocations, f.ScopeLocationIDs, f.LocationID, f.Type}
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM stock_movements m`+where, args...).Scan(&total); err != nil {
		return PartsPage{}, fmt.Errorf("count parts movements: %w", err)
	}
	rows, err := s.pool.Query(ctx, `SELECT `+partsMovementColumns+partsMovementFrom+where+` ORDER BY m.seq DESC LIMIT $6 OFFSET $7`,
		append(args, f.Limit, (f.Page-1)*f.Limit)...)
	if err != nil {
		return PartsPage{}, fmt.Errorf("list parts movements: %w", err)
	}
	defer rows.Close()
	items := []PartsMovement{}
	for rows.Next() {
		m, err := scanPartsMovement(rows)
		if err != nil {
			return PartsPage{}, err
		}
		items = append(items, m)
	}
	return PartsPage{Items: items, Page: f.Page, Limit: f.Limit, Total: total}, rows.Err()
}
