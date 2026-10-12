> Source: Version 1.2 approved project document; converted from DOCX for searchability in this prompt pack.

# Implementation Plan & Engineering Task Backlog

Product: Maa Ambey Enterprises — Inventory Management System | Version: 1.2 draft | Updated: 12 October 2026 | Planning estimates are indicative; implementation stack confirmed as React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository.

## 1. Delivery approach

Ship a robust single-location MVP before integrations. Work in vertical slices so the UI, API, validation, audit trail and tests for each workflow move together. Do not build only static screens and postpone core API/data constraints: duplicate serial prevention and transactional sale finalisation are foundational requirements.

### Proposed milestones

|Milestone|Outcome|Indicative duration|
|---|---|---|
|M0 Discovery & specification sign-off|Legal/entity data, actual brands/models, invoice policy, opening stock template, role matrix|3–5 working days|
|M1 Foundation|Repo/CI, app shell, auth/RBAC, DB migrations, audit, environments|1 week|
|M2 Catalogue & procurement|Brand/model masters, suppliers, POs, GRN, individual unit records, PDI|1–2 weeks|
|M3 Ledger & operational screens|Movement history, stock search, counts/adjustments, parts inventory|1 week|
|M4 Sales & delivery|Customers, quotations, reservations, payments, documents, delivery checklist|1–2 weeks|
|M5 Reports, hardening & pilot|Reports/import/export, security tests, backup restore, training, launch|1–2 weeks|

Expected MVP planning range: approximately 6–8 calendar weeks for a focused small team, assuming timely client answers and no major third-party integration. Add contingency for tax/invoice integration or migration complexity.

## 2. Backlog by epic

### EPIC A — Discovery, product decisions and source data

|ID|Task|Acceptance / deliverable|Priority|Estimate|
|---|---|---|---|---|
|A-01|Record confirmed launch brands and activation evidence|Create master records for Lectrix EV, Ceeon India and Ujagar Electric; allow future brands. Store evidence/status and block purchase/sales until supplier, authorisation and catalogue approvals are complete. Verify any Ujagar/Saffron relationship.|P0|0.5d|
|A-02|Capture verified legal and billing data|Capture exact legal name, GST details and showroom address from client documents; record financial year, invoice series and accountant approval. Client confirms GST/income-tax filing; return filing is out of scope.|P0|0.5d|
|A-03|Capture verified brand-specific model catalogues|Collect verified models, variants, colours, battery/motor details, identifiers, warranty and approved prices for all launch brands; track source and verification status. Approved catalogue files remain pending.|P0|1.5d|
|A-04|Prepare opening-stock import template|Provide opening-stock template with brand/model, identifiers, status and provenance; validate duplicates. Actual stock file, import owner and physical-count sign-off are pending.|P0|0.5d|
|A-05|Finalise role and permission matrix|Use the five confirmed role names. Obtain Owner sign-off for detailed permissions, approval limits, self-approval restrictions and export rules.|P0|0.5d|
|A-06|Finalise delivery checklist rules|Include keys, charger, manuals, warranty documents, payment, insurance and registration. Define mandatory/conditional items by model/deal and override policy after Owner approval.|P0|0.5d|

### EPIC B — Project foundation and security

|ID|Task|Acceptance / deliverable|Priority|Estimate|
|---|---|---|---|---|
|B-01|Create single-repository Go + React environment strategy|One Git repository with /frontend (React + TypeScript + Vite) and /backend (Go + Gin); document environment variables, Docker Compose, root Makefile and no-secret policy.|P0|0.5d|
|B-02|Configure lint, type-check, tests and CI|Pull request fails on lint/type/test/build failure|P0|0.5d|
|B-03|Bootstrap React + Vite app shell|Responsive React navigation, header, breadcrumbs and notification slot, route structure and placeholders for permission-aware navigation; no fake authentication.|P0|1d|
|B-04|Bootstrap Go + Gin API / OpenAPI|Versioned REST base path, request-struct validation, standard error envelope, request ID, health/readiness endpoints and OpenAPI documentation.|P0|1d|
|B-05|Database and migration baseline|PostgreSQL + pgx, reviewed SQL migrations and documented recovery. Seed five confirmed role names and three brand records idempotently; leave granular permissions for Owner sign-off.|P0|1d|
|B-06|Authentication and session handling|Login/logout, disabled account denial, secure cookies, CSRF/rate-limit controls|P0|1.5d|
|B-07|RBAC and location scope|API guards + object-level permissions; automated negative permission tests|P0|1.5d|
|B-08|Audit event framework|Captures actor/action/entity/reference/time for sensitive mutations; normal API cannot edit events|P0|1d|
|B-09|Attachment service|Private object storage, MIME/size validation and signed access after authorisation|P0|1d|

### EPIC C — Catalogue and price governance

|ID|Task|Acceptance / deliverable|Priority|Estimate|
|---|---|---|---|---|
|C-01|Implement brand/model/variant schema|Brand-model-variant relationships and unique codes enforced|P0|1d|
|C-02|Catalogue API and validation|CRUD with archive semantics, approval/provenance fields and RBAC|P0|1d|
|C-03|Catalogue UI|Search/table/editor, source status, brand activation warning and variant fields|P0|1.5d|
|C-04|Price list history|Effective dates, approval actor, quote snapshot; no silent retroactive changes|P0|1d|
|C-05|Seed multi-brand catalogue|Seed master records for Lectrix EV, Ceeon India and Ujagar Electric. Only activate verified/approved models for purchase/sales; keep future brands data-driven.|P0|0.75d|
|C-06|Brand-specific rules and isolation tests|Verify prices, warranties, serial-field rules and supplier mapping stay brand-specific; verify brand filters/exports and prevent cross-brand model selection or unauthorised activation|P0|1d|

### EPIC D — Procurement, vehicle units and PDI

|ID|Task|Acceptance / deliverable|Priority|Estimate|
|---|---|---|---|---|
|D-01|Supplier master|Supplier list/details, GSTIN/contact/documents and active/archive status|P0|0.75d|
|D-02|Purchase order API and UI|Draft/submit/approve lifecycle, line quantity/cost and PO history|P0|1.5d|
|D-03|Goods receipt draft workflow|Receipt against PO, partial receipt, serial capture, attachments and discrepancies|P0|1.5d|
|D-04|Vehicle unit registry|Unique internal ID, unique normalised chassis/frame, motor/battery/charger fields, status/location|P0|1.5d|
|D-05|Duplicate identifier controls|DB unique constraint + friendly server/client messages + integration test|P0|0.75d|
|D-06|PDI checklist and disposition|Pass/hold/reject, defect notes/photos; held units excluded from saleable results|P0|1d|
|D-07|Receipt posting transaction|Accepted units and stock movement posted atomically; idempotency and rollback test|P0|1.5d|
|D-08|Vehicle list/detail UI|Search/filter/chassis lookup, timeline, documents, permission-aware fields and actions|P0|1.5d|
|D-09|QR/barcode print/scan support|QR resolves to internal unit detail; scanner/manual entry fallback|P1/P0 if scanner needed|1d|

### EPIC E — Stock ledger, parts and reconciliation

|ID|Task|Acceptance / deliverable|Priority|Estimate|
|---|---|---|---|---|
|E-01|Append-only stock movement API|Movement includes source, actor, timestamp, state/location delta and idempotency key|P0|1d|
|E-02|Stock movement timeline UI|Filters by unit, event, location, actor, date and source reference|P0|0.75d|
|E-03|Spare/accessory SKU master|SKU, UOM, tracking mode, supplier, reorder level and compatibility metadata|P0-basic|1d|
|E-04|Parts receipts/issues|Quantity ledger; prevent negative stock; source references mandatory|P0-basic|1.5d|
|E-05|Stock count and adjustment requests|Count, variance, reason, approval and ledger posting workflow|P0|1.5d|
|E-06|Ledger reconciliation job/report|Detect difference between unit records and movement history; no silent correction|P0|1d|
|E-07|Stock ageing logic|Age from GRN/opening date with configurable bands and tests|P0|0.5d|

### EPIC F — Customer, reservation and sale

|ID|Task|Acceptance / deliverable|Priority|Estimate|
|---|---|---|---|---|
|F-01|Customer master|Search by phone/name, minimal data fields, duplicate warning and privacy-aware access|P0|1d|
|F-02|Quotation builder|Product/unit, approved price snapshot, validity, discount limits and printable preview|P0|1.5d|
|F-03|Discount approvals|Configurable role threshold, audit trail, cannot finalise while approval pending|P0|0.75d|
|F-04|Reservation lifecycle|Reserve a specific unit, expiry, cancellation reason and atomic collision protection|P0|1.5d|
|F-05|Reservation expiry handler|Idempotent scheduled/job action, audit event, safe release rules and tests|P0|0.75d|
|F-06|Sales order workflow|Draft, approval, payment checklist and exact unit allocation|P0|1d|
|F-07|Atomic sale finalisation|Transaction locks/validates unit, saves sale snapshot, posts movement and sets sold status|P0|1.5d|
|F-08|Payment receipt tracking|Multiple receipts/methods, reference/status, reversals and export|P0|1d|
|F-09|Delivery checklist|Include keys, charger, manuals, warranty documents, payment, insurance and registration. Configure required/conditional rules by model/deal; block missing requirements unless an audited override is permitted.|P0|1d|
|F-10|Invoice/receipt/delivery templates|Configurable series and tax fields, unique numbering, print/PDF; accountant approval tracked|P0|1.5d|
|F-11|Return/cancel/reversal workflow|No hard delete; return inspection and linked financial/stock reversal|P1 (basic cancellation P0)|1.5d|

### EPIC G — Reports and imports/exports

|ID|Task|Acceptance / deliverable|Priority|Estimate|
|---|---|---|---|---|
|G-01|Dashboard metrics|Physical vs saleable vs reserved vs held/in-transit metrics are clearly separated|P0|1d|
|G-02|Inventory reports|By brand/model/variant/status/location, ageing and acquisition-cost valuation|P0|1d|
|G-03|Sales and collections reports|Sales value, discount, tax, payment collected and outstanding shown separately|P0|1d|
|G-04|Pending PO/PDI/reservation reports|Ageing queue and drill-through filters|P0|0.75d|
|G-05|Export framework|CSV/XLSX, generated timestamp/filters, permission checks and export audit|P0|1d|
|G-06|Opening stock import preview|Validate schema, duplicate chassis, unresolved catalogue mapping and errors by row|P0|1.5d|
|G-07|Import commit/idempotency|Staging and all-or-nothing commit for fatal errors; batch IDs and audit log|P0|1d|

### EPIC H — QA, security, operations and launch

|ID|Task|Acceptance / deliverable|Priority|Estimate|
|---|---|---|---|---|
|H-01|Unit + integration test suite|Price snapshots, status transitions, permissions, duplicate identifiers, stock ledger|P0|2d|
|H-02|End-to-end critical workflows|PO -> GRN -> PDI -> available -> reserve -> sale -> delivery and failure paths|P0|2d|
|H-03|Concurrency/idempotency tests|Two users reserve/sell same unit; duplicate post/import request is safe|P0|1d|
|H-04|Security/access tests|IDOR, privilege escalation, attachment permissions, export restrictions, session checks|P0|1.5d|
|H-05|Performance test / indexes|Common list/search reports within target at agreed pilot data volume|P0|0.75d|
|H-06|Backup and restore drill|Restore backup in non-production and verify row counts/key reports|P0|1d|
|H-07|Staging/UAT and defect triage|Client signs critical journeys and invoice preview; all P0 acceptance defects closed|P0|1.5d|
|H-08|Data import and physical stock reconciliation|Opening stock import approved and discrepancies signed off|P0|1d|
|H-09|Training and runbooks|Admin guide, receiving/sales quick guide, backup/release and support runbook|P0|1d|
|H-10|Production rollout and smoke test|Production configuration, secure admin account, logs/backups/alerts, rollback option|P0|0.75d|

## 3. Suggested sprint sequencing

Sprint 0 / Confirmation and source-data sign-off: A-01 to A-06 are now answered at high level. Capture the three-brand roster, verify per-brand catalogue/authorisation, enter exact legal/GST/address details, prepare the opening-stock template, obtain owner sign-off on granular permissions and define mandatory/conditional delivery items. Sprint 1 / Foundation: B-01 to B-09 and skeleton dashboard/navigation. Sprint 2 / Catalogue & receiving: C-01 to C-06, D-01 to D-07. Sprint 3 / Inventory operations: D-08/D-09, E-01 to E-07. Sprint 4 / Sales: F-01 to F-11. Sprint 5 / Reporting and launch: G-01 to G-07, H-01 to H-10. This sequence is indicative; re-plan based on dependencies, team capacity and remaining client sign-offs.

## 4. Definition of Ready

A task is ready when user value and behaviour are clear, relevant screen states are described, API/data constraints are known, dependencies and permissions are listed, acceptance criteria are testable, and any required supplier/client decision is resolved or explicitly accepted as a blocker.

## 5. Definition of Done

Code reviewed; lint/type checks/build pass.

Unit/integration/E2E tests added for business-critical paths.

Server-side validation and permission enforcement implemented.

Audit trail and stock ledger behaviour verified for mutations.

Loading, empty, error, conflict and permission states present in UI.

API contract/docs updated; database migrations reviewed.

No secrets or real customer data in source, logs or tests.

Staging workflow verified; screenshot/QA evidence attached for UI work.

## 6. Release acceptance checklist

[x] Client confirms initial intended operational brands: Lectrix EV, Ceeon India and Ujagar Electric; collect verified supplier/authorisation evidence and current brand-specific catalogues before enabling purchase/sales.

[ ] Client confirms legal business name, GST details and showroom address exist, and GST/income-tax filings are performed; capture exact values and verify the accountant-approved invoice/numbering/tax configuration before production.

[ ] Opening stock import validated; physical count signed off.

[ ] No duplicate active chassis/frame numbers.

[ ] Receipt/PDI/reservation/sale/cancel/adjustment flows tested.

[ ] Two-user concurrency test cannot sell/reserve the same unit twice.

[ ] Roles tested using both UI and direct API requests.

[ ] Sensitive files are private; export permissions and audit checked.

[ ] Backup restore completed and runbook stored.

[ ] Staff trained; support owner and incident path communicated.

[ ] Production smoke test, rollback plan and monitoring are ready.

## 7. Deferred/P1 backlog

Public website lead/test-ride sync after data ownership and API feasibility review.

Service job cards, warranty claim records and part consumption.

Full BaaS contract/recurring subscription details after supplier requirements are confirmed.

Accounting system integration and statutory API integrations after accountant/provider assessment.

Additional locations, inter-store transfers and mobile scanner optimisation if business need warrants.

WhatsApp/SMS notifications subject to consent, provider cost and opt-out handling.

## 8. Dependencies and risks

Client must supply current catalogue, supplier relationship, warranty, approved pricing and relevant authorisation evidence for each brand/model to be enabled. The current public website substantiates Lectrix positioning only; it is not a substitute for per-brand sign-off.

Accountant must sign off invoice layout/numbering/tax configuration.

Opening stock quality determines migration effort; dirty serial fields may need a manual remediation sprint.

Any manufacturer/accounting/GST integration depends on accessible, documented APIs and credentials; it is not included in MVP estimates.

Security, backup and restore work cannot be cut from launch scope.

# 9. Client-answer reconciliation and updated implementation assumptions

Use this section to distinguish completed high-level decisions from data and approvals still needed. Do not reopen the confirmed choices unless the client changes them.

## Confirmed

Launch roster intent: Lectrix EV, Ceeon India and Ujagar Electric. Additional brands must be addable later.

Legal business name, GST details and showroom address exist; client files GST and income tax. Actual exact values and accountant-approved invoice settings still need to be collected.

The client agrees to collect verified model/variant/colour/battery-motor/warranty/approved-price/identifier data for each brand.

An opening-stock import template is required; actual inventory data is still pending.

Initial roles: Owner, Inventory Manager, Sales Executive, Accountant and Viewer. Fine-grained role/permission matrix remains pending approval.

Delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Whether each is mandatory or conditional remains to be signed off.

## Technology and data guardrails

Implementation stack: one Git repository, React + TypeScript + Vite frontend, Go + Gin backend, PostgreSQL via pgx and versioned SQL migrations. Do not switch to a different framework or database-access approach during setup without an approved architecture decision.

Create all three initial brand master records, but keep purchasing/sales blocked until per-brand supplier identity, authorisation and approved catalogue evidence is present. Do not invent Ujagar/Saffron affiliation or product values.

The MVP is inventory/showroom operations, not tax-return filing or a full accounting system. Invoice output and applicable tax configuration require accountant review.
