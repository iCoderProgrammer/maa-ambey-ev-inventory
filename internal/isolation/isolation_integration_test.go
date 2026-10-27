package isolation

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/audit"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/database/dbtest"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/access"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/auth"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/catalogue"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/internal/modules/pricing"
	"github.com/iCoderProgrammer/maa-ambey-ev-inventory/seeds"
)

// Placeholder data only: random brand names, example.com sources, round
// amounts. Nothing real is invented.

const password = "correct horse battery staple"

type grants struct {
	mu    sync.Mutex
	perms map[string][]string
}

func (g *grants) LoadAccess(_ context.Context, userID string, _ bool) (access.Access, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return access.NewAccess(userID, nil, g.perms[userID], nil, false), nil
}

func (g *grants) RoleMatrix(context.Context) (access.Matrix, error) { return access.Matrix{}, nil }

type session struct {
	id     string
	cookie *http.Cookie
	csrf   string
}

type env struct {
	t       *testing.T
	pool    *pgxpool.Pool
	router  *gin.Engine
	grants  *grants
	admin   session // catalogue + selling/cost price draft
	checker session // price approver
	seller  session // catalogue.view + selling price view only
	nobody  session // signed in, no grants
}

var allCatalogue = []string{"catalogue.view", "catalogue.manage", "catalogue.verify", "catalogue.activate",
	"catalogue.archive", "catalogue.brand.authorise", "catalogue.brand.activate"}

func setup(t *testing.T) *env {
	t.Helper()
	ctx := context.Background()
	pool := dbtest.MigratedPool(t)
	if _, err := database.ApplySeeds(ctx, pool, seeds.FS); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := auth.NewPGStore(pool)
	authSvc, err := auth.NewService(store, auth.Config{IdleTimeout: time.Hour, AbsoluteTimeout: 2 * time.Hour},
		auth.NewLoginThrottle(50, 200, time.Minute), log)
	if err != nil {
		t.Fatal(err)
	}
	authH := auth.NewHandler(authSvc, auth.CookieConfig{}, log)
	g := &grants{perms: map[string][]string{}}
	accessH := access.NewHandler(g, false, log)
	r := gin.New()
	r.HandleMethodNotAllowed = true
	api := r.Group("/api/v1")
	api.Use(authH.RequireSession(auth.PublicRoutes("/api/v1")...))
	authH.Register(api)
	catalogue.NewHandler(catalogue.NewService(catalogue.NewPGStore(pool), log), accessH, log).Register(api)
	pricing.NewHandler(pricing.NewService(pricing.NewPGStore(pool), log), accessH, log).Register(api)

	e := &env{t: t, pool: pool, router: r, grants: g}
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	mk := func(name string, perms ...string) session {
		u, err := store.CreateUser(ctx, audit.CLIActor("test"), name, name, nil, hash)
		if err != nil {
			t.Fatal(err)
		}
		g.perms[u.ID] = perms
		return e.login(name, u.ID)
	}
	e.admin = mk("iso.admin", append(append([]string{}, allCatalogue...),
		"catalogue.price.view", "catalogue.price.manage", "catalogue.cost_price.view", "catalogue.cost_price.edit")...)
	e.checker = mk("iso.checker", "catalogue.view", "catalogue.price.view", "catalogue.price.approve", "catalogue.cost_price.view")
	e.seller = mk("iso.seller", "catalogue.view", "catalogue.price.view")
	e.nobody = mk("iso.nobody")
	return e
}

func (e *env) do(method, path, body string, s *session) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if s != nil {
		req.AddCookie(s.cookie)
		if method != http.MethodGet {
			req.Header.Set(auth.CSRFHeader, s.csrf)
		}
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return w
}

func (e *env) login(name, id string) session {
	e.t.Helper()
	w := e.do(http.MethodPost, "/api/v1/auth/login", `{"username":"`+name+`","password":"`+password+`"}`, nil)
	var body struct {
		Data auth.SessionResponse `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	s := session{id: id, csrf: body.Data.CSRFToken}
	for _, c := range w.Result().Cookies() {
		if c.Name == "maa_session" {
			s.cookie = c
		}
	}
	if s.cookie == nil {
		e.t.Fatalf("login %s: %s", name, w.Body)
	}
	return s
}

// call sends JSON and returns status and the decoded body.
func (e *env) call(s session, method, path string, body any) (int, map[string]any) {
	e.t.Helper()
	var payload string
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			e.t.Fatal(err)
		}
		payload = string(b)
	}
	w := e.do(method, "/api/v1"+path, payload, &s)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

// ok calls and expects a 2xx, returning data.
func (e *env) ok(s session, method, path string, body any) map[string]any {
	e.t.Helper()
	code, out := e.call(s, method, path, body)
	if code/100 != 2 {
		e.t.Fatalf("%s %s: %d %v", method, path, code, out)
	}
	data, _ := out["data"].(map[string]any)
	return data
}

// fails expects status and error code; returns the error's detail fields.
func (e *env) fails(s session, method, path string, body any, status int, code string) []string {
	e.t.Helper()
	got, out := e.call(s, method, path, body)
	errBody, _ := out["error"].(map[string]any)
	if got != status || errBody["code"] != code {
		e.t.Fatalf("%s %s: got %d %v, want %d %s", method, path, got, errBody, status, code)
	}
	var fields []string
	details, _ := errBody["details"].([]any)
	for _, d := range details {
		if m, ok := d.(map[string]any); ok {
			fields = append(fields, m["field"].(string))
		}
	}
	return fields
}

func str(m map[string]any, k string) string { s, _ := m[k].(string); return s }
func num(m map[string]any, k string) int    { f, _ := m[k].(float64); return int(f) }

func randomName(prefix string) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return prefix + " " + hex.EncodeToString(b)
}

var source = map[string]any{"confidence": "SUPPLIER_CONFIRMED", "sourceUrl": "https://example.com/iso", "sourceDate": "2026-10-01"}

// brandSet is a brand with one verified, active model and variant, a colour and a price list.
type brandSet struct {
	brand, model, variant, colour, list string
}

func (e *env) brandWithCatalogue(name string) brandSet {
	e.t.Helper()
	var b brandSet
	b.brand = str(e.ok(e.admin, http.MethodPost, "/brands", map[string]any{"name": name}), "id")
	m := e.ok(e.admin, http.MethodPost, "/models", map[string]any{"brandId": b.brand, "name": name + " Model", "internalCode": "M1"})
	m = e.ok(e.admin, http.MethodPost, "/models/"+str(m, "id")+"/verify", merge(source, "version", num(m, "version")))
	m = e.ok(e.admin, http.MethodPost, "/models/"+str(m, "id")+"/activate", map[string]any{"version": num(m, "version")})
	b.model = str(m, "id")
	v := e.ok(e.admin, http.MethodPost, "/variants", map[string]any{"modelId": b.model, "name": name + " Variant", "variantCode": "STD"})
	v = e.ok(e.admin, http.MethodPost, "/variants/"+str(v, "id")+"/verify", merge(source, "version", num(v, "version")))
	v = e.ok(e.admin, http.MethodPost, "/variants/"+str(v, "id")+"/activate", map[string]any{"version": num(v, "version")})
	b.variant = str(v, "id")
	b.colour = str(e.ok(e.admin, http.MethodPost, "/colours", map[string]any{"variantId": b.variant, "name": name + " Colour"}), "id")
	b.list = str(e.ok(e.admin, http.MethodPost, "/price-lists", map[string]any{"brandId": b.brand, "name": "Showroom"}), "id")
	return b
}

// approvedPrice drafts (admin) and approves (checker) a price starting at from.
func (e *env) approvedPrice(b brandSet, priceType, amount string, from time.Time) string {
	e.t.Helper()
	d := e.ok(e.admin, http.MethodPost, "/price-items", map[string]any{"priceListId": b.list, "variantId": b.variant,
		"priceType": priceType, "amount": amount, "effectiveFrom": from.Format(time.RFC3339)})
	a := e.ok(e.checker, http.MethodPost, "/price-items/"+str(d, "id")+"/approve", map[string]any{"version": num(d, "version")})
	return str(a, "id")
}

func merge(m map[string]any, k string, v any) map[string]any {
	out := map[string]any{k: v}
	for kk, vv := range m {
		out[kk] = vv
	}
	return out
}

func sqlState(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

// ── Tests ────────────────────────────────────────────────────────────────────

// An arbitrary, never-seen brand works through catalogue, pricing, lookup and
// readiness with data only — no code knows its name.
func TestArbitraryNewBrandNeedsNoCodeChange(t *testing.T) {
	e := setup(t)
	name := randomName("Zeta Mobility")
	b := e.brandWithCatalogue(name)
	if got := str(e.ok(e.seller, http.MethodGet, "/brands/"+b.brand, nil), "name"); got != name {
		t.Fatalf("name %q", got)
	}
	start := time.Now().Add(time.Hour).Truncate(time.Second)
	id := e.approvedPrice(b, "SELLING", "1000.00", start)
	got := e.ok(e.seller, http.MethodGet, "/prices/lookup?priceListId="+b.list+"&variantId="+b.variant+"&priceType=SELLING&at="+
		start.Add(time.Minute).UTC().Format(time.RFC3339), nil)
	if str(got, "id") != id || str(got, "brandId") != b.brand {
		t.Fatalf("lookup %v", got)
	}
	// The new brand appears in the brand list next to the seeded roster, pending and not operational.
	_, out := e.call(e.seller, http.MethodGet, "/brands?q="+strings.ReplaceAll(name, " ", "%20"), nil)
	items := out["data"].(map[string]any)["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["operational"] != false {
		t.Fatalf("brand list %v", items)
	}
}

func TestCrossBrandMismatchesFail(t *testing.T) {
	e := setup(t)
	a, b := e.brandWithCatalogue(randomName("Brand A")), e.brandWithCatalogue(randomName("Brand B"))

	// Writes: a child never joins another brand's parent.
	e.fails(e.admin, http.MethodPost, "/variants", map[string]any{"modelId": a.model, "brandId": b.brand, "name": "X", "variantCode": "X"},
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	e.fails(e.admin, http.MethodPost, "/colours", map[string]any{"variantId": a.variant, "brandId": b.brand, "name": "X"},
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	m := e.ok(e.admin, http.MethodPost, "/models", map[string]any{"brandId": a.brand, "name": "Pending", "internalCode": "P1"})
	e.fails(e.admin, http.MethodPatch, "/models/"+str(m, "id"), map[string]any{"version": num(m, "version"), "brandId": b.brand},
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	if f := e.fails(e.admin, http.MethodPost, "/price-items", map[string]any{"priceListId": a.list, "variantId": b.variant,
		"priceType": "SELLING", "amount": "1.00", "effectiveFrom": time.Now().Add(time.Hour).Format(time.RFC3339)},
		http.StatusUnprocessableEntity, "VALIDATION_FAILED"); len(f) == 0 || f[0] != "variantId" {
		t.Fatalf("price item fields %v", f)
	}

	// Filters that pair one brand with another brand's parent fail instead of returning nothing.
	e.fails(e.seller, http.MethodGet, "/variants?brandId="+b.brand+"&modelId="+a.model, nil, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	e.fails(e.seller, http.MethodGet, "/colours?brandId="+b.brand+"&variantId="+a.variant, nil, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	e.fails(e.seller, http.MethodGet, "/price-items?priceListId="+a.list+"&variantId="+b.variant, nil, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	e.fails(e.seller, http.MethodGet, "/price-items?brandId="+b.brand+"&priceListId="+a.list, nil, http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	e.fails(e.seller, http.MethodGet, "/prices/lookup?priceListId="+a.list+"&variantId="+b.variant+"&priceType=SELLING", nil,
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")
	// Matching filters work.
	e.ok(e.seller, http.MethodGet, "/variants?brandId="+a.brand+"&modelId="+a.model, nil)
	e.ok(e.seller, http.MethodGet, "/price-items?brandId="+a.brand+"&priceListId="+a.list+"&variantId="+a.variant, nil)

	// Supersession never crosses lists (or brands).
	start := time.Now().Add(time.Hour).Truncate(time.Second)
	aPrice := e.approvedPrice(a, "SELLING", "1000.00", start)
	e.fails(e.admin, http.MethodPost, "/price-items", map[string]any{"priceListId": b.list, "variantId": b.variant, "priceType": "SELLING",
		"amount": "1.00", "effectiveFrom": start.Add(time.Hour).Format(time.RFC3339), "supersedesId": aPrice},
		http.StatusUnprocessableEntity, "VALIDATION_FAILED")

	// The database refuses cross-brand rows even without the API.
	ctx := context.Background()
	for name, sql := range map[string]string{
		"variant under other brand's model": `INSERT INTO variants (brand_id, model_id, name, variant_code) VALUES ($1, $2, 'X', 'X')`,
		"price list of other brand":         `INSERT INTO price_items (price_list_id, brand_id, model_id, variant_id, price_type, amount, effective_from, created_by) SELECT $2, $1, v.model_id, v.id, 'SELLING', 1, now(), u.id FROM variants v, users u WHERE v.brand_id = $1 LIMIT 1`,
	} {
		var err error
		if strings.HasPrefix(name, "variant") {
			_, err = e.pool.Exec(ctx, sql, b.brand, a.model)
		} else {
			_, err = e.pool.Exec(ctx, sql, b.brand, a.list)
		}
		if sqlState(err) != "23503" {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := e.pool.Exec(ctx, `INSERT INTO variant_colours (brand_id, variant_id, name) VALUES ($1, $2, 'X')`, b.brand, a.variant); sqlState(err) != "23503" {
		t.Errorf("colour under other brand's variant: %v", err)
	}
}

func TestBrandFiltersStayScoped(t *testing.T) {
	e := setup(t)
	sets := []brandSet{
		e.brandWithCatalogue(randomName("Scope One")),
		e.brandWithCatalogue(randomName("Scope Two")),
		e.brandWithCatalogue(randomName("Scope Three")),
	}
	start := time.Now().Add(time.Hour).Truncate(time.Second)
	for _, s := range sets {
		e.approvedPrice(s, "SELLING", "1000.00", start)
		e.approvedPrice(s, "COST", "800.00", start)
	}
	for _, s := range sets {
		for path, want := range map[string]int{
			"/models?brandId=" + s.brand:              1,
			"/variants?brandId=" + s.brand:            1,
			"/colours?brandId=" + s.brand:             1,
			"/models?brandId=" + s.brand + "&q=Model": 1, // search matches every brand's model; the filter still scopes it
			"/price-lists?brandId=" + s.brand:         1,
			"/price-items?brandId=" + s.brand:         2,
			"/price-items?priceListId=" + s.list:      2,
		} {
			_, out := e.call(e.checker, http.MethodGet, path, nil)
			data := out["data"].(map[string]any)
			items := data["items"].([]any)
			if num(data, "total") != want || len(items) != want {
				t.Errorf("%s: total %d items %d, want %d", path, num(data, "total"), len(items), want)
			}
			for _, it := range items {
				if bid := it.(map[string]any)["brandId"]; bid != s.brand {
					t.Errorf("%s leaked brand %v", path, bid)
				}
			}
		}
	}
	// A selling-only user sees no cost prices in any brand.
	for _, s := range sets {
		_, out := e.call(e.seller, http.MethodGet, "/price-items?brandId="+s.brand, nil)
		for _, it := range out["data"].(map[string]any)["items"].([]any) {
			if it.(map[string]any)["priceType"] != "SELLING" {
				t.Fatal("cost price leaked to a selling-only user")
			}
		}
	}
}

func TestReadinessUsesOnlyTheBrandsOwnData(t *testing.T) {
	e := setup(t)
	a, b := e.brandWithCatalogue(randomName("Ready A")), e.brandWithCatalogue(randomName("Ready B"))
	// An approved selling price valid now for brand B only (approved at a moment before its start).
	d := e.ok(e.admin, http.MethodPost, "/price-items", map[string]any{"priceListId": b.list, "variantId": b.variant, "priceType": "SELLING",
		"amount": "1000.00", "effectiveFrom": time.Now().Add(2 * time.Second).Format(time.RFC3339)})
	e.ok(e.checker, http.MethodPost, "/price-items/"+str(d, "id")+"/approve", map[string]any{"version": num(d, "version")})
	time.Sleep(3 * time.Second)
	met := func(brand string) bool {
		r := e.ok(e.seller, http.MethodGet, "/brands/"+brand+"/activation-readiness", nil)
		for _, c := range r["checks"].([]any) {
			if m := c.(map[string]any); m["key"] == "approved_prices" {
				return m["met"] == true
			}
		}
		t.Fatal("no approved_prices check")
		return false
	}
	if !met(b.brand) {
		t.Fatal("brand B's own price not counted")
	}
	if met(a.brand) {
		t.Fatal("brand A satisfied by brand B's price")
	}
	// Archiving brand B leaves brand A untouched.
	bb := e.ok(e.admin, http.MethodGet, "/brands/"+b.brand, nil)
	e.ok(e.admin, http.MethodPost, "/brands/"+b.brand+"/archive", map[string]any{"version": num(bb, "version")})
	if str(e.ok(e.seller, http.MethodGet, "/brands/"+a.brand, nil), "lifecycleStatus") != "PENDING_VERIFICATION" {
		t.Fatal("archiving one brand changed another")
	}
}

func TestUnauthorisedAccessIsDenied(t *testing.T) {
	e := setup(t)
	a := e.brandWithCatalogue(randomName("Denied"))
	cost := e.approvedPrice(a, "COST", "800.00", time.Now().Add(time.Hour))
	for _, p := range []string{"/brands", "/models", "/variants", "/colours", "/price-lists", "/price-items",
		"/prices/lookup?priceListId=" + a.list + "&variantId=" + a.variant + "&priceType=SELLING"} {
		e.fails(e.nobody, http.MethodGet, p, nil, http.StatusForbidden, "FORBIDDEN")
	}
	if w := e.do(http.MethodGet, "/api/v1/brands", "", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", w.Code)
	}
	// Selling-only: cost price hidden (404), cost filter and lookup forbidden.
	e.fails(e.seller, http.MethodGet, "/price-items/"+cost, nil, http.StatusNotFound, "NOT_FOUND")
	e.fails(e.seller, http.MethodGet, "/price-items?priceType=COST", nil, http.StatusForbidden, "FORBIDDEN")
	e.fails(e.seller, http.MethodGet, "/prices/lookup?priceListId="+a.list+"&variantId="+a.variant+"&priceType=COST", nil,
		http.StatusForbidden, "FORBIDDEN")
	// Authorisation details are changed only with catalogue.brand.authorise.
	br := e.ok(e.seller, http.MethodGet, "/brands/"+a.brand, nil)
	e.fails(e.seller, http.MethodPut, "/brands/"+a.brand+"/authorisation", map[string]any{"version": num(br, "version"),
		"authorisationStatus": "NOT_AUTHORISED"}, http.StatusForbidden, "FORBIDDEN")
}
