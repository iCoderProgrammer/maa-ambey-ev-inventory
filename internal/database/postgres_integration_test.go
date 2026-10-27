package database

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/config"
)

// These tests need a migrated PostgreSQL database. They are skipped unless
// TEST_DATABASE_URL is set (see `make test-integration`). With
// REQUIRE_INTEGRATION_TESTS=1 (set by `make test-integration` and CI) a missing
// URL fails the test instead of skipping it. Every write runs in a transaction
// that is rolled back, so no data is left behind.
func testConfig(t *testing.T) config.Config {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		if os.Getenv("REQUIRE_INTEGRATION_TESTS") == "1" {
			t.Fatal("TEST_DATABASE_URL not set but REQUIRE_INTEGRATION_TESTS=1")
		}
		t.Skip("TEST_DATABASE_URL not set; skipping PostgreSQL integration test")
	}
	return config.Config{
		DatabaseURL:       url,
		DBMaxConns:        2,
		DBMinConns:        0,
		DBMaxConnLifetime: time.Minute,
		DBMaxConnIdleTime: time.Minute,
	}
}

func TestConnectAndPing(t *testing.T) {
	pool, err := Connect(context.Background(), testConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestBrandsConstraints(t *testing.T) {
	ctx := context.Background()
	pool, err := Connect(ctx, testConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	insert := `INSERT INTO brands (name, slug, lifecycle_status) VALUES ($1, $2, $3) RETURNING lifecycle_status, authorisation_status`

	tests := []struct {
		name, brand, slug, status string
		wantSQLState              string // "" = must succeed
	}{
		{"defaults to pending + unverified", "Integration Test Brand", "integration-test-brand", "PENDING_VERIFICATION", ""},
		{"invalid slug", "Bad Slug", "Bad Slug!", "PENDING_VERIFICATION", "23514"},
		{"invalid lifecycle", "Bad Status", "bad-status", "SELLING", "23514"},
		{"blank name", "   ", "blank-name", "PENDING_VERIFICATION", "23514"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback(ctx) }()

			var lifecycle, auth string
			err = tx.QueryRow(ctx, insert, tt.brand, tt.slug, tt.status).Scan(&lifecycle, &auth)
			assertSQLState(t, err, tt.wantSQLState)
			if tt.wantSQLState == "" && (lifecycle != "PENDING_VERIFICATION" || auth != "UNVERIFIED") {
				t.Errorf("got lifecycle=%s auth=%s", lifecycle, auth)
			}
		})
	}

	t.Run("slug and case-insensitive name are unique", func(t *testing.T) {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(ctx) }()

		if _, err := tx.Exec(ctx, `INSERT INTO brands (name, slug) VALUES ('Dup Brand', 'dup-brand')`); err != nil {
			t.Fatal(err)
		}
		_, err = tx.Exec(ctx, `SAVEPOINT s1; INSERT INTO brands (name, slug) VALUES ('Other', 'dup-brand')`)
		assertSQLState(t, err, "23505")
		_, _ = tx.Exec(ctx, `ROLLBACK TO SAVEPOINT s1`)
		_, err = tx.Exec(ctx, `INSERT INTO brands (name, slug) VALUES ('DUP BRAND', 'dup-brand-2')`)
		assertSQLState(t, err, "23505")
	})
}

func assertSQLState(t *testing.T, err error, want string) {
	t.Helper()
	if want == "" {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != want {
		t.Fatalf("want SQLSTATE %s, got %v", want, err)
	}
}
