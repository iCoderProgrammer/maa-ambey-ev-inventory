package vehicles

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database/dbtest"
)

// D-05: duplicate vehicle identifier controls, on throwaway databases.
// Identifiers are placeholders ("CH-0001"); no real chassis number is used.

// receiptWithChassis adds a draft receipt on po/poLine with one ACCEPTED unit per
// chassis number ("" = no chassis) and returns the receipt and its unit IDs.
func (f fixture) receiptWithChassis(t *testing.T, po, poLine string, chassis ...string) (string, []string) {
	t.Helper()
	grn := f.newReceipt(t, po)
	line := f.receiptLine(t, grn, po, poLine, len(chassis), len(chassis), 0)
	units := make([]string, len(chassis))
	for i, c := range chassis {
		units[i] = f.receiptUnit(t, grn, line, i+1, "ACCEPTED", c, "", "", "")
	}
	return grn, units
}

// insertUnitDirect writes a unit for a captured unit on otherPO without the
// service, as code that skipped the checks (or data from before D-05) would.
func (f fixture) insertUnitDirect(ctx context.Context, tx pgx.Tx, grn, unit, chassis string) error {
	var line string
	if err := tx.QueryRow(ctx, `SELECT goods_receipt_line_id::text FROM goods_receipt_units WHERE id = $1`, unit).Scan(&line); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `INSERT INTO vehicle_units (brand_id, model_id, variant_id, chassis_number, source_type, supplier_id,
			purchase_order_id, purchase_order_line_id, goods_receipt_id, goods_receipt_line_id, goods_receipt_unit_id,
			received_date, unit_cost, registered_by)
		VALUES ($1, $2, $3, $4, 'GOODS_RECEIPT', $5, $6, $7, $8, $9, $10, DATE '2026-01-15', 97000, $11)`,
		f.brandA, f.modelA, f.variantA2, chassis, f.supplier, f.otherPO, f.otherPOLine, grn, line, unit, f.ids["poster"])
	return err
}

func isDuplicateChassis(err error, unit string) bool {
	var dup *DuplicateIdentifierError
	return errors.As(err, &dup) && errors.Is(err, ErrDuplicateIdentifier) && dup.Field == FieldChassisNumber &&
		dup.GoodsReceiptUnitID == unit
}

func TestDuplicateChassisIsRefusedInEveryBrandAndForm(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	// unitA1 carries "ch-0001" (normalised CH-0001).
	if _, err := f.registerAll(ctx, f.grn, true, nil, f.unitA1, f.unitA2); err != nil {
		t.Fatal(err)
	}
	// The same number in other forms, on other receipts: same brand, and another brand (line B is Brand B).
	sameBrand, units := f.receiptWithChassis(t, f.otherPO, f.otherPOLine, "  CH-0001 ")
	if _, err := f.registerAll(ctx, sameBrand, true, nil, units[0]); !isDuplicateChassis(err, units[0]) {
		t.Fatalf("same brand, other case and spaces: %v", err)
	}
	otherBrand, bUnits := f.receiptWithChassis(t, f.po, f.poLineB, "Ch-0001")
	if _, err := f.registerAll(ctx, otherBrand, true, nil, bUnits[0]); !isDuplicateChassis(err, bUnits[0]) {
		t.Fatalf("other brand: %v", err)
	}
	// The error names the field and the caller's own unit only, nothing about the registered vehicle.
	_, err := f.registerAll(ctx, otherBrand, false, nil, bUnits[0])
	for _, leak := range []string{"MAE-VEH", "ch-0001", "CH-0001", f.unitA1, "Brand", "Supplier", f.grn} {
		if strings.Contains(err.Error(), leak) {
			t.Errorf("error %q reveals %q", err.Error(), leak)
		}
	}
	// Separators are kept (OQ-26): these are different numbers. Units without a chassis never conflict.
	sep, sepUnits := f.receiptWithChassis(t, f.otherPO, f.otherPOLine, "CH 0001", "CH/0001", "CH0001", "", "")
	if _, err := f.registerAll(ctx, sep, true, nil, sepUnits...); err != nil {
		t.Fatalf("separator variants and missing chassis: %v", err)
	}
	if n := f.count(t, `SELECT count(*) FROM vehicle_units WHERE chassis_normalised IS NULL`); n != 3 {
		t.Errorf("units without chassis: %d", n)
	}
	if n := f.count(t, `SELECT count(*) FROM vehicle_units WHERE chassis_normalised = 'CH-0001'`); n != 1 {
		t.Errorf("CH-0001 units: %d", n)
	}
	// Refused registrations left nothing: one history row and one audit event per unit.
	units6 := f.count(t, `SELECT count(*) FROM vehicle_units`)
	if units6 != 7 || f.count(t, `SELECT count(*) FROM vehicle_unit_events`) != units6 ||
		f.count(t, `SELECT count(*) FROM audit_events WHERE action = 'vehicle_unit.registered'`) != units6 {
		t.Errorf("units %d, history %d, audit %d", units6, f.count(t, `SELECT count(*) FROM vehicle_unit_events`),
			f.count(t, `SELECT count(*) FROM audit_events WHERE action = 'vehicle_unit.registered'`))
	}
}

// The database itself refuses a second unit with the same normalised chassis,
// so no code path that skips the service check can store one.
func TestTheDatabaseRefusesDuplicateChassisDirectly(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if _, err := f.registerAll(ctx, f.grn, true, nil, f.unitA1); err != nil {
		t.Fatal(err)
	}
	grn, units := f.receiptWithChassis(t, f.otherPO, f.otherPOLine, "X")
	err := pgx.BeginFunc(ctx, f.pool, func(tx pgx.Tx) error {
		if err := f.post(ctx, tx, grn); err != nil {
			return err
		}
		return f.insertUnitDirect(ctx, tx, grn, units[0], " ch-0001")
	})
	if code, constraint := sqlState(err); code != "23505" || constraint != "vehicle_units_chassis_key" {
		t.Fatalf("direct insert: %s %s (%v)", code, constraint, err)
	}
	// The normalised column cannot be written: it is generated from the number as entered.
	_, err = f.pool.Exec(ctx, `UPDATE vehicle_units SET chassis_normalised = 'OTHER'`)
	if code, _ := sqlState(err); code != "428C9" && code != "42501" {
		t.Fatalf("writing the normalised column: %v", err)
	}
	// Only the named index means "duplicate identifier": the captured-unit key keeps its own error.
	if _, err := f.registerAll(ctx, f.grn, false, nil, f.unitA1); !errors.Is(err, ErrAlreadyRegistered) || errors.Is(err, ErrDuplicateIdentifier) {
		t.Fatalf("same captured unit again: %v", err)
	}
}

// Two transactions register the same chassis: both pass the early check (the
// other's row is not committed yet), the unique index makes the second wait
// and then fail, and it gets the same error as the early check.
func TestConcurrentRegistrationsOfOneChassisLetExactlyOneThrough(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	grnA, unitsA := f.receiptWithChassis(t, f.otherPO, f.otherPOLine, "CH-RACE")
	grnB, unitsB := f.receiptWithChassis(t, f.po, f.poLineB, " ch-race ")

	// Deterministic interleaving: A has inserted but not committed when B starts.
	txA, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = txA.Rollback(ctx) }()
	if _, err := RegisterFromReceiptUnit(ctx, txA, f.actor(), "req-a", unitsA[0], nil); err != nil {
		t.Fatal(err)
	}
	if err := f.post(ctx, txA, grnA); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	started := time.Now()
	go func() {
		_, err := f.registerAll(ctx, grnB, true, nil, unitsB[0])
		result <- err
	}()
	time.Sleep(300 * time.Millisecond)
	if err := txA.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	errB := <-result
	if !isDuplicateChassis(errB, unitsB[0]) {
		t.Fatalf("second transaction: %v", errB)
	}
	if time.Since(started) < 250*time.Millisecond {
		t.Errorf("B did not wait for A: it never reached the index")
	}
	if n := f.count(t, `SELECT count(*) FROM vehicle_units WHERE chassis_normalised = 'CH-RACE'`); n != 1 {
		t.Fatalf("CH-RACE units: %d", n)
	}
	if s := f.scalar(t, `SELECT status FROM goods_receipts WHERE id = $1`, grnB); s != "DRAFT" {
		t.Errorf("refused transaction left its receipt %s", s)
	}
}

func TestManyParallelRegistrationsOfOneChassis(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	const n = 8
	type attempt struct{ grn, unit string }
	attempts := make([]attempt, n)
	for i := range attempts {
		grn, units := f.receiptWithChassis(t, f.otherPO, f.otherPOLine, []string{"CH-PAR", "ch-par", " CH-PAR", "Ch-Par "}[i%4])
		attempts[i] = attempt{grn, units[0]}
	}
	start := make(chan struct{})
	results := make([]error, n)
	var wg sync.WaitGroup
	for i, a := range attempts {
		wg.Go(func() {
			<-start
			_, results[i] = f.registerAll(ctx, a.grn, true, nil, a.unit)
		})
	}
	close(start)
	wg.Wait()
	ok, dup := 0, 0
	for i, err := range results {
		switch {
		case err == nil:
			ok++
		case isDuplicateChassis(err, attempts[i].unit):
			dup++
		default:
			t.Errorf("attempt %d: %v", i, err)
		}
	}
	if ok != 1 || dup != n-1 {
		t.Fatalf("ok=%d duplicate=%d, want 1 and %d", ok, dup, n-1)
	}
	if c := f.count(t, `SELECT count(*) FROM vehicle_units`); c != 1 {
		t.Fatalf("units: %d", c)
	}
	if f.count(t, `SELECT count(*) FROM vehicle_unit_events`) != 1 ||
		f.count(t, `SELECT count(*) FROM audit_events WHERE action = 'vehicle_unit.registered'`) != 1 ||
		f.count(t, `SELECT count(*) FROM goods_receipts WHERE status = 'POSTED'`) != 1 {
		t.Fatal("refused attempts left history, audit rows or posted receipts")
	}
}

// Migration 000014 refuses to run over existing duplicates and changes nothing;
// without duplicates it applies, rolls back and applies again.
func TestUniquenessMigrationRefusesExistingDuplicates(t *testing.T) {
	ctx := context.Background()
	url := dbtest.ScratchURL(t)
	m := dbtest.NewMigrator(t, url)
	if err := m.Migrate(13); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	f := setupOn(t, pool)
	// Version 13 has no stock movement ledger (E-01) for today's registration to
	// write to, so both units are written directly, as version 13 allowed.
	if err := pgx.BeginFunc(ctx, f.pool, func(tx pgx.Tx) error {
		if err := f.post(ctx, tx, f.grn); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO vehicle_units (brand_id, model_id, variant_id, chassis_number, source_type, supplier_id,
				purchase_order_id, purchase_order_line_id, goods_receipt_id, goods_receipt_line_id, goods_receipt_unit_id,
				received_date, unit_cost, registered_by)
			VALUES ($1, $2, $3, 'ch-0001', 'GOODS_RECEIPT', $4, $5, $6, $7, $8, $9, DATE '2026-01-15', 95000, $10)`,
			f.brandA, f.modelA, f.variantA, f.supplier, f.po, f.poLineA, f.grn, f.grnLineA, f.unitA1, f.ids["poster"])
		return err
	}); err != nil {
		t.Fatal(err)
	}
	// Data from before D-05 (version 13 has no uniqueness): written directly, as the D-05 service would refuse it.
	grn, units := f.receiptWithChassis(t, f.otherPO, f.otherPOLine, "CH-0001 ")
	if err := pgx.BeginFunc(ctx, f.pool, func(tx pgx.Tx) error {
		if err := f.post(ctx, tx, grn); err != nil {
			return err
		}
		return f.insertUnitDirect(ctx, tx, grn, units[0], " ch-0001")
	}); err != nil {
		t.Fatalf("version 13 has no uniqueness yet: %v", err)
	}
	err = m.Up()
	if err == nil || !strings.Contains(err.Error(), "CH-0001") || !strings.Contains(err.Error(), "resolve them before applying D-05") {
		t.Fatalf("migration over duplicates: %v", err)
	}
	if n := f.count(t, `SELECT count(*) FROM vehicle_units`); n != 2 {
		t.Fatalf("units changed: %d", n)
	}
	if n := f.count(t, `SELECT count(*) FROM pg_indexes WHERE indexname = 'vehicle_units_chassis_key'`); n != 0 {
		t.Fatal("unique index created over duplicates")
	}
}

func TestUniquenessMigrationRoundTrip(t *testing.T) {
	ctx := context.Background()
	url := dbtest.ScratchURL(t)
	m := dbtest.NewMigrator(t, url)
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	index := func(name string) (exists, unique bool) {
		err := pool.QueryRow(ctx, `SELECT true, i.indisunique FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid
			WHERE c.relname = $1`, name).Scan(&exists, &unique)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			t.Fatal(err)
		}
		return exists, unique
	}
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	if e, u := index("vehicle_units_chassis_key"); !e || !u {
		t.Fatal("up: no unique chassis index")
	}
	if err := m.Migrate(13); err != nil {
		t.Fatal(err)
	}
	if e, _ := index("vehicle_units_chassis_key"); e {
		t.Fatal("down: unique index kept")
	}
	if e, u := index("vehicle_units_chassis_idx"); !e || u {
		t.Fatal("down: lookup index not restored")
	}
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	if e, u := index("vehicle_units_chassis_key"); !e || !u {
		t.Fatal("up again: no unique chassis index")
	}
}
