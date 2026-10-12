package database

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/jarvisconsulting/maa-ambey-enterprises/seeds"
)

// Verifies the real seed files are idempotent and seed only unverified,
// pending brands. Runs in a rolled-back transaction.
func TestSeedsAreIdempotent(t *testing.T) {
	ctx := context.Background()
	pool, err := Connect(ctx, testConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var before int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM brands`).Scan(&before); err != nil {
		t.Fatal(err)
	}

	for i := range 3 {
		if _, err := ApplySeeds(ctx, tx, seeds.FS); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}

	slugs := []string{"lectrix-ev", "ceeon-india", "ujagar-electric"}
	var n, unverified int
	err = tx.QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE lifecycle_status = 'PENDING_VERIFICATION'
		                          AND authorisation_status = 'UNVERIFIED'
		                          AND authorisation_reference IS NULL
		                          AND verified_at IS NULL)
		FROM brands WHERE slug = ANY($1)`, slugs).Scan(&n, &unverified)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(slugs) {
		t.Errorf("want %d seeded brands after 3 runs, got %d", len(slugs), n)
	}
	if unverified != n {
		t.Errorf("seeded brands must be pending/unverified with no evidence; %d of %d are", unverified, n)
	}

	var after int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM brands`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after-before > len(slugs) {
		t.Errorf("seeds inserted %d rows, want at most %d", after-before, len(slugs))
	}
}

func TestApplySeedsRejectsEmptySource(t *testing.T) {
	if _, err := ApplySeeds(context.Background(), nil, fstest.MapFS{}); err == nil {
		t.Fatal("an empty seed source must be an error, not a silent no-op")
	}
}

func TestSeedsAddOnlyTheConfirmedRolesWithoutGrants(t *testing.T) {
	ctx := context.Background()
	pool, err := Connect(ctx, testConfig(t))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for range 2 {
		if _, err := ApplySeeds(ctx, tx, seeds.FS); err != nil {
			t.Fatal(err)
		}
	}
	var roles int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM roles WHERE key IN ('owner','inventory_manager','sales_executive','accountant','viewer')`).Scan(&roles); err != nil {
		t.Fatal(err)
	}
	if roles != 5 {
		t.Errorf("want the 5 confirmed roles, got %d", roles)
	}
	// A developer database may already hold grants added with cmd/access; the
	// seeds themselves must add none (rows created in this transaction).
	var seeded int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM role_permissions rp JOIN roles r ON r.id = rp.role_id WHERE rp.created_at = now()`).Scan(&seeded); err != nil {
		t.Fatal(err)
	}
	if seeded != 0 {
		t.Errorf("seeds created %d grants; they must create none", seeded)
	}
}
