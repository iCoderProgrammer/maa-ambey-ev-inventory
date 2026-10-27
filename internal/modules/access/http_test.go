package access_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/httpx"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth/authtest"
)

const password = "correct horse battery staple"

// fakeLoader serves Access per user ID; proposed grants are tracked separately
// so the allowProposed switch is exercised like the real store.
type fakeLoader struct {
	approved map[string][]string
	proposed map[string][]string
	scope    map[string][]access.Location
	err      error
	calls    int
}

func (f *fakeLoader) LoadAccess(_ context.Context, userID string, allowProposed bool) (access.Access, error) {
	f.calls++
	if f.err != nil {
		return access.Access{}, f.err
	}
	perms := append([]string(nil), f.approved[userID]...)
	if allowProposed {
		perms = append(perms, f.proposed[userID]...)
	}
	return access.NewAccess(userID, []access.Role{{Key: "viewer", Name: "Viewer"}}, perms, f.scope[userID], allowProposed), nil
}

func (f *fakeLoader) RoleMatrix(context.Context) (access.Matrix, error) {
	return access.Matrix{MatrixStatus: access.MatrixPending, Roles: []access.MatrixRole{}, Permissions: []access.PermissionInfo{}}, nil
}

type harness struct {
	router *gin.Engine
	loader *fakeLoader
	userID string
	cookie *http.Cookie
	csrf   string
}

func newHarness(t *testing.T, allowProposed bool) *harness {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := authtest.NewStore()
	u := store.AddUser("asha", "Asha Verma", password, auth.StatusActive)
	svc, err := auth.NewService(store, auth.Config{IdleTimeout: time.Hour, AbsoluteTimeout: 2 * time.Hour},
		auth.NewLoginThrottle(5, 20, time.Minute), log)
	if err != nil {
		t.Fatal(err)
	}
	authH := auth.NewHandler(svc, auth.CookieConfig{}, log)
	loader := &fakeLoader{approved: map[string][]string{}, proposed: map[string][]string{}, scope: map[string][]access.Location{}}
	accessH := access.NewHandler(loader, allowProposed, log)

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(authH.RequireSession(auth.PublicRoutes("/api/v1")...))
	authH.Register(api)
	accessH.Register(api)
	// A stand-in for a future sensitive route and an object-level location check.
	api.GET("/costs", accessH.Require(access.PermCostPriceView), func(c *gin.Context) { httpx.OK(c, "costs") })
	api.POST("/costs", accessH.Require(access.PermCostPriceEdit), func(c *gin.Context) { httpx.OK(c, "saved") })
	api.GET("/locations/:id/stock", func(c *gin.Context) {
		a, ok := accessH.Current(c)
		if !ok {
			return
		}
		if err := a.Scope.Check(c.Param("id")); err != nil {
			access.Forbidden(c)
			return
		}
		httpx.OK(c, "stock")
	})

	h := &harness{router: r, loader: loader, userID: u.ID}
	w := h.do(http.MethodPost, "/api/v1/auth/login", `{"username":"asha","password":"`+password+`"}`, nil)
	var body struct {
		Data auth.SessionResponse `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	for _, c := range w.Result().Cookies() {
		if c.Name == "maa_session" {
			h.cookie = c
		}
	}
	h.csrf = body.Data.CSRFToken
	if h.cookie == nil {
		t.Fatalf("login failed: %s", w.Body)
	}
	return h
}

func (h *harness) do(method, path, body string, cookie *http.Cookie, headers ...string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, req)
	return w
}

func code(w *httptest.ResponseRecorder) string {
	var body httpx.ErrorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return body.Error.Code
}

func TestMeReturnsThePermissionsContract(t *testing.T) {
	h := newHarness(t, false)
	if w := h.do(http.MethodGet, "/api/v1/me", "", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("without session: %d", w.Code)
	}
	h.loader.approved[h.userID] = []string{access.PermRolesView}
	h.loader.scope[h.userID] = []access.Location{{ID: "loc-1", Code: "SR1", Name: "Showroom 1"}}
	w := h.do(http.MethodGet, "/api/v1/me", "", h.cookie)
	var body struct {
		Data access.MeResponse `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	d := body.Data
	if w.Code != http.StatusOK || d.User.Username != "asha" || len(d.Permissions) != 1 || d.Permissions[0] != access.PermRolesView ||
		d.LocationScope.All || len(d.LocationScope.Locations) != 1 || d.ProposedGrantsActive {
		t.Fatalf("status=%d body=%s", w.Code, w.Body)
	}
}

func TestMeWithNothingGrantedReturnsEmptyLists(t *testing.T) {
	h := newHarness(t, false)
	w := h.do(http.MethodGet, "/api/v1/me", "", h.cookie)
	if !strings.Contains(w.Body.String(), `"permissions":[]`) || !strings.Contains(w.Body.String(), `"locations":[]`) {
		t.Errorf("want empty arrays, not null: %s", w.Body)
	}
}

// Direct API access without an approved permission is denied.
func TestDirectAccessWithoutPermissionIsDenied(t *testing.T) {
	h := newHarness(t, false)
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/access/roles"},
		{http.MethodGet, "/api/v1/costs"},
		{http.MethodPost, "/api/v1/costs"},
	} {
		w := h.do(tc.method, tc.path, "{}", h.cookie, auth.CSRFHeader, h.csrf)
		if w.Code != http.StatusForbidden || code(w) != httpx.CodeForbidden {
			t.Errorf("%s %s: status=%d code=%s", tc.method, tc.path, w.Code, code(w))
		}
		if strings.Contains(w.Body.String(), "cost_price") || strings.Contains(w.Body.String(), "access.roles") {
			t.Errorf("%s %s: the 403 must not name the missing permission: %s", tc.method, tc.path, w.Body)
		}
	}
	// Without a session it is 401, before any permission check.
	if w := h.do(http.MethodGet, "/api/v1/costs", "", nil); w.Code != http.StatusUnauthorized {
		t.Errorf("no session: %d", w.Code)
	}
}

func TestGrantedPermissionAllowsOnlyThatAction(t *testing.T) {
	h := newHarness(t, false)
	h.loader.approved[h.userID] = []string{access.PermCostPriceView}
	if w := h.do(http.MethodGet, "/api/v1/costs", "", h.cookie); w.Code != http.StatusOK {
		t.Errorf("view with grant: %d", w.Code)
	}
	// Viewing does not imply editing (no escalation from a neighbouring permission).
	if w := h.do(http.MethodPost, "/api/v1/costs", "{}", h.cookie, auth.CSRFHeader, h.csrf); w.Code != http.StatusForbidden {
		t.Errorf("edit with view-only grant: %d", w.Code)
	}
}

func TestProposedGrantsCountOnlyWhenSwitchedOn(t *testing.T) {
	off := newHarness(t, false)
	off.loader.proposed[off.userID] = []string{access.PermCostPriceView}
	if w := off.do(http.MethodGet, "/api/v1/costs", "", off.cookie); w.Code != http.StatusForbidden {
		t.Errorf("proposed grant counted with the switch off: %d", w.Code)
	}
	on := newHarness(t, true)
	on.loader.proposed[on.userID] = []string{access.PermCostPriceView}
	if w := on.do(http.MethodGet, "/api/v1/costs", "", on.cookie); w.Code != http.StatusOK {
		t.Errorf("proposed grant ignored with the switch on: %d", w.Code)
	}
	if w := on.do(http.MethodGet, "/api/v1/me", "", on.cookie); !strings.Contains(w.Body.String(), `"proposedGrantsActive":true`) {
		t.Errorf("/me must say proposed grants are active: %s", w.Body)
	}
}

// Privilege escalation: nothing the client sends can add permissions, and the
// API has no way to change grants (only the audited cmd/access tool can).
func TestNoPrivilegeEscalationThroughTheAPI(t *testing.T) {
	h := newHarness(t, false)
	h.loader.approved[h.userID] = []string{access.PermRolesView}
	w := h.do(http.MethodGet, "/api/v1/costs", "", h.cookie,
		"X-Permissions", access.PermCostPriceView, "X-Role", "owner", "X-User-Id", "someone-else")
	if w.Code != http.StatusForbidden {
		t.Errorf("client-supplied headers granted access: %d", w.Code)
	}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		w := h.do(method, "/api/v1/access/roles", `{"role":"viewer","permission":"access.roles.manage"}`, h.cookie, auth.CSRFHeader, h.csrf)
		if w.Code != http.StatusMethodNotAllowed && w.Code != http.StatusNotFound {
			t.Errorf("%s /access/roles: status %d; grants must not be changeable over HTTP", method, w.Code)
		}
	}
}

func TestCrossLocationAccessIsDenied(t *testing.T) {
	h := newHarness(t, false)
	h.loader.scope[h.userID] = []access.Location{{ID: "loc-1", Code: "SR1", Name: "Showroom 1"}}
	if w := h.do(http.MethodGet, "/api/v1/locations/loc-1/stock", "", h.cookie); w.Code != http.StatusOK {
		t.Errorf("own location: %d", w.Code)
	}
	if w := h.do(http.MethodGet, "/api/v1/locations/loc-2/stock", "", h.cookie); w.Code != http.StatusForbidden {
		t.Errorf("other location: %d", w.Code)
	}
	h.loader.approved[h.userID] = []string{access.PermAllLocations}
	if w := h.do(http.MethodGet, "/api/v1/locations/loc-2/stock", "", h.cookie); w.Code != http.StatusOK {
		t.Errorf("locations.all: %d", w.Code)
	}
}

func TestAccessIsLoadedOncePerRequestAndFailuresAre503(t *testing.T) {
	h := newHarness(t, false)
	h.loader.approved[h.userID] = []string{access.PermRolesView}
	h.loader.calls = 0
	h.do(http.MethodGet, "/api/v1/access/roles", "", h.cookie)
	if h.loader.calls != 1 {
		t.Errorf("LoadAccess called %d times", h.loader.calls)
	}
	h.loader.err = errors.New("connection reset")
	w := h.do(http.MethodGet, "/api/v1/costs", "", h.cookie)
	if w.Code != http.StatusServiceUnavailable || strings.Contains(w.Body.String(), "connection reset") {
		t.Errorf("status=%d body=%s", w.Code, w.Body)
	}
}

func TestRequireRejectsUnknownPermissionKeys(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Require must panic for a key that is not in the catalogue")
		}
	}()
	access.NewHandler(&fakeLoader{}, false, slog.New(slog.NewTextHandler(io.Discard, nil))).Require("catalogue.cost_prices.view")
}
