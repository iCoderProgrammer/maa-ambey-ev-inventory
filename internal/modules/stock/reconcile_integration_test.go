package stock_test

import (
	"context"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/stock"
)

// Ledger reconciliation (E-06) on a throwaway migrated database (one scratch
// database per test). The ledger is built through the existing write paths;
// discrepancies are injected only by switching the user triggers of one table
// off inside a test transaction (as a database owner could), never through the
// application. Grants are test-only (in-memory harness).

type reconcileFixture struct {
	*countsFixture
	auditor  session // reconciliation + vehicle + parts view, every location
	l2Only   session // reconciliation + vehicle + parts view, location L2 only
	l1Avail  []string
	l2Unit   string
	pending  string // registered at L1, RECEIPT only
	noLoc    string
	l2Item   string // a second SKU with stock at L2
	itemPrev string // the PARTS_ISSUE movement at L1
}

func setupReconcile(t *testing.T) *reconcileFixture {
	t.Helper()
	rf := &reconcileFixture{countsFixture: setupCounts(t)} // PRT-1: 10 received at L1
	f, p := rf.fixture, rf.partsFixture
	rf.l1Avail = f.available(t, 2)
	rf.pending = f.register(t, &f.locL1, "CH")[0]
	rf.l2Unit = f.register(t, &f.locL2, "CH")[0]
	rf.noLoc = f.register(t, nil, "CH")[0]
	_, version := f.unit(t, rf.l1Avail[0])
	f.h.grant(f.mover, append(append([]string{access.PermVehicleUnitDisplayManage}, partsAll...), countCreate...)...)
	expect(t, f.h.display(f.mover, rf.l1Avail[0], true, `{"version":`+strconv.Itoa(version)+`}`, "display-0001"), http.StatusOK, nil)
	expect(t, p.call(p.mover, p.item, "issues", p.issue("2"), "issue-0001"), http.StatusOK, nil)
	rf.itemPrev = f.scalar(t, `SELECT id::text FROM stock_movements WHERE movement_type = 'PARTS_ISSUE'`)
	rf.l2Item = f.scalar(t, `INSERT INTO stock_items (sku, name, category_id, uom, tracking_mode, created_by)
		VALUES ('PRT-2', 'Item PRT-2', $1, 'PCS', 'QUANTITY', $2) RETURNING id::text`, p.category, f.ids["buyer"])
	expect(t, p.call(p.mover, rf.l2Item, "receipts", p.receipt("4", "locationId", f.locL2, "sourceReference", "DC-L2"), "rcpt-0002"),
		http.StatusOK, nil)

	rf.auditor = f.h.login("poster", f.ids["poster"])
	f.h.grant(rf.auditor, access.PermReconciliationView, access.PermVehicleUnitView, access.PermPartsView, access.PermAllLocations)
	rf.l2Only = f.scoped
	f.h.grant(rf.l2Only, access.PermReconciliationView, access.PermVehicleUnitView, access.PermPartsView)
	return rf
}

func (rf *reconcileFixture) run(t *testing.T, sess session, query string) stock.ReconcileReport {
	t.Helper()
	var r stock.ReconcileReport
	expect(t, rf.h.raw(http.MethodGet, "/api/v1/stock-reconciliation?limit=100&"+query, "", &sess, nil), http.StatusOK, &r)
	if r.Total < len(r.Items) || r.Status != stock.ReconcileStatusCompleted {
		t.Fatalf("report %+v", r)
	}
	return r
}

// tamper runs sql with the user triggers of table switched off, in one
// transaction — the only way to make records disagree with the ledger.
func (rf *reconcileFixture) tamper(t *testing.T, table, sql string, args ...any) {
	t.Helper()
	ctx := context.Background()
	err := pgx.BeginFunc(ctx, rf.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `ALTER TABLE `+table+` DISABLE TRIGGER USER`); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `ALTER TABLE `+table+` ENABLE TRIGGER USER`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// fingerprint is a digest of every business table reconciliation reads (and of the audit log).
func (rf *reconcileFixture) fingerprint(t *testing.T) string {
	t.Helper()
	var parts []string
	for _, table := range []string{"vehicle_units", "stock_movements", "stock_item_balances", "stock_items", "locations",
		"vehicle_unit_events", "audit_events"} {
		parts = append(parts, table+"="+rf.scalar(t, `SELECT count(*)::text || ':' || coalesce(md5(string_agg(t::text, '|' ORDER BY t::text)), '')
			FROM `+table+` t`))
	}
	return strings.Join(parts, " ")
}

func findDisc(r stock.ReconcileReport, typ, itemID string) *stock.ReconcileDiscrepancy {
	for i, d := range r.Items {
		if d.Type != typ {
			continue
		}
		if (d.VehicleUnit != nil && d.VehicleUnit.ID == itemID) || (d.StockItem != nil && d.StockItem.ID == itemID) {
			return &r.Items[i]
		}
	}
	return nil
}

func types(r stock.ReconcileReport) []string {
	out := []string{}
	for _, d := range r.Items {
		out = append(out, d.Type)
	}
	slices.Sort(out)
	return out
}

// Scenarios 1, 4, 8: a ledger built through the application reconciles; the
// counts are the database's; nothing changes.
func TestReconcileCleanLedger(t *testing.T) {
	rf := setupReconcile(t)
	before := rf.fingerprint(t)
	r := rf.run(t, rf.auditor, "")
	if r.Summary.Discrepancies != 0 || len(r.Items) != 0 || r.Total != 0 {
		t.Fatalf("clean ledger reported %v", types(r))
	}
	units := rf.count(t, `SELECT count(*) FROM vehicle_units`)
	balances := rf.count(t, `SELECT count(*) FROM stock_item_balances`)
	movements := rf.count(t, `SELECT count(*) FROM stock_movements`)
	if r.Summary.VehicleUnitsChecked != units || units != 5 || r.Summary.PartBalancesChecked != balances || balances != 2 ||
		r.Summary.RecordsChecked != units+balances || r.Summary.MovementsChecked != movements {
		t.Fatalf("summary %+v; database: %d units, %d balances, %d movements", r.Summary, units, balances, movements)
	}
	if !r.Coverage.VehicleUnits || !r.Coverage.Parts || !r.Coverage.AllLocations {
		t.Fatalf("coverage %+v", r.Coverage)
	}
	if after := rf.fingerprint(t); after != before {
		t.Fatalf("reconciliation changed data:\n%s\n%s", before, after)
	}
}

// Scenarios 2, 3: a unit's status or location changed without a movement is
// reported with the ledger's value as expected and the record's as actual.
func TestReconcileDetectsVehicleStateAndLocationDrift(t *testing.T) {
	rf := setupReconcile(t)
	f := rf.fixture
	statusUnit, locUnit := rf.l1Avail[1], rf.pending
	lastStatusMove := f.scalar(t, `SELECT id::text FROM stock_movements WHERE vehicle_unit_id = $1 ORDER BY seq DESC LIMIT 1`, statusUnit)
	rf.tamper(t, "vehicle_units", `UPDATE vehicle_units SET status = 'DISPLAY' WHERE id = $1`, statusUnit)
	rf.tamper(t, "vehicle_units", `UPDATE vehicle_units SET location_id = $2 WHERE id = $1`, locUnit, f.locL2)

	r := rf.run(t, rf.auditor, "")
	if got := types(r); !slices.Equal(got, []string{stock.DiscVehicleLocationMismatch, stock.DiscVehicleStatusMismatch}) {
		t.Fatalf("types %v", got)
	}
	st := findDisc(r, stock.DiscVehicleStatusMismatch, statusUnit)
	if st == nil || *st.Expected != "AVAILABLE" || *st.Actual != "DISPLAY" || st.Severity != "HIGH" ||
		!slices.Equal(st.MovementIDs, []string{lastStatusMove}) || *st.LocationCode != "L1" || *st.BrandName != "Brand A" ||
		st.Reference != stock.DiscVehicleStatusMismatch+":"+statusUnit || st.VehicleUnit.UnitNumber == "" {
		t.Fatalf("status discrepancy %+v", st)
	}
	lc := findDisc(r, stock.DiscVehicleLocationMismatch, locUnit)
	if lc == nil || *lc.Expected != "L1" || *lc.Actual != "L2" || *lc.LocationID != f.locL2 || lc.Severity != "HIGH" {
		t.Fatalf("location discrepancy %+v", lc)
	}
	if r.Summary.BySeverity["HIGH"] != 2 || r.Summary.ByType[stock.DiscVehicleStatusMismatch] != 1 {
		t.Fatalf("summary %+v", r.Summary)
	}
}

// Scenario 5 (+ LED-004): a balance that is not its ledger quantity, a ledger
// location without a balance, and a negative balance.
func TestReconcileDetectsPartsBalanceDrift(t *testing.T) {
	rf := setupReconcile(t)
	f, p := rf.fixture, rf.partsFixture
	rf.tamper(t, "stock_item_balances", `UPDATE stock_item_balances SET quantity = 12 WHERE stock_item_id = $1 AND location_id = $2`,
		p.item, f.locL1)
	rf.tamper(t, "stock_item_balances", `DELETE FROM stock_item_balances WHERE stock_item_id = $1 AND location_id = $2`, rf.l2Item, f.locL2)

	r := rf.run(t, rf.auditor, "itemType=STOCK_ITEM")
	if got := types(r); !slices.Equal(got, []string{stock.DiscPartsBalanceMismatch, stock.DiscPartsBalanceMissing}) {
		t.Fatalf("types %v", got)
	}
	mm := findDisc(r, stock.DiscPartsBalanceMismatch, p.item)
	if mm == nil || *mm.Expected != "8 PCS" || *mm.Actual != "12 PCS" || *mm.LocationCode != "L1" ||
		!slices.Equal(mm.MovementIDs, []string{rf.itemPrev}) || mm.StockItem.SKU != "PRT-1" || mm.BrandID != nil {
		t.Fatalf("mismatch %+v", mm)
	}
	ms := findDisc(r, stock.DiscPartsBalanceMissing, rf.l2Item)
	if ms == nil || *ms.Expected != "4 PCS" || *ms.Actual != "no balance record" || *ms.LocationID != f.locL2 {
		t.Fatalf("missing %+v", ms)
	}
	// The ledger location without a balance is still counted as checked.
	if r.Summary.PartBalancesChecked != 2 {
		t.Fatalf("checked %d", r.Summary.PartBalancesChecked)
	}

	// Negative: possible only once the CHECK is gone (scratch database).
	f.exec(t, `ALTER TABLE stock_item_balances DROP CONSTRAINT stock_item_balances_not_negative`)
	rf.tamper(t, "stock_item_balances", `UPDATE stock_item_balances SET quantity = -1 WHERE stock_item_id = $1 AND location_id = $2`,
		p.item, f.locL1)
	r = rf.run(t, rf.auditor, "type=PARTS_NEGATIVE_BALANCE")
	if r.Total != 1 || *r.Items[0].Actual != "-1 PCS" || r.Items[0].Severity != "HIGH" {
		t.Fatalf("negative %+v", r.Items)
	}
}

// Scenario 6: missing or inconsistent history.
func TestReconcileDetectsMissingAndInconsistentHistory(t *testing.T) {
	rf := setupReconcile(t)
	f, p := rf.fixture, rf.partsFixture
	// A unit whose only movement (its RECEIPT) is gone.
	rf.tamper(t, "stock_movements", `DELETE FROM stock_movements WHERE vehicle_unit_id = $1`, rf.pending)
	// A unit whose RECEIPT is gone: its history starts with PDI_PASS.
	avail := rf.l1Avail[1]
	passID := f.scalar(t, `SELECT id::text FROM stock_movements WHERE vehicle_unit_id = $1 AND movement_type = 'PDI_PASS'`, avail)
	rf.tamper(t, "stock_movements", `DELETE FROM stock_movements WHERE vehicle_unit_id = $1 AND movement_type = 'RECEIPT'`, avail)
	// A displayed unit whose DISPLAY_ON claims to start at L2 while PDI_PASS left it at L1 (and the unit is at L1).
	disp := rf.l1Avail[0]
	onID := f.scalar(t, `SELECT id::text FROM stock_movements WHERE vehicle_unit_id = $1 AND movement_type = 'DISPLAY_ON'`, disp)
	pdiID := f.scalar(t, `SELECT id::text FROM stock_movements WHERE vehicle_unit_id = $1 AND movement_type = 'PDI_PASS'`, disp)
	rf.tamper(t, "stock_movements", `UPDATE stock_movements SET from_location_id = $2, to_location_id = $2 WHERE id = $1`, onID, f.locL2)
	// A parts issue that starts from 9 although the receipt left 10 (the balance still matches the issue's 8).
	rf.tamper(t, "stock_movements", `UPDATE stock_movements SET quantity_before = 9 WHERE id = $1`, rf.itemPrev)
	receiptID := f.scalar(t, `SELECT id::text FROM stock_movements WHERE stock_item_id = $1 AND movement_type = 'PARTS_RECEIPT'`, p.item)

	r := rf.run(t, rf.auditor, "")
	want := []string{stock.DiscPartsBrokenChain, stock.DiscVehicleBrokenChain, stock.DiscVehicleLocationMismatch,
		stock.DiscVehicleMissingReceipt, stock.DiscVehicleNoMovements}
	if got := types(r); !slices.Equal(got, want) {
		t.Fatalf("types %v, want %v", got, want)
	}
	if d := findDisc(r, stock.DiscVehicleNoMovements, rf.pending); d == nil || d.Severity != "HIGH" || len(d.MovementIDs) != 0 {
		t.Fatalf("no movements %+v", d)
	}
	if d := findDisc(r, stock.DiscVehicleMissingReceipt, avail); d == nil || *d.Actual != "PDI_PASS" ||
		!slices.Equal(d.MovementIDs, []string{passID}) {
		t.Fatalf("missing receipt %+v", d)
	}
	bc := findDisc(r, stock.DiscVehicleBrokenChain, disp)
	if bc == nil || bc.Severity != "MEDIUM" || !slices.Equal(bc.MovementIDs, []string{pdiID, onID}) ||
		*bc.Expected != "AVAILABLE at L1, quantity 1" || *bc.Actual != "AVAILABLE at L2, quantity 1" {
		t.Fatalf("vehicle chain %+v", bc)
	}
	if d := findDisc(r, stock.DiscVehicleLocationMismatch, disp); d == nil || *d.Expected != "L2" || *d.Actual != "L1" {
		t.Fatalf("location after the broken chain %+v", d)
	}
	pc := findDisc(r, stock.DiscPartsBrokenChain, p.item)
	if pc == nil || *pc.Expected != "10 PCS" || *pc.Actual != "9 PCS" || !slices.Equal(pc.MovementIDs, []string{receiptID, rf.itemPrev}) ||
		pc.Reference != stock.DiscPartsBrokenChain+":"+p.item+":"+f.locL1+":"+rf.itemPrev {
		t.Fatalf("parts chain %+v", pc)
	}
}

// Scenarios 7, 8: reruns over unchanged data give the same findings and change nothing.
func TestReconcileIsRepeatableAndReadOnly(t *testing.T) {
	rf := setupReconcile(t)
	rf.tamper(t, "vehicle_units", `UPDATE vehicle_units SET status = 'DISPLAY' WHERE id = $1`, rf.l1Avail[1])
	rf.tamper(t, "stock_item_balances", `UPDATE stock_item_balances SET quantity = 3 WHERE stock_item_id = $1`, rf.l2Item)
	before := rf.fingerprint(t)
	first := rf.run(t, rf.auditor, "")
	strip := func(r stock.ReconcileReport) stock.ReconcileReport {
		r.RunID, r.RunAt, r.CompletedAt, r.DurationMs = "", first.RunAt, first.RunAt, 0
		items := slices.Clone(r.Items)
		for i := range items {
			items[i].DetectedAt = first.RunAt
		}
		r.Items = items
		return r
	}
	for i := range 3 {
		again := rf.run(t, rf.auditor, "")
		if again.RunID == first.RunID {
			t.Fatal("run IDs repeat")
		}
		if !reflect.DeepEqual(strip(again), strip(first)) {
			t.Fatalf("run %d differs:\n%+v\n%+v", i, strip(again), strip(first))
		}
	}
	if first.Total != 2 {
		t.Fatalf("total %d", first.Total)
	}
	if after := rf.fingerprint(t); after != before {
		t.Fatalf("reconciliation changed data:\n%s\n%s", before, after)
	}
}

// Scenarios 9, 10: permissions, location scope and item permissions decide the rows.
func TestReconcilePermissionsAndScope(t *testing.T) {
	rf := setupReconcile(t)
	f, p := rf.fixture, rf.partsFixture
	// Drift at L1 (unit and balance) and at L2 (unit whose ledger says L1 — outside L2Only's scope).
	rf.tamper(t, "vehicle_units", `UPDATE vehicle_units SET status = 'DISPLAY' WHERE id = $1`, rf.l1Avail[1])
	rf.tamper(t, "stock_item_balances", `UPDATE stock_item_balances SET quantity = 1 WHERE stock_item_id = $1 AND location_id = $2`,
		p.item, f.locL1)
	rf.tamper(t, "vehicle_units", `UPDATE vehicle_units SET location_id = $2 WHERE id = $1`, rf.pending, f.locL2)

	// Not allowed: no permission, the item view alone, the report permission alone.
	expectError(t, rf.h.raw(http.MethodGet, "/api/v1/stock-reconciliation", "", &f.viewer, nil), http.StatusForbidden, "FORBIDDEN")
	buyer := f.h.login("buyer", f.ids["buyer"])
	f.h.grant(buyer, access.PermReconciliationView, access.PermAllLocations)
	expectError(t, rf.h.raw(http.MethodGet, "/api/v1/stock-reconciliation", "", &buyer, nil), http.StatusForbidden, "FORBIDDEN")

	all := rf.run(t, rf.auditor, "")
	if all.Total != 3 {
		t.Fatalf("auditor sees %v", types(all))
	}
	// L2 only: the unit now at L2 (its ledger location L1 named by ID only) and L2's records; nothing at L1.
	scoped := rf.run(t, rf.l2Only, "")
	if scoped.Total != 1 || scoped.Items[0].Type != stock.DiscVehicleLocationMismatch || scoped.Items[0].VehicleUnit.ID != rf.pending {
		t.Fatalf("scoped sees %+v", scoped.Items)
	}
	exp := *scoped.Items[0].Expected
	if !strings.Contains(exp, f.locL1) || strings.Contains(exp, "L1 ") || exp == "L1" || *scoped.Items[0].Actual != "L2" {
		t.Fatalf("out-of-scope location shown as %q", exp)
	}
	// Counts cover the scope only: L2's unit, the unit moved to L2, the unit without a location; L2's balance.
	if scoped.Summary.VehicleUnitsChecked != 3 || scoped.Summary.PartBalancesChecked != 1 || scoped.Coverage.AllLocations ||
		!slices.Equal(scoped.Coverage.LocationIDs, []string{f.locL2}) {
		t.Fatalf("scoped summary %+v coverage %+v", scoped.Summary, scoped.Coverage)
	}
	expectError(t, rf.h.raw(http.MethodGet, "/api/v1/stock-reconciliation?locationId="+f.locL1, "", &rf.l2Only, nil),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")

	// Vehicles only: no parts rows or counts; parts only: no vehicle rows or counts.
	f.h.grant(buyer, access.PermReconciliationView, access.PermVehicleUnitView, access.PermAllLocations)
	if r := rf.run(t, buyer, ""); r.Summary.PartBalancesChecked != 0 || r.Total != 2 || r.Coverage.Parts {
		t.Fatalf("vehicles only: %+v %v", r.Summary, types(r))
	}
	f.h.grant(buyer, access.PermReconciliationView, access.PermPartsView, access.PermAllLocations)
	if r := rf.run(t, buyer, ""); r.Summary.VehicleUnitsChecked != 0 || r.Total != 1 || r.Items[0].Type != stock.DiscPartsBalanceMismatch {
		t.Fatalf("parts only: %+v %v", r.Summary, types(r))
	}
}

// Scenario 11: one snapshot. Stock that moves while a run is in progress is
// wholly outside that run, so it can never produce a false discrepancy; the
// next run sees it, consistent.
func TestReconcileUsesOneSnapshot(t *testing.T) {
	rf := setupReconcile(t)
	f, p := rf.fixture, rf.partsFixture
	store := stock.NewPGStore(rf.pool)
	movementsBefore := rf.count(t, `SELECT count(*) FROM stock_movements`)
	var hookErr error
	stock.SetReconcileHook(store, func(ctx context.Context, _ pgx.Tx) error {
		// Commits between the vehicle and the parts queries, on another connection:
		// a receipt of 5 more PRT-1 at L1 and a display move of the second unit.
		w := p.call(p.mover, p.item, "receipts", p.receipt("5"), "rcpt-during-run")
		if w.Code != http.StatusOK {
			hookErr = &httpErr{w.Code, w.Body.String()}
		}
		_, version := f.unit(t, rf.l1Avail[1])
		if w := f.h.display(f.mover, rf.l1Avail[1], true, `{"version":`+strconv.Itoa(version)+`}`, "display-during-run"); w.Code != http.StatusOK {
			hookErr = &httpErr{w.Code, w.Body.String()}
		}
		return nil
	})
	data, err := store.Reconcile(context.Background(), stock.ReconcileFilter{Vehicles: true, Parts: true, AllLocations: true})
	if err != nil || hookErr != nil {
		t.Fatal(err, hookErr)
	}
	if len(data.Discrepancies) != 0 || data.MovementsChecked != movementsBefore {
		t.Fatalf("run saw %d movements (want %d, the snapshot) and %+v", data.MovementsChecked, movementsBefore, data.Discrepancies)
	}
	if n := rf.count(t, `SELECT count(*) FROM stock_movements`); n != movementsBefore+2 {
		t.Fatalf("movements after %d", n)
	}
	r := rf.run(t, rf.auditor, "")
	if r.Total != 0 || r.Summary.MovementsChecked != movementsBefore+2 {
		t.Fatalf("next run %+v", r.Summary)
	}

	// Under load: receipts and issues commit continuously while runs repeat; no run reports a discrepancy.
	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Go(func() {
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			op, body := "receipts", p.receipt("1")
			if i%2 == 1 {
				op, body = "issues", p.issue("1")
			}
			p.call(p.mover, p.item, op, body, "load-"+strconv.Itoa(i))
		}
	})
	for range 15 {
		if r := rf.run(t, rf.auditor, ""); r.Summary.Discrepancies != 0 {
			close(stop)
			wg.Wait()
			t.Fatalf("run during load reported %v", types(r))
		}
	}
	close(stop)
	wg.Wait()
}

// Scenario 8 at the database level: the reconciliation transaction is read-only,
// so even a write attempted inside it is refused and nothing is changed.
func TestReconcileTransactionRefusesWrites(t *testing.T) {
	rf := setupReconcile(t)
	store := stock.NewPGStore(rf.pool)
	before := rf.fingerprint(t)
	stock.SetReconcileHook(store, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE stock_item_balances SET quantity = quantity WHERE false`)
		return err
	})
	_, err := store.Reconcile(context.Background(), stock.ReconcileFilter{Vehicles: true, Parts: true, AllLocations: true})
	if code, _ := sqlState(err); code != "25006" { // read_only_sql_transaction
		t.Fatalf("write inside the reconciliation: %v (SQLSTATE %s), want 25006", err, code)
	}
	if after := rf.fingerprint(t); after != before {
		t.Fatal("data changed")
	}
}

type httpErr struct {
	code int
	body string
}

func (e *httpErr) Error() string { return strconv.Itoa(e.code) + " " + e.body }

// Scenario 12: a database failure is a safe 500 with nothing partial.
func TestReconcileDatabaseFailure(t *testing.T) {
	rf := setupReconcile(t)
	f := rf.fixture
	// A broken query: a column the reconciliation reads is renamed (scratch database).
	f.exec(t, `ALTER TABLE stock_item_balances RENAME COLUMN quantity TO quantity_renamed`)
	w := rf.h.raw(http.MethodGet, "/api/v1/stock-reconciliation", "", &rf.auditor, nil)
	expectError(t, w, http.StatusInternalServerError, "INTERNAL_ERROR")
	if strings.Contains(w.Body.String(), "quantity") || strings.Contains(w.Body.String(), "summary") {
		t.Fatalf("failure leaked details: %s", w.Body)
	}
}
