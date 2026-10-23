package purchasing

import (
	"math/big"
	"time"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
)

// Lifecycle rules (PO-001, PO-002, ADR-016). They are pure functions so they
// can be tested without a database; the repository calls them inside the
// transaction after locking the rows they look at.

// checkEditable: only drafts change (header and lines).
func checkEditable(status string) error {
	if status != StatusDraft {
		return &StateError{Message: "Only draft purchase orders can be changed. Return it to draft or cancel it instead."}
	}
	return nil
}

// checkTransition says whether action may start from status. received is the
// quantity already received on the PO (cancelling needs nothing received).
func checkTransition(action, status string, received int) error {
	switch action {
	case ActionSubmitted:
		if status != StatusDraft {
			return &StateError{Message: "Only draft purchase orders can be submitted."}
		}
	case ActionApproved:
		if status != StatusSubmitted {
			return &StateError{Message: "Only submitted purchase orders can be approved."}
		}
	case ActionReturned:
		if status != StatusSubmitted {
			return &StateError{Message: "Only submitted purchase orders can be returned to draft."}
		}
	case ActionCancelled:
		switch status {
		case StatusDraft, StatusSubmitted:
		case StatusApproved:
			if received > 0 {
				return &StateError{Message: "Goods were received against this purchase order, so it cannot be cancelled."}
			}
		case StatusCancelled:
			return &StateError{Message: "This purchase order is already cancelled."}
		default:
			return &StateError{Message: "Purchase orders with received goods cannot be cancelled."}
		}
	default:
		return &StateError{Message: "Unknown action."}
	}
	return nil
}

// lineFacts is what the prerequisites look at for one line.
type lineFacts struct {
	LineID, BrandID, BrandName, BrandStatus, BrandAuthorisation, ModelName, ModelStatus, VariantName, VariantStatus string
	// SupplierLinkCurrent: the PO's supplier has a VERIFIED relationship with the
	// line's brand, current by its dates.
	SupplierLinkCurrent bool
}

// poFacts is what submission and approval prerequisites look at.
type poFacts struct {
	SupplierStatus string
	ExpectedDate   *time.Time
	Today          time.Time
	Lines          []lineFacts
}

// prerequisites lists every unmet condition for submitting or approving a PO:
// a verified supplier with a current verified relationship to each brand, and
// brand/model/variant enabled for purchase (BR-002 activation gate). A supplier
// record alone is not proof of authorisation.
func prerequisites(f poFacts) []Check {
	var unmet []Check
	if len(f.Lines) == 0 {
		unmet = append(unmet, Check{Key: "lines", Message: "Add at least one line."})
	}
	switch {
	case f.ExpectedDate == nil:
		unmet = append(unmet, Check{Key: "expected_date", Message: "Set the expected delivery date."})
	case f.ExpectedDate.Before(f.Today):
		unmet = append(unmet, Check{Key: "expected_date", Message: "The expected delivery date is in the past."})
	}
	switch f.SupplierStatus {
	case "VERIFIED":
	case "ARCHIVED":
		unmet = append(unmet, Check{Key: "supplier_verified", Message: "The supplier is archived."})
	default:
		unmet = append(unmet, Check{Key: "supplier_verified", Message: "The supplier is not verified yet (Suppliers)."})
	}
	seenBrand := map[string]bool{}
	for _, l := range f.Lines {
		id := l.LineID
		if !seenBrand[l.BrandID] {
			seenBrand[l.BrandID] = true
			if l.BrandStatus != "ACTIVE" || l.BrandAuthorisation != "AUTHORISED" {
				unmet = append(unmet, Check{Key: "brand_active", LineID: &id,
					Message: "Brand " + l.BrandName + " is not activated for purchase (Catalogue)."})
			}
			if !l.SupplierLinkCurrent {
				unmet = append(unmet, Check{Key: "supplier_brand_link", LineID: &id,
					Message: "The supplier has no current verified relationship with " + l.BrandName + " (Suppliers)."})
			}
		}
		if l.ModelStatus != "ACTIVE" {
			unmet = append(unmet, Check{Key: "model_active", LineID: &id, Message: "Model " + l.ModelName + " is not active."})
		}
		if l.VariantStatus != "ACTIVE" {
			unmet = append(unmet, Check{Key: "variant_active", LineID: &id, Message: "Variant " + l.VariantName + " is not active."})
		}
	}
	return unmet
}

// approvalRequirement decides whether submission needs an approver (PO-001):
// always, unless the owner signed off a threshold, the total is within it and
// (when enabled) the supplier already had an approved PO.
func approvalRequirement(total string, s Settings, supplierHadApprovedPO bool) (bool, string) {
	if s.ApprovalThreshold == nil {
		return true, "No approval threshold is configured, so every purchase order needs an approver."
	}
	if s.FirstTimeSupplierNeedsApproval && !supplierHadApprovedPO {
		return true, "This is the first purchase order for this supplier, which needs an approver."
	}
	t, ok1 := new(big.Rat).SetString(total)
	limit, ok2 := new(big.Rat).SetString(*s.ApprovalThreshold)
	if !ok1 || !ok2 || t.Cmp(limit) > 0 {
		return true, "The total is above the approval threshold of " + *s.ApprovalThreshold + "."
	}
	return false, "The total is within the signed-off approval threshold of " + *s.ApprovalThreshold + "."
}

// lineWarnings flags price variance against the visible snapshot and partial or
// over-receipt (PO-003, PO-007; the tolerance is not configured, so any difference warns).
func lineWarnings(l Line) []string {
	w := []string{}
	if l.CostVariance != nil {
		if v, ok := new(big.Rat).SetString(*l.CostVariance); ok && v.Sign() != 0 {
			w = append(w, WarnPriceVariance)
		}
	}
	switch {
	case l.ReceivedQuantity > l.Quantity:
		w = append(w, WarnOverReceipt)
	case l.ReceivedQuantity > 0 && l.ReceivedQuantity < l.Quantity:
		w = append(w, WarnPartialDelivery)
	}
	return w
}

// actionsFor is the caller's action hint. The creator and the submitter never
// approve their own PO.
func actionsFor(po PurchaseOrder, a access.Access) Actions {
	own := po.CreatedBy == a.UserID || (po.SubmittedBy != nil && *po.SubmittedBy == a.UserID)
	return Actions{
		Edit:    po.Status == StatusDraft && a.Has(access.PermPurchaseOrderManage),
		Submit:  po.Status == StatusDraft && a.Has(access.PermPurchaseOrderManage),
		Approve: po.Status == StatusSubmitted && a.Has(access.PermPurchaseOrderApprove) && !own,
		Return:  po.Status == StatusSubmitted && a.Has(access.PermPurchaseOrderApprove),
		Cancel:  checkTransition(ActionCancelled, po.Status, po.ReceivedQuantity) == nil && a.Has(access.PermPurchaseOrderCancel),
	}
}
