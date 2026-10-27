package database

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database/dbtest"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/migrations"
)

// TestMigrationsRoundTrip applies every migration, rolls all of them back and
// applies them again in a throwaway database (dbtest), so the developer's
// database is never touched. It proves each down script reverts its up script.
func TestMigrationsRoundTrip(t *testing.T) {
	ctx := context.Background()
	latest, err := migrations.Latest()
	if err != nil {
		t.Fatal(err)
	}
	scratchURL := dbtest.ScratchURL(t)
	m := dbtest.NewMigrator(t, scratchURL)

	scratch, err := pgx.Connect(ctx, scratchURL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = scratch.Close(ctx) }()

	userTables := func() int {
		t.Helper()
		var n int
		err := scratch.QueryRow(ctx, `
			SELECT count(*) FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name <> 'schema_migrations'`).Scan(&n)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}

	if err := CheckSchema(ctx, scratch, latest); !errors.Is(err, ErrSchemaNotMigrated) {
		t.Fatalf("fresh database: err = %v, want ErrSchemaNotMigrated", err)
	}

	for round := 1; round <= 2; round++ {
		if err := m.Up(); err != nil {
			t.Fatalf("round %d up: %v", round, err)
		}
		if err := CheckSchema(ctx, scratch, latest); err != nil {
			t.Fatalf("round %d after up: %v", round, err)
		}
		if userTables() == 0 {
			t.Fatalf("round %d: up created no tables", round)
		}

		if latest > 1 {
			if err := m.Steps(-1); err != nil {
				t.Fatalf("round %d one step down: %v", round, err)
			}
			if err := CheckSchema(ctx, scratch, latest); !errors.Is(err, ErrSchemaBehind) {
				t.Fatalf("round %d after one step down: err = %v, want ErrSchemaBehind", round, err)
			}
			if err := m.Up(); err != nil {
				t.Fatalf("round %d re-up: %v", round, err)
			}
		}

		if err := m.Steps(-int(latest)); err != nil {
			t.Fatalf("round %d down: %v", round, err)
		}
		if n := userTables(); n != 0 {
			t.Fatalf("round %d: %d table(s) left after rolling everything back", round, n)
		}
		if err := CheckSchema(ctx, scratch, latest); !errors.Is(err, ErrSchemaNotMigrated) {
			t.Fatalf("round %d after down: err = %v, want ErrSchemaNotMigrated", round, err)
		}
	}
}

// TestDevelopmentSchemaIsCurrent checks the database used by the integration
// tests (migrated by `make db-migrate-up`) matches this build.
func TestDevelopmentSchemaIsCurrent(t *testing.T) {
	ctx := context.Background()
	pool, err := Connect(ctx, testConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	latest, err := migrations.Latest()
	if err != nil {
		t.Fatal(err)
	}
	if err := (SchemaChecker{DB: pool, Want: latest}).CheckSchema(ctx); err != nil {
		t.Fatalf("schema: %v", err)
	}
}
