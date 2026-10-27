package purchasing

import (
	"net/http"
	"slices"
	"testing"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth/authtest"
)

// No database: the repository is nil, so reaching it would panic. Passing
// proves permission and input checks run first.

const someID = "3f2b8c1e-9a4d-4e6f-8b7a-1c2d3e4f5a6b"

var everyPO = []string{access.PermPurchaseOrderView, access.PermPurchaseOrderManage, access.PermPurchaseOrderApprove,
	access.PermPurchaseOrderCancel, access.PermPurchaseOrderSettings}

func unitHarness(t *testing.T) (*harness, session) {
	t.Helper()
	store := authtest.NewStore()
	u := store.AddUser("asha", "Asha Verma", testPassword, auth.StatusActive)
	h := newHarness(t, store, nil)
	return h, h.login("asha", u.ID)
}

var routes = []struct{ method, path, perm string }{
	{http.MethodGet, "/purchase-orders", access.PermPurchaseOrderView},
	{http.MethodPost, "/purchase-orders", access.PermPurchaseOrderManage},
	{http.MethodGet, "/purchase-orders/" + someID, access.PermPurchaseOrderView},
	{http.MethodPatch, "/purchase-orders/" + someID, access.PermPurchaseOrderManage},
	{http.MethodPost, "/purchase-orders/" + someID + "/lines", access.PermPurchaseOrderManage},
	{http.MethodPatch, "/purchase-orders/" + someID + "/lines/" + someID, access.PermPurchaseOrderManage},
	{http.MethodPost, "/purchase-orders/" + someID + "/lines/" + someID + "/remove", access.PermPurchaseOrderManage},
	{http.MethodPost, "/purchase-orders/" + someID + "/submit", access.PermPurchaseOrderManage},
	{http.MethodPost, "/purchase-orders/" + someID + "/approve", access.PermPurchaseOrderApprove},
	{http.MethodPost, "/purchase-orders/" + someID + "/return", access.PermPurchaseOrderApprove},
	{http.MethodPost, "/purchase-orders/" + someID + "/cancel", access.PermPurchaseOrderCancel},
	{http.MethodPut, "/purchase-order-settings", access.PermPurchaseOrderSettings},
}

func TestPurchaseOrderRoutesDenyByDefault(t *testing.T) {
	h, s := unitHarness(t)
	for _, r := range routes {
		if w := h.raw(r.method, "/api/v1"+r.path, "{}", "application/json", nil, nil); w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without session: %d", r.method, r.path, w.Code)
		}
		expectError(t, h.call(s, r.method, r.path, map[string]any{"version": 1}), http.StatusForbidden, "FORBIDDEN")
	}
	expectError(t, h.call(s, http.MethodGet, "/purchase-order-settings", nil), http.StatusForbidden, "FORBIDDEN")
	// Each route needs its own permission.
	for _, r := range routes {
		h.grant(s, slices.DeleteFunc(slices.Clone(everyPO), func(p string) bool { return p == r.perm })...)
		if w := h.call(s, r.method, r.path, map[string]any{"version": 1}); w.Code != http.StatusForbidden {
			t.Errorf("%s %s without %s: %d", r.method, r.path, r.perm, w.Code)
		}
	}
	// A cost price list reference also needs catalogue.cost_price.view.
	h.grant(s, everyPO...)
	expectError(t, h.call(s, http.MethodPost, "/purchase-orders", map[string]any{"supplierId": someID, "lines": []any{map[string]any{
		"brandId": someID, "modelId": someID, "variantId": someID, "quantity": 1, "unitCost": "1", "priceListId": someID}}}),
		http.StatusForbidden, "FORBIDDEN")
}

func TestPurchaseOrderInputValidation(t *testing.T) {
	h, s := unitHarness(t)
	h.grant(s, everyPO...)
	cases := []struct {
		method, path string
		body         map[string]any
		field        string
	}{
		{http.MethodPost, "/purchase-orders", map[string]any{}, "supplierId"},
		{http.MethodPost, "/purchase-orders", map[string]any{"supplierId": someID, "expectedDate": "15/11/2026"}, "expectedDate"},
		{http.MethodPost, "/purchase-orders", map[string]any{"supplierId": someID, "approvedBy": someID}, "approvedBy"},
		{http.MethodPost, "/purchase-orders", map[string]any{"supplierId": someID, "lines": []any{map[string]any{"brandId": "x"}}}, "lines[0].brandId"},
		{http.MethodPost, "/purchase-orders", map[string]any{"supplierId": someID, "lines": []any{map[string]any{
			"brandId": someID, "modelId": someID, "variantId": someID, "quantity": 10001, "unitCost": "1"}}}, "lines[0].quantity"},
		{http.MethodPost, "/purchase-orders", map[string]any{"supplierId": someID, "lines": []any{map[string]any{
			"brandId": someID, "modelId": someID, "variantId": someID, "quantity": 1, "unitCost": "1.234"}}}, "lines[0].unitCost"},
		{http.MethodPost, "/purchase-orders", map[string]any{"supplierId": someID, "lines": []any{map[string]any{
			"brandId": someID, "modelId": someID, "variantId": someID, "quantity": 1, "unitCost": "0"}}}, "lines[0].unitCost"},
		{http.MethodPatch, "/purchase-orders/" + someID, map[string]any{"version": 1}, "supplierId"},
		{http.MethodPatch, "/purchase-orders/" + someID, map[string]any{"notes": "x"}, "version"},
		{http.MethodPost, "/purchase-orders/" + someID + "/lines", map[string]any{"version": 1, "quantity": 1}, "brandId"},
		{http.MethodPatch, "/purchase-orders/" + someID + "/lines/" + someID, map[string]any{"version": 1}, "quantity"},
		{http.MethodPost, "/purchase-orders/" + someID + "/submit", map[string]any{}, "version"},
		{http.MethodPost, "/purchase-orders/" + someID + "/approve", map[string]any{}, "version"},
		{http.MethodPost, "/purchase-orders/" + someID + "/return", map[string]any{"version": 1}, "reason"},
		{http.MethodPost, "/purchase-orders/" + someID + "/cancel", map[string]any{"version": 1, "reason": " "}, "reason"},
		{http.MethodPut, "/purchase-order-settings", map[string]any{"version": 1, "approvalThreshold": "abc"}, "approvalThreshold"},
		{http.MethodPut, "/purchase-order-settings", map[string]any{"version": 1, "approvalThreshold": "10"}, "approvalReference"},
	}
	for _, c := range cases {
		e := expectError(t, h.call(s, c.method, c.path, c.body), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
		if !hasField(e.Details, c.field) {
			t.Errorf("%s %s %v: details %+v", c.method, c.path, c.body, e.Details)
		}
	}
	if w := h.raw(http.MethodPost, "/api/v1/purchase-orders", "{}", "text/plain", &s, nil); w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("content type: %d", w.Code)
	}
	expectError(t, h.callWith(s, http.MethodPost, "/purchase-orders", map[string]any{"supplierId": someID},
		map[string]string{IdempotencyHeader: "short"}), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	// Unknown IDs are 404 before any lookup.
	expectError(t, h.call(s, http.MethodGet, "/purchase-orders/nope", nil), http.StatusNotFound, "NOT_FOUND")
}
