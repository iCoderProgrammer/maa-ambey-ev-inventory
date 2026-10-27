package stockitems

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/audit"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
)

// Repository persists categories and SKUs. Each mutation is one transaction
// with its audit event.
type Repository interface {
	ListCategories(ctx context.Context, f CategoryFilter) (Page[Category], error)
	CreateCategory(ctx context.Context, in CategoryInput, createdBy string, ev audit.Event) (Category, error)
	UpdateCategory(ctx context.Context, id string, p CategoryPatch, ev audit.Event) (Category, error)
	ArchiveCategory(ctx context.Context, id string, in ArchiveInput, actorID string, ev audit.Event) (Category, error)

	ListItems(ctx context.Context, f ListFilter) (Page[Item], error)
	GetItem(ctx context.Context, id string) (Detail, error)
	CreateItem(ctx context.Context, in ItemInput, createdBy string, ev audit.Event) (Detail, error)
	UpdateItem(ctx context.Context, id string, p ItemPatch, actorID string, ev audit.Event) (Detail, error)
	ArchiveItem(ctx context.Context, id string, in ArchiveInput, actorID string, ev audit.Event) (Detail, error)
}

// Service checks permissions (again, after the route guard) and input, and
// shapes what each caller may see. The SKU master is not location-specific
// (stock per location is E-04), so there is no location scope here.
type Service struct {
	repo Repository
	log  *slog.Logger
}

// NewService builds the service.
func NewService(repo Repository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func event(actor Actor, action, entityType, entityID string) audit.Event {
	return audit.Event{Actor: audit.UserActor(actor.UserID, actor.Username), Action: action,
		EntityType: entityType, EntityID: entityID, RequestID: actor.RequestID}
}

func need(a access.Access, perms ...string) error {
	for _, p := range perms {
		if !a.Has(p) {
			return ErrForbidden
		}
	}
	return nil
}

func validID(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return ErrNotFound
	}
	return nil
}

// linkPerms: naming suppliers needs suppliers.view, naming brands/models needs
// catalogue.view (a user cannot attach records they may not see).
func linkPerms(a access.Access, suppliers, compatibility bool) error {
	if suppliers && !a.Has(access.PermSupplierView) {
		return ErrForbidden
	}
	if compatibility && !a.Has(access.PermCatalogueView) {
		return ErrForbidden
	}
	return nil
}

// shape hides supplier links from callers without suppliers.view.
func shape(d Detail, a access.Access) Detail {
	if !a.Has(access.PermSupplierView) {
		d.Suppliers, d.SuppliersHidden = []SupplierLink{}, true
	}
	return d
}

// ListCategories lists categories (stock_items.view).
func (s *Service) ListCategories(ctx context.Context, a access.Access, q CategoryQuery) (Page[Category], error) {
	if err := need(a, access.PermStockItemView); err != nil {
		return Page[Category]{}, err
	}
	f, err := validateCategoryList(q)
	if err != nil {
		return Page[Category]{}, err
	}
	return s.repo.ListCategories(ctx, f)
}

// CreateCategory adds a category (stock_items.manage).
func (s *Service) CreateCategory(ctx context.Context, actor Actor, a access.Access, in CategoryInput) (Category, error) {
	if err := need(a, access.PermStockItemManage); err != nil {
		return Category{}, err
	}
	if err := validateCategory(&in); err != nil {
		return Category{}, err
	}
	return s.repo.CreateCategory(ctx, in, actor.UserID, event(actor, "stock_item_category.created", "stock_item_category", ""))
}

// UpdateCategory edits an active category (stock_items.manage).
func (s *Service) UpdateCategory(ctx context.Context, actor Actor, a access.Access, id string, p CategoryPatch) (Category, error) {
	if err := need(a, access.PermStockItemManage); err != nil {
		return Category{}, err
	}
	if err := validID(id); err != nil {
		return Category{}, err
	}
	if err := validateCategoryPatch(&p); err != nil {
		return Category{}, err
	}
	return s.repo.UpdateCategory(ctx, id, p, event(actor, "stock_item_category.updated", "stock_item_category", id))
}

// ArchiveCategory archives a category without active SKUs (stock_items.archive).
func (s *Service) ArchiveCategory(ctx context.Context, actor Actor, a access.Access, id string, in ArchiveInput) (Category, error) {
	if err := need(a, access.PermStockItemArchive); err != nil {
		return Category{}, err
	}
	if err := validID(id); err != nil {
		return Category{}, err
	}
	if err := validateArchive(&in); err != nil {
		return Category{}, err
	}
	return s.repo.ArchiveCategory(ctx, id, in, actor.UserID, event(actor, "stock_item_category.archived", "stock_item_category", id))
}

// ListItems lists SKUs (stock_items.view). A supplier filter needs suppliers.view.
func (s *Service) ListItems(ctx context.Context, a access.Access, q ListQuery) (Page[Item], error) {
	if err := need(a, access.PermStockItemView); err != nil {
		return Page[Item]{}, err
	}
	f, err := validateList(q)
	if err != nil {
		return Page[Item]{}, err
	}
	if f.SupplierID != "" && !a.Has(access.PermSupplierView) {
		return Page[Item]{}, ErrForbidden
	}
	return s.repo.ListItems(ctx, f)
}

// GetItem returns a SKU with its supplier links and compatibility
// (stock_items.view); archived SKUs stay readable.
func (s *Service) GetItem(ctx context.Context, a access.Access, id string) (Detail, error) {
	if err := need(a, access.PermStockItemView); err != nil {
		return Detail{}, err
	}
	if err := validID(id); err != nil {
		return Detail{}, err
	}
	d, err := s.repo.GetItem(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	return shape(d, a), nil
}

// CreateItem adds a SKU (stock_items.manage). Records no stock.
func (s *Service) CreateItem(ctx context.Context, actor Actor, a access.Access, in ItemInput) (Detail, error) {
	if err := need(a, access.PermStockItemManage); err != nil {
		return Detail{}, err
	}
	if err := validateItem(&in); err != nil {
		return Detail{}, err
	}
	if err := linkPerms(a, len(in.Suppliers) > 0, len(in.Compatibility) > 0); err != nil {
		return Detail{}, err
	}
	d, err := s.repo.CreateItem(ctx, in, actor.UserID, event(actor, "stock_item.created", "stock_item", ""))
	if err != nil {
		return Detail{}, err
	}
	return shape(d, a), nil
}

// UpdateItem edits an active SKU (stock_items.manage).
func (s *Service) UpdateItem(ctx context.Context, actor Actor, a access.Access, id string, p ItemPatch) (Detail, error) {
	if err := need(a, access.PermStockItemManage); err != nil {
		return Detail{}, err
	}
	if err := validID(id); err != nil {
		return Detail{}, err
	}
	if err := validateItemPatch(&p); err != nil {
		return Detail{}, err
	}
	if err := linkPerms(a, p.Suppliers != nil, p.Compatibility != nil); err != nil {
		return Detail{}, err
	}
	d, err := s.repo.UpdateItem(ctx, id, p, actor.UserID, event(actor, "stock_item.updated", "stock_item", id))
	if err != nil {
		return Detail{}, err
	}
	return shape(d, a), nil
}

// ArchiveItem archives a SKU (stock_items.archive). It stays readable, keeps its
// code, links and references, and can no longer change or be chosen.
func (s *Service) ArchiveItem(ctx context.Context, actor Actor, a access.Access, id string, in ArchiveInput) (Detail, error) {
	if err := need(a, access.PermStockItemArchive); err != nil {
		return Detail{}, err
	}
	if err := validID(id); err != nil {
		return Detail{}, err
	}
	if err := validateArchive(&in); err != nil {
		return Detail{}, err
	}
	d, err := s.repo.ArchiveItem(ctx, id, in, actor.UserID, event(actor, "stock_item.archived", "stock_item", id))
	if err != nil {
		return Detail{}, err
	}
	return shape(d, a), nil
}
