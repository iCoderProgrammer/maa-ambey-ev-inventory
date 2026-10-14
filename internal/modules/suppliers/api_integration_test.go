package suppliers

import (
	"context"
	"crypto/sha256"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/audit"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/database"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/database/dbtest"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/access"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/auth"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/catalogue"
	"github.com/jarvisconsulting/maa-ambey-enterprises/seeds"
)

// Placeholder data only ("Example Supplier …", example GSTIN format, 9000000000):
// no real supplier, contact or document is invented.

type fixture struct {
	pool   *pgxpool.Pool
	h      *harness
	admin  session // every supplier permission + attachments.upload/view
	viewer session // suppliers.view only
	brandA string
	brandB string
}

func setup(t *testing.T) fixture {
	t.Helper()
	ctx := context.Background()
	pool := dbtest.MigratedPool(t)
	if _, err := database.ApplySeeds(ctx, pool, seeds.FS); err != nil {
		t.Fatal(err)
	}
	store := auth.NewPGStore(pool)
	hash, err := auth.HashPassword(testPassword)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{}
	for _, u := range []string{"admin", "viewer"} {
		created, err := store.CreateUser(ctx, audit.CLIActor("test"), u, u, nil, hash)
		if err != nil {
			t.Fatal(err)
		}
		ids[u] = created.ID
	}
	h := newHarness(t, store, NewPGStore(pool))
	f := fixture{pool: pool, h: h}
	f.admin = h.login("admin", ids["admin"])
	f.viewer = h.login("viewer", ids["viewer"])
	h.grant(f.admin, append(append([]string{}, allSupplier...), access.PermAttachmentsUpload, access.PermAttachmentsView)...)
	h.grant(f.viewer, access.PermSupplierView)
	for _, b := range []struct {
		name string
		dst  *string
	}{{"Brand A", &f.brandA}, {"Brand B", &f.brandB}} {
		if err := pool.QueryRow(ctx, `INSERT INTO brands (name, slug) VALUES ($1, lower(replace($1, ' ', '-'))) RETURNING id::text`, b.name).Scan(b.dst); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

func (f fixture) supplier(t *testing.T, name string) Supplier {
	t.Helper()
	var s Supplier
	expect(t, f.h.call(f.admin, http.MethodPost, "/suppliers", map[string]any{"legalName": name}), http.StatusCreated, &s)
	return s
}

// upload inserts a private attachment uploaded by the user (as POST /attachments would).
func (f fixture) upload(t *testing.T, userID string) string {
	t.Helper()
	id := uuid.NewString()
	sum := sha256.Sum256([]byte(id))
	if _, err := f.pool.Exec(context.Background(), `INSERT INTO attachments (id, object_key, storage_driver, content_type, size_bytes,
		sha256, original_name, uploaded_by) VALUES ($1, $2, 'local', 'application/pdf', 10, $3, 'evidence.pdf', $4)`,
		id, "2026/10/"+uuid.NewString(), sum[:], userID); err != nil {
		t.Fatal(err)
	}
	return id
}

func (f fixture) document(t *testing.T, supplierID string) string {
	t.Helper()
	att := f.upload(t, f.admin.userID)
	expect(t, f.h.call(f.admin, http.MethodPost, "/suppliers/"+supplierID+"/documents", map[string]any{"attachmentId": att, "documentType": "AUTHORISATION"}),
		http.StatusCreated, nil)
	return att
}

func (f fixture) detail(t *testing.T, id string) Detail {
	t.Helper()
	var d Detail
	expect(t, f.h.call(f.viewer, http.MethodGet, "/suppliers/"+id, nil), http.StatusOK, &d)
	return d
}

func (f fixture) verified(t *testing.T, name string) (Supplier, string) {
	t.Helper()
	s := f.supplier(t, name)
	doc := f.document(t, s.ID)
	expect(t, f.h.call(f.admin, http.MethodPost, "/suppliers/"+s.ID+"/verify", map[string]any{"version": s.Version, "reference": "Checked", "documentId": doc}),
		http.StatusOK, &s)
	return s, doc
}

func (f fixture) supplierLinkMet(t *testing.T, brandID string) bool {
	t.Helper()
	r, err := catalogue.NewPGStore(f.pool).BrandReadiness(context.Background(), brandID)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range r.Checks {
		if c.Key == "supplier_link" {
			return c.Met
		}
	}
	t.Fatal("no supplier_link check")
	return false
}

func sqlState(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

func TestSupplierCreateSearchAndDuplicates(t *testing.T) {
	f := setup(t)
	var s Supplier
	expect(t, f.h.call(f.admin, http.MethodPost, "/suppliers", map[string]any{"legalName": "Example Supplier One", "gstin": "27abcde1234f1z5",
		"city": "Pune", "state": "Maharashtra", "postalCode": "411001"}), http.StatusCreated, &s)
	if s.Status != StatusPending || s.GSTIN == nil || *s.GSTIN != "27ABCDE1234F1Z5" || s.CreatedBy != f.admin.userID {
		t.Fatalf("created %+v", s)
	}
	expectError(t, f.h.call(f.admin, http.MethodPost, "/suppliers", map[string]any{"legalName": "example supplier one"}), http.StatusConflict, "DUPLICATE")
	e := expectError(t, f.h.call(f.admin, http.MethodPost, "/suppliers", map[string]any{"legalName": "Other", "gstin": "27ABCDE1234F1Z5"}),
		http.StatusConflict, "DUPLICATE")
	if !hasField(e.Details, "gstin") {
		t.Fatal(e)
	}
	f.supplier(t, "Example Supplier Two")
	var page SupplierPage
	expect(t, f.h.call(f.viewer, http.MethodGet, "/suppliers?q=one", nil), http.StatusOK, &page)
	if page.Total != 1 || page.Items[0].ID != s.ID {
		t.Fatalf("search %+v", page)
	}
	expect(t, f.h.call(f.viewer, http.MethodGet, "/suppliers?q=27ABCDE", nil), http.StatusOK, &page)
	if page.Total != 1 {
		t.Fatal("GSTIN search")
	}
	expect(t, f.h.call(f.viewer, http.MethodGet, "/suppliers?status=VERIFIED", nil), http.StatusOK, &page)
	if page.Total != 0 {
		t.Fatal("status filter")
	}
	// Concurrent creates of one name: exactly one succeeds.
	var wg sync.WaitGroup
	codes := make([]int, 6)
	for i := range codes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes[i] = f.h.call(f.admin, http.MethodPost, "/suppliers", map[string]any{"legalName": "Race Supplier"}).Code
		}()
	}
	wg.Wait()
	created := 0
	for _, c := range codes {
		if c == http.StatusCreated {
			created++
		} else if c != http.StatusConflict {
			t.Fatalf("unexpected %d", c)
		}
	}
	if created != 1 {
		t.Fatalf("%d created", created)
	}
}

func TestContactsKeepHistoryAndPrivacy(t *testing.T) {
	f := setup(t)
	s := f.supplier(t, "Example Supplier")
	var c1, c2 Contact
	expect(t, f.h.call(f.admin, http.MethodPost, "/suppliers/"+s.ID+"/contacts", map[string]any{"name": "Contact One", "phone": "9000000000", "isPrimary": true}),
		http.StatusCreated, &c1)
	expect(t, f.h.call(f.admin, http.MethodPost, "/suppliers/"+s.ID+"/contacts", map[string]any{"name": "Contact Two", "email": "two@example.com", "isPrimary": true}),
		http.StatusCreated, &c2)
	d := f.detail(t, s.ID)
	primaries := 0
	for _, c := range d.Contacts {
		if c.IsPrimary {
			primaries++
		}
	}
	if primaries != 1 || !d.Contacts[0].IsPrimary || d.Contacts[0].ID != c2.ID {
		t.Fatalf("primary %+v", d.Contacts)
	}
	// Removing keeps the row, inactive.
	expect(t, f.h.call(f.admin, http.MethodPost, "/suppliers/"+s.ID+"/contacts/"+c2.ID+"/remove", map[string]any{"version": c2.Version}), http.StatusOK, &c2)
	if c2.Active || c2.IsPrimary {
		t.Fatalf("removed %+v", c2)
	}
	expectError(t, f.h.call(f.admin, http.MethodPatch, "/suppliers/"+s.ID+"/contacts/"+c2.ID, map[string]any{"version": c2.Version, "name": "X"}),
		http.StatusConflict, "INVALID_STATE")
	if len(f.detail(t, s.ID).Contacts) != 2 {
		t.Fatal("contact deleted")
	}
	// A contact of another supplier is not reachable through this one.
	other := f.supplier(t, "Other Supplier")
	expectError(t, f.h.call(f.admin, http.MethodPatch, "/suppliers/"+other.ID+"/contacts/"+c1.ID, map[string]any{"version": c1.Version, "name": "X"}),
		http.StatusNotFound, "NOT_FOUND")
	// Audit never stores phone or email.
	var leaked int
	if err := f.pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action LIKE 'supplier.contact%'
		AND (summary::text LIKE '%9000000000%' OR summary::text LIKE '%two@example.com%')`).Scan(&leaked); err != nil || leaked != 0 {
		t.Fatalf("contact data in audit: %d %v", leaked, err)
	}
}

func TestDocumentsUsePrivateAttachments(t *testing.T) {
	f := setup(t)
	s := f.supplier(t, "Example Supplier")
	att := f.upload(t, f.admin.userID)
	var d Document
	expect(t, f.h.call(f.admin, http.MethodPost, "/suppliers/"+s.ID+"/documents", map[string]any{"attachmentId": att, "documentType": "GST_REGISTRATION"}),
		http.StatusCreated, &d)
	if d.FileName != "evidence.pdf" || d.AttachmentID != att {
		t.Fatalf("document %+v", d)
	}
	var ownerType, ownerID string
	if err := f.pool.QueryRow(context.Background(), `SELECT owner_type, owner_id FROM attachments WHERE id = $1`, att).Scan(&ownerType, &ownerID); err != nil ||
		ownerType != "supplier" || ownerID != s.ID {
		t.Fatalf("attachment owner %s %s %v", ownerType, ownerID, err)
	}
	// Already linked, someone else's upload, or unknown: refused.
	for _, a := range []string{att, f.upload(t, f.viewer.userID), uuid.NewString()} {
		expectError(t, f.h.call(f.admin, http.MethodPost, "/suppliers/"+s.ID+"/documents", map[string]any{"attachmentId": a, "documentType": "OTHER"}),
			http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	}
	// Documents are immutable in the database.
	if _, err := f.pool.Exec(context.Background(), `DELETE FROM supplier_documents WHERE id = $1`, d.ID); sqlState(err) != "42501" {
		t.Fatalf("delete document: %v", err)
	}
}

func TestVerificationNeedsOwnEvidence(t *testing.T) {
	f := setup(t)
	s := f.supplier(t, "Example Supplier")
	other := f.supplier(t, "Other Supplier")
	otherDoc := f.document(t, other.ID)
	// Another supplier's document is not evidence for this one.
	e := expectError(t, f.h.call(f.admin, http.MethodPost, "/suppliers/"+s.ID+"/verify", map[string]any{"version": s.Version, "reference": "R", "documentId": otherDoc}),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if !hasField(e.Details, "documentId") {
		t.Fatal(e)
	}
	doc := f.document(t, s.ID)
	expect(t, f.h.call(f.admin, http.MethodPost, "/suppliers/"+s.ID+"/verify", map[string]any{"version": s.Version, "reference": "GST checked", "documentId": doc}),
		http.StatusOK, &s)
	if s.Status != StatusVerified || s.VerifiedBy == nil || *s.VerifiedBy != f.admin.userID {
		t.Fatalf("verified %+v", s)
	}
	// Address edits keep verification; a new legal name resets it.
	expect(t, f.h.call(f.admin, http.MethodPatch, "/suppliers/"+s.ID, map[string]any{"version": s.Version, "city": "Pune"}), http.StatusOK, &s)
	if s.Status != StatusVerified {
		t.Fatal("address edit reset verification")
	}
	expect(t, f.h.call(f.admin, http.MethodPatch, "/suppliers/"+s.ID, map[string]any{"version": s.Version, "legalName": "Renamed Supplier"}), http.StatusOK, &s)
	if s.Status != StatusPending || s.VerifiedBy != nil || s.VerificationDocumentID != nil {
		t.Fatalf("identity edit kept verification %+v", s)
	}
	expectError(t, f.h.call(f.admin, http.MethodPatch, "/suppliers/"+s.ID, map[string]any{"version": s.Version - 1, "city": "X"}),
		http.StatusConflict, "VERSION_CONFLICT")
}

func TestMultiBrandRelationships(t *testing.T) {
	f := setup(t)
	s, doc := f.verified(t, "Example Supplier")
	pending := f.supplier(t, "Pending Supplier")
	var la, lb, lp BrandLink
	expect(t, f.h.call(f.admin, http.MethodPost, "/suppliers/"+s.ID+"/brand-links", map[string]any{"brandId": f.brandA}), http.StatusCreated, &la)
	expect(t, f.h.call(f.admin, http.MethodPost, "/suppliers/"+s.ID+"/brand-links", map[string]any{"brandId": f.brandB}), http.StatusCreated, &lb)
	if la.Status != LinkUnverified || la.Current {
		t.Fatalf("proposed %+v", la)
	}
	expectError(t, f.h.call(f.admin, http.MethodPost, "/suppliers/"+s.ID+"/brand-links", map[string]any{"brandId": f.brandA}), http.StatusConflict, "DUPLICATE")
	if f.detail(t, s.ID).BrandCount != 2 {
		t.Fatal("brand count")
	}
	// An unverified link does not satisfy readiness; a pending supplier's link cannot be verified.
	if f.supplierLinkMet(t, f.brandA) {
		t.Fatal("unverified link counted")
	}
	expect(t, f.h.call(f.admin, http.MethodPost, "/suppliers/"+pending.ID+"/brand-links", map[string]any{"brandId": f.brandA}), http.StatusCreated, &lp)
	expectError(t, f.h.call(f.admin, http.MethodPost, "/supplier-brand-links/"+lp.ID+"/verify", map[string]any{"version": lp.Version,
		"authorisationReference": "R", "evidenceDocumentId": doc, "effectiveFrom": "2026-01-01"}), http.StatusConflict, "INVALID_STATE")
	// Evidence must be this supplier's document.
	otherDoc := f.document(t, pending.ID)
	expectError(t, f.h.call(f.admin, http.MethodPost, "/supplier-brand-links/"+la.ID+"/verify", map[string]any{"version": la.Version,
		"authorisationReference": "DLR-1", "evidenceDocumentId": otherDoc, "effectiveFrom": "2026-01-01"}), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	today := time.Now().Format(time.DateOnly)
	expect(t, f.h.call(f.admin, http.MethodPost, "/supplier-brand-links/"+la.ID+"/verify", map[string]any{"version": la.Version,
		"authorisationReference": "DLR-1", "evidenceDocumentId": doc, "effectiveFrom": today}), http.StatusOK, &la)
	if la.Status != LinkVerified || !la.Current || la.VerifiedBy == nil {
		t.Fatalf("verified link %+v", la)
	}
	// Readiness: brand A has a current verified supplier, brand B does not.
	if !f.supplierLinkMet(t, f.brandA) || f.supplierLinkMet(t, f.brandB) {
		t.Fatal("readiness scoping")
	}
	var links []BrandLink
	expect(t, f.h.call(f.viewer, http.MethodGet, "/supplier-brand-links?brandId="+f.brandA, nil), http.StatusOK, &links)
	if len(links) != 2 {
		t.Fatalf("links for brand A: %d", len(links))
	}
	for _, l := range links {
		if l.BrandID != f.brandA {
			t.Fatal("brand filter leaked")
		}
	}
	// Ending keeps history and clears readiness; a new relationship may follow.
	expect(t, f.h.call(f.admin, http.MethodPost, "/supplier-brand-links/"+la.ID+"/end", map[string]any{"version": la.Version, "reason": "Agreement ended"}),
		http.StatusOK, &la)
	if la.Status != LinkEnded || la.EffectiveTo == nil || la.Current {
		t.Fatalf("ended %+v", la)
	}
	if f.supplierLinkMet(t, f.brandA) {
		t.Fatal("ended link counted")
	}
	expectError(t, f.h.call(f.admin, http.MethodPost, "/supplier-brand-links/"+la.ID+"/end", map[string]any{"version": la.Version, "reason": "x"}),
		http.StatusConflict, "INVALID_STATE")
	expect(t, f.h.call(f.admin, http.MethodPost, "/suppliers/"+s.ID+"/brand-links", map[string]any{"brandId": f.brandA}), http.StatusCreated, nil)
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `UPDATE supplier_brand_links SET status = 'VERIFIED' WHERE id = $1`, la.ID); sqlState(err) != "42501" {
		t.Fatalf("ended link changed: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `DELETE FROM supplier_brand_links WHERE id = $1`, lb.ID); sqlState(err) != "42501" {
		t.Fatalf("link deleted: %v", err)
	}
}

func TestArchivePreservesHistory(t *testing.T) {
	f := setup(t)
	s, doc := f.verified(t, "Example Supplier")
	var l BrandLink
	expect(t, f.h.call(f.admin, http.MethodPost, "/suppliers/"+s.ID+"/brand-links", map[string]any{"brandId": f.brandA}), http.StatusCreated, &l)
	expect(t, f.h.call(f.admin, http.MethodPost, "/supplier-brand-links/"+l.ID+"/verify", map[string]any{"version": l.Version,
		"authorisationReference": "DLR-1", "evidenceDocumentId": doc, "effectiveFrom": time.Now().Format(time.DateOnly)}), http.StatusOK, &l)
	expect(t, f.h.call(f.admin, http.MethodPost, "/suppliers/"+s.ID+"/contacts", map[string]any{"name": "Contact", "phone": "9000000000"}), http.StatusCreated, nil)
	expect(t, f.h.call(f.admin, http.MethodPost, "/suppliers/"+s.ID+"/archive", map[string]any{"version": s.Version, "reason": "No longer supplying"}),
		http.StatusOK, &s)
	d := f.detail(t, s.ID)
	if d.Status != StatusArchived || len(d.Contacts) != 1 || len(d.Documents) != 1 || len(d.BrandLinks) != 1 {
		t.Fatalf("history lost %+v", d)
	}
	// An archived supplier no longer satisfies readiness and cannot change.
	if f.supplierLinkMet(t, f.brandA) {
		t.Fatal("archived supplier counted")
	}
	expectError(t, f.h.call(f.admin, http.MethodPost, "/suppliers/"+s.ID+"/contacts", map[string]any{"name": "X", "phone": "9000000001"}),
		http.StatusConflict, "INVALID_STATE")
	expectError(t, f.h.call(f.admin, http.MethodPatch, "/suppliers/"+s.ID, map[string]any{"version": s.Version, "city": "X"}), http.StatusConflict, "INVALID_STATE")
	if _, err := f.pool.Exec(context.Background(), `DELETE FROM suppliers WHERE id = $1`, s.ID); sqlState(err) != "42501" {
		t.Fatalf("supplier deleted: %v", err)
	}
	var archived int
	if err := f.pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE entity_id = $1 AND action = 'supplier.archived'`, s.ID).
		Scan(&archived); err != nil || archived != 1 {
		t.Fatalf("archive audit %d %v", archived, err)
	}
}

func TestViewerCannotChangeAnything(t *testing.T) {
	f := setup(t)
	s := f.supplier(t, "Example Supplier")
	for _, c := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, "/suppliers", map[string]any{"legalName": "X"}},
		{http.MethodPatch, "/suppliers/" + s.ID, map[string]any{"version": s.Version, "city": "X"}},
		{http.MethodPost, "/suppliers/" + s.ID + "/archive", map[string]any{"version": s.Version}},
		{http.MethodPost, "/suppliers/" + s.ID + "/brand-links", map[string]any{"brandId": f.brandA}},
	} {
		expectError(t, f.h.call(f.viewer, c.method, c.path, c.body), http.StatusForbidden, "FORBIDDEN")
	}
	if d := f.detail(t, s.ID); d.Version != s.Version {
		t.Fatalf("changed by viewer: %d", d.Version)
	}
}
