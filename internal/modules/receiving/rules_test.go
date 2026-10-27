package receiving

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
)

func ptr(s string) *string { return &s }

func TestAvailabilityCountsPostedAndOtherDrafts(t *testing.T) {
	cases := []struct {
		a    poLineAvailability
		want int
	}{
		{poLineAvailability{Ordered: 5}, 5},
		{poLineAvailability{Ordered: 5, Posted: 2}, 3},
		{poLineAvailability{Ordered: 5, Posted: 2, OtherDrafts: 2}, 1},
		{poLineAvailability{Ordered: 5, Posted: 4, OtherDrafts: 3}, 0}, // never below zero
	}
	for _, c := range cases {
		if got := c.a.available(); got != c.want {
			t.Errorf("%+v: %d, want %d", c.a, got, c.want)
		}
	}
}

func TestDecideOverReceipt(t *testing.T) {
	avail := map[string]poLineAvailability{"l1": {ID: "l1", LineNo: 1, Ordered: 5, Posted: 1, OtherDrafts: 1}}
	in := func(received int, reason *string) []indexedLine {
		return []indexedLine{{Index: 0, LineInput: LineInput{PurchaseOrderLineID: "l1", ReceivedQuantity: received,
			AcceptedQuantity: received, OverReceiptReason: reason}}}
	}
	// Within the 3 left: no authorisation, any earlier one dropped.
	d, err := decideOverReceipt(in(3, nil), avail, map[string]storedOverride{"l1": {Quantity: 4}}, false, "u1")
	if err != nil || d[0].Override != nil || d[0].Excess != 0 {
		t.Fatalf("within: %+v %v", d, err)
	}
	// Above, no reason: refused with the line's numbers.
	_, err = decideOverReceipt(in(4, nil), avail, nil, true, "u1")
	var over *OverReceiptError
	if !errors.As(err, &over) || over.Fields[0].Field != "lines[0].receivedQuantity" {
		t.Fatalf("no reason: %v", err)
	}
	// Above, reason, no permission: forbidden (wins over "needs").
	_, err = decideOverReceipt(in(4, ptr("Supplier sent an extra unit")), avail, nil, false, "u1")
	var forbidden *OverReceiptForbiddenError
	if !errors.As(err, &forbidden) || forbidden.Fields[0].Field != "lines[0].overReceiptReason" {
		t.Fatalf("no permission: %v", err)
	}
	// Above, reason, permission: a new authorisation for the received quantity.
	d, err = decideOverReceipt(in(4, ptr("Supplier sent an extra unit")), avail, nil, true, "u2")
	if err != nil || d[0].Override == nil || !d[0].NewOverride || d[0].Override.Quantity != 4 || d[0].Override.By != "u2" || d[0].Excess != 1 {
		t.Fatalf("authorised: %+v %v", d, err)
	}
	// An earlier authorisation that covers the quantity is kept, even for a caller without the permission.
	at := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	prev := map[string]storedOverride{"l1": {Quantity: 5, Reason: "agreed", By: "u2", At: at}}
	d, err = decideOverReceipt(in(4, nil), avail, prev, false, "u1")
	if err != nil || d[0].Override == nil || d[0].NewOverride || d[0].Override.By != "u2" || !d[0].Override.At.Equal(at) {
		t.Fatalf("kept: %+v %v", d, err)
	}
	// ... but not once the quantity goes beyond it.
	if _, err = decideOverReceipt(in(6, nil), avail, prev, false, "u1"); !errors.As(err, &over) {
		t.Fatalf("beyond earlier authorisation: %v", err)
	}
	// A line of another PO is refused.
	bad := []indexedLine{{Index: 2, LineInput: LineInput{PurchaseOrderLineID: "other", ReceivedQuantity: 1, AcceptedQuantity: 1}}}
	var verr *ValidationError
	if _, err = decideOverReceipt(bad, avail, nil, true, "u1"); !errors.As(err, &verr) || verr.Fields[0].Field != "lines[2].purchaseOrderLineId" {
		t.Fatalf("unknown line: %v", err)
	}
}

func TestLineAndUnitWarnings(t *testing.T) {
	chassis := ptr("CH-1")
	l := Line{AvailableQuantity: 5, ReceivedQuantity: 2, AcceptedQuantity: 1, RejectedQuantity: 1,
		Units: []Unit{{ChassisNumber: chassis}}}
	got := lineWarnings(l)
	want := []string{WarnPartialDelivery, WarnIdentifiersIncomplete, WarnRejectionNotExplained}
	if !slices.Equal(got, want) {
		t.Fatalf("warnings %v, want %v", got, want)
	}
	full := Line{AvailableQuantity: 1, ReceivedQuantity: 1, AcceptedQuantity: 1, Units: []Unit{{ChassisNumber: chassis}}}
	if got := lineWarnings(full); len(got) != 0 {
		t.Fatalf("complete line warns %v", got)
	}
	over := Line{AvailableQuantity: 1, ReceivedQuantity: 2, ExcessQuantity: 1, AcceptedQuantity: 2,
		Units: []Unit{{ChassisNumber: chassis}, {ChassisNumber: ptr("CH-2")}}}
	if got := lineWarnings(over); !slices.Equal(got, []string{WarnOverReceiptUnauthorised}) {
		t.Fatalf("unauthorised over-receipt %v", got)
	}
	over.OverReceipt = &OverReceipt{Quantity: 2}
	if got := lineWarnings(over); !slices.Equal(got, []string{WarnOverReceipt}) {
		t.Fatalf("authorised over-receipt %v", got)
	}
	// Serial repeats warn (case-insensitively) across lines; a missing chassis warns.
	lines := []Line{
		{Units: []Unit{{ChassisNumber: chassis, MotorSerial: ptr("m-1"), BatterySerial: ptr("B1")}}},
		{Units: []Unit{{MotorSerial: ptr("M-1"), BatterySerial: ptr("B2"), ChargerSerial: ptr("C1")}}},
	}
	unitWarnings(lines)
	if !slices.Equal(lines[0].Units[0].Warnings, []string{WarnDuplicateMotorSerial}) ||
		!slices.Equal(lines[1].Units[0].Warnings, []string{WarnChassisMissing, WarnDuplicateMotorSerial}) {
		t.Fatalf("unit warnings %v %v", lines[0].Units[0].Warnings, lines[1].Units[0].Warnings)
	}
}

func TestStateRules(t *testing.T) {
	if checkEditable(StatusDraft, "APPROVED") != nil || checkEditable(StatusDraft, "PARTIALLY_RECEIVED") != nil {
		t.Fatal("draft of an approved PO is editable")
	}
	for _, c := range [][2]string{{StatusDraft, "CANCELLED"}, {StatusDraft, "RECEIVED"}, {StatusCancelled, "APPROVED"}, {StatusPosted, "APPROVED"}} {
		if checkEditable(c[0], c[1]) == nil {
			t.Errorf("%s receipt of %s PO editable", c[0], c[1])
		}
	}
	if checkCancel(StatusDraft) != nil || checkCancel(StatusCancelled) == nil || checkCancel(StatusPosted) == nil {
		t.Fatal("only drafts are cancelled")
	}
	if NormaliseChassis("  ab-12/x ") != "AB-12/X" {
		t.Fatal("normalisation trims and upper-cases, keeping separators")
	}
}

func TestActionsFollowStateAndPermissions(t *testing.T) {
	d := Detail{Receipt: Receipt{Status: StatusDraft}, PurchaseOrderReceivable: true}
	viewer := access.NewAccess("u", nil, []string{access.PermGoodsReceiptView}, nil, false)
	if a := actionsFor(d, viewer); a.Edit || a.Cancel || a.AddDocument || a.AuthoriseOverReceipt || a.Post {
		t.Fatalf("viewer %+v", a)
	}
	all := access.NewAccess("u", nil, everyGRN, nil, false)
	if a := actionsFor(d, all); !a.Edit || !a.Cancel || !a.AddDocument || !a.AuthoriseOverReceipt || a.Post {
		t.Fatalf("manager %+v", a)
	}
	d.PurchaseOrderReceivable = false
	if a := actionsFor(d, all); a.Edit || !a.Cancel {
		t.Fatalf("PO closed %+v", a)
	}
	d.Status = StatusCancelled
	if a := actionsFor(d, all); a.Edit || a.Cancel || a.AddDocument {
		t.Fatalf("cancelled %+v", a)
	}
}
