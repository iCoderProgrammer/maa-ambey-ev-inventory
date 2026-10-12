# CLAUDE.md — Maa Ambey Enterprises Multi-Brand EV Inventory Management System

These are mandatory project-level engineering rules. Full procedure: `docs/TASK_EXECUTION_WORKFLOW.md`.
Progress tracking: `docs/TASK_EXECUTION_LOG.md`. Frontend work must also follow `docs/UI_GUIDELINES.md`
(component/dependency policy, tokens, motion, states, accessibility, UI checklist).

## Repository layout

- Go module at repo root: `cmd/` (api, migrate, seed), `internal/`, `migrations/`, `seeds/`, `docs/openapi/`.
- React app in `ui/`. Prompt-pack references to `/backend/...` map to repo root; `/frontend/...` maps to `ui/` (ADR-002).
- Root `Makefile` drives both: `make db-up`, `make db-migrate-up`, `make db-seed`, `make dev`,
  `make lint typecheck test build`, `make test-integration`, `make swagger`.
- Decisions and open questions: `docs/ARCHITECTURE_DECISIONS.md`. Setup: `docs/DEVELOPMENT_SETUP.md`.

## Confirmed stack (do not switch frameworks)

React + TypeScript + Vite, React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod
(Redux Toolkit for shell UI state only) · Go + Gin REST (`/api/v1`) · PostgreSQL via pgx, parameterised SQL ·
versioned SQL migrations (golang-migrate) · OpenAPI via swaggo · browser testing preferably Playwright.
No Next.js, NestJS, Node.js backend, Prisma or ORM.

## Sources of truth (read before every task)

| What | Where |
| --- | --- |
| Task IDs, titles, priorities, dependencies, prompt refs | `prompt-pack/TASK_REGISTRY.csv` (identical copy in `prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/`) |
| All prompts in one file | `prompt-pack/Maa_Ambey_Claude_Code_All_Task_Prompts_v1.2.md` (= `.../ALL_TASK_PROMPTS.md`) |
| Individual task prompt (Prompt A — Backend, Prompt B — Frontend) | `prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/<Epic Folder>/<file>.md` |
| PRD, Architecture, Business Rules, Design Spec, Implementation Tasks (Markdown) | `prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/REFERENCE_SPECIFICATIONS/0*_*.md` |
| Same documents (approved DOCX v1.2) | `documentation/` |
| Client decisions A-01–A-06 | `prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/00_START_HERE/00_Client_Discovery_Prompts.md` |

The task prompt defines scope and acceptance criteria; the approved documents define business rules.
Never silently change approved requirements to fit an implementation. Never edit `TASK_REGISTRY.csv`
or the approved documents unless explicitly asked.

## Client-confirmed baseline (v1.2)

- Initial brand roster: Lectrix EV, Ceeon India, Ujagar Electric. Brands are data, never code; more can be added.
  Roster ≠ supplier/dealership authorisation. Do not assume Ujagar Electric = Saffron EV Group.
- Roles: Owner, Inventory Manager, Sales Executive, Accountant, Viewer. Permission matrix and approval
  thresholds are **pending** — backend denies by default; never guess grants.
- Delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance, registration.
  Mandatory/conditional policy is **pending**.
- **Never invent:** model specifications, prices, warranties, supplier authorisation, opening stock,
  legal/GST values, invoice numbering, final permission rules, mandatory-vs-conditional delivery rules.

## Product invariants

- Every saleable scooter is an individual unit; normalised chassis/frame identifiers are unique.
- Stock-changing actions validate server-side, write append-only movement/audit history, and use
  transactions/idempotency where critical. No hard delete of posted stock/sale/payment/receipt/audit history.
- Server-side authorisation is authoritative; hiding a button is not security.
- No secrets/tokens in localStorage. Never reset or delete existing database data automatically.

## Mandatory task workflow (summary)

1. **One Task ID at a time.** Start only when the user says `Start <TASK_ID>`. Never pick the next task.
2. **Before starting:** find the registry row, read both prompts (A and B) fully, verify every dependency
   through actual code, tests and `docs/TASK_EXECUTION_LOG.md` (files existing ≠ complete). If a
   dependency is incomplete, report the blocker and ask whether to start it first — no duplicate
   workaround implementations. Define acceptance criteria in the log.
3. **Stages, in order, without pausing for approval** (unless a genuine blocker / high-risk decision):
   1. **UI first** — full task UI scope with loading/populated/empty/validation/API-error/permission-denied/
      confirm-cancel states, responsive + accessible. If no API yet, use a clearly isolated task-specific
      mock adapter (never scattered mocks; never fake a successful stock/financial transaction).
      Run frontend checks. Log `UI COMPLETE — MOCK DATA ONLY`.
   2. **Go backend** — migrations, Gin routes/handlers, DTOs, validation, services, pgx parameterised SQL,
      constraints/indexes, RBAC, audit, transactions, concurrency/duplicate protection, standard error
      envelope, tests. Run `gofmt`, `go vet`, `go test`, build. Log routes, migrations, results.
   3. **Integrate** — replace mock adapter in the production path with the typed API client + TanStack Query;
      remove mock success behaviour; handle real validation/permission/conflict/server errors; no success
      shown before backend confirms; verify persistence. Log endpoints and check results.
   4. **Browser test** — start PostgreSQL, Go API, Vite UI; exercise the real flow (success, validation,
      empty, errors, permissions, duplicates/conflicts, responsive, refresh, persistence; retries/concurrency
      for stock/financial flows). Inspect console + network. Fix only current-task issues and re-test.
      Never claim a browser pass that did not run → `BLOCKED — BROWSER TEST NOT VERIFIED`.
   - Backend/infra-only task: mark UI and/or browser stage `NOT APPLICABLE` with a reason; do not invent UI.
   - A-01–A-06 are discovery/client-input items: update documentation only; no UI/backend/browser work.
4. **Scope:** only the current task; no advance work on later tasks; no broad refactors; no disabling
   checks, weakening validation, hiding console errors or swapping real APIs for mocks to pass tests.
   Never discard uncommitted changes, reset Git history or overwrite unrelated work.
   If a critical requirement needs client approval, stop that part, record the blocker, ask one focused question.
5. **Log after every stage** in `docs/TASK_EXECUTION_LOG.md`. Statuses: `IN PROGRESS`, `BLOCKED`,
   `INCOMPLETE`, `COMPLETE`. A UI task is COMPLETE only when UI + backend + real API integration +
   automated checks + browser tests + acceptance criteria all pass with no task-blocking bugs.
6. **Finish:** update log, review final `git diff`, give the completion report (format in
   `docs/TASK_EXECUTION_WORKFLOW.md` §9), end with
   “Task [TASK_ID] is complete. I have not started the next task. Waiting for your instruction.” — then stop.
   Report honestly: never claim a check passed unless it actually ran and passed.
