package vehicles

import (
	"net/http"
	"testing"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/auth"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/auth/authtest"
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

func TestVehicleUnitRoutesDenyByDefault(t *testing.T) {
	h, s := unitHarness(t)
	for _, path := range []string{"/vehicle-units", "/vehicle-units/" + someID} {
		if w := h.raw(http.MethodGet, "/api/v1"+path, "", "", nil); w.Code != http.StatusUnauthorized {
			t.Errorf("%s without session: %d", path, w.Code)
		}
		expectError(t, h.get(s, path), http.StatusForbidden, "FORBIDDEN")
		// The cost permission alone, or neighbouring permissions, grant nothing here.
		h.grant(s, access.PermVehicleUnitCostView, access.PermGoodsReceiptView, access.PermPurchaseOrderView, access.PermCostPriceView)
		expectError(t, h.get(s, path), http.StatusForbidden, "FORBIDDEN")
		h.grant(s)
	}
}

// There is no way to create, change or delete a unit through the API.
func TestVehicleUnitsAreReadOnly(t *testing.T) {
	h, s := unitHarness(t)
	h.grant(s, access.PermVehicleUnitView, access.PermVehicleUnitCostView)
	for _, r := range []struct{ method, path string }{
		{http.MethodPost, "/vehicle-units"},
		{http.MethodPut, "/vehicle-units/" + someID},
		{http.MethodPatch, "/vehicle-units/" + someID},
		{http.MethodDelete, "/vehicle-units/" + someID},
	} {
		w := h.raw(r.method, "/api/v1"+r.path, "{}", "application/json", &s)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s: %d, want 405", r.method, r.path, w.Code)
		}
	}
}

func TestVehicleUnitInputIsCheckedBeforeTheDatabase(t *testing.T) {
	h, s := unitHarness(t)
	h.grant(s, access.PermVehicleUnitView)
	e := expectError(t, h.get(s, "/vehicle-units?status=PARKED&brandId=x&limit=0"), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	for _, f := range []string{"status", "brandId", "limit"} {
		if !hasField(e.Details, f) {
			t.Errorf("missing field %s: %+v", f, e.Details)
		}
	}
	expectError(t, h.get(s, "/vehicle-units/not-a-uuid"), http.StatusNotFound, "NOT_FOUND")
	// A location filter outside the caller's scope is refused, not silently dropped.
	h.scope(s, "11111111-1111-1111-1111-111111111111")
	e = expectError(t, h.get(s, "/vehicle-units?locationId=22222222-2222-2222-2222-222222222222"), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if !hasField(e.Details, "locationId") {
		t.Errorf("details = %+v", e.Details)
	}
}
