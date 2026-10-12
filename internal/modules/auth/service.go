// Package auth implements sign-in, server-side sessions and the session
// middleware (B-06). Roles and permissions are B-07; a signed-in user is not
// granted anything here beyond having an identity.
package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"strings"
	"time"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/audit"
)

// Login errors. Handlers map them to the error envelope.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountDisabled    = errors.New("account disabled")
	// ErrNoSession means the presented session token is unknown, expired,
	// revoked, or belongs to a user who can no longer sign in.
	ErrNoSession = errors.New("no valid session")
)

// ThrottledError means too many failed attempts; retry after RetryAfter.
type ThrottledError struct{ RetryAfter time.Duration }

func (e *ThrottledError) Error() string { return "too many failed sign-in attempts" }

// Config holds session lifetimes.
type Config struct {
	IdleTimeout     time.Duration
	AbsoluteTimeout time.Duration
	// TouchInterval limits how often last_seen_at is written (default 1m).
	TouchInterval time.Duration
	// Now is the clock (default time.Now); tests replace it.
	Now func() time.Time
}

// Principal is the authenticated identity attached to a request.
type Principal struct {
	SessionID   string
	UserID      string
	Username    string
	DisplayName string
	CSRFToken   string
	// ExpiresAt is when the session ends if unused: the earlier of the idle and absolute limits.
	ExpiresAt time.Time
}

// LoginInput is one sign-in attempt.
type LoginInput struct {
	Username  string
	Password  string
	IP        string
	UserAgent string
	// PreviousToken is the session cookie the browser already had, revoked on success.
	PreviousToken string
	// RequestID links audit events to the request log.
	RequestID string
}

// Service implements sign-in and session validation.
type Service struct {
	store     Store
	cfg       Config
	throttle  *LoginThrottle
	log       *slog.Logger
	now       func() time.Time
	dummyHash string
}

// NewService builds the auth service.
func NewService(store Store, cfg Config, throttle *LoginThrottle, log *slog.Logger) (*Service, error) {
	if cfg.TouchInterval == 0 {
		cfg.TouchInterval = time.Minute
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	// Verifying against a real hash for unknown usernames keeps response times
	// alike, so timing does not reveal which usernames exist.
	dummy, err := HashPassword("timing-equaliser-not-a-real-password")
	if err != nil {
		return nil, err
	}
	return &Service{store: store, cfg: cfg, throttle: throttle, log: log, now: cfg.Now, dummyHash: dummy}, nil
}

// NormaliseUsername is the canonical form stored in users.username.
func NormaliseUsername(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// Login verifies credentials and opens a new session. It returns the opaque
// token for the cookie; only its hash is stored.
func (s *Service) Login(ctx context.Context, in LoginInput) (string, Principal, error) {
	username := NormaliseUsername(in.Username)
	if wait, blocked := s.throttle.Blocked(username, in.IP); blocked {
		s.log.WarnContext(ctx, "sign-in throttled", slog.String("username", username), slog.String("ip", in.IP))
		return "", Principal{}, &ThrottledError{RetryAfter: wait}
	}

	user, err := s.store.FindUserByUsername(ctx, username)
	switch {
	case errors.Is(err, ErrNotFound):
		_, _ = VerifyPassword(s.dummyHash, in.Password)
		return "", Principal{}, s.fail(ctx, username, in.IP, "unknown username")
	case err != nil:
		return "", Principal{}, err
	}

	// Attempts against existing accounts are audited (unknown usernames are not:
	// they would store whatever was typed, possibly a password).
	userEvent := func(action, outcome, reason string) audit.Event {
		ev := audit.Event{
			Actor: audit.UserActor(user.ID, user.Username), Action: action, Outcome: outcome,
			EntityType: "user", EntityID: user.ID, RequestID: in.RequestID,
		}
		if reason != "" {
			ev.Summary = audit.Summary{"reason": reason}
		}
		return ev
	}
	recordAttempt := func(ev audit.Event) {
		if err := s.store.RecordAudit(ctx, ev); err != nil {
			s.log.ErrorContext(ctx, "could not audit sign-in attempt", slog.String("user_id", user.ID), slog.Any("error", err))
		}
	}

	ok, err := VerifyPassword(user.PasswordHash, in.Password)
	if err != nil {
		s.log.ErrorContext(ctx, "stored password hash is unusable", slog.String("user_id", user.ID), slog.Any("error", err))
		recordAttempt(userEvent("auth.sign_in_failed", audit.OutcomeFailure, "unusable_password_hash"))
		return "", Principal{}, s.fail(ctx, username, in.IP, "unusable password hash")
	}
	if !ok {
		recordAttempt(userEvent("auth.sign_in_failed", audit.OutcomeFailure, "wrong_password"))
		return "", Principal{}, s.fail(ctx, username, in.IP, "wrong password")
	}
	if user.Status != StatusActive {
		// Only revealed to someone who knows the password.
		s.throttle.Failure(username, in.IP)
		s.log.WarnContext(ctx, "sign-in refused: account disabled", slog.String("user_id", user.ID), slog.String("ip", in.IP))
		recordAttempt(userEvent("auth.sign_in_refused", audit.OutcomeDenied, "account_disabled"))
		return "", Principal{}, ErrAccountDisabled
	}

	token, err := newToken()
	if err != nil {
		return "", Principal{}, err
	}
	csrf, err := newToken()
	if err != nil {
		return "", Principal{}, err
	}
	now := s.now().UTC()
	ns := NewSession{
		UserID:    user.ID,
		TokenHash: hashToken(token),
		CSRFToken: csrf,
		CreatedAt: now,
		ExpiresAt: now.Add(s.cfg.AbsoluteTimeout),
		UserAgent: truncate(in.UserAgent, 256),
	}
	if ip, err := netip.ParseAddr(in.IP); err == nil {
		ns.IP = &ip
	}
	sessionID, err := s.store.CreateSession(ctx, ns, userEvent("auth.signed_in", audit.OutcomeSuccess, ""))
	if err != nil {
		return "", Principal{}, err
	}
	s.throttle.Success(username, in.IP)

	// A fresh token on every sign-in prevents session fixation; the old one is retired.
	if in.PreviousToken != "" {
		if err := s.store.RevokeSessionByTokenHash(ctx, hashToken(in.PreviousToken), RevokedSuperseded, now); err != nil {
			s.log.WarnContext(ctx, "could not revoke superseded session", slog.Any("error", err))
		}
	}

	s.log.InfoContext(ctx, "signed in", slog.String("user_id", user.ID), slog.String("session_id", sessionID), slog.String("ip", in.IP))
	return token, Principal{
		SessionID:   sessionID,
		UserID:      user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		CSRFToken:   csrf,
		ExpiresAt:   s.expiry(now, ns.ExpiresAt),
	}, nil
}

func (s *Service) fail(ctx context.Context, username, ip, reason string) error {
	s.throttle.Failure(username, ip)
	s.log.WarnContext(ctx, "sign-in failed", slog.String("username", username), slog.String("ip", ip), slog.String("reason", reason))
	return ErrInvalidCredentials
}

// Authenticate resolves a session token to a principal, enforcing revocation,
// idle and absolute expiry and the user's current status on every call.
func (s *Service) Authenticate(ctx context.Context, token string) (Principal, error) {
	if token == "" || len(token) > 128 {
		return Principal{}, ErrNoSession
	}
	rec, err := s.store.FindSessionByTokenHash(ctx, hashToken(token))
	if errors.Is(err, ErrNotFound) {
		return Principal{}, ErrNoSession
	}
	if err != nil {
		return Principal{}, err
	}
	now := s.now().UTC()
	switch {
	case rec.RevokedAt != nil,
		!now.Before(rec.ExpiresAt),
		!now.Before(rec.LastSeenAt.Add(s.cfg.IdleTimeout)),
		rec.User.Status != StatusActive:
		return Principal{}, ErrNoSession
	}

	lastSeen := rec.LastSeenAt
	if now.Sub(rec.LastSeenAt) >= s.cfg.TouchInterval {
		if err := s.store.TouchSession(ctx, rec.ID, now); err != nil {
			return Principal{}, fmt.Errorf("touch session: %w", err)
		}
		lastSeen = now
	}
	return Principal{
		SessionID:   rec.ID,
		UserID:      rec.User.ID,
		Username:    rec.User.Username,
		DisplayName: rec.User.DisplayName,
		CSRFToken:   rec.CSRFToken,
		ExpiresAt:   s.expiry(lastSeen, rec.ExpiresAt),
	}, nil
}

// Logout revokes the given session and audits it.
func (s *Service) Logout(ctx context.Context, p Principal, requestID string) error {
	ev := audit.Event{
		Actor: audit.UserActor(p.UserID, p.Username), Action: "auth.signed_out",
		EntityType: "user", EntityID: p.UserID, RequestID: requestID,
		Summary: audit.Summary{"sessionId": p.SessionID},
	}
	if err := s.store.RevokeSession(ctx, p.SessionID, RevokedLogout, s.now().UTC(), ev); err != nil {
		return err
	}
	s.log.InfoContext(ctx, "signed out", slog.String("user_id", p.UserID), slog.String("session_id", p.SessionID))
	return nil
}

func (s *Service) expiry(lastSeen, absolute time.Time) time.Time {
	idle := lastSeen.Add(s.cfg.IdleTimeout)
	if idle.Before(absolute) {
		return idle
	}
	return absolute
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	// Cut on a rune boundary.
	for n > 0 && n < len(s) && (s[n]&0xC0) == 0x80 {
		n--
	}
	return s[:n]
}
