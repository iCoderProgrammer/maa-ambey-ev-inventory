package stock_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/httpx"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/stock"
)

const testPassword = "correct horse battery staple"

// grants is an in-memory access source: permissions and location scope per user.
type grants struct {
	mu        sync.Mutex
	perms     map[string][]string
	locations map[string][]access.Location
}

func (g *grants) LoadAccess(_ context.Context, userID string, _ bool) (access.Access, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return access.NewAccess(userID, nil, g.perms[userID], g.locations[userID], false), nil
}

func (g *grants) RoleMatrix(context.Context) (access.Matrix, error) { return access.Matrix{}, nil }

type session struct {
	userID string
	cookie *http.Cookie
	csrf   string
}

type harness struct {
	t      *testing.T
	router *gin.Engine
	grants *grants
}

func newHarness(t *testing.T, authStore auth.Store, repo stock.Repository) *harness {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	authSvc, err := auth.NewService(authStore, auth.Config{IdleTimeout: time.Hour, AbsoluteTimeout: 2 * time.Hour},
		auth.NewLoginThrottle(50, 200, time.Minute), log)
	if err != nil {
		t.Fatal(err)
	}
	authH := auth.NewHandler(authSvc, auth.CookieConfig{}, log)
	h := &harness{t: t, grants: &grants{perms: map[string][]string{}, locations: map[string][]access.Location{}}}
	r := gin.New()
	r.HandleMethodNotAllowed = true
	api := r.Group("/api/v1")
	api.Use(authH.RequireSession(auth.PublicRoutes("/api/v1")...))
	authH.Register(api)
	stock.NewHandler(stock.NewService(repo, log), access.NewHandler(h.grants, false, log), log).Register(api)
	h.router = r
	return h
}

func (h *harness) grant(s session, perms ...string) {
	h.grants.mu.Lock()
	defer h.grants.mu.Unlock()
	h.grants.perms[s.userID] = perms
}

func (h *harness) scope(s session, ids ...string) {
	h.grants.mu.Lock()
	defer h.grants.mu.Unlock()
	locs := make([]access.Location, len(ids))
	for i, id := range ids {
		locs[i] = access.Location{ID: id}
	}
	h.grants.locations[s.userID] = locs
}

func (h *harness) login(username, id string) session {
	h.t.Helper()
	w := h.raw(http.MethodPost, "/api/v1/auth/login", `{"username":"`+username+`","password":"`+testPassword+`"}`, nil, nil)
	var body struct {
		Data auth.SessionResponse `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	s := session{userID: id, csrf: body.Data.CSRFToken}
	for _, c := range w.Result().Cookies() {
		if c.Name == "maa_session" {
			s.cookie = c
		}
	}
	if s.cookie == nil {
		h.t.Fatalf("login %s failed: %s", username, w.Body)
	}
	return s
}

// raw sends a request; headers may add Idempotency-Key or drop the CSRF token ("X-CSRF-Token": "").
func (h *harness) raw(method, target, body string, s *session, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if s != nil {
		req.AddCookie(s.cookie)
		if method != http.MethodGet {
			req.Header.Set(auth.CSRFHeader, s.csrf)
		}
	}
	for k, v := range headers {
		if v == "" {
			req.Header.Del(k)
		} else {
			req.Header.Set(k, v)
		}
	}
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, req)
	return w
}

// display calls put-on-display (on) or take-off-display.
func (h *harness) display(s session, unitID string, on bool, body, key string) *httptest.ResponseRecorder {
	path := "/api/v1/vehicle-units/" + unitID + "/take-off-display"
	if on {
		path = "/api/v1/vehicle-units/" + unitID + "/put-on-display"
	}
	return h.raw(http.MethodPost, path, body, &s, map[string]string{stock.IdempotencyHeader: key})
}

func expect(t *testing.T, w *httptest.ResponseRecorder, status int, out any) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status %d, want %d: %s", w.Code, status, w.Body)
	}
	if out != nil {
		var env struct {
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(env.Data, out); err != nil {
			t.Fatalf("decode %s: %v", env.Data, err)
		}
	}
}

func expectError(t *testing.T, w *httptest.ResponseRecorder, status int, code string) httpx.ErrorBody {
	t.Helper()
	var env httpx.ErrorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	if w.Code != status || env.Error.Code != code {
		t.Fatalf("got %d %s, want %d %s: %s", w.Code, env.Error.Code, status, code, w.Body)
	}
	return env.Error
}
