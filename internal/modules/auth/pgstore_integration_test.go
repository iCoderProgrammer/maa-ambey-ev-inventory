package auth_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/audit"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/database/dbtest"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/auth"
)

// These tests commit data, so they run in a throwaway database (dbtest).

var operator = audit.CLIActor("cmd/user:test")

func newPGService(t *testing.T, pool *pgxpool.Pool, now *time.Time) *auth.Service {
	t.Helper()
	svc, err := auth.NewService(auth.NewPGStore(pool), auth.Config{
		IdleTimeout:     2 * time.Hour,
		AbsoluteTimeout: 12 * time.Hour,
		Now:             func() time.Time { return *now },
	}, auth.NewLoginThrottle(50, 100, time.Minute), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func createUser(t *testing.T, store *auth.PGStore, username string) auth.User {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	u, err := store.CreateUser(context.Background(), operator, username, "Test "+username, nil, hash)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestPGUserConstraints(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.MigratedPool(t)
	store := auth.NewPGStore(pool)
	hash, _ := auth.HashPassword(password)

	email := "Asha@Example.com"
	if _, err := store.CreateUser(ctx, operator, "asha", "Asha", &email, hash); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateUser(ctx, operator, "asha", "Other", nil, hash); !errors.Is(err, auth.ErrUsernameTaken) {
		t.Errorf("duplicate username: err = %v", err)
	}
	other := "asha@example.COM"
	if _, err := store.CreateUser(ctx, operator, "asha2", "Other", &other, hash); !errors.Is(err, auth.ErrUsernameTaken) {
		t.Errorf("duplicate e-mail (case-insensitive): err = %v", err)
	}

	for name, sql := range map[string]string{
		"upper-case username": `INSERT INTO users (username, display_name, password_hash) VALUES ('Asha3', 'A', '$argon2id$x')`,
		"plaintext password":  `INSERT INTO users (username, display_name, password_hash) VALUES ('asha4', 'A', 'secret')`,
		"unknown status":      `INSERT INTO users (username, display_name, password_hash, status) VALUES ('asha5', 'A', '$argon2id$x', 'ADMIN')`,
	} {
		_, err := pool.Exec(ctx, sql)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
			t.Errorf("%s: want check violation, got %v", name, err)
		}
	}
}

func TestPGSessionLifecycle(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.MigratedPool(t)
	store := auth.NewPGStore(pool)
	now := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	svc := newPGService(t, pool, &now)
	user := createUser(t, store, "asha")

	token, p, err := svc.Login(ctx, auth.LoginInput{Username: "ASHA", Password: password, IP: "192.0.2.10", UserAgent: "test-agent"})
	if err != nil {
		t.Fatal(err)
	}
	if p.UserID != user.ID {
		t.Fatalf("principal = %+v", p)
	}

	// Only the token hash is stored; IP and user agent are kept for review.
	var storedHash []byte
	var ip, ua string
	var lastLogin *time.Time
	err = pool.QueryRow(ctx, `
		SELECT s.token_hash, host(s.ip_address), s.user_agent, u.last_login_at
		FROM user_sessions s JOIN users u ON u.id = s.user_id WHERE s.id = $1`, p.SessionID).
		Scan(&storedHash, &ip, &ua, &lastLogin)
	if err != nil {
		t.Fatal(err)
	}
	if string(storedHash) == token || len(storedHash) != 32 || ip != "192.0.2.10" || ua != "test-agent" {
		t.Errorf("stored session: hash=%x ip=%s ua=%s", storedHash, ip, ua)
	}
	if lastLogin == nil || !lastLogin.Equal(now) {
		t.Errorf("last_login_at = %v", lastLogin)
	}

	now = now.Add(5 * time.Minute)
	got, err := svc.Authenticate(ctx, token)
	if err != nil || got.SessionID != p.SessionID {
		t.Fatalf("authenticate: %+v %v", got, err)
	}
	var lastSeen time.Time
	_ = pool.QueryRow(ctx, `SELECT last_seen_at FROM user_sessions WHERE id = $1`, p.SessionID).Scan(&lastSeen)
	if !lastSeen.Equal(now) {
		t.Errorf("last_seen_at = %v, want %v", lastSeen, now)
	}

	if err := svc.Logout(ctx, got, "req-logout"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate(ctx, token); !errors.Is(err, auth.ErrNoSession) {
		t.Errorf("after logout: err = %v", err)
	}
	var reason string
	_ = pool.QueryRow(ctx, `SELECT revoked_reason FROM user_sessions WHERE id = $1`, p.SessionID).Scan(&reason)
	if reason != auth.RevokedLogout {
		t.Errorf("revoked_reason = %q", reason)
	}
}

func TestPGAdminActionsEndSessions(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.MigratedPool(t)
	store := auth.NewPGStore(pool)
	now := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	svc := newPGService(t, pool, &now)
	createUser(t, store, "asha")

	login := func() string {
		t.Helper()
		token, _, err := svc.Login(ctx, auth.LoginInput{Username: "asha", Password: password, IP: "192.0.2.1"})
		if err != nil {
			t.Fatal(err)
		}
		return token
	}

	a, b := login(), login()
	n, err := store.SetStatus(ctx, operator, "asha", auth.StatusDisabled, now)
	if err != nil || n != 2 {
		t.Fatalf("disable: n=%d err=%v", n, err)
	}
	for _, tok := range []string{a, b} {
		if _, err := svc.Authenticate(ctx, tok); !errors.Is(err, auth.ErrNoSession) {
			t.Errorf("session after disable: %v", err)
		}
	}
	if _, _, err := svc.Login(ctx, auth.LoginInput{Username: "asha", Password: password, IP: "192.0.2.1"}); !errors.Is(err, auth.ErrAccountDisabled) {
		t.Errorf("login while disabled: %v", err)
	}

	if _, err := store.SetStatus(ctx, operator, "asha", auth.StatusActive, now); err != nil {
		t.Fatal(err)
	}
	c := login()
	newHash, _ := auth.HashPassword("a completely new passphrase")
	if n, err := store.SetPassword(ctx, operator, "asha", newHash, now); err != nil || n != 1 {
		t.Fatalf("set password: n=%d err=%v", n, err)
	}
	if _, err := svc.Authenticate(ctx, c); !errors.Is(err, auth.ErrNoSession) {
		t.Errorf("session after password change: %v", err)
	}
	if _, _, err := svc.Login(ctx, auth.LoginInput{Username: "asha", Password: password, IP: "192.0.2.1"}); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("old password still works: %v", err)
	}

	if _, err := store.SetStatus(ctx, operator, "nobody", auth.StatusDisabled, now); !errors.Is(err, auth.ErrNotFound) {
		t.Errorf("unknown user: %v", err)
	}

	var reasons []string
	rows, _ := pool.Query(ctx, `SELECT revoked_reason FROM user_sessions ORDER BY created_at, id`)
	for rows.Next() {
		var r string
		_ = rows.Scan(&r)
		reasons = append(reasons, r)
	}
	if len(reasons) != 3 {
		t.Fatalf("sessions = %v", reasons)
	}
	// Session history is kept (revoked, not deleted); deleting a user with sessions is refused.
	if _, err := pool.Exec(ctx, `DELETE FROM users WHERE username = 'asha'`); err == nil {
		t.Error("deleting a user with session history must be refused")
	}
}

func TestPGConcurrentSignInsAndSignOuts(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.MigratedPool(t)
	store := auth.NewPGStore(pool)
	now := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	svc := newPGService(t, pool, &now)
	createUser(t, store, "asha")

	const n = 8
	tokens := make([]string, n)
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := range n {
		wg.Go(func() {
			tok, _, err := svc.Login(ctx, auth.LoginInput{Username: "asha", Password: password, IP: "192.0.2.1"})
			tokens[i] = tok
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	seen := map[string]bool{}
	for _, tok := range tokens {
		if seen[tok] {
			t.Fatal("duplicate session token issued")
		}
		seen[tok] = true
	}

	// Signing out the same session twice at once revokes it exactly once.
	p, err := svc.Authenticate(ctx, tokens[0])
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		wg.Go(func() { _ = svc.Logout(ctx, p, "") })
	}
	wg.Wait()
	var active, revoked int
	_ = pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE revoked_at IS NULL), count(*) FILTER (WHERE revoked_at IS NOT NULL) FROM user_sessions`).Scan(&active, &revoked)
	if active != n-1 || revoked != 1 {
		t.Errorf("active=%d revoked=%d", active, revoked)
	}
}

// auditRows returns "action/outcome" for every audit row, oldest first.
func auditRows(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT action || '/' || outcome FROM audit_events ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	return out
}

func TestPGAuthAndAdminActionsAreAudited(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.MigratedPool(t)
	store := auth.NewPGStore(pool)
	now := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	svc := newPGService(t, pool, &now)
	createUser(t, store, "asha")

	_, _, _ = svc.Login(ctx, auth.LoginInput{Username: "asha", Password: "wrong password here", IP: "192.0.2.1", RequestID: "req-1"})
	_, _, _ = svc.Login(ctx, auth.LoginInput{Username: "nobody", Password: "wrong password here", IP: "192.0.2.1"})
	token, _, err := svc.Login(ctx, auth.LoginInput{Username: "asha", Password: password, IP: "192.0.2.1", RequestID: "req-2"})
	if err != nil {
		t.Fatal(err)
	}
	p, _ := svc.Authenticate(ctx, token)
	if err := svc.Logout(ctx, p, "req-3"); err != nil {
		t.Fatal(err)
	}
	_ = svc.Logout(ctx, p, "req-4") // already signed out: no second event
	if _, err := store.SetStatus(ctx, operator, "asha", auth.StatusDisabled, now); err != nil {
		t.Fatal(err)
	}
	_, _, _ = svc.Login(ctx, auth.LoginInput{Username: "asha", Password: password, IP: "192.0.2.1"})
	newHash, _ := auth.HashPassword("another long passphrase")
	if _, err := store.SetPassword(ctx, operator, "asha", newHash, now); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"user.created/SUCCESS", "auth.sign_in_failed/FAILURE", "auth.signed_in/SUCCESS", "auth.signed_out/SUCCESS",
		"user.disabled/SUCCESS", "auth.sign_in_refused/DENIED", "user.password_changed/SUCCESS",
	}
	got := auditRows(t, pool)
	if len(got) != len(want) {
		t.Fatalf("audit rows = %v\nwant %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d = %s, want %s", i, got[i], want[i])
		}
	}

	// Nothing secret was stored: not the password, its hash, or the session token.
	var dump string
	if err := pool.QueryRow(ctx, `SELECT string_agg(row_to_json(a)::text, E'\n') FROM audit_events a`).Scan(&dump); err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{password, "wrong password here", "another long passphrase", "argon2", token} {
		if strings.Contains(dump, secret) {
			t.Errorf("audit trail contains %q", secret)
		}
	}
	var signedIn struct{ actorType, requestID, label string }
	_ = pool.QueryRow(ctx, `SELECT actor_type, request_id, actor_label FROM audit_events WHERE action = 'auth.signed_in'`).
		Scan(&signedIn.actorType, &signedIn.requestID, &signedIn.label)
	if signedIn.actorType != "USER" || signedIn.requestID != "req-2" || signedIn.label != "asha" {
		t.Errorf("signed_in row = %+v", signedIn)
	}
	// Failed/refused attempts are anonymous: the account is the entity, never the actor.
	rows, err := pool.Query(ctx, `SELECT actor_type, actor_user_id IS NULL, entity_type FROM audit_events
		WHERE action IN ('auth.sign_in_failed', 'auth.sign_in_refused')`)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for rows.Next() {
		var actorType, entityType string
		var noUser bool
		_ = rows.Scan(&actorType, &noUser, &entityType)
		n++
		if actorType != "ANONYMOUS" || !noUser || entityType != "user" {
			t.Errorf("attempt row: actor=%s noUser=%v entity=%s", actorType, noUser, entityType)
		}
	}
	rows.Close()
	if n != 2 {
		t.Errorf("want 2 attempt rows, got %d", n)
	}
	var disabled struct{ actorType, label, summary string }
	_ = pool.QueryRow(ctx, `SELECT actor_type, actor_label, summary::text FROM audit_events WHERE action = 'user.disabled'`).
		Scan(&disabled.actorType, &disabled.label, &disabled.summary)
	if disabled.actorType != "CLI" || disabled.label != "cmd/user:test" || !strings.Contains(disabled.summary, `"status": {"to": "DISABLED", "from": "ACTIVE"}`) {
		t.Errorf("disabled row = %+v", disabled)
	}
}

// The audit event is part of the change: if it cannot be written, the change is rolled back.
func TestPGSessionIsNotCreatedWithoutItsAuditEvent(t *testing.T) {
	ctx := context.Background()
	pool := dbtest.MigratedPool(t)
	store := auth.NewPGStore(pool)
	u := createUser(t, store, "asha")
	now := time.Now().UTC()
	_, err := store.CreateSession(ctx, auth.NewSession{
		UserID: u.ID, TokenHash: make([]byte, 32), CSRFToken: strings.Repeat("c", 43), CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	}, audit.Event{Actor: audit.UserActor(u.ID, u.Username), Action: "not-a-valid-action", EntityType: "user"})
	if err == nil {
		t.Fatal("want an error for an invalid audit event")
	}
	var sessions int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM user_sessions`).Scan(&sessions)
	if sessions != 0 {
		t.Errorf("session was created without its audit event (%d rows)", sessions)
	}
}
