// Package stockitems is the spare part and accessory SKU master (E-03;
// ADR-021; FR-015): item categories and SKUs with a unit of measure, a tracking
// mode, supplier links, a reorder level and optional brand/model compatibility.
//
// Master data only: creating or archiving an item records no quantity, serial
// instance or stock movement (stock in/out is E-04). QUANTITY items will be
// counted in their unit of measure; SERIAL items will have each piece
// identified. The SKU code and tracking mode are fixed at creation.
//
// Nothing is deleted: categories and SKUs are archived (and then never change),
// supplier links are ended. Categories, SKUs, units of measure, supplier links
// and compatibility are entered by staff; none is seeded or invented, and
// compatibility is metadata, not proof of an approved technical fitment.
package stockitems

import (
	"errors"
	"time"
)

// Tracking modes (DB CHECK stock_items_tracking_mode_valid).
const (
	TrackingQuantity = "QUANTITY"
	TrackingSerial   = "SERIAL"
)

// Statuses of categories and SKUs.
const (
	StatusActive   = "ACTIVE"
	StatusArchived = "ARCHIVED"
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

func (e *ValidationError) Error() string { return "invalid stock item input" }

func invalid(field, msg string) *ValidationError {
	return &ValidationError{Fields: []FieldProblem{{field, msg}}}
}

// DuplicateError: the SKU code or category name is already used (409).
type DuplicateError struct{ Field string }

func (e *DuplicateError) Error() string { return "duplicate " + e.Field }

// StateError: not allowed in the current state (409).
type StateError struct{ Message string }

func (e *StateError) Error() string { return e.Message }

// Category groups items. Categories are data: the business adds its own.
type Category struct {
	ID            string     `json:"id"`
	Name          string     `json:"name" example:"Brake parts"`
	Description   *string    `json:"description" extensions:"x-nullable"`
	Status        string     `json:"status" enums:"ACTIVE,ARCHIVED" example:"ACTIVE"`
	ArchivedBy    *string    `json:"archivedBy" extensions:"x-nullable"`
	ArchivedAt    *time.Time `json:"archivedAt" extensions:"x-nullable"`
	ArchiveReason *string    `json:"archiveReason" extensions:"x-nullable"`
	// ActiveItemCount is the number of ACTIVE SKUs in the category.
	ActiveItemCount int       `json:"activeItemCount" example:"0"`
	CreatedBy       string    `json:"createdBy"`
	Version         int       `json:"version" example:"1"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
} //@name StockItemCategory

// Item is a spare part or accessory SKU (list view).
type Item struct {
	ID  string `json:"id"`
	SKU string `json:"sku" example:"BRK-PAD-01"`
	// SKUNormalised is the comparison form (upper-case): one SKU per value.
	SKUNormalised  string  `json:"skuNormalised" example:"BRK-PAD-01"`
	Name           string  `json:"name" example:"Brake pad set"`
	Description    *string `json:"description" extensions:"x-nullable"`
	CategoryID     string  `json:"categoryId"`
	CategoryName   string  `json:"categoryName"`
	CategoryStatus string  `json:"categoryStatus" enums:"ACTIVE,ARCHIVED"`
	UOM            string  `json:"uom" example:"PCS"`
	TrackingMode   string  `json:"trackingMode" enums:"QUANTITY,SERIAL" example:"QUANTITY"`
	// ReorderLevel is a decimal string in the unit of measure (whole for SERIAL); null when not set.
	ReorderLevel  *string    `json:"reorderLevel" extensions:"x-nullable" example:"5.000"`
	Status        string     `json:"status" enums:"ACTIVE,ARCHIVED" example:"ACTIVE"`
	ArchivedBy    *string    `json:"archivedBy" extensions:"x-nullable"`
	ArchivedAt    *time.Time `json:"archivedAt" extensions:"x-nullable"`
	ArchiveReason *string    `json:"archiveReason" extensions:"x-nullable"`
	CreatedBy     string     `json:"createdBy"`
	Version       int        `json:"version" example:"1"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
} //@name StockItem

// SupplierLink says a supplier supplies the item. Ended links are kept as history.
type SupplierLink struct {
	ID                 string     `json:"id"`
	SupplierID         string     `json:"supplierId"`
	SupplierName       string     `json:"supplierName"`
	SupplierStatus     string     `json:"supplierStatus" enums:"PENDING_VERIFICATION,VERIFIED,ARCHIVED"`
	SupplierPartNumber *string    `json:"supplierPartNumber" extensions:"x-nullable"`
	CreatedBy          string     `json:"createdBy"`
	CreatedAt          time.Time  `json:"createdAt"`
	EndedBy            *string    `json:"endedBy" extensions:"x-nullable"`
	EndedAt            *time.Time `json:"endedAt" extensions:"x-nullable"`
} //@name StockItemSupplier

// Compatibility is one brand (modelId null: every model of it) or one model.
type Compatibility struct {
	ID          string  `json:"id"`
	BrandID     string  `json:"brandId"`
	BrandName   string  `json:"brandName"`
	BrandStatus string  `json:"brandStatus" enums:"PENDING_VERIFICATION,ACTIVE,ARCHIVED"`
	ModelID     *string `json:"modelId" extensions:"x-nullable"`
	ModelName   *string `json:"modelName" extensions:"x-nullable"`
} //@name StockItemCompatibility

// Detail is an item with its supplier links (current and ended) and compatibility.
// Without suppliers.view, suppliers is empty and suppliersHidden true.
type Detail struct {
	Item
	Suppliers       []SupplierLink  `json:"suppliers"`
	SuppliersHidden bool            `json:"suppliersHidden"`
	Compatibility   []Compatibility `json:"compatibility"`
} //@name StockItemDetail

// Page is one page of a list.
type Page[T any] struct {
	Items              []T
	Page, Limit, Total int
}

// Actor is the signed-in user.
type Actor struct{ UserID, Username, RequestID string }
