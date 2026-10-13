# Requirements Baseline — Client-Confirmed Decisions

Client-confirmed facts for the Maa Ambey Enterprises Multi-Brand EV Inventory Management System,
recorded 2026-10-12. These add to the approved documents in `documentation/`
(PRD, System Architecture, Business Rules, Design Specification, Implementation Tasks). Where
they differ, the approved Business Rules still apply until the client signs off a change.
Every "Pending" item needs a client (or accountant) decision before the related feature is built.

Status legend: **Confirmed** = client decision · **Pending** = waiting on client data or approval ·
**Implemented** = present in this repository.

---

## 1. Brands — multi-brand is mandatory

| Item | Status |
| --- | --- |
| Initial brands: **Lectrix EV**, **Ceeon India**, **Ujagar Electric** | Confirmed |
| More brands can be added later with no code or schema change | Confirmed · Implemented (brands are rows in `brands`) |
| Idempotent seed for the three brands (`seeds/0001_initial_brands.sql`, `make db-seed`) | Implemented |
| Authorisation evidence for each brand: supplier/manufacturer legal identity, authorisation reference, approval owner, verified date (BR-002) | **Pending** |

Seeded brands start as `lifecycle_status = PENDING_VERIFICATION` and
`authorisation_status = UNVERIFIED`. The client named them as operating brands, but approved
rule BR-002 says a brand becomes `ACTIVE` for purchase and sales only after authorisation evidence
is recorded. This seed does not invent that evidence. Brands are activated through the audited
catalogue workflow (backlog C-02) once the evidence is entered.

No brand name appears in application logic. The only place the names exist is the seed file and its
idempotency test.

## 2. Legal entity, GST and billing

| Item | Status |
| --- | --- |
| Legal business name, GST details and showroom address exist | Confirmed |
| Actual values (legal name, GSTIN, registered/showroom address, contact details) | **Pending** — enter from client source documents; none are stored or invented in code |
| Client files GST returns and income-tax returns | Confirmed |
| Configurable business profile and invoice settings (entity, GSTIN, address, invoice series per financial year) | Planned, backlog F-10 |
| Invoice format, HSN, tax rates, place-of-supply rules, numbering | **Pending** — must be verified by the client's accountant or tax adviser (DOC-002…DOC-004). No tax rates or invoice numbers are hardcoded. |
| GST-return filing, income-tax-return filing, tax-portal / e-invoice / e-way bill integration | **Out of scope.** Not part of setup or the MVP backlog. |

## 3. Brand-wise catalogue

The client approved collecting and maintaining **verified** data per brand:

| Data | Notes |
| --- | --- |
| Models, variants, colours | Model belongs to one brand; variant to one model (BR-003) |
| Battery and motor details | Where applicable; battery may be swappable (INV-004) |
| Warranty terms | Versioned per brand/model/variant (BR-005) |
| Approved prices | Effective-dated, approver recorded, never overwritten (BR-005, C-04) |
| Vehicle identification rules | Chassis/frame (mandatory, globally unique after normalisation), motor, battery and charger serial rules per brand/model (INV-002…INV-004) |
| Source and verification status for every record | `UNVERIFIED` / `SUPPLIER_CONFIRMED` / `CLIENT_APPROVED` (BR-004) |

**Status:** No catalogue data has been supplied. Nothing is seeded or invented. The schema exists
since C-01 (migration 000007, ADR-011); data comes with C-05, from client- or supplier-verified sources only.

## 4. Opening-stock import

| Item | Status |
| --- | --- |
| Import the showroom's real current stock | Confirmed |
| Blank template: `templates/opening-stock-template.csv` (header row only) | Implemented (draft columns) |
| Import workflow | Planned, backlog G-06 (preview) and G-07 (commit) |

### Draft template columns

One row per physical vehicle. Scooter quantity is the number of rows, because each scooter is a
serialised unit (D-02, INV-001).

| Column | Required | Notes |
| --- | --- | --- |
| `brand_slug` | Yes | Must match an existing brand (e.g. `lectrix-ev`) |
| `model_code` | Yes | Must match a catalogue model of that brand |
| `variant_code` | Yes, if the model has variants | Must belong to the model |
| `colour` | Pending | Depends on the catalogue colour model |
| `chassis_number` | Yes | Normalised (trim, upper-case, permitted separators); duplicates are fatal (IMP-002) |
| `motor_serial` | Pending | Per brand rule (INV-003) |
| `battery_serial` | Where applicable | Per brand/model rule (INV-004) |
| `charger_serial` | Pending | Per brand/model rule |
| `stock_status` | Yes | One of the INV-005 statuses; allowed opening statuses are pending (e.g. `AVAILABLE`, `DISPLAY`, `QUALITY_HOLD`) |
| `location_code` | Yes | Initial seed: one showroom |
| `opening_balance_date` | Yes | Starting date for stock ageing (LED-008) |
| `remarks` | No | Free text |

Column names and required flags are a **draft** for client review. Supplier and receipt context is
optional: opening stock is flagged as an opening balance, and no PO/GRN history is invented
(IMP-005).

### The future workflow must provide

- Upload staged as a batch: batch ID, uploader, file name and hash (IMP-003)
- Preview with row-level validation: required fields, formats, brand → model → variant mapping, serials
- Duplicate detection within the file and against existing units (chassis/motor/battery)
- Error report per row; nothing is committed while fatal errors remain
- Safe commit in one transaction that creates units plus opening-balance stock movements and an audit event
- Idempotency: re-submitting a committed batch creates no duplicates
- No fabricated rows, ever

## 5. User roles

| Role | Status |
| --- | --- |
| Owner | Confirmed |
| Inventory Manager | Confirmed |
| Sales Executive | Confirmed |
| Accountant | Confirmed |
| Viewer | Confirmed |
| Granular permission matrix | **Pending client approval** |

Authorisation is enforced by the backend: deny by default, with route guards plus object-level
and brand/location-scope checks (SEC-001, SEC-002). Hiding a button in the UI is not
authorisation. Built in B-07 (ADR-007): the five roles exist as data with **no permission granted**;
each cell below becomes a grant only when the owner approves it (recorded with approver and reference).

### Permission decisions awaiting approval

Each row needs a yes/no per role from the client. Nothing here is decided.

| Action | Owner | Inventory Mgr | Sales Exec | Accountant | Viewer |
| --- | --- | --- | --- | --- | --- |
| Manage brands, catalogue and brand activation | Pending | Pending | Pending | Pending | Pending |
| Approve or edit prices and discount limits | Pending | Pending | Pending | Pending | Pending |
| View cost price and stock valuation | Pending | Pending | Pending | Pending | Pending |
| Create / approve purchase orders | Pending | Pending | Pending | Pending | Pending |
| Post goods receipts and PDI outcome | Pending | Pending | Pending | Pending | Pending |
| Request / approve stock adjustments (LED-006: no self-approval) | Pending | Pending | Pending | Pending | Pending |
| Create quotations and reservations | Pending | Pending | Pending | Pending | Pending |
| Approve discounts above role limit | Pending | Pending | Pending | Pending | Pending |
| Finalise sales, record payments | Pending | Pending | Pending | Pending | Pending |
| Cancel sales, refunds and reversals | Pending | Pending | Pending | Pending | Pending |
| Mark delivered / override checklist | Pending | Pending | Pending | Pending | Pending |
| Business profile, invoice series, tax settings | Pending | Pending | Pending | Pending | Pending |
| Export reports and customer data (audited) | Pending | Pending | Pending | Pending | Pending |
| Manage users and roles | Pending | Pending | Pending | Pending | Pending |
| View audit log | Pending | Pending | Pending | Pending | Pending |
| Brand scope: all brands, or restricted per user? | Pending | Pending | Pending | Pending | Pending |

## 6. Vehicle delivery checklist

| Category | Status | Mandatory or conditional |
| --- | --- | --- |
| Keys | Confirmed | **Pending** |
| Charger | Confirmed | **Pending** (e.g. may depend on battery scheme / BaaS) |
| Manuals | Confirmed | **Pending** |
| Warranty documents | Confirmed | **Pending** |
| Payment | Confirmed | **Pending** (full payment vs. finance-approved) |
| Insurance | Confirmed | **Pending** |
| Registration | Confirmed | **Pending** (status vs. completed RTO registration) |

Checklist items will be configuration (per brand/model/transaction, SAL-009), not code constants.
A unit moves to `DELIVERED` only when the required items are satisfied, or through an authorised
override with a reason (SAL-010). The checklist is not built yet (backlog F-09).

## 7. Open decisions summary

| # | Decision | Owner | Blocks |
| --- | --- | --- | --- |
| RB-1 | Authorisation evidence per brand, and who approves activation | Client | C-02, C-05 |
| RB-2 | Verified catalogue data per brand (models, variants, colours, specs, warranty, prices, identifier rules) | Client / suppliers | C-05 |
| RB-3 | Legal entity, GSTIN, address, invoice series and format | Client + accountant | F-10 |
| RB-4 | Opening-stock template columns and allowed opening statuses | Client | G-06 |
| RB-5 | Permission matrix (§5) and whether brand scope is per user | Client | B-07 |
| RB-6 | Mandatory vs. conditional rule for each delivery-checklist item | Client | F-09 |
| RB-7 | Whether the three brands should be `ACTIVE` at launch once evidence exists, or stay pending until each supplier confirms | Client | C-02 |
