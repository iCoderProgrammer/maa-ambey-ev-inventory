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
- B-05 (2026-10-12): migrations and seeds are embedded (`migrations.FS`, `seeds.FS`), so
  `bin/migrate` / `bin/seed` are self-contained and the API knows the schema version it was built
  for (`migrations.Latest()`, which also enforces the naming/pairing policy). `/ready` reports
  `migrations: down` (503) when the schema is unmigrated, behind, ahead or dirty; the API still
  starts (logs a warning) so operators can see the reason, but never migrates. `cmd/seed` refuses a
  non-current schema. With `APP_ENV=staging|production`, `down` and `force` are refused unless
  `--confirm` (`make db-migrate-down CONFIRM=1`) is passed. There is no "drop all" command.

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
  (Models/variants/colours/specs: done in C-01, see ADR-011.)

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

## ADR-007 — Role/permission architecture (built in B-07)

- **Status:** Accepted (implemented 2026-10-12; grants pending owner sign-off). The five roles are confirmed (Owner, Inventory Manager, Sales Executive,
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
- B-06 added users, sessions and the deny-by-default session middleware (ADR-008).
- B-07 as built: tables `permissions` (catalogue, 15 keys, mirrored by `internal/modules/access`
  constants and checked by a test), `roles` (5 seeded identities), `role_permissions` (`PROPOSED` or
  `APPROVED`; approved needs approver + reference + date, DB-enforced), `user_roles`, `locations`
  (identity only; no seed — showroom details pending), `user_location_scopes`. `locations.all` is a
  permission. Effective access = APPROVED grants of the user's roles (+ PROPOSED only when
  `RBAC_ALLOW_PROPOSED_GRANTS=true`, accepted only with `APP_ENV=development`). Grants/roles/scope change only via
  `cmd/access`, audited (ADR-009). `GET /me` and `GET /access/roles`; `Require(permission)` per route;
  `Scope.Check/Restrict` for object and list level. Brand scope per user is still pending (RB-5) and
  not modelled.

## ADR-008 — Authentication: local accounts and server-side sessions (B-06)

- **Status:** Accepted (OQ-6 approved by the user on 2026-10-12 when starting B-06).
- **Credentials:** username + password; argon2id (19 MiB, t=2, p=1, PHC string with its own
  parameters so they can be raised later). Policy: 12–128 characters, not the username. No
  composition rules, no expiry (NIST SP 800-63B). Concurrent hashing capped at 4.
- **Sessions:** `user_sessions` rows with SHA-256 of a 256-bit random token, per-session CSRF token,
  idle + absolute expiry, revocation reason, IP and user agent. Never deleted (history); revoked
  instead. A user with session history cannot be deleted (FK RESTRICT); disable them instead.
- **Cookie:** HttpOnly, SameSite=Strict, Path=/, no Max-Age (ends with the browser), `Secure` +
  `__Host-` prefix outside development/test; staging/production refuse `SESSION_COOKIE_SECURE=false`.
- **CSRF:** SameSite=Strict plus a synchronizer token header on every mutation; login requires
  `application/json` (cannot be sent cross-site without a CORS preflight) and CORS rejects unknown origins.
- **Deny by default:** the session middleware protects every `/api/v1` route not explicitly public.
- **Throttling:** in-memory failure windows per username+IP and per IP (single instance, like the
  API rate limiter). A multi-instance deployment needs a shared store — revisit with OQ-5.
- **Enumeration:** unknown user and wrong password return the same error and take similar time
  (dummy hash). `ACCOUNT_DISABLED` is only shown after a correct password.
- **No built-in admin:** accounts are created with `cmd/user` (`make user ARGS=...`) by someone
  with database access. Passwords are never accepted as command-line flags.
- **Deferred:** audit events for sign-in/out go to the structured log until B-08 adds the audit table;
  self-service password reset, MFA and user-management screens are not in B-06.

## ADR-009 — Append-only audit events (B-08)

- **Status:** Accepted.
- One table `audit_events` (bigint identity, actor type/user/label, action, outcome, entity, reference,
  request ID, JSONB summary). `BEFORE UPDATE/DELETE/TRUNCATE` triggers raise `42501`, so no normal
  application path can rewrite history; only a migration rollback (refused in staging/production without
  `--confirm`) or a DBA can remove it.
- Actor types: `USER` (signed-in), `CLI` (operator command), `SYSTEM` (job) and `ANONYMOUS` (added in
  migration 000005: unauthenticated attempts such as failed sign-ins, which must not be attributed to the
  targeted account).
- Events are written inside the business transaction (`audit.Record(ctx, tx, …)`); an invalid event aborts
  the change. Failed/refused sign-ins have no data change and are written on their own.
- Secrets are blocked by key/value rules in `audit.Record` (fail closed). IP addresses are not stored in
  audit events (sessions keep the sign-in IP); an IP-retention policy is still open (Architecture §5).
- No read endpoint or UI yet: reading needs a permission key (B-07). The Audit Log screen has no explicit
  registry task — flagged in the execution log.
- Order: B-08 was done before B-07 by user decision (registry cycle), so B-07 audits role changes from the start.
- Registry fix (2026-10-12, at the user's request): `TASK_REGISTRY.csv` (both identical copies) now lists
  B-08's dependencies as `B-04, B-05, B-06` instead of `B-04, B-05, B-07`. B-08 needs `users` (audit
  `actor_user_id`) from B-06, not RBAC; B-07 keeps `B-06, B-08`. The prompt-pack Markdown (`B_08_*.md`,
  `ALL_TASK_PROMPTS.md`, `*_v1.2.md`) still says B-07 — approved prompt text, not edited; the CSV is
  the source of truth. A full-registry check found one other cycle, **D-08 ↔ E-02** (D-08 depends on E-02,
  E-02 on D-08), to be resolved before either starts (OQ-8).

## ADR-010 — Private attachments (B-09)

- **Status:** Accepted (limits and storage provider pending; see OQ-9, OQ-10).
- **Storage:** an `ObjectStore` interface; the only driver today is `local` — a private directory
  (`ATTACHMENT_STORAGE_DIR`, created `0700`, files `0600`, outside any web root; dev/test default
  `var/attachments`, git-ignored; staging/production must set it). Keys are server-generated
  `yyyy/mm/<uuid>`, never derived from the file name; writes go to a temp file and are linked into place.
  An S3-compatible driver (Architecture §2 "private object storage") waits for the hosting decision (OQ-5).
- **Metadata:** `attachments` (migration 000006): object key, driver, detected content type, size, SHA-256,
  sanitised display name, uploader, optional location, optional owner (`owner_type`/`owner_id`, set once by
  a later module), created at. Triggers refuse DELETE/TRUNCATE and any change except linking an unlinked
  row once. No hard delete (product invariant).
- **Validation (server, fail closed):** content type is sniffed from the bytes (`http.DetectContentType`);
  allow-list JPEG, PNG, WebP, PDF (also a DB CHECK); the multipart part's declared type and the file-name
  extension must match the detected type; size limit `ATTACHMENT_MAX_BYTES` (default 10 MB, max 25 MB),
  enforced while streaming, so an oversized upload never lands. These defaults are proposals (OQ-9).
- **Authorisation:** catalogue keys `attachments.upload` and `attachments.view` (no grants; RB-5).
  Object check (`canSee`): view permission + location scope (when set) + an unlinked attachment is visible
  to its uploader only; linked attachments are denied until the owning module adds its check. Attachments
  the caller cannot see return 404.
- **Access:** no public URL ever. `GET /attachments/{id}/link` returns a link signed with HMAC-SHA256
  (`ATTACHMENT_SIGNING_KEY`, required in staging/production) over attachment, user, disposition and expiry
  (`ATTACHMENT_LINK_TTL`, default 5 min). `GET …/content` needs the same user's session, re-checks access
  (revocation is immediate), audits `attachment.downloaded`, and serves with `nosniff`, a sandboxing CSP,
  `Cache-Control: private, no-store`. Inline display only for images.
- **Audit:** `attachment.uploaded` (in the metadata transaction; summary: type, size, location — not the
  file name) and `attachment.downloaded`.
- **Deferred:** malware scanning (Architecture §8 "if available"), retention/archiving, image re-encoding
  or EXIF stripping, linking to business records (D-03, D-06, F-09 …).

## ADR-011 — Catalogue schema: brand → model → variant (C-01)

- **Status:** Accepted (schema only; API/validation in C-02, verified data in C-05).
- Migration `000007_create_catalogue`: `models` (brand FK), `variants` (model), `variant_colours`,
  `variant_specs`; brands gain `manufacturer_legal_name`, `authorisation_evidence_id` (private attachment,
  B-09) and `verified_by`. Nothing is seeded; the three roster brands stay `PENDING_VERIFICATION` / `UNVERIFIED`.
- **Brand context by construction:** every child row stores `brand_id`, and composite foreign keys
  (`variants(model_id, brand_id) → models(id, brand_id)`, colours/specs → `variants(id, brand_id)`) make a
  cross-brand mix impossible and stop a model with variants from moving to another brand (BR-003, BR-008).
- **Uniqueness:** model `internal_code` per brand (upper-case format), model/variant `manufacturer_code` per
  brand (case-insensitive, when present), model name per brand, `variant_code` and variant name per model,
  colour name/code per variant, spec `(variant, key, effective_from)`. IDs are global UUIDs.
- **Provenance on every catalogue row (BR-004):** `confidence` UNVERIFIED / SUPPLIER_CONFIRMED /
  CLIENT_APPROVED, `source_url` (http/https only) and/or `source_document_id`, `source_date`, `verified_by`,
  `verified_at`. Above UNVERIFIED the source, date and verifier are mandatory (DB CHECK). Brands:
  `AUTHORISED` requires legal name, reference, verifier and date (BR-002).
- **Specifications are key/value rows** (`spec_key` lower-case identifier, text *or* numeric value, unit,
  `effective_from`) rather than typed columns, because the verified attribute list per brand is not yet
  known (RB-2); battery type/capacity etc. are keys. Verified rows are immutable (trigger, `42501`):
  changes are new rows with a later date (BR-005). Unverified drafts stay editable.
- **No hard delete of used catalogue items (BR-006):** all FKs are `ON DELETE RESTRICT` (SQLSTATE 23001);
  archive via `lifecycle_status`. Vehicle units (D-04) will add their own RESTRICT FK to variants.
- **Left to the service (C-02):** only ACTIVE + AUTHORISED brands and ACTIVE models/variants may be used for
  purchase/sales; parent/child lifecycle consistency; who may verify/approve (OQ-2, RB-5); audit events.
- **Not in C-01:** prices (C-04), warranty terms (C-04/RB-2), serial/identifier rules per brand/model (D-04,
  OQ-4), supplier links (D-01), vehicle-type value set (OQ-11).
- **Rollback:** the down migration drops catalogue tables and the new brand columns (data loss; refused in
  staging/production without `--confirm`); brands are kept.

## ADR-012 — Catalogue API, lifecycle and activation gate (C-02)

- **Status:** Accepted (technical design; business approvals still pending — RB-5, OQ-1, OQ-2, OQ-12–OQ-14).
- **Layout:** `internal/modules/catalogue` — `http.go` (Gin, swag) → `service.go` (permission re-check, validation)
  → `pgstore.go` (pgx; each mutation is one transaction that locks the item `FOR UPDATE` and the parent it depends
  on `FOR SHARE`, checks `version`, applies the pure rules in `rules.go`, writes and records the audit event).
- **Permissions** (migration 000008, catalogue keys only, no grants): `catalogue.view`, `.manage`, `.verify`,
  `.activate`, `.archive`, `catalogue.brand.authorise`, `catalogue.brand.activate`. Separate keys so the owner can
  split data entry, verification and commercial activation when the matrix is approved. Brand scope per user is
  still not modelled (RB-5).
- **Brand context:** a child's brand is always taken from its parent inside the transaction; a client `brandId`
  is only compared (422 on mismatch). Unknown body fields are refused, so items never move between brands/models.
- **Lifecycle:** create → `PENDING_VERIFICATION`/`UNVERIFIED`. Verify (pending items only) records confidence,
  source URL/document, source date; the caller is the verifier. Activate needs verification and an ACTIVE parent
  (model: brand not archived — brand activation comes last because it needs an active catalogue). Edits only while
  pending; editing a verified item resets it to `UNVERIFIED` (the verification covered the old values). ACTIVE items
  are not edited (BR-005: approved data is not overwritten) — they can only be archived. Archive never deletes and
  never cascades: children keep status and references (BR-006, BR-008); nothing new is added under an archived parent.
- **Brand activation gate (BR-002, fail closed):** readiness checks `lifecycle`, `authorisation` (AUTHORISED),
  `legal_identity`, `authorisation_evidence` (reference + private evidence attachment + verifier + date),
  `verified_catalogue` (≥1 ACTIVE variant under an ACTIVE model) plus **pending** checks `supplier_link` (D-01),
  `warranty_terms` (RB-2), `identifier_rules` (D-04, OQ-4) — `approved_prices` became a real check in C-04 (ADR-013). Pending checks are never met,
  so **no brand can be activated until those tasks add their checks**. This is deliberate: activation without that
  evidence would contradict BR-002.
- **Operational use (for later tasks):** `Brand.operational` = ACTIVE + AUTHORISED. Purchase/sales (D-02, D-03, F-*)
  must require the whole chain — operational brand, ACTIVE model, ACTIVE variant (and colour when chosen) — since
  archive does not cascade.
- **Audit:** create/update/authorisation/verify/activate/archive in the same transaction; refused activations as
  `DENIED` with the unmet check keys; archive reason kept in the summary.
- **Not in C-02:** spec value endpoints (`variant_specs`; to be scheduled with C-03/C-05), un-archive or
  correction of ACTIVE items (OQ-12), screens (C-03), prices (C-04).

## ADR-013 — Effective-dated price history (C-04)

- **Status:** Accepted (technical design; price types, backdating, maker-checker and tax treatment pending — OQ-15…OQ-18).
- **Schema (migration 000009):** `price_lists` (one brand, name unique per brand, currency fixed to INR, ACTIVE/ARCHIVED)
  and `price_items` (list, brand, model, variant via composite FKs so brand/model/variant cannot be mixed; `price_type`
  SELLING/COST; `amount numeric(12,2) > 0`; `effective_from`, exclusive `effective_to`; status DRAFT/APPROVED/CANCELLED;
  `created_by`, `approved_by`, `approved_at`, `approval_reference`, `source_note`, `supersedes_id`, `superseded_by_id`,
  `version`). `variants` gains `UNIQUE (id, model_id)` for the variant↔model FK. Extension `btree_gist` (trusted).
- **Immutability:** trigger `price_items_protect_history` refuses DELETE always, any change to CANCELLED rows, and any
  change to APPROVED rows except **controlled supersession**: `superseded_by_id` set once and `effective_to` moved earlier
  (42501). CHECKs: approved ⇔ approver + time; approver ≠ drafter; tax reference must be NULL.
- **No overlap:** exclusion constraint `price_items_no_overlap` (list, variant, type, `tstzrange(from, to)`) for APPROVED
  rows, so exactly one price is valid at any instant and concurrent approvals cannot both succeed (23P01 → 409
  `PRICE_OVERLAP`); the service also checks first to name the conflicting price.
- **Approval rules (service, in one transaction with row locks):** draft only; never the drafter (403
  `SELF_APPROVAL_NOT_ALLOWED`); list active, brand/model/variant not archived, variant verified (409
  `APPROVAL_BLOCKED`); **no retroactive start** — `effective_from` ≥ approval time minus 5 minutes clock grace (422);
  supersession only of an approved, not yet superseded price of the same list/variant/type that the new start overlaps;
  the predecessor's `effective_to` becomes the new start. Refusals are audited as DENIED.
- **Lookup:** `GET /prices/lookup?priceListId&variantId&priceType&at` returns the approved price whose `[from, to)`
  contains `at` (404 `PRICE_NOT_FOUND` otherwise). Approved rows never change, so the answer for a past instant never
  changes: quotations (F-02) snapshot it (SAL-003). Archived lists still answer for past instants.
- **Permissions (deny by default, no grants):** new `catalogue.price.view`, `catalogue.price.manage`,
  `catalogue.price.approve`; COST uses the existing `catalogue.cost_price.view` / `catalogue.cost_price.edit` (SEC-003).
  Lists show only types the caller may see; a cost price is 404 to selling-only users. `access.Handler.RequireAny`
  added for routes open to either permission family.
- **Tax configuration:** `tax_configuration_id` is part of the contract but must be NULL until an accountant-approved
  tax configuration exists (F-10, DOC-003); a later migration adds the FK and drops the CHECK. No rate is stored.
- **Catalogue readiness:** `approved_prices` is now a real check (approved SELLING price valid now for an ACTIVE variant
  under an ACTIVE model); brand activation stays blocked by the remaining pending checks (supplier, warranty, identifiers).
- **Not in C-04:** quotations and their snapshots (F-02), discounts (F-03), tax configuration (F-10), price imports.
- **Rollback:** down migration drops price tables (destructive; `--confirm` in staging/production) and refuses while a
  role holds a `catalogue.price.*` grant.

---

## Open questions (need client / accountant sign-off)

| # | Question | Blocks |
| --- | --- | --- |
| OQ-1 | Final value set for `brands.authorisation_status`. Current proposal: `UNVERIFIED`, `AUTHORISED`, `NOT_AUTHORISED`, `REVOKED`. Change via a new migration if needed. | C-01, C-02 |
| OQ-2 | Which user role is the "approval owner" for brand activation (BR-002)? An `approved_by` FK is added once `users` exists (B-06). | C-02 |
| OQ-3 | Launch roster confirmed (Lectrix EV, Ceeon India, Ujagar Electric; seeded). Authorisation evidence per brand still outstanding. | C-02, C-05 |
| OQ-4 | Are motor / battery / charger serials mandatory, and is battery-swap history needed (Architecture §12.3, INV-003/004)? | D-04 |
| OQ-5 | Deployment provider, API host, DB region, backup RPO/RTO (Architecture §12). | H-06, H-10 |
| ~~OQ-6~~ | Session strategy — **resolved 2026-10-12**: local accounts + server-side sessions in an HttpOnly cookie (ADR-008). Session lifetimes (2h idle / 12h absolute) and the password policy are proposals the owner may change via config. | — |
| OQ-8 | Registry cycle D-08 (vehicle list/detail UI) ↔ E-02 (stock movement timeline UI): which goes first, or should D-08 only link to the timeline once E-02 exists? | D-08, E-02 |
| OQ-9 | Attachment limits: accepted types (proposal JPEG, PNG, WebP, PDF) and per-file size (proposal 10 MB). Configurable size; type list in code + DB CHECK. | B-09 (defaults only) |
| OQ-10 | Production attachment storage: S3-compatible provider/bucket, region, encryption, retention. Until then only the local private-directory driver exists. | H-06, deploy |
| OQ-11 | Value set for `models.vehicle_type` (e.g. scooter / motorcycle classes) — free upper-case identifier until confirmed. | C-02, C-05 |
| OQ-12 | How are ACTIVE catalogue items corrected (typo in an approved name/code) and may archived items be restored? C-02 allows neither: active items can only be archived (BR-005). | C-03, C-05 |
| OQ-13 | Separation of duties in the catalogue: may the same person create, verify and activate an item, or record and use a brand's authorisation? C-02 has separate permissions but no maker-checker rule. | C-02, RB-5 |
| OQ-14 | Who may record `CLIENT_APPROVED` confidence (owner only?) versus `SUPPLIER_CONFIRMED`? C-02 lets any holder of `catalogue.verify` record either. | C-02, OQ-2 |
| OQ-15 | Price types: is `SELLING` (customer price, e.g. ex-showroom) + `COST` (supplier cost) enough, or are on-road, scheme or offer prices separate types? What does the selling amount include (GST, registration, insurance)? | C-04, F-02, accountant |
| OQ-16 | May an approved price ever be backdated (e.g. correcting yesterday's price)? C-04 refuses retroactive starts (5-minute clock grace). | C-04, owner |
| OQ-17 | Maker-checker for prices: C-04 never lets the drafter approve their own price. Is that the rule for a small team, or may the owner self-approve? Who approves (owner only?) and is there an amount threshold? | C-04, RB-5 |
| OQ-18 | Tax treatment of price amounts and the tax configuration a price must reference (HSN, rate set, inclusive/exclusive) — needs the accountant's configuration (F-10, DOC-003). Until then the reference must stay empty. | C-04, F-10 |
| OQ-7 | Are invoices legally issued by this app or by external accounting software? | F-10 |
