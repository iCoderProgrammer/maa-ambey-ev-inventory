package stock

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
)

// Stock count use cases (E-05). Permissions are checked again here after the
// route guard: stock.adjustment.view reads, stock.adjustment.create creates,
// records and submits, stock.adjustment.approve approves and rejects. Every
// count belongs to one location in the caller's scope (outside it: not found).

// CountListQuery is the raw list query.
type CountListQuery struct{ Status, LocationID, Page, Limit string }

// ListCounts lists counts at the caller's locations, newest first; status=SUBMITTED
// is the approval queue.
func (s *Service) ListCounts(ctx context.Context, a access.Access, q CountListQuery) (CountPage, error) {
	if !a.Has(access.PermAdjustmentView) {
		return CountPage{}, ErrForbidden
	}
	f := CountListFilter{Status: strings.TrimSpace(q.Status), LocationID: strings.TrimSpace(q.LocationID), Page: 1, Limit: 25}
	var probs []FieldProblem
	switch f.Status {
	case "", CountDraft, CountSubmitted, CountApproved, CountRejected:
	default:
		probs = append(probs, FieldProblem{"status", "must be DRAFT, SUBMITTED, APPROVED or REJECTED"})
	}
	if f.LocationID != "" {
		if _, err := uuid.Parse(f.LocationID); err != nil {
			probs = append(probs, FieldProblem{"locationId", "must be a valid ID"})
		} else if !a.Scope.Allows(f.LocationID) {
			probs = append(probs, FieldProblem{"locationId", "is not one of your locations"})
		}
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
		return CountPage{}, &ValidationError{Fields: probs}
	}
	ids, all, err := a.Scope.Restrict(nil)
	if err != nil {
		return CountPage{}, err
	}
	f.ScopeLocationIDs, f.AllLocations = ids, all
	return s.repo.ListCounts(ctx, f)
}

// GetCount reads a count with its lines (stock.adjustment.view).
func (s *Service) GetCount(ctx context.Context, a access.Access, id string) (StockCount, error) {
	if !a.Has(access.PermAdjustmentView) {
		return StockCount{}, ErrForbidden
	}
	if _, err := uuid.Parse(id); err != nil {
		return StockCount{}, ErrNotFound
	}
	return s.repo.GetCount(ctx, a.Scope, id)
}

// CreateCount starts a DRAFT count (stock.adjustment.create); Idempotency-Key required.
func (s *Service) CreateCount(ctx context.Context, a access.Access, actor audit.Actor, requestID string, in CountCreateInput,
	key string) (StockCount, bool, error) {
	if !a.Has(access.PermAdjustmentCreate) {
		return StockCount{}, false, ErrForbidden
	}
	check := in
	if err := validateCreate(&check, key); err != nil {
		return StockCount{}, false, err
	}
	return s.repo.CreateCount(ctx, actor, a.Scope, requestID, in, key)
}

// RecordLine records one counted item on a draft (stock.adjustment.create).
func (s *Service) RecordLine(ctx context.Context, a access.Access, actor audit.Actor, requestID, id string,
	in CountLineInput) (StockCount, error) {
	if !a.Has(access.PermAdjustmentCreate) {
		return StockCount{}, ErrForbidden
	}
	if _, err := uuid.Parse(id); err != nil {
		return StockCount{}, ErrNotFound
	}
	check := in
	if err := validateLine(&check); err != nil {
		return StockCount{}, err
	}
	return s.repo.RecordLine(ctx, actor, a.Scope, requestID, id, in)
}

// SubmitCount submits a draft for approval (stock.adjustment.create).
func (s *Service) SubmitCount(ctx context.Context, a access.Access, actor audit.Actor, requestID, id string,
	in CountVersionInput) (StockCount, error) {
	if !a.Has(access.PermAdjustmentCreate) {
		return StockCount{}, ErrForbidden
	}
	if _, err := uuid.Parse(id); err != nil {
		return StockCount{}, ErrNotFound
	}
	if err := problems(versionProblem(in.Version)); err != nil {
		return StockCount{}, err
	}
	return s.repo.SubmitCount(ctx, actor, a.Scope, requestID, id, in)
}

// ApproveCount approves a submitted count and posts its parts adjustments
// (stock.adjustment.approve; never one's own). Idempotency-Key required.
func (s *Service) ApproveCount(ctx context.Context, a access.Access, actor audit.Actor, requestID, id string, in CountApproveInput,
	key string) (StockCount, bool, error) {
	if !a.Has(access.PermAdjustmentApprove) {
		return StockCount{}, false, ErrForbidden
	}
	in.Note = trimOpt(in.Note)
	if err := validateDecision(in.Version, key, in.Note, "note", false); err != nil {
		return StockCount{}, false, err
	}
	return s.decide(ctx, a, actor, requestID, id, true, in.Version, in.Note, key)
}

// RejectCount rejects a submitted count with a reason; nothing is posted
// (stock.adjustment.approve; never one's own). Idempotency-Key required.
func (s *Service) RejectCount(ctx context.Context, a access.Access, actor audit.Actor, requestID, id string, in CountRejectInput,
	key string) (StockCount, bool, error) {
	if !a.Has(access.PermAdjustmentApprove) {
		return StockCount{}, false, ErrForbidden
	}
	reason := trimOpt(&in.Reason)
	if err := validateDecision(in.Version, key, reason, "reason", true); err != nil {
		return StockCount{}, false, err
	}
	return s.decide(ctx, a, actor, requestID, id, false, in.Version, reason, key)
}

func (s *Service) decide(ctx context.Context, a access.Access, actor audit.Actor, requestID, id string, approve bool, version *int,
	text *string, key string) (StockCount, bool, error) {
	if _, err := uuid.Parse(id); err != nil {
		return StockCount{}, false, ErrNotFound
	}
	c, replayed, err := s.repo.DecideCount(ctx, actor, a.Scope, requestID, id, approve, version, text, key)
	if err != nil {
		// Refusals by the rules are audited (DENIED), after the rollback.
		reason := ""
		var changed *StockChangedError
		switch {
		case errors.Is(err, ErrSelfDecision):
			reason = "self_approval"
		case errors.As(err, &changed):
			reason = "stock_changed_since_count"
		}
		if reason != "" {
			action := "approved"
			if !approve {
				action = "rejected"
			}
			if aerr := s.repo.RecordCountRefusal(context.WithoutCancel(ctx), actor, requestID, id, action, reason); aerr != nil {
				s.log.ErrorContext(ctx, "audit of refused stock count decision failed", slog.Any("error", aerr))
			}
		}
	}
	return c, replayed, err
}
