package vehicles

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/httpx"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/ageing"
)

// Stock ageing (E-07) on a throwaway migrated database. Units are registered
// from posted goods receipts with chosen received dates; the business clock
// is fixed (harness.setToday), so every age is exact. Band configuration: the
// migration's LED-008 proposal unless a test proposes another. Placeholder data.

const ageToday = "2026-06-30"

type ageFixture struct {
	fixture
	po, poLine string
	n          int
}

func setupAgeing(t *testing.T) *ageFixture {
	t.Helper()
	af := &ageFixture{fixture: setup(t)}
	f := af.fixture
	af.po = f.scalar(t, `INSERT INTO purchase_orders (supplier_id, created_by, expected_date) VALUES ($1, $2, current_date + 7) RETURNING id::text`,
		f.supplier, f.ids["buyer"])
	af.poLine = f.scalar(t, `INSERT INTO purchase_order_lines (purchase_order_id, line_no, brand_id, model_id, variant_id, quantity, unit_cost)
		VALUES ($1, 1, $2, $3, $4, 200, 95000.00) RETURNING id::text`, af.po, f.brandA, f.modelA, f.variantA)
	f.exec(t, `UPDATE purchase_orders SET status = 'APPROVED', submitted_by = $2, submitted_at = now(), approval_basis = 'APPROVER',
		approved_by = $3, approved_at = now() WHERE id = $1`, af.po, f.ids["buyer"], f.ids["approver"])
	f.h.setToday(ageToday)
	return af
}

// receivedDaysAgo registers one unit from a goods receipt received that many days before ageToday, posted.
func (af *ageFixture) receivedDaysAgo(t *testing.T, days int, location *string) string {
	t.Helper()
	d, _ := ageing.ParseDate(ageToday)
	return af.receivedOn(t, d.AddDate(0, 0, -days).Format("2006-01-02"), location)
}

func (af *ageFixture) receivedOn(t *testing.T, date string, location *string) string {
	t.Helper()
	f := af.fixture
	af.n++
	grn := f.scalar(t, `INSERT INTO goods_receipts (purchase_order_id, received_date, supplier_reference, created_by)
		VALUES ($1, $2::date, 'DC-AGE', $3) RETURNING id::text`, af.po, date, f.ids["poster"])
	line := f.receiptLine(t, grn, af.po, af.poLine, 1, 1, 0)
	unit := f.receiptUnit(t, grn, line, 1, "ACCEPTED", "CH-AGE-"+strconv.Itoa(af.n), "", "", "")
	regs, err := f.registerAll(context.Background(), grn, true, location, unit)
	if err != nil {
		t.Fatal(err)
	}
	return regs[0].ID
}

// tamper runs sql with the user triggers of table off (as a database owner could), one transaction.
func (af *ageFixture) tamper(t *testing.T, table, sql string, args ...any) {
	t.Helper()
	ctx := context.Background()
	err := pgx.BeginFunc(ctx, af.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `ALTER TABLE `+table+` DISABLE TRIGGER USER`); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `ALTER TABLE `+table+` ENABLE TRIGGER USER`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func (af *ageFixture) summary(t *testing.T, s session, query string) AgeingSummary {
	t.Helper()
	var sum AgeingSummary
	expect(t, af.h.get(s, "/vehicle-units/ageing-summary"+query), http.StatusOK, &sum)
	if sum.Total != sum.WithOrigin+sum.MissingOrigin {
		t.Fatalf("%s: total %d != %d + %d", query, sum.Total, sum.WithOrigin, sum.MissingOrigin)
	}
	banded := sum.Unbanded
	for _, b := range sum.Bands {
		banded += b.Count
	}
	if banded != sum.WithOrigin {
		t.Fatalf("%s: bands %d != withOrigin %d", query, banded, sum.WithOrigin)
	}
	missing := 0
	for _, n := range sum.MissingByReason {
		missing += n
	}
	if missing != sum.MissingOrigin {
		t.Fatalf("%s: reasons %d != missingOrigin %d", query, missing, sum.MissingOrigin)
	}
	return sum
}

func (af *ageFixture) units(t *testing.T, s session, query string) map[string]Unit {
	t.Helper()
	sep := "?"
	if strings.Contains(query, "?") {
		sep = "&"
	}
	p := fixtureList(t, af.fixture, s, query+sep+"limit=100")
	out := map[string]Unit{}
	for _, u := range p.Items {
		out[u.ID] = u
	}
	if len(out) != p.Total {
		t.Fatalf("%s: %d items, total %d", query, len(out), p.Total)
	}
	return out
}

func fixtureList(t *testing.T, f fixture, s session, query string) UnitPage {
	t.Helper()
	var p UnitPage
	expect(t, f.h.get(s, "/vehicle-units"+query), http.StatusOK, &p)
	return p
}

func bandCount(sum AgeingSummary, key string) int {
	for _, b := range sum.Bands {
		if b.Key == key {
			return b.Count
		}
	}
	return -1
}

// Scenarios A and D: GRN stock; age is business date − posted receipt date, every
// band at its lower and upper boundary and one day either side; list, filter and
// summary agree.
func TestAgeingBandBoundariesAndFilters(t *testing.T) {
	af := setupAgeing(t)
	ages := []int{0, 1, 29, 30, 31, 32, 59, 60, 61, 62, 89, 90, 91, 92, 400}
	wantBand := map[int]string{0: "0-30", 1: "0-30", 29: "0-30", 30: "0-30", 31: "31-60", 32: "31-60", 59: "31-60", 60: "31-60",
		61: "61-90", 62: "61-90", 89: "61-90", 90: "61-90", 91: "over-90", 92: "over-90", 400: "over-90"}
	byUnit := map[string]int{}
	for _, a := range ages {
		byUnit[af.receivedDaysAgo(t, a, ptr(af.locL1))] = a
	}
	all := af.units(t, af.allLocs, "")
	if len(all) != len(ages) {
		t.Fatalf("%d units listed", len(all))
	}
	for id, age := range byUnit {
		u := all[id]
		g := u.Ageing
		if g.Status != ageing.StatusAged || g.OriginSource == nil || *g.OriginSource != ageing.SourceGRN || g.AgeDays == nil ||
			*g.AgeDays != age || u.AgeDays == nil || *u.AgeDays != age || g.OriginDate == nil || *g.OriginDate != u.ReceivedDate ||
			g.BandKey == nil || *g.BandKey != wantBand[age] || g.MissingReason != nil || g.AsOf != ageToday {
			t.Errorf("age %d: %+v (ageDays %v)", age, g, u.AgeDays)
		}
	}
	// Detail says the same.
	for id, age := range byUnit {
		var d Detail
		expect(t, af.h.get(af.allLocs, "/vehicle-units/"+id), http.StatusOK, &d)
		if d.Ageing.AgeDays == nil || *d.Ageing.AgeDays != age || *d.Ageing.BandKey != wantBand[age] {
			t.Errorf("detail age %d: %+v", age, d.Ageing)
		}
		break
	}
	// Summary: per band, and each band filter lists exactly those units.
	sum := af.summary(t, af.allLocs, "")
	if !sum.Configured || sum.BandSet == nil || sum.BandSet.Version != 1 || sum.BandSet.Status != "PROPOSED" ||
		sum.AsOf != ageToday || sum.TimeZone != ageing.DefaultTimeZone || sum.Total != len(ages) || sum.MissingOrigin != 0 {
		t.Fatalf("summary %+v", sum)
	}
	for key, want := range map[string]int{"0-30": 4, "31-60": 4, "61-90": 4, "over-90": 3} {
		if got := bandCount(sum, key); got != want {
			t.Errorf("band %s: %d, want %d", key, got, want)
		}
		listed := af.units(t, af.allLocs, "?ageingBand="+key)
		if len(listed) != want {
			t.Errorf("ageingBand=%s lists %d, want %d", key, len(listed), want)
		}
		for id := range listed {
			if wantBand[byUnit[id]] != key {
				t.Errorf("ageingBand=%s lists a unit aged %d", key, byUnit[id])
			}
		}
		if s := af.summary(t, af.allLocs, "?ageingBand="+key); s.Total != want || bandCount(s, key) != want {
			t.Errorf("summary ageingBand=%s: %+v", key, s)
		}
	}
	// minAgeDays/maxAgeDays use the same business date (boundaries inclusive).
	if n := len(af.units(t, af.allLocs, "?minAgeDays=30&maxAgeDays=31")); n != 2 {
		t.Errorf("30..31 lists %d", n)
	}
	if n := len(af.units(t, af.allLocs, "?originStatus=KNOWN")); n != len(ages) {
		t.Errorf("KNOWN lists %d", n)
	}
	if n := len(af.units(t, af.allLocs, "?originStatus=MISSING")); n != 0 {
		t.Errorf("MISSING lists %d", n)
	}
	// Unknown band key: 422, not an empty list.
	expectError(t, af.h.get(af.allLocs, "/vehicle-units?ageingBand=0-45"), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	expectError(t, af.h.get(af.allLocs, "/vehicle-units/ageing-summary?ageingBand=0-45"), http.StatusUnprocessableEntity, "VALIDATION_FAILED")

	// The business date moves with the clock: one day later, the 30-day unit is 31 (next band).
	af.h.setToday("2026-07-01")
	for id, u := range af.units(t, af.allLocs, "") {
		if byUnit[id] == 30 && (*u.Ageing.AgeDays != 31 || *u.Ageing.BandKey != "31-60") {
			t.Errorf("next day: %+v", u.Ageing)
		}
	}
}

// Scenario C and §4.2: units without a reliable origin get no age and no band,
// are counted and listed separately with a reason, never as 0 days.
func TestAgeingMissingAndUnreliableOrigins(t *testing.T) {
	af := setupAgeing(t)
	good := af.receivedDaysAgo(t, 10, ptr(af.locL1))
	mismatch := af.receivedDaysAgo(t, 20, ptr(af.locL1))
	notPosted := af.receivedDaysAgo(t, 40, ptr(af.locL1))
	future := af.receivedDaysAgo(t, -3, ptr(af.locL1)) // received 3 days after the business date
	// Drift injected below the application (triggers off): the unit's date no longer the receipt's,
	// and a receipt no longer POSTED.
	af.tamper(t, "vehicle_units", `UPDATE vehicle_units SET received_date = received_date - 1 WHERE id = $1`, mismatch)
	af.tamper(t, "goods_receipts", `UPDATE goods_receipts SET status = 'DRAFT', posted_by = NULL, posted_at = NULL
		WHERE id = (SELECT goods_receipt_id FROM vehicle_units WHERE id = $1)`, notPosted)

	all := af.units(t, af.allLocs, "")
	want := map[string]string{mismatch: ageing.ReasonGRNDateMismatch, notPosted: ageing.ReasonGRNNotPosted,
		future: ageing.ReasonFutureOriginDate}
	for id, reason := range want {
		g := all[id].Ageing
		if g.Status != ageing.StatusMissingOrigin || g.AgeDays != nil || all[id].AgeDays != nil || g.BandKey != nil ||
			g.BandLabel != nil || g.OriginSource != nil || g.MissingReason == nil || *g.MissingReason != reason ||
			g.MissingExplanation == nil || *g.MissingExplanation == "" {
			t.Errorf("%s: %+v", reason, g)
		}
	}
	if g := all[good].Ageing; g.Status != ageing.StatusAged || *g.AgeDays != 10 {
		t.Errorf("good unit %+v", g)
	}
	missing := af.units(t, af.allLocs, "?originStatus=MISSING")
	if len(missing) != 3 || missing[good].ID != "" {
		t.Errorf("MISSING lists %d", len(missing))
	}
	known := af.units(t, af.allLocs, "?originStatus=KNOWN")
	if len(known) != 1 || known[good].ID == "" {
		t.Errorf("KNOWN lists %d", len(known))
	}
	// No band or age filter ever selects them (not 0 days).
	if n := len(af.units(t, af.allLocs, "?ageingBand=0-30")); n != 1 {
		t.Errorf("0-30 lists %d", n)
	}
	if n := len(af.units(t, af.allLocs, "?maxAgeDays=0")); n != 0 {
		t.Errorf("maxAgeDays=0 lists %d", n)
	}
	sum := af.summary(t, af.allLocs, "")
	if sum.Total != 4 || sum.WithOrigin != 1 || sum.MissingOrigin != 3 || sum.MissingByReason[ageing.ReasonGRNDateMismatch] != 1 ||
		sum.MissingByReason[ageing.ReasonGRNNotPosted] != 1 || sum.MissingByReason[ageing.ReasonFutureOriginDate] != 1 ||
		bandCount(sum, "0-30") != 1 {
		t.Fatalf("summary %+v", sum)
	}
	if s := af.summary(t, af.allLocs, "?originStatus=MISSING"); s.Total != 3 || s.WithOrigin != 0 {
		t.Fatalf("summary MISSING %+v", s)
	}
	// Conflicting filters are refused.
	for _, q := range []string{"?originStatus=MISSING&ageingBand=0-30", "?originStatus=MISSING&minAgeDays=1", "?originStatus=UNKNOWN"} {
		expectError(t, af.h.get(af.allLocs, "/vehicle-units"+q), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	}
}

// §5.5, Scenario E: permissions and location scope decide which units are aged and counted.
func TestAgeingPermissionsAndScope(t *testing.T) {
	af := setupAgeing(t)
	l1 := af.receivedDaysAgo(t, 5, ptr(af.locL1))
	af.receivedDaysAgo(t, 50, ptr(af.locL2))
	noLoc := af.receivedDaysAgo(t, 95, nil)
	// No permission.
	for _, p := range []string{"/vehicle-units/ageing-summary", "/stock-ageing/configuration", "/vehicle-units?ageingBand=0-30"} {
		expectError(t, af.h.get(af.nobody, p), http.StatusForbidden, "FORBIDDEN")
	}
	// scoped: L1 only (plus units without a location).
	sum := af.summary(t, af.scoped, "")
	if sum.Total != 2 || bandCount(sum, "0-30") != 1 || bandCount(sum, "over-90") != 1 || bandCount(sum, "31-60") != 0 {
		t.Fatalf("scoped summary %+v", sum)
	}
	got := af.units(t, af.scoped, "")
	if len(got) != 2 || got[l1].ID == "" || got[noLoc].ID == "" {
		t.Fatalf("scoped lists %v", got)
	}
	expectError(t, af.h.get(af.scoped, "/vehicle-units/ageing-summary?locationId="+af.locL2), http.StatusUnprocessableEntity,
		"VALIDATION_FAILED")
	// Every location; and a location filter narrows the summary.
	if s := af.summary(t, af.allLocs, ""); s.Total != 3 {
		t.Fatalf("all %+v", s)
	}
	if s := af.summary(t, af.allLocs, "?locationId="+af.locL2); s.Total != 1 || bandCount(s, "31-60") != 1 {
		t.Fatalf("L2 %+v", s)
	}
	// Brand filter (brand scope per user does not exist yet, RB-5).
	if s := af.summary(t, af.allLocs, "?brandId="+af.brandB); s.Total != 0 {
		t.Fatalf("brand B %+v", s)
	}
	if s := af.summary(t, af.allLocs, "?brandId="+af.brandA); s.Total != 3 {
		t.Fatalf("brand A %+v", s)
	}
	// Reading the configuration needs vehicle_units.view; changing it stock.ageing.configure.
	var cfg ageing.Configuration
	expect(t, af.h.get(af.viewer, "/stock-ageing/configuration"), http.StatusOK, &cfg)
	if !cfg.Configured || cfg.BandSet.Version != 1 || len(cfg.BandSet.Bands) != 4 || cfg.TimeZone != ageing.DefaultTimeZone ||
		cfg.AsOf != ageToday || cfg.BoundaryRule == "" {
		t.Fatalf("configuration %+v", cfg)
	}
	body := `{"expectedVersion":1,"source":"test","bands":[{"key":"all","label":"All","minDays":0,"maxDays":null}]}`
	expectError(t, af.h.raw(http.MethodPost, "/api/v1/stock-ageing/band-sets", body, "application/json", &af.viewer),
		http.StatusForbidden, "FORBIDDEN")
	expectError(t, af.h.raw(http.MethodPost, "/api/v1/stock-ageing/band-sets/approve", `{"version":1}`, "application/json", &af.viewer),
		http.StatusForbidden, "FORBIDDEN")
}

// §4.3: configuration is validated, versioned, approved by someone else, and a
// change re-bands stock on read without rewriting any stock record (§5.7 #18).
func TestAgeingBandConfigurationLifecycle(t *testing.T) {
	af := setupAgeing(t)
	f := af.fixture
	u := af.receivedDaysAgo(t, 45, ptr(af.locL1))
	proposer, approver := f.allLocs, f.coster
	f.h.grant(proposer, access.PermVehicleUnitView, access.PermAllLocations, access.PermAgeingConfigure)
	f.h.grant(approver, access.PermVehicleUnitView, access.PermAllLocations, access.PermAgeingConfigure)
	fingerprint := func() string {
		var parts []string
		for _, table := range []string{"vehicle_units", "goods_receipts", "goods_receipt_units", "stock_movements", "vehicle_unit_events"} {
			parts = append(parts, f.scalar(t, `SELECT count(*)::text || ':' || coalesce(md5(string_agg(t::text, '|' ORDER BY t::text)), '')
				FROM `+table+` t`))
		}
		return strings.Join(parts, " ")
	}
	before := fingerprint()
	post := func(s session, path, body string) *httptest.ResponseRecorder {
		return af.h.raw(http.MethodPost, "/api/v1/stock-ageing/"+path, body, "application/json", &s)
	}

	// Invalid configurations (overlap, gap, not from 0, open band not last, bounded last band, duplicate key, bad key/label).
	for name, c := range map[string]struct{ bands, field string }{
		"overlap":       {`[{"key":"a","label":"A","minDays":0,"maxDays":30},{"key":"b","label":"B","minDays":30,"maxDays":null}]`, "bands[1].minDays"},
		"gap":           {`[{"key":"a","label":"A","minDays":0,"maxDays":30},{"key":"b","label":"B","minDays":32,"maxDays":null}]`, "bands[1].minDays"},
		"not from 0":    {`[{"key":"a","label":"A","minDays":1,"maxDays":null}]`, "bands[0].minDays"},
		"open not last": {`[{"key":"a","label":"A","minDays":0,"maxDays":null},{"key":"b","label":"B","minDays":1,"maxDays":null}]`, "bands[0].maxDays"},
		"bounded last":  {`[{"key":"a","label":"A","minDays":0,"maxDays":30}]`, "bands[0].maxDays"},
		"duplicate key": {`[{"key":"a","label":"A","minDays":0,"maxDays":30},{"key":"a","label":"B","minDays":31,"maxDays":null}]`, "bands[1].key"},
		"bad key":       {`[{"key":"A B","label":"A","minDays":0,"maxDays":null}]`, "bands[0].key"},
		"negative":      {`[{"key":"a","label":"A","minDays":-1,"maxDays":null}]`, "bands[0].minDays"},
		"none":          {`[]`, "bands"},
	} {
		w := post(proposer, "band-sets", `{"expectedVersion":1,"source":"test","bands":`+c.bands+`}`)
		e := expectError(t, w, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
		if !slices.ContainsFunc(e.Details, func(d httpx.FieldError) bool { return d.Field == c.field }) {
			t.Errorf("%s: details %+v, want %s", name, e.Details, c.field)
		}
	}
	// Stale version.
	valid := `{"source":"Test thresholds","bands":[{"key":"fresh","label":"Up to 45 days","minDays":0,"maxDays":45},
		{"key":"old","label":"Over 45 days","minDays":46,"maxDays":null}]}`
	expectError(t, post(proposer, "band-sets", `{"expectedVersion":0,`+valid[1:]), http.StatusConflict, "VERSION_CONFLICT")
	// A valid proposal becomes version 2, PROPOSED, and applies on the next read.
	var set ageing.BandSet
	expect(t, post(proposer, "band-sets", `{"expectedVersion":1,`+valid[1:]), http.StatusCreated, &set)
	if set.Version != 2 || set.Status != ageing.SetProposed || len(set.Bands) != 2 || set.CreatedByID == nil {
		t.Fatalf("proposed %+v", set)
	}
	if g := af.units(t, f.allLocs, "")[u].Ageing; *g.AgeDays != 45 || *g.BandKey != "fresh" {
		t.Fatalf("after proposal %+v", g)
	}
	// The proposer cannot approve it (audited DENIED); another holder can, once.
	expectError(t, post(proposer, "band-sets/approve", `{"version":2}`), http.StatusForbidden, "SELF_APPROVAL_NOT_ALLOWED")
	if n := f.count(t, `SELECT count(*) FROM audit_events WHERE action = 'stock_ageing.bands_approved' AND outcome = 'DENIED'`); n != 1 {
		t.Fatalf("denied audits %d", n)
	}
	expectError(t, post(approver, "band-sets/approve", `{"version":1}`), http.StatusConflict, "INVALID_STATE")
	expect(t, post(approver, "band-sets/approve", `{"version":2,"reference":"Test approval"}`), http.StatusOK, &set)
	if set.Status != ageing.SetApproved || set.ApprovedByID == nil || *set.ApprovedByID != f.ids["coster"] {
		t.Fatalf("approved %+v", set)
	}
	expectError(t, post(approver, "band-sets/approve", `{"version":2}`), http.StatusConflict, "INVALID_STATE")
	if n := f.count(t, `SELECT count(*) FROM audit_events WHERE action IN ('stock_ageing.bands_proposed', 'stock_ageing.bands_approved')
		AND outcome = 'SUCCESS'`); n != 2 {
		t.Fatalf("audits %d", n)
	}
	// History kept: version 1 is unchanged; the database refuses edits, deletes and a gapped set.
	if n := f.count(t, `SELECT count(*) FROM stock_ageing_bands WHERE set_version = 1`); n != 4 {
		t.Fatalf("version 1 bands %d", n)
	}
	for _, sql := range []string{
		`UPDATE stock_ageing_bands SET max_days = 31 WHERE set_version = 1 AND position = 1`,
		`DELETE FROM stock_ageing_bands WHERE set_version = 1`,
		`DELETE FROM stock_ageing_band_sets WHERE version = 1`,
		`UPDATE stock_ageing_band_sets SET source = 'x' WHERE version = 2`,
	} {
		if _, err := f.pool.Exec(context.Background(), sql); err == nil {
			t.Errorf("database accepted %s", sql)
		}
	}
	err := pgx.BeginFunc(context.Background(), f.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(context.Background(), `INSERT INTO stock_ageing_band_sets (version, source) VALUES (3, 'gap')`); err != nil {
			return err
		}
		_, err := tx.Exec(context.Background(), `INSERT INTO stock_ageing_bands VALUES (3, 1, 'a', 'A', 0, 10), (3, 2, 'b', 'B', 12, NULL)`)
		return err
	})
	if code, constraint := sqlState(err); code != "23514" || constraint != "stock_ageing_bands_contiguous" {
		t.Fatalf("gapped set: %v (%s %s)", err, code, constraint)
	}
	// Nothing about stock changed through any of it.
	if after := fingerprint(); after != before {
		t.Fatal("ageing changed stock, receipt or movement records")
	}
}
