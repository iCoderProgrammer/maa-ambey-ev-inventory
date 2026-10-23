// Package receiving is the goods receipt draft workflow (D-03; ADR-017): a
// numbered goods receipt note (GRN) against one approved purchase order, with
// per PO line the delivered, accepted and rejected quantities and a discrepancy
// note, per physical unit the identifiers as captured, and the supplier's
// documents as private attachments.
//
// A draft changes no stock. It creates no vehicle unit or stock movement and
// leaves the PO's received quantities and status alone; posting (D-07) does
// that atomically. While drafts are open, each PO line's available quantity is
// ordered - posted - what other open drafts hold, computed under a lock on the
// PO so two drafts cannot both take the last units. Receiving more than that
// needs goods_receipts.over_receipt and a reason (PO-003); without it the save
// is refused. Identifier rules per brand are pending (OQ-4): identifiers are
// optional in a draft, and a chassis/frame number appears once per receipt and
// once across open receipts. No receipt, unit or grant is seeded.
package receiving

import (
	"errors"
	"time"
)

// Receipt statuses. POSTED is set by posting (D-07), never by this package.
const (
	StatusDraft     = "DRAFT"
	StatusPosted    = "POSTED"
	StatusCancelled = "CANCELLED"
)

// Statuses lists every status (DB CHECK).
var Statuses = []string{StatusDraft, StatusPosted, StatusCancelled}

// Unit conditions.
const (
	ConditionAccepted = "ACCEPTED"
	ConditionRejected = "REJECTED"
)

// Conditions lists the unit conditions (DB CHECK).
var Conditions = []string{ConditionAccepted, ConditionRejected}

// Document types: proposal pending confirmation (OQ-25); keep in step with the
// goods_receipt_documents_type_valid CHECK in migration 000012.
var DocumentTypes = []string{"DELIVERY_CHALLAN", "SUPPLIER_INVOICE", "PHOTO", "OTHER"}

// History actions (goods_receipt_events.action).
const (
	ActionCreated   = "CREATED"
	ActionCancelled = "CANCELLED"
)

// Line warning codes.
const (
	// More than the PO line has left, authorised with a reason.
	WarnOverReceipt = "OVER_RECEIPT"
	// More than the PO line has left and no valid authorisation (posting will refuse it).
	WarnOverReceiptUnauthorised = "OVER_RECEIPT_NOT_AUTHORISED"
	// The PO line still has units to come after this receipt.
	WarnPartialDelivery = "PARTIAL_DELIVERY"
	// Fewer units captured than received, or a unit without a chassis/frame number.
	WarnIdentifiersIncomplete = "IDENTIFIERS_INCOMPLETE"
	// Rejected units without a discrepancy note.
	WarnRejectionNotExplained = "REJECTION_NOT_EXPLAINED"
)

// Unit warning codes. Motor, battery and charger serial uniqueness is not yet
// an approved rule (INV-003, OQ-4): duplicates warn, they do not block.
const (
	WarnChassisMissing         = "CHASSIS_MISSING"
	WarnDuplicateMotorSerial   = "DUPLICATE_MOTOR_SERIAL"
	WarnDuplicateBatterySerial = "DUPLICATE_BATTERY_SERIAL"
	WarnDuplicateChargerSerial = "DUPLICATE_CHARGER_SERIAL"
)

// Errors mapped by the HTTP layer.
var (
	ErrNotFound        = errors.New("not found")
	ErrForbidden       = errors.New("not allowed")
	ErrVersionConflict = errors.New("stale version")
)

// FieldProblem is one invalid field.
type FieldProblem struct{ Field, Message string }

// ValidationError lists invalid fields (422).
type ValidationError struct{ Fields []FieldProblem }

func (e *ValidationError) Error() string { return "invalid goods receipt input" }

func invalid(field, msg string) *ValidationError {
	return &ValidationError{Fields: []FieldProblem{{field, msg}}}
}

// DuplicateError: identifiers or a unique value already used (409). Fields
// names each offending input path.
type DuplicateError struct{ Fields []FieldProblem }

func (e *DuplicateError) Error() string { return "duplicate goods receipt value" }

// StateError: not allowed in the current state (409).
type StateError struct{ Message string }

func (e *StateError) Error() string { return e.Message }

// OverReceiptError: lines receive more than the PO has left and carry no
// authorisation (409 OVER_RECEIPT_REQUIRES_OVERRIDE). Nothing was saved.
type OverReceiptError struct{ Fields []FieldProblem }

func (e *OverReceiptError) Error() string { return "over-receipt needs an authorised override" }

// OverReceiptForbiddenError: an override reason was sent by a caller without
// goods_receipts.over_receipt (403 OVER_RECEIPT_NOT_AUTHORISED). Audited DENIED.
type OverReceiptForbiddenError struct{ Fields []FieldProblem }

func (e *OverReceiptForbiddenError) Error() string { return "over-receipt override not permitted" }

// Receipt is a GRN header with server-computed totals (list view).
type Receipt struct {
	ID                  string     `json:"id"`
	Number              string     `json:"number" example:"GRN-000001"`
	Status              string     `json:"status" enums:"DRAFT,POSTED,CANCELLED" example:"DRAFT"`
	PurchaseOrderID     string     `json:"purchaseOrderId"`
	PurchaseOrderNumber string     `json:"purchaseOrderNumber" example:"PO-000001"`
	PurchaseOrderStatus string     `json:"purchaseOrderStatus" enums:"DRAFT,SUBMITTED,APPROVED,PARTIALLY_RECEIVED,RECEIVED,CLOSED,CANCELLED"`
	SupplierID          string     `json:"supplierId"`
	SupplierName        string     `json:"supplierName" example:"Example Supplier Private Limited"`
	ReceivedDate        string     `json:"receivedDate" example:"2026-10-23"`
	SupplierReference   *string    `json:"supplierReference" extensions:"x-nullable" example:"DC-1234"`
	Notes               *string    `json:"notes" extensions:"x-nullable"`
	CreatedBy           string     `json:"createdBy"`
	CreatedByName       string     `json:"createdByName"`
	CancelledBy         *string    `json:"cancelledBy" extensions:"x-nullable"`
	CancelledAt         *time.Time `json:"cancelledAt" extensions:"x-nullable"`
	CancelReason        *string    `json:"cancelReason" extensions:"x-nullable"`
	PostedBy            *string    `json:"postedBy" extensions:"x-nullable"`
	PostedAt            *time.Time `json:"postedAt" extensions:"x-nullable"`
	// Computed by the server from the lines and units.
	LineCount        int       `json:"lineCount" example:"0"`
	ReceivedQuantity int       `json:"receivedQuantity" example:"0"`
	AcceptedQuantity int       `json:"acceptedQuantity" example:"0"`
	RejectedQuantity int       `json:"rejectedQuantity" example:"0"`
	UnitCount        int       `json:"unitCount" example:"0"`
	Version          int       `json:"version" example:"1"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
} //@name GoodsReceipt

// OverReceipt is the authorisation to receive more than the PO line has left.
type OverReceipt struct {
	// Received quantity the authorisation covers.
	Quantity         int       `json:"quantity" example:"3"`
	Reason           string    `json:"reason"`
	AuthorisedBy     string    `json:"authorisedBy"`
	AuthorisedByName string    `json:"authorisedByName"`
	AuthorisedAt     time.Time `json:"authorisedAt"`
} //@name GoodsReceiptOverReceipt

// Unit is one physical unit as received.
type Unit struct {
	ID                string   `json:"id"`
	UnitNo            int      `json:"unitNo" example:"1"`
	Condition         string   `json:"condition" enums:"ACCEPTED,REJECTED"`
	ChassisNumber     *string  `json:"chassisNumber" extensions:"x-nullable"`
	ChassisNormalised *string  `json:"chassisNormalised" extensions:"x-nullable"`
	MotorSerial       *string  `json:"motorSerial" extensions:"x-nullable"`
	BatterySerial     *string  `json:"batterySerial" extensions:"x-nullable"`
	ChargerSerial     *string  `json:"chargerSerial" extensions:"x-nullable"`
	Note              *string  `json:"note" extensions:"x-nullable"`
	Warnings          []string `json:"warnings" enums:"CHASSIS_MISSING,DUPLICATE_MOTOR_SERIAL,DUPLICATE_BATTERY_SERIAL,DUPLICATE_CHARGER_SERIAL"`
} //@name GoodsReceiptUnit

// Line is one PO line as seen from this receipt. Every line of the PO is
// listed; ID is null when nothing of it is on this receipt.
type Line struct {
	ID                  *string `json:"id" extensions:"x-nullable"`
	PurchaseOrderLineID string  `json:"purchaseOrderLineId"`
	LineNo              int     `json:"lineNo" example:"1"`
	BrandID             string  `json:"brandId"`
	BrandName           string  `json:"brandName"`
	ModelName           string  `json:"modelName"`
	VariantName         string  `json:"variantName"`
	OrderedQuantity     int     `json:"orderedQuantity" example:"5"`
	// Already posted against the PO line (goods receipts posted by D-07).
	PostedQuantity int `json:"postedQuantity" example:"0"`
	// Held by other open draft receipts of the same PO.
	OtherDraftsQuantity int `json:"otherDraftsQuantity" example:"0"`
	// ordered - posted - other drafts (never below 0): what this receipt may take without an override.
	AvailableQuantity int `json:"availableQuantity" example:"5"`
	ReceivedQuantity  int `json:"receivedQuantity" example:"0"`
	AcceptedQuantity  int `json:"acceptedQuantity" example:"0"`
	RejectedQuantity  int `json:"rejectedQuantity" example:"0"`
	// received - available when positive.
	ExcessQuantity  int          `json:"excessQuantity" example:"0"`
	OverReceipt     *OverReceipt `json:"overReceipt" extensions:"x-nullable"`
	DiscrepancyNote *string      `json:"discrepancyNote" extensions:"x-nullable"`
	Units           []Unit       `json:"units"`
	Warnings        []string     `json:"warnings" enums:"OVER_RECEIPT,OVER_RECEIPT_NOT_AUTHORISED,PARTIAL_DELIVERY,IDENTIFIERS_INCOMPLETE,REJECTION_NOT_EXPLAINED"`
} //@name GoodsReceiptLine

// Document is a supplier document or photo linked to the receipt.
type Document struct {
	ID           string    `json:"id"`
	AttachmentID string    `json:"attachmentId"`
	DocumentType string    `json:"documentType" enums:"DELIVERY_CHALLAN,SUPPLIER_INVOICE,PHOTO,OTHER"`
	Note         *string   `json:"note" extensions:"x-nullable"`
	FileName     string    `json:"fileName"`
	ContentType  string    `json:"contentType" example:"application/pdf"`
	AddedBy      string    `json:"addedBy"`
	AddedByName  string    `json:"addedByName"`
	CreatedAt    time.Time `json:"createdAt"`
} //@name GoodsReceiptDocument

// Event is one entry of the status history.
type Event struct {
	ID         string    `json:"id"`
	Action     string    `json:"action" enums:"CREATED,CANCELLED,POSTED"`
	FromStatus *string   `json:"fromStatus" extensions:"x-nullable"`
	ToStatus   string    `json:"toStatus"`
	ActorID    *string   `json:"actorId" extensions:"x-nullable"`
	ActorName  *string   `json:"actorName" extensions:"x-nullable"`
	Note       *string   `json:"note" extensions:"x-nullable"`
	CreatedAt  time.Time `json:"createdAt"`
} //@name GoodsReceiptEvent

// Actions says what the caller may try now (state + permissions). It is a hint
// for the screen; the server checks again on every request.
type Actions struct {
	Edit        bool `json:"edit"`
	Cancel      bool `json:"cancel"`
	AddDocument bool `json:"addDocument"`
	// The caller may authorise over-receipt (goods_receipts.over_receipt).
	AuthoriseOverReceipt bool `json:"authoriseOverReceipt"`
	// Posting changes stock and is a separate step (D-07); always false here.
	Post bool `json:"post"`
} //@name GoodsReceiptActions

// Detail is a receipt with every PO line, its units, documents, history and
// the caller's actions.
type Detail struct {
	Receipt
	// The PO accepts receipts (APPROVED or PARTIALLY_RECEIVED).
	PurchaseOrderReceivable bool       `json:"purchaseOrderReceivable"`
	Lines                   []Line     `json:"lines"`
	Documents               []Document `json:"documents"`
	Events                  []Event    `json:"events"`
	// A draft changes no stock: always false until the receipt is posted.
	StockPosted bool    `json:"stockPosted"`
	Actions     Actions `json:"actions"`
} //@name GoodsReceiptDetail

// ReceivablePO is a purchase order goods can be received against.
type ReceivablePO struct {
	ID                string   `json:"id"`
	Number            string   `json:"number" example:"PO-000001"`
	Status            string   `json:"status" enums:"APPROVED,PARTIALLY_RECEIVED"`
	SupplierID        string   `json:"supplierId"`
	SupplierName      string   `json:"supplierName"`
	ExpectedDate      *string  `json:"expectedDate" extensions:"x-nullable" example:"2026-11-15"`
	OrderedQuantity   int      `json:"orderedQuantity" example:"5"`
	PostedQuantity    int      `json:"postedQuantity" example:"0"`
	DraftQuantity     int      `json:"draftQuantity" example:"0"`
	AvailableQuantity int      `json:"availableQuantity" example:"5"`
	OpenDraftCount    int      `json:"openDraftCount" example:"0"`
	BrandNames        []string `json:"brandNames"`
} //@name GoodsReceiptPurchaseOrder

// Page is one page of a list.
type Page[T any] struct {
	Items              []T
	Page, Limit, Total int
}

// Actor is the signed-in user.
type Actor struct{ UserID, Username, RequestID string }
