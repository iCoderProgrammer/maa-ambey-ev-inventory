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
| B-07 | RBAC and location scope | P0 | B-06, B-08 | COMPLETE |
| B-08 | Append-only audit event framework | P0 | B-04, B-05, B-06 (registry fixed 2026-10-12; was B-07 — cycle) | COMPLETE |
| B-09 | Private attachment service | P0 | B-04, B-05, B-07 | COMPLETE |
| C-01 | Brand/model/variant database schema | P0 | B-05 | COMPLETE |
| C-02 | Catalogue API and validation | P0 | C-01, B-07, B-08 | COMPLETE |
| C-03 | Catalogue and brand management UI | P0 | C-02, B-03 | COMPLETE |
| C-04 | Effective-dated price history | P0 | C-01, C-02 | COMPLETE |
| C-05 | Seed verified multi-brand catalogue | P0 | A-01, C-01, C-02 | NOT STARTED |
| C-06 | Brand-specific rules and isolation tests | P0 | C-02, C-04, B-07 | COMPLETE |
| D-01 | Supplier master | P0 | B-05, B-07, B-08 | COMPLETE |
| D-02 | Purchase order lifecycle | P0 | C-02, D-01, C-04, B-07 | COMPLETE |
| D-03 | Goods receipt draft workflow | P0 | D-02, B-09, C-01 | COMPLETE |
| D-04 | Vehicle unit registry | P0 | C-01, B-05 | COMPLETE (API + typed client/hooks; screen is D-08) |
| D-05 | Duplicate vehicle identifier controls | P0 | D-04 | IN PROGRESS — backend + frontend phases complete; live posting conflict pending D-07 |
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

| Item | Title | Status |
| --- | --- | --- |
| A-01 | Launch brand roster and activation evidence | INCOMPLETE — roster documented (`docs/client-inputs/launch-brand-roster.md`); per-brand client evidence pending |

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

### B-08 — Append-only audit event framework

- **Status:** COMPLETE
- **Started:** 2026-10-12 · **Finished:** 2026-10-12
- **Prompt file:** prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/01_Foundation_Security/B_08_append-only-audit-event-framework.md
- **Order decision:** the registry has a cycle (B-07 needs B-08, B-08 needs B-07). The user chose on 2026-10-12:
  **B-08 first, then B-07**, so role/permission changes in B-07 are audited from day one (SEC-004).
  B-08 therefore does not use permissions; it adds no read endpoint (reading needs a permission → B-07 or later).

#### Dependencies
| Dependency | Verified status | Evidence |
| --- | --- | --- |
| B-04 | COMPLETE | B-04 entry; review 2026-10-12 |
| B-05 | COMPLETE | B-05 entry; migrations/round-trip tests green |
| B-07 | NOT STARTED — cycle resolved by user decision above | — |

#### Acceptance criteria
- [x] AC1 Audit event table + service: actor, action, entity type/id, reference, timestamp, request ID, safe summary
- [x] AC2 Append-only: rows cannot be updated, deleted or truncated through normal application paths (DB-enforced), proven by test
- [x] AC3 Sensitive values (passwords, hashes, tokens, CSRF, secrets) cannot enter audit payloads, proven by test
- [x] AC4 Reusable helper for sensitive mutations, written in the same transaction as the change; documented
- [x] AC5 Thin integration example only: sign-in/sign-out and `cmd/user` account administration
- [x] AC6 UI: NOT APPLICABLE (no audit screen in scope; Audit Log stays a planned module)

#### Stage 1 — UI
Result: NOT APPLICABLE — Prompt B: no audit screen unless already in scope; none exists (Audit Log stays a
planned module). No permission contract exists yet for a read-only placeholder (B-07).

#### Stage 2 — Backend
Result: COMPLETE.
- Migration `000003_create_audit_events` (up/down): identity PK, actor type/user (FK RESTRICT)/label, action
  (dotted format CHECK), outcome, entity type/id, reference, request ID, JSONB summary (object, ≤ 8 KB),
  four indexes; `BEFORE UPDATE OR DELETE` row trigger and `BEFORE TRUNCATE` statement trigger raise `42501`.
- `internal/audit`: `Event`, `Actor` (`UserActor`, `CLIActor`), `Summary`, `Change`, `Record(ctx, execer, ev)`
  validating actor/action/outcome/entity/lengths and failing closed on secret-looking keys (password, token,
  hash, secret, csrf, cookie, otp, credential, api key …), password-hash-looking values, oversized/unsupported data.
  Package doc shows the transactional usage.
- Thin integration: `auth.Service` — `auth.signed_in` written in the session-creation transaction,
  `auth.signed_out` in the revoke transaction (idempotent: second sign-out writes nothing),
  `auth.sign_in_failed` / `auth.sign_in_refused` for existing accounts only; `cmd/user` — `user.created`,
  `user.enabled`, `user.disabled` (status from/to), `user.password_changed`, `user.sessions_revoked`, each in the
  same transaction as the change, actor `cmd/user:<OS user>`. Request IDs passed from handlers.
- Tests: `audit_test` (valid event, 7 invalid shapes, 16 secret/unsafe summaries, size limit),
  `audit_integration_test` (UPDATE/DELETE/TRUNCATE refused with 42501 and row unchanged; DB CHECKs; rollback
  of the caller's transaction removes the event), auth `http_test` (actions in order, unknown username not
  audited, no password/cookie/CSRF/hash in events), `pgstore_integration_test` (full sequence of 7 audited
  actions, no secrets or token in the table, request ID and actor fields, session not created when its event is
  invalid).
- Dev database migrated to version 3 (`make db-migrate-up`, additive).
- Live check on a throwaway database: `cmd/user create/disable` rows with actor `cmd/user:hp`; `DELETE` refused
  even for the database owner ("audit_events is append-only: DELETE is not allowed").
- Commands: `make test-integration` ✅, `make lint-backend` ✅, `make build-backend` ✅ (no route changes; OpenAPI
  unchanged).

#### Stage 3 — API integration
Result: NOT APPLICABLE — no new endpoint; existing auth endpoints unchanged in contract.

#### Stage 4 — Browser testing
Result: PASS (regression). Sign-in/sign-out now write audit rows in their transactions, so the full Playwright
suite ran against a fresh isolated stack (Postgres :5447, API :8091 new build, Vite :5174): 34 passed,
2 skipped (viewport-specific). The browser runs produced 37 audit rows (signed_in, signed_out, sign_in_failed,
sign_in_refused, user.created, user.disabled); a scan found no password, hash or test-password text; all 34
HTTP-originated rows carry a request ID.

#### Files changed (B-08)
`migrations/000003_create_audit_events.{up,down}.sql`, `internal/audit/{audit.go,audit_test.go,audit_integration_test.go}`,
`internal/modules/auth/{store.go,pgstore.go,service.go,http.go,http_test.go,pgstore_integration_test.go}`,
`internal/modules/auth/authtest/store.go`, `cmd/user/main.go`, `docs/API_CONVENTIONS.md` (Audit events),
`docs/ARCHITECTURE_DECISIONS.md` (ADR-009), `docs/DEVELOPMENT_SETUP.md`.

#### Known issues / pending approvals
- No audit read API/UI yet; needs a permission (B-07) and the Audit Log screen has no explicit registry task.
- IP-address retention policy for audit is open; IPs are not stored in audit events.
- Retention/archiving of audit rows is not defined (H-06 / operations).
- The user's running API (:8081) is a pre-B-08 build: `/ready` says the schema is newer — restart `make dev`.

#### Final status
COMPLETE — continuing with B-07 as instructed (B-08 first, then B-07).


### B-07 — RBAC and location scope

- **Status:** COMPLETE
- **Started:** 2026-10-12 · **Finished:** 2026-10-12
- **Prompt file:** prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/01_Foundation_Security/B_07_rbac-and-location-scope.md

#### Dependencies
| Dependency | Verified status | Evidence |
| --- | --- | --- |
| B-06 | COMPLETE | B-06 entry + review (E2E 34/34) |
| B-08 | COMPLETE | B-08 entry (done first by user decision) |

#### Pending client input (not guessed)
RB-5 / A-05: every cell of the permission matrix is "Pending"; brand scope per user is pending; the initial
showroom location's code/name/address were not supplied. B-07 therefore ships the mechanism with **no grants**
and **no locations**; both are entered later as data (with approval evidence) through an audited CLI.

#### Acceptance criteria
- [x] AC1 Direct API requests without an explicitly approved permission are denied (403), deny by default
- [x] AC2 Confirmed role identities represented: Owner, Inventory Manager, Sales Executive, Accountant, Viewer
- [x] AC3 Granular grants/approval thresholds clearly marked pending owner sign-off (data + API + UI); no grants
      assumed; sensitive permission keys (cost prices, refunds, adjustments, user roles, invoice settings, exports,
      tax, discounts, audit) defined but not granted
- [x] AC4 Unapproved (proposed) grants never count in staging/production
- [x] AC5 Location scope enforced by policy helpers; cross-location access denied; tested
- [x] AC6 Negative tests: direct API access, privilege escalation, cross-location access
- [x] AC7 Role/permission/scope changes audited (SEC-004)
- [x] AC8 Frontend: navigation/actions from the permissions contract, access-denied state, consistent 401/403,
      no sensitive fields leaked

#### Stage 1 — UI
Result: UI COMPLETE — MOCK DATA ONLY (fetch mocked in tests; no runtime mock adapter, the API follows in Stage 2).
- Contract: `GET /me` → `{user, roles[{key,name}], permissions[string], locationScope{all, locations[{id,code,name}]},
  proposedGrantsActive}`; `GET /access/roles` (needs `access.roles.view`) → `{matrixStatus, approvedGrantCount,
  roles[{key,name,description,grants[{permission,status,approvedBy,approvalReference,approvedAt}]}],
  permissions[{key,description,sensitive}]}`; 403 `FORBIDDEN`.
- `ui/src/features/access/`: `api/access.ts`, `hooks/useAccess.ts` (`useMe`, `usePermission` — unknown = not
  granted), `components/RequirePermission.tsx`, `RolesPage.tsx` (read-only: pending-sign-off banner, dev-mode
  banner for proposed grants, roles with "None — pending sign-off" / Approved / "Proposed — not approved",
  permission catalogue with sensitivity).
- `StateViews`: `AccessDeniedState`; `ErrorState` shows access denied for 403 `FORBIDDEN` only (not
  `CSRF_FAILED`), without retry. Navigation items can declare `permission`; Sidebar and planned-page related
  links hide gated items unless granted. Users & Roles is now implemented (route `/admin/users`).
- Account menu: roles, location scope, "No permissions are granted yet … pending the owner's sign-off".
- Tests: `access.test.tsx` (8). Vitest 84/84 ✅, tsc ✅, ESLint ✅.

#### Stage 2 — Backend
Result: COMPLETE.
- Migration `000004_create_access_control` (up/down): `permissions` (15 keys incl. cost price view/edit, refunds
  create/approve, stock adjustment create/approve, roles view/manage, invoice settings, tax settings, discount
  approval, report/customer export, audit view, `locations.all`), `roles`, `role_permissions` (PROPOSED/APPROVED;
  APPROVED requires approver + reference + date — CHECK), `user_roles`, `locations` (identity only),
  `user_location_scopes`; FKs RESTRICT. Seed `seeds/0002_roles.sql`: the five confirmed roles, neutral
  descriptions, **no grants**. No location seeded (showroom details pending).
- `internal/modules/access`: `policy.go` (permission constants, `Access.Has`, `Scope.Allows/Check/Restrict`,
  `ErrOutOfScope`), `store.go` (`LoadAccess` with/without proposed grants, `RoleMatrix`, audited `Grant` (no silent
  downgrade of an approved grant), `Revoke`, `AssignRole`/`UnassignRole`, `CreateLocation`, `Add/RemoveLocationScope`),
  `http.go` (`Current` loads access once per request, `Require(permission)` → 403 FORBIDDEN without naming the
  permission and panics on unknown keys, `GET /me`, `GET /access/roles` behind `access.roles.view`).
- Config `RBAC_ALLOW_PROPOSED_GRANTS` (default false; refused in staging/production; warning logged when on).
- `cmd/access` CLI (`make access ARGS=...`): roles, permissions, show, grant (PROPOSED, or APPROVED only with both
  `--approved-by` and `--approval-ref`), revoke, assign/unassign, location-create, scope-add/remove; schema check;
  actor `cmd/access:<OS user>` in every audit event.
- Tests: `policy_test` (only granted keys, case/space variants denied, scope allows/check/restrict incl.
  cross-location refusal), `http_test` (contract, empty arrays, 403 on direct GET/POST without permission and no
  permission name in the body, 401 before 403, view ≠ edit, proposed grants only with the switch, client headers
  cannot add permissions, no POST/PUT/PATCH/DELETE on `/access/roles`, cross-location 403 / own 200 /
  `locations.all` 200, single load per request, 503 on lookup failure, unknown key panic),
  `pgstore_integration_test` (code ↔ DB catalogue, fresh DB has 0 grants and the 5 roles pending, effective access
  through assignment/proposed/approved/unassign, downgrade refused, unknown user/role/permission, location scope
  incl. inactive location and `locations.all`, 9 audited actions in order with evidence, DB refuses APPROVED
  without evidence), seed test (5 roles, no grants added), server test (Access dependency required; every
  non-public route 401 without session), config test, CLI parse test.
- Dev database migrated to version 4 and seeded (roles), additive.
- Commands: `make lint` ✅, `make test-integration` ✅, `make build` ✅, `make swagger` ✅ (`/me`, `/access/roles`).

#### Stage 3 — API integration
Result: COMPLETE. UI uses `GET /me` and `GET /access/roles` through `apiRequest` + TanStack Query; no mocks in the
production path. Real checks on an isolated stack (Postgres :5447, API :8091, Vite :5174), API first started like
staging/production (proposed grants off): anonymous `/me` 401; `roles.review` (Viewer, PROPOSED
`access.roles.view`, scope SR-TEST) → `/me` permissions `[]`, `/access/roles` 403; `POST /access/roles` 405;
`owner.review` (no roles) → `/me` roles `[]`, `/access/roles` 403; denials logged.

#### Stage 4 — Browser testing
Result: PASS. Playwright (installed Chrome), desktop + Pixel 7, API with `RBAC_ALLOW_PROPOSED_GRANTS=true`:
38 passed, 2 skipped (viewport-specific). New: user without grants sees no Users & Roles link, direct
`/admin/users` → Access denied, API 403, account menu "No role assigned" + "No permissions are granted yet";
user with the (proposed) permission → link, page with pending-sign-off banner, development-mode banner, the five
roles, sensitive permission keys, no horizontal overflow, statuses inside the viewport. All earlier B-03…B-06
flows passed again.
- Fixed during testing: on phone width the grant statuses and sensitivity were pushed off-screen inside the
  table scroll (UI guidelines: never hide statuses on narrow screens) → description hidden and sensitivity moved
  under the key on small screens; E2E now asserts statuses fit the viewport.
- Evidence: `docs/evidence/B-07/01-access-denied-no-grants.png`, `02-account-menu-no-roles.png`,
  `03-roles-page-pending-signoff.png`, `04-account-menu-viewer-scope.png`, `05-roles-page-mobile.png`.
- Audit rows from the run: `access.role_assigned`, `access.grant_proposed`, `location.created`,
  `access.location_scope_added` by `cmd/access:hp`. Stack removed afterwards; throwaway passwords deleted.

#### Final checks
`make lint` ✅, `make typecheck-frontend` ✅, `make test-integration` ✅, `make test-frontend` ✅ (84),
`make build` ✅, `npm run test:e2e` ✅ 38/38.

#### Files changed (B-07)
Backend: `migrations/000004_create_access_control.{up,down}.sql`, `seeds/0002_roles.sql`,
`internal/modules/access/{policy,store,http}.go` + tests, `internal/config/config{,_test}.go`,
`internal/server/server{,_test}.go`, `internal/database/seed_integration_test.go`, `cmd/api/main.go`,
`cmd/access/{main,main_test}.go`, `Makefile` (`access` target, `bin/access`), `.env.example`, `docs/openapi/*`.
Frontend: `ui/src/features/access/*`, `ui/src/components/ui/StateViews.tsx`, `ui/src/app/{navigation,routes}.ts(x)`,
`ui/src/app/PlannedModulePage.tsx`, `ui/src/components/layout/{Sidebar,UserMenu}.tsx`, `ui/src/test/render.tsx`,
`ui/e2e/{support,access.e2e}.ts`.
Docs: `API_CONVENTIONS.md` (Authorisation), `ARCHITECTURE_DECISIONS.md` (ADR-007 as built), `DEVELOPMENT_SETUP.md`
(Roles, permissions and locations; E2E roles user), `REQUIREMENTS_BASELINE.md`.

#### Known issues / pending approvals
- **RB-5 / A-05:** the whole permission matrix and approval thresholds await the owner; until then every
  protected action is denied for everyone (by design). Enter decisions with `cmd/access grant … --approved-by
  --approval-ref`.
- Brand scope per user (RB-5) is not modelled yet. Initial showroom location details (code/name/address) not
  supplied, so no location exists.
- No user-management or grant-editing screens (CLI only); no registry task names them, nor the Audit Log screen.
- Discount/approval thresholds (amount limits) are not modelled; they are business values pending sign-off.
- JS bundle 676 kB (warning); lazy routes still planned.
- The user's running API (:8081) is an old build and the dev DB is now version 4: restart `make dev`.

#### Final status
COMPLETE — B-09 not started.


### VERIFY-B07 — Final manual verification of B-07 (user-requested; not a registry task)

- **Status:** COMPLETE · **Date:** 2026-10-12 · No new task started.
- **Note:** B-08 was already COMPLETE (done before B-07 by the user's earlier choice); the next registry task is B-09.

#### Evidence check
`docs/evidence/B-07/01…05` exist and match the B-07 entry (01–04 at 1366 px, 05 at 390 px re-captured after the
mobile status fix). Added: `06-manual-tester-roles-page.png`, `07-manual-tester-after-revoke.png`.

#### Manual test on the user's local stack (Vite :5173 → API :8081 → dev DB), account `manual.tester`
Setup as documented and already applied by the user: `.env` `APP_ENV=development`, `RBAC_ALLOW_PROPOSED_GRANTS=true`
(git-ignored), Viewer role, PROPOSED `access.roles.view`. Driven with Playwright on the installed Chrome (the Claude
in Chrome window was hidden earlier); password read from the git-ignored `.dev-test-accounts`, never printed.
28/28 checks passed:
- Users & Roles link visible; page shows pending-sign-off and development-mode banners; Viewer row
  `access.roles.view` "Proposed — not approved"; other four roles "None — pending sign-off"; 15 catalogue entries;
  no action buttons (read-only); account menu Viewer / "No location assigned" / development note.
- Direct API: `/me` permissions `["access.roles.view"]`, `proposedGrantsActive: true`; `/access/roles` 200;
  POST/PUT/PATCH/DELETE `/access/roles` 405; logout without CSRF 403 `CSRF_FAILED`; `X-Permissions`/`X-Role`
  headers ignored.
- `cmd/access revoke` while signed in → API 403 at once; after reload "Access denied" and the link disappears;
  `unassign` → `/me` empty; grant and role restored afterwards (audited in the dev DB).
- Deny by default: anonymous `/me`, `/access/roles` 401, anonymous logout 401, `/admin/users` → sign-in page;
  `disabled.tester` refused; no console/page errors.

#### Flag kept to local development
- Separate API instance (:8093, the user's :8081 untouched): `APP_ENV=staging`/`production` with the flag on →
  refuses to start; development with the flag off → `manual.tester` gets `permissions: []` and `/access/roles` 403.
- **Fixed:** config also accepted the flag with `APP_ENV=test`. Now only `APP_ENV=development` accepts it
  (`internal/config/config.go`, tests for test/staging/production refusal and development acceptance); comments,
  `.env.example`, DEVELOPMENT_SETUP, API_CONVENTIONS, ADR-007 updated. CI does not set the flag.
- Added a UI test for APPROVED grants / partial approval (coverage gap: no real approved grant exists yet).

#### Registry dependency cycle
- `prompt-pack/TASK_REGISTRY.csv` and its identical copy: **B-08 dependencies `B-04, B-05, B-07` → `B-04, B-05, B-06`**
  (B-08 needs `users`, not RBAC). B-07 keeps `B-06, B-08`. Copies still byte-identical; CRLF kept.
- Prompt-pack Markdown still lists B-07 for B-08 (approved text, not edited); CSV is the source of truth (ADR-009).
- Whole-registry cycle check: B-07/B-08 cycle gone; **one more cycle found: D-08 ↔ E-02** — recorded as OQ-8, not
  changed (outside B-07). C-05 → A-01 refers to a discovery item, expected.

#### Checks
`make lint` ✅, `make typecheck-frontend` ✅, `make test-integration` ✅, `make test-frontend` ✅ (85), `make build` ✅.

#### Remaining owner approvals
RB-5 permission matrix + approval thresholds (all grants), brand scope per user, initial showroom location
details, plus OQ-8 (D-08 ↔ E-02 order).

### VERIFY-B07-FINAL — Final technical verification of B-07 (user-requested; not a registry task)

- **Status:** COMPLETE · **Date:** 2026-10-12 · No other task started. No grant of any kind was added.

#### Dependency status (reported separately from the technical results)
- Registry (after the fix recorded in VERIFY-B07): B-07 → `B-06, B-08`; B-08 → `B-04, B-05, B-06`; no cycle.
- **B-06: COMPLETE. B-08: COMPLETE** (entry "B-08 — Append-only audit event framework": done before B-07 on the
  user's choice; migration 000003 applied — dev DB at version 4; `internal/audit` tests re-run green today in
  `make test-integration`). B-08 is backend-only, so its evidence is tests + audit rows, not screenshots.
- Therefore B-07 has no incomplete dependency. B-07 *business* blockers (owner approvals) remain — see below.

#### Evidence review
`docs/evidence/B-07/01…07` exist and match earlier entries (01–04 desktop 1366 px, 05 mobile 390 px after the
mobile fix, 06–07 from VERIFY-B07). Added today: `08-final-dev-roles-page.png`, `09-final-dev-roles-page-mobile.png`,
`10-final-no-grants-location-scope.png`, `11-final-session-revoked.png`.

#### A. User's local stack (Vite :5173 → API :8081 → dev DB), `manual.tester`, read-only — 23/23 PASS
Documented local setup already in place (`APP_ENV=development`, `RBAC_ALLOW_PROPOSED_GRANTS=true` in git-ignored
`.env`; Viewer + the user's own PROPOSED `access.roles.view`). Playwright + installed Chrome; no grant/role/scope changed.
- Roles page: pending-sign-off + development-mode banners, Viewer grant "Proposed — not approved", 0 approved,
  no buttons in main (read-only). `/me`: Viewer, only `access.roles.view`, no locations, `proposedGrantsActive: true`.
- Privilege escalation: POST/PUT/PATCH/DELETE `/access/roles` and POST/PUT `/me` → 405; spoofed
  `X-Permissions`/`X-Role`/`X-User-Id` ignored; unknown `/api/v1/access/grants` → 404; mutation without CSRF → 403.
- Frontend 401: forged session cookie (test context only) on Users & Roles → sign-in with "Your session has ended",
  re-sign-in returns to `/admin/users`. Mobile (Pixel 7): statuses inside the viewport, no horizontal overflow.
- Anonymous `/me`, `/access/roles` → 401; `/admin/users` → sign-in; `disabled.tester` refused; no console errors.

#### B. Isolated stack (throwaway Postgres :5447, API :8091 with proposed grants **off**, Vite :5174), 0 grants — 10/10 PASS
- User without roles/permissions: no Users & Roles link, `/admin/users` → Access denied (no table), API 403
  `FORBIDDEN` without naming the permission; `/me` empty, `proposedGrantsActive: false`.
- Location scope: user scoped to SR-A only (SR-B exists) → `/me` and account menu show SR-A only; `scope-remove`
  → `/me` has no locations. (No production endpoint consumes location scope yet; cross-location *enforcement* is
  covered by automated tests: `Scope.Check/Restrict`, HTTP stand-in route 403/200, PG integration.)
- `cmd/user revoke-sessions` while on the page → the page's next `/auth/session` → 401 `SESSION_EXPIRED` → sign-in
  with "Your session has ended". Audit rows: `location.created` ×2, `access.location_scope_added/removed`,
  `user.sessions_revoked`.
- One script artefact (not a bug): an extra API call before the reload received `SESSION_EXPIRED` and correctly
  cleared the cookie, so the reload showed the plain sign-in page; the check now uses the page's own request.

#### Bug found and fixed
- **Stale navigation after a server 403:** when the server refused a permission the cached `/me` still listed
  (e.g. revoked meanwhile), the page showed Access denied but the sidebar kept offering the link for up to 60 s.
  Fix: `ui/src/app/queryClient.ts` re-reads `/me` on any 403 `FORBIDDEN`. Regression test in `access.test.tsx`
  (reproduced without granting anything: cached `/me` with the permission, server 403); verified to **fail without
  the fix** and pass with it.

#### Automated checks (actual results)
`make lint-backend` 0 · `make lint-frontend` 0 · `make typecheck-frontend` 0 · `make test-integration` 0 (incl.
access, audit, auth, server, config, cmd/access) · `make test-frontend` 0 (86 tests) · `make swagger-check` 0 ·
`make build-backend` 0 · `make build-frontend` 0 (bundle 676 kB, warning only) ·
`npm run test:e2e` (isolated stack): **36 passed, 4 skipped** — 2 viewport-specific by design, 2 = "user with the
permission" (desktop+mobile) skipped on purpose: no `E2E_ROLES` user, because creating one needs a grant. That
positive path was verified in A on the user's own setup.

#### Proposed grants kept local
Only `APP_ENV=development` accepts `RBAC_ALLOW_PROPOSED_GRANTS=true` (config refuses test/staging/production; tests).
CI does not set it; `.env.example` ships `false`; the user's `.env` is git-ignored. Dev DB after testing:
1 PROPOSED grant (the user's), 0 APPROVED; isolated stack removed.

#### Owner approvals still pending (B-07 business blockers)
RB-5 permission matrix and approval thresholds (no role holds any approved permission); brand scope per user;
initial showroom location (code/name/address); OQ-8 (registry cycle D-08 ↔ E-02, outside B-07).

### B-08-FOLLOWUP — Audit actor fix and append-only re-verification (user-requested "Proceed with B-08")

- **Status:** COMPLETE · **Date:** 2026-10-12 · B-08 scope only; B-09 not started. No grant, role, brand scope or
  location was added anywhere.
- **Prompt file:** prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/01_Foundation_Security/B_08_append-only-audit-event-framework.md
  (re-read; scope and acceptance criteria unchanged). Dependencies B-04, B-05, B-06 COMPLETE (entries above;
  registry fixed in VERIFY-B07).

#### Issue found and fixed
- **Wrong actor on failed/refused sign-ins:** `auth.sign_in_failed` / `auth.sign_in_refused` were recorded with
  actor `USER` = the targeted account, i.e. the audit trail said the account holder did something an
  unauthenticated requester did (visible in `docs/evidence/B-08/audit-trail-demo.txt`, rows 4 and 7).
  Fix: new actor type `ANONYMOUS` (`audit.AnonymousActor`, label "sign-in attempt", never typed input); the account
  stays the entity. `auth.signed_in` / `auth.signed_out` keep the user as actor.
- Migration `000005_audit_anonymous_actor` (up: widen `audit_events_actor_type_valid`, add
  `audit_events_anonymous_has_no_user`; down: restore the narrower check `NOT VALID`, because audit rows cannot be
  changed or deleted to make it valid). Additive; dev DB already at version 5.
- Existing dev-DB rows from before the fix (2 × `auth.sign_in_refused` with actor `USER`) stay as they are —
  append-only by design; not rewritten.
- API append-only guard: `TestNoAPIRouteCanModifyAuditEvents` (server) fails if any `/audit` route other than GET
  is registered, and checks PUT/PATCH/DELETE/POST `/api/v1/audit-events/1` are not served.
- gofmt fix in `internal/audit/audit_integration_test.go` (map alignment; `make lint-backend` failed before it).

#### Acceptance criteria (re-verified)
- [x] Actor, action, entity type/id, reference, timestamp, request ID, safe summary recorded (evidence below)
- [x] Append-only from normal application paths: DB triggers (UPDATE/DELETE/TRUNCATE → 42501) and no mutating API
  route — `TestAuditEventsAreAppendOnly`, `TestNoAPIRouteCanModifyAuditEvents`
- [x] Sensitive values not logged: `TestRecordRejectsSecretsInSummary`, `TestAuthEventsAreAuditedWithoutSecrets`,
  PG test + evidence secret scan (0 rows)
- [x] Reusable service documented (`internal/audit` package doc, API_CONVENTIONS "Audit events", ADR-009)

#### Evidence
`docs/evidence/B-08/audit-trail-demo-anonymous-actor.txt` — isolated stack (throwaway Postgres :5447, API :8091
this build; random throwaway password never printed): failed/refused sign-ins now `ANONYMOUS` with no user,
signed-in/out `USER`, CLI rows `cmd/user:hp`, every HTTP row has its request ID; API PUT/PATCH/DELETE/POST
`/api/v1/audit-events/1` → 404; SQL UPDATE/DELETE/TRUNCATE refused, row count unchanged (7); `ANONYMOUS` row with
a user refused by the CHECK. The older `audit-trail-demo.txt` (pre-fix) is kept as the before state.

#### Checks (actual results)
`make lint-backend` 0 · `make lint-frontend` 0 · `make typecheck-frontend` 0 · `make test-integration` 0 ·
`make test-frontend` 0 (86 tests) · `make swagger-check` 0 · `make build-backend` 0 · `make build-frontend` 0.
Named B-08 tests all PASS: audit (8), server `TestNoAPIRouteCanModifyAuditEvents`, auth
`TestAuthEventsAreAuditedWithoutSecrets`, `TestPGAuthAndAdminActionsAreAudited`,
`TestPGSessionIsNotCreatedWithoutItsAuditEvent`.

#### B-07 re-verification (separate)
- Go: `internal/modules/access`, `cmd/access`, `internal/config`, `internal/server` — ok, 39 PASS, 0 FAIL, 0 SKIP.
- Playwright on an isolated stack (Postgres :5447, API :8091 proposed grants **off**, Vite :5174; 0 role
  permissions, 0 role assignments): **36 passed, 4 skipped** — same as VERIFY-B07-FINAL (2 viewport-specific by
  design; 2 "user with the permission" skipped because that needs a grant, which is not created). Audit rows from
  the run: sign_in_failed ×2 and sign_in_refused ×2 `ANONYMOUS`, signed_in ×30 / signed_out ×2 `USER`, all with
  request IDs; 0 rows contain a test password or argon2. Isolated stack removed afterwards.

#### Files changed (this follow-up)
`migrations/000005_audit_anonymous_actor.{up,down}.sql`, `internal/audit/{audit.go,audit_test.go,audit_integration_test.go}`,
`internal/modules/auth/{service.go,http_test.go,pgstore_integration_test.go}`, `internal/server/server_test.go`,
`docs/API_CONVENTIONS.md`, `docs/ARCHITECTURE_DECISIONS.md` (ADR-009 actor types),
`docs/evidence/B-08/{audit-trail-demo.txt,audit-trail-demo-anonymous-actor.txt}`, this log.

#### Remaining blockers / open items (unchanged)
No audit read API/UI (needs an approved permission; no registry task for the Audit Log screen); IP-address and
audit retention policies open; B-07 owner approvals (RB-5 matrix + thresholds, brand scope per user, showroom
location details) and OQ-8 still pending.

### VERIFY-B07-POSITIVE — Permission-positive browser tests (user-requested pre-work before B-09; not a registry task)

- **Status:** COMPLETE (technical verification) · **Date:** 2026-10-12 · B-09 not started.
- B-08 follow-up committed as `09a1726` (pushed). `git diff --check` then flagged trailing spaces in the psql output
  of `docs/evidence/B-08/*.txt`; stripped (content unchanged). Registry correction (B-08 deps) was already
  committed in `b5267c4`; no registry or prompt Markdown change since.

#### Environment (isolated, temporary)
Throwaway Postgres container (:5447, migrated + seeded roles/brands), API :8091 built from this tree with
`APP_ENV=development RBAC_ALLOW_PROPOSED_GRANTS=true`, Vite :5174 proxying to it. Test-only fixtures in that
database: `e2e.active`, `e2e.disabled` (disabled), `e2e.roles` → role `viewer` + **PROPOSED**
`access.roles.view` (via `cmd/access`, audited). No APPROVED grant. Random passwords, never printed. Stack and
container removed after the run. Shared dev DB unchanged (still 1 PROPOSED grant, 0 APPROVED, 15 audit rows).

#### Result — `npx playwright test`: **38 passed, 2 skipped**
- Both previously skipped permission-positive tests now PASS: "a user with the permission sees the roles pending
  owner sign-off" on desktop (1366 px) and mobile (Pixel 7) — link/route, pending-sign-off banner, 5 roles, catalogue
  keys, statuses inside the viewport, no horizontal overflow.
- The no-grant test also passes in the same run (desktop + mobile): no gated link, Access denied, API 403.
- 2 skips are the viewport-specific tests by design (desktop-navigation test on mobile, mobile-drawer test on desktop).
- Audit rows from the run: `access.role_assigned`, `access.grant_proposed` (CLI); sign-in failures/refusals
  `ANONYMOUS`; 32 signed_in / 2 signed_out `USER`, all with request IDs; 0 rows with a test password or argon2.
- Evidence: `docs/evidence/B-07/12-e2e-permission-positive.txt`.

#### Technical verification vs owner approvals
- Technically verified: deny-by-default, permission-gated navigation/route/API, read-only roles page, proposed
  grants honoured only with the development flag. The browser positive path used a PROPOSED grant; the APPROVED path
  is covered by unit/UI tests only, because no real approved grant may exist without owner sign-off.
- Pending owner approvals (business, not technical): RB-5 permission matrix + approval thresholds, brand scope per
  user, initial showroom location details; OQ-8 (D-08 ↔ E-02 order).

### B-09 — Private attachment service

- **Status:** COMPLETE (technical; limits and production storage provider pending owner/hosting decisions)
- **Started:** 2026-10-13 · **Finished:** 2026-10-13
- **Prompt file:** prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/01_Foundation_Security/B_09_private-attachment-service.md
- **Order:** backend first, then frontend, then integration and browser test (user instruction and the prompt's
  execution note), instead of the workflow's default UI-first order.
- **Decisions made with the user (2026-10-13):** (1) add catalogue keys `attachments.upload` / `attachments.view`
  with **no grants**; (2) proposed limits JPEG/PNG/WebP/PDF, 10 MB (configurable, pending confirmation — OQ-9);
  (3) local private-directory driver now, S3-compatible driver deferred until the hosting decision (OQ-10), and a
  permission-gated Documents page that hosts the reusable components.

#### Dependencies
| Dependency | Verified status | Evidence |
| --- | --- | --- |
| B-04 | COMPLETE | B-04 entry; API envelopes/middleware reused |
| B-05 | COMPLETE | migrations framework; round-trip test incl. 000006 green |
| B-07 | COMPLETE (technical; owner approvals pending) | VERIFY-B07-POSITIVE; `access.Require`, `Scope` reused |
Git: working tree clean at start (`dev` = `origin/dev` = `2b27cca`).

#### Acceptance criteria
- [x] AC1 Unauthorized downloads fail: anonymous 401; no permission 403; other user / out-of-scope 404; other
  user's, tampered or expired link 403 (`DOWNLOAD_LINK_INVALID` / `_EXPIRED`); revoked permission stops an issued link
- [x] AC2 Invalid MIME and oversized files rejected server-side (415 / 413 / 422), nothing stored; client checks too
- [x] AC3 Metadata and file access permission-checked (route guard + object check on every read and download)
- [x] AC4 UI displays upload errors (client and server), with retry where retrying can help
- [x] AC5 No permanently public URL: links are short-lived, bound to user + session; bytes shown from memory
- [x] AC6 Safe generated object keys; no secrets/file names in audit; RBAC fail-closed (no grants added)

#### Stage 2 — Backend (first)
Result: COMPLETE.
- Migration `000006_create_attachments` (up/down): catalogue keys (no grants), `attachments` table with CHECKs
  (key format, driver, content-type allow-list, size, SHA-256 length, name, owner pair), FKs to users/locations
  (RESTRICT), indexes; triggers refuse DELETE/TRUNCATE and any change except linking an unlinked row once.
  Dev DB migrated 5 → 6 (`make db-migrate-up`, additive; 0 attachment grants).
- `internal/modules/attachments`: `ObjectStore` + `LocalStore` (0700/0600, key pattern blocks traversal,
  temp file + link, no overwrite), `Signer` (HMAC-SHA256 over attachment/user/disposition/expiry), `Service`
  (sniffed type, declared-type and extension match, streaming size limit, SHA-256, orphan cleanup when metadata
  fails, `canSee` object policy, link issue, download with re-check + audit), `PGStore`, Gin `Handler` with
  swag annotations and security headers (nosniff, sandbox CSP, `private, no-store`, CORP same-origin).
- Routes (all behind session; mutations need CSRF): `GET /attachments/limits`, `POST /attachments`,
  `GET /attachments`, `GET /attachments/{id}`, `GET /attachments/{id}/link`, `GET /attachments/{id}/content`.
- Config: `ATTACHMENT_STORAGE_DRIVER`, `ATTACHMENT_STORAGE_DIR`, `ATTACHMENT_MAX_BYTES`, `ATTACHMENT_LINK_TTL`,
  `ATTACHMENT_SIGNING_KEY` (validated; key + dir required in staging/production). New error codes
  `FILE_TOO_LARGE`, `DOWNLOAD_LINK_INVALID`, `DOWNLOAD_LINK_EXPIRED`. `.gitignore` `/var/`.
- Tests: `http_test.go` (16 tests: 4 allowed types, 10 invalid-file cases incl. HTML-as-PDF, SVG, EXE, mismatch,
  empty, oversized; no storage leak in responses; request shape; orphan cleanup; 401/403/CSRF on every route;
  download headers + audit; 7 invalid-link cases + expiry; cross-user; revocation; location scope; linked
  attachments denied; limits; no mutating route; file-name sanitising; unsafe keys), `pgstore_integration_test.go`
  (5 tests: insert/get/list + audit row, unknown location and invalid audit event roll back, CHECKs, immutability
  and no delete/truncate, link-once, service round trip with both audit rows and no file name in summaries);
  config tests for the new settings. 21 PASS in the package, also with `-race`.

#### Stage 1 — UI (after backend)
Result: COMPLETE — built directly on the real API contract (no mock adapter needed).
- `lib/api/client.ts`: `apiUpload` (XHR for progress, same envelope/error handling, CSRF header), `apiFetchBlob`,
  `resolveApiPath`. `features/attachments`: API module + Zod schemas, hooks (TanStack Query/Mutation),
  `AttachmentUploader` (picker + drop zone, client checks from `/limits`, per-file progress bar, cancel, retry
  only for transient errors, server message + request ID), `DocumentRow` (type, size, date, Download, Preview),
  `ImagePreview` (dialog, bytes fetched through a fresh link, object URL), `DocumentsPage` (permission-aware:
  upload-only / view-only / both / denied; loading, empty, error-with-retry states).
- Navigation: "Documents" link shown with either attachment permission (`NavItem.permission` now accepts a list,
  any-of); route `/documents`.
- Tests: `attachments.test.tsx` (15): client checks, link resolution, hidden link/denied page without calling
  the API, upload-only view, client rejection without sending, success only after server 201 + CSRF sent, server
  415 shown without retry, 503 retry then success, 403 explained, empty list, rows + refused download, download
  via fresh link without rendering it, preview with expired link then retry, list error with retry.

#### Stage 3 — API integration
Result: COMPLETE. Production path uses the real endpoints only; no mock success anywhere. Upload success is shown
after the server's 201; the list refreshes from the server; downloads/previews re-authorise each time.

#### Stage 4 — Browser testing
Result: PASS. Isolated throwaway stack (Postgres :5447, API :8091 this build with `ATTACHMENT_LINK_TTL=30s` and
the development-only proposed-grants flag, Vite :5174); test-only PROPOSED grants on throwaway roles only
(see evidence header). Full Playwright suite: **45 passed, 3 skipped** (2 viewport-specific by design; the
API-level link test runs on desktop only). New `e2e/attachments.e2e.ts` covers: denied user (no link, page
denied, API 403); upload PDF + PNG through the UI, list, client rejection of `.txt`, image preview (decoded, `blob:`
src), download with byte-for-byte comparison, reload persistence, no signed link/storage path in the page, no
horizontal overflow, no console errors; server rejections with the client bypassed (text, HTML-as-PDF,
PDF-as-PNG, empty, 10 MB+ → 415/415/415/422/413) and no CSRF → 403; link checks (headers, tampered, changed
disposition, other user 404/403, anonymous 401, PUT/PATCH/DELETE 405, expiry after 30 s → `DOWNLOAD_LINK_EXPIRED`).
DB after the run: 5 attachment rows = 5 files on disk (mode 600, generated keys), rejected uploads left nothing;
audit `attachment.uploaded` ×5 / `attachment.downloaded` ×5, all with request IDs; 0 rows with a test password.
- Console/network: no page/console errors. API log: a few `context canceled` ERROR lines on `/auth/session`,
  `/access/roles` and one on `/attachments/limits` — requests cancelled by the browser during reloads/navigation
  (pre-existing logging behaviour in auth/access, not a B-09 failure; noted, not changed).
- Evidence: `docs/evidence/B-09/e2e-full-suite.txt`, screenshots `01–04` desktop (1366 px) and mobile (Pixel 7):
  denied, empty Documents, uploads with client errors + list, image preview (captured during the dialog's open
  animation, hence the slight transparency). Test images are generated, not real documents.

#### Checks (actual results)
`make lint-backend` 0 · `make lint-frontend` 0 · `make typecheck-frontend` 0 · `make test-integration` 0 (all
packages ok) · `make test-frontend` 0 (101 tests) · `make swagger-check` 0 · `make build-backend` 0 ·
`make build-frontend` 0 (bundle-size warning only, as before) · `go test -race ./internal/modules/attachments` ok ·
`git diff --check` clean.

#### Files changed (B-09)
`migrations/000006_create_attachments.{up,down}.sql`; `internal/modules/attachments/{service,storage,signer,pgstore,http}.go`
+ `{http_test,pgstore_integration_test}.go`; `internal/modules/access/policy.go` (2 keys); `internal/config/{config,config_test}.go`;
`internal/httpx/response.go` (3 codes); `internal/server/{server,server_test}.go`; `cmd/api/main.go`; `docs/openapi/*`
(regenerated); `.env.example`; `.gitignore`; `ui/src/lib/api/client.ts`; `ui/src/features/attachments/**`;
`ui/src/features/access/api/access.ts`; `ui/src/app/{navigation.ts,routes.tsx}`; `ui/src/components/layout/Sidebar.tsx`;
`ui/e2e/{attachments.e2e.ts,support.ts}`; `docs/{API_CONVENTIONS,ARCHITECTURE_DECISIONS,DEVELOPMENT_SETUP}.md`
(Attachments section, ADR-010, OQ-9/OQ-10, e2e setup); `docs/evidence/B-09/`; this log.

#### Known issues / pending approvals
- No role holds `attachments.upload`/`attachments.view` (RB-5 pending): in dev/staging/production nobody can use
  attachments until the owner approves grants. Positive paths verified only with PROPOSED grants on a throwaway DB.
- OQ-9 accepted types and size limit are proposals; OQ-10 production object storage (S3-compatible) not chosen —
  only the local driver exists; staging/production must set a private `ATTACHMENT_STORAGE_DIR` and signing key.
- Linking attachments to business records and their owner checks come with D-03/D-06/F-09 etc.; linked
  attachments are denied until then. Malware scanning, retention, EXIF stripping not implemented.
- Orphan objects are possible only if the process dies between writing the file and committing metadata (no
  sweeper yet).
- Users' running dev API (:8081) must be restarted to pick up B-09 (dev DB is already at version 6).

#### Post-completion check (2026-10-13, user-requested)
- Dev API on :8081 restarted gracefully (SIGINT, then `make dev-backend`); `/health` ok, `/ready` ready; dev DB at
  version 6 (not dirty). Private storage dir `var/attachments` created `0700` (default; `.env` sets no attachment
  variables). `ATTACHMENT_SIGNING_KEY` is not set in `.env`: development uses a random per-start key by design
  (links stop working after a restart); staging/production refuse to start without one.
- Smoke tests on the dev stack (no grants changed): anonymous `/attachments`, `/attachments/limits` → 401;
  signed-in dev tester (only `access.roles.view`) → 403 on list, limits, upload (with and without CSRF) and
  content; DELETE → 405. 0 attachment rows, 0 files, 0 attachment grants afterwards.
- `git diff --check` clean, no secrets in the diff, `make ci` exit 0 (101 UI tests). Committed after the checks.
- Incident: while checking `.dev-test-accounts` a faulty redaction printed the two local dev test-account
  passwords into the session's tool output (dev DB throwaway accounts only). Rotation recommended.

#### Final status
COMPLETE — B-09 only. Next registry task not started.

### C-01 — Brand/model/variant database schema

- **Status:** COMPLETE (schema; catalogue data and approvals pending — RB-1, RB-2, RB-7, OQ-1, OQ-2, OQ-11)
- **Started:** 2026-10-13 · **Finished:** 2026-10-13
- **Prompt file:** prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/02_Catalogue_Brands/C_01_brand-model-variant-database-schema.md
- Git at start: clean, `dev` = `origin/dev` = `77eb78d`.

#### Dependencies
| Dependency | Verified status | Evidence |
| --- | --- | --- |
| B-05 | COMPLETE | migration framework + `TestMigrationsRoundTrip`, `TestEmbeddedMigrationsFollowPolicy`; dev DB was at 6, clean |
(Also used: B-06 `users` for verifiers, B-09 `attachments` for evidence/source documents — both COMPLETE.)

#### Acceptance criteria
- [x] AC1 Migrations support multiple brands; brand context kept on every child via composite FKs
- [x] AC2 Relationship integrity: FKs (RESTRICT), cross-brand mixes refused, codes unique in brand scope
- [x] AC3 Another brand is added with data only (`TestAnotherBrandNeedsOnlyData`); no brand in code (UI source scan)
- [x] AC4 Lifecycle, authorisation evidence, provenance/verification fields and timestamps; verified specs immutable
- [x] AC5 No invented catalogue data seeded; migration tests/checks pass
- [x] AC6 Frontend: typed domain contracts only, no brand enums, no CRUD UI

#### Stage 1 — UI
Result: NOT APPLICABLE as a screen — Prompt B limits C-01 to typed contracts ("do not add UI CRUD in this
schema-only task"). Delivered `ui/src/features/catalogue/api/contracts.ts` (Zod: Brand, VehicleModel, Variant,
VariantColour, VariantSpec, provenance block, status value sets from the DB CHECKs, `isVerified`) and
`contracts.test.ts` (5 tests incl. a scan proving no roster brand name appears in UI source).

#### Stage 2 — Backend
Result: COMPLETE.
- Migration `000007_create_catalogue` (up/down): brands + `manufacturer_legal_name`, `authorisation_evidence_id`
  (→ attachments), `verified_by` (→ users), CHECK `AUTHORISED` ⇒ legal name + reference + verifier + date;
  tables `models`, `variants`, `variant_colours`, `variant_specs` with composite brand FKs, brand/model-scoped
  unique indexes, code/format/status/confidence CHECKs, provenance CHECK (above UNVERIFIED ⇒ source + date +
  verifier), `updated_at` triggers, version columns, trigger making verified spec rows immutable (`42501`).
  Down migration is destructive for catalogue data (documented; `--confirm` needed in staging/production).
- No endpoints (API and validation are C-02); `internal/modules/catalogue/doc.go` documents the schema.
- Tests `internal/modules/catalogue/schema_integration_test.go` (8): data-only new brand + nothing seeded;
  brand-scoped uniqueness (model code/name/manufacturer code, variant code per model, variant manufacturer code
  per brand, colour name/code); cross-brand mixes and brand moves refused; delete refused (23001) / archive
  works / `updated_at`; 10 value CHECKs; provenance required for models/variants/colours/specs; verified specs
  never overwritten, later versions accepted, drafts editable; brand authorisation evidence + seeded roster
  still pending/unverified.
- **Regression found and fixed:** the new FKs to `attachments` make plain `TRUNCATE attachments` fail with
  `0A000` before the B-09 guard trigger; still refused, but the B-09 test asserted the trigger message. The
  test now checks both plain TRUNCATE and `TRUNCATE … CASCADE` (guard trigger message).
- First run: 3 assertions expected `23503` for RESTRICT deletes; PostgreSQL raises `23001` — test fixed.
- Dev DB migrated 6 → 7 (`make db-migrate-up`, additive; 0 catalogue rows). Dev API (:8081) restarted:
  `/ready` was `not_ready` (schema newer than build) before, `ready` after.

#### Stage 3 — API integration
Result: NOT APPLICABLE — no endpoint in C-01. The expected C-02 JSON contract and DB-error mapping are in
API_CONVENTIONS "Catalogue contracts"; the Zod contracts mirror it.

#### Stage 4 — Browser testing
Result: NOT APPLICABLE — no screen or endpoint changed. Dev stack health re-checked after the restart.

#### Checks (actual results)
`make ci` exit 0 (lint incl. swagger-check, typecheck, test-integration — all 13 Go test packages ok —,
test-frontend 106 tests, build). `git diff --check` clean. First `make ci` run failed on the B-09 truncate
assertion described above; rerun after the fix passed.

#### Evidence
`docs/evidence/C-01/schema-dev-db.txt` — `\d` of brands/models/variants/variant_colours/variant_specs on the dev DB,
version 7, row counts (3 brands, 0 catalogue rows), roster still PENDING_VERIFICATION / UNVERIFIED.

#### Files changed (C-01)
`migrations/000007_create_catalogue.{up,down}.sql`, `internal/modules/catalogue/{doc.go,schema_integration_test.go}`,
`internal/modules/attachments/pgstore_integration_test.go` (truncate assertion), `ui/src/features/catalogue/{api/contracts.ts,contracts.test.ts}`,
`docs/{ARCHITECTURE_DECISIONS,API_CONVENTIONS,REQUIREMENTS_BASELINE}.md` (ADR-011, OQ-11, contract, status line),
`docs/evidence/C-01/`, this log.

#### Known issues / pending approvals
- RB-2: no verified models/variants/colours/specs supplied — nothing seeded (C-05).
- RB-1/RB-7/OQ-1/OQ-2: authorisation evidence, value set and approval owner pending; brands stay pending.
- OQ-11: vehicle-type value set not confirmed (free upper-case identifier for now).
- Operational rules (ACTIVE + AUTHORISED only, lifecycle consistency, permissions, audit) belong to C-02.
- Warranty terms, prices (C-04) and serial/identifier rules (D-04, OQ-4) are not part of this schema.

#### Final status
COMPLETE — C-01 only. C-02 not started.

### C-02 — Catalogue API and validation

- **Status:** COMPLETE (API + typed client/hooks; grants and approvals pending — RB-5, OQ-1, OQ-2, OQ-12–OQ-14)
- **Started:** 2026-10-13 · **Finished:** 2026-10-13
- **Prompt file:** prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/02_Catalogue_Brands/C_02_catalogue-api-and-validation.md
- Git at start: C-01 work was uncommitted; committed separately as `a7eb952` (not pushed) before any C-02 change.

#### Dependencies
| Dependency | Verified status | Evidence |
| --- | --- | --- |
| C-01 | COMPLETE | migration 000007 + 8 schema integration tests (re-run, pass); log entry above; commit `a7eb952` |
| B-07 | COMPLETE | `access.Handler.Require`, deny-by-default, `TestPermissionCatalogueMatchesDatabase` (pass with the 7 new keys) |
| B-08 | COMPLETE | `audit.Record` in the same transaction; audit integration tests pass |

#### Acceptance criteria
- [x] AC1 API validates brand relationships: child brand derived from the parent; mismatching `brandId` → 422; brand/model of an item cannot change; brand-scoped duplicates → 409 `DUPLICATE`
- [x] AC2 Permissions: 7 catalogue keys, deny by default (no grants), route guard + service re-check; each route refused without its own key
- [x] AC3 Archive avoids breaking history: no DELETE route (405), archive never deletes or cascades, children keep references, nothing new under archived parents
- [x] AC4 Invalid activation denied: unverified items, inactive parents, and every brand (pending prerequisites, fail closed) → 409 `ACTIVATION_BLOCKED`, audited as DENIED
- [x] AC5 Sensitive create/update/authorise/verify/activate/archive actions audited in the same transaction
- [x] AC6 CRUD/list/search with pagination and filters; optimistic concurrency (`version`)
- [x] AC7 OpenAPI regenerated (`make swagger`, `swagger-check` passes) and API conventions updated
- [x] AC8 Frontend: typed services + TanStack Query hooks matching OpenAPI; no screens (C-03); no hardcoded brands/statuses beyond DB value sets
- [x] AC9 Backend and frontend tests for valid requests, invalid relationships, permissions, archive, activation

#### Stage 1 — UI
Result: NOT APPLICABLE as a screen — Prompt B limits C-02 to service functions and hooks; screens are C-03. The user
asked for the backend-first order, so no mock adapter was built: the client and hooks were written against the real
contract in Stage 3.

#### Stage 2 — Backend
Result: COMPLETE.
- Migration `000008_catalogue_permissions` (up/down): permission keys `catalogue.view`, `.manage`, `.verify`,
  `.activate`, `.archive`, `catalogue.brand.authorise`, `catalogue.brand.activate`; no grants. Down fails if a grant exists.
- `internal/modules/catalogue`: `types.go`, `validate.go` (normalise/validate; unknown fields refused), `rules.go`
  (pure lifecycle/activation rules), `service.go` (permission re-check, audit events, DENIED audit of refused
  activations), `pgstore.go` (one transaction per mutation, `FOR UPDATE` item / `FOR SHARE` parent, version check,
  constraint → domain error mapping), `http.go` (29 routes with swag annotations). ADR-012 records the rules.
- New error codes (409): `DUPLICATE`, `VERSION_CONFLICT`, `INVALID_STATE`, `ACTIVATION_BLOCKED`.
- Router/main wiring; server tests updated (`testCatalogue`).
- Tests: `http_test.go` (no DB; nil repository proves checks run before data access): 401 without session, 403 deny
  by default on all 29 routes, each route 403 without its own key, no DELETE, 25 validation cases, no leaked DB detail.
  `rules_test.go`: activation/readiness/edit/archive rules, slug, PATCH null vs absent, normalisation, IST date edge.
  `api_integration_test.go` (real PostgreSQL via `dbtest`, real auth sessions + CSRF): happy path + paging/search/
  literal wildcards, brand-scoped uniqueness and cross-brand refusals, activation order and DENIED audit, brand
  authorisation evidence and the fail-closed gate, archive keeps history, edits reset verification, stale version,
  8 concurrent edits → exactly 1 succeeds, 6 concurrent creates of one code → exactly 1 created, read-only user.
- Dev DB migrated 7 → 8 (`make db-migrate-up`; adds 7 permission rows only, no grants, no catalogue data).
- Fixed during review: `sourceDate` "future" check compared a UTC date with server time, refusing "today" just after
  midnight IST; now a date is future only if it has not begun in any timezone (test added).

#### Stage 3 — API integration
Result: COMPLETE (client layer; no screen consumes it yet).
- `ui/src/features/catalogue/api/catalogue.ts`: typed functions for all 29 routes through `apiRequest` (session
  cookie, CSRF, Zod-validated responses); `activationBlockers`, `isVersionConflict` helpers.
- `ui/src/features/catalogue/hooks/useCatalogue.ts`: query keys, list/detail/readiness queries (`enabled` for
  permission gating), mutations that update the cache only after the server confirms, then invalidate the catalogue.
- `contracts.ts`: `Brand.operational`, `pageSchema`, readiness schemas.
- Tests `catalogue.test.tsx` (10): URLs/methods/bodies, 409 ACTIVATION_BLOCKED details, VERSION_CONFLICT, 403, 422
  relationship error, contract mismatch rejected, loading/empty list, disabled query makes no request, 403 surfaced,
  failed mutation leaves cache unchanged, confirmed create invalidates lists.

#### Stage 4 — Browser testing
Result: NOT APPLICABLE as a browser test — C-02 adds no screen or route in the UI (C-03 will). Instead the real API
binary was exercised over HTTP on an isolated stack: **36/36 checks passed** (401/403 deny-by-default, CSRF,
duplicates, cross-brand refusals, activation order, stale version, authorisation evidence, blocked brand activation,
archive keeps history, 405 on DELETE, paging, validation), 12 kinds of catalogue audit rows incl. 3 DENIED, 0 API
ERROR log lines. Stack (throwaway postgres:17-alpine :5448, API :8092 with `RBAC_ALLOW_PROPOSED_GRANTS=true`,
test-only users and PROPOSED grants) removed afterwards; shared dev DB untouched apart from migration 8.

#### Checks (actual results)
- `make ci`: exit 0 — gofmt, go vet, tidy, swagger-check, ESLint, Prettier, `tsc`, `go test` with PostgreSQL
  (`REQUIRE_INTEGRATION_TESTS=1`; 13 packages ok), Vitest 15 files / 116 tests, Go + Vite builds. golangci-lint not
  installed (skipped by the Makefile, as before). First `make ci` failed on gofmt (`cmd/api/main.go`, `pgstore.go`);
  formatted and rerun. Catalogue package re-run after the date fix: ok. `git diff --check` clean.

#### Evidence
`docs/evidence/C-02/live-api-smoke.txt` — live API run, results and audit rows.

#### Files changed (C-02)
`migrations/000008_catalogue_permissions.{up,down}.sql`; `internal/modules/catalogue/{types,validate,rules,service,pgstore,http}.go`
+ tests `{harness,http,rules,api_integration}_test.go`, `doc.go`; `internal/modules/access/policy.go`; `internal/httpx/response.go`;
`internal/server/{server,server_test}.go`; `cmd/api/main.go`; `docs/openapi/*` (generated);
`ui/src/features/catalogue/{api/catalogue.ts,api/contracts.ts,hooks/useCatalogue.ts,catalogue.test.tsx,contracts.test.ts}`;
`docs/{API_CONVENTIONS,ARCHITECTURE_DECISIONS,REQUIREMENTS_BASELINE}.md`; `docs/evidence/C-02/`; this log.

#### Known issues / pending approvals
- No brand can be activated: pending checks `supplier_link` (D-01), `approved_prices` (C-04), `warranty_terms`
  (RB-2), `identifier_rules` (D-04, OQ-4) are never met by design until those tasks add them.
- Nobody holds any catalogue permission until the owner approves the matrix (RB-5); approval owner OQ-2.
- OQ-12 correcting ACTIVE items / restoring archived ones; OQ-13 separation of duties; OQ-14 who may record CLIENT_APPROVED.
- Spec value (`variant_specs`) endpoints are not part of C-02's listed scope (brand/model/variant/colour); not built.
- Archive does not cascade: later purchase/sales tasks must require operational brand + ACTIVE model + ACTIVE variant.
- Brand scope per user (RB-5) still not modelled; lists show all brands to holders of `catalogue.view`.

#### Final verification before commit (user-requested, 2026-10-13)
- **Scope honesty:** C-02 has no catalogue screen, so no catalogue *UI* browser test exists or is claimed. The catalogue
  API was exercised from real signed-in browser sessions (Playwright) and over plain HTTP.
- **Isolated stack** (throwaway postgres:18-alpine :5449, API :8093 from this tree, Vite :5175; test-only users and
  PROPOSED grants; removed afterwards): `npx playwright test` desktop + Pixel 7.
  - Run 1: 46 passed, **1 failed**, 5 skipped. The new `e2e/catalogue.e2e.ts` hit `429 RATE_LIMITED` (per-IP limiter,
    20 rps / burst 40) because it fetched the CSRF token before every mutation. A test-harness issue, not a C-02
    defect: the spec now reads the token once, and the isolated API ran with `RATE_LIMIT_RPS=100 RATE_LIMIT_BURST=200`
    (documented in DEVELOPMENT_SETUP like the existing `LOGIN_MAX_FAILURES_PER_IP` note).
  - Run 2: **47 passed, 0 failed, 5 skipped** (by design: 2 viewport-specific shell tests, 3 API-level tests run on
    desktop only). Includes all existing B-03…B-09 regression tests (auth, session, CSRF, roles, attachments, shell)
    and the 2 new C-02 tests: deny by default (401 no session, 403 no grant, 403 `CSRF_FAILED`), and duplicates,
    unknown fields, cross-brand `brandId`, model brand move, activation order, `VERSION_CONFLICT`, edit of ACTIVE item,
    authorisation without evidence, readiness, blocked brand activation, archive keeps history, no child under an
    archived model, DELETE 405, paging/limit validation, read-only viewer — with no console errors.
  - Live API smoke script re-run on the same stack: **36/36 passed**.
  - API log: 8 ERROR lines, all `context canceled` on `/attachments`, `/access/roles`, `/auth/session` (browser aborted
    the request during navigation/teardown). Pre-existing logging behaviour from B-06/B-07/B-09, not on catalogue
    routes, not a C-02 regression — left unchanged (out of scope); worth a later look (log client aborts below ERROR).
- **Shared dev environment:** `make db-up`, `make db-migrate-up` (no change, v8), `make dev` → exit 2 because a Vite +
  API pair (started 17:40 from this repo, serving the C-02 routes) already held :5173/:8081; that stack was used as-is
  and left running. Headless Chrome, desktop + mobile: UI loads, sign-in renders on a deep link, empty submit validated
  client-side, wrong credentials → 401 `INVALID_CREDENTIALS` generic message, no horizontal overflow,
  `/api/v1/brands` via proxy → 401, **0 console/page errors**. No sign-in on dev (no test account exists; none created).
- **Dev DB permissions unchanged** (snapshot before/after identical): 1 PROPOSED grant (pre-existing), 1 user role,
  0 catalogue grants, 0 models. Only side effect: ANONYMOUS failed-sign-in audit rows from the 2 deliberate wrong logins.
- **`make ci`** on the final tree: exit 0 (13 Go packages ok with PostgreSQL, Vitest 15 files / 116 tests, lint,
  typecheck, swagger-check, builds; golangci-lint not installed → skipped). `git diff --check` clean.
- **C-02 bugs found in this pass:** none in the API. Test/docs changes only: `ui/e2e/catalogue.e2e.ts` (new),
  `ui/e2e/support.ts` (2 optional accounts), `docs/DEVELOPMENT_SETUP.md` (catalogue e2e accounts, rate-limit note).
- Evidence: `docs/evidence/C-02/final-verification-isolated.txt`, `docs/evidence/C-02/final-verification-dev.txt`.

#### Final status
COMPLETE — C-02 only. C-03 not started.

### C-03 — Catalogue and brand management UI

- **Status:** COMPLETE (screens over the C-02 API; grants, supplier links, prices and approvals still pending)
- **Started:** 2026-10-13 · **Finished:** 2026-10-13
- **Prompt file:** prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/02_Catalogue_Brands/C_03_catalogue-and-brand-management-ui.md
- Git at start: `dev` = `origin/dev` = `20429e0` (C-01 `a7eb952` and C-02 `20429e0` already pushed; nothing to push).
  Untracked `prompt-pack/.~lock.TASK_REGISTRY.csv#` (LibreOffice lock, not ours) left untouched.

#### Dependencies
| Dependency | Verified status | Evidence |
| --- | --- | --- |
| C-02 | COMPLETE | commit `20429e0`; catalogue API, typed client and hooks; final verification above (47 Playwright passed, `make ci` 0) |
| B-03 | COMPLETE | app shell, navigation, PageHeader, Table, Dialog, Field, StateViews in `ui/src/components` |

#### Acceptance criteria
- [x] AC1 Brand list/editor and model/variant (and colour) screens manage the catalogue through the real C-02 API only
- [x] AC2 Search, pagination, lifecycle/authorisation/verification badges; supplier relationship shown only from real data (readiness `supplier_link`, pending until D-01)
- [x] AC3 Create/edit forms (React Hook Form + Zod mirroring server rules); archive confirmation with reason
- [x] AC4 Choosing a brand filters model/variant options; cross-brand choices impossible in the UI, still rejected server-side
- [x] AC5 No unverified item appears operationally active; warnings for unverified brand, specifications and prices
- [x] AC6 Activation requires permission and a confirmation dialog stating the effect
- [x] AC7 Loading, empty, no-results, validation, network, 401, 403, duplicate, invalid-state, activation-blocked and stale-version states
- [x] AC8 Responsive (desktop + mobile) and accessible; Playwright on both viewports; `make ci` passes

#### Stage 1 — UI
Result: COMPLETE — built directly on the real C-02 typed client and hooks (the API already existed), so no mock
adapter was needed or used.
- Route `/admin/catalogue` (gated by `catalogue.view`; page header + Access denied otherwise) with tabs **Brands |
  Models | Variants**; detail pages `/admin/catalogue/brands/:id`, `/models/:id`, `/variants/:id`.
- Lists: server search (debounced), status/authorisation/verification/brand filters, server pagination, all kept in
  the URL (refresh/back keep state); loading, empty (with permitted action), no-results, error with reference + retry,
  403 → Access denied. Table on ≥ sm; stacked labelled list on phones (no sideways scrolling).
- Badges: lifecycle, authorisation, confidence, and "Enabled / Not enabled for purchase & sales" taken only from the
  server's `operational` flag — the UI never derives it, so nothing unverified looks operational.
- Brand detail: authorisation card + **Record authorisation** dialog (status, legal name, reference, private evidence
  document picked from the user's uploads or uploaded via the B-09 uploader); **activation readiness** straight from
  the server checklist (met / not met / not available yet); **Supplier relationship** card shows only real data —
  none exists until D-01, so it says so; **Specifications, prices and warranty** card states they are not recorded.
- Model/variant detail: details, source & verification card (Unverified warning), brand context with its purchase/
  sales badge, child lists (variants / colours with per-row actions).
- Forms (React Hook Form + Zod mirroring server rules): add/edit brand, model, variant, colour; verify (confidence,
  source link and/or document, date ≤ today); archive with optional reason; confirmations for activation stating the
  effect. Edit is offered only while pending; actions appear only with their permission. Dialogs mount only while
  open (fresh state each time), disable submit while pending, and show server errors: field-level VALIDATION_FAILED /
  DUPLICATE, ACTIVATION_BLOCKED with the server's unmet requirements, VERSION_CONFLICT with "Reload latest version",
  INVALID_STATE, 403, CSRF, network ("Nothing was saved"). 401 is handled globally (back to sign-in).
- Brand → model: the Add-variant dialog and the Variants filter offer only the chosen brand's models (and never
  archived brands/models); the request sends only `modelId`, the server derives the brand and still rejects mismatches.
- Navigation: "Catalogue & Brands" is no longer "Planned"; the link needs `catalogue.view` like Users & Roles (tests
  updated). Dashboard shortcut shows "Available / Open module" and no longer promises battery ratings or price history.
- New shared component `components/ui/Pagination.tsx`; `Permission` gains the 7 catalogue keys. No new dependency.
- Tests: `pages.test.tsx` (15) — access denied without call, server 403, list badges/operational, search + no
  results + clear, empty and error states, create (client validation, server duplicate on the field, navigate only
  after 201), brand→model cascade and archived brands excluded, server cross-brand refusal shown, variants filter
  cascade, readiness/supplier/price gaps and disabled activation, permission-hidden actions, not found, activation
  blocked list, stale-version reload, archive with reason, network loss.

#### Stage 2 — Backend
Result: NOT APPLICABLE — every list/detail/action the screens need exists in C-02 (checked against OpenAPI); no
endpoint, migration or Go change was made. Not available and therefore not shown: verifier names (only "You" or a
short user ID — there is no user lookup API), supplier links (D-01), spec values, prices (C-04).

#### Stage 3 — API integration
Result: COMPLETE — production path uses only `api/catalogue.ts` + `hooks/useCatalogue.ts` (added
`useRefreshCatalogue` for stale-version reloads). Success is shown only after the server confirms; caches update only
on success.

#### Stage 4 — Browser testing
Result: COMPLETE on an isolated throwaway stack (details: `docs/evidence/C-03/browser-verification.txt`).
- New `ui/e2e/catalogue-ui.e2e.ts` (desktop + Pixel 7): no-permission user has no link and gets Access denied; viewer
  sees the real seeded roster as pending/unverified/not enabled with no edit actions and the readiness list; admin
  creates a brand (client validation, then server), sees the duplicate on the field, adds a model, is refused
  activation before verification (server reason shown), verifies, activates (edit disappears), adds a variant through
  the brand→model cascade (only that brand's model offered), hits a real stale-version conflict from a second session
  and reloads, archives the model with a reason (history kept, no new variants), cannot confirm brand activation,
  refresh keeps server state, no sideways scroll, no console errors.
- Screenshot review found and fixed: duplicated readiness text, stretched cards, mobile tables hiding statuses
  (now a stacked list). First run 6/6; after fixes the full suite: **53 passed, 0 failed, 5 skipped** (by design).
- API log: only `context canceled` lines from aborted browser requests (pre-existing behaviour, see C-02 verification).

#### Checks (actual results)
- `npx tsc -b`, ESLint (`--max-warnings=0`), Prettier: clean. First lint run failed (setState-in-effect resets,
  `form.watch`, mixed exports, test string conversions) — fixed by mounting dialogs only while open, `useWatch`,
  moving list helpers to `lib/listParams.ts`, render-time search sync; rerun clean.
- Vitest: 16 files / **132 tests** passed (2 existing navigation assertions updated for the new permission gate).
- `make ci`: exit 0 (13 Go packages ok with PostgreSQL, Vitest, lint, typecheck, swagger-check, builds;
  golangci-lint not installed → skipped). `git diff --check` clean.

#### Evidence
`docs/evidence/C-03/browser-verification.txt`; screenshots `desktop-brands.png`, `desktop-brand-detail.png`,
`desktop-add-variant-dialog.png`, `mobile-brands.png`, `mobile-model-detail.png`.

#### Known issues / pending approvals
- Nobody holds catalogue permissions on real databases until the owner approves grants (RB-5, OQ-2); screens show
  Access denied until then.
- Brand activation cannot succeed until supplier links (D-01), prices (C-04), warranty terms (RB-2) and identifier rules
  (D-04) exist; the UI shows this from the server's readiness list.
- No user-name lookup API: verifiers show as "You" or "User <id prefix>".
- Spec values have no API yet; correction of active items / restoring archived ones pending (OQ-12).
- Option pickers load up to 100 brands/models; enough for the current roster, revisit if catalogues grow.

#### Final browser/regression verification (user-requested, 2026-10-13; verification only)
- Commit `9609244`, clean tree; no code changed in this pass.
- Shared dev stack (already running, used as-is): headless Chrome desktop + mobile — `/admin/catalogue` deep link
  goes to sign-in, empty submit validated client-side, wrong credentials → 401 generic message, no horizontal
  overflow, `/api/v1/brands` without session → 401, **0 console/page errors**. Dev DB permissions/data identical
  before and after (1 PROPOSED grant, 0 catalogue grants, 3 brands, 0 models).
- Isolated throwaway stack, full `npx playwright test` (desktop + Pixel 7): **53 passed, 0 failed, 5 skipped** (by
  design). API ERROR lines (7 requests) all traced by request ID to `context canceled` — client aborts, no server failure.
- `make ci` exit 0; `go test -json ./...` with PostgreSQL: 13 packages pass, 302 tests pass, 0 fail, 0 skipped;
  Vitest 132/132. `git diff --check` clean. No bugs found; no fixes needed.
- Evidence: `docs/evidence/C-03/final-verification.txt`.

#### Manual verification by the user (2026-10-14)
- Environment provided for it (2026-10-13): temporary throwaway stack — postgres:18-alpine :5449, API :8093 built from
  `9609244`, UI http://localhost:5175 — with test-only accounts `e2e.catalogue` (PROPOSED grants for the 7 catalogue keys
  and attachments), `e2e.catviewer` (`catalogue.view`) and `e2e.active` (no role), plus a suggested manual test script
  (admin flow, stale version, viewer, no access, mobile).
- **User's statement:** "C-03 manual testing is complete and working correctly, as verified by me."
- Recorded as an overall user confirmation only. Which individual steps of the suggested script were exercised was not
  reported, so no individual manual test case is claimed here.
- Temporary environment removed afterwards (API, Vite :5175, container and its database, temporary password and upload
  files). Normal development environment (:5173, :8081, DB :5436) and its database left untouched.

#### Final status
COMPLETE — C-03 only. C-04 not started.

### C-04 — Effective-dated price history

- **Status:** COMPLETE (mechanism; no price data, tax configuration or grants — OQ-15…OQ-18, RB-5)
- **Started:** 2026-10-14 · **Finished:** 2026-10-14
- **Prompt file:** prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/02_Catalogue_Brands/C_04_effective-dated-price-history.md
- Git at start: `dev` = `origin/dev` = `f652169`; C-01 `a7eb952`, C-02 `20429e0`, C-03 `9609244` committed and pushed.
  Untracked LibreOffice lock file left untouched.

#### Dependencies
| Dependency | Verified status | Evidence |
| --- | --- | --- |
| C-01 | COMPLETE | migration 000007, schema tests; commit `a7eb952` |
| C-02 | COMPLETE | catalogue API, readiness check `approved_prices` pending for C-04; commit `20429e0` |
(Also used: C-03 screens `9609244` for the price UI; B-07 permissions; B-08 audit.)

#### Acceptance criteria
- [x] AC1 Price lists and price items linked to brand, model and variant (brand context enforced by the database)
- [x] AC2 Price type, amount, effective from/to, approval status/actor/time/reference, tax configuration reference
- [x] AC3 Date ranges validated; overlapping approved periods refused (database exclusion + service check, concurrency-safe)
- [x] AC4 Approved prices immutable except controlled supersession; no silent retroactive change
- [x] AC5 Lookup of the price valid at a given time, test-covered including after supersession (old quotations stay stable)
- [x] AC6 Permissions deny by default; cost prices only with cost permissions; no invented prices, tax rates or grants
- [x] AC7 OpenAPI, typed client, hooks and UI (history with current/future/past, approval status, snapshot warning) with all states
- [x] AC8 Backend, API, date-range, approval, concurrency and lookup tests; browser tests; `make ci`

#### Stage 1 — UI
Result: built after the backend (backend-first order requested), directly on the real API — no mock adapter.
- Variant page **Price history** card (only with a price view permission): price list and type selectors (type switch
  only for cost viewers), history table/phone list with Current / Future / Past / Draft / Cancelled badges computed
  from server dates, amount in ₹, effective window, approver/time/reference, "Superseded"; snapshot warning ("never
  changes existing quotations"). Actions by permission: Draft price, Edit/Cancel draft, Approve (hidden for one's own
  draft), Change price (supersession draft). "Price valid at" lookup tool. Warning on drafts that start in the past.
- Brand page **Price lists** card: list, Add price list, Archive (confirmation).
- Dialogs (React Hook Form + Zod): draft/edit/supersede (amount string, local date-time → RFC 3339, end after start,
  source note; tax treatment explained as not available), approve (reference, rules, snapshot notice), cancel draft,
  create/archive list. Errors: PRICE_OVERLAP, APPROVAL_BLOCKED (server's unmet list), SELF_APPROVAL_NOT_ALLOWED,
  retroactive start, DUPLICATE, VERSION_CONFLICT (reload), 403, network.
- C-03 wording updated: prices now come only from approved history; phone lists hide empty fields.
- Tests `features/pricing/pricing.test.tsx` (10): formatting/periods, hidden without permission (no request),
  history badges + snapshot warning + selling-only requests, approve only others' drafts + overlap message,
  approval blockers, draft validation + payload (decimal string, RFC 3339, supersedesId), lookup found/none,
  cost behind type switch, no price list, price list duplicate.

#### Stage 2 — Backend
Result: COMPLETE.
- Migration `000009_create_price_history` (up/down): `price_lists`, `price_items`, `btree_gist`, `variants (id, model_id)`
  unique, 3 permission keys (no grants), immutability trigger, exclusion constraint, CHECKs (see ADR-013).
- `internal/modules/pricing` (types, validate, rules, service, pgstore, http; 12 routes). `access.Handler.RequireAny`.
  Catalogue readiness `approved_prices` is now a real check. New error codes: PRICE_OVERLAP, APPROVAL_BLOCKED,
  SELF_APPROVAL_NOT_ALLOWED, PRICE_NOT_FOUND.
- Tests: `http_test.go` (deny by default on all routes, cost permissions, 16 validation cases, approval rules, amount
  normalisation) and `api_integration_test.go` on PostgreSQL: draft → self-approval refused (audited DENIED) →
  approve; lookup before/at start; supersession ends the old period, historical lookup unchanged (old quotations
  stable), boundary instants; overlap refused; no retroactive approval (and allowed under a simulated earlier clock);
  unverified variant blocked; cross-brand refused by service and database; archived list keeps lookups; cost prices
  hidden from selling-only users; approved rows immutable in SQL (42501), self-approval and tax reference refused by
  CHECK (23514); **6 concurrent overlapping approvals → exactly 1 succeeds**; brand readiness counts only a current
  approved price. Catalogue tests updated for the real `approved_prices` check.
- Dev DB migrated 8 → 9 (`make db-migrate-up`; additive, no data, no grants).

#### Stage 3 — API integration
Result: COMPLETE — `features/pricing/api/pricing.ts` (Zod, decimal strings) + `hooks/usePricing.ts`; caches refresh
only after the server confirms (prices and catalogue readiness).

#### Stage 4 — Browser testing
Result: COMPLETE on an isolated throwaway stack (`docs/evidence/C-04/verification.txt`). New `e2e/prices.e2e.ts`
(desktop + Pixel 7) with two people: price list, draft, drafter cannot approve, other person approves with
reference, overlap refused, draft cancelled, controlled supersession, lookups (old instant ₹1,000 unchanged, later
₹1,200, before any price "none"), retroactive approval refused, cost type switch only for cost viewers, no
overflow, no console errors. Screenshot review fixes (button wrap, past-start hint, empty phone labels). Final full
suite: **55 passed, 0 failed, 5 skipped** (by design). API errors only client `context canceled`.

#### Checks (actual results)
- `make ci` exit 0 (gofmt, vet, tidy, swagger-check, ESLint, Prettier, tsc, Go tests with PostgreSQL, Vitest, builds;
  golangci-lint not installed → skipped). `go test -json ./...`: 14 packages pass, **330 tests** pass, 0 fail.
  Vitest 17 files / **142 tests**. `git diff --check` clean.
- Lint fixes during the task: hook exported from a component file moved to `hooks/usePricePermissions.ts`; two
  always-true conditions removed; test string conversions.

#### Evidence
`docs/evidence/C-04/verification.txt`, `desktop-variant-prices.png`, `mobile-draft-dialog.png`.

#### Known issues / pending approvals
- No price data exists or is seeded; prices must come from the owner's approved source (RB-2).
- OQ-15 price types and what the selling amount includes; OQ-16 backdating; OQ-17 maker-checker/approver/threshold;
  OQ-18 tax treatment — `taxConfigurationId` stays empty until the accountant-approved configuration (F-10).
- Nobody holds price permissions until the owner approves grants (RB-5).
- Quotations that snapshot prices are F-02; lookup is the contract they will use.
- Brand activation still blocked by supplier/warranty/identifier checks.

#### Final browser/regression verification (user-requested, 2026-10-14; verification only)
- Commit `2594848`, clean tree; no code changed.
- **Dev environment finding (not a C-04 defect):** the running dev API (:8081, started 10:09, build `9609244` = C-03) is
  stale — `/ready` not_ready (DB at 9, build expects 8) and price routes 404. Left untouched; restart `make dev-backend`
  to pick up C-04. The current build `2594848` was run on side ports against the dev DB: `/ready` ready; headless Chrome
  desktop + mobile: catalogue deep link → sign-in, client validation, wrong credentials → 401 generic message,
  `/price-lists`, `/price-items`, `/prices/lookup`, `/brands` without session → 401, no overflow, **0 console errors**.
  Dev DB identical before/after (0 catalogue/price grants, 0 price lists/items).
- Isolated stack, full `npx playwright test` (desktop + Pixel 7): **55 passed, 0 failed, 5 skipped** (by design).
  API errors (14 requests) all `context canceled` client aborts.
- `make ci` exit 0; Go 14 packages / 330 tests pass; Vitest 142/142. No bugs found; no fixes.
- Evidence: `docs/evidence/C-04/final-verification.txt`.

#### Final status
COMPLETE — C-04 only. C-05 not started.

### A-01 — Record confirmed launch brands and activation evidence (discovery item; documentation only)

- **Status:** INCOMPLETE — documentation done; **client evidence pending** (not COMPLETE)
- **Started/updated:** 2026-10-14
- **Source:** `prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/00_START_HERE/00_Client_Discovery_Prompts.md` §A-01;
  Research Brief A-01; REQUIREMENTS_BASELINE §1. A-items are not in the 59-task registry; C-05 depends on A-01.
- Git at start: `dev` = `origin/dev` = `44d880c`, clean apart from the untracked LibreOffice lock file (untouched).
- Workflow: documentation only — no UI, backend, browser, seed, database or permission change (TASK_EXECUTION_WORKFLOW §4).

#### Acceptance criteria (from the A-01 prompt)
- [x] `docs/client-inputs/launch-brand-roster.md` records the roster (Lectrix EV, Ceeon India, Ujagar Electric) as client-confirmed
- [x] Per brand: fields for legal supplier/manufacturer entity, supplier contact, authorisation reference/document,
  catalogue source, verification status, approval owner, verification date, allowed purchase/sales status
- [x] Roster selection stated not to be proof of authorisation; existence kept separate from operational eligibility
- [x] Ujagar Electric and Saffron EV Group explicitly not assumed to be the same legal entity
- [x] Nothing invented: every unavailable field reads `Pending — client evidence not supplied`
- [ ] **Per-brand evidence collected** (legal entity, contact, authorisation document, catalogue source, approval
  owner, verification date) — **not supplied by the client**; this is what keeps A-01 incomplete

#### Done
- `docs/client-inputs/launch-brand-roster.md`: confirmed facts; limits (roster ≠ authorisation; Ujagar ≠ Saffron
  assumption); per-brand tables with all evidence pending and purchase/sales **Not allowed**; the application's
  activation checklist and its current state; public references kept separate and labelled "not evidence"; how to
  complete A-01.
- Evidence `docs/evidence/A-01/brand-records-dev-db.txt`: read-only query — 3 roster brands PENDING_VERIFICATION /
  UNVERIFIED, no legal name/reference/evidence/verifier, 0 models/variants/prices, 0 catalogue/price grants.
- No change to `seeds/`, the database, permissions or code.

#### Dependency check for C-05
**Not satisfied.** C-05 depends on A-01, C-01, C-02. C-01 and C-02 are COMPLETE; A-01 is INCOMPLETE because no
per-brand evidence has been supplied. Separately, C-05 ("seed verified multi-brand catalogue") also needs verified
catalogue data (A-03 / RB-2), which is not supplied either. C-05 not started.

#### Remaining blockers (client)
For each of Lectrix EV, Ceeon India and Ujagar Electric: legal supplier/manufacturer entity and contact;
dealer/stocking authorisation reference and document; catalogue source; approval owner (OQ-2); verification date.
For Ujagar Electric: documented relationship (or none) to Saffron EV Group. Verified catalogue data: A-03.

### C-06 — Brand-specific rules and isolation tests

- **Status:** COMPLETE (for implemented features; supplier/warranty/identifier rules, exports, per-user brand scope pending)
- **Started:** 2026-10-14 · **Finished:** 2026-10-14
- **Prompt file:** prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/02_Catalogue_Brands/C_06_brand-specific-rules-and-isolation-tests.md
- Git at start: `dev` at `7b35af7` (1 ahead of `origin/dev`: A-01 docs, unpushed). Lock file untouched. C-05 not started
  (A-01 INCOMPLETE).

#### Dependencies
| Dependency | Verified status | Evidence |
| --- | --- | --- |
| C-02 | COMPLETE | `20429e0`; catalogue API + cross-brand refusals |
| C-04 | COMPLETE | `2594848`; price history, lookup, composite brand FKs |
| B-07 | COMPLETE | RBAC deny-by-default; location scope; brand scope per user pending (RB-5) |

#### Gap analysis at start
- Existing: cross-brand refusals on create/move (catalogue, price items), composite brand FKs, cost prices behind
  cost permissions, no brand names in code (only `seeds/0001_initial_brands.sql`).
- Gaps: contradictory cross-brand **filters** return silent empty lists (variants brandId+modelId, colours
  brandId+variantId, price items brandId/priceListId/variantId); price **lookup** with a variant of another brand than
  the list answers 404 instead of refusing; no end-to-end "arbitrary new brand" test; no Go-side scan for hardcoded brands.
- Not implemented anywhere yet (documented, not fabricated): supplier mappings (D-01), warranty terms (RB-2),
  identifier rules (D-04/OQ-4), export endpoints (G-05), per-user brand scope (RB-5).

#### Acceptance criteria
- [x] AC1 Cross-brand mismatch requests fail (writes, contradictory filters, price lookup) — API + PostgreSQL tests
- [x] AC2 Brand filters stay scoped (catalogue lists, price lists/items, readiness) — isolation tests across 3+ brands
- [x] AC3 An arbitrary new brand works end-to-end with data only; no brand name in Go or UI code (source scans)
- [x] AC4 Unauthorised access denied; cost/authorisation details permission-based (API + UI)
- [x] AC5 UI: brand choice constrains model/variant options; filters stay consistent (stale model filter dropped)
- [x] AC6 Missing features (supplier, warranty, identifier rules, exports, per-user brand scope) documented as limitations
- [x] AC7 Tests, lint, typecheck, builds, `make ci`, browser test of the changed UI

#### Stage 1 — UI
Result: COMPLETE. `VariantsPage` checks the brand of a `modelId` filter (by ID, not the 100-item option list) and drops
a model of another brand from the URL before calling the API (stale/edited links no longer error). Tests
`features/catalogue/isolation.test.tsx` (5): stale cross-brand model filter dropped and never sent; matching pair kept;
brand change clears the model and offers only that brand's models; price lists requested with the variant's own brand;
arbitrary brand name from data with activation warning, authorisation editing/enabling and price sections gated by
permission. (Existing: brand→model pickers C-03, cost-price gating C-04.)

#### Stage 2 — Backend
Result: COMPLETE. No migration.
- `catalogue/pgstore.go`: `filterBrandMatches` — `GET /variants` (brandId+modelId) and `GET /colours`
  (brandId+variantId) refuse cross-brand pairs and unknown parents with 422.
- `pricing/pgstore.go`: `checkFilterBrands` — `GET /price-items` (brandId/priceListId/variantId) and `GET /prices/lookup`
  (list vs variant) refuse cross-brand mismatches and unknown IDs with 422.
- OpenAPI descriptions updated (`make swagger`, swagger-check ok). ADR-014.
- New test package `internal/isolation` (real catalogue + pricing routes, real sessions, PostgreSQL): arbitrary new brand
  end to end (catalogue, price list, approved price, lookup, listing) with data only; cross-brand writes, filters,
  lookup, supersession and DB inserts (23503) refused; filters scoped across 3 brands (models, variants, colours,
  price lists, price items, search) with SELLING+COST prices; readiness uses only the brand's own price; archiving one
  brand leaves another; unauthorised/no-session/selling-only/authorisation-permission denials; source scan for brand
  names (read from the seed) in Go/SQL/UI code. Mutation check: without the store changes the cross-brand test fails.
- Existing C-02 integration test updated: cross-brand variant filter now expects 422 (was an empty 200).

#### Stage 3 — API integration
Result: COMPLETE — no client change needed beyond the Variants filter guard; the UI never sends mismatched pairs.

#### Stage 4 — Browser testing
Result: COMPLETE on an isolated throwaway stack. New `e2e/isolation.e2e.ts` (desktop + Pixel 7): two arbitrary brands;
stale link `?brandId=B&modelId=<A model>` becomes `?brandId=B`, only B's variant listed, no error; model filter offers
only B's model; signed-in API calls for cross-brand variant filter, price items and lookup → 422, cross-brand variant
create → 422; brand filters return only that brand's rows; no console errors. Full suite: **57 passed, 0 failed,
5 skipped** (by design). API errors only client `context canceled`.

#### Checks (actual results)
`make ci` exit 0; `go test -json ./...` with PostgreSQL: 15 packages / **336 tests** pass, 0 fail; Vitest 18 files /
**147 tests**; `git diff --check` clean. One first-run failure: the C-02 test asserting the old empty-200 filter
behaviour — updated to the intended 422.

#### Evidence
`docs/evidence/C-06/verification.txt`.

#### Known limitations / unresolved dependencies
- Supplier mappings (D-01), warranty terms (RB-2), identifier rules (D-04, OQ-4): not implemented; not fabricated;
  each needs brand-scoped resolution + isolation tests in its task.
- Export endpoints (G-05): none exist, so there are no export queries to test yet.
- Per-user brand scope (RB-5): not modelled — any holder of a catalogue/price view permission sees all brands. Needs the
  owner's decision on whether staff are limited to brands.
- C-05 still blocked (A-01 INCOMPLETE; verified catalogue data A-03/RB-2 not supplied).

#### Final browser/regression verification (user-requested, 2026-10-14; verification only)
- Commit `43be033`, clean tree; no code changed.
- Dev environment: running dev API (:8081) is the C-04 build `2594848`, ready on DB v9; it needs a `make dev-backend`
  restart to serve C-06's 422 behaviour. Left untouched. Current build `43be033` on side ports against the dev DB:
  `/ready` ready; headless Chrome desktop + mobile: catalogue deep link → sign-in, client validation, wrong credentials
  → 401 generic message, price/catalogue routes (incl. a brand+model filter) without session → 401, no overflow,
  **0 console errors**. Dev DB identical before/after.
- Isolated stack, full `npx playwright test` (desktop + Pixel 7): **57 passed, 0 failed, 5 skipped** (by design).
  API errors (7 requests) all `context canceled` client aborts.
- `make ci` exit 0; Go 15 packages / 336 tests pass; Vitest 147/147. No bugs found; no fixes.
- Evidence: `docs/evidence/C-06/final-verification.txt`.

#### Final status
COMPLETE — C-06 only. No other task started.

### D-01 — Supplier master

- **Status:** COMPLETE (mechanism; no supplier data, document-type sign-off or grants — OQ-19, OQ-20, RB-5)
- **Started:** 2026-10-14 · **Finished:** 2026-10-14
- **Prompt file:** prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/03_Procurement_PDI/D_01_supplier-master.md
- Git at start: A-01 `7b35af7`, C-06 `43be033` and C-06 verification docs `afafc00` pushed (`dev` = `origin/dev`);
  only the untracked LibreOffice lock file remains (untouched).

#### Dependencies
| Dependency | Verified status | Evidence |
| --- | --- | --- |
| B-05 | COMPLETE | migration framework; migrations at 9 |
| B-07 | COMPLETE | deny-by-default RBAC, `Require/RequireAny` |
| B-08 | COMPLETE | `audit.Record` in transactions |
(Also used: B-09 private attachments for supplier documents; C-01/C-02 brands; C-02 readiness `supplier_link` pending.)

#### Acceptance criteria
- [x] AC1 Supplier schema/API: legal name, trade name, GSTIN when applicable, address, status, contacts, document references
- [x] AC2 Searchable list with status (and brand) filter; detail with contact and document sections
- [x] AC3 Multiple brands per supplier; relationships validated server-side; verification needs evidence; a supplier record is not authorisation
- [x] AC4 Archive preserves history; nothing referenced is hard-deleted; contacts soft-removed; links ended, not deleted
- [x] AC5 Supplier documents via private attachments; download only with supplier + attachment permissions
- [x] AC6 Deny by default; audit of sensitive changes; duplicate/conflict/stale handling
- [x] AC7 UI list/detail/forms with loading/empty/validation/duplicate/network/401/403 states, desktop + mobile
- [x] AC8 Tests (backend, API, permission, multi-brand, duplicate, archive), Playwright, `make ci`


#### Stage 1 — UI
Result: COMPLETE (built against the typed API client; no production mock adapter)
- The backend was built in the same task before the screens, so the UI used the real client from the start; Vitest
  stubs `fetch` in tests only. `ui/src/features/suppliers/`: `api/suppliers.ts`, `hooks/useSuppliers.ts`,
  `lib/labels.ts`, `components/{SupplierDialogs,SupplierBadges,SupplierRelationshipCard}.tsx`, `SupplierPages.tsx`.
- Routes `/procurement/suppliers`, `/procurement/suppliers/:supplierId`; nav "Suppliers" (needs `suppliers.view`,
  no longer "planned"). List: search, status filter, table on desktop / stacked list on phones, "not proof of dealership
  authorisation" note. Detail: identity + verification, contacts (add/edit/remove, removed kept), documents (upload in
  dialog via private attachments, signed download), brand relationships (add, verify with evidence, end), archive with
  confirmation. Loading/empty/validation/duplicate-on-field/network/401/403/404/stale-version states.
- Brand detail: "Supplier relationship" card shows real links to `suppliers.view` holders, otherwise only whether the
  server counts one as current. Tests that used Suppliers as the example planned module now use Purchase Orders.

#### Stage 2 — Backend
Result: COMPLETE
- Migration `000010_create_suppliers` (up/down): permissions `suppliers.view|manage|verify|archive` (no grants);
  `suppliers`, `supplier_contacts`, `supplier_documents`, `supplier_brand_links`; CHECKs (GSTIN, PIN, verified needs
  evidence), composite FKs (evidence must be the same supplier's document), partial unique indexes (case-insensitive
  legal name, GSTIN, one active primary contact, one open link per supplier + brand), trigger `supplier_history_guard`.
- Module `internal/modules/suppliers` (types, validate, service, pgstore, http), 14 routes (API_CONVENTIONS "Suppliers").
  Transactions + row locks + `version` (409 `VERSION_CONFLICT`); audit `supplier.*` without phone/email.
- `attachments.canSee`: owner type `supplier` needs `attachments.view` + `suppliers.view`.
- Catalogue readiness `supplier_link` is a real check (verified current link to a verified supplier).
- Design: ADR-015; open questions OQ-19 (document types), OQ-20 (who verifies / maker-checker).

#### Stage 3 — API integration
Result: COMPLETE
- All 14 endpoints used through TanStack Query; nothing shown as saved before the server confirms; server field
  errors mapped to form fields; cache invalidation of supplier detail, lists and brand readiness/links.
- `make ci` PASS: gofmt, go vet, 175 Go tests (PostgreSQL integration required, 0 skipped), eslint, tsc,
  Vitest 156/156, vite build, Go builds. OpenAPI regenerated (additive).

#### Stage 4 — Browser testing
Result: PASS
- Isolated throwaway stack (PostgreSQL :5449 v10, API :8093, Vite :5175, test-only users, PROPOSED grants on that
  database only; removed afterwards). `e2e/suppliers.e2e.ts` 4/4 (desktop + Pixel 7). Full suite: 59 passed,
  5 skipped by design, 2 failed — the C-03 viewer test still expected the old "supplier master, D-01" placeholder;
  assertion updated to the real card; `e2e/catalogue-ui.e2e.ts` re-run 6/6 PASS.
- Console clean; API log 5xx only client-aborted requests ("context canceled").
- Evidence: `docs/evidence/D-01/verification.txt`, `desktop-supplier-detail.png`, `mobile-suppliers.png`.

#### Final verification (2026-10-23)
Result: PASS — commit `f4034a2` pushed to `origin/dev` (fast-forward `afafc00..f4034a2`, no force).
- Shared dev API :8081 restarted with `make dev-backend` (build `f4034a2`): `/api/v1/health` 200, `/api/v1/ready` 200
  (database up, migrations up). Its log after the run: 0 ERROR, 0 5xx.
- Full Playwright suite on a fresh isolated throwaway stack (PostgreSQL :5449 v10, API :8093, Vite :5175, test-only
  users, PROPOSED grants on that database only; removed afterwards), desktop + Pixel 7: **61 passed, 5 skipped (by
  design), 0 failed**, including the updated C-03 viewer test and both D-01 supplier tests on both viewports. A first
  run had 1 failure (B-09 link-expiry test) because the stack was started without `ATTACHMENT_LINK_TTL=30s`; stack
  recreated with it and the whole suite re-run.
- 5xx `context canceled`: 33 requests (run 2), all GET, latency ≤ 39 ms (timeout is 15 s), each cause
  `context canceled` — client-aborted requests (query aborts on navigation, pages closed). Same pattern in earlier
  tasks and pre-D-01 modules; not a regression, no fix. Logging client aborts as ERROR/5xx is noise only.
- `make ci` exit 0 (Go run via `/snap/go/current/bin` because the `/snap/bin/go` wrapper printed no output in this
  shell): gofmt clean, vet, swagger-check, lint, typecheck, 16 Go packages ok, Vitest 156/156, builds.
  `go test -json`: 175 top-level / 365 incl. subtests pass, 0 skip; all 10 supplier tests + `TestCanSeeLinkedAttachments`
  pass. `git diff --check` clean.
- Shared dev DB before == after: migration 10, one grant (`viewer → access.roles.view` PROPOSED), 1 user-role,
  0 suppliers/contacts/documents/links, 3 brands. Commit adds permission keys only — no grants, seeds or supplier data.
- Evidence: `docs/evidence/D-01/final-verification.txt`.

#### Known issues / pending approvals
- No supplier, contact, document or relationship data supplied — none seeded or invented. Any supplier stays
  `PENDING_VERIFICATION` until someone with `suppliers.verify` records evidence.
- `suppliers.*` permissions have no grants (RB-5); verifier/maker-checker OQ-20; document types OQ-19.
- **The shared dev database is now at migration 10.** The dev API was restarted with this build on 2026-10-23 and
  reports ready. Shared dev permissions were not changed.
- API logs client-aborted requests as ERROR with 500/503 (`context canceled`); cross-cutting log noise, not fixed here.

#### Final status
COMPLETE — D-01 only, verified and pushed (`f4034a2`). D-02 not started.

### D-02 — Purchase order lifecycle

- **Status:** COMPLETE (mechanism; no PO data, grants, approvers or threshold — OQ-21…OQ-24, RB-5)
- **Started:** 2026-10-23 · **Finished:** 2026-10-23
- **Prompt file:** prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/03_Procurement_PDI/D_02_purchase-order-lifecycle.md
- Git at start: D-01 `f4034a2` and its verification docs `6dff525` pushed (`dev` = `origin/dev`), clean tree.
  Order: backend first (new API/schema, prompt execution note), then UI on the real API.

#### Dependencies
| Dependency | Verified status | Evidence |
| --- | --- | --- |
| B-07 | COMPLETE | deny-by-default `Require`, `Access.Has`; access tests pass |
| C-02 | COMPLETE | brand/model/variant with composite brand FKs, lifecycle + brand activation gate; catalogue tests pass |
| C-04 | COMPLETE | `price_items` COST type, approved/immutable rows, supersession; pricing tests pass |
| D-01 | COMPLETE | suppliers + verified `supplier_brand_links`; final verification `6dff525`; supplier tests pass |

#### Acceptance criteria
- [x] AC1 PO number generation, supplier, creator, expected date, line items (brand/model/variant, quantity, agreed cost), status history
- [x] AC2 Transitions validated server-side (DRAFT → SUBMITTED → APPROVED, return to draft, cancel); later states reserved for D-03
- [x] AC3 Brand/model/variant compatibility and archived/cross-brand refusal server-side; supplier verified with a current link to each brand before submit/approve
- [x] AC4 Line costs/prices are snapshots: catalogue COST snapshot never changes after later price approvals; lines frozen after submission
- [x] AC5 Totals and remaining quantities computed by the server; client totals/statuses rejected
- [x] AC6 Unauthorised approvals fail (no permission, own PO); threshold config only with owner sign-off; none configured → every PO needs an approver
- [x] AC7 Concurrency (versions, row locks), duplicate protection (one line per variant, Idempotency-Key), audit, no hard delete
- [x] AC8 UI list/create/edit/detail/line editor/submit/approve/history with warnings (price variance, partial, over-receipt) and all states, desktop + mobile, real API
- [x] AC9 Tests (unit, Go integration, Vitest, Playwright desktop + mobile), `make ci`

#### Stage 1 — UI
Result: COMPLETE (built after the backend, directly on the typed API client; no production mock adapter)
- `ui/src/features/purchasing/`: `api/purchaseOrders.ts` (Zod contracts, 13 calls; create sends an `Idempotency-Key`
  kept for the dialog's lifetime), `hooks/usePurchaseOrders.ts`, `lib/labels.ts` (status/event/warning labels,
  `describePOError`), `components/PurchaseOrderDialogs.tsx` (create/edit header, line editor with brand → model →
  variant from the catalogue API and optional cost list, remove line, submit, approve, return, cancel, approval
  settings), `PurchaseOrderPages.tsx` (list with search/status/brand filters, server totals, received/remaining,
  approval policy note; detail with order/totals cards, server blockers, lines with snapshot/variance/partial/over-receipt
  warnings, status history, actions from the server's `actions`).
- Routes `/procurement/purchase-orders`, `/procurement/purchase-orders/:poId`; nav "Purchase Orders" now needs
  `purchase_orders.view` (no longer planned). Shared: PO check keys added to `checkLabel`; `ActionErrorBox` list keys
  made unique (several blockers can share a key). Tests that used Purchase Orders as the example planned module now use
  Goods Receipts & PDI.
- States: loading, empty, no results, validation (client + server field errors), duplicate, submission/approval
  blocked with the server's checks, self-approval, stale version (reload), invalid state, 401/403 (gate + messages),
  404, network errors; success only after the server confirms (cache replaced by the server's detail).

#### Stage 2 — Backend
Result: COMPLETE
- Migration `000011_create_purchase_orders` (up/down): permissions `purchase_orders.view|manage|approve|cancel|settings.manage`
  (no grants); sequence `purchase_order_number_seq` (`PO-000001` …); `purchase_order_settings` (one row, threshold NULL =
  not configured; a threshold needs a sign-off reference — CHECK); `purchase_orders` (status CHECK incl. later D-03 states,
  submitted/approved/cancelled pairs, approval basis `APPROVER` / `BELOW_THRESHOLD`, CHECK approver ≠ creator/submitter,
  per-user unique idempotency key); `purchase_order_lines` (composite FKs brand/model/variant and price list ↔ brand,
  one line per variant, quantity 1–10000, unit cost > 0, COST snapshot columns); `purchase_order_events` (append-only,
  ordered by identity `seq`). Trigger `purchase_order_history_guard`: no PO delete, events immutable, lines change only
  while DRAFT (afterwards only `received_quantity`, for D-03).
- Module `internal/modules/purchasing` (types, validate, rules, service, pgstore, http): 13 routes (see API_CONVENTIONS
  "Purchase orders"). Submit/approve lock the PO, share-lock supplier/brand/model/variant and settings, re-check
  prerequisites (supplier VERIFIED, current VERIFIED link per brand, brand ACTIVE + AUTHORISED, model/variant ACTIVE,
  ≥1 line, expected date not past). Self-approval 403 `SELF_APPROVAL_NOT_ALLOWED`, blocked 409 `SUBMISSION_BLOCKED` /
  `APPROVAL_BLOCKED`; refused approvals audited DENIED. Catalogue cost snapshots visible only with
  `catalogue.cost_price.view`. New error code `SUBMISSION_BLOCKED`.
- Tests: `rules_test.go` (transitions, prerequisites, approval requirement, warnings, action hints), `http_test.go`
  (deny by default per route, input validation, no DB), `api_integration_test.go` (numbering/totals/remaining, client
  totals refused, cross-brand/archived/duplicates, snapshots unchanged after supersession + redaction + DB freeze,
  transitions/self-approval/stale/approval re-check/cancel/history/DB guards, return-to-draft and edit, threshold
  settings incl. first-time supplier, concurrent approvals, idempotent parallel creates, parallel line adds, deny by default).
- `go vet` clean, gofmt clean; full `go test` with PostgreSQL: all packages pass after applying migration 11 to the dev DB
  (`TestDevelopmentSchemaIsCurrent` needs it). OpenAPI regenerated: 10 new paths, existing paths/definitions unchanged.
- Shared dev DB migrated to 11 (schema, 5 permission keys, the empty settings row; no grants, no PO data). Dev API :8081
  restarted with this build: `/ready` ready.

#### Stage 3 — API integration
Result: COMPLETE
- All 13 endpoints used through TanStack Query; supplier, brand/model/variant and cost price lists come from their APIs
  (selects disabled with an explanation when the user lacks `suppliers.view` / `catalogue.view`).
- Vitest 20 files / 165 tests (9 new: deny without view, list totals/policy/filters, create validation + one
  idempotency key across a failed and a successful attempt, blockers + SUBMISSION_BLOCKED, visible snapshot + variance,
  line cascade + client validation + duplicate, approve only after the server confirms, self-approval + stale version,
  not found). `make ci` exit 0 (Go run via `/snap/go/current/bin`); Go 17 packages / 190 top-level / 380 tests pass.

#### Stage 4 — Browser testing
Result: PASS
- Isolated throwaway stack (PostgreSQL :5449 v11, API :8093, Vite :5175, test-only users, PROPOSED grants and an
  eligible PO fixture inserted by SQL on that database only; removed afterwards). `e2e/purchase-orders.e2e.ts`
  6/6 (desktop + Pixel 7) after one test-locator fix (mobile matched the hidden desktop cell; the page was right).
  Full suite: **67 passed, 5 skipped (by design), 0 failed**.
- Console clean; API 5xx only client-aborted GETs (`context canceled`, ≤ 17 ms). Dev DB permissions/data unchanged.
- Evidence: `docs/evidence/D-02/verification.txt`, `desktop-po-detail.png`, `mobile-po-list.png`.

#### Files changed
Backend: `migrations/000011_create_purchase_orders.{up,down}.sql`, `internal/modules/purchasing/*` (new),
`internal/modules/access/policy.go`, `internal/httpx/response.go`, `internal/server/server.go` (+ test), `cmd/api/main.go`,
`docs/openapi/*`. UI: `ui/src/features/purchasing/*` (new), `ui/src/app/{routes,navigation}.ts(x)`,
`ui/src/features/access/api/access.ts`, `ui/src/features/catalogue/{lib/presentation.ts,components/ActionErrorBox.tsx}`,
tests (`App`, `AppShell`, `access`), `ui/e2e/{purchase-orders,access,shell}.e2e.ts`. Docs: ADR-016, OQ-21…OQ-24,
API_CONVENTIONS "Purchase orders", DEVELOPMENT_SETUP e2e grants, this log, `docs/evidence/D-02/`.

#### Known issues / pending approvals
- `purchase_orders.*` permissions have no grants (RB-5). Approvers, maker-checker exceptions and the threshold are
  pending (OQ-21): no threshold → every PO needs an approver who did not create or submit it.
- **No PO can be submitted in a real database yet:** submission needs an activated brand, and brand activation still
  has pending checks (warranty terms RB-2, identifier rules D-04/OQ-4). The approval flow was verified only with a
  test fixture inserted into the throwaway database.
- PO number format (OQ-22), variance tolerance (OQ-23), delivery location (OQ-24) and tax treatment (F-10) pending.
  Amounts exclude tax. Receiving (`PARTIALLY_RECEIVED`, `RECEIVED`, over-receipt override) is D-03.
- The shared dev DB is now at migration 11; dev API restarted on this build.

#### Final status
COMPLETE — D-02 only. Not pushed. D-03 not started.
Pushed later on instruction (fast-forward, no force): `origin/dev` = `4a5a9c0` (2026-10-23).

### D-03 — Goods receipt draft workflow

- **Status:** COMPLETE (draft mechanism; no receipt data or grants; identifier rules, document types and over-receipt
  policy pending — OQ-4, OQ-25…OQ-27, RB-5)
- **Started:** 2026-10-23 · **Finished:** 2026-10-23
- **Prompt file:** prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/03_Procurement_PDI/D_03_goods-receipt-draft-workflow.md
- Git at start: D-02 `4a5a9c0` pushed (`dev` = `origin/dev`), clean tree. Order: backend first (new API/schema), then UI
  on the real API.

#### Dependencies
| Dependency | Verified status | Evidence |
| --- | --- | --- |
| D-02 | COMPLETE | approved POs, `purchase_order_lines.received_quantity` reserved for receipts; purchasing tests pass |
| B-09 | COMPLETE | private attachments, owner linking once, signed links, `canSee` fail-closed; attachment tests pass |
| C-01 | COMPLETE | brand → model → variant schema; catalogue tests pass |

#### Acceptance criteria
- [x] AC1 Draft GRN linked to an approved PO (`APPROVED`/`PARTIALLY_RECEIVED`), numbered, idempotent create, no future date
- [x] AC2 Partial deliveries: per PO line received / accepted / rejected, discrepancy note; several drafts per PO
- [x] AC3 Remaining quantity validated server-side against posted quantities and other open drafts, under a PO row lock
- [x] AC4 Over-receipt needs `goods_receipts.over_receipt` and a reason (else 409/403, audited DENIED); authorisation stored
- [x] AC5 Per-unit capture (condition, chassis/frame, motor, battery, charger, note); duplicate chassis refused per receipt
  and across open/posted receipts and shown per unit; missing identifiers and repeated serials shown as warnings
- [x] AC6 Supplier documents via the private attachment service; access needs `attachments.view` + `goods_receipts.view`
- [x] AC7 Drafts change no stock: no vehicle unit/movement, PO received quantities, status and version untouched (tested)
- [x] AC8 Audit, no hard delete, invalid transitions refused, stale versions refused (DB history guard)
- [x] AC9 UI: list, PO chooser, stepper (lines → units → documents → review), Save draft separate from (unavailable) Post
  receipt, all states, desktop + mobile, real API
- [x] AC10 Tests: Go unit + PostgreSQL integration, Vitest, Playwright desktop + mobile, `make ci`

#### Stage 1 — UI
Result: COMPLETE (built after the backend, directly on the typed API client; no production mock adapter)
- `ui/src/features/receiving/`: `api/goodsReceipts.ts` (Zod contracts, 7 calls; create sends an `Idempotency-Key` kept for
  the dialog), `hooks/useGoodsReceipts.ts`, `lib/labels.ts` (labels, server field path → form path, `describeGRNError`),
  `components/ReceiptDialogs.tsx` (new receipt with receivable PO chooser, cancel with reason, attach document with
  upload), `components/ReceiptEditor.tsx` (stepper, React Hook Form + Zod: accepted = received − rejected, units with
  scanner-style focus, immediate duplicate-chassis check, over-receipt warning and reason field only for authorised
  users, review with disabled "Post receipt"; saves against the version it loaded and offers "Reload latest version";
  entries kept on every refusal), `GoodsReceiptPages.tsx` (list, detail, read-only view, documents, history, and a
  "Goods receipts" card with "Receive goods" on the PO page).
- Routes `/procurement/goods-receipts`, `/procurement/goods-receipts/:grnId`; nav "Goods Receipts" needs
  `goods_receipts.view` (no longer planned); dashboard shortcut marked available. Tests that used Goods Receipts as the
  example planned module now use Stock Movements.
- States: loading, empty, no results, validation, over-receipt (409/403), duplicates per unit, stale version, invalid
  state (PO closed), 401/403, 404, network errors, attachment upload/download failures; success only after the server
  confirms.

#### Stage 2 — Backend
Result: COMPLETE
- Migration `000012_create_goods_receipts` (up/down): permissions `goods_receipts.view|manage|over_receipt` (no grants);
  sequence `GRN-000001`…; `goods_receipts`, `goods_receipt_lines`, `goods_receipt_units`, `goods_receipt_documents`,
  `goods_receipt_events`; `UNIQUE (id, purchase_order_id)` added on `purchase_order_lines` for the composite FK; trigger
  `goods_receipt_history_guard`. `POSTED` + posting columns reserved for D-07.
- Module `internal/modules/receiving` (types, validate, rules, service, pgstore, http): 7 routes (API_CONVENTIONS
  "Goods receipts"). Attachments `canSee` gains owner `goods_receipt`. New codes `OVER_RECEIPT_REQUIRES_OVERRIDE`,
  `OVER_RECEIPT_NOT_AUTHORISED`.
- Tests: `rules_test.go`, `http_test.go` (deny by default per route, CSRF, validation, no delete/post route),
  `api_integration_test.go` (partial draft changes no PO/stock state, other drafts + posted counted, override refused /
  forbidden + DENIED audit / authorised / kept / dropped, cancel frees quantity, duplicate chassis across drafts + DB
  constraints, concurrent saves cannot both take the last units, idempotent create, states, history guard, document
  linking and attachment authorisation through the real attachment service), attachments `cansee_test.go`.
- gofmt/vet clean. OpenAPI regenerated: 5 new paths, 17 new schemas, existing ones unchanged.
- Shared dev DB migrated to 12 (schema + 3 permission keys only; no grants, no receipts; snapshot before/after). Dev API
  :8081 restarted on this build: `/ready` ready.

#### Stage 3 — API integration
Result: COMPLETE
- All 7 endpoints through TanStack Query; receipt mutations refresh receipts, receivable POs and purchase orders.
- Vitest 21 files / 178 tests (13 new). Go 18 packages / 204 top-level / 394 tests pass (PostgreSQL). `make ci` exit 0.

#### Stage 4 — Browser testing
Result: PASS
- Isolated throwaway stack (PostgreSQL :5451 at migration 12, API :8095, Vite :5177, test-only users, PROPOSED grants and
  the eligible PO fixture on that database only). `e2e/goods-receipts.e2e.ts` 4/4 (desktop + Pixel 7) after one
  test-locator fix (search not yet filtered on mobile). Full suite: **71 passed, 5 skipped (by design), 0 failed**.
- Console clean; API 5xx only client-aborted GETs (`context canceled`, 67, all GET). Dev DB unchanged.
- Evidence: `docs/evidence/D-03/verification.txt`, `desktop-grn-draft.png`, `mobile-grn-draft.png`.

#### Files changed
Backend: `migrations/000012_create_goods_receipts.{up,down}.sql`, `internal/modules/receiving/*` (new),
`internal/modules/attachments/{service.go,cansee_test.go,http_test.go}`, `internal/modules/access/policy.go`,
`internal/httpx/response.go`, `internal/server/server.go` (+ test), `cmd/api/main.go`, `docs/openapi/*`.
UI: `ui/src/features/receiving/*` (new), `ui/src/app/{routes,navigation}.ts(x)`, `ui/src/features/access/api/access.ts`,
`ui/src/features/purchasing/PurchaseOrderPages.tsx`, `ui/src/features/dashboard/DashboardPage.tsx`, tests (`App`,
`AppShell`, `access`), `ui/e2e/{goods-receipts,access,shell}.e2e.ts`. Docs: ADR-017, OQ-25…OQ-27, API_CONVENTIONS
"Goods receipts", DEVELOPMENT_SETUP e2e grants, this log, `docs/evidence/D-03/`.

#### Known issues / pending approvals
- `goods_receipts.*` have no grants (RB-5). Who may authorise over-receipt, whether rejected units count, and
  maker-checker are pending (OQ-27).
- Identifier rules per brand (formats, mandatory serials, motor/battery uniqueness, separators) are pending (OQ-4,
  OQ-26): identifiers are optional in drafts; checks against registered vehicles come with D-04/D-05 and posting (D-07).
- Document type list is a proposal (OQ-25). Receipts have no location yet (OQ-24).
- No receipt can be made in a real database yet: it needs an approved PO, which needs an activated brand (D-02 note).
- Posting, PDI and stock are D-06/D-07/E-01; nothing here changes stock.

#### Final status
COMPLETE — D-03 only. Not pushed. D-04 not started.

### REGRESSION-AUDIT-2026-10-26 — Full end-to-end regression audit (user-requested; not a registry task)

- **Status:** COMPLETE · **Date:** 2026-10-26 · Base `e962ee6` (D-03), clean tree. No product task started; D-04 not started.
- **Scope:** every implemented feature B-01…D-03 (shell, auth, RBAC, audit, attachments, catalogue, isolation, pricing,
  suppliers, purchase orders, goods receipt drafts) on desktop 1366×900, Pixel 7 and 320 px.
- **Environment:** isolated throwaway stack only — postgres:18-alpine `:5460` (migrated to 12 + seeds), API built from this tree
  on `:8099`, Vite `:5181`; test-only users, random password never printed; PROPOSED grants and the eligible-PO SQL
  fixture on that database only. Shared dev stack (`:5173`/`:8081`/`:5436`) not touched. Everything removed afterwards.

#### Results
- Existing Playwright suite before fixes: 71 passed, 5 skipped (viewport-specific by design), 0 failed.
- Exploratory: 74-operation deny-by-default sweep (anonymous 401 / no grant 403 / no CSRF 403 — all OK); input robustness
  (no 5xx); auth probes (generic errors, throttle + Retry-After, cookie flags, session rotation on re-login); concurrency
  (1 of 5 parallel PATCHes wins); audit append-only (UPDATE/DELETE/TRUNCATE refused); price overlap 409, cost prices
  hidden from selling-only users; double-submit (one POST); keyboard and offline behaviour; pagination and search;
  300-page crawl (roles × viewports) for headings, overflow, labels, console and 5xx.
- Defects fixed (details, root causes and evidence: `docs/evidence/REGRESSION-AUDIT-2026-10-26/report.md`):
  1. Denied PO/supplier/GRN detail pages lost their heading and breadcrumbs — `Gate` `header` prop (Low).
  2. PO line cost-variance warning overflowed by 23 px at 320 px — wrapping warning badge (Low).
  4. Dialog focus fell to `<body>` on close instead of the opener (WCAG 2.4.3) — `Dialog` restores focus (Medium).
  5. Offline saves paused silently with Cancel disabled and were sent later on reconnect — mutations
     `networkMode: 'always'`, failing at once with the existing network error, entries kept (Medium).
  6. A page past the end showed "2451–43 of 43" — `Pagination` explains it and leads back (Low).
  7. Status board showed D-02/D-03 NOT STARTED — rows corrected (docs).
  Each code fix has a regression test that failed before the fix (Vitest ×7 new, e2e ×1 new + 2 extended).
- After fixes: Playwright **73 passed, 5 skipped, 0 failed**; crawl 300 loads, 0 flagged; Vitest 23 files / 185 tests;
  Go 18 packages, 394 tests (204 top-level) pass with PostgreSQL required; `make ci` exit 0; `git diff --check` clean.
  `golangci-lint` not installed (skipped by the Makefile); no automated axe scanner available.
- API 5xx in the run: 251, all client-aborted GETs (`context canceled`), no failed write — as in earlier evidence.
- Re-run on a fresh throwaway stack (user request): Playwright 73 passed, 5 skipped, 0 failed; crawl 300 loads, 0 flagged;
  focus-return and offline checks PASS; 245 API 5xx, all client-cancelled GETs. No new defects (report §11).
- Pre-D-05 re-run (user request; HEAD `ba7ca84` + the uncommitted fixes; throwaway stack at migration 13 with a SQL
  stand-in for posting): Playwright 73 passed, 5 skipped, 0 failed; crawl 29 routes × 6 personas × 3 viewports = 522
  loads, 0 flagged; 84/84 operations deny by default; D-04 API smoke 31/31 (no D-04 screen exists); stock invariant held
  (drafts created no vehicle unit, PO quantities unchanged); Vitest 206, Go 410, `make ci` exit 0. No new defects
  (`report-pre-d05-addendum.md`).

#### Not changed (observations / pending)
- "Approved by User xxxxxxxx" on the PO card (no name lookup API yet — documented); 5xx logging of client aborts; JSON
  responses without nosniff/frame headers (H-04); size limit wording in some 400s. Business blockers unchanged (RB-2,
  RB-5, OQ-4, OQ-15, OQ-22…OQ-27, A-01 evidence, C-05).

#### Files changed
`ui/src/components/ui/{Dialog,Pagination}.tsx` (+ tests), `ui/src/app/queryClient.ts` (+ new test),
`ui/src/features/{purchasing,suppliers,receiving}/*Pages.tsx` (+ tests), `ui/e2e/{purchase-orders,suppliers}.e2e.ts`,
this log, `docs/evidence/REGRESSION-AUDIT-2026-10-26/`. No backend, migration, OpenAPI or seed change. Not committed or pushed.

### D-04 — Vehicle unit registry

- **Status:** COMPLETE — backend phase (`25a5578`) and frontend phase (typed API client, hooks, permission keys; the
  vehicle screens are D-08). Run as two user-requested sessions.
  Mechanism only: no unit, identifier or grant exists anywhere; unit number format, cost basis, battery packs,
  identifier rules and location pending (OQ-4, OQ-24, OQ-26, OQ-28…OQ-30, RB-5).
- **Started:** 2026-10-26 · **Finished:** 2026-10-26
- **Prompt file:** prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/03_Procurement_PDI/D_04_vehicle-unit-registry.md
- Git at start: `dev` at `e962ee6` (D-03, local, not pushed; `origin/dev` = `4a5a9c0`); working tree held the
  uncommitted regression-audit fixes, kept out of D-04.

#### Dependencies
| Dependency | Verified status | Evidence |
| --- | --- | --- |
| B-05 | COMPLETE | golang-migrate baseline, `dbtest` scratch databases, migration round-trip test; database tests pass |
| C-01 | COMPLETE | brand → model → variant with `models (id, brand_id)`, `variants (id, brand_id)`, `variants (id, model_id)` keys; catalogue tests pass |
| (used) D-01…D-03, B-07, B-08 | COMPLETE | suppliers, POs, goods receipt units as source; location scope; `audit.Record` |

#### Acceptance criteria
- [x] AC1 Each vehicle is an individual unit: `vehicle_units` row, internal `unit_number`, variant/brand/model, status
- [x] AC2 Identifier validation follows approved rules only: normalisation trim + upper case (INV-002, separators kept —
  OQ-26); chassis required after `RECEIVED_PENDING_PDI` (INV-002 CHECK); nothing mandatory before; no uniqueness (D-05)
- [x] AC3 Unit history links to source documents: supplier/PO/line/GRN/line/captured unit references (FK-consistent),
  `vehicle_unit_events` naming the GRN
- [x] AC4 Cost protected server-side: `vehicle_units.cost.view`, null + `costHidden` otherwise, never in audit summaries
- [x] AC5 API contract and schema documented (ADR-018, API_CONVENTIONS "Vehicle units", OpenAPI) and tested
- [x] AC6 No fake data or success: no seed, no create route; units only from a POSTED receipt's accepted unit (deferred DB check)

#### Stage 1 — UI
Result: COMPLETE (frontend phase, Prompt B scope: types and API hooks; no screen — D-08 builds the list/detail).
- No existing screen needs vehicle units: D-03 drafts hold *captured* units, which are not registered vehicles; units
  appear only after posting (D-07). So no page, form or speculative component was added.
- `ui/src/features/vehicles/api/vehicleUnits.ts`: Zod contracts (`VehicleUnit`, `VehicleUnitDetail` with `source` and
  `events`, page), the 13 statuses as in OpenAPI, `costHidden` refinement (a hidden cost never carries an amount),
  `listVehicleUnits`, `getVehicleUnit`; read-only, no write call. `hooks/useVehicleUnits.ts`: `vehicleUnitKeys`,
  `useVehicleUnitList`, `useVehicleUnit`, `useVehicleUnitPerms`. `lib/presentation.ts`: `unitCostText` (hidden →
  "Recorded (not visible to you)", never ₹0; amount only when sent and permitted) and
  `describeVehicleUnitReadError` (401/403/404/422/network → safe states; 404 never reveals an out-of-scope unit).
- `Permission.VehicleUnitView` / `VehicleUnitCostView` added to the access contract.
- Vitest: `vehicles.test.tsx` 21 tests (parameters, keys, hooks, no retry/fallback on 403/404/422, no write call,
  cost display, error states, permission keys, OpenAPI contract). Lint, Prettier, typecheck clean.

#### Stage 2 — Backend
Result: COMPLETE
- Migration `000013_create_vehicle_units` (up/down): permissions `vehicle_units.view`, `vehicle_units.cost.view` (no
  grants); sequence `MAE-VEH-000001`…; composite unique keys on `purchase_orders (id, supplier_id)`,
  `purchase_order_lines (id, variant_id)`, `goods_receipt_lines (id, purchase_order_line_id)`,
  `goods_receipt_units (id, goods_receipt_line_id)`; `vehicle_units` (INV-005 status CHECK, generated normalised
  identifiers, composite FKs, `UNIQUE goods_receipt_unit_id`, lookup indexes); `vehicle_unit_events`; guard trigger (no
  UPDATE/DELETE until E-01/D-05); deferred constraint trigger (receipt POSTED, unit ACCEPTED, same received date).
- Module `internal/modules/vehicles` (types, validate, service, pgstore, http): `GET /vehicle-units`,
  `GET /vehicle-units/{id}`; `RegisterFromReceiptUnit` + `CommitError` for D-07. Wired in `cmd/api`, `internal/server`.
- Tests: 16 new (rules, HTTP without DB, PostgreSQL integration); receiving draft test now asserts no vehicle unit.
- gofmt/vet clean. OpenAPI regenerated: 2 paths and 5 schemas added, nothing existing changed.
- Shared dev DB **not** migrated (still at 12; dev API untouched). To apply: `make db-migrate-up` then restart the API.

#### Stage 3 — API integration
Result: COMPLETE (frontend phase) — the typed client calls the real routes; no mock adapter, no production fixture.
- `GET /vehicle-units` and `GET /vehicle-units/{id}` through `apiRequest` + Zod; hooks with stable keys (empty
  parameters dropped), `keepPreviousData` for paging; refused reads (403/404/422) are not retried (queryClient
  defaults) and nothing falls back to a wider request. No mutation exists.
- Contract test compares the Zod fixtures field for field and the status enum with the committed OpenAPI
  (`docs/openapi/swagger.json`). Not exercised against a live D-04 API: no screen consumes it yet.
- Checks: `make ci` exit 0 (throwaway DB). Go 19 packages, 410 tests (220 top-level) pass, 0 skip; Vitest 23 / 185.

#### Stage 4 — Browser testing
Result: NOT APPLICABLE — neither phase adds or changes a screen (types, API calls and hooks only; Prompt B scope).
Browser tests come with the first screen that uses these hooks (D-08). No Playwright run was made for D-04.

#### Files changed
`migrations/000013_create_vehicle_units.{up,down}.sql`, `internal/modules/vehicles/*` (new),
`internal/modules/access/policy.go`, `internal/modules/receiving/api_integration_test.go`, `internal/server/server{,_test}.go`,
`cmd/api/main.go`, `docs/openapi/*`, `docs/{ARCHITECTURE_DECISIONS,API_CONVENTIONS}.md` (ADR-018, OQ-28…OQ-30), this log,
`docs/evidence/D-04/verification.txt`.
Frontend phase: `ui/src/features/vehicles/{api/vehicleUnits.ts,hooks/useVehicleUnits.ts,lib/presentation.ts,vehicles.test.tsx}`
(new), `ui/src/features/access/api/access.ts`, `docs/API_CONVENTIONS.md` (client line), this log,
`docs/evidence/D-04/frontend-verification.txt`.

#### Known issues / pending approvals
- No grants for `vehicle_units.*` (RB-5). No unit can exist yet: units come only from receipt posting (D-07), which
  needs an approved PO (activated brand pending).
- Chassis uniqueness, motor/battery policies and the conflict error are D-05; the lookup indexes are non-unique until then.
- Status/location changes and their movements are E-01 (guard refuses all updates meanwhile); PDI fields D-06; list
  ageing/colour/timeline D-08/E-02; opening stock source G-07.
- Shared dev DB still at migration 12: the D-04 API answers only after `make db-migrate-up` and an API restart there.

#### Final status
COMPLETE — D-04 backend (`25a5578`) and frontend phase committed locally, not pushed. D-05 not started.

### D-05 — Duplicate vehicle identifier controls

- **Status:** IN PROGRESS — backend phase COMPLETE; frontend phase COMPLETE (2026-10-27). Live `DUPLICATE_IDENTIFIER`
  integration is blocked until a route returns it (receipt posting, D-07).
  Chassis/frame only: motor/battery/charger policies are not approved (OQ-4, OQ-30, INV-003/004); separators OQ-26.
- **Started:** 2026-10-26
- **Prompt file:** prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/03_Procurement_PDI/D_05_duplicate-vehicle-identifier-controls.md
- Git at start: `dev` at `ba7ca84` (local ahead of `origin/dev` = `4a5a9c0` by 3); uncommitted regression-audit fixes kept
  out of D-05. Prior audit evidence (`report-pre-d05-addendum.md`): no open defect in the registration path.

#### Dependencies
| Dependency | Verified status | Evidence |
| --- | --- | --- |
| D-04 | COMPLETE | `vehicle_units` with generated `chassis_normalised`, `RegisterFromReceiptUnit`; vehicles tests pass |

#### Acceptance criteria
- [x] AC1 Duplicate chassis cannot be saved: unique index `vehicle_units_chassis_key` (normalised, all brands) + service check
- [x] AC2 Including concurrent attempts: tested with PostgreSQL (2-way interleaved race and 8 parallel: exactly one unit)
- [x] AC3 Friendly validation maps to the field: `DuplicateIdentifierError{Field: chassisNumber}` → 409 `DUPLICATE_IDENTIFIER`
  with `details[].field` (request path); no data about the existing unit
- [x] AC4 No bypass: no write route; direct SQL refused by the index; normalised column not writable
- [ ] Motor/battery policies — not approved, not enforced (documented, ADR-019)

#### Stage 1 — UI
Result: NOT APPLICABLE in this phase (backend-only session by instruction).

#### Stage 2 — Backend
Result: COMPLETE
- Migration `000014_vehicle_unit_identifier_uniqueness` (up/down): duplicate pre-check (fails without changing data),
  unique `vehicle_units_chassis_key` replaces the lookup index; down restores it.
- `internal/modules/vehicles`: early checks (same captured unit → `ErrAlreadyRegistered`; chassis → duplicate) and
  mapping of exactly `vehicle_units_chassis_key`; `WriteDuplicateIdentifier` for the posting route (D-07);
  `httpx.CodeDuplicateIdentifier`.
- Tests: 7 new (vehicles 23 total). `make ci` exit 0; Go 417 tests (227 top-level) pass; OpenAPI unchanged.
- Shared dev DB not migrated (still 12; would need 13 and 14).

#### Stage 3 — API integration
Result: NOT APPLICABLE in this phase (no route returns the conflict until D-07; frontend phase pending).

#### Stage 4 — Browser testing
Result: NOT APPLICABLE in this phase (no UI change).

#### Files changed
`migrations/000014_vehicle_unit_identifier_uniqueness.{up,down}.sql`, `internal/modules/vehicles/{types,pgstore,http}.go`,
`internal/modules/vehicles/{api_integration_test,duplicate_integration_test,http_test}.go`, `internal/httpx/response.go`,
`docs/{ARCHITECTURE_DECISIONS,API_CONVENTIONS}.md` (ADR-019, OQ-4), this log, `docs/evidence/D-05/verification.txt`.

#### Known issues / pending approvals
- Motor/battery/charger uniqueness and its exception route (INV-003/004, OQ-4, OQ-30); separators (OQ-26).
- D-03 drafts already refuse a chassis on another open/posted receipt; opening stock (G-07) will rely on the index.

#### Frontend phase (2026-10-27; base `2679542`)
- **UI:** `normaliseChassis` / `VehicleErrorCode` / identifier field names in `ui/src/features/vehicles/lib/identifiers.ts`;
  the receipt editor's in-draft check uses it. `describeGRNError` maps 409 `DUPLICATE_IDENTIFIER` to the unit's
  identifier field only when the path names an identifier of an existing line/unit (range-checked `isField`), otherwise a
  general message; generic field text, entries kept, no retry, no "saved", other 409s unchanged. No link to the existing
  unit: the answer carries no identifier of it and there is no unit detail route (D-08).
- **Integration:** no route returns the conflict yet (no posting route; vehicle unit API read-only) → live check BLOCKED
  BY D-07. No new endpoint, no mock success, no duplicate-check call.
- **Checks:** Prettier, ESLint, tsc, build PASS; Vitest 26 files / 232 tests (+26: `identifiers.test.ts`,
  `duplicateIdentifier.test.tsx`, API responses mocked); `make ci` exit 0 on a throwaway DB; golangci-lint not installed.
- **Browser:** headed Chrome on a throwaway stack (migration 14): `goods-receipts.e2e.ts` 4 passed (desktop + mobile);
  afterwards 0 vehicle units, PO received quantities unchanged. Live duplicate-identifier flow NOT RUN (no route).
- Files: `ui/src/features/vehicles/lib/identifiers.ts`, `ui/src/features/vehicles/identifiers.test.ts`,
  `ui/src/features/receiving/{lib/labels.ts,components/ReceiptEditor.tsx,duplicateIdentifier.test.tsx}`,
  `docs/API_CONVENTIONS.md`, this log, `docs/evidence/D-05/frontend-verification.txt`. Shared dev DB still 12, untouched.

#### Browser test in the user's Chrome (2026-10-27)
Throwaway stack (DB :5465 migration 14, API :8098 from this tree, Vite :5183), e2e.catalogue. Receipt editor:
in-draft case/space repeat blocked before sending (0 PUT); separator variant kept distinct and saved (PUT 200);
chassis on another draft (other case) -> one PUT 409 mapped to the unit field, entries kept, no retry; corrected value
saved and persisted after reload. No console errors; vehicle_units 0; only 5xx = 2 client-aborted GETs. Live
DUPLICATE_IDENTIFIER still NOT RUN (needs D-07). Evidence: `docs/evidence/D-05/browser-chrome-2026-10-27.txt`.

#### Final status
IN PROGRESS — D-05 backend and frontend phases complete, committed locally (not pushed). Remaining: exercise the
conflict against the real posting route once D-07 exists.

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
