// Package pricing is the effective-dated price history (C-04; ADR-013).
//
// A price list belongs to one brand. Price items price one variant (with its
// model and brand) for one price type over [effectiveFrom, effectiveTo). Items
// are drafted, then approved by another user. Approved items never change except
// through controlled supersession: approving a successor ends the predecessor's
// period at the successor's start and links the two. Approved periods for the
// same list, variant and type never overlap (database exclusion constraint), so
// Lookup returns exactly one price for any instant — the basis for quotation
// price snapshots (SAL-003). No price, tax rate or grant is seeded or invented.
package pricing

import (
	"errors"
	"time"
)

// Price types (proposed value set, OQ-15; DB CHECK in migration 000009).
const (
	TypeSelling = "SELLING"
	TypeCost    = "COST"
)

// Item statuses.
const (
	StatusDraft     = "DRAFT"
	StatusApproved  = "APPROVED"
	StatusCancelled = "CANCELLED"
)

// Price list lifecycle.
const (
	ListActive   = "ACTIVE"
	ListArchived = "ARCHIVED"
)

// Errors returned by the service; the HTTP layer maps them.
var (
	ErrNotFound        = errors.New("not found")
	ErrForbidden       = errors.New("not allowed")
	ErrVersionConflict = errors.New("stale version")
	// ErrNoPrice: no approved price is valid at the requested instant.
	ErrNoPrice = errors.New("no price valid at that time")
	// ErrSelfApproval: the drafter may not approve their own price.
	ErrSelfApproval = errors.New("own draft")
)

// FieldProblem is one invalid input field.
type FieldProblem struct{ Field, Message string }

// ValidationError lists invalid fields (422).
type ValidationError struct{ Fields []FieldProblem }

func (e *ValidationError) Error() string { return "invalid price input" }

func invalid(field, msg string) *ValidationError {
	return &ValidationError{Fields: []FieldProblem{{field, msg}}}
}

// DuplicateError: a unique value is already used (409).
type DuplicateError struct{ Field string }

func (e *DuplicateError) Error() string { return "duplicate " + e.Field }

// StateError: not allowed in the current state (409).
type StateError struct{ Message string }

func (e *StateError) Error() string { return e.Message }

// OverlapError: an approved price already covers part of the period (409).
type OverlapError struct{ ConflictingID string }

func (e *OverlapError) Error() string { return "overlapping approved price" }

// BlockedError: approval prerequisites are not met (409); Unmet keys and messages.
type BlockedError struct{ Unmet []FieldProblem }

func (e *BlockedError) Error() string { return "approval prerequisites not met" }

// PriceList is a brand's list of prices (e.g. showroom selling prices).
type PriceList struct {
	ID              string    `json:"id"`
	BrandID         string    `json:"brandId"`
	Name            string    `json:"name" example:"Showroom prices"`
	Currency        string    `json:"currency" example:"INR"`
	LifecycleStatus string    `json:"lifecycleStatus" enums:"ACTIVE,ARCHIVED" example:"ACTIVE"`
	CreatedBy       string    `json:"createdBy"`
	Version         int       `json:"version" example:"1"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
} //@name PriceList

// Item is one price for one variant and type over a period.
type Item struct {
	ID          string `json:"id"`
	PriceListID string `json:"priceListId"`
	BrandID     string `json:"brandId"`
	ModelID     string `json:"modelId"`
	VariantID   string `json:"variantId"`
	PriceType   string `json:"priceType" enums:"SELLING,COST" example:"SELLING"`
	// Decimal string with two places, e.g. "125000.00" (never a float).
	Amount        string     `json:"amount" example:"125000.00"`
	Currency      string     `json:"currency" example:"INR"`
	EffectiveFrom time.Time  `json:"effectiveFrom"`
	EffectiveTo   *time.Time `json:"effectiveTo" extensions:"x-nullable"`
	Status        string     `json:"status" enums:"DRAFT,APPROVED,CANCELLED" example:"DRAFT"`
	// Always null until an approved tax configuration exists (F-10, accountant).
	TaxConfigurationID *string    `json:"taxConfigurationId" extensions:"x-nullable"`
	SourceNote         *string    `json:"sourceNote" extensions:"x-nullable"`
	CreatedBy          string     `json:"createdBy"`
	ApprovedBy         *string    `json:"approvedBy" extensions:"x-nullable"`
	ApprovedAt         *time.Time `json:"approvedAt" extensions:"x-nullable"`
	ApprovalReference  *string    `json:"approvalReference" extensions:"x-nullable"`
	SupersedesID       *string    `json:"supersedesId" extensions:"x-nullable"`
	SupersededByID     *string    `json:"supersededById" extensions:"x-nullable"`
	Version            int        `json:"version" example:"1"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
} //@name PriceItem

// Page is one page of a list.
type Page[T any] struct {
	Items              []T
	Page, Limit, Total int
}

// Actor is the signed-in user.
type Actor struct {
	UserID, Username, RequestID string
}
