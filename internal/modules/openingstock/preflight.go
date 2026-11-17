package openingstock

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
)

// H-08: opening stock reconciliation preflight (ADR-032).
//
// GET /opening-stock-imports/{id}/preflight answers "what would stop this batch
// from becoming opening stock right now, and how does it compare with the
// physical count?" without changing anything. It runs exactly the checks the
// G-07 commit runs (stored file + SHA-256, template, identical staged lines,
// Validate against the current database, rowGate) but reads through one READ
// ONLY transaction, takes no locks, writes no audit event and never approves or
// commits. Warnings are reported, never accepted (OQ-49). Physical-count
// quantities are not held by the system (H-08 decision pending), so the
// comparison reports them as not recorded instead of inventing numbers.

// Preflight row categories.
const (
	PreflightValid      = "VALID"
	PreflightWarning    = "WARNING"
	PreflightError      = "BLOCKING_ERROR"
	PreflightUnverified = "UNVERIFIED"
	PreflightReconcile  = "MANUAL_RECONCILIATION"
)

// Source check states.
const (
	SourceVerified   = "VERIFIED"
	SourceInvalid    = "INVALID"
	SourceNotChecked = "NOT_CHECKED"
)

// Physical count states.
const (
	// PhysicalNotRecorded: no signed physical count is linked to the batch.
	PhysicalNotRecorded = "NOT_RECORDED"
	// PhysicalReferenceOnly: the G-07 approval names a signed count (reference and
	// date) but the system holds no counted quantities to compare.
	PhysicalReferenceOnly = "REFERENCE_ONLY"
)

// Preflight blocker codes.
const (
	BlockerAlreadyCommitted    = "ALREADY_COMMITTED"
	BlockerBatchRejected       = "BATCH_REJECTED"
	BlockerTemplate            = "TEMPLATE_NOT_SUPPORTED"
	BlockerExpired             = "PREVIEW_EXPIRED"
	BlockerSourceInvalid       = "SOURCE_INVALID"
	BlockerRowsNotValid        = "ROWS_NOT_VALID"
	BlockerApprovalMissing     = "APPROVAL_NOT_RECORDED"
	BlockerStatusesNotApproved = "OPENING_STATUSES_NOT_APPROVED"
	BlockerPhysicalCount       = "PHYSICAL_COUNT_QUANTITIES_NOT_RECORDED"
)

// PreflightSource is the integrity check of the stored file.
type PreflightSource struct {
	Status             string `json:"status" enums:"VERIFIED,INVALID,NOT_CHECKED" example:"VERIFIED"`
	FileAvailable      bool   `json:"fileAvailable"`
	ChecksumMatches    bool   `json:"checksumMatches"`
	SizeMatches        bool   `json:"sizeMatches"`
	TemplateCompatible bool   `json:"templateCompatible"`
	MatchesStagedRows  bool   `json:"matchesStagedRows"`
	Message            string `json:"message"`
} //@name OpeningStockPreflightSource

// PreflightSummary counts the rows by category.
type PreflightSummary struct {
	TotalRows                int `json:"totalRows" example:"120"`
	ValidRows                int `json:"validRows" example:"100"`
	WarningRows              int `json:"warningRows" example:"10"`
	BlockingErrorRows        int `json:"blockingErrorRows" example:"5"`
	UnverifiedRows           int `json:"unverifiedRows" example:"3"`
	ManualReconciliationRows int `json:"manualReconciliationRows" example:"2"`
	// DuplicateRows: rows with any duplicate identifier issue (file, inventory or open receipt).
	DuplicateRows int `json:"duplicateRows" example:"4"`
	// MissingChassisRows: rows without a chassis/frame number.
	MissingChassisRows int `json:"missingChassisRows" example:"1"`
	// ChangedSincePreviewRows: rows whose result today differs from the stored preview.
	ChangedSincePreviewRows int `json:"changedSincePreviewRows" example:"2"`
} //@name OpeningStockPreflightSummary

// PreflightIssueCount is how many rows carry an issue code.
type PreflightIssueCount struct {
	Code     string `json:"code" example:"DUPLICATE_CHASSIS_EXISTING"`
	Severity string `json:"severity" enums:"ERROR,BLOCKED,WARNING" example:"ERROR"`
	Rows     int    `json:"rows" example:"2"`
} //@name OpeningStockPreflightIssueCount

// PreflightRow is a row that is not VALID, with every reason.
type PreflightRow struct {
	RowNumber     int    `json:"rowNumber" example:"5"`
	Category      string `json:"category" enums:"WARNING,BLOCKING_ERROR,UNVERIFIED,MANUAL_RECONCILIATION" example:"BLOCKING_ERROR"`
	PreviewStatus string `json:"previewStatus" enums:"VALID,WARNING,ERROR,BLOCKED_UNVERIFIED_MAPPING" example:"VALID"`
	CurrentStatus string `json:"currentStatus" enums:"VALID,WARNING,ERROR,BLOCKED_UNVERIFIED_MAPPING" example:"ERROR"`
	// ChangedSincePreview: the row's result today differs from its stored preview.
	ChangedSincePreview bool              `json:"changedSincePreview"`
	Source              map[string]string `json:"source"`
	Issues              []Issue           `json:"issues"`
} //@name OpeningStockPreflightRow

// PreflightQuantity compares imported quantities per location, catalogue entry
// and status with the physical count (values as uploaded; nothing resolved or guessed).
type PreflightQuantity struct {
	LocationCode string `json:"locationCode" example:"L1"`
	Brand        string `json:"brand" example:"Brand A"`
	ModelCode    string `json:"modelCode" example:"MA"`
	VariantCode  string `json:"variantCode" example:"STD"`
	StockStatus  string `json:"stockStatus" example:"AVAILABLE"`
	// FileRows: vehicles listed in the file (the imported quantity).
	FileRows int `json:"fileRows" example:"12"`
	// ValidRows: of those, rows that pass every commit check today.
	ValidRows int `json:"validRows" example:"11"`
	// PhysicalCount and Difference stay null: the system holds no counted quantities (H-08).
	PhysicalCount *int `json:"physicalCount" extensions:"x-nullable"`
	Difference    *int `json:"difference" extensions:"x-nullable"`
} //@name OpeningStockPreflightQuantity

// PreflightPhysicalCount says what is known about the signed physical count.
type PreflightPhysicalCount struct {
	Status    string  `json:"status" enums:"NOT_RECORDED,REFERENCE_ONLY" example:"NOT_RECORDED"`
	Reference *string `json:"reference" extensions:"x-nullable" example:"PC-2026-11-15"`
	CountDate *string `json:"countDate" extensions:"x-nullable" example:"2026-11-15"`
	Message   string  `json:"message"`
} //@name OpeningStockPreflightPhysicalCount

// PreflightBlocker is one reason the batch is not ready, with the owner decision it waits for.
type PreflightBlocker struct {
	Code    string `json:"code" example:"OPENING_STATUSES_NOT_APPROVED"`
	Message string `json:"message"`
	// Decision: the open question or task that resolves it ("" when the data must be corrected).
	Decision string `json:"decision" example:"OQ-47"`
} //@name OpeningStockPreflightBlocker

// Preflight is the read-only reconciliation preflight of a batch.
type Preflight struct {
	BatchID         string    `json:"batchId"`
	BatchReference  string    `json:"batchReference" example:"OSI-000001"`
	BatchStatus     string    `json:"batchStatus" enums:"PREVIEWED,REJECTED" example:"PREVIEWED"`
	FileName        string    `json:"fileName" example:"opening-stock.csv"`
	FileSHA256      string    `json:"fileSha256"`
	FileSizeBytes   int64     `json:"fileSizeBytes" example:"2048"`
	TemplateVersion *string   `json:"templateVersion" extensions:"x-nullable" example:"OS-IMPORT-1"`
	UploadedByName  *string   `json:"uploadedByName" extensions:"x-nullable"`
	UploadedAt      time.Time `json:"uploadedAt"`
	ExpiresAt       time.Time `json:"expiresAt"`
	Expired         bool      `json:"expired"`
	CheckedAt       time.Time `json:"checkedAt"`
	BusinessDate    string    `json:"businessDate" example:"2026-11-17"`
	// NonMutating is always true: the preflight created, changed, approved and committed nothing.
	NonMutating bool `json:"nonMutating"`
	// RowsChecked: the rows were re-validated (false when the batch is committed,
	// rejected, in another template or its file failed the integrity check).
	RowsChecked             bool                   `json:"rowsChecked"`
	Source                  PreflightSource        `json:"source"`
	Summary                 PreflightSummary       `json:"summary"`
	PreviewSummary          Summary                `json:"previewSummary"`
	IssueCounts             []PreflightIssueCount  `json:"issueCounts"`
	Quantities              []PreflightQuantity    `json:"quantities"`
	PhysicalCount           PreflightPhysicalCount `json:"physicalCount"`
	ApprovedOpeningStatuses []string               `json:"approvedOpeningStatuses"`
	Approval                *Approval              `json:"approval" extensions:"x-nullable"`
	Commit                  *CommitSummary         `json:"commit" extensions:"x-nullable"`
	Blockers                []PreflightBlocker     `json:"blockers"`
	// Exceptions: every row that is not VALID, in file order.
	Exceptions []PreflightRow `json:"exceptions"`
} //@name OpeningStockPreflight

// PreflightSnapshot is what the preflight reads in its one read-only transaction.
type PreflightSnapshot struct {
	Stored       []StoredRow
	StagedStatus map[int]string
	Lookups      Lookups
}

// ── Pure parts (unit tested) ───────────────────────────────────────────────

// classify places a re-validated row (and its commit refusal, if any) in one
// category: BLOCKING_ERROR > UNVERIFIED > MANUAL_RECONCILIATION > WARNING > VALID.
func classify(r RowResult, refusal *RowRefusal, previewStatus string) string {
	added := map[string]bool{}
	if refusal != nil {
		for _, is := range refusal.Issues[min(len(r.Issues), len(refusal.Issues)):] {
			added[is.Code] = true
		}
	}
	switch {
	case r.Status == RowError || added[CodeCostNotPositive] || added[CodeStatusNotApproved]:
		return PreflightError
	case r.Status == RowBlocked:
		return PreflightUnverified
	case added[CodeChangedSincePreview] || added[CodePreviewRowNotEligible] || previewStatus != r.Status:
		return PreflightReconcile
	case r.Status == RowWarning:
		return PreflightWarning
	}
	return PreflightValid
}

func quantityKey(r RowResult) [5]string {
	c := func(col string) string { return strings.ToUpper(strings.TrimSpace(r.Source[col])) }
	return [5]string{c(ColLocation), c(ColBrand), c(ColModelCode), c(ColVariantCode), c(ColStatus)}
}

// evaluate builds the row-level part of the preflight. approved is the owner's
// opening-status list; when it is empty the status gate is reported once as a
// blocker instead of on every row.
func evaluate(results []RowResult, stored []StoredRow, staged map[int]string, approved []string) (PreflightSummary,
	[]PreflightIssueCount, []PreflightQuantity, []PreflightRow) {
	gate := approved
	if len(gate) == 0 {
		gate = StockStatuses
	}
	refusals := map[int]*RowRefusal{}
	for _, rf := range rowGate(results, stored, gate) {
		refusals[rf.RowNumber] = &rf
	}
	var sm PreflightSummary
	type countKey struct{ code, sev string }
	counts := map[countKey]int{}
	groups := map[[5]string]*PreflightQuantity{}
	exceptions := []PreflightRow{}
	for _, r := range results {
		refusal := refusals[r.Line]
		cat := classify(r, refusal, staged[r.Line])
		sm.TotalRows++
		switch cat {
		case PreflightValid:
			sm.ValidRows++
		case PreflightWarning:
			sm.WarningRows++
		case PreflightError:
			sm.BlockingErrorRows++
		case PreflightUnverified:
			sm.UnverifiedRows++
		case PreflightReconcile:
			sm.ManualReconciliationRows++
		}
		issues := r.Issues
		if refusal != nil {
			issues = slices.Clone(refusal.Issues)
			for i := range issues {
				// The commit's wording assumes an approval; the preflight may run before one.
				if issues[i].Code == CodePreviewRowNotEligible {
					issues[i].Message = "This line could not proceed in the stored preview. Upload a corrected file."
				}
			}
		}
		seen := map[countKey]bool{}
		for _, is := range issues {
			k := countKey{is.Code, is.Severity}
			if !seen[k] {
				seen[k] = true
				counts[k]++
			}
			if is.Field == ColChassis && is.Code == CodeRequired {
				sm.MissingChassisRows++
			}
		}
		if r.Duplicate {
			sm.DuplicateRows++
		}
		changed := staged[r.Line] != r.Status
		if refusal != nil && slices.ContainsFunc(refusal.Issues, func(is Issue) bool { return is.Code == CodeChangedSincePreview }) {
			changed = true
		}
		if changed {
			sm.ChangedSincePreviewRows++
		}
		k := quantityKey(r)
		g := groups[k]
		if g == nil {
			t := func(col string) string { return strings.TrimSpace(r.Source[col]) }
			g = &PreflightQuantity{LocationCode: t(ColLocation), Brand: t(ColBrand), ModelCode: t(ColModelCode),
				VariantCode: t(ColVariantCode), StockStatus: t(ColStatus)}
			groups[k] = g
		}
		g.FileRows++
		if cat == PreflightValid {
			g.ValidRows++
			continue
		}
		if issues == nil {
			issues = []Issue{}
		}
		exceptions = append(exceptions, PreflightRow{RowNumber: r.Line, Category: cat, PreviewStatus: staged[r.Line],
			CurrentStatus: r.Status, ChangedSincePreview: changed, Source: r.Source, Issues: issues})
	}
	ic := make([]PreflightIssueCount, 0, len(counts))
	for k, n := range counts {
		ic = append(ic, PreflightIssueCount{Code: k.code, Severity: k.sev, Rows: n})
	}
	slices.SortFunc(ic, func(a, b PreflightIssueCount) int {
		return cmp.Or(-cmp.Compare(a.Rows, b.Rows), cmp.Compare(a.Code, b.Code), cmp.Compare(a.Severity, b.Severity))
	})
	keys := make([][5]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b [5]string) int { return slices.Compare(a[:], b[:]) })
	qs := make([]PreflightQuantity, len(keys))
	for i, k := range keys {
		qs[i] = *groups[k]
	}
	return sm, ic, qs, exceptions
}

// blockers lists why the batch is not ready to become opening stock.
func blockers(p Preflight) []PreflightBlocker {
	out := []PreflightBlocker{}
	add := func(code, msg, decision string) {
		out = append(out, PreflightBlocker{Code: code, Message: msg, Decision: decision})
	}
	switch {
	case p.Commit != nil:
		add(BlockerAlreadyCommitted, "This batch was already committed ("+p.Commit.Reference+"). It can never be committed again.", "")
		return out
	case p.BatchStatus != BatchPreviewed:
		add(BlockerBatchRejected, "The file was rejected at upload (see its file issues). Upload a corrected file as a new batch.", "")
	case p.TemplateVersion == nil || *p.TemplateVersion != TemplateVersion:
		add(BlockerTemplate, "The file is not in template "+TemplateVersion+".", "A-04")
	}
	if p.Expired {
		add(BlockerExpired, "The preview has expired and is history only. Upload the file again for a new preview.", "OQ-48")
	}
	if p.Source.Status == SourceInvalid {
		add(BlockerSourceInvalid, p.Source.Message, "")
	}
	if p.RowsChecked && p.Summary.ValidRows != p.Summary.TotalRows {
		add(BlockerRowsNotValid, "Some rows are not valid today: correct the file and upload it again. Warnings are not accepted "+
			"(no approved rule, OQ-49).", "OQ-49")
	}
	if p.Approval == nil {
		add(BlockerApprovalMissing, "No physical-count approval is recorded for this batch (G-07). Who may approve is not decided.", "OQ-49")
	}
	if len(p.ApprovedOpeningStatuses) == 0 {
		add(BlockerStatusesNotApproved, "No opening-stock status is approved (OPENING_STOCK_APPROVED_STATUSES is empty), so every "+
			"commit is refused.", "OQ-47")
	}
	add(BlockerPhysicalCount, "The system holds no counted quantities from a signed physical count, so imported and counted "+
		"quantities cannot be compared here. The count format and sign-off are pending.", "H-08")
	return out
}

// ── Service ────────────────────────────────────────────────────────────────

// Preflight re-checks a visible batch exactly as a commit would, without
// writing anything (any of the three import permissions). Rows are validated
// with the caller's location scope.
func (s *Service) Preflight(ctx context.Context, a access.Access, id string) (Preflight, error) {
	if err := s.allowedView(a); err != nil {
		return Preflight{}, err
	}
	b, err := s.batchFor(ctx, a, id)
	if err != nil {
		return Preflight{}, err
	}
	p := Preflight{BatchID: b.ID, BatchReference: b.Reference, BatchStatus: b.Status, FileName: b.FileName, FileSHA256: b.FileSHA256,
		FileSizeBytes: b.FileSizeBytes, TemplateVersion: b.TemplateVersion, UploadedByName: b.UploadedByName, UploadedAt: b.UploadedAt,
		ExpiresAt: b.ExpiresAt, Expired: b.Expired, CheckedAt: s.now().UTC(), BusinessDate: s.cal.Today().Format(time.DateOnly),
		NonMutating: true, PreviewSummary: b.Summary, IssueCounts: []PreflightIssueCount{}, Quantities: []PreflightQuantity{},
		ApprovedOpeningStatuses: append([]string{}, s.limits.OpeningStatuses...), Approval: b.Approval, Commit: b.Commit,
		Exceptions: []PreflightRow{}, Source: PreflightSource{Status: SourceNotChecked}}
	p.PhysicalCount = PreflightPhysicalCount{Status: PhysicalNotRecorded,
		Message: "No signed physical count is linked to this batch. Physical-count quantities are not recorded in the system."}
	if b.Approval != nil {
		ref, date := b.Approval.PhysicalCountReference, b.Approval.PhysicalCountDate
		p.PhysicalCount = PreflightPhysicalCount{Status: PhysicalReferenceOnly, Reference: &ref, CountDate: &date,
			Message: "The approval names a signed physical count, but its quantities are not recorded in the system."}
	}
	defer func() {
		if s.log != nil {
			s.log.InfoContext(ctx, "opening stock preflight checked", slog.String("batch_id", b.ID), slog.String("user_id", a.UserID),
				slog.String("source", p.Source.Status), slog.Int("rows", p.Summary.TotalRows), slog.Int("valid", p.Summary.ValidRows))
		}
	}()

	switch {
	case b.Commit != nil:
		p.Source.Message = "Not checked: the batch is committed."
	case b.Status != BatchPreviewed || b.TemplateVersion == nil || *b.TemplateVersion != TemplateVersion:
		p.Source.Message = "Not checked: the file was rejected at upload."
	default:
		if err := s.preflightRows(ctx, a, b, &p); err != nil {
			return Preflight{}, err
		}
	}
	p.Blockers = blockers(p)
	return p, nil
}

// preflightRows checks the stored file and re-validates its rows.
func (s *Service) preflightRows(ctx context.Context, a access.Access, b Batch, p *Preflight) error {
	invalid := func(msg string) error {
		p.Source.Status, p.Source.Message = SourceInvalid, msg
		return nil
	}
	src, err := s.repo.SourceFile(ctx, b.ID)
	if err != nil {
		if errors.Is(err, ErrSourceInvalid) {
			return invalid("The stored file of this batch is missing.")
		}
		return err
	}
	data, err := s.readObject(ctx, src)
	if err != nil {
		if s.log != nil {
			s.log.ErrorContext(ctx, "opening stock import file unreadable", slog.String("batch_id", b.ID), slog.Any("error", err))
		}
		return invalid("The stored file of this batch cannot be read.")
	}
	p.Source.FileAvailable = true
	sum := sha256.Sum256(data)
	p.Source.ChecksumMatches = hex.EncodeToString(sum[:]) == b.FileSHA256 && src.SHA256 == b.FileSHA256
	p.Source.SizeMatches = int64(len(data)) == b.FileSizeBytes
	if !p.Source.ChecksumMatches || !p.Source.SizeMatches {
		return invalid("The stored file no longer matches the uploaded file (SHA-256 or size differs).")
	}
	parsed := Parse(data, s.limits.MaxRows)
	if len(parsed.Issues) > 0 || parsed.Version != TemplateVersion {
		return invalid("The stored file is no longer readable as template " + TemplateVersion + ".")
	}
	p.Source.TemplateCompatible = true
	snap, err := s.repo.PreflightSnapshot(ctx, b.ID, CollectRefs(parsed.Rows))
	if err != nil {
		return err
	}
	if !sameSource(parsed.Rows, snap.Stored) {
		return invalid("The stored file's lines differ from the staged preview rows.")
	}
	p.Source.MatchesStagedRows = true
	p.Source.Status, p.Source.Message = SourceVerified, "The stored file matches the upload (SHA-256, size, template and lines)."
	results := Validate(parsed.Rows, snap.Lookups, a.Scope, s.cal.Today())
	p.Summary, p.IssueCounts, p.Quantities, p.Exceptions = evaluate(results, snap.Stored, snap.StagedStatus, s.limits.OpeningStatuses)
	p.RowsChecked = true
	return nil
}
