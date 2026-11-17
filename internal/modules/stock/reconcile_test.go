package stock_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth/authtest"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/stock"
)

// Reconciliation (E-06) without a database: a fake repository returns raw
// findings, so permission checks, validation, shaping, ordering, filtering,
// paging and failure handling are tested on their own.

type fakeReconcileRepo struct {
	stock.Repository // nil: any other method would panic
	data             stock.ReconcileData
	err              error
	calls            int
	last             stock.ReconcileFilter
}

func (r *fakeReconcileRepo) Reconcile(_ context.Context, f stock.ReconcileFilter) (stock.ReconcileData, error) {
	r.calls++
	r.last = f
	return r.data, r.err
}

const (
	locA = "11111111-1111-4111-8111-111111111111"
	locB = "22222222-2222-4222-8222-222222222222"
)

func s(v string) *string { return &v }

func sampleData() stock.ReconcileData {
	return stock.ReconcileData{
		VehicleUnitsChecked: 3, PartBalancesChecked: 2, MovementsChecked: 9,
		Locations: map[string]stock.ReconcileLocation{locA: {Code: "SR1", Name: "Showroom"}, locB: {Code: "WH", Name: "Warehouse"}},
		Discrepancies: []stock.RawDiscrepancy{
			{Type: stock.DiscPartsBrokenChain, ItemType: stock.ItemStockItem,
				SKU: &stock.ReconcileSKU{ID: "sku-1", SKU: "PRT-1", Name: "Pad", UOM: "PCS"}, LocationID: s(locA),
				ExpectedQuantity: s("10"), ActualQuantity: s("9"), MovementIDs: []string{"m1", "m2"}},
			{Type: stock.DiscVehicleStatusMismatch, ItemType: stock.ItemVehicleUnit,
				Unit: &stock.ReconcileUnit{ID: "unit-1", UnitNumber: "MAE-VEH-000001"}, BrandID: s("b1"), BrandName: s("Brand A"),
				LocationID: s(locA), ExpectedStatus: s("AVAILABLE"), ActualStatus: s("DISPLAY"), MovementIDs: []string{"m3"}},
			{Type: stock.DiscVehicleLocationMismatch, ItemType: stock.ItemVehicleUnit,
				Unit: &stock.ReconcileUnit{ID: "unit-1", UnitNumber: "MAE-VEH-000001"}, LocationID: s(locA),
				ExpectedLocation: s(locB), ActualLocation: s(locA), MovementIDs: []string{"m3"}},
			{Type: stock.DiscPartsBalanceMismatch, ItemType: stock.ItemStockItem,
				SKU: &stock.ReconcileSKU{ID: "sku-1", SKU: "PRT-1", Name: "Pad", UOM: "PCS"}, LocationID: s(locB),
				ExpectedQuantity: s("4"), ActualQuantity: s("5.5"), MovementIDs: []string{"m4"}},
		},
	}
}

func reconcileHarness(t *testing.T, repo stock.Repository) (*harness, session) {
	t.Helper()
	store := authtest.NewStore()
	u := store.AddUser("asha", "Asha Verma", testPassword, auth.StatusActive)
	h := newHarness(t, store, repo)
	return h, h.login("asha", u.ID)
}

func getReport(t *testing.T, h *harness, sess session, query string, status int) stock.ReconcileReport {
	t.Helper()
	var r stock.ReconcileReport
	w := h.raw(http.MethodGet, "/api/v1/stock-reconciliation?"+query, "", &sess, nil)
	if status != http.StatusOK {
		if w.Code != status {
			t.Fatalf("%s: status %d, want %d: %s", query, w.Code, status, w.Body)
		}
		return r
	}
	expect(t, w, http.StatusOK, &r)
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q", w.Header().Get("Cache-Control"))
	}
	return r
}

func TestReconcileRouteDeniesByDefault(t *testing.T) {
	repo := &fakeReconcileRepo{data: sampleData()}
	h, sess := reconcileHarness(t, repo)
	if w := h.raw(http.MethodGet, "/api/v1/stock-reconciliation", "", nil, nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("without session: %d", w.Code)
	}
	expectError(t, h.raw(http.MethodGet, "/api/v1/stock-reconciliation", "", &sess, nil), http.StatusForbidden, "FORBIDDEN")
	// Viewing items (any location) grants no report.
	h.grant(sess, access.PermVehicleUnitView, access.PermPartsView, access.PermAdjustmentApprove, access.PermAllLocations)
	expectError(t, h.raw(http.MethodGet, "/api/v1/stock-reconciliation", "", &sess, nil), http.StatusForbidden, "FORBIDDEN")
	// The report permission alone shows no rows: it is refused rather than empty.
	h.grant(sess, access.PermReconciliationView, access.PermAllLocations)
	expectError(t, h.raw(http.MethodGet, "/api/v1/stock-reconciliation", "", &sess, nil), http.StatusForbidden, "FORBIDDEN")
	if repo.calls != 0 {
		t.Fatalf("repository reached %d times before permissions passed", repo.calls)
	}
	// No route writes, repairs or deletes through the reconciliation.
	h.grant(sess, access.PermReconciliationView, access.PermVehicleUnitView, access.PermPartsView, access.PermAllLocations)
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		w := h.raw(m, "/api/v1/stock-reconciliation", `{}`, &sess, nil)
		if w.Code != http.StatusMethodNotAllowed && w.Code != http.StatusNotFound {
			t.Errorf("%s /stock-reconciliation: %d", m, w.Code)
		}
	}
	for _, p := range []string{"/api/v1/stock-reconciliation/repair", "/api/v1/stock-reconciliation/runs"} {
		if w := h.raw(http.MethodPost, p, `{}`, &sess, nil); w.Code != http.StatusNotFound {
			t.Errorf("POST %s: %d", p, w.Code)
		}
	}
	if repo.calls != 0 {
		t.Fatalf("a write attempt reached the repository")
	}
}

func TestReconcileVisibilityFollowsItemPermissions(t *testing.T) {
	repo := &fakeReconcileRepo{data: sampleData()}
	h, sess := reconcileHarness(t, repo)
	h.grant(sess, access.PermReconciliationView, access.PermVehicleUnitView)
	h.scope(sess, locA)
	r := getReport(t, h, sess, "", http.StatusOK)
	if !repo.last.Vehicles || repo.last.Parts || repo.last.AllLocations || len(repo.last.ScopeIDs) != 1 || repo.last.ScopeIDs[0] != locA {
		t.Fatalf("filter %+v", repo.last)
	}
	if !r.Coverage.VehicleUnits || r.Coverage.Parts || r.Coverage.AllLocations || len(r.Coverage.LocationIDs) != 1 {
		t.Fatalf("coverage %+v", r.Coverage)
	}
	// A location outside the scope is a validation error, not an empty page.
	w := h.raw(http.MethodGet, "/api/v1/stock-reconciliation?locationId="+locB, "", &sess, nil)
	expectError(t, w, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	// Expected location B (outside the scope) is named by ID only; the record's own location by code.
	for _, d := range r.Items {
		if d.Type != stock.DiscVehicleLocationMismatch {
			continue
		}
		if d.Expected == nil || !strings.Contains(*d.Expected, locB) || strings.Contains(*d.Expected, "WH") {
			t.Errorf("expected = %v", d.Expected)
		}
		if d.Actual == nil || *d.Actual != "SR1" || d.LocationCode == nil || *d.LocationCode != "SR1" {
			t.Errorf("actual = %v, location code %v", d.Actual, d.LocationCode)
		}
	}
	// Parts only.
	h.grant(sess, access.PermReconciliationView, access.PermPartsView, access.PermAllLocations)
	getReport(t, h, sess, "", http.StatusOK)
	if repo.last.Vehicles || !repo.last.Parts || !repo.last.AllLocations {
		t.Fatalf("filter %+v", repo.last)
	}
}

func TestReconcileShapesOrdersFiltersAndPages(t *testing.T) {
	repo := &fakeReconcileRepo{data: sampleData()}
	h, sess := reconcileHarness(t, repo)
	h.grant(sess, access.PermReconciliationView, access.PermVehicleUnitView, access.PermPartsView, access.PermAllLocations)
	r := getReport(t, h, sess, "", http.StatusOK)
	if r.Status != "COMPLETED" || r.RunID == "" || r.RunAt.IsZero() || r.CompletedAt.Before(r.RunAt) {
		t.Fatalf("run %+v", r)
	}
	sm := r.Summary
	if sm.VehicleUnitsChecked != 3 || sm.PartBalancesChecked != 2 || sm.RecordsChecked != 5 || sm.MovementsChecked != 9 ||
		sm.Discrepancies != 4 || sm.BySeverity["HIGH"] != 3 || sm.BySeverity["MEDIUM"] != 1 || sm.ByType[stock.DiscVehicleStatusMismatch] != 1 {
		t.Fatalf("summary %+v", sm)
	}
	// Severity first (HIGH), then type, then reference.
	want := []string{stock.DiscPartsBalanceMismatch, stock.DiscVehicleLocationMismatch, stock.DiscVehicleStatusMismatch, stock.DiscPartsBrokenChain}
	if len(r.Items) != 4 || r.Total != 4 {
		t.Fatalf("items %d total %d", len(r.Items), r.Total)
	}
	for i, d := range r.Items {
		if d.Type != want[i] {
			t.Fatalf("item %d type %s, want %s", i, d.Type, want[i])
		}
		if d.Explanation == "" || !d.DetectedAt.Equal(r.RunAt) || d.MovementIDs == nil {
			t.Errorf("item %d: %+v", i, d)
		}
	}
	bal := r.Items[0]
	if bal.Reference != stock.DiscPartsBalanceMismatch+":sku-1:"+locB || *bal.Expected != "4 PCS" || *bal.Actual != "5.5 PCS" ||
		*bal.LocationCode != "WH" || bal.BrandID != nil || bal.VehicleUnit != nil || bal.StockItem.SKU != "PRT-1" {
		t.Errorf("balance row %+v", bal)
	}
	st := r.Items[2]
	if st.Reference != stock.DiscVehicleStatusMismatch+":unit-1" || *st.Expected != "AVAILABLE" || *st.Actual != "DISPLAY" ||
		st.Severity != "HIGH" || *st.BrandName != "Brand A" || st.MovementIDs[0] != "m3" {
		t.Errorf("status row %+v", st)
	}
	chain := r.Items[3]
	if chain.Reference != stock.DiscPartsBrokenChain+":sku-1:"+locA+":m2" || chain.Severity != "MEDIUM" || *chain.Expected != "10 PCS" {
		t.Errorf("chain row %+v", chain)
	}
	// Filters keep the whole-run summary.
	f := getReport(t, h, sess, "severity=MEDIUM", http.StatusOK)
	if f.Total != 1 || f.Items[0].Type != stock.DiscPartsBrokenChain || f.Summary.Discrepancies != 4 {
		t.Fatalf("severity filter: %+v", f)
	}
	if f := getReport(t, h, sess, "itemType=VEHICLE_UNIT", http.StatusOK); f.Total != 2 {
		t.Fatalf("itemType filter total %d", f.Total)
	}
	if f := getReport(t, h, sess, "type=PARTS_BALANCE_MISMATCH&locationId="+locB, http.StatusOK); f.Total != 1 {
		t.Fatalf("type+location filter total %d", f.Total)
	}
	if f := getReport(t, h, sess, "locationId="+locA, http.StatusOK); f.Total != 3 {
		t.Fatalf("location filter total %d", f.Total)
	}
	// Paging.
	p2 := getReport(t, h, sess, "limit=3&page=2", http.StatusOK)
	if p2.Total != 4 || len(p2.Items) != 1 || p2.Items[0].Type != stock.DiscPartsBrokenChain || p2.Page != 2 || p2.Limit != 3 {
		t.Fatalf("page 2: %+v", p2)
	}
	if p9 := getReport(t, h, sess, "page=9", http.StatusOK); len(p9.Items) != 0 || p9.Total != 4 {
		t.Fatalf("page 9: %+v", p9)
	}
}

func TestReconcileValidatesTheQuery(t *testing.T) {
	repo := &fakeReconcileRepo{data: sampleData()}
	h, sess := reconcileHarness(t, repo)
	h.grant(sess, access.PermReconciliationView, access.PermVehicleUnitView, access.PermAllLocations)
	for q, field := range map[string]string{
		"itemType=CAR": "itemType", "type=EVERYTHING": "type", "severity=LOW": "severity", "locationId=x": "locationId",
		"page=0": "page", "page=a": "page", "limit=0": "limit", "limit=101": "limit",
	} {
		e := expectError(t, h.raw(http.MethodGet, "/api/v1/stock-reconciliation?"+q, "", &sess, nil), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
		if len(e.Details) == 0 || e.Details[0].Field != field {
			t.Errorf("%s: details %+v", q, e.Details)
		}
	}
	if repo.calls != 0 {
		t.Fatalf("invalid queries reached the repository %d times", repo.calls)
	}
}

func TestReconcileFailureIsSafeAndLogged(t *testing.T) {
	repo := &fakeReconcileRepo{err: errors.New("connection reset by peer: secret-dsn-detail")}
	h, sess := reconcileHarness(t, repo)
	h.grant(sess, access.PermReconciliationView, access.PermVehicleUnitView, access.PermAllLocations)
	w := h.raw(http.MethodGet, "/api/v1/stock-reconciliation", "", &sess, nil)
	expectError(t, w, http.StatusInternalServerError, "INTERNAL_ERROR")
	if strings.Contains(w.Body.String(), "secret-dsn-detail") || strings.Contains(w.Body.String(), "items") {
		t.Fatalf("failure leaked details or a partial report: %s", w.Body)
	}

	var buf bytes.Buffer
	svc := stock.NewService(repo, slog.New(slog.NewJSONHandler(&buf, nil)))
	a := access.NewAccess("user-1", nil, []string{access.PermReconciliationView, access.PermPartsView, access.PermAllLocations}, nil, false)
	if _, err := svc.Reconcile(context.Background(), a, stock.ReconcileQuery{}); err == nil {
		t.Fatal("no error")
	}
	logs := buf.String()
	if !strings.Contains(logs, "stock reconciliation started") || !strings.Contains(logs, "stock reconciliation failed") ||
		!strings.Contains(logs, `"run_id"`) || !strings.Contains(logs, `"user_id":"user-1"`) {
		t.Fatalf("logs: %s", logs)
	}

	buf.Reset()
	repo.err, repo.data = nil, sampleData()
	if _, err := svc.Reconcile(context.Background(), a, stock.ReconcileQuery{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "stock reconciliation completed") || !strings.Contains(buf.String(), `"discrepancies":4`) {
		t.Fatalf("logs: %s", buf.String())
	}
}
