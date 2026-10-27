package pricing

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database/dbtest"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/catalogue"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/seeds"
)

// End-to-end price history tests on a throwaway migrated database. Amounts and
// names are placeholders ("Brand A", 1000.00): no real price is invented.

type fixture struct {
	pool     *pgxpool.Pool
	h        *harness
	drafter  session // drafts selling and cost prices
	approver session // approves (and views both types)
	seller   session // selling prices only, read
	brandA   string
	brandB   string
	variantA string // verified, active (brand A)
	variantB string // brand B
	listA    string
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
	ids := map[string]string{}
	for _, u := range []string{"drafter", "approver", "seller"} {
		created, err := store.CreateUser(ctx, audit.CLIActor("test"), u, u, nil, hash)
		if err != nil {
			t.Fatal(err)
		}
		ids[u] = created.ID
	}
	h := newHarness(t, store, NewPGStore(pool))
	f := fixture{pool: pool, h: h}
	f.drafter = h.login("drafter", ids["drafter"])
	f.approver = h.login("approver", ids["approver"])
	f.seller = h.login("seller", ids["seller"])
	h.grant(f.drafter, append([]string{"catalogue.price.view", "catalogue.price.manage"}, costAll...)...)
	h.grant(f.approver, "catalogue.price.view", "catalogue.price.approve", "catalogue.cost_price.view")
	h.grant(f.seller, "catalogue.price.view")

	f.brandA, f.variantA = f.catalogue(t, "Brand A", ids["approver"], "SUPPLIER_CONFIRMED")
	f.brandB, f.variantB = f.catalogue(t, "Brand B", ids["approver"], "SUPPLIER_CONFIRMED")
	var l PriceList
	expect(t, h.call(f.drafter, http.MethodPost, "/price-lists", map[string]any{"brandId": f.brandA, "name": "Showroom"}), http.StatusCreated, &l)
	f.listA = l.ID
	return f
}

// catalogue inserts a brand with one active model and variant of the given confidence.
func (f fixture) catalogue(t *testing.T, name, verifier, confidence string) (brandID, variantID string) {
	t.Helper()
	ctx := context.Background()
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	if err := f.pool.QueryRow(ctx, `INSERT INTO brands (name, slug) VALUES ($1, $2) RETURNING id::text`, name, slug).Scan(&brandID); err != nil {
		t.Fatal(err)
	}
	var modelID string
	if err := f.pool.QueryRow(ctx, `INSERT INTO models (brand_id, name, internal_code, lifecycle_status, confidence, source_url, source_date, verified_by, verified_at)
		VALUES ($1, 'Model 1', 'M1', 'ACTIVE', 'SUPPLIER_CONFIRMED', 'https://example.com/m', current_date, $2, now()) RETURNING id::text`,
		brandID, verifier).Scan(&modelID); err != nil {
		t.Fatal(err)
	}
	q := `INSERT INTO variants (brand_id, model_id, name, variant_code, lifecycle_status, confidence, source_url, source_date, verified_by, verified_at)
		VALUES ($1, $2, 'Variant 1', 'STD', 'ACTIVE', $3, 'https://example.com/v', current_date, $4, now()) RETURNING id::text`
	if confidence == "UNVERIFIED" {
		q = `INSERT INTO variants (brand_id, model_id, name, variant_code) VALUES ($1, $2, 'Variant 1', 'STD') RETURNING id::text`
		if err := f.pool.QueryRow(ctx, q, brandID, modelID).Scan(&variantID); err != nil {
			t.Fatal(err)
		}
		return brandID, variantID
	}
	if err := f.pool.QueryRow(ctx, q, brandID, modelID, confidence, verifier).Scan(&variantID); err != nil {
		t.Fatal(err)
	}
	return brandID, variantID
}

func ts(t time.Time) string { return t.Format(time.RFC3339) }

func (f fixture) draft(t *testing.T, body map[string]any) Item {
	t.Helper()
	m := map[string]any{"priceListId": f.listA, "variantId": f.variantA, "priceType": TypeSelling}
	for k, v := range body {
		m[k] = v
	}
	var it Item
	expect(t, f.h.call(f.drafter, http.MethodPost, "/price-items", m), http.StatusCreated, &it)
	return it
}

func (f fixture) approve(t *testing.T, it Item) Item {
	t.Helper()
	var out Item
	expect(t, f.h.call(f.approver, http.MethodPost, "/price-items/"+it.ID+"/approve", map[string]any{"version": it.Version, "approvalReference": "OWNER-TEST-1"}),
		http.StatusOK, &out)
	return out
}

func (f fixture) lookup(t *testing.T, s session, at time.Time) (Item, int) {
	t.Helper()
	w := f.h.call(s, http.MethodGet, "/prices/lookup?priceListId="+f.listA+"&variantId="+f.variantA+"&priceType=SELLING&at="+
		at.UTC().Format("2006-01-02T15:04:05Z"), nil)
	var it Item
	if w.Code == http.StatusOK {
		expect(t, w, http.StatusOK, &it)
	}
	return it, w.Code
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

func TestDraftApproveAndLookup(t *testing.T) {
	f := setup(t)
	start := time.Now().Add(time.Hour).Truncate(time.Second)
	d := f.draft(t, map[string]any{"amount": "1000", "effectiveFrom": ts(start), "sourceNote": "Placeholder source"})
	if d.Status != StatusDraft || d.Amount != "1000.00" || d.BrandID != f.brandA || d.ModelID == "" || d.TaxConfigurationID != nil {
		t.Fatalf("draft %+v", d)
	}
	// The drafter may edit a draft but never approve it.
	expect(t, f.h.call(f.drafter, http.MethodPatch, "/price-items/"+d.ID, map[string]any{"version": d.Version, "amount": "1100.50"}), http.StatusOK, &d)
	if d.Amount != "1100.50" {
		t.Fatal(d.Amount)
	}
	f.h.grant(f.drafter, append(sellingAll, costAll...)...)
	expectError(t, f.h.call(f.drafter, http.MethodPost, "/price-items/"+d.ID+"/approve", map[string]any{"version": d.Version}),
		http.StatusForbidden, "SELF_APPROVAL_NOT_ALLOWED")
	if f.audits(t, d.ID, "pricing.price_item.approved", "DENIED") != 1 {
		t.Fatal("self-approval not audited")
	}

	a := f.approve(t, d)
	if a.Status != StatusApproved || a.ApprovedBy == nil || *a.ApprovedBy != f.approver.userID || a.ApprovedAt == nil ||
		a.ApprovalReference == nil || *a.ApprovalReference != "OWNER-TEST-1" {
		t.Fatalf("approved %+v", a)
	}
	// Approved prices cannot be edited or cancelled through the API.
	expectError(t, f.h.call(f.drafter, http.MethodPatch, "/price-items/"+a.ID, map[string]any{"version": a.Version, "amount": "1.00"}),
		http.StatusConflict, "INVALID_STATE")
	expectError(t, f.h.call(f.drafter, http.MethodPost, "/price-items/"+a.ID+"/cancel", map[string]any{"version": a.Version}),
		http.StatusConflict, "INVALID_STATE")

	// Lookup: nothing before the start, the price from the start on.
	if _, code := f.lookup(t, f.seller, start.Add(-time.Second)); code != http.StatusNotFound {
		t.Fatalf("before start: %d", code)
	}
	got, code := f.lookup(t, f.seller, start)
	if code != http.StatusOK || got.ID != a.ID || got.Amount != "1100.50" {
		t.Fatalf("at start: %d %+v", code, got)
	}
	expectError(t, f.h.call(f.seller, http.MethodGet, "/prices/lookup?priceListId="+f.listA+"&variantId="+f.variantA+"&priceType=SELLING&at="+
		start.Add(-time.Hour).UTC().Format(time.RFC3339), nil), http.StatusNotFound, "PRICE_NOT_FOUND")
	if f.audits(t, a.ID, "pricing.price_item.approved", "SUCCESS") != 1 || f.audits(t, a.ID, "pricing.price_item.drafted", "SUCCESS") != 1 {
		t.Fatal("audit missing")
	}
}

func TestSupersessionKeepsHistoryStable(t *testing.T) {
	f := setup(t)
	t1 := time.Now().Add(time.Hour).Truncate(time.Second)
	t2 := t1.Add(24 * time.Hour)
	first := f.approve(t, f.draft(t, map[string]any{"amount": "1000.00", "effectiveFrom": ts(t1)}))

	// What a quotation made at t1+1h snapshots.
	quoted, _ := f.lookup(t, f.seller, t1.Add(time.Hour))

	// A change without supersession overlaps and is refused.
	clash := f.draft(t, map[string]any{"amount": "1200.00", "effectiveFrom": ts(t2)})
	e := expectError(t, f.h.call(f.approver, http.MethodPost, "/price-items/"+clash.ID+"/approve", map[string]any{"version": clash.Version}),
		http.StatusConflict, "PRICE_OVERLAP")
	if !hasField(e.Details, "conflictingPriceId") {
		t.Fatal(e)
	}
	expect(t, f.h.call(f.drafter, http.MethodPost, "/price-items/"+clash.ID+"/cancel", map[string]any{"version": clash.Version}), http.StatusOK, &clash)
	if clash.Status != StatusCancelled {
		t.Fatal("not cancelled")
	}

	// Controlled supersession: the old period ends where the new one starts.
	next := f.approve(t, f.draft(t, map[string]any{"amount": "1200.00", "effectiveFrom": ts(t2), "supersedesId": first.ID}))
	var old Item
	expect(t, f.h.call(f.seller, http.MethodGet, "/price-items/"+first.ID, nil), http.StatusOK, &old)
	if old.Status != StatusApproved || old.EffectiveTo == nil || !old.EffectiveTo.Equal(t2) || old.SupersededByID == nil ||
		*old.SupersededByID != next.ID || old.Amount != "1000.00" {
		t.Fatalf("superseded %+v", old)
	}
	if f.audits(t, first.ID, "pricing.price_item.superseded", "SUCCESS") != 1 {
		t.Fatal("supersession not audited")
	}

	// Old quotation instant still answers the old price; later instants the new one.
	again, _ := f.lookup(t, f.seller, t1.Add(time.Hour))
	if again.ID != quoted.ID || again.Amount != quoted.Amount {
		t.Fatalf("historical answer changed: %+v vs %+v", again, quoted)
	}
	if p, _ := f.lookup(t, f.seller, t2.Add(-time.Nanosecond)); p.ID != first.ID {
		t.Fatal("boundary before t2")
	}
	if p, _ := f.lookup(t, f.seller, t2); p.ID != next.ID || p.Amount != "1200.00" {
		t.Fatal("boundary at t2")
	}

	// A superseded price cannot be superseded again.
	expectError(t, f.h.call(f.drafter, http.MethodPost, "/price-items", map[string]any{"priceListId": f.listA, "variantId": f.variantA,
		"priceType": "SELLING", "amount": "1.00", "effectiveFrom": ts(t2.Add(2 * time.Hour)), "supersedesId": first.ID}),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
}

func TestNoRetroactiveApprovalAndApprovalBlocks(t *testing.T) {
	f := setup(t)
	past := time.Now().Add(-48 * time.Hour).Truncate(time.Second)
	d := f.draft(t, map[string]any{"amount": "1000.00", "effectiveFrom": ts(past)})
	e := expectError(t, f.h.call(f.approver, http.MethodPost, "/price-items/"+d.ID+"/approve", map[string]any{"version": d.Version}),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if !hasField(e.Details, "effectiveFrom") {
		t.Fatal(e)
	}
	// Simulated clock: approving two days earlier would have been fine.
	f.h.setClock(past.Add(-time.Minute))
	f.approve(t, d)
	f.h.setClock(time.Time{})

	// Unverified variant: approval blocked.
	brandU, variantU := f.catalogue(t, "Brand C", f.approver.userID, "UNVERIFIED")
	var l PriceList
	expect(t, f.h.call(f.drafter, http.MethodPost, "/price-lists", map[string]any{"brandId": brandU, "name": "Showroom"}), http.StatusCreated, &l)
	var u Item
	expect(t, f.h.call(f.drafter, http.MethodPost, "/price-items", map[string]any{"priceListId": l.ID, "variantId": variantU,
		"priceType": "SELLING", "amount": "10.00", "effectiveFrom": ts(time.Now().Add(time.Hour))}), http.StatusCreated, &u)
	e = expectError(t, f.h.call(f.approver, http.MethodPost, "/price-items/"+u.ID+"/approve", map[string]any{"version": u.Version}),
		http.StatusConflict, "APPROVAL_BLOCKED")
	if !hasField(e.Details, "variant_verification") || f.audits(t, u.ID, "pricing.price_item.approved", "DENIED") != 1 {
		t.Fatal(e)
	}
}

func TestBrandAndListRules(t *testing.T) {
	f := setup(t)
	// A variant of another brand cannot be priced on this brand's list.
	e := expectError(t, f.h.call(f.drafter, http.MethodPost, "/price-items", map[string]any{"priceListId": f.listA, "variantId": f.variantB,
		"priceType": "SELLING", "amount": "1.00", "effectiveFrom": ts(time.Now().Add(time.Hour))}), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if !hasField(e.Details, "variantId") {
		t.Fatal(e)
	}
	expectError(t, f.h.call(f.drafter, http.MethodPost, "/price-lists", map[string]any{"brandId": f.brandA, "name": "showroom"}),
		http.StatusConflict, "DUPLICATE")
	// The database refuses a cross-brand item even without the service.
	_, err := f.pool.Exec(context.Background(), `INSERT INTO price_items (price_list_id, brand_id, model_id, variant_id, price_type, amount, effective_from, created_by)
		SELECT $1, $2, v.model_id, v.id, 'SELLING', 1, now(), $3 FROM variants v WHERE v.id = $4`, f.listA, f.brandA, f.drafter.userID, f.variantB)
	if sqlState(err) != "23503" {
		t.Fatalf("cross-brand insert: %v", err)
	}

	// Archived list: history and lookups stay, nothing new.
	a := f.approve(t, f.draft(t, map[string]any{"amount": "500.00", "effectiveFrom": ts(time.Now().Add(time.Hour))}))
	var l PriceList
	expect(t, f.h.call(f.drafter, http.MethodGet, "/price-lists/"+f.listA, nil), http.StatusOK, &l)
	expect(t, f.h.call(f.drafter, http.MethodPost, "/price-lists/"+f.listA+"/archive", map[string]any{"version": l.Version}), http.StatusOK, &l)
	expectError(t, f.h.call(f.drafter, http.MethodPost, "/price-items", map[string]any{"priceListId": f.listA, "variantId": f.variantA,
		"priceType": "SELLING", "amount": "1.00", "effectiveFrom": ts(time.Now().Add(2 * time.Hour))}), http.StatusConflict, "INVALID_STATE")
	if p, code := f.lookup(t, f.seller, a.EffectiveFrom.Add(time.Minute)); code != http.StatusOK || p.ID != a.ID {
		t.Fatalf("lookup on archived list: %d", code)
	}
}

func TestCostPricesStayPrivate(t *testing.T) {
	f := setup(t)
	cost := f.approve(t, f.draft(t, map[string]any{"priceType": "COST", "amount": "700.00", "effectiveFrom": ts(time.Now().Add(time.Hour))}))
	f.approve(t, f.draft(t, map[string]any{"amount": "900.00", "effectiveFrom": ts(time.Now().Add(time.Hour))}))
	// Same period, different types: no overlap.
	var page ItemPage
	expect(t, f.h.call(f.seller, http.MethodGet, "/price-items?priceListId="+f.listA, nil), http.StatusOK, &page)
	if page.Total != 1 || page.Items[0].PriceType != TypeSelling {
		t.Fatalf("seller sees %+v", page.Items)
	}
	expectError(t, f.h.call(f.seller, http.MethodGet, "/price-items/"+cost.ID, nil), http.StatusNotFound, "NOT_FOUND")
	expectError(t, f.h.call(f.seller, http.MethodGet, "/price-items?priceType=COST", nil), http.StatusForbidden, "FORBIDDEN")
	expect(t, f.h.call(f.approver, http.MethodGet, "/price-items?priceListId="+f.listA, nil), http.StatusOK, &page)
	if page.Total != 2 {
		t.Fatalf("approver sees %d", page.Total)
	}
}

func TestApprovedRowsAreImmutableInTheDatabase(t *testing.T) {
	f := setup(t)
	a := f.approve(t, f.draft(t, map[string]any{"amount": "1000.00", "effectiveFrom": ts(time.Now().Add(time.Hour))}))
	ctx := context.Background()
	for _, sql := range []string{
		`UPDATE price_items SET amount = 1 WHERE id = $1`,
		`UPDATE price_items SET effective_from = effective_from - interval '1 day' WHERE id = $1`,
		`UPDATE price_items SET effective_to = now() + interval '5 days' WHERE id = $1`, // end without supersession
		`UPDATE price_items SET status = 'CANCELLED' WHERE id = $1`,
		`DELETE FROM price_items WHERE id = $1`,
	} {
		if _, err := f.pool.Exec(ctx, sql, a.ID); sqlState(err) != "42501" {
			t.Errorf("%s: %v", sql, err)
		}
	}
	// Self-approval is impossible even in SQL.
	d := f.draft(t, map[string]any{"amount": "5.00", "effectiveFrom": ts(time.Now().Add(48 * time.Hour))})
	if _, err := f.pool.Exec(ctx, `UPDATE price_items SET status = 'APPROVED', approved_by = created_by, approved_at = now() WHERE id = $1`, d.ID); sqlState(err) != "23514" {
		t.Fatalf("self-approval in SQL: %v", err)
	}
	// A tax configuration reference cannot be stored until one exists.
	if _, err := f.pool.Exec(ctx, `UPDATE price_items SET tax_configuration_id = gen_random_uuid() WHERE id = $1`, d.ID); sqlState(err) != "23514" {
		t.Fatalf("tax config: %v", err)
	}
}

func TestConcurrentApprovalsCannotOverlap(t *testing.T) {
	f := setup(t)
	start := time.Now().Add(time.Hour).Truncate(time.Second)
	drafts := make([]Item, 6)
	for i := range drafts {
		drafts[i] = f.draft(t, map[string]any{"amount": "1000.00", "effectiveFrom": ts(start.Add(time.Duration(i) * time.Minute))})
	}
	codes := make([]int, len(drafts))
	var wg sync.WaitGroup
	for i, d := range drafts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes[i] = f.h.call(f.approver, http.MethodPost, "/price-items/"+d.ID+"/approve", map[string]any{"version": d.Version}).Code
		}()
	}
	wg.Wait()
	ok := 0
	for _, c := range codes {
		switch c {
		case http.StatusOK:
			ok++
		case http.StatusConflict:
		default:
			t.Fatalf("unexpected %d", c)
		}
	}
	if ok != 1 {
		t.Fatalf("%d overlapping approvals succeeded", ok)
	}
	var n int
	if err := f.pool.QueryRow(context.Background(), `SELECT count(*) FROM price_items WHERE status = 'APPROVED'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("approved rows %d %v", n, err)
	}
}

func TestBrandReadinessSeesApprovedPrices(t *testing.T) {
	f := setup(t)
	store := catalogue.NewPGStore(f.pool)
	check := func() bool {
		r, err := store.BrandReadiness(context.Background(), f.brandA)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range r.Checks {
			if c.Key == "approved_prices" {
				return c.Met
			}
		}
		t.Fatal("no approved_prices check")
		return false
	}
	// A future price does not count yet; one valid now does.
	f.approve(t, f.draft(t, map[string]any{"amount": "1000.00", "effectiveFrom": ts(time.Now().Add(24 * time.Hour))}))
	if check() {
		t.Fatal("future price counted")
	}
	f.h.setClock(time.Now().Add(-3 * time.Minute))
	var l PriceList
	expect(t, f.h.call(f.drafter, http.MethodPost, "/price-lists", map[string]any{"brandId": f.brandA, "name": "Second"}), http.StatusCreated, &l)
	var d Item
	expect(t, f.h.call(f.drafter, http.MethodPost, "/price-items", map[string]any{"priceListId": l.ID, "variantId": f.variantA,
		"priceType": "SELLING", "amount": "1000.00", "effectiveFrom": ts(time.Now().Add(-time.Minute))}), http.StatusCreated, &d)
	f.approve(t, d)
	f.h.setClock(time.Time{})
	if !check() {
		t.Fatal("current approved price not counted")
	}
}

func sqlState(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}
