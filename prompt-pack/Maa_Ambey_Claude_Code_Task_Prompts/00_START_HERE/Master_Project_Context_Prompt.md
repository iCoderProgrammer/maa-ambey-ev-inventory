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
