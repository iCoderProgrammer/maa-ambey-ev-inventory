> Source: Version 1.2 approved project document; converted from DOCX for searchability in this prompt pack.

# UX / UI Design Specification

Product: Maa Ambey Enterprises — Inventory Management System | Version: 1.2 draft | Updated: 12 October 2026 | UX baseline updated to reflect client-confirmed brands, roles and delivery checklist

## 1. Design objective

Create a dependable, low-friction staff console for a vehicle showroom. Staff need to scan identifiers, search units quickly, receive vehicles without missing serial data and complete sales without accidentally selling unavailable stock. The product should feel modern and professional, but prioritise clarity, density and correct status cues over decorative marketing visuals.

## 2. Visual direction

Brand expression: clean, confident, EV/engineering-oriented; use the Maa Ambey name and a small electric-mobility accent rather than copying a manufacturer's consumer website.

Primary palette: deep ink/navy #101828, electric blue #175CD3, bright teal #12B8A6, white #FFFFFF, canvas #F6F8FB, border #E4E7EC, text secondary #667085.

Status colours: available/success green, warning amber, blocked/error red, neutral grey for draft/archived. Always pair colour with text/icon to avoid colour-only meaning.

Typography: Inter or system sans-serif; 14–16 px body, 12–13 px helper/dense metadata, 24–32 px page title and 18–20 px section title.

Geometry: 8 px spacing base, 8–12 px border radius, restrained shadows, clear separators. Avoid oversized dashboard charts that push operational tables below the fold.

Data density: compact but readable tables, sticky header for long lists, persistent filters, visible record count, server pagination and clear empty/loading/error states.

## 3. Navigation and shell

Desktop left nav, collapsible:

1. Dashboard

2. Inventory

Vehicle Units

Spare Parts & Accessories

Stock Movements

Stock Counts / Adjustments

3. Procurement

Suppliers

Purchase Orders

Goods Receipts / PDI

4. Sales

Customers

Quotations

Reservations

Sales Orders / Delivery

5. Reports

6. Admin

Catalogue / Brands / Models

Users & Roles

Store Settings / Invoice Series

Audit Log

Top bar: global search, quick-create menu, notifications/alerts, current location, user menu. Breadcrumbs should show hierarchy, e.g. Inventory / Vehicle Units / MAE-VEH-00042. On small screens, use a drawer nav; dense tables may horizontally scroll, but critical actions remain reachable.

## 4. Core screens and required states

### 4.1 Login

Fields: email/username, password, show/hide password, submit. Include invalid credentials, disabled account, rate-limit and network error states. Do not persist session tokens in localStorage. Use accessible labels, keyboard focus and clear error messages without indicating whether a specific username exists.

### 4.2 Dashboard

KPI cards: Saleable scooters, Reserved, In PDI/Hold, In Transit, Units sold today/this month, Stock ageing alert, Low-stock parts. Show “updated at” and location filter. Visualisations: sales trend and stock by status/brand only if they help a decision. Quick actions: Receive Stock, Add Vehicle, New Reservation, New Sale, Stock Count. Clicking a KPI opens the pre-filtered underlying list.

### 4.3 Vehicle units list

Primary table columns: internal ID, model/variant, colour, chassis/frame (masked if role requires), status, location, received date, stock age, reservation/sale reference, action. Filters: brand, model, variant, colour, status, location, supplier, received-date range, ageing band. Search: internal ID, chassis/frame, motor, battery, PO/GRN. Include empty, no-results, loading, error, permission-denied and stale-data states.

### 4.4 Vehicle unit details

Header: model/variant and status pill; internal ID and QR; primary actions based on permissions/status. Sections/tabs:

Overview: identifiers, colour, location, supplier, cost visibility by role.

Battery/charger: serials, scheme/ownership and component history if enabled.

Timeline: stock movement and status history.

Procurement/PDI: PO, GRN, inspection checklist, defects and photos.

Reservation/sale/delivery: links to related records.

Documents: private attachments with upload/download permissions.

### 4.5 Receive stock / goods receipt

Guided stepper: select PO -> confirm delivered lines -> capture unit identifiers -> upload supplier docs -> PDI triage -> review and post. Scanner-friendly inputs should autofocus appropriately and indicate duplicate identifiers immediately. Support partial receipt. Clearly separate Save draft from Post receipt, because posting changes stock. Error state must preserve safe draft data while never partially posting a receipt.

### 4.6 Catalogue management

Provide a first-class Brands list/editor alongside models, variants, colours and approved specs. Initialise brand records for Lectrix EV, Ceeon India and Ujagar Electric, and make adding future brands a normal catalogue action. Do not assume that a configured brand is automatically authorised for sale: show separate lifecycle, catalogue-verification, supplier/authorisation evidence and purchase/sales activation states. Restrict model/variant choices to the selected brand. Show source status (Unverified, Supplier confirmed, Client approved) and verified date; keep brand/variant price history visible to authorised roles. Activation for purchase/sales requires configured evidence and an audited action. Verify the legal supplier/brand relationship for Ujagar Electric / Saffron EV Group before recording it as an alias.

### 4.7 Purchase orders

PO list has number, supplier, expected date, total, status, received/remaining count and creator. PO detail has line quantities/costs, approvals, attachment history, receipt history and allowed actions. Clearly display over-receipt, price variance and partial delivery warnings.

### 4.8 Reservation / quotation

Select customer -> choose model/variant and available unit -> show approved price snapshot -> apply discount (permission/approval if needed) -> optional advance/payment receipt -> choose reservation expiry -> confirmation summary. Prevent users from choosing held/reserved/sold units. Revalidate availability server-side at save and sale finalisation.

### 4.9 Sale and delivery

Sale workspace should show customer, exact vehicle/chassis, price lines, discount approvals, payment collected/due and document status. Delivery checklist includes the client-confirmed categories keys, charger, manuals, warranty documents, payment, insurance and registration, plus PDI/customer acknowledgement and finance/RTO fields when approved. Each item must display whether it is required or conditional for this model/transaction. A persistent “Cannot deliver yet” panel lists missing required steps. Do not decide mandatory-versus-conditional policy without owner sign-off. Finalise button is disabled in UI until client-side checks pass, but API re-check remains mandatory.

### 4.10 Reports

Reports: inventory by status/model/brand/location, stock ageing, stock valuation, PO pending/partial receipts, PDI holds, reservation ageing, daily/monthly sales, receipts vs outstanding, movement audit, low-stock parts. Each report needs filters, definitions, generated timestamp and CSV/XLSX export. Do not combine sales value and money collected in one KPI.

### 4.11 Admin / audit

Role matrix uses the client-confirmed role names Owner, Inventory Manager, Sales Executive, Accountant and Viewer. Show permission groups and require confirmation for privilege changes; the exact role-action matrix remains pending owner sign-off. Invoice settings clearly indicate accountant-review status and effective date. Audit screen filters actor, action, entity, date and reference. Do not allow normal users to modify audit events.

## 5. Main interactions and microcopy

Primary actions use verbs: “Receive stock”, “Post receipt”, “Reserve vehicle”, “Finalise sale”, “Approve adjustment”.

Irreversible confirmation states the effect: “Posting this receipt will add 3 accepted vehicle units to saleable/inspection stock.”

Duplicate serial error: “This chassis/frame number is already registered. Open the existing unit or correct the entered number.”

Stale reservation: “This vehicle was changed by another user. Refresh availability before continuing.”

No stock: “No available units match these filters. View reserved/held units or adjust filters.”

Price/spec trust: label unconfirmed catalogue details, never present placeholders as accurate customer-facing values.

## 6. Responsive and accessibility requirements

Target WCAG 2.2 AA where practical: semantic headings, keyboard navigation, visible focus, sufficient contrast, labels, error association and no colour-only status.

Desktop first for inventory operations; usable tablet layout for receiving and sales. Mobile support is responsive, not a separate native app.

Tables switch to horizontal scroll or card summaries on narrow screens; do not hide unique identifiers or critical status.

Buttons and inputs should have minimum comfortable touch targets on tablet/mobile.

Support browser zoom to 200% without overlapping critical controls.

Provide accessible text for QR/barcode actions and scanner fallback via manual entry.

## 7. Loading, empty, error and offline behaviour

Loading: skeleton rows or progress only where data is genuinely loading; avoid indefinite spinners without explanation.

Empty state: explain what belongs here and offer an authorised next action.

Error state: show request/reference ID and retry action where safe; never expose stack traces or tokens.

Network loss: display connectivity warning and prevent finalising stock-changing actions. MVP is online-first and must not imply offline saves succeeded.

Partial form: support draft saves for PO/receipt/quotation where safe, but never create partial posted ledger events.

## 8. Design token starter set

:root {
  --color-primary: #175CD3;
  --color-accent: #12B8A6;
  --color-ink: #101828;
  --color-canvas: #F6F8FB;
  --color-surface: #FFFFFF;
  --color-border: #E4E7EC;
  --color-muted: #667085;
  --radius-card: 12px;
  --radius-control: 8px;
  --space-unit: 4px;
}

The tokens are a starting proposal and should be checked against the client's existing public website brand assets before finalising.

## 9. Component inventory

Application shell, sidebar, topbar, breadcrumbs, permission-aware action menu.

KPI card, status badge, activity timeline, alert banner.

Searchable/filterable data table, pagination, bulk selection where safe.

Form field wrapper, combobox, date range, currency field, serial scanner/manual entry.

Modal/drawer, confirmation dialog, stepper, tab group.

Attachment uploader, image preview, document row.

Empty/loading/error/skeleton components.

Price summary, approval strip, delivery checklist, audit table.

## 10. Design acceptance checklist

Staff can reach Vehicle Units, Receive Stock and New Sale from the shell in one action.

Search by chassis/frame and battery/motor serial is easy to find and works from the main inventory screen.

A user can distinguish physical, saleable, reserved and held inventory at a glance.

Statuses have labels/icons as well as colours.

Dangerous actions explain their effect and require confirmation.

Keyboard-only operation covers login, table search/filtering, receiving and finalising a sale.

Responsive layouts keep actions visible; validation errors preserve drafts.

# 11. Client-confirmed UX updates

Use the following as explicit screen-level requirements when the corresponding feature is implemented.

## Brand catalogue and settings

Show Lectrix EV, Ceeon India and Ujagar Electric as the initial configured brand records. Provide an Add Brand action for future brands.

Show distinct states for configured brand, catalogue verification, supplier/authorisation evidence and operational purchasing/sales readiness. Do not show an unverified brand as fully approved merely because it appears in the initial roster.

When Ujagar Electric / Saffron EV Group is encountered in source files, show a verification task rather than silently merging identities.

## Users and permissions

The initial role names are Owner, Inventory Manager, Sales Executive, Accountant and Viewer. The role editor must clearly mark detailed action permissions and approval thresholds as requiring Owner sign-off until finalised.

A disabled/unauthorised action should have clear explanation where appropriate; backend authorization remains authoritative.

## Legal settings, stock import and delivery

Company settings provide entry points for legal business name, GST details, showroom/billing address and invoice settings. Actual values must come from verified client documents; display accountant-review status.

Opening stock import must expose a downloadable template, upload/preview, row-level errors, duplicate identifier warnings, mapping resolution and explicit commit confirmation in its later implementation.

Delivery screens show keys, charger, manuals, warranty documents, payment, insurance and registration. Distinguish required from conditional checklist items and show which approval governs an override.
