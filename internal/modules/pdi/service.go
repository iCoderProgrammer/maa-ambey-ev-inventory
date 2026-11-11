package pdi

import (
	"context"
	"errors"
	"log/slog"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
)

// Repository is the data access the service needs (PGStore in production).
type Repository interface {
	// Overview reads the unit (ErrNotFound outside scope), its applicable
	// active checklist and its inspections, newest first.
	Overview(ctx context.Context, scope access.Scope, unitID string) (UnitState, []ChecklistItem, []Inspection, error)
	// Inspection reads one inspection with outcomes and evidence (ErrNotFound
	// when its unit is outside scope).
	Inspection(ctx context.Context, scope access.Scope, id string) (InspectionDetail, error)
	// Record records an inspection and its movement in one transaction
	// (replayed = true when the actor's key already recorded this request).
	Record(ctx context.Context, actor audit.Actor, scope access.Scope, requestID, unitID string, r normalisedRecord,
		key, hash string) (RecordResult, bool, error)
	// RecordRefusal audits a refused disposition (DENIED).
	RecordRefusal(ctx context.Context, actor audit.Actor, requestID, unitID, disposition, reason string) error
	ListChecklist(ctx context.Context, f checklistFilter) (ChecklistPage, error)
	CreateChecklistItem(ctx context.Context, actor audit.Actor, requestID string, in normalisedItem) (ChecklistItem, error)
	RetireChecklistItem(ctx context.Context, actor audit.Actor, requestID, id string, version int, reason string) (ChecklistItem, error)
}

// Service holds the PDI rules that are not database constraints.
type Service struct {
	repo Repository
	log  *slog.Logger
}

// NewService builds the service.
func NewService(repo Repository, log *slog.Logger) *Service { return &Service{repo: repo, log: log} }

func hideEvidence(d *InspectionDetail, a access.Access) {
	if !a.Has(access.PermAttachmentsView) {
		d.Evidence = []Evidence{}
		d.EvidenceHidden = true
	}
}

// Overview is the unit's PDI state, applicable checklist and history.
func (s *Service) Overview(ctx context.Context, a access.Access, unitID string) (Overview, error) {
	if !a.Has(access.PermVehicleUnitView) {
		return Overview{}, ErrForbidden
	}
	if !isUUID(unitID) {
		return Overview{}, ErrNotFound
	}
	u, items, history, err := s.repo.Overview(ctx, a.Scope, unitID)
	if err != nil {
		return Overview{}, err
	}
	return Overview{Unit: u, AllowedDispositions: AllowedDispositions(u.Status), CanRecord: a.Has(access.PermPDIRecord),
		Checklist: items, Inspections: history}, nil
}

// Inspection is one inspection with its outcomes (and evidence with attachments.view).
func (s *Service) Inspection(ctx context.Context, a access.Access, id string) (InspectionDetail, error) {
	if !a.Has(access.PermVehicleUnitView) {
		return InspectionDetail{}, ErrForbidden
	}
	if !isUUID(id) {
		return InspectionDetail{}, ErrNotFound
	}
	d, err := s.repo.Inspection(ctx, a.Scope, id)
	if err != nil {
		return InspectionDetail{}, err
	}
	hideEvidence(&d, a)
	return d, nil
}

// Record records an inspection of a unit and applies its disposition.
// Refusals because of the unit's status or a missing checklist are audited.
func (s *Service) Record(ctx context.Context, a access.Access, actor audit.Actor, requestID, unitID string, in RecordInput,
	key string) (RecordResult, bool, error) {
	if !a.Has(access.PermPDIRecord) {
		return RecordResult{}, false, ErrForbidden
	}
	if !isUUID(unitID) {
		return RecordResult{}, false, ErrNotFound
	}
	r, err := validateRecord(in, key)
	if err != nil {
		return RecordResult{}, false, err
	}
	res, replayed, err := s.repo.Record(ctx, actor, a.Scope, requestID, unitID, r, key, requestHash(unitID, r))
	var state *StateError
	switch {
	case errors.As(err, &state):
		s.refused(ctx, actor, requestID, unitID, r.Disposition, "status "+state.Status+" does not admit "+r.Disposition)
	case errors.Is(err, ErrChecklistNotConfigured):
		s.refused(ctx, actor, requestID, unitID, r.Disposition, "no checklist item applies to the unit")
	case errors.Is(err, ErrChassisRequired):
		s.refused(ctx, actor, requestID, unitID, r.Disposition, "the unit has no chassis/frame number")
	}
	if err != nil {
		return RecordResult{}, false, err
	}
	hideEvidence(&res.Inspection, a)
	return res, replayed, nil
}

func (s *Service) refused(ctx context.Context, actor audit.Actor, requestID, unitID, disposition, reason string) {
	if err := s.repo.RecordRefusal(ctx, actor, requestID, unitID, disposition, reason); err != nil {
		s.log.ErrorContext(ctx, "audit PDI refusal failed", slog.String("request_id", requestID), slog.Any("error", err))
	}
}

// ListChecklist lists checklist items (active first by order, then retired).
func (s *Service) ListChecklist(ctx context.Context, a access.Access, q ChecklistQuery) (ChecklistPage, error) {
	if !a.Has(access.PermVehicleUnitView) && !a.Has(access.PermPDIRecord) && !a.Has(access.PermPDIChecklistManage) {
		return ChecklistPage{}, ErrForbidden
	}
	f, err := validateChecklistQuery(q)
	if err != nil {
		return ChecklistPage{}, err
	}
	return s.repo.ListChecklist(ctx, f)
}

// CreateChecklistItem adds an active checklist item.
func (s *Service) CreateChecklistItem(ctx context.Context, a access.Access, actor audit.Actor, requestID string,
	in ChecklistCreateInput) (ChecklistItem, error) {
	if !a.Has(access.PermPDIChecklistManage) {
		return ChecklistItem{}, ErrForbidden
	}
	n, err := validateChecklistCreate(in)
	if err != nil {
		return ChecklistItem{}, err
	}
	return s.repo.CreateChecklistItem(ctx, actor, requestID, n)
}

// RetireChecklistItem retires an active item; recorded outcomes keep pointing at it.
func (s *Service) RetireChecklistItem(ctx context.Context, a access.Access, actor audit.Actor, requestID, id string,
	in ChecklistRetireInput) (ChecklistItem, error) {
	if !a.Has(access.PermPDIChecklistManage) {
		return ChecklistItem{}, ErrForbidden
	}
	if !isUUID(id) {
		return ChecklistItem{}, ErrNotFound
	}
	v, reason, err := validateRetire(in)
	if err != nil {
		return ChecklistItem{}, err
	}
	return s.repo.RetireChecklistItem(ctx, actor, requestID, id, v, reason)
}
