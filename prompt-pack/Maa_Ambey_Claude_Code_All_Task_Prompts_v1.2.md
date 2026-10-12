# Maa Ambey — All Claude Code Prompts (v1.2)

> Client-confirmed baseline updated 12 October 2026. Use one task prompt at a time. The pack contains 59 implementation tasks / 118 backend+frontend task prompts plus six client-input prompts.

## Start here: Go + React setup prompt

# Claude Code Prompt — Go + React Monorepo Setup (Client-Confirmed v2)

## Project
**Maa Ambey Enterprises — Multi-Brand EV Inventory Management System**

## 1. Role and Objective

Act as a Senior Software Architect, Senior Go Backend Engineer, and Senior React Frontend Engineer. Set up a maintainable, secure, production-oriented project foundation for an EV scooter showroom inventory management system.

The project must use **one Git repository** containing both frontend and backend:
- **Frontend:** React + TypeScript + Vite
- **Backend:** Go (Golang) + Gin REST API
- **Database:** PostgreSQL
- **Database driver/access:** pgx
- **Database migrations:** golang-migrate or another suitable, maintained Go migration tool
- **Frontend styling:** Tailwind CSS
- **Routing:** React Router
- **Server state:** TanStack Query
- **Client state:** Redux Toolkit only where shared client state is needed
- **Forms:** React Hook Form + Zod
- **API documentation:** OpenAPI/Swagger
- **Local development:** Docker Compose
- **Repository tooling:** Makefile, Git, ESLint, Prettier, gofmt, go vet, and automated tests

This task is for **project setup and architectural scaffolding**, not the full implementation of every inventory feature.

## 2. Client-Confirmed Requirements (Treat These as Project Facts)

### 2.1 Multi-brand requirement — mandatory
The client has confirmed that the showroom intends to operate with these initial brands:
1. Lectrix EV
2. Ceeon India
3. Ujagar Electric

The system must allow additional brands to be added in the future without redesigning the application or database.

Implementation guidance:
- Store brands as database records, not hardcoded UI constants.
- Add idempotent seed records for the three client-confirmed launch brands if the approved schema supports seeding.
- Do not invent models, variants, colours, technical specifications, approved prices, warranty terms, stock quantities, suppliers, or vehicle identifiers.
- The fact that a brand is configured in the system must not fabricate catalogue verification or supporting authorization documents. If verification/authorization fields are part of the agreed model, keep them separate from the brand's active status and record missing evidence as pending.

### 2.2 Legal and billing
The client has confirmed that legal business name, GST details, and showroom address exist. The client files income tax and GST.

Implementation guidance:
- Record that the actual legal and tax details must be entered from client-provided source documents; never invent them or seed dummy production values.
- Prepare the architecture for configurable business profile and invoice settings in a later feature task.
- GST-compliant invoice generation must follow requirements verified by the client's accountant/tax adviser and applicable rules.
- **Do not build GST-return filing, income-tax-return filing, or tax-portal integrations in this setup task.** The client filing those returns does not mean the inventory app needs to file them.

### 2.3 Brand-wise catalogue
The client has approved collecting and maintaining verified data for each brand's models, variants, colours, battery/motor details, warranty, approved prices, and vehicle identification rules.

Implementation guidance:
- Make the future catalogue model extensible.
- Keep prices, warranty/configuration and identifying fields data-driven where appropriate.
- Actual catalogue information is not yet supplied in this request. Do not fabricate catalogue seed data.

### 2.4 Opening stock import
The client wants an opening-stock import template for actual showroom stock, including scooter count, model, chassis number, battery serial (where applicable), and stock status.

Implementation guidance:
- Document the planned import requirements in `docs/REQUIREMENTS_BASELINE.md`.
- The setup task may add a sample **blank/template-only** CSV if useful, but do not import fabricated stock or implement the complete import workflow unless the existing approved implementation tasks explicitly place that feature in scope.
- Future import must support preview, row-level validation, duplicate detection, an error report, safe commit, and retry/idempotency protection.

### 2.5 User roles
The client has confirmed the intended role names:
- Owner
- Inventory Manager
- Sales Executive
- Accountant
- Viewer

The detailed permission matrix has not yet been approved.

Implementation guidance:
- Document the five roles as confirmed requirements.
- Prepare a role/permission architecture that can enforce authorization on the backend, not only hide frontend buttons.
- Do not invent the final granular permission matrix. Mark exact actions that need client approval as pending.
- Full login/session and complete RBAC workflows are outside this setup-only task unless already implemented in the repository.

### 2.6 Vehicle delivery checklist
The client confirmed that the delivery checklist should cover:
- Keys
- Charger
- Manuals
- Warranty documents
- Payment
- Insurance
- Registration

The detailed rule for whether each item is always mandatory or conditional still needs confirmation.

Implementation guidance:
- Record these checklist categories in `docs/REQUIREMENTS_BASELINE.md`.
- Keep future delivery checklist items configurable where appropriate.
- Do not implement the complete delivery workflow in this setup-only task.

## 3. Inspect Existing Repository First

Before modifying files:
1. Inspect the current directory, Git status, existing source code, package manager files, dependencies, environment files, and available docs.
2. Locate and read the approved project documents if available:
   - `00_Research_Brief_and_Decisions`
   - `01_PRD`
   - `02_System_Architecture`
   - `03_Business_Rules`
   - `04_Design_Specification`
   - `05_Implementation_Tasks`
3. Preserve existing implementation and approved business requirements. Extend the repository rather than replacing working code unnecessarily.
4. Do not reset Git history, delete files, overwrite source documents, or remove dependencies without a demonstrated reason.
5. The former Next.js/NestJS proposals are superseded for implementation by the client-selected **React + Vite and Go + Gin** stack. Do not introduce Next.js, NestJS, Express, or a Node.js backend.
6. If approved documentation conflicts with the chosen stack, preserve the business requirement and document only the implementation-level architecture change in `docs/ARCHITECTURE_DECISIONS.md`.

Create a concise plan after inspection, then proceed with setup end-to-end. Do not stop after providing a plan.

## 4. Target Single-Repository Structure

Adapt this structure to the existing repository if one already exists. For a greenfield setup, use a structure similar to:

```text
maa-ambey-inventory/
├── frontend/
│   ├── public/
│   ├── src/
│   │   ├── app/
│   │   ├── components/
│   │   │   ├── ui/
│   │   │   └── layout/
│   │   ├── features/
│   │   │   ├── dashboard/
│   │   │   ├── brands/
│   │   │   ├── vehicle-models/
│   │   │   ├── inventory/
│   │   │   ├── suppliers/
│   │   │   ├── purchases/
│   │   │   ├── sales/
│   │   │   ├── customers/
│   │   │   ├── reports/
│   │   │   └── users/
│   │   ├── hooks/
│   │   ├── lib/
│   │   │   ├── api/
│   │   │   ├── config/
│   │   │   └── utils/
│   │   ├── store/
│   │   ├── types/
│   │   └── styles/
│   ├── .env.example
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── backend/
│   ├── cmd/api/main.go
│   ├── internal/
│   │   ├── config/
│   │   ├── database/
│   │   ├── server/
│   │   ├── middleware/
│   │   ├── logger/
│   │   └── modules/
│   │       ├── health/
│   │       ├── auth/
│   │       ├── users/
│   │       ├── brands/
│   │       ├── vehiclemodels/
│   │       ├── inventory/
│   │       ├── suppliers/
│   │       ├── purchases/
│   │       ├── sales/
│   │       ├── customers/
│   │       ├── reports/
│   │       └── audit/
│   ├── migrations/
│   ├── docs/
│   ├── .env.example
│   ├── go.mod
│   └── go.sum
├── docs/
│   ├── REQUIREMENTS_BASELINE.md
│   ├── ARCHITECTURE_DECISIONS.md
│   ├── DEVELOPMENT_SETUP.md
│   └── API_CONVENTIONS.md
├── docker-compose.yml
├── Makefile
├── .gitignore
├── .editorconfig
└── README.md
```

Avoid unnecessary empty files and abstractions. Create purposeful starter modules only.

## 5. Go Backend Foundation

Set up a maintained Go version compatible with selected dependencies, Go modules, Gin, pgx, PostgreSQL connection pooling, environment configuration, structured logging, and database migrations.

Implement:
1. Bootstrap at `backend/cmd/api/main.go`.
2. Centralized config loading and required-variable validation.
3. PostgreSQL connection-pool creation and connection checks.
4. Graceful HTTP server shutdown and appropriate timeouts.
5. API routes under `/api/v1`.
6. Request ID, structured request logging, panic recovery, and consistent error responses.
7. CORS allowlist from configuration; do not use permissive production CORS.
8. Appropriate rate-limiting foundations.
9. Request validation and parameterized SQL.
10. OpenAPI/Swagger documentation at a documented development route.
11. Unit-test foundations and database integration tests where PostgreSQL is available.

Create at minimum:
- `GET /api/v1/health` — application process is running.
- `GET /api/v1/ready` — required dependencies, including PostgreSQL, are ready.

Do not expose SQL errors, credentials, stack traces, or secrets in public responses. Do not pretend unimplemented modules have working business logic.

## 6. PostgreSQL and Initial Multi-Brand Schema

Use PostgreSQL through Docker Compose and pgx. Use versioned migrations via a maintained migration tool.

Requirements:
- Database credentials only through backend environment variables.
- Persistent Docker volume for local development.
- Database health check.
- Documented migration up/down workflow.
- Never reset/drop data automatically on application startup.
- Never run destructive database operations without an explicit developer action.

Read the approved Architecture and Business Rules documents before finalizing business schema. At minimum, the architecture must support these entities and relationships over future tasks:
- Brands
- Vehicle models
- Vehicle variants
- Suppliers
- Supplier-brand relationships
- Individual vehicle inventory records
- Brand-specific catalogue/configuration details
- Future user roles/permissions and audit history

Where the existing specs provide enough detail, create an initial migration for a `brands` table with stable ID, name, unique slug, lifecycle status, and timestamps. Add idempotent seed records for **Lectrix EV, Ceeon India, and Ujagar Electric** because the client confirmed them as initial operational brands. Running seeds multiple times must not create duplicates.

Do not seed fabricated models, variants, prices, warranties, chassis numbers, motor/battery serials, supplier records, or stock quantities. If verification or authorization status is represented in the schema, keep it separate from the brand lifecycle status; do not invent evidence.

Do not hardcode these three brand names throughout application logic. Brand records must remain data-driven so future brands can be added through the planned Brand Management feature.

## 7. React Frontend Foundation

Use React + TypeScript + Vite with strict type checking. Configure React Router, Tailwind CSS, ESLint, Prettier, TanStack Query, and a centralized typed API client. Include Redux Toolkit only for appropriate shared client state; do not duplicate server state in Redux unnecessarily.

Create a responsive administration shell with:
- Sidebar navigation and header
- Main content region and breadcrumbs/page titles
- Professional showroom inventory dashboard placeholder
- Reusable button, input, select, table, badge, and dialog foundations where useful
- Loading, empty, error, and success states
- Light and dark themes, including system preference
- Accessible labels, keyboard navigation, and visible focus states
- CSS-variable-based design tokens for colour, typography, spacing, border, and radius

Show navigation placeholders for future Dashboard, Brands, Models, Inventory, Suppliers, Purchases, Sales, Customers, Reports, and Users modules. A placeholder must not imply a module is implemented.

Do not implement complete CRUD or business workflows in the setup task.

Frontend environment example:

```dotenv
VITE_API_BASE_URL=http://localhost:8080/api/v1
```

Never place private API keys, database URLs, or secrets in `VITE_*` variables. Do not use localStorage as a general-purpose persistence layer. Use secure session handling when authentication is later implemented.

## 8. Frontend–Backend Integration

Use `VITE_API_BASE_URL` as the single frontend API base URL. Configure a Vite development proxy if it improves local development.

Implement a centralized typed API client with consistent JSON parsing, error handling, cancellation support where appropriate, and no hardcoded environment-specific endpoint strings in feature components.

The initial React UI must call the actual Go health endpoint and display the returned status. Do not replace this with a fake success response.

## 9. Client Requirements Baseline

Create `docs/REQUIREMENTS_BASELINE.md` with these client-confirmed requirements and unresolved details.

### Confirmed initial brands
- Lectrix EV
- Ceeon India
- Ujagar Electric
- New brands must be addable without architecture changes.

### Verified catalogue data to collect
- Brand, model, variant, colour
- Battery/motor details where applicable
- Warranty terms
- Approved prices and effective dates where applicable
- Chassis/frame, motor, and battery identifier rules
- Authoritative source and verification status for each catalogue record

### Legal and billing
- Client confirmed legal business name, GST details, and showroom address exist, but has not supplied their actual values in this task.
- Client files GST and income tax.
- Mark actual business details and invoice/tax configuration as pending data entry and accountant/tax-adviser verification.
- Do not include tax-return filing or income-tax-return filing as a feature in this setup task.

### Opening-stock import planned fields
At minimum record the need to import actual stock with model, brand, chassis/frame number, battery serial where applicable, and current stock status. The later approved import design should add appropriate validation and supplier/receipt context, duplicate checks, preview, row-level errors, safe commit, and idempotency. Do not invent stock rows.

### Confirmed intended roles; detailed permissions pending
- Owner
- Inventory Manager
- Sales Executive
- Accountant
- Viewer

Do not invent final granular permissions. Document that authorization must be enforced by backend APIs, and track the permission matrix as an approval dependency.

### Confirmed delivery checklist categories
- Keys
- Charger
- Manuals
- Warranty documents
- Payment
- Insurance
- Registration

Whether each item is mandatory or conditional remains to be confirmed. Record this as an open decision; do not impose all conditions in code yet.

## 10. Environment and Security

Create:
- `frontend/.env.example`
- `backend/.env.example`

Backend example should document applicable variables, for example:

```dotenv
APP_ENV=development
APP_PORT=8080
DATABASE_URL=postgres://app_user:change_me@localhost:5432/maa_ambey_inventory?sslmode=disable
CORS_ALLOWED_ORIGINS=http://localhost:5173
LOG_LEVEL=debug
```

Use example-only values. Validate configuration at startup. Do not commit `.env` files containing real values. Do not add hardcoded administrator passwords. Keep private information out of logs and browser-accessible environment variables.

## 11. Root Developer Workflow

Use the root Makefile for reproducible common commands. Configure targets that actually work, such as:
- `make setup`
- `make dev`
- `make dev-frontend`
- `make dev-backend`
- `make db-up`
- `make db-down`
- `make db-migrate-up`
- `make db-migrate-down`
- `make test`
- `make lint`
- `make typecheck`
- `make build`

Document tool prerequisites and fail clearly when a required tool is absent. Do not document scripts that have not been implemented.

## 12. Quality Gates

### Backend
Run `gofmt`, `go vet ./...`, `go test ./...`, and build checks. Add basic tests for health/readiness handlers, configuration validation, and error responses. Run database integration tests if PostgreSQL is available; accurately mark any skipped/unavailable tests.

### Frontend
Run ESLint, TypeScript type checking, configured tests, and production build. Add a smoke test for the application shell and the live health endpoint integration where practical.

### Multi-brand readiness
Verify that:
- Three initial brand seeds are idempotent.
- No single-brand assumptions exist in shared code.
- Brand names are stored as data, not scattered hardcoded conditionals.
- Future brand filtering and backend authorization have clear implementation boundaries.
- No unverified models, prices, warranties, or stock are fabricated.

## 13. Documentation to Deliver

Create or update:
- Root `README.md` — overview, stack, repo structure, prerequisites, setup, environment variables, database, migrations, run/test/build commands, API docs route, troubleshooting.
- `docs/REQUIREMENTS_BASELINE.md` — the confirmed client decisions and open questions described above.
- `docs/DEVELOPMENT_SETUP.md` — local workflow and conventions for adding future features.
- `docs/API_CONVENTIONS.md` — endpoint naming, status codes, request validation, error responses, and versioning.
- `docs/ARCHITECTURE_DECISIONS.md` — Go + React stack decision, assumptions, deviations from prior architecture proposal, and pending technical/client decisions.

Do not overwrite the approved PRD, Architecture, Business Rules, Design Specification, or Implementation Tasks documents. Preserve business requirements and record implementation-level stack differences explicitly.

## 14. Strict Scope Boundaries

This task is only for project setup and the minimum functional foundation.

Do not implement complete:
- Authentication/login and user management workflows, unless already present
- Full brand CRUD screens
- Full model/variant CRUD
- Purchase orders, receiving, or PDI workflows
- Stock movements and adjustments
- Opening-stock import workflow
- Sales, invoicing, payment, and delivery workflows
- Tax-return filing
- Supplier integrations or payment gateways
- Full reporting suite

It is acceptable to create the initial brands migration/seed, health integration, blank opening-stock template, and requirements documentation if these remain small and setup-appropriate. Do not build the complete features in this task.

## 15. Execution Sequence

1. Inspect the repository and read the approved specifications.
2. Report the existing state and short setup plan, then continue implementation.
3. Record the client-confirmed facts in `docs/REQUIREMENTS_BASELINE.md`.
4. Create or adapt the single-repo structure.
5. Set up Go modules, Gin, configuration, middleware, and the health/readiness endpoints.
6. Configure PostgreSQL, pgx, migrations, and idempotent initial brand seed records.
7. Set up React, Vite, TypeScript, routing, and Tailwind.
8. Build the responsive dashboard shell and integrate it with the real Go health endpoint.
9. Add environment templates, Docker Compose, and Makefile commands.
10. Add tests, formatting, lint, type-check, and build workflows.
11. Write setup, API, requirements baseline, and architecture decision docs.
12. Run all available validation commands and fix setup-related problems.
13. Inspect the final Git diff and verify no unrelated files or approved business documents were changed.

If a dependency, command, or service is unavailable, explain the blocker accurately. Never report a command as passing unless it actually ran successfully.

## 16. Definition of Done

The setup is complete when:
- Frontend and backend are in the same Git repository.
- React starts locally using Vite.
- Go API starts locally using Gin.
- PostgreSQL starts with Docker Compose and a persistent development volume.
- Database connectivity and migration workflow are documented.
- Initial brand master data supports Lectrix EV, Ceeon India, and Ujagar Electric, with idempotent seeds and room for future brands.
- No fabricated catalogue values or inventory records exist.
- The frontend calls the real Go health endpoint.
- OpenAPI documentation is accessible at its documented route.
- Client decisions and pending approvals are documented.
- Quality check results are recorded honestly.
- Another developer can follow the README and reproduce local setup.

## 17. Final Response Required from Claude Code

When finished, provide:
1. Files/folders created or changed.
2. Final repo structure.
3. Exact commands to start PostgreSQL, backend, and frontend.
4. Required environment variables.
5. Migration and seed commands, including how to re-run seeds safely.
6. Health and OpenAPI URLs.
7. Results of tests, linting, type-checking, and builds.
8. Any skipped/unverified checks or open decisions.
9. The next recommended task from the approved implementation backlog, without starting that task automatically.

**Begin by inspecting the current repository and the approved documents. Then implement and verify the setup end to end.**


---

## Master project context prompt

# Master Prompt — Execute One Selected Task in Maa Ambey Inventory (v1.2)

Paste this once at the beginning of a Claude Code session. Then paste exactly one task-specific backend or frontend prompt from this pack.

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository or introduce Next.js/NestJS/Node.js backend.

CLIENT-CONFIRMED BASELINE (v1.2):
- Initial intended operational roster: Lectrix EV, Ceeon India, Ujagar Electric. The system must support adding brands later. This roster is client intent, not proof of supplier/dealership authorisation. Brand records may exist before they are commercially eligible; block purchase/sales until required legal supplier identity, authorisation evidence, verified catalogue, approved price/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, approved prices, warranties, suppliers, authorisation evidence or stock. Actual model catalogue and opening-stock data remain pending.
- Client confirms legal business name, GST details and showroom address exist, and GST/income-tax filings are performed. The exact values and accountant-approved invoice settings have not been supplied. GST/income-tax return filing and government tax-portal integration are outside MVP scope unless separately approved.
- Initial role names are Owner, Inventory Manager, Sales Executive, Accountant and Viewer. Detailed permissions and approval thresholds are pending owner sign-off; backend permission enforcement must deny by default and must not guess production grants.
- Client-confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Required/conditional/optional rules and overrides remain pending approval; keep policy configurable and block production delivery until approved.
- Opening-stock import template is required, but real stock data and physical reconciliation sign-off remain pending. Never create fabricated inventory.

MANDATORY PRODUCT RULES:
- Brand/model/variant/supplier/price/warranty rules must be brand-aware. Brand names must come from data/API, not hardcoded enums or frontend constants.
- Every saleable scooter is tracked as an individual unit. Normalized chassis/frame identifiers must be unique; motor/battery/charger uniqueness follows approved configuration.
- Stock-changing actions validate server-side, record append-only stock movement/audit history where required, and use transactions/idempotency where critical.
- Do not hard-delete posted stock, sale, payment, receipt, or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.
- Do not hardcode GST rates, thresholds, legal details, invoice numbering, or unapproved role permissions.

BEFORE CODING:
1. Inspect current Git status and relevant code/docs. Read v1.2 PRD, System Architecture, Business Rules, Design Specification, Implementation Tasks, and `docs/client-inputs/` records if present.
2. Preserve existing work and follow conventions. Resolve stack discrepancies in favour of this confirmed Go + React stack and document architectural deviations.
3. Do not guess unresolved business/client/tax decisions. Record blockers in `docs/ARCHITECTURE_DECISIONS.md` or task-specific notes.

TASK DISCIPLINE:
- Implement ONLY the named task. Do not start dependent/future tasks or unrelated refactors.
- Implement real production-path logic, not fake/mock success behavior.
- Add relevant tests, API/OpenAPI docs, migrations, and UI states only as required by the task.
- Use server-side validation and parameterized SQL with pgx. Use TanStack Query for remote state; do not persist session secrets/tokens in localStorage.
- Include loading, empty, error, conflict, stale-data and permission-denied states when a screen is involved.
- Run relevant formatters, lint, typecheck, tests and builds. Report actual results honestly.
- Inspect the final Git diff and do not overwrite approved specifications.

Final response must state changed files, summary, API/database changes, tests run/results, blockers and exact next task ID. Stop when the selected task is complete.
```

Never ask Claude Code to implement every task in one operation. Use the dependencies in `TASK_REGISTRY.csv` and sequence in README. The individual task prompts are the implementation source of truth for task-specific scope.


---

## Client decisions and remaining inputs (A-01–A-06)

# Client Decisions & Remaining Inputs — Version 1.2

Use these prompts before activating production operations. Client answers have already been received for the high-level decisions below. Do not ask the client to re-decide confirmed items; collect the missing source data/evidence and obtain any remaining approvals. Store outputs in `docs/client-inputs/`. Never invent answers.

## A-01 — Initial brand roster (roster confirmed; evidence pending)
**Confirmed by client:** The initial intended operational brands are Lectrix EV, Ceeon India, and Ujagar Electric. The system must support adding more brands later.

**Claude Code prompt:**

Create `docs/client-inputs/launch-brand-roster.md` that records the roster above as client-confirmed. For each brand collect legal supplier/manufacturer entity, supplier contact, dealer/stocking authorisation reference/document, catalogue source, verification status, approval owner, verification date, and allowed purchase/sales status. Do not treat roster selection as proof of authorisation. Keep brand existence/lifecycle separate from operational eligibility and block purchase/sales until required evidence is reviewed and approved. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity. Do not invent missing details.

## A-02 — Legal and billing data (existence confirmed; exact values and sign-off pending)
**Confirmed by client:** The legal business name, GST details, and showroom address exist. The client files GST and income tax.

**Claude Code prompt:**

Create `docs/client-inputs/billing-and-legal-signoff.md` listing the actual source documents/values still to be entered: legal entity name, GSTIN and registration status as applicable, registered/showroom address and state, financial year, invoice numbering series, required invoice fields, cancellation/credit-note process, accountant/tax-adviser reviewer, and sign-off date. Clearly mark values not yet provided as pending; do not fabricate them or hardcode tax rates/thresholds. This inventory system is not a GST-return or income-tax-return filing system. Do not add government portal integrations without separate approval.

## A-03 — Brand-specific model catalogues (scope confirmed; actual data pending)
**Confirmed by client:** For each brand, collect verified models, variants, colours, battery/motor details, warranties, approved prices, and vehicle-identification rules.

**Claude Code prompt:**

Create `docs/client-inputs/brand-catalogue-template.md` and a fillable template covering brand, model, variant, colour, applicable battery/motor details, chassis/frame/motor/battery identifier rules, warranty terms, approved price and effective date, supplier/manufacturer source document, supplier confirmation, client approval, verification date and approver. Distinguish blank/unverified from zero/not applicable. Do not scrape or fabricate production data. Include the initial brands confirmed by the client but leave unknown catalogue values blank and unverified.

## A-04 — Opening stock (template approved in principle; actual data pending)
**Confirmed by client:** An opening-stock import template is needed for actual scooter quantities, models, chassis numbers, applicable battery serials, and stock statuses.

**Claude Code prompt:**

Create `docs/client-inputs/opening-stock-template.csv` plus a data dictionary and validation rules. Include brand/model/variant, chassis/frame, motor/battery/charger serials where applicable, colour, location, received/opening-stock date, condition/status, source/provenance, and supplier/PO/GRN/cost references only where available and approved. Define identifier normalization and duplicate checks. Provide a clearly fake example only in separate documentation, never as a production import row. Actual stock and physical reconciliation sign-off remain pending.

## A-05 — Role permission matrix (role names confirmed; exact permissions pending)
**Confirmed by client:** Owner, Inventory Manager, Sales Executive, Accountant, and Viewer roles are required.

**Claude Code prompt:**

Create `docs/client-inputs/role-permission-matrix.md` using exactly these five initial roles. For each sensitive action, propose view/create/edit/approve/export grants, location scope, approval threshold and self-approval rules, but mark the granular matrix as **PROPOSED — OWNER SIGN-OFF REQUIRED** until approved. Include dedicated considerations for stock adjustment, discount approval, cost-price visibility, refunds, user/role administration, invoice settings and exports. Do not add Service Advisor as an initial role unless the client approves it later. Backend authorization must deny by default.

## A-06 — Delivery checklist (categories confirmed; required/conditional policy pending)
**Confirmed by client:** Checklist coverage includes keys, charger, manuals, warranty documents, payment, insurance, and registration.

**Claude Code prompt:**

Create `docs/client-inputs/delivery-checklist-decisions.md` with these categories and any existing approved PRD items such as PDI/customer acknowledgement. For each item ask the client to mark Required, Conditional, Optional, or Not Applicable and state the evidence to record. Confirm override roles, reason requirements, payment conditions, finance/insurance/registration states, and whether policy varies by model/transaction. Do not hardcode unresolved mandatory/conditional rules or allow production delivery when the checklist policy has not been approved.

## Exit gate
Before enabling real purchasing/sales for each brand, confirm the corresponding supplier/authorisation and catalogue evidence. Before issuing production invoices, obtain actual legal/GST values and accountant approval. Before production stock import, validate the client's actual opening stock and arrange physical reconciliation. Before enabling mutation grants in production, obtain owner sign-off for the granular permission matrix. Before delivery workflow goes live, approve mandatory/conditional checklist policy. Never substitute invented values for pending inputs.


---

## Feature implementation tasks (B-01 through H-10)


# EPIC B — Foundation & Security


# B-01 — Repository and environment strategy

**Epic folder:** `01_Foundation_Security`  
**Priority:** P0  
**Depends on:** Workspace inspection

## Task scope

Keep the single-repo layout /frontend and /backend. Verify .gitignore, root README, Docker Compose and separate frontend/backend .env.example files. Document local/staging/production configuration; ensure secrets are not committed. Do not change business behavior.

## Acceptance criteria

Fresh setup instructions are reproducible; no secrets tracked; environment variables are documented; existing uncommitted work is preserved.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: B-01 — Repository and environment strategy (BACKEND ONLY)
Priority: P0
Dependencies: Workspace inspection

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Keep the single-repo layout /frontend and /backend. Verify .gitignore, root README, Docker Compose and separate frontend/backend .env.example files. Document local/staging/production configuration; ensure secrets are not committed. Do not change business behavior.

ACCEPTANCE CRITERIA:
Fresh setup instructions are reproducible; no secrets tracked; environment variables are documented; existing uncommitted work is preserved.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: B-01 — Repository and environment strategy (FRONTEND ONLY)
Priority: P0
Dependencies: Workspace inspection

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Verify Vite env usage: only non-secret VITE_* values are exposed to the browser. Ensure frontend API base URL is centralized and documented. Keep one repository and avoid adding a second package workspace without need.

ACCEPTANCE CRITERIA:
Fresh setup instructions are reproducible; no secrets tracked; environment variables are documented; existing uncommitted work is preserved.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# B-02 — Lint, type-check, tests and CI

**Epic folder:** `01_Foundation_Security`  
**Priority:** P0  
**Depends on:** B-01

## Task scope

Configure Go format/vet/test/build checks and CI for backend; run checks from backend/. Ensure the pipeline fails on failure and does not silently skip tests.

## Acceptance criteria

CI or documented local commands run Go and frontend checks; failing checks return non-zero; actual commands are documented.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: B-02 — Lint, type-check, tests and CI (BACKEND ONLY)
Priority: P0
Dependencies: B-01

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Configure Go format/vet/test/build checks and CI for backend; run checks from backend/. Ensure the pipeline fails on failure and does not silently skip tests.

ACCEPTANCE CRITERIA:
CI or documented local commands run Go and frontend checks; failing checks return non-zero; actual commands are documented.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: B-02 — Lint, type-check, tests and CI (FRONTEND ONLY)
Priority: P0
Dependencies: B-01

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Configure ESLint, Prettier if present, TypeScript typecheck, frontend tests and Vite production build. Add root Makefile/CI commands only if existing conventions support them.

ACCEPTANCE CRITERIA:
CI or documented local commands run Go and frontend checks; failing checks return non-zero; actual commands are documented.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# B-03 — React application shell

**Epic folder:** `01_Foundation_Security`  
**Priority:** P0  
**Depends on:** B-01

## Task scope

Only add or verify the minimum health API contract consumed by the shell if needed. Do not implement business endpoints in this task.

## Acceptance criteria

Shell renders on desktop/tablet/mobile; navigation works; keyboard/focus states work; no placeholder navigation claims an unimplemented workflow is functional.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: B-03 — React application shell (BACKEND ONLY)
Priority: P0
Dependencies: B-01

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Only add or verify the minimum health API contract consumed by the shell if needed. Do not implement business endpoints in this task.

ACCEPTANCE CRITERIA:
Shell renders on desktop/tablet/mobile; navigation works; keyboard/focus states work; no placeholder navigation claims an unimplemented workflow is functional.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: B-03 — React application shell (FRONTEND ONLY)
Priority: P0
Dependencies: B-01

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build responsive shell: collapsible sidebar, top bar, breadcrumbs, page title, profile/menu placeholder, route layout and route-not-found page. Use approved design tokens, accessible navigation, light/dark themes and navigation placeholders for planned modules. No fake feature CRUD.

ACCEPTANCE CRITERIA:
Shell renders on desktop/tablet/mobile; navigation works; keyboard/focus states work; no placeholder navigation claims an unimplemented workflow is functional.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# B-04 — Go API bootstrap and OpenAPI

**Epic folder:** `01_Foundation_Security`  
**Priority:** P0  
**Depends on:** B-01

## Task scope

Bootstrap Gin in backend/cmd/api, /api/v1 routes, request IDs, structured logs, recovery middleware, timeout/graceful shutdown, consistent JSON errors and OpenAPI docs. Implement GET /health and GET /ready with readiness checking required dependencies when configured.

## Acceptance criteria

Go server starts; health endpoint responds; OpenAPI route is available; API errors have a consistent envelope; frontend calls real API.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: B-04 — Go API bootstrap and OpenAPI (BACKEND ONLY)
Priority: P0
Dependencies: B-01

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Bootstrap Gin in backend/cmd/api, /api/v1 routes, request IDs, structured logs, recovery middleware, timeout/graceful shutdown, consistent JSON errors and OpenAPI docs. Implement GET /health and GET /ready with readiness checking required dependencies when configured.

ACCEPTANCE CRITERIA:
Go server starts; health endpoint responds; OpenAPI route is available; API errors have a consistent envelope; frontend calls real API.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: B-04 — Go API bootstrap and OpenAPI (FRONTEND ONLY)
Priority: P0
Dependencies: B-01

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Connect shell status widget or a small developer diagnostics page to the real GET /api/v1/health endpoint via the shared API client. Show loading/error/offline states and no mock success. Keep UI simple.

ACCEPTANCE CRITERIA:
Go server starts; health endpoint responds; OpenAPI route is available; API errors have a consistent envelope; frontend calls real API.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# B-05 — PostgreSQL and migration baseline

**Epic folder:** `01_Foundation_Security`  
**Priority:** P0  
**Depends on:** B-01, B-04

## Task scope

Configure pgx pool, validated DATABASE_URL, migration tool and versioned SQL files; establish migration up/down scripts and safe seed mechanism. Never auto-drop/reset data on boot. Add health/readiness integration.

## Acceptance criteria

DB starts through Docker Compose; migrations apply/revert according to documented policy; bad configuration fails clearly; no destructive auto-reset.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: B-05 — PostgreSQL and migration baseline (BACKEND ONLY)
Priority: P0
Dependencies: B-01, B-04

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Configure pgx pool, validated DATABASE_URL, migration tool and versioned SQL files; establish migration up/down scripts and safe seed mechanism. Never auto-drop/reset data on boot. Add health/readiness integration.

ACCEPTANCE CRITERIA:
DB starts through Docker Compose; migrations apply/revert according to documented policy; bad configuration fails clearly; no destructive auto-reset.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: B-05 — PostgreSQL and migration baseline (FRONTEND ONLY)
Priority: P0
Dependencies: B-01, B-04

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Document local database startup and any frontend connection configuration needed; do not access PostgreSQL directly from the browser and do not add database-specific UI yet.

ACCEPTANCE CRITERIA:
DB starts through Docker Compose; migrations apply/revert according to documented policy; bad configuration fails clearly; no destructive auto-reset.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# B-06 — Authentication and session handling

**Epic folder:** `01_Foundation_Security`  
**Priority:** P0  
**Depends on:** B-04, B-05

## Task scope

Implement secure login/logout/session foundation only according to approved auth decision. Use password hashing if local credentials are approved, secure HttpOnly cookies, session revocation, CSRF protection where cookie auth requires it, rate limiting and generic login failures. Disabled users cannot authenticate. Do not hardcode admin credentials.

## Acceptance criteria

Unauthenticated API mutations are denied; disabled user cannot sign in; session invalidation works; UI never pretends a mocked session is real.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: B-06 — Authentication and session handling (BACKEND ONLY)
Priority: P0
Dependencies: B-04, B-05

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement secure login/logout/session foundation only according to approved auth decision. Use password hashing if local credentials are approved, secure HttpOnly cookies, session revocation, CSRF protection where cookie auth requires it, rate limiting and generic login failures. Disabled users cannot authenticate. Do not hardcode admin credentials.

ACCEPTANCE CRITERIA:
Unauthenticated API mutations are denied; disabled user cannot sign in; session invalidation works; UI never pretends a mocked session is real.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: B-06 — Authentication and session handling (FRONTEND ONLY)
Priority: P0
Dependencies: B-04, B-05

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Implement login/logout screens and session-aware route protection against the real API contract. Include invalid credentials, disabled account, rate-limit, expired session and network states. Never store session tokens in localStorage. If backend auth decision is blocked, implement only screens/API contract clearly marked nonfunctional and document blocker—do not fake auth success.

ACCEPTANCE CRITERIA:
Unauthenticated API mutations are denied; disabled user cannot sign in; session invalidation works; UI never pretends a mocked session is real.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# B-07 — RBAC and location scope

**Epic folder:** `01_Foundation_Security`  
**Priority:** P0  
**Depends on:** B-06, B-08

## Task scope

Implement the deny-by-default RBAC framework, permission mappings and assigned location scope at route and object/service level. The confirmed role names are Owner, Inventory Manager, Sales Executive, Accountant, and Viewer. Seed/represent these roles as appropriate, but do not guess the detailed grants or approval thresholds: record them as pending owner sign-off and keep unapproved role grants disabled for production mutations. Add policy helpers and negative tests for direct API access, privilege escalation and cross-location access. Include permission keys for cost prices, refunds, adjustments, user roles, invoice settings and exports without granting them by assumption.

## Acceptance criteria

Direct API requests without an explicitly approved permission are denied; confirmed role identities are represented; granular grants/approval thresholds are clearly marked pending owner sign-off; role/location scope is tested; frontend responds correctly to 401/403.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: B-07 — RBAC and location scope (BACKEND ONLY)
Priority: P0
Dependencies: B-06, B-08

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement the deny-by-default RBAC framework, permission mappings and assigned location scope at route and object/service level. The confirmed role names are Owner, Inventory Manager, Sales Executive, Accountant, and Viewer. Seed/represent these roles as appropriate, but do not guess the detailed grants or approval thresholds: record them as pending owner sign-off and keep unapproved role grants disabled for production mutations. Add policy helpers and negative tests for direct API access, privilege escalation and cross-location access. Include permission keys for cost prices, refunds, adjustments, user roles, invoice settings and exports without granting them by assumption.

ACCEPTANCE CRITERIA:
Direct API requests without an explicitly approved permission are denied; confirmed role identities are represented; granular grants/approval thresholds are clearly marked pending owner sign-off; role/location scope is tested; frontend responds correctly to 401/403.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: B-07 — RBAC and location scope (FRONTEND ONLY)
Priority: P0
Dependencies: B-06, B-08

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Render navigation and actions based on the permissions contract; provide access-denied states. Do not rely on hidden buttons as the security boundary; handle 401/403 from API consistently and avoid leaking sensitive fields.

ACCEPTANCE CRITERIA:
Direct API requests without an explicitly approved permission are denied; confirmed role identities are represented; granular grants/approval thresholds are clearly marked pending owner sign-off; role/location scope is tested; frontend responds correctly to 401/403.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# B-08 — Append-only audit event framework

**Epic folder:** `01_Foundation_Security`  
**Priority:** P0  
**Depends on:** B-04, B-05, B-07

## Task scope

Create audit-event model/table and service for actor, action, entity type/id, reference, timestamp, request ID and safe before/after summary where allowed. Protect audit rows from normal API update/delete. Add helper for sensitive mutations but do not instrument unrelated modules beyond a thin integration example. Avoid secrets/PII in payloads.

## Acceptance criteria

A test proves audit entries are append-only from normal application paths; sensitive values are not logged; reusable service is documented.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: B-08 — Append-only audit event framework (BACKEND ONLY)
Priority: P0
Dependencies: B-04, B-05, B-07

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Create audit-event model/table and service for actor, action, entity type/id, reference, timestamp, request ID and safe before/after summary where allowed. Protect audit rows from normal API update/delete. Add helper for sensitive mutations but do not instrument unrelated modules beyond a thin integration example. Avoid secrets/PII in payloads.

ACCEPTANCE CRITERIA:
A test proves audit entries are append-only from normal application paths; sensitive values are not logged; reusable service is documented.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: B-08 — Append-only audit event framework (FRONTEND ONLY)
Priority: P0
Dependencies: B-04, B-05, B-07

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
No audit UI is required unless already in scope. If an existing activity component is present, show a simple permission-aware read-only placeholder contract only; do not create a full audit screen until the audit-list task.

ACCEPTANCE CRITERIA:
A test proves audit entries are append-only from normal application paths; sensitive values are not logged; reusable service is documented.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# B-09 — Private attachment service

**Epic folder:** `01_Foundation_Security`  
**Priority:** P0  
**Depends on:** B-04, B-05, B-07

## Task scope

Implement private attachment metadata and storage abstraction, approved local dev storage or configured object storage, MIME/size limits, generated safe object keys, upload/download authorization and signed or proxied private access. Never return a permanently public URL.

## Acceptance criteria

Unauthorized downloads fail; invalid MIME/oversized files are rejected; metadata and file access are permission-checked; UI displays upload errors.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: B-09 — Private attachment service (BACKEND ONLY)
Priority: P0
Dependencies: B-04, B-05, B-07

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement private attachment metadata and storage abstraction, approved local dev storage or configured object storage, MIME/size limits, generated safe object keys, upload/download authorization and signed or proxied private access. Never return a permanently public URL.

ACCEPTANCE CRITERIA:
Unauthorized downloads fail; invalid MIME/oversized files are rejected; metadata and file access are permission-checked; UI displays upload errors.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: B-09 — Private attachment service (FRONTEND ONLY)
Priority: P0
Dependencies: B-04, B-05, B-07

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Create reusable uploader/document row/image preview components only, with upload progress, size/type validation, retry and permission-denied states. Wire the real API contract if available; do not expose private storage URLs.

ACCEPTANCE CRITERIA:
Unauthorized downloads fail; invalid MIME/oversized files are rejected; metadata and file access are permission-checked; UI displays upload errors.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# EPIC C — Catalogue & Brands


# C-01 — Brand/model/variant database schema

**Epic folder:** `02_Catalogue_Brands`  
**Priority:** P0  
**Depends on:** B-05

## Task scope

Add versioned migrations for brands, models, variants, colours/spec attributes and provenance/verification fields based on the approved architecture. Use stable IDs, foreign keys, brand-scoped code uniqueness, lifecycle status, authorisation status/evidence metadata and timestamps. Do not seed invented catalogue data.

## Acceptance criteria

Migrations support multiple brands and enforce relationship integrity; another brand can be added without schema/code changes; migration tests/checks pass.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: C-01 — Brand/model/variant database schema (BACKEND ONLY)
Priority: P0
Dependencies: B-05

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Add versioned migrations for brands, models, variants, colours/spec attributes and provenance/verification fields based on the approved architecture. Use stable IDs, foreign keys, brand-scoped code uniqueness, lifecycle status, authorisation status/evidence metadata and timestamps. Do not seed invented catalogue data.

ACCEPTANCE CRITERIA:
Migrations support multiple brands and enforce relationship integrity; another brand can be added without schema/code changes; migration tests/checks pass.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: C-01 — Brand/model/variant database schema (FRONTEND ONLY)
Priority: P0
Dependencies: B-05

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Create typed domain contracts mirroring the documented API/schema (or update existing generated contracts). Do not hardcode brand enums or add UI CRUD in this schema-only task.

ACCEPTANCE CRITERIA:
Migrations support multiple brands and enforce relationship integrity; another brand can be added without schema/code changes; migration tests/checks pass.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# C-02 — Catalogue API and validation

**Epic folder:** `02_Catalogue_Brands`  
**Priority:** P0  
**Depends on:** C-01, B-07, B-08

## Task scope

Implement REST handlers/services for brand, model, variant and colour CRUD/list/search with pagination, filtering, archive semantics, verification/provenance fields and permission checks. Prevent invalid cross-brand model/variant relationships and prevent purchasing/sales activation without required verification approval. Audit sensitive create/update/activate/archive actions.

## Acceptance criteria

API validates brand relationships and permissions; archive avoids breaking historical records; invalid activation is denied; OpenAPI and tests updated.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: C-02 — Catalogue API and validation (BACKEND ONLY)
Priority: P0
Dependencies: C-01, B-07, B-08

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement REST handlers/services for brand, model, variant and colour CRUD/list/search with pagination, filtering, archive semantics, verification/provenance fields and permission checks. Prevent invalid cross-brand model/variant relationships and prevent purchasing/sales activation without required verification approval. Audit sensitive create/update/activate/archive actions.

ACCEPTANCE CRITERIA:
API validates brand relationships and permissions; archive avoids breaking historical records; invalid activation is denied; OpenAPI and tests updated.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: C-02 — Catalogue API and validation (FRONTEND ONLY)
Priority: P0
Dependencies: C-01, B-07, B-08

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Implement typed API service functions and query/mutation hooks for catalogue endpoints only. Ensure contracts match OpenAPI. Do not build final catalogue screens unless C-03 is being worked separately.

ACCEPTANCE CRITERIA:
API validates brand relationships and permissions; archive avoids breaking historical records; invalid activation is denied; OpenAPI and tests updated.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# C-03 — Catalogue and brand management UI

**Epic folder:** `02_Catalogue_Brands`  
**Priority:** P0  
**Depends on:** C-02, B-03

## Task scope

Provide any missing list/detail APIs needed by the UI only if within this task and coordinate with the C-02 contract; avoid duplicating API logic.

## Acceptance criteria

User can manage catalogue through real APIs; no unverified item appears operationally active; cross-brand model choices are impossible in UI and still rejected server-side.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: C-03 — Catalogue and brand management UI (BACKEND ONLY)
Priority: P0
Dependencies: C-02, B-03

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Provide any missing list/detail APIs needed by the UI only if within this task and coordinate with the C-02 contract; avoid duplicating API logic.

ACCEPTANCE CRITERIA:
User can manage catalogue through real APIs; no unverified item appears operationally active; cross-brand model choices are impossible in UI and still rejected server-side.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: C-03 — Catalogue and brand management UI (FRONTEND ONLY)
Priority: P0
Dependencies: C-02, B-03

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build brand list/editor and model/variant catalogue screens: search, pagination, lifecycle/verification badges, supplier relationship display, create/edit forms and archive confirmation. Choosing a brand filters model/variant options to that brand. Show warnings for unverified brand/price/specs and require confirmation plus permission for activation. Include all loading/empty/error states.

ACCEPTANCE CRITERIA:
User can manage catalogue through real APIs; no unverified item appears operationally active; cross-brand model choices are impossible in UI and still rejected server-side.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# C-04 — Effective-dated price history

**Epic folder:** `02_Catalogue_Brands`  
**Priority:** P0  
**Depends on:** C-01, C-02

## Task scope

Create price-list/price-list-item history with effective date range, type, amount, tax config reference, approval actor/status and overlap/validity validation. Never overwrite previously approved price records silently. Support retrieving price valid at quotation time. Add tests.

## Acceptance criteria

Price history is immutable after approval except through controlled supersession; effective-dated lookup is test-covered; old quotations remain stable.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: C-04 — Effective-dated price history (BACKEND ONLY)
Priority: P0
Dependencies: C-01, C-02

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Create price-list/price-list-item history with effective date range, type, amount, tax config reference, approval actor/status and overlap/validity validation. Never overwrite previously approved price records silently. Support retrieving price valid at quotation time. Add tests.

ACCEPTANCE CRITERIA:
Price history is immutable after approval except through controlled supersession; effective-dated lookup is test-covered; old quotations remain stable.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: C-04 — Effective-dated price history (FRONTEND ONLY)
Priority: P0
Dependencies: C-01, C-02

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Create price-history table/detail or editor if design specifies. Show current/future/past price, effective dates and approval status; warn that updating a price does not mutate existing quotation snapshots. Keep cost price visible only to authorised roles.

ACCEPTANCE CRITERIA:
Price history is immutable after approval except through controlled supersession; effective-dated lookup is test-covered; old quotations remain stable.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# C-05 — Seed verified multi-brand catalogue

**Epic folder:** `02_Catalogue_Brands`  
**Priority:** P0  
**Depends on:** A-01, C-01, C-02

## Task scope

Create an idempotent seed process for the client-confirmed initial brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. The client intends to operate all three and add brands later; this is not proof of commercial/dealership authorisation. Create the brand master records without inventing legal supplier identity or authorisation evidence. Keep brand existence/lifecycle separate from purchase/sales eligibility. Until required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices, warranty and identifier rules, approver and verification date are present, retain the corresponding brand or catalogue status as PENDING_VERIFICATION (or the approved equivalent) and block purchase/sales activation. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity. Import models/specs/prices only from client/supplier-confirmed sources with provenance and verification dates. Reruns must not duplicate records.

## Acceptance criteria

Seed process is idempotent; exactly the three client-confirmed brand master records can be created without fabricated catalogue/stock data; brand existence is distinct from approved commercial activation; unverified brands/models/prices/warranties remain blocked from operational purchase/sales.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: C-05 — Seed verified multi-brand catalogue (BACKEND ONLY)
Priority: P0
Dependencies: A-01, C-01, C-02

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Create an idempotent seed process for the client-confirmed initial brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. The client intends to operate all three and add brands later; this is not proof of commercial/dealership authorisation. Create the brand master records without inventing legal supplier identity or authorisation evidence. Keep brand existence/lifecycle separate from purchase/sales eligibility. Until required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices, warranty and identifier rules, approver and verification date are present, retain the corresponding brand or catalogue status as PENDING_VERIFICATION (or the approved equivalent) and block purchase/sales activation. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity. Import models/specs/prices only from client/supplier-confirmed sources with provenance and verification dates. Reruns must not duplicate records.

ACCEPTANCE CRITERIA:
Seed process is idempotent; exactly the three client-confirmed brand master records can be created without fabricated catalogue/stock data; brand existence is distinct from approved commercial activation; unverified brands/models/prices/warranties remain blocked from operational purchase/sales.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: C-05 — Seed verified multi-brand catalogue (FRONTEND ONLY)
Priority: P0
Dependencies: A-01, C-01, C-02

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
If showing seeded brands, clearly identify Lectrix EV, Ceeon India and Ujagar Electric as the client-confirmed initial roster and show the API-provided verification/activation status. Do not imply authorisation from roster confirmation. Distinguish approved active catalogue from pending verification. Do not embed models/prices in frontend constants or show fabricated seed values.

ACCEPTANCE CRITERIA:
Seed process is idempotent; exactly the three client-confirmed brand master records can be created without fabricated catalogue/stock data; brand existence is distinct from approved commercial activation; unverified brands/models/prices/warranties remain blocked from operational purchase/sales.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# C-06 — Brand-specific rules and isolation tests

**Epic folder:** `02_Catalogue_Brands`  
**Priority:** P0  
**Depends on:** C-02, C-04, B-07

## Task scope

Add tests ensuring price, warranty, identifier rules, supplier mappings and catalogue relationships remain scoped to the selected brand. Reject cross-brand model/variant IDs; ensure filters and data export queries cannot leak data outside authorized scope. Implement configurable rule resolution without hardcoding individual brand names.

## Acceptance criteria

Cross-brand mismatch requests fail; brand filters/export results stay scoped; adding an arbitrary brand does not require code changes.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: C-06 — Brand-specific rules and isolation tests (BACKEND ONLY)
Priority: P0
Dependencies: C-02, C-04, B-07

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Add tests ensuring price, warranty, identifier rules, supplier mappings and catalogue relationships remain scoped to the selected brand. Reject cross-brand model/variant IDs; ensure filters and data export queries cannot leak data outside authorized scope. Implement configurable rule resolution without hardcoding individual brand names.

ACCEPTANCE CRITERIA:
Cross-brand mismatch requests fail; brand filters/export results stay scoped; adding an arbitrary brand does not require code changes.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: C-06 — Brand-specific rules and isolation tests (FRONTEND ONLY)
Priority: P0
Dependencies: C-02, C-04, B-07

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Add UI tests that selecting a brand constrains model/variant choices and that list/report filters keep brand selection consistent. Test activation warnings and permission-based cost/authorisation details.

ACCEPTANCE CRITERIA:
Cross-brand mismatch requests fail; brand filters/export results stay scoped; adding an arbitrary brand does not require code changes.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# EPIC D — Procurement & PDI


# D-01 — Supplier master

**Epic folder:** `03_Procurement_PDI`  
**Priority:** P0  
**Depends on:** B-05, B-07, B-08

## Task scope

Create supplier schema, migrations and API for supplier legal name, contacts, address, GSTIN when applicable, document references, status and brand relationships. Validate fields; support archive rather than delete; audit sensitive changes.

## Acceptance criteria

Supplier records are searchable; archive preserves history; brand association works for multiple brands; API authorization/tests pass.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: D-01 — Supplier master (BACKEND ONLY)
Priority: P0
Dependencies: B-05, B-07, B-08

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Create supplier schema, migrations and API for supplier legal name, contacts, address, GSTIN when applicable, document references, status and brand relationships. Validate fields; support archive rather than delete; audit sensitive changes.

ACCEPTANCE CRITERIA:
Supplier records are searchable; archive preserves history; brand association works for multiple brands; API authorization/tests pass.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: D-01 — Supplier master (FRONTEND ONLY)
Priority: P0
Dependencies: B-05, B-07, B-08

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build supplier list/detail/form with search, status filter, contact/document sections, brand relationship picker and archive confirmation. Keep sensitive details permission-aware, with loading/empty/error handling.

ACCEPTANCE CRITERIA:
Supplier records are searchable; archive preserves history; brand association works for multiple brands; API authorization/tests pass.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# D-02 — Purchase order lifecycle

**Epic folder:** `03_Procurement_PDI`  
**Priority:** P0  
**Depends on:** C-02, D-01, C-04, B-07

## Task scope

Implement PO number generation, line items, quantity/cost snapshots, expected date, submission/approval status transitions, creator and history. Enforce allowed transitions, approval threshold config and brand/model/variant compatibility. Calculate remaining quantity server-side.

## Acceptance criteria

PO state transitions are validated; line costs/prices are snapshots; unauthorized approvals fail; list/detail work against real API.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: D-02 — Purchase order lifecycle (BACKEND ONLY)
Priority: P0
Dependencies: C-02, D-01, C-04, B-07

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement PO number generation, line items, quantity/cost snapshots, expected date, submission/approval status transitions, creator and history. Enforce allowed transitions, approval threshold config and brand/model/variant compatibility. Calculate remaining quantity server-side.

ACCEPTANCE CRITERIA:
PO state transitions are validated; line costs/prices are snapshots; unauthorized approvals fail; list/detail work against real API.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: D-02 — Purchase order lifecycle (FRONTEND ONLY)
Priority: P0
Dependencies: C-02, D-01, C-04, B-07

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build PO list, create/edit draft, detail page, line editor, submit/approve controls and status history. Show over-receipt/price variance/partial delivery warnings. Restrict approval buttons by permission and display API validation details.

ACCEPTANCE CRITERIA:
PO state transitions are validated; line costs/prices are snapshots; unauthorized approvals fail; list/detail work against real API.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# D-03 — Goods receipt draft workflow

**Epic folder:** `03_Procurement_PDI`  
**Priority:** P0  
**Depends on:** D-02, B-09, C-01

## Task scope

Implement draft GRN linked to PO with partial deliveries, per-unit serial capture, accepted/rejected quantities, attachments and discrepancy notes. Do not post stock in draft. Validate PO remaining quantity and require permissioned override/reason for over-receipt.

## Acceptance criteria

Partial receipt supported; drafts do not alter saleable stock; over-receipt requires authorized override; serial validation visible.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: D-03 — Goods receipt draft workflow (BACKEND ONLY)
Priority: P0
Dependencies: D-02, B-09, C-01

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement draft GRN linked to PO with partial deliveries, per-unit serial capture, accepted/rejected quantities, attachments and discrepancy notes. Do not post stock in draft. Validate PO remaining quantity and require permissioned override/reason for over-receipt.

ACCEPTANCE CRITERIA:
Partial receipt supported; drafts do not alter saleable stock; over-receipt requires authorized override; serial validation visible.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: D-03 — Goods receipt draft workflow (FRONTEND ONLY)
Priority: P0
Dependencies: D-02, B-09, C-01

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build GRN stepper: choose PO, confirm delivered lines, capture each unit identifiers, attach supplier documents, enter discrepancy notes and save draft. Clearly separate Save Draft from Post Receipt and show duplicate serial validation. Preserve draft data on safe validation errors.

ACCEPTANCE CRITERIA:
Partial receipt supported; drafts do not alter saleable stock; over-receipt requires authorized override; serial validation visible.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# D-04 — Vehicle unit registry

**Epic folder:** `03_Procurement_PDI`  
**Priority:** P0  
**Depends on:** C-01, B-05

## Task scope

Create vehicle_units schema/API foundations for internal ID, variant, normalized chassis/frame, motor/battery/charger serials where applicable, location, supplier/PO/GRN refs, receipt date, cost permissions and lifecycle status. Add constraints and query indexes supported by approved rules.

## Acceptance criteria

Each vehicle is individually represented; required identifiers validated; unit history links to source documents; sensitive cost fields are permissioned.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: D-04 — Vehicle unit registry (BACKEND ONLY)
Priority: P0
Dependencies: C-01, B-05

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Create vehicle_units schema/API foundations for internal ID, variant, normalized chassis/frame, motor/battery/charger serials where applicable, location, supplier/PO/GRN refs, receipt date, cost permissions and lifecycle status. Add constraints and query indexes supported by approved rules.

ACCEPTANCE CRITERIA:
Each vehicle is individually represented; required identifiers validated; unit history links to source documents; sensitive cost fields are permissioned.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: D-04 — Vehicle unit registry (FRONTEND ONLY)
Priority: P0
Dependencies: C-01, B-05

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Create TypeScript vehicle-unit types and API hooks only, or add a minimal unit capture form if the receipt UI depends on it. Do not build the complete vehicle list/detail screen (D-08).

ACCEPTANCE CRITERIA:
Each vehicle is individually represented; required identifiers validated; unit history links to source documents; sensitive cost fields are permissioned.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# D-05 — Duplicate vehicle identifier controls

**Epic folder:** `03_Procurement_PDI`  
**Priority:** P0  
**Depends on:** D-04

## Task scope

Enforce normalized chassis/frame uniqueness at DB level and re-check in service layer. Apply configured uniqueness policies to motor/battery identifiers. Return a stable, safe conflict error that lets UI identify which field conflicts without exposing unrelated owner data. Add concurrent insert tests.

## Acceptance criteria

Duplicate chassis cannot be saved, including concurrent attempts; friendly validation maps to the correct field; no bypass through UI.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: D-05 — Duplicate vehicle identifier controls (BACKEND ONLY)
Priority: P0
Dependencies: D-04

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Enforce normalized chassis/frame uniqueness at DB level and re-check in service layer. Apply configured uniqueness policies to motor/battery identifiers. Return a stable, safe conflict error that lets UI identify which field conflicts without exposing unrelated owner data. Add concurrent insert tests.

ACCEPTANCE CRITERIA:
Duplicate chassis cannot be saved, including concurrent attempts; friendly validation maps to the correct field; no bypass through UI.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: D-05 — Duplicate vehicle identifier controls (FRONTEND ONLY)
Priority: P0
Dependencies: D-04

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Normalize identifiers consistently before submission, display field-level duplicate errors, provide a safe path to open an existing unit only when user is authorized, and preserve other form values. Test both immediate API conflict and race conflict.

ACCEPTANCE CRITERIA:
Duplicate chassis cannot be saved, including concurrent attempts; friendly validation maps to the correct field; no bypass through UI.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# D-06 — PDI checklist and disposition

**Epic folder:** `03_Procurement_PDI`  
**Priority:** P0  
**Depends on:** D-04, B-09, B-07, B-08

## Task scope

Implement PDI records/checklist with actor, timestamp, inspection result, defect notes, photos and disposition PASS/HOLD/REJECT. A failed/held PDI moves unit to non-saleable state and records audit/movement through approved domain service.

## Acceptance criteria

PDI decision is attributable; failed/held units excluded from saleable search; attached evidence remains private; statuses are tested.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: D-06 — PDI checklist and disposition (BACKEND ONLY)
Priority: P0
Dependencies: D-04, B-09, B-07, B-08

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement PDI records/checklist with actor, timestamp, inspection result, defect notes, photos and disposition PASS/HOLD/REJECT. A failed/held PDI moves unit to non-saleable state and records audit/movement through approved domain service.

ACCEPTANCE CRITERIA:
PDI decision is attributable; failed/held units excluded from saleable search; attached evidence remains private; statuses are tested.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: D-06 — PDI checklist and disposition (FRONTEND ONLY)
Priority: P0
Dependencies: D-04, B-09, B-07, B-08

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build PDI checklist form with configurable checklist items where specified, pass/hold/reject outcomes, notes, photo attachments and confirmation of disposition. Make held/rejected states visually clear and block reservation/sale actions.

ACCEPTANCE CRITERIA:
PDI decision is attributable; failed/held units excluded from saleable search; attached evidence remains private; statuses are tested.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# D-07 — Atomic receipt posting transaction

**Epic folder:** `03_Procurement_PDI`  
**Priority:** P0  
**Depends on:** D-03, D-04, D-05, D-06, E-01

## Task scope

Implement GRN posting as a single DB transaction: lock/revalidate PO remaining quantities, validate every unit identifier, create accepted unit records and movement entries, update PO receipt totals/status and mark GRN posted. Use idempotency/request key; a failed post must leave no partial stock or ledger changes.

## Acceptance criteria

Repeated post cannot duplicate stock; any validation/DB failure rolls back every affected entity; frontend reflects committed state only.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: D-07 — Atomic receipt posting transaction (BACKEND ONLY)
Priority: P0
Dependencies: D-03, D-04, D-05, D-06, E-01

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement GRN posting as a single DB transaction: lock/revalidate PO remaining quantities, validate every unit identifier, create accepted unit records and movement entries, update PO receipt totals/status and mark GRN posted. Use idempotency/request key; a failed post must leave no partial stock or ledger changes.

ACCEPTANCE CRITERIA:
Repeated post cannot duplicate stock; any validation/DB failure rolls back every affected entity; frontend reflects committed state only.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: D-07 — Atomic receipt posting transaction (FRONTEND ONLY)
Priority: P0
Dependencies: D-03, D-04, D-05, D-06, E-01

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Wire Post Receipt to the real API with disabled/loading state, impact summary, confirmation and clear conflict/retry handling. Refresh PO/GRN/vehicle queries only after success; do not optimistic-update saleable stock before commit.

ACCEPTANCE CRITERIA:
Repeated post cannot duplicate stock; any validation/DB failure rolls back every affected entity; frontend reflects committed state only.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# D-08 — Vehicle list and detail UI

**Epic folder:** `03_Procurement_PDI`  
**Priority:** P0  
**Depends on:** D-04, D-05, D-06, E-02

## Task scope

Complete filtered/paginated vehicle list and detail APIs if missing: brand/model/variant/colour/status/location/supplier/date/age filters, search serials, timeline and permission-filtered cost. Ensure query performance and object-level authorization.

## Acceptance criteria

Search works across approved identifiers; filters combine correctly; cost/attachments are protected; detail history is navigable.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: D-08 — Vehicle list and detail UI (BACKEND ONLY)
Priority: P0
Dependencies: D-04, D-05, D-06, E-02

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Complete filtered/paginated vehicle list and detail APIs if missing: brand/model/variant/colour/status/location/supplier/date/age filters, search serials, timeline and permission-filtered cost. Ensure query performance and object-level authorization.

ACCEPTANCE CRITERIA:
Search works across approved identifiers; filters combine correctly; cost/attachments are protected; detail history is navigable.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: D-08 — Vehicle list and detail UI (FRONTEND ONLY)
Priority: P0
Dependencies: D-04, D-05, D-06, E-02

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build inventory vehicle list/detail screens with server pagination, persistent filters, chassis/motor/battery search, status labels, internal ID/QR, timeline, PDI/procurement links, attachments and permission-aware actions. Include empty/no-results/loading/error/403/stale states.

ACCEPTANCE CRITERIA:
Search works across approved identifiers; filters combine correctly; cost/attachments are protected; detail history is navigable.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# D-09 — QR/barcode lookup and print

**Epic folder:** `03_Procurement_PDI`  
**Priority:** P0  
**Depends on:** D-04, D-08, B-07

## Task scope

Provide permission-checked lookup by internal unit ID/opaque token and a printable QR payload that does not expose personal/sensitive data. The QR is only a lookup token, not authorization. Validate malformed/unknown tokens.

## Acceptance criteria

QR resolves to an authorized unit page; direct guessing/scanning cannot bypass permissions; manual search remains available.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: D-09 — QR/barcode lookup and print (BACKEND ONLY)
Priority: P0
Dependencies: D-04, D-08, B-07

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Provide permission-checked lookup by internal unit ID/opaque token and a printable QR payload that does not expose personal/sensitive data. The QR is only a lookup token, not authorization. Validate malformed/unknown tokens.

ACCEPTANCE CRITERIA:
QR resolves to an authorized unit page; direct guessing/scanning cannot bypass permissions; manual search remains available.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: D-09 — QR/barcode lookup and print (FRONTEND ONLY)
Priority: P0
Dependencies: D-04, D-08, B-07

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Add QR code display/print/scan entry and manual-entry fallback to the unit detail screen. Provide scanner-friendly input that works with keyboard-emulating scanners; show unknown/unauthorized outcomes clearly.

ACCEPTANCE CRITERIA:
QR resolves to an authorized unit page; direct guessing/scanning cannot bypass permissions; manual search remains available.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# EPIC E — Inventory & Stock


# E-01 — Append-only stock movement API

**Epic folder:** `04_Inventory_Stock`  
**Priority:** P0  
**Depends on:** D-04, B-08

## Task scope

Implement movement service and schema storing item/unit, movement type, quantity/status/location before and after, source reference, actor, timestamp, idempotency key and reversal link. Prevent normal updates/deletes; status/location changes must use the service. Keep unit snapshot and ledger reconcilable.

## Acceptance criteria

All qualifying changes create exactly one movement; movement records cannot be edited/deleted via ordinary APIs; idempotency tested.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: E-01 — Append-only stock movement API (BACKEND ONLY)
Priority: P0
Dependencies: D-04, B-08

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement movement service and schema storing item/unit, movement type, quantity/status/location before and after, source reference, actor, timestamp, idempotency key and reversal link. Prevent normal updates/deletes; status/location changes must use the service. Keep unit snapshot and ledger reconcilable.

ACCEPTANCE CRITERIA:
All qualifying changes create exactly one movement; movement records cannot be edited/deleted via ordinary APIs; idempotency tested.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: E-01 — Append-only stock movement API (FRONTEND ONLY)
Priority: P0
Dependencies: D-04, B-08

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Expose typed movement contracts and query hooks; do not create timeline UI in this task (E-02). Ensure frontend mutation patterns do not update stock directly without invoking the approved backend operation.

ACCEPTANCE CRITERIA:
All qualifying changes create exactly one movement; movement records cannot be edited/deleted via ordinary APIs; idempotency tested.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# E-02 — Stock movement timeline UI

**Epic folder:** `04_Inventory_Stock`  
**Priority:** P0  
**Depends on:** E-01, D-08

## Task scope

Provide paginated, filterable movement-list/detail endpoint if missing, with permission-aware actor/reference fields. Avoid mutable audit behavior and apply location/brand access scope.

## Acceptance criteria

Timeline matches ledger data and filters; unauthorized linked records are not leaked; no edit/delete actions appear.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: E-02 — Stock movement timeline UI (BACKEND ONLY)
Priority: P0
Dependencies: E-01, D-08

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Provide paginated, filterable movement-list/detail endpoint if missing, with permission-aware actor/reference fields. Avoid mutable audit behavior and apply location/brand access scope.

ACCEPTANCE CRITERIA:
Timeline matches ledger data and filters; unauthorized linked records are not leaked; no edit/delete actions appear.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: E-02 — Stock movement timeline UI (FRONTEND ONLY)
Priority: P0
Dependencies: E-01, D-08

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build a timeline/table with filters for unit, movement type, actor, location, date and source reference. Link to vehicle/PO/GRN/sale where authorized; include clear loading/empty/error/403 states and show timestamps in documented timezone.

ACCEPTANCE CRITERIA:
Timeline matches ledger data and filters; unauthorized linked records are not leaked; no edit/delete actions appear.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# E-03 — Spare/accessory SKU master

**Epic folder:** `04_Inventory_Stock`  
**Priority:** P0  
**Depends on:** B-05, D-01, B-07

## Task scope

Create SKU and item category models for parts/accessories: unique SKU, UOM, tracking mode, supplier links, reorder level and optional compatible model/brand metadata. Keep quantity-tracked and serial-tracked behavior distinct.

## Acceptance criteria

SKU uniqueness and required fields are enforced; compatibility metadata is brand-aware; archived SKUs preserve history.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: E-03 — Spare/accessory SKU master (BACKEND ONLY)
Priority: P0
Dependencies: B-05, D-01, B-07

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Create SKU and item category models for parts/accessories: unique SKU, UOM, tracking mode, supplier links, reorder level and optional compatible model/brand metadata. Keep quantity-tracked and serial-tracked behavior distinct.

ACCEPTANCE CRITERIA:
SKU uniqueness and required fields are enforced; compatibility metadata is brand-aware; archived SKUs preserve history.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: E-03 — Spare/accessory SKU master (FRONTEND ONLY)
Priority: P0
Dependencies: B-05, D-01, B-07

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Create parts/accessories list and editor with SKU, category, UOM, tracking mode, supplier, compatible brand/model metadata, reorder level and archive status. Do not implement receipts/issues in this task.

ACCEPTANCE CRITERIA:
SKU uniqueness and required fields are enforced; compatibility metadata is brand-aware; archived SKUs preserve history.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# E-04 — Parts receipts and issues

**Epic folder:** `04_Inventory_Stock`  
**Priority:** P0  
**Depends on:** E-03, E-01, D-01

## Task scope

Implement stock-in and stock-out operations with quantity ledger, source reference, actor, reason and idempotency. Block negative stock; validate unit of measure; support supplier receipt and authorized issue/consumption reason.

## Acceptance criteria

No negative stock; every posted receipt/issue generates ledger movement; duplicate requests are safe; balance is server-authoritative.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: E-04 — Parts receipts and issues (BACKEND ONLY)
Priority: P0
Dependencies: E-03, E-01, D-01

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement stock-in and stock-out operations with quantity ledger, source reference, actor, reason and idempotency. Block negative stock; validate unit of measure; support supplier receipt and authorized issue/consumption reason.

ACCEPTANCE CRITERIA:
No negative stock; every posted receipt/issue generates ledger movement; duplicate requests are safe; balance is server-authoritative.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: E-04 — Parts receipts and issues (FRONTEND ONLY)
Priority: P0
Dependencies: E-03, E-01, D-01

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build stock receipt/issue forms and transaction history for parts. Show available balance, required reason/reference, validation and insufficient-stock response. Do not allow the client to set the balance directly.

ACCEPTANCE CRITERIA:
No negative stock; every posted receipt/issue generates ledger movement; duplicate requests are safe; balance is server-authoritative.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# E-05 — Stock count and adjustment approval

**Epic folder:** `04_Inventory_Stock`  
**Priority:** P0  
**Depends on:** E-01, B-07, B-08

## Task scope

Implement stock count/adjustment request lifecycle with observed quantity/status, variance, mandatory reason, submitter/approver and approval threshold. Prevent self-approval when required. Posting approval creates movement; rejected/pending adjustment does not change stock.

## Acceptance criteria

Pending adjustment never changes balance; authorized approval posts a movement; self-approval and unauthorized adjustments fail.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: E-05 — Stock count and adjustment approval (BACKEND ONLY)
Priority: P0
Dependencies: E-01, B-07, B-08

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement stock count/adjustment request lifecycle with observed quantity/status, variance, mandatory reason, submitter/approver and approval threshold. Prevent self-approval when required. Posting approval creates movement; rejected/pending adjustment does not change stock.

ACCEPTANCE CRITERIA:
Pending adjustment never changes balance; authorized approval posts a movement; self-approval and unauthorized adjustments fail.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: E-05 — Stock count and adjustment approval (FRONTEND ONLY)
Priority: P0
Dependencies: E-01, B-07, B-08

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build count session, variance entry, reason capture, pending approvals list, approve/reject flow and history. Display old/new quantity/status and require confirmation. Hide or disable self-approval according to permission, but rely on API enforcement.

ACCEPTANCE CRITERIA:
Pending adjustment never changes balance; authorized approval posts a movement; self-approval and unauthorized adjustments fail.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# E-06 — Ledger reconciliation job/report

**Epic folder:** `04_Inventory_Stock`  
**Priority:** P0  
**Depends on:** E-01, D-04, E-04

## Task scope

Implement read-only reconciliation comparing vehicle unit state/location and parts balances to movement history; return discrepancy records and a run summary. Do not silently repair; any correction must use approved adjustment/reversal flows. Provide safe rerun and logging.

## Acceptance criteria

Injected discrepancy is detected; report is repeatable; no mutation occurs during reconciliation; results are permission-checked.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: E-06 — Ledger reconciliation job/report (BACKEND ONLY)
Priority: P0
Dependencies: E-01, D-04, E-04

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement read-only reconciliation comparing vehicle unit state/location and parts balances to movement history; return discrepancy records and a run summary. Do not silently repair; any correction must use approved adjustment/reversal flows. Provide safe rerun and logging.

ACCEPTANCE CRITERIA:
Injected discrepancy is detected; report is repeatable; no mutation occurs during reconciliation; results are permission-checked.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: E-06 — Ledger reconciliation job/report (FRONTEND ONLY)
Priority: P0
Dependencies: E-01, D-04, E-04

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Create a read-only reconciliation result page/table with run time, discrepancy type, source IDs, severity and links to authorized records. Do not provide a one-click silent repair action.

ACCEPTANCE CRITERIA:
Injected discrepancy is detected; report is repeatable; no mutation occurs during reconciliation; results are permission-checked.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# E-07 — Stock ageing calculations

**Epic folder:** `04_Inventory_Stock`  
**Priority:** P0  
**Depends on:** D-04, D-07

## Task scope

Implement stock-age calculation based on accepted GRN date or approved opening-stock date; configurable bands; exclude/label units without reliable dates rather than inventing dates. Add query/report fields and boundary tests at band edges.

## Acceptance criteria

Age uses approved origin date; boundary behavior matches tests; filters and displayed values agree with backend.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: E-07 — Stock ageing calculations (BACKEND ONLY)
Priority: P0
Dependencies: D-04, D-07

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement stock-age calculation based on accepted GRN date or approved opening-stock date; configurable bands; exclude/label units without reliable dates rather than inventing dates. Add query/report fields and boundary tests at band edges.

ACCEPTANCE CRITERIA:
Age uses approved origin date; boundary behavior matches tests; filters and displayed values agree with backend.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: E-07 — Stock ageing calculations (FRONTEND ONLY)
Priority: P0
Dependencies: D-04, D-07

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Show age in days and ageing-band filter/badge in vehicle list and report surfaces as specified; use clear fallback label for missing origin date; do not calculate differently in the browser from server rules.

ACCEPTANCE CRITERIA:
Age uses approved origin date; boundary behavior matches tests; filters and displayed values agree with backend.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# EPIC F — Sales & Delivery


# F-01 — Customer master

**Epic folder:** `05_Sales_Delivery`  
**Priority:** P0  
**Depends on:** B-06, B-07, B-08

## Task scope

Implement customer model/API with minimal necessary identity/contact fields, normalized phone search, duplicate warnings, safe field validation and role-based visibility. Record lawful-basis/marketing consent separately only if required; do not store unnecessary sensitive data.

## Acceptance criteria

Phone/name search works; duplicate is warned/handled according to policy; protected fields require permissions; tests cover validation.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: F-01 — Customer master (BACKEND ONLY)
Priority: P0
Dependencies: B-06, B-07, B-08

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement customer model/API with minimal necessary identity/contact fields, normalized phone search, duplicate warnings, safe field validation and role-based visibility. Record lawful-basis/marketing consent separately only if required; do not store unnecessary sensitive data.

ACCEPTANCE CRITERIA:
Phone/name search works; duplicate is warned/handled according to policy; protected fields require permissions; tests cover validation.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: F-01 — Customer master (FRONTEND ONLY)
Priority: P0
Dependencies: B-06, B-07, B-08

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build customer search/list/create/edit/select flows integrated with API; duplicate warning, validation and permission states. Avoid exposing customer details in broad unfiltered tables and ensure forms explain required fields.

ACCEPTANCE CRITERIA:
Phone/name search works; duplicate is warned/handled according to policy; protected fields require permissions; tests cover validation.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# F-02 — Quotation builder and price snapshot

**Epic folder:** `05_Sales_Delivery`  
**Priority:** P0  
**Depends on:** C-04, F-01, D-04, B-07

## Task scope

Implement quotation create/update/preview APIs with customer, exact unit or approved item allocation, approved effective price snapshot, validity, discount and status. Future price-list changes must not mutate existing quotes. Validate unit availability and quote expiry.

## Acceptance criteria

Quote stores immutable price snapshot; unverified brand/product cannot be sold; expired/unavailable unit blocks confirmation.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: F-02 — Quotation builder and price snapshot (BACKEND ONLY)
Priority: P0
Dependencies: C-04, F-01, D-04, B-07

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement quotation create/update/preview APIs with customer, exact unit or approved item allocation, approved effective price snapshot, validity, discount and status. Future price-list changes must not mutate existing quotes. Validate unit availability and quote expiry.

ACCEPTANCE CRITERIA:
Quote stores immutable price snapshot; unverified brand/product cannot be sold; expired/unavailable unit blocks confirmation.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: F-02 — Quotation builder and price snapshot (FRONTEND ONLY)
Priority: P0
Dependencies: C-04, F-01, D-04, B-07

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build quote workflow: select customer, select brand/model and eligible unit, show price source/effective date, apply permitted discount, validity, preview and save draft/confirm. Preserve price snapshot display and handle stale availability conflicts.

ACCEPTANCE CRITERIA:
Quote stores immutable price snapshot; unverified brand/product cannot be sold; expired/unavailable unit blocks confirmation.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# F-03 — Discount approval workflow

**Epic folder:** `05_Sales_Delivery`  
**Priority:** P0  
**Depends on:** B-07, B-08, F-02

## Task scope

Implement configurable discount limits by role/amount and approval request/state, approver, reason, timestamp and audit. Pending or rejected discounts block quote/sale finalisation. Do not hardcode arbitrary thresholds.

## Acceptance criteria

Threshold is configurable; self/unauthorized approval is denied per policy; unapproved discount blocks finalisation.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: F-03 — Discount approval workflow (BACKEND ONLY)
Priority: P0
Dependencies: B-07, B-08, F-02

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement configurable discount limits by role/amount and approval request/state, approver, reason, timestamp and audit. Pending or rejected discounts block quote/sale finalisation. Do not hardcode arbitrary thresholds.

ACCEPTANCE CRITERIA:
Threshold is configurable; self/unauthorized approval is denied per policy; unapproved discount blocks finalisation.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: F-03 — Discount approval workflow (FRONTEND ONLY)
Priority: P0
Dependencies: B-07, B-08, F-02

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Add approval badge/strip, requested reason, pending approval queue and approve/reject controls shown by permission. Clearly explain why a quote cannot proceed; handle rejection and stale changes.

ACCEPTANCE CRITERIA:
Threshold is configurable; self/unauthorized approval is denied per policy; unapproved discount blocks finalisation.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# F-04 — Reservation lifecycle

**Epic folder:** `05_Sales_Delivery`  
**Priority:** P0  
**Depends on:** F-01, F-02, D-04, E-01

## Task scope

Implement reservation APIs for a specific vehicle unit with status, expiry timestamp, customer/order reference, advance reference if any, cancellation reason and audit/movement event. Enforce one active reservation per unit transactionally using locking/unique constraints.

## Acceptance criteria

Two concurrent reservations cannot claim same unit; cancellation/expiry history remains visible; unavailable units cannot be reserved.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: F-04 — Reservation lifecycle (BACKEND ONLY)
Priority: P0
Dependencies: F-01, F-02, D-04, E-01

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement reservation APIs for a specific vehicle unit with status, expiry timestamp, customer/order reference, advance reference if any, cancellation reason and audit/movement event. Enforce one active reservation per unit transactionally using locking/unique constraints.

ACCEPTANCE CRITERIA:
Two concurrent reservations cannot claim same unit; cancellation/expiry history remains visible; unavailable units cannot be reserved.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: F-04 — Reservation lifecycle (FRONTEND ONLY)
Priority: P0
Dependencies: F-01, F-02, D-04, E-01

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build reservation create/detail/list screens with available unit selector, expiry picker, confirmation summary, status, cancel/release action and clear collision error. Revalidate from API; never treat client availability as final.

ACCEPTANCE CRITERIA:
Two concurrent reservations cannot claim same unit; cancellation/expiry history remains visible; unavailable units cannot be reserved.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# F-05 — Reservation expiry job

**Epic folder:** `05_Sales_Delivery`  
**Priority:** P0  
**Depends on:** F-04, B-08

## Task scope

Implement an idempotent scheduled/worker handler that releases expired reservations only if still active and not linked to a finalised sale or protected extension. Record an explicit expiration event, update availability transactionally and make scheduling configurable.

## Acceptance criteria

Repeated job runs do not duplicate events or release the wrong unit; expired records remain in history; sale-linked reservations are protected.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: F-05 — Reservation expiry job (BACKEND ONLY)
Priority: P0
Dependencies: F-04, B-08

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement an idempotent scheduled/worker handler that releases expired reservations only if still active and not linked to a finalised sale or protected extension. Record an explicit expiration event, update availability transactionally and make scheduling configurable.

ACCEPTANCE CRITERIA:
Repeated job runs do not duplicate events or release the wrong unit; expired records remain in history; sale-linked reservations are protected.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: F-05 — Reservation expiry job (FRONTEND ONLY)
Priority: P0
Dependencies: F-04, B-08

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Show expiry status/time and release event in reservation history; refresh stale screens and explain when another process expired a reservation. No browser-only expiry timer may mutate inventory.

ACCEPTANCE CRITERIA:
Repeated job runs do not duplicate events or release the wrong unit; expired records remain in history; sale-linked reservations are protected.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# F-06 — Sales order workflow

**Epic folder:** `05_Sales_Delivery`  
**Priority:** P0  
**Depends on:** F-02, F-03, F-04

## Task scope

Implement sales order draft/submission/approval states, exact unit allocation, price snapshots, required payments/approval checklist and controlled transition rules. Never complete a sale from a draft order.

## Acceptance criteria

State transitions are validated; order cannot proceed with pending approvals or invalid unit; UI mirrors server workflow.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: F-06 — Sales order workflow (BACKEND ONLY)
Priority: P0
Dependencies: F-02, F-03, F-04

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement sales order draft/submission/approval states, exact unit allocation, price snapshots, required payments/approval checklist and controlled transition rules. Never complete a sale from a draft order.

ACCEPTANCE CRITERIA:
State transitions are validated; order cannot proceed with pending approvals or invalid unit; UI mirrors server workflow.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: F-06 — Sales order workflow (FRONTEND ONLY)
Priority: P0
Dependencies: F-02, F-03, F-04

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build sales order list/detail/creation screens with current status, customer, exact unit, amount breakdown, approval/payment state and action buttons. Provide a persistent blocker checklist and stale-data handling.

ACCEPTANCE CRITERIA:
State transitions are validated; order cannot proceed with pending approvals or invalid unit; UI mirrors server workflow.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# F-07 — Atomic sale finalisation

**Epic folder:** `05_Sales_Delivery`  
**Priority:** P0  
**Depends on:** F-06, F-04, F-08, F-09, E-01, B-08

## Task scope

Implement finalise-sale transaction: lock/revalidate unit and reservation, verify approvals/payments/checklist, create immutable sale snapshots, allocate unique document sequence if appropriate, post stock movement and mark unit SOLD/DELIVERED only according to workflow. Ensure idempotency and rollback on any error.

## Acceptance criteria

Concurrent finalisation cannot duplicate a sale; failures roll back all writes; sale snapshots and movement are created exactly once.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: F-07 — Atomic sale finalisation (BACKEND ONLY)
Priority: P0
Dependencies: F-06, F-04, F-08, F-09, E-01, B-08

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement finalise-sale transaction: lock/revalidate unit and reservation, verify approvals/payments/checklist, create immutable sale snapshots, allocate unique document sequence if appropriate, post stock movement and mark unit SOLD/DELIVERED only according to workflow. Ensure idempotency and rollback on any error.

ACCEPTANCE CRITERIA:
Concurrent finalisation cannot duplicate a sale; failures roll back all writes; sale snapshots and movement are created exactly once.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: F-07 — Atomic sale finalisation (FRONTEND ONLY)
Priority: P0
Dependencies: F-06, F-04, F-08, F-09, E-01, B-08

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Add final confirmation modal showing unit ID/chassis masked as permitted, customer and financial summary. Disable while submitting; do not optimistically mark unit sold; handle conflict if another user finalises first. Refresh dependent queries only after success.

ACCEPTANCE CRITERIA:
Concurrent finalisation cannot duplicate a sale; failures roll back all writes; sale snapshots and movement are created exactly once.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# F-08 — Payment receipt tracking

**Epic folder:** `05_Sales_Delivery`  
**Priority:** P0  
**Depends on:** F-06, B-07, B-08

## Task scope

Implement multiple receipt records per sale/order: amount, currency, method, timestamp, reference, status and allocation. Support reversal/refund with required permission, reason, original receipt reference and external reference where available. Do not store card or banking credentials.

## Acceptance criteria

Multiple receipts sum correctly; reversals are auditable and permissioned; UI clearly separates sale total, collected, refunded and outstanding.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: F-08 — Payment receipt tracking (BACKEND ONLY)
Priority: P0
Dependencies: F-06, B-07, B-08

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement multiple receipt records per sale/order: amount, currency, method, timestamp, reference, status and allocation. Support reversal/refund with required permission, reason, original receipt reference and external reference where available. Do not store card or banking credentials.

ACCEPTANCE CRITERIA:
Multiple receipts sum correctly; reversals are auditable and permissioned; UI clearly separates sale total, collected, refunded and outstanding.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: F-08 — Payment receipt tracking (FRONTEND ONLY)
Priority: P0
Dependencies: F-06, B-07, B-08

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build add-payment/receipt list/reversal flows with amount validation, method/reference fields, collected-vs-outstanding breakdown and permission states. Explain that this records payments but does not process them through a gateway.

ACCEPTANCE CRITERIA:
Multiple receipts sum correctly; reversals are auditable and permissioned; UI clearly separates sale total, collected, refunded and outstanding.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# F-09 — Delivery checklist

**Epic folder:** `05_Sales_Delivery`  
**Priority:** P0  
**Depends on:** F-06, D-06, B-07, B-08

## Task scope

Implement configurable checklist templates by model/transaction. The client-confirmed categories are keys, charger, manuals, warranty documents, payment, insurance and registration. Preserve the approved PRD/business-rule items such as PDI and customer acknowledgement where applicable. The client has not yet signed off which items are always mandatory, conditional or not applicable, nor the allowed override roles/reasons. Store policy configuration and approval status; do not silently infer missing policy. Production delivery must be blocked when checklist policy is not approved or any approved mandatory item is incomplete. Any authorised override requires an approved permission, reason and audit event.

## Acceptance criteria

Delivery is blocked until a client-approved checklist policy exists and all configured mandatory items pass, or an expressly authorised override with reason is recorded; the checklist snapshot/history is retained. Unapproved mandatory/conditional rules are not invented.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: F-09 — Delivery checklist (BACKEND ONLY)
Priority: P0
Dependencies: F-06, D-06, B-07, B-08

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement configurable checklist templates by model/transaction. The client-confirmed categories are keys, charger, manuals, warranty documents, payment, insurance and registration. Preserve the approved PRD/business-rule items such as PDI and customer acknowledgement where applicable. The client has not yet signed off which items are always mandatory, conditional or not applicable, nor the allowed override roles/reasons. Store policy configuration and approval status; do not silently infer missing policy. Production delivery must be blocked when checklist policy is not approved or any approved mandatory item is incomplete. Any authorised override requires an approved permission, reason and audit event.

ACCEPTANCE CRITERIA:
Delivery is blocked until a client-approved checklist policy exists and all configured mandatory items pass, or an expressly authorised override with reason is recorded; the checklist snapshot/history is retained. Unapproved mandatory/conditional rules are not invented.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: F-09 — Delivery checklist (FRONTEND ONLY)
Priority: P0
Dependencies: F-06, D-06, B-07, B-08

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build grouped checklist UI and a persistent “Cannot deliver yet” panel with missing items; support authorized override reason flow. Preserve form state and show completion status on sale detail.

ACCEPTANCE CRITERIA:
Delivery is blocked until a client-approved checklist policy exists and all configured mandatory items pass, or an expressly authorised override with reason is recorded; the checklist snapshot/history is retained. Unapproved mandatory/conditional rules are not invented.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# F-10 — Invoice, receipt and delivery templates

**Epic folder:** `05_Sales_Delivery`  
**Priority:** P0  
**Depends on:** F-07, F-08, B-05

## Task scope

Implement document records, unique numbering by configured financial year/series, immutable commercial/tax snapshots and printable/PDF generation or an extensible renderer. The client confirms legal business name, GST details and showroom address exist and files GST/income tax, but the exact values and accountant-approved invoice settings are still pending input. Keep tax fields/configuration effective-dated; require actual legal data and accountant approval before production invoice issuance. Never hardcode statutory rates/thresholds. GST-return filing, income-tax-return filing and government tax-portal integration are outside the MVP scope.

## Acceptance criteria

Document numbers do not collide or reuse after issue; snapshots remain stable; production issuance is blocked until client legal/GST data and accountant-approved invoice/tax configuration are complete. No tax-return filing workflow is introduced.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: F-10 — Invoice, receipt and delivery templates (BACKEND ONLY)
Priority: P0
Dependencies: F-07, F-08, B-05

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement document records, unique numbering by configured financial year/series, immutable commercial/tax snapshots and printable/PDF generation or an extensible renderer. The client confirms legal business name, GST details and showroom address exist and files GST/income tax, but the exact values and accountant-approved invoice settings are still pending input. Keep tax fields/configuration effective-dated; require actual legal data and accountant approval before production invoice issuance. Never hardcode statutory rates/thresholds. GST-return filing, income-tax-return filing and government tax-portal integration are outside the MVP scope.

ACCEPTANCE CRITERIA:
Document numbers do not collide or reuse after issue; snapshots remain stable; production issuance is blocked until client legal/GST data and accountant-approved invoice/tax configuration are complete. No tax-return filing workflow is introduced.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: F-10 — Invoice, receipt and delivery templates (FRONTEND ONLY)
Priority: P0
Dependencies: F-07, F-08, B-05

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build document preview and print/export UI for invoice, payment receipt and delivery note; show accountant-review/approval status; handle missing legal/billing configuration by blocking issue rather than printing fake details.

ACCEPTANCE CRITERIA:
Document numbers do not collide or reuse after issue; snapshots remain stable; production issuance is blocked until client legal/GST data and accountant-approved invoice/tax configuration are complete. No tax-return filing workflow is introduced.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# F-11 — Cancellation, return and reversal workflow

**Epic folder:** `05_Sales_Delivery`  
**Priority:** P0  
**Depends on:** F-07, F-08, E-01, B-08

## Task scope

Implement permissioned cancellation/return cases, reason, inspection/disposition, linked financial reversal/credit reference and stock disposition. Never hard-delete a finalised sale or set vehicle back to available by direct status edit. Distinguish pre-sale cancellation from post-delivery return.

## Acceptance criteria

History is preserved; financial and stock reversals are linked; returned unit is not saleable until authorized inspection/disposition.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: F-11 — Cancellation, return and reversal workflow (BACKEND ONLY)
Priority: P0
Dependencies: F-07, F-08, E-01, B-08

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement permissioned cancellation/return cases, reason, inspection/disposition, linked financial reversal/credit reference and stock disposition. Never hard-delete a finalised sale or set vehicle back to available by direct status edit. Distinguish pre-sale cancellation from post-delivery return.

ACCEPTANCE CRITERIA:
History is preserved; financial and stock reversals are linked; returned unit is not saleable until authorized inspection/disposition.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: F-11 — Cancellation, return and reversal workflow (FRONTEND ONLY)
Priority: P0
Dependencies: F-07, F-08, E-01, B-08

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build cancel/return request and details UI with confirmation of consequences, mandatory reason, linked receipt/credit references, inspection result and pending/approved states. Do not offer destructive delete.

ACCEPTANCE CRITERIA:
History is preserved; financial and stock reversals are linked; returned unit is not saleable until authorized inspection/disposition.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# EPIC G — Reports, Import & Export


# G-01 — Dashboard metrics

**Epic folder:** `06_Reports_Import_Export`  
**Priority:** P0  
**Depends on:** D-04, E-01, F-07, F-08

## Task scope

Implement aggregate metrics for saleable, reserved, quality/service hold, rejected, in-transit, sold today/month, stock ageing and low parts. Define metric queries centrally and apply permissions/location/brand filters; return updated_at and definitions.

## Acceptance criteria

Dashboard totals reconcile with filtered reports; metrics are labelled separately; loading/empty/error states and click-through work.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: G-01 — Dashboard metrics (BACKEND ONLY)
Priority: P0
Dependencies: D-04, E-01, F-07, F-08

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement aggregate metrics for saleable, reserved, quality/service hold, rejected, in-transit, sold today/month, stock ageing and low parts. Define metric queries centrally and apply permissions/location/brand filters; return updated_at and definitions.

ACCEPTANCE CRITERIA:
Dashboard totals reconcile with filtered reports; metrics are labelled separately; loading/empty/error states and click-through work.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: G-01 — Dashboard metrics (FRONTEND ONLY)
Priority: P0
Dependencies: D-04, E-01, F-07, F-08

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build KPI cards and small decision-oriented charts as appropriate, global brand/location/date filters, updated timestamp and quick actions. KPI click-through must navigate to an underlying prefiltered list. Distinguish physical, saleable, reserved and in-transit counts.

ACCEPTANCE CRITERIA:
Dashboard totals reconcile with filtered reports; metrics are labelled separately; loading/empty/error states and click-through work.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# G-02 — Inventory reports

**Epic folder:** `06_Reports_Import_Export`  
**Priority:** P0  
**Depends on:** D-04, E-07, C-06

## Task scope

Implement inventory reports by brand/model/variant/status/location, ageing and acquisition-cost valuation. Use consistent server-side filters and paging/aggregation; do not mix selling price with stock cost. Respect cost/report permissions.

## Acceptance criteria

Brand/status/location filters match dashboard and inventory; valuation uses declared cost basis; unauthorized cost values are omitted.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: G-02 — Inventory reports (BACKEND ONLY)
Priority: P0
Dependencies: D-04, E-07, C-06

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement inventory reports by brand/model/variant/status/location, ageing and acquisition-cost valuation. Use consistent server-side filters and paging/aggregation; do not mix selling price with stock cost. Respect cost/report permissions.

ACCEPTANCE CRITERIA:
Brand/status/location filters match dashboard and inventory; valuation uses declared cost basis; unauthorized cost values are omitted.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: G-02 — Inventory reports (FRONTEND ONLY)
Priority: P0
Dependencies: D-04, E-07, C-06

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build report filter panel, summary, paginated result table, ageing bands and available export action. Show report definitions, generation timestamp and filters; avoid client-side calculations inconsistent with API.

ACCEPTANCE CRITERIA:
Brand/status/location filters match dashboard and inventory; valuation uses declared cost basis; unauthorized cost values are omitted.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# G-03 — Sales and collections reports

**Epic folder:** `06_Reports_Import_Export`  
**Priority:** P0  
**Depends on:** F-07, F-08, F-10

## Task scope

Implement sales value, discounts, taxes, amount collected, outstanding and refunds as separate server-calculated metrics. Filter by date, brand, model, payment state and authorized location; document gross/net definitions.

## Acceptance criteria

All totals reconcile with sales/payment records; definitions are documented; data access respects role and brand/location scope.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: G-03 — Sales and collections reports (BACKEND ONLY)
Priority: P0
Dependencies: F-07, F-08, F-10

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement sales value, discounts, taxes, amount collected, outstanding and refunds as separate server-calculated metrics. Filter by date, brand, model, payment state and authorized location; document gross/net definitions.

ACCEPTANCE CRITERIA:
All totals reconcile with sales/payment records; definitions are documented; data access respects role and brand/location scope.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: G-03 — Sales and collections reports (FRONTEND ONLY)
Priority: P0
Dependencies: F-07, F-08, F-10

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build sales/collections report views with clear separate columns/cards for sales, discount, tax, collections, refunds and outstanding. Include date/brand filters and empty/error states. Do not label sales revenue as cash collected.

ACCEPTANCE CRITERIA:
All totals reconcile with sales/payment records; definitions are documented; data access respects role and brand/location scope.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# G-04 — Pending procurement/PDI/reservation queues

**Epic folder:** `06_Reports_Import_Export`  
**Priority:** P0  
**Depends on:** D-02, D-03, D-06, F-04, F-05

## Task scope

Implement queue endpoints for pending/partial POs, PDI holds, ageing reservations and other pending workflow records; include age/due fields and actionable references.

## Acceptance criteria

Queue count/details match source records; users can drill through only to authorized objects; status refreshes after actions.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: G-04 — Pending procurement/PDI/reservation queues (BACKEND ONLY)
Priority: P0
Dependencies: D-02, D-03, D-06, F-04, F-05

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement queue endpoints for pending/partial POs, PDI holds, ageing reservations and other pending workflow records; include age/due fields and actionable references.

ACCEPTANCE CRITERIA:
Queue count/details match source records; users can drill through only to authorized objects; status refreshes after actions.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: G-04 — Pending procurement/PDI/reservation queues (FRONTEND ONLY)
Priority: P0
Dependencies: D-02, D-03, D-06, F-04, F-05

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build queue dashboard/table with filter/sort, age badges and drill-through to the source record. Provide clear overdue/no-data states and permission-aware actions.

ACCEPTANCE CRITERIA:
Queue count/details match source records; users can drill through only to authorized objects; status refreshes after actions.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# G-05 — CSV/XLSX export framework

**Epic folder:** `06_Reports_Import_Export`  
**Priority:** P0  
**Depends on:** B-07, B-08, G-02, G-03

## Task scope

Implement authorized export service for CSV and, if approved dependency is added, XLSX; apply the same server filters and permissions as report queries, neutralize spreadsheet formula injection, log export actor/time/filters and include generation timestamp. Bound large exports or stream safely.

## Acceptance criteria

Export reflects filtered query and authorized scope; spreadsheet formula injection is mitigated; exports are audited; large datasets fail safely.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: G-05 — CSV/XLSX export framework (BACKEND ONLY)
Priority: P0
Dependencies: B-07, B-08, G-02, G-03

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement authorized export service for CSV and, if approved dependency is added, XLSX; apply the same server filters and permissions as report queries, neutralize spreadsheet formula injection, log export actor/time/filters and include generation timestamp. Bound large exports or stream safely.

ACCEPTANCE CRITERIA:
Export reflects filtered query and authorized scope; spreadsheet formula injection is mitigated; exports are audited; large datasets fail safely.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: G-05 — CSV/XLSX export framework (FRONTEND ONLY)
Priority: P0
Dependencies: B-07, B-08, G-02, G-03

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Add export controls, format selection where supported, progress/error/download handling and an active-filter summary. Prevent accidental export without permission and make filenames meaningful but non-sensitive.

ACCEPTANCE CRITERIA:
Export reflects filtered query and authorized scope; spreadsheet formula injection is mitigated; exports are audited; large datasets fail safely.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# G-06 — Opening stock import preview

**Epic folder:** `06_Reports_Import_Export`  
**Priority:** P0  
**Depends on:** C-01, D-04, E-03, B-09

## Task scope

Implement import upload/staging, file/schema validation, row-level errors, duplicate chassis/serial detection, brand/model/variant resolution and provenance column checks. The opening-stock import template is a confirmed requirement, but the actual opening-stock dataset has not been supplied. No inventory mutation occurs during preview; do not make up rows or auto-load examples. Include brand, model/variant, chassis/frame, applicable motor/battery/charger identifiers, colour, location, status/condition and available provenance/receipt references according to the approved template. Enforce file limits and private upload authorisation.

## Acceptance criteria

Preview is non-mutating; row errors map to lines/fields; unresolved catalogue mapping and brands/models pending verification are visible; upload/download is secure; no invented opening stock is committed.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: G-06 — Opening stock import preview (BACKEND ONLY)
Priority: P0
Dependencies: C-01, D-04, E-03, B-09

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement import upload/staging, file/schema validation, row-level errors, duplicate chassis/serial detection, brand/model/variant resolution and provenance column checks. The opening-stock import template is a confirmed requirement, but the actual opening-stock dataset has not been supplied. No inventory mutation occurs during preview; do not make up rows or auto-load examples. Include brand, model/variant, chassis/frame, applicable motor/battery/charger identifiers, colour, location, status/condition and available provenance/receipt references according to the approved template. Enforce file limits and private upload authorisation.

ACCEPTANCE CRITERIA:
Preview is non-mutating; row errors map to lines/fields; unresolved catalogue mapping and brands/models pending verification are visible; upload/download is secure; no invented opening stock is committed.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: G-06 — Opening stock import preview (FRONTEND ONLY)
Priority: P0
Dependencies: C-01, D-04, E-03, B-09

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build import wizard for template download, file upload, mapping/preview, row errors, duplicate highlights and summary counts. Provide corrective feedback and prevent Continue/Commit when fatal errors exist.

ACCEPTANCE CRITERIA:
Preview is non-mutating; row errors map to lines/fields; unresolved catalogue mapping and brands/models pending verification are visible; upload/download is secure; no invented opening stock is committed.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# G-07 — Opening stock import commit and idempotency

**Epic folder:** `06_Reports_Import_Export`  
**Priority:** P0  
**Depends on:** G-06, D-04, E-01, B-08

## Task scope

Implement commit by batch ID with transaction/staging strategy; fatal errors prevent partial commit, duplicate/replayed batch cannot create duplicate records, every imported item records opening-balance provenance, movements/audit are emitted and the batch stores uploader/hash/status. Commit must use the client-provided stock file after mapping/duplicate checks and physical reconciliation plan are approved. Do not create demo inventory, and do not allow unverified brand/model mappings to become saleable stock. Preserve unmatched rows for correction rather than guessing catalogue relationships. Commit must use the client-provided stock file after mapping/duplicate checks and physical reconciliation plan are approved. Do not create demo inventory, and do not allow unverified brand/model mappings to become saleable stock. Preserve unmatched rows for correction rather than guessing catalogue relationships.

## Acceptance criteria

A batch commits once; fatal errors yield zero committed rows; source provenance is preserved; replay is safe; no fabricated stock is imported and unverified catalogue mappings cannot become saleable stock.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: G-07 — Opening stock import commit and idempotency (BACKEND ONLY)
Priority: P0
Dependencies: G-06, D-04, E-01, B-08

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Implement commit by batch ID with transaction/staging strategy; fatal errors prevent partial commit, duplicate/replayed batch cannot create duplicate records, every imported item records opening-balance provenance, movements/audit are emitted and the batch stores uploader/hash/status. Commit must use the client-provided stock file after mapping/duplicate checks and physical reconciliation plan are approved. Do not create demo inventory, and do not allow unverified brand/model mappings to become saleable stock. Preserve unmatched rows for correction rather than guessing catalogue relationships. Commit must use the client-provided stock file after mapping/duplicate checks and physical reconciliation plan are approved. Do not create demo inventory, and do not allow unverified brand/model mappings to become saleable stock. Preserve unmatched rows for correction rather than guessing catalogue relationships.

ACCEPTANCE CRITERIA:
A batch commits once; fatal errors yield zero committed rows; source provenance is preserved; replay is safe; no fabricated stock is imported and unverified catalogue mappings cannot become saleable stock.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: G-07 — Opening stock import commit and idempotency (FRONTEND ONLY)
Priority: P0
Dependencies: G-06, D-04, E-01, B-08

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Build explicit review-and-commit confirmation showing create/update counts, fatal/warning counts and batch ID. Show commit result/report; if network retries occur, query batch status instead of blindly resubmitting.

ACCEPTANCE CRITERIA:
A batch commits once; fatal errors yield zero committed rows; source provenance is preserved; replay is safe; no fabricated stock is imported and unverified catalogue mappings cannot become saleable stock.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# EPIC H — QA & Operations


# H-01 — Unit and integration test suite

**Epic folder:** `07_QA_Operations`  
**Priority:** P0  
**Depends on:** Relevant feature tasks implemented

## Task scope

Create/expand Go unit and integration tests for price snapshots, state transitions, RBAC, duplicate IDs, ledger invariants and transactional rollbacks. Use isolated test DB/fixtures and cleanup; no real customer data. Do not change behavior merely to satisfy brittle tests.

## Acceptance criteria

Business-critical backend/frontend checks run repeatably; test commands documented; failures are diagnosed rather than skipped.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: H-01 — Unit and integration test suite (BACKEND ONLY)
Priority: P0
Dependencies: Relevant feature tasks implemented

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Create/expand Go unit and integration tests for price snapshots, state transitions, RBAC, duplicate IDs, ledger invariants and transactional rollbacks. Use isolated test DB/fixtures and cleanup; no real customer data. Do not change behavior merely to satisfy brittle tests.

ACCEPTANCE CRITERIA:
Business-critical backend/frontend checks run repeatably; test commands documented; failures are diagnosed rather than skipped.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: H-01 — Unit and integration test suite (FRONTEND ONLY)
Priority: P0
Dependencies: Relevant feature tasks implemented

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Create/expand frontend unit/component tests for critical forms, permission states, validation mappings, loading/error/conflict states and API hooks. Use deterministic test fixtures and mock only at test boundaries; keep at least one true API contract/integration test where feasible.

ACCEPTANCE CRITERIA:
Business-critical backend/frontend checks run repeatably; test commands documented; failures are diagnosed rather than skipped.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# H-02 — End-to-end critical workflows

**Epic folder:** `07_QA_Operations`  
**Priority:** P0  
**Depends on:** D-07, D-06, F-04, F-07, F-09

## Task scope

Provide test support or deterministic test data and execute an end-to-end workflow: PO -> GRN -> PDI -> available -> reserve -> sale -> delivery; include invalid PDI, out-of-stock, missing checklist and API failure paths. Keep tests in non-production environment.

## Acceptance criteria

Happy path and key blocked paths pass; test environment cleanup is safe; screenshots/logs or clear run output are documented.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: H-02 — End-to-end critical workflows (BACKEND ONLY)
Priority: P0
Dependencies: D-07, D-06, F-04, F-07, F-09

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Provide test support or deterministic test data and execute an end-to-end workflow: PO -> GRN -> PDI -> available -> reserve -> sale -> delivery; include invalid PDI, out-of-stock, missing checklist and API failure paths. Keep tests in non-production environment.

ACCEPTANCE CRITERIA:
Happy path and key blocked paths pass; test environment cleanup is safe; screenshots/logs or clear run output are documented.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: H-02 — End-to-end critical workflows (FRONTEND ONLY)
Priority: P0
Dependencies: D-07, D-06, F-04, F-07, F-09

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Implement Playwright or existing E2E tests for the full critical journey and failure paths. Use the real local API/database, seed disposable test data, verify visible statuses and avoid relying on arbitrary sleeps.

ACCEPTANCE CRITERIA:
Happy path and key blocked paths pass; test environment cleanup is safe; screenshots/logs or clear run output are documented.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# H-03 — Concurrency and idempotency tests

**Epic folder:** `07_QA_Operations`  
**Priority:** P0  
**Depends on:** D-07, F-04, F-05, F-07, G-07

## Task scope

Add race/concurrency tests for two reservations on one unit, two sale finalisations, duplicate GRN post and repeated import commit. Use DB transaction/locking/constraints, not in-memory-only protections. Verify exactly one winner and a clear conflict response.

## Acceptance criteria

Exactly one conflicting mutation succeeds; repeated safe request is idempotent; UI reflects authoritative server state.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: H-03 — Concurrency and idempotency tests (BACKEND ONLY)
Priority: P0
Dependencies: D-07, F-04, F-05, F-07, G-07

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Add race/concurrency tests for two reservations on one unit, two sale finalisations, duplicate GRN post and repeated import commit. Use DB transaction/locking/constraints, not in-memory-only protections. Verify exactly one winner and a clear conflict response.

ACCEPTANCE CRITERIA:
Exactly one conflicting mutation succeeds; repeated safe request is idempotent; UI reflects authoritative server state.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: H-03 — Concurrency and idempotency tests (FRONTEND ONLY)
Priority: P0
Dependencies: D-07, F-04, F-05, F-07, G-07

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Add UI-level tests for conflict/stale-data behavior and disabled submission while a request is pending; simulate server conflict responses and ensure the screen refreshes/reconciles state rather than displaying false success.

ACCEPTANCE CRITERIA:
Exactly one conflicting mutation succeeds; repeated safe request is idempotent; UI reflects authoritative server state.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# H-04 — Security and access tests

**Epic folder:** `07_QA_Operations`  
**Priority:** P0  
**Depends on:** B-06, B-07, B-09, G-05

## Task scope

Test authentication/session expiration, disabled users, IDOR/object scope, privilege escalation, attachment download controls, export permissions, audit immutability and unauthorized cross-brand access. Fix failures within this task only when fixes relate to the tested security path.

## Acceptance criteria

Negative security cases are automated; no unauthorized data disclosure on API or frontend; findings and exceptions documented.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: H-04 — Security and access tests (BACKEND ONLY)
Priority: P0
Dependencies: B-06, B-07, B-09, G-05

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Test authentication/session expiration, disabled users, IDOR/object scope, privilege escalation, attachment download controls, export permissions, audit immutability and unauthorized cross-brand access. Fix failures within this task only when fixes relate to the tested security path.

ACCEPTANCE CRITERIA:
Negative security cases are automated; no unauthorized data disclosure on API or frontend; findings and exceptions documented.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: H-04 — Security and access tests (FRONTEND ONLY)
Priority: P0
Dependencies: B-06, B-07, B-09, G-05

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Test route guards/action visibility and sensitive field masking; confirm direct API 401/403 responses are handled without leaking record existence or confidential details. Never treat frontend tests as a substitute for backend authorization tests.

ACCEPTANCE CRITERIA:
Negative security cases are automated; no unauthorized data disclosure on API or frontend; findings and exceptions documented.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# H-05 — Performance checks and indexes

**Epic folder:** `07_QA_Operations`  
**Priority:** P0  
**Depends on:** Core list/search/report endpoints implemented

## Task scope

Profile common queries: chassis lookup, vehicle list filters, brand-model selectors, dashboard and reports at agreed pilot volume. Add only evidence-based indexes; review query plans; set safe pagination limits and query timeouts. Record baseline and results.

## Acceptance criteria

Common query timings measured against a defined dataset; pagination enforced; no unbounded list/export; index changes have migration and rationale.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: H-05 — Performance checks and indexes (BACKEND ONLY)
Priority: P0
Dependencies: Core list/search/report endpoints implemented

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Profile common queries: chassis lookup, vehicle list filters, brand-model selectors, dashboard and reports at agreed pilot volume. Add only evidence-based indexes; review query plans; set safe pagination limits and query timeouts. Record baseline and results.

ACCEPTANCE CRITERIA:
Common query timings measured against a defined dataset; pagination enforced; no unbounded list/export; index changes have migration and rationale.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: H-05 — Performance checks and indexes (FRONTEND ONLY)
Priority: P0
Dependencies: Core list/search/report endpoints implemented

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Check table screens use server pagination and avoid excessive refetch/rerender. Add lightweight frontend performance diagnostics for large lists if useful; do not build a separate analytics UI.

ACCEPTANCE CRITERIA:
Common query timings measured against a defined dataset; pagination enforced; no unbounded list/export; index changes have migration and rationale.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# H-06 — Backup and restore drill

**Epic folder:** `07_QA_Operations`  
**Priority:** P0  
**Depends on:** B-05, production-like environment

## Task scope

Document and execute PostgreSQL backup/restore in a non-production environment. Include storage attachments if configured, secrets separation, retention and restore verification using row counts and key reports. Do not run destructive restore against production.

## Acceptance criteria

Restore drill is recorded; critical tables and attachments are verified; production is not overwritten; runbook is actionable.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: H-06 — Backup and restore drill (BACKEND ONLY)
Priority: P0
Dependencies: B-05, production-like environment

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Document and execute PostgreSQL backup/restore in a non-production environment. Include storage attachments if configured, secrets separation, retention and restore verification using row counts and key reports. Do not run destructive restore against production.

ACCEPTANCE CRITERIA:
Restore drill is recorded; critical tables and attachments are verified; production is not overwritten; runbook is actionable.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: H-06 — Backup and restore drill (FRONTEND ONLY)
Priority: P0
Dependencies: B-05, production-like environment

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Verify frontend behaves gracefully while API/database are unavailable and after restoration; document cache refresh/re-login required after restore if applicable. No new business UI needed.

ACCEPTANCE CRITERIA:
Restore drill is recorded; critical tables and attachments are verified; production is not overwritten; runbook is actionable.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# H-07 — Staging/UAT and defect triage

**Epic folder:** `07_QA_Operations`  
**Priority:** P0  
**Depends on:** All P0 feature tasks

## Task scope

Create a staged UAT checklist aligned with PRD: receiving/PDI, inventory search, reservation, discounts, sale, payment, delivery, invoices, reports and permissions. Classify defects by severity and verify blocking defects after fixes. Do not claim client sign-off without recorded approval.

## Acceptance criteria

UAT evidence and open defects are documented; P0 acceptance defects are closed or explicitly blocked; client approval is recorded, not assumed.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: H-07 — Staging/UAT and defect triage (BACKEND ONLY)
Priority: P0
Dependencies: All P0 feature tasks

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Create a staged UAT checklist aligned with PRD: receiving/PDI, inventory search, reservation, discounts, sale, payment, delivery, invoices, reports and permissions. Classify defects by severity and verify blocking defects after fixes. Do not claim client sign-off without recorded approval.

ACCEPTANCE CRITERIA:
UAT evidence and open defects are documented; P0 acceptance defects are closed or explicitly blocked; client approval is recorded, not assumed.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: H-07 — Staging/UAT and defect triage (FRONTEND ONLY)
Priority: P0
Dependencies: All P0 feature tasks

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Walk through the actual UI at staging, record screenshots/steps for defects, test desktop/tablet and keyboard accessibility for critical flows, and fix only defects approved within this task scope.

ACCEPTANCE CRITERIA:
UAT evidence and open defects are documented; P0 acceptance defects are closed or explicitly blocked; client approval is recorded, not assumed.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# H-08 — Opening data import and physical reconciliation

**Epic folder:** `07_QA_Operations`  
**Priority:** P0  
**Depends on:** G-06, G-07, D-04, E-06

## Task scope

Prepare reviewed opening inventory import, perform validation, generate row counts/duplicate report and reconcile imported units/parts to signed physical count. Do not fabricate provenance or silently edit discrepancies; create approved correction movements.

## Acceptance criteria

No unapproved opening stock is treated as verified; duplicates are resolved; physical-count discrepancy sign-off is recorded.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: H-08 — Opening data import and physical reconciliation (BACKEND ONLY)
Priority: P0
Dependencies: G-06, G-07, D-04, E-06

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Prepare reviewed opening inventory import, perform validation, generate row counts/duplicate report and reconcile imported units/parts to signed physical count. Do not fabricate provenance or silently edit discrepancies; create approved correction movements.

ACCEPTANCE CRITERIA:
No unapproved opening stock is treated as verified; duplicates are resolved; physical-count discrepancy sign-off is recorded.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: H-08 — Opening data import and physical reconciliation (FRONTEND ONLY)
Priority: P0
Dependencies: G-06, G-07, D-04, E-06

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Provide review screen/report for imported totals, discrepancies and sign-off status; make it easy for authorized staff to download error/reconciliation reports.

ACCEPTANCE CRITERIA:
No unapproved opening stock is treated as verified; duplicates are resolved; physical-count discrepancy sign-off is recorded.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# H-09 — Training documentation and runbooks

**Epic folder:** `07_QA_Operations`  
**Priority:** P0  
**Depends on:** Stable workflows

## Task scope

Create operator/admin guides for receiving/PDI, inventory search, reservations/sales, adjustments, reports, backups, release and incident support. Include screenshots or exact navigation names from the actual implementation. Keep sensitive credentials out of docs.

## Acceptance criteria

Guides match the shipped application; first-line troubleshooting and escalation are documented; no secrets or personal data are included.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: H-09 — Training documentation and runbooks (BACKEND ONLY)
Priority: P0
Dependencies: Stable workflows

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Create operator/admin guides for receiving/PDI, inventory search, reservations/sales, adjustments, reports, backups, release and incident support. Include screenshots or exact navigation names from the actual implementation. Keep sensitive credentials out of docs.

ACCEPTANCE CRITERIA:
Guides match the shipped application; first-line troubleshooting and escalation are documented; no secrets or personal data are included.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: H-09 — Training documentation and runbooks (FRONTEND ONLY)
Priority: P0
Dependencies: Stable workflows

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Add short in-app helper copy/tooltips only where it materially prevents workflow errors; align text with runbooks and actual feature states; do not invent controls or screens.

ACCEPTANCE CRITERIA:
Guides match the shipped application; first-line troubleshooting and escalation are documented; no secrets or personal data are included.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---


# H-10 — Production rollout and smoke test

**Epic folder:** `07_QA_Operations`  
**Priority:** P0  
**Depends on:** H-04, H-05, H-06, H-07, H-08

## Task scope

Prepare production config/secrets checklist, migration plan, backup-before-release, migration compatibility, health/readiness smoke tests, logs/alerts, secure admin provisioning without hardcoded password and rollback/forward-fix decision. Never execute production actions without explicit approval.

## Acceptance criteria

Deployment checklist, smoke test evidence, rollback plan and monitoring owners are documented; production change is not performed without authorization.

---

## Prompt A — Backend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: H-10 — Production rollout and smoke test (BACKEND ONLY)
Priority: P0
Dependencies: H-04, H-05, H-06, H-07, H-08

Implement the backend portion only for this task.

TASK-SPECIFIC BACKEND SCOPE:
Prepare production config/secrets checklist, migration plan, backup-before-release, migration compatibility, health/readiness smoke tests, logs/alerts, secure admin provisioning without hardcoded password and rollback/forward-fix decision. Never execute production actions without explicit approval.

ACCEPTANCE CRITERIA:
Deployment checklist, smoke test evidence, rollback plan and monitoring owners are documented; production change is not performed without authorization.

BACKEND-SPECIFIC QUALITY GATES:
- Use Gin handlers -> service/domain logic -> pgx repository/data-access boundaries consistent with the existing repository.
- Use versioned SQL migrations; include constraints/indexes and rollback considerations where schema changes are needed.
- Validate state transitions and permissions in the backend; never trust client-supplied totals, statuses or brand ownership.
- Use transactions and idempotency keys for any multi-row stock-changing operation in scope.
- Add unit/integration tests for the task’s happy path, invalid input, forbidden action and conflict/concurrency cases where relevant.
- Update OpenAPI/API conventions and give the frontend the exact request/response/error contract.
- Do not implement the frontend screen in this prompt.

```

---

## Prompt B — Frontend (copy this block into Claude Code)

```text
You are working in the existing single Git repository for Maa Ambey Enterprises — Multi-Brand EV Inventory Management System.

CURRENT APPROVED STACK (do not switch frameworks):
- Frontend: React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod; Redux Toolkit only for shared client state when justified.
- Backend: Go (Golang) + Gin REST API, PostgreSQL, pgx, versioned SQL migrations, structured logging, OpenAPI.
- Repository: one repo containing /frontend and /backend. Never create a second repository.

MANDATORY PRODUCT RULES:
- Multi-brand support is a confirmed P0 requirement. Brand names must come from the database, not hardcoded enums/constants.
- Client-confirmed initial intended operational brand roster: Lectrix EV, Ceeon India, and Ujagar Electric. Future brands must be addable through data/configuration, not code changes. This confirms intended use, not legal/dealership authorisation or catalogue verification. Keep brand existence/lifecycle separate from purchase/sales eligibility; block commercial activation until the required supplier/legal identity, authorisation evidence, verified model/variant catalogue, approved prices/warranty/identifier rules, approver and verification date are recorded. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity.
- Do not invent models, specifications, prices, warranties, supplier relationships, authorisation evidence, chassis/serial values or stock. Use only client/supplier-confirmed source data and approval evidence. Actual model catalogue and opening stock data are still pending collection.
- Client confirms legal business name, GST details and showroom address exist and that GST/income-tax filings are performed. Their exact values and accountant-approved invoice settings have not been supplied in this task. Do not invent them. GST-return filing, income-tax-return filing, government filing and tax-portal integration are outside the MVP/setup scope unless separately approved.
- Confirmed initial role names: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. The granular permission matrix and approval thresholds remain pending owner sign-off. Enforce deny-by-default on the backend and do not infer production grants from role names alone.
- Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional/not-applicable behaviour and permitted overrides remain pending client approval; keep checklist policy configurable and do not silently treat unconfigured items as optional.
- Opening-stock import template is required, but actual opening-stock data and physical reconciliation sign-off are pending. Never seed fabricated inventory.
- Vehicle identity is serialised. Normalised chassis/frame identifiers must be unique. Motor/battery/charger uniqueness is configurable by approved business rule.
- Stock-changing actions must validate server-side, record append-only stock movement/audit history where required, and be transactional/idempotent where critical.
- Do not hard-delete posted stock, sale, payment, receipt or audit history. Use approved reversal/correction flows.
- Server-side authorization is authoritative. Hiding a frontend button is not authorization.

BEFORE CODING:
1. Inspect current Git status and the relevant code/docs. Read the applicable docs in /docs (or the project’s existing location): PRD, System Architecture, Business Rules, Design Specification and Implementation Tasks.
2. Preserve existing work and follow existing conventions. The task IDs are traceable to the approved v1.2 backlog. The confirmed implementation stack is React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository; do not switch frameworks.
3. Do not guess unresolved business/tax/client decisions. Record blockers in docs/ARCHITECTURE_DECISIONS.md or a task-specific note.

TASK DISCIPLINE:
- Implement ONLY the named task in this prompt. Do not start dependent/future tasks or perform unrelated refactors.
- Implement actual production-path logic, not fake/mock success behavior.
- Add/adjust relevant tests, API/OpenAPI docs, migrations and frontend states only as this task requires.
- Validate request input on the server. Use parameterized SQL with pgx. Use React Query for remote state; do not persist secrets/tokens in localStorage.
- Include loading, empty, error, conflict and permission-denied UI states when a screen is involved.
- Run the relevant formatter, lint, typecheck, tests and build. Report commands and actual results honestly; never claim unrun checks passed.
- Inspect final Git diff and do not overwrite approved specification documents.

In your final response, state: files changed, implementation summary, API contract/database changes, tests/checks run, any blockers, and the exact next task ID. Stop after this task is complete.


TASK: H-10 — Production rollout and smoke test (FRONTEND ONLY)
Priority: P0
Dependencies: H-04, H-05, H-06, H-07, H-08

Implement the frontend portion only for this task.

TASK-SPECIFIC FRONTEND SCOPE:
Run production smoke checks: login, dashboard, brand filters, inventory search, receiving view, sales view and permission denial. Verify responsive shell and environment points to production API; do not expose staging endpoints or secrets.

ACCEPTANCE CRITERIA:
Deployment checklist, smoke test evidence, rollback plan and monitoring owners are documented; production change is not performed without authorization.

FRONTEND-SPECIFIC QUALITY GATES:
- Use React + TypeScript + Vite conventions already present in the repo, feature-based organization and accessible reusable components.
- Use TanStack Query for server state and the centralized API client; do not add direct fetch logic scattered across components.
- Use React Hook Form + Zod for complex forms where appropriate. Keep client validation consistent with, but not a replacement for, server validation.
- Handle loading, empty, no-results, API validation, 401, 403, conflict/stale data, network failure and success states where relevant.
- Never hardcode brands, catalogue specs, prices, statuses or permissions that should come from the API. Do not use fake success data in the real flow.
- Add focused component/hook tests and keep responsive/accessibility behavior intact.
- Do not implement Go endpoints or database migrations in this prompt. If an API contract is missing, document the expected contract and check existing OpenAPI rather than silently inventing incompatible fields.

```

## Execution note

Run the backend prompt first when the task introduces a new API/schema. Review and commit the API contract before running the frontend prompt. For UI-only tasks, use the existing API contract and do not expand backend scope without documenting a blocker. Run only one task per Claude Code session/turn.

---
