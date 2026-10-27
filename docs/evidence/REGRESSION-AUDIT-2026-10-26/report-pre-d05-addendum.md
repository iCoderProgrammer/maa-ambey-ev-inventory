# Regression audit addendum — before D-05 (2026-10-26)

User-requested browser regression audit of everything implemented through D-04, including the six uncommitted
regression-audit fixes. Testing only: no product code changed, no D-05 work. The earlier report (`report.md`) is
unchanged; this file and the `pre-d05-*` files are the new evidence.

## Scope and commit range

| Item | Value |
| --- | --- |
| Branch / HEAD | `dev` at `ba7ca84`; `origin/dev` = `4a5a9c0`; local ahead 3 (`e962ee6` D-03, `25a5578` D-04 backend, `ba7ca84` D-04 frontend), behind 0 |
| Working tree under test | HEAD + the uncommitted audit fixes: `ui/src/components/ui/{Dialog,Pagination}.tsx`, `ui/src/app/queryClient.ts`, `ui/src/features/{purchasing/PurchaseOrderPages,suppliers/SupplierPages,receiving/GoodsReceiptPages}.tsx`, their tests (`Dialog.test.tsx`, new `Pagination.test.tsx`, new `queryClient.test.ts`, `purchasing/suppliers/receiving .test.tsx`), `ui/e2e/{purchase-orders,suppliers}.e2e.ts`, `docs/TASK_EXECUTION_LOG.md` (status rows + audit entries), this evidence folder |
| Database | throwaway `postgres:18-alpine` container, `127.0.0.1:5460`, migrated to **13** + seeds; random password never printed |
| API | built from the working tree, `:8099`, `APP_ENV=development`, `RBAC_ALLOW_PROPOSED_GRANTS=true`, `ATTACHMENT_LINK_TTL=30s`, raised rate/login limits; `/api/v1/ready` → `ready` (database up, migrations up) before testing |
| UI | Vite from the working tree, `:5181`, proxy → `:8099` |
| Shared dev stack | `:5173` / `:8081` / `:5436` — not contacted, not migrated, not restarted (its API is not-ready: build expects 13, DB at 12) |

### Test personas (throwaway DB only; PROPOSED grants, not production evidence)

| Persona | Account | Role | Grants used |
| --- | --- | --- | --- |
| Owner | `e2e.catalogue` | owner | catalogue, prices, cost prices, suppliers, POs, GRNs, attachments, `vehicle_units.view`, `vehicle_units.cost.view` |
| Inventory Manager | `e2e.attach` | inventory_manager | attachments, `vehicle_units.view`; location scope `AUDIT-L1` |
| Sales Executive | `e2e.priceapprover` (+ `e2e.attachviewer`) | sales_executive | catalogue/price view+approve, PO view+approve, GRN view/manage/over-receipt, attachments.view |
| Accountant | `e2e.catviewer` | accountant | catalogue, suppliers, POs, GRNs view; `vehicle_units.view` |
| Viewer | `e2e.roles` | viewer | `access.roles.view` only |
| No grants / disabled | `e2e.active`, `e2e.disabled` | — | none |

D-04 fixture (throwaway DB only, placeholder values): locations `AUDIT-L1`, `AUDIT-L2`; one approved PO with one line
(3 × ₹900.00) on the existing eligible-PO fixture; one goods receipt with three accepted captured units marked
`POSTED` by SQL and three `vehicle_units` (`MAE-VEH-000001` at L1, `…002` at L2, `…003` without location). This
stands in for receipt posting (D-07), which does not exist yet.

## Results

| Check | Result |
| --- | --- |
| Playwright full suite (desktop 1366×900 + Pixel 7) | **73 passed, 5 skipped, 0 failed** (6.2 min) — `pre-d05-playwright-full-suite.txt`. Skips are the documented ones: mobile-drawer test on desktop, desktop-sidebar test on mobile, `catalogue.e2e.ts` ×2 and the signed-link test run once on desktop |
| Route crawl: 29 routes (all implemented pages, details, denied/not-found paths, every planned module) × 6 personas × 3 viewports (1366×900, Pixel 7, 320×640) | **522 loads, 0 flagged** — one `<h1>`, no horizontal overflow, no unnamed controls, no unlabelled inputs, no duplicate IDs, no console/page errors, no 5xx, no unexpected 4xx — `pre-d05-crawl-routes-personas-viewports.txt` |
| Deny-by-default sweep, all 84 OpenAPI operations (incl. `/vehicle-units`, `/vehicle-units/{id}`) | 84/84: anonymous 401, no grant 403, missing CSRF 403 on mutations |
| D-04 API smoke (no UI screen exists; API level only) | **31/31 PASS** — persona visibility, cost shown only with `vehicle_units.cost.view` (no amount in the body otherwise), location scope (in-scope 200, out-of-scope detail 404 concealed, out-of-scope `locationId` 422), 401/403, search normalisation (`%`/`_` literal), invalid filters 422, non-UUID/missing 404, POST/PUT/PATCH/DELETE 405 |
| Live D-04 responses parsed with the frontend Zod contracts | 8/8 (temporary Vitest file, deleted after the run) |
| D-04 hook/schema tests | `vehicles.test.tsx` 21/21 |
| Stock-safety invariant around the suite (GRN drafts created and edited by the tests) | before → after: vehicle units 3 → 3, unit history 3 → 3, PO received quantity sum 0 → 0, POs partially/fully received 0 → 0, posted receipts 1 → 1, draft receipts 0 → 2, captured draft units 3 → 7, stock tables 0 |
| Behaviour (audit fixes re-verified) | dialog focus returns to the opener after Escape/Cancel (Add supplier, Add brand, New PO); offline save → "Cannot reach the server. Nothing was saved…", entries kept, Cancel works, nothing created after reconnect (supplier dialog and GRN editor); double-submit sends one POST (brand, price list); focus trap; page past the end → "No suppliers on this page (36 in total)", Previous → last page; special-character search; 8 create dialogs at 320 px without overflow; 11 MB upload refused client-side; offline upload offers Retry |
| API 5xx | 243, all `GET` with `context canceled` (requests aborted by navigation/StrictMode); the browser saw no 5xx; no failed write |
| Vitest (full) | 24 files / 206 tests passed |
| Go unit + PostgreSQL integration (`REQUIRE_INTEGRATION_TESTS=1`) | 19 packages; 410 tests (220 top-level) passed, 0 failed, 0 skipped |
| `make ci` (throwaway DB) | exit 0 (gofmt, go vet, go mod tidy, swagger-check, ESLint, Prettier, tsc, Go tests, Vitest, Go + Vite builds) |
| `git diff --check` | clean |
| Not available | `golangci-lint` (not installed; Makefile skips it). No automated axe scanner. |

### Not applicable / not tested

- **D-04 browser UI:** NOT APPLICABLE — D-04 has no screen (types and hooks only; the list/detail screens are D-08).
  D-04 is covered by the API smoke tests, the live-response contract check and the hook/schema tests above.
- **Receipt posting, PDI, stock movements:** NOT APPLICABLE (D-06/D-07/E-01 not implemented).

## Findings

**No new defects.** Every check above passed. The six earlier fixes behave as recorded in `report.md`. The observations
listed there (approver shown as a user ID on the PO card, client-cancelled requests logged as 5xx, missing nosniff/frame
headers on JSON, size-limit wording) are unchanged and remain non-blocking.

## Cleanup

The throwaway container (with its volume), the test API and Vite processes, the built binaries, the attachment folder,
the secrets file and the recorded live responses were removed after the run (verified: ports 5460/8099/5181 closed,
no `maa-audit-pg` container).

## Recommendation

D-05 can start. Pending decisions that affect D-05 but do not block starting it: OQ-4 (identifier rules per brand),
OQ-26 (which separators normalisation ignores), OQ-30 (battery packs), INV-003 (motor serial uniqueness after brand
validation). The uncommitted audit fixes are still uncommitted and need your decision.
