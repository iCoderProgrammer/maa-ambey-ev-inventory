package purchasing

import (
	"context"
	"crypto/sha256"
	"errors"
	"net/http"
	"regexp"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database/dbtest"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/seeds"
)

// End-to-end purchase order tests on a throwaway migrated database. Names,
// amounts and references are placeholders ("Brand A", "Example Supplier",
// 1000.00): no real supplier, model, price or threshold is invented.

type variantRef struct{ brand, model, variant string }

type fixture struct {
	pool      *pgxpool.Pool
	h         *harness
	creator   session // manages, cancels, sees catalogue cost prices
	approver  session // views and approves
	approver2 session // views and approves
	viewer    session // views only
	settings  session // views and changes approval settings
	ids       map[string]string

	a, a2, archived, b, pending variantRef // pending: brand not activated
	supplier                    string     // verified, linked to brands A and B
	unverified                  string     // pending verification, no links
	archivedSupplier            string
	costList                    string // brand A cost list
	costItem                    string // approved COST 900.00 for a.variant
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
	for _, u := range []string{"creator", "approver", "approver2", "viewer", "settings"} {
		created, err := store.CreateUser(ctx, audit.CLIActor("test"), u, u, nil, hash)
		if err != nil {
			t.Fatal(err)
		}
		f.ids[u] = created.ID
	}
	f.h = newHarness(t, store, NewPGStore(pool))
	f.creator = f.h.login("creator", f.ids["creator"])
	f.approver = f.h.login("approver", f.ids["approver"])
	f.approver2 = f.h.login("approver2", f.ids["approver2"])
	f.viewer = f.h.login("viewer", f.ids["viewer"])
	f.settings = f.h.login("settings", f.ids["settings"])
	f.h.grant(f.creator, append(append([]string{}, allPO...), access.PermCostPriceView)...)
	f.h.grant(f.approver, access.PermPurchaseOrderView, access.PermPurchaseOrderApprove)
	f.h.grant(f.approver2, access.PermPurchaseOrderView, access.PermPurchaseOrderApprove)
	f.h.grant(f.viewer, access.PermPurchaseOrderView)
	f.h.grant(f.settings, access.PermPurchaseOrderView, access.PermPurchaseOrderSettings)

	verifier := f.ids["approver"]
	brandA := f.brand(t, "Brand A", true)
	brandB := f.brand(t, "Brand B", true)
	brandP := f.brand(t, "Brand Pending", false)
	modelA := f.model(t, brandA, "M1")
	f.a = variantRef{brandA, modelA, f.variant(t, brandA, modelA, "STD", "ACTIVE")}
	f.a2 = variantRef{brandA, modelA, f.variant(t, brandA, modelA, "PRO", "ACTIVE")}
	f.archived = variantRef{brandA, modelA, f.variant(t, brandA, modelA, "OLD", "ARCHIVED")}
	modelB := f.model(t, brandB, "M1")
	f.b = variantRef{brandB, modelB, f.variant(t, brandB, modelB, "STD", "ACTIVE")}
	modelP := f.model(t, brandP, "M1")
	f.pending = variantRef{brandP, modelP, f.variant(t, brandP, modelP, "STD", "ACTIVE")}

	f.supplier = f.verifiedSupplier(t, "Example Supplier One", brandA, brandB, brandP)
	f.unverified = f.rawSupplier(t, "Example Supplier Two", "PENDING_VERIFICATION")
	f.archivedSupplier = f.rawSupplier(t, "Example Supplier Three", "ARCHIVED")

	if err := pool.QueryRow(ctx, `INSERT INTO price_lists (brand_id, name, created_by) VALUES ($1, 'Supplier costs', $2) RETURNING id::text`,
		brandA, f.ids["creator"]).Scan(&f.costList); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO price_items (price_list_id, brand_id, model_id, variant_id, price_type, amount,
			effective_from, status, created_by, approved_by, approved_at)
		VALUES ($1, $2, $3, $4, 'COST', 900.00, now() - interval '1 hour', 'APPROVED', $5, $6, now()) RETURNING id::text`,
		f.costList, brandA, modelA, f.a.variant, f.ids["creator"], verifier).Scan(&f.costItem); err != nil {
		t.Fatal(err)
	}
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

// brand inserts a brand; active ones carry the authorisation evidence the schema requires.
func (f fixture) brand(t *testing.T, name string, active bool) string {
	t.Helper()
	if active {
		return f.scalar(t, `INSERT INTO brands (name, slug, lifecycle_status, authorisation_status, manufacturer_legal_name,
			authorisation_reference, verified_by, verified_at)
			VALUES ($1, lower(replace($1, ' ', '-')), 'ACTIVE', 'AUTHORISED', 'Example Manufacturer', 'TEST-AUTH', $2, now()) RETURNING id::text`,
			name, f.ids["approver"])
	}
	return f.scalar(t, `INSERT INTO brands (name, slug) VALUES ($1, lower(replace($1, ' ', '-'))) RETURNING id::text`, name)
}

func (f fixture) model(t *testing.T, brandID, code string) string {
	t.Helper()
	return f.scalar(t, `INSERT INTO models (brand_id, name, internal_code, lifecycle_status, confidence, source_url, source_date, verified_by, verified_at)
		VALUES ($1, 'Model '||$2, $2, 'ACTIVE', 'SUPPLIER_CONFIRMED', 'https://example.com/m', current_date, $3, now()) RETURNING id::text`,
		brandID, code, f.ids["approver"])
}

func (f fixture) variant(t *testing.T, brandID, modelID, code, status string) string {
	t.Helper()
	return f.scalar(t, `INSERT INTO variants (brand_id, model_id, name, variant_code, lifecycle_status, confidence, source_url, source_date, verified_by, verified_at)
		VALUES ($1, $2, 'Variant '||$3, $3, $4, 'SUPPLIER_CONFIRMED', 'https://example.com/v', current_date, $5, now()) RETURNING id::text`,
		brandID, modelID, code, status, f.ids["approver"])
}

func (f fixture) rawSupplier(t *testing.T, name, status string) string {
	t.Helper()
	return f.scalar(t, `INSERT INTO suppliers (legal_name, status, created_by) VALUES ($1, $2, $3) RETURNING id::text`, name, status, f.ids["creator"])
}

// verifiedSupplier records a supplier, a document, its verification and a
// verified current relationship with each brand.
func (f fixture) verifiedSupplier(t *testing.T, name string, brands ...string) string {
	t.Helper()
	id := f.rawSupplier(t, name, "PENDING_VERIFICATION")
	att := uuid.NewString()
	sum := sha256.Sum256([]byte(att))
	f.exec(t, `INSERT INTO attachments (id, object_key, storage_driver, content_type, size_bytes, sha256, original_name, uploaded_by)
		VALUES ($1, $2, 'local', 'application/pdf', 10, $3, 'evidence.pdf', $4)`, att, "2026/10/"+uuid.NewString(), sum[:], f.ids["creator"])
	f.exec(t, `INSERT INTO supplier_documents (supplier_id, attachment_id, document_type, added_by) VALUES ($1, $2, 'AUTHORISATION', $3)`,
		id, att, f.ids["creator"])
	f.exec(t, `UPDATE suppliers SET status = 'VERIFIED', verification_reference = 'TEST', verification_document_id = $2, verified_by = $3,
		verified_at = now() WHERE id = $1`, id, att, f.ids["approver"])
	for _, b := range brands {
		f.exec(t, `INSERT INTO supplier_brand_links (supplier_id, brand_id, status, authorisation_reference, evidence_document_id,
			effective_from, verified_by, verified_at, created_by)
			VALUES ($1, $2, 'VERIFIED', 'TEST-LINK', $3, current_date - 1, $4, now(), $5)`, id, b, att, f.ids["approver"], f.ids["creator"])
	}
	return id
}

func line(v variantRef, qty int, cost string) map[string]any {
	return map[string]any{"brandId": v.brand, "modelId": v.model, "variantId": v.variant, "quantity": qty, "unitCost": cost}
}

const future = "2099-01-31"

func (f fixture) create(t *testing.T, supplier string, lines ...map[string]any) Detail {
	t.Helper()
	ls := []map[string]any{}
	ls = append(ls, lines...)
	var d Detail
	expect(t, f.h.call(f.creator, http.MethodPost, "/purchase-orders",
		map[string]any{"supplierId": supplier, "expectedDate": future, "lines": ls}), http.StatusCreated, &d)
	return d
}

func (f fixture) get(t *testing.T, s session, id string) Detail {
	t.Helper()
	var d Detail
	expect(t, f.h.call(s, http.MethodGet, "/purchase-orders/"+id, nil), http.StatusOK, &d)
	return d
}

func (f fixture) submit(t *testing.T, d Detail) Detail {
	t.Helper()
	var out Detail
	expect(t, f.h.call(f.creator, http.MethodPost, "/purchase-orders/"+d.ID+"/submit", map[string]any{"version": d.Version}), http.StatusOK, &out)
	return out
}

func (f fixture) approve(t *testing.T, s session, d Detail) Detail {
	t.Helper()
	var out Detail
	expect(t, f.h.call(s, http.MethodPost, "/purchase-orders/"+d.ID+"/approve", map[string]any{"version": d.Version}), http.StatusOK, &out)
	return out
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

func sqlState(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

func TestNumberingTotalsAndRemaining(t *testing.T) {
	f := setup(t)
	d1 := f.create(t, f.supplier, line(f.a, 2, "1000.5"), line(f.b, 3, "250"))
	d2 := f.create(t, f.supplier)
	numberRe := regexp.MustCompile(`^PO-\d{6}$`)
	if !numberRe.MatchString(d1.Number) || !numberRe.MatchString(d2.Number) || d1.Number == d2.Number {
		t.Fatalf("numbers %q %q", d1.Number, d2.Number)
	}
	if d1.Status != StatusDraft || d1.CreatedBy != f.ids["creator"] || d1.CreatedByName != "creator" {
		t.Fatalf("header %+v", d1.PurchaseOrder)
	}
	// Server-side totals: 2 x 1000.50 + 3 x 250.00.
	if d1.TotalAmount != "2751.00" || d1.TotalQuantity != 5 || d1.RemainingQuantity != 5 || d1.ReceivedQuantity != 0 || d1.LineCount != 2 {
		t.Fatalf("totals %+v", d1.PurchaseOrder)
	}
	if d1.Lines[0].LineTotal != "2001.00" || d1.Lines[0].UnitCost != "1000.50" || d1.Lines[0].RemainingQuantity != 2 {
		t.Fatalf("line %+v", d1.Lines[0])
	}
	if len(d1.BrandNames) != 2 || len(d1.Events) != 1 || d1.Events[0].Action != ActionCreated {
		t.Fatalf("brands %v events %+v", d1.BrandNames, d1.Events)
	}
	// Client-supplied totals or statuses are not accepted at all.
	w := f.h.call(f.creator, http.MethodPost, "/purchase-orders", map[string]any{"supplierId": f.supplier, "totalAmount": "1.00"})
	if e := expectError(t, w, http.StatusUnprocessableEntity, "VALIDATION_FAILED"); !hasField(e.Details, "totalAmount") {
		t.Fatalf("details %+v", e.Details)
	}
	w = f.h.call(f.creator, http.MethodPost, "/purchase-orders", map[string]any{"supplierId": f.supplier, "status": "APPROVED"})
	expectError(t, w, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	// Quantity changes recompute totals.
	var d Detail
	expect(t, f.h.call(f.creator, http.MethodPatch, "/purchase-orders/"+d1.ID+"/lines/"+d1.Lines[1].ID,
		map[string]any{"version": d1.Version, "quantity": 4}), http.StatusOK, &d)
	if d.TotalAmount != "3001.00" || d.TotalQuantity != 6 || d.Version != d1.Version+1 {
		t.Fatalf("after edit %+v", d.PurchaseOrder)
	}
	// List and filters.
	var page POPage
	expect(t, f.h.call(f.viewer, http.MethodGet, "/purchase-orders?brandId="+f.b.brand, nil), http.StatusOK, &page)
	if page.Total != 1 || page.Items[0].ID != d1.ID || page.Items[0].TotalAmount != "3001.00" {
		t.Fatalf("brand filter %+v", page)
	}
	expect(t, f.h.call(f.viewer, http.MethodGet, "/purchase-orders?q="+d2.Number, nil), http.StatusOK, &page)
	if page.Total != 1 || page.Items[0].ID != d2.ID {
		t.Fatalf("search %+v", page)
	}
	expectError(t, f.h.call(f.viewer, http.MethodGet, "/purchase-orders?status=OPEN", nil), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
}

func TestBrandCompatibilityAndArchivedRefused(t *testing.T) {
	f := setup(t)
	cases := []struct {
		name  string
		line  map[string]any
		field string
	}{
		{"model of another brand", map[string]any{"brandId": f.a.brand, "modelId": f.b.model, "variantId": f.b.variant, "quantity": 1, "unitCost": "1"}, "lines[0].modelId"},
		{"variant of another model", map[string]any{"brandId": f.b.brand, "modelId": f.b.model, "variantId": f.a.variant, "quantity": 1, "unitCost": "1"}, "lines[0].variantId"},
		{"brand of another variant", map[string]any{"brandId": f.b.brand, "modelId": f.a.model, "variantId": f.a.variant, "quantity": 1, "unitCost": "1"}, "lines[0].modelId"},
		{"archived variant", line(f.archived, 1, "1"), "lines[0].variantId"},
		{"unknown variant", map[string]any{"brandId": f.a.brand, "modelId": f.a.model, "variantId": uuid.NewString(), "quantity": 1, "unitCost": "1"}, "lines[0].variantId"},
		{"zero quantity", line(f.a, 0, "1"), "lines[0].quantity"},
		{"negative cost", line(f.a, 1, "-5"), "lines[0].unitCost"},
	}
	for _, c := range cases {
		w := f.h.call(f.creator, http.MethodPost, "/purchase-orders", map[string]any{"supplierId": f.supplier, "lines": []any{c.line}})
		e := expectError(t, w, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
		if !hasField(e.Details, c.field) {
			t.Errorf("%s: details %+v", c.name, e.Details)
		}
	}
	e := expectError(t, f.h.call(f.creator, http.MethodPost, "/purchase-orders", map[string]any{"supplierId": f.archivedSupplier}),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if !hasField(e.Details, "supplierId") {
		t.Fatalf("archived supplier %+v", e.Details)
	}
	// One line per variant: adding the same variant again is a duplicate.
	d := f.create(t, f.supplier, line(f.a, 1, "10"))
	e = expectError(t, f.h.call(f.creator, http.MethodPost, "/purchase-orders/"+d.ID+"/lines",
		map[string]any{"version": d.Version, "brandId": f.a.brand, "modelId": f.a.model, "variantId": f.a.variant, "quantity": 1, "unitCost": "10"}),
		http.StatusConflict, "DUPLICATE")
	if !hasField(e.Details, "variantId") {
		t.Fatalf("duplicate %+v", e.Details)
	}
	// Two lines for one variant in one create are refused before the database.
	expectError(t, f.h.call(f.creator, http.MethodPost, "/purchase-orders", map[string]any{"supplierId": f.supplier,
		"lines": []any{line(f.a, 1, "1"), line(f.a, 2, "1")}}), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	// The database refuses a cross-brand line even outside the API.
	_, err := f.pool.Exec(context.Background(), `INSERT INTO purchase_order_lines (purchase_order_id, line_no, brand_id, model_id, variant_id, quantity, unit_cost)
		VALUES ($1, 9, $2, $3, $4, 1, 1)`, d.ID, f.b.brand, f.a2.model, f.a2.variant)
	if sqlState(err) != "23503" {
		t.Fatalf("cross-brand insert: %v", err)
	}
}

func TestPriceSnapshotsDoNotChange(t *testing.T) {
	f := setup(t)
	l := line(f.a, 2, "950")
	l["priceListId"] = f.costList
	d := f.create(t, f.supplier, l)
	got := d.Lines[0]
	if got.ReferenceUnitCost == nil || *got.ReferenceUnitCost != "900.00" || got.ReferencePriceItemID == nil || *got.ReferencePriceItemID != f.costItem {
		t.Fatalf("snapshot %+v", got)
	}
	if got.CostVariance == nil || *got.CostVariance != "50.00" || len(got.Warnings) != 1 || got.Warnings[0] != WarnPriceVariance {
		t.Fatalf("variance %+v", got)
	}
	// A later approved cost price supersedes the old one in the catalogue...
	ctx := context.Background()
	err := pgx.BeginFunc(ctx, f.pool, func(tx pgx.Tx) error {
		var next string
		if err := tx.QueryRow(ctx, `INSERT INTO price_items (price_list_id, brand_id, model_id, variant_id, price_type, amount, effective_from, created_by, supersedes_id)
			VALUES ($1, $2, $3, $4, 'COST', 990.00, now(), $5, $6) RETURNING id::text`,
			f.costList, f.a.brand, f.a.model, f.a.variant, f.ids["creator"], f.costItem).Scan(&next); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE price_items SET effective_to = now(), superseded_by_id = $2 WHERE id = $1`, f.costItem, next); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE price_items SET status = 'APPROVED', approved_by = $2, approved_at = now() WHERE id = $1`, next, f.ids["approver"])
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	// ...but the PO keeps what was agreed and snapshotted.
	after := f.get(t, f.creator, d.ID).Lines[0]
	if *after.ReferenceUnitCost != "900.00" || after.UnitCost != "950.00" || *after.ReferencePriceItemID != f.costItem {
		t.Fatalf("snapshot changed %+v", after)
	}
	// Without catalogue.cost_price.view the snapshot is hidden, the agreed cost is not.
	hidden := f.get(t, f.viewer, d.ID).Lines[0]
	if !hidden.ReferenceHidden || hidden.ReferenceUnitCost != nil || hidden.CostVariance != nil || hidden.UnitCost != "950.00" {
		t.Fatalf("redaction %+v", hidden)
	}
	// Naming a cost list needs catalogue.cost_price.view.
	f.h.grant(f.creator, allPO...)
	expectError(t, f.h.call(f.creator, http.MethodPost, "/purchase-orders", map[string]any{"supplierId": f.supplier, "lines": []any{l}}),
		http.StatusForbidden, "FORBIDDEN")
	f.h.grant(f.creator, append(append([]string{}, allPO...), access.PermCostPriceView)...)
	// A list of another brand is refused.
	lb := line(f.b, 1, "1")
	lb["priceListId"] = f.costList
	e := expectError(t, f.h.call(f.creator, http.MethodPost, "/purchase-orders", map[string]any{"supplierId": f.supplier, "lines": []any{lb}}),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if !hasField(e.Details, "lines[0].priceListId") {
		t.Fatalf("details %+v", e.Details)
	}
	// After submission the line, and so its snapshot, is frozen in the database.
	d = f.submit(t, f.get(t, f.creator, d.ID))
	if _, err := f.pool.Exec(ctx, `UPDATE purchase_order_lines SET unit_cost = 1 WHERE purchase_order_id = $1`, d.ID); sqlState(err) != "42501" {
		t.Fatalf("frozen line update: %v", err)
	}
}

func TestTransitionsAndApprovals(t *testing.T) {
	f := setup(t)
	// Submission prerequisites: verified supplier with a current link to each brand, active catalogue.
	d := f.create(t, f.unverified, line(f.a, 1, "100"))
	e := expectError(t, f.h.call(f.creator, http.MethodPost, "/purchase-orders/"+d.ID+"/submit", map[string]any{"version": d.Version}),
		http.StatusConflict, "SUBMISSION_BLOCKED")
	if !hasField(e.Details, "supplier_verified") || !hasField(e.Details, "supplier_brand_link") {
		t.Fatalf("blocked %+v", e.Details)
	}
	if got := f.get(t, f.viewer, d.ID); len(got.Blockers) != 2 || got.ApprovalRequired == nil || !*got.ApprovalRequired {
		t.Fatalf("detail blockers %+v required %v", got.Blockers, got.ApprovalRequired)
	}
	p := f.create(t, f.supplier, line(f.pending, 1, "100"))
	e = expectError(t, f.h.call(f.creator, http.MethodPost, "/purchase-orders/"+p.ID+"/submit", map[string]any{"version": p.Version}),
		http.StatusConflict, "SUBMISSION_BLOCKED")
	if !hasField(e.Details, "brand_active") || hasField(e.Details, "supplier_brand_link") {
		t.Fatalf("brand gate %+v", e.Details)
	}
	empty := f.create(t, f.supplier)
	e = expectError(t, f.h.call(f.creator, http.MethodPost, "/purchase-orders/"+empty.ID+"/submit", map[string]any{"version": empty.Version}),
		http.StatusConflict, "SUBMISSION_BLOCKED")
	if !hasField(e.Details, "lines") {
		t.Fatalf("empty %+v", e.Details)
	}

	d = f.create(t, f.supplier, line(f.a, 2, "100"), line(f.b, 1, "50"))
	// No threshold is configured: every PO waits for an approver.
	s := f.submit(t, d)
	if s.Status != StatusSubmitted || s.SubmittedBy == nil || *s.SubmittedBy != f.ids["creator"] || s.ThresholdApplied != nil {
		t.Fatalf("submitted %+v", s.PurchaseOrder)
	}
	// Drafts only: no edits after submission.
	expectError(t, f.h.call(f.creator, http.MethodPatch, "/purchase-orders/"+d.ID, map[string]any{"version": s.Version, "notes": "x"}),
		http.StatusConflict, "INVALID_STATE")
	expectError(t, f.h.call(f.creator, http.MethodPost, "/purchase-orders/"+d.ID+"/submit", map[string]any{"version": s.Version}),
		http.StatusConflict, "INVALID_STATE")
	// Unauthorised approvals fail: no permission, or the PO's own creator/submitter.
	expectError(t, f.h.call(f.viewer, http.MethodPost, "/purchase-orders/"+d.ID+"/approve", map[string]any{"version": s.Version}),
		http.StatusForbidden, "FORBIDDEN")
	f.h.grant(f.creator, append(append([]string{}, allPO...), access.PermPurchaseOrderApprove)...)
	if !f.get(t, f.creator, d.ID).Actions.Return || f.get(t, f.creator, d.ID).Actions.Approve {
		t.Fatal("creator must not be offered approval")
	}
	expectError(t, f.h.call(f.creator, http.MethodPost, "/purchase-orders/"+d.ID+"/approve", map[string]any{"version": s.Version}),
		http.StatusForbidden, "SELF_APPROVAL_NOT_ALLOWED")
	if f.audits(t, d.ID, "purchase_order.approved", "DENIED") != 1 {
		t.Fatal("self-approval refusal not audited")
	}
	// Stale version.
	expectError(t, f.h.call(f.approver, http.MethodPost, "/purchase-orders/"+d.ID+"/approve", map[string]any{"version": s.Version - 1}),
		http.StatusConflict, "VERSION_CONFLICT")
	// Approval re-checks prerequisites: an archived model blocks it.
	f.exec(t, `UPDATE models SET lifecycle_status = 'ARCHIVED' WHERE id = $1`, f.b.model)
	e = expectError(t, f.h.call(f.approver, http.MethodPost, "/purchase-orders/"+d.ID+"/approve", map[string]any{"version": s.Version}),
		http.StatusConflict, "APPROVAL_BLOCKED")
	if !hasField(e.Details, "model_active") {
		t.Fatalf("approval gate %+v", e.Details)
	}
	f.exec(t, `UPDATE models SET lifecycle_status = 'ACTIVE' WHERE id = $1`, f.b.model)
	a := f.approve(t, f.approver, s)
	if a.Status != StatusApproved || a.ApprovalBasis == nil || *a.ApprovalBasis != BasisApprover || *a.ApprovedBy != f.ids["approver"] {
		t.Fatalf("approved %+v", a.PurchaseOrder)
	}
	if f.audits(t, d.ID, "purchase_order.approved", "SUCCESS") != 1 || f.audits(t, d.ID, "purchase_order.submitted", "SUCCESS") != 1 {
		t.Fatal("transitions not audited")
	}
	// Forbidden transitions.
	expectError(t, f.h.call(f.approver2, http.MethodPost, "/purchase-orders/"+d.ID+"/approve", map[string]any{"version": a.Version}),
		http.StatusConflict, "INVALID_STATE")
	expectError(t, f.h.call(f.approver2, http.MethodPost, "/purchase-orders/"+d.ID+"/return", map[string]any{"version": a.Version, "reason": "x"}),
		http.StatusConflict, "INVALID_STATE")
	// Cancel an approved PO with nothing received; it stays as history.
	expectError(t, f.h.call(f.creator, http.MethodPost, "/purchase-orders/"+d.ID+"/cancel", map[string]any{"version": a.Version}),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	var c Detail
	expect(t, f.h.call(f.creator, http.MethodPost, "/purchase-orders/"+d.ID+"/cancel", map[string]any{"version": a.Version, "reason": "Supplier withdrew"}),
		http.StatusOK, &c)
	if c.Status != StatusCancelled || *c.CancelReason != "Supplier withdrew" {
		t.Fatalf("cancelled %+v", c.PurchaseOrder)
	}
	actions := []string{}
	for _, ev := range c.Events {
		actions = append(actions, ev.Action)
	}
	if len(actions) != 4 || actions[0] != ActionCreated || actions[1] != ActionSubmitted || actions[2] != ActionApproved || actions[3] != ActionCancelled {
		t.Fatalf("history %v", actions)
	}
	expectError(t, f.h.call(f.creator, http.MethodPost, "/purchase-orders/"+d.ID+"/cancel", map[string]any{"version": c.Version, "reason": "again"}),
		http.StatusConflict, "INVALID_STATE")
	// Nothing is deleted, history never changes.
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `DELETE FROM purchase_orders WHERE id = $1`, d.ID); sqlState(err) != "42501" {
		t.Fatalf("delete PO: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE purchase_order_events SET note = 'x' WHERE purchase_order_id = $1`, d.ID); sqlState(err) != "42501" {
		t.Fatalf("edit history: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `DELETE FROM purchase_order_lines WHERE purchase_order_id = $1`, d.ID); sqlState(err) != "42501" {
		t.Fatalf("delete lines: %v", err)
	}
	// The database refuses a self-approval even outside the API.
	r := f.submit(t, f.create(t, f.supplier, line(f.a, 1, "1")))
	if _, err := f.pool.Exec(ctx, `UPDATE purchase_orders SET status = 'APPROVED', approval_basis = 'APPROVER', approved_by = created_by,
		approved_at = now() WHERE id = $1`, r.ID); sqlState(err) != "23514" {
		t.Fatalf("db self-approval: %v", err)
	}
}

func TestReturnToDraftAndEdit(t *testing.T) {
	f := setup(t)
	s := f.submit(t, f.create(t, f.supplier, line(f.a, 1, "100")))
	expectError(t, f.h.call(f.approver, http.MethodPost, "/purchase-orders/"+s.ID+"/return", map[string]any{"version": s.Version, "reason": " "}),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	var r Detail
	expect(t, f.h.call(f.approver, http.MethodPost, "/purchase-orders/"+s.ID+"/return", map[string]any{"version": s.Version, "reason": "Check the quantity"}),
		http.StatusOK, &r)
	if r.Status != StatusDraft || r.SubmittedBy != nil || r.Events[len(r.Events)-1].Action != ActionReturned || *r.Events[len(r.Events)-1].Note != "Check the quantity" {
		t.Fatalf("returned %+v", r)
	}
	// Editable again: add a line, remove it, change header.
	var d Detail
	expect(t, f.h.call(f.creator, http.MethodPost, "/purchase-orders/"+r.ID+"/lines",
		map[string]any{"version": r.Version, "brandId": f.a2.brand, "modelId": f.a2.model, "variantId": f.a2.variant, "quantity": 3, "unitCost": "10"}),
		http.StatusCreated, &d)
	if len(d.Lines) != 2 || d.Lines[1].LineNo != 2 {
		t.Fatalf("added %+v", d.Lines)
	}
	expect(t, f.h.call(f.creator, http.MethodPost, "/purchase-orders/"+r.ID+"/lines/"+d.Lines[1].ID+"/remove", map[string]any{"version": d.Version}),
		http.StatusOK, &d)
	if len(d.Lines) != 1 || f.audits(t, r.ID, "purchase_order.line_removed", "SUCCESS") != 1 {
		t.Fatalf("removed %+v", d.Lines)
	}
	expect(t, f.h.call(f.creator, http.MethodPatch, "/purchase-orders/"+r.ID, map[string]any{"version": d.Version, "expectedDate": nil, "notes": "Deliver to showroom"}),
		http.StatusOK, &d)
	if d.ExpectedDate != nil || d.Notes == nil {
		t.Fatalf("header %+v", d.PurchaseOrder)
	}
	e := expectError(t, f.h.call(f.creator, http.MethodPost, "/purchase-orders/"+r.ID+"/submit", map[string]any{"version": d.Version}),
		http.StatusConflict, "SUBMISSION_BLOCKED")
	if !hasField(e.Details, "expected_date") {
		t.Fatalf("expected date %+v", e.Details)
	}
	expect(t, f.h.call(f.creator, http.MethodPatch, "/purchase-orders/"+r.ID, map[string]any{"version": d.Version, "expectedDate": "2000-01-01"}),
		http.StatusOK, &d)
	e = expectError(t, f.h.call(f.creator, http.MethodPost, "/purchase-orders/"+r.ID+"/submit", map[string]any{"version": d.Version}),
		http.StatusConflict, "SUBMISSION_BLOCKED")
	if !hasField(e.Details, "expected_date") {
		t.Fatalf("past date %+v", e.Details)
	}
	// Viewer cannot change anything.
	for _, c := range []struct{ method, path string }{
		{http.MethodPost, "/purchase-orders"}, {http.MethodPatch, "/purchase-orders/" + r.ID},
		{http.MethodPost, "/purchase-orders/" + r.ID + "/submit"}, {http.MethodPost, "/purchase-orders/" + r.ID + "/cancel"},
		{http.MethodPost, "/purchase-orders/" + r.ID + "/return"}, {http.MethodPut, "/purchase-order-settings"},
	} {
		expectError(t, f.h.call(f.viewer, c.method, c.path, map[string]any{"version": 1}), http.StatusForbidden, "FORBIDDEN")
	}
}

func TestApprovalThresholdSettings(t *testing.T) {
	f := setup(t)
	var st Settings
	expect(t, f.h.call(f.viewer, http.MethodGet, "/purchase-order-settings", nil), http.StatusOK, &st)
	if st.ApprovalThreshold != nil || !st.FirstTimeSupplierNeedsApproval || st.Version != 1 {
		t.Fatalf("default settings %+v", st)
	}
	expectError(t, f.h.call(f.creator, http.MethodPut, "/purchase-order-settings", map[string]any{"version": 1, "approvalThreshold": "5000"}),
		http.StatusForbidden, "FORBIDDEN")
	e := expectError(t, f.h.call(f.settings, http.MethodPut, "/purchase-order-settings", map[string]any{"version": 1, "approvalThreshold": "5000",
		"firstTimeSupplierNeedsApproval": true}), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if !hasField(e.Details, "approvalReference") {
		t.Fatalf("sign-off %+v", e.Details)
	}
	expect(t, f.h.call(f.settings, http.MethodPut, "/purchase-order-settings", map[string]any{"version": 1, "approvalThreshold": "5000",
		"firstTimeSupplierNeedsApproval": true, "approvalReference": "TEST-SIGNOFF"}), http.StatusOK, &st)
	if *st.ApprovalThreshold != "5000.00" || st.Version != 2 || *st.UpdatedBy != f.ids["settings"] {
		t.Fatalf("settings %+v", st)
	}
	expectError(t, f.h.call(f.settings, http.MethodPut, "/purchase-order-settings", map[string]any{"version": 1}), http.StatusConflict, "VERSION_CONFLICT")
	// First PO for the supplier still needs an approver.
	first := f.create(t, f.supplier, line(f.a, 1, "100"))
	if first.ApprovalRequired == nil || !*first.ApprovalRequired {
		t.Fatal("first-time supplier must need approval")
	}
	first = f.approve(t, f.approver, f.submit(t, first))
	// Afterwards a PO within the threshold needs no approver; one above it does.
	small := f.submit(t, f.create(t, f.supplier, line(f.a, 2, "2500")))
	if small.Status != StatusApproved || *small.ApprovalBasis != BasisBelowThreshold || small.ApprovedBy != nil || *small.ThresholdApplied != "5000.00" {
		t.Fatalf("below threshold %+v", small.PurchaseOrder)
	}
	if small.Events[len(small.Events)-1].Action != ActionApprovalNotRequired {
		t.Fatalf("history %+v", small.Events)
	}
	big := f.submit(t, f.create(t, f.supplier, line(f.a, 2, "2500.01")))
	if big.Status != StatusSubmitted {
		t.Fatalf("above threshold %+v", big.PurchaseOrder)
	}
	// Clearing the threshold makes every PO need an approver again.
	expect(t, f.h.call(f.settings, http.MethodPut, "/purchase-order-settings", map[string]any{"version": st.Version, "approvalThreshold": nil,
		"firstTimeSupplierNeedsApproval": true}), http.StatusOK, &st)
	again := f.submit(t, f.create(t, f.supplier, line(f.a, 1, "1")))
	if again.Status != StatusSubmitted {
		t.Fatalf("no threshold %+v", again.PurchaseOrder)
	}
}

func TestConcurrencyAndIdempotency(t *testing.T) {
	f := setup(t)
	// Two approvers race on the same version: exactly one wins.
	s := f.submit(t, f.create(t, f.supplier, line(f.a, 1, "100")))
	var wg sync.WaitGroup
	codes := make([]int, 2)
	for i, who := range []session{f.approver, f.approver2} {
		wg.Add(1)
		go func(i int, who session) {
			defer wg.Done()
			codes[i] = f.h.call(who, http.MethodPost, "/purchase-orders/"+s.ID+"/approve", map[string]any{"version": s.Version}).Code
		}(i, who)
	}
	wg.Wait()
	if !((codes[0] == 200 && codes[1] == 409) || (codes[0] == 409 && codes[1] == 200)) {
		t.Fatalf("approval race %v", codes)
	}
	if n := f.scalar(t, `SELECT count(*)::text FROM purchase_order_events WHERE purchase_order_id = $1 AND action = 'APPROVED'`, s.ID); n != "1" {
		t.Fatalf("approval events %s", n)
	}
	// Six parallel creates with one Idempotency-Key make one PO.
	key := "test-key-" + uuid.NewString()
	ids := make([]string, 6)
	statuses := make([]int, 6)
	for i := range ids {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			w := f.h.callWith(f.creator, http.MethodPost, "/purchase-orders", map[string]any{"supplierId": f.supplier},
				map[string]string{IdempotencyHeader: key})
			statuses[i] = w.Code
			var d Detail
			if w.Code < 300 {
				expect(t, w, w.Code, &d)
				ids[i] = d.ID
			}
		}(i)
	}
	wg.Wait()
	created := 0
	for i := range ids {
		if ids[i] == "" || ids[i] != ids[0] {
			t.Fatalf("idempotent creates %v %v", statuses, ids)
		}
		if statuses[i] == http.StatusCreated {
			created++
		}
	}
	if created != 1 || f.scalar(t, `SELECT count(*)::text FROM purchase_orders WHERE idempotency_key = $1`, key) != "1" {
		t.Fatalf("created %d %v", created, statuses)
	}
	expectError(t, f.h.callWith(f.creator, http.MethodPost, "/purchase-orders", map[string]any{"supplierId": f.supplier},
		map[string]string{IdempotencyHeader: "bad key"}), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	// Parallel line adds on one version: one succeeds, the rest are stale.
	d := f.create(t, f.supplier)
	addCodes := make([]int, 5)
	for i := range addCodes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			addCodes[i] = f.h.call(f.creator, http.MethodPost, "/purchase-orders/"+d.ID+"/lines", map[string]any{"version": d.Version,
				"brandId": f.a.brand, "modelId": f.a.model, "variantId": f.a.variant, "quantity": 1, "unitCost": "1"}).Code
		}(i)
	}
	wg.Wait()
	ok := 0
	for _, c := range addCodes {
		switch c {
		case http.StatusCreated:
			ok++
		case http.StatusConflict:
		default:
			t.Fatalf("line race %v", addCodes)
		}
	}
	if ok != 1 || len(f.get(t, f.viewer, d.ID).Lines) != 1 {
		t.Fatalf("line race %v", addCodes)
	}
}

func TestDenyByDefault(t *testing.T) {
	f := setup(t)
	d := f.create(t, f.supplier, line(f.a, 1, "1"))
	f.h.grant(f.viewer)
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/purchase-orders"}, {http.MethodGet, "/purchase-orders/" + d.ID}, {http.MethodGet, "/purchase-order-settings"},
		{http.MethodPost, "/purchase-orders/" + d.ID + "/approve"},
	} {
		expectError(t, f.h.call(f.viewer, c.method, c.path, map[string]any{"version": 1}), http.StatusForbidden, "FORBIDDEN")
	}
	expectError(t, f.h.call(f.creator, http.MethodGet, "/purchase-orders/"+uuid.NewString(), nil), http.StatusNotFound, "NOT_FOUND")
	expectError(t, f.h.call(f.creator, http.MethodGet, "/purchase-orders/not-an-id", nil), http.StatusNotFound, "NOT_FOUND")
}
