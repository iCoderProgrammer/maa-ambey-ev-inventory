package openingstock_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/openingstock"
)

func sp(s string) *string { return &s }
func bp(b bool) *bool     { return &b }

func TestBatchGate(t *testing.T) {
	ok := openingstock.Batch{Status: openingstock.BatchPreviewed, TemplateVersion: sp(openingstock.TemplateVersion)}
	approval := &openingstock.Approval{ID: "a"}
	statuses := []string{"AVAILABLE"}
	if err := openingstock.BatchGate(ok, approval, statuses); err != nil {
		t.Fatalf("eligible batch refused: %v", err)
	}
	var already *openingstock.AlreadyCommittedError
	committed := ok
	committed.Commit = &openingstock.CommitSummary{ID: "c"}
	if err := openingstock.BatchGate(committed, approval, statuses); !errors.As(err, &already) {
		t.Errorf("committed batch: %v", err)
	}
	for name, tc := range map[string]struct {
		b        openingstock.Batch
		approval *openingstock.Approval
		statuses []string
		want     error
	}{
		"rejected file":         {openingstock.Batch{Status: openingstock.BatchRejected, TemplateVersion: sp(openingstock.TemplateVersion)}, approval, statuses, openingstock.ErrNotCommittable},
		"no template version":   {openingstock.Batch{Status: openingstock.BatchPreviewed}, approval, statuses, openingstock.ErrNotCommittable},
		"other template":        {openingstock.Batch{Status: openingstock.BatchPreviewed, TemplateVersion: sp("OS-IMPORT-0")}, approval, statuses, openingstock.ErrNotCommittable},
		"expired":               {openingstock.Batch{Status: openingstock.BatchPreviewed, TemplateVersion: sp(openingstock.TemplateVersion), Expired: true}, approval, statuses, openingstock.ErrPreviewExpired},
		"no approval":           {ok, nil, statuses, openingstock.ErrApprovalRequired},
		"no status is approved": {ok, approval, nil, openingstock.ErrStatusesNotApproved},
	} {
		if err := openingstock.BatchGate(tc.b, tc.approval, tc.statuses); !errors.Is(err, tc.want) {
			t.Errorf("%s: got %v, want %v", name, err, tc.want)
		}
	}
}

func TestSameSource(t *testing.T) {
	vals := map[string]string{}
	for _, c := range openingstock.Columns {
		vals[c.Name] = ""
	}
	vals["chassis_number"] = "CH-1"
	parsed := []openingstock.SourceRow{{Line: 3, Values: vals}}
	stored := []openingstock.StoredRow{{Line: 3, Source: map[string]string{"chassis_number": "CH-1"}}}
	if !openingstock.SameSource(parsed, stored) {
		t.Fatal("identical rows reported as changed")
	}
	changed := []openingstock.StoredRow{{Line: 3, Source: map[string]string{"chassis_number": "CH-2"}}}
	if openingstock.SameSource(parsed, changed) {
		t.Error("changed value not detected")
	}
	if openingstock.SameSource(parsed, []openingstock.StoredRow{{Line: 4, Source: stored[0].Source}}) {
		t.Error("changed line number not detected")
	}
	if openingstock.SameSource(parsed, append(stored, stored[0])) {
		t.Error("different row count not detected")
	}
	mismatch := []openingstock.SourceRow{{Line: 3, Values: vals, FieldCountMismatch: true}}
	if openingstock.SameSource(mismatch, stored) {
		t.Error("a malformed line must never match")
	}
}

func TestRowGate(t *testing.T) {
	ids := func(r *openingstock.RowResult) {
		r.BrandID, r.ModelID, r.VariantID, r.LocationID = sp("b"), sp("m"), sp("v"), sp("l")
		r.ChassisNormalised, r.StockStatus, r.AcquisitionCost = sp("CH-1"), sp("AVAILABLE"), sp("90000")
	}
	valid := openingstock.RowResult{Line: 3, Status: openingstock.RowValid}
	ids(&valid)
	stored := []openingstock.StoredRow{{Line: 3, CanProceed: true, BrandID: sp("b"), ModelID: sp("m"), VariantID: sp("v"),
		LocationID: sp("l"), ChassisNormalised: sp("CH-1"), StockStatus: sp("AVAILABLE")}}
	approved := []string{"AVAILABLE", "RECEIVED_PENDING_PDI"}
	if got := openingstock.RowGate([]openingstock.RowResult{valid}, stored, approved); len(got) != 0 {
		t.Fatalf("valid row refused: %+v", got)
	}
	codes := func(rr []openingstock.RowRefusal) []string {
		var out []string
		for _, r := range rr {
			for _, is := range r.Issues {
				out = append(out, is.Code)
			}
		}
		return out
	}
	check := func(name string, r openingstock.RowResult, st []openingstock.StoredRow, want ...string) {
		t.Helper()
		got := openingstock.RowGate([]openingstock.RowResult{r}, st, approved)
		if len(got) != 1 || got[0].RowNumber != r.Line {
			t.Fatalf("%s: refusals %+v", name, got)
		}
		c := codes(got)
		for _, w := range want {
			if !slices.Contains(c, w) {
				t.Errorf("%s: codes %v miss %s", name, c, w)
			}
		}
	}
	errRow := valid
	errRow.Status = openingstock.RowError
	errRow.Issues = []openingstock.Issue{{Field: "chassis_number", Code: openingstock.CodeDuplicateChassisDB, Severity: openingstock.SevError}}
	check("fatal error", errRow, stored, openingstock.CodeDuplicateChassisDB)

	blocked := valid
	blocked.Status = openingstock.RowBlocked
	blocked.Issues = []openingstock.Issue{{Field: "brand", Code: openingstock.CodeBrandNotActive, Severity: openingstock.SevBlocked}}
	check("unverified mapping", blocked, stored, openingstock.CodeBrandNotActive)

	warn := valid
	warn.Status = openingstock.RowWarning
	warn.Issues = []openingstock.Issue{{Field: "motor_serial", Code: openingstock.CodePossibleDupSerialDB, Severity: openingstock.SevWarning}}
	check("warning", warn, stored, openingstock.CodeWarningNotAccepted, openingstock.CodePossibleDupSerialDB)

	hold := valid
	hold.StockStatus = sp("QUALITY_HOLD")
	holdStored := slices.Clone(stored)
	holdStored[0].StockStatus = sp("QUALITY_HOLD")
	check("status not approved", hold, holdStored, openingstock.CodeStatusNotApproved)

	zero := valid
	zero.AcquisitionCost = sp("0.00")
	check("zero cost", zero, stored, openingstock.CodeCostNotPositive)

	notEligible := slices.Clone(stored)
	notEligible[0].CanProceed = false
	check("not eligible at preview", valid, notEligible, openingstock.CodePreviewRowNotEligible)
	check("line missing from preview", valid, nil, openingstock.CodePreviewRowNotEligible)

	moved := slices.Clone(stored)
	moved[0].LocationID = sp("other")
	check("resolution changed", valid, moved, openingstock.CodeChangedSincePreview)
	variant := slices.Clone(stored)
	variant[0].VariantID = sp("other")
	check("variant changed", valid, variant, openingstock.CodeChangedSincePreview)

	// One refused line among valid ones: only it is listed, and the caller refuses the whole batch.
	second := valid
	second.Line = 4
	got := openingstock.RowGate([]openingstock.RowResult{valid, errRow}, stored, approved)
	if len(got) != 1 {
		t.Fatalf("mixed batch: %+v", got)
	}
	_ = second
}

func TestValidateApproval(t *testing.T) {
	today := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	good := openingstock.ApprovalInput{PhysicalCountReference: "PC-1", PhysicalCountDate: "2026-06-29",
		MappingsReviewed: bp(true), DuplicatesReviewed: bp(true), ReceivedDatesApproved: bp(false)}
	if probs := openingstock.ValidateApproval(good, today); len(probs) != 0 {
		t.Fatalf("good approval refused: %+v", probs)
	}
	fields := func(in openingstock.ApprovalInput) []string {
		var out []string
		for _, p := range openingstock.ValidateApproval(in, today) {
			out = append(out, p.Field)
		}
		return out
	}
	bad := openingstock.ApprovalInput{PhysicalCountReference: "  ", PhysicalCountDate: "30/06/2026", MappingsReviewed: bp(false)}
	got := fields(bad)
	for _, f := range []string{"physicalCountReference", "physicalCountDate", "mappingsReviewed", "duplicatesReviewed", "receivedDatesApproved"} {
		if !slices.Contains(got, f) {
			t.Errorf("missing problem for %s (got %v)", f, got)
		}
	}
	future := good
	future.PhysicalCountDate = "2026-07-01"
	if !slices.Contains(fields(future), "physicalCountDate") {
		t.Error("a physical count after today was accepted")
	}
	ctrl := good
	ctrl.PhysicalCountReference = "PC\x001"
	if !slices.Contains(fields(ctrl), "physicalCountReference") {
		t.Error("control characters accepted")
	}
	blank := good
	blank.Note = sp(" ")
	if !slices.Contains(fields(blank), "note") {
		t.Error("blank note accepted")
	}

	a := openingstock.Approval{ApprovedByID: "u1", PhysicalCountReference: "PC-1", PhysicalCountDate: "2026-06-29"}
	if !openingstock.SameApproval(a, "u1", good) {
		t.Error("identical approval not recognised as a replay")
	}
	if openingstock.SameApproval(a, "u2", good) {
		t.Error("another approver's request treated as a replay")
	}
	other := good
	other.ReceivedDatesApproved = bp(true)
	if openingstock.SameApproval(a, "u1", other) {
		t.Error("a different approval treated as a replay")
	}
}

func TestParseOpeningStatuses(t *testing.T) {
	got, err := openingstock.ParseOpeningStatuses("")
	if err != nil || len(got) != 0 {
		t.Fatalf("empty: %v %v (empty must approve nothing)", got, err)
	}
	got, err = openingstock.ParseOpeningStatuses(" AVAILABLE,RECEIVED_PENDING_PDI,AVAILABLE ")
	if err != nil || !slices.Equal(got, []string{"AVAILABLE", "RECEIVED_PENDING_PDI"}) {
		t.Fatalf("got %v %v", got, err)
	}
	for _, v := range []string{"SOLD", "available", "RESERVED"} {
		if _, err := openingstock.ParseOpeningStatuses(v); err == nil {
			t.Errorf("%q accepted", v)
		}
	}
}
