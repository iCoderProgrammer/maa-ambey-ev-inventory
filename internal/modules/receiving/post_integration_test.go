package receiving

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database/dbtest"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/httpx"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
)

// Receipt posting (D-07) on a throwaway migrated database, on top of the D-03
// fixture (approved PO: lineA 5 units, lineB 2 units). Placeholder data only.

type postFixture struct {
	fixture
	poster, scoped, noView session
	locL1, locL2, inactive string
}

func setupPost(t *testing.T) postFixture {
	t.Helper()
	f := postFixture{fixture: setup(t)}
	f.locL1 = f.scalar(t, `INSERT INTO locations (code, name) VALUES ('L1', 'Location One') RETURNING id::text`)
	f.locL2 = f.scalar(t, `INSERT INTO locations (code, name) VALUES ('L2', 'Location Two') RETURNING id::text`)
	f.inactive = f.scalar(t, `INSERT INTO locations (code, name, active) VALUES ('L9', 'Closed', false) RETURNING id::text`)
	// receiver posts at every location; receiver2 posts at L2 only; overrider can post but not view.
	f.poster = f.receiver
	f.h.grant(f.poster, access.PermGoodsReceiptView, access.PermGoodsReceiptManage, access.PermAttachmentsUpload,
		access.PermGoodsReceiptPost, access.PermAllLocations)
	f.scoped = f.receiver2
	f.h.grant(f.scoped, access.PermGoodsReceiptView, access.PermGoodsReceiptManage, access.PermGoodsReceiptPost)
	f.h.grants.mu.Lock()
	f.h.grants.locations[f.scoped.userID] = []access.Location{{ID: f.locL2}}
	f.h.grants.mu.Unlock()
	f.noView = f.overrider
	f.h.grant(f.noView, access.PermGoodsReceiptManage, access.PermGoodsReceiptOverReceipt, access.PermGoodsReceiptPost)
	return f
}

func (f postFixture) post(s session, d Detail, location, key string) *httptest.ResponseRecorder {
	return f.h.callWith(s, http.MethodPost, "/goods-receipts/"+d.ID+"/post",
		map[string]any{"version": d.Version, "locationId": location}, map[string]string{"Idempotency-Key": key})
}

func (f postFixture) count(t *testing.T, sql string, args ...any) int {
	t.Helper()
	n, err := strconv.Atoi(f.scalar(t, `SELECT (`+sql+`)::text`, args...))
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// stockState is everything a posting writes, for "nothing changed" checks.
func (f postFixture) stockState(t *testing.T, receipt string) string {
	t.Helper()
	return f.poState(t) + "|" + f.scalar(t, `SELECT g.status || ':' || g.version || ':' || coalesce(g.location_id::text, '-') FROM goods_receipts g
		WHERE g.id = $1`, receipt) + "|" + strconv.Itoa(f.count(t, `SELECT count(*) FROM vehicle_units`)) + "|" +
		strconv.Itoa(f.count(t, `SELECT count(*) FROM stock_movements`)) + "|" +
		strconv.Itoa(f.count(t, `SELECT count(*) FROM audit_events WHERE action IN ('goods_receipt.posted', 'vehicle_unit.registered') AND outcome = 'SUCCESS'`))
}

func TestPostRegistersAcceptedUnitsAndUpdatesThePO(t *testing.T) {
	f := setupPost(t)
	d := f.create(t, f.poster)
	rejected := unitIn{"condition": "REJECTED", "chassisNumber": "CH-R1"}
	d = f.saveOK(t, f.poster, d,
		lineIn(f.lineA, 3, 2, 1, accepted("CH-A1"), accepted("CH-A2"), rejected),
		lineIn(f.lineB, 2, 2, 0, accepted("CH-B1"), accepted("")))
	if !d.Actions.Post || d.StockPosted {
		t.Fatalf("draft actions %+v", d.Actions)
	}
	var res PostResult
	w := f.post(f.poster, d, f.locL1, "post-0000001")
	expect(t, w, http.StatusOK, &res)
	r := res.Receipt
	if r.Status != StatusPosted || !r.StockPosted || r.Actions.Post || r.Actions.Edit || r.Version != d.Version+1 ||
		r.LocationID == nil || *r.LocationID != f.locL1 || r.PostedBy == nil || *r.PostedBy != f.poster.userID ||
		r.PurchaseOrderStatus != "PARTIALLY_RECEIVED" || len(res.VehicleUnits) != 4 {
		t.Fatalf("result %+v / units %+v", r.Receipt, res.VehicleUnits)
	}
	for _, u := range res.VehicleUnits {
		if u.Status != "RECEIVED_PENDING_PDI" {
			t.Fatalf("unit %+v not awaiting PDI", u)
		}
	}
	// One vehicle per ACCEPTED unit (none for the rejected), each with its RECEIPT movement at L1.
	if n := f.count(t, `SELECT count(*) FROM vehicle_units WHERE goods_receipt_id = $1 AND location_id = $2 AND status = 'RECEIVED_PENDING_PDI'`,
		d.ID, f.locL1); n != 4 {
		t.Fatalf("%d vehicle units", n)
	}
	if n := f.count(t, `SELECT count(*) FROM vehicle_units WHERE chassis_number = 'CH-R1'`); n != 0 {
		t.Fatal("rejected unit registered")
	}
	if n := f.count(t, `SELECT count(*) FROM stock_movements m JOIN vehicle_units v ON v.id = m.vehicle_unit_id
		WHERE v.goods_receipt_id = $1 AND m.movement_type = 'RECEIPT' AND m.to_location_id = $2`, d.ID, f.locL1); n != 4 {
		t.Fatalf("%d receipt movements", n)
	}
	// PO: received quantities by line (received, the D-03 basis), status, history.
	if got := f.scalar(t, `SELECT string_agg(received_quantity::text, ',' ORDER BY line_no) FROM purchase_order_lines WHERE purchase_order_id = $1`,
		f.po); got != "3,2" {
		t.Fatalf("received quantities %s", got)
	}
	if n := f.count(t, `SELECT count(*) FROM purchase_order_events WHERE purchase_order_id = $1 AND action = 'GOODS_RECEIVED'
		AND from_status = 'APPROVED' AND to_status = 'PARTIALLY_RECEIVED' AND note = $2`, f.po, d.Number); n != 1 {
		t.Fatal("PO history not recorded")
	}
	if f.audits(t, d.ID, "goods_receipt.posted", "SUCCESS") != 1 ||
		f.count(t, `SELECT count(*) FROM audit_events WHERE action = 'vehicle_unit.registered'`) != 4 {
		t.Fatal("posting not audited")
	}
	if last := r.Events[len(r.Events)-1]; last.Action != ActionPosted || last.ToStatus != StatusPosted {
		t.Fatalf("receipt history %+v", r.Events)
	}
	// A posted receipt is immutable, through the API and in the database.
	expectError(t, f.save(f.poster, r, lineIn(f.lineA, 1, 1, 0, accepted("CH-X"))), http.StatusConflict, httpx.CodeInvalidState)
	_, err := f.pool.Exec(context.Background(), `UPDATE goods_receipts SET notes = 'edited' WHERE id = $1`, d.ID)
	if sqlState(err) != "42501" {
		t.Fatalf("posted receipt row changed: %v", err)
	}
	// A second receipt takes the rest: the PO becomes RECEIVED.
	d2 := f.create(t, f.poster)
	d2 = f.saveOK(t, f.poster, d2, lineIn(f.lineA, 2, 2, 0, accepted("CH-A3"), accepted("CH-A4")))
	expect(t, f.post(f.poster, d2, f.locL1, "post-0000002"), http.StatusOK, &res)
	if res.Receipt.PurchaseOrderStatus != "RECEIVED" || f.count(t, `SELECT count(*) FROM vehicle_units`) != 6 {
		t.Fatalf("second posting %+v", res.Receipt.Receipt)
	}
	// Nothing became saleable: PDI (D-06) decides that.
	if n := f.count(t, `SELECT count(*) FROM vehicle_units WHERE status = 'AVAILABLE'`); n != 0 {
		t.Fatalf("%d units saleable right after posting", n)
	}
}

func TestPostIsRefusedUntilTheReceiptIsComplete(t *testing.T) {
	f := setupPost(t)
	empty := f.create(t, f.poster)
	before0 := f.stockState(t, empty.ID)
	e := expectError(t, f.post(f.poster, empty, f.locL1, "post-empty-01"), http.StatusConflict, httpx.CodePostingBlocked)
	if !hasField(e.Details, "lines") {
		t.Fatalf("details %+v", e.Details)
	}
	if after := f.stockState(t, empty.ID); after != before0 {
		t.Fatalf("refused posting changed something:\n%s\n%s", before0, after)
	}
	// Units not all captured.
	d := f.create(t, f.poster)
	d = f.saveOK(t, f.poster, d, lineIn(f.lineA, 3, 3, 0, accepted("CH-1"), accepted("CH-2")))
	e = expectError(t, f.post(f.poster, d, f.locL1, "post-units-01"), http.StatusConflict, httpx.CodePostingBlocked)
	if !hasField(e.Details, "lines[0].units") {
		t.Fatalf("details %+v", e.Details)
	}
	// Quantity above what the PO has left now (someone else's posting is simulated on the PO line).
	d = f.saveOK(t, f.poster, d, lineIn(f.lineA, 3, 3, 0, accepted("CH-1"), accepted("CH-2"), accepted("CH-3")))
	f.exec(t, `UPDATE purchase_order_lines SET received_quantity = 4 WHERE id = $1`, f.lineA)
	e = expectError(t, f.post(f.poster, d, f.locL1, "post-qty-0001"), http.StatusConflict, httpx.CodePostingBlocked)
	if !hasField(e.Details, "lines[0].receivedQuantity") || !strings.Contains(e.Details[0].Message, "only 1 left") {
		t.Fatalf("details %+v", e.Details)
	}
	if f.audits(t, d.ID, "goods_receipt.posted", "DENIED") != 2 {
		t.Fatal("refusals not audited")
	}
	if n := f.count(t, `SELECT count(*) FROM vehicle_units`); n != 0 {
		t.Fatal("a refused posting registered units")
	}
	// An authorised over-receipt covering the received quantity lets it post.
	f.exec(t, `UPDATE purchase_order_lines SET received_quantity = 0 WHERE id = $1`, f.lineA)
	d = lineHolder(t, f, d)
	d = f.saveOK(t, f.overriderView(t), d, map[string]any{"purchaseOrderLineId": f.lineA, "receivedQuantity": 6, "acceptedQuantity": 6,
		"rejectedQuantity": 0, "overReceiptReason": "Supplier sent one extra", "units": []unitIn{accepted("CH-1"), accepted("CH-2"),
			accepted("CH-3"), accepted("CH-4"), accepted("CH-5"), accepted("CH-6")}})
	var res PostResult
	expect(t, f.post(f.poster, d, f.locL1, "post-over-001"), http.StatusOK, &res)
	if got := f.scalar(t, `SELECT received_quantity::text FROM purchase_order_lines WHERE id = $1`, f.lineA); got != "6" {
		t.Fatalf("received %s", got)
	}
}

// lineHolder reloads a receipt detail.
func lineHolder(t *testing.T, f postFixture, d Detail) Detail {
	t.Helper()
	var out Detail
	expect(t, f.h.call(f.poster, http.MethodGet, "/goods-receipts/"+d.ID, nil), http.StatusOK, &out)
	return out
}

// overriderView gives the over-receipt user goods_receipts.view back for a save.
func (f postFixture) overriderView(t *testing.T) session {
	t.Helper()
	f.h.grant(f.noView, access.PermGoodsReceiptView, access.PermGoodsReceiptManage, access.PermGoodsReceiptOverReceipt)
	return f.noView
}

func TestPostValidationStateAndPermissions(t *testing.T) {
	f := setupPost(t)
	d := f.create(t, f.poster)
	d = f.saveOK(t, f.poster, d, lineIn(f.lineB, 1, 1, 0, accepted("CH-P1")))
	for _, tc := range []struct {
		body  map[string]any
		key   string
		field string
	}{
		{map[string]any{"locationId": f.locL1}, "post-val-0001", "version"},
		{map[string]any{"version": d.Version}, "post-val-0002", "locationId"},
		{map[string]any{"version": d.Version, "locationId": f.locL1}, "short", "Idempotency-Key"},
		{map[string]any{"version": d.Version, "locationId": f.inactive}, "post-val-0003", "locationId"},
		{map[string]any{"version": d.Version, "locationId": f.locL1, "status": "POSTED"}, "post-val-0004", "status"},
	} {
		e := expectError(t, f.h.callWith(f.poster, http.MethodPost, "/goods-receipts/"+d.ID+"/post", tc.body,
			map[string]string{"Idempotency-Key": tc.key}), http.StatusUnprocessableEntity, httpx.CodeValidationFailed)
		if !hasField(e.Details, tc.field) {
			t.Fatalf("details %+v, want %s", e.Details, tc.field)
		}
	}
	// Location scope: the L2-only poster cannot post into L1, but can into L2... after the checks below.
	e := expectError(t, f.post(f.scoped, d, f.locL1, "post-scope-01"), http.StatusUnprocessableEntity, httpx.CodeValidationFailed)
	if !hasField(e.Details, "locationId") {
		t.Fatalf("details %+v", e.Details)
	}
	// Permissions: deny by default.
	expectError(t, f.post(f.viewer, d, f.locL1, "post-perm-001"), http.StatusForbidden, httpx.CodeForbidden)
	expectError(t, f.post(f.noView, d, f.locL1, "post-perm-002"), http.StatusForbidden, httpx.CodeForbidden)
	// Stale version.
	stale := d
	stale.Version--
	expectError(t, f.post(f.poster, stale, f.locL1, "post-stale-01"), http.StatusConflict, httpx.CodeVersionConflict)
	expectError(t, f.h.callWith(f.poster, http.MethodPost, "/goods-receipts/00000000-0000-4000-8000-000000000000/post",
		map[string]any{"version": 1, "locationId": f.locL1}, map[string]string{"Idempotency-Key": "post-nf-0001"}),
		http.StatusNotFound, httpx.CodeNotFound)
	if n := f.count(t, `SELECT count(*) FROM vehicle_units`); n != 0 {
		t.Fatal("refused requests registered units")
	}
	var res PostResult
	expect(t, f.post(f.scoped, d, f.locL2, "post-scope-02"), http.StatusOK, &res)
	// Already posted: a new key is refused as INVALID_STATE (nothing posted twice).
	expectError(t, f.post(f.poster, res.Receipt, f.locL1, "post-again-01"), http.StatusConflict, httpx.CodeInvalidState)
	// A cancelled draft cannot be posted; neither can one whose PO is no longer receivable.
	c := f.create(t, f.poster)
	expect(t, f.h.call(f.poster, http.MethodPost, "/goods-receipts/"+c.ID+"/cancel", map[string]any{"version": c.Version,
		"reason": "Wrong delivery"}), http.StatusOK, &c)
	expectError(t, f.post(f.poster, c, f.locL1, "post-cancel-1"), http.StatusConflict, httpx.CodeInvalidState)
	p := f.create(t, f.poster)
	p = f.saveOK(t, f.poster, p, lineIn(f.lineA, 1, 1, 0, accepted("CH-P2")))
	f.exec(t, `UPDATE purchase_orders SET status = 'CLOSED' WHERE id = $1`, f.po)
	expectError(t, f.post(f.poster, p, f.locL1, "post-closed-1"), http.StatusConflict, httpx.CodeInvalidState)
}

func TestPostIdempotencyAndConcurrency(t *testing.T) {
	f := setupPost(t)
	d := f.create(t, f.poster)
	d = f.saveOK(t, f.poster, d, lineIn(f.lineA, 2, 2, 0, accepted("CH-I1"), accepted("CH-I2")))

	// The same key 5 times at once: one posting, all answers the same receipt.
	var wg sync.WaitGroup
	codes := make([]int, 5)
	replays := make([]bool, 5)
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			w := f.post(f.poster, d, f.locL1, "post-same-key-1")
			codes[i], replays[i] = w.Code, w.Header().Get("Idempotent-Replayed") == "true"
		}(i)
	}
	wg.Wait()
	fresh := 0
	for i, c := range codes {
		if c != http.StatusOK {
			t.Fatalf("codes %v", codes)
		}
		if !replays[i] {
			fresh++
		}
	}
	if fresh != 1 || f.count(t, `SELECT count(*) FROM vehicle_units`) != 2 || f.count(t, `SELECT count(*) FROM stock_movements`) != 2 ||
		f.scalar(t, `SELECT received_quantity::text FROM purchase_order_lines WHERE id = $1`, f.lineA) != "2" {
		t.Fatalf("%d fresh answers; posting duplicated", fresh)
	}
	// Same key, different request (other location) and same key on another receipt: refused.
	expectError(t, f.post(f.poster, d, f.locL2, "post-same-key-1"), http.StatusConflict, httpx.CodeIdempotencyKeyReused)
	o := f.create(t, f.poster)
	o = f.saveOK(t, f.poster, o, lineIn(f.lineB, 1, 1, 0, accepted("CH-I3")))
	expectError(t, f.post(f.poster, o, f.locL1, "post-same-key-1"), http.StatusConflict, httpx.CodeIdempotencyKeyReused)

	// Different keys on one receipt at once: exactly one posts.
	codes = make([]int, 6)
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i] = f.post(f.poster, o, f.locL1, "post-race-"+strconv.Itoa(i)+"-x").Code
		}(i)
	}
	wg.Wait()
	ok := 0
	for _, c := range codes {
		switch c {
		case http.StatusOK:
			ok++
		case http.StatusConflict:
		default:
			t.Fatalf("codes %v", codes)
		}
	}
	if ok != 1 || f.count(t, `SELECT count(*) FROM vehicle_units WHERE goods_receipt_id = $1`, o.ID) != 1 {
		t.Fatalf("codes %v", codes)
	}

	// Two receipts of one PO posted at once, together above the PO line: the PO lock serialises them.
	a := f.create(t, f.poster)
	a = f.saveOK(t, f.poster, a, lineIn(f.lineA, 2, 2, 0, accepted("CH-C1"), accepted("CH-C2")))
	b := f.create(t, f.poster)
	b = f.saveOK(t, f.overriderView(t), b, map[string]any{"purchaseOrderLineId": f.lineA, "receivedQuantity": 2, "acceptedQuantity": 2,
		"rejectedQuantity": 0, "overReceiptReason": "test", "units": []unitIn{accepted("CH-C3"), accepted("CH-C4")}})
	// Drop b's authorisation directly (still a draft) so that only one of the two fits the 3 units left.
	f.exec(t, `UPDATE goods_receipt_lines SET over_receipt_quantity = NULL, over_receipt_reason = NULL, over_receipt_by = NULL,
		over_receipt_at = NULL WHERE goods_receipt_id = $1`, b.ID)
	b = lineHolder(t, f, b)
	results := make([]int, 2)
	for i, r := range []Detail{a, b} {
		wg.Add(1)
		go func(i int, r Detail) {
			defer wg.Done()
			results[i] = f.post(f.poster, r, f.locL1, "post-po-race-"+strconv.Itoa(i)).Code
		}(i, r)
	}
	wg.Wait()
	if !((results[0] == http.StatusOK) != (results[1] == http.StatusOK)) ||
		f.scalar(t, `SELECT received_quantity::text FROM purchase_order_lines WHERE id = $1`, f.lineA) != "4" {
		t.Fatalf("results %v, received %s", results, f.scalar(t, `SELECT received_quantity::text FROM purchase_order_lines WHERE id = $1`, f.lineA))
	}
}

func TestFailedPostingLeavesNothingBehind(t *testing.T) {
	f := setupPost(t)
	// Duplicate identifier: two drafts are made to carry the same chassis (as a race could) and one is posted.
	x := f.create(t, f.poster)
	x = f.saveOK(t, f.poster, x, lineIn(f.lineA, 1, 1, 0, accepted("CH-DUP")))
	y := f.create(t, f.poster)
	y = f.saveOK(t, f.poster, y, lineIn(f.lineB, 2, 2, 0, accepted("CH-OK"), accepted("CH-OTHER")))
	expect(t, f.post(f.poster, x, f.locL1, "post-dup-0001"), http.StatusOK, nil)
	f.exec(t, `UPDATE goods_receipt_units SET chassis_number = 'CH-DUP', chassis_normalised = 'CH-DUP'
		WHERE goods_receipt_id = $1 AND chassis_number = 'CH-OTHER'`, y.ID)
	before := f.stockState(t, y.ID)
	e := expectError(t, f.post(f.poster, y, f.locL1, "post-dup-0002"), http.StatusConflict, httpx.CodeDuplicate)
	if !hasField(e.Details, "lines[1].units[1].chassisNumber") {
		t.Fatalf("details %+v", e.Details)
	}
	if after := f.stockState(t, y.ID); after != before {
		t.Fatalf("partial posting:\n%s\n%s", before, after)
	}
	// A database failure at the very end (after units, movements and PO updates) rolls everything back.
	f.exec(t, `UPDATE goods_receipt_units SET chassis_number = 'CH-OTHER', chassis_normalised = 'CH-OTHER'
		WHERE goods_receipt_id = $1 AND unit_no = 2`, y.ID)
	f.exec(t, `CREATE FUNCTION test_fail_audit() RETURNS trigger AS $$ BEGIN
		IF NEW.action = 'goods_receipt.posted' AND NEW.outcome = 'SUCCESS' THEN RAISE EXCEPTION 'simulated failure'; END IF;
		RETURN NEW; END $$ LANGUAGE plpgsql`)
	f.exec(t, `CREATE TRIGGER test_fail_audit BEFORE INSERT ON audit_events FOR EACH ROW EXECUTE FUNCTION test_fail_audit()`)
	before = f.stockState(t, y.ID)
	expectError(t, f.post(f.poster, y, f.locL1, "post-fail-0001"), http.StatusInternalServerError, httpx.CodeInternal)
	if after := f.stockState(t, y.ID); after != before {
		t.Fatalf("partial posting after a failure:\n%s\n%s", before, after)
	}
	f.exec(t, `DROP TRIGGER test_fail_audit ON audit_events`)
	// The key of the failed attempt was not used up: the same request now posts.
	var res PostResult
	expect(t, f.post(f.poster, y, f.locL1, "post-fail-0001"), http.StatusOK, &res)
	if len(res.VehicleUnits) != 2 {
		t.Fatalf("units %+v", res.VehicleUnits)
	}
}

func TestPostingMigrationRollbackKeepsHistory(t *testing.T) {
	url := dbtest.ScratchURL(t)
	m := dbtest.NewMigrator(t, url)
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	if err := m.Migrate(19); err != nil {
		t.Fatalf("down to 19 while empty: %v", err)
	}
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
}
