package vehicles

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
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/ageing"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
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
	// clock is the business calendar's clock (E-07); setToday fixes it.
	mu    sync.Mutex
	clock time.Time
}

// setToday fixes the business date (noon in the business time zone).
func (h *harness) setToday(date string) {
	d, err := time.Parse(time.DateOnly, date)
	if err != nil {
		h.t.Fatal(err)
	}
	loc, _ := time.LoadLocation(ageing.DefaultTimeZone)
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clock = time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, loc)
}

func (h *harness) now() time.Time {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clock.IsZero() {
		return time.Now()
	}
	return h.clock
}

func newHarness(t *testing.T, authStore auth.Store, repo Repository) *harness {
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
	accessH := access.NewHandler(h.grants, false, log)
	svc := NewService(repo, log)
	// With a database: the real band configuration and a controllable business clock (E-07).
	if pg, ok := repo.(*PGStore); ok {
		cal, err := ageing.NewCalendar(ageing.DefaultTimeZone, h.now)
		if err != nil {
			t.Fatal(err)
		}
		ageingSvc := ageing.NewService(ageing.NewPGStore(pg.pool), cal, log)
		svc.WithAgeing(ageingSvc)
		ageing.NewHandler(ageingSvc, accessH, log).Register(api)
	}
	NewHandler(svc, accessH, log).Register(api)
	h.router = r
	return h
}

func (h *harness) grant(s session, perms ...string) {
	h.grants.mu.Lock()
	defer h.grants.mu.Unlock()
	h.grants.perms[s.userID] = perms
}

// scope limits a user to locations (locations.all in the grants means every location).
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
	w := h.raw(http.MethodPost, "/api/v1/auth/login", `{"username":"`+username+`","password":"`+testPassword+`"}`, "application/json", nil)
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

func (h *harness) raw(method, target, body, contentType string, s *session) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if s != nil {
		req.AddCookie(s.cookie)
		if method != http.MethodGet {
			req.Header.Set(auth.CSRFHeader, s.csrf)
		}
	}
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, req)
	return w
}

func (h *harness) get(s session, path string) *httptest.ResponseRecorder {
	return h.raw(http.MethodGet, "/api/v1"+path, "", "", &s)
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

func hasField(details []httpx.FieldError, field string) bool {
	for _, d := range details {
		if d.Field == field {
			return true
		}
	}
	return false
}
