# Full regression audit — 2026-10-26

User-requested end-to-end testing, bug-fixing and evidence pass over everything implemented up to D-03.
Not a registry task. No new product task started. Base: `dev` = `e962ee6` (D-03), clean tree at start.

## 1. Environment

| Item | Value |
| --- | --- |
| Isolated database | `postgres:18-alpine` container `maa-audit-pg`, `127.0.0.1:5460`, random password (never printed), migrated to 12 + seeds |
| Isolated API | built from this tree, `:8099`, `APP_ENV=development`, `RBAC_ALLOW_PROPOSED_GRANTS=true`, `ATTACHMENT_LINK_TTL=30s`, `LOGIN_MAX_FAILURES_PER_IP=1000`, `RATE_LIMIT_RPS=100/BURST=200`, private attachment dir in a scratch folder |
| Isolated UI | Vite `:5181`, proxy → `:8099` (`VITE_API_BASE_URL=/api/v1`, checked) |
| Accounts | 8 test-only users (`e2e.*`), one random password from a `0600` scratch file, never printed or committed |
| Grants | PROPOSED grants on the throwaway DB only (same set as DEVELOPMENT_SETUP "End-to-end browser tests") |
| Fixture | Eligible-PO fixture (placeholder brand/model/variant, verified supplier + link, cost list with approved COST 900.00) inserted by SQL on the throwaway DB only |
| Shared dev stack | Vite `:5173`, API `:8081`, DB `:5436` — **not touched** (no connection, no grant, no migration) |
| Browser | Installed Google Chrome 156 (Playwright `channel: chrome`), headless |
| Toolchain | Go 1.27.3 (`/snap/go/current/bin`), Node 24.3.0 |

## 2. Implemented-feature inventory

Source: `docs/TASK_EXECUTION_LOG.md`, `TASK_REGISTRY.csv`, `ui/src/app/{routes,navigation}.ts(x)`, `docs/openapi/swagger.json` (74 operations), `ui/e2e/*.e2e.ts`.

| Task | Feature | UI | API | Status in log |
| --- | --- | --- | --- | --- |
| B-01…B-05 | Shell, health/readiness, migrations, OpenAPI, CI | Dashboard, nav, theme, offline pill | `/health`, `/ready` | COMPLETE |
| B-06 | Sign-in, sessions, CSRF, throttling, sign-out | `/login`, account menu | `/auth/*`, `/me` | COMPLETE |
| B-07 | RBAC (deny by default), location scope, roles page | `/admin/users` | `/access/roles` | COMPLETE |
| B-08 | Append-only audit events | none (Audit Log stays planned by design) | written by every module | COMPLETE |
| B-09 | Private attachments, signed short-lived links | `/documents` | `/attachments*` | COMPLETE |
| C-01…C-03, C-06 | Brands, models, variants, colours, lifecycle, brand isolation | `/admin/catalogue/*` | `/brands`, `/models`, `/variants`, `/colours` | COMPLETE |
| C-04 | Effective-dated price history, maker-checker, cost visibility | brand/variant price cards | `/price-lists`, `/price-items`, `/prices/lookup` | COMPLETE |
| D-01 | Supplier master, contacts, documents, verification, brand links | `/procurement/suppliers*` | `/suppliers*`, `/supplier-brand-links*` | COMPLETE |
| D-02 | Purchase orders: draft, lines, snapshots, submit/approve/return/cancel | `/procurement/purchase-orders*` | `/purchase-orders*`, `/purchase-order-settings` | COMPLETE (status board row was stale — fixed) |
| D-03 | Goods receipt drafts (no stock change) | `/procurement/goods-receipts*` | `/goods-receipts*`, `/receivable-purchase-orders` | COMPLETE (status board row was stale — fixed) |
| C-05, D-04… | — | planned placeholders only | — | NOT STARTED (not tested as features; placeholders checked) |

## 3. Coverage matrix and results

Legend: **PASS** verified in the browser/API on the isolated stack this run · **FIXED** defect found, fixed, re-verified.

| Feature | Scenarios exercised (beyond the existing suite where noted) | Desktop 1366×900 | Mobile Pixel 7 | 320 px | Result |
| --- | --- | --- | --- | --- | --- |
| Shell / navigation | dashboard live status, planned modules, not-found, theme persistence, sidebar + drawer, focus after navigation; crawl of 20 routes × 5 accounts | PASS | PASS | PASS | PASS |
| Authentication | suite (11 tests/viewport) + probes: same message for unknown user / wrong password, disabled revealed only after correct password, empty/oversized body 400, form-encoded 415, case/space-insensitive username, `HttpOnly; SameSite=Strict` cookie, per-user throttle 5 → 429 `Retry-After: 900` (correct password also refused while throttled), logout needs CSRF, wrong CSRF 403, **session rotated on re-login and old cookie 401** | PASS | PASS | — | PASS |
| Access control | sweep of all 74 operations: anonymous 401, no-grant user 403 (`/me` 200), missing CSRF 403 on every mutation; nav hides gated links; direct URLs denied | PASS | PASS | PASS | **FIXED #1** |
| Attachments | suite (upload/list/preview/download/refresh, server rejects bad files, signed links expire, bound to one user) + 11 MB file refused client-side with size message, offline upload shows "Cannot reach the server" with Retry | PASS | PASS | — | PASS |
| Audit events | supplier create/update rows written (`supplier.created`, `supplier.updated` ×2); `UPDATE`, `DELETE`, `TRUNCATE` on `audit_events` refused by trigger | — | — | — | PASS |
| Catalogue & brands | suite (deny by default, relationships, duplicates, activation gate, archive, versions, viewer read-only, admin CRUD) + list param validation (bad limit/page/status/brandId → 422, search with `% ' " \ _` → 200), unknown fields 422, malformed JSON 400, non-UUID / missing IDs 404 | PASS | PASS | PASS | PASS |
| Brand isolation | suite (cross-brand filters/requests refused, stale links cleaned) | PASS | PASS | — | PASS |
| Pricing | suite + probes: overlapping approval → 409 `PRICE_OVERLAP`, user without approve permission 403, selling-only user: COST list/lookup 403, COST item 404, list shows only SELLING | PASS | PASS | — | PASS |
| Suppliers | suite + double-submit (button disabled, 1 POST), stale `version` 409, 5 parallel PATCHes on one version → 1×200 + 4×409, pagination (43 records), `?page=abc`, page past the end, special-character search, empty search state, back/forward | PASS | PASS | PASS | **FIXED #4 #5 #6** |
| Purchase orders | suite (draft/lines/totals, unverified supplier + inactive brand refused, submit, self-approval 403, approval by another user, cost snapshot hidden from approver, read-only viewer); PO cost redaction reviewed in code | PASS | PASS | PASS after fix | **FIXED #2** |
| Goods receipts | suite (partial draft with units/documents, over-receipt needs authorised override, duplicate chassis, stale save refused, cancel, read-only viewer, no stock change) + editor offline save | PASS | PASS | PASS | PASS (benefits from #5) |
| Roles page | suite (gated, pending sign-off shown) | PASS | PASS | PASS | PASS |
| Dialogs (all modules) | Escape/Cancel close, focus trap, initial focus, focus return, 8 create dialogs at 320 px (no overflow) | PASS | PASS | PASS | **FIXED #4** |

Crawl after fixes (`crawl-pages-roles-viewports.txt`): 20 routes × 5 accounts (owner-like, viewer, approver, no grants, roles) × 3 viewports = **300 page loads, 0 flagged** (exactly one `<h1>`, no horizontal overflow, no unnamed buttons/links, no unlabelled inputs, no duplicate IDs, no console/page errors, no 5xx).

## 4. Defects found and fixed

| # | Severity | Defect | Steps to reproduce | Root cause | Fix | Regression test | Re-verified |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | Low (a11y) | A denied **detail** page (PO, supplier, goods receipt) showed only "Access denied": no `<h1>`, no breadcrumbs (list pages keep theirs) | Sign in without `purchase_orders.view` → open `/procurement/purchase-orders/<id>` | Detail pages rendered `PageHeader` inside the permission `Gate`, which returns only the state view | `Gate` takes an optional `header` shown with its loading/error/denied states; the three detail pages pass their `PageHeader` | Vitest ×3 (`purchasing`, `suppliers`, `receiving`: "keeps the heading and breadcrumbs when a detail link is denied") — failed before, pass after; e2e `suppliers.e2e.ts` denied test now also opens a detail link | `fix1-denied-detail-keeps-heading.png`; crawl 0 flagged |
| 2 | Low (responsive) | PO line warning "Agreed cost differs from the catalogue cost" overflowed its card by 23 px at 320 px (WCAG reflow width) | 320 px viewport → PO with a cost-variance line | `Badge` is `whitespace-nowrap` (status pills); used here for a sentence-length message | Warning badges on PO lines wrap (`whitespace-normal`, card radius) | e2e `purchase-orders.e2e.ts` approval test asserts no sideways scroll at 320 px — failed (23 px) before, passes after | `fix2-po-warning-wraps-320px.png` |
| 4 | Medium (a11y, WCAG 2.4.3) | After any app dialog closed (Escape, Cancel, ×), keyboard focus fell to `<body>` instead of the button that opened it, although `Dialog` documented "focus return to the opener" | Focus "Add supplier" → Enter → Escape → focus is on `<body>` (same for every dialog) | Radix modal content restores focus only to its own `Dialog.Trigger`; the app opens dialogs from ordinary buttons, so `triggerRef` is null | `Dialog` remembers the focused element on open (`onOpenAutoFocus`) and restores it on close if still in the page (`onCloseAutoFocus`); otherwise the shell's navigation focus applies | Vitest `Dialog.test.tsx` "returns focus to the button that opened it after Escape or Cancel" (failed before); e2e `suppliers.e2e.ts` "dialogs return focus…" | Browser: Add supplier / Add brand / New purchase order return focus after Escape and Cancel |
| 5 | Medium (UI states guideline) | Saving while the browser was offline left the dialog stuck: Submit **and Cancel** disabled, no message, indefinitely; the write was then sent automatically on reconnect | Open "Add supplier", fill, go offline, click Add supplier → nothing explains the wait, Cancel disabled; reconnect → supplier created. Same pause for every mutation (GRN editor save etc.) | TanStack Query mutations default to `networkMode: 'online'`, which pauses them while `navigator.onLine` is false and resumes later | `createQueryClient`: `mutations.networkMode: 'always'` — writes are attempted, fail at once with the existing "Cannot reach the server. Nothing was saved…" error, entries kept, Cancel enabled, nothing replayed later | Vitest `queryClient.test.ts` (outcome "paused" before, "network error" after); e2e `suppliers.e2e.ts` offline save → alert, entries kept, Cancel, focus back, **0 suppliers created after reconnect** | `fix5-offline-save-explained.png`; GRN editor offline save: alert, entries kept, server value unchanged after reconnect |
| 6 | Low | A page past the end (stale link, shrunk result) showed "2451–43 of 43 suppliers · Page 99 of 2"; Previous went to page 98 (also empty) | Open `/procurement/suppliers?page=99` | `Pagination` computed the range without checking the page count | Shared `Pagination`: "No suppliers on this page (43 in total)"; Previous goes to the last real page | Vitest `Pagination.test.tsx` (failed before) | `fix6-pagination-beyond-last-page.png`; Previous → page 2 of 2 |
| 7 | Low (docs) | Status board in `TASK_EXECUTION_LOG.md` still showed D-02 and D-03 as NOT STARTED although both entries are COMPLETE | Read the board | Board not updated when the entries were finished | Board rows set to COMPLETE (entries unchanged) | — | — |

(#3 was triaged as not a defect, see §5.)

## 5. Observations (not defects / not changed)

- **"Approved by User 26965c61"** on the PO order card while the history shows the approver's name: documented design (`personLabel`: "no name lookup API yet"); the PO payload carries no `approvedByName`. Candidate UX improvement for a later task.
- API logs client-aborted GETs as 5xx (`context canceled` → 503 for session lookup, 500 for catalogue): 251 lines this run, all GET, all `context canceled` (browser navigation / StrictMode double fetch), no failed write. Already recorded in earlier evidence; mapping to 499 would remove the noise (hardening, H-04).
- Request bodies over the per-module limit get 400 "must be one JSON object" (catalogue/suppliers/pricing/purchasing do not mention the size; receiving does). A ~3 MB body is cut off by the server (connection reset). Safe; message wording only.
- JSON API responses carry no `X-Content-Type-Options`/`X-Frame-Options` (attachment content responses do, with a sandbox CSP). Hardening for H-04.
- List page changes use history `replace`, so Back leaves the list rather than stepping through pages. Consistent design.
- On a 320 px PO page the "Goods receipts" card header squeezes its description into a narrow column next to the button. Readable, no overflow; cosmetic.
- Every list request runs twice on first load in dev (React StrictMode), as documented in B-03.

## 6. Test totals

| Suite | Result |
| --- | --- |
| Playwright full suite, desktop + Pixel 7 (before fixes) | 71 passed, 5 skipped, 0 failed |
| Playwright full suite (after fixes, +2 tests) | **73 passed, 5 skipped, 0 failed** (6.1 min) — `playwright-full-suite.txt` |
| Skipped by design | `shell` desktop-sidebar test on mobile, mobile-drawer test on desktop; `catalogue.e2e.ts` ×2 and `attachments` signed-link test run once (desktop, API-level) |
| Exploratory crawl (after fixes) | 300 page loads, 0 flagged |
| Exploratory API probes | 74-operation deny sweep (all OK), input robustness (no 5xx), auth, concurrency, pricing, audit — all as expected |
| Vitest | 23 files / **185 tests** passed (178 before + 7 new) |
| Go unit + PostgreSQL integration (`REQUIRE_INTEGRATION_TESTS=1`, scratch DBs on the throwaway server) | 18 packages ok; **394 tests (204 top-level) passed, 0 failed, 0 skipped** |
| `make ci` (gofmt, go vet, go mod tidy, swagger-check, ESLint, Prettier, tsc, Go integration tests, Vitest, Go + Vite builds) | **exit 0** |
| `git diff --check` | clean |
| Not available / not run | `golangci-lint` not installed (Makefile skips it; `go vet` ran). No automated axe/WCAG scanner installed — a11y checked by the crawl rules and keyboard tests. No real-device mobile testing (Pixel 7 emulation + 320 px viewport). |

## 7. Commands executed (secrets omitted)

```
bash <scratch>/stack/setup.sh            # throwaway postgres :5460, build + migrate + seed, users, PROPOSED grants, fixture, API :8099, Vite :5181
<scratch>/stack/e2e.sh                   # npx playwright test with E2E_* from the 0600 secrets file (full suite, before and after)
<scratch>/stack/e2e.sh e2e/purchase-orders.e2e.ts | e2e/suppliers.e2e.ts   # targeted red/green runs
python3 <scratch>/stack/p1.py and inline probes   # API sweeps
node <scratch>/stack/x/{crawl,explore2..11,evidence}.mjs                   # exploratory browser scripts
npx vitest run <files>  (with fixes stashed, then restored, for red/green)
DATABASE_URL=<throwaway> TEST_DATABASE_URL=<throwaway> make ci
TEST_DATABASE_URL=<throwaway> REQUIRE_INTEGRATION_TESTS=1 go test -count=1 -json ./...
git diff --check
```

## 8. Remaining risks, business blockers, pending approvals (unchanged by this audit)

- Permission matrix and approval thresholds pending (RB-5): all grants used here were PROPOSED on the throwaway DB.
- No brand can be activated through the API (warranty terms RB-2, identifier rules OQ-4), so no real PO can be submitted and no receipt can be made in a real database; the approval/receipt flows were verified only with the SQL fixture.
- PO number format, variance tolerance, delivery location (OQ-22…OQ-24); GRN document types, over-receipt policy, identifier rules (OQ-25…OQ-27); price types and tax (OQ-15, F-10); attachment types/size limit (proposal).
- A-01 per-brand activation evidence pending; C-05 (verified catalogue seed) not started.
- Session cookie `Secure` is off only because `APP_ENV=development`; staging/production refuse to start without it.

## 9. Changed files

UI source: `ui/src/components/ui/{Dialog,Pagination}.tsx`, `ui/src/app/queryClient.ts`,
`ui/src/features/{purchasing/PurchaseOrderPages,suppliers/SupplierPages,receiving/GoodsReceiptPages}.tsx`.
Tests: `ui/src/components/ui/{Dialog,Pagination}.test.tsx`, `ui/src/app/queryClient.test.ts` (new),
`ui/src/features/{purchasing/purchasing,suppliers/suppliers,receiving/receiving}.test.tsx`,
`ui/e2e/{purchase-orders,suppliers}.e2e.ts`.
Docs: `docs/TASK_EXECUTION_LOG.md` (status board rows D-02/D-03, this audit's entry), `docs/evidence/REGRESSION-AUDIT-2026-10-26/`.
No backend, migration, OpenAPI, seed or business-rule change.

## 10. Clean-up

Throwaway container `maa-audit-pg` (and its anonymous volume), API `:8099`, Vite `:5181`, the scratch attachment directory,
built binaries and the secrets file were removed after the run. The shared dev stack (`:5173`, `:8081`, `:5436`) was left running and untouched.

## 11. Re-run (user-requested, 2026-10-26)

Fresh throwaway stack (new container, new random passwords, same isolation; dev stack untouched), same working tree.

| Check | Result |
| --- | --- |
| Playwright full suite, desktop + Pixel 7 | **73 passed, 5 skipped (by design), 0 failed** (5.5 min) — `playwright-full-suite-rerun.txt` |
| Crawl 20 routes × 5 accounts × 3 viewports | **300 loads, 0 flagged**, no console/page errors, no 5xx — `crawl-pages-roles-viewports-rerun.txt` |
| Dialog focus return (Add supplier, Add brand, New purchase order; Escape and Cancel) | PASS |
| Offline save → message, entries kept, Cancel, nothing created after reconnect | PASS (0 created) |
| API 5xx | 245, all GET, all client-cancelled (`context canceled`); no failed write |

No new defects.
