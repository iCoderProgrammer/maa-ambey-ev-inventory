package stock_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
)

// Timeline routes without a database (nil repository): permission and input
// checks must run before any query.

func TestTimelineDeniesByDefault(t *testing.T) {
	h, s := movementHarness(t)
	for _, path := range []string{"/api/v1/stock-movements", "/api/v1/stock-movements/" + someID} {
		if w := h.raw(http.MethodGet, path, "", nil, nil); w.Code != http.StatusUnauthorized {
			t.Errorf("%s without session: %d", path, w.Code)
		}
		expectError(t, h.raw(http.MethodGet, path, "", &s, nil), http.StatusForbidden, "FORBIDDEN")
		// Neighbouring permissions grant nothing: reading a linked record is not reading the ledger.
		h.grant(s, access.PermGoodsReceiptView, access.PermPurchaseOrderView, access.PermAdjustmentView,
			access.PermSupplierView, access.PermStockItemView, access.PermVehicleUnitCostView, access.PermAuditView,
			access.PermVehicleUnitDisplayManage, access.PermPartsReceive, access.PermAllLocations)
		expectError(t, h.raw(http.MethodGet, path, "", &s, nil), http.StatusForbidden, "FORBIDDEN")
		h.grant(s)
	}
}

func TestTimelineHasNoWriteRoutes(t *testing.T) {
	h, s := movementHarness(t)
	h.grant(s, access.PermVehicleUnitView, access.PermPartsView, access.PermAllLocations)
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		for _, path := range []string{"/api/v1/stock-movements", "/api/v1/stock-movements/" + someID,
			"/api/v1/stock-movements/" + someID + "/reverse"} {
			w := h.raw(method, path, `{"note":"x"}`, &s, nil)
			if w.Code != http.StatusNotFound && w.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s %s: %d", method, path, w.Code)
			}
		}
	}
}

func TestTimelineFiltersAreValidatedBeforeTheDatabase(t *testing.T) {
	h, s := movementHarness(t)
	h.grant(s, access.PermVehicleUnitView, access.PermPartsView)
	h.scope(s, someID)
	long := make([]byte, 101)
	for i := range long {
		long[i] = 'x'
	}
	for _, c := range []struct{ query, field string }{
		{"itemType=CAR", "itemType"},
		{"movementType=SALE", "movementType"},
		{"movementType=receipt", "movementType"},
		{"sourceType=INVOICE", "sourceType"},
		{"vehicleUnitId=not-a-uuid", "vehicleUnitId"},
		{"stockItemId=1", "stockItemId"},
		{"vehicleUnitId=" + someID + "&stockItemId=" + someID, "stockItemId"},
		{"actorId=x", "actorId"},
		{"locationId=x", "locationId"},
		// A location outside the caller's scope is refused, never silently dropped.
		{"locationId=0b6c2f4e-1111-4e6f-8b7a-1c2d3e4f5a6b", "locationId"},
		{"q=" + string(long), "q"},
		{"sourceReference=" + string(long), "sourceReference"},
		{"from=2026-01-15", "from"},
		{"to=yesterday", "to"},
		{"from=" + url.QueryEscape("2026-01-15T00:00:00+05:30") + "&to=" + url.QueryEscape("2026-01-15T00:00:00+05:30"), "to"},
		{"from=" + url.QueryEscape("2026-01-16T00:00:00Z") + "&to=" + url.QueryEscape("2026-01-15T00:00:00Z"), "to"},
		{"order=newest", "order"},
		{"page=0", "page"},
		{"page=x", "page"},
		{"limit=0", "limit"},
		{"limit=101", "limit"},
	} {
		e := expectError(t, h.raw(http.MethodGet, "/api/v1/stock-movements?"+c.query, "", &s, nil),
			http.StatusUnprocessableEntity, "VALIDATION_FAILED")
		if len(e.Details) == 0 || e.Details[0].Field != c.field {
			t.Errorf("%s: details %+v, want %s", c.query, e.Details, c.field)
		}
	}
}

func TestTimelineDetailMalformedIDIsNotFound(t *testing.T) {
	h, s := movementHarness(t)
	h.grant(s, access.PermVehicleUnitView)
	expectError(t, h.raw(http.MethodGet, "/api/v1/stock-movements/not-a-uuid", "", &s, nil), http.StatusNotFound, "NOT_FOUND")
}
