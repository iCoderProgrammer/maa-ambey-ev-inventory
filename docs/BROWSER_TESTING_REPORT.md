# Browser Testing Report — 2026-11-11

User-requested browser verification of every completed feature (not a registry task; no new feature or pending task
started). Project: Maa Ambey Enterprises Multi-Brand EV Inventory. The request named "Saral Sampark", but every
artefact it lists (CLAUDE.md, `docs/TASK_EXECUTION_LOG.md`, `TASK_REGISTRY.csv`, Playwright E2E files) exists only in
this repository, so it was run here. Base: `dev` at `e82572d` plus the uncommitted D-06 frontend (preserved).
Previous audit of the same kind: `docs/evidence/REGRESSION-AUDIT-2026-10-26/report.md` (up to D-03).

Evidence: `docs/evidence/BROWSER-TEST-2026-11-11/` (one Playwright output per feature, crawl results before and after
the fix, `make ci` output, fixture SQL, GRN screenshots, crawl script).

## 1. Environment

| Item | Value |
| --- | --- |
| Database | throwaway `postgres:18-alpine` container `maa-bt-pg` (:5494), migrated to 19 + seeds |
| API | built from this tree, :8096, `APP_ENV=development`, `RBAC_ALLOW_PROPOSED_GRANTS=true`, `ATTACHMENT_LINK_TTL=30s`, raised login/rate limits for the suite |
| UI | Vite :5188, proxy → :8096 |
| Accounts | 8 test-only `e2e.*` users, one random password in a 0600 scratch file (never printed or committed) |
| Grants | PROPOSED grants on the throwaway DB only, as listed in DEVELOPMENT_SETUP "End-to-end browser tests" (B-07 … D-06); location `BT1` |
| Fixtures | eligible-PO fixture (`po-fixture.sql`) and 8 vehicle units awaiting PDI (`docs/evidence/D-06/browser-fixture.sql`; stands in for D-07 posting) — placeholder data |
| Browsers | Playwright with installed Google Chrome (headless): desktop 1366×900 and Pixel 7; crawl also at tablet 768×1024; visible pass in the user's Chrome (Claude in Chrome) |
| Shared dev stack | not touched (shared DB still at migration 12; no connection, grant or migration) |

## 2. Completed-feature inventory

Sources: status board in `docs/TASK_EXECUTION_LOG.md`, `prompt-pack/TASK_REGISTRY.csv`, `ui/src/app/{routes,navigation}`,
`ui/e2e/*.e2e.ts`, OpenAPI. Verified against the code (routes and handlers exist).

| # | Task | Feature | Screen | Tested by |
| --- | --- | --- | --- | --- |
| 1 | B-01…B-05 | Shell, navigation, health/readiness, migrations status, theme | Dashboard, sidebar/drawer | `shell.e2e.ts`, crawl, Chrome |
| 2 | B-06 | Sign-in, sessions, CSRF, throttling, sign-out | `/login`, account menu | `auth.e2e.ts` |
| 3 | B-07 | RBAC deny-by-default, location scope, roles page | `/admin/users` | `access.e2e.ts`, crawl (5 accounts) |
| 4 | B-08 | Append-only audit events | none by design | via the feature flows + DB checks (D-06, E-05 runs) |
| 5 | B-09 | Private attachments, signed short-lived links | `/documents` | `attachments.e2e.ts`, Chrome |
| 6 | C-01…C-03 | Brand/model/variant catalogue, lifecycle, activation gate | `/admin/catalogue/*` | `catalogue.e2e.ts`, `catalogue-ui.e2e.ts`, Chrome |
| 7 | C-04 | Effective-dated prices, maker-checker, cost visibility | brand/variant price cards | `prices.e2e.ts` |
| 8 | C-06 | Brand isolation | catalogue filters | `isolation.e2e.ts` |
| 9 | D-01 | Supplier master | `/procurement/suppliers*` | `suppliers.e2e.ts`, Chrome |
| 10 | D-02 | Purchase orders | `/procurement/purchase-orders*` | `purchase-orders.e2e.ts`, Chrome |
| 11 | D-03 | Goods receipt drafts | `/procurement/goods-receipts*` | `goods-receipts.e2e.ts`, Chrome |
| 12 | D-04, E-01 | Vehicle unit registry, stock movement ledger (API + typed client; no own screen) | — (used by D-06/E-05) | via PDI and counts flows, DB checks |
| 13 | E-03 | Spare part / accessory SKU master | `/inventory/parts*` | `parts.e2e.ts`, Chrome |
| 14 | E-04 | Parts receipts and issues | SKU detail stock panel | `parts-stock.e2e.ts` |
| 15 | E-05 | Stock counts and adjustment approval | `/inventory/counts*` | `stock-counts.e2e.ts`, Chrome |
| 16 | D-06 | PDI checklist and disposition | `/procurement/pdi*` | `pdi.e2e.ts`, crawl, Chrome |

**Not in the completed list (reported separately):** D-05 duplicate identifier controls — IN PROGRESS (the editor's
in-draft duplicate handling is exercised by `goods-receipts.e2e.ts`; the live posting conflict needs D-07). C-05 —
blocked on client data (A-01/A-03). Everything else (D-07…D-09, E-02, E-06, E-07, F-*, G-*, H-*) is NOT STARTED: only
their "Planned" placeholders were checked by the crawl.

## 3. Results by feature

PASS = verified in a real browser on the isolated stack in this run. Counts are Playwright tests (desktop + mobile).

| Feature | Scenarios (unique) | Passed | Skipped (by design) | Failed | Result |
| --- | --- | --- | --- | --- | --- |
| Shell / navigation (B-03…B-05) | 7 | 12 | 2 (desktop-only / mobile-only nav tests) | 0 | PASS |
| Authentication (B-06) | 11 | 22 | 0 | 0 | PASS |
| Access control (B-07) | 2 | 4 | 0 | 0 | PASS |
| Attachments (B-09) | 4 | 7 | 1 (link-expiry runs once, desktop) | 0 | PASS |
| Catalogue API (C-02) | 2 | 2 | 2 (API-only, desktop once) | 0 | PASS |
| Catalogue UI (C-03) | 3 | 6 | 0 | 0 | PASS |
| Prices (C-04) | 1 | 2 | 0 | 0 | PASS |
| Brand isolation (C-06) | 1 | 2 | 0 | 0 | PASS |
| Suppliers (D-01) | 3 | 6 | 0 | 0 | PASS |
| Purchase orders (D-02) | 3 | 6 | 0 | 0 | PASS |
| Goods receipts (D-03, D-05 editor) | 2 | 4 | 0 | 0 | PASS |
| Parts master (E-03) | 3 | 6 | 0 | 0 | PASS |
| Parts stock (E-04) | 1 | 2 | 0 | 0 | PASS |
| Stock counts (E-05) | 1 | 2 | 0 | 0 | PASS |
| PDI (D-06) | 1 | 2 (+2 on re-run after the fix) | 0 | 0 | PASS (after fix) |
| **Total** | **45** | **85** | **5** | **0** | |

What the scenarios cover (beyond page loads): sign-in errors, disabled accounts, throttling, session expiry and
sign-out; deny-by-default for gated links and direct URLs; upload validation, private signed links that expire and are
bound to one user; catalogue duplicates, activation gate, archive, version conflicts and read-only viewers;
price drafting, approval by another person, cost visibility; cross-brand refusals; supplier create/document/verify/
brand links/contacts/archive, focus return, offline save; PO draft lines and totals, refused submission for an
unverified supplier/inactive brand, no self-approval, approval by another user, cost snapshot; GRN partial drafts with
units/documents, over-receipt authorisation, duplicate chassis in draft, stale save refused, cancel, no stock change;
SKU master with duplicates/archive; parts receive/issue with insufficient-stock refusal and one-time retry; stock count
record → submit → approve once (double click) → stale stock refused → reject; PDI checklist configuration, pass,
hold with note and photo, saleable search, stale version 409, evidence download, reject, read-only and denied users.

### Crawl (all routes × accounts × viewports)

33 routes (every implemented page and detail page with a real ID, every planned placeholder, an unknown path) × 5
accounts (owner-like, viewer, approver, no grants, roles-only) × 3 viewports (desktop, tablet 768, mobile 390) =
**495 page loads per run**. Checks: exactly one visible `<h1>`, no horizontal overflow, no unnamed buttons/links, no
unlabelled inputs, no duplicate IDs, no console or page errors, no 5xx, no unexpected redirect to sign-in.

- First run: **9 flagged** — one defect (BUG-1 below).
- After the fix: **0 flagged** (`crawl-rerun.txt`). Denied pages per account match each account's grants exactly
  (e.g. owner denied only the roles page; approver denied suppliers, parts and PDI; no-grant user denied every gated
  page; planned placeholders and not-found open for everyone).

### Visible pass in the user's Chrome (first, page-level)

Signed in as the test owner: dashboard, catalogue, suppliers, purchase orders, goods receipts, parts, stock counts,
PDI queue, documents and a planned module rendered with real data; no console errors. The user did not see this pass
(fast, separate tab group), so the interactive walkthrough below followed.

### Interactive walkthrough in the user's Chrome (user-requested follow-up)

After the user said the first Chrome pass was not visible, the stack was rebuilt (same throwaway setup, fresh DB,
same fixtures) and every screen was exercised step by step in the user's own Chrome (Claude in Chrome, new tab group),
with pauses and a narration of each action. Three rounds, all PASS, no console errors:

| # | Screen | Actions performed | Verified result |
| --- | --- | --- | --- |
| 1 | Sign in | empty submit; wrong password; correct password | field errors; "Incorrect username or password", password cleared; dashboard |
| 2 | Catalogue | empty Add brand; create "Chrome Demo Brand" | "Name is required"; brand created Pending verification / Authorisation unverified with readiness checklist |
| 3 | Suppliers | list; verified supplier detail | verified identity, verified brand relationship, contacts |
| 4 | Purchase orders | empty New PO; draft PO-000002; add line (dependent brand → model → variant, cost list); submit | "Choose a supplier"; "cannot be submitted yet: at least one line"; server total ₹12,350, cost snapshot ₹900 with variance warning; Awaiting approval, no Approve for its creator |
| 5 | PDI | empty checklist item; add BRAKES; empty review; Fail without note; Fail + note + notes + photo → Hold; second unit Pass | field errors; "Choose Pass or Fail"; "Describe what failed"; PDI-000001 Quality hold, not saleable; PDI-000002 Available, saleable; form closed afterwards |
| 6 | Permissions | sign in as no-grant user | gated links hidden; "Access denied" with page heading (BUG-1 fix confirmed) |
| 7 | PO approval | approver opens PO-000002, approves | catalogue cost hidden ("not visible to you"); Approved by the approver |
| 8 | Goods receipts | draft GRN-000002 on PO-000002 (3 received, 1 rejected); in-draft duplicate chassis (case/space); chassis already on a posted receipt; correct value; reload | accepted 2 computed by server, "does not change stock"; "Same chassis… Nothing was saved"; server 409 "already on the posted goods receipt GRN-000001", entries kept; draft saved; data persisted after reload |
| 9 | Spare parts | category, SKU BRK-PAD-01, receive 10, issue 15, issue 2 | balance 10; "not enough stock… Nothing was changed"; balance 8; history with both entries |
| 10 | Stock counts | count SC-000001: 7 counted (record 8) without reason, with reason, submit | "Give a reason"; counter sees no expected quantity and no Approve; Awaiting approval, stock unchanged |
| 11 | Documents | upload `notes.txt`; upload PNG; preview | "file type is not accepted"; stored privately; preview dialog, Escape closes |
| 12 | Count approval | approver: queue → SC-000001 → approve | 7 / 8 / −1 shown to approver; "1 adjustment was posted"; DB ledger 0→10 → 8 → 7, balance 7 |
| 13 | Users & roles | approver opens page; roles-only user opens page; same user opens Suppliers | "Access denied" and no link; read-only matrix (pending sign-off, proposed grants labelled, permission catalogue), no edit controls; Suppliers denied |

Notes: one PO quantity was entered as 13 instead of 3 (the field held "1" and the typing appended — operator input,
total correct for 13); one click missed Save after a scroll (API log showed no request) and was repeated — neither is
an app defect. The `e2e.roles` user was added to the throwaway DB for round 13 only.

## 4. Bugs

| ID | Feature | Severity | Type | Status |
| --- | --- | --- | --- | --- |
| BUG-1 | PDI inspection page (D-06, uncommitted frontend) | Low (accessibility) | Frontend | FIXED and re-tested |

**BUG-1 — PDI inspection page has no page heading while gated.**
- Steps: sign in as a user without `vehicle_units.view` (e.g. `e2e.active`), open `/procurement/pdi/<unit id>`.
- Expected: access-denied message under one page heading (the convention fixed in the 2026-10-26 audit, fix #1
  "denied detail keeps heading").
- Actual: the denied message with no `<h1>` (also while permissions load or fail), at every viewport.
- Root cause: `PdiInspectionPage` passed `header={undefined}` to the permission gate to avoid a double heading once the
  page renders its own, so the gated states had none.
- Fix: the gate takes a `gatedHeader` shown only while it holds the page (`ui/src/features/pdi/PdiPages.tsx`);
  regression assertion added to `pdi.test.tsx` (fails without the fix, passes with it).
- Re-test: crawl 495 loads, 0 flagged; `pdi.e2e.ts` 2 passed; Vitest 317 passed.

No other defect found. Not bugs (recorded so nobody chases them): the "evidence.pdf" in the owner's *My uploads* comes
from the PO SQL fixture (it links a supplier document without setting the attachment owner, as the Go test fixture
does); the goods receipt "Posted" in lists is the PDI fixture standing in for D-07.

## 5. Console and network

- Playwright `watchConsole` assertions in every spec: no console errors or page errors (expected non-2xx responses
  are asserted by the tests themselves).
- Crawl: no console/page errors and no 5xx in 990 page loads (two runs).
- Chrome visible pass: no console errors.

## 6. Desktop / mobile

All 15 specs on desktop (1366×900) and Pixel 7; crawl at 1366, 768 and 390 px: no horizontal overflow, one heading,
labelled controls on every page for every account.

## 7. Regression

Every feature's suite was re-run after the BUG-1 fix through the full gate (`make ci`), and the affected feature
(PDI) was re-run in the browser together with the full crawl. Cross-feature journeys exercised: catalogue → prices →
PO → GRN (procurement chain), parts master → parts stock → stock counts (ledger), vehicle units → PDI → saleable
search (lifecycle).

## 8. Final checks (actually executed)

| Check | Result |
| --- | --- |
| Playwright E2E (15 specs, isolated stack) | 85 passed, 5 skipped by design, 0 failed; PDI re-run 2 passed |
| Crawl | 495 loads, 0 flagged after fix |
| Interactive Chrome walkthrough (13 steps, 3 rounds) | all PASS, no console errors |
| `make ci` (TEST_DATABASE_URL = throwaway) | exit 0: swagger-check, gofmt, go vet, go mod tidy, ESLint (0 warnings), Prettier, tsc, Go tests with PostgreSQL (495 passing), Vitest 31 files / 317 tests, builds |
| golangci-lint | not installed (skipped by the Makefile; go vet ran) |

## 9. Not tested / blocked

- Features not completed (section 2) — out of scope by instruction.
- Real (non-fixture) vehicle units and the live `DUPLICATE_IDENTIFIER` posting conflict — need D-07.
- Production grants — none exist (RB-5); every grant used here is a PROPOSED test grant on the throwaway DB.
- Audit Log screen — does not exist by design (B-08 has no UI); audit rows were checked in the database.

## 10. Files changed by this run

- `ui/src/features/pdi/PdiPages.tsx` (BUG-1 fix), `ui/src/features/pdi/pdi.test.tsx` (regression assertion) — both
  part of the uncommitted D-06 frontend.
- `docs/BROWSER_TESTING_REPORT.md` (this file), `docs/evidence/BROWSER-TEST-2026-11-11/*`, `docs/TASK_EXECUTION_LOG.md`
  (one entry). No backend, migration, registry or approved document changed. Nothing committed or pushed.
