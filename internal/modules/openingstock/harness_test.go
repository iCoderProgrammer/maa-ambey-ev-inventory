package openingstock_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
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
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/attachments"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/openingstock"
)

const testPassword = "correct horse battery staple"

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
	svc    *openingstock.Service
	store  *attachments.LocalStore
}

// newHarness mounts the import routes. limits may be small to test the bounds;
// the business date is fixed at 2026-06-30 (Asia/Kolkata).
func newHarness(t *testing.T, authStore auth.Store, repo openingstock.Repository, limits openingstock.Limits) *harness {
	t.Helper()
	gin.SetMode(gin.TestMode)
	log := slog.New(slog.NewTextHandler(testLogOut, nil))
	authSvc, err := auth.NewService(authStore, auth.Config{IdleTimeout: time.Hour, AbsoluteTimeout: 2 * time.Hour},
		auth.NewLoginThrottle(50, 200, time.Minute), log)
	if err != nil {
		t.Fatal(err)
	}
	authH := auth.NewHandler(authSvc, auth.CookieConfig{}, log)
	store, err := attachments.NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation(ageing.DefaultTimeZone)
	cal, err := ageing.NewCalendar(ageing.DefaultTimeZone, func() time.Time { return time.Date(2026, 6, 30, 12, 0, 0, 0, loc) })
	if err != nil {
		t.Fatal(err)
	}
	h := &harness{t: t, grants: &grants{perms: map[string][]string{}, locations: map[string][]access.Location{}}, store: store}
	h.svc = openingstock.NewService(repo, store, cal, limits, log)
	r := gin.New()
	r.HandleMethodNotAllowed = true
	api := r.Group("/api/v1")
	api.Use(authH.RequireSession(auth.PublicRoutes("/api/v1")...))
	authH.Register(api)
	openingstock.NewHandler(h.svc, access.NewHandler(h.grants, false, log), log).Register(api)
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
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"username":"`+username+`","password":"`+testPassword+`"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, req)
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

func (h *harness) get(s *session, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1"+path, nil)
	if s != nil {
		req.AddCookie(s.cookie)
	}
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, req)
	return w
}

// upload posts a multipart file; key "" omits the Idempotency-Key.
func (h *harness) upload(s *session, name, contentType string, data []byte, key string) *httptest.ResponseRecorder {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	hdr := make(map[string][]string)
	hdr["Content-Disposition"] = []string{`form-data; name="file"; filename="` + name + `"`}
	if contentType != "" {
		hdr["Content-Type"] = []string{contentType}
	}
	part, _ := mw.CreatePart(hdr)
	_, _ = part.Write(data)
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/opening-stock-imports", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if s != nil {
		req.AddCookie(s.cookie)
		req.Header.Set(auth.CSRFHeader, s.csrf)
	}
	if key != "" {
		req.Header.Set(openingstock.IdempotencyHeader, key)
	}
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, req)
	return w
}

// post sends a JSON body (nil: none) with the CSRF token; key "" omits the Idempotency-Key.
func (h *harness) post(s *session, path string, body any, key string) *httptest.ResponseRecorder {
	var r io.Reader = http.NoBody
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1"+path, r)
	req.Header.Set("Content-Type", "application/json")
	if s != nil {
		req.AddCookie(s.cookie)
		req.Header.Set(auth.CSRFHeader, s.csrf)
	}
	if key != "" {
		req.Header.Set(openingstock.IdempotencyHeader, key)
	}
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, req)
	return w
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

// testLogOut receives the handlers' logs (io.Discard unless debugging).
var testLogOut io.Writer = io.Discard
