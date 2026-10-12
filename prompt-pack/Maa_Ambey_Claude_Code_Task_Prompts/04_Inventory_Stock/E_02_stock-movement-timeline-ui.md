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
