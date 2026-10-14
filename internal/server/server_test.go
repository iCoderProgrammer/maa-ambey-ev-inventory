package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/config"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/httpx"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/middleware"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/attachments"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/auth"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/auth/authtest"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/catalogue"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/pricing"
)

type okPinger struct{}

func (okPinger) Ping(context.Context) error { return nil }

// testAuth builds the real auth handler over an in-memory store.
func testAuth(t *testing.T) *auth.Handler {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc, err := auth.NewService(authtest.NewStore(), auth.Config{IdleTimeout: time.Hour, AbsoluteTimeout: 2 * time.Hour},
		auth.NewLoginThrottle(5, 20, time.Minute), log)
	if err != nil {
		t.Fatal(err)
	}
	return auth.NewHandler(svc, auth.CookieConfig{}, log)
}

// noAccess grants nothing, like a fresh database before owner sign-off.
type noAccess struct{}

func (noAccess) LoadAccess(_ context.Context, userID string, _ bool) (access.Access, error) {
	return access.NewAccess(userID, nil, nil, nil, false), nil
}

func (noAccess) RoleMatrix(context.Context) (access.Matrix, error) { return access.Matrix{}, nil }

func testAccess() *access.Handler {
	return access.NewHandler(noAccess{}, false, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

// testAttachments builds the real attachment routes over a temporary
// directory; with noAccess every request stops at the permission guard.
func testAttachments(t *testing.T) *attachments.Handler {
	t.Helper()
	store, err := attachments.NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	signer, err := attachments.NewSigner("", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return attachments.NewHandler(attachments.NewService(nil, store, signer, 1<<20, log), testAccess(), APIPrefix, log)
}

// testCatalogue builds the real catalogue routes without a database; with
// noAccess every request stops at the permission guard.
func testCatalogue() *catalogue.Handler {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return catalogue.NewHandler(catalogue.NewService(nil, log), testAccess(), log)
}

// testPricing builds the real price routes without a database.
func testPricing() *pricing.Handler {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return pricing.NewHandler(pricing.NewService(nil, log), testAccess(), log)
}

type okSchema struct{}

func (okSchema) CheckSchema(context.Context) error { return nil }

func testConfig() config.Config {
	return config.Config{
		AppEnv:             config.EnvTest,
		CORSAllowedOrigins: []string{"http://localhost:5173"},
		RequestTimeout:     5 * time.Second,
		RateLimitRPS:       1000,
		RateLimitBurst:     1000,
		SwaggerEnabled:     true,
	}
}

func newTestRouter(t *testing.T, cfg config.Config) *gin.Engine {
	t.Helper()
	r, err := NewRouter(Deps{Config: cfg, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), DB: okPinger{}, Schema: okSchema{}, Auth: testAuth(t), Access: testAccess(), Attachments: testAttachments(t), Catalogue: testCatalogue(), Pricing: testPricing(), Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func decodeError(t *testing.T, w *httptest.ResponseRecorder) httpx.ErrorBody {
	t.Helper()
	var body httpx.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid error envelope %q: %v", w.Body.String(), err)
	}
	return body.Error
}

func TestHealthRouteIsVersioned(t *testing.T) {
	r := newTestRouter(t, testConfig())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if w.Header().Get(middleware.RequestIDHeader) == "" {
		t.Error("missing X-Request-ID response header")
	}
}

func TestUnknownRouteReturnsErrorEnvelope(t *testing.T) {
	r := newTestRouter(t, testConfig())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/does-not-exist", nil)
	req.Header.Set(middleware.RequestIDHeader, "client-req-12345")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d", w.Code)
	}
	e := decodeError(t, w)
	if e.Code != httpx.CodeNotFound || e.RequestID != "client-req-12345" {
		t.Errorf("error = %+v", e)
	}
}

func TestWrongMethodReturns405(t *testing.T) {
	r := newTestRouter(t, testConfig())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/health", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", w.Code)
	}
	if e := decodeError(t, w); e.Code != httpx.CodeMethodNotAllowed {
		t.Errorf("code = %s", e.Code)
	}
}

func TestPanicIsRecoveredWithoutLeakingDetails(t *testing.T) {
	r := newTestRouter(t, testConfig())
	r.GET("/api/v1/boom", func(*gin.Context) { panic("db password is hunter2") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/boom", nil))
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "hunter2") || strings.Contains(w.Body.String(), "goroutine") {
		t.Errorf("response leaked panic details: %s", w.Body.String())
	}
	e := decodeError(t, w)
	if e.Code != httpx.CodeInternal || e.RequestID == "" {
		t.Errorf("error = %+v", e)
	}
}

func TestRateLimitReturns429(t *testing.T) {
	cfg := testConfig()
	cfg.RateLimitRPS = 1
	cfg.RateLimitBurst = 2
	r := newTestRouter(t, cfg)

	var last *httptest.ResponseRecorder
	for range 3 {
		last = httptest.NewRecorder()
		r.ServeHTTP(last, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	}
	if last.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d", last.Code)
	}
	if last.Header().Get("Retry-After") == "" {
		t.Error("missing Retry-After")
	}
	if e := decodeError(t, last); e.Code != httpx.CodeRateLimited {
		t.Errorf("code = %s", e.Code)
	}
}

func TestCORSAllowsOnlyConfiguredOrigins(t *testing.T) {
	r := newTestRouter(t, testConfig())

	for origin, allowed := range map[string]bool{"http://localhost:5173": true, "https://evil.example": false} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/health", nil)
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		r.ServeHTTP(w, req)
		got := w.Header().Get("Access-Control-Allow-Origin") == origin
		if got != allowed {
			t.Errorf("origin %s: allowed=%v, want %v", origin, got, allowed)
		}
	}
}

func TestSwaggerCanBeDisabled(t *testing.T) {
	cfg := testConfig()
	cfg.SwaggerEnabled = false
	r := newTestRouter(t, cfg)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestTimeoutReturnsErrorEnvelope(t *testing.T) {
	cfg := testConfig()
	cfg.RequestTimeout = 20 * time.Millisecond
	r := newTestRouter(t, cfg)
	r.GET("/api/v1/slow", func(c *gin.Context) {
		<-c.Request.Context().Done() // a ctx-aware handler (e.g. a pgx query) gives up silently
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/slow", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	if e := decodeError(t, w); e.Code != httpx.CodeTimeout || e.RequestID == "" {
		t.Errorf("error = %+v", e)
	}
}

func TestTimeoutKeepsHandlerResponse(t *testing.T) {
	cfg := testConfig()
	cfg.RequestTimeout = 20 * time.Millisecond
	r := newTestRouter(t, cfg)
	r.GET("/api/v1/slow-but-answered", func(c *gin.Context) {
		<-c.Request.Context().Done()
		httpx.Error(c, http.StatusConflict, httpx.CodeConflict, "handled")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/slow-but-answered", nil))
	if w.Code != http.StatusConflict || decodeError(t, w).Code != httpx.CodeConflict {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
}

func TestPanicIsAccessLoggedWithRequestID(t *testing.T) {
	var buf strings.Builder
	r, err := NewRouter(Deps{Config: testConfig(), Log: slog.New(slog.NewJSONHandler(&buf, nil)), DB: okPinger{}, Schema: okSchema{}, Auth: testAuth(t), Access: testAccess(), Attachments: testAttachments(t), Catalogue: testCatalogue(), Pricing: testPricing(), Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	r.GET("/api/v1/boom", func(*gin.Context) { panic("boom") })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/boom", nil)
	req.Header.Set(middleware.RequestIDHeader, "trace-12345678")
	r.ServeHTTP(httptest.NewRecorder(), req)

	var accessLogged bool
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("log line is not JSON: %q", line)
		}
		if entry["msg"] == "http_request" {
			accessLogged = true
			if entry["status"] != float64(http.StatusInternalServerError) || entry["request_id"] != "trace-12345678" || entry["route"] != "/api/v1/boom" {
				t.Errorf("access log = %v", entry)
			}
		}
	}
	if !accessLogged {
		t.Errorf("no http_request log line for a recovered panic; logs:\n%s", buf.String())
	}
}

func TestCORSRejectionUsesErrorEnvelope(t *testing.T) {
	r := newTestRouter(t, testConfig())

	for _, method := range []string{http.MethodGet, http.MethodOptions} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/api/v1/health", nil)
		req.Header.Set("Origin", "https://evil.example")
		req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("%s: status = %d", method, w.Code)
		}
		if e := decodeError(t, w); e.Code != httpx.CodeForbidden || e.RequestID == "" {
			t.Errorf("%s: error = %+v", method, e)
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("%s: rejected origin got CORS allow header", method)
		}
	}
}

func TestCORSIgnoresSameHostOrigin(t *testing.T) {
	r := newTestRouter(t, testConfig())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.Host = "api.internal:8080"
	req.Header.Set("Origin", "http://api.internal:8080")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("same-host origin: status = %d, body %s", w.Code, w.Body.String())
	}
}

func TestNewRouterRequiresSchemaCheckerAndAuth(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if _, err := NewRouter(Deps{Config: testConfig(), Log: log, DB: okPinger{}, Auth: testAuth(t), Access: testAccess(), Attachments: testAttachments(t), Catalogue: testCatalogue(), Pricing: testPricing(), Version: "test"}); err == nil {
		t.Error("NewRouter must refuse to start without a schema checker")
	}
	if _, err := NewRouter(Deps{Config: testConfig(), Log: log, DB: okPinger{}, Schema: okSchema{}, Access: testAccess(), Attachments: testAttachments(t), Catalogue: testCatalogue(), Pricing: testPricing(), Version: "test"}); err == nil {
		t.Error("NewRouter must refuse to start without authentication")
	}
	if _, err := NewRouter(Deps{Config: testConfig(), Log: log, DB: okPinger{}, Schema: okSchema{}, Auth: testAuth(t), Version: "test"}); err == nil {
		t.Error("NewRouter must refuse to start without access control")
	}
}

// Only health, readiness and the sign-in routes are reachable without a session.
func TestAPIRoutesArePrivateByDefault(t *testing.T) {
	r, err := NewRouter(Deps{Config: testConfig(), Log: slog.New(slog.NewTextHandler(io.Discard, nil)), DB: okPinger{}, Schema: okSchema{}, Auth: testAuth(t), Access: testAccess(), Attachments: testAttachments(t), Catalogue: testCatalogue(), Pricing: testPricing(), Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	public := map[string]bool{
		"GET /api/v1/health": true, "GET /api/v1/ready": true,
		"POST /api/v1/auth/login": true, "GET /api/v1/auth/session": true,
	}
	for _, route := range r.Routes() {
		if !strings.HasPrefix(route.Path, APIPrefix) {
			continue
		}
		key := route.Method + " " + route.Path
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(route.Method, route.Path, strings.NewReader("{}")))
		if public[key] {
			if w.Code == http.StatusUnauthorized && key != "GET /api/v1/auth/session" {
				t.Errorf("%s is public but answered 401", key)
			}
			continue
		}
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s without a session: status %d, want 401", key, w.Code)
		}
	}
}

// B-08: audit history is append-only. No API route may change or remove audit
// records; a future audit read API is allowed only with GET.
func TestNoAPIRouteCanModifyAuditEvents(t *testing.T) {
	r, err := NewRouter(Deps{Config: testConfig(), Log: slog.New(slog.NewTextHandler(io.Discard, nil)), DB: okPinger{}, Schema: okSchema{}, Auth: testAuth(t), Access: testAccess(), Attachments: testAttachments(t), Catalogue: testCatalogue(), Pricing: testPricing(), Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range r.Routes() {
		if strings.Contains(strings.ToLower(route.Path), "audit") && route.Method != http.MethodGet {
			t.Errorf("%s %s could modify audit events; audit history is append-only", route.Method, route.Path)
		}
	}
	for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(method, APIPrefix+"/audit-events/1", nil))
		if w.Code != http.StatusNotFound && w.Code != http.StatusMethodNotAllowed && w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s/audit-events/1: status %d", method, APIPrefix, w.Code)
		}
	}
}
