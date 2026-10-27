package pricing

import (
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth/authtest"
)

// No database: the repository is nil, so reaching it would panic. Passing
// proves permission and input checks run first.

const someID = "3f2b8c1e-9a4d-4e6f-8b7a-1c2d3e4f5a6b"

func unitHarness(t *testing.T) (*harness, session) {
	t.Helper()
	store := authtest.NewStore()
	u := store.AddUser("asha", "Asha Verma", testPassword, auth.StatusActive)
	h := newHarness(t, store, nil)
	return h, h.login("asha", u.ID)
}

var routes = []struct{ method, path string }{
	{http.MethodGet, "/price-lists"},
	{http.MethodPost, "/price-lists"},
	{http.MethodGet, "/price-lists/" + someID},
	{http.MethodPatch, "/price-lists/" + someID},
	{http.MethodPost, "/price-lists/" + someID + "/archive"},
	{http.MethodGet, "/price-items"},
	{http.MethodPost, "/price-items"},
	{http.MethodGet, "/price-items/" + someID},
	{http.MethodPatch, "/price-items/" + someID},
	{http.MethodPost, "/price-items/" + someID + "/cancel"},
	{http.MethodPost, "/price-items/" + someID + "/approve"},
	{http.MethodGet, "/prices/lookup?priceListId=" + someID + "&variantId=" + someID + "&priceType=SELLING"},
}

func TestPriceRoutesDenyByDefault(t *testing.T) {
	h, s := unitHarness(t)
	for _, r := range routes {
		if w := h.raw(r.method, "/api/v1"+r.path, "{}", "application/json", nil); w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without session: %d", r.method, r.path, w.Code)
		}
		expectError(t, h.call(s, r.method, r.path, map[string]any{"version": 1}), http.StatusForbidden, "FORBIDDEN")
	}
	// Catalogue permissions alone grant nothing here.
	h.grant(s, access.PermCatalogueView, access.PermCatalogueManage, access.PermCatalogueActivate)
	for _, r := range routes {
		expectError(t, h.call(s, r.method, r.path, map[string]any{"version": 1}), http.StatusForbidden, "FORBIDDEN")
	}
}

func TestCostPricesNeedCostPermissions(t *testing.T) {
	h, s := unitHarness(t)
	h.grant(s, sellingAll...)
	// Selling-price staff cannot ask for, look up or draft cost prices.
	expectError(t, h.call(s, http.MethodGet, "/price-items?priceType=COST", nil), http.StatusForbidden, "FORBIDDEN")
	expectError(t, h.call(s, http.MethodGet, "/prices/lookup?priceListId="+someID+"&variantId="+someID+"&priceType=COST", nil),
		http.StatusForbidden, "FORBIDDEN")
	expectError(t, h.call(s, http.MethodPost, "/price-items", map[string]any{
		"priceListId": someID, "variantId": someID, "priceType": "COST", "amount": "1.00", "effectiveFrom": "2026-11-01T00:00:00Z",
	}), http.StatusForbidden, "FORBIDDEN")
	// Cost staff cannot draft selling prices or approve anything.
	h.grant(s, costAll...)
	expectError(t, h.call(s, http.MethodPost, "/price-items", map[string]any{
		"priceListId": someID, "variantId": someID, "priceType": "SELLING", "amount": "1.00", "effectiveFrom": "2026-11-01T00:00:00Z",
	}), http.StatusForbidden, "FORBIDDEN")
	expectError(t, h.call(s, http.MethodPost, "/price-items/"+someID+"/approve", map[string]any{"version": 1}), http.StatusForbidden, "FORBIDDEN")
}

func TestPriceInputValidation(t *testing.T) {
	h, s := unitHarness(t)
	h.grant(s, slices.Concat(sellingAll, costAll)...)
	base := func(change map[string]any) map[string]any {
		m := map[string]any{"priceListId": someID, "variantId": someID, "priceType": "SELLING", "amount": "125000.00",
			"effectiveFrom": "2026-11-01T00:00:00+05:30"}
		for k, v := range change {
			if v == nil {
				delete(m, k)
			} else {
				m[k] = v
			}
		}
		return m
	}
	cases := map[string]struct {
		body  map[string]any
		field string
	}{
		"zero amount":           {base(map[string]any{"amount": "0.00"}), "amount"},
		"negative amount":       {base(map[string]any{"amount": "-5"}), "amount"},
		"three decimals":        {base(map[string]any{"amount": "10.005"}), "amount"},
		"too large":             {base(map[string]any{"amount": "12345678901.00"}), "amount"},
		"float not string":      {base(map[string]any{"amount": 125000.5}), "amount"},
		"unknown type":          {base(map[string]any{"priceType": "MRP"}), "priceType"},
		"no time zone":          {base(map[string]any{"effectiveFrom": "2026-11-01T00:00:00"}), "effectiveFrom"},
		"missing from":          {base(map[string]any{"effectiveFrom": nil}), "effectiveFrom"},
		"end before start":      {base(map[string]any{"effectiveTo": "2026-10-01T00:00:00+05:30"}), "effectiveTo"},
		"tax config not yet":    {base(map[string]any{"taxConfigurationId": someID}), "taxConfigurationId"},
		"bad supersedes":        {base(map[string]any{"supersedesId": "x"}), "supersedesId"},
		"status is server-side": {base(map[string]any{"status": "APPROVED"}), "status"},
		"approver is server":    {base(map[string]any{"approvedBy": someID}), "approvedBy"},
		"brand is derived":      {base(map[string]any{"brandId": someID}), "brandId"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := expectError(t, h.call(s, http.MethodPost, "/price-items", tc.body), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
			if !hasField(e.Details, tc.field) {
				t.Fatalf("no %s detail: %+v", tc.field, e.Details)
			}
		})
	}
	expectError(t, h.call(s, http.MethodGet, "/prices/lookup?priceListId=x&variantId="+someID+"&priceType=SELLING", nil),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	expectError(t, h.call(s, http.MethodGet, "/prices/lookup?priceListId="+someID+"&variantId="+someID+"&priceType=SELLING&at=yesterday", nil),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	expectError(t, h.call(s, http.MethodPatch, "/price-items/"+someID, map[string]any{"version": 1, "amount": "abc"}),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	expectError(t, h.call(s, http.MethodPost, "/price-items/"+someID+"/approve", map[string]any{}),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if w := h.call(s, http.MethodDelete, "/price-items/"+someID, nil); w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("DELETE: %d", w.Code)
	}
}

func TestApprovalRules(t *testing.T) {
	now := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	draft := Item{Status: StatusDraft, PriceListID: "L", VariantID: "V", PriceType: TypeSelling, EffectiveFrom: now.Add(time.Hour)}
	ok := approvalFacts{Item: draft, ListStatus: ListActive, BrandStatus: "PENDING_VERIFICATION", ModelStatus: "ACTIVE",
		VariantStatus: "ACTIVE", VariantConfidence: "SUPPLIER_CONFIRMED", Now: now}
	if err := checkApproval(ok); err != nil {
		t.Fatal(err)
	}
	f := ok
	f.VariantConfidence = "UNVERIFIED"
	f.ListStatus = ListArchived
	var blocked *BlockedError
	if err := checkApproval(f); !asErr(err, &blocked) || len(blocked.Unmet) != 2 {
		t.Fatalf("want 2 blockers, got %v", err)
	}
	// Retroactive start refused; within the grace allowed.
	f = ok
	f.Item.EffectiveFrom = now.Add(-time.Hour)
	if err := checkApproval(f); err == nil {
		t.Fatal("retroactive approval allowed")
	}
	f.Item.EffectiveFrom = now.Add(-time.Minute)
	if err := checkApproval(f); err != nil {
		t.Fatalf("grace: %v", err)
	}
	// Supersession must start after the old price and overlap it.
	old := Item{ID: "O", Status: StatusApproved, PriceListID: "L", VariantID: "V", PriceType: TypeSelling, EffectiveFrom: now}
	f = ok
	f.Superseded = &old
	if err := checkApproval(f); err != nil {
		t.Fatal(err)
	}
	end := now.Add(30 * time.Minute)
	old.EffectiveTo = &end
	if err := checkApproval(f); err == nil {
		t.Fatal("non-overlapping supersession allowed")
	}
	old.EffectiveTo = nil
	other := old
	other.PriceType = TypeCost
	f.Superseded = &other
	if err := checkApproval(f); err == nil {
		t.Fatal("cross-type supersession allowed")
	}
	f = ok
	f.Item.Status = StatusApproved
	var state *StateError
	if err := checkApproval(f); !asErr(err, &state) {
		t.Fatal("re-approval allowed")
	}
}

func TestAmountNormalisation(t *testing.T) {
	for in, want := range map[string]string{"125000": "125000.00", "0125.5": "125.50", " 99.99 ": "99.99", "0.01": "0.01"} {
		var c checker
		if got := normaliseAmount(&c, in); got != want || c.err() != nil {
			t.Errorf("%q → %q (%v), want %q", in, got, c.err(), want)
		}
	}
}

func asErr[T error](err error, target *T) bool { return errors.As(err, target) }
