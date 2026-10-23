// Package purchasing is the purchase order lifecycle (D-02; ADR-016): numbered
// purchase orders for one supplier with lines of one variant each, agreed unit
// costs, optional snapshots of the approved catalogue cost price, server-side
// totals and remaining quantities, and the transitions DRAFT -> SUBMITTED ->
// APPROVED (or back to DRAFT, or CANCELLED) with status history and audit.
//
// Submission and approval re-check, inside the transaction, that the supplier is
// verified and has a current verified relationship with every line's brand, and
// that each brand, model and variant is active (the brand activation gate). An
// approval needs purchase_orders.approve and never comes from the PO's creator or
// submitter. Approval is skipped only below a threshold the owner signed off;
// none is configured, so every PO needs an approver. No PO, line or threshold is
// seeded. Goods receipts (D-03) move APPROVED POs onwards.
package purchasing

import (
	"errors"
	"time"
)

// PO statuses (PO-002).
const (
	StatusDraft             = "DRAFT"
	StatusSubmitted         = "SUBMITTED"
	StatusApproved          = "APPROVED"
	StatusPartiallyReceived = "PARTIALLY_RECEIVED"
	StatusReceived          = "RECEIVED"
	StatusClosed            = "CLOSED"
	StatusCancelled         = "CANCELLED"
)

// Statuses lists every status (DB CHECK).
var Statuses = []string{StatusDraft, StatusSubmitted, StatusApproved, StatusPartiallyReceived, StatusReceived, StatusClosed, StatusCancelled}

// Approval bases.
const (
	BasisApprover       = "APPROVER"
	BasisBelowThreshold = "BELOW_THRESHOLD"
)

// History actions (purchase_order_events.action).
const (
	ActionCreated             = "CREATED"
	ActionSubmitted           = "SUBMITTED"
	ActionApproved            = "APPROVED"
	ActionApprovalNotRequired = "APPROVAL_NOT_REQUIRED"
	ActionReturned            = "RETURNED"
	ActionCancelled           = "CANCELLED"
)

// Line warning codes.
const (
	WarnPriceVariance   = "PRICE_VARIANCE"
	WarnPartialDelivery = "PARTIAL_DELIVERY"
	WarnOverReceipt     = "OVER_RECEIPT"
)

// Errors mapped by the HTTP layer.
var (
	ErrNotFound        = errors.New("not found")
	ErrForbidden       = errors.New("not allowed")
	ErrVersionConflict = errors.New("stale version")
	ErrSelfApproval    = errors.New("own purchase order")
)

// FieldProblem is one invalid field or unmet prerequisite.
type FieldProblem struct{ Field, Message string }

// ValidationError lists invalid fields (422).
type ValidationError struct{ Fields []FieldProblem }

func (e *ValidationError) Error() string { return "invalid purchase order input" }

func invalid(field, msg string) *ValidationError {
	return &ValidationError{Fields: []FieldProblem{{field, msg}}}
}

// DuplicateError: a unique value is already used (409).
type DuplicateError struct{ Field string }

func (e *DuplicateError) Error() string { return "duplicate " + e.Field }

// StateError: not allowed in the current state (409).
type StateError struct{ Message string }

func (e *StateError) Error() string { return e.Message }

// BlockedError: submission or approval prerequisites are not met (409).
type BlockedError struct {
	Approval bool // true: approval, false: submission
	Unmet    []Check
}

func (e *BlockedError) Error() string { return "purchase order prerequisites not met" }

// Check is one prerequisite for submitting or approving a PO.
type Check struct {
	Key     string  `json:"key" example:"supplier_verified"`
	Message string  `json:"message"`
	LineID  *string `json:"lineId" extensions:"x-nullable"`
} //@name PurchaseOrderCheck

// PurchaseOrder is a PO header with server-computed totals (list view).
type PurchaseOrder struct {
	ID             string  `json:"id"`
	Number         string  `json:"number" example:"PO-000001"`
	SupplierID     string  `json:"supplierId"`
	SupplierName   string  `json:"supplierName" example:"Example Supplier Private Limited"`
	SupplierStatus string  `json:"supplierStatus" enums:"PENDING_VERIFICATION,VERIFIED,ARCHIVED"`
	Status         string  `json:"status" enums:"DRAFT,SUBMITTED,APPROVED,PARTIALLY_RECEIVED,RECEIVED,CLOSED,CANCELLED" example:"DRAFT"`
	ExpectedDate   *string `json:"expectedDate" extensions:"x-nullable" example:"2026-11-15"`
	Notes          *string `json:"notes" extensions:"x-nullable"`
	// Indian rupees, excluding tax (tax configuration is pending, F-10).
	Currency      string     `json:"currency" example:"INR"`
	CreatedBy     string     `json:"createdBy"`
	CreatedByName string     `json:"createdByName"`
	SubmittedBy   *string    `json:"submittedBy" extensions:"x-nullable"`
	SubmittedAt   *time.Time `json:"submittedAt" extensions:"x-nullable"`
	ApprovalBasis *string    `json:"approvalBasis" enums:"APPROVER,BELOW_THRESHOLD" extensions:"x-nullable"`
	ApprovedBy    *string    `json:"approvedBy" extensions:"x-nullable"`
	ApprovedAt    *time.Time `json:"approvedAt" extensions:"x-nullable"`
	ApprovalNote  *string    `json:"approvalNote" extensions:"x-nullable"`
	// The approval threshold applied at submission; null when none was configured.
	ThresholdApplied *string    `json:"thresholdApplied" extensions:"x-nullable"`
	CancelledBy      *string    `json:"cancelledBy" extensions:"x-nullable"`
	CancelledAt      *time.Time `json:"cancelledAt" extensions:"x-nullable"`
	CancelReason     *string    `json:"cancelReason" extensions:"x-nullable"`
	// Computed by the server from the lines.
	LineCount         int    `json:"lineCount" example:"0"`
	TotalQuantity     int    `json:"totalQuantity" example:"0"`
	ReceivedQuantity  int    `json:"receivedQuantity" example:"0"`
	RemainingQuantity int    `json:"remainingQuantity" example:"0"`
	TotalAmount       string `json:"totalAmount" example:"0.00"`
	// Brands of the lines (names from the catalogue).
	BrandNames []string  `json:"brandNames"`
	Version    int       `json:"version" example:"1"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
} //@name PurchaseOrder

// Line is one variant on a PO. Reference* fields are the catalogue cost
// snapshot; they are null for callers without catalogue.cost_price.view
// (ReferenceHidden is then true when a snapshot exists).
type Line struct {
	ID                   string     `json:"id"`
	LineNo               int        `json:"lineNo" example:"1"`
	BrandID              string     `json:"brandId"`
	BrandName            string     `json:"brandName"`
	BrandStatus          string     `json:"brandStatus" enums:"PENDING_VERIFICATION,ACTIVE,ARCHIVED"`
	ModelID              string     `json:"modelId"`
	ModelName            string     `json:"modelName"`
	ModelStatus          string     `json:"modelStatus" enums:"PENDING_VERIFICATION,ACTIVE,ARCHIVED"`
	VariantID            string     `json:"variantId"`
	VariantName          string     `json:"variantName"`
	VariantStatus        string     `json:"variantStatus" enums:"PENDING_VERIFICATION,ACTIVE,ARCHIVED"`
	Quantity             int        `json:"quantity" example:"1"`
	UnitCost             string     `json:"unitCost" example:"1000.00"`
	LineTotal            string     `json:"lineTotal" example:"1000.00"`
	ReceivedQuantity     int        `json:"receivedQuantity" example:"0"`
	RemainingQuantity    int        `json:"remainingQuantity" example:"1"`
	ReferencePriceListID *string    `json:"referencePriceListId" extensions:"x-nullable"`
	ReferencePriceItemID *string    `json:"referencePriceItemId" extensions:"x-nullable"`
	ReferenceUnitCost    *string    `json:"referenceUnitCost" extensions:"x-nullable"`
	ReferenceCapturedAt  *time.Time `json:"referenceCapturedAt" extensions:"x-nullable"`
	ReferenceHidden      bool       `json:"referenceHidden"`
	// unitCost - referenceUnitCost (null without a visible snapshot).
	CostVariance *string   `json:"costVariance" extensions:"x-nullable" example:"0.00"`
	Note         *string   `json:"note" extensions:"x-nullable"`
	Warnings     []string  `json:"warnings" enums:"PRICE_VARIANCE,PARTIAL_DELIVERY,OVER_RECEIPT"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
} //@name PurchaseOrderLine

// Event is one entry of the status history.
type Event struct {
	ID         string    `json:"id"`
	Action     string    `json:"action" enums:"CREATED,SUBMITTED,APPROVED,APPROVAL_NOT_REQUIRED,RETURNED,CANCELLED"`
	FromStatus *string   `json:"fromStatus" extensions:"x-nullable"`
	ToStatus   string    `json:"toStatus"`
	ActorID    *string   `json:"actorId" extensions:"x-nullable"`
	ActorName  *string   `json:"actorName" extensions:"x-nullable"`
	Note       *string   `json:"note" extensions:"x-nullable"`
	CreatedAt  time.Time `json:"createdAt"`
} //@name PurchaseOrderEvent

// Actions says what the caller may try now (state + permissions). It is a hint
// for the screen; the server checks again on every request.
type Actions struct {
	Edit    bool `json:"edit"`
	Submit  bool `json:"submit"`
	Approve bool `json:"approve"`
	Return  bool `json:"return"`
	Cancel  bool `json:"cancel"`
} //@name PurchaseOrderActions

// Detail is a PO with lines, history, open prerequisites and the caller's actions.
type Detail struct {
	PurchaseOrder
	Lines  []Line  `json:"lines"`
	Events []Event `json:"events"`
	// Unmet prerequisites for the next step (submit for drafts, approve for submitted POs).
	Blockers []Check `json:"blockers"`
	// Whether submission will need an approver under the current settings (drafts only).
	ApprovalRequired *bool   `json:"approvalRequired" extensions:"x-nullable"`
	ApprovalReason   *string `json:"approvalReason" extensions:"x-nullable"`
	Actions          Actions `json:"actions"`
} //@name PurchaseOrderDetail

// Settings are the approval threshold settings (PO-001).
type Settings struct {
	// Null: not configured — every PO needs an approver.
	ApprovalThreshold              *string    `json:"approvalThreshold" extensions:"x-nullable" example:"100000.00"`
	FirstTimeSupplierNeedsApproval bool       `json:"firstTimeSupplierNeedsApproval"`
	ApprovalReference              *string    `json:"approvalReference" extensions:"x-nullable"`
	UpdatedBy                      *string    `json:"updatedBy" extensions:"x-nullable"`
	Version                        int        `json:"version" example:"1"`
	UpdatedAt                      *time.Time `json:"updatedAt" extensions:"x-nullable"`
} //@name PurchaseOrderSettings

// Page is one page of a list.
type Page[T any] struct {
	Items              []T
	Page, Limit, Total int
}

// Actor is the signed-in user.
type Actor struct{ UserID, Username, RequestID string }
