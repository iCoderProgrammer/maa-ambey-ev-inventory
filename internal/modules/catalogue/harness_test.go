package catalogue

import (
	"bytes"
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

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/httpx"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/auth"
)

const testPassword = "correct horse battery staple"

// allCataloguePermissions is every catalogue key (C-02).
var allCataloguePermissions = []string{
	access.PermCatalogueView, access.PermCatalogueManage, access.PermCatalogueVerify, access.PermCatalogueActivate,
	access.PermCatalogueArchive, access.PermBrandAuthorise, access.PermBrandActivate,
}

// grants serves Access per user ID; tests change grants at will.
type grants struct {
	mu    sync.Mutex
	perms map[string][]string
}

func (g *grants) LoadAccess(_ context.Context, userID string, _ bool) (access.Access, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return access.NewAccess(userID, nil, g.perms[userID], nil, false), nil
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

// newHarness mounts the real auth and catalogue routes over the given stores.
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
	g := &grants{perms: map[string][]string{}}
	r := gin.New()
	r.HandleMethodNotAllowed = true
	api := r.Group("/api/v1")
	api.Use(authH.RequireSession(auth.PublicRoutes("/api/v1")...))
	authH.Register(api)
	NewHandler(NewService(repo, log), access.NewHandler(g, false, log), log).Register(api)
	return &harness{t: t, router: r, grants: g}
}

func (h *harness) grant(s session, perms ...string) {
	h.grants.mu.Lock()
	defer h.grants.mu.Unlock()
	h.grants.perms[s.userID] = perms
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

// call sends JSON (body nil = no body) and returns the recorder.
func (h *harness) call(s session, method, path string, body any) *httptest.ResponseRecorder {
	h.t.Helper()
	var payload, ct string
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			h.t.Fatal(err)
		}
		payload, ct = string(b), "application/json"
	}
	return h.raw(method, "/api/v1"+path, payload, ct, &s)
}

// expect asserts the status and decodes data into out (when not nil).
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
		dec := json.NewDecoder(bytes.NewReader(env.Data))
		if err := dec.Decode(out); err != nil {
			t.Fatalf("decode %s: %v", env.Data, err)
		}
	}
}

// expectError asserts status and error code and returns the error body.
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
