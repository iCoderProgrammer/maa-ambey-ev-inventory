package database

import (
	"context"
	"testing"
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
		if _, err := ApplySeeds(ctx, tx, "../../seeds"); err != nil {
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
