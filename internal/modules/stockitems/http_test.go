package stockitems

import (
	"net/http"
	"strings"
	"testing"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth/authtest"
)

// No database: the repository is nil, so reaching it would panic. Passing
// proves permission and input checks run first.

const someID = "3f2b8c1e-9a4d-4e6f-8b7a-1c2d3e4f5a6b"

func itemHarness(t *testing.T) (*harness, session) {
	t.Helper()
	store := authtest.NewStore()
	u := store.AddUser("asha", "Asha Verma", testPassword, auth.StatusActive)
	h := newHarness(t, store, nil)
	return h, h.login("asha", u.ID)
}

var routes = []struct {
	method, path, perm string
}{
	{http.MethodGet, "/stock-item-categories", access.PermStockItemView},
	{http.MethodPost, "/stock-item-categories", access.PermStockItemManage},
	{http.MethodPatch, "/stock-item-categories/" + someID, access.PermStockItemManage},
	{http.MethodPost, "/stock-item-categories/" + someID + "/archive", access.PermStockItemArchive},
	{http.MethodGet, "/stock-items", access.PermStockItemView},
	{http.MethodPost, "/stock-items", access.PermStockItemManage},
	{http.MethodGet, "/stock-items/" + someID, access.PermStockItemView},
	{http.MethodPatch, "/stock-items/" + someID, access.PermStockItemManage},
	{http.MethodPost, "/stock-items/" + someID + "/archive", access.PermStockItemArchive},
}

// Each route needs its own key; the other two stock_items keys, and neighbouring
// supplier, catalogue and stock keys, grant nothing.
func TestStockItemRoutesDenyByDefault(t *testing.T) {
	h, s := itemHarness(t)
	all := []string{access.PermStockItemView, access.PermStockItemManage, access.PermStockItemArchive}
	for _, r := range routes {
		var body any
		if r.method != http.MethodGet {
			body = map[string]any{}
		}
		if w := h.raw(r.method, "/api/v1"+r.path, "{}", "application/json", nil); w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without session: %d", r.method, r.path, w.Code)
		}
		h.grant(s)
		expectError(t, h.call(s, r.method, r.path, body), http.StatusForbidden, "FORBIDDEN")
		var others []string
		for _, p := range all {
			if p != r.perm {
				others = append(others, p)
			}
		}
		h.grant(s, append(others, access.PermSupplierManage, access.PermCatalogueManage, access.PermAdjustmentCreate,
			access.PermVehicleUnitView, access.PermAllLocations)...)
		expectError(t, h.call(s, r.method, r.path, body), http.StatusForbidden, "FORBIDDEN")
	}
}

// There is no DELETE, and no stock route.
func TestNoDeleteOrStockRoutes(t *testing.T) {
	h, s := itemHarness(t)
	h.grant(s, access.PermStockItemView, access.PermStockItemManage, access.PermStockItemArchive)
	for _, r := range []struct{ method, path string }{
		{http.MethodDelete, "/stock-items/" + someID},
		{http.MethodDelete, "/stock-item-categories/" + someID},
		{http.MethodPut, "/stock-items/" + someID},
		{http.MethodPost, "/stock-items/" + someID + "/receipts"},
		{http.MethodPost, "/stock-items/" + someID + "/issues"},
		{http.MethodPost, "/stock-items/" + someID + "/reactivate"},
		{http.MethodGet, "/stock-items/" + someID + "/balance"},
	} {
		w := h.raw(r.method, "/api/v1"+r.path, "{}", "application/json", &s)
		if w.Code != http.StatusMethodNotAllowed && w.Code != http.StatusNotFound {
			t.Errorf("%s %s: %d", r.method, r.path, w.Code)
		}
	}
}

func TestItemInputIsCheckedBeforeTheDatabase(t *testing.T) {
	h, s := itemHarness(t)
	h.grant(s, access.PermStockItemView, access.PermStockItemManage, access.PermStockItemArchive,
		access.PermSupplierView, access.PermCatalogueView)
	valid := func() map[string]any {
		return map[string]any{"sku": "BRK-PAD-01", "name": "Brake pad set", "categoryId": someID, "uom": "pcs",
			"trackingMode": "QUANTITY"}
	}
	with := func(k string, v any) map[string]any {
		b := valid()
		if v == nil {
			delete(b, k)
		} else {
			b[k] = v
		}
		return b
	}
	for _, c := range []struct {
		name  string
		body  map[string]any
		field string
	}{
		{"no sku", with("sku", nil), "sku"},
		{"blank sku", with("sku", "   "), "sku"},
		{"sku with spaces", with("sku", "BRK PAD 01"), "sku"},
		{"sku too long", with("sku", strings.Repeat("A", 41)), "sku"},
		{"sku bad characters", with("sku", "BRK#01"), "sku"},
		{"sku starts with a separator", with("sku", "-BRK"), "sku"},
		{"no name", with("name", nil), "name"},
		{"no category", with("categoryId", nil), "categoryId"},
		{"bad category", with("categoryId", "x"), "categoryId"},
		{"no uom", with("uom", nil), "uom"},
		{"bad uom", with("uom", "per piece"), "uom"},
		{"no tracking mode", with("trackingMode", nil), "trackingMode"},
		{"unknown tracking mode", with("trackingMode", "BATCH"), "trackingMode"},
		{"lower-case tracking mode", with("trackingMode", "quantity"), "trackingMode"},
		{"negative reorder level", with("reorderLevel", "-1"), "reorderLevel"},
		{"malformed reorder level", with("reorderLevel", "5,5"), "reorderLevel"},
		{"too many decimals", with("reorderLevel", "1.2345"), "reorderLevel"},
		{"fraction for serial", map[string]any{"sku": "BAT-01", "name": "Battery pack", "categoryId": someID, "uom": "PCS",
			"trackingMode": "SERIAL", "reorderLevel": "2.5"}, "reorderLevel"},
		{"reorder level as a number", with("reorderLevel", 5), "reorderLevel"},
		{"supplier twice", with("suppliers", []map[string]any{{"supplierId": someID}, {"supplierId": someID}}), "suppliers[1].supplierId"},
		{"bad supplier id", with("suppliers", []map[string]any{{"supplierId": "x"}}), "suppliers[0].supplierId"},
		{"compatibility twice", with("compatibility", []map[string]any{{"brandId": someID}, {"brandId": someID}}), "compatibility[1]"},
		{"model under a listed brand", with("compatibility", []map[string]any{{"brandId": someID},
			{"brandId": someID, "modelId": "11111111-1111-1111-1111-111111111111"}}), "compatibility[1].modelId"},
		{"bad brand id", with("compatibility", []map[string]any{{"brandId": "x"}}), "compatibility[0].brandId"},
		{"client-set status", with("status", "ARCHIVED"), "status"},
		{"client-set actor", with("createdBy", someID), "createdBy"},
	} {
		e := expectError(t, h.call(s, http.MethodPost, "/stock-items", c.body), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
		if !hasField(e.Details, c.field) {
			t.Errorf("%s: missing %s in %+v", c.name, c.field, e.Details)
		}
	}
	// The SKU code and tracking mode cannot be edited.
	for _, f := range []string{"sku", "trackingMode", "skuNormalised", "status"} {
		e := expectError(t, h.call(s, http.MethodPatch, "/stock-items/"+someID, map[string]any{"version": 1, f: "X"}),
			http.StatusUnprocessableEntity, "VALIDATION_FAILED")
		if !hasField(e.Details, f) {
			t.Errorf("patch %s: %+v", f, e.Details)
		}
	}
	for _, c := range []struct {
		body  map[string]any
		field string
	}{
		{map[string]any{"name": "x"}, "version"},
		{map[string]any{"version": 1}, "body"},
		{map[string]any{"version": 1, "name": nil}, "name"},
		{map[string]any{"version": 1, "categoryId": nil}, "categoryId"},
		{map[string]any{"version": 1, "uom": "a b"}, "uom"},
	} {
		e := expectError(t, h.call(s, http.MethodPatch, "/stock-items/"+someID, c.body), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
		if !hasField(e.Details, c.field) {
			t.Errorf("patch %v: missing %s in %+v", c.body, c.field, e.Details)
		}
	}
	for _, path := range []string{"/stock-items/" + someID + "/archive", "/stock-item-categories/" + someID + "/archive"} {
		e := expectError(t, h.call(s, http.MethodPost, path, map[string]any{}), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
		if !hasField(e.Details, "version") || !hasField(e.Details, "reason") {
			t.Errorf("%s: %+v", path, e.Details)
		}
	}
	e := expectError(t, h.call(s, http.MethodPost, "/stock-item-categories", map[string]any{"name": " "}), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if !hasField(e.Details, "name") {
		t.Errorf("category: %+v", e.Details)
	}
	e = expectError(t, h.call(s, http.MethodGet, "/stock-items?status=GONE&trackingMode=LOT&categoryId=x&limit=0", nil),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	for _, f := range []string{"status", "trackingMode", "categoryId", "limit"} {
		if !hasField(e.Details, f) {
			t.Errorf("list: missing %s in %+v", f, e.Details)
		}
	}
	expectError(t, h.call(s, http.MethodGet, "/stock-items/not-a-uuid", nil), http.StatusNotFound, "NOT_FOUND")
}

// Naming suppliers needs suppliers.view; naming brands/models needs catalogue.view;
// filtering by supplier needs suppliers.view.
func TestLinksNeedTheLinkedRecordsPermission(t *testing.T) {
	h, s := itemHarness(t)
	h.grant(s, access.PermStockItemView, access.PermStockItemManage)
	base := map[string]any{"sku": "BRK-PAD-01", "name": "Brake pad set", "categoryId": someID, "uom": "PCS", "trackingMode": "QUANTITY"}
	withLinks := func(k string, v any) map[string]any {
		b := map[string]any{k: v}
		for kk, vv := range base {
			b[kk] = vv
		}
		return b
	}
	expectError(t, h.call(s, http.MethodPost, "/stock-items", withLinks("suppliers", []map[string]any{{"supplierId": someID}})),
		http.StatusForbidden, "FORBIDDEN")
	expectError(t, h.call(s, http.MethodPost, "/stock-items", withLinks("compatibility", []map[string]any{{"brandId": someID}})),
		http.StatusForbidden, "FORBIDDEN")
	expectError(t, h.call(s, http.MethodPatch, "/stock-items/"+someID, map[string]any{"version": 1, "suppliers": []any{}}),
		http.StatusForbidden, "FORBIDDEN")
	expectError(t, h.call(s, http.MethodPatch, "/stock-items/"+someID, map[string]any{"version": 1, "compatibility": []any{}}),
		http.StatusForbidden, "FORBIDDEN")
	expectError(t, h.call(s, http.MethodGet, "/stock-items?supplierId="+someID, nil), http.StatusForbidden, "FORBIDDEN")
}

func TestNormalisationAndWholeNumbers(t *testing.T) {
	for in, want := range map[string]string{" brk-pad-01 ": "BRK-PAD-01", "Abc.1/2_x": "ABC.1/2_X", "X": "X"} {
		if got := NormaliseSKU(in); got != want {
			t.Errorf("NormaliseSKU(%q) = %q", in, got)
		}
	}
	for v, want := range map[string]bool{"5": true, "5.000": true, "0": true, "10.0": true, "100": true, "5.5": false, "0.001": false, "2.50": false} {
		if got := wholeNumber(v); got != want {
			t.Errorf("wholeNumber(%q) = %v", v, got)
		}
	}
}
