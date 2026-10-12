# Task Execution Log

Tracks progress of the 59 implementation tasks in `prompt-pack/TASK_REGISTRY.csv`, one task at a time.
Process: `docs/TASK_EXECUTION_WORKFLOW.md`. Statuses: `IN PROGRESS`, `BLOCKED`, `INCOMPLETE`, `COMPLETE`
(`NOT STARTED` = no task work recorded yet). Only one task may be `IN PROGRESS` at a time.

## Workflow configuration — 2026-10-12

- Created `CLAUDE.md`, `docs/TASK_EXECUTION_WORKFLOW.md` and this log. No feature implementation started.
- Git: branch `main`, no commits yet; every project file is untracked.
- Registry: `prompt-pack/TASK_REGISTRY.csv` (59 tasks, B-01 … H-10); the copy in
  `prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/` is byte-identical, as is `ALL_TASK_PROMPTS.md`
  vs `Maa_Ambey_Claude_Code_All_Task_Prompts_v1.2.md`.

### Pre-existing repository state (observed, NOT verified as task completion)

A foundation scaffold already exists from the monorepo setup prompt
(`00_START_HERE/Go_React_Monorepo_Setup_Prompt_v2.md`); README says "foundation only":

- Go API: `cmd/api`, `cmd/migrate`, `cmd/seed`; `internal/{config,database,httpx,logger,middleware,server,modules/health}` with unit and integration tests.
- Migration `000001_create_brands`; seed `seeds/0001_initial_brands.sql`; `templates/opening-stock-template.csv`.
- OpenAPI output in `docs/openapi/`; `docs/API_CONVENTIONS.md`, `ARCHITECTURE_DECISIONS.md`, `DEVELOPMENT_SETUP.md`, `REQUIREMENTS_BASELINE.md`.
- React app in `ui/` (Vite, Tailwind v4, TanStack Query, RHF + Zod, Redux Toolkit, Vitest). No Playwright installed.
- Root `Makefile`, `docker-compose.yml`, `.env.example`.

This overlaps B-01–B-05 (and the brand table touches C-01) but no task has been checked against its
acceptance criteria. Each must be verified when started; until then all tasks stay `NOT STARTED`.

## Status board

| Task ID | Title | Priority | Dependencies | Status |
| --- | --- | --- | --- | --- |
| B-01 | Repository and environment strategy | P0 | Workspace inspection | COMPLETE |
| B-02 | Lint, type-check, tests and CI | P0 | B-01 | COMPLETE |
| B-03 | React application shell | P0 | B-01 | COMPLETE |
| B-04 | Go API bootstrap and OpenAPI | P0 | B-01 | COMPLETE |
| B-05 | PostgreSQL and migration baseline | P0 | B-01, B-04 | NOT STARTED |
| B-06 | Authentication and session handling | P0 | B-04, B-05 | NOT STARTED |
| B-07 | RBAC and location scope | P0 | B-06, B-08 | NOT STARTED |
| B-08 | Append-only audit event framework | P0 | B-04, B-05, B-07 | NOT STARTED |
| B-09 | Private attachment service | P0 | B-04, B-05, B-07 | NOT STARTED |
| C-01 | Brand/model/variant database schema | P0 | B-05 | NOT STARTED |
| C-02 | Catalogue API and validation | P0 | C-01, B-07, B-08 | NOT STARTED |
| C-03 | Catalogue and brand management UI | P0 | C-02, B-03 | NOT STARTED |
| C-04 | Effective-dated price history | P0 | C-01, C-02 | NOT STARTED |
| C-05 | Seed verified multi-brand catalogue | P0 | A-01, C-01, C-02 | NOT STARTED |
| C-06 | Brand-specific rules and isolation tests | P0 | C-02, C-04, B-07 | NOT STARTED |
| D-01 | Supplier master | P0 | B-05, B-07, B-08 | NOT STARTED |
| D-02 | Purchase order lifecycle | P0 | C-02, D-01, C-04, B-07 | NOT STARTED |
| D-03 | Goods receipt draft workflow | P0 | D-02, B-09, C-01 | NOT STARTED |
| D-04 | Vehicle unit registry | P0 | C-01, B-05 | NOT STARTED |
| D-05 | Duplicate vehicle identifier controls | P0 | D-04 | NOT STARTED |
| D-06 | PDI checklist and disposition | P0 | D-04, B-09, B-07, B-08 | NOT STARTED |
| D-07 | Atomic receipt posting transaction | P0 | D-03, D-04, D-05, D-06, E-01 | NOT STARTED |
| D-08 | Vehicle list and detail UI | P0 | D-04, D-05, D-06, E-02 | NOT STARTED |
| D-09 | QR/barcode lookup and print | P0 | D-04, D-08, B-07 | NOT STARTED |
| E-01 | Append-only stock movement API | P0 | D-04, B-08 | NOT STARTED |
| E-02 | Stock movement timeline UI | P0 | E-01, D-08 | NOT STARTED |
| E-03 | Spare/accessory SKU master | P0 | B-05, D-01, B-07 | NOT STARTED |
| E-04 | Parts receipts and issues | P0 | E-03, E-01, D-01 | NOT STARTED |
| E-05 | Stock count and adjustment approval | P0 | E-01, B-07, B-08 | NOT STARTED |
| E-06 | Ledger reconciliation job/report | P0 | E-01, D-04, E-04 | NOT STARTED |
| E-07 | Stock ageing calculations | P0 | D-04, D-07 | NOT STARTED |
| F-01 | Customer master | P0 | B-06, B-07, B-08 | NOT STARTED |
| F-02 | Quotation builder and price snapshot | P0 | C-04, F-01, D-04, B-07 | NOT STARTED |
| F-03 | Discount approval workflow | P0 | B-07, B-08, F-02 | NOT STARTED |
| F-04 | Reservation lifecycle | P0 | F-01, F-02, D-04, E-01 | NOT STARTED |
| F-05 | Reservation expiry job | P0 | F-04, B-08 | NOT STARTED |
| F-06 | Sales order workflow | P0 | F-02, F-03, F-04 | NOT STARTED |
| F-07 | Atomic sale finalisation | P0 | F-06, F-04, F-08, F-09, E-01, B-08 | NOT STARTED |
| F-08 | Payment receipt tracking | P0 | F-06, B-07, B-08 | NOT STARTED |
| F-09 | Delivery checklist | P0 | F-06, D-06, B-07, B-08 | NOT STARTED |
| F-10 | Invoice, receipt and delivery templates | P0 | F-07, F-08, B-05 | NOT STARTED |
| F-11 | Cancellation, return and reversal workflow | P0 | F-07, F-08, E-01, B-08 | NOT STARTED |
| G-01 | Dashboard metrics | P0 | D-04, E-01, F-07, F-08 | NOT STARTED |
| G-02 | Inventory reports | P0 | D-04, E-07, C-06 | NOT STARTED |
| G-03 | Sales and collections reports | P0 | F-07, F-08, F-10 | NOT STARTED |
| G-04 | Pending procurement/PDI/reservation queues | P0 | D-02, D-03, D-06, F-04, F-05 | NOT STARTED |
| G-05 | CSV/XLSX export framework | P0 | B-07, B-08, G-02, G-03 | NOT STARTED |
| G-06 | Opening stock import preview | P0 | C-01, D-04, E-03, B-09 | NOT STARTED |
| G-07 | Opening stock import commit and idempotency | P0 | G-06, D-04, E-01, B-08 | NOT STARTED |
| H-01 | Unit and integration test suite | P0 | Relevant feature tasks implemented | NOT STARTED |
| H-02 | End-to-end critical workflows | P0 | D-07, D-06, F-04, F-07, F-09 | NOT STARTED |
| H-03 | Concurrency and idempotency tests | P0 | D-07, F-04, F-05, F-07, G-07 | NOT STARTED |
| H-04 | Security and access tests | P0 | B-06, B-07, B-09, G-05 | NOT STARTED |
| H-05 | Performance checks and indexes | P0 | Core list/search/report endpoints implemented | NOT STARTED |
| H-06 | Backup and restore drill | P0 | B-05, production-like environment | NOT STARTED |
| H-07 | Staging/UAT and defect triage | P0 | All P0 feature tasks | NOT STARTED |
| H-08 | Opening data import and physical reconciliation | P0 | G-06, G-07, D-04, E-06 | NOT STARTED |
| H-09 | Training documentation and runbooks | P0 | Stable workflows | NOT STARTED |
| H-10 | Production rollout and smoke test | P0 | H-04, H-05, H-06, H-07, H-08 | NOT STARTED |

Discovery items A-01–A-06 (client inputs; documentation-only) are tracked in
`prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/00_START_HERE/00_Client_Discovery_Prompts.md`.

---

## Task entries

### B-01 — Repository and environment strategy

- **Status:** COMPLETE
- **Started:** 2026-10-12 · **Finished:** 2026-10-12
- **Prompt file:** prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/01_Foundation_Security/B_01_repository-and-environment-strategy.md

#### Dependencies
| Dependency | Verified status | Evidence |
| --- | --- | --- |
| Workspace inspection | Done | Workflow configuration step (above): Git status, registry, prompt pack, specs and existing scaffold inspected |

#### Acceptance criteria
- [x] AC1 Fresh setup reproducible: clean copy of commit-eligible files → README quick start succeeded (see Stage 2)
- [x] AC2 No secrets tracked: `.env`, `.env.*`, `ui/.env*`, `*.pem/*.key/*.p12/*.pfx` ignored; only `*.env.example` eligible; secret-pattern scan of all 190 eligible files found only `change_me`/doc placeholders
- [x] AC3 Env vars documented: README tables (backend + UI) + new per-environment matrix and secrets policy in `docs/DEVELOPMENT_SETUP.md`
- [x] AC4 Uncommitted work preserved: nothing deleted/reset; user's running dev servers (5173/8081) and DB container untouched; scratch DB isolated via own Compose project and removed afterwards
- [x] AC5 (FE) Only `VITE_API_BASE_URL` reaches the bundle; built JS grep for `API_PROXY_TARGET`, DB URL, `change_me` = 0 hits
- [x] AC6 (FE) API base URL centralised in `ui/src/lib/config/env.ts`; ESLint now rejects `import.meta.env` elsewhere; documented
- [x] AC7 One repo, one npm package (`ui/`), no workspace added; no business behaviour changed

#### Stage 1 — UI
Result: NOT APPLICABLE — B-01 is repository/environment configuration with no user-facing screen
(Prompt B scope is env/config verification only).

#### Stage 2 — Backend / repository configuration
Result: COMPLETE
- Verified: `.gitignore`, root README, `docker-compose.yml` (DB bound to 127.0.0.1, creds from `.env`),
  `.env.example` + `ui/.env.example`, config validation (`internal/config`: prod rejects `sslmode=disable`,
  Swagger off by default in prod, `*` CORS rejected, DB URL never echoed), Gin release mode outside development.
- Defect fixed: `go test ./...` / `go vet ./...` included `ui/node_modules/flatted/golang/pkg/flatted`
  (npm package shipping Go code). Added `ignore ./ui` to `go.mod`; `go list ./...` now 11 packages, 0 from `ui/`.
- Hardening: `.gitignore` adds `*.pem`, `*.key`, `*.p12`, `*.pfx`; `.env.example` documents optional
  `COMPOSE_PROJECT_NAME` for a second checkout (shared container/volume otherwise).
- Docs: `docs/DEVELOPMENT_SETUP.md` secrets policy + local/staging/production matrix (derived from code);
  ADR-002 module-boundary note.
- Migrations / endpoints: none.
- Fresh-copy run (scratch dir, own Compose project `maa-ambey-b01-fresh`, ports 5441/8091/5181 because 5432/8080 are
  used by other projects on this machine — the documented troubleshooting step):
  `make setup` ✅ (npm ci 267 pkgs, 0 vulns) · `make db-up` ✅ healthy · `make db-migrate-up` ✅ v1 clean ·
  `make db-seed` ×2 ✅ (brands = 3, idempotent) · `make lint` ✅ · `make typecheck` ✅ · `make test` ✅ ·
  `make test-integration` ✅ (config, database, health, server ok) · `make build` ✅.
- Real repo after changes: `make lint` ✅ · `make typecheck` ✅ · `make test` ✅ (Go ok; Vitest 5 files / 22 tests) · `make build` ✅.
  (Go via `GO=/snap/go/current/bin/go`, as documented for snap Go.)

#### Stage 3 — Integration
Result: COMPLETE (no new API). Fresh API: `/api/v1/health` 200, `/api/v1/ready` 200 (database up),
Swagger 200; Vite proxy `/api/v1/health` 200. Added `ui/src/lib/config/env.test.ts` (default base URL,
trailing-slash trim, only `apiBaseUrl` exposed). ESLint rule verified by probe file (error raised; probe removed).

#### Stage 4 — Browser
Result: PASS (smoke) — no task-specific screen; checked that a fresh setup serves a working UI.
- Claude in Chrome → `http://localhost:5181/` (fresh copy): dashboard rendered, header "API online",
  System status: API Online, Database Ready. Console errors: none.
- Network: each load shows an aborted first `health`/`ready` pair (Resource Timing `responseStatus` 0,
  extension labels it 503) immediately re-issued → 200. API log: all 17 requests 200, no 503. Cause: React
  StrictMode dev double-mount cancelling the first query; dev-only, harmless, not B-01 scope.
- Evidence: `docs/evidence/B-01/fresh-setup-dashboard.jpg`.

#### Files changed
`go.mod`, `.gitignore`, `.env.example`, `ui/eslint.config.js`, `ui/src/lib/config/env.test.ts` (new),
`docs/DEVELOPMENT_SETUP.md`, `docs/ARCHITECTURE_DECISIONS.md`, `docs/evidence/B-01/fresh-setup-dashboard.jpg` (new),
`docs/TASK_EXECUTION_LOG.md`.

#### Known issues / pending approvals
- `go mod tidy -diff` (pre-existing): `github.com/swaggo/swag` marked `// indirect` but imported directly by
  `docs/openapi`. Harmless; belongs to B-02 (CI tidy check) / B-04 (OpenAPI).
- Deployment provider/host/region/backup RPO-RTO still open (OQ-5) → no staging/production manifests.
- Repo has no commits yet; nothing committed by this task.

#### Final status
COMPLETE

### B-02 — Lint, type-check, tests and CI

- **Status:** COMPLETE
- **Started:** 2026-10-12 · **Finished:** 2026-10-12
- **Prompt file:** prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/01_Foundation_Security/B_02_lint-type-check-tests-and-ci.md

#### Dependencies
| Dependency | Verified status | Evidence |
| --- | --- | --- |
| B-01 | COMPLETE | B-01 entry below; `go.mod ignore ./ui`, env docs, fresh-setup run |

#### Acceptance criteria
- [x] AC1 CI + documented local commands run Go checks: gofmt, vet, `go mod tidy -diff`, unit + PostgreSQL integration tests, build
- [x] AC2 CI + local commands run frontend checks: ESLint (0 warnings), Prettier, `tsc -b`, Vitest, Vite build
- [x] AC3 Every failing check returns non-zero — 14 injected failures, all non-zero (table below)
- [x] AC4 No silent skips where required: `make test-integration` exits 1 with empty DB URL; `REQUIRE_INTEGRATION_TESTS=1` turns the skip into `t.Fatal`; ESLint warnings now fail
- [x] AC5 Commands documented: README "Quality commands", `docs/DEVELOPMENT_SETUP.md` "Checks and CI"

#### Stage 1 — UI
Result: NOT APPLICABLE — tooling/CI task, no user-facing screen.

#### Stage 2 — Backend / tooling
Result: COMPLETE
- Gaps found: no CI at all; `make test-integration` silently skipped DB tests when the URL was empty;
  ESLint warnings (warn-level rules exist: `react-hooks/exhaustive-deps`, `incompatible-library`,
  `unsupported-syntax`) exited 0; no module-tidiness check; `swaggo/swag` wrongly `// indirect` (B-01 note).
- Changes:
  - `.github/workflows/ci.yml` (new): `backend` job (Go from `go.mod`, Postgres 18 service, lint-backend →
    migrate → seed → test-integration → build-backend) and `frontend` job (Node 24, npm ci, lint → typecheck →
    test → build). Read-only permissions, concurrency cancel. Runs Makefile targets only, so local == CI.
  - `Makefile`: `lint-backend` adds `go mod tidy -diff`; `test-integration` refuses an empty DB URL and sets
    `REQUIRE_INTEGRATION_TESTS=1`; new `typecheck-frontend` (no Go needed) with `typecheck` delegating to it;
    new `ci` aggregate target.
  - `internal/database/postgres_integration_test.go`: `testConfig` fails instead of skipping when
    `REQUIRE_INTEGRATION_TESTS=1`.
  - `ui/package.json`: `lint` = `eslint . --max-warnings=0`.
  - `go.mod`: `go mod tidy` (swag moved to direct require).
- golangci-lint kept optional (not installed here; no verified pinned config) — documented as not in CI.
- Migrations / endpoints: none.

#### Stage 3 — Integration / verification
Result: COMPLETE
- `actionlint` v1.7.7 (via `go run`, nothing installed globally) on `ci.yml`: exit 0.
- CI emulated locally in a clean copy (no `.env`, CI env vars, throwaway `postgres:18-alpine` container `b02-ci-pg`,
  removed afterwards). Backend job: lint-backend 0, db-migrate-up 0, db-seed 0, test-integration 0
  (`-v` shows TestConnectAndPing, TestBrandsConstraints, TestSeedsAreIdempotent **PASS**, not skipped),
  build-backend 0. Frontend job: npm ci 0, lint-frontend 0, typecheck-frontend 0, test-frontend 0 (22 tests), build-frontend 0.
- Real repo: `make ci` exit 0 (integration tests against local DB; writes rolled back).
- Injected-failure matrix (each injected alone, then removed):

| Injected failure | Target | Exit |
| --- | --- | --- |
| Unformatted Go file | lint-backend | 2 (`✗ gofmt needed`) |
| `go vet` printf mismatch | lint-backend | 2 |
| Untidy `go.mod` | lint-backend | 2 (`✗ go.mod/go.sum not tidy`) |
| Failing Go test | test-integration | 2 |
| Go compile error | build-backend | 2 |
| Empty `TEST_DATABASE_URL`/`DATABASE_URL` | test-integration | 2 (`✗ ... cannot run`) |
| `REQUIRE_INTEGRATION_TESTS=1`, no URL | `go test ./internal/database` | 1 (`t.Fatal`) |
| ESLint error (direct `fetch`) | lint-frontend | 2 |
| ESLint warning only (`no-console: warn`) | `npm run lint` | 1 (`too many warnings`); plain eslint exit 0 before this change |
| Prettier violation | lint-frontend | 2 |
| TypeScript error | typecheck-frontend | 2 |
| Failing Vitest test | test-frontend | 2 |
| Missing import | build-frontend | 2 |
| Vitest with no matching tests | `vitest run` | 1 |

- Not verified: the workflow has not run on GitHub (repo has no remote and no commits). First push will be its first real run.

#### Stage 4 — Browser
Result: NOT APPLICABLE — no UI or browser behaviour changed. E2E/Playwright is out of scope here.

#### Files changed
`.github/workflows/ci.yml` (new), `Makefile`, `go.mod`, `internal/database/postgres_integration_test.go`,
`ui/package.json`, `README.md`, `docs/DEVELOPMENT_SETUP.md`, `docs/TASK_EXECUTION_LOG.md`.

#### Known issues / pending approvals
- GitHub Actions run unverified until a remote exists and code is pushed.
- golangci-lint not enforced (optional locally); adopt with a pinned version + `.golangci.yml` if wanted.
- OpenAPI drift check belongs to B-04.

#### Final status
COMPLETE

### B-03 — React application shell

- **Status:** COMPLETE
- **Started:** 2026-10-12 · **Finished:** 2026-10-12
- **Prompt file:** prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/01_Foundation_Security/B_03_react-application-shell.md

#### Dependencies
| Dependency | Verified status | Evidence |
| --- | --- | --- |
| B-01 | COMPLETE | B-01 entry below |

#### Acceptance criteria
- [x] AC1 Shell renders on desktop / tablet / mobile (collapsible sidebar, drawer on < lg, top bar, breadcrumbs, page title)
- [x] AC2 Navigation works (sidebar, drawer, breadcrumbs, not-found, refresh/deep links)
- [x] AC3 Keyboard/focus: skip link, visible focus, drawer is a modal (trap, Escape, focus return), focus moves to content on navigation
- [x] AC4 No placeholder navigation claims an unimplemented workflow works (planned label in every state incl. collapsed + screen readers; profile placeholder offers no fake actions)
- [x] AC5 Light/dark themes with approved tokens; route-not-found and route-error pages
- [x] AC6 Backend: minimum health contract consumed by the shell verified (no business endpoints)

#### Stage 1 — UI
Result: UI COMPLETE — no mock data needed (the shell only consumes the existing real `/health` and `/ready`).
- Audit of existing shell vs Design Spec §2–3, §6, §8 and task scope. Present: collapsible sidebar, drawer, top bar,
  breadcrumbs, page title (`document.title`), planned-module pages with backlog IDs, not-found page, light/dark/system
  theme on spec tokens, skip link, API status.
- Gaps fixed:
  - Profile/menu placeholder missing → `UserMenu` ("Not signed in", disclosure panel explaining B-06/B-07; no actions;
    Escape/outside click close; focus returns to button).
  - Mobile drawer not a modal → `role="dialog"`, `aria-modal`, focus trap (Tab/Shift+Tab wrap), body scroll lock,
    focus returns to opener; opener has `aria-expanded`/`aria-controls`.
  - Focus lost after route change → focus moves to `<main>`; `ScrollRestoration` added.
  - Accessible name read "Vehicle UnitsPlanned" (pre-existing) and collapsed sidebar dropped "planned" → SR text
    ", planned" always; collapsed tooltip "(planned, not implemented yet)"; visual badge `aria-hidden`.
  - "Planned" badge contrast (white/40 on #101828 at 10 px) → white/65.
  - No route error boundary → `RouteErrorPage` inside the shell (safe message, reload, no stack trace).
- Two defects found by the new tests and fixed: `autoFocus` ran before the effect captured the opener (focus returned
  to a removed element); SR name concatenation (above).
- Checks: `npm run typecheck` ✅ · `npm run lint` (0 warnings) ✅ · `npm run format:check` ✅ · `vitest run` ✅ 6 files / 28 tests
  (new `AppShell.test.tsx`: user menu, drawer modal/trap/focus return, drawer navigation focus, collapsed planned labels,
  keyboard navigation focus + title, route error page).

#### Stage 2 — Backend
Result: COMPLETE (verify only, no code change). `/api/v1/health` (`HealthStatus`) and `/api/v1/ready` (`Readiness`,
200/503 same envelope) match the UI Zod schemas in `ui/src/features/system/api/health.ts`; documented in
`docs/openapi`; covered by 4 handler tests (`internal/modules/health`). No business endpoints added.

#### Stage 3 — Integration
Result: COMPLETE. Shell already uses the central `apiRequest` client + TanStack Query for health/readiness; no mock in
the production path. Verified in browser (Stage 4).

#### Stage 4 — Browser
Result: PASS. Claude in Chrome against the running dev stack (Vite :5173 → Go API :8081 → PostgreSQL :5436, all from
this repo, HMR). The browser window could not be resized (maximised at 1920), so tablet/mobile were tested in
same-origin iframes of exactly 820 px and 390 px (media queries use the iframe width).
- Desktop 1440/1920: sidebar, top bar, breadcrumbs, title, API status, "Not signed in" menu (opens, explains B-06/B-07,
  Escape closes) ✅; sidebar → Suppliers ✅; collapse → icon rail, names "Vehicle Units, planned" + tooltip ✅.
- Keyboard: first Tab shows "Skip to content", Enter focuses `#main`; focus ring 2 px primary outline, `:focus-visible` ✅.
- Dark theme ✅; preference persisted across reload ✅.
- Tablet 820 / mobile 390: desktop sidebar hidden, hamburger shown, no horizontal overflow (scrollWidth < width),
  account label collapses on mobile ✅. Drawer: `role=dialog aria-modal`, close button focused, body scroll locked,
  Shift+Tab wraps to last link, Tab wraps to close, Escape closes and focuses "Open navigation", link click navigates,
  closes drawer, focuses `<main>`, updates h1/title ✅.
- Deep link + refresh: `/inventory/does-not-exist` → "Page not found" (breadcrumb Home / Not found);
  `/sales/reservations` → planned page, active nav item ✅.
- Console: no errors or warnings. Network: every `/api/v1/health|ready` request completes 200, except one cancelled
  pair per full page load (Resource Timing `responseStatus` 0, extension shows "503") from React StrictMode's dev
  double mount — dev only, same as noted in B-01.
- Bugs found in browser and fixed:
  1. Page scrolled 41 px on load, breadcrumbs under the sticky header: new sr-only ", planned" spans were absolutely
     positioned relative to the sticky `<aside>`, escaping the nav's scroll clip and extending the document; plus
     `ScrollRestoration` restored a stale position under its initial `"default"` key across full loads. Fix: links are
     `relative`; `ScrollRestoration` replaced by scroll-to-top on route change. Re-verified: scrollHeight = viewport,
     scrollY 0.
  2. Pre-existing: "Collapse sidebar" button visible on tablet/mobile (no effect there) because `hidden` clashed with
     Button's base `inline-flex` (`cn` does not merge classes). Fix: desktop-only wrapper `hidden lg:flex`. Re-verified
     hidden at 820/390, visible on desktop.
- After testing, the theme was switched back to "system" via the UI (it had been changed during testing).
- Evidence: `docs/evidence/B-03/` (desktop, dark, tablet+mobile, planned module).

#### Final checks
`make ci` exit 0: lint (gofmt, vet, tidy, ESLint 0 warnings, Prettier), typecheck, Go unit + integration tests, Vitest
6 files / 28 tests, backend + frontend builds.

#### Files changed
`ui/src/components/layout/{AppShell,Header,Sidebar}.tsx`, `ui/src/components/layout/UserMenu.tsx` (new),
`ui/src/components/layout/AppShell.test.tsx` (new), `ui/src/app/routes.tsx`, `ui/src/app/RouteErrorPage.tsx` (new),
`docs/evidence/B-03/*` (new), `docs/TASK_EXECUTION_LOG.md`.

#### Known issues / pending approvals
- Design Spec §3 top-bar items (global search, quick-create, notifications, current location) not added: each needs a
  backing feature (D-08/D-09 search, B-07 location scope, etc.); adding them now would imply functionality.
- Design tokens are the spec's "starting proposal"; check against the client's public brand assets (Design Spec §8).
- Sidebar collapsed state is not persisted (not required).
- Dev-only StrictMode cancelled request pair per page load (harmless).

#### Final status
COMPLETE

### B-04 — Go API bootstrap and OpenAPI

- **Status:** COMPLETE
- **Started:** 2026-10-12 · **Finished:** 2026-10-12
- **Prompt file:** prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/01_Foundation_Security/B_04_go-api-bootstrap-and-openapi.md

#### Dependencies
| Dependency | Verified status | Evidence |
| --- | --- | --- |
| B-01 | COMPLETE | B-01 entry below |

#### Acceptance criteria
- [x] AC1 Go server starts (Gin, `/api/v1`, request IDs, structured JSON logs, recovery, timeout, graceful shutdown)
- [x] AC2 `GET /api/v1/health` responds; `GET /api/v1/ready` checks PostgreSQL (200 / 503)
- [x] AC3 OpenAPI route available; committed spec matches annotations (drift check)
- [x] AC4 API errors use one envelope (404, 405, 429, 500 panic, timeout)
- [x] AC5 Frontend calls the real API via the shared client with loading / error / offline states, no mock success

#### Stage 1 — UI
Result: UI COMPLETE — no mock (real `/health` + `/ready` already exist and are used through `apiRequest` + TanStack Query).
- Gap: with the browser offline TanStack Query pauses queries, so the widget showed "Contacting API…" forever
  (Design Spec §7: no unexplained indefinite spinner; show connectivity warning).
- Added `ui/src/hooks/useOnlineStatus.ts` (TanStack `onlineManager`); header pill shows "Offline"; System status card
  shows a warning alert (stale-data wording when cached data exists), hides spinners, disables Refresh; checks resume
  automatically on reconnect.
- New `SystemStatus.test.tsx`: loading states, offline (no fetch, alert, no spinner, Refresh disabled), resume on
  reconnect, server error envelope message + request ID shown. Vitest 7 files / 32 tests ✅; typecheck, ESLint, Prettier ✅.

#### Stage 2 — Backend
Result: COMPLETE.
- Verified existing: `cmd/api` (config validation, slog JSON, pgx pool, SIGINT/SIGTERM graceful shutdown with
  `SHUTDOWN_TIMEOUT`), `/api/v1` group, request ID (reuse valid / generate, echoed header + `requestId`), CORS allow-list,
  per-IP rate limit with `Retry-After`, NoRoute 404 / NoMethod 405 envelopes, Swagger UI toggle, `/health` + `/ready`.
- Fixed:
  1. `Timeout` middleware only cancelled the context: a handler giving up at the deadline without writing produced an
     empty 200. Now returns 503 `REQUEST_TIMEOUT` envelope (handler responses are kept). Message does not claim nothing
     changed (a write may have committed).
  2. Middleware order: access logger was inside recovery, so a recovered panic produced no `http_request` line. Logger
     now wraps recovery. Regression test fails with the old order (verified) and passes with the new.
  3. OpenAPI drift check: `make swagger-check` (regenerate into temp dir, diff), included in `make lint-backend` → CI.
     Verified: clean = 0, changed annotation = exit 2, restored = 0. (First version compared the wrong package name;
     fixed by generating into `$tmp/openapi`.)
  4. `/ready` OpenAPI lacked `429` (rate limiter applies to all `/api/v1`); annotated and regenerated.
- Docs: `docs/API_CONVENTIONS.md` (REQUEST_TIMEOUT row, middleware chain, OpenAPI section), README, DEVELOPMENT_SETUP.
- New tests: `TestTimeoutReturnsErrorEnvelope`, `TestTimeoutKeepsHandlerResponse`, `TestPanicIsAccessLoggedWithRequestID`
  (server package now 10 tests).
- Not changed (decision): the API still refuses to start when PostgreSQL is unreachable at boot (fail fast); `/ready`
  reports 503 if the database drops later.
- Migrations: none. Endpoints: unchanged (`GET /api/v1/health`, `GET /api/v1/ready`).
- `make ci` exit 0.
- Later in Stage 3/4 (found by live testing): CORS rejections from `gin-contrib/cors` were a bare 403 with empty body.
  `middleware.CORS` now pre-checks the origin with the same rules (no Origin / same host = not CORS; exact allow-list)
  and returns the 403 `FORBIDDEN` envelope. Tests `TestCORSRejectionUsesErrorEnvelope` (GET + preflight),
  `TestCORSIgnoresSameHostOrigin`.

#### Stage 3 — Integration
Result: COMPLETE. Shell uses the real `/api/v1/health` + `/ready` via `apiRequest` + TanStack Query; no mock path.
Defects found against the live API and fixed:
1. `/ready` 503 was an error to the client, so TanStack retried it twice and the card kept showing the cached "Ready"
   while the database was down. `apiRequest` gained `acceptStatuses`; `getReadiness` accepts 503 with a readiness body
   as data (`not_ready`), any other 503 body still throws.
2. Header pill said "Database unavailable" for any readiness error, including the API itself being down (proxy 502).
   Now: `not_ready` → "Database unavailable"; failed check → "Status unknown"; card shows "Readiness check failed"
   instead of stale cached "Ready".
3. While a failed check waits to retry (TanStack also pauses retries in hidden tabs), the last good values stayed on
   screen as current. Now pill "Reconnecting…" and card notice "The last check failed. Retrying… Values below may be
   out of date."
4. 502/504 had no friendly message ("Something went wrong"). Added `BAD_GATEWAY` / `GATEWAY_TIMEOUT` fallbacks.
- Tests added: client `acceptStatuses` (data / still-throws), 502 message; readiness 503 shown at once with production
  query defaults (1 `/ready` call — fails without the fix, verified); "Status unknown" vs DB outage; no stale "Ready"
  after failed refresh; "Reconnecting…" with production retry defaults. Vitest 7 files / 39 tests ✅.

#### Stage 4 — Browser
Result: PASS. Isolated stack so the user's dev servers stayed untouched: scratch copy, throwaway `postgres:18-alpine`
(`b04-pg`, :5443), freshly built API (`version=b04-test`, :8092), Vite :5182 proxying to it. Claude in Chrome.
- curl: health 200 + `X-Request-Id`; ready 200; valid incoming request ID reused, invalid one replaced; 404/405
  envelopes; Swagger UI 200 and `doc.json` (title, `/api/v1`, `/health`, `/ready`); allowed origin gets ACAO header;
  rejected origin → 403 `FORBIDDEN` envelope (after fix); Vite proxy → API 200; access logs are JSON with request IDs.
- Graceful shutdown: SIGTERM → `shutting down` → `shutdown complete`, process exits (twice).
- Browser: healthy dashboard shows real version `b04-test`, DB Ready. DB stopped → Refresh → "Database unavailable"
  pill + Unavailable row in ~1.0 s (before fix: stale "Ready" for several seconds). DB started → Refresh → Ready.
  API stopped → "The last check failed. Retrying…" then "API unreachable — The API could not be reached. Try again
  shortly." + "API offline"; API restarted → Try again → Online/Ready. Browser `offline` event → "Offline" pill, warning
  banner, Refresh disabled; `online` → normal. Swagger UI renders both operations and models.
- Console: no errors/warnings. Network: only expected 502/503 during the induced outages, plus the dev-only cancelled
  pair per page load (B-01 note).
- Note: during testing the tab once changed to `/admin/users` and the viewport size changed without any tool action
  (probably the user using that window); tests continued after returning to the dashboard.
- Evidence: `docs/evidence/B-04/` (7 screenshots).
- Cleanup: scratch API/Vite stopped, `b04-pg` removed; user's `maa-ambey-inventory-postgres-1` untouched.

#### Final checks
`make ci` exit 0 (incl. `swagger-check`), Go server tests 12, Vitest 39.

#### Files changed
Backend: `internal/middleware/timeout.go`, `internal/middleware/cors.go`, `internal/server/server.go`,
`internal/server/server_test.go`, `internal/modules/health/handler.go` (429 annotation), `docs/openapi/*` (regenerated),
`Makefile` (`swagger-check`, `SWAG_GEN_FLAGS`).
Frontend: `ui/src/lib/api/client.ts`, `ui/src/lib/api/client.test.ts`, `ui/src/hooks/useOnlineStatus.ts` (new),
`ui/src/features/system/{api/health.ts,hooks/useSystemStatus.ts,components/ApiStatusIndicator.tsx,components/SystemStatusCard.tsx}`,
`ui/src/features/system/components/SystemStatus.test.tsx` (new).
Docs: `docs/API_CONVENTIONS.md`, `docs/DEVELOPMENT_SETUP.md`, `README.md`, `docs/evidence/B-04/*`, this log.

#### Known issues / pending approvals
- The user's already-running dev API (:8081, started before this task) still runs the old backend code; restart
  `make dev` to pick up the timeout/CORS/log-order changes.
- API still fails fast at boot if PostgreSQL is unreachable (decision kept; `/ready` covers later outages).
- golangci-lint still optional (B-02 note).

#### Final status
COMPLETE

### UI-POLISH-01 — Existing UI polish pass (user-requested before B-05; not a registry task)

- **Status:** COMPLETE
- **Started:** 2026-10-12 · **Finished:** 2026-10-12
- **Instruction:** user asked to fix/polish the existing UI before B-05, following the "Master UI/UX Engineering
  Prompt v2" (persisted as `docs/UI_GUIDELINES.md`). Scope: existing shell, dashboard, planned/not-found/error pages,
  status widgets and shared UI primitives. No new business features, no backend changes. B-05 not started.

#### Dependency decision (guidelines §3)
Inspected `package.json`/lockfile: no component library; hand-built primitives (Button, Badge, Card, Dialog on native
`<dialog>`, Field, Input, Select, Table, StateViews); `cn` was a plain join (caused the B-03 class-conflict bug).
Installed (versions ≥ 2 weeks old, lockfile updated):
- `radix-ui@1.6.7` (2026-07-24; 1.7.0/1.8.0 are newer than 2 weeks) — accessible Dialog, Popover, Tooltip primitives.
- `clsx@2.1.1`, `tailwind-merge@3.7.0` (2026-09-12) — shadcn-style `cn` with Tailwind class conflict resolution.
Not installed: shadcn CLI scaffolding (components written in the same style by hand, only those needed), animation
libraries (CSS keyframes suffice), smooth-scroll libraries (native scrolling), fonts.

#### Changes (UI stage)
- `cn` = clsx + tailwind-merge (custom radius tokens registered) + `cn.test.ts`; removed the B-03 Header wrapper workaround.
- Motion tokens (`animate-fade/pop/drawer-*`) for Radix `data-state`; `prefers-reduced-motion` → 1 ms; native smooth
  scrolling only for anchors and only without reduced motion; route changes scroll instantly.
- Mobile drawer → Radix Dialog (focus trap, scroll lock, `aria-hidden` background, slide animation); custom trap code
  removed. Account menu → Radix Popover (info panel, no actions). Collapsed sidebar → Radix Tooltip (keyboard
  accessible; replaces `title`). `ui/Dialog` → Radix Dialog (same props) so modals and drawer share one system.
- Visual polish: Button (pointer cursor, pressed feedback, 150 ms transitions, `icon` size, `buttonVariants` for links),
  Badge inset ring, Card description + roomier padding, EmptyState icon/action, PageHeader status slot, header spacing,
  sidebar active accent bar + accent icon, quieter "Planned" chip, wrapped (not truncated) labels, width 256 → 272 px,
  themed thin scrollbar (hidden in the collapsed rail), planned-module page (status badge, "Delivered by", Back to
  dashboard), not-found page with "Go to dashboard".
- Docs: `docs/UI_GUIDELINES.md` (new, persisted user guidelines); `CLAUDE.md` and workflow Stage 1 point to it.

#### Defects found and fixed during the pass
1. Radix Dialog returns focus only to its own Trigger; the drawer opener is in the Header → focus fell to `<body>`
   after Escape (caught by test). Fix: header trigger id + explicit focus in `onCloseAutoFocus`. Browser-verified.
2. Radix `Slot` (Tooltip `asChild`) stringified NavLink's className function → collapsed rail lost all styling/icons
   (found in browser). Fix: `SidebarLink` computes `isActive` via `useMatch` and passes a string; regression test.
3. Four sidebar labels were truncated by the chip (found in browser) → wrap to two lines + wider sidebar.
4. Collapsed rail icons off-centre by the scrollbar (link 37 px) → scrollbar hidden in collapsed mode (47 px, centred).

#### Stage 2 — Backend
NOT APPLICABLE — UI-only pass; no API, schema or Go changes.

#### Stage 3 — Integration
COMPLETE — status widgets still use the real `/health` + `/ready` (no mocks introduced); verified live.

#### Stage 4 — Browser
PASS (user's dev stack :5173 → API :8081). Desktop light/dark; collapsed rail with keyboard tooltip
("Audit Log · Planned"); account popover; planned page; not-found; tablet 820 / mobile 390 in same-origin iframes (no
horizontal overflow, hamburger only below `lg`); drawer opened as titled modal with focus on close, background
`aria-hidden`, scroll locked; Escape → focus back to `mobile-nav-trigger`; drawer link → page changes, focus on
`<main>`. Console: no errors. The automation tab repeatedly became `hidden` (another window focused), which pauses CSS
exit animations until visible again — background-tab behaviour, not an app defect; unit tests cover close/focus.
Theme reset to "system" afterwards. Evidence: `docs/evidence/UI-POLISH-01/` (6 screenshots).

#### Checks
`make ci` exit 0: Go lint/tests/build unchanged; ESLint 0 warnings, Prettier, tsc, Vitest 8 files / 44 tests, build.

#### Files changed
`ui/package.json`, `ui/package-lock.json`, `ui/src/lib/utils/cn.ts` (+test), `ui/src/styles/index.css`,
`ui/src/components/ui/{Button,buttonVariants,Badge,Card,Dialog,Tooltip,StateViews}.tsx|ts`,
`ui/src/components/layout/{AppShell,Header,Sidebar,UserMenu,PageHeader,ThemeToggle,ids}.tsx|ts`,
`ui/src/components/layout/AppShell.test.tsx`, `ui/src/app/{PlannedModulePage,NotFoundPage}.tsx`, `ui/src/test/setup.ts`,
`docs/UI_GUIDELINES.md` (new), `CLAUDE.md`, `docs/TASK_EXECUTION_WORKFLOW.md`, `docs/evidence/UI-POLISH-01/*`.

#### Known issues
- JS bundle 485 → 599 kB (+36 kB gzip) from the Radix primitives; Vite warns about the 500 kB chunk limit. Not hidden;
  plan route-level code splitting (lazy routes) once real feature pages exist.
- Inter is not bundled; system sans-serif fallback (guidelines allow it).

#### Final status
COMPLETE — B-05 not started.

<!-- Copy this template for each started task. Update after every stage. -->

<!--
### [TASK_ID] — [Title]

- **Status:** IN PROGRESS / BLOCKED / INCOMPLETE / COMPLETE
- **Started:** YYYY-MM-DD · **Finished:** YYYY-MM-DD
- **Prompt file:** prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/<Epic>/<file>.md

#### Dependencies
| Dependency | Verified status | Evidence (code/tests/log) |
| --- | --- | --- |

#### Acceptance criteria
- [ ] …

#### Stage 1 — UI
Result: UI COMPLETE — MOCK DATA ONLY / NOT APPLICABLE (reason) / BLOCKED
Notes, mock adapter location, checks run + outcomes.

#### Stage 2 — Backend
Result: COMPLETE / NOT APPLICABLE / BLOCKED
Migrations: … · Routes: … · Checks run + outcomes.

#### Stage 3 — API integration
Result: COMPLETE / NOT APPLICABLE / BLOCKED
Endpoints integrated: … · Mocks removed: … · Checks run + outcomes.

#### Stage 4 — Browser testing
Result: PASS / FAIL / NOT APPLICABLE / NOT VERIFIED
Environment: … · Scenarios + results: … · Console/network findings: … · Evidence: docs/evidence/<TASK_ID>/

#### Files changed
#### Known issues / pending approvals
#### Final status
-->
