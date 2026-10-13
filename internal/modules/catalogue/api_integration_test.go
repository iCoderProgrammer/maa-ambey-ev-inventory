package catalogue

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/audit"
	"github.com/jarvisconsulting/maa-ambey-enterprises/internal/modules/auth"
)

// End-to-end API tests over a throwaway migrated database (dbtest). Data is
// placeholder only ("Brand A", "M1", example.com sources): nothing real is
// invented and nothing is seeded.

type apiFixture struct {
	fixture
	h       *harness
	admin   session // every catalogue permission
	viewer  session // catalogue.view only
	adminID string
}

func apiSetup(t *testing.T) apiFixture {
	t.Helper()
	f := setup(t)
	store := auth.NewPGStore(f.pool)
	hash, err := auth.HashPassword(testPassword)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	admin, err := store.CreateUser(ctx, audit.CLIActor("test"), "cat.admin", "Catalogue Admin", nil, hash)
	if err != nil {
		t.Fatal(err)
	}
	viewer, err := store.CreateUser(ctx, audit.CLIActor("test"), "cat.viewer", "Catalogue Viewer", nil, hash)
	if err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, store, NewPGStore(f.pool))
	a := apiFixture{fixture: f, h: h, adminID: admin.ID}
	a.admin = h.login("cat.admin", admin.ID)
	a.viewer = h.login("cat.viewer", viewer.ID)
	h.grant(a.admin, allCataloguePermissions...)
	h.grant(a.viewer, "catalogue.view")
	return a
}

func (a apiFixture) createBrand(t *testing.T, name string) Brand {
	t.Helper()
	var b Brand
	expect(t, a.h.call(a.admin, http.MethodPost, "/brands", map[string]any{"name": name}), http.StatusCreated, &b)
	return b
}

func (a apiFixture) createModel(t *testing.T, brandID, code string) Model {
	t.Helper()
	var m Model
	expect(t, a.h.call(a.admin, http.MethodPost, "/models", map[string]any{"brandId": brandID, "name": "Model " + code, "internalCode": code}), http.StatusCreated, &m)
	return m
}

func (a apiFixture) createVariant(t *testing.T, modelID, code string) Variant {
	t.Helper()
	var v Variant
	expect(t, a.h.call(a.admin, http.MethodPost, "/variants", map[string]any{"modelId": modelID, "name": "Variant " + code, "variantCode": code}), http.StatusCreated, &v)
	return v
}

func verifyBody(version int) map[string]any {
	return map[string]any{"version": version, "confidence": "SUPPLIER_CONFIRMED", "sourceUrl": "https://example.com/catalogue", "sourceDate": "2026-10-01"}
}

// verifyAndActivate verifies and activates a model/variant/colour at path ("/models/<id>").
func (a apiFixture) verifyAndActivate(t *testing.T, path string, version int) int {
	t.Helper()
	var item struct {
		Version         int    `json:"version"`
		LifecycleStatus string `json:"lifecycleStatus"`
	}
	expect(t, a.h.call(a.admin, http.MethodPost, path+"/verify", verifyBody(version)), http.StatusOK, &item)
	expect(t, a.h.call(a.admin, http.MethodPost, path+"/activate", map[string]any{"version": item.Version}), http.StatusOK, &item)
	if item.LifecycleStatus != StatusActive {
		t.Fatalf("%s not active", path)
	}
	return item.Version
}

func (a apiFixture) attachment(t *testing.T) string {
	t.Helper()
	id := uuid.NewString()
	sum := sha256.Sum256([]byte(id))
	if _, err := a.pool.Exec(context.Background(), `INSERT INTO attachments (id, object_key, storage_driver, content_type, size_bytes,
		sha256, original_name, uploaded_by) VALUES ($1, $2, 'local', 'application/pdf', 10, $3, 'evidence.pdf', $4)`,
		id, "2026/10/"+uuid.NewString(), sum[:], a.adminID); err != nil {
		t.Fatal(err)
	}
	return id
}

// auditCount counts audit events for an entity with an action and outcome.
func (a apiFixture) auditCount(t *testing.T, entityID, action, outcome string) int {
	t.Helper()
	var n int
	if err := a.pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events
		WHERE entity_id = $1 AND action = $2 AND outcome = $3`, entityID, action, outcome).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestCatalogueHappyPathAndListing(t *testing.T) {
	a := apiSetup(t)
	b := a.createBrand(t, "Brand A")
	if b.Slug != "brand-a" || b.LifecycleStatus != StatusPending || b.AuthorisationStatus != AuthUnverified || b.Operational {
		t.Fatalf("new brand %+v", b)
	}
	m := a.createModel(t, b.ID, "m1")
	if m.InternalCode != "M1" || m.BrandID != b.ID || m.Confidence != ConfidenceUnverified {
		t.Fatalf("model %+v", m)
	}
	v := a.createVariant(t, m.ID, "std")
	if v.BrandID != b.ID || v.VariantCode != "STD" {
		t.Fatalf("variant %+v", v)
	}
	var c Colour
	expect(t, a.h.call(a.admin, http.MethodPost, "/colours", map[string]any{"variantId": v.ID, "brandId": b.ID, "name": "Colour A", "colourCode": "c-a"}),
		http.StatusCreated, &c)
	if c.BrandID != b.ID || c.VariantID != v.ID {
		t.Fatalf("colour %+v", c)
	}

	// Viewers read; every audited create happened.
	var got Variant
	expect(t, a.h.call(a.viewer, http.MethodGet, "/variants/"+v.ID, nil), http.StatusOK, &got)
	for id, action := range map[string]string{b.ID: "catalogue.brand.created", m.ID: "catalogue.model.created",
		v.ID: "catalogue.variant.created", c.ID: "catalogue.colour.created"} {
		if a.auditCount(t, id, action, "SUCCESS") != 1 {
			t.Errorf("no audit %s", action)
		}
	}

	// Paging and filters. The seeded roster brands are listed too.
	for i := 2; i <= 4; i++ {
		a.createModel(t, b.ID, fmt.Sprintf("M%d", i))
	}
	var page ModelList
	expect(t, a.h.call(a.viewer, http.MethodGet, "/models?brandId="+b.ID+"&limit=2&page=2", nil), http.StatusOK, &page)
	if page.Total != 4 || len(page.Items) != 2 || page.Page != 2 || page.Limit != 2 {
		t.Fatalf("page %+v", page)
	}
	expect(t, a.h.call(a.viewer, http.MethodGet, "/models?q=m3", nil), http.StatusOK, &page)
	if page.Total != 1 || page.Items[0].InternalCode != "M3" {
		t.Fatalf("search %+v", page)
	}
	// LIKE wildcards are literal.
	expect(t, a.h.call(a.viewer, http.MethodGet, "/models?q=%25", nil), http.StatusOK, &page)
	if page.Total != 0 {
		t.Fatalf("wildcard matched %d", page.Total)
	}
	var brands BrandList
	expect(t, a.h.call(a.viewer, http.MethodGet, "/brands?status=PENDING_VERIFICATION&limit=100", nil), http.StatusOK, &brands)
	if brands.Total != 4 {
		t.Fatalf("brands %d (3 seeded + 1)", brands.Total)
	}
	var empty VariantList
	expect(t, a.h.call(a.viewer, http.MethodGet, "/variants?status=ACTIVE", nil), http.StatusOK, &empty)
	if empty.Total != 0 || empty.Items == nil {
		t.Fatalf("empty list must be [] not null: %+v", empty)
	}
}

func TestBrandScopedUniquenessAndCrossBrandRules(t *testing.T) {
	a := apiSetup(t)
	b1, b2 := a.createBrand(t, "Brand A"), a.createBrand(t, "Brand B")
	m1 := a.createModel(t, b1.ID, "M1")
	a.createModel(t, b2.ID, "M1") // same code in another brand is fine

	e := expectError(t, a.h.call(a.admin, http.MethodPost, "/models", map[string]any{"brandId": b1.ID, "name": "Other", "internalCode": "m1"}),
		http.StatusConflict, "DUPLICATE")
	if !hasField(e.Details, "internalCode") {
		t.Fatal(e)
	}
	expectError(t, a.h.call(a.admin, http.MethodPost, "/brands", map[string]any{"name": "brand a", "slug": "other"}), http.StatusConflict, "DUPLICATE")

	// A variant cannot claim another brand than its model's.
	e = expectError(t, a.h.call(a.admin, http.MethodPost, "/variants", map[string]any{"modelId": m1.ID, "brandId": b2.ID, "name": "V", "variantCode": "STD"}),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if !hasField(e.Details, "brandId") {
		t.Fatal(e)
	}
	v := a.createVariant(t, m1.ID, "STD")
	expectError(t, a.h.call(a.admin, http.MethodPost, "/colours", map[string]any{"variantId": v.ID, "brandId": b2.ID, "name": "C"}),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	// Unknown parents.
	expectError(t, a.h.call(a.admin, http.MethodPost, "/models", map[string]any{"brandId": uuid.NewString(), "name": "M", "internalCode": "M9"}),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	expectError(t, a.h.call(a.admin, http.MethodPost, "/variants", map[string]any{"modelId": uuid.NewString(), "name": "V", "variantCode": "X"}),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	// The brand of a model never changes.
	expectError(t, a.h.call(a.admin, http.MethodPatch, "/models/"+m1.ID, map[string]any{"version": m1.Version, "brandId": b2.ID}),
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	// Filtering one brand never returns another brand's rows.
	var list VariantList
	expect(t, a.h.call(a.viewer, http.MethodGet, "/variants?brandId="+b2.ID+"&modelId="+m1.ID, nil), http.StatusOK, &list)
	if list.Total != 0 {
		t.Fatal("cross-brand filter returned rows")
	}
}

func TestActivationRequiresVerificationAndActiveParents(t *testing.T) {
	a := apiSetup(t)
	b := a.createBrand(t, "Brand A")
	m := a.createModel(t, b.ID, "M1")
	v := a.createVariant(t, m.ID, "STD")

	// Unverified model: refused and audited as DENIED.
	e := expectError(t, a.h.call(a.admin, http.MethodPost, "/models/"+m.ID+"/activate", map[string]any{"version": m.Version}),
		http.StatusConflict, "ACTIVATION_BLOCKED")
	if !hasField(e.Details, "verification") {
		t.Fatal(e)
	}
	if a.auditCount(t, m.ID, "catalogue.model.activated", "DENIED") != 1 {
		t.Fatal("denied activation not audited")
	}
	// Verified variant under a pending model: refused.
	var vv Variant
	expect(t, a.h.call(a.admin, http.MethodPost, "/variants/"+v.ID+"/verify", verifyBody(v.Version)), http.StatusOK, &vv)
	if vv.Confidence != ConfidenceSupplierConfirmed || vv.VerifiedBy == nil || *vv.VerifiedBy != a.adminID || vv.SourceDate == nil || *vv.SourceDate != "2026-10-01" {
		t.Fatalf("verification %+v", vv.Provenance)
	}
	e = expectError(t, a.h.call(a.admin, http.MethodPost, "/variants/"+v.ID+"/activate", map[string]any{"version": vv.Version}),
		http.StatusConflict, "ACTIVATION_BLOCKED")
	if !hasField(e.Details, "parent") {
		t.Fatal(e)
	}

	// The right order works: model, then variant.
	a.verifyAndActivate(t, "/models/"+m.ID, m.Version)
	expect(t, a.h.call(a.admin, http.MethodPost, "/variants/"+v.ID+"/activate", map[string]any{"version": vv.Version}), http.StatusOK, &vv)
	if vv.LifecycleStatus != StatusActive || a.auditCount(t, v.ID, "catalogue.variant.activated", "SUCCESS") != 1 {
		t.Fatal("variant activation")
	}
	// Activating twice is a state error.
	expectError(t, a.h.call(a.admin, http.MethodPost, "/variants/"+v.ID+"/activate", map[string]any{"version": vv.Version}),
		http.StatusConflict, "INVALID_STATE")
	// The activate permission is separate from manage.
	a.h.grant(a.viewer, "catalogue.view", "catalogue.manage", "catalogue.verify")
	expectError(t, a.h.call(a.viewer, http.MethodPost, "/variants/"+v.ID+"/activate", map[string]any{"version": vv.Version}),
		http.StatusForbidden, "FORBIDDEN")
	// Source documents must exist.
	m2 := a.createModel(t, b.ID, "M2")
	body := verifyBody(m2.Version)
	body["sourceDocumentId"] = uuid.NewString()
	expectError(t, a.h.call(a.admin, http.MethodPost, "/models/"+m2.ID+"/verify", body), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
}

func TestBrandAuthorisationAndActivationGate(t *testing.T) {
	a := apiSetup(t)
	b := a.createBrand(t, "Brand A")

	var r Readiness
	expect(t, a.h.call(a.viewer, http.MethodGet, "/brands/"+b.ID+"/activation-readiness", nil), http.StatusOK, &r)
	if r.Ready {
		t.Fatal("bare brand ready")
	}
	e := expectError(t, a.h.call(a.admin, http.MethodPost, "/brands/"+b.ID+"/activate", map[string]any{"version": b.Version}),
		http.StatusConflict, "ACTIVATION_BLOCKED")
	for _, k := range []string{"authorisation", "legal_identity", "authorisation_evidence", "verified_catalogue", "approved_prices"} {
		if !hasField(e.Details, k) {
			t.Errorf("missing unmet %s", k)
		}
	}
	if a.auditCount(t, b.ID, "catalogue.brand.activated", "DENIED") != 1 {
		t.Fatal("denied brand activation not audited")
	}

	// Unknown evidence document is refused; a real one is accepted and the caller is the verifier.
	body := map[string]any{"version": b.Version, "authorisationStatus": "AUTHORISED", "manufacturerLegalName": "Placeholder Legal Name",
		"authorisationReference": "REF-TEST-1", "authorisationEvidenceId": uuid.NewString()}
	expectError(t, a.h.call(a.admin, http.MethodPut, "/brands/"+b.ID+"/authorisation", body), http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	body["authorisationEvidenceId"] = a.attachment(t)
	expect(t, a.h.call(a.admin, http.MethodPut, "/brands/"+b.ID+"/authorisation", body), http.StatusOK, &b)
	if b.AuthorisationStatus != AuthAuthorised || b.VerifiedBy == nil || *b.VerifiedBy != a.adminID || b.VerifiedAt == nil || b.Operational {
		t.Fatalf("authorised brand %+v", b)
	}
	if a.auditCount(t, b.ID, "catalogue.brand.authorisation_recorded", "SUCCESS") != 1 {
		t.Fatal("authorisation not audited")
	}

	// With a verified, active catalogue only the not-yet-built prerequisites remain, so activation stays blocked.
	m := a.createModel(t, b.ID, "M1")
	a.verifyAndActivate(t, "/models/"+m.ID, m.Version)
	v := a.createVariant(t, m.ID, "STD")
	a.verifyAndActivate(t, "/variants/"+v.ID, v.Version)
	expect(t, a.h.call(a.viewer, http.MethodGet, "/brands/"+b.ID+"/activation-readiness", nil), http.StatusOK, &r)
	for _, c := range r.Checks {
		if c.Met == c.Pending {
			t.Errorf("check %s met=%v pending=%v", c.Key, c.Met, c.Pending)
		}
	}
	e = expectError(t, a.h.call(a.admin, http.MethodPost, "/brands/"+b.ID+"/activate", map[string]any{"version": b.Version}),
		http.StatusConflict, "ACTIVATION_BLOCKED")
	if len(e.Details) != len(pendingChecks) {
		t.Fatalf("unmet %+v", e.Details)
	}
	var stored Brand
	expect(t, a.h.call(a.viewer, http.MethodGet, "/brands/"+b.ID, nil), http.StatusOK, &stored)
	if stored.LifecycleStatus != StatusPending || stored.Operational {
		t.Fatal("blocked activation changed the brand")
	}

	// Revoking clears operational use; UNVERIFIED clears the verifier.
	expect(t, a.h.call(a.admin, http.MethodPut, "/brands/"+b.ID+"/authorisation", map[string]any{"version": b.Version, "authorisationStatus": "UNVERIFIED"}),
		http.StatusOK, &b)
	if b.VerifiedBy != nil || b.VerifiedAt != nil {
		t.Fatalf("verifier kept %+v", b)
	}
	// The seeded roster brands are still pending and unverified.
	var roster BrandList
	expect(t, a.h.call(a.viewer, http.MethodGet, "/brands?authorisationStatus=AUTHORISED", nil), http.StatusOK, &roster)
	if roster.Total != 0 {
		t.Fatal("a brand became authorised without evidence")
	}
}

func TestArchiveKeepsHistory(t *testing.T) {
	a := apiSetup(t)
	b := a.createBrand(t, "Brand A")
	m := a.createModel(t, b.ID, "M1")
	mv := a.verifyAndActivate(t, "/models/"+m.ID, m.Version)
	v := a.createVariant(t, m.ID, "STD")
	vv := a.verifyAndActivate(t, "/variants/"+v.ID, v.Version)

	// Active items are not edited (approved data is not overwritten).
	expectError(t, a.h.call(a.admin, http.MethodPatch, "/variants/"+v.ID, map[string]any{"version": vv, "name": "Renamed"}),
		http.StatusConflict, "INVALID_STATE")

	expect(t, a.h.call(a.admin, http.MethodPost, "/models/"+m.ID+"/archive", map[string]any{"version": mv, "reason": "Discontinued by supplier"}),
		http.StatusOK, &m)
	if m.LifecycleStatus != StatusArchived {
		t.Fatal("not archived")
	}
	var ev struct{ reason string }
	if err := a.pool.QueryRow(context.Background(), `SELECT summary->>'reason' FROM audit_events WHERE entity_id = $1 AND action = 'catalogue.model.archived'`,
		m.ID).Scan(&ev.reason); err != nil || ev.reason != "Discontinued by supplier" {
		t.Fatalf("archive audit: %v %q", err, ev.reason)
	}
	// Children and their references survive; nothing was deleted.
	var got Variant
	expect(t, a.h.call(a.viewer, http.MethodGet, "/variants/"+v.ID, nil), http.StatusOK, &got)
	if got.ModelID != m.ID || got.BrandID != b.ID {
		t.Fatal("history lost its references")
	}
	// Archived parents take no new children, archived items no edits, and archive is not repeated.
	expectError(t, a.h.call(a.admin, http.MethodPost, "/variants", map[string]any{"modelId": m.ID, "name": "New", "variantCode": "NEW"}),
		http.StatusConflict, "INVALID_STATE")
	expectError(t, a.h.call(a.admin, http.MethodPatch, "/models/"+m.ID, map[string]any{"version": m.Version, "name": "X"}),
		http.StatusConflict, "INVALID_STATE")
	expectError(t, a.h.call(a.admin, http.MethodPost, "/models/"+m.ID+"/archive", map[string]any{"version": m.Version}),
		http.StatusConflict, "INVALID_STATE")
	// There is no hard delete.
	if w := a.h.call(a.admin, http.MethodDelete, "/models/"+m.ID, nil); w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("DELETE: %d", w.Code)
	}

	// Archived brand: no new models, no edits.
	expect(t, a.h.call(a.admin, http.MethodPost, "/brands/"+b.ID+"/archive", map[string]any{"version": b.Version}), http.StatusOK, &b)
	expectError(t, a.h.call(a.admin, http.MethodPost, "/models", map[string]any{"brandId": b.ID, "name": "M", "internalCode": "M2"}),
		http.StatusConflict, "INVALID_STATE")
	expectError(t, a.h.call(a.admin, http.MethodPatch, "/brands/"+b.ID, map[string]any{"version": b.Version, "name": "X"}),
		http.StatusConflict, "INVALID_STATE")
	var n int
	if err := a.pool.QueryRow(context.Background(), `SELECT count(*) FROM variants WHERE brand_id = $1`, b.ID).Scan(&n); err != nil || n != 1 {
		t.Fatalf("variants after archive: %d %v", n, err)
	}
}

func TestEditsAndOptimisticConcurrency(t *testing.T) {
	a := apiSetup(t)
	b := a.createBrand(t, "Brand A")
	m := a.createModel(t, b.ID, "M1")

	// Editing a verified (still pending) model resets its verification.
	expect(t, a.h.call(a.admin, http.MethodPost, "/models/"+m.ID+"/verify", verifyBody(m.Version)), http.StatusOK, &m)
	expect(t, a.h.call(a.admin, http.MethodPatch, "/models/"+m.ID, map[string]any{"version": m.Version, "name": "Model Renamed", "vehicleType": "scooter"}),
		http.StatusOK, &m)
	if m.Confidence != ConfidenceUnverified || m.VerifiedBy != nil || m.SourceURL != nil || m.Name != "Model Renamed" || *m.VehicleType != "SCOOTER" {
		t.Fatalf("edit %+v", m)
	}
	expect(t, a.h.call(a.admin, http.MethodPatch, "/models/"+m.ID, map[string]any{"version": m.Version, "vehicleType": nil}), http.StatusOK, &m)
	if m.VehicleType != nil {
		t.Fatal("null did not clear")
	}
	if a.auditCount(t, m.ID, "catalogue.model.updated", "SUCCESS") != 2 {
		t.Fatal("edits not audited")
	}

	// Stale version.
	expectError(t, a.h.call(a.admin, http.MethodPatch, "/models/"+m.ID, map[string]any{"version": m.Version - 1, "name": "Stale"}),
		http.StatusConflict, "VERSION_CONFLICT")

	// Two concurrent edits of the same version: exactly one wins.
	var wg sync.WaitGroup
	codes := make([]int, 8)
	for i := range codes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes[i] = a.h.call(a.admin, http.MethodPatch, "/brands/"+b.ID, map[string]any{"version": b.Version, "name": fmt.Sprintf("Brand A %d", i)}).Code
		}()
	}
	wg.Wait()
	ok := 0
	for _, c := range codes {
		switch c {
		case http.StatusOK:
			ok++
		case http.StatusConflict:
		default:
			t.Fatalf("unexpected status %d", c)
		}
	}
	if ok != 1 {
		t.Fatalf("%d concurrent edits succeeded", ok)
	}
	// Concurrent creates of the same code: one created, the rest DUPLICATE.
	codes = make([]int, 6)
	for i := range codes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes[i] = a.h.call(a.admin, http.MethodPost, "/models", map[string]any{"brandId": b.ID, "name": fmt.Sprintf("Race %d", i), "internalCode": "RACE"}).Code
		}()
	}
	wg.Wait()
	created := 0
	for _, c := range codes {
		if c == http.StatusCreated {
			created++
		} else if c != http.StatusConflict {
			t.Fatalf("unexpected status %d", c)
		}
	}
	if created != 1 {
		t.Fatalf("%d duplicates created", created)
	}
}

func TestReadOnlyUserCannotChangeAnything(t *testing.T) {
	a := apiSetup(t)
	b := a.createBrand(t, "Brand A")
	m := a.createModel(t, b.ID, "M1")
	for _, c := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, "/brands", map[string]any{"name": "Brand B"}},
		{http.MethodPatch, "/brands/" + b.ID, map[string]any{"version": b.Version, "name": "X"}},
		{http.MethodPut, "/brands/" + b.ID + "/authorisation", map[string]any{"version": b.Version, "authorisationStatus": "NOT_AUTHORISED"}},
		{http.MethodPost, "/brands/" + b.ID + "/activate", map[string]any{"version": b.Version}},
		{http.MethodPost, "/brands/" + b.ID + "/archive", map[string]any{"version": b.Version}},
		{http.MethodPost, "/models/" + m.ID + "/verify", verifyBody(m.Version)},
		{http.MethodPost, "/models/" + m.ID + "/archive", map[string]any{"version": m.Version}},
	} {
		expectError(t, a.h.call(a.viewer, c.method, c.path, c.body), http.StatusForbidden, "FORBIDDEN")
	}
	var stored Brand
	expect(t, a.h.call(a.viewer, http.MethodGet, "/brands/"+b.ID, nil), http.StatusOK, &stored)
	if stored.Version != b.Version {
		t.Fatal("forbidden request changed the brand")
	}
}
