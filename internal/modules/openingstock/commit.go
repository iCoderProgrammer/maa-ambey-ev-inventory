package openingstock

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/ageing"
)

// G-07: approving and committing a G-06 preview batch (ADR-031).
//
// A batch becomes stock only through POST /opening-stock-imports/{id}/commit,
// once, all rows or none, after its approval (physical count reconciliation,
// POST …/approval) is recorded. The commit never trusts the preview: it reads
// the stored file again (SHA-256 checked), parses it, compares it with the
// staged rows and re-validates every row against the database inside the
// committing transaction, under locks.

// Commit-time issue codes (added to the G-06 row codes).
const (
	// CodeWarningNotAccepted: the row has preview warnings; no approved rule
	// accepts warnings at commit (OQ-49), so they must be resolved first.
	CodeWarningNotAccepted = "WARNING_NOT_ACCEPTED"
	// CodeStatusNotApproved: the stock status is not one the owner approved for
	// opening stock (OQ-47, OPENING_STOCK_APPROVED_STATUSES).
	CodeStatusNotApproved = "STATUS_NOT_APPROVED"
	// CodeCostNotPositive: a vehicle unit needs an acquisition cost above zero.
	CodeCostNotPositive = "COST_NOT_POSITIVE"
	// CodePreviewRowNotEligible: the line could not proceed at preview; a corrected
	// file must be uploaded (a new batch) and approved.
	CodePreviewRowNotEligible = "PREVIEW_ROW_NOT_ELIGIBLE"
	// CodeChangedSincePreview: the line now resolves to another brand, model,
	// variant, location, chassis or status than its approved preview.
	CodeChangedSincePreview = "CHANGED_SINCE_PREVIEW"
)

// BatchCommitted is the batch state reported once its commit exists.
const BatchCommitted = "COMMITTED"

// Commit errors (the HTTP adapter maps each to its code).
var (
	ErrNotCommittable      = errors.New("batch cannot be approved or committed")
	ErrPreviewExpired      = errors.New("preview expired")
	ErrApprovalRequired    = errors.New("approval required")
	ErrAlreadyApproved     = errors.New("batch already approved differently")
	ErrApprovalBlocked     = errors.New("batch has rows that cannot be imported")
	ErrStatusesNotApproved = errors.New("no opening-stock status is approved")
	ErrSourceInvalid       = errors.New("source file missing or changed")
)

// AlreadyCommittedError: the batch was committed (by another request).
type AlreadyCommittedError struct{ Commit CommitResult }

func (e *AlreadyCommittedError) Error() string { return "batch already committed" }

// CommitBlockedError lists the rows that stop the commit; nothing was created.
type CommitBlockedError struct{ Rows []RowRefusal }

func (e *CommitBlockedError) Error() string { return "commit blocked by rows" }

// DuplicateAtCommitError: a chassis was registered by another transaction
// between the re-validation and the insert (the unique key refused it).
type DuplicateAtCommitError struct{ RowNumber int }

func (e *DuplicateAtCommitError) Error() string { return "duplicate chassis at commit" }

// RowRefusal is one line that cannot be committed and why.
type RowRefusal struct {
	RowNumber int     `json:"rowNumber" example:"5"`
	Issues    []Issue `json:"issues"`
} //@name OpeningStockCommitRowRefusal

// Approval records that a batch was reconciled to a signed physical count.
type Approval struct {
	ID                     string    `json:"id"`
	ApprovedByID           string    `json:"approvedById"`
	ApprovedByName         *string   `json:"approvedByName" extensions:"x-nullable"`
	ApprovedAt             time.Time `json:"approvedAt"`
	PhysicalCountReference string    `json:"physicalCountReference" example:"PC-2026-11-15"`
	PhysicalCountDate      string    `json:"physicalCountDate" example:"2026-11-15"`
	MappingsReviewed       bool      `json:"mappingsReviewed"`
	DuplicatesReviewed     bool      `json:"duplicatesReviewed"`
	// ReceivedDatesApproved: the file's received dates are the agreed opening-stock
	// dates (LED-008); false keeps the units' age MISSING_ORIGIN (E-07).
	ReceivedDatesApproved bool    `json:"receivedDatesApproved"`
	Note                  *string `json:"note" extensions:"x-nullable"`
} //@name OpeningStockImportApproval

// ApprovalInput is the approval request body.
type ApprovalInput struct {
	PhysicalCountReference string  `json:"physicalCountReference" example:"PC-2026-11-15"`
	PhysicalCountDate      string  `json:"physicalCountDate" example:"2026-11-15"`
	MappingsReviewed       *bool   `json:"mappingsReviewed"`
	DuplicatesReviewed     *bool   `json:"duplicatesReviewed"`
	ReceivedDatesApproved  *bool   `json:"receivedDatesApproved"`
	Note                   *string `json:"note" extensions:"x-nullable"`
} //@name OpeningStockImportApprovalRequest

// CommitSummary is a batch's commit as shown on the batch.
type CommitSummary struct {
	ID              string    `json:"id"`
	Reference       string    `json:"reference" example:"OSC-000001"`
	CommittedByID   string    `json:"committedById"`
	CommittedByName *string   `json:"committedByName" extensions:"x-nullable"`
	CommittedAt     time.Time `json:"committedAt"`
	UnitsCreated    int       `json:"unitsCreated" example:"120"`
} //@name OpeningStockImportCommitSummary

// CommittedUnit is one vehicle unit a commit created.
type CommittedUnit struct {
	RowNumber     int    `json:"rowNumber" example:"3"`
	VehicleUnitID string `json:"vehicleUnitId"`
	UnitNumber    string `json:"unitNumber" example:"MAE-VEH-000123"`
	Status        string `json:"status" example:"RECEIVED_PENDING_PDI"`
	MovementID    string `json:"movementId"`
} //@name OpeningStockCommittedUnit

// CommitResult is the commit response.
type CommitResult struct {
	CommitID       string `json:"commitId"`
	Reference      string `json:"reference" example:"OSC-000001"`
	BatchID        string `json:"batchId"`
	BatchReference string `json:"batchReference" example:"OSI-000001"`
	// BatchStatus is COMMITTED: the batch can never be committed again.
	BatchStatus     string    `json:"batchStatus" enums:"COMMITTED" example:"COMMITTED"`
	UnitsCreated    int       `json:"unitsCreated" example:"120"`
	CommittedByID   string    `json:"committedById"`
	CommittedByName *string   `json:"committedByName" extensions:"x-nullable"`
	CommittedAt     time.Time `json:"committedAt"`
	ApprovalID      string    `json:"approvalId"`
	FileSHA256      string    `json:"fileSha256"`
	TemplateVersion string    `json:"templateVersion" example:"OS-IMPORT-1"`
	// Provenance: units have source OPENING_STOCK and an OPENING_BALANCE movement
	// sourced from the batch; each points at its file line.
	SourceType   string          `json:"sourceType" enums:"OPENING_STOCK" example:"OPENING_STOCK"`
	MovementType string          `json:"movementType" enums:"OPENING_BALANCE" example:"OPENING_BALANCE"`
	Units        []CommittedUnit `json:"units"`
	// Replayed: this request repeated an earlier successful commit (same key); nothing new was created.
	Replayed bool `json:"replayed"`
} //@name OpeningStockImportCommit

// StoredRow is what the commit compares with the re-validated file.
type StoredRow struct {
	Line                                    int
	CanProceed                              bool
	Source                                  map[string]string
	BrandID, ModelID, VariantID, LocationID *string
	ChassisNormalised, StockStatus          *string
}

// SourceFile is the stored file of a batch.
type SourceFile struct {
	ObjectKey string
	SHA256    string
	SizeBytes int64
}

// CommitInput is everything the transaction needs.
type CommitInput struct {
	CommitID, BatchID, BatchReference, FileSHA256 string
	Actor                                         audit.Actor
	IdempotencyKey, RequestID                     string
	Rows                                          []SourceRow
	Scope                                         Scope
	BusinessDate, Now                             time.Time
	ApprovedStatuses                              []string
}

// NewApproval is an approval to store.
type NewApproval struct {
	BatchID, FileSHA256 string
	Actor               audit.Actor
	RequestID           string
	Input               ApprovalInput
}

// ── Gates (pure; unit tested) ───────────────────────────────────────────────

// batchGate checks what does not depend on the rows (a friendly early answer;
// the transaction checks again under the batch lock).
func batchGate(b Batch, approval *Approval, statuses []string) error {
	switch {
	case b.Commit != nil:
		return &AlreadyCommittedError{}
	case b.Status != BatchPreviewed:
		return ErrNotCommittable
	case b.TemplateVersion == nil || *b.TemplateVersion != TemplateVersion:
		return ErrNotCommittable
	case b.Expired:
		return ErrPreviewExpired
	case approval == nil:
		return ErrApprovalRequired
	case len(statuses) == 0:
		return ErrStatusesNotApproved
	}
	return nil
}

// sameSource: the parsed file has exactly the staged lines with the same values.
func sameSource(parsed []SourceRow, stored []StoredRow) bool {
	if len(parsed) != len(stored) {
		return false
	}
	for i, p := range parsed {
		s := stored[i]
		if p.Line != s.Line || p.FieldCountMismatch {
			return false
		}
		for _, c := range Columns {
			if p.Values[c.Name] != s.Source[c.Name] {
				return false
			}
		}
	}
	return true
}

func eqp(a, b *string) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }

// rowGate turns the commit-time validation into refusals: only rows that are
// VALID now, could proceed at preview, still resolve to the same records, have
// an approved opening status and a positive cost may be committed.
func rowGate(results []RowResult, stored []StoredRow, approved []string) []RowRefusal {
	byLine := make(map[int]StoredRow, len(stored))
	for _, s := range stored {
		byLine[s.Line] = s
	}
	var out []RowRefusal
	for _, r := range results {
		issues := slices.Clone(r.Issues)
		add := func(field, code, msg string) {
			issues = append(issues, Issue{Field: field, Code: code, Severity: SevError, Message: msg})
		}
		if r.Status == RowWarning {
			add("", CodeWarningNotAccepted, "This line has warnings. No approved rule accepts warnings at commit (OQ-49): "+
				"correct the file and upload it again.")
		}
		if r.StockStatus != nil && !slices.Contains(approved, *r.StockStatus) {
			add(ColStatus, CodeStatusNotApproved, "Stock status "+*r.StockStatus+" is not approved for opening stock (OQ-47).")
		}
		if r.AcquisitionCost != nil {
			if v, err := strconv.ParseFloat(*r.AcquisitionCost, 64); err == nil && v <= 0 {
				add(ColCost, CodeCostNotPositive, "A vehicle unit needs an acquisition cost above zero.")
			}
		}
		s, ok := byLine[r.Line]
		switch {
		case !ok || !s.CanProceed:
			add("", CodePreviewRowNotEligible, "This line could not proceed in the approved preview. Upload a corrected file.")
		case r.Status == RowValid && !(eqp(s.BrandID, r.BrandID) && eqp(s.ModelID, r.ModelID) && eqp(s.VariantID, r.VariantID) &&
			eqp(s.LocationID, r.LocationID) && eqp(s.ChassisNormalised, r.ChassisNormalised) && eqp(s.StockStatus, r.StockStatus)):
			add("", CodeChangedSincePreview, "The catalogue, location or identifiers changed since the preview. Upload the file again.")
		}
		if r.Status != RowValid || len(issues) > len(r.Issues) {
			out = append(out, RowRefusal{RowNumber: r.Line, Issues: issues})
		}
	}
	return out
}

// validateApproval checks the request body.
func validateApproval(in ApprovalInput, businessDate time.Time) []FieldProblem {
	var probs []FieldProblem
	ref := strings.TrimSpace(in.PhysicalCountReference)
	switch {
	case ref == "":
		probs = append(probs, FieldProblem{"physicalCountReference", "is required: the signed physical count this file was reconciled to"})
	case utf8.RuneCountInString(in.PhysicalCountReference) > 100 || hasControl(in.PhysicalCountReference):
		probs = append(probs, FieldProblem{"physicalCountReference", "must be at most 100 characters without control characters"})
	}
	if d, err := ageing.ParseDate(in.PhysicalCountDate); err != nil {
		probs = append(probs, FieldProblem{"physicalCountDate", "must be a date (YYYY-MM-DD)"})
	} else if d.After(businessDate) {
		probs = append(probs, FieldProblem{"physicalCountDate", "cannot be after today"})
	}
	if in.MappingsReviewed == nil || !*in.MappingsReviewed {
		probs = append(probs, FieldProblem{"mappingsReviewed", "must be true: confirm the brand/model/variant and location mappings were reviewed"})
	}
	if in.DuplicatesReviewed == nil || !*in.DuplicatesReviewed {
		probs = append(probs, FieldProblem{"duplicatesReviewed", "must be true: confirm the duplicate checks were reviewed"})
	}
	if in.ReceivedDatesApproved == nil {
		probs = append(probs, FieldProblem{"receivedDatesApproved", "is required: whether the received dates are the agreed opening-stock dates"})
	}
	if in.Note != nil {
		n := strings.TrimSpace(*in.Note)
		if n == "" || utf8.RuneCountInString(*in.Note) > 500 {
			probs = append(probs, FieldProblem{"note", "must be 1-500 characters when given"})
		}
	}
	return probs
}

func hasControl(s string) bool {
	return strings.ContainsFunc(s, func(r rune) bool { return r < 0x20 || r == 0x7f })
}

// sameApproval: a repeated request recording the same approval.
func sameApproval(a Approval, actor string, in ApprovalInput) bool {
	note := ""
	if in.Note != nil {
		note = strings.TrimSpace(*in.Note)
	}
	have := ""
	if a.Note != nil {
		have = *a.Note
	}
	return a.ApprovedByID == actor && a.PhysicalCountReference == strings.TrimSpace(in.PhysicalCountReference) &&
		a.PhysicalCountDate == in.PhysicalCountDate && a.ReceivedDatesApproved == *in.ReceivedDatesApproved && have == note
}

// ── Service ────────────────────────────────────────────────────────────────

// Approve records a batch's approval (stock.import.approve). The same approval
// repeated by its approver is a replay (true); any other is ErrAlreadyApproved.
func (s *Service) Approve(ctx context.Context, a access.Access, actor audit.Actor, id string, in ApprovalInput,
	requestID string) (Approval, bool, error) {
	if !a.Has(access.PermImportApprove) {
		return Approval{}, false, ErrForbidden
	}
	b, err := s.batchFor(ctx, a, id)
	if err != nil {
		return Approval{}, false, err
	}
	if probs := validateApproval(in, s.cal.Today()); len(probs) > 0 {
		return Approval{}, false, &ValidationError{Fields: probs}
	}
	if b.Approval != nil {
		if sameApproval(*b.Approval, actor.UserID, in) {
			return *b.Approval, true, nil
		}
		return Approval{}, false, ErrAlreadyApproved
	}
	switch {
	case b.Commit != nil:
		return Approval{}, false, &AlreadyCommittedError{}
	case b.Status != BatchPreviewed || b.TemplateVersion == nil || *b.TemplateVersion != TemplateVersion:
		return Approval{}, false, ErrNotCommittable
	case b.Expired:
		return Approval{}, false, ErrPreviewExpired
	case b.Summary.TotalRows == 0 || b.Summary.ValidRows != b.Summary.TotalRows:
		return Approval{}, false, ErrApprovalBlocked
	}
	stored, err := s.repo.StoredRows(ctx, b.ID)
	if err != nil {
		return Approval{}, false, err
	}
	if !inScope(a, stored) {
		return Approval{}, false, ErrForbidden
	}
	if err := s.repo.SaveApproval(ctx, NewApproval{BatchID: b.ID, FileSHA256: b.FileSHA256, Actor: actor, RequestID: requestID,
		Input: in}); err != nil && !errors.Is(err, ErrAlreadyApproved) {
		return Approval{}, false, err
	}
	b, err = s.repo.Get(ctx, b.ID)
	if err != nil {
		return Approval{}, false, err
	}
	if b.Approval == nil {
		return Approval{}, false, errors.New("approval not stored")
	}
	if b.Approval.ApprovedByID != actor.UserID || !sameApproval(*b.Approval, actor.UserID, in) {
		return Approval{}, false, ErrAlreadyApproved // a concurrent, different approval won
	}
	return *b.Approval, false, nil
}

// GetCommit returns a batch's commit (ErrNotFound until committed).
func (s *Service) GetCommit(ctx context.Context, a access.Access, id string) (CommitResult, error) {
	if err := s.allowedView(a); err != nil {
		return CommitResult{}, err
	}
	b, err := s.batchFor(ctx, a, id)
	if err != nil {
		return CommitResult{}, err
	}
	c, ok, err := s.repo.CommitOf(ctx, b.ID)
	if err == nil && !ok {
		err = ErrNotFound
	}
	return c, err
}

// inScope: every staged row's location is in the caller's location scope.
func inScope(a access.Access, rows []StoredRow) bool {
	for _, r := range rows {
		if r.LocationID != nil && !a.Scope.Allows(*r.LocationID) {
			return false
		}
	}
	return true
}

// batchFor reads a batch the caller may see.
func (s *Service) batchFor(ctx context.Context, a access.Access, id string) (Batch, error) {
	if _, err := uuid.Parse(id); err != nil {
		return Batch{}, ErrNotFound
	}
	b, err := s.repo.Get(ctx, id)
	if err != nil {
		return Batch{}, err
	}
	if !visible(a, b) {
		return Batch{}, ErrNotFound
	}
	return b, nil
}

// Commit commits a batch (stock.import.commit): exactly once, all rows or none.
// The same Idempotency-Key from the same user after a success returns that
// commit (Replayed); a commit by anyone else is AlreadyCommittedError.
func (s *Service) Commit(ctx context.Context, a access.Access, actor audit.Actor, id, key, requestID string) (CommitResult, error) {
	if !a.Has(access.PermImportCommit) {
		return CommitResult{}, ErrForbidden
	}
	if !keyPattern.MatchString(key) {
		return CommitResult{}, ErrIdempotencyKey
	}
	if c, ok, err := s.repo.CommitByKey(ctx, actor.UserID, key); err != nil {
		return CommitResult{}, err
	} else if ok {
		if c.BatchID != id {
			return CommitResult{}, ErrKeyReused
		}
		c.Replayed = true
		return c, nil
	}
	b, err := s.batchFor(ctx, a, id)
	if err != nil {
		return CommitResult{}, err
	}
	if err := batchGate(b, b.Approval, s.limits.OpeningStatuses); err != nil {
		return s.committed(ctx, err, b.ID, actor.UserID, key)
	}

	// The client-provided file, exactly as approved.
	src, err := s.repo.SourceFile(ctx, b.ID)
	if err != nil {
		return CommitResult{}, err
	}
	data, err := s.readObject(ctx, src)
	if err != nil {
		if s.log != nil {
			s.log.ErrorContext(ctx, "opening stock import file unreadable", slog.String("batch_id", b.ID), slog.Any("error", err))
		}
		return CommitResult{}, ErrSourceInvalid
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != b.FileSHA256 || src.SHA256 != b.FileSHA256 || int64(len(data)) != b.FileSizeBytes {
		return CommitResult{}, ErrSourceInvalid
	}
	parsed := Parse(data, s.limits.MaxRows)
	if len(parsed.Issues) > 0 || parsed.Version != TemplateVersion {
		return CommitResult{}, ErrSourceInvalid
	}
	stored, err := s.repo.StoredRows(ctx, b.ID)
	if err != nil {
		return CommitResult{}, err
	}
	if !inScope(a, stored) {
		return CommitResult{}, ErrForbidden
	}
	if !sameSource(parsed.Rows, stored) {
		return CommitResult{}, ErrSourceInvalid
	}

	in := CommitInput{CommitID: uuid.NewString(), BatchID: b.ID, BatchReference: b.Reference, FileSHA256: b.FileSHA256,
		Actor: actor, IdempotencyKey: key, RequestID: requestID, Rows: parsed.Rows, Scope: a.Scope,
		BusinessDate: s.cal.Today(), Now: s.now().UTC(), ApprovedStatuses: s.limits.OpeningStatuses}
	c, err := s.repo.Commit(ctx, in)
	switch {
	case errors.Is(err, ErrKeyTaken):
		// A concurrent request with the same key won.
		c, ok, ferr := s.repo.CommitByKey(ctx, actor.UserID, key)
		if ferr != nil || !ok {
			return CommitResult{}, errors.Join(err, ferr)
		}
		if c.BatchID != id {
			return CommitResult{}, ErrKeyReused
		}
		c.Replayed = true
		return c, nil
	case err != nil:
		var blocked *CommitBlockedError
		if errors.As(err, &blocked) && s.log != nil {
			s.log.InfoContext(ctx, "opening stock import commit blocked", slog.String("batch_id", b.ID),
				slog.String("user_id", actor.UserID), slog.Int("rows_refused", len(blocked.Rows)))
		}
		return s.committed(ctx, err, b.ID, actor.UserID, key)
	}
	if s.log != nil {
		s.log.InfoContext(ctx, "opening stock import committed", slog.String("batch_id", b.ID), slog.String("commit_id", c.CommitID),
			slog.String("user_id", actor.UserID), slog.Int("units", c.UnitsCreated))
	}
	return c, nil
}

// committed answers an AlreadyCommittedError: a replay when the commit is the
// caller's own with this key (a concurrent duplicate request), otherwise the
// error carrying the existing commit.
func (s *Service) committed(ctx context.Context, err error, batchID, committer, key string) (CommitResult, error) {
	var ac *AlreadyCommittedError
	if !errors.As(err, &ac) {
		return CommitResult{}, err
	}
	if c, ok, rerr := s.repo.CommitByKey(ctx, committer, key); rerr != nil {
		return CommitResult{}, rerr
	} else if ok && c.BatchID == batchID {
		c.Replayed = true
		return c, nil
	}
	c, ok, rerr := s.repo.CommitOf(ctx, batchID)
	if rerr != nil {
		return CommitResult{}, rerr
	}
	if ok {
		ac.Commit = c
	}
	return CommitResult{}, ac
}

func (s *Service) readObject(ctx context.Context, src SourceFile) ([]byte, error) {
	f, err := s.store.Open(ctx, src.ObjectKey)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, io.LimitReader(f, s.limits.MaxBytes+1)); err != nil {
		return nil, fmt.Errorf("read import file: %w", err)
	}
	return buf.Bytes(), nil
}

// statusSet is OpeningStatuses checked against the statuses the template allows.
func statusSet(list []string) ([]string, error) {
	seen := map[string]bool{}
	for _, v := range list {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if !slices.Contains(StockStatuses, v) {
			return nil, fmt.Errorf("opening stock status %q is not one of %s", v, strings.Join(StockStatuses, ", "))
		}
		seen[v] = true
	}
	return slices.Sorted(maps.Keys(seen)), nil
}

// ParseOpeningStatuses reads OPENING_STOCK_APPROVED_STATUSES (comma-separated).
func ParseOpeningStatuses(v string) ([]string, error) { return statusSet(strings.Split(v, ",")) }
