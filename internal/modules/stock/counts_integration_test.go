package stock_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database/dbtest"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/stock"
)

// Stock counts and adjustment approval (E-05) on a throwaway migrated database.
// Grants are test-only (in-memory harness); SKU codes and quantities are placeholders.

type countsFixture struct {
	*partsFixture
	approver session // stock.adjustment.view/approve, every location
}

var (
	countCreate  = []string{access.PermAdjustmentView, access.PermAdjustmentCreate}
	countApprove = []string{access.PermAdjustmentView, access.PermAdjustmentApprove}
)

func setupCounts(t *testing.T) *countsFixture {
	t.Helper()
	p := setupParts(t)
	c := &countsFixture{partsFixture: p}
	// The counter (mover) receives parts and counts; the approver only decides.
	p.h.grant(p.mover, append(append([]string{}, partsAll...), countCreate...)...)
	c.approver = p.h.login("approver", p.ids["approver"])
	p.h.grant(c.approver, append(countApprove, access.PermAllLocations)...)
	expect(t, p.call(p.mover, p.item, "receipts", p.receipt("10"), "rcpt-0001"), http.StatusOK, nil)
	return c
}

func (c *countsFixture) post(s session, path string, body any, key string) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	var h map[string]string
	if key != "" {
		h = map[string]string{stock.IdempotencyHeader: key}
	}
	return c.h.raw(http.MethodPost, "/api/v1/stock-counts"+path, string(b), &s, h)
}

func (c *countsFixture) create(t *testing.T, s session, loc, key string) stock.StockCount {
	t.Helper()
	var sc stock.StockCount
	expect(t, c.post(s, "", map[string]any{"locationId": loc}, key), http.StatusCreated, &sc)
	return sc
}

func (c *countsFixture) partsLine(s session, sc stock.StockCount, item, observed, reason string) *httptest.ResponseRecorder {
	body := map[string]any{"version": sc.Version, "stockItemId": item, "observedQuantity": observed, "uom": "PCS"}
	if reason != "" {
		body["reason"] = reason
	}
	return c.post(s, "/"+sc.ID+"/lines", body, "")
}

func (c *countsFixture) submit(s session, sc stock.StockCount) *httptest.ResponseRecorder {
	return c.post(s, "/"+sc.ID+"/submit", map[string]any{"version": sc.Version}, "")
}

func (c *countsFixture) approve(s session, sc stock.StockCount, key string) *httptest.ResponseRecorder {
	return c.post(s, "/"+sc.ID+"/approve", map[string]any{"version": sc.Version}, key)
}

// submitted creates a count at L1 with one parts line (observed, reason) and submits it.
func (c *countsFixture) submitted(t *testing.T, key, observed string) stock.StockCount {
	t.Helper()
	sc := c.create(t, c.mover, c.locL1, key)
	expect(t, c.partsLine(c.mover, sc, c.item, observed, "Counted on the shelf"), http.StatusOK, &sc)
	expect(t, c.submit(c.mover, sc), http.StatusOK, &sc)
	return sc
}

func (c *countsFixture) adjustments(t *testing.T) int {
	return c.count(t, `SELECT count(*) FROM stock_movements WHERE movement_type = 'COUNT_ADJUSTMENT'`)
}

func (c *countsFixture) audits(t *testing.T, action, outcome, entity string) int {
	return c.count(t, `SELECT count(*) FROM audit_events WHERE action = $1 AND outcome = $2 AND entity_id = $3`, action, outcome, entity)
}

// 1, 2, 4, 14: recording and submitting change nothing; approval posts once, atomically, audited.
func TestCountApprovalPostsTheAdjustmentOnce(t *testing.T) {
	c := setupCounts(t)
	sc := c.create(t, c.mover, c.locL1, "count-0001")
	if sc.Status != stock.CountDraft || !strings.HasPrefix(sc.CountNumber, "SC-") || sc.ApprovalPolicy != stock.PolicyApproverRequired {
		t.Fatalf("created %+v", sc)
	}
	// A variance needs a reason; client-sent expected values are not accepted.
	expectError(t, c.partsLine(c.mover, sc, c.item, "8", ""), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	w := c.post(c.mover, "/"+sc.ID+"/lines", map[string]any{"version": sc.Version, "stockItemId": c.item, "observedQuantity": "8",
		"uom": "PCS", "reason": "x", "expectedQuantity": "8"}, "")
	expectError(t, w, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	expect(t, c.partsLine(c.mover, sc, c.item, "8", "Two pieces damaged"), http.StatusOK, &sc)
	l := sc.Lines[0]
	if len(sc.Lines) != 1 || l.ExpectedQuantity != "10.000" || l.ObservedQuantity != "8.000" || l.Variance != "-2.000" ||
		l.Outcome != stock.OutcomePending || l.StockChanged || *l.Reason != "Two pieces damaged" || sc.VarianceLines != 1 {
		t.Fatalf("line %+v", sc)
	}
	expect(t, c.submit(c.mover, sc), http.StatusOK, &sc)
	if sc.Status != stock.CountSubmitted || *sc.SubmittedBy != c.ids["mover"] {
		t.Fatalf("submitted %+v", sc)
	}
	// Pending: no ledger effect.
	if b := c.balance(t, c.item, c.locL1); b != "10.000" || c.adjustments(t) != 0 {
		t.Fatalf("pending changed stock: %s, %d", b, c.adjustments(t))
	}
	var ap stock.StockCount
	w = c.approve(c.approver, sc, "approve-0001")
	expect(t, w, http.StatusOK, &ap)
	if ap.Status != stock.CountApproved || *ap.DecidedBy != c.ids["approver"] || ap.Lines[0].Outcome != stock.OutcomePosted ||
		ap.Lines[0].PostedMovementID == nil || w.Header().Get("Idempotent-Replayed") != "" {
		t.Fatalf("approved %+v", ap)
	}
	if b := c.balance(t, c.item, c.locL1); b != "8.000" {
		t.Fatalf("balance %s", b)
	}
	var typ, src, ref, reason, before, after, actor string
	if err := c.pool.QueryRow(context.Background(), `SELECT movement_type, source_type, source_reference, reason, quantity_before::text,
		quantity_after::text, actor_id::text FROM stock_movements WHERE id = $1`, *ap.Lines[0].PostedMovementID).
		Scan(&typ, &src, &ref, &reason, &before, &after, &actor); err != nil {
		t.Fatal(err)
	}
	if typ != stock.TypeCountAdjustment || src != stock.SourceStockCount || ref != sc.CountNumber || reason != "Two pieces damaged" ||
		before != "10.000" || after != "8.000" || actor != c.ids["approver"] {
		t.Fatalf("movement %s %s %s %s %s->%s by %s", typ, src, ref, reason, before, after, actor)
	}
	for _, a := range []string{"stock_count.created", "stock_count.line_recorded", "stock_count.submitted", "stock_count.approved"} {
		if n := c.audits(t, a, "SUCCESS", sc.ID); n != 1 {
			t.Errorf("audit %s: %d", a, n)
		}
	}
	if n := c.audits(t, "stock_item.adjusted", "SUCCESS", c.item); n != 1 {
		t.Errorf("adjusted audit %d", n)
	}
	// 5: the same approval again is a replay; another decision is refused.
	w = c.approve(c.approver, sc, "approve-0001")
	expect(t, w, http.StatusOK, &ap)
	if w.Header().Get("Idempotent-Replayed") != "true" || c.adjustments(t) != 1 || c.balance(t, c.item, c.locL1) != "8.000" {
		t.Fatalf("replay posted again: %d", c.adjustments(t))
	}
	expectError(t, c.post(c.approver, "/"+sc.ID+"/approve", map[string]any{"version": sc.Version, "note": "other"}, "approve-0001"),
		http.StatusConflict, "IDEMPOTENCY_KEY_REUSED")
	expectError(t, c.approve(c.approver, sc, "approve-0002"), http.StatusConflict, "INVALID_STATE")
	expectError(t, c.post(c.approver, "/"+sc.ID+"/reject", map[string]any{"version": ap.Version, "reason": "late"}, "reject-0001"),
		http.StatusConflict, "INVALID_STATE")
	expectError(t, c.partsLine(c.mover, ap, c.item, "1", "late"), http.StatusConflict, "INVALID_STATE")
	// 15: the E-04 receipts/issues history keeps its contract; the ledger reconciles.
	var hist stock.PartsPage
	expect(t, c.h.raw(http.MethodGet, "/api/v1/stock-items/"+c.item+"/stock-movements", "", &c.mover, nil), http.StatusOK, &hist)
	if hist.Total != 1 || hist.Items[0].MovementType != stock.TypePartsReceipt {
		t.Fatalf("parts history %+v", hist)
	}
	c.reconcilesParts(t)
	// Reads: detail and the approval queue.
	var got stock.StockCount
	expect(t, c.h.raw(http.MethodGet, "/api/v1/stock-counts/"+sc.ID, "", &c.approver, nil), http.StatusOK, &got)
	if got.Status != stock.CountApproved || len(got.Lines) != 1 || *got.Lines[0].SKU != "PRT-1" {
		t.Fatalf("detail %+v", got)
	}
	var page stock.CountPage
	expect(t, c.h.raw(http.MethodGet, "/api/v1/stock-counts?status=SUBMITTED", "", &c.approver, nil), http.StatusOK, &page)
	if page.Total != 0 {
		t.Fatalf("queue %+v", page)
	}
}

// 3, 10: a rejection needs a reason, posts nothing and is final.
func TestRejectedCountPostsNothing(t *testing.T) {
	c := setupCounts(t)
	sc := c.submitted(t, "count-0001", "3")
	expectError(t, c.post(c.approver, "/"+sc.ID+"/reject", map[string]any{"version": sc.Version}, "reject-0001"),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	var rj stock.StockCount
	expect(t, c.post(c.approver, "/"+sc.ID+"/reject", map[string]any{"version": sc.Version, "reason": "Recount the shelf"}, "reject-0001"),
		http.StatusOK, &rj)
	if rj.Status != stock.CountRejected || *rj.DecisionNote != "Recount the shelf" || rj.Lines[0].Outcome != stock.OutcomeNotPosted {
		t.Fatalf("rejected %+v", rj)
	}
	if c.balance(t, c.item, c.locL1) != "10.000" || c.adjustments(t) != 0 {
		t.Fatal("a rejection changed stock")
	}
	expectError(t, c.approve(c.approver, rj, "approve-0001"), http.StatusConflict, "INVALID_STATE")
	if n := c.audits(t, "stock_count.rejected", "SUCCESS", sc.ID); n != 1 {
		t.Fatalf("audit %d", n)
	}
}

// 7, 8, 9, 12: permissions, separation of duties, no approver = stays pending, scope.
func TestCountPermissionsSelfApprovalAndScope(t *testing.T) {
	c := setupCounts(t)
	// The counter may also hold the approve permission: still never decides their own count.
	c.h.grant(c.mover, append(append(append([]string{}, partsAll...), countCreate...), access.PermAdjustmentApprove)...)
	sc := c.submitted(t, "count-0001", "9")
	expectError(t, c.approve(c.mover, sc, "self-0001"), http.StatusForbidden, "SELF_APPROVAL_NOT_ALLOWED")
	expectError(t, c.post(c.mover, "/"+sc.ID+"/reject", map[string]any{"version": sc.Version, "reason": "no"}, "self-0002"),
		http.StatusForbidden, "SELF_APPROVAL_NOT_ALLOWED")
	if n := c.audits(t, "stock_count.approved", "DENIED", sc.ID); n != 1 {
		t.Fatalf("denied audit %d", n)
	}
	// Someone who counted a line of a draft cannot decide it either.
	c.h.grant(c.other, append(append([]string{}, countCreate...), access.PermAdjustmentApprove, access.PermAllLocations)...)
	d := c.create(t, c.mover, c.locL1, "count-0002")
	expect(t, c.partsLine(c.other, d, c.item, "9", "Shelf B"), http.StatusOK, &d)
	expect(t, c.submit(c.mover, d), http.StatusOK, &d)
	expectError(t, c.approve(c.other, d, "other-0001"), http.StatusForbidden, "SELF_APPROVAL_NOT_ALLOWED")
	// Without the approve permission: 403, and nothing else can decide it -> it stays pending (fail closed).
	c.h.grant(c.viewer, access.PermAdjustmentView, access.PermAllLocations)
	expectError(t, c.approve(c.viewer, sc, "view-0001"), http.StatusForbidden, "FORBIDDEN")
	expectError(t, c.post(c.viewer, "", map[string]any{"locationId": c.locL1}, "view-0002"), http.StatusForbidden, "FORBIDDEN")
	expectError(t, c.partsLine(c.viewer, sc, c.item, "1", "x"), http.StatusForbidden, "FORBIDDEN")
	expectError(t, c.submit(c.viewer, sc), http.StatusForbidden, "FORBIDDEN")
	c.h.grant(c.viewer)
	expectError(t, c.h.raw(http.MethodGet, "/api/v1/stock-counts", "", &c.viewer, nil), http.StatusForbidden, "FORBIDDEN")
	var still stock.StockCount
	expect(t, c.h.raw(http.MethodGet, "/api/v1/stock-counts/"+sc.ID, "", &c.approver, nil), http.StatusOK, &still)
	if still.Status != stock.CountSubmitted || c.adjustments(t) != 0 || c.balance(t, c.item, c.locL1) != "10.000" {
		t.Fatalf("refusals changed something: %s %d", still.Status, c.adjustments(t))
	}
	// Location scope: a user limited to L2 neither sees nor decides L1 counts.
	c.h.grant(c.scoped, append(append([]string{}, countCreate...), access.PermAdjustmentApprove)...)
	expectError(t, c.h.raw(http.MethodGet, "/api/v1/stock-counts/"+sc.ID, "", &c.scoped, nil), http.StatusNotFound, "NOT_FOUND")
	expectError(t, c.approve(c.scoped, sc, "scoped-0001"), http.StatusNotFound, "NOT_FOUND")
	expectError(t, c.post(c.scoped, "", map[string]any{"locationId": c.locL1}, "scoped-0002"), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	expectError(t, c.h.raw(http.MethodGet, "/api/v1/stock-counts?locationId="+c.locL1, "", &c.scoped, nil),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	var page stock.CountPage
	expect(t, c.h.raw(http.MethodGet, "/api/v1/stock-counts", "", &c.scoped, nil), http.StatusOK, &page)
	if page.Total != 0 {
		t.Fatalf("scoped list leaks %+v", page)
	}
	// The real approver can.
	expect(t, c.approve(c.approver, sc, "approve-0001"), http.StatusOK, nil)
	if c.balance(t, c.item, c.locL1) != "9.000" {
		t.Fatal("approval did not post")
	}
}

// 10: stale versions and stock that moved after the count are refused, nothing posted.
func TestStaleVersionsAndStockChangedSinceCount(t *testing.T) {
	c := setupCounts(t)
	sc := c.create(t, c.mover, c.locL1, "count-0001")
	stale := sc
	expect(t, c.partsLine(c.mover, sc, c.item, "10", ""), http.StatusOK, &sc) // no variance, no reason needed
	expectError(t, c.partsLine(c.mover, stale, c.item, "9", "x"), http.StatusConflict, "VERSION_CONFLICT")
	expectError(t, c.submit(c.mover, stale), http.StatusConflict, "VERSION_CONFLICT")
	// Stock moves after the line was counted: submit names the line.
	expect(t, c.call(c.mover, c.item, "issues", c.issue("1"), "issue-0001"), http.StatusOK, nil)
	e := expectError(t, c.submit(c.mover, sc), http.StatusConflict, "STOCK_CHANGED_SINCE_COUNT")
	if len(e.Details) != 1 || e.Details[0].Field != "lines."+sc.Lines[0].ID {
		t.Fatalf("details %+v", e.Details)
	}
	var got stock.StockCount
	expect(t, c.h.raw(http.MethodGet, "/api/v1/stock-counts/"+sc.ID, "", &c.mover, nil), http.StatusOK, &got)
	if !got.Lines[0].StockChanged {
		t.Fatal("stockChanged not shown")
	}
	// Re-recording reads the new baseline.
	expect(t, c.partsLine(c.mover, got, c.item, "7", "Two missing"), http.StatusOK, &sc)
	if sc.Lines[0].ExpectedQuantity != "9.000" || sc.Lines[0].Variance != "-2.000" || sc.Lines[0].StockChanged {
		t.Fatalf("re-recorded %+v", sc.Lines[0])
	}
	expect(t, c.submit(c.mover, sc), http.StatusOK, &sc)
	// Stock moves again before approval: refused (audited), nothing posted, can be rejected.
	expect(t, c.call(c.mover, c.item, "receipts", c.receipt("5"), "rcpt-0002"), http.StatusOK, nil)
	expectError(t, c.approve(c.approver, sc, "approve-0001"), http.StatusConflict, "STOCK_CHANGED_SINCE_COUNT")
	if c.adjustments(t) != 0 || c.balance(t, c.item, c.locL1) != "14.000" {
		t.Fatal("stale approval posted")
	}
	if n := c.audits(t, "stock_count.approved", "DENIED", sc.ID); n != 1 {
		t.Fatalf("denied audit %d", n)
	}
	expectError(t, c.approve(c.approver, stock.StockCount{ID: sc.ID, Version: sc.Version - 1}, "approve-0002"),
		http.StatusConflict, "VERSION_CONFLICT")
	expect(t, c.post(c.approver, "/"+sc.ID+"/reject", map[string]any{"version": sc.Version, "reason": "Stock moved"}, "reject-0001"),
		http.StatusOK, nil)
	// A submit with no lines is refused.
	empty := c.create(t, c.mover, c.locL1, "count-0002")
	expectError(t, c.submit(c.mover, empty), http.StatusConflict, "INVALID_STATE")
}

// 11 and input validation: quantities, units, tracking modes, item references.
func TestCountLineValidationAndNoNegativeStock(t *testing.T) {
	c := setupCounts(t)
	sc := c.create(t, c.mover, c.locL1, "count-0001")
	for _, b := range []map[string]any{
		{"stockItemId": c.item, "observedQuantity": "-1", "uom": "PCS", "reason": "x"},
		{"stockItemId": c.item, "observedQuantity": "1.2345", "uom": "PCS", "reason": "x"},
		{"stockItemId": c.item, "observedQuantity": "1e3", "uom": "PCS", "reason": "x"},
		{"stockItemId": c.item, "observedQuantity": "5", "reason": "x"},
		{"stockItemId": c.item, "observedQuantity": "5", "uom": "BOX", "reason": "x"},
		{"stockItemId": c.item, "vehicleUnitId": c.item, "observedQuantity": "5", "uom": "PCS"},
		{"observedQuantity": "5", "uom": "PCS"},
		{"stockItemId": "not-an-id", "observedQuantity": "5", "uom": "PCS"},
		{"stockItemId": c.locL1, "observedQuantity": "5", "uom": "PCS", "reason": "x"},
		{"stockItemId": c.item, "observedQuantity": "5", "uom": "PCS", "reason": "x", "found": true},
	} {
		b["version"] = sc.Version
		expectError(t, c.post(c.mover, "/"+sc.ID+"/lines", b, ""), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	}
	b := map[string]any{"version": sc.Version, "stockItemId": c.serial, "observedQuantity": "1", "uom": "PCS", "reason": "x"}
	expectError(t, c.post(c.mover, "/"+sc.ID+"/lines", b, ""), http.StatusConflict, "TRACKING_NOT_SUPPORTED")
	b["stockItemId"] = c.archived
	expectError(t, c.post(c.mover, "/"+sc.ID+"/lines", b, ""), http.StatusConflict, "INVALID_STATE")
	// Counting zero takes the balance to zero, never below; a SKU never stocked can be counted up.
	expect(t, c.partsLine(c.mover, sc, c.item, "0", "All used, not recorded"), http.StatusOK, &sc)
	other := c.scalar(t, `INSERT INTO stock_items (sku, name, category_id, uom, tracking_mode, created_by)
		VALUES ('PRT-2', 'Item 2', $1, 'PCS', 'QUANTITY', $2) RETURNING id::text`, c.category, c.ids["buyer"])
	expect(t, c.partsLine(c.mover, sc, other, "2.5", "Found in the back room"), http.StatusOK, &sc)
	expect(t, c.submit(c.mover, sc), http.StatusOK, &sc)
	expect(t, c.approve(c.approver, sc, "approve-0001"), http.StatusOK, &sc)
	if c.balance(t, c.item, c.locL1) != "0.000" || c.balance(t, other, c.locL1) != "2.500" || c.adjustments(t) != 2 {
		t.Fatalf("balances %s %s", c.balance(t, c.item, c.locL1), c.balance(t, other, c.locL1))
	}
	c.reconcilesParts(t)
	// The ledger itself refuses a negative or unchanged adjustment.
	for _, q := range [][2]string{{"0", "-1"}, {"0", "0"}} {
		_, err := c.pool.Exec(context.Background(), `INSERT INTO stock_movements (item_type, stock_item_id, movement_type,
			quantity_before, quantity_after, from_location_id, to_location_id, uom, source_type, source_id, source_reference, reason,
			actor_id, idempotency_key, request_hash) VALUES ('STOCK_ITEM', $1, 'COUNT_ADJUSTMENT', $2::numeric, $3::numeric, $4, $4,
			'PCS', 'STOCK_COUNT', $5, 'SC-X', 'x', $6, 'direct-'||md5(random()::text), repeat('0', 64))`,
			c.item, q[0], q[1], c.locL1, sc.ID, c.ids["buyer"])
		if code, con := sqlState(err); code != "23514" || con != "stock_movements_quantity_valid" {
			t.Errorf("%v: %v", q, err)
		}
	}
}

// 6: concurrent approvals post once.
func TestConcurrentApprovalsPostOnce(t *testing.T) {
	c := setupCounts(t)
	c.h.grant(c.other, append(countApprove, access.PermAllLocations)...)
	// Same approver, same key, in parallel: one approval, the rest are replays.
	sc := c.submitted(t, "count-0001", "6")
	ok, other := codes(parallel(8, func(int) *httptest.ResponseRecorder { return c.approve(c.approver, sc, "approve-same") }))
	if ok != 8 || len(other) != 0 || c.adjustments(t) != 1 || c.balance(t, c.item, c.locL1) != "6.000" {
		t.Fatalf("same key: ok %d other %v adjustments %d", ok, other, c.adjustments(t))
	}
	// Two approvers with different keys: exactly one wins.
	sc = c.submitted(t, "count-0002", "4")
	ok, other = codes(parallel(8, func(i int) *httptest.ResponseRecorder {
		s := c.approver
		if i%2 == 1 {
			s = c.other
		}
		return c.approve(s, sc, "approve-"+strconv.Itoa(1000+i))
	}))
	if ok != 1 || other["INVALID_STATE"] != 7 || c.adjustments(t) != 2 || c.balance(t, c.item, c.locL1) != "4.000" {
		t.Fatalf("different keys: ok %d other %v", ok, other)
	}
	// An approval racing a stock change: either the approval sees the issue
	// (refused) or the issue sees the adjusted balance; stock never double-moves.
	sc = c.submitted(t, "count-0003", "3")
	ws := parallel(2, func(i int) *httptest.ResponseRecorder {
		if i == 0 {
			return c.approve(c.approver, sc, "approve-race")
		}
		return c.call(c.mover, c.item, "issues", c.issue("1"), "issue-race")
	})
	switch {
	case ws[0].Code == http.StatusOK && ws[1].Code == http.StatusOK:
		if c.balance(t, c.item, c.locL1) != "2.000" {
			t.Fatalf("approve then issue: %s", c.balance(t, c.item, c.locL1))
		}
	case ws[0].Code == http.StatusConflict && ws[1].Code == http.StatusOK:
		if c.balance(t, c.item, c.locL1) != "3.000" || c.adjustments(t) != 2 {
			t.Fatalf("issue first: %s", c.balance(t, c.item, c.locL1))
		}
	default:
		t.Fatalf("race: %d %s / %d %s", ws[0].Code, ws[0].Body, ws[1].Code, ws[1].Body)
	}
	c.reconcilesParts(t)
}

// 13: a vehicle discrepancy is recorded and approved as a finding; the unit never changes.
func TestVehicleDiscrepancyIsRecordedOnly(t *testing.T) {
	c := setupCounts(t)
	units := c.available(t, 2)
	elsewhere := c.register(t, &c.locL2, "CH")[0]
	movements := c.count(t, `SELECT count(*) FROM stock_movements WHERE item_type = 'VEHICLE_UNIT'`)
	sc := c.create(t, c.mover, c.locL1, "count-0001")
	line := func(unit string, found bool, reason string) *httptest.ResponseRecorder {
		b := map[string]any{"version": sc.Version, "vehicleUnitId": unit, "found": found}
		if reason != "" {
			b["reason"] = reason
		}
		return c.post(c.mover, "/"+sc.ID+"/lines", b, "")
	}
	expectError(t, line(units[0], false, ""), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	expectError(t, line(elsewhere, true, ""), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	expect(t, line(units[0], false, "Not on the floor"), http.StatusOK, &sc)
	expect(t, line(units[1], true, ""), http.StatusOK, &sc)
	before, version := c.unit(t, units[0])
	expect(t, c.submit(c.mover, sc), http.StatusOK, &sc)
	expect(t, c.approve(c.approver, sc, "approve-0001"), http.StatusOK, &sc)
	out := map[string]string{}
	for _, l := range sc.Lines {
		out[*l.VehicleUnitID] = l.Outcome
		if l.PostedMovementID != nil {
			t.Fatal("vehicle line posted a movement")
		}
	}
	if out[units[0]] != stock.OutcomeDiscrepancyRecorded || out[units[1]] != stock.OutcomeNoChange {
		t.Fatalf("outcomes %v", out)
	}
	if st, v := c.unit(t, units[0]); st != before || v != version ||
		c.count(t, `SELECT count(*) FROM stock_movements WHERE item_type = 'VEHICLE_UNIT'`) != movements || c.adjustments(t) != 0 {
		t.Fatalf("unit changed: %s v%d", st, v)
	}
	c.reconciles(t)
	// A unit moved after it was counted makes the count stale (E-01 moves still work).
	d := c.create(t, c.mover, c.locL1, "count-0002")
	sc = d
	expect(t, line(units[1], true, ""), http.StatusOK, &sc)
	if _, _, err := c.move(t, units[1], stock.TypeDisplayOn, "display-on-0001", nil); err != nil {
		t.Fatal(err)
	}
	expectError(t, c.submit(c.mover, sc), http.StatusConflict, "STOCK_CHANGED_SINCE_COUNT")
}

// Create idempotency: same key and body replays; another body is refused.
func TestCountCreateIdempotency(t *testing.T) {
	c := setupCounts(t)
	first := c.create(t, c.mover, c.locL1, "count-0001")
	var again stock.StockCount
	w := c.post(c.mover, "", map[string]any{"locationId": c.locL1}, "count-0001")
	expect(t, w, http.StatusOK, &again)
	if again.ID != first.ID || w.Header().Get("Idempotent-Replayed") != "true" {
		t.Fatalf("replay %+v", again)
	}
	expectError(t, c.post(c.mover, "", map[string]any{"locationId": c.locL1, "note": "other"}, "count-0001"),
		http.StatusConflict, "IDEMPOTENCY_KEY_REUSED")
	expectError(t, c.post(c.mover, "", map[string]any{"locationId": c.locL1}, ""), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	expectError(t, c.post(c.mover, "", map[string]any{"locationId": c.inactiveLoc}, "count-0002"),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	ok, _ := codes(parallel(6, func(int) *httptest.ResponseRecorder {
		return c.post(c.mover, "", map[string]any{"locationId": c.locL1}, "count-par")
	}))
	created := c.count(t, `SELECT count(*) FROM stock_counts WHERE create_idempotency_key = 'count-par'`)
	if created != 1 || ok != 5 {
		t.Fatalf("parallel creates: %d rows, %d ok", created, ok)
	}
}

// 14 and LED-006 in the database: counts and lines are never deleted, decided
// counts never change, an adjustment needs its approved count, no self-decision.
func TestDatabaseGuardsCounts(t *testing.T) {
	c := setupCounts(t)
	ctx := context.Background()
	ap := c.submitted(t, "count-0001", "8")
	expect(t, c.approve(c.approver, ap, "approve-0001"), http.StatusOK, &ap)
	pending := c.submitted(t, "count-0002", "8")
	for _, sql := range []string{
		`DELETE FROM stock_counts WHERE id = '` + ap.ID + `'`,
		`TRUNCATE stock_counts CASCADE`,
		`DELETE FROM stock_count_lines`,
	} {
		if _, err := c.pool.Exec(ctx, sql); func() bool { code, _ := sqlState(err); return code != "42501" }() {
			t.Errorf("%s: %v", sql, err)
		}
	}
	for _, q := range []struct{ sql, constraint string }{
		{`UPDATE stock_counts SET decision_note = 'changed', version = version + 1 WHERE id = '` + ap.ID + `'`, "stock_counts_lifecycle_valid"},
		{`UPDATE stock_counts SET status = 'DRAFT', version = version + 1 WHERE id = '` + pending.ID + `'`, "stock_counts_lifecycle_valid"},
		{`UPDATE stock_count_lines SET observed_quantity = 1 WHERE stock_count_id = '` + pending.ID + `'`, "stock_count_lines_editable"},
		{`UPDATE stock_counts SET status = 'REJECTED', decided_by = submitted_by, decided_at = now(), decision_note = 'x',
			decision_idempotency_key = 'self-decide', decision_request_hash = repeat('0', 64), version = version + 1
			WHERE id = '` + pending.ID + `'`, "stock_counts_no_self_decision"},
	} {
		if _, err := c.pool.Exec(ctx, q.sql); func() bool { _, con := sqlState(err); return con != q.constraint }() {
			t.Errorf("%s: %v", q.constraint, err)
		}
	}
	// A COUNT_ADJUSTMENT for a count that is not approved fails at commit.
	err := pgx.BeginFunc(ctx, c.pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO stock_movements (item_type, stock_item_id, movement_type, quantity_before, quantity_after,
			from_location_id, to_location_id, uom, source_type, source_id, source_reference, reason, actor_id, idempotency_key, request_hash)
			VALUES ('STOCK_ITEM', $1, 'COUNT_ADJUSTMENT', 8, 5, $2, $2, 'PCS', 'STOCK_COUNT', $3, 'SC-X', 'x', $4, 'direct-0001', repeat('0', 64))`,
			c.item, c.locL1, pending.ID, c.ids["buyer"])
		return err
	})
	if _, con := sqlState(err); con != "stock_movements_count_approved" {
		t.Fatalf("unapproved adjustment: %v", err)
	}
	if c.balance(t, c.item, c.locL1) != "8.000" || c.adjustments(t) != 1 {
		t.Fatal("guard left a change")
	}
}

// Migration 000018: down refuses while counts exist; without them it round-trips.
func TestStockCountMigrationRoundTrip(t *testing.T) {
	ctx := context.Background()
	url := dbtest.ScratchURL(t)
	m := dbtest.NewMigrator(t, url)
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	if err := m.Migrate(17); err != nil {
		t.Fatal(err)
	}
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	f := setupOn(t, pool)
	p := partsOn(t, f)
	c := &countsFixture{partsFixture: p}
	p.h.grant(p.mover, append(append([]string{}, partsAll...), countCreate...)...)
	c.create(t, p.mover, p.locL1, "count-0001")
	if err := m.Migrate(17); err == nil || !strings.Contains(err.Error(), "would destroy count and ledger history") {
		t.Fatalf("down over counts: %v", err)
	}
}
