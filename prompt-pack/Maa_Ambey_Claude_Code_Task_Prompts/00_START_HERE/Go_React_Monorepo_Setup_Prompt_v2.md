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
