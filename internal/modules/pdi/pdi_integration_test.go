package pdi_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database/dbtest"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/httpx"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/attachments"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/pdi"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/stock"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/vehicles"
)

// PASS: actor, database time, every applicable outcome; the unit becomes
// AVAILABLE through one PDI_PASS movement naming the inspection; audited.
func TestPDIPassMovesThePostedUnitToAvailable(t *testing.T) {
	f := setup(t)
	brakes := f.checklistItem("BRAKES", nil, 10)
	lights := f.checklistItem("LIGHTS", nil, 20)
	f.checklistItem("BRAND_B_ONLY", &f.brandB, 5) // does not apply to Brand A units
	unit := f.units(&f.locL1, f.poLineA, 1, false)[0]

	var ov pdi.Overview
	expect(t, f.get(f.inspector, "/vehicle-units/"+unit+"/pdi"), http.StatusOK, &ov)
	if len(ov.Checklist) != 2 || ov.Checklist[0].ID != brakes.ID || ov.Checklist[1].ID != lights.ID ||
		!ov.CanRecord || len(ov.AllowedDispositions) != 2 || ov.Unit.Saleable {
		t.Fatalf("overview %+v", ov)
	}
	photo := f.upload("inspector", &f.locL1) // optional evidence on a pass
	var res pdi.RecordResult
	expect(t, f.record(f.inspector, unit, map[string]any{"version": ov.Unit.Version, "disposition": "PASS",
		"items": outcomes(ov.Checklist), "attachmentIds": []string{photo}}, "pass-00000001"), http.StatusCreated, &res)
	in := res.Inspection
	if in.Disposition != "PASS" || in.Result != "ALL_PASSED" || in.FromStatus != "RECEIVED_PENDING_PDI" || in.ToStatus != "AVAILABLE" ||
		in.InspectedBy != f.ids["inspector"] || in.InspectedAt.IsZero() || len(in.Items) != 2 || len(in.Evidence) != 1 ||
		in.InspectionNumber == "" || res.Unit.Status != "AVAILABLE" || !res.Unit.Saleable || res.Unit.Version != ov.Unit.Version+1 {
		t.Fatalf("result %+v", res)
	}
	// One movement, sourced from the inspection; unit state equals the latest movement.
	var typ, src, srcID, srcRef, to string
	if err := f.pool.QueryRow(context.Background(), `SELECT movement_type, source_type, source_id::text, source_reference, to_status
		FROM stock_movements WHERE id = $1`, in.MovementID).Scan(&typ, &src, &srcID, &srcRef, &to); err != nil {
		t.Fatal(err)
	}
	if typ != stock.TypePDIPass || src != stock.SourcePDIInspection || srcID != in.ID || srcRef != in.InspectionNumber || to != "AVAILABLE" {
		t.Fatalf("movement %s %s %s %s %s", typ, src, srcID, srcRef, to)
	}
	if n := f.count(`SELECT count(*) FROM stock_movements WHERE vehicle_unit_id = $1 AND movement_type <> 'RECEIPT'`, unit); n != 1 {
		t.Fatalf("%d PDI movements", n)
	}
	if f.audits("pdi_inspection.recorded", "SUCCESS", in.ID) != 1 || f.audits("vehicle_unit.moved", "SUCCESS", unit) != 1 {
		t.Fatal("inspection or movement not audited")
	}
	if owner := f.scalar(`SELECT owner_type || ':' || owner_id FROM attachments WHERE id = $1`, photo); owner != "pdi_inspection:"+in.ID {
		t.Fatalf("evidence owner %s", owner)
	}
	// The unit's goods receipt is posted (a unit exists only after posting): no premature AVAILABLE.
	if st := f.scalar(`SELECT g.status FROM vehicle_units v JOIN goods_receipts g ON g.id = v.goods_receipt_id WHERE v.id = $1`, unit); st != "POSTED" {
		t.Fatalf("receipt %s", st)
	}
	// History and detail.
	expect(t, f.get(f.viewer, "/vehicle-units/"+unit+"/pdi"), http.StatusOK, &ov)
	if len(ov.Inspections) != 1 || ov.Inspections[0].ItemCount != 2 || ov.CanRecord || len(ov.AllowedDispositions) != 0 {
		t.Fatalf("history %+v", ov)
	}
	var d pdi.InspectionDetail
	expect(t, f.get(f.viewer, "/pdi-inspections/"+in.ID), http.StatusOK, &d)
	if d.Items[0].Code != "BRAKES" || !d.EvidenceHidden || len(d.Evidence) != 0 || d.EvidenceCount != 1 {
		t.Fatalf("viewer without attachments.view sees evidence details: %+v", d)
	}
	// Passed units cannot be inspected again (no edge from AVAILABLE).
	expectError(t, f.record(f.inspector, unit, map[string]any{"version": res.Unit.Version, "disposition": "HOLD", "notes": "late",
		"items": outcomes(ov.Checklist), "attachmentIds": []string{f.upload("inspector", &f.locL1)}}, "hold-after-pass"),
		http.StatusConflict, httpx.CodeInvalidState)
}

// HOLD then REJECT: notes and evidence, non-saleable, excluded from saleable search.
func TestPDIHoldThenRejectKeepsTheUnitOutOfSaleableStock(t *testing.T) {
	f := setup(t)
	f.checklistItem("BRAKES", nil, 1)
	f.checklistItem("PAINT", &f.brandA, 2)
	units := f.units(&f.locL1, f.poLineA, 3, false)
	held, passed, pending := units[0], units[1], units[2]
	var ov pdi.Overview
	expect(t, f.get(f.inspector, "/vehicle-units/"+held+"/pdi"), http.StatusOK, &ov)

	ev := f.upload("inspector", &f.locL1)
	var res pdi.RecordResult
	expect(t, f.record(f.inspector, held, map[string]any{"version": ov.Unit.Version, "disposition": "HOLD",
		"items": outcomes(ov.Checklist, "PAINT"), "notes": "Scratch on side panel", "attachmentIds": []string{ev}}, "hold-00000001"),
		http.StatusCreated, &res)
	if res.Unit.Status != "QUALITY_HOLD" || res.Unit.Saleable || res.Inspection.Result != "HAS_FAILURES" ||
		res.Inspection.FailedCount != 1 || *res.Inspection.Notes != "Scratch on side panel" {
		t.Fatalf("hold %+v", res)
	}
	// Re-inspection of a held unit: only REJECT (OQ-31: no release, no second hold).
	for _, disp := range []string{"PASS", "HOLD"} {
		body := map[string]any{"version": res.Unit.Version, "disposition": disp, "items": outcomes(ov.Checklist)}
		if disp == "HOLD" {
			body["notes"], body["attachmentIds"] = "again", []string{f.upload("inspector", &f.locL1)}
		}
		expectError(t, f.record(f.inspector, held, body, "rehold-"+disp+"-01"), http.StatusConflict, httpx.CodeInvalidState)
	}
	ev2 := f.upload("other", &f.locL1)
	var rej pdi.RecordResult
	expect(t, f.record(f.other, held, map[string]any{"version": res.Unit.Version, "disposition": "REJECT",
		"items": outcomes(ov.Checklist, "PAINT"), "notes": "Panel cannot be repaired", "attachmentIds": []string{ev2}}, "reject-0000001"),
		http.StatusCreated, &rej)
	if rej.Unit.Status != "REJECTED" || rej.Inspection.FromStatus != "QUALITY_HOLD" || rej.Inspection.InspectedBy != f.ids["other"] {
		t.Fatalf("reject %+v", rej)
	}
	expect(t, f.get(f.inspector, "/vehicle-units/"+held+"/pdi"), http.StatusOK, &ov)
	if len(ov.Inspections) != 2 || ov.Inspections[0].Disposition != "REJECT" || ov.Inspections[1].Disposition != "HOLD" ||
		len(ov.AllowedDispositions) != 0 {
		t.Fatalf("history %+v", ov.Inspections)
	}
	// One unit passed: only it is saleable.
	expect(t, f.get(f.inspector, "/vehicle-units/"+passed+"/pdi"), http.StatusOK, &ov)
	expect(t, f.record(f.inspector, passed, map[string]any{"version": ov.Unit.Version, "disposition": "PASS",
		"items": outcomes(ov.Checklist)}, "pass-00000002"), http.StatusCreated, nil)
	var page vehicles.UnitPage
	expect(t, f.get(f.viewer, "/vehicle-units?saleable=true"), http.StatusOK, &page)
	if page.Total != 1 || page.Items[0].ID != passed || page.Items[0].Status != "AVAILABLE" {
		t.Fatalf("saleable search %+v", page)
	}
	expect(t, f.get(f.viewer, "/vehicle-units?saleable=false"), http.StatusOK, &page)
	got := map[string]bool{}
	for _, u := range page.Items {
		got[u.ID] = !vehicles.IsSaleable(u.Status)
	}
	if page.Total != 2 || !got[held] || !got[pending] {
		t.Fatalf("non-saleable search %+v", page)
	}
	expectError(t, f.get(f.viewer, "/vehicle-units?saleable=maybe"), http.StatusUnprocessableEntity, httpx.CodeValidationFailed)
	// Ledger and unit state agree for every unit.
	if n := f.count(`SELECT count(*) FROM vehicle_units v WHERE v.status <> (SELECT m.to_status FROM stock_movements m
			WHERE m.vehicle_unit_id = v.id ORDER BY m.seq DESC LIMIT 1)`); n != 0 {
		t.Fatalf("%d units disagree with their latest movement", n)
	}
}

// Validation: missing outcomes, unknown/retired items, failed item without
// note, hold without notes/evidence, invalid disposition, server-owned fields.
func TestPDIValidation(t *testing.T) {
	f := setup(t)
	brakes := f.checklistItem("BRAKES", nil, 1)
	lights := f.checklistItem("LIGHTS", nil, 2)
	otherBrand := f.checklistItem("B_ONLY", &f.brandB, 3)
	unit := f.units(&f.locL1, f.poLineA, 1, false)[0]
	_, v := f.unit(unit)
	cases := []struct {
		name  string
		body  map[string]any
		field string
	}{
		{"missing outcome", map[string]any{"version": v, "disposition": "PASS",
			"items": []map[string]any{{"checklistItemId": brakes.ID, "outcome": "PASS"}}}, "items"},
		{"other brand item", map[string]any{"version": v, "disposition": "PASS", "items": append(outcomes([]pdi.ChecklistItem{brakes, lights}),
			map[string]any{"checklistItemId": otherBrand.ID, "outcome": "PASS"})}, "items[2].checklistItemId"},
		{"failed item without note", map[string]any{"version": v, "disposition": "HOLD", "notes": "x",
			"attachmentIds": []string{f.upload("inspector", &f.locL1)},
			"items":         []map[string]any{{"checklistItemId": brakes.ID, "outcome": "FAIL"}, {"checklistItemId": lights.ID, "outcome": "PASS"}}},
			"items[0].note"},
		{"hold without notes", map[string]any{"version": v, "disposition": "HOLD", "attachmentIds": []string{f.upload("inspector", &f.locL1)},
			"items": outcomes([]pdi.ChecklistItem{brakes, lights})}, "notes"},
		{"hold without evidence", map[string]any{"version": v, "disposition": "HOLD", "notes": "Dent",
			"items": outcomes([]pdi.ChecklistItem{brakes, lights})}, "attachmentIds"},
		{"pass with a failure", map[string]any{"version": v, "disposition": "PASS",
			"items": outcomes([]pdi.ChecklistItem{brakes, lights}, "LIGHTS")}, "disposition"},
		{"invalid disposition", map[string]any{"version": v, "disposition": "SCRAP"}, "disposition"},
		{"no version", map[string]any{"disposition": "PASS", "items": outcomes([]pdi.ChecklistItem{brakes, lights})}, "version"},
		{"status is server-owned", map[string]any{"version": v, "disposition": "PASS", "status": "AVAILABLE"}, "status"},
		{"actor is server-owned", map[string]any{"version": v, "disposition": "PASS", "inspectedBy": f.ids["other"]}, "inspectedBy"},
	}
	for i, tc := range cases {
		e := expectError(t, f.record(f.inspector, unit, tc.body, "val-"+strconv.Itoa(i)+"-0000"), http.StatusUnprocessableEntity,
			httpx.CodeValidationFailed)
		if !hasField(e, tc.field) {
			t.Fatalf("%s: details %+v, want %s", tc.name, e.Details, tc.field)
		}
	}
	expectError(t, f.record(f.inspector, unit, map[string]any{"version": v, "disposition": "PASS"}, "bad key"),
		http.StatusUnprocessableEntity, httpx.CodeValidationFailed)
	// A retired item no longer applies.
	expect(t, f.post(f.manager, "/pdi-checklist-items/"+lights.ID+"/retire", map[string]any{"version": lights.Version,
		"reason": "Merged into BRAKES"}, ""), http.StatusOK, nil)
	e := expectError(t, f.record(f.inspector, unit, map[string]any{"version": v, "disposition": "PASS",
		"items": outcomes([]pdi.ChecklistItem{brakes, lights})}, "val-retired-01"), http.StatusUnprocessableEntity, httpx.CodeValidationFailed)
	if !hasField(e, "items[1].checklistItemId") {
		t.Fatalf("retired item accepted: %+v", e.Details)
	}
	if st, v2 := f.unit(unit); st != "RECEIVED_PENDING_PDI" || v2 != v || f.count(`SELECT count(*) FROM pdi_inspections`) != 0 {
		t.Fatal("a refused request changed something")
	}
}

// No checklist configured: PASS refused (nothing unconfigured counts as passed),
// HOLD allowed; refusals audited DENIED; direct REJECT of a pending unit refused (OQ-31).
func TestPDIChecklistNotConfiguredAndDeniedAudit(t *testing.T) {
	f := setup(t)
	units := f.units(&f.locL1, f.poLineA, 2, false)
	_, v := f.unit(units[0])
	expectError(t, f.record(f.inspector, units[0], map[string]any{"version": v, "disposition": "PASS"}, "pass-none-0001"),
		http.StatusConflict, httpx.CodePDIChecklistNotConfigured)
	e := expectError(t, f.record(f.inspector, units[0], map[string]any{"version": v, "disposition": "REJECT", "notes": "Broken",
		"attachmentIds": []string{f.upload("inspector", &f.locL1)}}, "reject-direct-1"), http.StatusConflict, httpx.CodeInvalidState)
	if e.Message == "" {
		t.Fatal("no explanation")
	}
	if f.audits("pdi_inspection.recorded", "DENIED", units[0]) != 2 {
		t.Fatal("refusals not audited")
	}
	// A brand-B-only item does not configure a Brand A unit.
	f.checklistItem("B_ONLY", &f.brandB, 1)
	expectError(t, f.record(f.inspector, units[0], map[string]any{"version": v, "disposition": "PASS"}, "pass-none-0002"),
		http.StatusConflict, httpx.CodePDIChecklistNotConfigured)
	var res pdi.RecordResult
	expect(t, f.record(f.inspector, units[1], map[string]any{"version": v, "disposition": "HOLD", "notes": "Battery not charging",
		"attachmentIds": []string{f.upload("inspector", &f.locL1)}}, "hold-none-0001"), http.StatusCreated, &res)
	if res.Inspection.Result != "NOT_CHECKED" || res.Unit.Status != "QUALITY_HOLD" {
		t.Fatalf("hold without checklist %+v", res)
	}
	// INV-002: a unit without a chassis/frame number cannot leave RECEIVED_PENDING_PDI.
	f.checklistItem("BRAKES", nil, 1)
	bare := f.units(&f.locL1, f.poLineA, 1, true)[0]
	_, bv := f.unit(bare)
	var ov pdi.Overview
	expect(t, f.get(f.inspector, "/vehicle-units/"+bare+"/pdi"), http.StatusOK, &ov)
	expectError(t, f.record(f.inspector, bare, map[string]any{"version": bv, "disposition": "PASS", "items": outcomes(ov.Checklist)},
		"pass-bare-0001"), http.StatusConflict, httpx.CodeInvalidState)
	if st, _ := f.unit(bare); st != "RECEIVED_PENDING_PDI" {
		t.Fatal("unit without chassis moved")
	}
}

// Permissions (deny by default) and location scope (outside = not found).
func TestPDIPermissionsAndLocationScope(t *testing.T) {
	f := setup(t)
	f.checklistItem("BRAKES", nil, 1)
	atL1 := f.units(&f.locL1, f.poLineA, 1, false)[0]
	atL2 := f.units(&f.locL2, f.poLineA, 1, false)[0]
	var ov pdi.Overview
	expect(t, f.get(f.inspector, "/vehicle-units/"+atL1+"/pdi"), http.StatusOK, &ov)
	body := map[string]any{"version": ov.Unit.Version, "disposition": "PASS", "items": outcomes(ov.Checklist)}
	expectError(t, f.record(f.viewer, atL1, body, "perm-00000001"), http.StatusForbidden, httpx.CodeForbidden)
	expectError(t, f.record(f.nobody, atL1, body, "perm-00000002"), http.StatusForbidden, httpx.CodeForbidden)
	expectError(t, f.get(f.nobody, "/vehicle-units/"+atL1+"/pdi"), http.StatusForbidden, httpx.CodeForbidden)
	expectError(t, f.get(f.nobody, "/pdi-checklist-items"), http.StatusForbidden, httpx.CodeForbidden)
	expectError(t, f.post(f.inspector, "/pdi-checklist-items", map[string]any{"code": "X1", "label": "X"}, ""),
		http.StatusForbidden, httpx.CodeForbidden)
	// The scoped inspector (L2 only) cannot see or inspect the L1 unit.
	expectError(t, f.get(f.scoped, "/vehicle-units/"+atL1+"/pdi"), http.StatusNotFound, httpx.CodeNotFound)
	expectError(t, f.record(f.scoped, atL1, body, "scope-0000001"), http.StatusNotFound, httpx.CodeNotFound)
	var res pdi.RecordResult
	expect(t, f.record(f.inspector, atL1, body, "scope-0000002"), http.StatusCreated, &res)
	expectError(t, f.get(f.scoped, "/pdi-inspections/"+res.Inspection.ID), http.StatusNotFound, httpx.CodeNotFound)
	// ... but can inspect its own location's unit.
	expect(t, f.get(f.scoped, "/vehicle-units/"+atL2+"/pdi"), http.StatusOK, &ov)
	expect(t, f.record(f.scoped, atL2, map[string]any{"version": ov.Unit.Version, "disposition": "PASS",
		"items": outcomes(ov.Checklist)}, "scope-0000003"), http.StatusCreated, nil)
	expectError(t, f.get(f.inspector, "/vehicle-units/00000000-0000-4000-8000-000000000000/pdi"), http.StatusNotFound, httpx.CodeNotFound)
	expectError(t, f.get(f.inspector, "/pdi-inspections/not-a-uuid"), http.StatusNotFound, httpx.CodeNotFound)
}

// Evidence: own unlinked uploads at the unit's location only, linked once;
// readable through the attachments service by unit viewers with attachments.view in scope.
func TestPDIEvidenceStaysPrivate(t *testing.T) {
	f := setup(t)
	f.checklistItem("BRAKES", nil, 1)
	units := f.units(&f.locL1, f.poLineA, 2, false)
	_, v := f.unit(units[0])
	var ov pdi.Overview
	expect(t, f.get(f.inspector, "/vehicle-units/"+units[0]+"/pdi"), http.StatusOK, &ov)
	hold := func(unit, key string, atts ...string) *httptest.ResponseRecorder {
		return f.record(f.inspector, unit, map[string]any{"version": v, "disposition": "HOLD", "notes": "Dent",
			"items": outcomes(ov.Checklist, "BRAKES"), "attachmentIds": atts}, key)
	}
	someoneElses := f.upload("other", &f.locL1)
	wrongLocation := f.upload("inspector", &f.locL2)
	noLocation := f.upload("inspector", nil)
	e := expectError(t, hold(units[0], "ev-00000001", someoneElses, wrongLocation, noLocation, "55555555-5555-4555-8555-555555555555"),
		http.StatusUnprocessableEntity, httpx.CodeValidationFailed)
	for _, fld := range []string{"attachmentIds[0]", "attachmentIds[1]", "attachmentIds[2]", "attachmentIds[3]"} {
		if !hasField(e, fld) {
			t.Fatalf("details %+v, want %s", e.Details, fld)
		}
	}
	good := f.upload("inspector", &f.locL1)
	var res pdi.RecordResult
	expect(t, hold(units[0], "ev-00000002", good), http.StatusCreated, &res)
	// The same file cannot become evidence of another vehicle.
	e = expectError(t, hold(units[1], "ev-00000003", good), http.StatusUnprocessableEntity, httpx.CodeValidationFailed)
	if !hasField(e, "attachmentIds[0]") {
		t.Fatalf("cross-vehicle reuse accepted: %+v", e.Details)
	}
	svc := attachments.NewService(attachments.NewPGStore(f.pool), nil, nil, 0, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx := context.Background()
	if _, err := svc.Get(ctx, f.grants.access(f.inspector.userID), good); err != nil {
		t.Fatalf("unit viewer with attachments.view cannot read evidence: %v", err)
	}
	for _, who := range []session{f.viewer, f.scoped, f.manager} {
		if _, err := svc.Get(ctx, f.grants.access(who.userID), good); !errors.Is(err, attachments.ErrNotFound) {
			t.Fatalf("evidence visible to %s: %v", who.userID, err)
		}
	}
	// The uploader keeps no special access once linked: without vehicle_units.view it is hidden.
	f.grant(f.inspector, access.PermAttachmentsView, access.PermAllLocations)
	if _, err := svc.Get(ctx, f.grants.access(f.inspector.userID), good); !errors.Is(err, attachments.ErrNotFound) {
		t.Fatal("uploader sees linked evidence without vehicle_units.view")
	}
}

// Concurrency: parallel inspections of one unit record exactly one; stale
// versions conflict; repeated keys replay; a reused key for another request is refused.
func TestPDIConcurrencyVersionsAndIdempotency(t *testing.T) {
	f := setup(t)
	f.checklistItem("BRAKES", nil, 1)
	units := f.units(&f.locL1, f.poLineA, 2, false)
	var ov pdi.Overview
	expect(t, f.get(f.inspector, "/vehicle-units/"+units[0]+"/pdi"), http.StatusOK, &ov)
	body := map[string]any{"version": ov.Unit.Version, "disposition": "PASS", "items": outcomes(ov.Checklist)}

	// 8 different keys at once: one wins, the rest see a changed unit.
	var wg sync.WaitGroup
	codes := make([]int, 8)
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s := f.inspector
			if i%2 == 1 {
				s = f.other
			}
			codes[i] = f.record(s, units[0], body, "race-key-"+strconv.Itoa(i)).Code
		}(i)
	}
	wg.Wait()
	created, conflicts := 0, 0
	for _, c := range codes {
		switch c {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			conflicts++
		}
	}
	if created != 1 || conflicts != 7 {
		t.Fatalf("codes %v", codes)
	}
	if f.count(`SELECT count(*) FROM pdi_inspections WHERE vehicle_unit_id = $1`, units[0]) != 1 ||
		f.count(`SELECT count(*) FROM stock_movements WHERE vehicle_unit_id = $1 AND movement_type = 'PDI_PASS'`, units[0]) != 1 {
		t.Fatal("more than one inspection or movement")
	}
	// Stale version.
	expectError(t, f.record(f.inspector, units[1], map[string]any{"version": 99, "disposition": "PASS", "items": outcomes(ov.Checklist)},
		"stale-0000001"), http.StatusConflict, httpx.CodeVersionConflict)

	// The same key 5 times at once: one inspection, all answers equal.
	_, v := f.unit(units[1])
	body = map[string]any{"version": v, "disposition": "PASS", "items": outcomes(ov.Checklist)}
	ids := make([]string, 5)
	replays := make([]bool, 5)
	for i := range ids {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			w := f.record(f.inspector, units[1], body, "same-key-0001")
			var res pdi.RecordResult
			if w.Code == http.StatusCreated || w.Code == http.StatusOK {
				expect(t, w, w.Code, &res)
				ids[i], replays[i] = res.Inspection.ID, w.Header().Get("Idempotent-Replayed") == "true"
			}
		}(i)
	}
	wg.Wait()
	fresh := 0
	for i := range ids {
		if ids[i] == "" || ids[i] != ids[0] {
			t.Fatalf("answers %v", ids)
		}
		if !replays[i] {
			fresh++
		}
	}
	if fresh != 1 || f.count(`SELECT count(*) FROM pdi_inspections WHERE vehicle_unit_id = $1`, units[1]) != 1 {
		t.Fatalf("%d fresh answers", fresh)
	}
	// Same key, different request.
	expectError(t, f.record(f.inspector, units[1], map[string]any{"version": v, "disposition": "HOLD", "notes": "x",
		"attachmentIds": []string{f.upload("inspector", &f.locL1)}}, "same-key-0001"), http.StatusConflict, httpx.CodeIdempotencyKeyReused)
}

// Checklist configuration: create, duplicate code per brand scope, retire once, audited, never edited.
func TestPDIChecklistConfiguration(t *testing.T) {
	f := setup(t)
	a := f.checklistItem("BRAKES", nil, 1)
	f.checklistItem("BRAKES", &f.brandA, 1) // same code, different brand scope
	expectError(t, f.post(f.manager, "/pdi-checklist-items", map[string]any{"code": "brakes", "label": "Again"}, ""),
		http.StatusConflict, httpx.CodeDuplicate)
	expectError(t, f.post(f.manager, "/pdi-checklist-items", map[string]any{"code": "TYRES", "label": "Tyres",
		"brandId": "00000000-0000-4000-8000-000000000000"}, ""), http.StatusUnprocessableEntity, httpx.CodeValidationFailed)
	expectError(t, f.post(f.manager, "/pdi-checklist-items/"+a.ID+"/retire", map[string]any{"version": a.Version + 1, "reason": "x"}, ""),
		http.StatusConflict, httpx.CodeVersionConflict)
	var retired pdi.ChecklistItem
	expect(t, f.post(f.manager, "/pdi-checklist-items/"+a.ID+"/retire", map[string]any{"version": a.Version, "reason": "Split"}, ""),
		http.StatusOK, &retired)
	if retired.Status != "RETIRED" || retired.RetiredBy == nil || *retired.RetireReason != "Split" {
		t.Fatalf("retired %+v", retired)
	}
	expectError(t, f.post(f.manager, "/pdi-checklist-items/"+a.ID+"/retire", map[string]any{"version": retired.Version, "reason": "x"}, ""),
		http.StatusConflict, httpx.CodeInvalidState)
	// The code is free again once retired.
	f.checklistItem("BRAKES", nil, 1)
	var page pdi.ChecklistPage
	expect(t, f.get(f.viewer, "/pdi-checklist-items?status=ACTIVE&brandId="+f.brandB), http.StatusOK, &page)
	if page.Total != 1 {
		t.Fatalf("brand B applicable items %+v", page)
	}
	if f.count(`SELECT count(*) FROM audit_events WHERE action IN ('pdi_checklist_item.created', 'pdi_checklist_item.retired')`) != 4 {
		t.Fatal("checklist changes not audited")
	}
	_, err := f.pool.Exec(context.Background(), `UPDATE pdi_checklist_items SET label = 'Edited' WHERE id = $1`, a.ID)
	expectPGCode(t, err, "42501")
	_, err = f.pool.Exec(context.Background(), `DELETE FROM pdi_checklist_items WHERE id = $1`, a.ID)
	expectPGCode(t, err, "42501")
}

// Database guards: inspections are append-only; a PDI movement or inspection
// cannot exist without its counterpart; another brand's outcome is refused.
func TestPDIDatabaseGuards(t *testing.T) {
	f := setup(t)
	f.checklistItem("BRAKES", nil, 1)
	units := f.units(&f.locL1, f.poLineA, 2, false)
	var ov pdi.Overview
	expect(t, f.get(f.inspector, "/vehicle-units/"+units[0]+"/pdi"), http.StatusOK, &ov)
	var res pdi.RecordResult
	expect(t, f.record(f.inspector, units[0], map[string]any{"version": ov.Unit.Version, "disposition": "PASS",
		"items": outcomes(ov.Checklist), "attachmentIds": []string{f.upload("inspector", &f.locL1)}}, "guard-0000001"),
		http.StatusCreated, &res)
	ctx := context.Background()
	for _, sql := range []string{
		`UPDATE pdi_inspections SET disposition = 'HOLD' WHERE id = $1`,
		`DELETE FROM pdi_inspections WHERE id = $1`,
		`UPDATE pdi_inspection_items SET outcome = 'FAIL', note = 'x' WHERE inspection_id = $1`,
		`DELETE FROM pdi_inspection_items WHERE inspection_id = $1`,
		`DELETE FROM pdi_inspection_attachments WHERE inspection_id = $1`,
	} {
		_, err := f.pool.Exec(ctx, sql, res.Inspection.ID)
		expectPGCode(t, err, "42501")
	}
	// A PDI_INSPECTION movement without its inspection is refused at commit.
	actor := audit.UserActor(f.ids["inspector"], "inspector")
	err := pgx.BeginFunc(ctx, f.pool, func(tx pgx.Tx) error {
		_, _, err := stock.MoveUnit(ctx, tx, actor, access.Scope{All: true}, stock.MoveRequest{UnitID: units[1], Type: stock.TypeQualityHold,
			IdempotencyKey: "orphan-pdi-01", Source: &stock.MoveSource{Type: stock.SourcePDIInspection,
				ID: "66666666-6666-4666-8666-666666666666", Reference: "PDI-999999"}})
		return err
	})
	expectPGCode(t, err, "23514")
	if st, _ := f.unit(units[1]); st != "RECEIVED_PENDING_PDI" {
		t.Fatal("orphan PDI movement applied")
	}
	// A source on a non-PDI movement is refused by the service.
	err = pgx.BeginFunc(ctx, f.pool, func(tx pgx.Tx) error {
		_, _, err := stock.MoveUnit(ctx, tx, actor, access.Scope{All: true}, stock.MoveRequest{UnitID: res.Unit.ID, Type: stock.TypeDisplayOn,
			IdempotencyKey: "display-src-01", Source: &stock.MoveSource{Type: stock.SourcePDIInspection, ID: res.Inspection.ID, Reference: "x"}})
		return err
	})
	var verr *stock.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("source on a display move: %v", err)
	}
}

// Rolling back 000019 is refused once PDI configuration or history exists.
func TestPDIMigrationRollbackKeepsHistory(t *testing.T) {
	ctx := context.Background()
	url := dbtest.ScratchURL(t)
	m := dbtest.NewMigrator(t, url)
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	if err := m.Migrate(18); err != nil {
		t.Fatalf("down to 18 while empty: %v", err)
	}
	if err := m.Up(); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	f := setupOn(t, pool)
	f.checklistItem("BRAKES", nil, 1)
	if err := m.Migrate(18); err == nil || !strings.Contains(err.Error(), "would destroy inspection and ledger history") {
		t.Fatalf("down over PDI records: %v", err)
	}
}

func expectPGCode(t *testing.T, err error, code string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != code {
		t.Fatalf("err %v, want SQLSTATE %s", err, code)
	}
}
