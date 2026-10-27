package receiving

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth/authtest"
)

// No database: the repository is nil, so reaching it would panic. Passing
// proves permission and input checks run first.

const someID = "3f2b8c1e-9a4d-4e6f-8b7a-1c2d3e4f5a6b"

var everyGRN = []string{access.PermGoodsReceiptView, access.PermGoodsReceiptManage, access.PermGoodsReceiptOverReceipt,
	access.PermAttachmentsUpload}

func unitHarness(t *testing.T) (*harness, session) {
	t.Helper()
	store := authtest.NewStore()
	u := store.AddUser("asha", "Asha Verma", testPassword, auth.StatusActive)
	h := newHarness(t, store, nil)
	return h, h.login("asha", u.ID)
}

var routes = []struct{ method, path, perm string }{
	{http.MethodGet, "/goods-receipts", access.PermGoodsReceiptView},
	{http.MethodPost, "/goods-receipts", access.PermGoodsReceiptManage},
	{http.MethodGet, "/goods-receipts/" + someID, access.PermGoodsReceiptView},
	{http.MethodPut, "/goods-receipts/" + someID, access.PermGoodsReceiptManage},
	{http.MethodPost, "/goods-receipts/" + someID + "/cancel", access.PermGoodsReceiptManage},
	{http.MethodPost, "/goods-receipts/" + someID + "/documents", access.PermGoodsReceiptManage},
	{http.MethodGet, "/receivable-purchase-orders", access.PermGoodsReceiptManage},
}

func TestGoodsReceiptRoutesDenyByDefault(t *testing.T) {
	h, s := unitHarness(t)
	for _, r := range routes {
		if w := h.raw(r.method, "/api/v1"+r.path, "{}", "application/json", nil, nil); w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without session: %d", r.method, r.path, w.Code)
		}
		expectError(t, h.call(s, r.method, r.path, map[string]any{"version": 1}), http.StatusForbidden, "FORBIDDEN")
	}
	// Each route needs its own permission.
	for _, r := range routes {
		h.grant(s, slices.DeleteFunc(slices.Clone(everyGRN), func(p string) bool { return p == r.perm })...)
		if w := h.call(s, r.method, r.path, map[string]any{"version": 1}); w.Code != http.StatusForbidden {
			t.Errorf("%s %s without %s: %d", r.method, r.path, r.perm, w.Code)
		}
	}
	// Attaching a document also needs attachments.upload.
	h.grant(s, access.PermGoodsReceiptView, access.PermGoodsReceiptManage)
	expectError(t, h.call(s, http.MethodPost, "/goods-receipts/"+someID+"/documents",
		map[string]any{"attachmentId": someID, "documentType": "PHOTO"}), http.StatusForbidden, "FORBIDDEN")
	// Mutations need the CSRF token.
	h.grant(s, everyGRN...)
	req := h.raw(http.MethodPost, "/api/v1/goods-receipts", `{}`, "application/json", nil, nil)
	if req.Code != http.StatusUnauthorized {
		t.Errorf("no session: %d", req.Code)
	}
	noCSRF := s
	noCSRF.csrf = ""
	if w := h.call(noCSRF, http.MethodPost, "/goods-receipts", map[string]any{}); w.Code != http.StatusForbidden {
		t.Errorf("without CSRF token: %d %s", w.Code, w.Body)
	}
}

func TestGoodsReceiptInputValidation(t *testing.T) {
	h, s := unitHarness(t)
	h.grant(s, everyGRN...)
	unit := func(extra map[string]any) map[string]any {
		u := map[string]any{"condition": "ACCEPTED"}
		for k, v := range extra {
			u[k] = v
		}
		return u
	}
	line := func(received, accepted, rejected int, units ...map[string]any) map[string]any {
		return map[string]any{"purchaseOrderLineId": someID, "receivedQuantity": received, "acceptedQuantity": accepted,
			"rejectedQuantity": rejected, "units": units}
	}
	save := func(lines ...map[string]any) map[string]any {
		return map[string]any{"version": 1, "receivedDate": "2026-10-01", "lines": lines}
	}
	put := "/goods-receipts/" + someID
	cases := []struct {
		method, path string
		body         map[string]any
		field        string
	}{
		{http.MethodPost, "/goods-receipts", map[string]any{}, "purchaseOrderId"},
		{http.MethodPost, "/goods-receipts", map[string]any{"purchaseOrderId": someID}, "receivedDate"},
		{http.MethodPost, "/goods-receipts", map[string]any{"purchaseOrderId": someID, "receivedDate": "01/10/2026"}, "receivedDate"},
		{http.MethodPost, "/goods-receipts", map[string]any{"purchaseOrderId": someID, "receivedDate": "2026-10-01", "status": "POSTED"}, "status"},
		{http.MethodPut, put, map[string]any{"receivedDate": "2026-10-01"}, "version"},
		{http.MethodPut, put, save(line(2, 1, 0)), "lines[0].acceptedQuantity"},
		{http.MethodPut, put, save(line(-1, 0, 0)), "lines[0].receivedQuantity"},
		{http.MethodPut, put, save(line(10001, 10001, 0)), "lines[0].receivedQuantity"},
		{http.MethodPut, put, save(line(0, 0, 0, unit(nil))), "lines[0].receivedQuantity"},
		{http.MethodPut, put, save(line(1, 1, 0, unit(nil), unit(nil))), "lines[0].units"},
		{http.MethodPut, put, save(line(2, 1, 1, unit(nil), unit(nil))), "lines[0].units"},
		{http.MethodPut, put, save(line(1, 1, 0, map[string]any{"condition": "DAMAGED"})), "lines[0].units[0].condition"},
		{http.MethodPut, put, save(line(1, 1, 0, unit(map[string]any{"chassisNumber": strings.Repeat("A", 51)}))), "lines[0].units[0].chassisNumber"},
		{http.MethodPut, put, save(line(1, 1, 0, unit(map[string]any{"motorSerial": "MÖTOR-1"}))), "lines[0].units[0].motorSerial"},
		{http.MethodPut, put, save(line(2, 2, 0, unit(map[string]any{"chassisNumber": "ab-1"}), unit(map[string]any{"chassisNumber": " AB-1 "}))), "lines[0].units[1].chassisNumber"},
		{http.MethodPut, put, save(line(1, 1, 0), line(1, 1, 0)), "lines[1].purchaseOrderLineId"},
		{http.MethodPost, put + "/cancel", map[string]any{"version": 1, "reason": " "}, "reason"},
		{http.MethodPost, put + "/documents", map[string]any{"attachmentId": someID, "documentType": "INVOICE"}, "documentType"},
		{http.MethodPost, put + "/documents", map[string]any{"attachmentId": "x", "documentType": "PHOTO"}, "attachmentId"},
		{http.MethodGet, "/goods-receipts?status=OPEN", nil, "status"},
		{http.MethodGet, "/goods-receipts?purchaseOrderId=x", nil, "purchaseOrderId"},
		{http.MethodGet, "/receivable-purchase-orders?limit=500", nil, "limit"},
	}
	for _, c := range cases {
		e := expectError(t, h.call(s, c.method, c.path, c.body), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
		if !hasField(e.Details, c.field) {
			t.Errorf("%s %s %v: details %+v", c.method, c.path, c.body, e.Details)
		}
	}
	if w := h.raw(http.MethodPut, "/api/v1"+put, "{}", "text/plain", &s, nil); w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("content type: %d", w.Code)
	}
	expectError(t, h.callWith(s, http.MethodPost, "/goods-receipts", map[string]any{"purchaseOrderId": someID, "receivedDate": "2026-10-01"},
		map[string]string{IdempotencyHeader: "short"}), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	// Unknown IDs are 404 before any lookup.
	expectError(t, h.call(s, http.MethodGet, "/goods-receipts/nope", nil), http.StatusNotFound, "NOT_FOUND")
}

func TestNoRouteDeletesOrPostsReceipts(t *testing.T) {
	h, _ := unitHarness(t)
	for _, r := range h.router.Routes() {
		if !strings.Contains(r.Path, "goods-receipts") {
			continue
		}
		if r.Method == http.MethodDelete || strings.HasSuffix(r.Path, "/post") {
			t.Errorf("unexpected route %s %s", r.Method, r.Path)
		}
	}
}
