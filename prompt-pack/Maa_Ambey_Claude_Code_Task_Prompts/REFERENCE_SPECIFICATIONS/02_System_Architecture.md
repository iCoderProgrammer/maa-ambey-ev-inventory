> Source: Version 1.2 approved project document; converted from DOCX for searchability in this prompt pack.

# System Architecture & Technical Design

Product: Maa Ambey Enterprises — Inventory Management System | Version: 1.2 draft | Updated: 12 October 2026 | Implementation stack confirmed: React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository

## 1. Architecture goals

Keep stock, serial identities, financial references and audit history consistent under concurrent use.

Separate the web UI from business rules and persistent storage.

Enforce access control on the server and maintain an auditable history of sensitive operations.

Confirmed P0: support multiple EV brands from day one in one shared deployment and catalogue. Keep brand-specific model/variant data, supplier relationships, pricing, warranty, serial-field policies, purchasing, sales and reporting correctly scoped; do not hardcode a single brand. Start with one showroom while preserving location IDs for future expansion.

Avoid direct vendor integrations until documented APIs, authorisation and maintenance ownership are confirmed.

## 2. Recommended stack

|Layer|Recommendation|Rationale / notes|
|---|---|---|
|Web frontend|React + TypeScript + Vite, React Router, Tailwind CSS, accessible component primitives and TanStack Query|Single-repository SPA frontend with responsive staff screens, strict TypeScript, accessible components and a straightforward Vite development/build workflow.|
|API|Go (Golang) + Gin REST API (OpenAPI/Swagger)|Modular monolith with explicit handlers, request validation, middleware, services and testable business logic. Keep all stock/sales invariants in the Go API layer.|
|Database|PostgreSQL (managed)|Transactions, constraints, relational integrity, reporting and stable stock/sales records.|
|Database access / migrations|pgx connection pool with reviewed versioned SQL migrations (golang-migrate or equivalent)|Explicit SQL access and repeatable deployments; critical constraints, transactions and indexes remain visible and testable.|
|Authentication|Server-side session or managed identity; secure HttpOnly cookies|Avoid browser localStorage for session tokens. Enforce CSRF/session protections.|
|File storage|Private S3-compatible object storage|PDI images, supplier bills and documents; use short-lived signed URLs after authorisation.|
|Background jobs|Queue/worker (add only if needed for alerts, imports or scheduled expiry)|Avoid making core sales depend on an async queue; scheduled tasks must be idempotent.|
|Hosting|Managed app hosting + managed PostgreSQL + private object storage|Simple operations, backups, environment separation and scalable path. Provider choice is a deployment decision.|
|Observability|Structured logs, error tracking, uptime checks, DB metrics and backup alerts|Diagnose failed imports, unexpected stock movements and release issues.|

A modular monolith is preferred for the MVP. Do not introduce microservices, event streaming or a service mesh until usage and integration demands justify the operational overhead.

## 3. Logical architecture

Staff Browser (desktop/tablet)
        |
        | HTTPS / secure session cookie
        v
React + Vite Web App
  - Dashboard and reports
  - Product / vehicle registry
  - Purchasing / receiving / PDI
  - Reservations / sales / delivery
  - Admin / users / audit
        |
        | REST / JSON + validation
        v
Go + Gin API (modular monolith)
  - Auth & RBAC guards
  - Catalogue module
  - Suppliers & purchasing module
  - Inventory / stock ledger module
  - Customers & reservations module
  - Sales / billing module
  - Reporting / export module
  - Audit / attachment module
        |                   \
        | ACID transactions   \ signed URL / metadata
        v                     v
Managed PostgreSQL      Private Object Storage
        |
        +--> Backup / point-in-time recovery / monitoring

Optional later: email/SMS provider, accounting software, manufacturer APIs,
website lead integration, GST/e-invoice/e-way bill service (after approval).

## 4. Domain boundaries / modules

1. Identity & access: users, roles, permission keys, session invalidation, location scope.

2. Catalogue: brands, models, variants, colours, spec provenance, scheme/ownership options, approved prices and validity windows.

3. Supplier & procurement: supplier profile, purchase orders, PO lines, receipt notes, received quantities, bill attachment and PDI disposition.

4. Inventory: individual vehicle units, parts SKUs, location, statuses, stock movement ledger, cycle counts and adjustment approvals.

5. CRM-lite: customer records, enquiries/test rides only if needed, quotation and reservations.

6. Sales & delivery: order, unit allocation, payment receipts, discount approval, delivery checklist, invoice/receipt print record.

7. Reporting: derived dashboards, filtered reports and export jobs.

8. Audit & attachments: append-only audit events, document metadata, authorised access and retention policy.

Each module should own its business operations. Cross-module changes (for example finalising a sale and marking a vehicle sold) must execute in one database transaction or use a carefully designed idempotent workflow if an external system is involved.

## 5. Data model (recommended core tables)

|Entity|Important attributes / relationship|
|---|---|
|users|id, name, email/username, password/identity subject, active, last_login_at|
|roles, permissions, user_roles, role_permissions|RBAC grants; location scope may be separately modelled|
|locations|id, code, name, type, address, active; initial seed is one showroom|
|brands|id, name, lifecycle_status (PENDING_VERIFICATION / ACTIVE / ARCHIVED), authorisation_status, authorisation_reference, verified_at, approval owner|
|models|id, brand_id (required FK), name, internal_code, vehicle_type, active; enforce brand-scoped code uniqueness|
|variants|id, model_id (and thus brand), variant_code, battery_type/capacity, colour, specs JSON or typed columns, spec_source, spec_verified_at; brand-specific serial/warranty/scheme settings|
|price_lists / price_list_items|variant/brand context, effective_from/to, price_type, amount, tax treatment reference, approval actor; keep price and discount history by brand/variant/scheme, never overwrite approved history|
|suppliers|legal name, contact, GSTIN if applicable, address, status, documents; supplier may supply one or more brands|
|purchase_orders, purchase_order_lines|supplier, dates, status, qty, agreed cost, currency, terms/reference|
|goods_receipts, goods_receipt_lines|PO link, receipt date, received quantity, accepted/rejected quantity, attachments|
|vehicle_units|internal_unit_id, variant_id, chassis_number (unique), motor_serial, battery_serials, charger_serial, location_id, status, supplier_id, PO/GRN refs, cost, PDI fields|
|stock_items|SKU, name, category, tracking_mode, UOM, reorder_point, active|
|stock_item_balances (optional cache)|item/location and derived quantity; reconciled against ledger; do not use as sole truth|
|stock_movements|immutable event, unit_id or stock_item_id, qty delta, from/to location/status, reason, source reference, actor, time, idempotency key|
|stock_counts, stock_count_lines, stock_adjustments|observed count, variance, reason, approval, posted movement reference|
|customers|name, phone, email, address, state, GSTIN only when applicable, marketing consent/purpose fields|
|quotations, quotation_lines|customer, model/unit, effective price, discount approval, valid_until, status|
|reservations|unit_id, customer_id, expiry_at, advance amount/status, cancellation reason, state|
|sales_orders, sales_order_lines|customer, exact unit, approved commercial snapshot, status, totals|
|payment_receipts|order_id, amount, method, external reference, received_at, status, reversal reference|
|sales_documents|type, number, financial_year, series, immutable commercial/tax snapshot, file reference/status|
|delivery_checklists|required item keys, checked_by, checked_at, evidence/document reference|
|audit_events|actor, action, entity, entity_id, before/after diff or structured change summary, request ID, IP policy, timestamp|
|attachments|owner entity, private object key, content type, size, checksum, uploader and access policy|

Multi-brand data integrity: Add a brand_supplier_links junction table when suppliers can supply multiple brands. Store each relationship’s verification/authorisation status, effective dates, supporting document and verifier. Every query, export and report that includes catalogue or vehicle records must retain brand_id context and enforce the user’s permissions server-side. Use brand-scoped uniqueness for manufacturer model/variant codes where codes can overlap between brands; chassis/VIN remains globally unique across the showroom inventory.

Serial data note: If one model can have multiple battery packs or replacement packs, store batteries in a separate battery_units table and link installation/removal history; do not pack multiple battery serials into an unvalidated text field. Use the simplest model that fits supplier documentation, but preserve traceability.

## 6. Key state machines

### Vehicle lifecycle

ORDERED -> IN_TRANSIT -> RECEIVED_PENDING_PDI
                              |             |
                              v             v
                        QUALITY_HOLD     AVAILABLE <-> DISPLAY
                              |             |
                              v             v
                         REJECTED       RESERVED -> AVAILABLE (cancel/expire)
                                            |
                                            v
                                       SOLD -> DELIVERED

Any physical unit needing inspection/repair may move to SERVICE_HOLD only via
an authorised transition. Sold/delivered units are not returned to available
without an explicit return/credit/inspection workflow and approvals.

### Purchase order lifecycle

DRAFT -> SUBMITTED -> APPROVED -> PARTIALLY_RECEIVED -> RECEIVED/CLOSED with explicit CANCELLED state where allowed. Receipt quantities cannot exceed accepted PO remaining quantity without an authorised override and reason.

### Reservation / sales lifecycle

DRAFT_QUOTE -> QUOTED -> RESERVED -> SALE_PENDING_APPROVAL -> CONFIRMED -> DELIVERED. Cancellation, expiry, refund and reversal must be explicit states/events. Avoid deleting completed records.

## 7. API design conventions

Base path: /api/v1.

JSON request/response, consistent error envelope (code, message, field errors, requestId).

Pagination: cursor or page/limit with server maximum; stable sort required.

Filtering: query params for brand, model, variant, colour, status, location, receipt date and ageing.

Use idempotency keys for sale finalisation, receipt posting, payment capture/reversal and bulk imports.

Use optimistic concurrency (version or updatedAt) for edits to reservations, units and POs.

OpenAPI contract committed with code; validation schemas are server-side source of truth.

### Endpoint sketch

|Method|Endpoint|Purpose|
|---|---|---|
|GET/POST|/api/v1/brands|List/create brands (write permission required)|
|GET/POST|/api/v1/models|Catalogue models and variants|
|GET|/api/v1/vehicle-units|Search/filter individual vehicle records|
|POST|/api/v1/purchase-orders|Create purchase order|
|POST|/api/v1/goods-receipts|Create draft receipt|
|POST|/api/v1/goods-receipts/:id/post|Validate and post receipt / stock movements|
|POST|/api/v1/vehicle-units/:id/pdi|Record PDI outcome|
|POST|/api/v1/reservations|Create reservation with unit lock/validation|
|POST|/api/v1/reservations/:id/cancel|Cancel reservation with reason|
|POST|/api/v1/sales-orders|Create sale draft|
|POST|/api/v1/sales-orders/:id/finalise|Atomic stock/payment/approval checks, sale commit|
|POST|/api/v1/stock-adjustments|Request adjustment|
|POST|/api/v1/stock-adjustments/:id/approve|Approval and ledger posting|
|GET|/api/v1/reports/stock-summary|Current stock derived by model/status/location|
|GET|/api/v1/reports/sales-summary|Sales totals by time/product/owner|
|GET|/api/v1/audit-events|Permission-limited audit search|

Brand-aware list/report endpoints accept validated brandId filters; all-brand totals are available only within the caller’s authorised scope. Provide a permissioned supplier-brand relationship workflow/API as part of catalogue governance. This is a contract sketch, not a final locked API. Refine after discovery and write OpenAPI definitions before frontend integration.

## 8. Transaction and inventory consistency rules

A vehicle unit has at most one active reservation at a time. Enforce via transaction and/or database constraint.

Sale finalisation must lock/check the unit row, verify saleable status, create sale/document snapshot, post movement and set unit status in the same transaction.

Never calculate current stock solely from editable form fields. Use the vehicle registry plus posted movement ledger; parts balances may be cached but ledger reconciliation is mandatory.

Stock movement rows are append-only for normal app roles. Correct an error using reversal/correction events, not destructive edits.

Negative quantity is blocked for tracked stock unless an explicitly approved exceptional workflow exists (MVP recommendation: block it).

Imported rows are first staged and validated; commit only after preview/confirmation. Idempotency prevents repeated imports from duplicating stock.

## 9. Security and privacy

RBAC enforcement in API guards, not just UI. Deny by default.

Secure cookies, short session lifetime appropriate to staff use, CSRF protection, rate limiting and session revocation.

Do not store authentication tokens in localStorage. Do not include sensitive customer data in logs or analytics.

Private file bucket, MIME/size validation, malware scanning if available, short-lived signed links and authorised download checks.

Apply least privilege to exports; record who exported sensitive reports.

Secrets managed by deployment platform, distinct dev/staging/production environments, encrypted connections and managed DB encryption at rest.

Define backup retention, RPO/RTO with client. Suggested starting target: daily backups, 30-day retention, RPO <= 24 hours and RTO <= 8 business hours, subject to hosting capabilities and owner approval.

## 10. Testing strategy

Unit tests for price snapshots, status transitions, permissions and validation rules.

Database integration tests for duplicate chassis, reservation collisions, transaction rollback and idempotency.

API contract tests for role and location restrictions.

End-to-end tests: PO -> receipt -> PDI pass -> available -> reserve -> sale -> delivery; PDI fail; cancelled booking/refund; stock adjustment approval; duplicate import.

Security tests: IDOR, privilege escalation, unauthorised attachment downloads, CSRF/session behaviour, rate limiting and export permissions.

Reconciliation tests: movements vs current vehicle statuses and parts-ledger balances.

Restore test: restore a backup into non-production and verify record counts and audit continuity.

## 11. Deployment and operations

Environments: local, staging and production with isolated databases/storage.

CI: lint, type-check, unit/integration tests, build, migration validation and dependency/security scan.

Production release: backup -> migration dry-run where supported -> deployment -> smoke checks -> monitor; rollback steps documented.

Do not run destructive migrations against production without a verified backup and explicit approval.

Seed client-confirmed role names (Owner, Inventory Manager, Sales Executive, Accountant, Viewer) and the three client-confirmed brand master records idempotently. Keep fine-grained permission grants pending owner sign-off. Use only sample data in non-production; production stock comes from a verified, signed-off import.

Monitor failed login spikes, import errors, sale-finalisation failures, background expiry jobs and backup status.

## 12. Decisions requiring sign-off

1. Confirm deployment provider, API host and database region.

2. Decide whether billing documents are legally issued by this app or by external accounting software.

3. Approve whether motor/battery/charger serials are mandatory and whether battery swaps need historical component links.

4. Approve location hierarchy and who can view/cross-transfer inventory.

5. Confirm BaaS modelling depth and whether only a label is needed in MVP.

6. Confirm backup/RPO/RTO target and document retention policy.

## 13. Reference inputs

Catalogue and business assumptions are based on Maa Ambey Enterprises https://maa-ambey-enterprisesev.vercel.app/, Lectrix EV https://www.lectrixev.com/, Ceeon India https://ceeonindia.co.in/, CBIC invoice rules https://cbic-gst.gov.in/gst-invoice-rules.html and NIC E-Way Bill portal https://docs.ewaybillgst.gov.in/. See Research Brief for specific findings and caveats.

# 14. Confirmed implementation stack and client configuration

The business architecture stays a modular monolith with one shared multi-brand application. The implementation stack is now explicitly React + TypeScript + Vite for the frontend, Go + Gin for the REST API, PostgreSQL accessed with pgx, and versioned SQL migrations. Frontend and backend must remain in one Git repository.

## Repository and runtime boundaries

Use /frontend and /backend directories in one repository, with common root documentation, Docker Compose, Makefile and validation commands. Do not create separate frontend/backend repositories.

Frontend calls the actual Go API using an environment-based API URL. Never place private secrets or the PostgreSQL connection string in Vite-exposed variables.

Use explicit SQL through pgx; do not introduce an ORM or switch away from the confirmed Go + React stack. Use versioned migrations for schema changes and tests for critical uniqueness/transaction constraints.

## Client-confirmed configuration

Seed the initial brand records for Lectrix EV, Ceeon India and Ujagar Electric, and keep the brand model extensible for future brands. Brand lifecycle status and evidence-based purchase/sales authorisation must remain separate.

The five initial role names are Owner, Inventory Manager, Sales Executive, Accountant and Viewer. Fine-grained permission assignments and approval thresholds require owner sign-off.

The client confirms legal name, GST details and showroom address exist, and that GST and income-tax filings are performed. Exact values and accountant-approved invoice configuration still need to be collected. Do not implement return filing in this MVP.

Opening-stock template and delivery checklist are confirmed needs. Actual stock data, reconciliation sign-off, and mandatory-versus-conditional delivery fields remain pending.
