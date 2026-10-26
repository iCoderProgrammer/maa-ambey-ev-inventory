package vehicles

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
)

// Repository reads vehicle units. Registration is RegisterFromReceiptUnit,
// which runs inside the posting transaction rather than through this interface.
type Repository interface {
	List(ctx context.Context, f ListFilter) (Page[Unit], error)
	GetDetail(ctx context.Context, id string) (Detail, error)
}

// Service checks permissions (again, after the route guard), the caller's
// location scope, and shapes what each caller may see.
type Service struct {
	repo Repository
	log  *slog.Logger
}

// NewService builds the service.
func NewService(repo Repository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// List lists units (vehicle_units.view) in the caller's location scope: units
// without a location, and units at a location the caller may work in. A
// location filter outside the scope is refused.
func (s *Service) List(ctx context.Context, a access.Access, q ListQuery) (Page[Unit], error) {
	if !a.Has(access.PermVehicleUnitView) {
		return Page[Unit]{}, ErrForbidden
	}
	f, err := validateList(q)
	if err != nil {
		return Page[Unit]{}, err
	}
	requested := f.LocationIDs
	ids, all, err := a.Scope.Restrict(requested)
	if errors.Is(err, access.ErrOutOfScope) {
		return Page[Unit]{}, &ValidationError{Fields: []FieldProblem{{"locationId", "is not one of your locations"}}}
	}
	if err != nil {
		return Page[Unit]{}, err
	}
	if len(requested) == 0 {
		f.LocationIDs, f.ScopeLocationIDs, f.AllLocations = nil, ids, all
	}
	p, err := s.repo.List(ctx, f)
	if err != nil {
		return Page[Unit]{}, err
	}
	for i := range p.Items {
		shapeCost(&p.Items[i], a)
	}
	return p, nil
}

// GetDetail returns one unit with its source and history (vehicle_units.view).
// A unit at a location outside the caller's scope is reported as not found.
func (s *Service) GetDetail(ctx context.Context, a access.Access, id string) (Detail, error) {
	if !a.Has(access.PermVehicleUnitView) {
		return Detail{}, ErrForbidden
	}
	if _, err := uuid.Parse(id); err != nil {
		return Detail{}, ErrNotFound
	}
	d, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	if d.LocationID != nil && !a.Scope.Allows(*d.LocationID) {
		return Detail{}, ErrNotFound
	}
	shapeCost(&d.Unit, a)
	return d, nil
}

// shapeCost removes the acquisition cost for callers without vehicle_units.cost.view.
func shapeCost(u *Unit, a access.Access) {
	if a.Has(access.PermVehicleUnitCostView) {
		return
	}
	u.UnitCost, u.CostCurrency, u.CostHidden = nil, nil, true
}
