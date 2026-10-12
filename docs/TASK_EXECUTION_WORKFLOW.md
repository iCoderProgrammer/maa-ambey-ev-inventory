# Task Execution Workflow — One Task at a Time

Mandatory process for every task in this repository (Maa Ambey Enterprises — Multi-Brand EV
Inventory Management System). Summary rules live in the root `CLAUDE.md`; progress is tracked in
`docs/TASK_EXECUTION_LOG.md`.

Goal: complete exactly one task end-to-end, verify it in the browser where applicable, report it,
then wait for an explicit instruction (e.g. `Start B-02`) before starting another.

---

## 1. Stack and confirmed requirements

| Layer | Choice |
| --- | --- |
| Frontend | React + TypeScript + Vite (`ui/`), React Router, Tailwind CSS, TanStack Query, React Hook Form + Zod |
| Backend | Go + Gin REST API (repo root: `cmd/`, `internal/`), base path `/api/v1` |
| Database | PostgreSQL via pgx, parameterised SQL; versioned SQL migrations in `migrations/` |
| API docs | OpenAPI/Swagger via swaggo (`make swagger` → `docs/openapi/`) |
| Browser testing | Playwright preferred (not yet installed — see §5 Stage 4) |

Client-confirmed baseline:

- Initial operational brands: **Lectrix EV, Ceeon India, Ujagar Electric**. Architecture must allow more.
- Role names: **Owner, Inventory Manager, Sales Executive, Accountant, Viewer**. Detailed permission matrix pending.
- Never invent official model specs, prices, warranties, supplier authorisation, opening stock,
  legal/GST values, final permission rules, or mandatory-vs-conditional delivery requirements.

## 2. Documents to read before each task

| Document | Location |
| --- | --- |
| Task registry (source of truth for IDs, titles, priority, dependencies, prompt refs) | `prompt-pack/TASK_REGISTRY.csv` |
| All prompts | `prompt-pack/Maa_Ambey_Claude_Code_All_Task_Prompts_v1.2.md` (identical: `prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/ALL_TASK_PROMPTS.md`) |
| Selected task file | `prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/<Epic Folder>/<file>.md` — read **Prompt A — Backend** and **Prompt B — Frontend** completely |
| PRD | `.../REFERENCE_SPECIFICATIONS/01_PRD.md` (DOCX: `documentation/01_PRD_v1.2.docx`) |
| System Architecture | `.../REFERENCE_SPECIFICATIONS/02_System_Architecture.md` |
| Business Rules | `.../REFERENCE_SPECIFICATIONS/03_Business_Rules.md` |
| Design Specification | `.../REFERENCE_SPECIFICATIONS/04_Design_Specification.md` |
| Implementation Tasks | `.../REFERENCE_SPECIFICATIONS/05_Implementation_Tasks.md` |
| Client decisions A-01–A-06 | `.../00_START_HERE/00_Client_Discovery_Prompts.md` |
| Implementation decisions / stack deviations | `docs/ARCHITECTURE_DECISIONS.md` (ADR-001: Go+React supersedes Next/Nest; ADR-002: Go at root, React in `ui/`) |

The task prompt defines scope and acceptance criteria. The approved documents define business rules.
Do not silently change approved requirements. Do not edit `TASK_REGISTRY.csv` unless explicitly asked.

## 3. Session start

- Master workflow prompt given **without** a Task ID → inspect repo/Git status, read registry and docs,
  create/update `CLAUDE.md`, this file and the execution log, start no feature work, report ready, wait.
- `Start <TASK_ID>` → work only on that task.

## 4. Dependency verification (before any code)

1. Find the exact row in `TASK_REGISTRY.csv`.
2. Read both prompts of the task file.
3. Check every listed dependency.
4. Verify each through actual code, tests and `docs/TASK_EXECUTION_LOG.md`. Files existing ≠ complete.
5. Inspect relevant components, API routes, migrations, tests and Git changes.
6. Write the task's acceptance-criteria checklist into the log.

Incomplete dependency → report the blocker and ask whether to start that dependency first. Never build
a duplicate implementation to bypass it. Never discard uncommitted changes, reset Git history or
overwrite unrelated work. **Only one Task ID may be IN PROGRESS at any time.**

A-01–A-06 are discovery/client-input items, not part of the 59-task registry: update the relevant
documentation only; do not create UI, backend or browser work for them.

## 5. Implementation stages (UI tasks)

Run the four stages in this exact order. Do not pause for approval between stages unless a genuine
blocker or high-risk decision needs the user's input.

### Stage 1 — UI first

Follow `docs/UI_GUIDELINES.md` (components, dependency policy, tokens, motion, states, accessibility).

- Implement the full UI scope from Prompt B and the Design Specification: screens, components, forms,
  tables, navigation, filters, interactions.
- States: initial/loading, populated, empty, validation errors, API/error, permission-denied (where
  applicable), confirmation and cancellation, responsive layout, accessible controls.
- Reuse existing design tokens, components and conventions (`ui/src/components`, `ui/src/lib`, …).
- No backend API yet → a clearly isolated, task-specific mock/fixture adapter (single module, swappable).
  No hardcoded mock data scattered in components. Mock data never presented as verified production data.
  Never simulate a successful financial or stock transaction as if a backend had committed it.
- Stay inside the task's scope.
- Run frontend checks (`make lint-frontend`, `make typecheck`, `make test-frontend`, `make build-frontend`)
  and fix UI errors.
- Log: `UI COMPLETE — MOCK DATA ONLY`. Proceed to Stage 2.

### Stage 2 — Go backend and business logic

- From Prompt A and approved business rules, as the task requires: migrations (with working `down`),
  Gin routes/handlers, request/response DTOs, input validation, service logic, pgx parameterised SQL,
  constraints and indexes, role/permission enforcement (deny by default), audit events, transactions,
  duplicate/concurrency protection, standard error envelope `{code, message, details, requestId}`, tests.
- Only current-task backend scope. Critical invariants enforced on the server; frontend validation
  never replaces it.
- Never reset or delete existing database data automatically. Never invent commercial, catalogue,
  tax, stock, supplier or permission data.
- Run `make lint-backend` (gofmt + go vet), `make test-backend`, `make test-integration` where DB tests
  apply, `make build-backend`, `make swagger` when routes change. Fix current-task failures.
- Log routes, migrations, changes and actual results. Proceed to Stage 3.

### Stage 3 — Real API integration

1. Replace mock adapters in the real application path with the centralised typed API client.
2. Remove obsolete mock success behaviour from the production path.
3. Implement real create/read/update and other permitted actions for this task.
4. Wire tables, filters, search, pagination, forms and buttons to endpoints.
5. Handle real backend validation errors and permission failures.
6. Show loading, empty, success, network-error, conflict and server-error states correctly.
7. Never show success before the backend confirms it.
8. Revalidate task business rules through server responses.
9. Verify mutations persist and refresh (TanStack Query invalidation).
10. Leave unrelated features and APIs untouched.

Run frontend checks, backend tests, type checks and builds; fix integration errors. Log endpoints
integrated and results. Proceed to Stage 4.

### Stage 4 — Browser testing (mandatory for any user-facing screen)

**A. Start the app** (documented commands): `make db-up` → `make db-migrate-up` → `make db-seed` →
`make dev` (API `http://localhost:8080/api/v1/health`, UI `http://localhost:5173`). Confirm health.
Prefer Playwright. None is configured yet: the first task needing browser tests adds the smallest
suitable Playwright setup in `ui/` (record it in the log) without unrelated changes.

**B. Exercise the real flow** — choose per acceptance criteria: main success flow; validation and
invalid input; empty and no-result searches; loading and API-error states; permission restrictions;
duplicates/conflicts; responsive layout; navigation and refresh; persistence and retrieval.
Stock/purchase/reservation/payment/sale tasks: also verify retries, concurrent requests and failures
create no duplicate or partial transactions.

**C. Investigate and fix** — layout problems, broken interactions, console errors, unhandled
exceptions, failed network calls, unexpected responses, validation/permission issues, persistence
problems. Find the root cause, fix only current-task issues, rerun browser and automated checks,
repeat until passing or a genuine blocker remains. Never hide console errors, replace the live API
with mocks, weaken validation or skip failing tests. Use non-production test data. Save evidence
(screenshots, reports) under `docs/evidence/<TASK_ID>/` where supported.

**Never claim browser testing passed if it was not executed.** If the browser or environment is
unavailable, mark the task `BLOCKED — BROWSER TEST NOT VERIFIED` and explain why.

**D. Tasks without a user-facing UI** — do not invent UI. Mark the UI stage `NOT APPLICABLE` with a
short reason; run backend, integration and automated validation; mark browser testing
`NOT APPLICABLE` only when the task scope justifies it.

## 6. Scope and bug-fixing boundaries

- Work only on the selected Task ID; fix bugs introduced by or directly affecting it.
- No advance implementation of later tasks; no broad unrelated refactors.
- No mocks to make tests pass; no disabling quality checks.
- No fabricated brand data or client-approved business rules.
- Preserve existing working functionality.
- Document unavailable tools, external dependencies and unverified behaviour honestly.
- Critical requirement awaiting client approval → stop that part safely, record the blocker,
  ask one focused question.

## 7. Execution log

Update `docs/TASK_EXECUTION_LOG.md` after **each stage**. Per task record: ID and title; dependencies
and verified status; acceptance-criteria checklist; UI / backend / integration / browser results;
commands run with actual outcomes; important files changed; migrations and endpoints; browser
scenarios; evidence paths; known issues / pending approvals; final status.

Statuses: `IN PROGRESS`, `BLOCKED`, `INCOMPLETE`, `COMPLETE`.

A UI task is COMPLETE only when: (1) UI finished, (2) backend finished, (3) real API integrated,
(4) applicable automated checks passed, (5) browser tests passed, (6) acceptance criteria met,
(7) no unresolved task-blocking bugs.

## 8. Stop after one task

Update the log → review the final `git diff` → stop. Do not choose, start or prepare the next task or
other work from the same epic. Report and wait for `Start <TASK_ID>`. If blocked or incomplete,
report it honestly with what is needed to continue.

## 9. Completion report template

```markdown
### Task: [TASK_ID] — [Task Title]

- **Overall status:** COMPLETE / BLOCKED / INCOMPLETE
- **UI stage:** COMPLETE / NOT APPLICABLE / BLOCKED
- **Backend stage:** COMPLETE / NOT APPLICABLE / BLOCKED
- **API integration:** COMPLETE / NOT APPLICABLE / BLOCKED
- **Browser testing:** PASS / FAIL / NOT APPLICABLE / NOT VERIFIED
- **Files changed:** Important files
- **API/database changes:** Endpoints and migrations, if any
- **Validation:** Commands executed and their actual outcomes
- **Browser checks:** Scenarios tested and results
- **Evidence:** Screenshot or test-report location, if available
- **Pending issues/approvals:** Explicitly list them
- **Execution log:** docs/TASK_EXECUTION_LOG.md
```

If complete, end with: **“Task [TASK_ID] is complete. I have not started the next task. Waiting for your instruction.”**
If blocked, state the exact blocker and the input needed; do not move to another task.
