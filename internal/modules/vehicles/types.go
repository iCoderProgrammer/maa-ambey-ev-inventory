// Package vehicles is the vehicle unit registry (D-04; ADR-018): one record per
// physical scooter (INV-001) with its internal unit number, verified catalogue
// variant (and so brand and model), identifiers as captured with their
// normalised form, location, lifecycle status, the supplier, purchase order and
// goods receipt it came from, and its acquisition cost.
//
// Units are read through the API (vehicle_units.view); the cost needs
// vehicle_units.cost.view and is left out otherwise. There is no create, edit or
// delete route: the only way in is RegisterFromReceiptUnit, called by receipt
// posting (D-07) inside its transaction for an accepted unit of a POSTED goods
// receipt, and the database refuses a unit whose receipt is not posted at
// commit. Status and location change only through the stock movement ledger
// (package stock, E-01, INV-009): the database refuses every other update. A normalised
// chassis/frame number belongs to one unit only (D-05: checked before the
// insert and enforced by the vehicle_units_chassis_key index); motor, battery
// and charger numbers have no approved uniqueness policy yet, and identifier
// formats per brand are pending (OQ-4, OQ-26, OQ-30). No unit, identifier or
// grant is seeded.
package vehicles

import (
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/ageing"

	"errors"
	"slices"
	"strings"
	"time"
)

// Lifecycle statuses (INV-005; DB CHECK vehicle_units_status_valid). Which
// transitions are allowed, and to whom, is decided with the stock movement
// service (E-01) and the PDI, reservation and sales tasks.
const (
	StatusOrdered                 = "ORDERED"
	StatusInTransit               = "IN_TRANSIT"
	StatusReceivedPendingPDI      = "RECEIVED_PENDING_PDI"
	StatusQualityHold             = "QUALITY_HOLD"
	StatusAvailable               = "AVAILABLE"
	StatusDisplay                 = "DISPLAY"
	StatusReserved                = "RESERVED"
	StatusServiceHold             = "SERVICE_HOLD"
	StatusSold                    = "SOLD"
	StatusDelivered               = "DELIVERED"
	StatusRejected                = "REJECTED"
	StatusReturnPendingInspection = "RETURN_PENDING_INSPECTION"
	StatusReturnedToSupplier      = "RETURNED_TO_SUPPLIER"
)

// IsSaleable reports whether a unit in status may be newly reserved or sold:
// AVAILABLE only (INV-006). DISPLAY needs an authorised saleable transition
// first; RECEIVED_PENDING_PDI, QUALITY_HOLD, SERVICE_HOLD, REJECTED, SOLD and
// DELIVERED are never saleable stock (INV-007, PO-006).
func IsSaleable(status string) bool { return slices.Contains(SaleableStatuses, status) }

// SaleableStatuses are the statuses IsSaleable accepts.
var SaleableStatuses = []string{StatusAvailable}

// Statuses lists every status in the approved order (INV-005).
var Statuses = []string{StatusOrdered, StatusInTransit, StatusReceivedPendingPDI, StatusQualityHold, StatusAvailable,
	StatusDisplay, StatusReserved, StatusServiceHold, StatusSold, StatusDelivered, StatusRejected,
	StatusReturnPendingInspection, StatusReturnedToSupplier}

// Source types (DB CHECK vehicle_units_source_valid).
const (
	SourceGoodsReceipt = "GOODS_RECEIPT"
	// SourceOpeningStock: created by a committed opening-stock import batch (G-07).
	SourceOpeningStock = "OPENING_STOCK"
)

// History actions (vehicle_unit_events.action).
const ActionRegistered = "REGISTERED"

// NormaliseIdentifier is the comparison form of a chassis/frame number or
// serial: trimmed and upper-cased (INV-002). It matches the database's
// generated *_normalised columns and goods_receipt_units.chassis_normalised.
// Which separators may also be ignored is pending (OQ-26); D-05 builds its
// uniqueness checks on this form.
func NormaliseIdentifier(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }

// Unit is a vehicle unit as listed. unitCost is null and costHidden true for
// callers without vehicle_units.cost.view.
type Unit struct {
	ID                string  `json:"id" example:"5f0c6f3e-8a52-4d7a-9a51-0d9e3c1b2a77"`
	UnitNumber        string  `json:"unitNumber" example:"MAE-VEH-000001"`
	Status            string  `json:"status" example:"RECEIVED_PENDING_PDI" enums:"ORDERED,IN_TRANSIT,RECEIVED_PENDING_PDI,QUALITY_HOLD,AVAILABLE,DISPLAY,RESERVED,SERVICE_HOLD,SOLD,DELIVERED,REJECTED,RETURN_PENDING_INSPECTION,RETURNED_TO_SUPPLIER"`
	BrandID           string  `json:"brandId"`
	BrandName         string  `json:"brandName" example:"Brand A"`
	ModelID           string  `json:"modelId"`
	ModelName         string  `json:"modelName" example:"Model 1"`
	VariantID         string  `json:"variantId"`
	VariantName       string  `json:"variantName" example:"Standard"`
	ChassisNumber     *string `json:"chassisNumber" extensions:"x-nullable" example:"CH-0001"`
	ChassisNormalised *string `json:"chassisNormalised" extensions:"x-nullable" example:"CH-0001"`
	MotorSerial       *string `json:"motorSerial" extensions:"x-nullable"`
	BatterySerial     *string `json:"batterySerial" extensions:"x-nullable"`
	ChargerSerial     *string `json:"chargerSerial" extensions:"x-nullable"`
	LocationID        *string `json:"locationId" extensions:"x-nullable"`
	LocationCode      *string `json:"locationCode" extensions:"x-nullable" example:"SR1"`
	LocationName      *string `json:"locationName" extensions:"x-nullable"`
	SourceType        string  `json:"sourceType" example:"GOODS_RECEIPT" enums:"GOODS_RECEIPT,OPENING_STOCK"`
	SupplierID        *string `json:"supplierId" extensions:"x-nullable"`
	SupplierName      *string `json:"supplierName" extensions:"x-nullable"`
	ReceivedDate      string  `json:"receivedDate" example:"2026-10-23"`
	// AgeDays is the stock age in whole days from the approved origin date to the
	// business date (E-07, ADR-029) — the same value as ageing.ageDays; null when the
	// unit has no reliable origin date (never a fabricated age).
	AgeDays *int `json:"ageDays" extensions:"x-nullable" example:"12"`
	// Ageing is the unit's stock ageing (E-07): origin date and source, age, band, or
	// why no reliable origin date exists. Calculated by the server on every read.
	Ageing ageing.Result `json:"ageing"`
	// grnStatus/grnDate: the unit's goods receipt as read (origin check).
	grnStatus, grnDate *string
	// openingDateApproved: an opening-stock unit's batch approval says its received
	// dates are the agreed opening-stock dates (origin check, G-07).
	openingDateApproved *bool
	// UnitCost is the acquisition cost (excluding tax); null without vehicle_units.cost.view.
	UnitCost     *string   `json:"unitCost" extensions:"x-nullable" example:"95000.00"`
	CostCurrency *string   `json:"costCurrency" extensions:"x-nullable" example:"INR"`
	CostHidden   bool      `json:"costHidden"`
	Version      int       `json:"version" example:"1"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
} //@name VehicleUnit

// Source links a unit to the documents it came from. Each referenced record is
// opened through its own API and permission (purchase_orders.view,
// goods_receipts.view; the receipt's documents also need attachments.view).
type Source struct {
	Type                string  `json:"type" example:"GOODS_RECEIPT" enums:"GOODS_RECEIPT,OPENING_STOCK"`
	SupplierID          *string `json:"supplierId" extensions:"x-nullable"`
	SupplierName        *string `json:"supplierName" extensions:"x-nullable"`
	PurchaseOrderID     *string `json:"purchaseOrderId" extensions:"x-nullable"`
	PurchaseOrderNumber *string `json:"purchaseOrderNumber" extensions:"x-nullable" example:"PO-000001"`
	PurchaseOrderLineID *string `json:"purchaseOrderLineId" extensions:"x-nullable"`
	GoodsReceiptID      *string `json:"goodsReceiptId" extensions:"x-nullable"`
	GoodsReceiptNumber  *string `json:"goodsReceiptNumber" extensions:"x-nullable" example:"GRN-000001"`
	GoodsReceiptLineID  *string `json:"goodsReceiptLineId" extensions:"x-nullable"`
	GoodsReceiptUnitID  *string `json:"goodsReceiptUnitId" extensions:"x-nullable"`
	SupplierReference   *string `json:"supplierReference" extensions:"x-nullable" example:"DC-1234"`
	ReceivedDate        string  `json:"receivedDate" example:"2026-10-23"`
	// Opening stock (G-07): the import batch, its OSI reference and the file line the unit came from.
	OpeningStockBatchID        *string `json:"openingStockBatchId" extensions:"x-nullable"`
	OpeningStockBatchReference *string `json:"openingStockBatchReference" extensions:"x-nullable" example:"OSI-000001"`
	OpeningStockRowNumber      *int    `json:"openingStockRowNumber" extensions:"x-nullable" example:"3"`
} //@name VehicleUnitSource

// Event is one entry of a unit's append-only history, pointing at its source document.
type Event struct {
	ID              string    `json:"id"`
	Action          string    `json:"action" example:"REGISTERED" enums:"REGISTERED"`
	ToStatus        string    `json:"toStatus" example:"RECEIVED_PENDING_PDI"`
	SourceType      string    `json:"sourceType" example:"GOODS_RECEIPT" enums:"GOODS_RECEIPT,OPENING_STOCK"`
	SourceID        string    `json:"sourceId"`
	SourceReference string    `json:"sourceReference" example:"GRN-000001"`
	ActorID         string    `json:"actorId"`
	ActorName       *string   `json:"actorName" extensions:"x-nullable"`
	Note            *string   `json:"note" extensions:"x-nullable"`
	CreatedAt       time.Time `json:"createdAt"`
} //@name VehicleUnitEvent

// Document sources and the document type of PDI evidence.
const (
	DocumentSourceGoodsReceipt  = "GOODS_RECEIPT"
	DocumentSourcePDIInspection = "PDI_INSPECTION"
	DocumentTypePDIEvidence     = "PDI_EVIDENCE"
)

// Document is the metadata of a private file about the unit: a document of its
// goods receipt or evidence of one of its PDI inspections. Only files the caller
// may open are listed (the attachment rules: attachments.view, the file's
// location in scope, and goods_receipts.view for receipt documents); the file
// itself is fetched through GET /attachments/{id}/link or /content, which check
// again. No storage key or URL is returned.
type Document struct {
	AttachmentID    string    `json:"attachmentId"`
	SourceType      string    `json:"sourceType" enums:"GOODS_RECEIPT,PDI_INSPECTION" example:"GOODS_RECEIPT"`
	SourceID        string    `json:"sourceId"`
	SourceReference string    `json:"sourceReference" example:"GRN-000001"`
	DocumentType    string    `json:"documentType" enums:"DELIVERY_CHALLAN,SUPPLIER_INVOICE,PHOTO,OTHER,PDI_EVIDENCE" example:"DELIVERY_CHALLAN"`
	FileName        string    `json:"fileName" example:"challan.pdf"`
	ContentType     string    `json:"contentType" example:"application/pdf"`
	SizeBytes       int64     `json:"sizeBytes" example:"48211"`
	CreatedAt       time.Time `json:"createdAt"`
	// Owner checks (not serialised): the attachment rules decide visibility.
	ownerType, uploadedBy, locationID string
} //@name VehicleUnitDocument

// Detail is a unit with its source, registration history and permitted documents.
// Status changes after registration are its stock movements
// (GET /vehicle-units/{id}/stock-movements).
type Detail struct {
	Unit
	Source Source  `json:"source"`
	Events []Event `json:"events"`
	// Documents the caller may open; empty without attachments.view, and then documentsHidden is true.
	Documents       []Document `json:"documents"`
	DocumentsHidden bool       `json:"documentsHidden"`
} //@name VehicleUnitDetail

// Movement types and sources of a vehicle unit's stock movements (E-01, D-06).
var (
	MovementTypes       = []string{"RECEIPT", "PDI_PASS", "QUALITY_HOLD", "REJECT", "DISPLAY_ON", "DISPLAY_OFF", "OPENING_BALANCE"}
	MovementSourceTypes = []string{"GOODS_RECEIPT", "PDI_INSPECTION", "STAFF_ACTION", "OPENING_STOCK"}
)

// Movement is one entry of a unit's append-only stock movement ledger (E-01),
// read-only. The source names what caused it: its goods receipt, its PDI
// inspection, or a staff action through the API. Location code and name are
// null for a location outside the caller's scope.
type Movement struct {
	ID               string    `json:"id"`
	MovementType     string    `json:"movementType" enums:"RECEIPT,PDI_PASS,QUALITY_HOLD,REJECT,DISPLAY_ON,DISPLAY_OFF,OPENING_BALANCE" example:"RECEIPT"`
	FromStatus       *string   `json:"fromStatus" extensions:"x-nullable"`
	ToStatus         string    `json:"toStatus" example:"RECEIVED_PENDING_PDI"`
	FromLocationID   *string   `json:"fromLocationId" extensions:"x-nullable"`
	FromLocationCode *string   `json:"fromLocationCode" extensions:"x-nullable"`
	FromLocationName *string   `json:"fromLocationName" extensions:"x-nullable"`
	ToLocationID     *string   `json:"toLocationId" extensions:"x-nullable"`
	ToLocationCode   *string   `json:"toLocationCode" extensions:"x-nullable" example:"SR1"`
	ToLocationName   *string   `json:"toLocationName" extensions:"x-nullable"`
	SourceType       string    `json:"sourceType" enums:"GOODS_RECEIPT,PDI_INSPECTION,STAFF_ACTION,OPENING_STOCK" example:"GOODS_RECEIPT"`
	SourceID         *string   `json:"sourceId" extensions:"x-nullable"`
	SourceReference  *string   `json:"sourceReference" extensions:"x-nullable" example:"GRN-000001"`
	ActorID          string    `json:"actorId"`
	ActorName        *string   `json:"actorName" extensions:"x-nullable"`
	Note             *string   `json:"note" extensions:"x-nullable"`
	CreatedAt        time.Time `json:"createdAt"`
} //@name VehicleUnitMovement

// Page is one page of a list.
type Page[T any] struct {
	Items              []T
	Page, Limit, Total int
}

// FieldProblem is one invalid input field.
type FieldProblem struct {
	Field   string
	Message string
}

// ValidationError lists invalid fields (422).
type ValidationError struct{ Fields []FieldProblem }

func (e *ValidationError) Error() string { return "invalid vehicle unit request" }

// Errors the HTTP adapter maps to responses.
var (
	ErrForbidden = errors.New("forbidden")
	ErrNotFound  = errors.New("vehicle unit not found")
)

// Errors RegisterFromReceiptUnit returns to its caller (receipt posting).
var (
	// ErrReceiptUnitNotFound: no captured goods receipt unit with that ID.
	ErrReceiptUnitNotFound = errors.New("goods receipt unit not found")
	// ErrReceiptUnitNotAccepted: only ACCEPTED captured units become vehicle units.
	ErrReceiptUnitNotAccepted = errors.New("goods receipt unit was not accepted")
	// ErrAlreadyRegistered: the captured unit already has a vehicle unit.
	ErrAlreadyRegistered = errors.New("goods receipt unit is already a vehicle unit")
	// ErrReceiptNotPosted: the goods receipt is not POSTED when the transaction commits.
	ErrReceiptNotPosted = errors.New("goods receipt is not posted")
	// ErrInvalidLocation: the location does not exist.
	ErrInvalidLocation = errors.New("unknown location")
	// ErrDuplicateIdentifier is matched (errors.Is) by every *DuplicateIdentifierError.
	ErrDuplicateIdentifier = errors.New("vehicle identifier already registered")
)

// Identifier fields a DuplicateIdentifierError can name (the JSON names of the
// captured unit's identifiers). Only the chassis/frame number is unique so far:
// motor, battery and charger numbers have no approved uniqueness policy (OQ-4, OQ-30).
const FieldChassisNumber = "chassisNumber"

// DuplicateIdentifierError means a unit's identifier is already registered on
// another vehicle unit (D-05). It names the conflicting field and the caller's
// own captured unit, never the existing vehicle: posting (D-07) maps it to the
// field of the unit being posted.
type DuplicateIdentifierError struct {
	Field              string
	GoodsReceiptUnitID string
}

func (e *DuplicateIdentifierError) Error() string {
	return "vehicle identifier already registered: " + e.Field
}

// Is makes errors.Is(err, ErrDuplicateIdentifier) true.
func (e *DuplicateIdentifierError) Is(target error) bool { return target == ErrDuplicateIdentifier }
