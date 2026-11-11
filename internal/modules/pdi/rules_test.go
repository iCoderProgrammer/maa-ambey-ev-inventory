package pdi

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

const (
	itemA = "11111111-1111-4111-8111-111111111111"
	itemB = "22222222-2222-4222-8222-222222222222"
	attA  = "33333333-3333-4333-8333-333333333333"
	attB  = "44444444-4444-4444-8444-444444444444"
)

func ptr[T any](v T) *T { return &v }

func fields(err error) []string {
	var v *ValidationError
	if !errors.As(err, &v) {
		return nil
	}
	out := make([]string, len(v.Fields))
	for i, f := range v.Fields {
		out[i] = f.Field
	}
	return out
}

func TestValidateRecordPassHoldReject(t *testing.T) {
	pass := RecordInput{Version: ptr(1), Disposition: "PASS", Items: []ItemOutcomeInput{
		{ChecklistItemID: itemA, Outcome: "PASS"}, {ChecklistItemID: itemB, Outcome: "PASS", Note: ptr(" fine ")}}}
	r, err := validateRecord(pass, "key-00000001")
	if err != nil || r.Disposition != DispositionPass || *r.Items[1].Note != "fine" {
		t.Fatalf("valid pass refused or not normalised: %v %+v", err, r)
	}
	hold := RecordInput{Version: ptr(2), Disposition: "HOLD", Notes: ptr("Scratch on panel"),
		Items: []ItemOutcomeInput{{ChecklistItemID: itemA, Outcome: "FAIL", Note: ptr("scratch")}}, AttachmentIDs: []string{attA}}
	if _, err := validateRecord(hold, "key-00000002"); err != nil {
		t.Fatalf("valid hold refused: %v", err)
	}
	// A hold with no checklist answers (none configured) is fine: it only makes the unit non-saleable.
	if _, err := validateRecord(RecordInput{Version: ptr(1), Disposition: "REJECT", Notes: ptr("Frame cracked"),
		AttachmentIDs: []string{attA}}, "key-00000003"); err != nil {
		t.Fatalf("valid reject refused: %v", err)
	}
}

func TestValidateRecordRefusals(t *testing.T) {
	tests := []struct {
		name string
		in   RecordInput
		key  string
		want []string
	}{
		{"no version, bad disposition, bad key", RecordInput{Disposition: "OK"}, "short", []string{"Idempotency-Key", "version", "disposition"}},
		{"pass with a failed item", RecordInput{Version: ptr(1), Disposition: "PASS",
			Items: []ItemOutcomeInput{{ChecklistItemID: itemA, Outcome: "FAIL", Note: ptr("loose")}}}, "key-00000001", []string{"disposition"}},
		{"failed item without note", RecordInput{Version: ptr(1), Disposition: "HOLD", Notes: ptr("x"), AttachmentIDs: []string{attA},
			Items: []ItemOutcomeInput{{ChecklistItemID: itemA, Outcome: "FAIL", Note: ptr("  ")}}}, "key-00000001",
			[]string{"items[0].note"}},
		{"hold without notes or evidence", RecordInput{Version: ptr(1), Disposition: "HOLD"}, "key-00000001",
			[]string{"notes", "attachmentIds"}},
		{"reject without evidence", RecordInput{Version: ptr(1), Disposition: "REJECT", Notes: ptr("bad")}, "key-00000001",
			[]string{"attachmentIds"}},
		{"duplicates and bad ids", RecordInput{Version: ptr(1), Disposition: "PASS", Items: []ItemOutcomeInput{
			{ChecklistItemID: itemA, Outcome: "PASS"}, {ChecklistItemID: strings.ToUpper(itemA), Outcome: "PASS"},
			{ChecklistItemID: "nope", Outcome: "MAYBE"}}, AttachmentIDs: []string{attA, attA, "x"}}, "key-00000001",
			[]string{"items[1].checklistItemId", "items[2].checklistItemId", "items[2].outcome", "attachmentIds[1]", "attachmentIds[2]"}},
		{"version zero, blank notes", RecordInput{Version: ptr(0), Disposition: "HOLD", Notes: ptr(" "), AttachmentIDs: []string{attA}},
			"key-00000001", []string{"version", "notes"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateRecord(tt.in, tt.key)
			got := fields(err)
			for _, w := range tt.want {
				if !slices.Contains(got, w) {
					t.Fatalf("fields %v, want %q", got, w)
				}
			}
		})
	}
}

func TestRequestHashIgnoresOrderButNotContent(t *testing.T) {
	a := normalisedRecord{Version: 1, Disposition: "HOLD", Notes: ptr("n"), AttachmentIDs: []string{attA, attB},
		Items: []ItemOutcomeInput{{ChecklistItemID: itemA, Outcome: "PASS"}, {ChecklistItemID: itemB, Outcome: "FAIL", Note: ptr("x")}}}
	b := a
	b.AttachmentIDs = []string{attB, attA}
	b.Items = []ItemOutcomeInput{a.Items[1], a.Items[0]}
	if requestHash("u", a) != requestHash("u", b) {
		t.Fatal("order changed the hash")
	}
	c := a
	c.Version = 2
	if requestHash("u", a) == requestHash("u", c) || requestHash("u", a) == requestHash("v", a) {
		t.Fatal("different request, same hash")
	}
}

func TestAllowedDispositionsFollowTheApprovedEdgesOnly(t *testing.T) {
	cases := map[string][]string{
		"RECEIVED_PENDING_PDI": {"PASS", "HOLD"}, // no direct REJECT (OQ-31)
		"QUALITY_HOLD":         {"REJECT"},       // no release to AVAILABLE (OQ-31)
		"AVAILABLE":            {}, "REJECTED": {}, "DISPLAY": {}, "SOLD": {},
	}
	for status, want := range cases {
		if got := AllowedDispositions(status); !slices.Equal(got, want) {
			t.Fatalf("%s: got %v, want %v", status, got, want)
		}
	}
}

func TestMatchChecklistNeedsExactlyTheApplicableItems(t *testing.T) {
	list := []ChecklistItem{{ID: itemA, Code: "BRAKES", Label: "Brakes"}, {ID: itemB, Code: "LIGHTS", Label: "Lights"}}
	if err := matchChecklist(list, []ItemOutcomeInput{{ChecklistItemID: itemB}, {ChecklistItemID: itemA}}); err != nil {
		t.Fatal(err)
	}
	got := fields(matchChecklist(list, []ItemOutcomeInput{{ChecklistItemID: itemA}, {ChecklistItemID: attA}}))
	if !slices.Contains(got, "items[1].checklistItemId") || !slices.Contains(got, "items") {
		t.Fatalf("missing/unknown items not reported: %v", got)
	}
}

func TestValidateChecklistCreate(t *testing.T) {
	n, err := validateChecklistCreate(ChecklistCreateInput{Code: " brakes_1 ", Label: " Brakes ", SortOrder: ptr(5)})
	if err != nil || n.Code != "BRAKES_1" || n.Label != "Brakes" || n.SortOrder != 5 {
		t.Fatalf("got %+v %v", n, err)
	}
	got := fields(func() error {
		_, err := validateChecklistCreate(ChecklistCreateInput{Code: "1x", Label: "", BrandID: ptr("b"), SortOrder: ptr(10000)})
		return err
	}())
	for _, w := range []string{"code", "label", "brandId", "sortOrder"} {
		if !slices.Contains(got, w) {
			t.Fatalf("fields %v, want %q", got, w)
		}
	}
}
