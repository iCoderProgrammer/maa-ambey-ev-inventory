package stock_test

import (
	"maps"
	"net/http"
	"slices"
	"testing"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/httpx"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth/authtest"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/stock"
)

// No database: the repository is nil, so reaching it would panic. Passing
// proves permission and input checks run first.

const someID = "3f2b8c1e-9a4d-4e6f-8b7a-1c2d3e4f5a6b"

func movementHarness(t *testing.T) (*harness, session) {
	t.Helper()
	store := authtest.NewStore()
	u := store.AddUser("asha", "Asha Verma", testPassword, auth.StatusActive)
	h := newHarness(t, store, nil)
	return h, h.login("asha", u.ID)
}

func TestMovementRoutesDenyByDefault(t *testing.T) {
	h, s := movementHarness(t)
	for _, on := range []bool{true, false} {
		path := "/api/v1/vehicle-units/" + someID + "/take-off-display"
		if on {
			path = "/api/v1/vehicle-units/" + someID + "/put-on-display"
		}
		if w := h.raw(http.MethodPost, path, `{"version":1}`, nil, map[string]string{stock.IdempotencyHeader: "key-00000001"}); w.Code != http.StatusUnauthorized {
			t.Errorf("%s without session: %d", path, w.Code)
		}
		expectError(t, h.display(s, someID, on, `{"version":1}`, "key-00000001"), http.StatusForbidden, "FORBIDDEN")
		// Viewing units, their cost, or any neighbouring permission grants nothing here.
		h.grant(s, access.PermVehicleUnitView, access.PermVehicleUnitCostView, access.PermGoodsReceiptManage,
			access.PermAdjustmentCreate, access.PermAdjustmentApprove, access.PermAllLocations)
		expectError(t, h.display(s, someID, on, `{"version":1}`, "key-00000001"), http.StatusForbidden, "FORBIDDEN")
		// Without the CSRF token the request is refused before anything else.
		h.grant(s, access.PermVehicleUnitDisplayManage)
		w := h.raw(http.MethodPost, path, `{"version":1}`, &s, map[string]string{stock.IdempotencyHeader: "key-00000001", auth.CSRFHeader: ""})
		expectError(t, w, http.StatusForbidden, "CSRF_FAILED")
		h.grant(s)
	}
}

// No route writes an arbitrary movement or status, or edits or deletes one.
func TestNoGenericOrDestructiveMovementRoutes(t *testing.T) {
	h, s := movementHarness(t)
	h.grant(s, access.PermVehicleUnitDisplayManage, access.PermVehicleUnitView)
	for _, r := range []struct{ method, path string }{
		{http.MethodPost, "/stock-movements"},
		{http.MethodPost, "/vehicle-units/" + someID + "/movements"},
		{http.MethodPost, "/vehicle-units/" + someID + "/status"},
		{http.MethodPatch, "/vehicle-units/" + someID},
		{http.MethodPut, "/stock-movements/" + someID},
		{http.MethodDelete, "/stock-movements/" + someID},
		{http.MethodDelete, "/vehicle-units/" + someID + "/put-on-display"},
		{http.MethodGet, "/vehicle-units/" + someID + "/put-on-display"},
	} {
		w := h.raw(r.method, "/api/v1"+r.path, `{"status":"AVAILABLE"}`, &s, nil)
		if w.Code != http.StatusNotFound && w.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s: %d", r.method, r.path, w.Code)
		}
	}
}

func TestMovementInputIsCheckedBeforeTheDatabase(t *testing.T) {
	h, s := movementHarness(t)
	h.grant(s, access.PermVehicleUnitDisplayManage)
	long := make([]byte, 501)
	for i := range long {
		long[i] = 'x'
	}
	for _, c := range []struct {
		body, key string
		fields    []string
	}{
		{`{}`, "key-00000001", []string{"version"}},
		{`{"version":0}`, "key-00000001", []string{"version"}},
		{`{"version":1}`, "", []string{"Idempotency-Key"}},
		{`{"version":1}`, "short", []string{"Idempotency-Key"}},
		{`{"version":1}`, "bad key with spaces", []string{"Idempotency-Key"}},
		{`{"version":1,"note":"  "}`, "key-00000001", []string{"note"}},
		{`{"version":1,"note":"` + string(long) + `"}`, "key-00000001", []string{"note"}},
		// The client cannot name the status, type, actor, time or source.
		{`{"version":1,"toStatus":"AVAILABLE"}`, "key-00000001", []string{"toStatus"}},
		{`{"version":1,"movementType":"PDI_PASS"}`, "key-00000001", []string{"movementType"}},
		{`{"version":1,"actorId":"` + someID + `"}`, "key-00000001", []string{"actorId"}},
		{`{"version":"1"}`, "key-00000001", []string{"version"}},
	} {
		e := expectError(t, h.display(s, someID, true, c.body, c.key), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
		for _, f := range c.fields {
			if !slices.ContainsFunc(e.Details, func(d httpx.FieldError) bool { return d.Field == f }) {
				t.Errorf("%s / %q: missing %s in %+v", c.body, c.key, f, e.Details)
			}
		}
	}
	expectError(t, h.display(s, "not-a-uuid", true, `{"version":1}`, "key-00000001"), http.StatusNotFound, "NOT_FOUND")
	w := h.raw(http.MethodPost, "/api/v1/vehicle-units/"+someID+"/put-on-display", `{"version":1}{}`, &s,
		map[string]string{stock.IdempotencyHeader: "key-00000001"})
	expectError(t, w, http.StatusBadRequest, "BAD_REQUEST")
}

// The Go map of allowed moves is exactly the approved lifecycle edges with an
// owner (System Architecture §6); nothing leaves SOLD, DELIVERED or REJECTED,
// and there is no PENDING_PDI -> REJECTED or QUALITY_HOLD -> AVAILABLE shortcut.
func TestTransitionsAreTheApprovedEdgesOnly(t *testing.T) {
	want := map[string]stock.Transition{
		stock.TypePDIPass:     {From: "RECEIVED_PENDING_PDI", To: "AVAILABLE"},
		stock.TypeQualityHold: {From: "RECEIVED_PENDING_PDI", To: "QUALITY_HOLD"},
		stock.TypeReject:      {From: "QUALITY_HOLD", To: "REJECTED"},
		stock.TypeDisplayOn:   {From: "AVAILABLE", To: "DISPLAY"},
		stock.TypeDisplayOff:  {From: "DISPLAY", To: "AVAILABLE"},
	}
	if !maps.Equal(stock.Transitions, want) {
		t.Fatalf("transitions = %v", stock.Transitions)
	}
	for _, tr := range stock.Transitions {
		switch tr.From {
		case "SOLD", "DELIVERED", "REJECTED", "RETURNED_TO_SUPPLIER":
			t.Errorf("a move leaves %s", tr.From)
		}
		if tr == (stock.Transition{From: "RECEIVED_PENDING_PDI", To: "REJECTED"}) || tr == (stock.Transition{From: "QUALITY_HOLD", To: "AVAILABLE"}) {
			t.Errorf("unapproved edge %v", tr)
		}
	}
}

func TestPartsRoutesDenyByDefault(t *testing.T) {
	h, s := movementHarness(t)
	routes := []struct{ method, path, perm string }{
		{http.MethodGet, "/balances", access.PermPartsView},
		{http.MethodGet, "/stock-movements", access.PermPartsView},
		{http.MethodPost, "/receipts", access.PermPartsReceive},
		{http.MethodPost, "/issues", access.PermPartsIssue},
	}
	for _, r := range routes {
		path := "/api/v1/stock-items/" + someID + r.path
		if w := h.raw(r.method, path, `{}`, nil, nil); w.Code != http.StatusUnauthorized {
			t.Errorf("%s without session: %d", path, w.Code)
		}
		// Every other parts key, stock-item master keys and every location grant nothing here.
		var others []string
		for _, p := range []string{access.PermPartsView, access.PermPartsReceive, access.PermPartsIssue} {
			if p != r.perm {
				others = append(others, p)
			}
		}
		h.grant(s, append(others, access.PermStockItemView, access.PermStockItemManage, access.PermAdjustmentCreate,
			access.PermAllLocations)...)
		expectError(t, h.raw(r.method, path, `{}`, &s, map[string]string{stock.IdempotencyHeader: "key-00000001"}),
			http.StatusForbidden, "FORBIDDEN")
	}
	// A receipt names a supplier, so it also needs suppliers.view.
	h.grant(s, access.PermPartsReceive, access.PermAllLocations)
	expectError(t, h.raw(http.MethodPost, "/api/v1/stock-items/"+someID+"/receipts", `{}`, &s,
		map[string]string{stock.IdempotencyHeader: "key-00000001"}), http.StatusForbidden, "FORBIDDEN")
}

func TestPartsInputIsCheckedBeforeTheDatabase(t *testing.T) {
	h, s := movementHarness(t)
	h.grant(s, access.PermPartsReceive, access.PermPartsIssue, access.PermPartsView, access.PermSupplierView, access.PermAllLocations)
	post := func(op, body, key string) httpx.ErrorBody {
		return expectError(t, h.raw(http.MethodPost, "/api/v1/stock-items/"+someID+"/"+op, body, &s,
			map[string]string{stock.IdempotencyHeader: key}), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	}
	e := post("receipts", `{"quantity":"1.2345","uom":"","locationId":"x"}`, "short")
	for _, f := range []string{"Idempotency-Key", "locationId", "quantity", "uom", "supplierId", "sourceReference"} {
		if !slices.ContainsFunc(e.Details, func(d httpx.FieldError) bool { return d.Field == f }) {
			t.Errorf("receipt: missing %s in %+v", f, e.Details)
		}
	}
	e = post("issues", `{"quantity":"0","uom":"PCS","locationId":"`+someID+`"}`, "key-00000001")
	for _, f := range []string{"quantity", "reason"} {
		if !slices.ContainsFunc(e.Details, func(d httpx.FieldError) bool { return d.Field == f }) {
			t.Errorf("issue: missing %s in %+v", f, e.Details)
		}
	}
	// An issue names no supplier; the client cannot send the balance or the result.
	if e := post("issues", `{"quantity":"1","uom":"PCS","locationId":"`+someID+`","reason":"x","supplierId":"`+someID+`"}`,
		"key-00000001"); e.Details[0].Field != "supplierId" {
		t.Errorf("issue with supplier: %+v", e.Details)
	}
	e = post("issues", `{"quantity":"1","uom":"PCS","locationId":"`+someID+`","reason":"x","quantityAfter":"9"}`, "key-00000001")
	if e.Details[0].Field != "quantityAfter" {
		t.Errorf("details %+v", e.Details)
	}
	expectError(t, h.raw(http.MethodGet, "/api/v1/stock-items/"+someID+"/stock-movements?type=SALE&limit=0&locationId=x", "", &s, nil),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	expectError(t, h.raw(http.MethodGet, "/api/v1/stock-items/not-a-uuid/balances", "", &s, nil), http.StatusNotFound, "NOT_FOUND")
}
