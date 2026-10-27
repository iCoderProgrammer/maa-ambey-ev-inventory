package pricing

import (
	"slices"
	"time"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
)

// retroactiveGrace tolerates small clock differences between browser and server
// when a price is approved to start "now".
const retroactiveGrace = 5 * time.Minute

// viewPermission / editPermission per price type: cost prices keep their
// dedicated permissions (SEC-003).
func viewPermission(priceType string) string {
	if priceType == TypeCost {
		return access.PermCostPriceView
	}
	return access.PermPriceView
}

func editPermission(priceType string) string {
	if priceType == TypeCost {
		return access.PermCostPriceEdit
	}
	return access.PermPriceManage
}

// visibleTypes are the price types the caller may read.
func visibleTypes(a access.Access) []string {
	var out []string
	for _, t := range []string{TypeSelling, TypeCost} {
		if a.Has(viewPermission(t)) {
			out = append(out, t)
		}
	}
	return out
}

// restrictTypes intersects a requested type with the visible ones. Asking for
// a type you may not see is forbidden rather than silently empty.
func restrictTypes(a access.Access, requested string) ([]string, error) {
	visible := visibleTypes(a)
	if len(visible) == 0 {
		return nil, ErrForbidden
	}
	if requested == "" {
		return visible, nil
	}
	if !slices.Contains(visible, requested) {
		return nil, ErrForbidden
	}
	return []string{requested}, nil
}

// approvalFacts is what approval looks at, read under lock.
type approvalFacts struct {
	Item              Item
	ListStatus        string
	BrandStatus       string
	ModelStatus       string
	VariantStatus     string
	VariantConfidence string
	// Superseded is the approved item named by Item.SupersedesID, if any.
	Superseded *Item
	Now        time.Time
}

// checkApproval applies the approval rules (BR-005, ADR-013). Self-approval is
// checked by the caller before (it is a permission matter, 403).
func checkApproval(f approvalFacts) error {
	it := f.Item
	if it.Status != StatusDraft {
		return &StateError{Message: "Only draft prices can be approved."}
	}
	var unmet []FieldProblem
	if f.ListStatus != ListActive {
		unmet = append(unmet, FieldProblem{"price_list", "The price list is archived."})
	}
	if f.BrandStatus == "ARCHIVED" {
		unmet = append(unmet, FieldProblem{"brand", "The brand is archived."})
	}
	if f.ModelStatus == "ARCHIVED" || f.VariantStatus == "ARCHIVED" {
		unmet = append(unmet, FieldProblem{"variant", "The model or variant is archived."})
	}
	if f.VariantConfidence == "UNVERIFIED" {
		unmet = append(unmet, FieldProblem{"variant_verification", "Verify the variant (source and verifier) before approving its price."})
	}
	if len(unmet) > 0 {
		return &BlockedError{Unmet: unmet}
	}
	// No silent retroactive change: an approved price starts now or later.
	if it.EffectiveFrom.Before(f.Now.Add(-retroactiveGrace)) {
		return invalid("effectiveFrom", "is in the past; approved prices cannot start retroactively — set it to now or later")
	}
	if it.EffectiveTo != nil && !it.EffectiveTo.After(f.Now) {
		return invalid("effectiveTo", "is already over; nothing would be priced")
	}
	if s := f.Superseded; s != nil {
		switch {
		case s.Status != StatusApproved:
			return invalid("supersedesId", "must be an approved price")
		case s.SupersededByID != nil:
			return &StateError{Message: "That price has already been superseded."}
		case s.PriceListID != it.PriceListID || s.VariantID != it.VariantID || s.PriceType != it.PriceType:
			return invalid("supersedesId", "must be a price of the same list, variant and type")
		case !it.EffectiveFrom.After(s.EffectiveFrom):
			return invalid("effectiveFrom", "must be after the start of the price it supersedes")
		case s.EffectiveTo != nil && !it.EffectiveFrom.Before(*s.EffectiveTo):
			return invalid("supersedesId", "does not overlap the new price; approve it without superseding")
		}
	}
	return nil
}

// checkDraftEditable: only drafts change.
func checkDraftEditable(status string) error {
	if status != StatusDraft {
		return &StateError{Message: "Only draft prices can be changed. Approved prices are history: draft a superseding price instead."}
	}
	return nil
}
