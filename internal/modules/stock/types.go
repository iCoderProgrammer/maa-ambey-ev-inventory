// Package stock is the append-only stock movement ledger (E-01; ADR-020;
// INV-009, LED-001…LED-003). Every change of a vehicle unit's stock state is one
// movement: quantity, status and location before and after, source, actor,
// server time and the idempotency key of the request that made it.
//
// The ledger is the only way a unit changes. Inserting a movement is what sets
// vehicle_units.status / location_id (a database trigger applies it, and the
// unit guard accepts no other update); the database refuses a movement whose
// "before" is not the unit's current state, and at commit every unit must agree
// with its latest movement. Movements are never updated or deleted.
//
// Callers inside a transaction use RecordReceipt (receipt posting, D-07, through
// vehicles.RegisterFromReceiptUnit) and MoveUnit (status changes owned by other
// tasks, e.g. PDI, D-06). The API offers only the moves no other workflow owns:
// putting an AVAILABLE unit on display and back. Allowed movement types are the
// approved lifecycle edges that have an owner so far (see Transitions).
package stock

import (
	"errors"
	"time"
)

// Movement types (DB CHECK stock_movements_type_valid).
const (
	TypeReceipt     = "RECEIPT"      // registration from a posted goods receipt: none -> RECEIVED_PENDING_PDI
	TypePDIPass     = "PDI_PASS"     // RECEIVED_PENDING_PDI -> AVAILABLE (caller: PDI, D-06)
	TypeQualityHold = "QUALITY_HOLD" // RECEIVED_PENDING_PDI -> QUALITY_HOLD (caller: PDI, D-06)
	TypeReject      = "REJECT"       // QUALITY_HOLD -> REJECTED (caller: PDI, D-06)
	TypeDisplayOn   = "DISPLAY_ON"   // AVAILABLE -> DISPLAY (API)
	TypeDisplayOff  = "DISPLAY_OFF"  // DISPLAY -> AVAILABLE (API)
)

// Transition is one approved lifecycle edge (System Architecture §6).
type Transition struct{ From, To string }

// Transitions maps each status-changing movement type to its edge. It matches
// the database CHECK stock_movements_transition_valid; anything else is refused.
var Transitions = map[string]Transition{
	TypePDIPass:     {"RECEIVED_PENDING_PDI", "AVAILABLE"},
	TypeQualityHold: {"RECEIVED_PENDING_PDI", "QUALITY_HOLD"},
	TypeReject:      {"QUALITY_HOLD", "REJECTED"},
	TypeDisplayOn:   {"AVAILABLE", "DISPLAY"},
	TypeDisplayOff:  {"DISPLAY", "AVAILABLE"},
}

// ReceivedStatus is the status a unit is registered in.
const ReceivedStatus = "RECEIVED_PENDING_PDI"

// Source types (DB CHECK stock_movements_source_valid).
const (
	SourceGoodsReceipt = "GOODS_RECEIPT"
	SourceStaffAction  = "STAFF_ACTION"
)

// ItemVehicleUnit is the only item type until spare parts exist (E-03).
const ItemVehicleUnit = "VEHICLE_UNIT"

// Movement is one ledger entry.
type Movement struct {
	ID              string    `json:"id" example:"3f2a1c4e-7b6d-4e8f-9a0b-1c2d3e4f5a6b"`
	ItemType        string    `json:"itemType" example:"VEHICLE_UNIT" enums:"VEHICLE_UNIT"`
	VehicleUnitID   string    `json:"vehicleUnitId"`
	MovementType    string    `json:"movementType" example:"DISPLAY_ON" enums:"RECEIPT,PDI_PASS,QUALITY_HOLD,REJECT,DISPLAY_ON,DISPLAY_OFF"`
	QuantityBefore  int       `json:"quantityBefore" example:"1"`
	QuantityAfter   int       `json:"quantityAfter" example:"1"`
	FromStatus      *string   `json:"fromStatus" extensions:"x-nullable" example:"AVAILABLE"`
	ToStatus        string    `json:"toStatus" example:"DISPLAY"`
	FromLocationID  *string   `json:"fromLocationId" extensions:"x-nullable"`
	ToLocationID    *string   `json:"toLocationId" extensions:"x-nullable"`
	SourceType      string    `json:"sourceType" example:"STAFF_ACTION" enums:"GOODS_RECEIPT,STAFF_ACTION"`
	SourceID        *string   `json:"sourceId" extensions:"x-nullable"`
	SourceReference *string   `json:"sourceReference" extensions:"x-nullable" example:"GRN-000001"`
	ActorID         string    `json:"actorId"`
	Note            *string   `json:"note" extensions:"x-nullable"`
	CreatedAt       time.Time `json:"createdAt"`
} //@name StockMovement

// UnitState is a vehicle unit's state as it is now.
type UnitState struct {
	ID         string  `json:"id"`
	UnitNumber string  `json:"unitNumber" example:"MAE-VEH-000001"`
	Status     string  `json:"status" example:"DISPLAY"`
	LocationID *string `json:"locationId" extensions:"x-nullable"`
	Version    int     `json:"version" example:"2"`
} //@name StockUnitState

// MoveResult is a recorded movement and the unit as it is now (on a replay the
// unit may have moved on since).
type MoveResult struct {
	Movement Movement  `json:"movement"`
	Unit     UnitState `json:"unit"`
} //@name StockMoveResult

// MoveRequest is one status change of a vehicle unit, for MoveUnit.
type MoveRequest struct {
	UnitID string
	Type   string
	// ExpectedVersion, when set, must be the unit's current version.
	ExpectedVersion *int
	// IdempotencyKey is required: 8-100 of A-Z a-z 0-9 _ -, per actor.
	IdempotencyKey string
	Note           *string
	RequestID      string
}

// FieldProblem is one invalid input field.
type FieldProblem struct {
	Field   string
	Message string
}

// ValidationError lists invalid fields (422).
type ValidationError struct{ Fields []FieldProblem }

func (e *ValidationError) Error() string { return "invalid stock movement request" }

// InvalidStateError means the unit is not in the movement's starting status.
type InvalidStateError struct{ Current, Required string }

func (e *InvalidStateError) Error() string {
	return "vehicle unit is " + e.Current + ", the movement needs " + e.Required
}

// Errors the HTTP adapter (and callers of MoveUnit) map to responses.
var (
	ErrForbidden = errors.New("forbidden")
	// ErrNotFound: no such unit, or one outside the caller's location scope.
	ErrNotFound = errors.New("vehicle unit not found")
	// ErrVersionConflict: ExpectedVersion is not the unit's version.
	ErrVersionConflict = errors.New("vehicle unit changed since it was read")
	// ErrChassisRequired: a unit without a chassis/frame number cannot leave
	// RECEIVED_PENDING_PDI (INV-002; DB CHECK vehicle_units_chassis_after_receipt).
	ErrChassisRequired = errors.New("vehicle unit has no chassis/frame number")
	// ErrIdempotencyKeyReused: the actor used the key for a different operation.
	ErrIdempotencyKeyReused = errors.New("idempotency key already used for a different operation")
	// errKeyRace: another transaction committed the same key first; the caller
	// rolls back and resolves it with Replay.
	errKeyRace = errors.New("idempotency key taken concurrently")
)
