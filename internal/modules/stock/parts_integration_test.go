package stock_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database/dbtest"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/stock"
)

// Parts stock (E-04) on a throwaway migrated database. SKU codes, units and
// supplier names are placeholders ("PRT-1", "PCS", "Example Parts Supplier").

var partsAll = []string{access.PermPartsView, access.PermPartsReceive, access.PermPartsIssue, access.PermSupplierView,
	access.PermAllLocations}

type partsFixture struct {
	*fixture
	category, item, serial, archived        string
	supplier, archivedSupplier, inactiveLoc string
}

func setupParts(t *testing.T) *partsFixture {
	t.Helper()
	return partsOn(t, setup(t))
}

func partsOn(t *testing.T, f *fixture) *partsFixture {
	t.Helper()
	p := &partsFixture{fixture: f}
	f.h.grant(f.mover, partsAll...)
	p.category = f.scalar(t, `INSERT INTO stock_item_categories (name, created_by) VALUES ('Brake parts', $1) RETURNING id::text`, f.ids["buyer"])
	item := func(sku, mode string) string {
		return f.scalar(t, `INSERT INTO stock_items (sku, name, category_id, uom, tracking_mode, created_by)
			VALUES ($1, 'Item '||$1, $2, 'PCS', $3, $4) RETURNING id::text`, sku, p.category, mode, f.ids["buyer"])
	}
	p.item, p.serial, p.archived = item("PRT-1", "QUANTITY"), item("PRT-SER", "SERIAL"), item("PRT-OLD", "QUANTITY")
	f.exec(t, `UPDATE stock_items SET status = 'ARCHIVED', archived_by = $2, archived_at = now(), archive_reason = 'test' WHERE id = $1`,
		p.archived, f.ids["buyer"])
	p.supplier = f.scalar(t, `INSERT INTO suppliers (legal_name, created_by) VALUES ('Example Parts Supplier', $1) RETURNING id::text`, f.ids["buyer"])
	p.archivedSupplier = f.scalar(t, `INSERT INTO suppliers (legal_name, created_by) VALUES ('Example Parts Supplier Old', $1) RETURNING id::text`, f.ids["buyer"])
	f.exec(t, `UPDATE suppliers SET status = 'ARCHIVED' WHERE id = $1`, p.archivedSupplier)
	p.inactiveLoc = f.scalar(t, `INSERT INTO locations (code, name, active) VALUES ('L9', 'Closed', false) RETURNING id::text`)
	return p
}

func (p *partsFixture) call(s session, item, op string, body map[string]any, key string) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	return p.h.raw(http.MethodPost, "/api/v1/stock-items/"+item+"/"+op, string(b), &s, map[string]string{stock.IdempotencyHeader: key})
}

func (p *partsFixture) receipt(qty string, extra ...any) map[string]any {
	b := map[string]any{"locationId": p.locL1, "quantity": qty, "uom": "PCS", "supplierId": p.supplier, "sourceReference": "DC-1"}
	for i := 0; i+1 < len(extra); i += 2 {
		if extra[i+1] == nil {
			delete(b, extra[i].(string))
		} else {
			b[extra[i].(string)] = extra[i+1]
		}
	}
	return b
}

func (p *partsFixture) issue(qty string, extra ...any) map[string]any {
	b := map[string]any{"locationId": p.locL1, "quantity": qty, "uom": "PCS", "reason": "Fitted in a repair"}
	for i := 0; i+1 < len(extra); i += 2 {
		if extra[i+1] == nil {
			delete(b, extra[i].(string))
		} else {
			b[extra[i].(string)] = extra[i+1]
		}
	}
	return b
}

func (p *partsFixture) balance(t *testing.T, item, loc string) string {
	t.Helper()
	return p.scalar(t, `SELECT coalesce((SELECT quantity::text FROM stock_item_balances WHERE stock_item_id = $1 AND location_id = $2), 'none')`,
		item, loc)
}

func (p *partsFixture) partsRows(t *testing.T) int {
	return p.count(t, `SELECT count(*) FROM stock_movements WHERE item_type = 'STOCK_ITEM'`) +
		p.count(t, `SELECT count(*) FROM audit_events WHERE action IN ('stock_item.received', 'stock_item.issued')`)
}

// reconcilesParts: every balance equals its latest movement; each movement starts where the previous ended.
func (p *partsFixture) reconcilesParts(t *testing.T) {
	t.Helper()
	if n := p.count(t, `SELECT count(*) FROM stock_item_balances b WHERE b.quantity IS DISTINCT FROM coalesce((SELECT m.quantity_after
		FROM stock_movements m WHERE m.stock_item_id = b.stock_item_id AND m.to_location_id = b.location_id ORDER BY m.seq DESC LIMIT 1), 0)`); n != 0 {
		t.Errorf("%d balances differ from their ledger", n)
	}
	if n := p.count(t, `SELECT count(*) FROM (SELECT quantity_before, lag(quantity_after, 1, 0::numeric) OVER (PARTITION BY stock_item_id,
		to_location_id ORDER BY seq) AS prev FROM stock_movements WHERE item_type = 'STOCK_ITEM') x WHERE quantity_before <> prev`); n != 0 {
		t.Errorf("%d movements do not start where the previous one ended", n)
	}
}

func TestPartsReceiptAndIssueMoveTheBalance(t *testing.T) {
	p := setupParts(t)
	var res stock.PartsResult
	expect(t, p.call(p.mover, p.item, "receipts", p.receipt("10", "note", " first delivery "), "rcpt-0001"), http.StatusOK, &res)
	m := res.Movement
	if m.MovementType != stock.TypePartsReceipt || m.Quantity != "10.000" || m.QuantityBefore != "0.000" || m.QuantityAfter != "10.000" ||
		m.UOM != "PCS" || m.SourceType != stock.SourceSupplierDelivery || *m.SupplierID != p.supplier || *m.SupplierName != "Example Parts Supplier" ||
		*m.SourceReference != "DC-1" || *m.Note != "first delivery" || m.ActorID != p.ids["mover"] || m.LocationID != p.locL1 || m.SKU != "PRT-1" {
		t.Fatalf("receipt %+v", m)
	}
	if res.Balance.Quantity != "10.000" || res.Balance.Version != 2 {
		t.Fatalf("balance %+v", res.Balance)
	}
	expect(t, p.call(p.mover, p.item, "issues", p.issue("3.5", "sourceReference", "JOB-1"), "issue-0001"), http.StatusOK, &res)
	if res.Movement.QuantityBefore != "10.000" || res.Movement.QuantityAfter != "6.500" || res.Movement.Quantity != "3.500" ||
		*res.Movement.Reason != "Fitted in a repair" || res.Movement.SupplierID != nil || res.Balance.Quantity != "6.500" {
		t.Fatalf("issue %+v", res)
	}
	// Exactly one movement and one audit event each.
	if n := p.count(t, `SELECT count(*) FROM stock_movements WHERE stock_item_id = $1`, p.item); n != 2 {
		t.Fatalf("movements %d", n)
	}
	if n := p.count(t, `SELECT count(*) FROM audit_events a JOIN stock_movements m ON m.id::text = a.summary->>'movementId'
		WHERE a.action IN ('stock_item.received', 'stock_item.issued') AND a.entity_id = $1 AND a.occurred_at = m.created_at`, p.item); n != 2 {
		t.Fatalf("audit %d", n)
	}
	// Reads: balances and history (newest first), with the reading user's scope.
	var bal []stock.PartsBalance
	expect(t, p.h.raw(http.MethodGet, "/api/v1/stock-items/"+p.item+"/balances", "", &p.mover, nil), http.StatusOK, &bal)
	if len(bal) != 1 || bal[0].Quantity != "6.500" || bal[0].LocationCode != "L1" || bal[0].UOM != "PCS" {
		t.Fatalf("balances %+v", bal)
	}
	var hist stock.PartsPage
	expect(t, p.h.raw(http.MethodGet, "/api/v1/stock-items/"+p.item+"/stock-movements", "", &p.mover, nil), http.StatusOK, &hist)
	if hist.Total != 2 || hist.Items[0].MovementType != stock.TypePartsIssue || hist.Items[1].MovementType != stock.TypePartsReceipt {
		t.Fatalf("history %+v", hist)
	}
	expect(t, p.h.raw(http.MethodGet, "/api/v1/stock-items/"+p.item+"/stock-movements?type=PARTS_RECEIPT", "", &p.mover, nil), http.StatusOK, &hist)
	if hist.Total != 1 {
		t.Fatalf("type filter %+v", hist)
	}
	// Parts movements touch no vehicle unit.
	if n := p.count(t, `SELECT count(*) FROM vehicle_units`); n != 0 {
		t.Fatalf("vehicle units %d", n)
	}
	p.reconcilesParts(t)
}

func TestIssueCannotExceedTheBalance(t *testing.T) {
	p := setupParts(t)
	expect(t, p.call(p.mover, p.item, "receipts", p.receipt("2"), "rcpt-0001"), http.StatusOK, nil)
	before := p.partsRows(t)
	e := expectError(t, p.call(p.mover, p.item, "issues", p.issue("2.001"), "issue-0001"), http.StatusConflict, "INSUFFICIENT_STOCK")
	if len(e.Details) != 1 || e.Details[0].Field != "quantity" {
		t.Fatalf("details %+v", e.Details)
	}
	// A location that never had stock: refused, and no empty balance row is left behind.
	expectError(t, p.call(p.mover, p.item, "issues", p.issue("1", "locationId", p.locL2), "issue-0002"), http.StatusConflict, "INSUFFICIENT_STOCK")
	if after := p.partsRows(t); after != before {
		t.Fatalf("refused issues left rows: %d -> %d", before, after)
	}
	if b := p.balance(t, p.item, p.locL1); b != "2.000" {
		t.Fatalf("balance %s", b)
	}
	if b := p.balance(t, p.item, p.locL2); b != "none" {
		t.Fatalf("L2 balance row %s", b)
	}
	// The refused key was not used up: the same key can be sent with a quantity that fits.
	expect(t, p.call(p.mover, p.item, "issues", p.issue("2"), "issue-0001"), http.StatusOK, nil)
	if b := p.balance(t, p.item, p.locL1); b != "0.000" {
		t.Fatalf("balance %s", b)
	}
	p.reconcilesParts(t)
}

func TestPartsRequestsAreValidated(t *testing.T) {
	p := setupParts(t)
	expectError(t, p.call(p.mover, "00000000-0000-0000-0000-000000000000", "receipts", p.receipt("1"), "v-0000001"), http.StatusNotFound, "NOT_FOUND")
	expectError(t, p.call(p.mover, p.archived, "receipts", p.receipt("1"), "v-0000002"), http.StatusConflict, "INVALID_STATE")
	expectError(t, p.call(p.mover, p.serial, "receipts", p.receipt("1"), "v-0000003"), http.StatusConflict, "TRACKING_NOT_SUPPORTED")
	expectError(t, p.call(p.mover, p.serial, "issues", p.issue("1"), "v-0000004"), http.StatusConflict, "TRACKING_NOT_SUPPORTED")
	for i, c := range []struct {
		op    string
		body  map[string]any
		field string
	}{
		{"receipts", p.receipt("1", "uom", "SET"), "uom"},
		{"receipts", p.receipt("1.2345"), "quantity"},
		{"receipts", p.receipt("0"), "quantity"},
		{"receipts", p.receipt("0.000"), "quantity"},
		{"receipts", p.receipt("-1"), "quantity"},
		{"receipts", p.receipt("1e3"), "quantity"},
		{"receipts", p.receipt("1", "supplierId", nil), "supplierId"},
		{"receipts", p.receipt("1", "sourceReference", "  "), "sourceReference"},
		{"receipts", p.receipt("1", "supplierId", p.archivedSupplier), "supplierId"},
		{"receipts", p.receipt("1", "supplierId", "00000000-0000-0000-0000-000000000000"), "supplierId"},
		{"receipts", p.receipt("1", "locationId", "00000000-0000-0000-0000-000000000000"), "locationId"},
		{"receipts", p.receipt("1", "locationId", p.inactiveLoc), "locationId"},
		// The client cannot name the balance, the result or the actor.
		{"receipts", p.receipt("1", "quantityAfter", "100"), "quantityAfter"},
		{"receipts", p.receipt("1", "balance", "100"), "balance"},
		{"receipts", p.receipt("1", "actorId", p.ids["buyer"]), "actorId"},
		{"issues", p.issue("1", "reason", nil), "reason"},
		{"issues", p.issue("1", "supplierId", p.supplier), "supplierId"},
		{"issues", p.issue("1", "quantity", 5), "quantity"},
	} {
		e := expectError(t, p.call(p.mover, p.item, c.op, c.body, "val-"+strconv.Itoa(1000+i)), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
		if !strings.Contains(e.Details[0].Field, c.field) {
			t.Errorf("%d %s: %+v", i, c.field, e.Details)
		}
	}
	expectError(t, p.call(p.mover, p.item, "receipts", p.receipt("1"), ""), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if n := p.partsRows(t); n != 0 {
		t.Fatalf("refused requests left %d rows", n)
	}
}

func TestPartsPermissionsAndLocationScope(t *testing.T) {
	p := setupParts(t)
	expect(t, p.call(p.mover, p.item, "receipts", p.receipt("5"), "rcpt-0001"), http.StatusOK, nil)
	expect(t, p.call(p.mover, p.item, "receipts", p.receipt("4", "locationId", p.locL2), "rcpt-0002"), http.StatusOK, nil)
	paths := []struct{ method, path string }{
		{http.MethodGet, "/balances"}, {http.MethodGet, "/stock-movements"}, {http.MethodPost, "/receipts"}, {http.MethodPost, "/issues"},
	}
	// No grant, and neighbouring grants: nothing.
	for _, perms := range [][]string{nil, {access.PermStockItemView, access.PermStockItemManage, access.PermAdjustmentCreate,
		access.PermVehicleUnitDisplayManage, access.PermAllLocations}} {
		p.h.grant(p.other, perms...)
		for _, r := range paths {
			body := ""
			if r.method == http.MethodPost {
				b, _ := json.Marshal(p.receipt("1"))
				body = string(b)
			}
			w := p.h.raw(r.method, "/api/v1/stock-items/"+p.item+r.path, body, &p.other, map[string]string{stock.IdempotencyHeader: "perm-0001"})
			expectError(t, w, http.StatusForbidden, "FORBIDDEN")
		}
	}
	// View only; issue only; receipt needs suppliers.view too.
	p.h.grant(p.other, access.PermPartsView, access.PermAllLocations)
	expectError(t, p.call(p.other, p.item, "issues", p.issue("1"), "perm-0002"), http.StatusForbidden, "FORBIDDEN")
	p.h.grant(p.other, access.PermPartsIssue, access.PermAllLocations)
	expectError(t, p.call(p.other, p.item, "receipts", p.receipt("1"), "perm-0003"), http.StatusForbidden, "FORBIDDEN")
	expect(t, p.call(p.other, p.item, "issues", p.issue("1"), "perm-0004"), http.StatusOK, nil)
	p.h.grant(p.other, access.PermPartsReceive, access.PermAllLocations)
	expectError(t, p.call(p.other, p.item, "receipts", p.receipt("1"), "perm-0005"), http.StatusForbidden, "FORBIDDEN")
	// Scope: the L2-only user sees L2 only, cannot post at L1, and does not see suppliers without suppliers.view.
	p.h.grant(p.scoped, access.PermPartsView, access.PermPartsIssue)
	var bal []stock.PartsBalance
	expect(t, p.h.raw(http.MethodGet, "/api/v1/stock-items/"+p.item+"/balances", "", &p.scoped, nil), http.StatusOK, &bal)
	if len(bal) != 1 || bal[0].LocationID != p.locL2 || bal[0].Quantity != "4.000" {
		t.Fatalf("scoped balances %+v", bal)
	}
	var hist stock.PartsPage
	expect(t, p.h.raw(http.MethodGet, "/api/v1/stock-items/"+p.item+"/stock-movements", "", &p.scoped, nil), http.StatusOK, &hist)
	if hist.Total != 1 || hist.Items[0].LocationID != p.locL2 || hist.Items[0].SupplierID != nil || hist.Items[0].SupplierName != nil {
		t.Fatalf("scoped history %+v", hist)
	}
	expectError(t, p.h.raw(http.MethodGet, "/api/v1/stock-items/"+p.item+"/stock-movements?locationId="+p.locL1, "", &p.scoped, nil),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	e := expectError(t, p.call(p.scoped, p.item, "issues", p.issue("1"), "scope-0001"), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if e.Details[0].Field != "locationId" {
		t.Fatalf("details %+v", e.Details)
	}
	expect(t, p.call(p.scoped, p.item, "issues", p.issue("1", "locationId", p.locL2), "scope-0002"), http.StatusOK, nil)
	if b := p.balance(t, p.item, p.locL1); b != "4.000" {
		t.Fatalf("L1 %s", b)
	}
	p.reconcilesParts(t)
}

func TestPartsIdempotency(t *testing.T) {
	p := setupParts(t)
	var first, again stock.PartsResult
	expect(t, p.call(p.mover, p.item, "receipts", p.receipt("5"), "same-key-0001"), http.StatusOK, &first)
	// The same request (5 and 5.000 are the same quantity) replays; nothing changes.
	w := p.call(p.mover, p.item, "receipts", p.receipt("5.000"), "same-key-0001")
	expect(t, w, http.StatusOK, &again)
	if w.Header().Get("Idempotent-Replayed") != "true" || again.Movement.ID != first.Movement.ID || again.Balance.Quantity != "5.000" {
		t.Fatalf("replay %s %+v", w.Header().Get("Idempotent-Replayed"), again)
	}
	// The same key for anything else: refused.
	for _, body := range []map[string]any{p.receipt("6"), p.receipt("5", "locationId", p.locL2), p.receipt("5", "sourceReference", "DC-2")} {
		expectError(t, p.call(p.mover, p.item, "receipts", body, "same-key-0001"), http.StatusConflict, "IDEMPOTENCY_KEY_REUSED")
	}
	expectError(t, p.call(p.mover, p.item, "issues", p.issue("5"), "same-key-0001"), http.StatusConflict, "IDEMPOTENCY_KEY_REUSED")
	// One key space per user across the ledger: a parts key is not a vehicle key and vice versa.
	unit := p.available(t, 1)[0]
	p.h.grant(p.mover, append(partsAll, access.PermVehicleUnitDisplayManage)...)
	expectError(t, p.h.display(p.mover, unit, true, `{"version":2}`, "same-key-0001"), http.StatusConflict, "IDEMPOTENCY_KEY_REUSED")
	expect(t, p.h.display(p.mover, unit, true, `{"version":2}`, "vehicle-key-01"), http.StatusOK, nil)
	expectError(t, p.call(p.mover, p.item, "receipts", p.receipt("1"), "vehicle-key-01"), http.StatusConflict, "IDEMPOTENCY_KEY_REUSED")
	if b := p.balance(t, p.item, p.locL1); b != "5.000" {
		t.Fatalf("balance %s", b)
	}
	if n := p.count(t, `SELECT count(*) FROM stock_movements WHERE item_type = 'STOCK_ITEM'`); n != 1 {
		t.Fatalf("parts movements %d", n)
	}
	p.reconcilesParts(t)
}

func parallel(n int, fn func(i int) *httptest.ResponseRecorder) []*httptest.ResponseRecorder {
	var wg sync.WaitGroup
	out := make([]*httptest.ResponseRecorder, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out[i] = fn(i)
		}()
	}
	wg.Wait()
	return out
}

func codes(ws []*httptest.ResponseRecorder) (ok int, other map[string]int) {
	other = map[string]int{}
	for _, w := range ws {
		if w.Code == http.StatusOK {
			ok++
			continue
		}
		var env struct {
			Error struct{ Code string } `json:"error"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		other[env.Error.Code]++
	}
	return ok, other
}

// Stock never goes negative, whatever arrives at the same time.
func TestConcurrentIssuesNeverGoNegative(t *testing.T) {
	p := setupParts(t)
	expect(t, p.call(p.mover, p.item, "receipts", p.receipt("10"), "rcpt-0001"), http.StatusOK, nil)
	ok, other := codes(parallel(2, func(i int) *httptest.ResponseRecorder {
		return p.call(p.mover, p.item, "issues", p.issue("7"), "seven-"+strconv.Itoa(1000+i))
	}))
	if ok != 1 || other["INSUFFICIENT_STOCK"] != 1 {
		t.Fatalf("two issues of 7 from 10: %d ok, %v", ok, other)
	}
	if b := p.balance(t, p.item, p.locL1); b != "3.000" {
		t.Fatalf("balance %s", b)
	}
	expect(t, p.call(p.mover, p.item, "issues", p.issue("2"), "down-to-one"), http.StatusOK, nil)
	ok, other = codes(parallel(8, func(i int) *httptest.ResponseRecorder {
		return p.call(p.mover, p.item, "issues", p.issue("1"), "one-"+strconv.Itoa(1000+i))
	}))
	if ok != 1 || other["INSUFFICIENT_STOCK"] != 7 {
		t.Fatalf("eight issues of 1 from 1: %d ok, %v", ok, other)
	}
	if b := p.balance(t, p.item, p.locL1); b != "0.000" {
		t.Fatalf("balance %s", b)
	}
	if n := p.count(t, `SELECT count(*) FROM stock_movements WHERE item_type = 'STOCK_ITEM'`); n != 4 {
		t.Fatalf("movements %d", n)
	}
	p.reconcilesParts(t)
}

// Concurrent first receipts at a new location all count; one key at a time counts once.
func TestConcurrentReceipts(t *testing.T) {
	p := setupParts(t)
	ok, other := codes(parallel(8, func(i int) *httptest.ResponseRecorder {
		return p.call(p.mover, p.item, "receipts", p.receipt("1.5", "locationId", p.locL2), "first-"+strconv.Itoa(1000+i))
	}))
	if ok != 8 || len(other) != 0 {
		t.Fatalf("first receipts: %d ok, %v", ok, other)
	}
	if b := p.balance(t, p.item, p.locL2); b != "12.000" {
		t.Fatalf("balance %s", b)
	}
	ws := parallel(8, func(int) *httptest.ResponseRecorder {
		return p.call(p.mover, p.item, "receipts", p.receipt("2"), "one-key-0001")
	})
	fresh, replays := 0, 0
	for _, w := range ws {
		expect(t, w, http.StatusOK, nil)
		if w.Header().Get("Idempotent-Replayed") == "true" {
			replays++
		} else {
			fresh++
		}
	}
	if fresh != 1 || replays != 7 {
		t.Fatalf("one key: %d fresh, %d replays", fresh, replays)
	}
	if b := p.balance(t, p.item, p.locL1); b != "2.000" {
		t.Fatalf("balance %s", b)
	}
	if n := p.count(t, `SELECT count(*) FROM stock_movements WHERE item_type = 'STOCK_ITEM'`) +
		p.count(t, `SELECT count(*) FROM audit_events WHERE action = 'stock_item.received'`); n != 18 {
		t.Fatalf("movements + audit %d", n)
	}
	p.reconcilesParts(t)
}

// The database, not only the API, keeps balances tied to the ledger and the vehicle rules intact.
func TestDatabaseGuardsPartsStock(t *testing.T) {
	p := setupParts(t)
	ctx := context.Background()
	expect(t, p.call(p.mover, p.item, "receipts", p.receipt("5"), "rcpt-0001"), http.StatusOK, nil)
	mov := p.scalar(t, `SELECT id::text FROM stock_movements WHERE item_type = 'STOCK_ITEM'`)
	for _, sql := range []string{
		`UPDATE stock_item_balances SET quantity = 100`,
		`UPDATE stock_item_balances SET quantity = 100, version = version + 1`,
		`DELETE FROM stock_item_balances`,
		`TRUNCATE stock_item_balances`,
		`INSERT INTO stock_item_balances (stock_item_id, location_id, quantity) VALUES ('` + p.item + `', '` + p.locL2 + `', 5)`,
		`UPDATE stock_movements SET quantity_after = 100 WHERE id = '` + mov + `'`,
		`DELETE FROM stock_movements WHERE id = '` + mov + `'`,
	} {
		if _, err := p.pool.Exec(ctx, sql); func() bool { c, _ := sqlState(err); return c != "42501" }() {
			t.Errorf("%s: %v", sql, err)
		}
	}
	insert := func(item, typ, before, after, uom, extra string) error {
		_, err := p.pool.Exec(ctx, `INSERT INTO stock_movements (item_type, stock_item_id, movement_type, quantity_before, quantity_after,
				from_location_id, to_location_id, uom, source_type, supplier_id, source_reference, reason, actor_id, idempotency_key, request_hash`+extra+`)
			VALUES ('STOCK_ITEM', $1, $2, $3::numeric, $4::numeric, $5, $5, $6,
				CASE WHEN $2 = 'PARTS_RECEIPT' THEN 'SUPPLIER_DELIVERY' ELSE 'STAFF_ACTION' END,
				CASE WHEN $2 = 'PARTS_RECEIPT' THEN $7::uuid END, CASE WHEN $2 = 'PARTS_RECEIPT' THEN 'DC-9' END,
				CASE WHEN $2 = 'PARTS_ISSUE' THEN 'direct' END, $8, 'direct-' || md5(random()::text), repeat('0', 64))`,
			item, typ, before, after, p.locL1, uom, p.supplier, p.ids["buyer"])
		return err
	}
	for _, c := range []struct {
		name, item, typ, before, after, uom, constraint string
	}{
		{"stale before", p.item, "PARTS_ISSUE", "4", "3", "PCS", "stock_movements_from_matches_balance"},
		{"below zero", p.item, "PARTS_ISSUE", "5", "-1", "PCS", "stock_movements_quantity_valid"},
		{"receipt that takes away", p.item, "PARTS_RECEIPT", "5", "4", "PCS", "stock_movements_quantity_valid"},
		{"other unit", p.item, "PARTS_RECEIPT", "5", "6", "SET", "stock_movements_uom_matches_item"},
		{"serial SKU", p.serial, "PARTS_RECEIPT", "0", "1", "PCS", "stock_movements_item_quantity_tracked"},
		{"archived SKU", p.archived, "PARTS_RECEIPT", "0", "1", "PCS", "stock_movements_item_active"},
		// Fails stock_movements_quantity_valid too; PostgreSQL reports the first CHECK it evaluates.
		{"vehicle type on a part", p.item, "DISPLAY_ON", "1", "1", "PCS", "stock_movements_transition_valid|stock_movements_quantity_valid"},
	} {
		if _, constraint := sqlState(insert(c.item, c.typ, c.before, c.after, c.uom, "")); constraint == "" ||
			!strings.Contains("|"+c.constraint+"|", "|"+constraint+"|") {
			t.Errorf("%s: got %q, want %s", c.name, constraint, c.constraint)
		}
	}
	// A parts row cannot carry a vehicle unit.
	_, err := p.pool.Exec(ctx, `INSERT INTO stock_movements (item_type, stock_item_id, vehicle_unit_id, movement_type, quantity_before,
			quantity_after, from_location_id, to_location_id, uom, source_type, supplier_id, source_reference, actor_id, idempotency_key, request_hash)
		VALUES ('STOCK_ITEM', $1, gen_random_uuid(), 'PARTS_RECEIPT', 5, 6, $2, $2, 'PCS', 'SUPPLIER_DELIVERY', $3, 'DC', $4, 'direct-mixed', repeat('0', 64))`,
		p.item, p.locL1, p.supplier, p.ids["buyer"])
	if c, _ := sqlState(err); c != "23514" && c != "23503" {
		t.Errorf("mixed row: %v", err)
	}
	// A valid direct movement is still the ledger path: it applies to the balance and reconciles.
	if err := insert(p.item, "PARTS_ISSUE", "5", "4.25", "PCS", ""); err != nil {
		t.Fatal(err)
	}
	if b := p.balance(t, p.item, p.locL1); b != "4.250" {
		t.Fatalf("balance %s", b)
	}
	// Vehicle rules are unchanged (E-01): a vehicle row still needs its unit and its edge.
	_, err = p.pool.Exec(ctx, `INSERT INTO stock_movements (vehicle_unit_id, movement_type, quantity_before, quantity_after, from_status,
			to_status, source_type, actor_id, idempotency_key, request_hash)
		VALUES (gen_random_uuid(), 'PARTS_ISSUE', 1, 1, 'AVAILABLE', 'DISPLAY', 'STAFF_ACTION', $1, 'direct-vehicle', repeat('0', 64))`, p.ids["buyer"])
	if c, _ := sqlState(err); c != "23514" {
		t.Errorf("parts type on a vehicle row: %v", err)
	}
	p.reconcilesParts(t)
}

// Migration 000017: up from 16 keeps the vehicle ledger; down is refused while
// parts movements exist (history is not destroyed) and works without them.
func TestPartsMigrationRoundTrip(t *testing.T) {
	ctx := context.Background()
	url := dbtest.ScratchURL(t)
	m := dbtest.NewMigrator(t, url)
	if err := m.Migrate(16); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	f := setupOn(t, pool)
	unit := f.register(t, &f.locL1, "CH")[0] // a vehicle movement under 16
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	p := partsOn(t, f)
	if _, _, err := f.move(t, unit, stock.TypePDIPass, "after-up-pass", nil); err != nil {
		t.Fatalf("vehicle move after up: %v", err)
	}
	f.reconciles(t)
	expect(t, p.call(p.mover, p.item, "receipts", p.receipt("1"), "rcpt-0001"), http.StatusOK, nil)
	if err := m.Migrate(16); err == nil || !strings.Contains(err.Error(), "would destroy ledger history") {
		t.Fatalf("down over parts movements: %v", err)
	}
}

func TestPartsMigrationDownWithoutPartsMovements(t *testing.T) {
	ctx := context.Background()
	url := dbtest.ScratchURL(t)
	m := dbtest.NewMigrator(t, url)
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	if err := m.Migrate(16); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	var tbl *string
	if err := pool.QueryRow(ctx, `SELECT to_regclass('stock_item_balances')::text`).Scan(&tbl); err != nil || tbl != nil {
		t.Fatalf("down: balances table %v %v", tbl, err)
	}
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT to_regclass('stock_item_balances')::text`).Scan(&tbl); err != nil || tbl == nil {
		t.Fatalf("up again: %v %v", tbl, err)
	}
}
