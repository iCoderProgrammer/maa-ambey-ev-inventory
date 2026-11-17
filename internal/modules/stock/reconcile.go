package stock

// Ledger reconciliation (E-06; ADR-028; LED-001…LED-004, INV-009). A read-only
// comparison of the current records with the stock movement ledger:
//
//   - Vehicle units: each unit's status and location against its latest
//     movement, that it still holds one item, that its history starts with its
//     RECEIPT, and that every movement starts where the previous one ended.
//   - Parts balances (per SKU and location): each balance against the quantity
//     after its latest movement (0 without one), a ledger location without a
//     balance row, negative balances, and that every movement starts from the
//     quantity the previous one left (the first from 0).
//
// The database already refuses these disagreements at commit
// (vehicle_units_ledger_reconciles, stock_item_balances_ledger_reconciles);
// reconciliation is the independent after-the-fact check that detects drift if
// those guards were ever bypassed (e.g. a direct change by a database owner).
//
// Nothing is written: the queries run in one READ ONLY, REPEATABLE READ
// transaction, so the database itself refuses any write and every query sees the
// same snapshot (a movement committed during a run is either wholly in it or not
// at all). There is no run table and no repair: corrections use the approved
// adjustment flow (E-05) or a future approved reversal flow (OQ-33).
//
// Who sees what: the report needs stock.reconciliation.view, and its rows the
// item's own view rule (as in the timeline, E-02): vehicle units need
// vehicle_units.view and a current location in scope (or none yet); parts
// balances need stock.parts.view at a location in scope. Locations outside the
// scope are named by ID only. No cost is read. Per-user brand scope does not
// exist yet (RB-5).

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
)

// Discrepancy types. Each names one ledger rule a record breaks.
const (
	// The unit has no stock movement at all (LED-003: every unit has its RECEIPT).
	DiscVehicleNoMovements = "VEHICLE_NO_MOVEMENTS"
	// The unit's history does not start with its RECEIPT (INV-001, LED-003).
	DiscVehicleMissingReceipt = "VEHICLE_MISSING_RECEIPT"
	// A RECEIPT appears after the unit's first movement (a unit is received once).
	DiscVehicleRepeatedReceipt = "VEHICLE_REPEATED_RECEIPT"
	// The unit's status is not the status after its latest movement (INV-009).
	DiscVehicleStatusMismatch = "VEHICLE_STATUS_MISMATCH"
	// The unit's location is not the location after its latest movement (INV-009).
	DiscVehicleLocationMismatch = "VEHICLE_LOCATION_MISMATCH"
	// The latest movement does not leave the unit held (quantity after is not 1).
	DiscVehicleNotHeld = "VEHICLE_NOT_HELD"
	// A movement does not start at the status/location/quantity the previous one ended at.
	DiscVehicleBrokenChain = "VEHICLE_BROKEN_CHAIN"
	// The balance is not the quantity after its latest movement (0 without one) (LED-003).
	DiscPartsBalanceMismatch = "PARTS_BALANCE_MISMATCH"
	// Movements exist for a SKU and location that has no balance record.
	DiscPartsBalanceMissing = "PARTS_BALANCE_MISSING"
	// The balance is below zero (LED-004).
	DiscPartsNegativeBalance = "PARTS_NEGATIVE_BALANCE"
	// A movement does not start from the quantity the previous one left (the first from 0).
	DiscPartsBrokenChain = "PARTS_BROKEN_CHAIN"
)

// Severities. HIGH: the current record disagrees with the ledger, or the ledger
// cannot explain where the item came from (stock figures are wrong). MEDIUM:
// the history is internally inconsistent while the current record still matches
// its latest movement.
const (
	SeverityHigh   = "HIGH"
	SeverityMedium = "MEDIUM"
)

// ReconcileTypes and ReconcileSeverities are the values a filter accepts.
var (
	ReconcileTypes = []string{DiscVehicleNoMovements, DiscVehicleMissingReceipt, DiscVehicleRepeatedReceipt,
		DiscVehicleStatusMismatch, DiscVehicleLocationMismatch, DiscVehicleNotHeld, DiscVehicleBrokenChain,
		DiscPartsBalanceMismatch, DiscPartsBalanceMissing, DiscPartsNegativeBalance, DiscPartsBrokenChain}
	ReconcileSeverities = []string{SeverityHigh, SeverityMedium}
)

var discSeverity = map[string]string{
	DiscVehicleNoMovements: SeverityHigh, DiscVehicleMissingReceipt: SeverityHigh, DiscVehicleRepeatedReceipt: SeverityMedium,
	DiscVehicleStatusMismatch: SeverityHigh, DiscVehicleLocationMismatch: SeverityHigh, DiscVehicleNotHeld: SeverityHigh,
	DiscVehicleBrokenChain: SeverityMedium, DiscPartsBalanceMismatch: SeverityHigh, DiscPartsBalanceMissing: SeverityHigh,
	DiscPartsNegativeBalance: SeverityHigh, DiscPartsBrokenChain: SeverityMedium,
}

// ReconcileStatusCompleted is the only status a returned report has: a failed
// run returns an error response instead of a partial report.
const ReconcileStatusCompleted = "COMPLETED"

// ReconcileUnit identifies the vehicle unit of a discrepancy.
type ReconcileUnit struct {
	ID            string  `json:"id"`
	UnitNumber    string  `json:"unitNumber" example:"MAE-VEH-000001"`
	ChassisNumber *string `json:"chassisNumber" extensions:"x-nullable" example:"CH-0001"`
} //@name ReconciliationVehicleUnit

// ReconcileSKU identifies the SKU of a parts discrepancy.
type ReconcileSKU struct {
	ID   string `json:"id"`
	SKU  string `json:"sku" example:"BRK-PAD-01"`
	Name string `json:"name"`
	UOM  string `json:"uom" example:"PCS"`
} //@name ReconciliationStockItem

// ReconcileDiscrepancy is one disagreement between a record and the ledger.
type ReconcileDiscrepancy struct {
	// Reference is stable for the same finding across runs: TYPE:item ID[:location ID][:movement ID].
	Reference string `json:"reference" example:"VEHICLE_STATUS_MISMATCH:3f2b8c1e-9a4d-4e6f-8b7a-1c2d3e4f5a6b"`
	ItemType  string `json:"itemType" enums:"VEHICLE_UNIT,STOCK_ITEM" example:"VEHICLE_UNIT"`
	Type      string `json:"type" enums:"VEHICLE_NO_MOVEMENTS,VEHICLE_MISSING_RECEIPT,VEHICLE_REPEATED_RECEIPT,VEHICLE_STATUS_MISMATCH,VEHICLE_LOCATION_MISMATCH,VEHICLE_NOT_HELD,VEHICLE_BROKEN_CHAIN,PARTS_BALANCE_MISMATCH,PARTS_BALANCE_MISSING,PARTS_NEGATIVE_BALANCE,PARTS_BROKEN_CHAIN" example:"VEHICLE_STATUS_MISMATCH"`
	Severity  string `json:"severity" enums:"HIGH,MEDIUM" example:"HIGH"`
	// VehicleUnit: VEHICLE_UNIT rows; null on STOCK_ITEM rows.
	VehicleUnit *ReconcileUnit `json:"vehicleUnit" extensions:"x-nullable"`
	// StockItem: STOCK_ITEM rows; null on VEHICLE_UNIT rows.
	StockItem *ReconcileSKU `json:"stockItem" extensions:"x-nullable"`
	// Brand of a vehicle unit (parts have no brand); null on STOCK_ITEM rows.
	BrandID   *string `json:"brandId" extensions:"x-nullable"`
	BrandName *string `json:"brandName" extensions:"x-nullable" example:"Brand A"`
	// The record's location (the unit's current location, the balance's location); code/name
	// are null when it is outside the caller's scope, all three when there is none.
	LocationID   *string `json:"locationId" extensions:"x-nullable"`
	LocationCode *string `json:"locationCode" extensions:"x-nullable" example:"SR1"`
	LocationName *string `json:"locationName" extensions:"x-nullable"`
	// Expected: the value the ledger implies. Actual: the value the record (or movement) holds.
	Expected    *string `json:"expected" extensions:"x-nullable" example:"AVAILABLE"`
	Actual      *string `json:"actual" extensions:"x-nullable" example:"DISPLAY"`
	Explanation string  `json:"explanation"`
	// MovementIDs: the ledger entries the finding is about (oldest first); may be empty.
	MovementIDs []string  `json:"movementIds"`
	DetectedAt  time.Time `json:"detectedAt"`
} //@name ReconciliationDiscrepancy

// ReconcileCoverage says what this caller's run looked at.
type ReconcileCoverage struct {
	VehicleUnits bool `json:"vehicleUnits"`
	Parts        bool `json:"parts"`
	// AllLocations: every location; otherwise LocationIDs (the caller's scope).
	AllLocations bool     `json:"allLocations"`
	LocationIDs  []string `json:"locationIds"`
} //@name ReconciliationCoverage

// ReconcileSummary counts the whole run (not only the returned page).
type ReconcileSummary struct {
	VehicleUnitsChecked int            `json:"vehicleUnitsChecked" example:"120"`
	PartBalancesChecked int            `json:"partBalancesChecked" example:"40"`
	RecordsChecked      int            `json:"recordsChecked" example:"160"`
	MovementsChecked    int            `json:"movementsChecked" example:"910"`
	Discrepancies       int            `json:"discrepancies" example:"0"`
	ByType              map[string]int `json:"byType"`
	BySeverity          map[string]int `json:"bySeverity"`
} //@name ReconciliationSummary

// ReconcileReport is one reconciliation run: its summary and one page of its
// discrepancies (after the filters).
type ReconcileReport struct {
	// RunID identifies this run in the server logs; runs are not stored.
	RunID       string            `json:"runId"`
	Status      string            `json:"status" enums:"COMPLETED" example:"COMPLETED"`
	RunAt       time.Time         `json:"runAt"`
	CompletedAt time.Time         `json:"completedAt"`
	DurationMs  int64             `json:"durationMs" example:"35"`
	Coverage    ReconcileCoverage `json:"coverage"`
	Summary     ReconcileSummary  `json:"summary"`
	// Items: the discrepancies matching the filters, severity then type then reference.
	Items []ReconcileDiscrepancy `json:"items"`
	Page  int                    `json:"page" example:"1"`
	Limit int                    `json:"limit" example:"50"`
	// Total: discrepancies matching the filters.
	Total int `json:"total" example:"0"`
} //@name ReconciliationReport

// ReconcileQuery is the raw request.
type ReconcileQuery struct {
	ItemType, Type, Severity, LocationID, Page, Limit string
}

// ReconcileFilter is what the repository reads: the caller's visibility only
// (filters and paging are applied to the whole result afterwards, so the summary
// always describes the whole run).
type ReconcileFilter struct {
	Vehicles, Parts bool
	ScopeIDs        []string
	AllLocations    bool
}

// ReconcileLocation is a location as recorded.
type ReconcileLocation struct{ Code, Name string }

// RawDiscrepancy is a finding as the repository reads it: raw values, before
// the caller's shaping.
type RawDiscrepancy struct {
	Type, ItemType   string
	Unit             *ReconcileUnit
	SKU              *ReconcileSKU
	BrandID          *string
	BrandName        *string
	LocationID       *string
	ExpectedStatus   *string
	ActualStatus     *string
	ExpectedLocation *string
	ActualLocation   *string
	ExpectedQuantity *string
	ActualQuantity   *string
	MovementIDs      []string
}

// ReconcileData is everything a run read from one snapshot.
type ReconcileData struct {
	VehicleUnitsChecked, PartBalancesChecked, MovementsChecked int
	Discrepancies                                              []RawDiscrepancy
	Locations                                                  map[string]ReconcileLocation
}

func validateReconcile(q ReconcileQuery) (itemType, typ, severity, location string, page, limit int, probs []FieldProblem) {
	page, limit = 1, 50
	bad := func(field, msg string) { probs = append(probs, FieldProblem{field, msg}) }
	oneOf := func(field, v string, allowed []string) string {
		v = strings.TrimSpace(v)
		if v != "" && !slices.Contains(allowed, v) {
			bad(field, "must be one of "+strings.Join(allowed, ", "))
			return ""
		}
		return v
	}
	itemType = oneOf("itemType", q.ItemType, TimelineItemTypes)
	typ = oneOf("type", q.Type, ReconcileTypes)
	severity = oneOf("severity", q.Severity, ReconcileSeverities)
	if v := strings.TrimSpace(q.LocationID); v != "" {
		if _, err := uuid.Parse(v); err != nil {
			bad("locationId", "must be a valid ID")
		} else {
			location = strings.ToLower(v)
		}
	}
	if q.Page != "" {
		if n, err := strconv.Atoi(q.Page); err != nil || n < 1 || n > 100000 {
			bad("page", "must be a whole number from 1")
		} else {
			page = n
		}
	}
	if q.Limit != "" {
		if n, err := strconv.Atoi(q.Limit); err != nil || n < 1 || n > 100 {
			bad("limit", "must be a whole number from 1 to 100")
		} else {
			limit = n
		}
	}
	return itemType, typ, severity, location, page, limit, probs
}

// Reconcile runs the reconciliation for the caller (stock.reconciliation.view
// plus vehicle_units.view and/or stock.parts.view) and returns its summary and
// one page of discrepancies. It never writes.
func (s *Service) Reconcile(ctx context.Context, a access.Access, q ReconcileQuery) (ReconcileReport, error) {
	if !a.Has(access.PermReconciliationView) {
		return ReconcileReport{}, ErrForbidden
	}
	f := ReconcileFilter{Vehicles: a.Has(access.PermVehicleUnitView), Parts: a.Has(access.PermPartsView)}
	if !f.Vehicles && !f.Parts {
		return ReconcileReport{}, ErrForbidden
	}
	itemType, typ, severity, location, page, limit, probs := validateReconcile(q)
	if location != "" && !a.Scope.Allows(location) {
		probs = append(probs, FieldProblem{"locationId", "is not one of your locations"})
	}
	if len(probs) > 0 {
		return ReconcileReport{}, &ValidationError{Fields: probs}
	}
	ids, all, err := a.Scope.Restrict(nil)
	if err != nil {
		return ReconcileReport{}, err
	}
	f.ScopeIDs, f.AllLocations = ids, all

	r := ReconcileReport{RunID: uuid.NewString(), Status: ReconcileStatusCompleted, RunAt: time.Now().UTC(), Page: page, Limit: limit,
		Coverage: ReconcileCoverage{VehicleUnits: f.Vehicles, Parts: f.Parts, AllLocations: all, LocationIDs: ids}}
	if r.Coverage.LocationIDs == nil {
		r.Coverage.LocationIDs = []string{}
	}
	s.logger().InfoContext(ctx, "stock reconciliation started", slog.String("run_id", r.RunID), slog.String("user_id", a.UserID),
		slog.Bool("vehicle_units", f.Vehicles), slog.Bool("parts", f.Parts), slog.Bool("all_locations", all))
	d, err := s.repo.Reconcile(ctx, f)
	if err != nil {
		s.logger().ErrorContext(ctx, "stock reconciliation failed", slog.String("run_id", r.RunID), slog.String("user_id", a.UserID),
			slog.Int64("duration_ms", time.Since(r.RunAt).Milliseconds()), slog.Any("error", err))
		return ReconcileReport{}, fmt.Errorf("reconcile: %w", err)
	}
	r.CompletedAt = time.Now().UTC()
	r.DurationMs = r.CompletedAt.Sub(r.RunAt).Milliseconds()

	findings := make([]ReconcileDiscrepancy, 0, len(d.Discrepancies))
	for _, raw := range d.Discrepancies {
		findings = append(findings, shapeDiscrepancy(raw, d.Locations, a.Scope, r.RunAt))
	}
	slices.SortFunc(findings, func(x, y ReconcileDiscrepancy) int {
		if c := strings.Compare(severityRank(x.Severity), severityRank(y.Severity)); c != 0 {
			return c
		}
		if c := strings.Compare(x.Type, y.Type); c != 0 {
			return c
		}
		return strings.Compare(x.Reference, y.Reference)
	})
	r.Summary = ReconcileSummary{VehicleUnitsChecked: d.VehicleUnitsChecked, PartBalancesChecked: d.PartBalancesChecked,
		RecordsChecked: d.VehicleUnitsChecked + d.PartBalancesChecked, MovementsChecked: d.MovementsChecked,
		Discrepancies: len(findings), ByType: map[string]int{}, BySeverity: map[string]int{}}
	for _, x := range findings {
		r.Summary.ByType[x.Type]++
		r.Summary.BySeverity[x.Severity]++
	}
	filtered := make([]ReconcileDiscrepancy, 0, len(findings))
	for _, x := range findings {
		if (itemType == "" || x.ItemType == itemType) && (typ == "" || x.Type == typ) && (severity == "" || x.Severity == severity) &&
			(location == "" || (x.LocationID != nil && *x.LocationID == location)) {
			filtered = append(filtered, x)
		}
	}
	r.Total = len(filtered)
	start := min((page-1)*limit, len(filtered))
	r.Items = filtered[start:min(start+limit, len(filtered))]
	s.logger().InfoContext(ctx, "stock reconciliation completed", slog.String("run_id", r.RunID), slog.String("user_id", a.UserID),
		slog.Int("vehicle_units_checked", d.VehicleUnitsChecked), slog.Int("part_balances_checked", d.PartBalancesChecked),
		slog.Int("movements_checked", d.MovementsChecked), slog.Int("discrepancies", len(findings)),
		slog.Int("high", r.Summary.BySeverity[SeverityHigh]), slog.Int("medium", r.Summary.BySeverity[SeverityMedium]),
		slog.Int64("duration_ms", r.DurationMs))
	return r, nil
}

func (s *Service) logger() *slog.Logger {
	if s.log == nil {
		return slog.New(slog.DiscardHandler)
	}
	return s.log
}

func severityRank(s string) string {
	if s == SeverityHigh {
		return "0"
	}
	return "1"
}

// shapeDiscrepancy turns a raw finding into what the caller sees: locations
// outside the scope by ID only, and the explanation.
func shapeDiscrepancy(raw RawDiscrepancy, locs map[string]ReconcileLocation, scope access.Scope, at time.Time) ReconcileDiscrepancy {
	d := ReconcileDiscrepancy{ItemType: raw.ItemType, Type: raw.Type, Severity: discSeverity[raw.Type], VehicleUnit: raw.Unit,
		StockItem: raw.SKU, BrandID: raw.BrandID, BrandName: raw.BrandName, LocationID: raw.LocationID,
		MovementIDs: raw.MovementIDs, DetectedAt: at}
	if d.MovementIDs == nil {
		d.MovementIDs = []string{}
	}
	if raw.LocationID != nil && scope.Allows(*raw.LocationID) {
		if l, ok := locs[*raw.LocationID]; ok {
			d.LocationCode, d.LocationName = &l.Code, &l.Name
		}
	}
	itemID := ""
	if raw.Unit != nil {
		itemID = raw.Unit.ID
	} else if raw.SKU != nil {
		itemID = raw.SKU.ID
	}
	ref := []string{raw.Type, itemID}
	if raw.ItemType == ItemStockItem && raw.LocationID != nil {
		ref = append(ref, *raw.LocationID)
	}
	if raw.Type == DiscVehicleBrokenChain || raw.Type == DiscVehicleRepeatedReceipt || raw.Type == DiscPartsBrokenChain ||
		raw.Type == DiscVehicleMissingReceipt {
		if n := len(raw.MovementIDs); n > 0 {
			ref = append(ref, raw.MovementIDs[n-1])
		}
	}
	d.Reference = strings.Join(ref, ":")

	loc := func(id *string) string {
		if id == nil {
			return "no location"
		}
		if scope.Allows(*id) {
			if l, ok := locs[*id]; ok {
				return l.Code
			}
		}
		return "location " + *id + " (outside your locations)"
	}
	state := func(status, location *string) string {
		st := "no status"
		if status != nil {
			st = *status
		}
		return st + " at " + loc(location)
	}
	str := func(v string) *string { return &v }
	uom := ""
	if raw.SKU != nil {
		uom = " " + raw.SKU.UOM
	}
	qty := func(v *string) *string {
		if v == nil {
			return nil
		}
		return str(*v + uom)
	}
	switch raw.Type {
	case DiscVehicleNoMovements:
		d.Expected, d.Actual = str("a RECEIPT movement"), str("no movements")
		d.Explanation = "The vehicle unit has no stock movement, so the ledger cannot explain its status or location."
	case DiscVehicleMissingReceipt:
		d.Expected, d.Actual = str(TypeReceipt), raw.ActualStatus
		d.Explanation = "The unit's first stock movement is not its RECEIPT, so the ledger does not show how the unit was received."
	case DiscVehicleRepeatedReceipt:
		d.Expected, d.Actual = str("one RECEIPT, first"), str("another RECEIPT later in the history")
		d.Explanation = "A RECEIPT movement appears after the unit's first movement; a unit is received once."
	case DiscVehicleStatusMismatch:
		d.Expected, d.Actual = raw.ExpectedStatus, raw.ActualStatus
		d.Explanation = "The unit's status is not the status its latest stock movement left it in."
	case DiscVehicleLocationMismatch:
		d.Expected, d.Actual = str(loc(raw.ExpectedLocation)), str(loc(raw.ActualLocation))
		d.Explanation = "The unit's location is not the location its latest stock movement left it at."
	case DiscVehicleNotHeld:
		d.Expected, d.Actual = str("1"), raw.ActualQuantity
		d.Explanation = "The unit's latest stock movement does not leave it held (quantity after should be 1)."
	case DiscVehicleBrokenChain:
		d.Expected = str(state(raw.ExpectedStatus, raw.ExpectedLocation) + ", quantity " + deref(raw.ExpectedQuantity))
		d.Actual = str(state(raw.ActualStatus, raw.ActualLocation) + ", quantity " + deref(raw.ActualQuantity))
		d.Explanation = "A stock movement does not start where the unit's previous movement ended; the history has a gap or an inconsistent entry."
	case DiscPartsBalanceMismatch:
		d.Expected, d.Actual = qty(raw.ExpectedQuantity), qty(raw.ActualQuantity)
		d.Explanation = "The balance is not the quantity after the latest stock movement for this SKU at this location (0 when there is none)."
	case DiscPartsBalanceMissing:
		d.Expected, d.Actual = qty(raw.ExpectedQuantity), str("no balance record")
		d.Explanation = "Stock movements exist for this SKU at this location, but there is no balance record."
	case DiscPartsNegativeBalance:
		d.Expected, d.Actual = str("0"+uom+" or more"), qty(raw.ActualQuantity)
		d.Explanation = "The balance is below zero; negative stock is not allowed."
	case DiscPartsBrokenChain:
		d.Expected, d.Actual = qty(raw.ExpectedQuantity), qty(raw.ActualQuantity)
		d.Explanation = "A stock movement does not start from the quantity the previous movement left (the first from 0); the history has a gap or an inconsistent entry."
	}
	return d
}

func deref(s *string) string {
	if s == nil {
		return "none"
	}
	return *s
}

// ── Repository ──────────────────────────────────────────────────────────────

// Reconcile implements Repository: every query in one READ ONLY, REPEATABLE
// READ transaction (one snapshot; the database refuses any write).
func (s *PGStore) Reconcile(ctx context.Context, f ReconcileFilter) (ReconcileData, error) {
	var d ReconcileData
	err := pgx.BeginTxFunc(ctx, s.pool, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}, func(tx pgx.Tx) error {
		var err error
		d, err = reconcile(ctx, tx, f, s.reconcileHook)
		return err
	})
	return d, err
}

// scopeCond returns the SQL condition limiting col to the caller's locations.
func scopeCond(f ReconcileFilter, col string, args *[]any, allowNull bool) string {
	if f.AllLocations {
		return "true"
	}
	*args = append(*args, f.ScopeIDs)
	cond := col + " = ANY($" + strconv.Itoa(len(*args)) + "::uuid[])"
	if allowNull {
		cond = "(" + col + " IS NULL OR " + cond + ")"
	}
	return cond
}

func reconcile(ctx context.Context, tx pgx.Tx, f ReconcileFilter, hook func(context.Context, pgx.Tx) error) (ReconcileData, error) {
	d := ReconcileData{Locations: map[string]ReconcileLocation{}}
	rows, err := tx.Query(ctx, `SELECT id::text, code, name FROM locations`)
	if err != nil {
		return d, fmt.Errorf("read locations: %w", err)
	}
	var id string
	var l ReconcileLocation
	if _, err := pgx.ForEachRow(rows, []any{&id, &l.Code, &l.Name}, func() error {
		d.Locations[id] = l
		return nil
	}); err != nil {
		return d, fmt.Errorf("read locations: %w", err)
	}
	if f.Vehicles {
		if err := reconcileVehicles(ctx, tx, f, &d); err != nil {
			return d, err
		}
	}
	if hook != nil {
		if err := hook(ctx, tx); err != nil {
			return d, err
		}
	}
	if f.Parts {
		if err := reconcileParts(ctx, tx, f, &d); err != nil {
			return d, err
		}
	}
	return d, nil
}

func reconcileVehicles(ctx context.Context, tx pgx.Tx, f ReconcileFilter, d *ReconcileData) error {
	var args []any
	scope := scopeCond(f, "v.location_id", &args, true)
	var units, movements int
	if err := tx.QueryRow(ctx, `SELECT count(DISTINCT v.id), count(m.id)
		FROM vehicle_units v LEFT JOIN stock_movements m ON m.vehicle_unit_id = v.id WHERE `+scope, args...).Scan(&units, &movements); err != nil {
		return fmt.Errorf("count vehicle units: %w", err)
	}
	d.VehicleUnitsChecked += units
	d.MovementsChecked += movements

	// The unit against its latest movement (status, location, still held) and its first movement (a RECEIPT).
	rows, err := tx.Query(ctx, `SELECT v.id::text, v.unit_number, v.chassis_number, v.brand_id::text, b.name, v.status,
			v.location_id::text, lm.id::text, lm.to_status, lm.to_location_id::text, lm.quantity_after::text,
			fm.id::text, fm.movement_type
		FROM vehicle_units v
		JOIN brands b ON b.id = v.brand_id
		LEFT JOIN LATERAL (SELECT id, to_status, to_location_id, quantity_after FROM stock_movements m
			WHERE m.vehicle_unit_id = v.id ORDER BY m.seq DESC LIMIT 1) lm ON true
		LEFT JOIN LATERAL (SELECT id, movement_type FROM stock_movements m
			WHERE m.vehicle_unit_id = v.id ORDER BY m.seq LIMIT 1) fm ON true
		WHERE `+scope+` AND (lm.id IS NULL OR fm.movement_type NOT IN ('RECEIPT', 'OPENING_BALANCE') OR lm.to_status IS DISTINCT FROM v.status
			OR lm.to_location_id IS DISTINCT FROM v.location_id OR lm.quantity_after <> 1)
		ORDER BY v.unit_number`, args...)
	if err != nil {
		return fmt.Errorf("reconcile vehicle units: %w", err)
	}
	var u ReconcileUnit
	var brandID, brandName, status string
	var loc, lastID, lastStatus, lastLoc, lastQty, firstID, firstType *string
	if _, err := pgx.ForEachRow(rows, []any{&u.ID, &u.UnitNumber, &u.ChassisNumber, &brandID, &brandName, &status, &loc,
		&lastID, &lastStatus, &lastLoc, &lastQty, &firstID, &firstType}, func() error {
		unit := u
		base := RawDiscrepancy{ItemType: ItemVehicleUnit, Unit: &unit, BrandID: ptrTo(brandID), BrandName: ptrTo(brandName),
			LocationID: copyPtr(loc)}
		add := func(typ string, mod func(*RawDiscrepancy)) {
			r := base
			r.Type = typ
			mod(&r)
			d.Discrepancies = append(d.Discrepancies, r)
		}
		if lastID == nil {
			add(DiscVehicleNoMovements, func(*RawDiscrepancy) {})
			return nil
		}
		if *firstType != TypeReceipt && *firstType != TypeOpeningBalance {
			add(DiscVehicleMissingReceipt, func(r *RawDiscrepancy) {
				r.ActualStatus = ptrTo(*firstType)
				r.MovementIDs = []string{*firstID}
			})
		}
		if *lastStatus != status {
			add(DiscVehicleStatusMismatch, func(r *RawDiscrepancy) {
				r.ExpectedStatus, r.ActualStatus = copyPtr(lastStatus), ptrTo(status)
				r.MovementIDs = []string{*lastID}
			})
		}
		if !eqPtr(lastLoc, loc) {
			add(DiscVehicleLocationMismatch, func(r *RawDiscrepancy) {
				r.ExpectedLocation, r.ActualLocation = copyPtr(lastLoc), copyPtr(loc)
				r.MovementIDs = []string{*lastID}
			})
		}
		if q := trimQty(*lastQty); q != "1" {
			add(DiscVehicleNotHeld, func(r *RawDiscrepancy) {
				r.ActualQuantity = ptrTo(q)
				r.MovementIDs = []string{*lastID}
			})
		}
		return nil
	}); err != nil {
		return fmt.Errorf("reconcile vehicle units: %w", err)
	}

	// The history itself: every movement after the first starts where the previous one ended, and only the first is a RECEIPT.
	args = nil
	scope = scopeCond(f, "v.location_id", &args, true)
	rows, err = tx.Query(ctx, `WITH m AS (
			SELECT m.id, m.seq, m.vehicle_unit_id, m.movement_type, m.from_status, m.from_location_id, m.quantity_before,
				lag(m.id) OVER w AS prev_id, lag(m.to_status) OVER w AS prev_status,
				lag(m.to_location_id) OVER w AS prev_location, lag(m.quantity_after) OVER w AS prev_quantity,
				row_number() OVER w AS n
			FROM stock_movements m WHERE m.item_type = 'VEHICLE_UNIT'
			WINDOW w AS (PARTITION BY m.vehicle_unit_id ORDER BY m.seq))
		SELECT v.id::text, v.unit_number, v.chassis_number, v.brand_id::text, b.name, v.location_id::text,
			m.id::text, m.prev_id::text, m.movement_type, m.from_status, m.from_location_id::text, m.quantity_before::text,
			m.prev_status, m.prev_location::text, m.prev_quantity::text
		FROM m JOIN vehicle_units v ON v.id = m.vehicle_unit_id JOIN brands b ON b.id = v.brand_id
		WHERE m.n > 1 AND `+scope+` AND (m.movement_type IN ('RECEIPT', 'OPENING_BALANCE') OR m.from_status IS DISTINCT FROM m.prev_status
			OR m.from_location_id IS DISTINCT FROM m.prev_location OR m.quantity_before <> m.prev_quantity)
		ORDER BY m.seq`, args...)
	if err != nil {
		return fmt.Errorf("reconcile vehicle history: %w", err)
	}
	var mid, prevID, mtype string
	var fromStatus, fromLoc, prevStatus, prevLoc *string
	var qBefore, qPrev string
	if _, err := pgx.ForEachRow(rows, []any{&u.ID, &u.UnitNumber, &u.ChassisNumber, &brandID, &brandName, &loc,
		&mid, &prevID, &mtype, &fromStatus, &fromLoc, &qBefore, &prevStatus, &prevLoc, &qPrev}, func() error {
		unit := u
		r := RawDiscrepancy{ItemType: ItemVehicleUnit, Unit: &unit, BrandID: ptrTo(brandID), BrandName: ptrTo(brandName),
			LocationID: copyPtr(loc), MovementIDs: []string{prevID, mid}}
		if mtype == TypeReceipt || mtype == TypeOpeningBalance {
			r.Type = DiscVehicleRepeatedReceipt
		} else {
			r.Type = DiscVehicleBrokenChain
			r.ExpectedStatus, r.ExpectedLocation, r.ExpectedQuantity = copyPtr(prevStatus), copyPtr(prevLoc), ptrTo(trimQty(qPrev))
			r.ActualStatus, r.ActualLocation, r.ActualQuantity = copyPtr(fromStatus), copyPtr(fromLoc), ptrTo(trimQty(qBefore))
		}
		d.Discrepancies = append(d.Discrepancies, r)
		return nil
	}); err != nil {
		return fmt.Errorf("reconcile vehicle history: %w", err)
	}
	return nil
}

func reconcileParts(ctx context.Context, tx pgx.Tx, f ReconcileFilter, d *ReconcileData) error {
	var args []any
	scope := scopeCond(f, "x.location_id", &args, false)
	// Every SKU/location with a balance or a movement, against its latest movement.
	rows, err := tx.Query(ctx, `WITH last AS (
			SELECT DISTINCT ON (stock_item_id, to_location_id) stock_item_id, to_location_id AS location_id, id, quantity_after
			FROM stock_movements WHERE item_type = 'STOCK_ITEM' ORDER BY stock_item_id, to_location_id, seq DESC),
		counted AS (
			SELECT stock_item_id, to_location_id AS location_id, count(*) AS movements
			FROM stock_movements WHERE item_type = 'STOCK_ITEM' GROUP BY 1, 2),
		x AS (
			SELECT coalesce(b.stock_item_id, l.stock_item_id) AS stock_item_id, coalesce(b.location_id, l.location_id) AS location_id,
				b.quantity, b.stock_item_id IS NOT NULL AS has_balance, l.id AS last_id, l.quantity_after
			FROM stock_item_balances b
			FULL JOIN last l ON l.stock_item_id = b.stock_item_id AND l.location_id = b.location_id)
		SELECT x.stock_item_id::text, si.sku, si.name, si.uom, x.location_id::text, x.has_balance, x.quantity::text,
			x.last_id::text, x.quantity_after::text, coalesce(c.movements, 0)
		FROM x JOIN stock_items si ON si.id = x.stock_item_id
		LEFT JOIN counted c ON c.stock_item_id = x.stock_item_id AND c.location_id = x.location_id
		WHERE `+scope+`
		ORDER BY si.sku, x.location_id`, args...)
	if err != nil {
		return fmt.Errorf("reconcile parts balances: %w", err)
	}
	var sku ReconcileSKU
	var loc string
	var hasBalance bool
	var qty, lastID, lastQty *string
	var movements int
	if _, err := pgx.ForEachRow(rows, []any{&sku.ID, &sku.SKU, &sku.Name, &sku.UOM, &loc, &hasBalance, &qty, &lastID, &lastQty,
		&movements}, func() error {
		d.PartBalancesChecked++
		d.MovementsChecked += movements
		item := sku
		base := RawDiscrepancy{ItemType: ItemStockItem, SKU: &item, LocationID: ptrTo(loc)}
		expected := "0"
		var ids []string
		if lastID != nil {
			expected, ids = trimQty(*lastQty), []string{*lastID}
		}
		if !hasBalance {
			r := base
			r.Type, r.ExpectedQuantity, r.MovementIDs = DiscPartsBalanceMissing, ptrTo(expected), ids
			d.Discrepancies = append(d.Discrepancies, r)
			return nil
		}
		actual := trimQty(*qty)
		if actual != expected {
			r := base
			r.Type, r.ExpectedQuantity, r.ActualQuantity, r.MovementIDs = DiscPartsBalanceMismatch, ptrTo(expected), ptrTo(actual), ids
			d.Discrepancies = append(d.Discrepancies, r)
		}
		if strings.HasPrefix(actual, "-") {
			r := base
			r.Type, r.ActualQuantity = DiscPartsNegativeBalance, ptrTo(actual)
			d.Discrepancies = append(d.Discrepancies, r)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("reconcile parts balances: %w", err)
	}

	// The history: each movement starts from what the previous one left at that SKU and location (the first from 0).
	args = nil
	scope = scopeCond(f, "m.to_location_id", &args, false)
	rows, err = tx.Query(ctx, `WITH m AS (
			SELECT id, seq, stock_item_id, to_location_id, quantity_before,
				lag(id) OVER w AS prev_id, lag(quantity_after) OVER w AS prev_quantity
			FROM stock_movements WHERE item_type = 'STOCK_ITEM'
			WINDOW w AS (PARTITION BY stock_item_id, to_location_id ORDER BY seq))
		SELECT m.stock_item_id::text, si.sku, si.name, si.uom, m.to_location_id::text, m.id::text, m.prev_id::text,
			m.quantity_before::text, coalesce(m.prev_quantity, 0)::text
		FROM m JOIN stock_items si ON si.id = m.stock_item_id
		WHERE `+scope+` AND m.quantity_before <> coalesce(m.prev_quantity, 0)
		ORDER BY m.seq`, args...)
	if err != nil {
		return fmt.Errorf("reconcile parts history: %w", err)
	}
	var mid string
	var prevID *string
	var before, prev string
	if _, err := pgx.ForEachRow(rows, []any{&sku.ID, &sku.SKU, &sku.Name, &sku.UOM, &loc, &mid, &prevID, &before, &prev}, func() error {
		item := sku
		ids := []string{mid}
		if prevID != nil {
			ids = []string{*prevID, mid}
		}
		d.Discrepancies = append(d.Discrepancies, RawDiscrepancy{Type: DiscPartsBrokenChain, ItemType: ItemStockItem, SKU: &item,
			LocationID: ptrTo(loc), ExpectedQuantity: ptrTo(trimQty(prev)), ActualQuantity: ptrTo(trimQty(before)), MovementIDs: ids})
		return nil
	}); err != nil {
		return fmt.Errorf("reconcile parts history: %w", err)
	}
	return nil
}

// trimQty prints a numeric(12,3) without trailing zeros ("10.500" -> "10.5", "7.000" -> "7").
func trimQty(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	if s == "" || s == "-" {
		return "0"
	}
	return s
}

func ptrTo(s string) *string { return &s }

func copyPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := *s
	return &v
}

func eqPtr(a, b *string) bool { return (a == nil && b == nil) || (a != nil && b != nil && *a == *b) }
