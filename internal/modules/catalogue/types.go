package catalogue

import (
	"bytes"
	"encoding/json"
	"errors"
	"time"
)

// Lifecycle statuses (BR-001, BR-008); the value set is the DB CHECK in migrations 000001/000007.
const (
	StatusPending  = "PENDING_VERIFICATION"
	StatusActive   = "ACTIVE"
	StatusArchived = "ARCHIVED"
)

// Brand authorisation statuses (BR-002; value set pending sign-off, OQ-1).
const (
	AuthUnverified    = "UNVERIFIED"
	AuthAuthorised    = "AUTHORISED"
	AuthNotAuthorised = "NOT_AUTHORISED"
	AuthRevoked       = "REVOKED"
)

// Source confidence (BR-004).
const (
	ConfidenceUnverified        = "UNVERIFIED"
	ConfidenceSupplierConfirmed = "SUPPLIER_CONFIRMED"
	ConfidenceClientApproved    = "CLIENT_APPROVED"
)

// Errors returned by the service. The HTTP layer maps them to status codes.
var (
	ErrNotFound  = errors.New("catalogue item not found")
	ErrForbidden = errors.New("not allowed")
	// ErrVersionConflict: the item changed since the client read it.
	ErrVersionConflict = errors.New("stale version")
)

// ValidationError lists invalid input fields (422).
type ValidationError struct {
	Fields []FieldProblem
}

// FieldProblem is one invalid field.
type FieldProblem struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string { return "invalid catalogue input" }

// DuplicateError: a unique name or code is already used in its scope (409).
type DuplicateError struct {
	Field string
}

func (e *DuplicateError) Error() string { return "duplicate " + e.Field }

// StateError: the action is not allowed in the item's current state (409).
type StateError struct {
	Message string
}

func (e *StateError) Error() string { return e.Message }

// ActivationBlockedError: activation prerequisites are not met (409). Unmet
// lists the failing readiness checks.
type ActivationBlockedError struct {
	Unmet []ReadinessCheck
}

func (e *ActivationBlockedError) Error() string { return "activation prerequisites not met" }

// Brand is a brand master record. Lifecycle and authorisation are separate:
// a brand is usable for purchase/sales (operational) only when it is ACTIVE
// and AUTHORISED (BR-001, BR-002).
type Brand struct {
	ID                      string     `json:"id" example:"3f2b8c1e-9a4d-4e6f-8b7a-1c2d3e4f5a6b"`
	Name                    string     `json:"name" example:"Brand A"`
	Slug                    string     `json:"slug" example:"brand-a"`
	LifecycleStatus         string     `json:"lifecycleStatus" enums:"PENDING_VERIFICATION,ACTIVE,ARCHIVED" example:"PENDING_VERIFICATION"`
	AuthorisationStatus     string     `json:"authorisationStatus" enums:"UNVERIFIED,AUTHORISED,NOT_AUTHORISED,REVOKED" example:"UNVERIFIED"`
	ManufacturerLegalName   *string    `json:"manufacturerLegalName" extensions:"x-nullable"`
	AuthorisationReference  *string    `json:"authorisationReference" extensions:"x-nullable"`
	AuthorisationEvidenceID *string    `json:"authorisationEvidenceId" extensions:"x-nullable"`
	VerifiedBy              *string    `json:"verifiedBy" extensions:"x-nullable"`
	VerifiedAt              *time.Time `json:"verifiedAt" extensions:"x-nullable"`
	// Operational is true only for ACTIVE + AUTHORISED brands (derived, read-only).
	Operational bool      `json:"operational" example:"false"`
	Version     int       `json:"version" example:"1"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
} //@name CatalogueBrand

// Provenance records where a catalogue value came from and who checked it (BR-004).
type Provenance struct {
	Confidence       string     `json:"confidence" enums:"UNVERIFIED,SUPPLIER_CONFIRMED,CLIENT_APPROVED" example:"UNVERIFIED"`
	SourceURL        *string    `json:"sourceUrl" extensions:"x-nullable"`
	SourceDocumentID *string    `json:"sourceDocumentId" extensions:"x-nullable"`
	SourceDate       *string    `json:"sourceDate" extensions:"x-nullable" example:"2026-10-01"`
	VerifiedBy       *string    `json:"verifiedBy" extensions:"x-nullable"`
	VerifiedAt       *time.Time `json:"verifiedAt" extensions:"x-nullable"`
}

// Model is a vehicle model of one brand (BR-003).
type Model struct {
	ID               string  `json:"id"`
	BrandID          string  `json:"brandId"`
	Name             string  `json:"name" example:"Model One"`
	InternalCode     string  `json:"internalCode" example:"M1"`
	ManufacturerCode *string `json:"manufacturerCode" extensions:"x-nullable"`
	VehicleType      *string `json:"vehicleType" extensions:"x-nullable"`
	LifecycleStatus  string  `json:"lifecycleStatus" enums:"PENDING_VERIFICATION,ACTIVE,ARCHIVED"`
	Provenance
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
} //@name CatalogueModel

// Variant is a variant of one model; brandId always equals the model's brand.
type Variant struct {
	ID               string  `json:"id"`
	BrandID          string  `json:"brandId"`
	ModelID          string  `json:"modelId"`
	Name             string  `json:"name" example:"Standard"`
	VariantCode      string  `json:"variantCode" example:"STD"`
	ManufacturerCode *string `json:"manufacturerCode" extensions:"x-nullable"`
	LifecycleStatus  string  `json:"lifecycleStatus" enums:"PENDING_VERIFICATION,ACTIVE,ARCHIVED"`
	Provenance
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
} //@name CatalogueVariant

// Colour is a colour offered for one variant.
type Colour struct {
	ID              string  `json:"id"`
	BrandID         string  `json:"brandId"`
	VariantID       string  `json:"variantId"`
	Name            string  `json:"name" example:"Colour A"`
	ColourCode      *string `json:"colourCode" extensions:"x-nullable"`
	LifecycleStatus string  `json:"lifecycleStatus" enums:"PENDING_VERIFICATION,ACTIVE,ARCHIVED"`
	Provenance
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
} //@name CatalogueColour

// ReadinessCheck is one prerequisite for enabling a brand for purchase/sales.
// Pending checks belong to features not built yet; they are never met, so
// activation stays blocked until those features record approved data (fail closed).
type ReadinessCheck struct {
	Key     string `json:"key" example:"authorisation"`
	Met     bool   `json:"met" example:"false"`
	Pending bool   `json:"pending" example:"false"`
	Message string `json:"message" example:"Record AUTHORISED status with legal name, reference and evidence."`
} //@name BrandReadinessCheck

// Readiness is the brand activation checklist.
type Readiness struct {
	BrandID string           `json:"brandId"`
	Ready   bool             `json:"ready" example:"false"`
	Checks  []ReadinessCheck `json:"checks"`
} //@name BrandActivationReadiness

// Page is one page of a list.
type Page[T any] struct {
	Items []T
	Page  int
	Limit int
	Total int
}

// ListFilter is a validated list query.
type ListFilter struct {
	Query      string
	Status     string
	Confidence string
	AuthStatus string
	BrandID    string
	ModelID    string
	VariantID  string
	Page       int
	Limit      int
}

// Offset is the row offset of the page.
func (f ListFilter) Offset() int { return (f.Page - 1) * f.Limit }

// OptString is a PATCH field: Set reports whether the key was present; a
// JSON null sets Value to nil (clear the field).
type OptString struct {
	Set   bool
	Value *string
}

// UnmarshalJSON implements json.Unmarshaler.
func (o *OptString) UnmarshalJSON(b []byte) error {
	o.Set = true
	if bytes.Equal(b, []byte("null")) {
		o.Value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	o.Value = &s
	return nil
}

// Actor is the signed-in user making the request.
type Actor struct {
	UserID    string
	Username  string
	RequestID string
}
