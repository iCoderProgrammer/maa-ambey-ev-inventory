package purchasing

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
)

// Repository persists purchase orders.
type Repository interface {
	List(ctx context.Context, f ListFilter) (Page[PurchaseOrder], error)
	GetSnapshot(ctx context.Context, id string) (snapshot, error)
	Create(ctx context.Context, in CreateInput, createdBy, idemKey string, ev audit.Event) (id string, replayed bool, err error)
	UpdateHeader(ctx context.Context, id string, p HeaderPatch, ev audit.Event) error
	AddLine(ctx context.Context, id string, version int, in LineInput, ev audit.Event) error
	UpdateLine(ctx context.Context, id, lineID string, p LinePatch, ev audit.Event) error
	RemoveLine(ctx context.Context, id, lineID string, version int, ev audit.Event) error
	Submit(ctx context.Context, id string, version int, actorID string, ev audit.Event) error
	Approve(ctx context.Context, id string, in ApproveInput, approverID string, ev audit.Event) error
	Return(ctx context.Context, id string, in ReasonInput, actorID string, ev audit.Event) error
	Cancel(ctx context.Context, id string, in ReasonInput, actorID string, ev audit.Event) error
	GetSettings(ctx context.Context) (Settings, error)
	UpdateSettings(ctx context.Context, in SettingsInput, actorID string, ev audit.Event) (Settings, error)
	RecordAudit(ctx context.Context, ev audit.Event) error
}

// Service checks permissions (again, after the route guard) and input, and
// shapes what each caller may see.
type Service struct {
	repo Repository
	log  *slog.Logger
}

// NewService builds the service.
func NewService(repo Repository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func event(actor Actor, action, entityID string) audit.Event {
	return audit.Event{Actor: audit.UserActor(actor.UserID, actor.Username), Action: action,
		EntityType: "purchase_order", EntityID: entityID, RequestID: actor.RequestID}
}

func need(a access.Access, perm string) error {
	if !a.Has(perm) {
		return ErrForbidden
	}
	return nil
}

func validID(ids ...string) error {
	for _, id := range ids {
		if _, err := uuid.Parse(id); err != nil {
			return ErrNotFound
		}
	}
	return nil
}

// List lists purchase orders (purchase_orders.view).
func (s *Service) List(ctx context.Context, a access.Access, q ListQuery) (Page[PurchaseOrder], error) {
	if err := need(a, access.PermPurchaseOrderView); err != nil {
		return Page[PurchaseOrder]{}, err
	}
	f, err := validateList(q)
	if err != nil {
		return Page[PurchaseOrder]{}, err
	}
	return s.repo.List(ctx, f)
}

// GetDetail returns a PO with lines, history, open prerequisites and the
// caller's actions (purchase_orders.view). Catalogue cost snapshots are shown
// only with catalogue.cost_price.view.
func (s *Service) GetDetail(ctx context.Context, a access.Access, id string) (Detail, error) {
	if err := need(a, access.PermPurchaseOrderView); err != nil {
		return Detail{}, err
	}
	if err := validID(id); err != nil {
		return Detail{}, err
	}
	sn, err := s.repo.GetSnapshot(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	return shape(sn, a), nil
}

// shape turns a stored snapshot into what this caller sees.
func shape(sn snapshot, a access.Access) Detail {
	d := sn.Detail
	seeCost := a.Has(access.PermCostPriceView)
	for i := range d.Lines {
		l := &d.Lines[i]
		if !seeCost && l.ReferencePriceItemID != nil {
			l.ReferencePriceListID, l.ReferencePriceItemID, l.ReferenceUnitCost, l.ReferenceCapturedAt, l.CostVariance = nil, nil, nil, nil, nil
			l.ReferenceHidden = true
		}
		l.Warnings = lineWarnings(*l)
	}
	d.Blockers = []Check{}
	if d.Status == StatusDraft || d.Status == StatusSubmitted {
		if unmet := prerequisites(sn.Facts); unmet != nil {
			d.Blockers = unmet
		}
	}
	if d.Status == StatusDraft {
		required, reason := approvalRequirement(d.TotalAmount, sn.Settings, sn.SupplierHadApprovedPO)
		d.ApprovalRequired, d.ApprovalReason = &required, &reason
	}
	d.Actions = actionsFor(d.PurchaseOrder, a)
	return d
}

func (s *Service) after(ctx context.Context, a access.Access, id string, err error) (Detail, error) {
	if err != nil {
		return Detail{}, err
	}
	return s.GetDetail(ctx, a, id)
}

func needCostView(a access.Access, lines ...LineInput) error {
	for _, l := range lines {
		if l.PriceListID != nil && !a.Has(access.PermCostPriceView) {
			return ErrForbidden
		}
	}
	return nil
}

// Create starts a draft PO with optional lines (purchase_orders.manage). The
// same Idempotency-Key from the same user returns the first PO (replayed).
func (s *Service) Create(ctx context.Context, actor Actor, a access.Access, in CreateInput, idemKey string) (Detail, bool, error) {
	if err := need(a, access.PermPurchaseOrderManage); err != nil {
		return Detail{}, false, err
	}
	key, err := validateIdempotencyKey(idemKey)
	if err != nil {
		return Detail{}, false, err
	}
	if err := validateCreate(&in); err != nil {
		return Detail{}, false, err
	}
	if err := needCostView(a, in.Lines...); err != nil {
		return Detail{}, false, err
	}
	id, replayed, err := s.repo.Create(ctx, in, actor.UserID, key, event(actor, "purchase_order.created", ""))
	d, err := s.after(ctx, a, id, err)
	return d, replayed, err
}

// UpdateHeader edits a draft's supplier, expected date or notes (purchase_orders.manage).
func (s *Service) UpdateHeader(ctx context.Context, actor Actor, a access.Access, id string, p HeaderPatch) (Detail, error) {
	if err := need(a, access.PermPurchaseOrderManage); err != nil {
		return Detail{}, err
	}
	if err := validID(id); err != nil {
		return Detail{}, err
	}
	if err := validateHeaderPatch(&p); err != nil {
		return Detail{}, err
	}
	return s.after(ctx, a, id, s.repo.UpdateHeader(ctx, id, p, event(actor, "purchase_order.updated", id)))
}

// AddLine adds a line to a draft (purchase_orders.manage; a reference price list
// also needs catalogue.cost_price.view).
func (s *Service) AddLine(ctx context.Context, actor Actor, a access.Access, id string, in LineAddInput) (Detail, error) {
	if err := need(a, access.PermPurchaseOrderManage); err != nil {
		return Detail{}, err
	}
	if err := validID(id); err != nil {
		return Detail{}, err
	}
	l, err := validateLineAdd(&in)
	if err != nil {
		return Detail{}, err
	}
	if err := needCostView(a, l); err != nil {
		return Detail{}, err
	}
	return s.after(ctx, a, id, s.repo.AddLine(ctx, id, in.Version, l, event(actor, "purchase_order.line_added", id)))
}

// UpdateLine edits a draft line (purchase_orders.manage).
func (s *Service) UpdateLine(ctx context.Context, actor Actor, a access.Access, id, lineID string, p LinePatch) (Detail, error) {
	if err := need(a, access.PermPurchaseOrderManage); err != nil {
		return Detail{}, err
	}
	if err := validID(id, lineID); err != nil {
		return Detail{}, err
	}
	if err := validateLinePatch(&p); err != nil {
		return Detail{}, err
	}
	if p.PriceListID.Value != nil && !a.Has(access.PermCostPriceView) {
		return Detail{}, ErrForbidden
	}
	return s.after(ctx, a, id, s.repo.UpdateLine(ctx, id, lineID, p, event(actor, "purchase_order.line_updated", id)))
}

// RemoveLine removes a draft line (purchase_orders.manage).
func (s *Service) RemoveLine(ctx context.Context, actor Actor, a access.Access, id, lineID string, in VersionInput) (Detail, error) {
	if err := need(a, access.PermPurchaseOrderManage); err != nil {
		return Detail{}, err
	}
	if err := validID(id, lineID); err != nil {
		return Detail{}, err
	}
	if in.Version < 1 {
		return Detail{}, invalid("version", "is required: send the version you last read")
	}
	return s.after(ctx, a, id, s.repo.RemoveLine(ctx, id, lineID, in.Version, event(actor, "purchase_order.line_removed", id)))
}

// Submit submits a draft (purchase_orders.manage).
func (s *Service) Submit(ctx context.Context, actor Actor, a access.Access, id string, in VersionInput) (Detail, error) {
	if err := need(a, access.PermPurchaseOrderManage); err != nil {
		return Detail{}, err
	}
	if err := validID(id); err != nil {
		return Detail{}, err
	}
	if in.Version < 1 {
		return Detail{}, invalid("version", "is required: send the version you last read")
	}
	return s.after(ctx, a, id, s.repo.Submit(ctx, id, in.Version, actor.UserID, event(actor, "purchase_order.submitted", id)))
}

// Approve approves a submitted PO (purchase_orders.approve; never one's own).
// Refusals by the rules are audited as DENIED.
func (s *Service) Approve(ctx context.Context, actor Actor, a access.Access, id string, in ApproveInput) (Detail, error) {
	if err := need(a, access.PermPurchaseOrderApprove); err != nil {
		return Detail{}, err
	}
	if err := validID(id); err != nil {
		return Detail{}, err
	}
	if err := validateApprove(&in); err != nil {
		return Detail{}, err
	}
	ev := event(actor, "purchase_order.approved", id)
	err := s.repo.Approve(ctx, id, in, actor.UserID, ev)
	if err != nil {
		s.auditRefusal(ctx, ev, err)
	}
	return s.after(ctx, a, id, err)
}

func (s *Service) auditRefusal(ctx context.Context, ev audit.Event, err error) {
	var blocked *BlockedError
	var reason string
	switch {
	case errors.Is(err, ErrSelfApproval):
		reason = "self_approval"
	case errors.As(err, &blocked):
		keys := make([]string, len(blocked.Unmet))
		for i, u := range blocked.Unmet {
			keys[i] = u.Key
		}
		reason = strings.Join(keys, ",")
	default:
		return
	}
	ev.Outcome = audit.OutcomeDenied
	ev.Summary = audit.Summary{"reason": reason}
	if aerr := s.repo.RecordAudit(context.WithoutCancel(ctx), ev); aerr != nil {
		s.log.ErrorContext(ctx, "audit of refused purchase order approval failed", slog.Any("error", aerr))
	}
}

// Return sends a submitted PO back to draft with a reason (purchase_orders.approve).
func (s *Service) Return(ctx context.Context, actor Actor, a access.Access, id string, in ReasonInput) (Detail, error) {
	if err := need(a, access.PermPurchaseOrderApprove); err != nil {
		return Detail{}, err
	}
	if err := validID(id); err != nil {
		return Detail{}, err
	}
	if err := validateReason(&in); err != nil {
		return Detail{}, err
	}
	return s.after(ctx, a, id, s.repo.Return(ctx, id, in, actor.UserID, event(actor, "purchase_order.returned", id)))
}

// Cancel cancels a PO with nothing received (purchase_orders.cancel).
func (s *Service) Cancel(ctx context.Context, actor Actor, a access.Access, id string, in ReasonInput) (Detail, error) {
	if err := need(a, access.PermPurchaseOrderCancel); err != nil {
		return Detail{}, err
	}
	if err := validID(id); err != nil {
		return Detail{}, err
	}
	if err := validateReason(&in); err != nil {
		return Detail{}, err
	}
	return s.after(ctx, a, id, s.repo.Cancel(ctx, id, in, actor.UserID, event(actor, "purchase_order.cancelled", id)))
}

// GetSettings returns the approval settings (purchase_orders.view or settings.manage).
func (s *Service) GetSettings(ctx context.Context, a access.Access) (Settings, error) {
	if !a.Has(access.PermPurchaseOrderView) && !a.Has(access.PermPurchaseOrderSettings) {
		return Settings{}, ErrForbidden
	}
	return s.repo.GetSettings(ctx)
}

// UpdateSettings changes the approval settings (purchase_orders.settings.manage).
func (s *Service) UpdateSettings(ctx context.Context, actor Actor, a access.Access, in SettingsInput) (Settings, error) {
	if err := need(a, access.PermPurchaseOrderSettings); err != nil {
		return Settings{}, err
	}
	if err := validateSettings(&in); err != nil {
		return Settings{}, err
	}
	ev := event(actor, "purchase_order.settings_updated", "settings")
	ev.EntityType = "purchase_order_settings"
	return s.repo.UpdateSettings(ctx, in, actor.UserID, ev)
}
