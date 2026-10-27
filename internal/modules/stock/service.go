package stock

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/audit"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
)

// Repository runs a movement in its own transaction.
type Repository interface {
	Move(ctx context.Context, actor audit.Actor, scope access.Scope, req MoveRequest) (MoveResult, bool, error)
}

// Service checks who may make the API's movements (again, after the route
// guard) and runs them.
type Service struct {
	repo Repository
	log  *slog.Logger
}

// NewService builds the service.
func NewService(repo Repository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// DisplayInput is the body of the display moves.
type DisplayInput struct {
	// Version is the unit's current version (required).
	Version *int    `json:"version" example:"2"`
	Note    *string `json:"note" extensions:"x-nullable" example:"Front window"`
} //@name StockDisplayInput

// Display puts an AVAILABLE unit on display (on) or returns a DISPLAY unit to
// AVAILABLE (vehicle_units.display.manage) in the caller's location scope. The
// version and an Idempotency-Key are required; a repeat with the same key
// returns the first movement (replayed = true).
func (s *Service) Display(ctx context.Context, a access.Access, actor audit.Actor, requestID, unitID string, on bool,
	in DisplayInput, key string) (MoveResult, bool, error) {
	if !a.Has(access.PermVehicleUnitDisplayManage) {
		return MoveResult{}, false, ErrForbidden
	}
	t := TypeDisplayOff
	if on {
		t = TypeDisplayOn
	}
	req := MoveRequest{UnitID: unitID, Type: t, ExpectedVersion: in.Version, IdempotencyKey: key, Note: in.Note, RequestID: requestID}
	if err := validateMove(req); err != nil {
		return MoveResult{}, false, err
	}
	if in.Version == nil {
		return MoveResult{}, false, &ValidationError{Fields: []FieldProblem{{"version", "is required"}}}
	}
	if _, err := uuid.Parse(unitID); err != nil {
		return MoveResult{}, false, ErrNotFound
	}
	return s.repo.Move(ctx, actor, a.Scope, req)
}
