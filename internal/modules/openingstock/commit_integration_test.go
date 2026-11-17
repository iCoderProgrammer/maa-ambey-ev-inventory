package openingstock_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/httpx"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/ageing"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/openingstock"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/stock"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/vehicles"
)

// Opening stock commit (G-07) on a throwaway migrated database. Every brand,
// chassis and serial is a placeholder fixture; the approved statuses are a
// test configuration, not an owner decision.

var commitLimits = openingstock.Limits{MaxBytes: 64 << 10, MaxRows: 40, PreviewTTL: 7 * 24 * time.Hour,
	OpeningStatuses: []string{"RECEIVED_PENDING_PDI", "AVAILABLE", "DISPLAY"}}

type commitFixture struct {
	*fixture
	approver, committer, committer2 session
}

func setupCommit(t *testing.T, limits openingstock.Limits) *commitFixture {
	t.Helper()
	f := setupWith(t, limits)
	c := &commitFixture{fixture: f}
	c.approver = f.h.login("approver", f.ids["approver"])
	c.committer = f.h.login("committer", f.ids["committer"])
	c.committer2 = f.h.login("committer2", f.ids["committer2"])
	// Committing needs stock.import.commit; approving stock.import.approve. Neither comes from preview.
	f.h.grant(f.importer, access.PermImportPreview)
	f.h.grant(c.approver, access.PermImportApprove, access.PermAllLocations)
	f.h.grant(c.committer, access.PermImportCommit, access.PermAllLocations)
	f.h.grant(c.committer2, access.PermImportCommit, access.PermAllLocations)
	return c
}

// validFile: three VALID lines at L1 (one names the existing supplier).
func validFile(prefix string) []byte {
	return csvFile(
		row("Brand A", "MA", "STD", "", prefix+"-1", prefix+"-M1", "L1", "AVAILABLE", "2026-01-10", "Showroom register", "", "90000"),
		row("Brand A", "MA", "STD", "", prefix+"-2", "", "L1", "RECEIVED_PENDING_PDI", "2026-02-11", "Showroom register", "Example Supplier", "91000.50"),
		row("Brand A", "MA", "STD", "", prefix+"-3", "", "L1", "DISPLAY", "2026-03-12", "Showroom register", "", "92000"),
	)
}

func (c *commitFixture) previewOf(t *testing.T, data []byte, key string) openingstock.Batch {
	t.Helper()
	var b openingstock.Batch
	expect(t, c.h.upload(&c.importer, "opening.csv", "text/csv", data, key), http.StatusCreated, &b)
	return b
}

func approval(datesApproved bool) map[string]any {
	return map[string]any{"physicalCountReference": "PC-TEST-1", "physicalCountDate": "2026-06-29", "mappingsReviewed": true,
		"duplicatesReviewed": true, "receivedDatesApproved": datesApproved}
}

func (c *commitFixture) approve(t *testing.T, id string, datesApproved bool) {
	t.Helper()
	expect(t, c.h.post(&c.approver, "/opening-stock-imports/"+id+"/approval", approval(datesApproved), ""), http.StatusCreated, nil)
}

// stockFingerprint: the tables a refused commit must leave unchanged.
func (c *commitFixture) stockFingerprint(t *testing.T) string {
	t.Helper()
	var parts []string
	for _, table := range []string{"vehicle_units", "vehicle_unit_events", "stock_movements", "stock_item_balances",
		"opening_stock_import_commits", "opening_stock_import_batches", "opening_stock_import_rows"} {
		var s string
		if err := c.pool.QueryRow(context.Background(), `SELECT count(*)::text || ':' || coalesce(md5(string_agg(t::text, '|' ORDER BY t::text)), '')
			FROM `+table+` t`).Scan(&s); err != nil {
			t.Fatal(err)
		}
		parts = append(parts, table+"="+s)
	}
	return strings.Join(parts, " ")
}

func (c *commitFixture) loadAccess(t *testing.T, userID string) access.Access {
	t.Helper()
	a, err := c.h.grants.LoadAccess(context.Background(), userID, false)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestCommitCreatesStockOnceWithProvenance(t *testing.T) {
	c := setupCommit(t, commitLimits)
	ctx := context.Background()
	b := c.previewOf(t, validFile("OS"), "upload-key-0001")
	if b.Summary.ValidRows != 3 || b.Approval != nil || b.Commit != nil || b.CommitAvailable {
		t.Fatalf("preview: %+v", b)
	}
	before := c.stockFingerprint(t)
	path := "/opening-stock-imports/" + b.ID

	// No approval yet: refused, nothing created.
	expectError(t, c.h.post(&c.committer, path+"/commit", nil, "commit-key-0001"), http.StatusConflict, "IMPORT_APPROVAL_REQUIRED")
	// Approval: permission, body checks, once.
	expectError(t, c.h.post(&c.importer, path+"/approval", approval(true), ""), http.StatusForbidden, "FORBIDDEN")
	expectError(t, c.h.post(&c.committer, path+"/approval", approval(true), ""), http.StatusForbidden, "FORBIDDEN")
	bad := approval(true)
	bad["mappingsReviewed"], bad["physicalCountReference"] = false, ""
	e := expectError(t, c.h.post(&c.approver, path+"/approval", bad, ""), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if len(e.Details) != 2 {
		t.Errorf("approval problems: %+v", e.Details)
	}
	var ap openingstock.Approval
	expect(t, c.h.post(&c.approver, path+"/approval", approval(true), ""), http.StatusCreated, &ap)
	if !ap.ReceivedDatesApproved || ap.PhysicalCountReference != "PC-TEST-1" || ap.ApprovedByID != c.ids["approver"] {
		t.Fatalf("approval: %+v", ap)
	}
	w := c.h.post(&c.approver, path+"/approval", approval(true), "")
	expect(t, w, http.StatusOK, nil)
	if w.Header().Get("Idempotent-Replayed") != "true" {
		t.Error("repeated approval not marked as a replay")
	}
	expectError(t, c.h.post(&c.approver, path+"/approval", approval(false), ""), http.StatusConflict, "IMPORT_ALREADY_APPROVED")
	if got := c.stockFingerprint(t); got != before {
		t.Fatal("approving changed stock or the batch")
	}

	// Preview permission is not commit permission.
	expectError(t, c.h.post(&c.importer, path+"/commit", nil, "commit-key-0001"), http.StatusForbidden, "FORBIDDEN")
	expectError(t, c.h.post(&c.committer, path+"/commit", nil, ""), http.StatusUnprocessableEntity, "VALIDATION_FAILED")

	// Commit.
	var res openingstock.CommitResult
	expect(t, c.h.post(&c.committer, path+"/commit", nil, "commit-key-0001"), http.StatusCreated, &res)
	if res.BatchID != b.ID || res.BatchStatus != "COMMITTED" || res.UnitsCreated != 3 || len(res.Units) != 3 || res.Replayed ||
		res.FileSHA256 != b.FileSHA256 || res.TemplateVersion != "OS-IMPORT-1" || res.ApprovalID != ap.ID ||
		res.SourceType != "OPENING_STOCK" || res.MovementType != "OPENING_BALANCE" || !strings.HasPrefix(res.Reference, "OSC-") {
		t.Fatalf("commit: %+v", res)
	}
	for i, u := range res.Units {
		if u.RowNumber != 3+i || u.VehicleUnitID == "" || u.MovementID == "" || !strings.HasPrefix(u.UnitNumber, "MAE-VEH-") {
			t.Errorf("unit %d: %+v", i, u)
		}
	}

	// Units: exactly one per line, with provenance and no invented PO/GRN history.
	rows, err := c.pool.Query(ctx, `SELECT v.opening_stock_row_number, v.status, v.chassis_number, v.motor_serial, v.location_id::text,
			v.supplier_id::text, to_char(v.received_date, 'YYYY-MM-DD'), v.unit_cost::text, v.registered_by::text,
			v.purchase_order_id IS NULL AND v.goods_receipt_id IS NULL AND v.goods_receipt_unit_id IS NULL,
			m.movement_type, m.from_status, m.to_status, m.quantity_before, m.quantity_after, m.source_type, m.source_id::text,
			m.source_reference, m.actor_id::text, e.source_type, e.source_reference
		FROM vehicle_units v
		JOIN stock_movements m ON m.vehicle_unit_id = v.id
		JOIN vehicle_unit_events e ON e.vehicle_unit_id = v.id
		WHERE v.source_type = 'OPENING_STOCK' AND v.opening_stock_batch_id = $1 ORDER BY 1`, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	type unitRow struct {
		line                                   int
		status, chassis                        string
		motor, loc, supplier                   *string
		date, cost, by                         string
		noHistory                              bool
		mtype                                  string
		from                                   *string
		to                                     string
		qb, qa                                 int
		msrc, msrcID, mref, mactor, esrc, eref string
	}
	got, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (unitRow, error) {
		var u unitRow
		return u, r.Scan(&u.line, &u.status, &u.chassis, &u.motor, &u.loc, &u.supplier, &u.date, &u.cost, &u.by, &u.noHistory,
			&u.mtype, &u.from, &u.to, &u.qb, &u.qa, &u.msrc, &u.msrcID, &u.mref, &u.mactor, &u.esrc, &u.eref)
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []struct{ status, chassis, date, cost string }{
		{"AVAILABLE", "OS-1", "2026-01-10", "90000.00"}, {"RECEIVED_PENDING_PDI", "OS-2", "2026-02-11", "91000.50"},
		{"DISPLAY", "OS-3", "2026-03-12", "92000.00"}}
	if len(got) != 3 {
		t.Fatalf("units: %+v", got)
	}
	for i, u := range got {
		w := want[i]
		if u.line != 3+i || u.status != w.status || u.chassis != w.chassis || u.date != w.date || u.cost != w.cost ||
			u.by != c.ids["committer"] || !u.noHistory || *u.loc != c.locL1 || u.mtype != "OPENING_BALANCE" || u.from != nil ||
			u.to != w.status || u.qb != 0 || u.qa != 1 || u.msrc != "OPENING_STOCK" || u.msrcID != b.ID || u.mref != b.Reference ||
			u.mactor != c.ids["committer"] || u.esrc != "OPENING_STOCK" || u.eref != b.Reference {
			t.Errorf("line %d: %+v", 3+i, u)
		}
	}
	if *got[0].motor != "OS-M1" || got[0].supplier != nil || got[1].supplier == nil || *got[1].supplier != c.ids["supplier"] {
		t.Errorf("serial/supplier provenance: %+v / %+v", got[0], got[1])
	}
	// The staged row still holds the original source values and the batch the file reference.
	var src string
	if err := c.pool.QueryRow(ctx, `SELECT r.source->>'supplier' FROM vehicle_units v JOIN opening_stock_import_rows r
		ON r.batch_id = v.opening_stock_batch_id AND r.row_number = v.opening_stock_row_number WHERE v.chassis_number = 'OS-2'`).Scan(&src); err != nil || src != "Example Supplier" {
		t.Errorf("source values: %q %v", src, err)
	}
	for action, n := range map[string]int{"opening_stock_import.approved": 1, "opening_stock_import.committed": 1} {
		if got := c.count(t, `SELECT count(*) FROM audit_events WHERE action = $1 AND entity_id = $2`, action, b.ID); got != n {
			t.Errorf("audit %s: %d", action, got)
		}
	}
	if got := c.count(t, `SELECT count(*) FROM audit_events WHERE action = 'vehicle_unit.registered'
		AND summary->>'openingStockBatchId' = $1 AND summary->>'sourceType' = 'OPENING_STOCK' AND NOT summary ? 'unitCost'`, b.ID); got != 3 {
		t.Errorf("unit audit events: %d", got)
	}

	// Replay: same key → same commit, nothing new.
	afterCommit := c.count(t, `SELECT count(*) FROM stock_movements`)
	var again openingstock.CommitResult
	w = c.h.post(&c.committer, path+"/commit", nil, "commit-key-0001")
	expect(t, w, http.StatusOK, &again)
	if !again.Replayed || again.CommitID != res.CommitID || again.UnitsCreated != 3 || w.Header().Get("Idempotent-Replayed") != "true" {
		t.Errorf("replay: %+v", again)
	}
	// Anyone else, any key: already committed.
	e = expectError(t, c.h.post(&c.committer2, path+"/commit", nil, "commit-key-0002"), http.StatusConflict, "IMPORT_ALREADY_COMMITTED")
	if !strings.Contains(e.Message, res.Reference) {
		t.Errorf("already committed message: %s", e.Message)
	}
	expectError(t, c.h.post(&c.committer, path+"/commit", nil, "commit-key-0003"), http.StatusConflict, "IMPORT_ALREADY_COMMITTED")
	// The key belongs to that commit: reusing it for another batch is refused.
	other := c.previewOf(t, validFile("OT"), "upload-key-0002")
	expectError(t, c.h.post(&c.committer, "/opening-stock-imports/"+other.ID+"/commit", nil, "commit-key-0001"), http.StatusConflict,
		"IDEMPOTENCY_KEY_REUSED")
	// Approval after commit is refused.
	expectError(t, c.h.post(&c.approver, path+"/approval", approval(false), ""), http.StatusConflict, "IMPORT_ALREADY_APPROVED")
	if got := c.stockFingerprint(t); !strings.Contains(got, "vehicle_units=4:") {
		t.Fatalf("units after retries: %s", got)
	}
	if n := c.count(t, `SELECT count(*) FROM stock_movements`); n != afterCommit {
		t.Fatalf("retries added movements: %d -> %d", afterCommit, n)
	}
	if n := c.count(t, `SELECT count(*) FROM vehicle_units WHERE opening_stock_batch_id = $1`, b.ID); n != 3 {
		t.Fatalf("units after retries: %d", n)
	}

	// Reading: batch shows approval and commit; GET commit.
	var view openingstock.Batch
	expect(t, c.h.get(&c.committer, path), http.StatusOK, &view)
	if view.Approval == nil || view.Commit == nil || view.Commit.ID != res.CommitID || view.Commit.UnitsCreated != 3 ||
		view.Status != openingstock.BatchPreviewed || view.CommitAvailable {
		t.Errorf("batch view: %+v", view)
	}
	var read openingstock.CommitResult
	expect(t, c.h.get(&c.importer, path+"/commit"), http.StatusOK, &read)
	if read.CommitID != res.CommitID || len(read.Units) != 3 {
		t.Errorf("GET commit: %+v", read)
	}
	expectError(t, c.h.get(&c.importer, "/opening-stock-imports/"+other.ID+"/commit"), http.StatusNotFound, "NOT_FOUND")
	expectError(t, c.h.get(&c.noPerm, path+"/commit"), http.StatusForbidden, "FORBIDDEN")

	// Committed history cannot be changed or removed.
	for _, sql := range []string{
		`UPDATE opening_stock_import_commits SET units_created = 4`, `DELETE FROM opening_stock_import_commits`,
		`UPDATE opening_stock_import_approvals SET received_dates_approved = false`, `DELETE FROM opening_stock_import_approvals`,
		`DELETE FROM vehicle_units WHERE source_type = 'OPENING_STOCK'`, `DELETE FROM stock_movements WHERE movement_type = 'OPENING_BALANCE'`,
	} {
		if _, err := c.pool.Exec(ctx, sql); err == nil {
			t.Errorf("%s was allowed", sql)
		}
	}
	// An opening-stock unit cannot be created outside a commit (here: the second, uncommitted batch).
	if _, err := c.pool.Exec(ctx, `INSERT INTO vehicle_units (brand_id, model_id, variant_id, status, chassis_number, location_id,
			source_type, received_date, unit_cost, registered_by, opening_stock_batch_id, opening_stock_row_number)
		VALUES ($1, $2, $3, 'AVAILABLE', 'OT-1', $4, 'OPENING_STOCK', '2026-01-10', 1, $5, $6, 3)`,
		c.ids["brandA"], c.ids["modelA"], c.ids["variantA"], c.locL1, c.ids["committer"], other.ID); err == nil {
		t.Error("an opening-stock unit was created without a commit")
	}

	// E-07: the approved received date is the unit's origin (source OPENING_STOCK).
	vsvc := vehicles.NewService(vehicles.NewPGStore(c.pool), slog.New(slog.NewTextHandler(io.Discard, nil)))
	viewer := c.loadAccess(t, c.ids["committer"])
	viewer = access.NewAccess(viewer.UserID, nil, []string{access.PermVehicleUnitView, access.PermAllLocations}, nil, false)
	d, err := vsvc.GetDetail(ctx, viewer, res.Units[0].VehicleUnitID)
	if err != nil {
		t.Fatal(err)
	}
	if d.Unit.Ageing.Status != ageing.StatusAged || d.Unit.Ageing.OriginSource == nil || *d.Unit.Ageing.OriginSource != ageing.SourceOpeningStock ||
		*d.Unit.Ageing.OriginDate != "2026-01-10" || d.Source.Type != "OPENING_STOCK" || d.Source.OpeningStockBatchID == nil ||
		*d.Source.OpeningStockBatchID != b.ID || *d.Source.OpeningStockBatchReference != b.Reference || *d.Source.OpeningStockRowNumber != 3 ||
		d.Source.PurchaseOrderID != nil || d.Source.GoodsReceiptID != nil {
		t.Errorf("vehicle detail: ageing %+v source %+v", d.Unit.Ageing, d.Source)
	}
	page, err := vsvc.List(ctx, viewer, vehicles.ListQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 4 {
		t.Errorf("vehicle list: %d units", page.Total)
	}

	// E-06: the ledger reconciles (OPENING_BALANCE is an entry movement).
	ssvc := stock.NewService(stock.NewPGStore(c.pool), slog.New(slog.NewTextHandler(io.Discard, nil)))
	rec, err := ssvc.Reconcile(ctx, access.NewAccess(viewer.UserID, nil, []string{access.PermReconciliationView, access.PermAllLocations,
		access.PermVehicleUnitView, access.PermPartsView}, nil, false), stock.ReconcileQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.Items) != 0 {
		t.Errorf("reconciliation discrepancies: %+v", rec.Items)
	}
}

func TestCommitRefusesAndCreatesNothing(t *testing.T) {
	c := setupCommit(t, commitLimits)
	ctx := context.Background()

	// A preview with any non-VALID line cannot be approved, so never committed.
	mixed := c.previewOf(t, csvFile(
		row("Brand A", "MA", "STD", "", "MX-1", "", "L1", "AVAILABLE", "2026-01-10", "Register", "", "90000"),
		row("Brand A", "MA", "STD", "", "CH-EXIST", "", "L1", "AVAILABLE", "2026-01-10", "Register", "", "90000"),
	), "upload-key-1001")
	expectError(t, c.h.post(&c.approver, "/opening-stock-imports/"+mixed.ID+"/approval", approval(true), ""), http.StatusConflict, "INVALID_STATE")
	// An unverified brand (BLOCKED) likewise.
	pending := c.previewOf(t, csvFile(row("Brand P", "PA", "STD", "", "BP-1", "", "L1", "AVAILABLE", "2026-01-10", "Register", "", "90000")),
		"upload-key-1002")
	if pending.Summary.BlockedRows != 1 {
		t.Fatalf("pending brand preview: %+v", pending.Summary)
	}
	expectError(t, c.h.post(&c.approver, "/opening-stock-imports/"+pending.ID+"/approval", approval(true), ""), http.StatusConflict, "INVALID_STATE")
	expectError(t, c.h.post(&c.committer, "/opening-stock-imports/"+pending.ID+"/commit", nil, "commit-key-1002"), http.StatusConflict,
		"IMPORT_APPROVAL_REQUIRED")
	// A rejected file.
	var rejected openingstock.Batch
	expect(t, c.h.upload(&c.importer, "bad.csv", "text/csv", []byte("brand\nx\n"), "upload-key-1003"), http.StatusCreated, &rejected)
	expectError(t, c.h.post(&c.approver, "/opening-stock-imports/"+rejected.ID+"/approval", approval(true), ""), http.StatusConflict, "INVALID_STATE")
	expectError(t, c.h.post(&c.committer, "/opening-stock-imports/"+rejected.ID+"/commit", nil, "commit-key-1003"), http.StatusConflict, "INVALID_STATE")
	expectError(t, c.h.post(&c.committer, "/opening-stock-imports/00000000-0000-0000-0000-000000000000/commit", nil, "commit-key-1004"),
		http.StatusNotFound, "NOT_FOUND")

	// A conflict that appears after preview and approval: one line fails → the whole batch is refused.
	b := c.previewOf(t, validFile("CF"), "upload-key-1010")
	c.approve(t, b.ID, true)
	path := "/opening-stock-imports/" + b.ID + "/commit"
	before := c.stockFingerprint(t)
	c.receipt("CF-2", "") // an open receipt draft now carries line 4's chassis
	e := expectError(t, c.h.post(&c.committer, path, nil, "commit-key-1010"), http.StatusUnprocessableEntity, "IMPORT_ROWS_BLOCKED")
	if len(e.Details) == 0 || e.Details[0].Field != "rows[4].chassis_number" || !strings.HasPrefix(e.Details[0].Message, "DUPLICATE_CHASSIS_ON_RECEIPT (ERROR)") {
		t.Errorf("details: %+v", e.Details)
	}
	if got := c.stockFingerprint(t); got != before {
		t.Fatal("a refused commit changed stock or the batch")
	}
	// Registered meanwhile as a vehicle unit: still refused, still nothing.
	grn, unit := c.receipt("CF-3", "")
	if err := pgx.BeginFunc(ctx, c.pool, func(tx pgx.Tx) error {
		if _, err := vehicles.RegisterFromReceiptUnit(ctx, tx, audit.UserActor(c.ids["poster"], "poster"), "req", unit, &c.locL1); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE goods_receipts SET status = 'POSTED', posted_by = $2, posted_at = now() WHERE id = $1`, grn, c.ids["poster"])
		return err
	}); err != nil {
		t.Fatal(vehicles.CommitError(err))
	}
	e = expectError(t, c.h.post(&c.committer, path, nil, "commit-key-1011"), http.StatusUnprocessableEntity, "IMPORT_ROWS_BLOCKED")
	if !detailHas(e.Details, "rows[5].chassis_number", "DUPLICATE_CHASSIS_EXISTING") || !detailHas(e.Details, "rows[4].chassis_number", "DUPLICATE_CHASSIS_ON_RECEIPT") {
		t.Errorf("details: %+v", e.Details)
	}
	if n := c.count(t, `SELECT count(*) FROM vehicle_units WHERE opening_stock_batch_id = $1`, b.ID); n != 0 {
		t.Fatalf("%d units created by a refused commit", n)
	}

	// A catalogue entry unverified after approval: never becomes stock.
	b2 := c.previewOf(t, validFile("UV"), "upload-key-1020")
	c.approve(t, b2.ID, true)
	if _, err := c.pool.Exec(ctx, `UPDATE variants SET lifecycle_status = 'PENDING_VERIFICATION' WHERE id = $1`, c.ids["variantA"]); err != nil {
		t.Fatal(err)
	}
	e = expectError(t, c.h.post(&c.committer, "/opening-stock-imports/"+b2.ID+"/commit", nil, "commit-key-1020"), http.StatusUnprocessableEntity,
		"IMPORT_ROWS_BLOCKED")
	if !detailHas(e.Details, "rows[3].variant_code", "VARIANT_NOT_ACTIVE (BLOCKED)") {
		t.Errorf("unverified variant details: %+v", e.Details)
	}
	if _, err := c.pool.Exec(ctx, `UPDATE variants SET lifecycle_status = 'ACTIVE' WHERE id = $1`, c.ids["variantA"]); err != nil {
		t.Fatal(err)
	}

	// A status the owner has not approved (QUALITY_HOLD is not in this test's configuration).
	hold := c.previewOf(t, csvFile(row("Brand A", "MA", "STD", "", "QH-1", "", "L1", "QUALITY_HOLD", "2026-01-10", "Register", "", "90000")),
		"upload-key-1030")
	c.approve(t, hold.ID, true)
	e = expectError(t, c.h.post(&c.committer, "/opening-stock-imports/"+hold.ID+"/commit", nil, "commit-key-1030"), http.StatusUnprocessableEntity,
		"IMPORT_ROWS_BLOCKED")
	if !detailHas(e.Details, "rows[3].stock_status", "STATUS_NOT_APPROVED") {
		t.Errorf("status details: %+v", e.Details)
	}
	// Warnings are not accepted (no approved rule); such a preview is not even approvable.
	warn := c.previewOf(t, csvFile(row("Brand A", "MA", "STD", "", "WN-1", "MOT-EXIST", "L1", "AVAILABLE", "2026-01-10", "Register", "", "90000")),
		"upload-key-1040")
	if warn.Summary.WarningRows != 1 {
		t.Fatalf("warning preview: %+v", warn.Summary)
	}
	expectError(t, c.h.post(&c.approver, "/opening-stock-imports/"+warn.ID+"/approval", approval(true), ""), http.StatusConflict, "INVALID_STATE")

	// Location scope: the uploader may commit only if every line's location is in scope.
	sc := c.previewOf(t, validFile("SC"), "upload-key-1050")
	c.approve(t, sc.ID, true)
	c.h.grant(c.importer, access.PermImportPreview, access.PermImportCommit)
	c.h.scope(c.importer, c.locL2)
	expectError(t, c.h.post(&c.importer, "/opening-stock-imports/"+sc.ID+"/commit", nil, "commit-key-1050"), http.StatusForbidden, "FORBIDDEN")
	// A committer without every location cannot see another user's batch.
	c.h.grant(c.committer2, access.PermImportCommit)
	c.h.scope(c.committer2, c.locL1)
	expectError(t, c.h.post(&c.committer2, "/opening-stock-imports/"+sc.ID+"/commit", nil, "commit-key-1051"), http.StatusNotFound, "NOT_FOUND")
	c.h.scope(c.importer, c.locL1)
	c.h.grant(c.importer, access.PermImportPreview)

	// The stored file must still be the approved one.
	var key string
	if err := c.pool.QueryRow(ctx, `SELECT a.object_key FROM attachments a JOIN opening_stock_import_batches b ON b.file_attachment_id = a.id
		WHERE b.id = $1`, sc.ID).Scan(&key); err != nil {
		t.Fatal(err)
	}
	if err := c.h.store.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	expectError(t, c.h.post(&c.committer, "/opening-stock-imports/"+sc.ID+"/commit", nil, "commit-key-1052"), http.StatusConflict, "IMPORT_SOURCE_INVALID")
	tampered := bytes.Replace(validFile("SC"), []byte("SC-1"), []byte("SC-9"), 1)
	if _, err := c.h.store.Put(ctx, key, bytes.NewReader(tampered)); err != nil {
		t.Fatal(err)
	}
	expectError(t, c.h.post(&c.committer, "/opening-stock-imports/"+sc.ID+"/commit", nil, "commit-key-1053"), http.StatusConflict, "IMPORT_SOURCE_INVALID")

	// Expired preview.
	ex := c.previewOf(t, validFile("EX"), "upload-key-1060")
	c.approve(t, ex.ID, true)
	openingstock.SetNow(c.h.svc, func() time.Time { return time.Now().Add(8 * 24 * time.Hour) })
	expectError(t, c.h.post(&c.committer, "/opening-stock-imports/"+ex.ID+"/commit", nil, "commit-key-1060"), http.StatusConflict,
		"IMPORT_PREVIEW_EXPIRED")
	openingstock.SetNow(c.h.svc, time.Now)

	if n := c.count(t, `SELECT count(*) FROM vehicle_units WHERE source_type = 'OPENING_STOCK'`); n != 0 {
		t.Fatalf("%d opening-stock units after refusals", n)
	}
	if n := c.count(t, `SELECT count(*) FROM opening_stock_import_commits`); n != 0 {
		t.Fatalf("%d commits after refusals", n)
	}
}

func detailHas(ds []httpx.FieldError, field, prefix string) bool {
	for _, d := range ds {
		if d.Field == field && strings.HasPrefix(d.Message, prefix) {
			return true
		}
	}
	return false
}

func TestCommitBlockedWithoutApprovedStatuses(t *testing.T) {
	// The production default: no opening status approved (OQ-47) → nothing can be committed.
	limits := commitLimits
	limits.OpeningStatuses = nil
	c := setupCommit(t, limits)
	var spec openingstock.Specification
	expect(t, c.h.get(&c.importer, "/opening-stock-imports/specification"), http.StatusOK, &spec)
	if len(spec.ApprovedOpeningStatuses) != 0 {
		t.Fatalf("approved statuses: %v", spec.ApprovedOpeningStatuses)
	}
	b := c.previewOf(t, validFile("NS"), "upload-key-2001")
	c.approve(t, b.ID, true)
	before := c.stockFingerprint(t)
	expectError(t, c.h.post(&c.committer, "/opening-stock-imports/"+b.ID+"/commit", nil, "commit-key-2001"), http.StatusConflict,
		"OPENING_STATUSES_NOT_APPROVED")
	if c.stockFingerprint(t) != before {
		t.Fatal("refused commit changed data")
	}
}

func TestCommitRollsBackOnFailureAndRetries(t *testing.T) {
	c := setupCommit(t, commitLimits)
	b := c.previewOf(t, validFile("RB"), "upload-key-3001")
	c.approve(t, b.ID, false)
	before := c.stockFingerprint(t)
	audits := c.count(t, `SELECT count(*) FROM audit_events`)
	openingstock.SetCommitFault(func(context.Context, pgx.Tx) error { return errors.New("injected failure after the units") })
	expectError(t, c.h.post(&c.committer, "/opening-stock-imports/"+b.ID+"/commit", nil, "commit-key-3001"), http.StatusInternalServerError,
		"INTERNAL_ERROR")
	openingstock.SetCommitFault(nil)
	if c.stockFingerprint(t) != before || c.count(t, `SELECT count(*) FROM audit_events`) != audits {
		t.Fatal("a failed commit left units, movements, history, audit or a commit behind")
	}
	// The same request again succeeds once the failure is gone.
	var res openingstock.CommitResult
	expect(t, c.h.post(&c.committer, "/opening-stock-imports/"+b.ID+"/commit", nil, "commit-key-3001"), http.StatusCreated, &res)
	if res.UnitsCreated != 3 || res.Replayed {
		t.Fatalf("retry: %+v", res)
	}
	// E-07: dates not approved → no age (never zero, never invented).
	vsvc := vehicles.NewService(vehicles.NewPGStore(c.pool), slog.New(slog.NewTextHandler(io.Discard, nil)))
	viewer := access.NewAccess(c.ids["committer"], nil, []string{access.PermVehicleUnitView, access.PermAllLocations}, nil, false)
	d, err := vsvc.GetDetail(context.Background(), viewer, res.Units[0].VehicleUnitID)
	if err != nil {
		t.Fatal(err)
	}
	if d.Unit.Ageing.Status != ageing.StatusMissingOrigin || d.Unit.Ageing.MissingReason == nil ||
		*d.Unit.Ageing.MissingReason != ageing.ReasonOpeningNotApproved || d.Unit.Ageing.AgeDays != nil || d.Unit.AgeDays != nil {
		t.Errorf("ageing without approved dates: %+v", d.Unit.Ageing)
	}
}

func TestConcurrentCommitsCreateOnce(t *testing.T) {
	c := setupCommit(t, commitLimits)
	b := c.previewOf(t, validFile("CC"), "upload-key-4001")
	c.approve(t, b.ID, true)
	var wg sync.WaitGroup
	type out struct {
		who  string
		code int
		body string
	}
	results := make(chan out, 10)
	start := make(chan struct{})
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			s, key, who := c.committer, "commit-key-4001", "A" // the same request retried
			if i%2 == 1 {
				s, key, who = c.committer2, "commit-key-4002-"+string(rune('a'+i)), "B" // other users, other keys
			}
			w := c.h.post(&s, "/opening-stock-imports/"+b.ID+"/commit", nil, key)
			results <- out{who, w.Code, w.Body.String()}
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)
	created := 0
	winner := ""
	var all []out
	for r := range results {
		all = append(all, r)
		if r.code == http.StatusCreated {
			created++
			winner = r.who
		}
	}
	if created != 1 {
		t.Fatalf("%d requests created stock: %+v", created, all)
	}
	for _, r := range all {
		switch {
		case r.code == http.StatusCreated:
		case r.who == winner && winner == "A" && r.code == http.StatusOK && strings.Contains(r.body, `"replayed":true`):
		case r.code == http.StatusConflict && strings.Contains(r.body, "IMPORT_ALREADY_COMMITTED"):
		default:
			t.Errorf("unexpected answer (winner %s): %+v", winner, r)
		}
	}
	if n := c.count(t, `SELECT count(*) FROM vehicle_units WHERE opening_stock_batch_id = $1`, b.ID); n != 3 {
		t.Fatalf("units: %d", n)
	}
	if n := c.count(t, `SELECT count(*) FROM stock_movements WHERE source_id = $1 AND movement_type = 'OPENING_BALANCE'`, b.ID); n != 3 {
		t.Fatalf("movements: %d", n)
	}
	if n := c.count(t, `SELECT count(*) FROM opening_stock_import_commits WHERE batch_id = $1`, b.ID); n != 1 {
		t.Fatalf("commits: %d", n)
	}
}
