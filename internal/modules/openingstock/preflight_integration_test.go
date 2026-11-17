package openingstock_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/openingstock"
)

// H-08 reconciliation preflight on a throwaway migrated database. Every brand,
// chassis and count reference is a placeholder fixture, not client stock.

// everythingFingerprint digests the business tables, the import tables, the
// attachments and the audit log: a preflight must change none of them.
func (f *fixture) everythingFingerprint(t *testing.T) string {
	t.Helper()
	parts := []string{f.businessFingerprint(t)}
	for _, table := range []string{"opening_stock_import_batches", "opening_stock_import_rows", "opening_stock_import_approvals",
		"opening_stock_import_commits", "attachments", "audit_events"} {
		parts = append(parts, table+"="+strconv.Itoa(f.count(t, `SELECT count(*) FROM `+table)))
	}
	parts = append(parts, "rows="+f.digest(t, "opening_stock_import_rows"), "batches="+f.digest(t, "opening_stock_import_batches"))
	return strings.Join(parts, " ")
}

func (f *fixture) digest(t *testing.T, table string) string {
	t.Helper()
	var s string
	if err := f.pool.QueryRow(context.Background(), `SELECT count(*)::text || ':' || coalesce(md5(string_agg(t::text, '|' ORDER BY t::text)), '')
		FROM `+table+` t`).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

func (f *fixture) preflight(t *testing.T, s session, id string) openingstock.Preflight {
	t.Helper()
	var p openingstock.Preflight
	expect(t, f.h.get(&s, "/opening-stock-imports/"+id+"/preflight"), http.StatusOK, &p)
	if !p.NonMutating {
		t.Fatal("preflight not marked non-mutating")
	}
	return p
}

func blockerCodes(p openingstock.Preflight) []string {
	out := []string{}
	for _, b := range p.Blockers {
		out = append(out, b.Code)
	}
	return out
}

func exceptionByLine(p openingstock.Preflight) map[int]openingstock.PreflightRow {
	m := map[int]openingstock.PreflightRow{}
	for _, r := range p.Exceptions {
		m[r.RowNumber] = r
	}
	return m
}

func hasCode(r openingstock.PreflightRow, code string) bool {
	return slices.ContainsFunc(r.Issues, func(i openingstock.Issue) bool { return i.Code == code })
}

func TestPreflightReportsEveryCategoryAndChangesNothing(t *testing.T) {
	f := setup(t)
	var b openingstock.Batch
	expect(t, f.h.upload(&f.importer, "opening.csv", "text/csv", csvFile(
		row("Brand A", "MA", "STD", "", "PF-1", "", "L1", "AVAILABLE", "2026-01-10", "Register", "", "90000"),     // 3 valid
		row("Brand A", "MA", "STD", "", "PF-2", "", "L1", "DISPLAY", "2026-01-11", "Register", "", "90000"),       // 4 valid
		row("Brand A", "MA", "STD", "", "PF-3", "", "L1", "AVAILABLE", "2026-01-12", "Register", "", ""),          // 5 warning (cost)
		row("Brand A", "MA", "STD", "", "CH-EXIST", "", "L1", "AVAILABLE", "2026-01-10", "Register", "", "90000"), // 6 duplicate in inventory
		row("Brand A", "MA", "STD", "", "PF-DUP", "", "L1", "AVAILABLE", "2026-01-10", "Register", "", "90000"),   // 7 duplicate in file
		row("Brand A", "MA", "STD", "", "pf-dup", "", "L1", "AVAILABLE", "2026-01-10", "Register", "", "90000"),   // 8 duplicate in file
		row("Brand P", "PA", "STD", "", "BP-1", "", "L1", "AVAILABLE", "2026-01-10", "Register", "", "90000"),     // 9 unverified brand
		row("Brand A", "MA", "STD", "", "", "", "L1", "AVAILABLE", "2026-01-10", "Register", "", "90000"),         // 10 no chassis
		row("Brand A", "MA", "STD", "", "CH-DRAFT", "", "L1", "AVAILABLE", "2026-01-10", "Register", "", "90000"), // 11 on a receipt draft
		row("Brand A", "MA", "STD", "", "PF-4", "", "L9", "AVAILABLE", "2026-01-10", "Register", "", "90000"),     // 12 inactive location
	), "upload-key-h08-01"), http.StatusCreated, &b)

	before := f.everythingFingerprint(t)
	p := f.preflight(t, f.importer, b.ID)
	again := f.preflight(t, f.importer, b.ID)
	if after := f.everythingFingerprint(t); after != before {
		t.Fatalf("preflight changed the database:\nbefore %s\nafter  %s", before, after)
	}

	if !p.RowsChecked || p.Source.Status != openingstock.SourceVerified || !p.Source.FileAvailable || !p.Source.ChecksumMatches ||
		!p.Source.SizeMatches || !p.Source.TemplateCompatible || !p.Source.MatchesStagedRows {
		t.Fatalf("source: %+v", p.Source)
	}
	want := openingstock.PreflightSummary{TotalRows: 10, ValidRows: 2, WarningRows: 1, BlockingErrorRows: 6, UnverifiedRows: 1,
		DuplicateRows: 4, MissingChassisRows: 1}
	if p.Summary != want {
		t.Fatalf("summary %+v, want %+v", p.Summary, want)
	}
	if p.PreviewSummary.TotalRows != 10 || p.PreviewSummary.ValidRows != 2 {
		t.Errorf("preview summary %+v", p.PreviewSummary)
	}
	ex := exceptionByLine(p)
	for line, c := range map[int]struct{ cat, code string }{
		5:  {openingstock.PreflightWarning, openingstock.CodeWarningNotAccepted},
		6:  {openingstock.PreflightError, openingstock.CodeDuplicateChassisDB},
		7:  {openingstock.PreflightError, openingstock.CodeDuplicateChassisFile},
		8:  {openingstock.PreflightError, openingstock.CodeDuplicateChassisFile},
		9:  {openingstock.PreflightUnverified, openingstock.CodeBrandNotActive},
		10: {openingstock.PreflightError, openingstock.CodeRequired},
		11: {openingstock.PreflightError, openingstock.CodeDuplicateChassisGRN},
		12: {openingstock.PreflightError, openingstock.CodeLocationInactive},
	} {
		r, ok := ex[line]
		if !ok || r.Category != c.cat || !hasCode(r, c.code) || r.ChangedSincePreview {
			t.Errorf("line %d: %+v, want %s with %s", line, r, c.cat, c.code)
		}
	}
	if _, ok := ex[3]; ok || len(p.Exceptions) != 8 {
		t.Errorf("exceptions: %d rows", len(p.Exceptions))
	}
	// Quantities as uploaded; counted quantities never invented.
	var fileRows int
	for _, q := range p.Quantities {
		fileRows += q.FileRows
		if q.PhysicalCount != nil || q.Difference != nil {
			t.Errorf("physical count invented: %+v", q)
		}
	}
	if fileRows != 10 || p.PhysicalCount.Status != openingstock.PhysicalNotRecorded || p.PhysicalCount.Reference != nil {
		t.Errorf("quantities %+v physical %+v", p.Quantities, p.PhysicalCount)
	}
	if got := blockerCodes(p); !slices.Equal(got, []string{openingstock.BlockerRowsNotValid, openingstock.BlockerApprovalMissing,
		openingstock.BlockerStatusesNotApproved, openingstock.BlockerPhysicalCount}) {
		t.Errorf("blockers %v", got)
	}
	// Repeatable: the same answer each time.
	p.CheckedAt, again.CheckedAt = again.CheckedAt, p.CheckedAt
	if p.Summary != again.Summary || len(p.Exceptions) != len(again.Exceptions) || !slices.Equal(blockerCodes(p), blockerCodes(again)) {
		t.Error("preflight not repeatable")
	}
	// The staged preview itself is untouched.
	if g, err := f.h.svc.Get(context.Background(), f.loadAccess(t, f.ids["importer"]), b.ID); err != nil || g.Summary != b.Summary ||
		g.Approval != nil || g.Commit != nil || g.CommitAvailable {
		t.Errorf("batch changed: %+v %v", g, err)
	}
}

func (f *fixture) loadAccess(t *testing.T, userID string) access.Access {
	t.Helper()
	a, err := f.h.grants.LoadAccess(context.Background(), userID, false)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestPreflightSeesChangesSincePreview(t *testing.T) {
	f := setup(t)
	var b openingstock.Batch
	expect(t, f.h.upload(&f.importer, "opening.csv", "text/csv", csvFile(
		row("Brand A", "MA", "STD", "", "PC-1", "", "L1", "AVAILABLE", "2026-01-10", "Register", "", "90000"),
		row("Brand A", "MA", "STD", "", "PC-2", "", "L1", "AVAILABLE", "2026-01-10", "Register", "", "90000"),
	), "upload-key-h08-02"), http.StatusCreated, &b)
	if p := f.preflight(t, f.importer, b.ID); p.Summary.ValidRows != 2 || len(p.Exceptions) != 0 {
		t.Fatalf("clean preflight: %+v", p.Summary)
	}
	// After the preview, PC-2 appears on a goods receipt draft: the preflight reports it, the preview does not change.
	f.receipt("PC-2", "")
	p := f.preflight(t, f.importer, b.ID)
	r, ok := exceptionByLine(p)[4]
	if !ok || r.Category != openingstock.PreflightError || !r.ChangedSincePreview || r.PreviewStatus != openingstock.RowValid ||
		r.CurrentStatus != openingstock.RowError || !hasCode(r, openingstock.CodeDuplicateChassisGRN) {
		t.Fatalf("line 4: %+v", r)
	}
	if p.Summary.ValidRows != 1 || p.Summary.ChangedSincePreviewRows != 1 {
		t.Fatalf("summary %+v", p.Summary)
	}
	if staged := byLine(f.rows(t, f.importer, b.ID, ""))[4]; staged.Status != openingstock.RowValid {
		t.Errorf("staged row changed: %+v", staged)
	}
}

func TestPreflightPermissionsAndReadOnlyTransaction(t *testing.T) {
	c := setupCommit(t, commitLimits)
	b := c.previewOf(t, validFile("PP"), "upload-key-h08-03")
	path := "/opening-stock-imports/" + b.ID + "/preflight"

	expectError(t, c.h.get(nil, path), http.StatusUnauthorized, "UNAUTHENTICATED")
	expectError(t, c.h.get(&c.noPerm, path), http.StatusForbidden, "FORBIDDEN")
	// Another previewer limited to L1 does not see the importer's batch.
	expectError(t, c.h.get(&c.other, path), http.StatusNotFound, "NOT_FOUND")
	expectError(t, c.h.get(&c.importer, "/opening-stock-imports/not-a-uuid/preflight"), http.StatusNotFound, "NOT_FOUND")
	// Approvers and committers may read it; reading grants nothing.
	c.preflight(t, c.approver, b.ID)
	c.preflight(t, c.committer, b.ID)
	// Only GET exists: a POST is not a commit path.
	if w := c.h.post(&c.committer, path, nil, "commit-key-h08-01"); w.Code != http.StatusMethodNotAllowed && w.Code != http.StatusNotFound {
		t.Errorf("POST preflight: %d", w.Code)
	}

	// The transaction refuses writes (SQLSTATE 25006).
	var writeErr error
	openingstock.SetPreflightProbe(func(ctx context.Context, tx pgx.Tx) error {
		// In a savepoint, so the refused write does not abort the preflight's transaction.
		sp, err := tx.Begin(ctx)
		if err != nil {
			return err
		}
		_, writeErr = sp.Exec(ctx, `UPDATE opening_stock_import_batches SET file_name = file_name WHERE false`)
		return sp.Rollback(ctx)
	})
	t.Cleanup(func() { openingstock.SetPreflightProbe(nil) })
	c.preflight(t, c.importer, b.ID)
	var pgErr *pgconn.PgError
	if !errors.As(writeErr, &pgErr) || pgErr.Code != "25006" {
		t.Fatalf("write inside preflight transaction: %v", writeErr)
	}
	openingstock.SetPreflightProbe(nil)

	// Approved with approved statuses configured: only the physical-count quantities remain (H-08).
	before := c.stockFingerprint(t)
	c.approve(t, b.ID, true)
	p := c.preflight(t, c.importer, b.ID)
	if got := blockerCodes(p); !slices.Equal(got, []string{openingstock.BlockerPhysicalCount}) {
		t.Fatalf("approved blockers %v", got)
	}
	if p.PhysicalCount.Status != openingstock.PhysicalReferenceOnly || p.PhysicalCount.Reference == nil ||
		*p.PhysicalCount.Reference != "PC-TEST-1" || p.Summary.ValidRows != 3 {
		t.Fatalf("approved preflight %+v %+v", p.PhysicalCount, p.Summary)
	}
	if c.stockFingerprint(t) != before {
		t.Fatal("preflight created stock")
	}
	if n := c.count(t, `SELECT count(*) FROM vehicle_units WHERE source_type = 'OPENING_STOCK'`); n != 0 {
		t.Fatalf("opening stock units: %d", n)
	}

	// Committed (test configuration only): rows are not re-checked, the batch is final.
	expect(t, c.h.post(&c.committer, "/opening-stock-imports/"+b.ID+"/commit", nil, "commit-key-h08-02"), http.StatusCreated, nil)
	p = c.preflight(t, c.importer, b.ID)
	if p.RowsChecked || p.Source.Status != openingstock.SourceNotChecked || !slices.Equal(blockerCodes(p),
		[]string{openingstock.BlockerAlreadyCommitted}) || p.Commit == nil {
		t.Fatalf("committed preflight %+v %v", p.Source, blockerCodes(p))
	}
}

func TestPreflightSourceIntegrity(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	data := csvFile(row("Brand A", "MA", "STD", "", "SI-1", "", "L1", "AVAILABLE", "2026-01-10", "Register", "", "90000"))
	var b openingstock.Batch
	expect(t, f.h.upload(&f.importer, "opening.csv", "text/csv", data, "upload-key-h08-04"), http.StatusCreated, &b)
	var key string
	if err := f.pool.QueryRow(ctx, `SELECT a.object_key FROM attachments a JOIN opening_stock_import_batches b ON b.file_attachment_id = a.id
		WHERE b.id = $1`, b.ID).Scan(&key); err != nil {
		t.Fatal(err)
	}
	if err := f.h.store.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	p := f.preflight(t, f.importer, b.ID)
	if p.Source.Status != openingstock.SourceInvalid || p.Source.FileAvailable || p.RowsChecked ||
		!slices.Contains(blockerCodes(p), openingstock.BlockerSourceInvalid) {
		t.Fatalf("missing file: %+v %v", p.Source, blockerCodes(p))
	}
	if _, err := f.h.store.Put(ctx, key, bytes.NewReader(bytes.Replace(data, []byte("SI-1"), []byte("SI-9"), 1))); err != nil {
		t.Fatal(err)
	}
	p = f.preflight(t, f.importer, b.ID)
	if p.Source.Status != openingstock.SourceInvalid || !p.Source.FileAvailable || p.Source.ChecksumMatches || p.RowsChecked {
		t.Fatalf("tampered file: %+v", p.Source)
	}

	// A rejected upload: nothing to re-check.
	var rejected openingstock.Batch
	expect(t, f.h.upload(&f.importer, "bad.csv", "text/csv", []byte("brand\nx\n"), "upload-key-h08-05"), http.StatusCreated, &rejected)
	p = f.preflight(t, f.importer, rejected.ID)
	if p.RowsChecked || p.Source.Status != openingstock.SourceNotChecked || blockerCodes(p)[0] != openingstock.BlockerBatchRejected {
		t.Fatalf("rejected: %+v %v", p.Source, blockerCodes(p))
	}
}
