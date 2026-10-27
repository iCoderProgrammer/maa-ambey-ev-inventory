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

## ADR-014 — Brand isolation and brand-specific rule resolution (C-06)

- **Status:** Accepted for what exists; supplier, warranty, identifier rules, exports and per-user brand scope pending.
- **Principle:** brands are data. Every brand-specific value is resolved by IDs that carry their brand (composite FKs),
  never by brand name. A test (`internal/isolation/nocode_test.go`) scans Go, SQL and UI source (tests excluded) for
  the seeded brand names (read from the seed, so the test hardcodes none) and the name "Saffron"; any hit fails.
- **Cross-brand mismatches fail (422)** at every entry point:
  - writes: child under another brand's parent, client `brandId` differing from the parent's, moving a model/variant,
    a price for a variant of another brand than its list, supersession across lists (C-02, C-04);
  - **filters (new in C-06):** `GET /variants` with `brandId` + another brand's `modelId`, `GET /colours` with
    `brandId` + another brand's `variantId`, `GET /price-items` with `brandId` / `priceListId` / `variantId` of
    different brands — previously these returned an empty list; an unknown referenced ID is also 422;
  - **price lookup (new in C-06):** a variant of another brand than the list (or an unknown list/variant) is 422 instead
    of "no price";
  - database: composite FKs refuse cross-brand rows even without the API (23503).
- **Scoped results:** brand filters return only that brand's rows; readiness of a brand reads only its own variants and
  prices; archiving one brand changes no other; cost prices stay behind cost permissions in every brand.
- **Rule resolution today:** price = approved item of (list, variant, type) valid at an instant (ADR-013); activation
  readiness = generic checks over the brand's own rows. Both are configured by data per brand/variant.
- **Limitations (not built, not fabricated):** supplier mappings (`brand_supplier_links`, D-01), warranty terms (RB-2),
  vehicle identifier rules (D-04, OQ-4), export endpoints (G-05) and per-user brand scope (RB-5: whether users are
  limited to some brands). Each must resolve through brand/model/variant IDs like prices do and get isolation tests in
  its own task; until then the readiness checklist keeps them pending.
- **UI:** the Variants list drops a model filter that belongs to another brand (stale or edited link) before calling the
  API; model/variant pickers list only the chosen brand's models; price data is requested with the record's own brand.

## ADR-015 — Supplier master and supplier ↔ brand relationships (D-01)

- **Status:** Accepted (technical design; document types, grants and who verifies are pending — OQ-19, OQ-20, RB-5).
- **Schema (migration 000010):** `suppliers` (legal name unique case-insensitively, optional trade name, optional GSTIN —
  format CHECK + unique, address, PIN CHECK, status `PENDING_VERIFICATION` / `VERIFIED` / `ARCHIVED`, verification
  reference + evidence document + verifier + time; a CHECK makes `VERIFIED` impossible without evidence),
  `supplier_contacts` (removed = `active=false`, at most one active primary, phone or email required),
  `supplier_documents` (links a private attachment to the supplier; type `GST_REGISTRATION` / `AUTHORISATION` /
  `AGREEMENT` / `OTHER`, immutable), `supplier_brand_links` (status `UNVERIFIED` / `VERIFIED` / `ENDED`, effective
  dates, authorisation reference, evidence document of the **same** supplier via composite FK, at most one open link
  per supplier + brand). Trigger `supplier_history_guard` refuses deletes and changes to ended links and documents.
  This is the `brand_supplier_links` table planned in ADR-006 (named `supplier_brand_links`).
- **A supplier record is not dealership authorisation.** New suppliers and links start pending/unverified; only a
  holder of `suppliers.verify` can verify, and only with an uploaded document and a reference. A link can be verified
  only when the supplier is `VERIFIED`. Changing the legal name or GSTIN of a verified supplier returns it to
  `PENDING_VERIFICATION`. No supplier, contact, document or relationship is seeded.
- **Archive, not delete:** archived suppliers are read-only (`INVALID_STATE`); their contacts, documents and links stay
  as history, and their links no longer count for readiness (the supplier is no longer `VERIFIED`). Contacts are
  removed (kept), links ended with a reason (kept). Every mutation is audited (`supplier.*`) without phone/email.
- **Documents** use the private attachment service (ADR-010): the caller uploads (needs `attachments.upload`), then
  links their own, not-yet-linked upload; downloads go through short-lived signed links. Attachments with owner type
  `supplier` are visible to users with `attachments.view` **and** `suppliers.view`.
- **Catalogue readiness:** `supplier_link` is now a real check — a `VERIFIED` link, current by its dates, to a
  `VERIFIED` supplier. Pending checks left: `warranty_terms`, `identifier_rules`.
- **Permissions:** `suppliers.view`, `suppliers.manage`, `suppliers.verify`, `suppliers.archive` exist with **no
  grants** (deny by default). Concurrency: `version` on suppliers, contacts and links (409 `VERSION_CONFLICT`).

## ADR-016 — Purchase order lifecycle (D-02)

- **Status:** Accepted (technical design; approvers, threshold, number format, variance tolerance and grants pending —
  OQ-21…OQ-24, RB-5).
- **Schema (migration 000011):** `purchase_orders` (number `PO-000001…` from `purchase_order_number_seq`, supplier,
  status, expected date, creator, submitter, approval basis/approver/note, threshold applied, cancellation, `version`,
  per-user idempotency key), `purchase_order_lines` (brand/model/variant via composite FKs, one line per variant,
  quantity, agreed `unit_cost`, `received_quantity` for D-03, catalogue COST snapshot: list, item, amount, time),
  `purchase_order_events` (append-only status history), `purchase_order_settings` (one row; threshold NULL).
  Trigger `purchase_order_history_guard`: POs never deleted, history immutable, lines change only while `DRAFT`
  (afterwards only `received_quantity`). CHECKs: approver ≠ creator/submitter, threshold needs a sign-off reference,
  approved states carry a basis.
- **Lifecycle:** `DRAFT → SUBMITTED → APPROVED`; `SUBMITTED → DRAFT` (return, reason); cancel from `DRAFT`, `SUBMITTED`
  or `APPROVED` with nothing received. `PARTIALLY_RECEIVED`, `RECEIVED`, `CLOSED` exist for goods receipts (D-03).
- **Eligibility (submit and approve, re-checked in the transaction with share locks):** supplier `VERIFIED` with a
  current `VERIFIED` relationship to every line's brand; each brand `ACTIVE` + `AUTHORISED` (C-02 activation gate);
  model and variant `ACTIVE`; ≥ 1 line; expected date set and not past. Drafts may use pending (not archived)
  suppliers and catalogue items, so work can start before verification; nothing past a draft does. Because brand
  activation still has pending checks (warranty terms, identifier rules), no PO can be submitted in a real database yet.
- **Approval:** `purchase_orders.approve`, never the creator or submitter (maker-checker, like C-04 prices).
  PO-001's threshold is configuration recorded only with the owner's sign-off reference; none is set, so every PO
  needs an approver. With a threshold, a PO within it (and not a supplier's first, when enabled) is approved on
  submission with basis `BELOW_THRESHOLD`; the threshold used is stored on the PO.
- **Snapshots:** the agreed unit cost is the PO's own value; a named cost price list snapshots its approved `COST`
  price at that moment, and later price approvals never change it. Totals and remaining quantities are computed from
  the lines on every read; client totals/statuses are not accepted.
- **Concurrency/duplicates:** row lock + `version` (409 `VERSION_CONFLICT`); unique variant per PO; idempotent create.
- **Permissions:** `purchase_orders.view|manage|approve|cancel|settings.manage` with **no grants**.

## ADR-017 — Goods receipt drafts (D-03)

- **Status:** Accepted (technical design; identifier rules, document types, over-receipt basis and grants pending —
  OQ-4, OQ-25…OQ-27, RB-5).
- **Schema (migration 000012):** `goods_receipts` (number `GRN-000001…`, PO, status `DRAFT`/`POSTED`/`CANCELLED`,
  received date, supplier reference, notes, creator, cancellation, posting columns for D-07, `version`, per-user
  idempotency key), `goods_receipt_lines` (one per PO line, composite FKs keep it on the receipt's own PO; received =
  accepted + rejected; discrepancy note; over-receipt authorisation: quantity covered, reason, who, when),
  `goods_receipt_units` (one row per physical unit: condition, chassis/frame number + normalised form, motor, battery,
  charger serials, note; unique chassis per receipt), `goods_receipt_documents` (private attachments, owner type
  `goods_receipt`), `goods_receipt_events`. Trigger `goods_receipt_history_guard`: nothing deleted except lines/units of a
  draft; documents and events immutable; a non-draft receipt's lines never change.
- **Drafts change no stock.** No vehicle unit, movement, PO `received_quantity` or PO status is touched; posting (D-07)
  does that in one transaction. `POSTED` exists in the schema; no D-03 route sets it.
- **Quantities and concurrency:** a save locks the receipt (version) and then its PO (`FOR UPDATE`), so saves of
  drafts of one PO are serialised. Available = ordered − posted − held by other open drafts. Receiving more needs
  `goods_receipts.over_receipt` and a reason (PO-003); the authorisation is stored per line, covers the quantity it was
  given for and is dropped when no longer needed. Without it the save is refused and nothing is stored.
- **Identifiers:** per-brand rules are not approved (OQ-4, `identifier_rules` pending), so all identifiers are optional
  in a draft; the API shows missing ones as warnings. A chassis/frame number is compared trimmed and upper-cased
  (INV-002; separators kept — OQ-26) and refused when already on this receipt or another open/posted receipt
  (transaction advisory lock per number). Motor/battery/charger repeats only warn. Registered vehicles are checked at
  posting (D-04/D-05/D-07).
- **Permissions:** `goods_receipts.view|manage|over_receipt` with **no grants**. Linked documents are visible with
  `attachments.view` + `goods_receipts.view` (attachments `canSee`).

---

## ADR-018 — Vehicle unit registry (D-04)

- **Status:** Accepted (technical design; unit number format, cost basis, battery packs, identifier rules, location and
  grants pending — OQ-4, OQ-24, OQ-26, OQ-28…OQ-30, RB-5).
- **Schema (migration 000013):** `vehicle_units` — one row per physical scooter (INV-001): `unit_number`
  (`MAE-VEH-000001…`, sequence), brand/model/variant via composite FKs (C-06), the 13 approved statuses (INV-005,
  default `RECEIVED_PENDING_PDI`), chassis/frame, motor, battery and charger identifiers as captured with **generated**
  `*_normalised` columns (`upper(btrim(x))`, the same rule as `goods_receipt_units.chassis_normalised`), optional
  `location_id`, `source_type` (`GOODS_RECEIPT` only), supplier/PO/PO line/GRN/GRN line/captured unit references,
  `received_date`, `unit_cost` + `cost_currency` (`INR`), `registered_by`, `version`. `vehicle_unit_events` is the
  append-only history; each entry names its source document (type, id, number).
- **References prove themselves by FK:** the PO is the supplier's, the PO line is the PO's and for this variant, the
  receipt is against the PO, the receipt line is the receipt's and for the PO line, the captured unit is on the receipt
  line (new composite unique keys on `purchase_orders`, `purchase_order_lines`, `goods_receipt_lines`,
  `goods_receipt_units`). One captured unit → at most one vehicle unit (`UNIQUE goods_receipt_unit_id`).
- **Only posted receipts create units:** a deferred constraint trigger checks at commit that the receipt is `POSTED`,
  the captured unit `ACCEPTED` and the received date the receipt's — a draft (D-03) can never have a unit, and posting
  (D-07) may register and mark the receipt posted in either order in one transaction. There is no create route:
  `vehicles.RegisterFromReceiptUnit(ctx, tx, actor, requestID, receiptUnitID, locationID)` is the entry point for D-07,
  reading every value from stored rows (variant, supplier, references, date, identifiers, the PO line's agreed cost),
  writing the `REGISTERED` history entry and the `vehicle_unit.registered` audit event (cost left out) in the caller's
  transaction; `vehicles.CommitError` maps the commit-time checks.
- **Identifier rules (INV-002…INV-004):** a chassis/frame number is required for every status after
  `RECEIVED_PENDING_PDI` (CHECK); nothing is mandatory before that (OQ-4). Separators are kept (OQ-26). No uniqueness
  here: chassis uniqueness, motor/battery policies and the conflict error are D-05, which builds on the normalised
  columns (non-unique lookup indexes for now).
- **Changes:** a guard trigger refuses every UPDATE and DELETE of units and history. Status/location change only through
  the stock movement service (E-01, INV-009) and identifier corrections need an approved flow (D-05); those tasks replace
  the guard. No transition matrix is implemented here.
- **Read API:** `GET /vehicle-units` (search unit number + normalised identifiers; filters status, brand, variant,
  supplier, PO, receipt, location; paging) and `GET /vehicle-units/{id}` (source + history), both `vehicle_units.view`.
  Location scope (B-07) as for attachments: a unit without a location is visible to every viewer; a unit at a location
  needs it in scope (list filtered, detail 404, out-of-scope `locationId` filter 422).
- **Cost:** `vehicle_units.cost.view` (new, separate from `catalogue.cost_price.view`, which covers catalogue price
  lists); without it `unitCost`/`costCurrency` are null and `costHidden` true in list and detail.
- **Permissions:** `vehicle_units.view`, `vehicle_units.cost.view` with **no grants**.
- **Not here:** colour per unit (not captured on receipt), PDI fields (D-06), list filters beyond the above, ageing and
  timeline (D-08/E-02), opening stock source (G-07), battery component history (OQ-30).

---

## ADR-019 — Duplicate vehicle identifier controls (D-05)

- **Status:** Accepted (chassis/frame only; motor/battery/charger policies and separator rules pending — OQ-4, OQ-26,
  OQ-30, INV-003, INV-004).
- **Database (migration 000014):** unique index `vehicle_units_chassis_key` on the generated `chassis_normalised`
  (`upper(btrim(chassis_number))`, migration 000013) replaces the lookup index. Global across brands (INV-002;
  architecture: "chassis/VIN remains globally unique"). NULLs never conflict, so units without a chassis (allowed while
  `RECEIVED_PENDING_PDI`) are fine; blanks are refused by the length CHECK; the normalised column cannot be written.
  The up migration first looks for existing duplicates and stops with their normalised values (no other data), changing
  nothing. Down restores the non-unique lookup index.
- **Service:** `RegisterFromReceiptUnit` first answers `ErrAlreadyRegistered` for a repeat of the same captured unit,
  then checks the chassis with the column's own expression and returns `*DuplicateIdentifierError{Field:
  "chassisNumber", GoodsReceiptUnitID}` (`errors.Is(err, ErrDuplicateIdentifier)`). Two transactions can both pass that
  check; the unique index makes the second wait for the first and fail, and `vehicle_units_chassis_key` (only that
  name) maps to the same error. The caller rolls back, so no history or audit row remains.
- **API contract:** `DUPLICATE_IDENTIFIER`, 409, `details[].field` = request path of the identifier, generic message,
  nothing about the existing vehicle (`vehicles.WriteDuplicateIdentifier`, for D-07's posting route).
- **Not decided, so not enforced:** motor/battery/charger uniqueness (INV-003 asks to define it after brand validation;
  INV-004 makes battery/charger rules per brand/model/variant; no approved policy or policy model exists), which
  separators are ignored (OQ-26), several or swappable battery packs (OQ-30), and the "authorised exception" route for
  duplicate serials (INV-003). Motor/battery/charger lookup indexes stay non-unique.

---

## ADR-020 — Append-only stock movement ledger (E-01)

- **Status:** Accepted (technical design; who may make which move, reversal/correction flow, PDI policy, transfer rules
  and grants pending — OQ-31…OQ-33, RB-5).
- **Schema (migration 000015):** `stock_movements` — one row per change of a vehicle unit's stock state:
  `item_type` (`VEHICLE_UNIT` only until spare parts, E-03), `vehicle_unit_id`, `movement_type`, `quantity_before/after`
  (0/1: a unit is one item), `from/to_status`, `from/to_location_id`, `source_type` (`GOODS_RECEIPT` | `STAFF_ACTION`) +
  `source_id` + `source_reference`, `reverses_movement_id`, `actor_id`, `request_id`, `idempotency_key` +
  `request_hash` (SHA-256 of the operation), `note`, `created_at` (database clock), `seq`.
- **Movement types = approved lifecycle edges with an owner** (System Architecture §6), enforced by CHECK
  `stock_movements_transition_valid` and mirrored by `stock.Transitions`: `RECEIPT` (none → RECEIVED_PENDING_PDI,
  0 → 1, source the unit's own goods receipt), `PDI_PASS` (RECEIVED_PENDING_PDI → AVAILABLE), `QUALITY_HOLD`
  (RECEIVED_PENDING_PDI → QUALITY_HOLD), `REJECT` (QUALITY_HOLD → REJECTED), `DISPLAY_ON` / `DISPLAY_OFF`
  (AVAILABLE ↔ DISPLAY). Every non-receipt movement is 1 → 1 at an unchanged location. **Not implemented** (no
  approved rule or owner yet): RECEIVED_PENDING_PDI → REJECTED and QUALITY_HOLD → AVAILABLE (not in the diagram; PDI
  policy, OQ-31), reservation/sale/delivery edges (F-04/F-07 add their types by migration), SERVICE_HOLD and returns
  (no approved exit/approval), ORDERED/IN_TRANSIT (units are registered at receipt), location transfers (OQ-32),
  reversals (OQ-33: the link column, its same-unit FK and one-reversal-per-movement key exist; a CHECK keeps it empty).
- **The ledger drives the snapshot.** An `AFTER INSERT` trigger on `stock_movements` locks the unit, requires its
  current status/location to equal the movement's "before" (`stock_movements_from_matches_unit`; a RECEIPT must be the
  unit's first movement, from its own goods receipt, at its registered state: `stock_movements_receipt_matches_unit`),
  then sets status, location, `version + 1`, `updated_at`. The D-04 guard on `vehicle_units` now accepts exactly that
  update: issued from inside a trigger (`pg_trigger_depth() > 1`, which no client statement satisfies, even one that sets
  the `maa.stock_movement_unit` marker), for the unit the marker names, changing nothing but those four columns.
  Direct UPDATE/DELETE of units, and UPDATE/DELETE/TRUNCATE of movements, stay `insufficient_privilege` (tested).
- **Reconciliation (LED-003):** a deferred constraint trigger checks at commit that every inserted/updated unit has a
  RECEIPT movement and equals its latest movement (status, location, quantity 1): `vehicle_units_ledger_reconciles`.
  A unit therefore cannot exist or change without its movement, and the chain from → to is continuous.
- **Registration:** `vehicles.RegisterFromReceiptUnit` now writes the unit's RECEIPT movement (`stock.RecordReceipt`,
  key `receipt-unit-<captured unit id>`) in the same transaction; the registration audit event names `movementId`.
  Units registered before 000015 get the same movement backfilled (actor, time, receipt and location of the
  registration); the migration refuses to run if any unit is not RECEIVED_PENDING_PDI.
- **Service:** `stock.MoveUnit(ctx, tx, actor, scope, req)` runs inside the caller's transaction: lock unit → location
  scope (outside = not found) → idempotency lookup (after the lock, so concurrent repeats see the first commit) →
  `ExpectedVersion` → starting status → insert (trigger applies it) → audit `vehicle_unit.moved`. Permissions are the
  caller's (per task). `stock.PGStore.Move` wraps it in its own transaction for the API and resolves a concurrent key
  collision (`errKeyRace` on `stock_movements_idempotency_key`) by rolling back and replaying.
- **Idempotency:** unique `(actor_id, idempotency_key)` in the database; same key + same `request_hash` = replay of the
  first movement (no second movement, audit or unit change); same key + different operation = `IDEMPOTENCY_KEY_REUSED`.
  Refused requests write nothing, so their key stays free.
- **API:** only the moves no other workflow owns: `POST /vehicle-units/{id}/put-on-display` and `/take-off-display`
  (`vehicle_units.display.manage`, new, **no grants**; `version` and `Idempotency-Key` required). PDI moves are reached
  through D-06's routes, receipt through D-07's posting route. No generic movement/status route; list/detail is E-02.
- **D-07 integration:** in one transaction — lock and revalidate the receipt and PO lines; for each accepted captured
  unit call `vehicles.RegisterFromReceiptUnit` (unit + RECEIPT movement + audit); update PO received quantities and
  status; mark the receipt POSTED; commit (the deferred checks verify the receipt is POSTED and every unit reconciles).
  Map errors with `vehicles.CommitError`. Posting idempotency is the receipt's own key/status; a repeated post finds the
  receipt POSTED (and `ErrAlreadyRegistered` per unit). A draft never reaches this path.

---

## Open questions (need client / accountant sign-off)

| # | Question | Blocks |
| --- | --- | --- |
| OQ-1 | Final value set for `brands.authorisation_status`. Current proposal: `UNVERIFIED`, `AUTHORISED`, `NOT_AUTHORISED`, `REVOKED`. Change via a new migration if needed. | C-01, C-02 |
| OQ-2 | Which user role is the "approval owner" for brand activation (BR-002)? An `approved_by` FK is added once `users` exists (B-06). | C-02 |
| OQ-3 | Launch roster confirmed (Lectrix EV, Ceeon India, Ujagar Electric; seeded). Authorisation evidence per brand still outstanding. | C-02, C-05 |
| OQ-4 | Are motor / battery / charger serials mandatory, and is battery-swap history needed (Architecture §12.3, INV-003/004)? Are motor/battery/charger serials unique — globally, per brand or per variant — and who may approve a documented exception (INV-003)? D-05 enforces chassis only. | D-04, D-05 |
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
| OQ-19 | Supplier document types: is `GST_REGISTRATION`, `AUTHORISATION`, `AGREEMENT`, `OTHER` the right set, and which documents are required to verify a supplier or a brand relationship? | D-01, D-02 |
| OQ-20 | Who verifies suppliers and supplier ↔ brand relationships (owner only?), and may the person who recorded a supplier also verify it? D-01 has a separate `suppliers.verify` permission but no maker-checker rule. | D-01, RB-5, OQ-2 |
| OQ-21 | Purchase order approvals: who approves (owner only?), may the creator approve their own PO in a small team (D-02 never allows it), what is the approval threshold, and does a supplier's first PO always need approval (D-02 default: yes)? Until signed off, no threshold is set and every PO needs an approver. | D-02, RB-5, PO-001 |
| OQ-22 | PO number format: D-02 uses `PO-000001` from one sequence (gaps possible). Is a prefix per financial year/location wanted? Not a tax document number. | D-02 |
| OQ-23 | Price variance tolerance for PO lines and supplier bills (PO-007): D-02 warns on any difference from the catalogue cost snapshot; tolerance and any approval requirement are not configured. | D-02, D-03 |
| OQ-24 | Do purchase orders need a delivery location (location scope, B-07)? D-02 has none. | D-02, D-03 |
| OQ-25 | Goods receipt document types: `DELIVERY_CHALLAN`, `SUPPLIER_INVOICE`, `PHOTO`, `OTHER` (D-03 proposal). Which are required before posting? | D-03, D-07 |
| OQ-26 | Chassis/frame normalisation: D-03 trims and upper-cases only. Which separators (spaces, `-`, `/`) may be ignored when comparing (INV-002 "permitted separators"), per brand? | D-03, D-04, D-05 |
| OQ-27 | Over-receipt: D-03 counts the *received* quantity (accepted + rejected) against the PO and lets any holder of `goods_receipts.over_receipt` authorise it, including on their own draft. Should rejected units count, and who may authorise (owner only, maker-checker)? | D-03, D-07, RB-5, PO-003 |
| OQ-28 | Internal vehicle unit number: D-04 uses `MAE-VEH-000001…` after the design specification example (`MAE-VEH-00042`). Confirm prefix, digits and whether it restarts per year or location. | D-04, D-09 |
| OQ-29 | Vehicle unit cost basis (REP-003): D-04 records the PO line's agreed unit cost at registration. Should it be the supplier bill amount, include freight/landed costs, and who may see it (`vehicle_units.cost.view`)? | D-04, D-07, E-07, G-02, RB-5 |
| OQ-30 | Batteries: one `battery_serial` per unit (as captured on receipts). Do any models carry several packs or swappable packs needing component history (INV-004, architecture "battery_units")? | D-04, D-05, D-06 |
| OQ-31 | PDI dispositions (E-01 implements only the diagram's edges): may a failed PDI go straight from RECEIVED_PENDING_PDI to REJECTED (INV-008 says "quality hold or rejected"; the diagram only shows QUALITY_HOLD → REJECTED)? Can a QUALITY_HOLD unit be re-inspected and released to AVAILABLE? Is REJECTED final, and what follows it (RETURNED_TO_SUPPLIER)? | D-06, E-01 |
| OQ-32 | Location transfers: who may move a unit between locations, with what approval (System Architecture §12 item 4)? E-01 records locations on every movement but has no transfer movement. | E-01, later |
| OQ-33 | Stock movement corrections (LED-002): which mistakes may be corrected by a reversal movement, by whom, with what approval? Until then no reversal type exists. | E-01, E-05 |
| OQ-7 | Are invoices legally issued by this app or by external accounting software? | F-10 |
