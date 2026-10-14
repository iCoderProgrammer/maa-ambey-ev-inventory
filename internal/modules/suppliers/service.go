package suppliers

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/audit"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
)

// Repository persists suppliers.
type Repository interface {
	ListSuppliers(ctx context.Context, f ListFilter) (Page[Supplier], error)
	GetDetail(ctx context.Context, id string) (Detail, error)
	CreateSupplier(ctx context.Context, in SupplierInput, createdBy string, ev audit.Event) (Supplier, error)
	UpdateSupplier(ctx context.Context, id string, p SupplierPatch, ev audit.Event) (Supplier, error)
	VerifySupplier(ctx context.Context, id string, in VerifyInput, verifier string, ev audit.Event) (Supplier, error)
	ArchiveSupplier(ctx context.Context, id string, in ArchiveInput, ev audit.Event) (Supplier, error)

	AddContact(ctx context.Context, supplierID string, in ContactInput, ev audit.Event) (Contact, error)
	UpdateContact(ctx context.Context, supplierID, contactID string, p ContactPatch, ev audit.Event) (Contact, error)
	RemoveContact(ctx context.Context, supplierID, contactID string, version int, ev audit.Event) (Contact, error)

	AddDocument(ctx context.Context, supplierID string, in DocumentInput, actorID string, ev audit.Event) (Document, error)

	ListLinks(ctx context.Context, q LinkQuery) ([]BrandLink, error)
	CreateLink(ctx context.Context, supplierID string, in LinkInput, actorID string, ev audit.Event) (BrandLink, error)
	VerifyLink(ctx context.Context, id string, in LinkVerifyInput, verifier string, ev audit.Event) (BrandLink, error)
	EndLink(ctx context.Context, id string, in LinkEndInput, today time.Time, ev audit.Event) (BrandLink, error)
}

// Service checks permissions (again, after the route guard) and input.
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

// ListSuppliers lists suppliers (suppliers.view).
func (s *Service) ListSuppliers(ctx context.Context, a access.Access, q ListQuery) (Page[Supplier], error) {
	if err := need(a, access.PermSupplierView); err != nil {
		return Page[Supplier]{}, err
	}
	f, err := validateList(q)
	if err != nil {
		return Page[Supplier]{}, err
	}
	return s.repo.ListSuppliers(ctx, f)
}

// GetDetail returns a supplier with contacts, documents and relationships (suppliers.view).
func (s *Service) GetDetail(ctx context.Context, a access.Access, id string) (Detail, error) {
	if err := need(a, access.PermSupplierView); err != nil {
		return Detail{}, err
	}
	if err := validID(id); err != nil {
		return Detail{}, err
	}
	return s.repo.GetDetail(ctx, id)
}

// CreateSupplier adds a supplier, pending verification (suppliers.manage).
func (s *Service) CreateSupplier(ctx context.Context, actor Actor, a access.Access, in SupplierInput) (Supplier, error) {
	if err := need(a, access.PermSupplierManage); err != nil {
		return Supplier{}, err
	}
	if err := validateSupplier(&in); err != nil {
		return Supplier{}, err
	}
	return s.repo.CreateSupplier(ctx, in, actor.UserID, event(actor, "supplier.created", "supplier", ""))
}

// UpdateSupplier edits a supplier (suppliers.manage).
func (s *Service) UpdateSupplier(ctx context.Context, actor Actor, a access.Access, id string, p SupplierPatch) (Supplier, error) {
	if err := need(a, access.PermSupplierManage); err != nil {
		return Supplier{}, err
	}
	if err := validID(id); err != nil {
		return Supplier{}, err
	}
	if err := validateSupplierPatch(&p); err != nil {
		return Supplier{}, err
	}
	return s.repo.UpdateSupplier(ctx, id, p, event(actor, "supplier.updated", "supplier", id))
}

// VerifySupplier records the verification with a reference and supplier document (suppliers.verify).
func (s *Service) VerifySupplier(ctx context.Context, actor Actor, a access.Access, id string, in VerifyInput) (Supplier, error) {
	if err := need(a, access.PermSupplierVerify); err != nil {
		return Supplier{}, err
	}
	if err := validID(id); err != nil {
		return Supplier{}, err
	}
	if err := validateVerify(&in); err != nil {
		return Supplier{}, err
	}
	return s.repo.VerifySupplier(ctx, id, in, actor.UserID, event(actor, "supplier.verified", "supplier", id))
}

// ArchiveSupplier archives a supplier (suppliers.archive).
func (s *Service) ArchiveSupplier(ctx context.Context, actor Actor, a access.Access, id string, in ArchiveInput) (Supplier, error) {
	if err := need(a, access.PermSupplierArchive); err != nil {
		return Supplier{}, err
	}
	if err := validID(id); err != nil {
		return Supplier{}, err
	}
	if err := validateArchive(&in); err != nil {
		return Supplier{}, err
	}
	return s.repo.ArchiveSupplier(ctx, id, in, event(actor, "supplier.archived", "supplier", id))
}

// AddContact adds a business contact (suppliers.manage).
func (s *Service) AddContact(ctx context.Context, actor Actor, a access.Access, supplierID string, in ContactInput) (Contact, error) {
	if err := need(a, access.PermSupplierManage); err != nil {
		return Contact{}, err
	}
	if err := validID(supplierID); err != nil {
		return Contact{}, err
	}
	if err := validateContact(&in); err != nil {
		return Contact{}, err
	}
	return s.repo.AddContact(ctx, supplierID, in, event(actor, "supplier.contact_added", "supplier", supplierID))
}

// UpdateContact edits a contact (suppliers.manage).
func (s *Service) UpdateContact(ctx context.Context, actor Actor, a access.Access, supplierID, contactID string, p ContactPatch) (Contact, error) {
	if err := need(a, access.PermSupplierManage); err != nil {
		return Contact{}, err
	}
	if err := validID(supplierID, contactID); err != nil {
		return Contact{}, err
	}
	if err := validateContactPatch(&p); err != nil {
		return Contact{}, err
	}
	return s.repo.UpdateContact(ctx, supplierID, contactID, p, event(actor, "supplier.contact_updated", "supplier", supplierID))
}

// RemoveContact deactivates a contact (suppliers.manage).
func (s *Service) RemoveContact(ctx context.Context, actor Actor, a access.Access, supplierID, contactID string, in VersionInput) (Contact, error) {
	if err := need(a, access.PermSupplierManage); err != nil {
		return Contact{}, err
	}
	if err := validID(supplierID, contactID); err != nil {
		return Contact{}, err
	}
	if in.Version < 1 {
		return Contact{}, invalid("version", "is required: send the version you last read")
	}
	return s.repo.RemoveContact(ctx, supplierID, contactID, in.Version, event(actor, "supplier.contact_removed", "supplier", supplierID))
}

// AddDocument links one of the caller's private uploads to the supplier
// (suppliers.manage and attachments.upload, since it publishes a file to
// everyone who may view suppliers and attachments).
func (s *Service) AddDocument(ctx context.Context, actor Actor, a access.Access, supplierID string, in DocumentInput) (Document, error) {
	if err := need(a, access.PermSupplierManage); err != nil {
		return Document{}, err
	}
	if err := need(a, access.PermAttachmentsUpload); err != nil {
		return Document{}, err
	}
	if err := validID(supplierID); err != nil {
		return Document{}, err
	}
	if err := validateDocument(&in); err != nil {
		return Document{}, err
	}
	return s.repo.AddDocument(ctx, supplierID, in, actor.UserID, event(actor, "supplier.document_added", "supplier", supplierID))
}

// ListLinks lists supplier-brand relationships by brand or supplier (suppliers.view).
func (s *Service) ListLinks(ctx context.Context, a access.Access, q LinkQuery) ([]BrandLink, error) {
	if err := need(a, access.PermSupplierView); err != nil {
		return nil, err
	}
	if err := validateLinkQuery(&q); err != nil {
		return nil, err
	}
	return s.repo.ListLinks(ctx, q)
}

// CreateLink proposes a relationship with a brand (suppliers.manage). It is
// unverified and proves nothing until verified with evidence.
func (s *Service) CreateLink(ctx context.Context, actor Actor, a access.Access, supplierID string, in LinkInput) (BrandLink, error) {
	if err := need(a, access.PermSupplierManage); err != nil {
		return BrandLink{}, err
	}
	if err := validID(supplierID); err != nil {
		return BrandLink{}, err
	}
	if err := validateLink(&in); err != nil {
		return BrandLink{}, err
	}
	return s.repo.CreateLink(ctx, supplierID, in, actor.UserID, event(actor, "supplier.brand_link_proposed", "supplier_brand_link", ""))
}

// VerifyLink verifies a relationship with evidence (suppliers.verify).
func (s *Service) VerifyLink(ctx context.Context, actor Actor, a access.Access, id string, in LinkVerifyInput) (BrandLink, error) {
	if err := need(a, access.PermSupplierVerify); err != nil {
		return BrandLink{}, err
	}
	if err := validID(id); err != nil {
		return BrandLink{}, err
	}
	if err := validateLinkVerify(&in); err != nil {
		return BrandLink{}, err
	}
	return s.repo.VerifyLink(ctx, id, in, actor.UserID, event(actor, "supplier.brand_link_verified", "supplier_brand_link", id))
}

// EndLink ends a relationship; it stays as history (suppliers.archive).
func (s *Service) EndLink(ctx context.Context, actor Actor, a access.Access, id string, in LinkEndInput) (BrandLink, error) {
	if err := need(a, access.PermSupplierArchive); err != nil {
		return BrandLink{}, err
	}
	if err := validID(id); err != nil {
		return BrandLink{}, err
	}
	if err := validateLinkEnd(&in); err != nil {
		return BrandLink{}, err
	}
	return s.repo.EndLink(ctx, id, in, s.now(), event(actor, "supplier.brand_link_ended", "supplier_brand_link", id))
}
