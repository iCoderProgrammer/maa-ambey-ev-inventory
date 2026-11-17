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
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/ageing"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/attachments"
)

// Limits are the technical bounds of an upload (configurable; proposals, not
// business values).
type Limits struct {
	MaxBytes   int64
	MaxRows    int
	PreviewTTL time.Duration
	// OpeningStatuses are the stock statuses the owner approved for opening stock
	// (OQ-47; OPENING_STOCK_APPROVED_STATUSES). Empty: every commit is blocked.
	OpeningStatuses []string
}

// DefaultLimits: 5 MiB, 5000 rows, previews usable for 7 days.
var DefaultLimits = Limits{MaxBytes: 5 << 20, MaxRows: 5000, PreviewTTL: 7 * 24 * time.Hour}

// Service runs uploads and previews.
type Service struct {
	repo   Repository
	store  attachments.ObjectStore
	cal    ageing.Calendar
	limits Limits
	log    *slog.Logger
	now    func() time.Time
}

// NewService builds the service.
func NewService(repo Repository, store attachments.ObjectStore, cal ageing.Calendar, limits Limits, log *slog.Logger) *Service {
	return &Service{repo: repo, store: store, cal: cal, limits: limits, log: log, now: time.Now}
}

// Specification describes the template and the upload rules.
type Specification struct {
	TemplateVersion string   `json:"templateVersion" example:"OS-IMPORT-1"`
	Format          string   `json:"format"`
	FirstLine       string   `json:"firstLine" example:"#template:OS-IMPORT-1"`
	MaxBytes        int64    `json:"maxBytes" example:"5242880"`
	MaxRows         int      `json:"maxRows" example:"5000"`
	PreviewTTLHours int      `json:"previewTtlHours" example:"168"`
	Columns         []Column `json:"columns"`
	StockStatuses   []string `json:"stockStatuses"`
	// ApprovedOpeningStatuses: the statuses a commit accepts (OQ-47); empty blocks every commit (G-07).
	ApprovedOpeningStatuses []string `json:"approvedOpeningStatuses"`
	BusinessDate            string   `json:"businessDate" example:"2026-11-16"`
	TimeZone                string   `json:"timeZone" example:"Asia/Kolkata"`
	Notes                   []string `json:"notes"`
} //@name OpeningStockImportSpecification

func (s *Service) allowed(a access.Access) error {
	if !a.Has(access.PermImportPreview) {
		return ErrForbidden
	}
	return nil
}

// allowedView: reading batches and rows also serves approvers and committers (G-07).
func (s *Service) allowedView(a access.Access) error {
	if !a.Has(access.PermImportPreview) && !a.Has(access.PermImportApprove) && !a.Has(access.PermImportCommit) {
		return ErrForbidden
	}
	return nil
}

// Spec returns the template specification (stock.import.preview).
func (s *Service) Spec(a access.Access) (Specification, error) {
	if err := s.allowed(a); err != nil {
		return Specification{}, err
	}
	return Specification{TemplateVersion: TemplateVersion,
		Format:    "CSV, UTF-8, comma-separated (Excel: \"CSV UTF-8 (comma delimited)\"). XLSX is not accepted.",
		FirstLine: TemplateMarker + TemplateVersion, MaxBytes: s.limits.MaxBytes, MaxRows: s.limits.MaxRows,
		PreviewTTLHours: int(s.limits.PreviewTTL.Hours()), Columns: Columns, StockStatuses: StockStatuses,
		ApprovedOpeningStatuses: append([]string{}, s.limits.OpeningStatuses...),
		BusinessDate:            s.cal.Today().Format(time.DateOnly), TimeZone: s.cal.Zone(),
		Notes: []string{
			"Keep the first line and the header line exactly as in the template; one vehicle per line from line 3.",
			"Columns may be in any order; optional columns may be left out; no other columns are allowed.",
			"Values are never corrected or guessed: brands, models, variants, colours and locations must match the catalogue.",
			"A preview creates no stock. Committing an approved preview is a separate, authorised step (G-07).",
		}}, nil
}

// Template returns the template file (stock.import.preview).
func (s *Service) Template(a access.Access) ([]byte, string, error) {
	if err := s.allowed(a); err != nil {
		return nil, "", err
	}
	return TemplateCSV(), "opening-stock-template-" + TemplateVersion + ".csv", nil
}

var keyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,100}$`)

// UploadInput is one uploaded file.
type UploadInput struct {
	FileName, DeclaredType, IdempotencyKey, RequestID string
	Body                                              io.Reader
}

func sanitizeName(name string) string {
	name = path.Base(strings.ReplaceAll(name, `\`, "/"))
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, strings.TrimSpace(name))
	if name == "" || name == "." {
		name = "upload.csv"
	}
	if len([]rune(name)) > 200 {
		name = string([]rune(name)[:200])
	}
	return name
}

// Upload stores the file privately, validates it and stages the preview. It
// returns the batch and whether it is a replay of an earlier request with the
// same Idempotency-Key (same file). Nothing outside the import tables, the
// file's attachment row and the audit log is written.
func (s *Service) Upload(ctx context.Context, a access.Access, actor audit.Actor, in UploadInput) (Batch, bool, error) {
	if err := s.allowed(a); err != nil {
		return Batch{}, false, err
	}
	if !keyPattern.MatchString(in.IdempotencyKey) {
		return Batch{}, false, ErrIdempotencyKey
	}
	name := sanitizeName(in.FileName)
	data, err := io.ReadAll(io.LimitReader(in.Body, s.limits.MaxBytes+1))
	if err != nil {
		return Batch{}, false, err
	}
	switch {
	case len(data) == 0:
		return Batch{}, false, ErrEmptyFile
	case int64(len(data)) > s.limits.MaxBytes:
		return Batch{}, false, ErrTooLarge
	}
	// Content first, then the name: a CSV is plain text; spreadsheets (zip), PDFs,
	// images and binaries are refused whatever they are called.
	detected, _, _ := strings.Cut(http.DetectContentType(data), ";")
	ext := strings.ToLower(path.Ext(name))
	declared := strings.ToLower(strings.TrimSpace(strings.SplitN(in.DeclaredType, ";", 2)[0]))
	if detected != "text/plain" || ext != ".csv" || (declared != "" && declared != "text/csv" && declared != "application/octet-stream" &&
		declared != "application/vnd.ms-excel" && declared != "text/plain") {
		return Batch{}, false, ErrUnsupportedType
	}
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])

	if b, ok, err := s.repo.FindByKey(ctx, actor.UserID, in.IdempotencyKey); err != nil {
		return Batch{}, false, err
	} else if ok {
		if b.FileSHA256 != hash {
			return Batch{}, false, ErrKeyReused
		}
		return withFlags(b), true, nil
	}

	businessDate := s.cal.Today()
	parsed := Parse(data, s.limits.MaxRows)
	nb := NewBatch{ID: uuid.NewString(), AttachmentID: uuid.NewString(), StorageDriver: s.store.Driver(), FileName: name,
		SHA256Hex: hash, SHA256: sum[:], SizeBytes: int64(len(data)), UploadedBy: actor.UserID, IdempotencyKey: in.IdempotencyKey,
		RequestID: in.RequestID, Actor: actor, BusinessDate: businessDate, FileIssues: parsed.Issues}
	if parsed.Version != "" {
		v := parsed.Version
		nb.TemplateVersion = &v
	}
	nb.Summary.SkippedBlankRows = parsed.SkippedBlank
	if len(parsed.Issues) > 0 {
		nb.Status, nb.Summary.SkippedBlankRows = BatchRejected, 0
	} else {
		nb.Status = BatchPreviewed
		lookups, err := s.repo.Lookups(ctx, CollectRefs(parsed.Rows))
		if err != nil {
			return Batch{}, false, err
		}
		nb.Rows = Validate(parsed.Rows, lookups, a.Scope, businessDate)
		for _, r := range nb.Rows {
			sm := &nb.Summary
			sm.TotalRows++
			switch r.Status {
			case RowValid:
				sm.ValidRows++
			case RowWarning:
				sm.WarningRows++
			case RowError:
				sm.ErrorRows++
			case RowBlocked:
				sm.BlockedRows++
			}
			if r.Duplicate {
				sm.DuplicateRows++
			}
			if r.Unresolved {
				sm.UnresolvedRows++
			}
		}
	}
	now := s.now().UTC()
	nb.UploadedAt, nb.ExpiresAt = now, now.Add(s.limits.PreviewTTL)
	nb.ObjectKey = fmt.Sprintf("%04d/%02d/%s", now.Year(), int(now.Month()), uuid.NewString())
	if _, err := s.store.Put(ctx, nb.ObjectKey, bytes.NewReader(data)); err != nil {
		return Batch{}, false, fmt.Errorf("store import file: %w", err)
	}
	if err := s.repo.Save(ctx, nb); err != nil {
		if derr := s.store.Delete(context.WithoutCancel(ctx), nb.ObjectKey); derr != nil && s.log != nil {
			s.log.ErrorContext(ctx, "orphaned import file object", slog.String("object_key", nb.ObjectKey), slog.Any("error", derr))
		}
		if errors.Is(err, ErrKeyTaken) {
			// A concurrent request with the same key won: answer like a replay.
			b, ok, ferr := s.repo.FindByKey(ctx, actor.UserID, in.IdempotencyKey)
			switch {
			case ferr != nil:
				return Batch{}, false, ferr
			case !ok:
				return Batch{}, false, err
			case b.FileSHA256 != hash:
				return Batch{}, false, ErrKeyReused
			}
			return withFlags(b), true, nil
		}
		return Batch{}, false, err
	}
	if s.log != nil {
		s.log.InfoContext(ctx, "opening stock import previewed", slog.String("batch_id", nb.ID), slog.String("user_id", actor.UserID),
			slog.String("status", nb.Status), slog.Int("rows", nb.Summary.TotalRows), slog.Int("errors", nb.Summary.ErrorRows),
			slog.Int("blocked", nb.Summary.BlockedRows))
	}
	b, err := s.repo.Get(ctx, nb.ID)
	return withFlags(b), false, err
}

func withFlags(b Batch) Batch {
	b.CommitAvailable = false
	return b
}

// visible: the uploader, or a holder of the permission with every location
// (rows may name any location the uploader works in).
func visible(a access.Access, b Batch) bool {
	return b.UploadedByID == a.UserID || a.Scope.All
}

// Get returns a batch the caller may see (others: ErrNotFound).
func (s *Service) Get(ctx context.Context, a access.Access, id string) (Batch, error) {
	if err := s.allowedView(a); err != nil {
		return Batch{}, err
	}
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
	return withFlags(b), nil
}

// BatchPage is a page of batches.
type BatchPage struct {
	Items []Batch `json:"items"`
	Page  int     `json:"page" example:"1"`
	Limit int     `json:"limit" example:"25"`
	Total int     `json:"total" example:"0"`
} //@name OpeningStockImportBatchPage

// RowPage is a page of rows.
type RowPage struct {
	Items []Row `json:"items"`
	Page  int   `json:"page" example:"1"`
	Limit int   `json:"limit" example:"50"`
	Total int   `json:"total" example:"0"`
} //@name OpeningStockImportRowPage

// ValidationError lists invalid query fields.
type ValidationError struct{ Fields []FieldProblem }

// FieldProblem is one invalid field.
type FieldProblem struct{ Field, Message string }

func (e *ValidationError) Error() string { return "invalid request" }

func paging(page, limit string, def, max int) (int, int, []FieldProblem) {
	p, l := 1, def
	var probs []FieldProblem
	if page != "" {
		n, err := strconv.Atoi(page)
		if err != nil || n < 1 || n > 100000 {
			probs = append(probs, FieldProblem{"page", "must be a whole number from 1"})
		} else {
			p = n
		}
	}
	if limit != "" {
		n, err := strconv.Atoi(limit)
		if err != nil || n < 1 || n > max {
			probs = append(probs, FieldProblem{"limit", "must be a whole number from 1 to " + strconv.Itoa(max)})
		} else {
			l = n
		}
	}
	return p, l, probs
}

// List lists the caller's batches (all batches with every location).
func (s *Service) List(ctx context.Context, a access.Access, page, limit string) (BatchPage, error) {
	if err := s.allowedView(a); err != nil {
		return BatchPage{}, err
	}
	p, l, probs := paging(page, limit, 25, 100)
	if len(probs) > 0 {
		return BatchPage{}, &ValidationError{Fields: probs}
	}
	items, total, err := s.repo.List(ctx, a.UserID, a.Scope.All, p, l)
	if err != nil {
		return BatchPage{}, err
	}
	for i := range items {
		items[i] = withFlags(items[i])
	}
	if items == nil {
		items = []Batch{}
	}
	return BatchPage{Items: items, Page: p, Limit: l, Total: total}, nil
}

// RowQuery is the raw rows request.
type RowQuery struct{ Status, Duplicate, Unresolved, Page, Limit string }

// Rows lists a visible batch's rows in file order, with filters.
func (s *Service) Rows(ctx context.Context, a access.Access, id string, q RowQuery) (RowPage, error) {
	if _, err := s.Get(ctx, a, id); err != nil {
		return RowPage{}, err
	}
	f := RowFilter{BatchID: id}
	var probs []FieldProblem
	switch q.Status {
	case "", RowValid, RowWarning, RowError, RowBlocked:
		f.Status = q.Status
	default:
		probs = append(probs, FieldProblem{"status", "must be one of VALID, WARNING, ERROR, BLOCKED_UNVERIFIED_MAPPING"})
	}
	boolean := func(field, v string) bool {
		switch v {
		case "", "false":
			return false
		case "true":
			return true
		}
		probs = append(probs, FieldProblem{field, "must be true or false"})
		return false
	}
	f.DuplicateOnly, f.Unresolved = boolean("duplicate", q.Duplicate), boolean("unresolved", q.Unresolved)
	var pp []FieldProblem
	f.Page, f.Limit, pp = paging(q.Page, q.Limit, 50, 200)
	probs = append(probs, pp...)
	if len(probs) > 0 {
		return RowPage{}, &ValidationError{Fields: probs}
	}
	items, total, err := s.repo.Rows(ctx, f)
	if err != nil {
		return RowPage{}, err
	}
	if items == nil {
		items = []Row{}
	}
	return RowPage{Items: items, Page: f.Page, Limit: f.Limit, Total: total}, nil
}
