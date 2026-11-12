package stock

// The stock movement timeline (E-02; ADR-027): a read-only, paginated view of
// the whole ledger across items and locations. It reads stock_movements as
// recorded; nothing here writes, corrects or derives a movement.
//
// Who sees what — no new permission: a movement is part of its item's history,
// so a row is visible exactly when its item's own history is.
//   - VEHICLE_UNIT rows: vehicle_units.view, and the unit is in the caller's
//     location scope by the same rule as the vehicle list/detail (D-04/D-08): its
//     current location is in scope, or it has none yet (OQ-24).
//   - STOCK_ITEM rows: stock.parts.view, at a location in scope (E-04 history rule).
// Seeing a movement does not open its linked records. The source reference/ID
// is returned only when the caller could open that source: goods receipts need
// goods_receipts.view, stock counts stock.adjustment.view; PDI inspections are
// readable with vehicle_units.view (D-06), supplier challan numbers with
// stock.parts.view (E-04, no linked record), staff actions have none. The
// supplier needs suppliers.view (E-04). A location outside the scope is named by
// ID only (D-08). No cost, price, document or storage reference is part of a
// movement. Per-user brand scope does not exist yet (RB-5).

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
)

// Item types, movement types and source types a timeline filter accepts
// (exactly the ledger's CHECK constraints; nothing invented).
var (
	TimelineItemTypes     = []string{ItemVehicleUnit, ItemStockItem}
	TimelineMovementTypes = []string{TypeReceipt, TypePDIPass, TypeQualityHold, TypeReject, TypeDisplayOn, TypeDisplayOff,
		TypePartsReceipt, TypePartsIssue, TypeCountAdjustment}
	TimelineSourceTypes = []string{SourceGoodsReceipt, SourcePDIInspection, SourceStaffAction, SourceSupplierDelivery, SourceStockCount}
)

// TimelineMovement is one ledger entry as the caller may see it.
type TimelineMovement struct {
	ID           string    `json:"id"`
	ItemType     string    `json:"itemType" enums:"VEHICLE_UNIT,STOCK_ITEM" example:"VEHICLE_UNIT"`
	MovementType string    `json:"movementType" enums:"RECEIPT,PDI_PASS,QUALITY_HOLD,REJECT,DISPLAY_ON,DISPLAY_OFF,PARTS_RECEIPT,PARTS_ISSUE,COUNT_ADJUSTMENT" example:"PDI_PASS"`
	CreatedAt    time.Time `json:"createdAt"`
	// VEHICLE_UNIT rows: the unit; null on STOCK_ITEM rows.
	VehicleUnit *TimelineUnit `json:"vehicleUnit" extensions:"x-nullable"`
	// STOCK_ITEM rows: the SKU; null on VEHICLE_UNIT rows.
	StockItem *TimelineSKU `json:"stockItem" extensions:"x-nullable"`
	// Quantities before and after (decimal strings: 0 -> 1 for a vehicle receipt, 1 -> 1 after); uom on STOCK_ITEM rows.
	QuantityBefore string  `json:"quantityBefore" example:"1.000"`
	QuantityAfter  string  `json:"quantityAfter" example:"1.000"`
	UOM            *string `json:"uom" extensions:"x-nullable" example:"PCS"`
	// Statuses: VEHICLE_UNIT rows only (fromStatus null on a receipt); null on STOCK_ITEM rows.
	FromStatus *string `json:"fromStatus" extensions:"x-nullable" example:"RECEIVED_PENDING_PDI"`
	ToStatus   *string `json:"toStatus" extensions:"x-nullable" example:"AVAILABLE"`
	// Locations as recorded; code/name are null for a location outside the caller's scope.
	FromLocationID   *string `json:"fromLocationId" extensions:"x-nullable"`
	FromLocationCode *string `json:"fromLocationCode" extensions:"x-nullable"`
	FromLocationName *string `json:"fromLocationName" extensions:"x-nullable"`
	ToLocationID     *string `json:"toLocationId" extensions:"x-nullable"`
	ToLocationCode   *string `json:"toLocationCode" extensions:"x-nullable" example:"SR1"`
	ToLocationName   *string `json:"toLocationName" extensions:"x-nullable"`
	SourceType       string  `json:"sourceType" enums:"GOODS_RECEIPT,PDI_INSPECTION,STAFF_ACTION,SUPPLIER_DELIVERY,STOCK_COUNT" example:"PDI_INSPECTION"`
	// SourceID/SourceReference: null when the source has none, or when sourceHidden is true.
	SourceID        *string `json:"sourceId" extensions:"x-nullable"`
	SourceReference *string `json:"sourceReference" extensions:"x-nullable" example:"PDI-000001"`
	// SourceHidden: the movement has a source record the caller may not open (permission-derived);
	// its reason and note are then null too.
	SourceHidden bool `json:"sourceHidden"`
	// SupplierID/SupplierName: PARTS_RECEIPT rows; null without suppliers.view.
	SupplierID   *string `json:"supplierId" extensions:"x-nullable"`
	SupplierName *string `json:"supplierName" extensions:"x-nullable"`
	Reason       *string `json:"reason" extensions:"x-nullable"`
	Note         *string `json:"note" extensions:"x-nullable"`
	// Actor: user ID and display name only.
	ActorID   string  `json:"actorId"`
	ActorName *string `json:"actorName" extensions:"x-nullable"`
} //@name TimelineMovement

// TimelineUnit identifies the vehicle unit of a movement.
type TimelineUnit struct {
	ID            string  `json:"id"`
	UnitNumber    string  `json:"unitNumber" example:"MAE-VEH-000001"`
	ChassisNumber *string `json:"chassisNumber" extensions:"x-nullable" example:"CH-0001"`
} //@name TimelineVehicleUnit

// TimelineSKU identifies the stock item of a movement.
type TimelineSKU struct {
	ID   string `json:"id"`
	SKU  string `json:"sku" example:"BRK-PAD-01"`
	Name string `json:"name"`
} //@name TimelineStockItem

// TimelinePage is one page of movements.
type TimelinePage struct {
	Items []TimelineMovement `json:"items"`
	Page  int                `json:"page" example:"1"`
	Limit int                `json:"limit" example:"25"`
	Total int                `json:"total" example:"0"`
} //@name TimelineMovementPage

// TimelineQuery is the raw list request.
type TimelineQuery struct {
	ItemType, VehicleUnitID, StockItemID, Query, MovementType, SourceType, SourceReference, ActorID, LocationID,
	From, To, Order, Page, Limit string
}

// TimelineFilter is a validated, scoped list request.
type TimelineFilter struct {
	ItemType, VehicleUnitID, StockItemID, MovementType, SourceType, ActorID, LocationID string
	// Query: upper-cased, trimmed text matched as a substring of the unit number, the
	// normalised chassis/frame number or the normalised SKU code.
	Query string
	// SourceReference: case-insensitive exact match, only on sources the caller may open.
	SourceReference string
	From, To        *time.Time
	Newest          bool
	Page, Limit     int
	// Visibility, from the caller's access.
	Vehicles, Parts    bool
	ShowGRN, ShowCount bool
	ScopeIDs           []string
	AllLocations       bool
	ID                 string // detail only
}

func likePattern(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(q) + "%"
}

// ErrMovementNotFound: no such movement, or one the caller may not see.
var ErrMovementNotFound = errors.New("stock movement not found")

func validateTimeline(q TimelineQuery) (TimelineFilter, []FieldProblem) {
	f := TimelineFilter{Newest: true, Page: 1, Limit: 25}
	var probs []FieldProblem
	bad := func(field, msg string) { probs = append(probs, FieldProblem{field, msg}) }
	oneOf := func(field, v string, allowed []string) string {
		v = strings.TrimSpace(v)
		if v != "" && !slices.Contains(allowed, v) {
			bad(field, "must be one of "+strings.Join(allowed, ", "))
			return ""
		}
		return v
	}
	id := func(field, v string) string {
		v = strings.TrimSpace(v)
		if v == "" {
			return ""
		}
		if _, err := uuid.Parse(v); err != nil {
			bad(field, "must be a valid ID")
			return ""
		}
		return strings.ToLower(v)
	}
	f.ItemType = oneOf("itemType", q.ItemType, TimelineItemTypes)
	f.MovementType = oneOf("movementType", q.MovementType, TimelineMovementTypes)
	f.SourceType = oneOf("sourceType", q.SourceType, TimelineSourceTypes)
	f.VehicleUnitID = id("vehicleUnitId", q.VehicleUnitID)
	f.StockItemID = id("stockItemId", q.StockItemID)
	f.ActorID = id("actorId", q.ActorID)
	f.LocationID = id("locationId", q.LocationID)
	if u := strings.TrimSpace(q.Query); u != "" {
		if len(u) > 100 {
			bad("q", "must be at most 100 characters")
		} else {
			f.Query = strings.ToUpper(u)
		}
	}
	if r := strings.TrimSpace(q.SourceReference); r != "" {
		if len(r) > 100 {
			bad("sourceReference", "must be at most 100 characters")
		} else {
			f.SourceReference = r
		}
	}
	instant := func(field, v string) *time.Time {
		v = strings.TrimSpace(v)
		if v == "" {
			return nil
		}
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			bad(field, "must be an RFC 3339 date-time with a time zone, e.g. 2026-01-15T00:00:00+05:30")
			return nil
		}
		return &t
	}
	f.From, f.To = instant("from", q.From), instant("to", q.To)
	if f.From != nil && f.To != nil && !f.From.Before(*f.To) {
		bad("to", "must be after from")
	}
	switch q.Order {
	case "", "desc":
	case "asc":
		f.Newest = false
	default:
		bad("order", "must be asc or desc")
	}
	if q.Page != "" {
		if n, err := strconv.Atoi(q.Page); err != nil || n < 1 || n > 100000 {
			bad("page", "must be a whole number from 1")
		} else {
			f.Page = n
		}
	}
	if q.Limit != "" {
		if n, err := strconv.Atoi(q.Limit); err != nil || n < 1 || n > 100 {
			bad("limit", "must be a whole number from 1 to 100")
		} else {
			f.Limit = n
		}
	}
	// Filters that only one item type has.
	if f.VehicleUnitID != "" && f.StockItemID != "" {
		bad("stockItemId", "cannot be combined with vehicleUnitId")
	}
	return f, probs
}

// scopeTimeline adds the caller's visibility to f: which item types, which
// sources and which locations.
func scopeTimeline(a access.Access, f *TimelineFilter) error {
	f.Vehicles, f.Parts = a.Has(access.PermVehicleUnitView), a.Has(access.PermPartsView)
	if !f.Vehicles && !f.Parts {
		return ErrForbidden
	}
	f.ShowGRN, f.ShowCount = a.Has(access.PermGoodsReceiptView), a.Has(access.PermAdjustmentView)
	ids, all, err := a.Scope.Restrict(nil)
	if err != nil {
		return err
	}
	f.ScopeIDs, f.AllLocations = ids, all
	return nil
}

// Timeline lists the ledger entries the caller may see, newest first by
// default (vehicle_units.view and/or stock.parts.view).
func (s *Service) Timeline(ctx context.Context, a access.Access, q TimelineQuery) (TimelinePage, error) {
	if !a.Has(access.PermVehicleUnitView) && !a.Has(access.PermPartsView) {
		return TimelinePage{}, ErrForbidden
	}
	f, probs := validateTimeline(q)
	if f.LocationID != "" && !a.Scope.Allows(f.LocationID) {
		probs = append(probs, FieldProblem{"locationId", "is not one of your locations"})
	}
	if len(probs) > 0 {
		return TimelinePage{}, &ValidationError{Fields: probs}
	}
	if err := scopeTimeline(a, &f); err != nil {
		return TimelinePage{}, err
	}
	p, err := s.repo.Timeline(ctx, f)
	if err != nil {
		return TimelinePage{}, err
	}
	for i := range p.Items {
		shapeTimeline(&p.Items[i], a, f)
	}
	return p, nil
}

// TimelineMovement returns one ledger entry the caller may see; any other ID
// (unknown, malformed, out of scope or of an item type the caller may not view)
// is ErrMovementNotFound.
func (s *Service) TimelineMovement(ctx context.Context, a access.Access, id string) (TimelineMovement, error) {
	if !a.Has(access.PermVehicleUnitView) && !a.Has(access.PermPartsView) {
		return TimelineMovement{}, ErrForbidden
	}
	if _, err := uuid.Parse(id); err != nil {
		return TimelineMovement{}, ErrMovementNotFound
	}
	f := TimelineFilter{ID: strings.ToLower(id), Page: 1, Limit: 1, Newest: true}
	if err := scopeTimeline(a, &f); err != nil {
		return TimelineMovement{}, err
	}
	p, err := s.repo.Timeline(ctx, f)
	if err != nil {
		return TimelineMovement{}, err
	}
	if len(p.Items) == 0 {
		return TimelineMovement{}, ErrMovementNotFound
	}
	m := p.Items[0]
	shapeTimeline(&m, a, f)
	return m, nil
}

// shapeTimeline removes what the caller may not see from a visible movement.
func shapeTimeline(m *TimelineMovement, a access.Access, f TimelineFilter) {
	hide := (m.SourceType == SourceGoodsReceipt && !f.ShowGRN) || (m.SourceType == SourceStockCount && !f.ShowCount)
	if hide {
		// The reason and note of a hidden source's movement come from that record (e.g. a count's approval).
		m.SourceID, m.SourceReference, m.SourceHidden = nil, nil, true
		m.Reason, m.Note = nil, nil
	}
	if !a.Has(access.PermSupplierView) {
		m.SupplierID, m.SupplierName = nil, nil
	}
	if m.FromLocationID != nil && !a.Scope.Allows(*m.FromLocationID) {
		m.FromLocationCode, m.FromLocationName = nil, nil
	}
	if m.ToLocationID != nil && !a.Scope.Allows(*m.ToLocationID) {
		m.ToLocationCode, m.ToLocationName = nil, nil
	}
}

// Timeline implements Repository. Visibility, scope and every filter are in the
// one WHERE that both the count and the page use. Joins are to one row each
// (unit, SKU, locations, supplier, actor), so no movement repeats; the order is
// the ledger sequence (unique), so pages are stable.
func (s *PGStore) Timeline(ctx context.Context, f TimelineFilter) (TimelinePage, error) {
	var conds []string
	var args []any
	add := func(cond string, vs ...any) {
		for _, v := range vs {
			args = append(args, v)
			cond = strings.Replace(cond, "?", "$"+strconv.Itoa(len(args)), 1)
		}
		conds = append(conds, cond)
	}
	// Visibility: item type permission and location scope.
	var vis []string
	if f.Vehicles {
		if f.AllLocations {
			vis = append(vis, "m.item_type = 'VEHICLE_UNIT'")
		} else {
			args = append(args, f.ScopeIDs)
			vis = append(vis, "(m.item_type = 'VEHICLE_UNIT' AND (v.location_id IS NULL OR v.location_id = ANY($"+strconv.Itoa(len(args))+"::uuid[])))")
		}
	}
	if f.Parts {
		if f.AllLocations {
			vis = append(vis, "m.item_type = 'STOCK_ITEM'")
		} else {
			args = append(args, f.ScopeIDs)
			vis = append(vis, "(m.item_type = 'STOCK_ITEM' AND m.to_location_id = ANY($"+strconv.Itoa(len(args))+"::uuid[]))")
		}
	}
	if len(vis) == 0 {
		return TimelinePage{Items: []TimelineMovement{}, Page: f.Page, Limit: f.Limit}, nil
	}
	conds = append(conds, "("+strings.Join(vis, " OR ")+")")
	if f.ID != "" {
		add("m.id = ?::uuid", f.ID)
	}
	if f.ItemType != "" {
		add("m.item_type = ?", f.ItemType)
	}
	if f.VehicleUnitID != "" {
		add("m.vehicle_unit_id = ?::uuid", f.VehicleUnitID)
	}
	if f.StockItemID != "" {
		add("m.stock_item_id = ?::uuid", f.StockItemID)
	}
	if f.Query != "" {
		// Unit number or chassis/frame (normalised), or the SKU code (normalised), as a substring.
		add(`(upper(v.unit_number) LIKE ? ESCAPE '\' OR v.chassis_normalised LIKE ? ESCAPE '\' OR si.sku_normalised LIKE ? ESCAPE '\')`,
			likePattern(f.Query), likePattern(f.Query), likePattern(f.Query))
	}
	if f.MovementType != "" {
		add("m.movement_type = ?", f.MovementType)
	}
	if f.SourceType != "" {
		add("m.source_type = ?", f.SourceType)
	}
	if f.SourceReference != "" {
		// Only sources the caller may open match, so a hidden reference cannot be probed.
		hidden := []string{}
		if !f.ShowGRN {
			hidden = append(hidden, SourceGoodsReceipt)
		}
		if !f.ShowCount {
			hidden = append(hidden, SourceStockCount)
		}
		add("(lower(m.source_reference) = lower(?) AND NOT (m.source_type = ANY(?::text[])))", f.SourceReference, hidden)
	}
	if f.ActorID != "" {
		add("m.actor_id = ?::uuid", f.ActorID)
	}
	if f.LocationID != "" {
		add("m.to_location_id = ?::uuid", f.LocationID)
	}
	if f.From != nil {
		add("m.created_at >= ?", *f.From)
	}
	if f.To != nil {
		add("m.created_at < ?", *f.To)
	}
	from := ` FROM stock_movements m
		LEFT JOIN vehicle_units v ON v.id = m.vehicle_unit_id
		LEFT JOIN stock_items si ON si.id = m.stock_item_id`
	where := " WHERE " + strings.Join(conds, " AND ")
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*)`+from+where, args...).Scan(&total); err != nil {
		return TimelinePage{}, fmt.Errorf("count stock movements: %w", err)
	}
	order := "m.seq DESC"
	if !f.Newest {
		order = "m.seq"
	}
	n := len(args)
	args = append(args, f.Limit, (f.Page-1)*f.Limit)
	rows, err := s.pool.Query(ctx, `SELECT m.id::text, m.item_type, m.movement_type, m.created_at,
			v.id::text, v.unit_number, v.chassis_number, si.id::text, si.sku, si.name,
			m.quantity_before::text, m.quantity_after::text, m.uom, m.from_status, m.to_status,
			m.from_location_id::text, fl.code, fl.name, m.to_location_id::text, tl.code, tl.name,
			m.source_type, m.source_id::text, m.source_reference, m.supplier_id::text, sp.legal_name, m.reason, m.note,
			m.actor_id::text, u.display_name`+from+`
		LEFT JOIN locations fl ON fl.id = m.from_location_id
		LEFT JOIN locations tl ON tl.id = m.to_location_id
		LEFT JOIN suppliers sp ON sp.id = m.supplier_id
		LEFT JOIN users u ON u.id = m.actor_id`+where+` ORDER BY `+order+
		` LIMIT $`+strconv.Itoa(n+1)+` OFFSET $`+strconv.Itoa(n+2), args...)
	if err != nil {
		return TimelinePage{}, fmt.Errorf("list stock movements: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (TimelineMovement, error) {
		var m TimelineMovement
		var unitID, unitNumber, skuID, sku, skuName *string
		var chassis *string
		err := r.Scan(&m.ID, &m.ItemType, &m.MovementType, &m.CreatedAt, &unitID, &unitNumber, &chassis, &skuID, &sku, &skuName,
			&m.QuantityBefore, &m.QuantityAfter, &m.UOM, &m.FromStatus, &m.ToStatus,
			&m.FromLocationID, &m.FromLocationCode, &m.FromLocationName, &m.ToLocationID, &m.ToLocationCode, &m.ToLocationName,
			&m.SourceType, &m.SourceID, &m.SourceReference, &m.SupplierID, &m.SupplierName, &m.Reason, &m.Note,
			&m.ActorID, &m.ActorName)
		if unitID != nil && unitNumber != nil {
			m.VehicleUnit = &TimelineUnit{ID: *unitID, UnitNumber: *unitNumber, ChassisNumber: chassis}
		}
		if skuID != nil && sku != nil && skuName != nil {
			m.StockItem = &TimelineSKU{ID: *skuID, SKU: *sku, Name: *skuName}
		}
		return m, err
	})
	if err != nil {
		return TimelinePage{}, fmt.Errorf("list stock movements: %w", err)
	}
	return TimelinePage{Items: items, Page: f.Page, Limit: f.Limit, Total: total}, nil
}
