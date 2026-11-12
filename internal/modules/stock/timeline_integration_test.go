package stock_test

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/stock"
)

// Stock movement timeline (E-02) on a throwaway migrated database. The ledger is
// built only through the existing write paths (registration, MoveUnit, the
// display/parts/count routes); every expectation is checked against the
// stock_movements table itself. Grants are test-only (in-memory harness).

type timelineFixture struct {
	*countsFixture
	full     session // everything the timeline can show, every location
	vehicles session // vehicle_units.view only, every location (= the E-01 "viewer")
	scoped   session // vehicle_units.view + stock.parts.view at L2 only
	parts    session // stock.parts.view only, every location: no suppliers, counts or receipts
	l1Units  []string
	l2Unit   string
	noLoc    string
	grnRef   string
	countRef string
	mark     time.Time // between the vehicle movements and the parts movements
}

func setupTimeline(t *testing.T) *timelineFixture {
	t.Helper()
	tf := &timelineFixture{countsFixture: setupCounts(t)}
	f := tf.fixture
	// Vehicles: two units at L1 (RECEIPT, PDI_PASS), one at L2 and one without a location (RECEIPT only).
	tf.l1Units = f.available(t, 2)
	tf.l2Unit = f.register(t, &f.locL2, "CH")[0]
	tf.noLoc = f.register(t, nil, "CH")[0]
	// The mover puts one L1 unit on display through the API (actor: mover, source STAFF_ACTION).
	_, version := f.unit(t, tf.l1Units[0])
	f.h.grant(f.mover, append(append([]string{access.PermVehicleUnitDisplayManage}, partsAll...), countCreate...)...)
	expect(t, f.h.display(f.mover, tf.l1Units[0], true, `{"version":`+strconv.Itoa(version)+`}`, "display-0001"), http.StatusOK, nil)
	tf.grnRef = f.scalar(t, `SELECT source_reference FROM stock_movements WHERE vehicle_unit_id = $1 AND movement_type = 'RECEIPT'`, tf.l2Unit)

	time.Sleep(20 * time.Millisecond)
	tf.mark = time.Now()
	time.Sleep(20 * time.Millisecond)

	// Parts: setupCounts received 10 at L1 (PARTS_RECEIPT, DC-1). Issue 2 at L1, receive 4 at L2, then a count at L1
	// approved by the approver (COUNT_ADJUSTMENT 8 -> 7).
	p := tf.partsFixture
	expect(t, p.call(p.mover, p.item, "issues", p.issue("2"), "issue-0001"), http.StatusOK, nil)
	expect(t, p.call(p.mover, p.item, "receipts", p.receipt("4", "locationId", f.locL2, "sourceReference", "DC-L2"), "rcpt-0002"),
		http.StatusOK, nil)
	sc := tf.submitted(t, "count-0001", "7")
	expect(t, tf.approve(tf.approver, sc, "approve-0001"), http.StatusOK, nil)
	tf.countRef = sc.CountNumber

	tf.full = f.other
	f.h.grant(tf.full, access.PermVehicleUnitView, access.PermPartsView, access.PermGoodsReceiptView, access.PermAdjustmentView,
		access.PermSupplierView, access.PermAllLocations)
	tf.vehicles = f.viewer
	tf.scoped = f.scoped
	f.h.grant(tf.scoped, access.PermVehicleUnitView, access.PermPartsView, access.PermGoodsReceiptView, access.PermAdjustmentView)
	tf.parts = f.h.login("buyer", f.ids["buyer"])
	f.h.grant(tf.parts, access.PermPartsView, access.PermAllLocations)
	return tf
}

func (tf *timelineFixture) list(t *testing.T, s session, query string) stock.TimelinePage {
	t.Helper()
	var p stock.TimelinePage
	expect(t, tf.h.raw(http.MethodGet, "/api/v1/stock-movements?"+query, "", &s, nil), http.StatusOK, &p)
	if len(p.Items) > p.Limit || p.Total < len(p.Items) {
		t.Fatalf("%s: page %d/%d of %d", query, len(p.Items), p.Limit, p.Total)
	}
	return p
}

// ids lists the ledger IDs a SQL condition selects, in ledger order.
func (tf *timelineFixture) ids(t *testing.T, where string, args ...any) []string {
	t.Helper()
	rows, err := tf.pool.Query(context.Background(), `SELECT m.id::text FROM stock_movements m
		LEFT JOIN vehicle_units v ON v.id = m.vehicle_unit_id WHERE `+where+` ORDER BY m.seq`, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		out = append(out, id)
	}
	return out
}

func idsOf(p stock.TimelinePage) []string {
	out := make([]string, len(p.Items))
	for i, m := range p.Items {
		out[i] = m.ID
	}
	return out
}

// sameAs: the API returned exactly the SQL selection, oldest first (order=asc).
func (tf *timelineFixture) sameAs(t *testing.T, s session, query, where string, args ...any) stock.TimelinePage {
	t.Helper()
	p := tf.list(t, s, "order=asc&limit=100&"+query)
	want := tf.ids(t, where, args...)
	if got := idsOf(p); !slices.Equal(got, want) || p.Total != len(want) {
		t.Fatalf("%s: got %d %v (total %d), want %d %v", query, len(got), got, p.Total, len(want), want)
	}
	return p
}

func q(kv ...string) string {
	v := url.Values{}
	for i := 0; i+1 < len(kv); i += 2 {
		v.Set(kv[i], kv[i+1])
	}
	return v.Encode()
}

// 1, 2, 3, 11, 24: the timeline is the ledger — same rows, same order, same values; pages are stable.
func TestTimelineMatchesTheLedger(t *testing.T) {
	tf := setupTimeline(t)
	all := tf.sameAs(t, tf.full, "", "true")
	// 4 vehicle receipts, 2 PDI passes, 1 display, 2 parts receipts, 1 issue, 1 count adjustment.
	if len(all.Items) != 11 {
		t.Fatalf("ledger has %d movements, want 11", len(all.Items))
	}
	// Every field as recorded.
	ctx := context.Background()
	for _, m := range all.Items {
		var itemType, typ, qb, qa, src string
		var fromStatus, toStatus, fromLoc, toLoc, srcID, srcRef, uom *string
		var actor string
		var created time.Time
		if err := tf.pool.QueryRow(ctx, `SELECT item_type, movement_type, quantity_before::text, quantity_after::text, source_type,
				from_status, to_status, from_location_id::text, to_location_id::text, source_id::text, source_reference, uom,
				actor_id::text, created_at FROM stock_movements WHERE id = $1`, m.ID).Scan(&itemType, &typ, &qb, &qa, &src,
			&fromStatus, &toStatus, &fromLoc, &toLoc, &srcID, &srcRef, &uom, &actor, &created); err != nil {
			t.Fatal(err)
		}
		if m.ItemType != itemType || m.MovementType != typ || m.QuantityBefore != qb || m.QuantityAfter != qa || m.SourceType != src ||
			!eq(m.FromStatus, fromStatus) || !eq(m.ToStatus, toStatus) || !eq(m.FromLocationID, fromLoc) || !eq(m.ToLocationID, toLoc) ||
			!eq(m.SourceID, srcID) || !eq(m.SourceReference, srcRef) || !eq(m.UOM, uom) || m.ActorID != actor ||
			!m.CreatedAt.Equal(created) || m.SourceHidden {
			t.Errorf("movement %s differs from the ledger: %+v", m.ID, m)
		}
		if (itemType == stock.ItemVehicleUnit) != (m.VehicleUnit != nil) || (itemType == stock.ItemStockItem) != (m.StockItem != nil) {
			t.Errorf("movement %s item reference: %+v", m.ID, m)
		}
		if m.ActorName == nil || *m.ActorName == "" {
			t.Errorf("movement %s has no actor name", m.ID)
		}
	}
	// Newest first by default: the exact reverse.
	desc := tf.list(t, tf.full, "limit=100")
	rev := idsOf(all)
	slices.Reverse(rev)
	if !slices.Equal(idsOf(desc), rev) {
		t.Fatalf("default order is not newest first")
	}
	// Pagination: pages of 3 cover the ledger once, in order, with the same total.
	var paged []string
	for page := 1; ; page++ {
		p := tf.list(t, tf.full, q("order", "asc", "limit", "3", "page", strconv.Itoa(page)))
		if p.Total != all.Total || p.Page != page || p.Limit != 3 {
			t.Fatalf("page %d meta %+v", page, p)
		}
		if len(p.Items) == 0 {
			break
		}
		paged = append(paged, idsOf(p)...)
	}
	if !slices.Equal(paged, idsOf(all)) {
		t.Fatalf("pages %v, want %v", paged, idsOf(all))
	}
	// Empty result: a page past the end and a filter that matches nothing.
	if p := tf.list(t, tf.full, q("page", "99")); len(p.Items) != 0 || p.Total != all.Total {
		t.Fatalf("past the end: %+v", p)
	}
	if p := tf.list(t, tf.full, q("movementType", stock.TypeReject)); len(p.Items) != 0 || p.Total != 0 || p.Items == nil {
		t.Fatalf("no rejects: %+v", p)
	}
	// Detail is the same entry.
	var one stock.TimelineMovement
	expect(t, tf.h.raw(http.MethodGet, "/api/v1/stock-movements/"+all.Items[0].ID, "", &tf.full, nil), http.StatusOK, &one)
	if one.ID != all.Items[0].ID || one.MovementType != all.Items[0].MovementType || !one.CreatedAt.Equal(all.Items[0].CreatedAt) {
		t.Fatalf("detail %+v, list %+v", one, all.Items[0])
	}
	expectError(t, tf.h.raw(http.MethodGet, "/api/v1/stock-movements/"+someID, "", &tf.full, nil), http.StatusNotFound, "NOT_FOUND")
}

// 4–10, 16: each filter and combinations of them select exactly the matching ledger rows.
func TestTimelineFiltersMatchTheLedger(t *testing.T) {
	tf := setupTimeline(t)
	f := tf.fixture
	unitNo := f.scalar(t, `SELECT unit_number FROM vehicle_units WHERE id = $1`, tf.l1Units[1])
	chassis := f.scalar(t, `SELECT chassis_number FROM vehicle_units WHERE id = $1`, tf.l2Unit)
	mark := tf.mark.Format(time.RFC3339Nano)
	tomorrow := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	for _, c := range []struct {
		name, query, where string
		args               []any
	}{
		{"item type", q("itemType", "STOCK_ITEM"), "m.item_type = 'STOCK_ITEM'", nil},
		{"vehicle unit", q("vehicleUnitId", tf.l1Units[0]), "m.vehicle_unit_id = $1", []any{tf.l1Units[0]}},
		{"stock item", q("stockItemId", tf.item), "m.stock_item_id = $1", []any{tf.item}},
		{"unit number", q("q", strings.ToLower(unitNo)), "m.vehicle_unit_id = $1", []any{tf.l1Units[1]}},
		{"chassis", q("q", " "+strings.ToLower(chassis)+" "), "m.vehicle_unit_id = $1", []any{tf.l2Unit}},
		{"sku", q("q", "prt-1"), "m.stock_item_id = $1", []any{tf.item}},
		{"movement type", q("movementType", "PDI_PASS"), "m.movement_type = 'PDI_PASS'", nil},
		{"actor", q("actorId", f.ids["mover"]), "m.actor_id = $1", []any{f.ids["mover"]}},
		{"location", q("locationId", f.locL2), "m.to_location_id = $1", []any{f.locL2}},
		{"date from", q("from", mark), "m.created_at >= $1", []any{tf.mark}},
		{"date to", q("to", mark), "m.created_at < $1", []any{tf.mark}},
		{"date window, nothing", q("from", tomorrow), "false", nil},
		{"source type", q("sourceType", "GOODS_RECEIPT"), "m.source_type = 'GOODS_RECEIPT'", nil},
		{"source reference (any case)", q("sourceReference", strings.ToLower(tf.grnRef)), "m.source_reference = $1", []any{tf.grnRef}},
		{"count reference", q("sourceReference", tf.countRef), "m.source_reference = $1", []any{tf.countRef}},
		// Combinations.
		{"unit + type", q("vehicleUnitId", tf.l1Units[0], "movementType", "DISPLAY_ON"),
			"m.vehicle_unit_id = $1 AND m.movement_type = 'DISPLAY_ON'", []any{tf.l1Units[0]}},
		{"location + date", q("locationId", f.locL1, "from", mark), "m.to_location_id = $1 AND m.created_at >= $2", []any{f.locL1, tf.mark}},
		{"actor + type", q("actorId", f.ids["mover"], "movementType", "PARTS_ISSUE"),
			"m.actor_id = $1 AND m.movement_type = 'PARTS_ISSUE'", []any{f.ids["mover"]}},
		{"source type + reference", q("sourceType", "SUPPLIER_DELIVERY", "sourceReference", "dc-l2"),
			"m.source_type = 'SUPPLIER_DELIVERY' AND m.source_reference = 'DC-L2'", nil},
		{"location + actor + date", q("locationId", f.locL1, "actorId", f.ids["approver"], "from", mark, "to", tomorrow),
			"m.to_location_id = $1 AND m.actor_id = $2 AND m.created_at >= $3", []any{f.locL1, f.ids["approver"], tf.mark}},
		{"unit + source reference", q("vehicleUnitId", tf.l2Unit, "sourceReference", tf.grnRef),
			"m.vehicle_unit_id = $1 AND m.source_reference = $2", []any{tf.l2Unit, tf.grnRef}},
		{"unit + other unit's reference", q("vehicleUnitId", tf.l1Units[0], "sourceReference", "DC-L2"), "false", nil},
		{"item type + movement type of the other item", q("itemType", "VEHICLE_UNIT", "movementType", "PARTS_ISSUE"), "false", nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			p := tf.sameAs(t, tf.full, c.query, c.where, c.args...)
			if c.where != "false" && len(p.Items) == 0 {
				t.Fatalf("%s selected nothing: the case proves nothing", c.name)
			}
		})
	}
}

// 15, 17, 19, 20, 21: rows, sources, suppliers and locations follow the caller's own permissions and scope.
func TestTimelineVisibilityAndShaping(t *testing.T) {
	tf := setupTimeline(t)
	f := tf.fixture

	// Vehicles only: no parts row; receipts' GRN hidden (no goods_receipts.view) and not searchable.
	p := tf.sameAs(t, tf.vehicles, "", "m.item_type = 'VEHICLE_UNIT'")
	for _, m := range p.Items {
		hidden := m.SourceType == stock.SourceGoodsReceipt
		if m.SourceHidden != hidden || (hidden && (m.SourceID != nil || m.SourceReference != nil)) {
			t.Errorf("vehicles-only %s: hidden %v ref %v", m.MovementType, m.SourceHidden, m.SourceReference)
		}
	}
	if p := tf.list(t, tf.vehicles, q("sourceReference", tf.grnRef)); p.Total != 0 {
		t.Fatalf("a hidden GRN reference can be probed: %+v", p)
	}
	if p := tf.list(t, tf.vehicles, q("itemType", "STOCK_ITEM")); p.Total != 0 {
		t.Fatalf("parts rows without stock.parts.view: %d", p.Total)
	}
	partsMove := tf.ids(t, "m.item_type = 'STOCK_ITEM'")[0]
	expectError(t, tf.h.raw(http.MethodGet, "/api/v1/stock-movements/"+partsMove, "", &tf.vehicles, nil), http.StatusNotFound, "NOT_FOUND")

	// Scoped to L2: L2's unit, the unit without a location, and L2's parts — nothing at L1.
	tf.sameAs(t, tf.scoped, "", "(m.item_type = 'VEHICLE_UNIT' AND (v.location_id IS NULL OR v.location_id = $1)) OR "+
		"(m.item_type = 'STOCK_ITEM' AND m.to_location_id = $1)", f.locL2)
	expectError(t, tf.h.raw(http.MethodGet, "/api/v1/stock-movements?"+q("locationId", f.locL1), "", &tf.scoped, nil),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	l1Move := tf.ids(t, "m.vehicle_unit_id = $1", tf.l1Units[0])[0]
	expectError(t, tf.h.raw(http.MethodGet, "/api/v1/stock-movements/"+l1Move, "", &tf.scoped, nil), http.StatusNotFound, "NOT_FOUND")
	// Filtering by an out-of-scope unit's ID reveals nothing.
	if p := tf.list(t, tf.scoped, q("vehicleUnitId", tf.l1Units[0])); p.Total != 0 {
		t.Fatalf("out-of-scope unit: %d", p.Total)
	}

	// Parts only, without suppliers.view or stock.adjustment.view: no vehicle row, no supplier, the count hidden.
	p = tf.sameAs(t, tf.parts, "", "m.item_type = 'STOCK_ITEM'")
	for _, m := range p.Items {
		if m.SupplierID != nil || m.SupplierName != nil {
			t.Errorf("supplier without suppliers.view: %+v", m)
		}
		isCount := m.MovementType == stock.TypeCountAdjustment
		if m.SourceHidden != isCount || (isCount && (m.SourceReference != nil || m.SourceID != nil || m.Reason != nil || m.Note != nil)) {
			t.Errorf("count source shaping: %+v", m)
		}
		if m.MovementType == stock.TypePartsReceipt && (m.SourceReference == nil || m.UOM == nil || *m.UOM != "PCS") {
			t.Errorf("challan reference / unit missing: %+v", m)
		}
	}
	if p := tf.list(t, tf.parts, q("sourceReference", tf.countRef)); p.Total != 0 {
		t.Fatalf("a hidden count reference can be probed")
	}

	// Full access: the supplier and the count are shown.
	p = tf.list(t, tf.full, q("itemType", "STOCK_ITEM", "limit", "100"))
	var sawSupplier, sawCount bool
	for _, m := range p.Items {
		sawSupplier = sawSupplier || (m.SupplierName != nil && *m.SupplierName == "Example Parts Supplier")
		sawCount = sawCount || (m.MovementType == stock.TypeCountAdjustment && m.SourceReference != nil && *m.SourceReference == tf.countRef &&
			m.Reason != nil)
	}
	if !sawSupplier || !sawCount {
		t.Fatalf("full access: supplier %v count %v", sawSupplier, sawCount)
	}

	// No cost, price, document or storage data anywhere in the payload.
	w := tf.h.raw(http.MethodGet, "/api/v1/stock-movements?limit=100", "", &tf.full, nil)
	for _, leak := range []string{"cost", "Cost", "price", "storage", "attachment", "request_hash", "requestHash", "idempotency",
		"username", "password"} {
		if strings.Contains(w.Body.String(), leak) {
			t.Errorf("payload contains %q", leak)
		}
	}
}

// 22: reading changes nothing, and the ledger refuses changes whatever the timeline does.
func TestTimelineIsReadOnly(t *testing.T) {
	tf := setupTimeline(t)
	snapshot := func() string {
		return tf.scalar(t, `SELECT count(*)::text || ':' || coalesce(max(seq), 0)::text || ':' ||
			md5(coalesce(string_agg(id::text || movement_type || coalesce(note, ''), ',' ORDER BY seq), '')) FROM stock_movements`)
	}
	before := snapshot()
	audits := tf.count(t, `SELECT count(*) FROM audit_events`)
	for _, s := range []session{tf.full, tf.vehicles, tf.scoped, tf.parts} {
		tf.list(t, s, "limit=100")
		tf.list(t, s, q("order", "asc", "movementType", "RECEIPT"))
	}
	if after := snapshot(); after != before {
		t.Fatalf("reading the timeline changed the ledger: %s -> %s", before, after)
	}
	if n := tf.count(t, `SELECT count(*) FROM audit_events`); n != audits {
		t.Fatalf("reading wrote %d audit events", n-audits)
	}
	ctx := context.Background()
	for _, sql := range []string{`UPDATE stock_movements SET note = 'edited'`, `DELETE FROM stock_movements`} {
		if _, err := tf.pool.Exec(ctx, sql); err == nil {
			t.Fatalf("%s was allowed", sql)
		}
	}
}
