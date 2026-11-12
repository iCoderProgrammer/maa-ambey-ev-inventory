package vehicles

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/attachments"
)

// Repository reads vehicle units. Registration is RegisterFromReceiptUnit,
// which runs inside the posting transaction rather than through this interface.
type Repository interface {
	List(ctx context.Context, f ListFilter) (Page[Unit], error)
	GetDetail(ctx context.Context, id string) (Detail, error)
	// UnitLocation returns the unit's location (nil = none) or ErrNotFound.
	UnitLocation(ctx context.Context, id string) (*string, error)
	Movements(ctx context.Context, f MovementFilter) (Page[Movement], error)
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

// GetDetail returns one unit with its source, history and the documents the
// caller may open (vehicle_units.view). A unit at a location outside the
// caller's scope is reported as not found.
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
	visible := make([]Document, 0, len(d.Documents))
	for _, doc := range d.Documents {
		if attachments.CanSee(a, attachments.Attachment{ID: doc.AttachmentID, OwnerType: doc.ownerType,
			UploadedBy: doc.uploadedBy, LocationID: doc.locationID}) {
			visible = append(visible, doc)
		}
	}
	d.Documents, d.DocumentsHidden = visible, !a.Has(access.PermAttachmentsView)
	return d, nil
}

// Movements returns one page of a unit's stock movements (vehicle_units.view),
// oldest first unless order=desc. A unit outside the caller's scope is not
// found; a location outside the scope is named by ID only.
func (s *Service) Movements(ctx context.Context, a access.Access, id string, q MovementQuery) (Page[Movement], error) {
	if !a.Has(access.PermVehicleUnitView) {
		return Page[Movement]{}, ErrForbidden
	}
	if _, err := uuid.Parse(id); err != nil {
		return Page[Movement]{}, ErrNotFound
	}
	f, err := validateMovements(id, q)
	if err != nil {
		return Page[Movement]{}, err
	}
	loc, err := s.repo.UnitLocation(ctx, id)
	if err != nil {
		return Page[Movement]{}, err
	}
	if loc != nil && !a.Scope.Allows(*loc) {
		return Page[Movement]{}, ErrNotFound
	}
	p, err := s.repo.Movements(ctx, f)
	if err != nil {
		return Page[Movement]{}, err
	}
	for i := range p.Items {
		m := &p.Items[i]
		if m.FromLocationID != nil && !a.Scope.Allows(*m.FromLocationID) {
			m.FromLocationCode, m.FromLocationName = nil, nil
		}
		if m.ToLocationID != nil && !a.Scope.Allows(*m.ToLocationID) {
			m.ToLocationCode, m.ToLocationName = nil, nil
		}
	}
	return p, nil
}

// shapeCost removes the acquisition cost for callers without vehicle_units.cost.view.
func shapeCost(u *Unit, a access.Access) {
	if a.Has(access.PermVehicleUnitCostView) {
		return
	}
	u.UnitCost, u.CostCurrency, u.CostHidden = nil, nil, true
}
