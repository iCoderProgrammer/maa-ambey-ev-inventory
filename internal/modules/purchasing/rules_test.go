package purchasing

import (
	"errors"
	"testing"
	"time"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
)

func TestTransitions(t *testing.T) {
	allowed := map[string][]string{
		ActionSubmitted: {StatusDraft},
		ActionApproved:  {StatusSubmitted},
		ActionReturned:  {StatusSubmitted},
		ActionCancelled: {StatusDraft, StatusSubmitted, StatusApproved},
	}
	for action, from := range allowed {
		for _, status := range Statuses {
			err := checkTransition(action, status, 0)
			want := false
			for _, f := range from {
				want = want || f == status
			}
			if (err == nil) != want {
				t.Errorf("%s from %s: err %v", action, status, err)
			}
			var se *StateError
			if err != nil && !errors.As(err, &se) {
				t.Errorf("%s from %s: want StateError, got %T", action, status, err)
			}
		}
	}
	if checkTransition(ActionCancelled, StatusApproved, 1) == nil {
		t.Error("cancel with received goods must be refused")
	}
	if checkEditable(StatusDraft) != nil || checkEditable(StatusSubmitted) == nil {
		t.Error("only drafts are editable")
	}
}

func TestPrerequisites(t *testing.T) {
	today := time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC)
	ok := lineFacts{LineID: "l1", BrandID: "b1", BrandName: "Brand A", BrandStatus: "ACTIVE", BrandAuthorisation: "AUTHORISED",
		ModelName: "M", ModelStatus: "ACTIVE", VariantName: "V", VariantStatus: "ACTIVE", SupplierLinkCurrent: true}
	if got := prerequisites(poFacts{SupplierStatus: "VERIFIED", ExpectedDate: &today, Today: today, Lines: []lineFacts{ok}}); len(got) != 0 {
		t.Fatalf("all met: %+v", got)
	}
	bad := ok
	bad.LineID, bad.BrandStatus, bad.SupplierLinkCurrent, bad.ModelStatus, bad.VariantStatus = "l2", "PENDING_VERIFICATION", false, "ARCHIVED", "PENDING_VERIFICATION"
	revoked := ok
	revoked.LineID, revoked.BrandID, revoked.BrandAuthorisation = "l3", "b2", "REVOKED"
	yesterday := today.AddDate(0, 0, -1)
	got := prerequisites(poFacts{SupplierStatus: "PENDING_VERIFICATION", ExpectedDate: &yesterday, Today: today, Lines: []lineFacts{bad, revoked}})
	keys := map[string]int{}
	for _, c := range got {
		keys[c.Key]++
	}
	// One brand check per brand (b1 once even though two lines could share it), model/variant per line.
	want := map[string]int{"expected_date": 1, "supplier_verified": 1, "brand_active": 2, "supplier_brand_link": 1, "model_active": 1, "variant_active": 1}
	for k, n := range want {
		if keys[k] != n {
			t.Errorf("%s: %d, want %d (%+v)", k, keys[k], n, got)
		}
	}
	got = prerequisites(poFacts{SupplierStatus: "ARCHIVED", Today: today})
	if len(got) != 3 || got[0].Key != "lines" || got[1].Key != "expected_date" || got[2].Key != "supplier_verified" {
		t.Fatalf("empty: %+v", got)
	}
}

func TestApprovalRequirement(t *testing.T) {
	ref := "Owner sign-off"
	limit := "5000.00"
	none := Settings{FirstTimeSupplierNeedsApproval: true}
	if req, _ := approvalRequirement("1.00", none, true); !req {
		t.Error("without a threshold every PO needs approval")
	}
	set := Settings{ApprovalThreshold: &limit, ApprovalReference: &ref, FirstTimeSupplierNeedsApproval: true}
	if req, _ := approvalRequirement("1.00", set, false); !req {
		t.Error("first-time supplier needs approval when enabled")
	}
	if req, _ := approvalRequirement("5000.00", set, true); req {
		t.Error("total equal to the threshold is within it")
	}
	if req, _ := approvalRequirement("5000.01", set, true); !req {
		t.Error("above the threshold needs approval")
	}
	set.FirstTimeSupplierNeedsApproval = false
	if req, _ := approvalRequirement("10.00", set, false); req {
		t.Error("first-time rule disabled")
	}
}

func TestLineWarnings(t *testing.T) {
	zero, diff := "0.00", "-10.00"
	cases := []struct {
		l    Line
		want []string
	}{
		{Line{Quantity: 5}, nil},
		{Line{Quantity: 5, CostVariance: &zero}, nil},
		{Line{Quantity: 5, CostVariance: &diff}, []string{WarnPriceVariance}},
		{Line{Quantity: 5, ReceivedQuantity: 2}, []string{WarnPartialDelivery}},
		{Line{Quantity: 5, ReceivedQuantity: 5}, nil},
		{Line{Quantity: 5, ReceivedQuantity: 6, CostVariance: &diff}, []string{WarnPriceVariance, WarnOverReceipt}},
	}
	for _, c := range cases {
		got := lineWarnings(c.l)
		if len(got) != len(c.want) {
			t.Errorf("%+v: %v, want %v", c.l, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%+v: %v, want %v", c.l, got, c.want)
			}
		}
	}
}

func TestActionsHint(t *testing.T) {
	sub := "u1"
	po := PurchaseOrder{Status: StatusSubmitted, CreatedBy: "u1", SubmittedBy: &sub}
	all := []string{access.PermPurchaseOrderManage, access.PermPurchaseOrderApprove, access.PermPurchaseOrderCancel}
	own := actionsFor(po, access.NewAccess("u1", nil, all, nil, false))
	if own.Approve || !own.Return || !own.Cancel || own.Edit {
		t.Errorf("own PO: %+v", own)
	}
	other := actionsFor(po, access.NewAccess("u2", nil, all, nil, false))
	if !other.Approve {
		t.Errorf("other approver: %+v", other)
	}
	none := actionsFor(po, access.NewAccess("u2", nil, nil, nil, false))
	if none != (Actions{}) {
		t.Errorf("no permissions: %+v", none)
	}
}
