package stock_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/audit"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/database"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/database/dbtest"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/auth"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/stock"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/vehicles"
	"github.com/jarvisconsulting/maa-ambey-enterprises/seeds"
)

// Stock movement tests on a throwaway migrated database. Names, identifiers and
// amounts are placeholders ("Brand A", "CH-0001", 95000): no real supplier,
// vehicle, serial or cost is invented. Approved purchase orders and goods
// receipts are inserted directly and "posting" is simulated by setting the
// receipt POSTED in the registering transaction: receipt posting is D-07.

type fixture struct {
	pool    *pgxpool.Pool
	h       *harness
	mover   session // vehicle_units.display.manage, every location
	scoped  session // vehicle_units.display.manage, location L2 only
	viewer  session // vehicle_units.view only
	other   session // vehicle_units.display.manage, every location (a second user)
	ids     map[string]string
	locL1   string
	locL2   string
	po      string
	poLine  string
	nextNo  int
	brandID string
}

func setup(t *testing.T) *fixture {
	t.Helper()
	return setupOn(t, dbtest.MigratedPool(t))
}

func setupOn(t *testing.T, pool *pgxpool.Pool) *fixture {
	t.Helper()
	ctx := context.Background()
	if _, err := database.ApplySeeds(ctx, pool, seeds.FS); err != nil {
		t.Fatal(err)
	}
	store := auth.NewPGStore(pool)
	hash, err := auth.HashPassword(testPassword)
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{pool: pool, ids: map[string]string{}}
	for _, u := range []string{"mover", "scoped", "viewer", "other", "buyer", "approver", "poster"} {
		created, err := store.CreateUser(ctx, audit.CLIActor("test"), u, u, nil, hash)
		if err != nil {
			t.Fatal(err)
		}
		f.ids[u] = created.ID
	}
	f.h = newHarness(t, store, stock.NewPGStore(pool))
	f.mover = f.h.login("mover", f.ids["mover"])
	f.scoped = f.h.login("scoped", f.ids["scoped"])
	f.viewer = f.h.login("viewer", f.ids["viewer"])
	f.other = f.h.login("other", f.ids["other"])
	f.h.grant(f.mover, access.PermVehicleUnitDisplayManage, access.PermAllLocations)
	f.h.grant(f.other, access.PermVehicleUnitDisplayManage, access.PermAllLocations)
	f.h.grant(f.scoped, access.PermVehicleUnitDisplayManage)
	f.h.grant(f.viewer, access.PermVehicleUnitView, access.PermAllLocations)

	f.locL1 = f.scalar(t, `INSERT INTO locations (code, name) VALUES ('L1', 'Location One') RETURNING id::text`)
	f.locL2 = f.scalar(t, `INSERT INTO locations (code, name) VALUES ('L2', 'Location Two') RETURNING id::text`)
	f.h.scope(f.scoped, f.locL2)

	f.brandID = f.scalar(t, `INSERT INTO brands (name, slug, lifecycle_status, authorisation_status, manufacturer_legal_name,
		authorisation_reference, verified_by, verified_at)
		VALUES ('Brand A', 'brand-a', 'ACTIVE', 'AUTHORISED', 'Example Manufacturer', 'TEST-AUTH', $1, now()) RETURNING id::text`, f.ids["approver"])
	model := f.scalar(t, `INSERT INTO models (brand_id, name, internal_code, lifecycle_status, confidence, source_url, source_date, verified_by, verified_at)
		VALUES ($1, 'Model MA', 'MA', 'ACTIVE', 'SUPPLIER_CONFIRMED', 'https://example.com/m', current_date, $2, now()) RETURNING id::text`,
		f.brandID, f.ids["approver"])
	variant := f.scalar(t, `INSERT INTO variants (brand_id, model_id, name, variant_code, lifecycle_status, confidence, source_url, source_date, verified_by, verified_at)
		VALUES ($1, $2, 'Variant STD', 'STD', 'ACTIVE', 'SUPPLIER_CONFIRMED', 'https://example.com/v', current_date, $3, now()) RETURNING id::text`,
		f.brandID, model, f.ids["approver"])
	supplier := f.scalar(t, `INSERT INTO suppliers (legal_name, created_by) VALUES ('Example Supplier One', $1) RETURNING id::text`, f.ids["buyer"])
	f.po = f.scalar(t, `INSERT INTO purchase_orders (supplier_id, created_by, expected_date) VALUES ($1, $2, current_date + 7) RETURNING id::text`,
		supplier, f.ids["buyer"])
	f.poLine = f.scalar(t, `INSERT INTO purchase_order_lines (purchase_order_id, line_no, brand_id, model_id, variant_id, quantity, unit_cost)
		VALUES ($1, 1, $2, $3, $4, 50, 95000.00) RETURNING id::text`, f.po, f.brandID, model, variant)
	f.exec(t, `UPDATE purchase_orders SET status = 'APPROVED', submitted_by = $2, submitted_at = now(), approval_basis = 'APPROVER',
		approved_by = $3, approved_at = now() WHERE id = $1`, f.po, f.ids["buyer"], f.ids["approver"])
	return f
}

func (f *fixture) exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) scalar(t *testing.T, sql string, args ...any) string {
	t.Helper()
	var s string
	if err := f.pool.QueryRow(context.Background(), sql, args...).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

func (f *fixture) count(t *testing.T, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func (f *fixture) actor() audit.Actor { return audit.UserActor(f.ids["poster"], "poster") }

// receipt captures accepted units (one per chassis; "" = no chassis) on a new draft receipt.
func (f *fixture) receipt(t *testing.T, chassis ...string) (grn string, units []string) {
	t.Helper()
	grn = f.scalar(t, `INSERT INTO goods_receipts (purchase_order_id, received_date, supplier_reference, created_by)
		VALUES ($1, DATE '2026-01-15', 'DC-1', $2) RETURNING id::text`, f.po, f.ids["poster"])
	line := f.scalar(t, `INSERT INTO goods_receipt_lines (goods_receipt_id, purchase_order_id, purchase_order_line_id,
		received_quantity, accepted_quantity, rejected_quantity) VALUES ($1, $2, $3, $4, $4, 0) RETURNING id::text`,
		grn, f.po, f.poLine, len(chassis))
	for i, c := range chassis {
		var ch any
		if c != "" {
			f.nextNo++
			ch = c + "-" + strconv.Itoa(f.nextNo)
		}
		units = append(units, f.scalar(t, `INSERT INTO goods_receipt_units (goods_receipt_id, goods_receipt_line_id, unit_no, condition,
			chassis_number, chassis_normalised) VALUES ($1, $2, $3, 'ACCEPTED', $4::text, upper(btrim($4::text))) RETURNING id::text`,
			grn, line, i+1, ch))
	}
	return grn, units
}

// register posts the receipt and registers its units (standing in for D-07) at location.
func (f *fixture) register(t *testing.T, location *string, chassis ...string) []string {
	t.Helper()
	ctx := context.Background()
	grn, units := f.receipt(t, chassis...)
	var ids []string
	err := pgx.BeginFunc(ctx, f.pool, func(tx pgx.Tx) error {
		for _, u := range units {
			r, err := vehicles.RegisterFromReceiptUnit(ctx, tx, f.actor(), "req-test", u, location)
			if err != nil {
				return err
			}
			ids = append(ids, r.ID)
		}
		_, err := tx.Exec(ctx, `UPDATE goods_receipts SET status = 'POSTED', posted_by = $2, posted_at = now() WHERE id = $1`, grn, f.ids["poster"])
		return err
	})
	if err != nil {
		t.Fatal(vehicles.CommitError(err))
	}
	return ids
}

// move runs stock.MoveUnit in its own transaction as the poster.
func (f *fixture) move(t *testing.T, unit, typ, key string, version *int) (stock.MoveResult, bool, error) {
	t.Helper()
	ctx := context.Background()
	var res stock.MoveResult
	var replayed bool
	err := pgx.BeginFunc(ctx, f.pool, func(tx pgx.Tx) error {
		var err error
		res, replayed, err = stock.MoveUnit(ctx, tx, f.actor(), access.Scope{All: true},
			stock.MoveRequest{UnitID: unit, Type: typ, IdempotencyKey: key, ExpectedVersion: version, RequestID: "req-test"})
		return err
	})
	return res, replayed, err
}

// available registers units with chassis numbers at L1 and passes them (PDI_PASS, as D-06 will).
func (f *fixture) available(t *testing.T, n int) []string {
	t.Helper()
	ch := make([]string, n)
	for i := range ch {
		ch[i] = "CH"
	}
	ids := f.register(t, &f.locL1, ch...)
	for i, id := range ids {
		if _, _, err := f.move(t, id, stock.TypePDIPass, "pdi-pass-"+strconv.Itoa(i)+"-"+id[:8], nil); err != nil {
			t.Fatal(err)
		}
	}
	return ids
}

func (f *fixture) unit(t *testing.T, id string) (status string, version int) {
	t.Helper()
	if err := f.pool.QueryRow(context.Background(), `SELECT status, version FROM vehicle_units WHERE id = $1`, id).Scan(&status, &version); err != nil {
		t.Fatal(err)
	}
	return status, version
}

// reconciles checks LED-003 for every unit: a RECEIPT first, each movement starting
// where the previous one ended, the last one equal to the unit, one item held.
func (f *fixture) reconciles(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	rows, err := f.pool.Query(ctx, `SELECT v.id::text, v.status, v.location_id::text, m.movement_type, m.from_status, m.to_status,
			m.from_location_id::text, m.to_location_id::text, m.quantity_before, m.quantity_after
		FROM vehicle_units v LEFT JOIN stock_movements m ON m.vehicle_unit_id = v.id ORDER BY v.id, m.seq`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	type state struct {
		status, loc *string
		qty         int
		started     bool
	}
	last := map[string]*state{}
	unitState := map[string][2]*string{}
	for rows.Next() {
		var id, status string
		var loc, typ, from, to, fromLoc, toLoc *string
		var qb, qa *int
		if err := rows.Scan(&id, &status, &loc, &typ, &from, &to, &fromLoc, &toLoc, &qb, &qa); err != nil {
			t.Fatal(err)
		}
		unitState[id] = [2]*string{&status, loc}
		s := last[id]
		if s == nil {
			s = &state{}
			last[id] = s
		}
		if typ == nil {
			t.Errorf("unit %s has no movement", id)
			continue
		}
		if !s.started && *typ != stock.TypeReceipt {
			t.Errorf("unit %s starts with %s", id, *typ)
		}
		if s.started && (!eq(s.status, from) || !eq(s.loc, fromLoc) || s.qty != *qb) {
			t.Errorf("unit %s: %s does not start where the previous movement ended", id, *typ)
		}
		s.status, s.loc, s.qty, s.started = to, toLoc, *qa, true
	}
	for id, s := range last {
		u := unitState[id]
		if s.started && (!eq(s.status, u[0]) || !eq(s.loc, u[1]) || s.qty != 1) {
			t.Errorf("unit %s is %v at %v, its ledger says %v at %v", id, *u[0], u[1], *s.status, s.loc)
		}
	}
}

func eq(a, b *string) bool { return (a == nil && b == nil) || (a != nil && b != nil && *a == *b) }

func sqlState(err error) (code, constraint string) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code, pgErr.ConstraintName
	}
	return "", ""
}

func ptr[T any](v T) *T { return &v }

// ── Tests ───────────────────────────────────────────────────────────────────

func TestRegistrationRecordsExactlyOneReceiptMovement(t *testing.T) {
	f := setup(t)
	ids := f.register(t, &f.locL1, "CH", "")
	for _, id := range ids {
		var typ, to, srcType, srcRef, actor, key, loc string
		var from *string
		var qb, qa int
		err := f.pool.QueryRow(context.Background(), `SELECT m.movement_type, m.from_status, m.to_status, m.quantity_before, m.quantity_after,
				m.source_type, m.source_reference, m.actor_id::text, m.idempotency_key, m.to_location_id::text
			FROM stock_movements m WHERE m.vehicle_unit_id = $1`, id).Scan(&typ, &from, &to, &qb, &qa, &srcType, &srcRef, &actor, &key, &loc)
		if err != nil {
			t.Fatalf("one movement per unit: %v", err)
		}
		if typ != stock.TypeReceipt || from != nil || to != stock.ReceivedStatus || qb != 0 || qa != 1 || srcType != stock.SourceGoodsReceipt ||
			!strings.HasPrefix(srcRef, "GRN-") || actor != f.ids["poster"] || !strings.HasPrefix(key, "receipt-unit-") || loc != f.locL1 {
			t.Errorf("receipt movement: %s %v->%s %d->%d %s %s %s %s %s", typ, from, to, qb, qa, srcType, srcRef, actor, key, loc)
		}
		// The registration's audit event names the movement; the movement's time is the server's.
		if n := f.count(t, `SELECT count(*) FROM audit_events a JOIN stock_movements m ON m.id::text = a.summary->>'movementId'
			WHERE a.action = 'vehicle_unit.registered' AND a.entity_id = $1 AND m.created_at = a.occurred_at`, id); n != 1 {
			t.Errorf("registration audit for %s: %d", id, n)
		}
	}
	// Registering the same captured unit again is refused and adds nothing.
	grnUnit := f.scalar(t, `SELECT goods_receipt_unit_id::text FROM vehicle_units WHERE id = $1`, ids[0])
	err := pgx.BeginFunc(context.Background(), f.pool, func(tx pgx.Tx) error {
		_, err := vehicles.RegisterFromReceiptUnit(context.Background(), tx, f.actor(), "", grnUnit, nil)
		return err
	})
	if !errors.Is(err, vehicles.ErrAlreadyRegistered) {
		t.Fatalf("second registration: %v", err)
	}
	if n := f.count(t, `SELECT count(*) FROM stock_movements`); n != 2 {
		t.Fatalf("movements: %d", n)
	}
	f.reconciles(t)
}

// A draft receipt never gets a unit or a movement: the whole registration rolls back.
func TestDraftReceiptCreatesNoUnitOrMovement(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	_, units := f.receipt(t, "CH")
	err := pgx.BeginFunc(ctx, f.pool, func(tx pgx.Tx) error {
		_, err := vehicles.RegisterFromReceiptUnit(ctx, tx, f.actor(), "", units[0], nil)
		return err
	})
	if !errors.Is(vehicles.CommitError(err), vehicles.ErrReceiptNotPosted) {
		t.Fatalf("draft: %v", err)
	}
	if n := f.count(t, `SELECT count(*) FROM vehicle_units`) + f.count(t, `SELECT count(*) FROM stock_movements`) +
		f.count(t, `SELECT count(*) FROM audit_events WHERE action LIKE 'vehicle_unit.%'`); n != 0 {
		t.Fatalf("rows left: %d", n)
	}
}

func TestDisplayMovesRecordOneMovementWithAudit(t *testing.T) {
	f := setup(t)
	id := f.available(t, 1)[0]
	_, v := f.unit(t, id)
	var res stock.MoveResult
	expect(t, f.h.display(f.mover, id, true, `{"version":`+strconv.Itoa(v)+`,"note":" Front window "}`, "display-on-0001"), http.StatusOK, &res)
	m := res.Movement
	if m.MovementType != stock.TypeDisplayOn || *m.FromStatus != "AVAILABLE" || m.ToStatus != "DISPLAY" || m.QuantityBefore != 1 ||
		m.QuantityAfter != 1 || *m.FromLocationID != f.locL1 || *m.ToLocationID != f.locL1 || m.SourceType != stock.SourceStaffAction ||
		m.SourceID != nil || m.ActorID != f.ids["mover"] || *m.Note != "Front window" || m.VehicleUnitID != id {
		t.Fatalf("movement %+v", m)
	}
	if res.Unit.Status != "DISPLAY" || res.Unit.Version != v+1 {
		t.Fatalf("unit %+v", res.Unit)
	}
	if s, ver := f.unit(t, id); s != "DISPLAY" || ver != v+1 {
		t.Fatalf("stored unit %s v%d", s, ver)
	}
	if n := f.count(t, `SELECT count(*) FROM audit_events WHERE action = 'vehicle_unit.moved' AND entity_id = $1
		AND actor_user_id = $2 AND summary->>'movementId' = $3 AND summary->'status'->>'to' = 'DISPLAY'
		AND occurred_at = (SELECT created_at FROM stock_movements WHERE id = $3::uuid)`, id, f.ids["mover"], m.ID); n != 1 {
		t.Fatalf("audit events: %d", n)
	}
	expect(t, f.h.display(f.mover, id, false, `{"version":`+strconv.Itoa(v+1)+`}`, "display-off-0001"), http.StatusOK, &res)
	if res.Movement.MovementType != stock.TypeDisplayOff || res.Unit.Status != "AVAILABLE" || res.Unit.Version != v+2 {
		t.Fatalf("take off display: %+v", res)
	}
	if n := f.count(t, `SELECT count(*) FROM stock_movements WHERE vehicle_unit_id = $1`, id); n != 4 {
		t.Fatalf("movements: %d (receipt, pdi pass, on, off)", n)
	}
	f.reconciles(t)
}

func TestIdempotentReplayAndKeyReuse(t *testing.T) {
	f := setup(t)
	ids := f.available(t, 2)
	body := `{"version":2,"note":"window"}`
	var first, again stock.MoveResult
	expect(t, f.h.display(f.mover, ids[0], true, body, "same-key-0001"), http.StatusOK, &first)
	w := f.h.display(f.mover, ids[0], true, body, "same-key-0001")
	expect(t, w, http.StatusOK, &again)
	if w.Header().Get("Idempotent-Replayed") != "true" || again.Movement.ID != first.Movement.ID || again.Unit.Version != 3 {
		t.Fatalf("replay: %s %+v", w.Header().Get("Idempotent-Replayed"), again)
	}
	// The same key with a different request: refused, nothing written.
	expectError(t, f.h.display(f.mover, ids[0], true, `{"version":2,"note":"other"}`, "same-key-0001"), http.StatusConflict, "IDEMPOTENCY_KEY_REUSED")
	expectError(t, f.h.display(f.mover, ids[0], false, body, "same-key-0001"), http.StatusConflict, "IDEMPOTENCY_KEY_REUSED")
	expectError(t, f.h.display(f.mover, ids[1], true, body, "same-key-0001"), http.StatusConflict, "IDEMPOTENCY_KEY_REUSED")
	// Keys are per user: another user's identical key is a new request.
	expect(t, f.h.display(f.other, ids[1], true, body, "same-key-0001"), http.StatusOK, nil)
	if n := f.count(t, `SELECT count(*) FROM stock_movements WHERE movement_type = 'DISPLAY_ON'`); n != 2 {
		t.Fatalf("display movements: %d", n)
	}
	if n := f.count(t, `SELECT count(*) FROM audit_events WHERE action = 'vehicle_unit.moved' AND summary->>'movementType' = 'DISPLAY_ON'`); n != 2 {
		t.Fatalf("audit events: %d", n)
	}
	// A retry after the unit moved on still returns the first movement, with the unit as it is now.
	expect(t, f.h.display(f.mover, ids[0], false, `{"version":3}`, "off-key-0001"), http.StatusOK, nil)
	expect(t, f.h.display(f.mover, ids[0], true, body, "same-key-0001"), http.StatusOK, &again)
	if again.Movement.ID != first.Movement.ID || again.Unit.Status != "AVAILABLE" || again.Unit.Version != 4 {
		t.Fatalf("late replay: %+v", again)
	}
	f.reconciles(t)
}

func TestStaleVersionInvalidStateAndChassisAreRefused(t *testing.T) {
	f := setup(t)
	id := f.available(t, 1)[0]
	expectError(t, f.h.display(f.mover, id, true, `{"version":1}`, "stale-0001"), http.StatusConflict, "VERSION_CONFLICT")
	expectError(t, f.h.display(f.mover, id, false, `{"version":2}`, "not-on-display"), http.StatusConflict, "INVALID_STATE")
	// A refused request does not burn its key: it can be sent again once corrected.
	expect(t, f.h.display(f.mover, id, true, `{"version":2}`, "stale-0001"), http.StatusOK, nil)
	// Internal moves follow the same rules (D-06 will call these).
	pending := f.register(t, nil, "CH", "")
	var inv *stock.InvalidStateError
	if _, _, err := f.move(t, pending[0], stock.TypeReject, "reject-0001", nil); !errors.As(err, &inv) || inv.Current != stock.ReceivedStatus {
		t.Fatalf("reject from pending PDI: %v", err)
	}
	if _, _, err := f.move(t, pending[0], stock.TypeDisplayOn, "display-0001", nil); !errors.As(err, &inv) {
		t.Fatalf("display from pending PDI: %v", err)
	}
	// INV-002: no chassis/frame number, no leaving RECEIVED_PENDING_PDI.
	if _, _, err := f.move(t, pending[1], stock.TypePDIPass, "pass-0001", nil); !errors.Is(err, stock.ErrChassisRequired) {
		t.Fatalf("pass without chassis: %v", err)
	}
	if _, _, err := f.move(t, pending[0], "SELL", "sell-0001", nil); err == nil {
		t.Fatal("unknown movement type accepted")
	}
	// QUALITY_HOLD then REJECT: the approved path to REJECTED.
	if _, _, err := f.move(t, pending[0], stock.TypeQualityHold, "hold-0001", nil); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.move(t, pending[0], stock.TypePDIPass, "pass-0002", nil); !errors.As(err, &inv) || inv.Current != "QUALITY_HOLD" {
		t.Fatalf("pass from hold: %v", err)
	}
	if res, _, err := f.move(t, pending[0], stock.TypeReject, "reject-0002", nil); err != nil || res.Unit.Status != "REJECTED" {
		t.Fatalf("reject from hold: %+v %v", res, err)
	}
	for _, typ := range []string{stock.TypePDIPass, stock.TypeQualityHold, stock.TypeDisplayOn, stock.TypeDisplayOff, stock.TypeReject} {
		if _, _, err := f.move(t, pending[0], typ, "after-reject-"+typ, nil); !errors.As(err, &inv) {
			t.Errorf("%s after REJECTED: %v", typ, err)
		}
	}
	if n := f.count(t, `SELECT count(*) FROM stock_movements WHERE vehicle_unit_id = $1`, pending[0]); n != 3 {
		t.Fatalf("movements: %d", n)
	}
	f.reconciles(t)
}

func TestLocationScopeAndPermissions(t *testing.T) {
	f := setup(t)
	id := f.available(t, 1)[0] // at L1
	expectError(t, f.h.display(f.scoped, id, true, `{"version":2}`, "scope-0001"), http.StatusNotFound, "NOT_FOUND")
	expectError(t, f.h.display(f.viewer, id, true, `{"version":2}`, "scope-0002"), http.StatusForbidden, "FORBIDDEN")
	expectError(t, f.h.display(f.mover, "00000000-0000-0000-0000-000000000000", true, `{"version":2}`, "scope-0003"), http.StatusNotFound, "NOT_FOUND")
	f.h.scope(f.scoped, f.locL1)
	expect(t, f.h.display(f.scoped, id, true, `{"version":2}`, "scope-0001"), http.StatusOK, nil)
	// A unit without a location is in everyone's scope (as for reading units, ADR-018).
	f.h.scope(f.scoped, f.locL2)
	noLoc := f.register(t, nil, "CH")[0]
	if _, _, err := f.move(t, noLoc, stock.TypePDIPass, "pass-noloc", nil); err != nil {
		t.Fatal(err)
	}
	expect(t, f.h.display(f.scoped, noLoc, true, `{"version":2}`, "scope-0004"), http.StatusOK, nil)
	if n := f.count(t, `SELECT count(*) FROM stock_movements WHERE movement_type = 'DISPLAY_ON'`); n != 2 {
		t.Fatalf("display movements: %d", n)
	}
}

// Many parallel requests with one key: exactly one movement; the rest replay it.
func TestConcurrentRequestsWithOneKeyApplyOnce(t *testing.T) {
	f := setup(t)
	id := f.available(t, 1)[0]
	const n = 8
	var wg sync.WaitGroup
	ws := make([]*httptest.ResponseRecorder, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ws[i] = f.h.display(f.mover, id, true, `{"version":2}`, "parallel-0001")
		}()
	}
	wg.Wait()
	fresh := 0
	var first string
	for i, w := range ws {
		var res stock.MoveResult
		expect(t, w, http.StatusOK, &res)
		if first == "" {
			first = res.Movement.ID
		}
		if res.Movement.ID != first {
			t.Errorf("request %d: movement %s, want %s", i, res.Movement.ID, first)
		}
		if w.Header().Get("Idempotent-Replayed") == "" {
			fresh++
		}
	}
	if fresh != 1 {
		t.Errorf("%d requests applied the move", fresh)
	}
	if c := f.count(t, `SELECT count(*) FROM stock_movements WHERE movement_type = 'DISPLAY_ON'`) +
		f.count(t, `SELECT count(*) FROM audit_events WHERE action = 'vehicle_unit.moved' AND summary->>'movementType' = 'DISPLAY_ON'`); c != 2 {
		t.Fatalf("movement + audit rows: %d", c)
	}
	if s, v := f.unit(t, id); s != "DISPLAY" || v != 3 {
		t.Fatalf("unit %s v%d", s, v)
	}
	f.reconciles(t)
}

// Independent requests (different keys) on one unit: the lock serialises them,
// the first wins and the others see the change instead of overwriting it.
func TestConcurrentIndependentMovesDoNotOverwrite(t *testing.T) {
	f := setup(t)
	id := f.available(t, 1)[0]
	const n = 8
	var wg sync.WaitGroup
	ws := make([]*httptest.ResponseRecorder, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ws[i] = f.h.display(f.mover, id, true, `{"version":2}`, "independent-"+strconv.Itoa(1000+i))
		}()
	}
	wg.Wait()
	ok := 0
	for _, w := range ws {
		if w.Code == http.StatusOK {
			ok++
			continue
		}
		expectError(t, w, http.StatusConflict, "VERSION_CONFLICT")
	}
	if ok != 1 {
		t.Fatalf("%d of %d moves applied", ok, n)
	}
	if c := f.count(t, `SELECT count(*) FROM stock_movements WHERE movement_type = 'DISPLAY_ON'`); c != 1 {
		t.Fatalf("display movements: %d", c)
	}
	f.reconciles(t)
}

// One key on two units at once: the unique key decides; the loser rolls back
// and is told the key was used for something else.
func TestConcurrentKeyReuseAcrossUnits(t *testing.T) {
	f := setup(t)
	ids := f.available(t, 2)
	ctx := context.Background()
	store := stock.NewPGStore(f.pool)
	actor := audit.UserActor(f.ids["mover"], "mover")
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, _, err := stock.MoveUnit(ctx, tx, actor, access.Scope{All: true},
		stock.MoveRequest{UnitID: ids[0], Type: stock.TypeDisplayOn, IdempotencyKey: "race-key-0001"}); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, _, err := store.Move(ctx, actor, access.Scope{All: true},
			stock.MoveRequest{UnitID: ids[1], Type: stock.TypeDisplayOn, IdempotencyKey: "race-key-0001"})
		done <- err
	}()
	// The second insert waits on the first transaction's key, which then commits.
	if err := waitForLock(ctx, f.pool); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, stock.ErrIdempotencyKeyReused) {
		t.Fatalf("second unit with the same key: %v", err)
	}
	if s, _ := f.unit(t, ids[1]); s != "AVAILABLE" {
		t.Fatalf("second unit moved: %s", s)
	}
	if c := f.count(t, `SELECT count(*) FROM stock_movements WHERE movement_type = 'DISPLAY_ON'`) +
		f.count(t, `SELECT count(*) FROM audit_events WHERE action = 'vehicle_unit.moved' AND summary->>'movementType' = 'DISPLAY_ON'`); c != 2 {
		t.Fatalf("rows: %d", c)
	}
	f.reconciles(t)
}

// waitForLock waits until some backend is blocked on a lock.
func waitForLock(ctx context.Context, pool *pgxpool.Pool) error {
	for range 200 {
		var waiting bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE wait_event_type = 'Lock'
			AND datname = current_database())`).Scan(&waiting); err != nil {
			return err
		}
		if waiting {
			return nil
		}
		if _, err := pool.Exec(ctx, `SELECT pg_sleep(0.02)`); err != nil {
			return err
		}
	}
	return errors.New("no transaction waited on the key")
}

// A failure after the movement leaves nothing: no movement, audit or unit change.
func TestAFailedMoveLeavesNothing(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	id := f.available(t, 1)[0]
	before := f.count(t, `SELECT count(*) FROM stock_movements`) + f.count(t, `SELECT count(*) FROM audit_events`)
	// The caller fails after MoveUnit succeeded (e.g. its own write failed): rolled back.
	err := pgx.BeginFunc(ctx, f.pool, func(tx pgx.Tx) error {
		if _, _, err := stock.MoveUnit(ctx, tx, f.actor(), access.Scope{All: true},
			stock.MoveRequest{UnitID: id, Type: stock.TypeDisplayOn, IdempotencyKey: "fail-after-0001"}); err != nil {
			return err
		}
		return errors.New("caller failed")
	})
	if err == nil {
		t.Fatal("expected failure")
	}
	// A write inside MoveUnit fails (request id too long for the movement row): nothing either.
	_, _, err = func() (stock.MoveResult, bool, error) {
		var res stock.MoveResult
		var rep bool
		err := pgx.BeginFunc(ctx, f.pool, func(tx pgx.Tx) error {
			var err error
			res, rep, err = stock.MoveUnit(ctx, tx, f.actor(), access.Scope{All: true},
				stock.MoveRequest{UnitID: id, Type: stock.TypeDisplayOn, IdempotencyKey: "fail-inside-0001", RequestID: strings.Repeat("r", 65)})
			return err
		})
		return res, rep, err
	}()
	if code, _ := sqlState(err); code != "23514" {
		t.Fatalf("over-long request id: %v", err)
	}
	if after := f.count(t, `SELECT count(*) FROM stock_movements`) + f.count(t, `SELECT count(*) FROM audit_events`); after != before {
		t.Fatalf("rows changed: %d -> %d", before, after)
	}
	if s, v := f.unit(t, id); s != "AVAILABLE" || v != 2 {
		t.Fatalf("unit %s v%d", s, v)
	}
	// The keys were not used up: the same requests now succeed.
	if _, rep, err := f.move(t, id, stock.TypeDisplayOn, "fail-after-0001", nil); err != nil || rep {
		t.Fatalf("retry: %v %v", rep, err)
	}
	f.reconciles(t)
}

// The database, not only the API, keeps the ledger append-only and the units
// changeable only through it.
func TestDatabaseRefusesDirectChanges(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	id := f.available(t, 1)[0]
	mov := f.scalar(t, `SELECT id::text FROM stock_movements WHERE vehicle_unit_id = $1 AND movement_type = 'PDI_PASS'`, id)
	for _, sql := range []string{
		`UPDATE stock_movements SET note = 'x' WHERE id = '` + mov + `'`,
		`UPDATE stock_movements SET to_status = 'SOLD' WHERE id = '` + mov + `'`,
		`DELETE FROM stock_movements WHERE id = '` + mov + `'`,
		`TRUNCATE stock_movements CASCADE`,
		`UPDATE vehicle_units SET status = 'DISPLAY' WHERE id = '` + id + `'`,
		`UPDATE vehicle_units SET location_id = '` + f.locL2 + `' WHERE id = '` + id + `'`,
		`UPDATE vehicle_units SET version = version + 1 WHERE id = '` + id + `'`,
		`DELETE FROM vehicle_units WHERE id = '` + id + `'`,
	} {
		if _, err := f.pool.Exec(ctx, sql); func() bool { c, _ := sqlState(err); return c != "42501" }() {
			t.Errorf("%s: %v", sql, err)
		}
	}
	// Setting the marker the movement trigger uses does not open a back door: a
	// statement sent by a client is never inside that trigger.
	err := pgx.BeginFunc(ctx, f.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT set_config('maa.stock_movement_unit', $1, true)`, id); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE vehicle_units SET status = 'DISPLAY', version = version + 1 WHERE id = $1`, id)
		return err
	})
	if c, _ := sqlState(err); c != "42501" {
		t.Fatalf("update with the marker set: %v", err)
	}
	insert := func(typ, from, to string) error {
		_, err := f.pool.Exec(ctx, `INSERT INTO stock_movements (vehicle_unit_id, movement_type, quantity_before, quantity_after,
				from_status, to_status, from_location_id, to_location_id, source_type, actor_id, idempotency_key, request_hash)
			SELECT id, $2::text, 1, 1, $3, $4::text, location_id, location_id, 'STAFF_ACTION', $5, 'direct-' || $2 || '-' || $4,
				repeat('0', 64) FROM vehicle_units WHERE id = $1`, id, typ, from, to, f.ids["poster"])
		return err
	}
	for _, c := range []struct {
		typ, from, to, constraint string
	}{
		{stock.TypeDisplayOn, "AVAILABLE", "SOLD", "stock_movements_transition_valid"},
		// An unknown type fails stock_movements_type_valid too; PostgreSQL reports the first CHECK it evaluates.
		{"SELL", "AVAILABLE", "SOLD", "stock_movements_transition_valid"},
		{stock.TypePDIPass, "QUALITY_HOLD", "AVAILABLE", "stock_movements_transition_valid"},
		{stock.TypeReject, "RECEIVED_PENDING_PDI", "REJECTED", "stock_movements_transition_valid"},
		// Valid edge, but the unit is AVAILABLE, not DISPLAY: refused by the apply trigger.
		{stock.TypeDisplayOff, "DISPLAY", "AVAILABLE", "stock_movements_from_matches_unit"},
	} {
		if _, constraint := sqlState(insert(c.typ, c.from, c.to)); constraint != c.constraint {
			t.Errorf("%s %s->%s: got %s, want %s", c.typ, c.from, c.to, constraint, c.constraint)
		}
	}
	// A reversal link is not approved yet; a second receipt is refused.
	_, err = f.pool.Exec(ctx, `INSERT INTO stock_movements (vehicle_unit_id, movement_type, quantity_before, quantity_after,
			from_status, to_status, from_location_id, to_location_id, source_type, actor_id, idempotency_key, request_hash, reverses_movement_id)
		SELECT id, 'DISPLAY_ON', 1, 1, 'AVAILABLE', 'DISPLAY', location_id, location_id, 'STAFF_ACTION', $2, 'direct-reversal', repeat('0', 64), $3
		FROM vehicle_units WHERE id = $1`, id, f.ids["poster"], mov)
	if _, c := sqlState(err); c != "stock_movements_reversal_not_approved" {
		t.Errorf("reversal link: %v", err)
	}
	_, err = f.pool.Exec(ctx, `INSERT INTO stock_movements (vehicle_unit_id, movement_type, quantity_before, quantity_after,
			to_status, source_type, source_id, source_reference, actor_id, idempotency_key, request_hash)
		SELECT id, 'RECEIPT', 0, 1, 'RECEIVED_PENDING_PDI', 'GOODS_RECEIPT', goods_receipt_id, 'GRN-X', $2, 'direct-receipt', repeat('0', 64)
		FROM vehicle_units WHERE id = $1`, id, f.ids["poster"])
	if _, c := sqlState(err); c != "stock_movements_receipt_once" {
		t.Errorf("second receipt: %v", err)
	}
	if s, v := f.unit(t, id); s != "AVAILABLE" || v != 2 {
		t.Fatalf("unit %s v%d", s, v)
	}
	f.reconciles(t)
}

// At commit every unit must agree with its ledger: a unit written without its
// RECEIPT movement, or a receipt movement from another goods receipt, is refused.
func TestUnitsMustReconcileWithTheLedgerAtCommit(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	grn, units := f.receipt(t, "CH")
	otherGRN, _ := f.receipt(t, "")
	direct := func(tx pgx.Tx) (string, error) {
		var id string
		err := tx.QueryRow(ctx, `INSERT INTO vehicle_units (brand_id, model_id, variant_id, source_type, supplier_id, purchase_order_id,
				purchase_order_line_id, goods_receipt_id, goods_receipt_line_id, goods_receipt_unit_id, received_date, unit_cost, registered_by)
			SELECT pl.brand_id, pl.model_id, pl.variant_id, 'GOODS_RECEIPT', po.supplier_id, po.id, pl.id, u.goods_receipt_id,
				u.goods_receipt_line_id, u.id, DATE '2026-01-15', 95000, $2
			FROM goods_receipt_units u JOIN goods_receipt_lines gl ON gl.id = u.goods_receipt_line_id
			JOIN purchase_order_lines pl ON pl.id = gl.purchase_order_line_id JOIN purchase_orders po ON po.id = pl.purchase_order_id
			WHERE u.id = $1 RETURNING id::text`, units[0], f.ids["poster"]).Scan(&id)
		if err == nil {
			_, err = tx.Exec(ctx, `UPDATE goods_receipts SET status = 'POSTED', posted_by = $2, posted_at = now() WHERE id = $1`, grn, f.ids["poster"])
		}
		return id, err
	}
	err := pgx.BeginFunc(ctx, f.pool, func(tx pgx.Tx) error {
		_, err := direct(tx)
		return err
	})
	if _, c := sqlState(err); c != "vehicle_units_ledger_reconciles" {
		t.Fatalf("unit without a movement: %v", err)
	}
	err = pgx.BeginFunc(ctx, f.pool, func(tx pgx.Tx) error {
		id, err := direct(tx)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO stock_movements (vehicle_unit_id, movement_type, quantity_before, quantity_after, to_status,
				source_type, source_id, source_reference, actor_id, idempotency_key, request_hash)
			VALUES ($1, 'RECEIPT', 0, 1, 'RECEIVED_PENDING_PDI', 'GOODS_RECEIPT', $2, 'GRN-X', $3, 'direct-receipt', repeat('0', 64))`,
			id, otherGRN, f.ids["poster"])
		return err
	})
	if _, c := sqlState(err); c != "stock_movements_receipt_matches_unit" {
		t.Fatalf("receipt movement from another receipt: %v", err)
	}
	if n := f.count(t, `SELECT count(*) FROM vehicle_units`) + f.count(t, `SELECT count(*) FROM stock_movements`); n != 0 {
		t.Fatalf("rows left: %d", n)
	}
}

// Migration 000015: units registered before E-01 get their RECEIPT movement;
// down removes the ledger and the guard refuses updates again; up again works.
func TestStockMovementMigrationRoundTrip(t *testing.T) {
	ctx := context.Background()
	url := dbtest.ScratchURL(t)
	m := dbtest.NewMigrator(t, url)
	if err := m.Migrate(14); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	f := setupOn(t, pool)
	// Version 14 has no ledger: a unit is written as version 14's registration did.
	grn, units := f.receipt(t, "CH")
	if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE goods_receipts SET status = 'POSTED', posted_by = $2, posted_at = now() WHERE id = $1`, grn, f.ids["poster"]); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO vehicle_units (brand_id, model_id, variant_id, chassis_number, location_id, source_type,
				supplier_id, purchase_order_id, purchase_order_line_id, goods_receipt_id, goods_receipt_line_id, goods_receipt_unit_id,
				received_date, unit_cost, registered_by)
			SELECT pl.brand_id, pl.model_id, pl.variant_id, u.chassis_number, $3, 'GOODS_RECEIPT', po.supplier_id, po.id, pl.id,
				u.goods_receipt_id, u.goods_receipt_line_id, u.id, DATE '2026-01-15', 95000, $2
			FROM goods_receipt_units u JOIN goods_receipt_lines gl ON gl.id = u.goods_receipt_line_id
			JOIN purchase_order_lines pl ON pl.id = gl.purchase_order_line_id JOIN purchase_orders po ON po.id = pl.purchase_order_id
			WHERE u.id = $1`, units[0], f.ids["poster"], f.locL1)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	unit := f.scalar(t, `SELECT id::text FROM vehicle_units`)
	if n := f.count(t, `SELECT count(*) FROM stock_movements m JOIN vehicle_units v ON v.id = m.vehicle_unit_id
		WHERE m.movement_type = 'RECEIPT' AND m.actor_id = v.registered_by AND m.created_at = v.created_at
		AND m.source_id = v.goods_receipt_id AND m.to_location_id = v.location_id
		AND m.idempotency_key = 'receipt-unit-' || v.goods_receipt_unit_id`); n != 1 {
		t.Fatalf("backfilled receipt movements: %d", n)
	}
	f.reconciles(t)
	if _, _, err := f.move(t, unit, stock.TypePDIPass, "after-up-0001", nil); err != nil {
		t.Fatal(err)
	}
	if err := m.Migrate(14); err != nil {
		t.Fatal(err)
	}
	if n := f.count(t, `SELECT count(*) FROM pg_tables WHERE tablename = 'stock_movements'`); n != 0 {
		t.Fatal("down: ledger kept")
	}
	if _, err := pool.Exec(ctx, `UPDATE vehicle_units SET status = 'DISPLAY' WHERE id = $1`, unit); func() bool { c, _ := sqlState(err); return c != "42501" }() {
		t.Fatalf("down: update allowed: %v", err)
	}
	// Up again over a unit already past RECEIVED_PENDING_PDI (moved under 15): refused, nothing changed.
	if err := m.Up(); err == nil || !strings.Contains(err.Error(), "resolve them before applying E-01") {
		t.Fatalf("up over a moved unit: %v", err)
	}
}
