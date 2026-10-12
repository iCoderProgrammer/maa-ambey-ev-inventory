package auth_test

import (
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

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/httpx"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/auth"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/auth/authtest"
)

const password = "correct horse battery staple"

type harness struct {
	t      *testing.T
	store  *authtest.Store
	router *gin.Engine
	now    time.Time
}

func newHarness(t *testing.T, secure bool) *harness {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := &harness{t: t, store: authtest.NewStore(), now: time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)}
	h.store.AddUser("asha", "Asha Verma", password, auth.StatusActive)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc, err := auth.NewService(h.store, auth.Config{
		IdleTimeout:     2 * time.Hour,
		AbsoluteTimeout: 12 * time.Hour,
		Now:             func() time.Time { return h.now },
	}, auth.NewLoginThrottle(5, 20, 15*time.Minute), log)
	if err != nil {
		t.Fatal(err)
	}
	handler := auth.NewHandler(svc, auth.CookieConfig{Secure: secure}, log)

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(handler.RequireSession(append(auth.PublicRoutes("/api/v1"), "GET /api/v1/open")...))
	handler.Register(api)
	// Stand-ins for future feature routes: protected unless listed as public.
	api.GET("/open", func(c *gin.Context) { httpx.OK(c, "open") })
	api.GET("/things", func(c *gin.Context) {
		p, _ := auth.CurrentPrincipal(c)
		httpx.OK(c, p.Username)
	})
	api.POST("/things", func(c *gin.Context) { httpx.OK(c, "created") })
	h.router = r
	return h
}

type response struct {
	*httptest.ResponseRecorder
}

func (r response) errorCode() string {
	var body httpx.ErrorResponse
	_ = json.Unmarshal(r.Body.Bytes(), &body)
	return body.Error.Code
}

func (r response) cookie(name string) *http.Cookie {
	for _, c := range r.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func (h *harness) do(method, path, body string, headers map[string]string, cookies ...*http.Cookie) response {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, req)
	return response{w}
}

func (h *harness) login(username, pw string) response {
	b, _ := json.Marshal(map[string]string{"username": username, "password": pw})
	return h.do(http.MethodPost, "/api/v1/auth/login", string(b), nil)
}

// signIn logs in and returns the session cookie and CSRF token.
func (h *harness) signIn() (*http.Cookie, string) {
	h.t.Helper()
	w := h.login("asha", password)
	if w.Code != http.StatusOK {
		h.t.Fatalf("login status = %d body=%s", w.Code, w.Body)
	}
	var body struct {
		Data auth.SessionResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		h.t.Fatal(err)
	}
	return w.cookie("maa_session"), body.Data.CSRFToken
}

func TestLoginSetsHardenedSessionCookie(t *testing.T) {
	h := newHarness(t, false)
	w := h.login("  ASHA ", password)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body)
	}
	c := w.cookie("maa_session")
	if c == nil || len(c.Value) < 40 {
		t.Fatalf("session cookie missing or short: %+v", c)
	}
	if !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.Path != "/" || c.MaxAge != 0 || !c.Expires.IsZero() {
		t.Errorf("cookie attributes = %+v; want HttpOnly, SameSite=Strict, Path=/, browser-session lifetime", c)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Error("login response must not be cached")
	}

	var body struct {
		Data auth.SessionResponse `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Data.User.Username != "asha" || body.Data.User.DisplayName != "Asha Verma" || len(body.Data.CSRFToken) < 40 {
		t.Errorf("body = %+v", body.Data)
	}
	if want := h.now.Add(2 * time.Hour); !body.Data.ExpiresAt.Equal(want) {
		t.Errorf("expiresAt = %v, want idle expiry %v", body.Data.ExpiresAt, want)
	}
	if strings.Contains(w.Body.String(), c.Value) || strings.Contains(w.Body.String(), "argon2") {
		t.Error("response body must not contain the session token or password hash")
	}
}

func TestSecureCookieUsesHostPrefix(t *testing.T) {
	h := newHarness(t, true)
	c := h.login("asha", password).cookie("__Host-maa_session")
	if c == nil || !c.Secure {
		t.Fatalf("want a Secure __Host- cookie, got %+v", c)
	}
}

func TestLoginFailuresAreGeneric(t *testing.T) {
	h := newHarness(t, false)
	wrong := h.login("asha", "wrong password here")
	unknown := h.login("nobody", "wrong password here")
	for name, w := range map[string]response{"wrong password": wrong, "unknown user": unknown} {
		if w.Code != http.StatusUnauthorized || w.errorCode() != httpx.CodeInvalidCredentials {
			t.Errorf("%s: status=%d code=%s", name, w.Code, w.errorCode())
		}
		if w.cookie("maa_session") != nil {
			t.Errorf("%s: must not set a cookie", name)
		}
	}
	var a, b httpx.ErrorResponse
	_ = json.Unmarshal(wrong.Body.Bytes(), &a)
	_ = json.Unmarshal(unknown.Body.Bytes(), &b)
	if a.Error.Message != b.Error.Message {
		t.Errorf("messages differ: %q vs %q", a.Error.Message, b.Error.Message)
	}
}

func TestDisabledUserCannotSignIn(t *testing.T) {
	h := newHarness(t, false)
	h.store.SetStatus("asha", auth.StatusDisabled)

	if w := h.login("asha", password); w.Code != http.StatusForbidden || w.errorCode() != httpx.CodeAccountDisabled {
		t.Errorf("correct password: status=%d code=%s", w.Code, w.errorCode())
	}
	// Without the password, a disabled account looks like any failed sign-in.
	if w := h.login("asha", "wrong password here"); w.errorCode() != httpx.CodeInvalidCredentials {
		t.Errorf("wrong password: code=%s", w.errorCode())
	}
}

func TestDisablingUserEndsExistingSession(t *testing.T) {
	h := newHarness(t, false)
	cookie, _ := h.signIn()
	h.store.SetStatus("asha", auth.StatusDisabled)

	w := h.do(http.MethodGet, "/api/v1/things", "", nil, cookie)
	if w.Code != http.StatusUnauthorized || w.errorCode() != httpx.CodeSessionExpired {
		t.Fatalf("status=%d code=%s", w.Code, w.errorCode())
	}
	if c := w.cookie("maa_session"); c == nil || c.MaxAge >= 0 {
		t.Error("an ended session must clear the cookie")
	}
}

func TestLoginValidation(t *testing.T) {
	h := newHarness(t, false)

	w := h.do(http.MethodPost, "/api/v1/auth/login", `{"username":"  ","password":""}`, nil)
	if w.Code != http.StatusBadRequest || w.errorCode() != httpx.CodeValidationFailed {
		t.Fatalf("empty: status=%d code=%s", w.Code, w.errorCode())
	}
	var body httpx.ErrorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if len(body.Error.Details) != 2 {
		t.Errorf("details = %+v", body.Error.Details)
	}

	long := `{"username":"asha","password":"` + strings.Repeat("x", 257) + `"}`
	if w := h.do(http.MethodPost, "/api/v1/auth/login", long, nil); w.errorCode() != httpx.CodeValidationFailed {
		t.Errorf("long password: code=%s", w.errorCode())
	}
	if w := h.do(http.MethodPost, "/api/v1/auth/login", `{"username":`, nil); w.errorCode() != httpx.CodeBadRequest {
		t.Errorf("malformed JSON: code=%s", w.errorCode())
	}
	form := h.do(http.MethodPost, "/api/v1/auth/login", "", map[string]string{"Content-Type": "application/x-www-form-urlencoded"})
	if form.Code != http.StatusUnsupportedMediaType {
		t.Errorf("form post: status=%d", form.Code)
	}
}

func TestRepeatedFailuresAreThrottled(t *testing.T) {
	h := newHarness(t, false)
	for i := range 5 {
		if w := h.login("asha", "wrong password here"); w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status=%d", i+1, w.Code)
		}
	}
	w := h.login("asha", password)
	if w.Code != http.StatusTooManyRequests || w.errorCode() != httpx.CodeRateLimited {
		t.Fatalf("status=%d code=%s", w.Code, w.errorCode())
	}
	if w.Header().Get("Retry-After") != "900" {
		t.Errorf("Retry-After = %q, want 900", w.Header().Get("Retry-After"))
	}
	// Window expiry is covered in throttle_test.go (the throttle has its own clock).
}

func TestSessionEndpoint(t *testing.T) {
	h := newHarness(t, false)
	if w := h.do(http.MethodGet, "/api/v1/auth/session", "", nil); w.Code != http.StatusUnauthorized || w.errorCode() != httpx.CodeUnauthenticated {
		t.Errorf("no cookie: status=%d code=%s", w.Code, w.errorCode())
	}
	bogus := &http.Cookie{Name: "maa_session", Value: "forged-token"}
	if w := h.do(http.MethodGet, "/api/v1/auth/session", "", nil, bogus); w.errorCode() != httpx.CodeSessionExpired {
		t.Errorf("forged cookie: code=%s", w.errorCode())
	}

	cookie, csrf := h.signIn()
	w := h.do(http.MethodGet, "/api/v1/auth/session", "", nil, cookie)
	var body struct {
		Data auth.SessionResponse `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if w.Code != http.StatusOK || body.Data.CSRFToken != csrf || body.Data.User.Username != "asha" {
		t.Errorf("status=%d body=%+v", w.Code, body.Data)
	}
}

func TestDenyByDefault(t *testing.T) {
	h := newHarness(t, false)
	if w := h.do(http.MethodGet, "/api/v1/open", "", nil); w.Code != http.StatusOK {
		t.Errorf("public route: status=%d", w.Code)
	}
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/things"},
		{http.MethodPost, "/api/v1/things"},
		{http.MethodPost, "/api/v1/auth/logout"},
	} {
		w := h.do(tc.method, tc.path, "{}", nil)
		if w.Code != http.StatusUnauthorized || w.errorCode() != httpx.CodeUnauthenticated {
			t.Errorf("%s %s without session: status=%d code=%s", tc.method, tc.path, w.Code, w.errorCode())
		}
	}

	cookie, csrf := h.signIn()
	if w := h.do(http.MethodGet, "/api/v1/things", "", nil, cookie); w.Code != http.StatusOK {
		t.Errorf("GET with session: status=%d", w.Code)
	}
	if w := h.do(http.MethodPost, "/api/v1/things", "{}", nil, cookie); w.Code != http.StatusForbidden || w.errorCode() != httpx.CodeCSRFFailed {
		t.Errorf("POST without CSRF: status=%d code=%s", w.Code, w.errorCode())
	}
	if w := h.do(http.MethodPost, "/api/v1/things", "{}", map[string]string{auth.CSRFHeader: csrf + "x"}, cookie); w.errorCode() != httpx.CodeCSRFFailed {
		t.Errorf("POST with wrong CSRF: code=%s", w.errorCode())
	}
	if w := h.do(http.MethodPost, "/api/v1/things", "{}", map[string]string{auth.CSRFHeader: csrf}, cookie); w.Code != http.StatusOK {
		t.Errorf("POST with CSRF: status=%d", w.Code)
	}
}

func TestLogoutRevokesSession(t *testing.T) {
	h := newHarness(t, false)
	cookie, csrf := h.signIn()

	if w := h.do(http.MethodPost, "/api/v1/auth/logout", "", nil, cookie); w.errorCode() != httpx.CodeCSRFFailed {
		t.Fatalf("logout without CSRF: code=%s", w.errorCode())
	}
	w := h.do(http.MethodPost, "/api/v1/auth/logout", "", map[string]string{auth.CSRFHeader: csrf}, cookie)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"signedOut":true`) {
		t.Fatalf("logout: status=%d body=%s", w.Code, w.Body)
	}
	if c := w.cookie("maa_session"); c == nil || c.MaxAge >= 0 {
		t.Error("logout must clear the cookie")
	}
	// The old cookie is dead on the server even if a client keeps sending it.
	if w := h.do(http.MethodGet, "/api/v1/things", "", nil, cookie); w.errorCode() != httpx.CodeSessionExpired {
		t.Errorf("after logout: code=%s", w.errorCode())
	}
	if got := h.store.RevokedReasons(); len(got) != 1 || got[0] != auth.RevokedLogout {
		t.Errorf("revoked reasons = %v", got)
	}
}

func TestSessionExpiry(t *testing.T) {
	h := newHarness(t, false)
	cookie, _ := h.signIn()

	// Activity within the idle timeout keeps the session alive…
	for range 5 {
		h.now = h.now.Add(90 * time.Minute)
		if w := h.do(http.MethodGet, "/api/v1/things", "", nil, cookie); w.Code != http.StatusOK {
			t.Fatalf("at %v: status=%d", h.now, w.Code)
		}
	}
	// …but never past the absolute lifetime (12h).
	h.now = time.Date(2026, 10, 12, 21, 0, 0, 0, time.UTC)
	if w := h.do(http.MethodGet, "/api/v1/things", "", nil, cookie); w.errorCode() != httpx.CodeSessionExpired {
		t.Errorf("after absolute timeout: code=%s", w.errorCode())
	}

	cookie, _ = h.signIn()
	h.now = h.now.Add(2 * time.Hour)
	if w := h.do(http.MethodGet, "/api/v1/things", "", nil, cookie); w.errorCode() != httpx.CodeSessionExpired {
		t.Errorf("after idle timeout: code=%s", w.errorCode())
	}
}

func TestNewSignInReplacesOldSession(t *testing.T) {
	h := newHarness(t, false)
	first, _ := h.signIn()
	b, _ := json.Marshal(map[string]string{"username": "asha", "password": password})
	w := h.do(http.MethodPost, "/api/v1/auth/login", string(b), nil, first)
	second := w.cookie("maa_session")
	if second == nil || second.Value == first.Value {
		t.Fatal("each sign-in must issue a new token")
	}
	if got := h.store.RevokedReasons(); got[0] != auth.RevokedSuperseded || got[1] != "" {
		t.Errorf("revoked reasons = %v", got)
	}
}

func TestSessionLookupFailureIsNotTreatedAsSignedOut(t *testing.T) {
	h := newHarness(t, false)
	cookie, _ := h.signIn()
	h.store.FailNext = errors.New("connection reset")
	w := h.do(http.MethodGet, "/api/v1/things", "", nil, cookie)
	if w.Code != http.StatusServiceUnavailable || w.cookie("maa_session") != nil {
		t.Fatalf("status=%d; a database error must not end the session", w.Code)
	}
	if strings.Contains(w.Body.String(), "connection reset") {
		t.Error("internal error leaked")
	}
}

func TestAuthEventsAreAuditedWithoutSecrets(t *testing.T) {
	h := newHarness(t, false)
	h.login("asha", "wrong password here")
	h.login("ghost", "wrong password here")
	cookie, csrf := h.signIn()
	h.do(http.MethodPost, "/api/v1/auth/logout", "", map[string]string{auth.CSRFHeader: csrf}, cookie)
	h.store.SetStatus("asha", auth.StatusDisabled)
	h.login("asha", password)

	want := []string{"auth.sign_in_failed", "auth.signed_in", "auth.signed_out", "auth.sign_in_refused"}
	got := h.store.Actions()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("audited actions = %v, want %v (unknown usernames are not audited)", got, want)
	}
	for _, ev := range h.store.Events {
		b, _ := json.Marshal(ev)
		for _, secret := range []string{password, "wrong password here", cookie.Value, csrf, "argon2"} {
			if strings.Contains(string(b), secret) {
				t.Errorf("%s event contains a secret", ev.Action)
			}
		}
	}
}
