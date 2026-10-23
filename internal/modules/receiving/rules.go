package receiving

import (
	"strconv"
	"strings"
	"time"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
)

// Receipt rules (PO-003, PO-004, ADR-017). They are pure functions so they can
// be tested without a database; the repository calls them inside the
// transaction after locking the receipt and its purchase order.

// receivablePOStatuses are the PO statuses goods can be received against.
var receivablePOStatuses = []string{"APPROVED", "PARTIALLY_RECEIVED"}

func poReceivable(status string) bool {
	for _, s := range receivablePOStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// checkEditable: only drafts of a receivable PO change.
func checkEditable(status, poStatus string) error {
	if status != StatusDraft {
		return &StateError{Message: "Only draft goods receipts can be changed."}
	}
	if !poReceivable(poStatus) {
		return &StateError{Message: "The purchase order is no longer open for receipts (" + poStatus + "). Cancel this draft instead."}
	}
	return nil
}

// checkCancel: only drafts are cancelled; posted receipts are corrected by a
// reversal workflow (PO-005), never cancelled.
func checkCancel(status string) error {
	switch status {
	case StatusDraft:
		return nil
	case StatusCancelled:
		return &StateError{Message: "This goods receipt is already cancelled."}
	default:
		return &StateError{Message: "Posted goods receipts cannot be cancelled; they are corrected by a reversal."}
	}
}

// poLineAvailability is one PO line's quantities as seen by one receipt.
type poLineAvailability struct {
	ID          string
	LineNo      int
	Ordered     int
	Posted      int // purchase_order_lines.received_quantity
	OtherDrafts int // received on other DRAFT receipts of the PO
}

// available is what this receipt may take without an override.
func (a poLineAvailability) available() int {
	return max(a.Ordered-a.Posted-a.OtherDrafts, 0)
}

// storedOverride is an over-receipt authorisation already on a draft line.
type storedOverride struct {
	Quantity int
	Reason   string
	By       string
	At       time.Time
}

// override is the authorisation a saved line will carry (nil: none).
type override struct {
	Quantity int
	Reason   string
	By       string
	// At is nil for a new authorisation (the database time is used).
	At *time.Time
}

// lineDecision is a validated input line with the authorisation it keeps or gets.
type lineDecision struct {
	indexedLine
	Override *override
	// NewOverride: authorised by this request (audited).
	NewOverride bool
	Excess      int
}

// decideOverReceipt checks every line against its PO line's availability. A
// line above what is available needs an authorisation: a reason sent now by a
// holder of goods_receipts.over_receipt, or an earlier one on this draft that
// still covers the received quantity. Without one the whole save is refused
// (nothing is stored); a reason from a caller without the permission is
// refused as forbidden. Lines within availability drop any earlier authorisation.
func decideOverReceipt(lines []indexedLine, avail map[string]poLineAvailability, existing map[string]storedOverride,
	canOverride bool, actorID string) ([]lineDecision, error) {
	var unknown, needs, forbidden []FieldProblem
	out := make([]lineDecision, 0, len(lines))
	for _, l := range lines {
		a, ok := avail[l.PurchaseOrderLineID]
		if !ok {
			unknown = append(unknown, FieldProblem{l.path("purchaseOrderLineId"), "is not a line of this purchase order"})
			continue
		}
		d := lineDecision{indexedLine: l, Excess: max(l.ReceivedQuantity-a.available(), 0)}
		if d.Excess > 0 {
			prev, hasPrev := existing[l.PurchaseOrderLineID]
			switch {
			case l.OverReceiptReason != nil && !canOverride:
				forbidden = append(forbidden, FieldProblem{l.path("overReceiptReason"),
					"needs the goods_receipts.over_receipt permission; ask an authorised person to approve the extra units"})
			case l.OverReceiptReason != nil:
				d.Override = &override{Quantity: l.ReceivedQuantity, Reason: *l.OverReceiptReason, By: actorID}
				d.NewOverride = true
			case hasPrev && prev.Quantity >= l.ReceivedQuantity:
				at := prev.At
				d.Override = &override{Quantity: prev.Quantity, Reason: prev.Reason, By: prev.By, At: &at}
			default:
				needs = append(needs, FieldProblem{l.path("receivedQuantity"), overReceiptMessage(a, l.ReceivedQuantity)})
			}
		}
		out = append(out, d)
	}
	switch {
	case len(unknown) > 0:
		return nil, &ValidationError{Fields: unknown}
	case len(forbidden) > 0:
		return nil, &OverReceiptForbiddenError{Fields: forbidden}
	case len(needs) > 0:
		return nil, &OverReceiptError{Fields: needs}
	}
	return out, nil
}

func overReceiptMessage(a poLineAvailability, received int) string {
	msg := "only " + strconv.Itoa(a.available()) + " left on PO line " + strconv.Itoa(a.LineNo) +
		" (ordered " + strconv.Itoa(a.Ordered) + ", posted " + strconv.Itoa(a.Posted)
	if a.OtherDrafts > 0 {
		msg += ", on other draft receipts " + strconv.Itoa(a.OtherDrafts)
	}
	return msg + "); receiving " + strconv.Itoa(received) + " needs an authorised over-receipt with a reason"
}

// lineWarnings flags what a reviewer should look at before posting.
func lineWarnings(l Line) []string {
	w := []string{}
	if l.ReceivedQuantity == 0 {
		return w
	}
	if l.ExcessQuantity > 0 {
		if l.OverReceipt != nil && l.OverReceipt.Quantity >= l.ReceivedQuantity {
			w = append(w, WarnOverReceipt)
		} else {
			w = append(w, WarnOverReceiptUnauthorised)
		}
	}
	if l.AvailableQuantity-l.ReceivedQuantity > 0 {
		w = append(w, WarnPartialDelivery)
	}
	incomplete := len(l.Units) < l.ReceivedQuantity
	for _, u := range l.Units {
		if u.ChassisNumber == nil {
			incomplete = true
		}
	}
	if incomplete {
		w = append(w, WarnIdentifiersIncomplete)
	}
	if l.RejectedQuantity > 0 && l.DiscrepancyNote == nil {
		w = append(w, WarnRejectionNotExplained)
	}
	return w
}

// unitWarnings marks units without a chassis/frame number and serials that
// repeat on the receipt (warning only: their uniqueness is not yet a rule).
func unitWarnings(lines []Line) {
	counts := map[string]int{}
	key := func(kind string, s *string) string {
		if s == nil {
			return ""
		}
		return kind + ":" + strings.ToUpper(*s)
	}
	for _, l := range lines {
		for _, u := range l.Units {
			for _, k := range []string{key("m", u.MotorSerial), key("b", u.BatterySerial), key("c", u.ChargerSerial)} {
				if k != "" {
					counts[k]++
				}
			}
		}
	}
	for i := range lines {
		for j := range lines[i].Units {
			u := &lines[i].Units[j]
			u.Warnings = []string{}
			if u.ChassisNumber == nil {
				u.Warnings = append(u.Warnings, WarnChassisMissing)
			}
			if counts[key("m", u.MotorSerial)] > 1 {
				u.Warnings = append(u.Warnings, WarnDuplicateMotorSerial)
			}
			if counts[key("b", u.BatterySerial)] > 1 {
				u.Warnings = append(u.Warnings, WarnDuplicateBatterySerial)
			}
			if counts[key("c", u.ChargerSerial)] > 1 {
				u.Warnings = append(u.Warnings, WarnDuplicateChargerSerial)
			}
		}
	}
}

// actionsFor says what the caller may try on the receipt now.
func actionsFor(d Detail, a access.Access) Actions {
	manage := a.Has(access.PermGoodsReceiptManage)
	draft := d.Status == StatusDraft
	return Actions{
		Edit:                 draft && manage && d.PurchaseOrderReceivable,
		Cancel:               draft && manage,
		AddDocument:          draft && manage && a.Has(access.PermAttachmentsUpload),
		AuthoriseOverReceipt: draft && manage && a.Has(access.PermGoodsReceiptOverReceipt),
	}
}
