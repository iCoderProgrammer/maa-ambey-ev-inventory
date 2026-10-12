# Maa Ambey Enterprises — Claude Code Feature/Task Prompt Pack (v1.2)

**Purpose:** Task-by-task backend and frontend prompts for implementing the Multi-Brand EV Inventory Management System.

**Confirmed implementation stack:** one Git repository; React + TypeScript + Vite frontend; Go + Gin REST API; PostgreSQL via pgx; versioned SQL migrations; Docker Compose.

## What is included

- **59 feature implementation tasks** across EPIC B–H.
- **118 task prompts:** one backend prompt and one frontend prompt for each implementation task.
- **6 client-decision/input prompts** (A-01–A-06), updated to record already-received answers and collect only missing source data/approvals.
- Master project context prompt and the updated Go + React monorepo setup prompt.
- Task registry with dependencies and prompt file references.
- V1.2 source specifications in Markdown and editable DOCX form.

## Client-confirmed baseline (v1.2)

- Initial intended operational brand roster: **Lectrix EV, Ceeon India, Ujagar Electric**. The architecture must support future brands. Roster selection is not proof of supplier/dealership authorisation; block purchase/sales activation until required supplier/legal identity, authorisation, verified catalogue, price/warranty/identifier evidence and approval metadata are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- The client confirms legal business name, GST details and showroom address exist, and that GST/income-tax filings are performed. Exact values and accountant-approved invoice settings remain to be entered. GST-return and income-tax-return filing are outside MVP scope unless separately approved.
- Verified models, variants, colours, battery/motor data, warranties, approved prices and identification rules must be collected; actual catalogue data is pending.
- Opening-stock import template is required; actual data and physical reconciliation sign-off are pending.
- Initial roles are **Owner, Inventory Manager, Sales Executive, Accountant, Viewer**. Granular permissions/thresholds require owner sign-off. Do not guess grants; backend access must deny by default.
- Delivery checklist categories confirmed: **keys, charger, manuals, warranty documents, payment, insurance, registration**. Required/conditional rules and overrides still need approval.

## How to use

1. Copy the pack or selected prompt files into the project repository, for example under `docs/claude-code-prompts/`. Keep v1.2 specs accessible under `docs/`.
2. Start Claude Code from the repository root.
3. First use `00_START_HERE/Go_React_Monorepo_Setup_Prompt_v2.md` only if the repository foundation is not yet established. Otherwise start at the next incomplete task in the backlog.
4. Paste `00_START_HERE/Master_Project_Context_Prompt.md` once per session (or put its core rules in Claude Code project instructions).
5. Run **one task prompt at a time**. For a full-stack task, backend usually comes first; review/commit the API and migration contract, then run the frontend prompt.
6. Review Git diff and test results before moving to the next task. Commit task-sized changes.
7. Do not ask Claude Code to execute the whole pack in one operation.

## Stack guardrail

The implementation stack in version 1.2 is React + Vite and Go + Gin in a single repository, with PostgreSQL + pgx. If any old material outside this pack mentions Next.js/NestJS/Prisma, follow the v1.2 implementation stack and record the architecture decision; do not create a Node.js backend.

## Safety and scope guardrails

- Brand records are data-driven; never hardcode a three-brand limit.
- Create seed records for the three client-confirmed brands when C-05 is run, but do not fabricate catalogue/stock information. Commercial activation stays blocked until approval evidence is recorded.
- Chassis/frame identifiers are normalized and unique. Motor/battery/charger uniqueness follows approved brand/model rules.
- Stock movement, receipt and sale operations must be transactional/idempotent where necessary and auditable. Do not hard-delete posted history.
- Exact billing/legal values, tax logic and invoice numbering require real client data and accountant sign-off. No tax return filing workflows in MVP.
- The roles are named, but fine-grained grants and approval thresholds remain pending. Do not silently infer them.
- The delivery checklist categories are agreed, but mandatory/conditional rules remain pending. Keep them configurable and gate production delivery until approved.
- Frontend hiding is not a security boundary; authorization is enforced in Go APIs.

## Recommended execution sequence

### Phase 0 — Client inputs and approvals
- `00_START_HERE/00_Client_Discovery_Prompts.md` — A-01 to A-06. These now reflect confirmed answers and document remaining evidence/approval requests.

### Phase 1 — Foundation and security
- B-01 Repository/environment
- B-02 Lint/typecheck/tests/CI
- B-03 React application shell
- B-04 Go API/OpenAPI
- B-05 PostgreSQL/migrations
- B-06 Authentication/session
- B-07 RBAC/location scope (role grants need owner sign-off)
- B-08 Audit framework
- B-09 Private attachment service

### Phase 2 — Multi-brand catalogue
- C-01 Schema
- C-02 Catalogue API/validation
- C-03 Brand/model management UI
- C-04 Effective-dated price history
- C-05 Seed initial brand roster and verified catalogue (seed brands, not fabricated models/stock)
- C-06 Brand-specific rules/isolation tests

### Phase 3 — Procurement, receiving and PDI
- D-01 Supplier master
- D-02 Purchase order lifecycle
- D-03 Goods receipt draft
- D-04 Vehicle registry
- D-05 Duplicate identifier controls
- D-06 PDI checklist/disposition
- E-01 Stock movement API (foundational dependency for posting)
- D-07 Atomic receipt posting
- D-08 Vehicle list/detail UI
- D-09 QR/barcode lookup/print

### Phase 4 — Inventory operations
- E-02 Movement timeline UI
- E-03 Spare/accessory SKU master
- E-04 Parts receipts/issues
- E-05 Stock counts/adjustment approval
- E-06 Ledger reconciliation
- E-07 Stock ageing

### Phase 5 — Sales and delivery
- F-01 Customer master
- F-02 Quotation builder/price snapshot
- F-03 Discount approval
- F-04 Reservation lifecycle
- F-05 Reservation expiry
- F-06 Sales order workflow
- F-07 Atomic sale finalisation
- F-08 Payment receipt tracking
- F-09 Configurable delivery checklist (policy approval required)
- F-10 Invoice/receipt/delivery templates (accountant sign-off required)
- F-11 Cancellation/return/reversal

### Phase 6 — Reports and imports/exports
- G-01 Dashboard metrics
- G-02 Inventory reports
- G-03 Sales/collections reports
- G-04 Pending-work queues
- G-05 CSV/XLSX export
- G-06 Opening-stock import preview
- G-07 Opening-stock import commit/idempotency

### Phase 7 — QA, operations and launch
- H-01 Unit/integration test suite
- H-02 End-to-end critical workflows
- H-03 Concurrency/idempotency tests
- H-04 Security/access tests
- H-05 Performance/indexes
- H-06 Backup/restore drill
- H-07 Staging/UAT/defect triage
- H-08 Opening-data import/physical reconciliation
- H-09 Training/runbooks
- H-10 Production rollout/smoke test

## Documentation references

Use the v1.2 copies under `REFERENCE_SPECIFICATIONS/` as the prompt pack's source baseline. DOCX copies are also included in `REFERENCE_SPECIFICATIONS/DOCX_v1.2/`.

## Files

- `ALL_TASK_PROMPTS.md` — all task prompts in one searchable Markdown file, including setup context and discovery prompts.
- `TASK_REGISTRY.csv` — task IDs, priorities, dependencies, backend/front-end file references.
- `00_START_HERE/` — setup prompt, master context and client input prompts.
- `01_...` to `07_...` — separate task prompt files.
- `REFERENCE_SPECIFICATIONS/` — latest project docs v1.2 in Markdown and DOCX.
