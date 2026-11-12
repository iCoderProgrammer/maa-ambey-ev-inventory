package vehicles

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
)

// D-08: vehicle list search, filters, ordering and age; detail documents;
// per-unit stock movements. Placeholder data only, throwaway database.

// d08 registers: A1 (all identifiers) and B1 at L1 from the fixture receipt
// (received 2026-01-15), and O1 at L1 from a second receipt of the other
// purchase order received 2026-03-01.
type d08 struct {
	fixture
	a1, b1, o1, otherGRN string
}

func setupD08(t *testing.T) d08 {
	t.Helper()
	f := setup(t)
	ctx := context.Background()
	regs, err := f.registerAll(ctx, f.grn, true, ptr(f.locL1), f.unitA1, f.unitB1)
	if err != nil {
		t.Fatal(err)
	}
	other := f.scalar(t, `INSERT INTO goods_receipts (purchase_order_id, received_date, supplier_reference, created_by)
		VALUES ($1, DATE '2026-03-01', 'DC-2', $2) RETURNING id::text`, f.otherPO, f.ids["poster"])
	line := f.receiptLine(t, other, f.otherPO, f.otherPOLine, 1, 1, 0)
	unit := f.receiptUnit(t, other, line, 1, "ACCEPTED", "CHO-0001", "MOT-0002", "", "")
	o, err := f.registerAll(ctx, other, true, ptr(f.locL1), unit)
	if err != nil {
		t.Fatal(err)
	}
	return d08{fixture: f, a1: regs[0].ID, b1: regs[1].ID, o1: o[0].ID, otherGRN: other}
}

func (f d08) list(t *testing.T, s session, query string) UnitPage {
	t.Helper()
	var p UnitPage
	expect(t, f.h.get(s, "/vehicle-units"+query), http.StatusOK, &p)
	return p
}

func ids(p UnitPage) []string {
	out := make([]string, len(p.Items))
	for i, u := range p.Items {
		out[i] = u.ID
	}
	return out
}

func TestSearchCoversEveryStoredIdentifier(t *testing.T) {
	f := setupD08(t)
	f.h.scope(f.viewer, f.locL1)
	unitNo := f.scalar(t, `SELECT unit_number FROM vehicle_units WHERE id = $1`, f.o1)
	for q, want := range map[string]string{
		"ch-0001": f.a1, " CH-0001 ": f.a1, "mot-0001": f.a1, "BAT-0001": f.a1, "chg-0001": f.a1,
		"MOT-0002": f.o1, "cho-": f.o1, strings.ToLower(unitNo): f.o1,
	} {
		p := f.list(t, f.viewer, "?q="+urlQuery(q))
		if p.Total != 1 || len(p.Items) != 1 || p.Items[0].ID != want {
			t.Errorf("q=%q: %v (total %d)", q, ids(p), p.Total)
		}
	}
	// LIKE wildcards are literal: "%" and "_" match nothing here.
	for _, q := range []string{"%", "_", "NO-SUCH-ID"} {
		p := f.list(t, f.viewer, "?q="+urlQuery(q))
		if p.Total != 0 || p.Items == nil || len(p.Items) != 0 {
			t.Errorf("q=%q: %+v", q, p)
		}
	}
}

func TestFiltersCombineAcrossModelDatesAgeAndStatus(t *testing.T) {
	f := setupD08(t)
	f.h.scope(f.viewer, f.locL1)
	ageA := f.count(t, `SELECT current_date - DATE '2026-01-15'`)
	ageO := f.count(t, `SELECT current_date - DATE '2026-03-01'`)
	for query, want := range map[string][]string{
		"":                     {f.a1, f.b1, f.o1},
		"?modelId=" + f.modelA: {f.a1, f.o1},
		"?brandId=" + f.brandA + "&modelId=" + f.modelB:                                                            {},
		"?brandId=" + f.brandA + "&variantId=" + f.variantA2:                                                       {f.o1},
		"?modelId=" + f.modelA + "&status=RECEIVED_PENDING_PDI":                                                    {f.a1, f.o1},
		"?locationId=" + f.locL1 + "&status=AVAILABLE":                                                             {},
		"?supplierId=" + f.supplier + "&receivedFrom=2026-02-01":                                                   {f.o1},
		"?receivedFrom=2026-01-15&receivedTo=2026-01-15":                                                           {f.a1, f.b1},
		"?receivedTo=2026-01-14":                                                                                   {},
		"?minAgeDays=" + strconv.Itoa(ageA):                                                                        {f.a1, f.b1},
		"?maxAgeDays=" + strconv.Itoa(ageO):                                                                        {f.o1},
		"?minAgeDays=" + strconv.Itoa(ageO) + "&maxAgeDays=" + strconv.Itoa(ageA) + "&status=RECEIVED_PENDING_PDI": {f.a1, f.b1, f.o1},
		"?minAgeDays=" + strconv.Itoa(ageA+1):                                                                      {},
		"?q=ch&brandId=" + f.brandB:                                                                                {f.b1},
		"?q=ch&maxAgeDays=" + strconv.Itoa(ageO):                                                                   {f.o1},
		"?saleable=true":                                                                                           {},
		"?purchaseOrderId=" + f.otherPO + "&goodsReceiptId=" + f.grn:                                               {},
	} {
		sep := "&"
		if query == "" {
			sep = "?"
		}
		p := f.list(t, f.viewer, query+sep+"sort=unitNumber")
		got := map[string]bool{}
		for _, id := range ids(p) {
			got[id] = true
		}
		if p.Total != len(want) || len(got) != len(want) {
			t.Errorf("%q: got %v (total %d), want %v", query, ids(p), p.Total, want)
			continue
		}
		for _, id := range want {
			if !got[id] {
				t.Errorf("%q: missing %s in %v", query, id, ids(p))
			}
		}
	}
	// Each unit carries its age in whole days.
	for _, u := range f.list(t, f.viewer, "").Items {
		want := ageA
		if u.ID == f.o1 {
			want = ageO
		}
		if u.AgeDays != want {
			t.Errorf("%s: ageDays %d, want %d", u.UnitNumber, u.AgeDays, want)
		}
	}
}

func TestOrderingIsStableAcrossPages(t *testing.T) {
	f := setupD08(t)
	f.h.scope(f.viewer, f.locL1)
	for _, sort := range SortOrders {
		all := ids(f.list(t, f.viewer, "?sort="+sort))
		if len(all) != 3 {
			t.Fatalf("%s: %v", sort, all)
		}
		var paged []string
		for page := 1; page <= 3; page++ {
			p := f.list(t, f.viewer, "?limit=1&sort="+sort+"&page="+strconv.Itoa(page))
			if p.Total != 3 || p.Page != page || p.Limit != 1 || len(p.Items) != 1 {
				t.Fatalf("%s page %d: %+v", sort, page, p)
			}
			paged = append(paged, p.Items[0].ID)
		}
		if strings.Join(paged, ",") != strings.Join(all, ",") {
			t.Errorf("%s: pages %v, list %v", sort, paged, all)
		}
		if p := f.list(t, f.viewer, "?limit=1&page=4&sort="+sort); len(p.Items) != 0 || p.Total != 3 {
			t.Errorf("%s: past the end %+v", sort, p)
		}
	}
	if first := ids(f.list(t, f.viewer, "?sort=-receivedDate"))[0]; first != f.o1 {
		t.Errorf("-receivedDate starts with %s, want the March unit", first)
	}
	if first := ids(f.list(t, f.viewer, "?sort=receivedDate"))[2]; first != f.o1 {
		t.Errorf("receivedDate ends with %s, want the March unit", first)
	}
	numbers := []string{}
	for _, u := range f.list(t, f.viewer, "?sort=unitNumber").Items {
		numbers = append(numbers, u.UnitNumber)
	}
	if !(numbers[0] < numbers[1] && numbers[1] < numbers[2]) {
		t.Errorf("unitNumber order %v", numbers)
	}
}

// attach stores attachment metadata (no file) owned by a record at a location.
func (f d08) attach(t *testing.T, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, name, ownerType, ownerID, location string) string {
	t.Helper()
	var id string
	if err := q.QueryRow(context.Background(), `INSERT INTO attachments (object_key, storage_driver, content_type, size_bytes,
			sha256, original_name, uploaded_by, location_id, owner_type, owner_id)
		VALUES (to_char(now(), 'YYYY/MM/') || gen_random_uuid()::text, 'local', 'application/pdf', 1234,
			sha256(gen_random_uuid()::text::bytea), $1, $2, $3, $4, $5) RETURNING id::text`,
		name, f.ids["poster"], location, ownerType, ownerID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// holdOnPDI records a PDI HOLD with one evidence file for a unit at L1, as the
// PDI route (D-06) would: attachment, movement, inspection and link in one transaction.
func (f d08) holdOnPDI(t *testing.T, unit, number string) (inspection, evidence string) {
	t.Helper()
	ctx := context.Background()
	inspection = uuid.NewString()
	hash := strings.Repeat("ab", 32)
	err := pgx.BeginFunc(ctx, f.pool, func(tx pgx.Tx) error {
		evidence = f.attach(t, tx, "scratch.pdf", "pdi_inspection", inspection, f.locL1)
		var movement string
		if err := tx.QueryRow(ctx, `INSERT INTO stock_movements (vehicle_unit_id, movement_type, quantity_before, quantity_after,
				from_status, to_status, from_location_id, to_location_id, source_type, source_id, source_reference, actor_id,
				idempotency_key, request_hash, note)
			VALUES ($1, 'QUALITY_HOLD', 1, 1, 'RECEIVED_PENDING_PDI', 'QUALITY_HOLD', $2, $2, 'PDI_INSPECTION', $3::uuid, $4, $5,
				'pdi-test-' || left(md5($3::text), 12), $6, 'Scratched panel') RETURNING id::text`,
			unit, f.locL1, inspection, number, f.ids["poster"], hash).Scan(&movement); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO pdi_inspections (id, inspection_number, vehicle_unit_id, disposition, result,
				from_status, to_status, location_id, notes, movement_id, inspected_by, idempotency_key, request_hash)
			VALUES ($1::uuid, $2, $3, 'HOLD', 'HAS_FAILURES', 'RECEIVED_PENDING_PDI', 'QUALITY_HOLD', $4, 'Scratched panel', $5, $6,
				'pdi-test-' || left(md5($1::text), 12), $7)`,
			inspection, number, unit, f.locL1, movement, f.ids["poster"], hash); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO pdi_inspection_attachments (inspection_id, attachment_id) VALUES ($1, $2)`,
			inspection, evidence)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return inspection, evidence
}

func TestDetailListsOnlyDocumentsTheCallerMayOpen(t *testing.T) {
	f := setupD08(t)
	ctx := context.Background()
	challan := f.attach(t, f.pool, "challan.pdf", "goods_receipt", f.grn, f.locL1)
	elsewhere := f.attach(t, f.pool, "other-site.pdf", "goods_receipt", f.grn, f.locL2)
	for i, a := range []string{challan, elsewhere} {
		if _, err := f.pool.Exec(ctx, `INSERT INTO goods_receipt_documents (goods_receipt_id, attachment_id, document_type, added_by)
			VALUES ($1, $2, $3, $4)`, f.grn, a, []string{"DELIVERY_CHALLAN", "PHOTO"}[i], f.ids["poster"]); err != nil {
			t.Fatal(err)
		}
	}
	_, evidence := f.holdOnPDI(t, f.a1, "PDI-000001")
	objectKey := f.scalar(t, `SELECT object_key FROM attachments WHERE id = $1`, challan)

	docs := func(perms ...string) Detail {
		t.Helper()
		f.h.grant(f.scoped, append([]string{access.PermVehicleUnitView}, perms...)...)
		var d Detail
		w := f.h.get(f.scoped, "/vehicle-units/"+f.a1)
		expect(t, w, http.StatusOK, &d)
		if strings.Contains(w.Body.String(), objectKey) || strings.Contains(w.Body.String(), "objectKey") ||
			strings.Contains(w.Body.String(), "storage") {
			t.Fatalf("storage details leaked: %s", w.Body)
		}
		return d
	}
	byID := func(d Detail) map[string]Document {
		out := map[string]Document{}
		for _, doc := range d.Documents {
			out[doc.AttachmentID] = doc
		}
		return out
	}
	// Receipt documents and PDI evidence, at the caller's location only.
	d := docs(access.PermAttachmentsView, access.PermGoodsReceiptView)
	got := byID(d)
	if len(got) != 2 || d.DocumentsHidden {
		t.Fatalf("documents %+v hidden=%v", d.Documents, d.DocumentsHidden)
	}
	if c := got[challan]; c.SourceType != DocumentSourceGoodsReceipt || c.SourceID != f.grn || c.DocumentType != "DELIVERY_CHALLAN" ||
		c.FileName != "challan.pdf" || c.SizeBytes != 1234 || c.ContentType != "application/pdf" || !strings.HasPrefix(c.SourceReference, "GRN-") {
		t.Errorf("challan %+v", c)
	}
	if e := got[evidence]; e.SourceType != DocumentSourcePDIInspection || e.SourceReference != "PDI-000001" ||
		e.DocumentType != DocumentTypePDIEvidence {
		t.Errorf("evidence %+v", e)
	}
	// Without goods_receipts.view: PDI evidence only.
	if got := byID(docs(access.PermAttachmentsView)); len(got) != 1 || got[evidence].AttachmentID == "" {
		t.Errorf("without goods receipt view: %v", got)
	}
	// Without attachments.view: nothing, and the response says documents are hidden.
	if d := docs(access.PermGoodsReceiptView); len(d.Documents) != 0 || !d.DocumentsHidden || d.Documents == nil {
		t.Errorf("without attachments.view: %+v hidden=%v", d.Documents, d.DocumentsHidden)
	}
	// Every location: the L2 document too.
	f.h.grant(f.allLocs, access.PermVehicleUnitView, access.PermAllLocations, access.PermAttachmentsView, access.PermGoodsReceiptView)
	var all Detail
	expect(t, f.h.get(f.allLocs, "/vehicle-units/"+f.a1), http.StatusOK, &all)
	if len(all.Documents) != 3 {
		t.Errorf("all locations: %+v", all.Documents)
	}
	// A unit with no documents has an empty list.
	var b Detail
	expect(t, f.h.get(f.allLocs, "/vehicle-units/"+f.o1), http.StatusOK, &b)
	if b.Documents == nil || len(b.Documents) != 0 {
		t.Errorf("o1 documents %+v", b.Documents)
	}
}

func TestMovementsShowTheLedgerInOrderWithSources(t *testing.T) {
	f := setupD08(t)
	inspection, _ := f.holdOnPDI(t, f.a1, "PDI-000001")
	f.h.grant(f.scoped, access.PermVehicleUnitView)
	path := "/vehicle-units/" + f.a1 + "/stock-movements"
	var p MovementPage
	expect(t, f.h.get(f.scoped, path), http.StatusOK, &p)
	if p.Total != 2 || len(p.Items) != 2 || p.Limit != 50 {
		t.Fatalf("%+v", p)
	}
	r, h := p.Items[0], p.Items[1]
	grnNumber := f.scalar(t, `SELECT grn_number FROM goods_receipts WHERE id = $1`, f.grn)
	if r.MovementType != "RECEIPT" || r.FromStatus != nil || r.ToStatus != StatusReceivedPendingPDI || r.SourceType != "GOODS_RECEIPT" ||
		r.SourceID == nil || *r.SourceID != f.grn || r.SourceReference == nil || *r.SourceReference != grnNumber ||
		r.ToLocationCode == nil || *r.ToLocationCode != "L1" || r.FromLocationID != nil || r.ActorID != f.ids["poster"] {
		t.Errorf("receipt %+v", r)
	}
	if h.MovementType != "QUALITY_HOLD" || h.FromStatus == nil || *h.FromStatus != StatusReceivedPendingPDI ||
		h.ToStatus != StatusQualityHold || h.SourceType != "PDI_INSPECTION" || h.SourceID == nil || *h.SourceID != inspection ||
		*h.SourceReference != "PDI-000001" || h.Note == nil || *h.Note != "Scratched panel" || h.CreatedAt.Before(r.CreatedAt) {
		t.Errorf("hold %+v", h)
	}
	if h.ActorName == nil || *h.ActorName != "poster" {
		t.Errorf("actor name %v", h.ActorName)
	}
	// Newest first, and paging.
	expect(t, f.h.get(f.scoped, path+"?order=desc"), http.StatusOK, &p)
	if p.Items[0].ID != h.ID || p.Items[1].ID != r.ID {
		t.Errorf("desc %+v", p.Items)
	}
	expect(t, f.h.get(f.scoped, path+"?limit=1&page=2"), http.StatusOK, &p)
	if p.Total != 2 || len(p.Items) != 1 || p.Items[0].ID != h.ID {
		t.Errorf("page 2 %+v", p)
	}
	// The detail now reports QUALITY_HOLD, never saleable; the list agrees.
	var d Detail
	expect(t, f.h.get(f.scoped, "/vehicle-units/"+f.a1), http.StatusOK, &d)
	if d.Status != StatusQualityHold || len(d.Events) != 1 {
		t.Errorf("detail status %s events %d", d.Status, len(d.Events))
	}
	if got := ids(f.list(t, f.scoped, "?saleable=true")); len(got) != 0 {
		t.Errorf("saleable %v", got)
	}
	// No cost in the ledger answer, even for a caller who may see costs.
	f.h.grant(f.coster, access.PermVehicleUnitView, access.PermVehicleUnitCostView, access.PermAllLocations)
	if w := f.h.get(f.coster, path); w.Code != http.StatusOK || strings.Contains(w.Body.String(), "95000") ||
		strings.Contains(w.Body.String(), "unitCost") {
		t.Errorf("cost in movements: %s", w.Body)
	}
}

func TestMovementsFollowViewPermissionAndLocationScope(t *testing.T) {
	f := setupD08(t)
	ctx := context.Background()
	// A unit at L2, outside the scoped user's L1.
	grn := f.scalar(t, `INSERT INTO goods_receipts (purchase_order_id, received_date, supplier_reference, created_by)
		VALUES ($1, DATE '2026-03-02', 'DC-3', $2) RETURNING id::text`, f.po, f.ids["poster"])
	line := f.receiptLine(t, grn, f.po, f.poLineA, 1, 1, 0)
	u := f.receiptUnit(t, grn, line, 1, "ACCEPTED", "CHL2-0001", "", "", "")
	l2, err := f.registerAll(ctx, grn, true, ptr(f.locL2), u)
	if err != nil {
		t.Fatal(err)
	}
	f.h.grant(f.scoped, access.PermVehicleUnitView)
	expectError(t, f.h.get(f.scoped, "/vehicle-units/"+l2[0].ID+"/stock-movements"), http.StatusNotFound, "NOT_FOUND")
	expectError(t, f.h.get(f.scoped, "/vehicle-units/"+l2[0].ID), http.StatusNotFound, "NOT_FOUND")
	expect(t, f.h.get(f.scoped, "/vehicle-units/"+f.a1+"/stock-movements"), http.StatusOK, nil)
	expectError(t, f.h.get(f.scoped, "/vehicle-units/"+uuid.NewString()+"/stock-movements"), http.StatusNotFound, "NOT_FOUND")
	// Not by guessing IDs, and not with neighbouring permissions.
	f.h.grant(f.nobody, access.PermVehicleUnitCostView, access.PermAttachmentsView, access.PermGoodsReceiptView, access.PermAllLocations)
	for _, p := range []string{"/vehicle-units", "/vehicle-units/" + f.a1, "/vehicle-units/" + f.a1 + "/stock-movements"} {
		expectError(t, f.h.get(f.nobody, p), http.StatusForbidden, "FORBIDDEN")
	}
	// The all-locations user sees L2's movement with its location named.
	var p MovementPage
	expect(t, f.h.get(f.allLocs, "/vehicle-units/"+l2[0].ID+"/stock-movements"), http.StatusOK, &p)
	if len(p.Items) != 1 || p.Items[0].ToLocationCode == nil || *p.Items[0].ToLocationCode != "L2" {
		t.Errorf("%+v", p.Items)
	}
}
