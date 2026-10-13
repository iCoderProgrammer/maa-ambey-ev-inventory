package catalogue

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/auth"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/auth/authtest"
)

// These tests run without a database: the repository is nil, so any request
// that reached it would panic. Passing proves permission and input checks
// happen before data access.

const someID = "3f2b8c1e-9a4d-4e6f-8b7a-1c2d3e4f5a6b"

type route struct {
	method, path, permission string
}

var routes = []route{
	{http.MethodGet, "/brands", access.PermCatalogueView},
	{http.MethodPost, "/brands", access.PermCatalogueManage},
	{http.MethodGet, "/brands/" + someID, access.PermCatalogueView},
	{http.MethodPatch, "/brands/" + someID, access.PermCatalogueManage},
	{http.MethodPut, "/brands/" + someID + "/authorisation", access.PermBrandAuthorise},
	{http.MethodGet, "/brands/" + someID + "/activation-readiness", access.PermCatalogueView},
	{http.MethodPost, "/brands/" + someID + "/activate", access.PermBrandActivate},
	{http.MethodPost, "/brands/" + someID + "/archive", access.PermCatalogueArchive},
}

func init() {
	for _, kind := range []string{"models", "variants", "colours"} {
		routes = append(routes,
			route{http.MethodGet, "/" + kind, access.PermCatalogueView},
			route{http.MethodPost, "/" + kind, access.PermCatalogueManage},
			route{http.MethodGet, "/" + kind + "/" + someID, access.PermCatalogueView},
			route{http.MethodPatch, "/" + kind + "/" + someID, access.PermCatalogueManage},
			route{http.MethodPost, "/" + kind + "/" + someID + "/verify", access.PermCatalogueVerify},
			route{http.MethodPost, "/" + kind + "/" + someID + "/activate", access.PermCatalogueActivate},
			route{http.MethodPost, "/" + kind + "/" + someID + "/archive", access.PermCatalogueArchive},
		)
	}
}

func unitHarness(t *testing.T) (*harness, session) {
	t.Helper()
	store := authtest.NewStore()
	u := store.AddUser("asha", "Asha Verma", testPassword, auth.StatusActive)
	h := newHarness(t, store, nil)
	return h, h.login("asha", u.ID)
}

func TestRoutesNeedASession(t *testing.T) {
	h, _ := unitHarness(t)
	for _, r := range routes {
		w := h.raw(r.method, "/api/v1"+r.path, "{}", "application/json", nil)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without session: %d", r.method, r.path, w.Code)
		}
	}
}

func TestDenyByDefault(t *testing.T) {
	h, s := unitHarness(t)
	// A signed-in user with no grants (the state until the owner approves the matrix).
	for _, r := range routes {
		expectError(t, h.call(s, r.method, r.path, map[string]any{"version": 1}), http.StatusForbidden, "FORBIDDEN")
	}
}

func TestEachRouteNeedsItsOwnPermission(t *testing.T) {
	h, s := unitHarness(t)
	for _, r := range routes {
		// Every catalogue permission except the one this route needs.
		others := slices.DeleteFunc(slices.Clone(allCataloguePermissions), func(p string) bool { return p == r.permission })
		h.grant(s, others...)
		w := h.call(s, r.method, r.path, map[string]any{"version": 1})
		if w.Code != http.StatusForbidden {
			t.Errorf("%s %s without %s: %d %s", r.method, r.path, r.permission, w.Code, w.Body)
		}
	}
}

func TestNoDeleteRoutes(t *testing.T) {
	h, s := unitHarness(t)
	h.grant(s, allCataloguePermissions...)
	for _, p := range []string{"/brands/", "/models/", "/variants/", "/colours/"} {
		if w := h.call(s, http.MethodDelete, p+someID, nil); w.Code != http.StatusMethodNotAllowed {
			t.Errorf("DELETE %s: %d", p, w.Code)
		}
	}
}

func TestRequestValidationBeforeDataAccess(t *testing.T) {
	h, s := unitHarness(t)
	h.grant(s, allCataloguePermissions...)

	// Not JSON.
	w := h.raw(http.MethodPost, "/api/v1/brands", "name=x", "application/x-www-form-urlencoded", &s)
	expectError(t, w, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE")
	// Malformed JSON.
	expectError(t, h.raw(http.MethodPost, "/api/v1/brands", "{", "application/json", &s), http.StatusBadRequest, "BAD_REQUEST")

	cases := []struct {
		name, method, path string
		body               any
		fields             []string
	}{
		{"blank brand", http.MethodPost, "/brands", map[string]any{"name": "  "}, []string{"name"}},
		{"bad slug", http.MethodPost, "/brands", map[string]any{"name": "Brand A", "slug": "Brand A"}, []string{"slug"}},
		{"status is server-owned", http.MethodPost, "/brands", map[string]any{"name": "Brand A", "lifecycleStatus": "ACTIVE"}, []string{"lifecycleStatus"}},
		{"brand edit without version", http.MethodPatch, "/brands/" + someID, map[string]any{"name": "B"}, []string{"version"}},
		{"brand name cannot be cleared", http.MethodPatch, "/brands/" + someID, map[string]any{"version": 1, "name": nil}, []string{"name"}},
		{"model needs brand", http.MethodPost, "/models", map[string]any{"name": "M", "internalCode": "M1"}, []string{"brandId"}},
		{"model code format", http.MethodPost, "/models", map[string]any{"brandId": someID, "name": "M", "internalCode": "has space"}, []string{"internalCode"}},
		{"vehicle type format", http.MethodPost, "/models", map[string]any{"brandId": someID, "name": "M", "internalCode": "M1", "vehicleType": "two wheeler"}, []string{"vehicleType"}},
		{"model brand cannot move", http.MethodPatch, "/models/" + someID, map[string]any{"version": 1, "brandId": someID}, []string{"brandId"}},
		{"variant model cannot move", http.MethodPatch, "/variants/" + someID, map[string]any{"version": 1, "modelId": someID}, []string{"modelId"}},
		{"variant brand id not a uuid", http.MethodPost, "/variants", map[string]any{"modelId": someID, "brandId": "x", "name": "V", "variantCode": "STD"}, []string{"brandId"}},
		{"colour needs variant", http.MethodPost, "/colours", map[string]any{"name": "C"}, []string{"variantId"}},
		{"empty patch", http.MethodPatch, "/colours/" + someID, map[string]any{"version": 1}, []string{"name"}},
		{"authorised needs evidence", http.MethodPut, "/brands/" + someID + "/authorisation", map[string]any{"version": 1, "authorisationStatus": "AUTHORISED"},
			[]string{"manufacturerLegalName", "authorisationReference", "authorisationEvidenceId"}},
		{"unknown authorisation status", http.MethodPut, "/brands/" + someID + "/authorisation", map[string]any{"version": 1, "authorisationStatus": "MAYBE"}, []string{"authorisationStatus"}},
		{"verify needs source", http.MethodPost, "/models/" + someID + "/verify", map[string]any{"version": 1, "confidence": "SUPPLIER_CONFIRMED", "sourceDate": "2026-01-01"}, []string{"sourceUrl"}},
		{"verify cannot set UNVERIFIED", http.MethodPost, "/models/" + someID + "/verify", map[string]any{"version": 1, "confidence": "UNVERIFIED", "sourceUrl": "https://example.com/spec", "sourceDate": "2026-01-01"}, []string{"confidence"}},
		{"verify future date", http.MethodPost, "/variants/" + someID + "/verify", map[string]any{"version": 1, "confidence": "CLIENT_APPROVED", "sourceUrl": "https://example.com", "sourceDate": "2999-01-01"}, []string{"sourceDate"}},
		{"verify non-http url", http.MethodPost, "/colours/" + someID + "/verify", map[string]any{"version": 1, "confidence": "CLIENT_APPROVED", "sourceUrl": "javascript:alert(1)", "sourceDate": "2026-01-01"}, []string{"sourceUrl"}},
		{"verifier is server-owned", http.MethodPost, "/models/" + someID + "/verify", map[string]any{"version": 1, "confidence": "CLIENT_APPROVED", "sourceUrl": "https://example.com", "sourceDate": "2026-01-01", "verifiedBy": someID}, []string{"verifiedBy"}},
		{"activate without version", http.MethodPost, "/brands/" + someID + "/activate", map[string]any{}, []string{"version"}},
		{"version wrong type", http.MethodPost, "/models/" + someID + "/archive", map[string]any{"version": "1"}, []string{"version"}},
		{"list limit", http.MethodGet, "/models?limit=1000", nil, []string{"limit"}},
		{"list status", http.MethodGet, "/variants?status=DELETED", nil, []string{"status"}},
		{"list brand filter", http.MethodGet, "/colours?brandId=nope", nil, []string{"brandId"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := expectError(t, h.call(s, tc.method, tc.path, tc.body), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
			for _, f := range tc.fields {
				if !hasField(e.Details, f) {
					t.Errorf("missing detail for %s: %+v", f, e.Details)
				}
			}
		})
	}

	// A malformed ID cannot exist.
	expectError(t, h.call(s, http.MethodGet, "/models/not-a-uuid", nil), http.StatusNotFound, "NOT_FOUND")
}

func TestErrorsNeverLeakDatabaseDetail(t *testing.T) {
	h, s := unitHarness(t)
	h.grant(s, allCataloguePermissions...)
	w := h.call(s, http.MethodPost, "/brands", map[string]any{"name": strings.Repeat("x", 121)})
	if strings.Contains(w.Body.String(), "constraint") || strings.Contains(w.Body.String(), "SQL") {
		t.Fatalf("leaks detail: %s", w.Body)
	}
}
