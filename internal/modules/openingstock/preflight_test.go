package openingstock_test

import (
	"slices"
	"testing"
	"time"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/openingstock"
)

// H-08 preflight: classification, counts, quantities and blockers (pure).

func preflightRow(line int, status, chassis, stockStatus string, issues ...openingstock.Issue) openingstock.RowResult {
	r := openingstock.RowResult{Line: line, Status: status, Issues: issues, Source: map[string]string{
		"brand": "Brand A", "model_code": "MA", "variant_code": "STD", "location_code": "L1", "stock_status": stockStatus,
		"chassis_number": chassis}}
	r.BrandID, r.ModelID, r.VariantID, r.LocationID = sp("b"), sp("m"), sp("v"), sp("l")
	r.ChassisNormalised, r.StockStatus, r.AcquisitionCost = sp(chassis), sp(stockStatus), sp("90000")
	return r
}

func storedFor(r openingstock.RowResult, canProceed bool) openingstock.StoredRow {
	return openingstock.StoredRow{Line: r.Line, CanProceed: canProceed, BrandID: r.BrandID, ModelID: r.ModelID, VariantID: r.VariantID,
		LocationID: r.LocationID, ChassisNormalised: r.ChassisNormalised, StockStatus: r.StockStatus}
}

func TestPreflightClassify(t *testing.T) {
	valid := preflightRow(3, openingstock.RowValid, "CH-1", "AVAILABLE")
	refusal := func(r openingstock.RowResult, codes ...string) *openingstock.RowRefusal {
		is := slices.Clone(r.Issues)
		for _, c := range codes {
			is = append(is, openingstock.Issue{Code: c, Severity: openingstock.SevError})
		}
		return &openingstock.RowRefusal{RowNumber: r.Line, Issues: is}
	}
	warn := preflightRow(4, openingstock.RowWarning, "CH-2", "AVAILABLE",
		openingstock.Issue{Field: "received_date", Code: openingstock.CodeOriginDateMissing, Severity: openingstock.SevWarning})
	errRow := preflightRow(5, openingstock.RowError, "CH-3", "AVAILABLE",
		openingstock.Issue{Field: "chassis_number", Code: openingstock.CodeDuplicateChassisDB, Severity: openingstock.SevError})
	blocked := preflightRow(6, openingstock.RowBlocked, "CH-4", "AVAILABLE",
		openingstock.Issue{Field: "brand", Code: openingstock.CodeBrandNotActive, Severity: openingstock.SevBlocked})
	for _, c := range []struct {
		name    string
		r       openingstock.RowResult
		refusal *openingstock.RowRefusal
		preview string
		want    string
	}{
		{"valid", valid, nil, openingstock.RowValid, openingstock.PreflightValid},
		// Warnings are never accepted, but they are not blocking errors either: their own category.
		{"warning", warn, refusal(warn, openingstock.CodeWarningNotAccepted), openingstock.RowWarning, openingstock.PreflightWarning},
		{"error", errRow, refusal(errRow), openingstock.RowError, openingstock.PreflightError},
		// A duplicate that appeared after the preview is still a blocking error first.
		{"error now, valid at preview", errRow, refusal(errRow, openingstock.CodeChangedSincePreview), openingstock.RowValid,
			openingstock.PreflightError},
		{"unverified", blocked, refusal(blocked, openingstock.CodePreviewRowNotEligible), openingstock.RowBlocked,
			openingstock.PreflightUnverified},
		{"status not approved", valid, refusal(valid, openingstock.CodeStatusNotApproved), openingstock.RowValid, openingstock.PreflightError},
		{"cost not positive", valid, refusal(valid, openingstock.CodeCostNotPositive), openingstock.RowValid, openingstock.PreflightError},
		{"changed since preview", valid, refusal(valid, openingstock.CodeChangedSincePreview), openingstock.RowValid,
			openingstock.PreflightReconcile},
		{"not eligible at preview, valid now", valid, refusal(valid, openingstock.CodePreviewRowNotEligible),
			openingstock.RowBlocked, openingstock.PreflightReconcile},
		{"status differs from preview", valid, nil, openingstock.RowWarning, openingstock.PreflightReconcile},
	} {
		if got := openingstock.Classify(c.r, c.refusal, c.preview); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
}

func TestPreflightEvaluate(t *testing.T) {
	valid := preflightRow(3, openingstock.RowValid, "CH-1", "AVAILABLE")
	valid2 := preflightRow(4, openingstock.RowValid, "CH-2", "DISPLAY")
	warn := preflightRow(5, openingstock.RowWarning, "CH-3", "AVAILABLE",
		openingstock.Issue{Field: "acquisition_cost", Code: openingstock.CodeCostMissing, Severity: openingstock.SevWarning})
	warn.AcquisitionCost = nil
	missing := preflightRow(6, openingstock.RowError, "", "AVAILABLE",
		openingstock.Issue{Field: "chassis_number", Code: openingstock.CodeRequired, Severity: openingstock.SevError})
	missing.ChassisNormalised = nil
	dup := preflightRow(7, openingstock.RowError, "CH-1", "AVAILABLE",
		openingstock.Issue{Field: "chassis_number", Code: openingstock.CodeDuplicateChassisFile, Severity: openingstock.SevError})
	dup.Duplicate = true
	results := []openingstock.RowResult{valid, valid2, warn, missing, dup}
	stored := []openingstock.StoredRow{storedFor(valid, true), storedFor(valid2, true), storedFor(warn, true), storedFor(missing, false),
		storedFor(dup, false)}
	staged := map[int]string{3: openingstock.RowValid, 4: openingstock.RowValid, 5: openingstock.RowWarning, 6: openingstock.RowError,
		7: openingstock.RowError}

	// No approved statuses: the status gate is a batch blocker, not a per-row error.
	sm, counts, qs, ex := openingstock.Evaluate(results, stored, staged, nil)
	want := openingstock.PreflightSummary{TotalRows: 5, ValidRows: 2, WarningRows: 1, BlockingErrorRows: 2, DuplicateRows: 1,
		MissingChassisRows: 1}
	if sm != want {
		t.Fatalf("summary %+v, want %+v", sm, want)
	}
	if len(ex) != 3 || ex[0].RowNumber != 5 || ex[0].Category != openingstock.PreflightWarning || ex[1].RowNumber != 6 || ex[2].RowNumber != 7 {
		t.Fatalf("exceptions %+v", ex)
	}
	if !slices.ContainsFunc(ex[0].Issues, func(i openingstock.Issue) bool { return i.Code == openingstock.CodeWarningNotAccepted }) {
		t.Errorf("warning row does not say warnings are not accepted: %+v", ex[0].Issues)
	}
	byCode := map[string]int{}
	for _, c := range counts {
		byCode[c.Code] = c.Rows
	}
	if byCode[openingstock.CodeRequired] != 1 || byCode[openingstock.CodeDuplicateChassisFile] != 1 || byCode[openingstock.CodeCostMissing] != 1 ||
		byCode[openingstock.CodePreviewRowNotEligible] != 2 {
		t.Errorf("issue counts %+v", counts)
	}
	// Quantities: per location/brand/model/variant/status; counted quantities are never invented.
	if len(qs) != 2 || qs[0].StockStatus != "AVAILABLE" || qs[0].FileRows != 4 || qs[0].ValidRows != 1 ||
		qs[1].StockStatus != "DISPLAY" || qs[1].FileRows != 1 || qs[1].ValidRows != 1 {
		t.Fatalf("quantities %+v", qs)
	}
	for _, q := range qs {
		if q.PhysicalCount != nil || q.Difference != nil {
			t.Errorf("physical count invented: %+v", q)
		}
	}

	// Approved statuses that exclude DISPLAY: that row becomes a blocking error.
	sm, _, _, ex = openingstock.Evaluate(results, stored, staged, []string{"AVAILABLE"})
	if sm.ValidRows != 1 || sm.BlockingErrorRows != 3 {
		t.Fatalf("with approved statuses: %+v", sm)
	}
	if !slices.ContainsFunc(ex, func(r openingstock.PreflightRow) bool {
		return r.RowNumber == 4 && slices.ContainsFunc(r.Issues, func(i openingstock.Issue) bool { return i.Code == openingstock.CodeStatusNotApproved })
	}) {
		t.Errorf("row 4 not refused for its status: %+v", ex)
	}

	// The catalogue changed since the preview: manual reconciliation, flagged as changed.
	moved := valid
	moved.VariantID = sp("v2")
	sm, _, _, ex = openingstock.Evaluate([]openingstock.RowResult{moved}, []openingstock.StoredRow{storedFor(valid, true)},
		map[int]string{3: openingstock.RowValid}, nil)
	if sm.ManualReconciliationRows != 1 || sm.ChangedSincePreviewRows != 1 || len(ex) != 1 || !ex[0].ChangedSincePreview {
		t.Fatalf("changed since preview: %+v %+v", sm, ex)
	}
}

func TestPreflightBlockers(t *testing.T) {
	v := openingstock.TemplateVersion
	codes := func(p openingstock.Preflight) []string {
		var out []string
		for _, b := range openingstock.Blockers(p) {
			out = append(out, b.Code+"/"+b.Decision)
		}
		return out
	}
	base := openingstock.Preflight{BatchStatus: openingstock.BatchPreviewed, TemplateVersion: &v, RowsChecked: true,
		Source:  openingstock.PreflightSource{Status: openingstock.SourceVerified},
		Summary: openingstock.PreflightSummary{TotalRows: 2, ValidRows: 2}}
	got := codes(base)
	want := []string{"APPROVAL_NOT_RECORDED/OQ-49", "OPENING_STATUSES_NOT_APPROVED/OQ-47", "PHYSICAL_COUNT_QUANTITIES_NOT_RECORDED/H-08"}
	if !slices.Equal(got, want) {
		t.Fatalf("clean batch blockers %v, want %v", got, want)
	}
	// Approved and statuses configured: the physical count quantities still block H-08.
	ready := base
	ready.Approval = &openingstock.Approval{PhysicalCountReference: "PC-1"}
	ready.ApprovedOpeningStatuses = []string{"AVAILABLE"}
	if got := codes(ready); !slices.Equal(got, []string{"PHYSICAL_COUNT_QUANTITIES_NOT_RECORDED/H-08"}) {
		t.Fatalf("approved batch blockers %v", got)
	}
	bad := base
	bad.Summary.ValidRows, bad.Expired = 1, true
	bad.Source.Status = openingstock.SourceInvalid
	if got := codes(bad); !slices.Contains(got, "ROWS_NOT_VALID/OQ-49") || !slices.Contains(got, "PREVIEW_EXPIRED/OQ-48") ||
		!slices.Contains(got, "SOURCE_INVALID/") {
		t.Fatalf("bad batch blockers %v", got)
	}
	committed := base
	committed.Commit = &openingstock.CommitSummary{Reference: "OSC-000001", CommittedAt: time.Now()}
	if got := codes(committed); !slices.Equal(got, []string{"ALREADY_COMMITTED/"}) {
		t.Fatalf("committed blockers %v", got)
	}
	rejected := base
	rejected.BatchStatus, rejected.RowsChecked = openingstock.BatchRejected, false
	if got := codes(rejected); got[0] != "BATCH_REJECTED/" {
		t.Fatalf("rejected blockers %v", got)
	}
}
