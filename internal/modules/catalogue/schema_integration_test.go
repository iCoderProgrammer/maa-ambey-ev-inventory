package catalogue

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database/dbtest"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/seeds"
)

// Test data is placeholder only ("Brand A", "M1"): no real model, variant or
// specification is invented, and nothing here is seeded.

type fixture struct {
	pool *pgxpool.Pool
	user string
}

func setup(t *testing.T) fixture {
	t.Helper()
	ctx := context.Background()
	pool := dbtest.MigratedPool(t)
	if _, err := database.ApplySeeds(ctx, pool, seeds.FS); err != nil {
		t.Fatal(err)
	}
	var user string
	err := pool.QueryRow(ctx, `INSERT INTO users (username, display_name, password_hash)
		VALUES ('verifier', 'Verifier', '$argon2id$v=19$m=65536,t=3,p=2$c2FsdHNhbHQ$aGFzaGhhc2hoYXNoaGFzaA')
		RETURNING id::text`).Scan(&user)
	if err != nil {
		t.Fatal(err)
	}
	return fixture{pool: pool, user: user}
}

func (f fixture) brand(t *testing.T, name, slug string) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), `INSERT INTO brands (name, slug) VALUES ($1, $2) RETURNING id::text`, name, slug).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func (f fixture) model(t *testing.T, brandID, code string) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), `INSERT INTO models (brand_id, name, internal_code) VALUES ($1, $2, $3) RETURNING id::text`,
		brandID, "Model "+code, code).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func (f fixture) variant(t *testing.T, brandID, modelID, code string) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), `INSERT INTO variants (brand_id, model_id, name, variant_code) VALUES ($1, $2, $3, $4) RETURNING id::text`,
		brandID, modelID, "Variant "+code, code).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func sqlState(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

func expectState(t *testing.T, name string, err error, want string) {
	t.Helper()
	if got := sqlState(err); got != want {
		t.Errorf("%s: SQLSTATE %q (err %v), want %q", name, got, err, want)
	}
}

const (
	uniqueViolation = "23505"
	fkViolation     = "23503"
	checkViolation  = "23514"
	refused         = "42501"
	restrictDelete  = "23001" // ON DELETE RESTRICT
)

// Acceptance: another brand is added with an INSERT and immediately gets its
// own models and variants — no schema or code change.
func TestAnotherBrandNeedsOnlyData(t *testing.T) {
	ctx := context.Background()
	f := setup(t)
	var seeded int
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM brands`).Scan(&seeded)
	if seeded != 3 {
		t.Fatalf("seeded brands = %d, want 3 (client roster only)", seeded)
	}
	for _, table := range []string{"models", "variants", "variant_colours", "variant_specs"} {
		var n int
		_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM `+table).Scan(&n)
		if n != 0 {
			t.Errorf("%s has %d seeded rows; catalogue data must not be seeded", table, n)
		}
	}

	b := f.brand(t, "Brand A", "brand-a")
	m := f.model(t, b, "M1")
	v := f.variant(t, b, m, "STD")
	if _, err := f.pool.Exec(ctx, `INSERT INTO variant_colours (brand_id, variant_id, name) VALUES ($1, $2, 'Colour 1')`, b, v); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO variant_specs (brand_id, variant_id, spec_key, value_numeric, unit) VALUES ($1, $2, 'attribute_a', 1.5, 'u')`, b, v); err != nil {
		t.Fatal(err)
	}
	var status, confidence string
	_ = f.pool.QueryRow(ctx, `SELECT lifecycle_status, confidence FROM variants WHERE id = $1`, v).Scan(&status, &confidence)
	if status != "PENDING_VERIFICATION" || confidence != "UNVERIFIED" {
		t.Errorf("new variant defaults = %s/%s, want PENDING_VERIFICATION/UNVERIFIED", status, confidence)
	}
}

func TestCodesAreUniqueWithinTheirBrandScope(t *testing.T) {
	ctx := context.Background()
	f := setup(t)
	a := f.brand(t, "Brand A", "brand-a")
	b := f.brand(t, "Brand B", "brand-b")
	ma := f.model(t, a, "M1")
	mb := f.model(t, b, "M1") // same internal code in another brand: allowed

	_, err := f.pool.Exec(ctx, `INSERT INTO models (brand_id, name, internal_code) VALUES ($1, 'Other', 'M1')`, a)
	expectState(t, "duplicate model code in brand", err, uniqueViolation)
	_, err = f.pool.Exec(ctx, `INSERT INTO models (brand_id, name, internal_code) VALUES ($1, 'model m1', 'M2')`, a)
	expectState(t, "duplicate model name (case-insensitive) in brand", err, uniqueViolation)

	_, err = f.pool.Exec(ctx, `UPDATE models SET manufacturer_code = 'mx-1' WHERE id = $1`, ma)
	if err != nil {
		t.Fatal(err)
	}
	m2 := f.model(t, a, "M2")
	_, err = f.pool.Exec(ctx, `UPDATE models SET manufacturer_code = 'MX-1' WHERE id = $1`, m2)
	expectState(t, "duplicate manufacturer code (case-insensitive) in brand", err, uniqueViolation)
	if _, err := f.pool.Exec(ctx, `UPDATE models SET manufacturer_code = 'MX-1' WHERE id = $1`, mb); err != nil {
		t.Errorf("same manufacturer code in another brand: %v", err)
	}

	// Variant codes repeat across models, not within one.
	f.variant(t, a, ma, "STD")
	f.variant(t, a, m2, "STD")
	_, err = f.pool.Exec(ctx, `INSERT INTO variants (brand_id, model_id, name, variant_code) VALUES ($1, $2, 'Again', 'STD')`, a, ma)
	expectState(t, "duplicate variant code in model", err, uniqueViolation)
	v1 := f.variant(t, a, ma, "PRO")
	if _, err := f.pool.Exec(ctx, `UPDATE variants SET manufacturer_code = 'SKU-1' WHERE id = $1`, v1); err != nil {
		t.Fatal(err)
	}
	_, err = f.pool.Exec(ctx, `INSERT INTO variants (brand_id, model_id, name, variant_code, manufacturer_code) VALUES ($1, $2, 'X', 'X', 'sku-1')`, a, m2)
	expectState(t, "duplicate variant manufacturer code in brand", err, uniqueViolation)

	vb := f.variant(t, b, mb, "STD")
	if _, err := f.pool.Exec(ctx, `INSERT INTO variant_colours (brand_id, variant_id, name, colour_code) VALUES ($1, $2, 'Colour 1', 'C1')`, b, vb); err != nil {
		t.Fatal(err)
	}
	_, err = f.pool.Exec(ctx, `INSERT INTO variant_colours (brand_id, variant_id, name, colour_code) VALUES ($1, $2, 'COLOUR 1', 'C2')`, b, vb)
	expectState(t, "duplicate colour name in variant", err, uniqueViolation)
	_, err = f.pool.Exec(ctx, `INSERT INTO variant_colours (brand_id, variant_id, name, colour_code) VALUES ($1, $2, 'Colour 2', 'c1')`, b, vb)
	expectState(t, "duplicate colour code in variant", err, uniqueViolation)
}

func TestBrandContextCannotBeMixed(t *testing.T) {
	ctx := context.Background()
	f := setup(t)
	a := f.brand(t, "Brand A", "brand-a")
	b := f.brand(t, "Brand B", "brand-b")
	ma := f.model(t, a, "M1")
	va := f.variant(t, a, ma, "STD")

	_, err := f.pool.Exec(ctx, `INSERT INTO variants (brand_id, model_id, name, variant_code) VALUES ($1, $2, 'Wrong', 'W')`, b, ma)
	expectState(t, "variant claiming another brand's model", err, fkViolation)
	_, err = f.pool.Exec(ctx, `INSERT INTO variant_colours (brand_id, variant_id, name) VALUES ($1, $2, 'Wrong')`, b, va)
	expectState(t, "colour claiming another brand's variant", err, fkViolation)
	_, err = f.pool.Exec(ctx, `INSERT INTO variant_specs (brand_id, variant_id, spec_key, value_text) VALUES ($1, $2, 'k', 'v')`, b, va)
	expectState(t, "spec claiming another brand's variant", err, fkViolation)
	_, err = f.pool.Exec(ctx, `INSERT INTO models (brand_id, name, internal_code) VALUES (gen_random_uuid(), 'Orphan', 'O1')`)
	expectState(t, "model of unknown brand", err, fkViolation)

	// History keeps its brand: a model with variants cannot move to another brand.
	_, err = f.pool.Exec(ctx, `UPDATE models SET brand_id = $1 WHERE id = $2`, b, ma)
	expectState(t, "moving a model with variants to another brand", err, fkViolation)
}

func TestCatalogueRowsAreArchivedNotDeleted(t *testing.T) {
	ctx := context.Background()
	f := setup(t)
	a := f.brand(t, "Brand A", "brand-a")
	m := f.model(t, a, "M1")
	v := f.variant(t, a, m, "STD")
	if _, err := f.pool.Exec(ctx, `INSERT INTO variant_specs (brand_id, variant_id, spec_key, value_text) VALUES ($1, $2, 'k', 'v')`, a, v); err != nil {
		t.Fatal(err)
	}
	_, err := f.pool.Exec(ctx, `DELETE FROM brands WHERE id = $1`, a)
	expectState(t, "delete brand with models", err, restrictDelete)
	_, err = f.pool.Exec(ctx, `DELETE FROM models WHERE id = $1`, m)
	expectState(t, "delete model with variants", err, restrictDelete)
	_, err = f.pool.Exec(ctx, `DELETE FROM variants WHERE id = $1`, v)
	expectState(t, "delete variant with specs", err, restrictDelete)
	if _, err := f.pool.Exec(ctx, `UPDATE models SET lifecycle_status = 'ARCHIVED' WHERE id = $1`, m); err != nil {
		t.Errorf("archive model: %v", err)
	}
	var updated bool
	_ = f.pool.QueryRow(ctx, `SELECT updated_at > created_at FROM models WHERE id = $1`, m).Scan(&updated)
	if !updated {
		t.Error("updated_at was not maintained")
	}
}

func TestValueConstraints(t *testing.T) {
	ctx := context.Background()
	f := setup(t)
	a := f.brand(t, "Brand A", "brand-a")
	f.variant(t, a, f.model(t, a, "M1"), "STD")
	for name, sql := range map[string]string{
		"lower-case model code":  `INSERT INTO models (brand_id, name, internal_code) VALUES ($1, 'x', 'm9')`,
		"blank model name":       `INSERT INTO models (brand_id, name, internal_code) VALUES ($1, ' ', 'M9')`,
		"unknown model status":   `INSERT INTO models (brand_id, name, internal_code, lifecycle_status) VALUES ($1, 'x', 'M9', 'LIVE')`,
		"unknown confidence":     `INSERT INTO models (brand_id, name, internal_code, confidence) VALUES ($1, 'x', 'M9', 'TRUSTED')`,
		"bad vehicle type":       `INSERT INTO models (brand_id, name, internal_code, vehicle_type) VALUES ($1, 'x', 'M9', 'e scooter')`,
		"non-http source":        `INSERT INTO models (brand_id, name, internal_code, source_url) VALUES ($1, 'x', 'M9', 'javascript:alert(1)')`,
		"spec with two values":   `INSERT INTO variant_specs (brand_id, variant_id, spec_key, value_text, value_numeric) SELECT $1, id, 'k', 'v', 1 FROM variants LIMIT 1`,
		"spec with no value":     `INSERT INTO variant_specs (brand_id, variant_id, spec_key) SELECT $1, id, 'k' FROM variants LIMIT 1`,
		"spec key not lowercase": `INSERT INTO variant_specs (brand_id, variant_id, spec_key, value_text) SELECT $1, id, 'Range KM', 'v' FROM variants LIMIT 1`,
		"variant code spaces":    `INSERT INTO variants (brand_id, model_id, name, variant_code) SELECT $1, id, 'x', 'A B' FROM models LIMIT 1`,
	} {
		_, err := f.pool.Exec(ctx, sql, a)
		expectState(t, name, err, checkViolation)
	}
}

// BR-004: anything above UNVERIFIED needs a source, a source date and a verifier.
func TestProvenanceIsRequiredForVerifiedData(t *testing.T) {
	ctx := context.Background()
	f := setup(t)
	a := f.brand(t, "Brand A", "brand-a")
	m := f.model(t, a, "M1")
	v := f.variant(t, a, m, "STD")

	for _, table := range []string{"models", "variants"} {
		id := m
		if table == "variants" {
			id = v
		}
		_, err := f.pool.Exec(ctx, `UPDATE `+table+` SET confidence = 'CLIENT_APPROVED' WHERE id = $1`, id)
		expectState(t, table+" approved without provenance", err, checkViolation)
		_, err = f.pool.Exec(ctx, `UPDATE `+table+` SET confidence = 'SUPPLIER_CONFIRMED', source_url = 'https://example.invalid/doc', source_date = '2026-10-01' WHERE id = $1`, id)
		expectState(t, table+" confirmed without verifier", err, checkViolation)
		_, err = f.pool.Exec(ctx, `UPDATE `+table+` SET verified_by = $2 WHERE id = $1`, id, f.user)
		expectState(t, table+" verifier without time", err, checkViolation)
		_, err = f.pool.Exec(ctx, `UPDATE `+table+` SET confidence = 'SUPPLIER_CONFIRMED', source_url = 'https://example.invalid/doc',
			source_date = '2026-10-01', verified_by = $2, verified_at = now() WHERE id = $1`, id, f.user)
		if err != nil {
			t.Errorf("%s with full provenance: %v", table, err)
		}
	}
	_, err := f.pool.Exec(ctx, `INSERT INTO variant_colours (brand_id, variant_id, name, confidence) VALUES ($1, $2, 'C', 'CLIENT_APPROVED')`, a, v)
	expectState(t, "colour approved without provenance", err, checkViolation)
	_, err = f.pool.Exec(ctx, `INSERT INTO variant_specs (brand_id, variant_id, spec_key, value_text, confidence, source_url) VALUES ($1, $2, 'k', 'v', 'CLIENT_APPROVED', 'https://example.invalid')`, a, v)
	expectState(t, "spec approved without verifier", err, checkViolation)
}

// BR-005: verified specification values are snapshots; changes are new rows.
func TestVerifiedSpecsAreNeverOverwritten(t *testing.T) {
	ctx := context.Background()
	f := setup(t)
	a := f.brand(t, "Brand A", "brand-a")
	v := f.variant(t, a, f.model(t, a, "M1"), "STD")

	var draft, approved string
	if err := f.pool.QueryRow(ctx, `INSERT INTO variant_specs (brand_id, variant_id, spec_key, value_numeric, effective_from)
		VALUES ($1, $2, 'attribute_a', 1, '2026-01-01') RETURNING id::text`, a, v).Scan(&draft); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE variant_specs SET value_numeric = 2 WHERE id = $1`, draft); err != nil {
		t.Errorf("unverified draft must stay editable: %v", err)
	}
	if err := f.pool.QueryRow(ctx, `INSERT INTO variant_specs (brand_id, variant_id, spec_key, value_numeric, effective_from, confidence,
		source_url, source_date, verified_by, verified_at)
		VALUES ($1, $2, 'attribute_a', 3, '2026-02-01', 'CLIENT_APPROVED', 'https://example.invalid/doc', '2026-01-20', $3, now())
		RETURNING id::text`, a, v, f.user).Scan(&approved); err != nil {
		t.Fatal(err)
	}
	_, err := f.pool.Exec(ctx, `UPDATE variant_specs SET value_numeric = 4 WHERE id = $1`, approved)
	expectState(t, "edit approved spec", err, refused)
	_, err = f.pool.Exec(ctx, `UPDATE variant_specs SET confidence = 'UNVERIFIED', verified_by = NULL, verified_at = NULL WHERE id = $1`, approved)
	expectState(t, "downgrade approved spec", err, refused)
	_, err = f.pool.Exec(ctx, `DELETE FROM variant_specs WHERE id = $1`, approved)
	expectState(t, "delete approved spec", err, refused)
	_, err = f.pool.Exec(ctx, `INSERT INTO variant_specs (brand_id, variant_id, spec_key, value_numeric, effective_from) VALUES ($1, $2, 'attribute_a', 5, '2026-02-01')`, a, v)
	expectState(t, "second value for the same key and date", err, uniqueViolation)
	if _, err := f.pool.Exec(ctx, `INSERT INTO variant_specs (brand_id, variant_id, spec_key, value_numeric, effective_from) VALUES ($1, $2, 'attribute_a', 5, '2026-03-01')`, a, v); err != nil {
		t.Errorf("a later version must be accepted: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `DELETE FROM variant_specs WHERE id = $1`, draft); err != nil {
		t.Errorf("unverified draft may be removed: %v", err)
	}
}

// BR-002: a brand cannot be AUTHORISED without the recorded evidence.
func TestBrandAuthorisationNeedsEvidence(t *testing.T) {
	ctx := context.Background()
	f := setup(t)
	a := f.brand(t, "Brand A", "brand-a")
	_, err := f.pool.Exec(ctx, `UPDATE brands SET authorisation_status = 'AUTHORISED' WHERE id = $1`, a)
	expectState(t, "authorised without evidence", err, checkViolation)
	_, err = f.pool.Exec(ctx, `UPDATE brands SET authorisation_status = 'AUTHORISED', manufacturer_legal_name = 'Legal Name',
		authorisation_reference = 'REF-1', verified_at = now() WHERE id = $1`, a)
	expectState(t, "authorised without verifier", err, checkViolation)
	_, err = f.pool.Exec(ctx, `UPDATE brands SET authorisation_status = 'AUTHORISED', manufacturer_legal_name = 'Legal Name',
		authorisation_reference = 'REF-1', verified_by = $2, verified_at = now() WHERE id = $1`, a, f.user)
	if err != nil {
		t.Errorf("authorised with evidence: %v", err)
	}
	_, err = f.pool.Exec(ctx, `UPDATE brands SET authorisation_evidence_id = gen_random_uuid() WHERE id = $1`, a)
	expectState(t, "evidence pointing at no attachment", err, fkViolation)

	// The seeded roster stays pending and unverified: the roster is not proof of authorisation.
	var n int
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM brands WHERE slug IN ('lectrix-ev','ceeon-india','ujagar-electric')
		AND lifecycle_status = 'PENDING_VERIFICATION' AND authorisation_status = 'UNVERIFIED' AND verified_by IS NULL`).Scan(&n)
	if n != 3 {
		t.Errorf("seeded brands pending/unverified = %d, want 3", n)
	}
}
