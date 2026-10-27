package receiving

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database/dbtest"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/attachments"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/seeds"
)

// End-to-end goods receipt tests on a throwaway migrated database. Names,
// quantities and identifiers are placeholders ("Brand A", "CH-0001"): no real
// supplier, purchase order, vehicle or serial is invented. Approved purchase
// orders are inserted directly, because a real one needs an activated brand.

type fixture struct {
	pool      *pgxpool.Pool
	h         *harness
	receiver  session // manages receipts, uploads documents
	receiver2 session // manages receipts
	overrider session // manages receipts and authorises over-receipt
	viewer    session // views receipts and attachments
	ids       map[string]string

	po      string // APPROVED, lines lineA (5 units) and lineB (2 units)
	lineA   string
	lineB   string
	draftPO string // still DRAFT
}

func setup(t *testing.T) fixture {
	t.Helper()
	ctx := context.Background()
	pool := dbtest.MigratedPool(t)
	if _, err := database.ApplySeeds(ctx, pool, seeds.FS); err != nil {
		t.Fatal(err)
	}
	store := auth.NewPGStore(pool)
	hash, err := auth.HashPassword(testPassword)
	if err != nil {
		t.Fatal(err)
	}
	f := fixture{pool: pool, ids: map[string]string{}}
	for _, u := range []string{"receiver", "receiver2", "overrider", "viewer", "buyer", "approver"} {
		created, err := store.CreateUser(ctx, audit.CLIActor("test"), u, u, nil, hash)
		if err != nil {
			t.Fatal(err)
		}
		f.ids[u] = created.ID
	}
	f.h = newHarness(t, store, NewPGStore(pool))
	f.receiver = f.h.login("receiver", f.ids["receiver"])
	f.receiver2 = f.h.login("receiver2", f.ids["receiver2"])
	f.overrider = f.h.login("overrider", f.ids["overrider"])
	f.viewer = f.h.login("viewer", f.ids["viewer"])
	f.h.grant(f.receiver, access.PermGoodsReceiptView, access.PermGoodsReceiptManage, access.PermAttachmentsUpload)
	f.h.grant(f.receiver2, access.PermGoodsReceiptView, access.PermGoodsReceiptManage)
	f.h.grant(f.overrider, access.PermGoodsReceiptView, access.PermGoodsReceiptManage, access.PermGoodsReceiptOverReceipt)
	f.h.grant(f.viewer, access.PermGoodsReceiptView, access.PermAttachmentsView)

	brand := f.scalar(t, `INSERT INTO brands (name, slug, lifecycle_status, authorisation_status, manufacturer_legal_name,
		authorisation_reference, verified_by, verified_at)
		VALUES ('Brand A', 'brand-a', 'ACTIVE', 'AUTHORISED', 'Example Manufacturer', 'TEST-AUTH', $1, now()) RETURNING id::text`, f.ids["approver"])
	model := f.scalar(t, `INSERT INTO models (brand_id, name, internal_code, lifecycle_status, confidence, source_url, source_date, verified_by, verified_at)
		VALUES ($1, 'Model 1', 'M1', 'ACTIVE', 'SUPPLIER_CONFIRMED', 'https://example.com/m', current_date, $2, now()) RETURNING id::text`,
		brand, f.ids["approver"])
	variant := func(code string) string {
		return f.scalar(t, `INSERT INTO variants (brand_id, model_id, name, variant_code, lifecycle_status, confidence, source_url, source_date, verified_by, verified_at)
			VALUES ($1, $2, 'Variant '||$3, $3, 'ACTIVE', 'SUPPLIER_CONFIRMED', 'https://example.com/v', current_date, $4, now()) RETURNING id::text`,
			brand, model, code, f.ids["approver"])
	}
	vA, vB := variant("STD"), variant("PRO")
	supplier := f.scalar(t, `INSERT INTO suppliers (legal_name, created_by) VALUES ('Example Supplier One', $1) RETURNING id::text`, f.ids["buyer"])
	newPO := func() string {
		return f.scalar(t, `INSERT INTO purchase_orders (supplier_id, created_by, expected_date) VALUES ($1, $2, current_date + 7) RETURNING id::text`,
			supplier, f.ids["buyer"])
	}
	addLine := func(po string, no int, v string, qty int) string {
		return f.scalar(t, `INSERT INTO purchase_order_lines (purchase_order_id, line_no, brand_id, model_id, variant_id, quantity, unit_cost)
			VALUES ($1, $2, $3, $4, $5, $6, 1000) RETURNING id::text`, po, no, brand, model, v, qty)
	}
	f.po = newPO()
	f.lineA = addLine(f.po, 1, vA, 5)
	f.lineB = addLine(f.po, 2, vB, 2)
	f.exec(t, `UPDATE purchase_orders SET status = 'APPROVED', submitted_by = $2, submitted_at = now(), approval_basis = 'APPROVER',
		approved_by = $3, approved_at = now() WHERE id = $1`, f.po, f.ids["buyer"], f.ids["approver"])
	f.draftPO = newPO()
	addLine(f.draftPO, 1, vA, 1)
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

const pastDate = "2026-01-15"

func (f fixture) create(t *testing.T, s session) Detail {
	t.Helper()
	var d Detail
	expect(t, f.h.call(s, http.MethodPost, "/goods-receipts", map[string]any{"purchaseOrderId": f.po, "receivedDate": pastDate,
		"supplierReference": "DC-1"}), http.StatusCreated, &d)
	return d
}

type unitIn map[string]any

func accepted(chassis string) unitIn {
	u := unitIn{"condition": "ACCEPTED"}
	if chassis != "" {
		u["chassisNumber"] = chassis
	}
	return u
}

func lineIn(poLine string, received, acc, rej int, units ...unitIn) map[string]any {
	us := []unitIn{}
	us = append(us, units...)
	return map[string]any{"purchaseOrderLineId": poLine, "receivedQuantity": received, "acceptedQuantity": acc,
		"rejectedQuantity": rej, "units": us}
}

func saveBody(version int, lines ...map[string]any) map[string]any {
	ls := []map[string]any{}
	ls = append(ls, lines...)
	return map[string]any{"version": version, "receivedDate": pastDate, "supplierReference": "DC-1", "lines": ls}
}

func (f fixture) save(s session, d Detail, lines ...map[string]any) *httptest.ResponseRecorder {
	return f.h.call(s, http.MethodPut, "/goods-receipts/"+d.ID, saveBody(d.Version, lines...))
}

func (f fixture) saveOK(t *testing.T, s session, d Detail, lines ...map[string]any) Detail {
	t.Helper()
	var out Detail
	expect(t, f.save(s, d, lines...), http.StatusOK, &out)
	return out
}

func lineOf(t *testing.T, d Detail, poLine string) Line {
	t.Helper()
	for _, l := range d.Lines {
		if l.PurchaseOrderLineID == poLine {
			return l
		}
	}
	t.Fatalf("PO line %s not in receipt", poLine)
	return Line{}
}

func (f fixture) audits(t *testing.T, entityID, action, outcome string) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE entity_id = $1 AND action = $2 AND outcome = $3`,
		entityID, action, outcome).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// poState is everything about the PO that stock posting would change.
func (f fixture) poState(t *testing.T) string {
	t.Helper()
	return f.scalar(t, `SELECT po.status || ':' || po.version || ':' ||
			(SELECT string_agg(l.id::text || '=' || l.received_quantity || '@' || l.updated_at::text, ',' ORDER BY l.line_no)
			 FROM purchase_order_lines l WHERE l.purchase_order_id = po.id) || ':' ||
			(SELECT count(*) FROM purchase_order_events e WHERE e.purchase_order_id = po.id)
		FROM purchase_orders po WHERE po.id = $1`, f.po)
}

func sqlState(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

func TestPartialDraftReceiptChangesNoStock(t *testing.T) {
	f := setup(t)
	before := f.poState(t)
	d := f.create(t, f.receiver)
	if !regexp.MustCompile(`^GRN-\d{6}$`).MatchString(d.Number) || d.Status != StatusDraft || d.PurchaseOrderNumber == "" ||
		!d.PurchaseOrderReceivable || d.StockPosted || len(d.Lines) != 2 || len(d.Events) != 1 {
		t.Fatalf("created %+v", d)
	}
	if a := lineOf(t, d, f.lineA); a.ID != nil || a.OrderedQuantity != 5 || a.AvailableQuantity != 5 || a.ReceivedQuantity != 0 {
		t.Fatalf("empty line %+v", a)
	}
	if !d.Actions.Edit || !d.Actions.Cancel || !d.Actions.AddDocument || d.Actions.AuthoriseOverReceipt || d.Actions.Post {
		t.Fatalf("actions %+v", d.Actions)
	}
	// Partial delivery: 3 of 5 on line A (one rejected), nothing of line B.
	d = f.saveOK(t, f.receiver, d, lineIn(f.lineA, 3, 2, 1, accepted("ch-0001"), accepted("CH-0002"), unitIn{"condition": "REJECTED",
		"chassisNumber": "CH-0003", "motorSerial": "MOT-1", "note": "Scratched panel"}))
	a := lineOf(t, d, f.lineA)
	if a.ID == nil || a.ReceivedQuantity != 3 || a.AcceptedQuantity != 2 || a.RejectedQuantity != 1 || len(a.Units) != 3 ||
		*a.Units[0].ChassisNormalised != "CH-0001" || *a.Units[0].ChassisNumber != "ch-0001" {
		t.Fatalf("line A %+v", a)
	}
	if !slicesHave(a.Warnings, WarnPartialDelivery, WarnRejectionNotExplained) || slicesHave(a.Warnings, WarnIdentifiersIncomplete) {
		t.Fatalf("line A warnings %v", a.Warnings)
	}
	if d.ReceivedQuantity != 3 || d.AcceptedQuantity != 2 || d.RejectedQuantity != 1 || d.UnitCount != 3 || d.LineCount != 1 || d.Version != 2 {
		t.Fatalf("totals %+v", d.Receipt)
	}
	if lineOf(t, d, f.lineB).ID != nil {
		t.Fatal("line B should not be on the receipt")
	}
	// Identifiers are optional in a draft: fewer units than received is a warning, not an error.
	d = f.saveOK(t, f.receiver, d, lineIn(f.lineA, 3, 3, 0, accepted("CH-0001")), lineIn(f.lineB, 1, 1, 0))
	if w := lineOf(t, d, f.lineA).Warnings; !slicesHave(w, WarnIdentifiersIncomplete) {
		t.Fatalf("incomplete identifiers %v", w)
	}
	// A draft changed nothing stock-related: PO status, version, received quantities, history.
	if after := f.poState(t); after != before {
		t.Fatalf("PO changed by a draft:\n%s\n%s", before, after)
	}
	// Saving drafts registered no vehicle unit (D-04).
	if n := f.scalar(t, `SELECT (SELECT count(*) FROM vehicle_units) + (SELECT count(*) FROM vehicle_unit_events)`); n != "0" {
		t.Fatalf("a draft registered vehicle units (%s)", n)
	}
	// ...and recorded no stock movement (E-01).
	if n := f.scalar(t, `SELECT count(*)::text FROM stock_movements`); n != "0" {
		t.Fatalf("a draft recorded stock movements (%s)", n)
	}
	// Server-side values only.
	expectError(t, f.h.call(f.receiver, http.MethodPut, "/goods-receipts/"+d.ID, map[string]any{"version": d.Version, "receivedDate": pastDate,
		"status": "POSTED"}), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if f.audits(t, d.ID, "goods_receipt.saved", "SUCCESS") != 2 || f.audits(t, d.ID, "goods_receipt.created", "SUCCESS") != 1 {
		t.Fatal("saves not audited")
	}
	// Viewers read; list and filters.
	var page ReceiptPage
	expect(t, f.h.call(f.viewer, http.MethodGet, "/goods-receipts?purchaseOrderId="+f.po, nil), http.StatusOK, &page)
	if page.Total != 1 || page.Items[0].ReceivedQuantity != 4 {
		t.Fatalf("list %+v", page)
	}
	expect(t, f.h.call(f.viewer, http.MethodGet, "/goods-receipts?q=DC-1&status=DRAFT", nil), http.StatusOK, &page)
	if page.Total != 1 {
		t.Fatalf("search %+v", page)
	}
	var v Detail
	expect(t, f.h.call(f.viewer, http.MethodGet, "/goods-receipts/"+d.ID, nil), http.StatusOK, &v)
	if v.Actions.Edit || v.Actions.Cancel {
		t.Fatalf("viewer actions %+v", v.Actions)
	}
	expectError(t, f.h.call(f.viewer, http.MethodPut, "/goods-receipts/"+d.ID, saveBody(v.Version)), http.StatusForbidden, "FORBIDDEN")
}

func slicesHave(list []string, want ...string) bool {
	for _, w := range want {
		found := false
		for _, l := range list {
			if l == w {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func TestRemainingCountsPostedAndOtherDraftsAndOverReceiptNeedsOverride(t *testing.T) {
	f := setup(t)
	first := f.saveOK(t, f.receiver, f.create(t, f.receiver), lineIn(f.lineA, 4, 4, 0))
	second := f.create(t, f.receiver2)
	if a := lineOf(t, second, f.lineA); a.OtherDraftsQuantity != 4 || a.AvailableQuantity != 1 {
		t.Fatalf("other drafts %+v", a)
	}
	// 2 of the 1 left: refused, nothing saved.
	e := expectError(t, f.save(f.receiver2, second, lineIn(f.lineA, 2, 2, 0)), http.StatusConflict, "OVER_RECEIPT_REQUIRES_OVERRIDE")
	if !hasField(e.Details, "lines[0].receivedQuantity") {
		t.Fatalf("details %+v", e.Details)
	}
	// A reason from someone without the permission: forbidden and audited.
	body := lineIn(f.lineA, 2, 2, 0)
	body["overReceiptReason"] = "Supplier sent one extra unit"
	expectError(t, f.save(f.receiver2, second, body), http.StatusForbidden, "OVER_RECEIPT_NOT_AUTHORISED")
	if f.audits(t, second.ID, "goods_receipt.over_receipt_authorised", "DENIED") != 1 {
		t.Fatal("refused override not audited")
	}
	var reread Detail
	expect(t, f.h.call(f.receiver2, http.MethodGet, "/goods-receipts/"+second.ID, nil), http.StatusOK, &reread)
	if reread.Version != second.Version || reread.LineCount != 0 {
		t.Fatalf("refused save changed the draft %+v", reread.Receipt)
	}
	// Authorised with a reason by a holder of goods_receipts.over_receipt.
	d := f.saveOK(t, f.overrider, second, body)
	a := lineOf(t, d, f.lineA)
	if a.OverReceipt == nil || a.OverReceipt.Quantity != 2 || a.OverReceipt.AuthorisedBy != f.ids["overrider"] ||
		a.OverReceipt.AuthorisedByName != "overrider" || a.ExcessQuantity != 1 || !slicesHave(a.Warnings, WarnOverReceipt) {
		t.Fatalf("authorised line %+v", a)
	}
	if f.audits(t, d.ID, "goods_receipt.over_receipt_authorised", "SUCCESS") != 1 {
		t.Fatal("override not audited")
	}
	// Someone else editing the draft keeps the authorisation while it covers the quantity ...
	d = f.saveOK(t, f.receiver2, d, lineIn(f.lineA, 2, 1, 1, accepted("CH-9")))
	if lineOf(t, d, f.lineA).OverReceipt == nil {
		t.Fatal("authorisation dropped by an unrelated edit")
	}
	// ... but not beyond it.
	expectError(t, f.save(f.receiver2, d, lineIn(f.lineA, 3, 3, 0)), http.StatusConflict, "OVER_RECEIPT_REQUIRES_OVERRIDE")
	// Cancelling the first draft frees its 4 units: no override needed, and it is dropped.
	var cancelled Detail
	expect(t, f.h.call(f.receiver, http.MethodPost, "/goods-receipts/"+first.ID+"/cancel",
		map[string]any{"version": first.Version, "reason": "Entered twice"}), http.StatusOK, &cancelled)
	if cancelled.Status != StatusCancelled || *cancelled.CancelReason != "Entered twice" || cancelled.LineCount != 1 || cancelled.Actions.Edit {
		t.Fatalf("cancelled %+v", cancelled.Receipt)
	}
	d = f.saveOK(t, f.receiver2, d, lineIn(f.lineA, 2, 2, 0))
	if a := lineOf(t, d, f.lineA); a.OverReceipt != nil || a.AvailableQuantity != 5 || a.ExcessQuantity != 0 {
		t.Fatalf("after cancel %+v", a)
	}
	// Posted quantities (set by D-07) count too: 4 posted leave 1.
	f.exec(t, `UPDATE purchase_order_lines SET received_quantity = 4 WHERE id = $1`, f.lineA)
	expectError(t, f.save(f.receiver2, d, lineIn(f.lineA, 2, 2, 0)), http.StatusConflict, "OVER_RECEIPT_REQUIRES_OVERRIDE")
	// Receivable POs show what is left: line A none (5 - 4 posted - 2 drafted), line B its 2.
	var page ReceivablePage
	expect(t, f.h.call(f.receiver, http.MethodGet, "/receivable-purchase-orders", nil), http.StatusOK, &page)
	if page.Total != 1 || page.Items[0].ID != f.po || page.Items[0].PostedQuantity != 4 || page.Items[0].DraftQuantity != 2 ||
		page.Items[0].AvailableQuantity != 2 || page.Items[0].OpenDraftCount != 1 {
		t.Fatalf("receivable %+v", page)
	}
}

func TestSerialCaptureAndDuplicateIdentifiers(t *testing.T) {
	f := setup(t)
	d1 := f.saveOK(t, f.receiver, f.create(t, f.receiver), lineIn(f.lineA, 2, 2, 0, accepted("CH-0001"),
		unitIn{"condition": "ACCEPTED", "chassisNumber": "CH-0002", "motorSerial": "M-1", "batterySerial": "B-1", "chargerSerial": "C-1"}))
	u := lineOf(t, d1, f.lineA).Units[1]
	if u.UnitNo != 2 || *u.MotorSerial != "M-1" || *u.BatterySerial != "B-1" || *u.ChargerSerial != "C-1" || len(u.Warnings) != 0 {
		t.Fatalf("unit %+v", u)
	}
	// The same chassis (any case, surrounding spaces) on another open receipt: refused per unit.
	d2 := f.create(t, f.receiver2)
	e := expectError(t, f.save(f.receiver2, d2, lineIn(f.lineB, 2, 2, 0, accepted("ch-0003"), accepted("  ch-0002 "))),
		http.StatusConflict, "DUPLICATE")
	if len(e.Details) != 1 || e.Details[0].Field != "lines[0].units[1].chassisNumber" ||
		!regexp.MustCompile(`draft goods receipt GRN-\d{6}`).MatchString(e.Details[0].Message) {
		t.Fatalf("details %+v", e.Details)
	}
	// Same chassis twice on one receipt: refused before the database.
	expectError(t, f.save(f.receiver2, d2, lineIn(f.lineB, 2, 2, 0, accepted("CH-7"), accepted("ch-7"))),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	// Motor/battery serial repeats only warn (their uniqueness is not yet a rule).
	d2 = f.saveOK(t, f.receiver2, d2, lineIn(f.lineB, 2, 2, 0, unitIn{"condition": "ACCEPTED", "motorSerial": "M-5"},
		unitIn{"condition": "ACCEPTED", "chassisNumber": "CH-0004", "motorSerial": "m-5"}))
	units := lineOf(t, d2, f.lineB).Units
	if !slicesHave(units[0].Warnings, WarnChassisMissing, WarnDuplicateMotorSerial) || !slicesHave(units[1].Warnings, WarnDuplicateMotorSerial) {
		t.Fatalf("serial warnings %v %v", units[0].Warnings, units[1].Warnings)
	}
	// Once the other draft is cancelled its chassis numbers are free again.
	expect(t, f.h.call(f.receiver, http.MethodPost, "/goods-receipts/"+d1.ID+"/cancel",
		map[string]any{"version": d1.Version, "reason": "Wrong delivery"}), http.StatusOK, nil)
	f.saveOK(t, f.receiver2, d2, lineIn(f.lineB, 1, 1, 0, accepted("CH-0002")))
	// The database refuses a duplicate chassis on one receipt even outside the API.
	lineID := f.scalar(t, `SELECT id::text FROM goods_receipt_lines WHERE goods_receipt_id = $1`, d2.ID)
	_, err := f.pool.Exec(context.Background(), `INSERT INTO goods_receipt_units (goods_receipt_id, goods_receipt_line_id, unit_no, condition,
		chassis_number, chassis_normalised) VALUES ($1, $2, 9, 'ACCEPTED', 'ch-0002', 'CH-0002')`, d2.ID, lineID)
	if sqlState(err) != "23505" {
		t.Fatalf("duplicate chassis insert: %v", err)
	}
	_, err = f.pool.Exec(context.Background(), `INSERT INTO goods_receipt_units (goods_receipt_id, goods_receipt_line_id, unit_no, condition,
		chassis_number, chassis_normalised) VALUES ($1, $2, 9, 'ACCEPTED', 'x-1', 'Y-1')`, d2.ID, lineID)
	if sqlState(err) != "23514" {
		t.Fatalf("inconsistent normalised chassis: %v", err)
	}
}

func TestConcurrentDraftsCannotOverTakeThePO(t *testing.T) {
	f := setup(t)
	// Two receivers each try to take 3 of the 5 units at the same moment.
	drafts := []Detail{f.create(t, f.receiver), f.create(t, f.receiver2)}
	sessions := []session{f.receiver, f.receiver2}
	codes := make([]int, 2)
	var wg sync.WaitGroup
	for i := range drafts {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i] = f.save(sessions[i], drafts[i], lineIn(f.lineA, 3, 3, 0, accepted("CH-SAME"))).Code
		}(i)
	}
	wg.Wait()
	ok, refused := 0, 0
	for _, c := range codes {
		switch c {
		case http.StatusOK:
			ok++
		case http.StatusConflict:
			refused++
		}
	}
	if ok != 1 || refused != 1 {
		t.Fatalf("codes %v: exactly one save may win", codes)
	}
	held := f.scalar(t, `SELECT coalesce(sum(gl.received_quantity), 0)::text FROM goods_receipt_lines gl
		JOIN goods_receipts g ON g.id = gl.goods_receipt_id WHERE g.status = 'DRAFT' AND gl.purchase_order_line_id = $1`, f.lineA)
	if held != "3" {
		t.Fatalf("drafts hold %s units", held)
	}
	// Different chassis, within the PO: both concurrent saves of line B succeed.
	for i := range drafts {
		var d Detail
		expect(t, f.h.call(sessions[i], http.MethodGet, "/goods-receipts/"+drafts[i].ID, nil), http.StatusOK, &d)
		drafts[i] = d
	}
	for i := range drafts {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i] = f.save(sessions[i], drafts[i], lineIn(f.lineB, 1, 1, 0, accepted("CH-B"+string(rune('0'+i))))).Code
		}(i)
	}
	wg.Wait()
	if codes[0] != http.StatusOK || codes[1] != http.StatusOK {
		t.Fatalf("independent saves %v", codes)
	}
}

func TestVersionsStatesAndHistoryGuard(t *testing.T) {
	f := setup(t)
	d := f.create(t, f.receiver)
	// Idempotent create.
	key := map[string]string{IdempotencyHeader: "grn-create-0001"}
	body := map[string]any{"purchaseOrderId": f.po, "receivedDate": pastDate}
	var a, b Detail
	expect(t, f.h.callWith(f.receiver, http.MethodPost, "/goods-receipts", body, key), http.StatusCreated, &a)
	w := f.h.callWith(f.receiver, http.MethodPost, "/goods-receipts", body, key)
	expect(t, w, http.StatusOK, &b)
	if a.ID != b.ID || w.Header().Get("Idempotent-Replayed") != "true" {
		t.Fatalf("replay %s %s", a.ID, b.ID)
	}
	// Only approved POs, no future dates.
	e := expectError(t, f.h.call(f.receiver, http.MethodPost, "/goods-receipts", map[string]any{"purchaseOrderId": f.draftPO, "receivedDate": pastDate}),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if !hasField(e.Details, "purchaseOrderId") {
		t.Fatalf("draft PO %+v", e.Details)
	}
	e = expectError(t, f.h.call(f.receiver, http.MethodPost, "/goods-receipts", map[string]any{"purchaseOrderId": f.po, "receivedDate": "2999-01-01"}),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if !hasField(e.Details, "receivedDate") {
		t.Fatalf("future date %+v", e.Details)
	}
	expectError(t, f.h.call(f.receiver, http.MethodPost, "/goods-receipts", map[string]any{"purchaseOrderId": uuid.NewString(), "receivedDate": pastDate}),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	expectError(t, f.h.call(f.receiver, http.MethodGet, "/goods-receipts/"+uuid.NewString(), nil), http.StatusNotFound, "NOT_FOUND")
	// A line of another PO is refused.
	other := f.scalar(t, `SELECT id::text FROM purchase_order_lines WHERE purchase_order_id = $1`, f.draftPO)
	e = expectError(t, f.save(f.receiver, d, lineIn(other, 1, 1, 0)), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if !hasField(e.Details, "lines[0].purchaseOrderLineId") {
		t.Fatalf("foreign line %+v", e.Details)
	}
	// Stale version.
	saved := f.saveOK(t, f.receiver, d, lineIn(f.lineA, 1, 1, 0))
	expectError(t, f.save(f.receiver2, d, lineIn(f.lineA, 2, 2, 0)), http.StatusConflict, "VERSION_CONFLICT")
	// The PO closes (cancelled with nothing posted): the draft can no longer change, only be cancelled.
	f.exec(t, `UPDATE purchase_orders SET status = 'CANCELLED', cancelled_by = $2, cancelled_at = now(), cancel_reason = 'test' WHERE id = $1`,
		f.po, f.ids["buyer"])
	expectError(t, f.save(f.receiver, saved, lineIn(f.lineA, 1, 1, 0)), http.StatusConflict, "INVALID_STATE")
	var cancelled Detail
	expect(t, f.h.call(f.receiver, http.MethodPost, "/goods-receipts/"+saved.ID+"/cancel",
		map[string]any{"version": saved.Version, "reason": "PO cancelled"}), http.StatusOK, &cancelled)
	expectError(t, f.h.call(f.receiver, http.MethodPost, "/goods-receipts/"+saved.ID+"/cancel",
		map[string]any{"version": cancelled.Version, "reason": "again"}), http.StatusConflict, "INVALID_STATE")
	if len(cancelled.Events) != 2 || cancelled.Events[1].Action != ActionCancelled || f.audits(t, saved.ID, "goods_receipt.cancelled", "SUCCESS") != 1 {
		t.Fatalf("history %+v", cancelled.Events)
	}
	// History guard: nothing is deleted; lines of a non-draft receipt never change.
	ctx := context.Background()
	for _, sql := range []string{
		`DELETE FROM goods_receipts WHERE id = $1`,
		`DELETE FROM goods_receipt_events WHERE goods_receipt_id = $1`,
		`UPDATE goods_receipt_lines SET received_quantity = 2, accepted_quantity = 2 WHERE goods_receipt_id = $1`,
		`DELETE FROM goods_receipt_lines WHERE goods_receipt_id = $1`,
	} {
		if _, err := f.pool.Exec(ctx, sql, saved.ID); sqlState(err) != "42501" {
			t.Errorf("%s: %v", sql, err)
		}
	}
	// Posting needs its own complete record (posted_by/at); the API has no way to set it.
	if _, err := f.pool.Exec(ctx, `UPDATE goods_receipts SET status = 'POSTED' WHERE id = $1`, a.ID); sqlState(err) != "23514" {
		t.Fatalf("incomplete posting: %v", err)
	}
}

func TestDocumentsUseThePrivateAttachmentService(t *testing.T) {
	f := setup(t)
	d := f.create(t, f.receiver)
	upload := func(by string) string {
		id := uuid.NewString()
		sum := sha256.Sum256([]byte(id))
		f.exec(t, `INSERT INTO attachments (id, object_key, storage_driver, content_type, size_bytes, sha256, original_name, uploaded_by)
			VALUES ($1, $2, 'local', 'application/pdf', 10, $3, 'challan.pdf', $4)`, id, "2026/10/"+uuid.NewString(), sum[:], by)
		return id
	}
	mine := upload(f.ids["receiver"])
	var out Detail
	expect(t, f.h.call(f.receiver, http.MethodPost, "/goods-receipts/"+d.ID+"/documents",
		map[string]any{"attachmentId": mine, "documentType": "DELIVERY_CHALLAN", "note": "Driver copy"}), http.StatusCreated, &out)
	if len(out.Documents) != 1 || out.Documents[0].FileName != "challan.pdf" || out.Documents[0].DocumentType != "DELIVERY_CHALLAN" ||
		out.Version != d.Version {
		t.Fatalf("documents %+v version %d", out.Documents, out.Version)
	}
	if owner := f.scalar(t, `SELECT owner_type || ':' || owner_id FROM attachments WHERE id = $1`, mine); owner != "goods_receipt:"+d.ID {
		t.Fatalf("owner %s", owner)
	}
	// Someone else's upload, or one already linked, cannot be attached.
	for _, att := range []string{upload(f.ids["viewer"]), mine} {
		e := expectError(t, f.h.call(f.receiver, http.MethodPost, "/goods-receipts/"+d.ID+"/documents",
			map[string]any{"attachmentId": att, "documentType": "PHOTO"}), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
		if !hasField(e.Details, "attachmentId") {
			t.Fatalf("details %+v", e.Details)
		}
	}
	// Server-side authorisation of the file itself: attachments.view and goods_receipts.view are both needed.
	svc := attachments.NewService(attachments.NewPGStore(f.pool), nil, nil, 1<<20, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx := context.Background()
	for _, c := range []struct {
		perms []string
		ok    bool
	}{
		{[]string{access.PermAttachmentsView, access.PermGoodsReceiptView}, true},
		{[]string{access.PermAttachmentsView}, false},
		{[]string{access.PermGoodsReceiptView}, false},
		{[]string{access.PermAttachmentsView, access.PermSupplierView}, false},
	} {
		_, err := svc.Get(ctx, access.NewAccess(f.ids["viewer"], nil, c.perms, nil, false), mine)
		if (err == nil) != c.ok {
			t.Errorf("perms %v: err %v, want visible=%v", c.perms, err, c.ok)
		}
	}
	// Even the uploader loses the file once it is linked, without goods_receipts.view.
	if _, err := svc.Get(ctx, access.NewAccess(f.ids["receiver"], nil, []string{access.PermAttachmentsView}, nil, false), mine); err == nil {
		t.Fatal("uploader sees the linked document without goods_receipts.view")
	}
	// Document links are history; cancelled receipts take no more documents.
	if _, err := f.pool.Exec(ctx, `DELETE FROM goods_receipt_documents WHERE goods_receipt_id = $1`, d.ID); sqlState(err) != "42501" {
		t.Fatalf("document delete: %v", err)
	}
	expect(t, f.h.call(f.receiver, http.MethodPost, "/goods-receipts/"+d.ID+"/cancel", map[string]any{"version": d.Version, "reason": "x"}),
		http.StatusOK, nil)
	expectError(t, f.h.call(f.receiver, http.MethodPost, "/goods-receipts/"+d.ID+"/documents",
		map[string]any{"attachmentId": upload(f.ids["receiver"]), "documentType": "PHOTO"}), http.StatusConflict, "INVALID_STATE")
	if f.audits(t, d.ID, "goods_receipt.document_added", "SUCCESS") != 1 {
		t.Fatal("document link not audited")
	}
}
