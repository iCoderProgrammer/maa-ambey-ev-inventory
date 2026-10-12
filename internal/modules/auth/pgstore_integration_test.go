package auth_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/database/dbtest"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/auth"
)

// These tests commit data, so they run in a throwaway database (dbtest).

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
	u, err := store.CreateUser(context.Background(), username, "Test "+username, nil, hash)
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
	if _, err := store.CreateUser(ctx, "asha", "Asha", &email, hash); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateUser(ctx, "asha", "Other", nil, hash); !errors.Is(err, auth.ErrUsernameTaken) {
		t.Errorf("duplicate username: err = %v", err)
	}
	other := "asha@example.COM"
	if _, err := store.CreateUser(ctx, "asha2", "Other", &other, hash); !errors.Is(err, auth.ErrUsernameTaken) {
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

	if err := svc.Logout(ctx, got); err != nil {
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
	n, err := store.SetStatus(ctx, "asha", auth.StatusDisabled, now)
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

	if _, err := store.SetStatus(ctx, "asha", auth.StatusActive, now); err != nil {
		t.Fatal(err)
	}
	c := login()
	newHash, _ := auth.HashPassword("a completely new passphrase")
	if n, err := store.SetPassword(ctx, "asha", newHash, now); err != nil || n != 1 {
		t.Fatalf("set password: n=%d err=%v", n, err)
	}
	if _, err := svc.Authenticate(ctx, c); !errors.Is(err, auth.ErrNoSession) {
		t.Errorf("session after password change: %v", err)
	}
	if _, _, err := svc.Login(ctx, auth.LoginInput{Username: "asha", Password: password, IP: "192.0.2.1"}); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("old password still works: %v", err)
	}

	if _, err := store.SetStatus(ctx, "nobody", auth.StatusDisabled, now); !errors.Is(err, auth.ErrNotFound) {
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
		wg.Go(func() { _ = svc.Logout(ctx, p) })
	}
	wg.Wait()
	var active, revoked int
	_ = pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE revoked_at IS NULL), count(*) FILTER (WHERE revoked_at IS NOT NULL) FROM user_sessions`).Scan(&active, &revoked)
	if active != n-1 || revoked != 1 {
		t.Errorf("active=%d revoked=%d", active, revoked)
	}
}
