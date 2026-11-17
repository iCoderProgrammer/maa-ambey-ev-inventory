package openingstock_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database/dbtest"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/openingstock"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/vehicles"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/seeds"
)

// Opening stock import preview (G-06) on a throwaway migrated database (one
// scratch database per test). Catalogue, locations, an existing vehicle unit
// and an open receipt draft are fixtures; every name and identifier is a
// placeholder. Grants are test-only (in-memory harness).

type fixture struct {
	pool                 *pgxpool.Pool
	h                    *harness
	ids                  map[string]string
	importer, other, all session
	noPerm               session
	locL1, locL2         string
	// receipt adds a goods receipt DRAFT with one accepted unit (G-07 conflict tests).
	receipt func(chassis, motor string) (grn, unit string)
}

var testLimits = openingstock.Limits{MaxBytes: 64 << 10, MaxRows: 40, PreviewTTL: 7 * 24 * time.Hour}

func setup(t *testing.T) *fixture { t.Helper(); return setupWith(t, testLimits) }

// setupWith builds the fixture with the given service limits (G-07 sets OpeningStatuses).
func setupWith(t *testing.T, limits openingstock.Limits) *fixture {
	t.Helper()
	ctx := context.Background()
	pool := dbtest.MigratedPool(t)
	if _, err := database.ApplySeeds(ctx, pool, seeds.FS); err != nil {
		t.Fatal(err)
	}
	authStore := auth.NewPGStore(pool)
	hash, err := auth.HashPassword(testPassword)
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{pool: pool, ids: map[string]string{}}
	for _, u := range []string{"importer", "other", "allloc", "noperm", "poster", "approver", "committer", "committer2"} {
		created, err := authStore.CreateUser(ctx, audit.CLIActor("test"), u, u, nil, hash)
		if err != nil {
			t.Fatal(err)
		}
		f.ids[u] = created.ID
	}
	f.h = newHarness(t, authStore, openingstock.NewPGStore(pool), limits)
	f.importer = f.h.login("importer", f.ids["importer"])
	f.other = f.h.login("other", f.ids["other"])
	f.all = f.h.login("allloc", f.ids["allloc"])
	f.noPerm = f.h.login("noperm", f.ids["noperm"])

	sc := func(sql string, args ...any) string {
		var s string
		if err := pool.QueryRow(ctx, sql, args...).Scan(&s); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
		return s
	}
	f.locL1 = sc(`INSERT INTO locations (code, name) VALUES ('L1', 'Location One') RETURNING id::text`)
	f.locL2 = sc(`INSERT INTO locations (code, name) VALUES ('L2', 'Location Two') RETURNING id::text`)
	sc(`INSERT INTO locations (code, name, active) VALUES ('L9', 'Closed', false) RETURNING id::text`)
	f.h.grant(f.importer, access.PermImportPreview)
	f.h.scope(f.importer, f.locL1)
	f.h.grant(f.other, access.PermImportPreview)
	f.h.scope(f.other, f.locL1)
	f.h.grant(f.all, access.PermImportPreview, access.PermAllLocations)

	approver := f.ids["poster"]
	brand := func(name, status, auth string) string {
		return sc(`INSERT INTO brands (name, slug, lifecycle_status, authorisation_status, manufacturer_legal_name, authorisation_reference,
			verified_by, verified_at) VALUES ($1, lower(replace($1, ' ', '-')), $2, $3, 'Example Manufacturer', 'TEST-AUTH', $4, now())
			RETURNING id::text`, name, status, auth, approver)
	}
	model := func(b, code, status string) string {
		return sc(`INSERT INTO models (brand_id, name, internal_code, lifecycle_status, confidence, source_url, source_date, verified_by, verified_at)
			VALUES ($1, 'Model '||$2, $2, $3, 'SUPPLIER_CONFIRMED', 'https://example.com/m', current_date, $4, now()) RETURNING id::text`,
			b, code, status, approver)
	}
	variant := func(b, m, code, status string) string {
		return sc(`INSERT INTO variants (brand_id, model_id, name, variant_code, lifecycle_status, confidence, source_url, source_date, verified_by, verified_at)
			VALUES ($1, $2, 'Variant '||$3, $3, $4, 'SUPPLIER_CONFIRMED', 'https://example.com/v', current_date, $5, now()) RETURNING id::text`,
			b, m, code, status, approver)
	}
	bA := brand("Brand A", "ACTIVE", "AUTHORISED")
	mA := model(bA, "MA", "ACTIVE")
	vA := variant(bA, mA, "STD", "ACTIVE")
	variant(bA, mA, "PRO", "PENDING_VERIFICATION")
	f.ids["brandA"], f.ids["variantA"] = bA, vA
	sc(`INSERT INTO variant_colours (brand_id, variant_id, name, colour_code) VALUES ($1, $2, 'Red', 'RD') RETURNING id::text`, bA, vA)
	// A brand that exists but is pending verification (as the real launch brands are today).
	bP := sc(`INSERT INTO brands (name, slug) VALUES ('Brand P', 'brand-p') RETURNING id::text`)
	mP := sc(`INSERT INTO models (brand_id, name, internal_code) VALUES ($1, 'Model PA', 'PA') RETURNING id::text`, bP)
	sc(`INSERT INTO variants (brand_id, model_id, name, variant_code) VALUES ($1, $2, 'Variant PA', 'STD') RETURNING id::text`, bP, mP)
	supplier := sc(`INSERT INTO suppliers (legal_name, created_by) VALUES ('Example Supplier', $1) RETURNING id::text`, approver)
	f.ids["supplier"] = supplier

	// An existing vehicle unit (chassis CH-EXIST, motor MOT-EXIST) and an open receipt draft (chassis CH-DRAFT).
	po := sc(`INSERT INTO purchase_orders (supplier_id, created_by, expected_date) VALUES ($1, $2, current_date + 7) RETURNING id::text`, supplier, approver)
	line := sc(`INSERT INTO purchase_order_lines (purchase_order_id, line_no, brand_id, model_id, variant_id, quantity, unit_cost)
		VALUES ($1, 1, $2, $3, $4, 10, 95000.00) RETURNING id::text`, po, bA, mA, vA)
	if _, err := pool.Exec(ctx, `UPDATE purchase_orders SET status = 'APPROVED', submitted_by = $2, submitted_at = now(), approval_basis = 'APPROVER',
		approved_by = $3, approved_at = now() WHERE id = $1`, po, approver, f.ids["other"]); err != nil {
		t.Fatal(err)
	}
	receipt := func(chassis, motor string) (string, string) {
		grn := sc(`INSERT INTO goods_receipts (purchase_order_id, received_date, supplier_reference, created_by)
			VALUES ($1, DATE '2026-01-15', 'DC-1', $2) RETURNING id::text`, po, approver)
		gl := sc(`INSERT INTO goods_receipt_lines (goods_receipt_id, purchase_order_id, purchase_order_line_id, received_quantity,
			accepted_quantity, rejected_quantity) VALUES ($1, $2, $3, 1, 1, 0) RETURNING id::text`, grn, po, line)
		unit := sc(`INSERT INTO goods_receipt_units (goods_receipt_id, goods_receipt_line_id, unit_no, condition, chassis_number,
			chassis_normalised, motor_serial) VALUES ($1, $2, 1, 'ACCEPTED', $3::text, upper($3::text), NULLIF($4, '')) RETURNING id::text`, grn, gl, chassis, motor)
		return grn, unit
	}
	grn, unit := receipt("CH-EXIST", "MOT-EXIST")
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if _, err := vehicles.RegisterFromReceiptUnit(ctx, tx, audit.UserActor(approver, "poster"), "req-test", unit, &f.locL1); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE goods_receipts SET status = 'POSTED', posted_by = $2, posted_at = now() WHERE id = $1`, grn, approver)
		return err
	})
	if err != nil {
		t.Fatal(vehicles.CommitError(err))
	}
	receipt("CH-DRAFT", "")
	f.receipt = receipt
	f.ids["po"], f.ids["poLine"], f.ids["modelA"] = po, line, mA
	return f
}

const hdr = "brand,model_code,variant_code,colour,chassis_number,motor_serial,battery_serial,charger_serial,location_code," +
	"stock_status,received_date,provenance,supplier,purchase_order_reference,goods_receipt_reference,supplier_invoice_reference," +
	"acquisition_cost,notes"

func csvFile(rows ...string) []byte {
	return []byte("#template:OS-IMPORT-1\n" + hdr + "\n" + strings.Join(rows, "\n") + "\n")
}

// row builds a template line: brand,model,variant,colour,chassis,motor,location,status,date,provenance,supplier,cost.
func row(brand, model, variant, colour, chassis, motor, location, status, date, provenance, supplier, cost string) string {
	return strings.Join([]string{brand, model, variant, colour, chassis, motor, "", "", location, status, date, provenance, supplier,
		"", "", "", cost, ""}, ",")
}

func (f *fixture) count(t *testing.T, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// businessFingerprint digests every table a preview must never change.
func (f *fixture) businessFingerprint(t *testing.T) string {
	t.Helper()
	var parts []string
	for _, table := range []string{"vehicle_units", "vehicle_unit_events", "stock_movements", "stock_item_balances", "goods_receipts",
		"goods_receipt_units", "purchase_orders", "purchase_order_lines", "brands", "models", "variants", "variant_colours", "locations",
		"suppliers", "stock_ageing_band_sets"} {
		var s string
		if err := f.pool.QueryRow(context.Background(), `SELECT count(*)::text || ':' || coalesce(md5(string_agg(t::text, '|' ORDER BY t::text)), '')
			FROM `+table+` t`).Scan(&s); err != nil {
			t.Fatal(err)
		}
		parts = append(parts, table+"="+s)
	}
	return strings.Join(parts, " ")
}

func (f *fixture) rows(t *testing.T, s session, id, query string) openingstock.RowPage {
	t.Helper()
	var p openingstock.RowPage
	if !strings.Contains(query, "limit=") {
		query = "limit=200&" + query
	}
	expect(t, f.h.get(&s, "/opening-stock-imports/"+id+"/rows?"+query), http.StatusOK, &p)
	return p
}

func byLine(p openingstock.RowPage) map[int]openingstock.Row {
	m := map[int]openingstock.Row{}
	for _, r := range p.Items {
		m[r.RowNumber] = r
	}
	return m
}

func codesOf(r openingstock.Row) []string {
	out := []string{}
	for _, i := range r.Issues {
		out = append(out, i.Field+":"+i.Code)
	}
	return out
}

// #1, #2, #10-#21, #28: template, a mixed file, row-level results mapped to
// lines and fields, batch metadata, and no change to any business table (#21).
func TestPreviewValidatesEveryRowAndChangesNoStock(t *testing.T) {
	f := setup(t)
	// Template and specification.
	w := f.h.get(&f.importer, "/opening-stock-imports/template")
	if w.Code != http.StatusOK || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/csv") ||
		!strings.Contains(w.Header().Get("Content-Disposition"), "opening-stock-template-OS-IMPORT-1.csv") ||
		w.Body.String() != string(openingstock.TemplateCSV()) {
		t.Fatalf("template %d %v %q", w.Code, w.Header(), w.Body)
	}
	var spec openingstock.Specification
	expect(t, f.h.get(&f.importer, "/opening-stock-imports/specification"), http.StatusOK, &spec)
	if spec.TemplateVersion != "OS-IMPORT-1" || spec.MaxRows != 40 || len(spec.Columns) != 18 || spec.BusinessDate != "2026-06-30" {
		t.Fatalf("spec %+v", spec)
	}

	before := f.businessFingerprint(t)
	data := csvFile(
		row("Brand A", "MA", "STD", "", " ch-new-1 ", "MOT-1", "L1", "AVAILABLE", "2026-01-15", "Stock register p.1", "Example Supplier", "95000"), // 3 VALID
		row("Brand A", "MA", "STD", "", "CH-NEW-2", "", "L1", "available", "", "Stock register p.1", "", ""),                                       // 4 WARNING (no date, no cost)
		row("Brand A", "MA", "STD", "", "CH-EXIST", "", "L1", "AVAILABLE", "2026-01-15", "Register", "", "1"),                                      // 5 dup existing
		row("Brand A", "MA", "STD", "", "CH-DRAFT", "", "L1", "AVAILABLE", "2026-01-15", "Register", "", "1"),                                      // 6 dup draft
		row("Brand A", "MA", "STD", "", "CH-TWIN", "MOT-EXIST", "L1", "AVAILABLE", "2026-01-15", "Register", "", "1"),                              // 7 twin + motor existing
		row("Brand A", "MA", "STD", "", "ch-twin", "", "L1", "AVAILABLE", "2026-01-15", "Register", "", "1"),                                       // 8 twin
		row("Brand Q", "MA", "STD", "", "CH-NEW-3", "", "L1", "AVAILABLE", "2026-01-15", "Register", "", "1"),                                      // 9 unknown brand
		row("Brand P", "PA", "STD", "", "CH-NEW-4", "", "L1", "AVAILABLE", "2026-01-15", "Register", "", "1"),                                      // 10 pending brand
		row("Brand A", "ZZ", "STD", "", "CH-NEW-5", "", "L1", "AVAILABLE", "2026-01-15", "Register", "", "1"),                                      // 11 unknown model
		row("Brand A", "PA", "STD", "", "CH-NEW-6", "", "L1", "AVAILABLE", "2026-01-15", "Register", "", "1"),                                      // 12 model of other brand
		row("Brand A", "MA", "PRO", "", "CH-NEW-7", "", "L1", "AVAILABLE", "2026-01-15", "Register", "", "1"),                                      // 13 pending variant
		row("Brand A", "MA", "STD", "", "CH-NEW-8", "", "L2", "AVAILABLE", "2026-01-15", "Register", "", "1"),                                      // 14 out of scope
		row("Brand A", "MA", "STD", "", "CH-NEW-9", "", "L9", "AVAILABLE", "2026-01-15", "Register", "", "1"),                                      // 15 inactive location
		row("Brand A", "MA", "STD", "", "CH-NEW-10", "", "L1", "SOLD", "2026-07-01", "", "", "abc"),                                                // 16 status, date, provenance, cost
		row("Brand A", "MA", "STD", "rd", "CH-NEW-11", "", "L1", "DISPLAY", "2026-01-15", "Register", "", "1"),                                     // 17 pending colour (by code)
	)
	var b openingstock.Batch
	expect(t, f.h.upload(&f.importer, "opening-stock.csv", "text/csv", data, "upload-0001"), http.StatusCreated, &b)
	sum := sha256.Sum256(data)
	s := b.Summary
	if b.Status != openingstock.BatchPreviewed || b.FileSHA256 != hex.EncodeToString(sum[:]) || b.FileSizeBytes != int64(len(data)) ||
		b.TemplateVersion == nil || *b.TemplateVersion != "OS-IMPORT-1" || b.UploadedByID != f.ids["importer"] || b.BusinessDate != "2026-06-30" ||
		b.Expired || b.CommitAvailable || !strings.HasPrefix(b.Reference, "OSI-") || len(b.FileIssues) != 0 {
		t.Fatalf("batch %+v", b)
	}
	if s.TotalRows != 15 || s.ValidRows != 1 || s.WarningRows != 1 || s.BlockedRows != 3 || s.ErrorRows != 10 || s.CanProceedRows != 2 ||
		s.DuplicateRows != 4 || s.UnresolvedRows != 8 {
		t.Fatalf("summary %+v", s)
	}
	rows := byLine(f.rows(t, f.importer, b.ID, ""))
	want := map[int]struct {
		status string
		codes  []string
	}{
		3:  {openingstock.RowValid, []string{}},
		4:  {openingstock.RowWarning, []string{"received_date:ORIGIN_DATE_MISSING", "acquisition_cost:COST_MISSING"}},
		5:  {openingstock.RowError, []string{"chassis_number:DUPLICATE_CHASSIS_EXISTING"}},
		6:  {openingstock.RowError, []string{"chassis_number:DUPLICATE_CHASSIS_ON_RECEIPT"}},
		7:  {openingstock.RowError, []string{"chassis_number:DUPLICATE_CHASSIS_IN_FILE", "motor_serial:POSSIBLE_DUPLICATE_SERIAL_EXISTING"}},
		8:  {openingstock.RowError, []string{"chassis_number:DUPLICATE_CHASSIS_IN_FILE"}},
		9:  {openingstock.RowError, []string{"brand:UNKNOWN_BRAND"}},
		10: {openingstock.RowBlocked, []string{"brand:BRAND_NOT_ACTIVE", "model_code:MODEL_NOT_ACTIVE", "variant_code:VARIANT_NOT_ACTIVE"}},
		11: {openingstock.RowError, []string{"model_code:UNKNOWN_MODEL"}},
		12: {openingstock.RowError, []string{"model_code:MODEL_NOT_IN_BRAND"}},
		13: {openingstock.RowBlocked, []string{"variant_code:VARIANT_NOT_ACTIVE"}},
		14: {openingstock.RowError, []string{"location_code:LOCATION_OUT_OF_SCOPE"}},
		15: {openingstock.RowError, []string{"location_code:LOCATION_INACTIVE"}},
		16: {openingstock.RowError, []string{"provenance:REQUIRED", "stock_status:INVALID_STATUS", "received_date:FUTURE_DATE", "acquisition_cost:INVALID_COST"}},
		17: {openingstock.RowBlocked, []string{"colour:COLOUR_NOT_ACTIVE"}},
	}
	for line, w := range want {
		r := rows[line]
		if r.Status != w.status || !slices.Equal(codesOf(r), w.codes) {
			t.Errorf("line %d: %s %v, want %s %v", line, r.Status, codesOf(r), w.status, w.codes)
		}
	}
	// Row 3 is resolved and normalised for G-07, and keeps the values as uploaded.
	v := rows[3]
	n := v.Normalised
	if n.BrandID == nil || *n.BrandID != f.ids["brandA"] || *n.VariantID != f.ids["variantA"] || n.ColourID != nil || *n.LocationID != f.locL1 ||
		*n.SupplierID != f.ids["supplier"] || *n.ChassisNumber != "ch-new-1" || *n.ChassisNormalised != "CH-NEW-1" || *n.StockStatus != "AVAILABLE" ||
		*n.ReceivedDate != "2026-01-15" || *n.AcquisitionCost != "95000.00" || v.Source["chassis_number"] != " ch-new-1 " || v.Resolved["brand"] == nil {
		t.Errorf("row 3 %+v", v)
	}
	// The file is stored privately as the batch's attachment, byte for byte.
	var key, owner, ct string
	if err := f.pool.QueryRow(context.Background(), `SELECT object_key, owner_type || ':' || owner_id, content_type FROM attachments WHERE id = $1`,
		b.FileAttachmentID).Scan(&key, &owner, &ct); err != nil {
		t.Fatal(err)
	}
	rc, err := f.h.store.Open(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	stored, _ := io.ReadAll(rc)
	_ = rc.Close()
	if string(stored) != string(data) || owner != "opening_stock_import:"+b.ID || ct != "text/csv" {
		t.Fatalf("stored file: owner %s type %s same=%v", owner, ct, string(stored) == string(data))
	}
	if n := f.count(t, `SELECT count(*) FROM audit_events WHERE action = 'opening_stock_import.previewed' AND entity_id = $1`, b.ID); n != 1 {
		t.Fatalf("audit %d", n)
	}
	// #21: nothing outside the import tables, the attachment and the audit log changed.
	if after := f.businessFingerprint(t); after != before {
		t.Fatalf("preview changed business data:\n%s\n%s", before, after)
	}
	// Filters.
	if p := f.rows(t, f.importer, b.ID, "status=ERROR"); p.Total != 10 {
		t.Errorf("ERROR rows %d", p.Total)
	}
	if p := f.rows(t, f.importer, b.ID, "duplicate=true"); p.Total != 4 {
		t.Errorf("duplicate rows %d", p.Total)
	}
	if p := f.rows(t, f.importer, b.ID, "unresolved=true"); p.Total != 8 {
		t.Errorf("unresolved rows %d", p.Total)
	}
	if p := f.rows(t, f.importer, b.ID, "status=BLOCKED_UNVERIFIED_MAPPING&page=2&limit=1"); p.Total != 3 || len(p.Items) != 1 || p.Items[0].RowNumber != 13 {
		t.Errorf("paged blocked %+v", p)
	}
	for _, q := range []string{"status=OK", "duplicate=maybe", "limit=0", "page=-1"} {
		expectError(t, f.h.get(&f.importer, "/opening-stock-imports/"+b.ID+"/rows?"+q), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	}
	// The database refuses changes to a preview.
	for _, sql := range []string{`UPDATE opening_stock_import_rows SET status = 'VALID'`, `DELETE FROM opening_stock_import_batches`} {
		if _, err := f.pool.Exec(context.Background(), sql); err == nil {
			t.Errorf("database accepted %s", sql)
		}
	}
}

// #3-#9: file-level refusals. Unreadable files store nothing; a CSV that is not
// the template is kept as a REJECTED batch with its issues and no rows.
func TestFileLevelRefusals(t *testing.T) {
	f := setup(t)
	valid := csvFile(row("Brand A", "MA", "STD", "", "CH-1", "", "L1", "AVAILABLE", "2026-01-15", "Register", "", "1"))
	batches := func() int { return f.count(t, `SELECT count(*) FROM opening_stock_import_batches`) }
	files := func() int { return f.count(t, `SELECT count(*) FROM attachments`) }
	b0, a0 := batches(), files()
	for name, c := range map[string]struct {
		file, ct string
		data     []byte
		status   int
		code     string
	}{
		"xlsx":         {"stock.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", append([]byte("PK\x03\x04"), make([]byte, 64)...), 415, "UNSUPPORTED_MEDIA_TYPE"},
		"zip as csv":   {"stock.csv", "text/csv", append([]byte("PK\x03\x04"), make([]byte, 64)...), 415, "UNSUPPORTED_MEDIA_TYPE"},
		"pdf":          {"stock.pdf", "application/pdf", []byte("%PDF-1.4\n%..."), 415, "UNSUPPORTED_MEDIA_TYPE"},
		"txt name":     {"stock.txt", "text/plain", valid, 415, "UNSUPPORTED_MEDIA_TYPE"},
		"declared img": {"stock.csv", "image/png", valid, 415, "UNSUPPORTED_MEDIA_TYPE"},
		"empty":        {"stock.csv", "text/csv", nil, 422, "VALIDATION_FAILED"},
		"too large":    {"stock.csv", "text/csv", append(valid, []byte(strings.Repeat("x", 70<<10))...), 413, "FILE_TOO_LARGE"},
	} {
		expectError(t, f.h.upload(&f.importer, c.file, c.ct, c.data, "refuse-"+strings.ReplaceAll(name, " ", "-")), c.status, c.code)
	}
	expectError(t, f.h.upload(&f.importer, "stock.csv", "text/csv", valid, ""), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if batches() != b0 || files() != a0 {
		t.Fatal("a refused upload stored something")
	}
	for name, c := range map[string]struct {
		data []byte
		code string
	}{
		"missing columns":   {[]byte("#template:OS-IMPORT-1\nbrand,model_code\nA,B\n"), openingstock.FileMissingColumn},
		"duplicate columns": {[]byte("#template:OS-IMPORT-1\n" + hdr + ",notes\n"), openingstock.FileDuplicateColumn},
		"unknown column":    {[]byte("#template:OS-IMPORT-1\n" + hdr + ",price\n"), openingstock.FileUnknownColumn},
		"no marker":         {[]byte(hdr + "\n"), openingstock.FileMarkerMissing},
		"no rows":           {[]byte("#template:OS-IMPORT-1\n" + hdr + "\n"), openingstock.FileNoRows},
		"too many rows":     {csvFile(slices.Repeat([]string{row("Brand A", "MA", "STD", "", "CH-1", "", "L1", "AVAILABLE", "", "R", "", "")}, 41)...), openingstock.FileTooManyRows},
	} {
		var b openingstock.Batch
		expect(t, f.h.upload(&f.importer, "stock.csv", "text/csv", c.data, "reject-"+strings.ReplaceAll(name, " ", "-")), http.StatusCreated, &b)
		if b.Status != openingstock.BatchRejected || b.Summary.TotalRows != 0 || len(b.FileIssues) == 0 || b.FileIssues[0].Code != c.code {
			t.Errorf("%s: %+v", name, b)
		}
		if p := f.rows(t, f.importer, b.ID, ""); p.Total != 0 {
			t.Errorf("%s: rows staged", name)
		}
	}
}

// #22-#25: permissions, visibility and expiry.
func TestPermissionsVisibilityAndExpiry(t *testing.T) {
	f := setup(t)
	data := csvFile(row("Brand A", "MA", "STD", "", "CH-1", "", "L1", "AVAILABLE", "2026-01-15", "Register", "", "1"))
	for _, p := range []string{"/opening-stock-imports/template", "/opening-stock-imports/specification", "/opening-stock-imports"} {
		if w := f.h.get(nil, p); w.Code != http.StatusUnauthorized {
			t.Errorf("%s without session: %d", p, w.Code)
		}
		expectError(t, f.h.get(&f.noPerm, p), http.StatusForbidden, "FORBIDDEN")
	}
	expectError(t, f.h.upload(&f.noPerm, "s.csv", "text/csv", data, "noperm-0001"), http.StatusForbidden, "FORBIDDEN")
	var b openingstock.Batch
	expect(t, f.h.upload(&f.importer, "s.csv", "text/csv", data, "vis-00001"), http.StatusCreated, &b)
	// Another importer does not see it (404, not 403); a user with every location does.
	expectError(t, f.h.get(&f.other, "/opening-stock-imports/"+b.ID), http.StatusNotFound, "NOT_FOUND")
	expectError(t, f.h.get(&f.other, "/opening-stock-imports/"+b.ID+"/rows"), http.StatusNotFound, "NOT_FOUND")
	expectError(t, f.h.get(&f.noPerm, "/opening-stock-imports/"+b.ID), http.StatusForbidden, "FORBIDDEN")
	expectError(t, f.h.get(&f.importer, "/opening-stock-imports/not-a-uuid"), http.StatusNotFound, "NOT_FOUND")
	var got openingstock.Batch
	expect(t, f.h.get(&f.all, "/opening-stock-imports/"+b.ID), http.StatusOK, &got)
	var mine openingstock.BatchPage
	expect(t, f.h.get(&f.other, "/opening-stock-imports"), http.StatusOK, &mine)
	if mine.Total != 0 {
		t.Fatalf("other lists %d", mine.Total)
	}
	expect(t, f.h.get(&f.all, "/opening-stock-imports"), http.StatusOK, &mine)
	if mine.Total != 1 {
		t.Fatalf("all-locations lists %d", mine.Total)
	}
	// Expiry: a preview made 8 days ago (7-day lifetime) is history only.
	openingstock.SetNow(f.h.svc, func() time.Time { return time.Now().Add(-8 * 24 * time.Hour) })
	var old openingstock.Batch
	expect(t, f.h.upload(&f.importer, "s.csv", "text/csv", data, "expired-0001"), http.StatusCreated, &old)
	if !old.Expired || old.CommitAvailable || b.Expired {
		t.Fatalf("expiry: old %v, new %v", old.Expired, b.Expired)
	}
}

// #26, #27: the same key replays (also when sent concurrently); another key is a new preview.
func TestIdempotentAndConcurrentUploads(t *testing.T) {
	f := setup(t)
	data := csvFile(row("Brand A", "MA", "STD", "", "CH-1", "", "L1", "AVAILABLE", "2026-01-15", "Register", "", "1"))
	results := make([]int, 8)
	ids := make([]string, 8)
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Go(func() {
			w := f.h.upload(&f.importer, "s.csv", "text/csv", data, "concurrent-1")
			results[i] = w.Code
			var b openingstock.Batch
			if w.Code < 300 {
				expect(t, w, w.Code, &b)
				ids[i] = b.ID
			}
		})
	}
	wg.Wait()
	created := 0
	for i, c := range results {
		if c == http.StatusCreated {
			created++
		} else if c != http.StatusOK {
			t.Errorf("request %d: %d", i, c)
		}
		if ids[i] != ids[0] {
			t.Errorf("request %d got batch %s, want %s", i, ids[i], ids[0])
		}
	}
	if created != 1 || f.count(t, `SELECT count(*) FROM opening_stock_import_batches`) != 1 || f.count(t, `SELECT count(*) FROM attachments`) != 1 {
		t.Fatalf("created %d", created)
	}
	w := f.h.upload(&f.importer, "s.csv", "text/csv", data, "concurrent-1")
	if w.Code != http.StatusOK || w.Header().Get("Idempotent-Replayed") != "true" {
		t.Fatalf("replay %d %v", w.Code, w.Header())
	}
	changed := append([]byte{}, data...)
	changed = append(changed, []byte(row("Brand A", "MA", "STD", "", "CH-2", "", "L1", "AVAILABLE", "", "R", "", "")+"\n")...)
	expectError(t, f.h.upload(&f.importer, "s.csv", "text/csv", changed, "concurrent-1"), http.StatusConflict, "IDEMPOTENCY_KEY_REUSED")
	// A corrected file with a new key is a new preview; the first one stays as it was.
	var second openingstock.Batch
	expect(t, f.h.upload(&f.importer, "s.csv", "text/csv", changed, "corrected-1"), http.StatusCreated, &second)
	if second.ID == ids[0] || second.Summary.TotalRows != 2 {
		t.Fatalf("second %+v", second)
	}
	if n := f.count(t, `SELECT count(*) FROM opening_stock_import_rows WHERE batch_id = $1`, ids[0]); n != 1 {
		t.Fatalf("first batch rows %d", n)
	}
	_ = strconv.Itoa
}
