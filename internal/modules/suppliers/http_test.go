package suppliers

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

func unitHarness(t *testing.T) (*harness, session) {
	t.Helper()
	store := authtest.NewStore()
	u := store.AddUser("asha", "Asha Verma", testPassword, auth.StatusActive)
	h := newHarness(t, store, nil)
	return h, h.login("asha", u.ID)
}

var routes = []struct{ method, path, perm string }{
	{http.MethodGet, "/suppliers", access.PermSupplierView},
	{http.MethodPost, "/suppliers", access.PermSupplierManage},
	{http.MethodGet, "/suppliers/" + someID, access.PermSupplierView},
	{http.MethodPatch, "/suppliers/" + someID, access.PermSupplierManage},
	{http.MethodPost, "/suppliers/" + someID + "/verify", access.PermSupplierVerify},
	{http.MethodPost, "/suppliers/" + someID + "/archive", access.PermSupplierArchive},
	{http.MethodPost, "/suppliers/" + someID + "/contacts", access.PermSupplierManage},
	{http.MethodPatch, "/suppliers/" + someID + "/contacts/" + someID, access.PermSupplierManage},
	{http.MethodPost, "/suppliers/" + someID + "/contacts/" + someID + "/remove", access.PermSupplierManage},
	{http.MethodPost, "/suppliers/" + someID + "/documents", access.PermSupplierManage},
	{http.MethodPost, "/suppliers/" + someID + "/brand-links", access.PermSupplierManage},
	{http.MethodGet, "/supplier-brand-links?brandId=" + someID, access.PermSupplierView},
	{http.MethodPost, "/supplier-brand-links/" + someID + "/verify", access.PermSupplierVerify},
	{http.MethodPost, "/supplier-brand-links/" + someID + "/end", access.PermSupplierArchive},
}

func TestSupplierRoutesDenyByDefault(t *testing.T) {
	h, s := unitHarness(t)
	for _, r := range routes {
		if w := h.raw(r.method, "/api/v1"+r.path, "{}", "application/json", nil); w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without session: %d", r.method, r.path, w.Code)
		}
		expectError(t, h.call(s, r.method, r.path, map[string]any{"version": 1}), http.StatusForbidden, "FORBIDDEN")
	}
	// Each route needs its own permission.
	for _, r := range routes {
		h.grant(s, slices.DeleteFunc(slices.Clone(allSupplier), func(p string) bool { return p == r.perm })...)
		if w := h.call(s, r.method, r.path, map[string]any{"version": 1}); w.Code != http.StatusForbidden {
			t.Errorf("%s %s without %s: %d", r.method, r.path, r.perm, w.Code)
		}
	}
	// Linking a document also needs attachments.upload.
	h.grant(s, allSupplier...)
	expectError(t, h.call(s, http.MethodPost, "/suppliers/"+someID+"/documents", map[string]any{"attachmentId": someID, "documentType": "OTHER"}),
		http.StatusForbidden, "FORBIDDEN")
	if w := h.call(s, http.MethodDelete, "/suppliers/"+someID, nil); w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("DELETE: %d", w.Code)
	}
}

func TestSupplierInputValidation(t *testing.T) {
	h, s := unitHarness(t)
	h.grant(s, append(slices.Clone(allSupplier), access.PermAttachmentsUpload)...)
	cases := map[string]struct {
		method, path string
		body         map[string]any
		field        string
	}{
		"legal name required":    {http.MethodPost, "/suppliers", map[string]any{"legalName": "  "}, "legalName"},
		"bad GSTIN":              {http.MethodPost, "/suppliers", map[string]any{"legalName": "X", "gstin": "12345"}, "gstin"},
		"bad PIN":                {http.MethodPost, "/suppliers", map[string]any{"legalName": "X", "postalCode": "01234"}, "postalCode"},
		"status is server-owned": {http.MethodPost, "/suppliers", map[string]any{"legalName": "X", "status": "VERIFIED"}, "status"},
		"verifier server-owned":  {http.MethodPost, "/suppliers", map[string]any{"legalName": "X", "verifiedBy": someID}, "verifiedBy"},
		"empty patch":            {http.MethodPatch, "/suppliers/" + someID, map[string]any{"version": 1}, "legalName"},
		"legal name not cleared": {http.MethodPatch, "/suppliers/" + someID, map[string]any{"version": 1, "legalName": nil}, "legalName"},
		"verify needs reference": {http.MethodPost, "/suppliers/" + someID + "/verify", map[string]any{"version": 1, "documentId": someID}, "reference"},
		"contact needs channel":  {http.MethodPost, "/suppliers/" + someID + "/contacts", map[string]any{"name": "A"}, "phone"},
		"bad email":              {http.MethodPost, "/suppliers/" + someID + "/contacts", map[string]any{"name": "A", "email": "not-mail"}, "email"},
		"bad phone":              {http.MethodPost, "/suppliers/" + someID + "/contacts", map[string]any{"name": "A", "phone": "abc"}, "phone"},
		"unknown document type":  {http.MethodPost, "/suppliers/" + someID + "/documents", map[string]any{"attachmentId": someID, "documentType": "BANK"}, "documentType"},
		"link needs brand":       {http.MethodPost, "/suppliers/" + someID + "/brand-links", map[string]any{}, "brandId"},
		"link verify needs date": {http.MethodPost, "/supplier-brand-links/" + someID + "/verify", map[string]any{"version": 1, "authorisationReference": "R", "evidenceDocumentId": someID}, "effectiveFrom"},
		"link end before start":  {http.MethodPost, "/supplier-brand-links/" + someID + "/verify", map[string]any{"version": 1, "authorisationReference": "R", "evidenceDocumentId": someID, "effectiveFrom": "2026-10-10", "effectiveTo": "2026-10-01"}, "effectiveTo"},
		"end needs reason":       {http.MethodPost, "/supplier-brand-links/" + someID + "/end", map[string]any{"version": 1}, "reason"},
		"links need a filter":    {http.MethodGet, "/supplier-brand-links", nil, "brandId"},
		"list status":            {http.MethodGet, "/suppliers?status=GONE", nil, "status"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var body any
			if tc.body != nil {
				body = tc.body
			}
			e := expectError(t, h.call(s, tc.method, tc.path, body), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
			if !hasField(e.Details, tc.field) {
				t.Fatalf("no %s detail: %+v", tc.field, e.Details)
			}
		})
	}
}

func TestGSTINNormalised(t *testing.T) {
	in := SupplierInput{LegalName: " X ", GSTIN: ptr(" 27abcde1234f1z5 ")}
	if err := validateSupplier(&in); err != nil {
		t.Fatal(err)
	}
	if in.LegalName != "X" || *in.GSTIN != "27ABCDE1234F1Z5" {
		t.Fatalf("%+v", in)
	}
}

func ptr(s string) *string { return &s }
