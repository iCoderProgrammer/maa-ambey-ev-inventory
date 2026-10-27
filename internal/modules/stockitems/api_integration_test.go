package stockitems

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database/dbtest"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/seeds"
)

// SKU master tests on a throwaway migrated database. Names and codes are
// placeholders ("Brand A", "Example Supplier One", "BRK-PAD-01"): no real part,
// supplier, compatibility or unit of measure is invented.

type fixture struct {
	pool      *pgxpool.Pool
	h         *harness
	manager   session // view + manage + archive + suppliers.view + catalogue.view
	viewer    session // stock_items.view only
	editor    session // view + manage, no archive, no supplier or catalogue view
	ids       map[string]string
	brandA    string
	brandB    string
	modelA    string
	modelB    string
	archived  struct{ supplier, brand, model string }
	supplier  string
	supplier2 string
}

func setup(t *testing.T) *fixture {
	t.Helper()
	return setupOn(t, dbtest.MigratedPool(t))
}

func setupOn(t *testing.T, pool *pgxpool.Pool) *fixture {
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
	f := &fixture{pool: pool, ids: map[string]string{}}
	for _, u := range []string{"manager", "viewer", "editor", "approver"} {
		created, err := store.CreateUser(ctx, audit.CLIActor("test"), u, u, nil, hash)
		if err != nil {
			t.Fatal(err)
		}
		f.ids[u] = created.ID
	}
	f.h = newHarness(t, store, NewPGStore(pool))
	f.manager = f.h.login("manager", f.ids["manager"])
	f.viewer = f.h.login("viewer", f.ids["viewer"])
	f.editor = f.h.login("editor", f.ids["editor"])
	f.h.grant(f.manager, access.PermStockItemView, access.PermStockItemManage, access.PermStockItemArchive,
		access.PermSupplierView, access.PermCatalogueView)
	f.h.grant(f.viewer, access.PermStockItemView)
	f.h.grant(f.editor, access.PermStockItemView, access.PermStockItemManage)

	brand := func(name, status string) string {
		return f.scalar(t, `INSERT INTO brands (name, slug, lifecycle_status) VALUES ($1, lower(replace($1, ' ', '-')), $2) RETURNING id::text`, name, status)
	}
	model := func(b, code, status string) string {
		return f.scalar(t, `INSERT INTO models (brand_id, name, internal_code, lifecycle_status) VALUES ($1, 'Model '||$2, $2, $3) RETURNING id::text`,
			b, code, status)
	}
	f.brandA, f.brandB = brand("Brand A", "PENDING_VERIFICATION"), brand("Brand B", "PENDING_VERIFICATION")
	f.modelA, f.modelB = model(f.brandA, "MA", "PENDING_VERIFICATION"), model(f.brandB, "MB", "PENDING_VERIFICATION")
	f.archived.brand = brand("Brand Old", "ARCHIVED")
	f.archived.model = model(f.brandA, "MOLD", "ARCHIVED")
	supplier := func(name string) string {
		return f.scalar(t, `INSERT INTO suppliers (legal_name, created_by) VALUES ($1, $2) RETURNING id::text`, name, f.ids["approver"])
	}
	f.supplier, f.supplier2 = supplier("Example Supplier One"), supplier("Example Supplier Two")
	f.archived.supplier = supplier("Example Supplier Old")
	f.exec(t, `UPDATE suppliers SET status = 'ARCHIVED' WHERE id = $1`, f.archived.supplier)
	return f
}

func (f *fixture) exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) scalar(t *testing.T, sql string, args ...any) string {
	t.Helper()
	var s string
	if err := f.pool.QueryRow(context.Background(), sql, args...).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

func (f *fixture) count(t *testing.T, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func (f *fixture) audits(t *testing.T, action string) int {
	return f.count(t, `SELECT count(*) FROM audit_events WHERE action = $1`, action)
}

func (f *fixture) category(t *testing.T, name string) Category {
	t.Helper()
	var c Category
	expect(t, f.h.call(f.manager, http.MethodPost, "/stock-item-categories", map[string]any{"name": name}), http.StatusCreated, &c)
	return c
}

func (f *fixture) item(t *testing.T, body map[string]any) Detail {
	t.Helper()
	var d Detail
	expect(t, f.h.call(f.manager, http.MethodPost, "/stock-items", body), http.StatusCreated, &d)
	return d
}

func itemBody(sku, category string, extra ...any) map[string]any {
	b := map[string]any{"sku": sku, "name": "Item " + sku, "categoryId": category, "uom": "PCS", "trackingMode": TrackingQuantity}
	for i := 0; i+1 < len(extra); i += 2 {
		b[extra[i].(string)] = extra[i+1]
	}
	return b
}

func sqlState(err error) (code, constraint string) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code, pgErr.ConstraintName
	}
	return "", ""
}

// ── Tests ───────────────────────────────────────────────────────────────────

func TestCategoriesAreUniqueEditableAndArchivedSafely(t *testing.T) {
	f := setup(t)
	c := f.category(t, " Brake parts ")
	if c.Name != "Brake parts" || c.Status != StatusActive || c.Version != 1 || c.CreatedBy != f.ids["manager"] {
		t.Fatalf("category %+v", c)
	}
	e := expectError(t, f.h.call(f.manager, http.MethodPost, "/stock-item-categories", map[string]any{"name": "BRAKE PARTS"}),
		http.StatusConflict, "DUPLICATE")
	if !hasField(e.Details, "name") {
		t.Fatalf("details %+v", e.Details)
	}
	var u Category
	expect(t, f.h.call(f.manager, http.MethodPatch, "/stock-item-categories/"+c.ID, map[string]any{"version": 1, "name": "Brakes",
		"description": "Pads and shoes"}), http.StatusOK, &u)
	if u.Name != "Brakes" || *u.Description != "Pads and shoes" || u.Version != 2 {
		t.Fatalf("updated %+v", u)
	}
	expectError(t, f.h.call(f.manager, http.MethodPatch, "/stock-item-categories/"+c.ID, map[string]any{"version": 1, "name": "X"}),
		http.StatusConflict, "VERSION_CONFLICT")
	// A category with an active SKU cannot be archived.
	d := f.item(t, itemBody("BRK-01", c.ID))
	expectError(t, f.h.call(f.manager, http.MethodPost, "/stock-item-categories/"+c.ID+"/archive", map[string]any{"version": 2, "reason": "Unused"}),
		http.StatusConflict, "INVALID_STATE")
	expect(t, f.h.call(f.manager, http.MethodPost, "/stock-items/"+d.ID+"/archive", map[string]any{"version": 1, "reason": "Discontinued"}),
		http.StatusOK, nil)
	var a Category
	expect(t, f.h.call(f.manager, http.MethodPost, "/stock-item-categories/"+c.ID+"/archive", map[string]any{"version": 2, "reason": "Unused"}),
		http.StatusOK, &a)
	if a.Status != StatusArchived || *a.ArchivedBy != f.ids["manager"] || *a.ArchiveReason != "Unused" || a.ArchivedAt == nil {
		t.Fatalf("archived %+v", a)
	}
	// Archived: no change, no new SKUs, name stays taken; the archived SKU keeps its category.
	expectError(t, f.h.call(f.manager, http.MethodPatch, "/stock-item-categories/"+c.ID, map[string]any{"version": 3, "name": "Y"}),
		http.StatusConflict, "INVALID_STATE")
	expectError(t, f.h.call(f.manager, http.MethodPost, "/stock-item-categories/"+c.ID+"/archive", map[string]any{"version": 3, "reason": "again"}),
		http.StatusConflict, "INVALID_STATE")
	e = expectError(t, f.h.call(f.manager, http.MethodPost, "/stock-items", itemBody("BRK-02", c.ID)), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if !hasField(e.Details, "categoryId") {
		t.Fatalf("details %+v", e.Details)
	}
	expectError(t, f.h.call(f.manager, http.MethodPost, "/stock-item-categories", map[string]any{"name": "brakes"}), http.StatusConflict, "DUPLICATE")
	var got Detail
	expect(t, f.h.call(f.viewer, http.MethodGet, "/stock-items/"+d.ID, nil), http.StatusOK, &got)
	if got.CategoryID != c.ID || got.CategoryStatus != StatusArchived {
		t.Fatalf("archived item's category %+v", got.Item)
	}
	var page CategoryPage
	expect(t, f.h.call(f.viewer, http.MethodGet, "/stock-item-categories?status=ARCHIVED&q=brak", nil), http.StatusOK, &page)
	if page.Total != 1 || page.Items[0].ActiveItemCount != 0 {
		t.Fatalf("list %+v", page)
	}
	if f.audits(t, "stock_item_category.created") != 1 || f.audits(t, "stock_item_category.updated") != 1 ||
		f.audits(t, "stock_item_category.archived") != 1 {
		t.Fatal("category changes not audited once each")
	}
}

func TestItemIsCreatedWithSuppliersAndCompatibilityAndNoStock(t *testing.T) {
	f := setup(t)
	c := f.category(t, "Brake parts")
	d := f.item(t, itemBody(" brk-pad-01 ", c.ID, "uom", " set ", "reorderLevel", "2.5", "description", "Front",
		"suppliers", []map[string]any{{"supplierId": f.supplier, "supplierPartNumber": "SP-9"}, {"supplierId": f.supplier2}},
		"compatibility", []map[string]any{{"brandId": f.brandA, "modelId": f.modelA}, {"brandId": f.brandB}}))
	if d.SKU != "brk-pad-01" || d.SKUNormalised != "BRK-PAD-01" || d.UOM != "SET" || d.TrackingMode != TrackingQuantity ||
		*d.ReorderLevel != "2.500" || d.Status != StatusActive || d.CategoryName != "Brake parts" || d.CreatedBy != f.ids["manager"] {
		t.Fatalf("item %+v", d.Item)
	}
	if len(d.Suppliers) != 2 || d.SuppliersHidden || len(d.Compatibility) != 2 {
		t.Fatalf("links %+v %+v", d.Suppliers, d.Compatibility)
	}
	if d.Compatibility[0].BrandName != "Brand A" || *d.Compatibility[0].ModelName != "Model MA" || d.Compatibility[1].ModelID != nil {
		t.Fatalf("compatibility %+v", d.Compatibility)
	}
	// Serial-tracked: whole reorder levels; no compatibility is fine.
	s := f.item(t, itemBody("BAT-SER-01", c.ID, "trackingMode", TrackingSerial, "reorderLevel", "3"))
	if s.TrackingMode != TrackingSerial || *s.ReorderLevel != "3.000" || len(s.Compatibility) != 0 || len(s.Suppliers) != 0 {
		t.Fatalf("serial item %+v", s)
	}
	// Master data only: no stock, no movement.
	if n := f.count(t, `SELECT count(*) FROM stock_movements`); n != 0 {
		t.Fatalf("stock movements: %d", n)
	}
	if n := f.count(t, `SELECT count(*) FROM audit_events WHERE action = 'stock_item.created' AND entity_id = $1
		AND summary->>'sku' = 'brk-pad-01' AND summary->>'trackingMode' = 'QUANTITY' AND actor_user_id = $2`, d.ID, f.ids["manager"]); n != 1 {
		t.Fatalf("audit: %d", n)
	}
	// A viewer sees the item but not who supplies it.
	var v Detail
	expect(t, f.h.call(f.viewer, http.MethodGet, "/stock-items/"+d.ID, nil), http.StatusOK, &v)
	if !v.SuppliersHidden || len(v.Suppliers) != 0 || len(v.Compatibility) != 2 {
		t.Fatalf("viewer %+v", v)
	}
}

func TestSKUCodesAreUniqueInEveryFormAndUnderRaces(t *testing.T) {
	f := setup(t)
	c := f.category(t, "Brake parts")
	f.item(t, itemBody("BRK-PAD-01", c.ID))
	for _, dup := range []string{"BRK-PAD-01", "brk-pad-01", "  Brk-Pad-01 "} {
		e := expectError(t, f.h.call(f.manager, http.MethodPost, "/stock-items", itemBody(dup, c.ID)), http.StatusConflict, "DUPLICATE")
		if !hasField(e.Details, "sku") {
			t.Fatalf("%q: %+v", dup, e.Details)
		}
	}
	// Separators are part of the code: these are different SKUs.
	f.item(t, itemBody("BRK.PAD.01", c.ID))
	f.item(t, itemBody("BRKPAD01", c.ID))
	// The database refuses a duplicate written directly.
	_, err := f.pool.Exec(context.Background(), `INSERT INTO stock_items (sku, name, category_id, uom, tracking_mode, created_by)
		VALUES ('bRk-PaD-01', 'X', $1, 'PCS', 'QUANTITY', $2)`, c.ID, f.ids["manager"])
	if code, constraint := sqlState(err); code != "23505" || constraint != "stock_items_sku_key" {
		t.Fatalf("direct duplicate: %v", err)
	}
	// Eight parallel creates of one code in different cases: exactly one wins.
	const n = 8
	var wg sync.WaitGroup
	ws := make([]*httptest.ResponseRecorder, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sku := "RACE-01"
			if i%2 == 1 {
				sku = "race-01"
			}
			ws[i] = f.h.call(f.manager, http.MethodPost, "/stock-items", itemBody(sku, c.ID))
		}()
	}
	wg.Wait()
	created := 0
	for _, w := range ws {
		if w.Code == http.StatusCreated {
			created++
			continue
		}
		expectError(t, w, http.StatusConflict, "DUPLICATE")
	}
	if created != 1 || f.count(t, `SELECT count(*) FROM stock_items WHERE sku_normalised = 'RACE-01'`) != 1 {
		t.Fatalf("created %d", created)
	}
	if f.audits(t, "stock_item.created") != 4 {
		t.Fatalf("refused creates audited: %d", f.audits(t, "stock_item.created"))
	}
}

func TestDatabaseRefusesInvalidItems(t *testing.T) {
	f := setup(t)
	c := f.category(t, "Brake parts")
	insert := func(sku, uom, mode string, level any) error {
		_, err := f.pool.Exec(context.Background(), `INSERT INTO stock_items (sku, name, category_id, uom, tracking_mode, reorder_level, created_by)
			VALUES ($1, 'X', $2, $3, $4, $5::numeric, $6)`, sku, c.ID, uom, mode, level, f.ids["manager"])
		return err
	}
	for _, x := range []struct {
		name, sku, uom, mode string
		level                any
		constraint           string
	}{
		{"unknown tracking mode", "A-1", "PCS", "BATCH", nil, "stock_items_tracking_mode_valid"},
		{"lower-case uom", "A-2", "pcs", "QUANTITY", nil, "stock_items_uom_format"},
		{"spaces in sku", "A 3", "PCS", "QUANTITY", nil, "stock_items_sku_format"},
		{"untrimmed sku", " A-4", "PCS", "QUANTITY", nil, "stock_items_sku_format"},
		{"negative reorder level", "A-5", "PCS", "QUANTITY", "-1", "stock_items_reorder_level_valid"},
		{"fractional serial level", "A-6", "PCS", "SERIAL", "1.5", "stock_items_serial_reorder_whole"},
	} {
		if _, constraint := sqlState(insert(x.sku, x.uom, x.mode, x.level)); constraint != x.constraint {
			t.Errorf("%s: got %q, want %s", x.name, constraint, x.constraint)
		}
	}
	// A required category, and an existing one.
	_, err := f.pool.Exec(context.Background(), `INSERT INTO stock_items (sku, name, category_id, uom, tracking_mode, created_by)
		VALUES ('A-7', 'X', gen_random_uuid(), 'PCS', 'QUANTITY', $1)`, f.ids["manager"])
	if code, _ := sqlState(err); code != "23503" && code != "23514" {
		t.Errorf("unknown category: %v", err)
	}
	if n := f.count(t, `SELECT count(*) FROM stock_items`); n != 0 {
		t.Fatalf("items: %d", n)
	}
}

func TestSupplierLinksAreValidatedAndKeptAsHistory(t *testing.T) {
	f := setup(t)
	c := f.category(t, "Brake parts")
	for _, x := range []struct {
		id, msg string
	}{
		{"00000000-0000-0000-0000-000000000000", "does not exist"},
		{f.archived.supplier, "is archived"},
	} {
		e := expectError(t, f.h.call(f.manager, http.MethodPost, "/stock-items", itemBody("S-1", c.ID,
			"suppliers", []map[string]any{{"supplierId": f.supplier}, {"supplierId": x.id}})), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
		if !hasField(e.Details, "suppliers[1].supplierId") || e.Details[0].Message != x.msg {
			t.Fatalf("%s: %+v", x.msg, e.Details)
		}
	}
	if n := f.count(t, `SELECT count(*) FROM stock_items`) + f.count(t, `SELECT count(*) FROM stock_item_suppliers`) +
		f.audits(t, "stock_item.created"); n != 0 {
		t.Fatalf("refused creates left %d rows", n)
	}
	d := f.item(t, itemBody("S-1", c.ID, "suppliers", []map[string]any{{"supplierId": f.supplier, "supplierPartNumber": "P-1"}}))
	// Replace: supplier 1 with a new part number, plus supplier 2.
	var u Detail
	expect(t, f.h.call(f.manager, http.MethodPatch, "/stock-items/"+d.ID, map[string]any{"version": 1,
		"suppliers": []map[string]any{{"supplierId": f.supplier, "supplierPartNumber": "P-2"}, {"supplierId": f.supplier2}}}), http.StatusOK, &u)
	open, ended := 0, 0
	for _, l := range u.Suppliers {
		if l.EndedAt == nil {
			open++
		} else {
			ended++
			if *l.SupplierPartNumber != "P-1" || *l.EndedBy != f.ids["manager"] {
				t.Fatalf("ended link %+v", l)
			}
		}
	}
	if open != 2 || ended != 1 || u.Version != 2 {
		t.Fatalf("links after replace: %d open, %d ended, v%d", open, ended, u.Version)
	}
	// Removing every supplier ends the links; the history stays.
	expect(t, f.h.call(f.manager, http.MethodPatch, "/stock-items/"+d.ID, map[string]any{"version": 2, "suppliers": []any{}}), http.StatusOK, &u)
	if n := f.count(t, `SELECT count(*) FROM stock_item_suppliers WHERE stock_item_id = $1`, d.ID); n != 3 || len(u.Suppliers) != 3 {
		t.Fatalf("history rows %d", n)
	}
	// A supplier archived later keeps its link (history); the list filter finds current links only.
	var page ItemPage
	expect(t, f.h.call(f.manager, http.MethodGet, "/stock-items?supplierId="+f.supplier, nil), http.StatusOK, &page)
	if page.Total != 0 {
		t.Fatalf("ended links listed: %+v", page)
	}
	// Links cannot be edited or deleted directly.
	for _, sql := range []string{
		`UPDATE stock_item_suppliers SET supplier_part_number = 'X' WHERE stock_item_id = '` + d.ID + `'`,
		`UPDATE stock_item_suppliers SET ended_at = NULL, ended_by = NULL WHERE stock_item_id = '` + d.ID + `'`,
		`DELETE FROM stock_item_suppliers WHERE stock_item_id = '` + d.ID + `'`,
	} {
		if _, err := f.pool.Exec(context.Background(), sql); func() bool { c, _ := sqlState(err); return c != "42501" }() {
			t.Errorf("%s: %v", sql, err)
		}
	}
}

func TestCompatibilityIsBrandAware(t *testing.T) {
	f := setup(t)
	c := f.category(t, "Brake parts")
	for _, x := range []struct {
		entry      map[string]any
		field, msg string
	}{
		{map[string]any{"brandId": "00000000-0000-0000-0000-000000000000"}, "compatibility[0].brandId", "does not exist"},
		{map[string]any{"brandId": f.brandA, "modelId": "00000000-0000-0000-0000-000000000000"}, "compatibility[0].modelId", "does not exist"},
		{map[string]any{"brandId": f.brandA, "modelId": f.modelB}, "compatibility[0].modelId", "belongs to another brand"},
		{map[string]any{"brandId": f.archived.brand}, "compatibility[0].brandId", "is archived"},
		{map[string]any{"brandId": f.brandA, "modelId": f.archived.model}, "compatibility[0].modelId", "is archived"},
	} {
		e := expectError(t, f.h.call(f.manager, http.MethodPost, "/stock-items", itemBody("C-1", c.ID,
			"compatibility", []map[string]any{x.entry})), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
		if len(e.Details) != 1 || e.Details[0].Field != x.field || e.Details[0].Message != x.msg {
			t.Errorf("%s: %+v", x.msg, e.Details)
		}
	}
	// The database refuses a model of another brand even when written directly.
	d := f.item(t, itemBody("C-1", c.ID, "compatibility", []map[string]any{{"brandId": f.brandA}}))
	_, err := f.pool.Exec(context.Background(), `INSERT INTO stock_item_compatibility (stock_item_id, brand_id, model_id, created_by)
		VALUES ($1, $2, $3, $4)`, d.ID, f.brandA, f.modelB, f.ids["manager"])
	if _, constraint := sqlState(err); constraint != "stock_item_compatibility_model_brand_fk" {
		t.Fatalf("cross-brand model: %v", err)
	}
	// Replace the set; the list filters by brand.
	var u Detail
	expect(t, f.h.call(f.manager, http.MethodPatch, "/stock-items/"+d.ID, map[string]any{"version": 1,
		"compatibility": []map[string]any{{"brandId": f.brandB, "modelId": f.modelB}}}), http.StatusOK, &u)
	if len(u.Compatibility) != 1 || u.Compatibility[0].BrandID != f.brandB || *u.Compatibility[0].ModelID != f.modelB {
		t.Fatalf("replaced %+v", u.Compatibility)
	}
	var page ItemPage
	expect(t, f.h.call(f.viewer, http.MethodGet, "/stock-items?brandId="+f.brandB, nil), http.StatusOK, &page)
	if page.Total != 1 {
		t.Fatalf("brand filter %+v", page)
	}
	expect(t, f.h.call(f.viewer, http.MethodGet, "/stock-items?brandId="+f.brandA, nil), http.StatusOK, &page)
	if page.Total != 0 {
		t.Fatalf("old brand still listed %+v", page)
	}
	if n := f.count(t, `SELECT count(*) FROM audit_events WHERE action = 'stock_item.updated' AND entity_id = $1
		AND summary ? 'compatibility'`, d.ID); n != 1 {
		t.Fatalf("compatibility change audit: %d", n)
	}
	// Brand names are data: a brand added later works the same, with no code change.
	newBrand := f.scalar(t, `INSERT INTO brands (name, slug) VALUES ('Brand New', 'brand-new') RETURNING id::text`)
	f.item(t, itemBody("C-2", c.ID, "compatibility", []map[string]any{{"brandId": newBrand}}))
	if n := f.count(t, `SELECT count(*) FROM stock_item_compatibility`); n != 2 {
		t.Fatalf("compatibility rows %d (none seeded)", n)
	}
}

func TestArchivedItemsKeepTheirHistory(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	c := f.category(t, "Brake parts")
	d := f.item(t, itemBody("ARC-01", c.ID, "suppliers", []map[string]any{{"supplierId": f.supplier}},
		"compatibility", []map[string]any{{"brandId": f.brandA}}))
	// The editor (no archive permission) cannot archive.
	expectError(t, f.h.call(f.editor, http.MethodPost, "/stock-items/"+d.ID+"/archive", map[string]any{"version": 1, "reason": "x"}),
		http.StatusForbidden, "FORBIDDEN")
	expectError(t, f.h.call(f.manager, http.MethodPost, "/stock-items/"+d.ID+"/archive", map[string]any{"version": 9, "reason": "x"}),
		http.StatusConflict, "VERSION_CONFLICT")
	var a Detail
	expect(t, f.h.call(f.manager, http.MethodPost, "/stock-items/"+d.ID+"/archive", map[string]any{"version": 1, "reason": "Discontinued"}),
		http.StatusOK, &a)
	if a.Status != StatusArchived || *a.ArchiveReason != "Discontinued" || *a.ArchivedBy != f.ids["manager"] || a.Version != 2 ||
		len(a.Suppliers) != 1 || len(a.Compatibility) != 1 || a.SKU != "ARC-01" {
		t.Fatalf("archived %+v", a)
	}
	// Still readable and listed under ARCHIVED; no longer among the ACTIVE choices.
	var page ItemPage
	expect(t, f.h.call(f.viewer, http.MethodGet, "/stock-items?status=ACTIVE", nil), http.StatusOK, &page)
	if page.Total != 0 {
		t.Fatalf("archived offered as active: %+v", page)
	}
	expect(t, f.h.call(f.viewer, http.MethodGet, "/stock-items?status=ARCHIVED&q=arc", nil), http.StatusOK, &page)
	if page.Total != 1 {
		t.Fatalf("archived not listed: %+v", page)
	}
	// It cannot change, be archived again or be reactivated; its code stays taken.
	expectError(t, f.h.call(f.manager, http.MethodPatch, "/stock-items/"+d.ID, map[string]any{"version": 2, "name": "New"}),
		http.StatusConflict, "INVALID_STATE")
	expectError(t, f.h.call(f.manager, http.MethodPost, "/stock-items/"+d.ID+"/archive", map[string]any{"version": 2, "reason": "x"}),
		http.StatusConflict, "INVALID_STATE")
	expectError(t, f.h.call(f.manager, http.MethodPost, "/stock-items", itemBody("arc-01", c.ID)), http.StatusConflict, "DUPLICATE")
	// The database keeps it: no delete, no update, no new links, no truncate.
	for _, sql := range []string{
		`DELETE FROM stock_items WHERE id = '` + d.ID + `'`,
		`UPDATE stock_items SET status = 'ACTIVE' WHERE id = '` + d.ID + `'`,
		`UPDATE stock_items SET name = 'Y' WHERE id = '` + d.ID + `'`,
		`DELETE FROM stock_item_compatibility WHERE stock_item_id = '` + d.ID + `'`,
		`DELETE FROM stock_item_categories WHERE id = '` + c.ID + `'`,
		`TRUNCATE stock_items CASCADE`,
	} {
		if _, err := f.pool.Exec(ctx, sql); func() bool { c, _ := sqlState(err); return c != "42501" }() {
			t.Errorf("%s: %v", sql, err)
		}
	}
	_, err := f.pool.Exec(ctx, `INSERT INTO stock_item_suppliers (stock_item_id, supplier_id, created_by) VALUES ($1, $2, $3)`,
		d.ID, f.supplier2, f.ids["manager"])
	if _, constraint := sqlState(err); constraint != "stock_items_active" {
		t.Fatalf("link to an archived item: %v", err)
	}
	// The SKU code and tracking mode of an active item cannot change either, even directly.
	other := f.item(t, itemBody("ACT-01", c.ID))
	for _, sql := range []string{
		`UPDATE stock_items SET sku = 'ACT-02' WHERE id = '` + other.ID + `'`,
		`UPDATE stock_items SET tracking_mode = 'SERIAL' WHERE id = '` + other.ID + `'`,
	} {
		if _, err := f.pool.Exec(ctx, sql); func() bool { c, _ := sqlState(err); return c != "42501" }() {
			t.Errorf("%s: %v", sql, err)
		}
	}
	if f.audits(t, "stock_item.archived") != 1 {
		t.Fatal("archive not audited once")
	}
}

func TestUpdatesFollowTheRulesAndPermissions(t *testing.T) {
	f := setup(t)
	c := f.category(t, "Brake parts")
	c2 := f.category(t, "Accessories")
	s := f.item(t, itemBody("SER-01", c.ID, "trackingMode", TrackingSerial))
	// Serial: the stored mode decides that the level must be whole.
	e := expectError(t, f.h.call(f.manager, http.MethodPatch, "/stock-items/"+s.ID, map[string]any{"version": 1, "reorderLevel": "1.5"}),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if !hasField(e.Details, "reorderLevel") {
		t.Fatalf("details %+v", e.Details)
	}
	var u Detail
	expect(t, f.h.call(f.editor, http.MethodPatch, "/stock-items/"+s.ID, map[string]any{"version": 1, "reorderLevel": "4",
		"categoryId": c2.ID, "uom": "box", "name": "Renamed", "description": "d"}), http.StatusOK, &u)
	if *u.ReorderLevel != "4.000" || u.CategoryID != c2.ID || u.UOM != "BOX" || u.Name != "Renamed" || u.Version != 2 {
		t.Fatalf("updated %+v", u.Item)
	}
	expect(t, f.h.call(f.editor, http.MethodPatch, "/stock-items/"+s.ID, map[string]any{"version": 2, "reorderLevel": nil,
		"description": nil}), http.StatusOK, &u)
	if u.ReorderLevel != nil || u.Description != nil {
		t.Fatalf("cleared %+v", u.Item)
	}
	// The viewer reads but does not write; the editor cannot name suppliers it may not see.
	expectError(t, f.h.call(f.viewer, http.MethodPatch, "/stock-items/"+s.ID, map[string]any{"version": 3, "name": "X"}),
		http.StatusForbidden, "FORBIDDEN")
	expectError(t, f.h.call(f.viewer, http.MethodPost, "/stock-items", itemBody("V-1", c.ID)), http.StatusForbidden, "FORBIDDEN")
	expectError(t, f.h.call(f.editor, http.MethodPatch, "/stock-items/"+s.ID, map[string]any{"version": 3,
		"suppliers": []map[string]any{{"supplierId": f.supplier}}}), http.StatusForbidden, "FORBIDDEN")
	expectError(t, f.h.call(f.manager, http.MethodPatch, "/stock-items/00000000-0000-0000-0000-000000000000",
		map[string]any{"version": 1, "name": "X"}), http.StatusNotFound, "NOT_FOUND")
	if f.audits(t, "stock_item.updated") != 2 {
		t.Fatalf("updates audited: %d", f.audits(t, "stock_item.updated"))
	}
	// Category filter and tracking-mode filter.
	var page ItemPage
	expect(t, f.h.call(f.viewer, http.MethodGet, "/stock-items?categoryId="+c2.ID+"&trackingMode=SERIAL", nil), http.StatusOK, &page)
	if page.Total != 1 || page.Items[0].ID != s.ID {
		t.Fatalf("filters %+v", page)
	}
	expect(t, f.h.call(f.viewer, http.MethodGet, "/stock-items?trackingMode=QUANTITY", nil), http.StatusOK, &page)
	if page.Total != 0 {
		t.Fatalf("tracking filter %+v", page)
	}
}

// Migration 000016 up, down, up: the tables, triggers and permissions come and go together.
func TestStockItemMigrationRoundTrip(t *testing.T) {
	ctx := context.Background()
	url := dbtest.ScratchURL(t)
	m := dbtest.NewMigrator(t, url)
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	state := func() string {
		var s string
		if err := pool.QueryRow(ctx, `SELECT coalesce(to_regclass('stock_items')::text, '-') || '/' ||
			(SELECT count(*) FROM permissions WHERE key LIKE 'stock_items.%')::text || '/' ||
			(SELECT count(*) FROM pg_trigger WHERE tgname LIKE 'stock_item%')::text`).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	up := state()
	if !strings.HasPrefix(up, "stock_items/3/") {
		t.Fatalf("up: %s", up)
	}
	if err := m.Migrate(15); err != nil {
		t.Fatal(err)
	}
	if s := state(); s != "-/0/0" {
		t.Fatalf("down: %s", s)
	}
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	if s := state(); s != up {
		t.Fatalf("up again: %s, want %s", s, up)
	}
}
