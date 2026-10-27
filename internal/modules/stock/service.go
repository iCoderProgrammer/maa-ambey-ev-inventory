package stock

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
)

// Repository runs a movement in its own transaction.
type Repository interface {
	Move(ctx context.Context, actor audit.Actor, scope access.Scope, req MoveRequest) (MoveResult, bool, error)
	PostParts(ctx context.Context, actor audit.Actor, scope access.Scope, req PartsRequest) (PartsResult, bool, error)
	PartsBalances(ctx context.Context, itemID string, scopeIDs []string, all bool) ([]PartsBalance, error)
	PartsHistory(ctx context.Context, f PartsHistoryFilter) (PartsPage, error)
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

// hideSupplier removes the supplier from movements for callers without suppliers.view.
func hideSupplier(m *PartsMovement, a access.Access) {
	if !a.Has(access.PermSupplierView) {
		m.SupplierID, m.SupplierName = nil, nil
	}
}

// ReceiveParts records a supplier receipt (stock.parts.receive; naming the
// supplier also needs suppliers.view) at a location in the caller's scope.
func (s *Service) ReceiveParts(ctx context.Context, a access.Access, actor audit.Actor, requestID, itemID string,
	in PartsReceiptInput, key string) (PartsResult, bool, error) {
	if !a.Has(access.PermPartsReceive) || !a.Has(access.PermSupplierView) {
		return PartsResult{}, false, ErrForbidden
	}
	return s.postParts(ctx, a, actor, PartsRequest{Type: TypePartsReceipt, ItemID: itemID, LocationID: in.LocationID,
		Quantity: in.Quantity, UOM: in.UOM, SupplierID: &in.SupplierID, SourceReference: &in.SourceReference, Note: in.Note,
		IdempotencyKey: key, RequestID: requestID})
}

// IssueParts records an issue/consumption with its reason (stock.parts.issue).
func (s *Service) IssueParts(ctx context.Context, a access.Access, actor audit.Actor, requestID, itemID string,
	in PartsIssueInput, key string) (PartsResult, bool, error) {
	if !a.Has(access.PermPartsIssue) {
		return PartsResult{}, false, ErrForbidden
	}
	return s.postParts(ctx, a, actor, PartsRequest{Type: TypePartsIssue, ItemID: itemID, LocationID: in.LocationID,
		Quantity: in.Quantity, UOM: in.UOM, Reason: &in.Reason, SourceReference: in.SourceReference, Note: in.Note,
		IdempotencyKey: key, RequestID: requestID})
}

func (s *Service) postParts(ctx context.Context, a access.Access, actor audit.Actor, req PartsRequest) (PartsResult, bool, error) {
	if _, err := uuid.Parse(req.ItemID); err != nil {
		return PartsResult{}, false, ErrNotFound
	}
	check := req
	if err := validateParts(&check); err != nil {
		return PartsResult{}, false, err
	}
	res, replayed, err := s.repo.PostParts(ctx, actor, a.Scope, req)
	if err == nil {
		hideSupplier(&res.Movement, a)
	}
	return res, replayed, err
}

// PartsBalances lists a SKU's balances at the caller's locations (stock.parts.view).
func (s *Service) PartsBalances(ctx context.Context, a access.Access, itemID string) ([]PartsBalance, error) {
	if !a.Has(access.PermPartsView) {
		return nil, ErrForbidden
	}
	if _, err := uuid.Parse(itemID); err != nil {
		return nil, ErrNotFound
	}
	ids, all, err := a.Scope.Restrict(nil)
	if err != nil {
		return nil, err
	}
	return s.repo.PartsBalances(ctx, itemID, ids, all)
}

// PartsHistoryQuery is the raw history query.
type PartsHistoryQuery struct{ LocationID, Type, Page, Limit string }

// PartsHistory lists a SKU's receipts and issues at the caller's locations,
// newest first (stock.parts.view). A location filter outside the scope is refused.
func (s *Service) PartsHistory(ctx context.Context, a access.Access, itemID string, q PartsHistoryQuery) (PartsPage, error) {
	if !a.Has(access.PermPartsView) {
		return PartsPage{}, ErrForbidden
	}
	if _, err := uuid.Parse(itemID); err != nil {
		return PartsPage{}, ErrNotFound
	}
	f := PartsHistoryFilter{ItemID: itemID, LocationID: strings.TrimSpace(q.LocationID), Type: q.Type, Page: 1, Limit: 25}
	var probs []FieldProblem
	if f.LocationID != "" {
		if _, err := uuid.Parse(f.LocationID); err != nil {
			probs = append(probs, FieldProblem{"locationId", "must be a valid ID"})
		} else if !a.Scope.Allows(f.LocationID) {
			probs = append(probs, FieldProblem{"locationId", "is not one of your locations"})
		}
	}
	switch f.Type {
	case "", TypePartsReceipt, TypePartsIssue:
	default:
		probs = append(probs, FieldProblem{"type", "must be PARTS_RECEIPT or PARTS_ISSUE"})
	}
	if q.Page != "" {
		if n, err := strconv.Atoi(q.Page); err != nil || n < 1 || n > 100000 {
			probs = append(probs, FieldProblem{"page", "must be a whole number from 1"})
		} else {
			f.Page = n
		}
	}
	if q.Limit != "" {
		if n, err := strconv.Atoi(q.Limit); err != nil || n < 1 || n > 100 {
			probs = append(probs, FieldProblem{"limit", "must be a whole number from 1 to 100"})
		} else {
			f.Limit = n
		}
	}
	if len(probs) > 0 {
		return PartsPage{}, &ValidationError{Fields: probs}
	}
	ids, all, err := a.Scope.Restrict(nil)
	if err != nil {
		return PartsPage{}, err
	}
	f.ScopeLocationIDs, f.AllLocations = ids, all
	p, err := s.repo.PartsHistory(ctx, f)
	for i := range p.Items {
		hideSupplier(&p.Items[i], a)
	}
	return p, err
}
