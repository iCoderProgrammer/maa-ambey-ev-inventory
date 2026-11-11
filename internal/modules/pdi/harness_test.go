package pdi_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database/dbtest"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/httpx"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/pdi"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/vehicles"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/seeds"
)

// PDI tests on a throwaway migrated database. Brands, models, suppliers,
// chassis numbers, checklist items and amounts are placeholders ("Brand A",
// "CH-1", "BRAKES"): no real catalogue, checklist or vehicle data is invented.
// Approved purchase orders and goods receipts are inserted directly and
// "posting" is simulated by setting the receipt POSTED in the registering
// transaction (receipt posting is D-07). Attachments are inserted as metadata
// rows (the upload path is B-09's).

const testPassword = "correct horse battery staple"

type grants struct {
	mu        sync.Mutex
	perms     map[string][]string
	locations map[string][]access.Location
}

func (g *grants) LoadAccess(_ context.Context, userID string, _ bool) (access.Access, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return access.NewAccess(userID, nil, g.perms[userID], g.locations[userID], false), nil
}

func (g *grants) RoleMatrix(context.Context) (access.Matrix, error) { return access.Matrix{}, nil }

func (g *grants) access(userID string) access.Access {
	a, _ := g.LoadAccess(context.Background(), userID, false)
	return a
}

type session struct {
	userID string
	cookie *http.Cookie
	csrf   string
}

type fixture struct {
	t      *testing.T
	pool   *pgxpool.Pool
	router *gin.Engine
	grants *grants
	ids    map[string]string
	// inspector: record + view + attachments.view, every location.
	// scoped: record + view, location L2 only. viewer: view only (no attachments.view).
	// manager: checklist.manage. nobody: no permission.
	inspector, other, scoped, viewer, manager, nobody session
	locL1, locL2                                      string
	brandA, brandB                                    string
	po, poLineA, poLineB                              string
	nextNo                                            int
}

func setup(t *testing.T) *fixture {
	t.Helper()
	return setupOn(t, dbtest.MigratedPool(t))
}

func setupOn(t *testing.T, pool *pgxpool.Pool) *fixture {
	t.Helper()
	ctx := context.Background()
	if _, err := database.ApplySeeds(ctx, pool, seeds.FS); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := auth.NewPGStore(pool)
	authSvc, err := auth.NewService(store, auth.Config{IdleTimeout: time.Hour, AbsoluteTimeout: 2 * time.Hour},
		auth.NewLoginThrottle(100, 500, time.Minute), log)
	if err != nil {
		t.Fatal(err)
	}
	authH := auth.NewHandler(authSvc, auth.CookieConfig{}, log)
	f := &fixture{t: t, pool: pool, ids: map[string]string{},
		grants: &grants{perms: map[string][]string{}, locations: map[string][]access.Location{}}}
	r := gin.New()
	r.HandleMethodNotAllowed = true
	api := r.Group("/api/v1")
	api.Use(authH.RequireSession(auth.PublicRoutes("/api/v1")...))
	authH.Register(api)
	pdi.NewHandler(pdi.NewService(pdi.NewPGStore(pool), log), access.NewHandler(f.grants, false, log), log).Register(api)
	vehicles.NewHandler(vehicles.NewService(vehicles.NewPGStore(pool), log), access.NewHandler(f.grants, false, log), log).Register(api)
	f.router = r

	hash, err := auth.HashPassword(testPassword)
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range []string{"inspector", "other", "scoped", "viewer", "manager", "nobody", "buyer", "approver", "poster"} {
		created, err := store.CreateUser(ctx, audit.CLIActor("test"), u, u, nil, hash)
		if err != nil {
			t.Fatal(err)
		}
		f.ids[u] = created.ID
	}
	f.inspector = f.login("inspector")
	f.other = f.login("other")
	f.scoped = f.login("scoped")
	f.viewer = f.login("viewer")
	f.manager = f.login("manager")
	f.nobody = f.login("nobody")
	f.grant(f.inspector, access.PermPDIRecord, access.PermVehicleUnitView, access.PermAttachmentsView, access.PermAllLocations)
	f.grant(f.other, access.PermPDIRecord, access.PermVehicleUnitView, access.PermAttachmentsView, access.PermAllLocations)
	f.grant(f.scoped, access.PermPDIRecord, access.PermVehicleUnitView, access.PermAttachmentsView)
	f.grant(f.viewer, access.PermVehicleUnitView, access.PermAllLocations)
	f.grant(f.manager, access.PermPDIChecklistManage)

	f.locL1 = f.scalar(`INSERT INTO locations (code, name) VALUES ('L1', 'Location One') RETURNING id::text`)
	f.locL2 = f.scalar(`INSERT INTO locations (code, name) VALUES ('L2', 'Location Two') RETURNING id::text`)
	f.grants.mu.Lock()
	f.grants.locations[f.scoped.userID] = []access.Location{{ID: f.locL2}}
	f.grants.mu.Unlock()

	supplier := f.scalar(`INSERT INTO suppliers (legal_name, created_by) VALUES ('Example Supplier One', $1) RETURNING id::text`, f.ids["buyer"])
	f.po = f.scalar(`INSERT INTO purchase_orders (supplier_id, created_by, expected_date) VALUES ($1, $2, current_date + 7) RETURNING id::text`,
		supplier, f.ids["buyer"])
	f.brandA, f.poLineA = f.brandLine("Brand A", "brand-a", 1)
	f.brandB, f.poLineB = f.brandLine("Brand B", "brand-b", 2)
	f.exec(`UPDATE purchase_orders SET status = 'APPROVED', submitted_by = $2, submitted_at = now(), approval_basis = 'APPROVER',
		approved_by = $3, approved_at = now() WHERE id = $1`, f.po, f.ids["buyer"], f.ids["approver"])
	return f
}

func (f *fixture) brandLine(name, slug string, lineNo int) (brand, line string) {
	brand = f.scalar(`INSERT INTO brands (name, slug, lifecycle_status, authorisation_status, manufacturer_legal_name,
		authorisation_reference, verified_by, verified_at)
		VALUES ($1, $2, 'ACTIVE', 'AUTHORISED', 'Example Manufacturer', 'TEST-AUTH', $3, now()) RETURNING id::text`,
		name, slug, f.ids["approver"])
	model := f.scalar(`INSERT INTO models (brand_id, name, internal_code, lifecycle_status, confidence, source_url, source_date, verified_by, verified_at)
		VALUES ($1, 'Model M', 'M', 'ACTIVE', 'SUPPLIER_CONFIRMED', 'https://example.com/m', current_date, $2, now()) RETURNING id::text`,
		brand, f.ids["approver"])
	variant := f.scalar(`INSERT INTO variants (brand_id, model_id, name, variant_code, lifecycle_status, confidence, source_url, source_date, verified_by, verified_at)
		VALUES ($1, $2, 'Variant STD', 'STD', 'ACTIVE', 'SUPPLIER_CONFIRMED', 'https://example.com/v', current_date, $3, now()) RETURNING id::text`,
		brand, model, f.ids["approver"])
	line = f.scalar(`INSERT INTO purchase_order_lines (purchase_order_id, line_no, brand_id, model_id, variant_id, quantity, unit_cost)
		VALUES ($1, $2, $3, $4, $5, 50, 95000.00) RETURNING id::text`, f.po, lineNo, brand, model, variant)
	return brand, line
}

func (f *fixture) exec(sql string, args ...any) {
	f.t.Helper()
	if _, err := f.pool.Exec(context.Background(), sql, args...); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) scalar(sql string, args ...any) string {
	f.t.Helper()
	var s string
	if err := f.pool.QueryRow(context.Background(), sql, args...).Scan(&s); err != nil {
		f.t.Fatal(err)
	}
	return s
}

func (f *fixture) count(sql string, args ...any) int {
	f.t.Helper()
	n, _ := strconv.Atoi(f.scalar(`SELECT (`+sql+`)::text`, args...))
	return n
}

func (f *fixture) grant(s session, perms ...string) {
	f.grants.mu.Lock()
	defer f.grants.mu.Unlock()
	f.grants.perms[s.userID] = perms
}

func (f *fixture) login(username string) session {
	f.t.Helper()
	w := f.raw(http.MethodPost, "/api/v1/auth/login", `{"username":"`+username+`","password":"`+testPassword+`"}`, nil, nil)
	var body struct {
		Data auth.SessionResponse `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	s := session{userID: f.ids[username], csrf: body.Data.CSRFToken}
	for _, c := range w.Result().Cookies() {
		if c.Name == "maa_session" {
			s.cookie = c
		}
	}
	if s.cookie == nil {
		f.t.Fatalf("login %s failed: %s", username, w.Body)
	}
	return s
}

func (f *fixture) raw(method, target, body string, s *session, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if s != nil {
		req.AddCookie(s.cookie)
		if method != http.MethodGet {
			req.Header.Set(auth.CSRFHeader, s.csrf)
		}
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	return w
}

func (f *fixture) get(s session, path string) *httptest.ResponseRecorder {
	return f.raw(http.MethodGet, "/api/v1"+path, "", &s, nil)
}

func (f *fixture) post(s session, path string, body any, key string) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	h := map[string]string{}
	if key != "" {
		h["Idempotency-Key"] = key
	}
	return f.raw(http.MethodPost, "/api/v1"+path, string(b), &s, h)
}

func expect(t *testing.T, w *httptest.ResponseRecorder, status int, out any) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status %d, want %d: %s", w.Code, status, w.Body)
	}
	if out != nil {
		var env struct {
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(env.Data, out); err != nil {
			t.Fatalf("decode %s: %v", env.Data, err)
		}
	}
}

func expectError(t *testing.T, w *httptest.ResponseRecorder, status int, code string) httpx.ErrorBody {
	t.Helper()
	var env httpx.ErrorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	if w.Code != status || env.Error.Code != code {
		t.Fatalf("got %d %s, want %d %s: %s", w.Code, env.Error.Code, status, code, w.Body)
	}
	return env.Error
}

func hasField(e httpx.ErrorBody, field string) bool {
	for _, d := range e.Details {
		if d.Field == field {
			return true
		}
	}
	return false
}

// units registers units of a brand line at location (each with a chassis
// number unless noChassis) on a posted goods receipt, standing in for D-07.
func (f *fixture) units(location *string, line string, n int, noChassis bool) []string {
	f.t.Helper()
	ctx := context.Background()
	grn := f.scalar(`INSERT INTO goods_receipts (purchase_order_id, received_date, supplier_reference, created_by)
		VALUES ($1, DATE '2026-01-15', 'DC-1', $2) RETURNING id::text`, f.po, f.ids["poster"])
	grnLine := f.scalar(`INSERT INTO goods_receipt_lines (goods_receipt_id, purchase_order_id, purchase_order_line_id,
		received_quantity, accepted_quantity, rejected_quantity) VALUES ($1, $2, $3, $4, $4, 0) RETURNING id::text`,
		grn, f.po, line, n)
	var captured []string
	for i := 0; i < n; i++ {
		var ch any
		if !noChassis {
			f.nextNo++
			ch = "CH-" + strconv.Itoa(f.nextNo)
		}
		captured = append(captured, f.scalar(`INSERT INTO goods_receipt_units (goods_receipt_id, goods_receipt_line_id, unit_no,
			condition, chassis_number, chassis_normalised) VALUES ($1, $2, $3, 'ACCEPTED', $4::text, upper(btrim($4::text)))
			RETURNING id::text`, grn, grnLine, i+1, ch))
	}
	var ids []string
	actor := audit.UserActor(f.ids["poster"], "poster")
	err := pgx.BeginFunc(ctx, f.pool, func(tx pgx.Tx) error {
		for _, u := range captured {
			r, err := vehicles.RegisterFromReceiptUnit(ctx, tx, actor, "req-test", u, location)
			if err != nil {
				return err
			}
			ids = append(ids, r.ID)
		}
		_, err := tx.Exec(ctx, `UPDATE goods_receipts SET status = 'POSTED', posted_by = $2, posted_at = now() WHERE id = $1`,
			grn, f.ids["poster"])
		return err
	})
	if err != nil {
		f.t.Fatal(vehicles.CommitError(err))
	}
	return ids
}

func (f *fixture) unit(id string) (status string, version int) {
	f.t.Helper()
	if err := f.pool.QueryRow(context.Background(), `SELECT status, version FROM vehicle_units WHERE id = $1`, id).
		Scan(&status, &version); err != nil {
		f.t.Fatal(err)
	}
	return status, version
}

// checklistItem creates an active item through the API as the manager.
func (f *fixture) checklistItem(code string, brand *string, sort int) pdi.ChecklistItem {
	f.t.Helper()
	body := map[string]any{"code": code, "label": "Check " + code, "sortOrder": sort}
	if brand != nil {
		body["brandId"] = *brand
	}
	var c pdi.ChecklistItem
	expect(f.t, f.post(f.manager, "/pdi-checklist-items", body, ""), http.StatusCreated, &c)
	return c
}

// upload inserts an unlinked attachment of uploader at location (metadata only).
func (f *fixture) upload(uploader string, location *string) string {
	f.t.Helper()
	var loc any
	if location != nil {
		loc = *location
	}
	return f.scalar(`INSERT INTO attachments (object_key, storage_driver, content_type, size_bytes, sha256, original_name,
			uploaded_by, location_id)
		VALUES (to_char(now(), 'YYYY/MM/') || gen_random_uuid()::text, 'local', 'image/jpeg', 1024, sha256('x'::bytea), 'photo.jpg', $1, $2)
		RETURNING id::text`, f.ids[uploader], loc)
}

func outcomes(items []pdi.ChecklistItem, fail ...string) []map[string]any {
	out := []map[string]any{}
	for _, it := range items {
		o := map[string]any{"checklistItemId": it.ID, "outcome": "PASS"}
		for _, code := range fail {
			if it.Code == code {
				o["outcome"], o["note"] = "FAIL", "Defect on "+code
			}
		}
		out = append(out, o)
	}
	return out
}

func (f *fixture) record(s session, unitID string, body map[string]any, key string) *httptest.ResponseRecorder {
	return f.post(s, "/vehicle-units/"+unitID+"/pdi", body, key)
}

func (f *fixture) audits(action, outcome, entityID string) int {
	return f.count(`SELECT count(*) FROM audit_events WHERE action = $1 AND outcome = $2 AND entity_id = $3`, action, outcome, entityID)
}
