// Package dbtest creates throwaway PostgreSQL databases for integration tests,
// so tests that commit data never touch the developer's database. It needs
// TEST_DATABASE_URL (a role allowed to CREATE DATABASE); see `make test-integration`.
package dbtest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/url"
	"os"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // migrate driver
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jarvisconsulting/maa-ambey-enterprises/migrations"
)

// BaseURL returns TEST_DATABASE_URL, skipping the test when it is unset
// (failing instead when REQUIRE_INTEGRATION_TESTS=1).
func BaseURL(t *testing.T) string {
	t.Helper()
	u := os.Getenv("TEST_DATABASE_URL")
	if u == "" {
		if os.Getenv("REQUIRE_INTEGRATION_TESTS") == "1" {
			t.Fatal("TEST_DATABASE_URL not set but REQUIRE_INTEGRATION_TESTS=1")
		}
		t.Skip("TEST_DATABASE_URL not set; skipping PostgreSQL integration test")
	}
	return u
}

// ScratchURL creates an empty database named maa_test_<random> and drops it
// when the test ends. It returns the database's postgres:// URL.
func ScratchURL(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	base := BaseURL(t)
	admin, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	// Cleanups run last-in first-out: the database is dropped before this connection closes.
	t.Cleanup(func() { _ = admin.Close(context.Background()) })

	suffix := make([]byte, 6)
	_, _ = rand.Read(suffix)
	name := "maa_test_" + hex.EncodeToString(suffix)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42501" {
			t.Skip("database role may not CREATE DATABASE")
		}
		t.Fatalf("create scratch database: %v", err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(context.Background(), "DROP DATABASE "+pgx.Identifier{name}.Sanitize()+" WITH (FORCE)"); err != nil {
			t.Errorf("drop scratch database %s: %v", name, err)
		}
	})

	u, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + name
	return u.String()
}

// NewMigrator opens golang-migrate on the embedded migrations for dbURL.
func NewMigrator(t *testing.T, dbURL string) *migrate.Migrate {
	t.Helper()
	u, err := url.Parse(dbURL)
	if err != nil {
		t.Fatal(err)
	}
	u.Scheme = "pgx5"
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		t.Fatal(err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, u.String())
	if err != nil {
		t.Fatalf("open migrate: %v", err)
	}
	t.Cleanup(func() { _, _ = m.Close() })
	return m
}

// MigratedPool returns a pool on a fresh scratch database with every migration applied.
func MigratedPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := ScratchURL(t)
	if err := NewMigrator(t, dbURL).Up(); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}
