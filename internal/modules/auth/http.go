package auth

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/httpx"
)

// CSRFHeader carries the per-session CSRF token on state-changing requests.
const CSRFHeader = "X-CSRF-Token"

// principalKey is the gin context key holding the request's Principal.
const principalKey = "auth.principal"

// CookieConfig controls the session cookie.
type CookieConfig struct {
	// Secure marks the cookie HTTPS-only and enables the __Host- prefix.
	Secure bool
}

func (c CookieConfig) name() string {
	if c.Secure {
		// __Host-: Secure, Path=/, no Domain — cannot be set or shadowed by subdomains.
		return "__Host-maa_session"
	}
	return "maa_session"
}

// Handler serves the auth endpoints and the session middleware.
type Handler struct {
	svc    *Service
	cookie CookieConfig
	log    *slog.Logger
}

// NewHandler builds the HTTP adapter for the auth service.
func NewHandler(svc *Service, cookie CookieConfig, log *slog.Logger) *Handler {
	return &Handler{svc: svc, cookie: cookie, log: log}
}

// PublicRoutes are the auth routes reachable without a session (method + full path).
func PublicRoutes(prefix string) []string {
	return []string{"POST " + prefix + "/auth/login", "GET " + prefix + "/auth/session"}
}

// Register mounts the auth routes.
func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.POST("/auth/login", h.Login)
	rg.GET("/auth/session", h.Session)
	rg.POST("/auth/logout", h.Logout)
}

// CurrentPrincipal returns the authenticated principal set by RequireSession.
func CurrentPrincipal(c *gin.Context) (Principal, bool) {
	v, ok := c.Get(principalKey)
	if !ok {
		return Principal{}, false
	}
	p, ok := v.(Principal)
	return p, ok
}

// RequireSession denies by default: every route in the group needs a valid
// session unless its "METHOD /full/path" is listed in public. State-changing
// methods also need the session's CSRF token in X-CSRF-Token.
func (h *Handler) RequireSession(public ...string) gin.HandlerFunc {
	open := make(map[string]struct{}, len(public))
	for _, r := range public {
		open[r] = struct{}{}
	}
	return func(c *gin.Context) {
		if _, ok := open[c.Request.Method+" "+c.FullPath()]; ok {
			c.Next()
			return
		}
		p, ok := h.authenticate(c)
		if !ok {
			return
		}
		if !isSafeMethod(c.Request.Method) && !validCSRF(c.GetHeader(CSRFHeader), p.CSRFToken) {
			h.log.WarnContext(c.Request.Context(), "csrf check failed",
				slog.String("request_id", c.GetString(httpx.RequestIDKey)), slog.String("user_id", p.UserID))
			httpx.Error(c, http.StatusForbidden, httpx.CodeCSRFFailed,
				"This request could not be verified. Reload the page and try again.")
			return
		}
		c.Set(principalKey, p)
		c.Next()
	}
}

// authenticate resolves the cookie or writes the error response.
func (h *Handler) authenticate(c *gin.Context) (Principal, bool) {
	token, err := c.Cookie(h.cookie.name())
	if err != nil || token == "" {
		httpx.Error(c, http.StatusUnauthorized, httpx.CodeUnauthenticated, "Please sign in to continue.")
		return Principal{}, false
	}
	p, err := h.svc.Authenticate(c.Request.Context(), token)
	if errors.Is(err, ErrNoSession) {
		h.clearCookie(c)
		httpx.Error(c, http.StatusUnauthorized, httpx.CodeSessionExpired, "Your session has ended. Please sign in again.")
		return Principal{}, false
	}
	if err != nil {
		h.log.ErrorContext(c.Request.Context(), "session lookup failed",
			slog.String("request_id", c.GetString(httpx.RequestIDKey)), slog.Any("error", err))
		httpx.Error(c, http.StatusServiceUnavailable, httpx.CodeUnavailable,
			"Your session could not be checked right now. Try again shortly.")
		return Principal{}, false
	}
	return p, true
}

func isSafeMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions
}

func validCSRF(got, want string) bool {
	return got != "" && want != "" && subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

// LoginRequest is the body of POST /auth/login.
type LoginRequest struct {
	Username string `json:"username" example:"asha.verma"`
	Password string `json:"password" example:"correct horse battery staple"`
} //@name LoginRequest

// SessionUser identifies the signed-in user.
type SessionUser struct {
	ID          string `json:"id" example:"5b0f8a52-3c1e-4d6f-9a8b-0c1d2e3f4a5b"`
	Username    string `json:"username" example:"asha.verma"`
	DisplayName string `json:"displayName" example:"Asha Verma"`
} //@name SessionUser

// SessionResponse describes the current session. csrfToken must be sent as
// X-CSRF-Token on every POST/PUT/PATCH/DELETE; keep it in memory only.
type SessionResponse struct {
	User      SessionUser `json:"user"`
	CSRFToken string      `json:"csrfToken" example:"q1w2e3r4t5y6u7i8o9p0a1s2d3f4g5h6j7k8l9z0x1c"`
	ExpiresAt time.Time   `json:"expiresAt" example:"2026-10-12T12:00:00Z"`
} //@name Session

// SignOutResponse is the payload of POST /auth/logout.
type SignOutResponse struct {
	SignedOut bool `json:"signedOut" example:"true"`
} //@name SignOutResult

func sessionResponse(p Principal) SessionResponse {
	return SessionResponse{
		User:      SessionUser{ID: p.UserID, Username: p.Username, DisplayName: p.DisplayName},
		CSRFToken: p.CSRFToken,
		ExpiresAt: p.ExpiresAt.UTC(),
	}
}

// Login godoc
//
//	@Summary		Sign in
//	@Description	Verifies username and password and starts a server-side session. The session token is returned only in an HttpOnly, SameSite=Strict cookie (`__Host-maa_session` over HTTPS, `maa_session` in local development); the body carries the CSRF token for later mutations. Failures are generic: unknown user and wrong password both return INVALID_CREDENTIALS. ACCOUNT_DISABLED is only returned when the password was correct. Repeated failures return 429 with Retry-After.
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			body	body		auth.LoginRequest	true	"Credentials"
//	@Success		200		{object}	httpx.DataResponse{data=auth.SessionResponse}
//	@Failure		400		{object}	httpx.ErrorResponse	"VALIDATION_FAILED or BAD_REQUEST"
//	@Failure		401		{object}	httpx.ErrorResponse	"INVALID_CREDENTIALS"
//	@Failure		403		{object}	httpx.ErrorResponse	"ACCOUNT_DISABLED"
//	@Failure		415		{object}	httpx.ErrorResponse	"UNSUPPORTED_MEDIA_TYPE"
//	@Failure		429		{object}	httpx.ErrorResponse	"RATE_LIMITED"
//	@Router			/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	// Requiring JSON also blocks cross-site HTML form posts (login CSRF): a
	// browser cannot send application/json cross-site without a CORS preflight.
	if c.ContentType() != "application/json" {
		httpx.Error(c, http.StatusUnsupportedMediaType, httpx.CodeUnsupportedMediaType, "Send the request body as application/json.")
		return
	}
	var req LoginRequest
	dec := json.NewDecoder(io.LimitReader(c.Request.Body, 8<<10))
	if err := dec.Decode(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeBadRequest, "The request body must be a JSON object with username and password.")
		return
	}
	if details := validateLogin(req); len(details) > 0 {
		httpx.Error(c, http.StatusBadRequest, httpx.CodeValidationFailed, "Check the highlighted fields.", details...)
		return
	}

	previous, _ := c.Cookie(h.cookie.name())
	token, p, err := h.svc.Login(c.Request.Context(), LoginInput{
		Username:      req.Username,
		Password:      req.Password,
		IP:            c.ClientIP(),
		UserAgent:     c.Request.UserAgent(),
		PreviousToken: previous,
		RequestID:     c.GetString(httpx.RequestIDKey),
	})
	var throttled *ThrottledError
	switch {
	case errors.As(err, &throttled):
		c.Header("Retry-After", strconv.Itoa(int(math.Ceil(throttled.RetryAfter.Seconds()))))
		httpx.Error(c, http.StatusTooManyRequests, httpx.CodeRateLimited, "Too many sign-in attempts.")
		return
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Error(c, http.StatusUnauthorized, httpx.CodeInvalidCredentials, "Incorrect username or password.")
		return
	case errors.Is(err, ErrAccountDisabled):
		httpx.Error(c, http.StatusForbidden, httpx.CodeAccountDisabled, "This account is disabled. Contact your administrator.")
		return
	case err != nil:
		h.log.ErrorContext(c.Request.Context(), "sign-in failed with an internal error",
			slog.String("request_id", c.GetString(httpx.RequestIDKey)), slog.Any("error", err))
		httpx.InternalError(c)
		return
	}

	h.setCookie(c, token)
	c.Header("Cache-Control", "no-store")
	httpx.OK(c, sessionResponse(p))
}

func validateLogin(req LoginRequest) []httpx.FieldError {
	var details []httpx.FieldError
	switch u := strings.TrimSpace(req.Username); {
	case u == "":
		details = append(details, httpx.FieldError{Field: "username", Message: "Enter your username."})
	case utf8.RuneCountInString(u) > 100:
		details = append(details, httpx.FieldError{Field: "username", Message: "Username is too long."})
	}
	switch {
	case req.Password == "":
		details = append(details, httpx.FieldError{Field: "password", Message: "Enter your password."})
	case utf8.RuneCountInString(req.Password) > 256:
		details = append(details, httpx.FieldError{Field: "password", Message: "Password is too long."})
	}
	return details
}

// Session godoc
//
//	@Summary		Current session
//	@Description	Returns the signed-in user and a CSRF token. 401 UNAUTHENTICATED when no session cookie was sent; 401 SESSION_EXPIRED when the session ended (idle or absolute timeout, sign-out elsewhere, password change, account disabled) — the cookie is then cleared.
//	@Tags			auth
//	@Produce		json
//	@Success		200	{object}	httpx.DataResponse{data=auth.SessionResponse}
//	@Failure		401	{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		503	{object}	httpx.ErrorResponse
//	@Router			/auth/session [get]
func (h *Handler) Session(c *gin.Context) {
	p, ok := h.authenticate(c)
	if !ok {
		return
	}
	c.Header("Cache-Control", "no-store")
	httpx.OK(c, sessionResponse(p))
}

// Logout godoc
//
//	@Summary		Sign out
//	@Description	Revokes the current session on the server and clears the cookie. Requires the session cookie and the X-CSRF-Token header.
//	@Tags			auth
//	@Produce		json
//	@Param			X-CSRF-Token	header		string	true	"CSRF token from the session"
//	@Success		200				{object}	httpx.DataResponse{data=auth.SignOutResponse}
//	@Failure		401				{object}	httpx.ErrorResponse	"UNAUTHENTICATED or SESSION_EXPIRED"
//	@Failure		403				{object}	httpx.ErrorResponse	"CSRF_FAILED"
//	@Router			/auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	p, ok := CurrentPrincipal(c)
	if !ok {
		// RequireSession guards this route; reaching here without it is a wiring bug.
		httpx.Error(c, http.StatusUnauthorized, httpx.CodeUnauthenticated, "Please sign in to continue.")
		return
	}
	if err := h.svc.Logout(c.Request.Context(), p, c.GetString(httpx.RequestIDKey)); err != nil {
		h.log.ErrorContext(c.Request.Context(), "sign-out failed",
			slog.String("request_id", c.GetString(httpx.RequestIDKey)), slog.Any("error", err))
		httpx.InternalError(c)
		return
	}
	h.clearCookie(c)
	httpx.OK(c, SignOutResponse{SignedOut: true})
}

func (h *Handler) setCookie(c *gin.Context, token string) {
	// No Max-Age: a browser-session cookie, gone when the browser closes (shared
	// showroom PCs). The server enforces idle and absolute expiry regardless.
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     h.cookie.name(),
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookie.Secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *Handler) clearCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     h.cookie.name(),
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookie.Secure,
		SameSite: http.SameSiteStrictMode,
	})
}
