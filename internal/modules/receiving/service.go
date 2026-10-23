package receiving

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/audit"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
)

// Repository persists goods receipts.
type Repository interface {
	List(ctx context.Context, f ListFilter) (Page[Receipt], error)
	ListReceivable(ctx context.Context, f ListFilter) (Page[ReceivablePO], error)
	GetDetail(ctx context.Context, id string) (Detail, error)
	Create(ctx context.Context, in CreateInput, createdBy, idemKey string, ev audit.Event) (id string, replayed bool, err error)
	// Save replaces a draft's header, lines and units. canOverride says whether
	// the caller holds goods_receipts.over_receipt.
	Save(ctx context.Context, id string, in SaveInput, lines []indexedLine, actorID string, canOverride bool, ev audit.Event) error
	Cancel(ctx context.Context, id string, in CancelInput, actorID string, ev audit.Event) error
	AddDocument(ctx context.Context, id string, in DocumentInput, actorID string, ev audit.Event) error
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
		EntityType: "goods_receipt", EntityID: entityID, RequestID: actor.RequestID}
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

// List lists receipts (goods_receipts.view).
func (s *Service) List(ctx context.Context, a access.Access, q ListQuery) (Page[Receipt], error) {
	if err := need(a, access.PermGoodsReceiptView); err != nil {
		return Page[Receipt]{}, err
	}
	f, err := validateList(q)
	if err != nil {
		return Page[Receipt]{}, err
	}
	return s.repo.List(ctx, f)
}

// ListReceivable lists purchase orders goods can be received against
// (goods_receipts.manage: it is the first step of a new receipt).
func (s *Service) ListReceivable(ctx context.Context, a access.Access, query, page, limit string) (Page[ReceivablePO], error) {
	if err := need(a, access.PermGoodsReceiptManage); err != nil {
		return Page[ReceivablePO]{}, err
	}
	f, err := validateReceivable(query, page, limit)
	if err != nil {
		return Page[ReceivablePO]{}, err
	}
	return s.repo.ListReceivable(ctx, f)
}

// GetDetail returns a receipt with every PO line, units, documents, history,
// warnings and the caller's actions (goods_receipts.view).
func (s *Service) GetDetail(ctx context.Context, a access.Access, id string) (Detail, error) {
	if err := need(a, access.PermGoodsReceiptView); err != nil {
		return Detail{}, err
	}
	if err := validID(id); err != nil {
		return Detail{}, err
	}
	d, err := s.repo.GetDetail(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	return shape(d, a), nil
}

func shape(d Detail, a access.Access) Detail {
	unitWarnings(d.Lines)
	for i := range d.Lines {
		d.Lines[i].Warnings = lineWarnings(d.Lines[i])
	}
	d.StockPosted = d.Status == StatusPosted
	d.Actions = actionsFor(d, a)
	return d
}

func (s *Service) after(ctx context.Context, a access.Access, id string, err error) (Detail, error) {
	if err != nil {
		return Detail{}, err
	}
	return s.GetDetail(ctx, a, id)
}

// Create starts a draft receipt against an approved PO (goods_receipts.manage).
// The same Idempotency-Key from the same user returns the first receipt.
func (s *Service) Create(ctx context.Context, actor Actor, a access.Access, in CreateInput, idemKey string) (Detail, bool, error) {
	if err := need(a, access.PermGoodsReceiptManage); err != nil {
		return Detail{}, false, err
	}
	key, err := validateIdempotencyKey(idemKey)
	if err != nil {
		return Detail{}, false, err
	}
	if err := validateCreate(&in); err != nil {
		return Detail{}, false, err
	}
	id, replayed, err := s.repo.Create(ctx, in, actor.UserID, key, event(actor, "goods_receipt.created", ""))
	d, err := s.after(ctx, a, id, err)
	return d, replayed, err
}

// Save replaces a draft's header, lines and units (goods_receipts.manage;
// over-receipt authorisations need goods_receipts.over_receipt). A refused
// authorisation is audited as DENIED. Nothing changes stock.
func (s *Service) Save(ctx context.Context, actor Actor, a access.Access, id string, in SaveInput) (Detail, error) {
	if err := need(a, access.PermGoodsReceiptManage); err != nil {
		return Detail{}, err
	}
	if err := validID(id); err != nil {
		return Detail{}, err
	}
	lines, err := validateSave(&in)
	if err != nil {
		return Detail{}, err
	}
	err = s.repo.Save(ctx, id, in, lines, actor.UserID, a.Has(access.PermGoodsReceiptOverReceipt), event(actor, "goods_receipt.saved", id))
	var forbidden *OverReceiptForbiddenError
	if errors.As(err, &forbidden) {
		ev := event(actor, "goods_receipt.over_receipt_authorised", id)
		ev.Outcome = audit.OutcomeDenied
		fields := make([]string, len(forbidden.Fields))
		for i, f := range forbidden.Fields {
			fields[i] = f.Field
		}
		ev.Summary = audit.Summary{"reason": "missing_permission", "fields": strings.Join(fields, ",")}
		if aerr := s.repo.RecordAudit(context.WithoutCancel(ctx), ev); aerr != nil {
			s.log.ErrorContext(ctx, "audit of refused over-receipt failed", slog.Any("error", aerr))
		}
	}
	return s.after(ctx, a, id, err)
}

// Cancel cancels a draft with a reason (goods_receipts.manage). The receipt,
// its lines and units stay as history; its quantities no longer hold PO stock.
func (s *Service) Cancel(ctx context.Context, actor Actor, a access.Access, id string, in CancelInput) (Detail, error) {
	if err := need(a, access.PermGoodsReceiptManage); err != nil {
		return Detail{}, err
	}
	if err := validID(id); err != nil {
		return Detail{}, err
	}
	if err := validateCancel(&in); err != nil {
		return Detail{}, err
	}
	return s.after(ctx, a, id, s.repo.Cancel(ctx, id, in, actor.UserID, event(actor, "goods_receipt.cancelled", id)))
}

// AddDocument links one of the caller's own unlinked uploads to a draft
// (goods_receipts.manage and attachments.upload). Downloading it later needs
// attachments.view and goods_receipts.view.
func (s *Service) AddDocument(ctx context.Context, actor Actor, a access.Access, id string, in DocumentInput) (Detail, error) {
	if err := need(a, access.PermGoodsReceiptManage, access.PermAttachmentsUpload); err != nil {
		return Detail{}, err
	}
	if err := validID(id); err != nil {
		return Detail{}, err
	}
	if err := validateDocument(&in); err != nil {
		return Detail{}, err
	}
	return s.after(ctx, a, id, s.repo.AddDocument(ctx, id, in, actor.UserID, event(actor, "goods_receipt.document_added", id)))
}
