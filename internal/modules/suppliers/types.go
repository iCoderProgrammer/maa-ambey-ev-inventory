// Package suppliers is the supplier master (D-01; ADR-015): suppliers with
// contacts, private documents (B-09 attachments) and relationships to one or
// more brands. A supplier record is not proof of dealership authorisation:
// suppliers and supplier-brand relationships start unverified and are verified
// only with a reference, a supplier document and a verifier. Nothing is ever
// deleted: suppliers are archived, contacts deactivated, relationships ended.
// No supplier, contact or document is seeded or invented.
package suppliers

import (
	"errors"
	"time"
)

// Supplier statuses.
const (
	StatusPending  = "PENDING_VERIFICATION"
	StatusVerified = "VERIFIED"
	StatusArchived = "ARCHIVED"
)

// Supplier-brand relationship statuses.
const (
	LinkUnverified = "UNVERIFIED"
	LinkVerified   = "VERIFIED"
	LinkEnded      = "ENDED"
)

// DocumentTypes is the proposed document type value set (OQ-19; DB CHECK).
var DocumentTypes = []string{"GST_REGISTRATION", "AUTHORISATION", "AGREEMENT", "OTHER"}

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

func (e *ValidationError) Error() string { return "invalid supplier input" }

func invalid(field, msg string) *ValidationError {
	return &ValidationError{Fields: []FieldProblem{{field, msg}}}
}

// DuplicateError: a unique value is already used (409).
type DuplicateError struct{ Field string }

func (e *DuplicateError) Error() string { return "duplicate " + e.Field }

// StateError: not allowed in the current state (409).
type StateError struct{ Message string }

func (e *StateError) Error() string { return e.Message }

// Supplier is the supplier record (list view).
type Supplier struct {
	ID                     string     `json:"id"`
	LegalName              string     `json:"legalName" example:"Example Supplier Private Limited"`
	TradeName              *string    `json:"tradeName" extensions:"x-nullable"`
	GSTIN                  *string    `json:"gstin" extensions:"x-nullable"`
	AddressLine1           *string    `json:"addressLine1" extensions:"x-nullable"`
	AddressLine2           *string    `json:"addressLine2" extensions:"x-nullable"`
	City                   *string    `json:"city" extensions:"x-nullable"`
	State                  *string    `json:"state" extensions:"x-nullable"`
	PostalCode             *string    `json:"postalCode" extensions:"x-nullable"`
	Status                 string     `json:"status" enums:"PENDING_VERIFICATION,VERIFIED,ARCHIVED" example:"PENDING_VERIFICATION"`
	VerificationReference  *string    `json:"verificationReference" extensions:"x-nullable"`
	VerificationDocumentID *string    `json:"verificationDocumentId" extensions:"x-nullable"`
	VerifiedBy             *string    `json:"verifiedBy" extensions:"x-nullable"`
	VerifiedAt             *time.Time `json:"verifiedAt" extensions:"x-nullable"`
	Notes                  *string    `json:"notes" extensions:"x-nullable"`
	CreatedBy              string     `json:"createdBy"`
	// Brands with an open (unverified or verified) relationship.
	BrandCount int       `json:"brandCount" example:"0"`
	Version    int       `json:"version" example:"1"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
} //@name Supplier

// Contact is a business contact at a supplier.
type Contact struct {
	ID         string    `json:"id"`
	SupplierID string    `json:"supplierId"`
	Name       string    `json:"name"`
	Role       *string   `json:"role" extensions:"x-nullable"`
	Phone      *string   `json:"phone" extensions:"x-nullable"`
	Email      *string   `json:"email" extensions:"x-nullable"`
	IsPrimary  bool      `json:"isPrimary"`
	Active     bool      `json:"active"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
} //@name SupplierContact

// Document is a private attachment linked to the supplier. Open it through
// the attachments API (short-lived signed link); there is no public URL.
type Document struct {
	ID           string    `json:"id"`
	SupplierID   string    `json:"supplierId"`
	AttachmentID string    `json:"attachmentId"`
	DocumentType string    `json:"documentType" enums:"GST_REGISTRATION,AUTHORISATION,AGREEMENT,OTHER"`
	Note         *string   `json:"note" extensions:"x-nullable"`
	FileName     string    `json:"fileName"`
	ContentType  string    `json:"contentType"`
	AddedBy      string    `json:"addedBy"`
	CreatedAt    time.Time `json:"createdAt"`
} //@name SupplierDocument

// BrandLink is a supplier-brand relationship with its own evidence.
type BrandLink struct {
	ID                     string     `json:"id"`
	SupplierID             string     `json:"supplierId"`
	SupplierName           string     `json:"supplierName"`
	SupplierStatus         string     `json:"supplierStatus" enums:"PENDING_VERIFICATION,VERIFIED,ARCHIVED"`
	BrandID                string     `json:"brandId"`
	BrandName              string     `json:"brandName"`
	Status                 string     `json:"status" enums:"UNVERIFIED,VERIFIED,ENDED" example:"UNVERIFIED"`
	AuthorisationReference *string    `json:"authorisationReference" extensions:"x-nullable"`
	EvidenceDocumentID     *string    `json:"evidenceDocumentId" extensions:"x-nullable"`
	EffectiveFrom          *string    `json:"effectiveFrom" extensions:"x-nullable" example:"2026-10-01"`
	EffectiveTo            *string    `json:"effectiveTo" extensions:"x-nullable"`
	VerifiedBy             *string    `json:"verifiedBy" extensions:"x-nullable"`
	VerifiedAt             *time.Time `json:"verifiedAt" extensions:"x-nullable"`
	Note                   *string    `json:"note" extensions:"x-nullable"`
	EndedReason            *string    `json:"endedReason" extensions:"x-nullable"`
	// Current: VERIFIED, supplier VERIFIED and today within the effective dates.
	Current   bool      `json:"current"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
} //@name SupplierBrandLink

// Detail is a supplier with its contacts, documents and brand relationships.
type Detail struct {
	Supplier
	Contacts   []Contact   `json:"contacts"`
	Documents  []Document  `json:"documents"`
	BrandLinks []BrandLink `json:"brandLinks"`
} //@name SupplierDetail

// Page is one page of a list.
type Page[T any] struct {
	Items              []T
	Page, Limit, Total int
}

// Actor is the signed-in user.
type Actor struct{ UserID, Username, RequestID string }
