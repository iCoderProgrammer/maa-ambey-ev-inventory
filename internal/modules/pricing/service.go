package pricing

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
)

// Repository persists price lists and items.
type Repository interface {
	ListPriceLists(ctx context.Context, f ListFilter) (Page[PriceList], error)
	GetPriceList(ctx context.Context, id string) (PriceList, error)
	CreatePriceList(ctx context.Context, in PriceListCreateRequest, createdBy string, ev audit.Event) (PriceList, error)
	RenamePriceList(ctx context.Context, id string, in PriceListUpdateRequest, ev audit.Event) (PriceList, error)
	ArchivePriceList(ctx context.Context, id string, version int, ev audit.Event) (PriceList, error)

	ListItems(ctx context.Context, f ListFilter) (Page[Item], error)
	GetItem(ctx context.Context, id string) (Item, error)
	CreateItem(ctx context.Context, in itemInput, createdBy string, ev audit.Event) (Item, error)
	UpdateItem(ctx context.Context, id string, p itemPatch, ev audit.Event) (Item, error)
	CancelItem(ctx context.Context, id string, version int, ev audit.Event) (Item, error)
	ApproveItem(ctx context.Context, id string, in ApproveRequest, approverID string, now time.Time, ev, supersededEv audit.Event) (Item, error)
	Lookup(ctx context.Context, priceListID, variantID, priceType string, at time.Time) (Item, error)

	RecordAudit(ctx context.Context, ev audit.Event) error
}

// Service applies permissions and validation. Routes are guarded too; the
// service decides per price type (cost prices need the cost permissions).
type Service struct {
	repo Repository
	log  *slog.Logger
	now  func() time.Time
}

// NewService builds the service.
func NewService(repo Repository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log, now: time.Now}
}

func event(actor Actor, action, entityType, entityID string) audit.Event {
	return audit.Event{Actor: audit.UserActor(actor.UserID, actor.Username), Action: action,
		EntityType: entityType, EntityID: entityID, RequestID: actor.RequestID}
}

func validID(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return ErrNotFound
	}
	return nil
}

func canSeeAnyPrice(a access.Access) bool { return len(visibleTypes(a)) > 0 }

func canManageAnyPrice(a access.Access) bool {
	return a.Has(access.PermPriceManage) || a.Has(access.PermCostPriceEdit)
}

// ── Price lists ──────────────────────────────────────────────────────────────

// ListPriceLists lists price lists (any price view permission).
func (s *Service) ListPriceLists(ctx context.Context, a access.Access, q ListQuery) (Page[PriceList], error) {
	if !canSeeAnyPrice(a) {
		return Page[PriceList]{}, ErrForbidden
	}
	f, _, err := validateList(q)
	if err != nil {
		return Page[PriceList]{}, err
	}
	switch f.Status {
	case "", ListActive, ListArchived:
	default:
		return Page[PriceList]{}, invalid("status", "must be ACTIVE or ARCHIVED")
	}
	return s.repo.ListPriceLists(ctx, f)
}

// GetPriceList returns one list (any price view permission).
func (s *Service) GetPriceList(ctx context.Context, a access.Access, id string) (PriceList, error) {
	if !canSeeAnyPrice(a) {
		return PriceList{}, ErrForbidden
	}
	if err := validID(id); err != nil {
		return PriceList{}, err
	}
	return s.repo.GetPriceList(ctx, id)
}

// CreatePriceList adds a list for a brand (price or cost management permission).
func (s *Service) CreatePriceList(ctx context.Context, actor Actor, a access.Access, in PriceListCreateRequest) (PriceList, error) {
	if !canManageAnyPrice(a) {
		return PriceList{}, ErrForbidden
	}
	if err := validateListCreate(&in); err != nil {
		return PriceList{}, err
	}
	return s.repo.CreatePriceList(ctx, in, actor.UserID, event(actor, "pricing.price_list.created", "price_list", ""))
}

// RenamePriceList renames a list.
func (s *Service) RenamePriceList(ctx context.Context, actor Actor, a access.Access, id string, in PriceListUpdateRequest) (PriceList, error) {
	if !canManageAnyPrice(a) {
		return PriceList{}, ErrForbidden
	}
	if err := validID(id); err != nil {
		return PriceList{}, err
	}
	if err := validateListUpdate(&in); err != nil {
		return PriceList{}, err
	}
	return s.repo.RenamePriceList(ctx, id, in, event(actor, "pricing.price_list.renamed", "price_list", id))
}

// ArchivePriceList archives a list; its approved history stays readable.
func (s *Service) ArchivePriceList(ctx context.Context, actor Actor, a access.Access, id string, in VersionRequest) (PriceList, error) {
	if !canManageAnyPrice(a) {
		return PriceList{}, ErrForbidden
	}
	if err := validID(id); err != nil {
		return PriceList{}, err
	}
	if in.Version < 1 {
		return PriceList{}, invalid("version", "is required: send the version you last read")
	}
	return s.repo.ArchivePriceList(ctx, id, in.Version, event(actor, "pricing.price_list.archived", "price_list", id))
}

// ── Price items ──────────────────────────────────────────────────────────────

// ListItems lists price history; only price types the caller may see.
func (s *Service) ListItems(ctx context.Context, a access.Access, q ListQuery) (Page[Item], error) {
	f, requested, err := validateList(q)
	if err != nil {
		return Page[Item]{}, err
	}
	switch f.Status {
	case "", StatusDraft, StatusApproved, StatusCancelled:
	default:
		return Page[Item]{}, invalid("status", "must be DRAFT, APPROVED or CANCELLED")
	}
	if f.Types, err = restrictTypes(a, requested); err != nil {
		return Page[Item]{}, err
	}
	return s.repo.ListItems(ctx, f)
}

// GetItem returns one price if the caller may see its type (otherwise 404).
func (s *Service) GetItem(ctx context.Context, a access.Access, id string) (Item, error) {
	if !canSeeAnyPrice(a) {
		return Item{}, ErrForbidden
	}
	if err := validID(id); err != nil {
		return Item{}, err
	}
	it, err := s.repo.GetItem(ctx, id)
	if err != nil {
		return Item{}, err
	}
	if !a.Has(viewPermission(it.PriceType)) {
		return Item{}, ErrNotFound
	}
	return it, nil
}

// CreateItem drafts a price (edit permission of its type).
func (s *Service) CreateItem(ctx context.Context, actor Actor, a access.Access, r ItemCreateRequest) (Item, error) {
	if !canManageAnyPrice(a) {
		return Item{}, ErrForbidden
	}
	in, err := validateItemCreate(r)
	if (in.PriceType == TypeSelling || in.PriceType == TypeCost) && !a.Has(editPermission(in.PriceType)) {
		return Item{}, ErrForbidden
	}
	if err != nil {
		return Item{}, err
	}
	return s.repo.CreateItem(ctx, in, actor.UserID, event(actor, "pricing.price_item.drafted", "price_item", ""))
}

// loadForChange returns the item after checking the edit permission of its type.
func (s *Service) loadForChange(ctx context.Context, a access.Access, id string, perm func(string) string) (Item, error) {
	if err := validID(id); err != nil {
		return Item{}, err
	}
	it, err := s.repo.GetItem(ctx, id)
	if err != nil {
		return Item{}, err
	}
	if !a.Has(viewPermission(it.PriceType)) {
		return Item{}, ErrNotFound
	}
	if !a.Has(perm(it.PriceType)) {
		return Item{}, ErrForbidden
	}
	return it, nil
}

// UpdateItem edits a draft.
func (s *Service) UpdateItem(ctx context.Context, actor Actor, a access.Access, id string, r ItemUpdateRequest) (Item, error) {
	if !canManageAnyPrice(a) {
		return Item{}, ErrForbidden
	}
	if err := validID(id); err != nil {
		return Item{}, err
	}
	p, err := validateItemUpdate(r)
	if err != nil {
		return Item{}, err
	}
	if _, err := s.loadForChange(ctx, a, id, editPermission); err != nil {
		return Item{}, err
	}
	return s.repo.UpdateItem(ctx, id, p, event(actor, "pricing.price_item.updated", "price_item", id))
}

// CancelItem cancels a draft (kept as history).
func (s *Service) CancelItem(ctx context.Context, actor Actor, a access.Access, id string, in VersionRequest) (Item, error) {
	if !canManageAnyPrice(a) {
		return Item{}, ErrForbidden
	}
	if err := validID(id); err != nil {
		return Item{}, err
	}
	if in.Version < 1 {
		return Item{}, invalid("version", "is required: send the version you last read")
	}
	if _, err := s.loadForChange(ctx, a, id, editPermission); err != nil {
		return Item{}, err
	}
	return s.repo.CancelItem(ctx, id, in.Version, event(actor, "pricing.price_item.cancelled", "price_item", id))
}

// ApproveItem approves a draft: catalogue.price.approve plus view of its type;
// never the drafter's own. Refusals by the rules are audited as DENIED.
func (s *Service) ApproveItem(ctx context.Context, actor Actor, a access.Access, id string, in ApproveRequest) (Item, error) {
	if !a.Has(access.PermPriceApprove) {
		return Item{}, ErrForbidden
	}
	if err := validID(id); err != nil {
		return Item{}, err
	}
	if err := validateApprove(&in); err != nil {
		return Item{}, err
	}
	if _, err := s.loadForChange(ctx, a, id, func(string) string { return access.PermPriceApprove }); err != nil {
		return Item{}, err
	}
	ev := event(actor, "pricing.price_item.approved", "price_item", id)
	it, err := s.repo.ApproveItem(ctx, id, in, actor.UserID, s.now().UTC(), ev,
		event(actor, "pricing.price_item.superseded", "price_item", ""))
	if err != nil {
		s.auditRefusal(ctx, ev, err)
	}
	return it, err
}

func (s *Service) auditRefusal(ctx context.Context, ev audit.Event, err error) {
	var (
		blocked *BlockedError
		overlap *OverlapError
		verr    *ValidationError
	)
	var reason string
	switch {
	case errors.Is(err, ErrSelfApproval):
		reason = "self_approval"
	case errors.As(err, &blocked):
		keys := make([]string, len(blocked.Unmet))
		for i, u := range blocked.Unmet {
			keys[i] = u.Field
		}
		reason = strings.Join(keys, ",")
	case errors.As(err, &overlap):
		reason = "overlap"
	case errors.As(err, &verr):
		reason = "invalid:" + verr.Fields[0].Field
	default:
		return
	}
	ev.Outcome = audit.OutcomeDenied
	ev.Summary = audit.Summary{"reason": reason}
	if aerr := s.repo.RecordAudit(context.WithoutCancel(ctx), ev); aerr != nil {
		s.log.ErrorContext(ctx, "audit of refused price approval failed", slog.Any("error", aerr))
	}
}

// LookupQuery is the raw lookup query.
type LookupQuery struct {
	PriceListID, VariantID, PriceType, At string
}

// Lookup returns the approved price valid at the instant (default: now).
// Quotations snapshot this answer (SAL-003); later approvals do not change it.
func (s *Service) Lookup(ctx context.Context, a access.Access, q LookupQuery) (Item, error) {
	var c checker
	c.id("priceListId", &q.PriceListID, true)
	c.id("variantId", &q.VariantID, true)
	switch q.PriceType {
	case TypeSelling, TypeCost:
	case "":
		c.add("priceType", "is required")
	default:
		c.add("priceType", "must be SELLING or COST")
	}
	at := s.now()
	if strings.TrimSpace(q.At) != "" {
		at = parseTime(&c, "at", q.At)
	}
	if q.PriceType != "" && (q.PriceType == TypeSelling || q.PriceType == TypeCost) && !a.Has(viewPermission(q.PriceType)) {
		return Item{}, ErrForbidden
	}
	if err := c.err(); err != nil {
		return Item{}, err
	}
	return s.repo.Lookup(ctx, q.PriceListID, q.VariantID, q.PriceType, at)
}
