// Package pdi records pre-delivery inspections of vehicle units (D-06;
// ADR-024; INV-008, INV-009, FR-006).
//
// An inspection is recorded complete, in one request, for one unit: the
// outcome of every applicable checklist item, notes, private evidence (B-09
// attachments) and a disposition. The disposition moves the unit through the
// stock movement ledger (stock.MoveUnit, E-01) in the same transaction, along
// the approved lifecycle edges only (System Architecture §6):
//
//	PASS   RECEIVED_PENDING_PDI -> AVAILABLE    (PDI_PASS)
//	HOLD   RECEIVED_PENDING_PDI -> QUALITY_HOLD (QUALITY_HOLD)
//	REJECT QUALITY_HOLD         -> REJECTED     (REJECT)
//
// A failed PDI going straight to REJECTED, and releasing a held unit back to
// AVAILABLE, are not approved (OQ-31) and are refused. A unit exists only once
// its goods receipt is POSTED (vehicles.RegisterFromReceiptUnit, D-07), so a
// pass never makes an unposted unit saleable.
//
// The checklist is data: items are created and retired by holders of
// vehicle_units.pdi.checklist.manage, for every brand or one brand. None is
// seeded (no approved checklist yet), so a PASS is refused until at least one
// item applies to the unit; every applicable item must be answered, and a pass
// needs every answer PASS. Nothing unconfigured counts as passed.
//
// Inspections, outcomes and evidence links are append-only (database guards).
// Reading needs vehicle_units.view (and attachments.view for evidence details);
// recording needs vehicle_units.pdi.record. No grant is seeded (RB-5).
package pdi

import (
	"errors"
	"time"
)

// Dispositions.
const (
	DispositionPass   = "PASS"
	DispositionHold   = "HOLD"
	DispositionReject = "REJECT"
)

// Dispositions lists every disposition.
var Dispositions = []string{DispositionPass, DispositionHold, DispositionReject}

// Item outcomes.
const (
	OutcomePass = "PASS"
	OutcomeFail = "FAIL"
)

// Inspection results: a server-computed summary of the recorded outcomes.
const (
	ResultAllPassed   = "ALL_PASSED"   // at least one outcome, all PASS
	ResultHasFailures = "HAS_FAILURES" // at least one FAIL
	ResultNotChecked  = "NOT_CHECKED"  // no checklist item applied to the unit
)

// Checklist item statuses.
const (
	ItemActive  = "ACTIVE"
	ItemRetired = "RETIRED"
)

// AttachmentOwner is the attachments.owner_type of PDI evidence.
const AttachmentOwner = "pdi_inspection"

// Limits.
const (
	maxItems       = 100
	maxAttachments = 20
	maxNotes       = 2000
	maxItemNote    = 500
	maxLabel       = 200
	maxDescription = 1000
	maxReason      = 500
)

// Edge is a disposition's lifecycle edge and movement type.
type Edge struct{ From, To, MovementType string }

// Edges maps each disposition to its only allowed edge (OQ-31: nothing else).
var Edges = map[string]Edge{
	DispositionPass:   {"RECEIVED_PENDING_PDI", "AVAILABLE", "PDI_PASS"},
	DispositionHold:   {"RECEIVED_PENDING_PDI", "QUALITY_HOLD", "QUALITY_HOLD"},
	DispositionReject: {"QUALITY_HOLD", "REJECTED", "REJECT"},
}

// AllowedDispositions returns the dispositions a unit in status can receive.
func AllowedDispositions(status string) []string {
	out := []string{}
	for _, d := range Dispositions {
		if Edges[d].From == status {
			out = append(out, d)
		}
	}
	return out
}

// ChecklistItem is one configured checklist item.
type ChecklistItem struct {
	ID           string     `json:"id"`
	Code         string     `json:"code" example:"BRAKES"`
	Label        string     `json:"label" example:"Brakes"`
	Description  *string    `json:"description" extensions:"x-nullable"`
	BrandID      *string    `json:"brandId" extensions:"x-nullable"`
	BrandName    *string    `json:"brandName" extensions:"x-nullable"`
	SortOrder    int        `json:"sortOrder" example:"10"`
	Status       string     `json:"status" example:"ACTIVE" enums:"ACTIVE,RETIRED"`
	Version      int        `json:"version" example:"1"`
	CreatedBy    string     `json:"createdBy"`
	CreatedAt    time.Time  `json:"createdAt"`
	RetiredBy    *string    `json:"retiredBy" extensions:"x-nullable"`
	RetiredAt    *time.Time `json:"retiredAt" extensions:"x-nullable"`
	RetireReason *string    `json:"retireReason" extensions:"x-nullable"`
} //@name PDIChecklistItem

// ChecklistPage is one page of checklist items.
type ChecklistPage struct {
	Items []ChecklistItem `json:"items"`
	Page  int             `json:"page" example:"1"`
	Limit int             `json:"limit" example:"50"`
	Total int             `json:"total" example:"0"`
} //@name PDIChecklistPage

// ChecklistCreateInput creates a checklist item. sortOrder orders the
// checklist (then code).
type ChecklistCreateInput struct {
	Code        string  `json:"code" example:"BRAKES"`
	Label       string  `json:"label" example:"Brakes"`
	Description *string `json:"description" extensions:"x-nullable"`
	BrandID     *string `json:"brandId" extensions:"x-nullable"`
	SortOrder   *int    `json:"sortOrder" extensions:"x-nullable" example:"10"`
} //@name PDIChecklistItemCreate

// ChecklistRetireInput retires an item.
type ChecklistRetireInput struct {
	Version *int   `json:"version" example:"1"`
	Reason  string `json:"reason" example:"Replaced by a brand-specific item"`
} //@name PDIChecklistItemRetire

// ChecklistQuery is the raw checklist list request.
type ChecklistQuery struct{ Status, BrandID, Page, Limit string }

// ItemOutcomeInput is the outcome of one checklist item.
type ItemOutcomeInput struct {
	ChecklistItemID string  `json:"checklistItemId"`
	Outcome         string  `json:"outcome" example:"FAIL" enums:"PASS,FAIL"`
	Note            *string `json:"note" extensions:"x-nullable" example:"Rear brake lever loose"`
} //@name PDIItemOutcomeInput

// RecordInput records one inspection. version is the unit's current version.
type RecordInput struct {
	Version       *int               `json:"version" example:"1"`
	Disposition   string             `json:"disposition" example:"HOLD" enums:"PASS,HOLD,REJECT"`
	Items         []ItemOutcomeInput `json:"items"`
	Notes         *string            `json:"notes" extensions:"x-nullable" example:"Rear brake lever loose; supplier informed"`
	AttachmentIDs []string           `json:"attachmentIds"`
} //@name PDIInspectionCreate

// ItemOutcome is one recorded outcome.
type ItemOutcome struct {
	LineNo          int     `json:"lineNo" example:"1"`
	ChecklistItemID string  `json:"checklistItemId"`
	Code            string  `json:"code" example:"BRAKES"`
	Label           string  `json:"label" example:"Brakes"`
	Outcome         string  `json:"outcome" example:"FAIL" enums:"PASS,FAIL"`
	Note            *string `json:"note" extensions:"x-nullable"`
} //@name PDIItemOutcome

// Evidence is one linked private attachment. Download it through the
// attachments API (signed link after its own checks).
type Evidence struct {
	AttachmentID string `json:"attachmentId"`
	OriginalName string `json:"originalName" example:"brake.jpg"`
	ContentType  string `json:"contentType" example:"image/jpeg"`
	SizeBytes    int64  `json:"sizeBytes" example:"204800"`
} //@name PDIEvidence

// Inspection is a recorded inspection as listed (history).
type Inspection struct {
	ID               string    `json:"id"`
	InspectionNumber string    `json:"inspectionNumber" example:"PDI-000001"`
	VehicleUnitID    string    `json:"vehicleUnitId"`
	UnitNumber       string    `json:"unitNumber" example:"MAE-VEH-000001"`
	Disposition      string    `json:"disposition" example:"HOLD" enums:"PASS,HOLD,REJECT"`
	Result           string    `json:"result" example:"HAS_FAILURES" enums:"ALL_PASSED,HAS_FAILURES,NOT_CHECKED"`
	FromStatus       string    `json:"fromStatus" example:"RECEIVED_PENDING_PDI"`
	ToStatus         string    `json:"toStatus" example:"QUALITY_HOLD"`
	LocationID       *string   `json:"locationId" extensions:"x-nullable"`
	Notes            *string   `json:"notes" extensions:"x-nullable"`
	MovementID       string    `json:"movementId"`
	InspectedBy      string    `json:"inspectedBy"`
	InspectedByName  *string   `json:"inspectedByName" extensions:"x-nullable"`
	InspectedAt      time.Time `json:"inspectedAt"`
	ItemCount        int       `json:"itemCount" example:"12"`
	FailedCount      int       `json:"failedCount" example:"1"`
	EvidenceCount    int       `json:"evidenceCount" example:"2"`
} //@name PDIInspection

// InspectionDetail is an inspection with its outcomes and evidence. Evidence
// is empty and evidenceHidden true without attachments.view.
type InspectionDetail struct {
	Inspection
	Items          []ItemOutcome `json:"items"`
	Evidence       []Evidence    `json:"evidence"`
	EvidenceHidden bool          `json:"evidenceHidden"`
} //@name PDIInspectionDetail

// UnitState is the unit as it is now.
type UnitState struct {
	ID         string  `json:"id"`
	UnitNumber string  `json:"unitNumber" example:"MAE-VEH-000001"`
	Status     string  `json:"status" example:"RECEIVED_PENDING_PDI"`
	BrandID    string  `json:"brandId"`
	LocationID *string `json:"locationId" extensions:"x-nullable"`
	Version    int     `json:"version" example:"1"`
	// Saleable is true only for AVAILABLE units (INV-006/007).
	Saleable bool `json:"saleable" example:"false"`
} //@name PDIUnitState

// Overview is everything a PDI screen needs for one unit.
type Overview struct {
	Unit UnitState `json:"unit"`
	// AllowedDispositions are the dispositions the unit's status admits
	// (empty when no inspection can be recorded); recording also needs
	// vehicle_units.pdi.record (canRecord).
	AllowedDispositions []string `json:"allowedDispositions"`
	CanRecord           bool     `json:"canRecord"`
	// Checklist is the active checklist that applies to the unit (every-brand
	// items and items of the unit's brand), in order. Empty = not configured:
	// a PASS is refused (409 PDI_CHECKLIST_NOT_CONFIGURED).
	Checklist   []ChecklistItem `json:"checklist"`
	Inspections []Inspection    `json:"inspections"`
} //@name PDIOverview

// RecordResult is a recorded inspection and the unit as it is now.
type RecordResult struct {
	Inspection InspectionDetail `json:"inspection"`
	Unit       UnitState        `json:"unit"`
} //@name PDIRecordResult

// FieldProblem is one invalid input field.
type FieldProblem struct {
	Field   string
	Message string
}

// ValidationError lists invalid fields (422).
type ValidationError struct{ Fields []FieldProblem }

func (e *ValidationError) Error() string { return "invalid PDI request" }

// StateError means the unit's status does not admit the disposition (409
// INVALID_STATE; audited DENIED).
type StateError struct{ Status, Disposition, Message string }

func (e *StateError) Error() string { return e.Message }

// Errors the HTTP adapter maps to responses.
var (
	ErrForbidden = errors.New("forbidden")
	ErrNotFound  = errors.New("not found")
	// ErrVersionConflict: the unit (or checklist item) changed since it was read.
	ErrVersionConflict = errors.New("changed since it was read")
	// ErrChecklistNotConfigured: a PASS needs at least one applicable checklist item.
	ErrChecklistNotConfigured = errors.New("no PDI checklist item applies to this unit")
	// ErrChassisRequired: INV-002, the unit has no chassis/frame number.
	ErrChassisRequired = errors.New("vehicle unit has no chassis/frame number")
	// ErrIdempotencyKeyReused: the actor used the key for a different request.
	ErrIdempotencyKeyReused = errors.New("idempotency key already used for a different request")
	// ErrDuplicate: an active checklist item with that code already exists in that brand scope.
	ErrDuplicate = errors.New("duplicate checklist item")
	// ErrItemRetired: the checklist item is already retired.
	ErrItemRetired = errors.New("checklist item already retired")
	// errKeyRace: another transaction committed the same key first.
	errKeyRace = errors.New("idempotency key taken concurrently")
)
