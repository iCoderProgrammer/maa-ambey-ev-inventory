package catalogue

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/audit"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
)

// Repository persists the catalogue. Mutations check the version and the
// lifecycle rules inside their transaction and record the audit event there.
type Repository interface {
	ListBrands(ctx context.Context, f ListFilter) (Page[Brand], error)
	GetBrand(ctx context.Context, id string) (Brand, error)
	CreateBrand(ctx context.Context, in BrandCreateRequest, ev audit.Event) (Brand, error)
	UpdateBrand(ctx context.Context, id string, in BrandUpdateRequest, ev audit.Event) (Brand, error)
	RecordBrandAuthorisation(ctx context.Context, id string, in BrandAuthorisationRequest, verifierID string, ev audit.Event) (Brand, error)
	BrandReadiness(ctx context.Context, id string) (Readiness, error)
	ActivateBrand(ctx context.Context, id string, version int, ev audit.Event) (Brand, error)
	ArchiveBrand(ctx context.Context, id string, version int, ev audit.Event) (Brand, error)

	ListModels(ctx context.Context, f ListFilter) (Page[Model], error)
	GetModel(ctx context.Context, id string) (Model, error)
	CreateModel(ctx context.Context, in ModelCreateRequest, ev audit.Event) (Model, error)
	UpdateModel(ctx context.Context, id string, in ModelUpdateRequest, ev audit.Event) (Model, error)

	ListVariants(ctx context.Context, f ListFilter) (Page[Variant], error)
	GetVariant(ctx context.Context, id string) (Variant, error)
	CreateVariant(ctx context.Context, in VariantCreateRequest, ev audit.Event) (Variant, error)
	UpdateVariant(ctx context.Context, id string, in VariantUpdateRequest, ev audit.Event) (Variant, error)

	ListColours(ctx context.Context, f ListFilter) (Page[Colour], error)
	GetColour(ctx context.Context, id string) (Colour, error)
	CreateColour(ctx context.Context, in ColourCreateRequest, ev audit.Event) (Colour, error)
	UpdateColour(ctx context.Context, id string, in ColourUpdateRequest, ev audit.Event) (Colour, error)

	VerifyItem(ctx context.Context, k kind, id string, in VerifyRequest, verifierID string, ev audit.Event) error
	ActivateItem(ctx context.Context, k kind, id string, version int, ev audit.Event) error
	ArchiveItem(ctx context.Context, k kind, id string, version int, ev audit.Event) error

	RecordAudit(ctx context.Context, ev audit.Event) error
}

// Service applies permissions and validation, then calls the repository.
// Routes are guarded too; the service checks again so no caller can skip it.
type Service struct {
	repo Repository
	log  *slog.Logger
	now  func() time.Time
}

// NewService builds the service.
func NewService(repo Repository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log, now: time.Now}
}

func require(a access.Access, permission string) error {
	if !a.Has(permission) {
		return ErrForbidden
	}
	return nil
}

// validID: an ID that is not a UUID cannot exist (404 rather than a database error).
func validID(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return ErrNotFound
	}
	return nil
}

func event(actor Actor, action, entityType, entityID string) audit.Event {
	return audit.Event{
		Actor: audit.UserActor(actor.UserID, actor.Username), Action: action, EntityType: entityType,
		EntityID: entityID, RequestID: actor.RequestID,
	}
}

// ── Brands ───────────────────────────────────────────────────────────────────

// ListBrands lists brands (catalogue.view). Pending and archived brands are
// included for catalogue administrators (BR-008); filter by status as needed.
func (s *Service) ListBrands(ctx context.Context, a access.Access, q ListQuery) (Page[Brand], error) {
	if err := require(a, access.PermCatalogueView); err != nil {
		return Page[Brand]{}, err
	}
	f, err := validateList(q)
	if err != nil {
		return Page[Brand]{}, err
	}
	return s.repo.ListBrands(ctx, f)
}

// GetBrand returns one brand (catalogue.view).
func (s *Service) GetBrand(ctx context.Context, a access.Access, id string) (Brand, error) {
	if err := require(a, access.PermCatalogueView); err != nil {
		return Brand{}, err
	}
	if err := validID(id); err != nil {
		return Brand{}, err
	}
	return s.repo.GetBrand(ctx, id)
}

// CreateBrand adds a brand as data (catalogue.manage). It starts pending and unverified.
func (s *Service) CreateBrand(ctx context.Context, actor Actor, a access.Access, in BrandCreateRequest) (Brand, error) {
	if err := require(a, access.PermCatalogueManage); err != nil {
		return Brand{}, err
	}
	if err := validateBrandCreate(&in); err != nil {
		return Brand{}, err
	}
	return s.repo.CreateBrand(ctx, in, event(actor, "catalogue.brand.created", "brand", ""))
}

// UpdateBrand renames a brand or changes its slug (catalogue.manage).
func (s *Service) UpdateBrand(ctx context.Context, actor Actor, a access.Access, id string, in BrandUpdateRequest) (Brand, error) {
	if err := require(a, access.PermCatalogueManage); err != nil {
		return Brand{}, err
	}
	if err := validID(id); err != nil {
		return Brand{}, err
	}
	if err := validateBrandUpdate(&in); err != nil {
		return Brand{}, err
	}
	return s.repo.UpdateBrand(ctx, id, in, event(actor, "catalogue.brand.updated", "brand", id))
}

// RecordBrandAuthorisation records authorisation status and evidence
// (catalogue.brand.authorise). The caller becomes the verifier.
func (s *Service) RecordBrandAuthorisation(ctx context.Context, actor Actor, a access.Access, id string, in BrandAuthorisationRequest) (Brand, error) {
	if err := require(a, access.PermBrandAuthorise); err != nil {
		return Brand{}, err
	}
	if err := validID(id); err != nil {
		return Brand{}, err
	}
	if err := validateAuthorisation(&in); err != nil {
		return Brand{}, err
	}
	return s.repo.RecordBrandAuthorisation(ctx, id, in, actor.UserID,
		event(actor, "catalogue.brand.authorisation_recorded", "brand", id))
}

// BrandReadiness lists the purchase/sales activation prerequisites (catalogue.view).
func (s *Service) BrandReadiness(ctx context.Context, a access.Access, id string) (Readiness, error) {
	if err := require(a, access.PermCatalogueView); err != nil {
		return Readiness{}, err
	}
	if err := validID(id); err != nil {
		return Readiness{}, err
	}
	return s.repo.BrandReadiness(ctx, id)
}

// ActivateBrand enables a brand for purchase/sales (catalogue.brand.activate)
// when every prerequisite is met. A refused attempt is audited as DENIED.
func (s *Service) ActivateBrand(ctx context.Context, actor Actor, a access.Access, id string, in ActionRequest) (Brand, error) {
	if err := require(a, access.PermBrandActivate); err != nil {
		return Brand{}, err
	}
	if err := validID(id); err != nil {
		return Brand{}, err
	}
	if err := validateAction(in.Version); err != nil {
		return Brand{}, err
	}
	ev := event(actor, "catalogue.brand.activated", "brand", id)
	b, err := s.repo.ActivateBrand(ctx, id, in.Version, ev)
	s.auditBlocked(ctx, ev, err)
	return b, err
}

// ArchiveBrand archives a brand (catalogue.archive). Nothing is deleted.
func (s *Service) ArchiveBrand(ctx context.Context, actor Actor, a access.Access, id string, in ArchiveRequest) (Brand, error) {
	if err := require(a, access.PermCatalogueArchive); err != nil {
		return Brand{}, err
	}
	if err := validID(id); err != nil {
		return Brand{}, err
	}
	if err := validateArchive(&in); err != nil {
		return Brand{}, err
	}
	return s.repo.ArchiveBrand(ctx, id, in.Version, withReason(event(actor, "catalogue.brand.archived", "brand", id), in.Reason))
}

func withReason(ev audit.Event, reason *string) audit.Event {
	if reason != nil {
		ev.Summary = audit.Summary{"reason": *reason}
	}
	return ev
}

// auditBlocked records a refused activation (outcome DENIED) with the unmet checks.
func (s *Service) auditBlocked(ctx context.Context, ev audit.Event, err error) {
	var blocked *ActivationBlockedError
	if !errors.As(err, &blocked) {
		return
	}
	keys := make([]string, len(blocked.Unmet))
	for i, c := range blocked.Unmet {
		keys[i] = c.Key
	}
	ev.Outcome = audit.OutcomeDenied
	ev.Summary = audit.Summary{"unmet": keys}
	if aerr := s.repo.RecordAudit(context.WithoutCancel(ctx), ev); aerr != nil {
		s.log.ErrorContext(ctx, "audit of refused activation failed", slog.String("action", ev.Action), slog.Any("error", aerr))
	}
}

// ── Models, variants, colours ────────────────────────────────────────────────

// ListModels lists models (catalogue.view), optionally by brand.
func (s *Service) ListModels(ctx context.Context, a access.Access, q ListQuery) (Page[Model], error) {
	if err := require(a, access.PermCatalogueView); err != nil {
		return Page[Model]{}, err
	}
	f, err := validateList(q)
	if err != nil {
		return Page[Model]{}, err
	}
	return s.repo.ListModels(ctx, f)
}

// GetModel returns one model (catalogue.view).
func (s *Service) GetModel(ctx context.Context, a access.Access, id string) (Model, error) {
	if err := require(a, access.PermCatalogueView); err != nil {
		return Model{}, err
	}
	if err := validID(id); err != nil {
		return Model{}, err
	}
	return s.repo.GetModel(ctx, id)
}

// CreateModel adds a pending, unverified model to a brand (catalogue.manage).
func (s *Service) CreateModel(ctx context.Context, actor Actor, a access.Access, in ModelCreateRequest) (Model, error) {
	if err := require(a, access.PermCatalogueManage); err != nil {
		return Model{}, err
	}
	if err := validateModelCreate(&in); err != nil {
		return Model{}, err
	}
	return s.repo.CreateModel(ctx, in, event(actor, "catalogue.model.created", kindModel.entityType, ""))
}

// UpdateModel edits a pending model (catalogue.manage).
func (s *Service) UpdateModel(ctx context.Context, actor Actor, a access.Access, id string, in ModelUpdateRequest) (Model, error) {
	if err := require(a, access.PermCatalogueManage); err != nil {
		return Model{}, err
	}
	if err := validID(id); err != nil {
		return Model{}, err
	}
	if err := validateModelUpdate(&in); err != nil {
		return Model{}, err
	}
	return s.repo.UpdateModel(ctx, id, in, event(actor, "catalogue.model.updated", kindModel.entityType, id))
}

// ListVariants lists variants (catalogue.view), optionally by brand or model.
func (s *Service) ListVariants(ctx context.Context, a access.Access, q ListQuery) (Page[Variant], error) {
	if err := require(a, access.PermCatalogueView); err != nil {
		return Page[Variant]{}, err
	}
	f, err := validateList(q)
	if err != nil {
		return Page[Variant]{}, err
	}
	return s.repo.ListVariants(ctx, f)
}

// GetVariant returns one variant (catalogue.view).
func (s *Service) GetVariant(ctx context.Context, a access.Access, id string) (Variant, error) {
	if err := require(a, access.PermCatalogueView); err != nil {
		return Variant{}, err
	}
	if err := validID(id); err != nil {
		return Variant{}, err
	}
	return s.repo.GetVariant(ctx, id)
}

// CreateVariant adds a pending variant to a model (catalogue.manage).
func (s *Service) CreateVariant(ctx context.Context, actor Actor, a access.Access, in VariantCreateRequest) (Variant, error) {
	if err := require(a, access.PermCatalogueManage); err != nil {
		return Variant{}, err
	}
	if err := validateVariantCreate(&in); err != nil {
		return Variant{}, err
	}
	return s.repo.CreateVariant(ctx, in, event(actor, "catalogue.variant.created", kindVariant.entityType, ""))
}

// UpdateVariant edits a pending variant (catalogue.manage).
func (s *Service) UpdateVariant(ctx context.Context, actor Actor, a access.Access, id string, in VariantUpdateRequest) (Variant, error) {
	if err := require(a, access.PermCatalogueManage); err != nil {
		return Variant{}, err
	}
	if err := validID(id); err != nil {
		return Variant{}, err
	}
	if err := validateVariantUpdate(&in); err != nil {
		return Variant{}, err
	}
	return s.repo.UpdateVariant(ctx, id, in, event(actor, "catalogue.variant.updated", kindVariant.entityType, id))
}

// ListColours lists colours (catalogue.view), optionally by brand or variant.
func (s *Service) ListColours(ctx context.Context, a access.Access, q ListQuery) (Page[Colour], error) {
	if err := require(a, access.PermCatalogueView); err != nil {
		return Page[Colour]{}, err
	}
	f, err := validateList(q)
	if err != nil {
		return Page[Colour]{}, err
	}
	return s.repo.ListColours(ctx, f)
}

// GetColour returns one colour (catalogue.view).
func (s *Service) GetColour(ctx context.Context, a access.Access, id string) (Colour, error) {
	if err := require(a, access.PermCatalogueView); err != nil {
		return Colour{}, err
	}
	if err := validID(id); err != nil {
		return Colour{}, err
	}
	return s.repo.GetColour(ctx, id)
}

// CreateColour adds a pending colour to a variant (catalogue.manage).
func (s *Service) CreateColour(ctx context.Context, actor Actor, a access.Access, in ColourCreateRequest) (Colour, error) {
	if err := require(a, access.PermCatalogueManage); err != nil {
		return Colour{}, err
	}
	if err := validateColourCreate(&in); err != nil {
		return Colour{}, err
	}
	return s.repo.CreateColour(ctx, in, event(actor, "catalogue.colour.created", kindColour.entityType, ""))
}

// UpdateColour edits a pending colour (catalogue.manage).
func (s *Service) UpdateColour(ctx context.Context, actor Actor, a access.Access, id string, in ColourUpdateRequest) (Colour, error) {
	if err := require(a, access.PermCatalogueManage); err != nil {
		return Colour{}, err
	}
	if err := validID(id); err != nil {
		return Colour{}, err
	}
	if err := validateColourUpdate(&in); err != nil {
		return Colour{}, err
	}
	return s.repo.UpdateColour(ctx, id, in, event(actor, "catalogue.colour.updated", kindColour.entityType, id))
}

// Verify records provenance for a pending model, variant or colour (catalogue.verify).
func (s *Service) Verify(ctx context.Context, actor Actor, a access.Access, k kind, id string, in VerifyRequest) (any, error) {
	if err := require(a, access.PermCatalogueVerify); err != nil {
		return nil, err
	}
	if err := validID(id); err != nil {
		return nil, err
	}
	if err := validateVerify(&in, s.now()); err != nil {
		return nil, err
	}
	if err := s.repo.VerifyItem(ctx, k, id, in, actor.UserID, event(actor, "catalogue."+k.label+".verified", k.entityType, id)); err != nil {
		return nil, err
	}
	return s.fetch(ctx, k, id)
}

// Activate activates a verified model, variant or colour (catalogue.activate).
// A refused attempt is audited as DENIED.
func (s *Service) Activate(ctx context.Context, actor Actor, a access.Access, k kind, id string, in ActionRequest) (any, error) {
	if err := require(a, access.PermCatalogueActivate); err != nil {
		return nil, err
	}
	if err := validID(id); err != nil {
		return nil, err
	}
	if err := validateAction(in.Version); err != nil {
		return nil, err
	}
	ev := event(actor, "catalogue."+k.label+".activated", k.entityType, id)
	if err := s.repo.ActivateItem(ctx, k, id, in.Version, ev); err != nil {
		s.auditBlocked(ctx, ev, err)
		return nil, err
	}
	return s.fetch(ctx, k, id)
}

// Archive archives a model, variant or colour (catalogue.archive).
func (s *Service) Archive(ctx context.Context, actor Actor, a access.Access, k kind, id string, in ArchiveRequest) (any, error) {
	if err := require(a, access.PermCatalogueArchive); err != nil {
		return nil, err
	}
	if err := validID(id); err != nil {
		return nil, err
	}
	if err := validateArchive(&in); err != nil {
		return nil, err
	}
	if err := s.repo.ArchiveItem(ctx, k, id, in.Version, withReason(event(actor, "catalogue."+k.label+".archived", k.entityType, id), in.Reason)); err != nil {
		return nil, err
	}
	return s.fetch(ctx, k, id)
}

// fetch re-reads an item after a lifecycle action. The action's own
// permission covers returning the result, so catalogue.view is not required.
func (s *Service) fetch(ctx context.Context, k kind, id string) (any, error) {
	switch k {
	case kindModel:
		return s.repo.GetModel(ctx, id)
	case kindVariant:
		return s.repo.GetVariant(ctx, id)
	default:
		return s.repo.GetColour(ctx, id)
	}
}
