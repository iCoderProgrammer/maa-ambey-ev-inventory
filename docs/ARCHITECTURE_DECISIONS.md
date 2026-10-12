# Architecture Decisions

Decision log for the Maa Ambey Enterprises Multi-Brand EV Inventory Management System.
Approved business documents live in `documentation/` (PRD, System Architecture, Business Rules,
Design Specification, Implementation Tasks) and are **not** modified here. This file records
implementation decisions and open questions.

---

## ADR-001 — Go + Gin backend and React + Vite frontend (supersedes Next.js / NestJS)

- **Status:** Accepted — 2026-10-12
- **Context:** `02_System_Architecture` §2 recommends Next.js + NestJS + Prisma. The client has
  since selected React + Vite for the web UI and Go + Gin for the API (see prompt-pack README).
- **Decision:**
  | Layer | Choice |
  | --- | --- |
  | Web UI | React 19 + TypeScript (strict) + Vite, React Router, Tailwind CSS v4 |
  | UI server state | TanStack Query |
  | UI client state | Redux Toolkit — shell UI state only (theme, sidebar) |
  | Forms | React Hook Form + Zod |
  | API | Go + Gin REST, `/api/v1`, OpenAPI via swaggo |
  | DB access | PostgreSQL + pgx v5 (parameterised SQL, no ORM) |
  | Migrations | golang-migrate (library, wrapped by `cmd/migrate`) |
  | Logging | `log/slog` JSON |
- **Unchanged from the approved architecture:** modular monolith, PostgreSQL as source of truth,
  server-side RBAC, append-only stock ledger, transactional sale finalisation, error envelope
  `{code, message, details, requestId}`, `/api/v1` base path.
- **Not used:** Next.js, NestJS, Express or any Node.js API; Prisma/GORM.

## ADR-002 — Single repository, Go at root and React in `ui/`

- **Status:** Accepted — 2026-10-12
- **Context:** The setup prompt sketches `frontend/` + `backend/`. The team's existing Go + React
  monorepos (`saral-sampark`, `mann-ki-baat`) keep the Go module at the repository root and the
  React app in `ui/`. The client asked for the same layout.
- **Decision:** One Git repository. Go module at the root (`cmd/`, `internal/`, `migrations/`,
  `docs/openapi/`); React app in `ui/`. Root `Makefile` drives both.
- **Consequence:** Prompt-pack references to `/backend/...` map to the repo root and `/frontend/...`
  maps to `ui/`.
- **Module boundary (B-01):** `go.mod` declares `ignore ./ui` (Go 1.25+ directive). Without it
  `go test ./...` and `go vet ./...` pick up Go sources shipped inside `ui/node_modules`
  (e.g. `flatted/golang`). No second package workspace is used: `ui/` is a single npm package.

## ADR-003 — Migrations are explicit; the API never migrates or resets the database

- **Status:** Accepted
- `cmd/migrate` (`make db-migrate-up|down|version`) is the only path that changes schema.
  The API only connects and pings at startup and fails fast with a clear error.
- `down` rolls back one step by default (`STEPS=N` to override). Never run `down` against
  production without a verified backup and explicit approval (Architecture §11).
- Migration files are sequential (`000001_name.up.sql` / `.down.sql`).

## ADR-004 — Error and success envelopes

- Success: `{"data": ...}`. Error: `{"error": {"code","message","requestId","details"}}`.
- `/ready` returns the `data` envelope with HTTP 503 when a dependency is down so probes and the
  UI can see which check failed. Failure reasons are logged, never returned.
- See `docs/API_CONVENTIONS.md`.

## ADR-005 — Baseline abuse protection

- Per-client-IP token bucket on `/api/v1` (`RATE_LIMIT_RPS`, `RATE_LIMIT_BURST`). In-memory, so
  it is per instance; add a gateway/load-balancer limit for multi-instance deployments.
- Gin trusts `X-Forwarded-For` only from `TRUSTED_PROXIES` (default: none), so clients cannot spoof
  their IP to evade limits.
- Login-specific throttling is part of B-06.

## ADR-006 — Multi-brand foundation: `brands` table only in setup

- **Status:** Accepted (setup scope)
- Multi-brand support is P0 (Research Brief D-01, BR-001…BR-008). Brands are data; no brand name,
  enum or constant appears in application code.
- **Seeds (v2, client-confirmed):** `seeds/0001_initial_brands.sql` inserts Lectrix EV, Ceeon India
  and Ujagar Electric with `ON CONFLICT DO NOTHING`. `make db-seed` (`cmd/seed`) runs all
  `seeds/*.sql` in one transaction, and they are safe to re-run. The integration test
  `TestSeedsAreIdempotent` checks this. Seeds are separate from migrations so reference data can
  evolve without schema versions.
- Seeded brands stay `PENDING_VERIFICATION` / `UNVERIFIED` because BR-002 requires authorisation
  evidence before `ACTIVE`, and that evidence is not fabricated. See REQUIREMENTS_BASELINE RB-1 and RB-7.
- Migration `000001_create_brands` creates `brands` with: UUID id, `name` (unique,
  case-insensitive), `slug` (unique, kebab-case), `lifecycle_status`
  (`PENDING_VERIFICATION` default / `ACTIVE` / `ARCHIVED` — BR-001), separate
  `authorisation_status` (default `UNVERIFIED`), `authorisation_reference`, `verified_at`,
  `version` (optimistic concurrency) and timestamps with an `updated_at` trigger.
- **Lifecycle vs. authorisation are deliberately separate** (BR-002, BR-008): a brand can be in the
  catalogue without being authorised for purchase/sale. The rule "only ACTIVE + authorised brands
  may be used operationally" will be enforced in the service layer in C-02, not by a DB constraint,
  so that history and pending setup remain representable.
- Models, variants, suppliers, `brand_supplier_links`, vehicle units and price lists are deferred
  to backlog tasks C-01, C-04, D-01 and D-04, where their prompts specify fields and constraints.

### Proposed schema for later tasks (not yet implemented)

| Table | Key points (from Architecture §5 / Business Rules) |
| --- | --- |
| `models` | `brand_id` FK NOT NULL; unique `(brand_id, internal_code)` (BR-003) |
| `variants` | `model_id` FK; unique `(model_id, variant_code)`; spec source/confidence (BR-004) |
| `suppliers` | legal name, GSTIN if applicable, status |
| `brand_supplier_links` | `(brand_id, supplier_id)`, authorisation status, effective dates, evidence, verifier |
| `vehicle_units` | `variant_id`, normalised `chassis_number` globally unique (INV-002), status (INV-005), `location_id` |
| `price_list_items` | brand/variant context, `effective_from/to`, approver; never overwritten (BR-005) |

Brand-scoped access: every brand-filtered query will take the caller's permitted brand set from
the server-side session (B-07) and intersect it with any client-supplied `brandId`; a client
`brandId` is never trusted on its own.

## ADR-007 — Role/permission architecture (design only; built in B-06/B-07)

- **Status:** Proposed. The five roles are confirmed (Owner, Inventory Manager, Sales Executive,
  Accountant, Viewer); the permission matrix is pending (REQUIREMENTS_BASELINE §5).
- **Model:** `roles`, `permissions` (stable keys such as `catalogue.brand.activate` or
  `stock.adjustment.approve`), `role_permissions`, `user_roles`, plus optional user brand and
  location scope tables. Roles map to permission keys in data, so changing the matrix needs no
  code change.
- **Enforcement:** an auth middleware resolves the session into a principal (user, permission
  set, brand/location scope). Each route declares a required permission through a
  `RequirePermission(key)` middleware, deny by default. Services re-check object-level rules such as
  brand scope, self-approval (LED-006) and state. Repository queries always filter by the
  principal's permitted brand IDs, intersected with any client `brandId`.
- **UI:** may hide actions using permissions returned by `/me`, but never relies on that.
- Not implemented now: no users or sessions exist yet, so a guard would have nothing to check.

---

## Open questions (need client / accountant sign-off)

| # | Question | Blocks |
| --- | --- | --- |
| OQ-1 | Final value set for `brands.authorisation_status`. Current proposal: `UNVERIFIED`, `AUTHORISED`, `NOT_AUTHORISED`, `REVOKED`. Change via a new migration if needed. | C-01, C-02 |
| OQ-2 | Which user role is the "approval owner" for brand activation (BR-002)? An `approved_by` FK is added once `users` exists (B-06). | C-02 |
| OQ-3 | Launch roster confirmed (Lectrix EV, Ceeon India, Ujagar Electric; seeded). Authorisation evidence per brand still outstanding. | C-02, C-05 |
| OQ-4 | Are motor / battery / charger serials mandatory, and is battery-swap history needed (Architecture §12.3, INV-003/004)? | D-04 |
| OQ-5 | Deployment provider, API host, DB region, backup RPO/RTO (Architecture §12). | H-06, H-10 |
| OQ-6 | Session strategy: server-side session with HttpOnly cookie is the documented preference (SEC-006). Confirm before B-06. | B-06 |
| OQ-7 | Are invoices legally issued by this app or by external accounting software? | F-10 |
