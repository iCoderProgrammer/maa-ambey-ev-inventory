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
| B-05 | PostgreSQL and migration baseline | P0 | B-01, B-04 | COMPLETE |
| B-06 | Authentication and session handling | P0 | B-04, B-05 | COMPLETE |
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

### B-05 — PostgreSQL and migration baseline

- **Status:** COMPLETE
- **Started:** 2026-10-12 · **Finished:** 2026-10-12
- **Prompt file:** prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/01_Foundation_Security/B_05_postgresql-and-migration-baseline.md
- **Branch:** `dev`. Pre-existing uncommitted changes not made by this task (Makefile `push` target, 21 UI
  styling files) are left untouched and are not part of B-05.

#### Dependencies
| Dependency | Verified status | Evidence (code/tests/log) |
| --- | --- | --- |
| B-01 | COMPLETE | B-01 entry; `.env.example`, compose, config validation tests |
| B-04 | COMPLETE | B-04 entry; `/api/v1/ready` pings PostgreSQL, error envelope, swagger-check |

#### Baseline found (foundation scaffold)
pgx pool (`internal/database/postgres.go`), config validation of `DATABASE_URL`, `cmd/migrate` (golang-migrate,
up/down N/version/force), migration `000001_create_brands`, transactional idempotent seeds (`cmd/seed`), Compose
PostgreSQL 18 with healthcheck, no auto-migrate on boot.

Gaps vs acceptance criteria: migration files are read from a working-directory path (binary not self-contained,
no check that every `up` has a `down`); `down`/`force` have no guard outside development; `cmd/migrate` duplicates
`DATABASE_URL` validation; `/ready` reports ready against an unmigrated or dirty schema; seeding an unmigrated DB
fails with a raw SQL error; no automated `up → down → up` test.

#### Acceptance criteria
- [x] AC1 DB starts through Docker Compose (healthcheck, localhost-only port, persistent volume)
- [x] AC2 Migrations apply and revert per a documented policy: every version has an up and a down, `up → down → up`
      tested automatically, `down`/`force` refused in staging/production without explicit confirmation, no "drop all"
- [x] AC3 Bad configuration fails clearly without leaking credentials (missing/invalid URL, missing database name,
      unreachable DB, unmigrated/dirty schema for seed)
- [x] AC4 No destructive auto-reset: API never migrates/drops on boot; seeds idempotent and transactional
- [x] AC5 Health/readiness integration: `/ready` reports database and schema state (503 when schema pending/dirty),
      frontend shows the reason without guessing
- [x] AC6 Frontend: local database startup and connection configuration documented; browser never talks to PostgreSQL

#### Stage 1 — UI
Result: COMPLETE — no mock needed (`/ready` already real; contract extended with a `migrations` check).
Prompt B scope is documentation only; the one UI change is so the new check is reported honestly:
- `ui/src/features/system/readinessChecks.ts` (new): check labels (`migrations` → "Database schema") and the
  not-ready reason (database first, then schema, else "Not ready").
- Header pill shows "Schema not ready" instead of the misleading "Database unavailable" when only the schema is
  behind; System status card labels the row and explains that an administrator must run migrations and data is
  never reset automatically.
- Tests: `readinessChecks.test.ts` (new), `SystemStatus.test.tsx` (+ schema-pending case). Vitest 47/47 ✅.

#### Stage 2 — Backend
Result: COMPLETE. No new migration, no new route (contract change on `GET /api/v1/ready` only).
- `migrations/migrations.go` (new): `//go:embed *.sql`, `Validate`/`Latest` enforce naming, version ≥ 1, no gaps,
  one name per version, every version has up + down.
- `seeds/seeds.go` (new): embedded seeds; `ApplySeeds` takes an `fs.FS` and errors when no files are found
  (found during CLI testing: running `bin/seed` outside the repo root printed "ok: 0 seed file(s) applied").
- `internal/database/schema.go` (new): `SchemaVersion`, `CheckSchema` (not migrated / dirty / behind / ahead),
  `SchemaChecker`.
- `internal/modules/health`: `/ready` adds `migrations` check (skipped and `down` when DB unreachable); reasons
  logged only. `server.NewRouter` requires the schema checker. `cmd/api` logs schema state at boot, never migrates.
- `internal/config`: exported `ValidateDatabaseURL` (adds "must name a database"), shared by API, migrate, seed.
- `cmd/migrate`: embedded source (iofs), shared URL validation, strict arg parsing, `down`/`force` refused for
  `APP_ENV=staging|production` without `--confirm`, `force` bounded to 0..latest, no drop command, connection
  errors redact the password (userinfo form only; first version mangled text when the password was short — fixed).
- `cmd/seed`: refuses a non-current schema with the reason.
- `Makefile`: `db-migrate-down` passes `CONFIRM=1` → `--confirm` (only this line; the pre-existing uncommitted
  `push` target is untouched).
- Tests: `migrations/migrations_test.go`, `cmd/migrate/main_test.go` (policy + bad config + redaction),
  `internal/database/schema_test.go`, `migrations_integration_test.go` (round trip `up → down → up` ×2 in a
  scratch database created and dropped by the test; dev schema current), seed empty-source test, health and
  server tests. Mutation check: removing `DROP TABLE` from the down script makes the round-trip test fail ✅.
- Commands: `make lint-backend` ✅ (gofmt, vet, tidy, swagger-check), `make test-integration` ✅ (all packages),
  `make build-backend` ✅, `make swagger` (description + example updated) ✅.
- CLI checks against throwaway DB `maa_b05_check` (dev DB untouched): missing URL / no DB name / wrong password
  (SQLSTATE 28P01, no password shown) / unreachable port / prod `down` refused / `drop` unknown / seed on
  unmigrated DB refused / up / up again "no change" / seed ×2 idempotent (3 brands, pending + unverified) /
  down / up / dirty flag → `/ready` 503 + seed refused → `force 1` → ready. API with bad URL and with DB down
  exits 1 with clear message.

#### Stage 3 — API integration
Result: COMPLETE. Frontend already consumes the real `/ready` via `apiRequest` + TanStack Query; no mock exists.
`docs/API_CONVENTIONS.md` and OpenAPI document the `migrations` check.

#### Stage 4 — Browser testing
Result: PASS. Claude in Chrome → Vite :5174 (`API_PROXY_TARGET=http://localhost:8091`) → freshly built API :8091 →
scratch DB `maa_b05_check` in the Compose PostgreSQL. The user's own servers (:5173/:8081) were not touched.
- Unmigrated schema: header "Schema not ready", rows Database = Ready, Database schema = Unavailable, explanation
  shown → `docs/evidence/B-05/01-schema-not-ready.jpg`.
- `migrate up` then Refresh (no API restart): "API online", both rows Ready →
  `docs/evidence/B-05/02-schema-ready-after-migrate-up.jpg`.
- Rolled back + reload: no console errors. Network: `/ready` 503 handled as data; an extra `/health`+`/ready`
  pair appears with status 0 (aborted by React StrictMode's dev double-mount; the extension labels it 503).
  API log confirms `/health` never returned non-200.
- Cleanup: tab closed, :5174/:8091 stopped, scratch DBs dropped; dev DB still version 1, clean, 3 brands.

#### Files changed (B-05)
`migrations/migrations.go`, `migrations/migrations_test.go`, `seeds/seeds.go`, `internal/database/{schema.go,
schema_test.go,seed.go,seed_integration_test.go,migrations_integration_test.go}`, `internal/config/{config.go,
config_test.go}`, `internal/modules/health/{handler.go,handler_test.go}`, `internal/server/{server.go,server_test.go}`,
`cmd/{api,migrate,seed}/main.go`, `cmd/migrate/main_test.go`, `docs/openapi/*`, `Makefile` (one target),
`ui/src/features/system/{readinessChecks.ts,readinessChecks.test.ts,api/health.ts,components/ApiStatusIndicator.tsx,
components/SystemStatusCard.tsx,components/SystemStatus.test.tsx}`, `docs/{DEVELOPMENT_SETUP,ARCHITECTURE_DECISIONS,
API_CONVENTIONS}.md`, `docs/evidence/B-05/*`.

#### Known issues / pending approvals
- `make lint-frontend` fails on Prettier for 9 files with pre-existing uncommitted UI edits not made by B-05
  (`PlannedModulePage`, `PageHeader`, `Card`, `Field`, `StateViews`, `DashboardPage`, `index.css`, plus those
  edits' lines in `ApiStatusIndicator`/`SystemStatusCard`). B-05's own lines pass Prettier and ESLint. Not
  reformatted here to avoid rewriting someone else's work. `make ci` will fail until they are formatted.
- The same uncommitted `DashboardPage` edit hardcodes the brand roster and labels it "Initial authorised and
  partner EV brands"; this conflicts with the rule that brands come from the API and roster ≠ authorisation.
  Not changed (outside B-05).
- The API already running on :8081 is a pre-B-05 build; restart it to get the `migrations` check.
- Backup/restore drill and production migration runbook remain H-06 / OQ-5.

#### Final status
COMPLETE — B-06 not started.


### B-06 — Authentication and session handling

- **Status:** COMPLETE
- **Started:** 2026-10-12 · **Finished:** 2026-10-12
- **Prompt file:** prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/01_Foundation_Security/B_06_authentication-and-session-handling.md
- **Auth decision (OQ-6):** approved by the user on 2026-10-12 ("Start B-06" after the proposal): local
  username/password with strong hashing (argon2id), server-side sessions in PostgreSQL, opaque token in a
  secure HttpOnly SameSite=Strict cookie, CSRF header token, login throttling. Matches PRD §NFR security,
  Architecture §2/§9 and SEC-006. No hardcoded admin: the first user is created with a CLI.
- **Branch / tree:** `dev`; B-05 is still uncommitted (user has not asked for a commit); pre-existing UI edits
  by someone else remain untouched except `UserMenu.tsx`, which is B-03's explicit B-06 placeholder.

#### Dependencies
| Dependency | Verified status | Evidence (code/tests/log) |
| --- | --- | --- |
| B-04 | COMPLETE | B-04 entry; error envelope, rate limiter, CORS; `make test-integration` ✅ 2026-10-12 |
| B-05 | COMPLETE | B-05 entry; embedded migrations, schema check, seeds; `make test-integration` ✅ 2026-10-12 |

#### Scope boundaries
- In: users + sessions schema, password hashing, login/logout/session endpoints, session middleware (deny by
  default for every non-public route), CSRF for mutations, login throttling, disabled-user denial, session
  revocation (logout, disable, password change, CLI), user-admin CLI, login screen, route protection, expired
  session handling, account menu with real sign-out.
- Out (later tasks): roles/permissions and location scope (B-07), persisted audit events (B-08; login events
  are logged with slog for now), user-management screens, password reset by e-mail, MFA.

#### Acceptance criteria
- [x] AC1 Unauthenticated API mutations are denied (401) — and, deny by default, every non-public route
- [x] AC2 Disabled user cannot sign in; disabling a user ends their existing sessions
- [x] AC3 Session invalidation works: logout, expiry (idle + absolute), disable, password change, CLI revoke
- [x] AC4 UI never pretends a mocked session is real: protected routes need a server-confirmed session
- [x] AC5 Passwords hashed (argon2id), generic login failure, throttled failures (429 + Retry-After), no
      credentials in localStorage, HttpOnly/SameSite cookie, CSRF on cookie-authenticated mutations
- [x] AC6 Login UI states: invalid credentials, disabled account, rate-limited, expired session, network/server
      error, pending (no double submit); accessible and responsive

#### Stage 1 — UI
Result: UI COMPLETE — MOCK DATA ONLY (API mocked in tests only; no runtime mock adapter, because the real
endpoints are built in Stage 2 of this task and a fake login must never reach the production path).
- Contract (also in `docs/API_CONVENTIONS.md` after Stage 2): `POST /auth/login {username,password}` →
  `{data:{user:{id,username,displayName},csrfToken,expiresAt}}` + HttpOnly cookie; `GET /auth/session` (401
  `UNAUTHENTICATED` / `SESSION_EXPIRED` = not signed in); `POST /auth/logout` → `{data:{signedOut:true}}`;
  mutations send `X-CSRF-Token`; errors `INVALID_CREDENTIALS` 401, `ACCOUNT_DISABLED` 403, `RATE_LIMITED` 429 +
  `Retry-After`, `CSRF_FAILED` 403.
- `ui/src/lib/api/csrf.ts` (in-memory CSRF token), `client.ts` (sends it on non-GET, `retryAfterSeconds`).
- `ui/src/features/auth/`: `api/auth.ts` (Zod schemas, 401 → anonymous state), `hooks/useSession.ts` (session
  query, login/logout mutations that drop other users' cached data), `components/RequireAuth.tsx` (pending,
  offline, cannot-check error, redirect with `from`), `LoginPage.tsx` (RHF + Zod, show/hide password, all error
  states, ended-session notice, double-submit guard), `redirect.ts` (no open redirect).
- `ui/src/app/queryClient.ts`: any 401 from another call marks the session ended → sign-in page;
  `CSRF_FAILED` re-reads the session. `routes.tsx`: `/login` public, everything else under `RequireAuth`.
- `UserMenu.tsx`: B-03 placeholder replaced with the confirmed user and a real "Sign out" (navigates only after
  the server confirms; failure says "You are still signed in").
- Tests: `auth.test.tsx` (14), `redirect.test.ts`, client CSRF/Retry-After tests; test render helper seeds a
  session fixture. Vitest 73/73 ✅, tsc ✅, ESLint ✅, Prettier ✅ for B-06 files.

#### Stage 2 — Backend
Result: COMPLETE.
- Migration `000002_create_users_and_sessions` (up/down): `users` (normalised username CHECK, unique username,
  case-insensitive unique e-mail, argon2id-only hash CHECK, status ACTIVE/DISABLED, version, `updated_at` trigger) and
  `user_sessions` (SHA-256 token hash unique, CSRF token, created/last seen/absolute expiry, revocation reason CHECK,
  IP, user agent; FK RESTRICT; partial index on active sessions). Down drops only these tables.
- `internal/modules/auth`: `password.go` (argon2id PHC, bounded verify params, 4 concurrent hashes, policy 12–128),
  `token.go`, `throttle.go` (per username+IP and per IP), `store.go` + `pgstore.go` (pgx, parameterised; admin
  ops revoke sessions in the same transaction), `service.go` (login with dummy-hash timing, disabled check after
  password, new token per sign-in + superseded revoke, idle/absolute expiry, status re-checked every request,
  throttled touch), `http.go` (handlers, `RequireSession` deny-by-default middleware with CSRF, cookie handling).
  `authtest` = in-memory store for tests only.
- Routes: `POST /api/v1/auth/login`, `GET /api/v1/auth/session`, `POST /api/v1/auth/logout`. Router requires the
  auth handler; public list = health, ready, login, session. CORS allows `X-CSRF-Token`.
- Config: `SESSION_IDLE_TIMEOUT` 2h, `SESSION_ABSOLUTE_TIMEOUT` 12h, `SESSION_COOKIE_SECURE` (default false only in
  development/test; refused false in staging/production), `LOGIN_MAX_FAILURES` 5, `LOGIN_MAX_FAILURES_PER_IP` 20,
  `LOGIN_FAILURE_WINDOW` 15m — documented as proposals pending owner sign-off.
- `cmd/user` CLI (`make user ARGS=...`): create/list/disable/enable/set-password/revoke-sessions; hidden prompt,
  no password flags, schema check first. Dependency: `golang.org/x/term v0.46.0` (2026-09-08), which bumped
  indirect `golang.org/x/sys` to v0.48.0 (2026-08-31); `golang.org/x/crypto` became a direct dependency (argon2).
- `internal/database/dbtest`: shared throwaway-database helper (create → migrate → drop); B-05 round-trip test
  moved onto it.
- Tests: `password_test`, `throttle_test`, `http_test` (cookie flags, `__Host-`, generic failures, disabled, disable
  ends session, validation/415, throttling 429 + Retry-After, session endpoint, deny by default incl. CSRF, logout
  revokes, idle + absolute expiry, re-sign-in supersedes, store outage → 503 not sign-out), `pgstore_integration_test`
  (constraints, token hash only, last login, touch, logout, disable/password change/unknown user, history kept,
  8 concurrent sign-ins unique, double logout revokes once), server test that every `/api/v1` route except the four
  public ones answers 401, CLI parse/password tests, config tests.
- Dev database migrated to version 2 with `make db-migrate-up` (additive: two new tables).
- Commands: `make test-integration` ✅ (all packages), `make lint-backend` ✅ (gofmt, vet, tidy, swagger-check),
  `make build-backend` ✅, `make swagger` ✅ (auth endpoints + schemas in OpenAPI).
- Docs: `API_CONVENTIONS.md` (Authentication section), ADR-008, OQ-6 resolved, `DEVELOPMENT_SETUP.md` (Users and
  sign-in), `.env.example`.

#### Stage 3 — API integration
Result: COMPLETE. The UI calls the real endpoints through `apiRequest` + TanStack Query; no mock exists in
the production path. Contract checked with curl through the Vite proxy against a fresh API (:8091) on a
throwaway database `maa_b06_check`: unauthenticated mutation 401, session 401, wrong password and unknown user
identical 401, disabled + correct password 403, form post 415, foreign Origin 403, login 200 with
`Set-Cookie: maa_session=…; Path=/; HttpOnly; SameSite=Strict`, session 200, logout without CSRF 403, logout
200, replayed revoked cookie 401 `SESSION_EXPIRED`, 6th failed attempt 429 `Retry-After: 900`.
Test users were created with `cmd/user` (random throwaway passwords kept in the scratchpad, deleted after);
`create` with a short password and a duplicate username were refused.

#### Stage 4 — Browser testing
Result: PASS. Claude in Chrome → Vite :5174 (`API_PROXY_TARGET=http://localhost:8091`) → API :8091 →
`maa_b06_check`. The user's own servers (:5173/:8081) were not touched.
- `/inventory/vehicles` without a session → `/login`; no app shell or dashboard requests.
- Wrong password → alert, username kept, password cleared and focused → `docs/evidence/B-06/01-invalid-credentials.jpg`.
- Disabled user with correct password → "Account disabled" → `02-account-disabled.jpg`.
- Correct sign-in → back to `/inventory/vehicles`, header "Asha Test"; full reload keeps the session; the session
  cookie is not readable by JavaScript; web storage only holds the theme key → `03-signed-in-account-menu.jpg`.
- `cmd/user revoke-sessions` while signed in → next action (Sign out) got 401 → "Your session has ended. Sign in
  again" → `04-revoked-session-ended.jpg`.
- Locked username → "Too many sign-in attempts … about 14 minutes" → `05-rate-limited.jpg`.
- Normal sign-out → sign-in page without the ended notice; browser Back stays on sign-in.
- 390 px (same-origin iframe; the window could not be resized): no horizontal overflow, 40 px submit button →
  `06-login-mobile-390.png`.
- Console: no errors. Network: an extra aborted `/auth/session` per load from React StrictMode (dev only; the
  extension labels it with a status, the API log shows no such response).
- API log: sign-in/out events logged; no passwords, hashes or tokens.
- Fixed during testing: the alert repeated the same sentence as title and message for wrong-password and
  rate-limit errors (now: actionable message / wait time only); a session ended on the server was only noticed on
  the next protected call, so the session is now re-checked every 5 minutes as well as on focus.
- Cleanup: tab closed, :5174/:8091 stopped, `maa_b06_check` dropped, throwaway passwords deleted.

#### Final checks
`make lint-backend` ✅, `make test-integration` ✅, `make build-backend` ✅, `make typecheck-frontend` ✅,
`make test-frontend` ✅ (73 tests), `make build-frontend` ✅, ESLint ✅. Prettier: B-06 files clean; the same 9
pre-existing uncommitted UI files still fail (not B-06).

#### Files changed (B-06)
Backend: `migrations/000002_create_users_and_sessions.{up,down}.sql`, `internal/modules/auth/*` (+ `authtest/`),
`internal/database/dbtest/dbtest.go`, `internal/database/migrations_integration_test.go`, `internal/httpx/response.go`,
`internal/config/config{,_test}.go`, `internal/middleware/cors.go`, `internal/server/server{,_test}.go`,
`cmd/api/main.go`, `cmd/user/{main,main_test}.go`, `go.mod`, `go.sum`, `Makefile` (`user` target, `bin/user`),
`.env.example`, `docs/openapi/*`.
Frontend: `ui/src/features/auth/*`, `ui/src/lib/api/{csrf.ts,client.ts,client.test.ts}`, `ui/src/app/{queryClient,routes}.tsx|ts`,
`ui/src/components/layout/UserMenu.tsx`, `ui/src/test/render.tsx`, `AppShell.test.tsx`, `SystemStatus.test.tsx`.
Docs: `API_CONVENTIONS.md`, `ARCHITECTURE_DECISIONS.md` (ADR-008, OQ-6), `DEVELOPMENT_SETUP.md`, this log,
`docs/evidence/B-06/*`.

#### Known issues / pending approvals
- Session lifetimes (2h idle / 12h absolute), password policy (12–128) and throttle limits are proposals; the owner
  can change them via environment variables.
- Login throttling is per API instance (in memory); multi-instance deployment needs a shared store (OQ-5).
- Audit events for sign-in/out are only in the structured log until B-08.
- No self-service password reset, MFA or user-management UI (not in B-06 scope). Accounts are created with `cmd/user`.
- Signed-in users have no permissions yet; every future feature route must add its permission check (B-07).
- The dev database was migrated to version 2. The user's running API on :8081 is an old build and does not know the
  auth routes; restart `make dev`, then create a first account with `make user ARGS="create ..."` to sign in.
- JS bundle 599 → 665 kB (B-06 auth UI plus the pre-existing uncommitted dashboard edits); lazy routes still planned.
- Pre-existing uncommitted UI edits (not B-06) still fail Prettier and include a hardcoded "authorised" brand roster.
- B-05 and B-06 are both uncommitted on `dev`.

#### Final status
COMPLETE — B-07 not started.


### REVIEW-B01-B06 — Verification of B-01…B-06 before B-07 (user-requested; not a registry task)

- **Status:** COMPLETE · **Date:** 2026-10-12 · B-07 not started (waiting for explicit approval).
- **Environment:** isolated stack — throwaway PostgreSQL container `maa-review-pg` (:5447), fresh migrate +
  seed, two throwaway accounts via `cmd/user`, API :8091, Vite :5174. The user's :5173/:8081 and dev
  database were not modified (still 0 users, 3 brands). All removed afterwards; passwords deleted.
- **Browser tool:** the Claude in Chrome window reported `visibilityState: hidden`, so real input did not
  reach the page. Switched to Playwright (workflow-preferred) with the installed Chrome: added
  `@playwright/test@1.63.0` (2026-09-04) as a dev dependency, `ui/playwright.config.ts`, `ui/tsconfig.e2e.json`,
  `ui/e2e/{support,auth.e2e,shell.e2e}.ts`, `npm run test:e2e`; docs in DEVELOPMENT_SETUP "End-to-end browser tests".

#### Bugs found (reproduced first) and fixed
| # | Task | Bug | Fix |
| --- | --- | --- | --- |
| 1 | B-06 | A failed background session re-check (API briefly unreachable) replaced the whole app with "Could not check your session" for a signed-in user (E2E reproduced). | `RequireAuth` keeps a confirmed session usable; error page only when the session was never confirmed. Unit + E2E test. |
| 2 | B-06 | Sign-in page spun "Checking your session…" forever when the browser was offline (paused query). | Shows the form with an offline notice. Unit test. |
| 3 | pre-existing UI edits | Dashboard "Multi-Brand Showroom Roster" invented brand specs/taglines ("Modular Lithium Packs", "Dual Battery Options", …), called the brands "authorised" and hardcoded them (E2E reproduced). Violates never-invent-specs and brands-from-API rules. | Card and its data removed (brands arrive via C-01/C-02 API). |
| 4 | pre-existing UI edits | Sidebar footer hardcoded "3 Brands · Lectrix · Ceeon · Ujagar". | Removed. |
| 5 | pre-existing UI edits | Dashboard quick-launch cards said "Open module" for unbuilt modules (B-03 requires "Planned" everywhere) and claimed "multi-tier approval rules" (thresholds pending). | "Planned" badge + "View plan"; wording "approval rules pending owner sign-off". E2E test. |
| 6 | pre-existing UI edits | Prettier failed on 9 files → `make lint` / `make ci` / GitHub CI frontend job would fail. | Formatting only (`prettier --write`), no behaviour change. |
| 7 | B-01 | Makefile `-include .env` overrode variables set in the shell: `DATABASE_URL=… make db-migrate-down` would silently target the `.env` database. Found when the review's own migrate/seed ran against the dev DB (both no-ops: already v2, seed idempotent — verified unchanged). | Environment and command-line values now win over `.env` (verified, incl. `$` in values). |
| 8 | B-06 docs | README quick start still said "open :5173 and see the dashboard"; no step to create the first account; `make setup` hint likewise. | README + `make setup` hint include `make user ARGS="create …"`. |
| 9 | B-05 | During a database outage the status card also said "schema … migrations pending", because `/ready` reports `migrations: down` when the DB is unreachable. | Explanation only when the database is up. Unit test. |

Not bugs (test artefacts, recorded for clarity): Back after sign-out landed on `about:blank` in the first E2E draft
(no in-app history entry; real browser stays on sign-in, verified in B-06); disabled-account E2E hit the login
throttle on the third run within 15 min (throttle working; docs say restart the API between runs); Prettier
scanned Playwright output (added `test-results`, `playwright-report` to `ui/.prettierignore`).

#### Verified flows (E2E, desktop 1366×900 and Pixel 7) — 34 passed, 2 skipped (viewport-specific)
Deep link → sign-in → back to the same URL (query + hash, username trimmed/case-insensitive); empty submit
validated without a request; wrong password generic + password cleared/focused; disabled account; reload keeps
session; cookie HttpOnly + SameSite=Strict + browser-session, not readable by JS, no auth data in web storage;
/login when signed in → dashboard; sign-out revokes on server (API 401 afterwards) and Back stays on sign-in;
forged/revoked cookie → "Your session has ended" → sign-in returns to the page; API mutation without session 401,
without CSRF 403 and session unaffected; API outage keeps the user signed in with "API offline", recovers;
throttling message with wait time; dashboard live API/database/schema status; no invented catalogue data and
planned shortcuts; planned module page + 404; no horizontal overflow on three pages; theme persists; desktop
sidebar navigation, focus on route change, collapse/expand; mobile drawer navigation + account menu.
Plus a real database outage (container stopped): signed-in user stays in with "Database unavailable", status card
shows both checks down without the misleading schema text (after fix 9); fresh load retries then offers "Try
again", which recovers once the database is back. No page errors.

#### Automated checks (final)
`make lint` ✅ (incl. Prettier, now clean), `make typecheck-frontend` ✅, `make test-integration` ✅,
`make test-frontend` ✅ (76 tests), `make build` ✅, `npm run test:e2e` ✅ 34/34.

#### Evidence
`docs/evidence/B-06-review/07-db-down-signed-in.png`, `08-db-down-reload-retrying.png`; E2E specs are the
repeatable record (`ui/e2e/`).

#### Remaining known issues (not fixed here)
- JS bundle 662 kB (> 500 kB warning); route-level code splitting still planned.
- Login throttle is per API instance; production behind a proxy must set `TRUSTED_PROXIES` or all clients share one IP.
- E2E needs a running stack and accounts, so it is not part of `make ci` yet.
- B-07 ↔ B-08 dependency cycle in `TASK_REGISTRY.csv` needs a decision before starting B-07.
- B-05, B-06 and this review are uncommitted on `dev`.

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
