package vehicles

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/stock"
)

// PGStore is the PostgreSQL Repository.
type PGStore struct{ pool *pgxpool.Pool }

// NewPGStore returns a Repository backed by pool.
func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

const unitColumns = `v.id::text, v.unit_number, v.status, v.brand_id::text, b.name, v.model_id::text, m.name,
	v.variant_id::text, va.name, v.chassis_number, v.chassis_normalised, v.motor_serial, v.battery_serial, v.charger_serial,
	v.location_id::text, l.code, l.name, v.source_type, v.supplier_id::text, s.legal_name,
	to_char(v.received_date, 'YYYY-MM-DD'), v.unit_cost::text, v.cost_currency, v.version, v.created_at, v.updated_at`

const unitFrom = ` FROM vehicle_units v
	JOIN brands b ON b.id = v.brand_id
	JOIN models m ON m.id = v.model_id
	JOIN variants va ON va.id = v.variant_id
	LEFT JOIN locations l ON l.id = v.location_id
	LEFT JOIN suppliers s ON s.id = v.supplier_id`

func scanUnit(row pgx.Row) (Unit, error) { return scanUnitWith(row) }

// scanUnitWith scans unitColumns followed by extra destinations.
func scanUnitWith(row pgx.Row, extra ...any) (Unit, error) {
	var u Unit
	var cost, currency string
	dest := append([]any{&u.ID, &u.UnitNumber, &u.Status, &u.BrandID, &u.BrandName, &u.ModelID, &u.ModelName,
		&u.VariantID, &u.VariantName, &u.ChassisNumber, &u.ChassisNormalised, &u.MotorSerial, &u.BatterySerial, &u.ChargerSerial,
		&u.LocationID, &u.LocationCode, &u.LocationName, &u.SourceType, &u.SupplierID, &u.SupplierName,
		&u.ReceivedDate, &cost, &currency, &u.Version, &u.CreatedAt, &u.UpdatedAt}, extra...)
	err := row.Scan(dest...)
	u.UnitCost, u.CostCurrency = &cost, &currency
	return u, err
}

type querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func queryAll[T any](ctx context.Context, q querier, scan func(pgx.Row) (T, error), sql string, args ...any) ([]T, error) {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []T{}
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func likePattern(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(q) + "%"
}

// List implements Repository. The search matches the unit number and the
// normalised chassis/frame, motor, battery and charger identifiers.
func (s *PGStore) List(ctx context.Context, f ListFilter) (Page[Unit], error) {
	var conds []string
	var args []any
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(args))))
	}
	if f.Query != "" {
		add(`(upper(v.unit_number) LIKE ? ESCAPE '\' OR v.chassis_normalised LIKE ? ESCAPE '\' OR v.motor_normalised LIKE ? ESCAPE '\'
			OR v.battery_normalised LIKE ? ESCAPE '\' OR v.charger_normalised LIKE ? ESCAPE '\')`, likePattern(f.Query))
	}
	for _, c := range []struct{ col, v string }{{"v.status", f.Status}, {"v.brand_id", f.BrandID}, {"v.variant_id", f.VariantID},
		{"v.supplier_id", f.SupplierID}, {"v.purchase_order_id", f.PurchaseOrderID}, {"v.goods_receipt_id", f.GoodsReceiptID}} {
		if c.v != "" {
			add(c.col+" = ?", c.v)
		}
	}
	switch {
	case len(f.LocationIDs) > 0:
		add("v.location_id = ANY(?::uuid[])", f.LocationIDs)
	case !f.AllLocations:
		add("(v.location_id IS NULL OR v.location_id = ANY(?::uuid[]))", f.ScopeLocationIDs)
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM vehicle_units v`+where, args...).Scan(&total); err != nil {
		return Page[Unit]{}, fmt.Errorf("count vehicle units: %w", err)
	}
	n := len(args)
	items, err := queryAll(ctx, s.pool, scanUnit, `SELECT `+unitColumns+unitFrom+where+` ORDER BY v.created_at DESC, v.id LIMIT $`+
		strconv.Itoa(n+1)+` OFFSET $`+strconv.Itoa(n+2), append(args, f.Limit, (f.Page-1)*f.Limit)...)
	if err != nil {
		return Page[Unit]{}, fmt.Errorf("list vehicle units: %w", err)
	}
	return Page[Unit]{Items: items, Page: f.Page, Limit: f.Limit, Total: total}, nil
}

// GetDetail implements Repository.
func (s *PGStore) GetDetail(ctx context.Context, id string) (Detail, error) {
	var d Detail
	var src Source
	row := s.pool.QueryRow(ctx, `SELECT `+unitColumns+`, v.purchase_order_id::text, po.po_number, v.purchase_order_line_id::text,
			v.goods_receipt_id::text, g.grn_number, v.goods_receipt_line_id::text, v.goods_receipt_unit_id::text, g.supplier_reference
		`+unitFrom+`
		LEFT JOIN purchase_orders po ON po.id = v.purchase_order_id
		LEFT JOIN goods_receipts g ON g.id = v.goods_receipt_id
		WHERE v.id = $1`, id)
	u, err := scanUnitWith(row, &src.PurchaseOrderID, &src.PurchaseOrderNumber, &src.PurchaseOrderLineID,
		&src.GoodsReceiptID, &src.GoodsReceiptNumber, &src.GoodsReceiptLineID, &src.GoodsReceiptUnitID, &src.SupplierReference)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, ErrNotFound
	}
	if err != nil {
		return d, fmt.Errorf("get vehicle unit: %w", err)
	}
	src.Type, src.SupplierID, src.SupplierName, src.ReceivedDate = u.SourceType, u.SupplierID, u.SupplierName, u.ReceivedDate
	d.Unit, d.Source = u, src
	if d.Events, err = queryAll(ctx, s.pool, func(r pgx.Row) (Event, error) {
		var e Event
		err := r.Scan(&e.ID, &e.Action, &e.ToStatus, &e.SourceType, &e.SourceID, &e.SourceReference, &e.ActorID, &e.ActorName,
			&e.Note, &e.CreatedAt)
		return e, err
	}, `SELECT e.id::text, e.action, e.to_status, e.source_type, e.source_id::text, e.source_reference, e.actor_id::text,
			u.display_name, e.note, e.created_at
		FROM vehicle_unit_events e LEFT JOIN users u ON u.id = e.actor_id
		WHERE e.vehicle_unit_id = $1 ORDER BY e.seq`, id); err != nil {
		return d, fmt.Errorf("vehicle unit events: %w", err)
	}
	return d, nil
}

// ── Registration ────────────────────────────────────────────────────────────

// Registration is what RegisterFromReceiptUnit created.
type Registration struct {
	ID         string
	UnitNumber string
}

// RegisterFromReceiptUnit turns one ACCEPTED captured unit of a goods receipt
// into a vehicle unit at RECEIVED_PENDING_PDI, inside the caller's transaction:
// receipt posting (D-07) calls it for each accepted unit while it marks the
// receipt POSTED. Every value comes from the stored receipt, purchase order and
// catalogue rows, never from client input: variant (and so brand and model),
// supplier, references, received date, the identifiers as captured and the PO
// line's agreed unit cost (cost basis pending, OQ-29). locationID is optional
// until receipts carry a location (OQ-24). It records the REGISTERED history
// entry, the unit's RECEIPT stock movement (E-01) and an audit event in the
// same transaction.
//
// The database re-checks at commit that the receipt is POSTED (a draft never
// has vehicle units); pass the commit error through CommitError. A chassis/frame
// number already registered (compared in its normalised form, across every
// brand) returns a *DuplicateIdentifierError; the unique index decides when two
// transactions race, and its violation maps to the same error. The caller must
// then roll back: the transaction is aborted.
func RegisterFromReceiptUnit(ctx context.Context, tx pgx.Tx, actor audit.Actor, requestID, receiptUnitID string, locationID *string) (Registration, error) {
	if actor.Type != audit.ActorUser || actor.UserID == "" {
		return Registration{}, errors.New("vehicle units are registered by a signed-in user")
	}
	if _, err := uuid.Parse(receiptUnitID); err != nil {
		return Registration{}, ErrReceiptUnitNotFound
	}
	if locationID != nil {
		if _, err := uuid.Parse(*locationID); err != nil {
			return Registration{}, ErrInvalidLocation
		}
	}
	var (
		condition, lineID, receiptID, poLineID, poID, grnNumber, supplierID string
		brandID, modelID, variantID, cost                                   string
		received                                                            time.Time
		chassis, motor, battery, charger                                    *string
	)
	err := tx.QueryRow(ctx, `SELECT u.condition, u.goods_receipt_line_id::text, u.goods_receipt_id::text,
			gl.purchase_order_line_id::text, g.purchase_order_id::text, g.grn_number, g.received_date, po.supplier_id::text,
			pl.brand_id::text, pl.model_id::text, pl.variant_id::text, pl.unit_cost::text,
			u.chassis_number, u.motor_serial, u.battery_serial, u.charger_serial
		FROM goods_receipt_units u
		JOIN goods_receipt_lines gl ON gl.id = u.goods_receipt_line_id
		JOIN goods_receipts g ON g.id = u.goods_receipt_id
		JOIN purchase_orders po ON po.id = g.purchase_order_id
		JOIN purchase_order_lines pl ON pl.id = gl.purchase_order_line_id
		WHERE u.id = $1
		FOR SHARE OF u`, receiptUnitID).Scan(&condition, &lineID, &receiptID, &poLineID, &poID, &grnNumber, &received, &supplierID,
		&brandID, &modelID, &variantID, &cost, &chassis, &motor, &battery, &charger)
	if errors.Is(err, pgx.ErrNoRows) {
		return Registration{}, ErrReceiptUnitNotFound
	}
	if err != nil {
		return Registration{}, fmt.Errorf("read goods receipt unit: %w", err)
	}
	if condition != "ACCEPTED" {
		return Registration{}, ErrReceiptUnitNotAccepted
	}
	// Friendly early answers; the unique keys are the final word under concurrency.
	// A repeat for the same captured unit is ErrAlreadyRegistered, not a duplicate identifier.
	// The chassis comparison uses the column's own normalisation expression.
	var registered, taken bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM vehicle_units WHERE goods_receipt_unit_id = $1),
			$2::text IS NOT NULL AND EXISTS (SELECT 1 FROM vehicle_units WHERE chassis_normalised = upper(btrim($2::text)))`,
		receiptUnitID, chassis).Scan(&registered, &taken); err != nil {
		return Registration{}, fmt.Errorf("check identifiers: %w", err)
	}
	if registered {
		return Registration{}, ErrAlreadyRegistered
	}
	if taken {
		return Registration{}, &DuplicateIdentifierError{Field: FieldChassisNumber, GoodsReceiptUnitID: receiptUnitID}
	}
	var r Registration
	err = tx.QueryRow(ctx, `INSERT INTO vehicle_units (brand_id, model_id, variant_id, status, chassis_number, motor_serial,
			battery_serial, charger_serial, location_id, source_type, supplier_id, purchase_order_id, purchase_order_line_id,
			goods_receipt_id, goods_receipt_line_id, goods_receipt_unit_id, received_date, unit_cost, registered_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		RETURNING id::text, unit_number`,
		brandID, modelID, variantID, StatusReceivedPendingPDI, chassis, motor, battery, charger, locationID, SourceGoodsReceipt,
		supplierID, poID, poLineID, receiptID, lineID, receiptUnitID, received, cost, actor.UserID).Scan(&r.ID, &r.UnitNumber)
	if err != nil {
		return Registration{}, registrationError(err, receiptUnitID)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO vehicle_unit_events (vehicle_unit_id, action, to_status, source_type, source_id,
			source_reference, actor_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		r.ID, ActionRegistered, StatusReceivedPendingPDI, SourceGoodsReceipt, receiptID, grnNumber, actor.UserID); err != nil {
		return Registration{}, fmt.Errorf("record vehicle unit history: %w", err)
	}
	// The unit enters stock through its RECEIPT movement (E-01); the database
	// refuses at commit a unit without one.
	movementID, err := stock.RecordReceipt(ctx, tx, actor, stock.Receipt{UnitID: r.ID, GoodsReceiptID: receiptID,
		GoodsReceiptNumber: grnNumber, GoodsReceiptUnitID: receiptUnitID, LocationID: locationID, RequestID: requestID})
	if err != nil {
		return Registration{}, err
	}
	// The cost is left out of the audit summary: audit readers need not hold vehicle_units.cost.view.
	if err := audit.Record(ctx, tx, audit.Event{Actor: actor, Action: "vehicle_unit.registered", EntityType: "vehicle_unit",
		EntityID: r.ID, Reference: r.UnitNumber, RequestID: requestID, Summary: audit.Summary{
			"status": StatusReceivedPendingPDI, "sourceType": SourceGoodsReceipt, "goodsReceiptId": receiptID,
			"goodsReceiptNumber": grnNumber, "goodsReceiptUnitId": receiptUnitID, "purchaseOrderId": poID,
			"brandId": brandID, "variantId": variantID, "movementId": movementID,
		}}); err != nil {
		return Registration{}, err
	}
	return r, nil
}

// registrationError maps the named constraints only; any other database error stays an internal error.
func registrationError(err error, receiptUnitID string) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return fmt.Errorf("register vehicle unit: %w", err)
	}
	switch {
	case pgErr.Code == "23505" && pgErr.ConstraintName == "vehicle_units_receipt_unit_key":
		return ErrAlreadyRegistered
	case pgErr.Code == "23505" && pgErr.ConstraintName == "vehicle_units_chassis_key":
		return &DuplicateIdentifierError{Field: FieldChassisNumber, GoodsReceiptUnitID: receiptUnitID}
	case pgErr.Code == "23503" && pgErr.ConstraintName == "vehicle_units_location_id_fkey":
		return ErrInvalidLocation
	}
	return fmt.Errorf("register vehicle unit: %w", err)
}

// CommitError maps the commit-time checks of a transaction that registered
// units: a receipt that is not POSTED (or a unit that was not accepted, or a
// different received date) at commit.
func CommitError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
		return err
	}
	switch pgErr.ConstraintName {
	case "vehicle_units_receipt_posted":
		return ErrReceiptNotPosted
	case "vehicle_units_receipt_unit_accepted":
		return ErrReceiptUnitNotAccepted
	}
	return err
}
