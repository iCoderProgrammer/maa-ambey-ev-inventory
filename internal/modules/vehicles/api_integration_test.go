package vehicles

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/audit"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/database"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/database/dbtest"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/auth"
	"github.com/jarvisconsulting/maa-ambey-enterprises/seeds"
)

// Vehicle unit tests on a throwaway migrated database. Names, identifiers and
// amounts are placeholders ("Brand A", "CH-0001", 95000): no real supplier,
// vehicle, serial or cost is invented. Approved purchase orders and goods
// receipts are inserted directly, and "posting" a receipt is simulated by
// setting it POSTED in the test transaction: receipt posting itself is D-07.

type fixture struct {
	pool    *pgxpool.Pool
	h       *harness
	viewer  session // vehicle_units.view
	coster  session // vehicle_units.view + vehicle_units.cost.view
	scoped  session // vehicle_units.view, location L1 only
	allLocs session // vehicle_units.view + locations.all
	nobody  session // no grant
	ids     map[string]string

	brandA, brandB, modelA, modelB, variantA, variantA2, variantB   string
	supplier, supplier2, po, poLineA, poLineB, otherPO, otherPOLine string
	grn, grnLineA, grnLineB                                         string
	unitA1, unitA2, unitA3, unitB1                                  string // A1, A2 accepted; A3 rejected; B1 accepted
	locL1, locL2                                                    string
}

func setup(t *testing.T) fixture {
	t.Helper()
	return setupOn(t, dbtest.MigratedPool(t))
}

// setupOn builds the fixture on an already migrated pool.
func setupOn(t *testing.T, pool *pgxpool.Pool) fixture {
	t.Helper()
	ctx := context.Background()
	if _, err := database.ApplySeeds(ctx, pool, seeds.FS); err != nil {
		t.Fatal(err)
	}
	store := auth.NewPGStore(pool)
	hash, err := auth.HashPassword(testPassword)
	if err != nil {
		t.Fatal(err)
	}
	f := fixture{pool: pool, ids: map[string]string{}}
	for _, u := range []string{"viewer", "coster", "scoped", "alllocs", "nobody", "buyer", "approver", "poster"} {
		created, err := store.CreateUser(ctx, audit.CLIActor("test"), u, u, nil, hash)
		if err != nil {
			t.Fatal(err)
		}
		f.ids[u] = created.ID
	}
	f.h = newHarness(t, store, NewPGStore(pool))
	f.viewer = f.h.login("viewer", f.ids["viewer"])
	f.coster = f.h.login("coster", f.ids["coster"])
	f.scoped = f.h.login("scoped", f.ids["scoped"])
	f.allLocs = f.h.login("alllocs", f.ids["alllocs"])
	f.nobody = f.h.login("nobody", f.ids["nobody"])
	f.h.grant(f.viewer, access.PermVehicleUnitView)
	f.h.grant(f.coster, access.PermVehicleUnitView, access.PermVehicleUnitCostView)
	f.h.grant(f.scoped, access.PermVehicleUnitView)
	f.h.grant(f.allLocs, access.PermVehicleUnitView, access.PermAllLocations)

	f.locL1 = f.scalar(t, `INSERT INTO locations (code, name) VALUES ('L1', 'Location One') RETURNING id::text`)
	f.locL2 = f.scalar(t, `INSERT INTO locations (code, name) VALUES ('L2', 'Location Two') RETURNING id::text`)
	f.h.scope(f.scoped, f.locL1)

	brand := func(name string) string {
		return f.scalar(t, `INSERT INTO brands (name, slug, lifecycle_status, authorisation_status, manufacturer_legal_name,
			authorisation_reference, verified_by, verified_at)
			VALUES ($1, lower(replace($1, ' ', '-')), 'ACTIVE', 'AUTHORISED', 'Example Manufacturer', 'TEST-AUTH', $2, now()) RETURNING id::text`,
			name, f.ids["approver"])
	}
	model := func(b, code string) string {
		return f.scalar(t, `INSERT INTO models (brand_id, name, internal_code, lifecycle_status, confidence, source_url, source_date, verified_by, verified_at)
			VALUES ($1, 'Model '||$2, $2, 'ACTIVE', 'SUPPLIER_CONFIRMED', 'https://example.com/m', current_date, $3, now()) RETURNING id::text`,
			b, code, f.ids["approver"])
	}
	variant := func(b, m, code string) string {
		return f.scalar(t, `INSERT INTO variants (brand_id, model_id, name, variant_code, lifecycle_status, confidence, source_url, source_date, verified_by, verified_at)
			VALUES ($1, $2, 'Variant '||$3, $3, 'ACTIVE', 'SUPPLIER_CONFIRMED', 'https://example.com/v', current_date, $4, now()) RETURNING id::text`,
			b, m, code, f.ids["approver"])
	}
	f.brandA, f.brandB = brand("Brand A"), brand("Brand B")
	f.modelA, f.modelB = model(f.brandA, "MA"), model(f.brandB, "MB")
	f.variantA, f.variantA2, f.variantB = variant(f.brandA, f.modelA, "STD"), variant(f.brandA, f.modelA, "PRO"), variant(f.brandB, f.modelB, "STD")
	f.supplier = f.scalar(t, `INSERT INTO suppliers (legal_name, created_by) VALUES ('Example Supplier One', $1) RETURNING id::text`, f.ids["buyer"])
	f.supplier2 = f.scalar(t, `INSERT INTO suppliers (legal_name, created_by) VALUES ('Example Supplier Two', $1) RETURNING id::text`, f.ids["buyer"])

	newPO := func(supplier string) string {
		po := f.scalar(t, `INSERT INTO purchase_orders (supplier_id, created_by, expected_date) VALUES ($1, $2, current_date + 7) RETURNING id::text`,
			supplier, f.ids["buyer"])
		return po
	}
	addLine := func(po string, no int, b, m, v string, qty int, cost string) string {
		return f.scalar(t, `INSERT INTO purchase_order_lines (purchase_order_id, line_no, brand_id, model_id, variant_id, quantity, unit_cost)
			VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id::text`, po, no, b, m, v, qty, cost)
	}
	approve := func(po string) {
		f.exec(t, `UPDATE purchase_orders SET status = 'APPROVED', submitted_by = $2, submitted_at = now(), approval_basis = 'APPROVER',
			approved_by = $3, approved_at = now() WHERE id = $1`, po, f.ids["buyer"], f.ids["approver"])
	}
	f.po = newPO(f.supplier)
	f.poLineA = addLine(f.po, 1, f.brandA, f.modelA, f.variantA, 3, "95000.00")
	f.poLineB = addLine(f.po, 2, f.brandB, f.modelB, f.variantB, 1, "88000.00")
	approve(f.po)
	f.otherPO = newPO(f.supplier)
	f.otherPOLine = addLine(f.otherPO, 1, f.brandA, f.modelA, f.variantA2, 1, "97000.00")
	approve(f.otherPO)

	// A draft receipt: line A 3 received (2 accepted, 1 rejected), line B 1 accepted.
	f.grn = f.newReceipt(t, f.po)
	f.grnLineA = f.receiptLine(t, f.grn, f.po, f.poLineA, 3, 2, 1)
	f.grnLineB = f.receiptLine(t, f.grn, f.po, f.poLineB, 1, 1, 0)
	f.unitA1 = f.receiptUnit(t, f.grn, f.grnLineA, 1, "ACCEPTED", "ch-0001", "mot-0001", "bat-0001", "chg-0001")
	f.unitA2 = f.receiptUnit(t, f.grn, f.grnLineA, 2, "ACCEPTED", "", "", "", "")
	f.unitA3 = f.receiptUnit(t, f.grn, f.grnLineA, 3, "REJECTED", "CH-0003", "", "", "")
	f.unitB1 = f.receiptUnit(t, f.grn, f.grnLineB, 1, "ACCEPTED", "CHB-0001", "", "", "")
	return f
}

func (f fixture) exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

func (f fixture) scalar(t *testing.T, sql string, args ...any) string {
	t.Helper()
	var s string
	if err := f.pool.QueryRow(context.Background(), sql, args...).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

func (f fixture) count(t *testing.T, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func (f fixture) newReceipt(t *testing.T, po string) string {
	t.Helper()
	return f.scalar(t, `INSERT INTO goods_receipts (purchase_order_id, received_date, supplier_reference, created_by)
		VALUES ($1, DATE '2026-01-15', 'DC-1', $2) RETURNING id::text`, po, f.ids["poster"])
}

func (f fixture) receiptLine(t *testing.T, grn, po, poLine string, received, accepted, rejected int) string {
	t.Helper()
	return f.scalar(t, `INSERT INTO goods_receipt_lines (goods_receipt_id, purchase_order_id, purchase_order_line_id,
		received_quantity, accepted_quantity, rejected_quantity) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id::text`,
		grn, po, poLine, received, accepted, rejected)
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (f fixture) receiptUnit(t *testing.T, grn, line string, no int, condition, chassis, motor, battery, charger string) string {
	t.Helper()
	return f.scalar(t, `INSERT INTO goods_receipt_units (goods_receipt_id, goods_receipt_line_id, unit_no, condition,
		chassis_number, chassis_normalised, motor_serial, battery_serial, charger_serial)
		VALUES ($1, $2, $3, $4, $5::text, upper(btrim($5::text)), $6, $7, $8) RETURNING id::text`,
		grn, line, no, condition, nullable(chassis), nullable(motor), nullable(battery), nullable(charger))
}

// post marks a receipt POSTED, standing in for D-07 inside a test transaction.
func (f fixture) post(ctx context.Context, tx pgx.Tx, grn string) error {
	_, err := tx.Exec(ctx, `UPDATE goods_receipts SET status = 'POSTED', posted_by = $2, posted_at = now() WHERE id = $1`,
		grn, f.ids["poster"])
	return err
}

func (f fixture) actor() audit.Actor { return audit.UserActor(f.ids["poster"], "poster") }

// registerAll registers units in one transaction; postFirst decides whether the
// receipt is marked POSTED before or after (the database checks at commit).
func (f fixture) registerAll(ctx context.Context, grn string, post bool, location *string, units ...string) ([]Registration, error) {
	var out []Registration
	err := pgx.BeginFunc(ctx, f.pool, func(tx pgx.Tx) error {
		for _, u := range units {
			r, err := RegisterFromReceiptUnit(ctx, tx, f.actor(), "req-test", u, location)
			if err != nil {
				return err
			}
			out = append(out, r)
		}
		if post {
			return f.post(ctx, tx, grn)
		}
		return nil
	})
	return out, CommitError(err)
}

func ptr(s string) *string { return &s }

func sqlState(err error) (code, constraint string) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code, pgErr.ConstraintName
	}
	return "", ""
}

// ── Tests ───────────────────────────────────────────────────────────────────

func TestRegisteredUnitIsRepresentedWithItsSources(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	// Registered before the receipt is marked POSTED in the same transaction: checked at commit.
	regs, err := f.registerAll(ctx, f.grn, true, ptr(f.locL1), f.unitA1)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^MAE-VEH-\d{6}$`).MatchString(regs[0].UnitNumber) {
		t.Errorf("unit number %q", regs[0].UnitNumber)
	}
	var d Detail
	// The unit is at L1: a reader needs L1 in their location scope.
	expectError(t, f.h.get(f.coster, "/vehicle-units/"+regs[0].ID), http.StatusNotFound, "NOT_FOUND")
	f.h.scope(f.coster, f.locL1)
	expect(t, f.h.get(f.coster, "/vehicle-units/"+regs[0].ID), http.StatusOK, &d)
	grnNumber := f.scalar(t, `SELECT grn_number FROM goods_receipts WHERE id = $1`, f.grn)
	poNumber := f.scalar(t, `SELECT po_number FROM purchase_orders WHERE id = $1`, f.po)
	checks := map[string]bool{
		"status":        d.Status == StatusReceivedPendingPDI,
		"brand":         d.BrandID == f.brandA && d.BrandName == "Brand A",
		"model/variant": d.ModelID == f.modelA && d.VariantID == f.variantA && d.VariantName == "Variant STD",
		"chassis":       d.ChassisNumber != nil && *d.ChassisNumber == "ch-0001" && *d.ChassisNormalised == "CH-0001",
		"serials":       *d.MotorSerial == "mot-0001" && *d.BatterySerial == "bat-0001" && *d.ChargerSerial == "chg-0001",
		"location":      d.LocationID != nil && *d.LocationID == f.locL1 && *d.LocationCode == "L1",
		"received date": d.ReceivedDate == "2026-01-15",
		"cost":          d.UnitCost != nil && *d.UnitCost == "95000.00" && *d.CostCurrency == "INR" && !d.CostHidden,
		"source": d.Source.Type == SourceGoodsReceipt && *d.Source.SupplierID == f.supplier &&
			*d.Source.SupplierName == "Example Supplier One" && *d.Source.PurchaseOrderID == f.po &&
			*d.Source.PurchaseOrderNumber == poNumber && *d.Source.PurchaseOrderLineID == f.poLineA &&
			*d.Source.GoodsReceiptID == f.grn && *d.Source.GoodsReceiptNumber == grnNumber &&
			*d.Source.GoodsReceiptLineID == f.grnLineA && *d.Source.GoodsReceiptUnitID == f.unitA1 &&
			*d.Source.SupplierReference == "DC-1" && d.Source.ReceivedDate == "2026-01-15",
		"history": len(d.Events) == 1 && d.Events[0].Action == ActionRegistered && d.Events[0].ToStatus == StatusReceivedPendingPDI &&
			d.Events[0].SourceType == SourceGoodsReceipt && d.Events[0].SourceID == f.grn &&
			d.Events[0].SourceReference == grnNumber && d.Events[0].ActorID == f.ids["poster"],
	}
	for name, ok := range checks {
		if !ok {
			t.Errorf("%s wrong: %+v", name, d)
		}
	}
	// Audited in the same transaction, without the cost.
	var summary string
	if err := f.pool.QueryRow(ctx, `SELECT summary::text FROM audit_events WHERE action = 'vehicle_unit.registered' AND entity_id = $1
		AND actor_user_id = $2 AND reference = $3`, regs[0].ID, f.ids["poster"], regs[0].UnitNumber).Scan(&summary); err != nil {
		t.Fatalf("audit event: %v", err)
	}
	if strings.Contains(summary, "95000") || !strings.Contains(summary, grnNumber) {
		t.Errorf("audit summary %s", summary)
	}
	// Brand, model and variant come from the PO line, not from any caller input.
	if n := f.count(t, `SELECT count(*) FROM vehicle_units v JOIN purchase_order_lines l ON l.id = v.purchase_order_line_id
		WHERE v.variant_id = l.variant_id AND v.brand_id = l.brand_id AND v.unit_cost = l.unit_cost`); n != 1 {
		t.Errorf("unit does not match its PO line (%d)", n)
	}
}

func TestDraftReceiptsNeverHaveVehicleUnits(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	// Saving a draft (lines and units are already stored) creates no unit and changes no stock or PO quantity.
	if n := f.count(t, `SELECT count(*) FROM vehicle_units`); n != 0 {
		t.Fatalf("units after a draft: %d", n)
	}
	if n := f.count(t, `SELECT coalesce(sum(received_quantity), 0)::int FROM purchase_order_lines WHERE purchase_order_id = $1`, f.po); n != 0 {
		t.Fatalf("PO received quantity changed: %d", n)
	}
	// Registering from a receipt that is still a draft is refused at commit, leaving nothing behind.
	if _, err := f.registerAll(ctx, f.grn, false, nil, f.unitA1); !errors.Is(err, ErrReceiptNotPosted) {
		t.Fatalf("draft receipt: %v", err)
	}
	// A cancelled receipt is not posted either.
	cancelled := f.newReceipt(t, f.otherPO)
	line := f.receiptLine(t, cancelled, f.otherPO, f.otherPOLine, 1, 1, 0)
	cu := f.receiptUnit(t, cancelled, line, 1, "ACCEPTED", "CHX-1", "", "", "")
	f.exec(t, `UPDATE goods_receipts SET status = 'CANCELLED', cancelled_by = $2, cancelled_at = now(), cancel_reason = 'test' WHERE id = $1`,
		cancelled, f.ids["poster"])
	if _, err := f.registerAll(ctx, cancelled, false, nil, cu); !errors.Is(err, ErrReceiptNotPosted) {
		t.Fatalf("cancelled receipt: %v", err)
	}
	if n := f.count(t, `SELECT count(*) FROM vehicle_units`) + f.count(t, `SELECT count(*) FROM vehicle_unit_events`) +
		f.count(t, `SELECT count(*) FROM audit_events WHERE action = 'vehicle_unit.registered'`); n != 0 {
		t.Fatalf("refused registrations left %d rows", n)
	}
	// Rejected and unknown captured units are refused.
	if _, err := f.registerAll(ctx, f.grn, true, nil, f.unitA3); !errors.Is(err, ErrReceiptUnitNotAccepted) {
		t.Fatalf("rejected unit: %v", err)
	}
	for _, id := range []string{"00000000-0000-0000-0000-000000000000", "not-a-uuid"} {
		if _, err := f.registerAll(ctx, f.grn, true, nil, id); !errors.Is(err, ErrReceiptUnitNotFound) {
			t.Fatalf("unknown unit %s: %v", id, err)
		}
	}
	if _, err := f.registerAll(ctx, f.grn, true, ptr("00000000-0000-0000-0000-000000000000"), f.unitA1); !errors.Is(err, ErrInvalidLocation) {
		t.Fatalf("unknown location: %v", err)
	}
	// Only a signed-in user registers units.
	err := pgx.BeginFunc(ctx, f.pool, func(tx pgx.Tx) error {
		_, err := RegisterFromReceiptUnit(ctx, tx, audit.CLIActor("script"), "", f.unitA1, nil)
		return err
	})
	if err == nil {
		t.Fatal("CLI actor registered a unit")
	}
	if n := f.count(t, `SELECT count(*) FROM vehicle_units`); n != 0 {
		t.Fatalf("units: %d", n)
	}
}

func TestACapturedUnitBecomesOneVehicleUnitEvenConcurrently(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if _, err := f.registerAll(ctx, f.grn, true, nil, f.unitA1); err != nil {
		t.Fatal(err)
	}
	if _, err := f.registerAll(ctx, f.grn, false, nil, f.unitA1); !errors.Is(err, ErrAlreadyRegistered) {
		t.Fatalf("second registration: %v", err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 6)
	for range 6 {
		wg.Go(func() {
			_, err := f.registerAll(ctx, f.grn, false, nil, f.unitA2)
			results <- err
		})
	}
	wg.Wait()
	close(results)
	ok, dup := 0, 0
	for err := range results {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, ErrAlreadyRegistered):
			dup++
		default:
			t.Errorf("unexpected: %v", err)
		}
	}
	if ok != 1 || dup != 5 {
		t.Fatalf("ok=%d duplicate=%d, want 1 and 5", ok, dup)
	}
	if n := f.count(t, `SELECT count(*) FROM vehicle_units WHERE goods_receipt_unit_id = $1`, f.unitA2); n != 1 {
		t.Fatalf("units for one captured unit: %d", n)
	}
	if n := f.count(t, `SELECT count(*) FROM vehicle_unit_events`); n != 2 {
		t.Fatalf("history rows: %d", n)
	}
}

// References are checked by the database too, so no caller can link a unit to
// records that do not belong together.
func TestReferencesCannotCrossRecordsOrBrands(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	base := map[string]any{
		"brand_id": f.brandA, "model_id": f.modelA, "variant_id": f.variantA, "source_type": "GOODS_RECEIPT",
		"supplier_id": f.supplier, "purchase_order_id": f.po, "purchase_order_line_id": f.poLineA,
		"goods_receipt_id": f.grn, "goods_receipt_line_id": f.grnLineA, "goods_receipt_unit_id": f.unitA1,
		"received_date": "2026-01-15", "unit_cost": "95000", "registered_by": f.ids["poster"],
	}
	otherGRN := f.newReceipt(t, f.otherPO)
	otherGRNLine := f.receiptLine(t, otherGRN, f.otherPO, f.otherPOLine, 1, 1, 0)
	otherUnit := f.receiptUnit(t, otherGRN, otherGRNLine, 1, "ACCEPTED", "", "", "", "")
	cases := []struct {
		name       string
		change     map[string]any
		constraint string
	}{
		{"variant of another brand", map[string]any{"variant_id": f.variantB}, "vehicle_units_variant_brand_fk"},
		{"model of another brand", map[string]any{"model_id": f.modelB}, "vehicle_units_model_fk"},
		{"variant other than the PO line's", map[string]any{"variant_id": f.variantA2}, "vehicle_units_po_line_variant_fk"},
		{"supplier other than the PO's", map[string]any{"supplier_id": f.supplier2}, "vehicle_units_po_supplier_fk"},
		{"PO line of another PO", map[string]any{"purchase_order_line_id": f.otherPOLine}, "vehicle_units_po_line_fk"},
		{"receipt of another PO", map[string]any{"goods_receipt_id": otherGRN}, "vehicle_units_receipt_fk"},
		{"receipt line of another receipt", map[string]any{"goods_receipt_line_id": f.grnLineB}, "vehicle_units_receipt_line_po_line_fk"},
		{"captured unit of another line", map[string]any{"goods_receipt_unit_id": f.unitB1}, "vehicle_units_receipt_unit_fk"},
		{"captured unit of another receipt", map[string]any{"goods_receipt_unit_id": otherUnit}, "vehicle_units_receipt_unit_fk"},
		{"no goods receipt reference", map[string]any{"goods_receipt_unit_id": nil}, "vehicle_units_goods_receipt_source_complete"},
		{"unknown source", map[string]any{"source_type": "MANUAL"}, "vehicle_units_source_valid"},
		{"no cost", map[string]any{"unit_cost": "0"}, "vehicle_units_cost_positive"},
		{"other currency", map[string]any{"cost_currency": "USD"}, "vehicle_units_cost_currency_valid"},
		{"invented status", map[string]any{"status": "PARKED", "chassis_number": "CH-0001"}, "vehicle_units_status_valid"},
		// INV-002: a unit cannot be past RECEIVED_PENDING_PDI without a chassis/frame number.
		{"available without chassis", map[string]any{"status": "AVAILABLE"}, "vehicle_units_chassis_after_receipt"},
	}
	for _, c := range cases {
		row := map[string]any{}
		for k, v := range base {
			row[k] = v
		}
		for k, v := range c.change {
			row[k] = v
		}
		var cols, ph []string
		var args []any
		for k, v := range row {
			args = append(args, v)
			cols = append(cols, k)
			ph = append(ph, "$"+strconv.Itoa(len(args)))
		}
		_, err := f.pool.Exec(ctx, `INSERT INTO vehicle_units (`+strings.Join(cols, ", ")+`) VALUES (`+strings.Join(ph, ", ")+`)`, args...)
		if code, constraint := sqlState(err); constraint != c.constraint {
			t.Errorf("%s: got %s %s (%v), want %s", c.name, code, constraint, err, c.constraint)
		}
	}
	if n := f.count(t, `SELECT count(*) FROM vehicle_units`); n != 0 {
		t.Fatalf("units: %d", n)
	}
}

func TestIdentifiersAreOptionalOnReceiptAndNormalised(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	// No identifier is made mandatory while a unit is pending PDI (OQ-4): A2 has none.
	regs, err := f.registerAll(ctx, f.grn, true, nil, f.unitA1, f.unitA2, f.unitB1)
	if err != nil {
		t.Fatal(err)
	}
	var d Detail
	expect(t, f.h.get(f.viewer, "/vehicle-units/"+regs[1].ID), http.StatusOK, &d)
	if d.ChassisNumber != nil || d.MotorSerial != nil || d.BatterySerial != nil || d.ChargerSerial != nil {
		t.Errorf("identifiers invented: %+v", d.Unit)
	}
	// The database's normalised forms equal NormaliseIdentifier of the stored values.
	rows, err := f.pool.Query(ctx, `SELECT chassis_number, chassis_normalised, motor_serial, motor_normalised, battery_serial,
		battery_normalised, charger_serial, charger_normalised FROM vehicle_units`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var v [8]*string
		if err := rows.Scan(&v[0], &v[1], &v[2], &v[3], &v[4], &v[5], &v[6], &v[7]); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 8; i += 2 {
			if (v[i] == nil) != (v[i+1] == nil) || (v[i] != nil && NormaliseIdentifier(*v[i]) != *v[i+1]) {
				t.Errorf("normalised %v / %v", v[i], v[i+1])
			}
		}
	}
	// Search uses the normalised form: case and surrounding spaces do not matter, part of a value is enough.
	for q, want := range map[string]int{"ch-0001": 1, " CH-00 ": 1, "chb": 1, "MOT-0001": 1, "bat-": 1, "chg-0001": 1,
		"MAE-VEH": 3, "mae-veh": 3, "zzz": 0, "%": 0, "_": 0} {
		var p UnitPage
		expect(t, f.h.get(f.viewer, "/vehicle-units?q="+urlQuery(q)), http.StatusOK, &p)
		if p.Total != want {
			t.Errorf("q=%q: total %d, want %d", q, p.Total, want)
		}
	}
}

func urlQuery(s string) string {
	r := strings.NewReplacer(" ", "%20", "%", "%25", "&", "%26", "+", "%2B")
	return r.Replace(s)
}

func TestCostIsServerSidePermissioned(t *testing.T) {
	f := setup(t)
	regs, err := f.registerAll(context.Background(), f.grn, true, nil, f.unitA1, f.unitB1)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/vehicle-units", "/vehicle-units/" + regs[0].ID, "/vehicle-units/" + regs[1].ID} {
		w := f.h.get(f.viewer, path)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: %d", path, w.Code)
		}
		body := w.Body.String()
		if strings.Contains(body, "95000") || strings.Contains(body, "88000") || !strings.Contains(body, `"costHidden":true`) ||
			!strings.Contains(body, `"unitCost":null`) {
			t.Errorf("%s leaked or did not hide the cost: %s", path, body)
		}
		w = f.h.get(f.coster, path)
		if !strings.Contains(w.Body.String(), `"unitCost":"`) || strings.Contains(w.Body.String(), `"costHidden":true`) {
			t.Errorf("%s with cost permission: %s", path, w.Body)
		}
	}
	// Without vehicle_units.view, nothing at all.
	expectError(t, f.h.get(f.nobody, "/vehicle-units"), http.StatusForbidden, "FORBIDDEN")
	expectError(t, f.h.get(f.nobody, "/vehicle-units/"+regs[0].ID), http.StatusForbidden, "FORBIDDEN")
	f.h.grant(f.nobody, access.PermVehicleUnitCostView)
	expectError(t, f.h.get(f.nobody, "/vehicle-units/"+regs[0].ID), http.StatusForbidden, "FORBIDDEN")
}

func TestLocationScopeAppliesToUnits(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	l1, err := f.registerAll(ctx, f.grn, true, ptr(f.locL1), f.unitA1)
	if err != nil {
		t.Fatal(err)
	}
	l2, err := f.registerAll(ctx, f.grn, false, ptr(f.locL2), f.unitB1)
	if err != nil {
		t.Fatal(err)
	}
	none, err := f.registerAll(ctx, f.grn, false, nil, f.unitA2)
	if err != nil {
		t.Fatal(err)
	}
	ids := func(s session, query string) map[string]bool {
		var p UnitPage
		expect(t, f.h.get(s, "/vehicle-units"+query), http.StatusOK, &p)
		out := map[string]bool{}
		for _, u := range p.Items {
			out[u.ID] = true
		}
		if p.Total != len(out) {
			t.Errorf("total %d for %d items", p.Total, len(out))
		}
		return out
	}
	got := ids(f.scoped, "")
	if len(got) != 2 || !got[l1[0].ID] || !got[none[0].ID] {
		t.Errorf("scoped user sees %v", got)
	}
	expect(t, f.h.get(f.scoped, "/vehicle-units/"+l1[0].ID), http.StatusOK, nil)
	expectError(t, f.h.get(f.scoped, "/vehicle-units/"+l2[0].ID), http.StatusNotFound, "NOT_FOUND")
	expectError(t, f.h.get(f.scoped, "/vehicle-units?locationId="+f.locL2), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if got := ids(f.scoped, "?locationId="+f.locL1); len(got) != 1 || !got[l1[0].ID] {
		t.Errorf("scoped user, L1 filter: %v", got)
	}
	if got := ids(f.allLocs, ""); len(got) != 3 {
		t.Errorf("all-locations user sees %v", got)
	}
	if got := ids(f.allLocs, "?locationId="+f.locL2); len(got) != 1 || !got[l2[0].ID] {
		t.Errorf("all-locations user, L2 filter: %v", got)
	}
	// A user with no location at all still sees units without a location only.
	if got := ids(f.viewer, ""); len(got) != 1 || !got[none[0].ID] {
		t.Errorf("viewer without locations sees %v", got)
	}
}

func TestListFiltersCombine(t *testing.T) {
	f := setup(t)
	if _, err := f.registerAll(context.Background(), f.grn, true, nil, f.unitA1, f.unitA2, f.unitB1); err != nil {
		t.Fatal(err)
	}
	for query, want := range map[string]int{
		"":                                 3,
		"?brandId=" + f.brandA:             2,
		"?brandId=" + f.brandB:             1,
		"?variantId=" + f.variantA:         2,
		"?variantId=" + f.variantA2:        0,
		"?supplierId=" + f.supplier:        3,
		"?supplierId=" + f.supplier2:       0,
		"?purchaseOrderId=" + f.po:         3,
		"?purchaseOrderId=" + f.otherPO:    0,
		"?goodsReceiptId=" + f.grn:         3,
		"?status=RECEIVED_PENDING_PDI":     3,
		"?status=AVAILABLE":                0,
		"?brandId=" + f.brandA + "&q=ch-0": 1,
		"?brandId=" + f.brandB + "&q=ch-0": 0,
		"?limit=2":                         3,
		"?limit=2&page=2":                  3,
	} {
		var p UnitPage
		expect(t, f.h.get(f.viewer, "/vehicle-units"+query), http.StatusOK, &p)
		if p.Total != want {
			t.Errorf("%q: total %d, want %d", query, p.Total, want)
		}
	}
	var p UnitPage
	expect(t, f.h.get(f.viewer, "/vehicle-units?limit=2&page=2"), http.StatusOK, &p)
	if len(p.Items) != 1 || p.Page != 2 || p.Limit != 2 {
		t.Errorf("page 2: %+v", p)
	}
	expectError(t, f.h.get(f.viewer, "/vehicle-units/00000000-0000-0000-0000-000000000000"), http.StatusNotFound, "NOT_FOUND")
}

// Status, location, identifiers and history cannot be changed or deleted
// directly: lifecycle changes belong to the stock movement service (E-01).
func TestUnitsAndHistoryCannotChangeOrBeDeleted(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	regs, err := f.registerAll(ctx, f.grn, true, nil, f.unitA1)
	if err != nil {
		t.Fatal(err)
	}
	id := regs[0].ID
	for _, sql := range []string{
		`UPDATE vehicle_units SET status = 'AVAILABLE' WHERE id = $1`,
		`UPDATE vehicle_units SET status = 'SOLD' WHERE id = $1`,
		`UPDATE vehicle_units SET location_id = (SELECT id FROM locations LIMIT 1) WHERE id = $1`,
		`UPDATE vehicle_units SET chassis_number = 'CH-9999' WHERE id = $1`,
		`UPDATE vehicle_units SET unit_cost = 1 WHERE id = $1`,
		`DELETE FROM vehicle_units WHERE id = $1`,
		`UPDATE vehicle_unit_events SET note = 'x' WHERE vehicle_unit_id = $1`,
		`DELETE FROM vehicle_unit_events WHERE vehicle_unit_id = $1`,
	} {
		_, err := f.pool.Exec(ctx, sql, id)
		if code, _ := sqlState(err); code != "42501" {
			t.Errorf("%s: %v", sql, err)
		}
	}
	if s := f.scalar(t, `SELECT status FROM vehicle_units WHERE id = $1`, id); s != StatusReceivedPendingPDI {
		t.Errorf("status %s", s)
	}
}
