# Complete Project Documentation — Version 1.2

This file consolidates the six v1.2 project documents supplied with this prompt pack. The individual files remain the authoritative separated documents.


---

<!-- Source file: 00_Research_Brief_and_Decisions.md -->

> Source: Version 1.2 approved project document; converted from DOCX for searchability in this prompt pack.

# Research Brief & Decision Register

Project: Maa Ambey Enterprises — EV Showroom Inventory Management System | Updated: 12 October 2026 | Status: Updated after client answers; source data and per-brand activation evidence remain pending | Primary use case: Internal showroom operations, stock traceability, vehicle sales and reporting

## 1. Executive summary

Build a web-based inventory and showroom operations system for electric two-wheelers, with support for serialised scooters, model/variant catalogue, supplier purchases, goods receipt, stock movements, reservations, sales delivery and management reports. Unlike a generic retail inventory tool, every saleable scooter must be tracked as an individual physical unit using its chassis/VIN/frame identifier and, where available, motor and battery serials. This avoids treating two otherwise identical scooters as interchangeable stock.

Client-confirmed requirement: the architecture must support multiple EV brands from day one. The client has confirmed that Lectrix EV, Ceeon India and Ujagar Electric are the initial brands it intends to use operationally, and that additional brands must be addable later without an architectural redesign. Create configurable master records for all three brands. This roster confirms client intent, not by itself the evidence needed to enable transactions: each brand still requires verified supplier/manufacturer identity, relevant authorisation, current catalogue, approved pricing and warranty/serial rules before purchasing and sales are enabled. The public Maa Ambey Enterprises website currently identifies the business as an authorised Lectrix EV dealership and lists NDuro, LXS 3.0, LXS 2.0, ZYRO and SX25, while noting that some specifications and prices need confirmation. Public positioning does not establish current selling rights for Ceeon India or Ujagar Electric [1]. The exact relationship, if any, between Ujagar Electric and Saffron EV Group also requires documentary confirmation.

Lectrix EV's official website currently presents NDuro, ZYRO, LXS 3.0, LXS 2.0 and SX25 and has a dealer locator and business/partner information [2][3]. Ceeon India's official site advertises electric scooters and lists Mareno+, Mysta, Faxion, Trueno+ and Pasito, with varying specifications and prices that should be treated as catalogue reference data only—not showroom stock or a price feed [4]. Public research did not establish a reliable current Ujagar Electric catalogue or confirm the exact supplier/manufacturer relationship to Saffron EV Group. The client nevertheless intends to include Ujagar Electric in the initial roster; collect current supplier documentation, an approved catalogue and authorisation evidence before enabling transactional use.

## 2. What public research confirms

|Research area|Evidence found|Product implication|
|---|---|---|
|Maa Ambey public site|Describes Maa Ambey Enterprises as an authorised Lectrix EV dealership and mentions sales, service, test rides, finance, registration and after-sales support. Public contact/address fields look like placeholders. [1]|Use Maa Ambey identity in the system, but collect verified legal entity, GSTIN, address, phone, email and store hours before production.|
|Lectrix EV|Official catalogue currently links to NDuro, ZYRO, LXS 3.0, LXS 2.0 and SX25; site content shows product/specification variation by model and commercial terms such as battery subscription on selected products. [2][3]|Catalogue needs configurable model/variant attributes, pricing history, battery/ownership scheme and per-unit serials. Do not assume all variants or BaaS options are available to this showroom.|
|Ceeon India|Official home page advertises two-wheelers and other EV types; featured scooters include Mareno+, Mysta, Faxion, Trueno+ and Pasito. Public pages show different battery types, speeds, ranges, and warranty claims. [4]|The confirmed multi-brand architecture must support Ceeon through brand-specific master data. The MVP inventory category remains electric two-wheelers unless scope is deliberately expanded. Treat public specifications/prices as reference only; obtain current supplier-confirmed models, variants, warranty and commercial terms before activation.|
|Ujagar Electric (relationship to Saffron EV Group to verify)|Public search did not surface a sufficiently reliable official model/variant catalogue for configuration.|Client confirms intent to use Ujagar Electric at launch. The current official catalogue and supplier/manufacturer relationship still need verification; require legal supplier identity, current catalogue, tax details, approved price list, warranty terms and authorisation evidence before enabling purchasing/sales.|
|GST invoice fields|CBIC's invoice rules list supplier details/GSTIN, a financial-year-unique serial number, issue date, recipient fields as applicable, HSN, description, quantity, value, taxable value, rate and tax amount, among other applicable fields. [5]|Invoice templates must be configurable and reviewed by the client's accountant. Avoid hardcoding tax rates/threshold rules; keep an audit trail and use a compliant external tax integration if required.|
|E-way bill|The official NIC E-Way Bill portal describes it as an electronic document for movement of goods and explains portal/API options and applicability, including the general ₹50,000 movement threshold described on the portal. [6]|Do not treat this MVP as a government filing system. Store reference numbers and offer a later integration after a tax professional confirms applicability and eligibility.|

## 3. Current website review

Reviewed website: https://maa-ambey-enterprisesev.vercel.app/

Observed sections include Electric Scooters, Compare, BaaS, Why Lectrix EV, Our Showroom, Service, About and Contact. The product area currently lists five Lectrix models. Several product attributes/prices are displayed as unavailable or “price on request”, and the public page says specifications and prices are indicative and must be confirmed with the showroom. Address and contact fields include example/placeholder-looking values. The public site is useful for business positioning and initial navigation context; it is not authoritative for live on-hand inventory, cost, current dealer price, tax, supplier authorisation, or warranty obligations.

Recommendation: Keep the inventory app private (authenticated staff system) and independent from the public marketing site for the MVP. Consider connecting public enquiry/test-ride leads later through a documented API or controlled CSV import. Public product cards must never be used as the stock ledger.

## 4. Recommended scope

### MVP — include

Secure login, role-based permissions and audit log.

Brand, model, variant, colour and configurable specification masters.

Individual vehicle registry with chassis/VIN/frame number, motor serial, battery serial(s), charger/accessories, stock location and lifecycle status.

Purchase orders, supplier records, goods receipt, PDI acceptance/rejection and attachment storage.

Stock ledger and movement history; cycle count/stock adjustment with approval and reason.

Customer records, optional enquiry/test-ride record, quotation/reservation, sales order, payment receipt tracking, delivery checklist and invoice/receipt print/export (invoice format and tax reviewed by the client's accountant).

Search, filters, barcode/QR label support, low-stock alerts for spares/accessories, ageing stock and daily/period reports.

CSV/XLSX exports, backups, environment separation, monitoring and basic operational documentation.

### Phase 2 — only after MVP is stable

Service/workshop job cards, warranty claims and spare-parts consumption.

Finance/insurance/RTO status workflow beyond simple checklist fields.

WhatsApp/SMS/email notifications, supplier portal, multi-branch transfers, public website lead sync.

Accounting/GST/e-invoice/e-way bill integrations after vendor and compliance assessment.

Battery-as-a-Service/subscription contract tracking beyond a basic scheme/ownership field.

### Exclude initially

Direct manufacturer ERP integrations without documented APIs and supplier permission.

Automatic live prices/specifications scraped from brand websites.

Full accounting/general ledger, payroll, loan underwriting, insurance underwriting, RTO filing or government return filing.

Customer self-service portal and native mobile apps.

## 5. Key design decisions

|ID|Decision|Recommendation|Reason / dependency|
|---|---|---|---|
|D-01|Brand support|Confirmed P0 requirement: one shared application and catalogue must support multiple brands from day one; every brand has separate configuration and activation status.|The client confirmed multi-brand architecture. The public Maa Ambey site currently positions the showroom as authorised Lectrix dealer; therefore architecture support must be separated from per-brand commercial authorisation and catalogue verification.|
|D-02|Vehicle stock unit|Track each complete vehicle as a serialised asset; track spare parts/accessories as quantity-based or serialised items according to risk.|Chassis, motor and battery traceability is important for delivery, warranty and audit.|
|D-03|MVP hosting|Web app + API + managed PostgreSQL + private object storage.|Reliable central data and controlled access; avoid spreadsheet-only stock state.|
|D-04|MVP operating model|One showroom/location in first release; data model includes location IDs to avoid a costly redesign.|Fits lean launch while leaving a clean path to a second store/warehouse.|
|D-05|Invoicing|Create configurable printable documents; tax logic and legal format must be signed off by client's accountant before live use.|Legal applicability depends on entity, registration and current rules.|
|D-06|Source of truth|Stock ledger and individual unit records are canonical; dashboard figures are derived.|Avoid silent stock changes or duplicated counts.|
|D-07|Pricing|Store purchase cost, approved sale price, offers/discounts and effective dates separately, with history.|Website prices may be indicative and can change by variant/location/offer.|

## 6. Client discovery questions — resolve before build completion

1. Launch brand intent is confirmed for Lectrix EV, Ceeon India and Ujagar Electric, with future brands supported. For each of the three, provide the current supplier/manufacturer legal identity, dealership/supply authorisation, official model/variant catalogue, approved price list, warranty terms and serial-field rules. Keep a brand record configured while blocking purchases/sales until the required evidence is reviewed and approved.

2. The client confirms the legal business name, GST details and showroom address exist and that the business files GST and income tax. Capture their exact values from verified client documents, along with billing/ship-from address and state, financial-year invoice series, existing accounting software (if any), and accountant-approved invoice format. Do not add tax-return filing functionality to this MVP.

3. One showroom only, or showroom plus godown/service centre? Who can transfer stock or approve adjustments?

4. Are vehicles received fully assembled with battery/charger, or are these received/managed as separate serialised parts? Must battery/charger serial be mandatory?

5. The client agrees to collect verified model/variant, colour, battery/motor details, warranty, approved prices and vehicle-identification rules for each launch brand. The actual approved data files, active variants and current on-hand models still need to be supplied and reconciled.

6. Does any Lectrix vehicle use Battery-as-a-Service (BaaS) at this dealer? If yes, which fields/documents are required and who owns the battery on sale?

7. Do bookings take refundable advances? Define cancellation/refund policy and how payment receipts reconcile with bank/UPI/cash.

8. Should invoices be generated in this app or remain in existing accounting software? Which software, if any?

9. The delivery checklist categories are confirmed: keys, charger, manuals, warranty documents, payment, insurance and registration. Confirm which are mandatory versus conditional by model or transaction, and whether any additional customer acknowledgement, PDI, finance or RTO fields are required.

10. The client confirms an opening-stock import template is needed. The actual vehicle/part stock file, person responsible for preparing/importing it and authorised person who signs off the physical-count reconciliation still need to be agreed.

11. Is internet access reliable in the showroom? Is offline operation genuinely required, or is normal online operation acceptable?

12. Who owns maintenance and support after launch, and what are the expected backup/recovery and uptime targets?

## 7. Recommended delivery approach

1. Discovery and data sign-off (3–5 working days): validate legal details, brands, models, opening stock and invoice requirements.

2. UX/data/API foundation (1 week): design system, database, auth/RBAC, audit and reference masters.

3. Inventory intake (1–2 weeks): suppliers, purchase orders, receipts, vehicle serial registry, PDI, ledger and stock screens.

4. Sales and delivery (1–2 weeks): customer, quotation/reservation, sales order, payment tracking, delivery checklist and printable documents.

5. Reports, QA and pilot (1–2 weeks): reports, imports/exports, access testing, inventory reconciliation, user training and launch checklist.

Planning estimate: about 6–8 calendar weeks for a focused MVP with one experienced full-stack engineer plus part-time QA/product input; adjust based on team size, invoice requirements and integration scope. This is an indicative planning estimate, not a fixed commitment.

## 8. Risks and mitigations

Unverified dealership rights: do not activate a brand/model until client provides written supplier evidence.

Incorrect opening stock: bulk import and require physical count/sign-off, including duplicate serial detection.

Tax/invoice mistakes: accountant approval; configurable tax settings; do not hardcode statutory thresholds/rates.

Data leakage: strict RBAC, private attachments, secure sessions, audit logs and least privilege.

Untraceable stock changes: append-only stock movements, negative-stock prevention and approved adjustments.

Overbuilding: prioritise serialised stock and sales first; defer integrations and workshop management.

## 9. Sources

[1] Maa Ambey Enterprises public website, reviewed 10 Oct 2026: https://maa-ambey-enterprisesev.vercel.app/ [2] Lectrix EV official website: https://www.lectrixev.com/ [3] Lectrix EV dealer locator and business pages: https://www.lectrixev.com/dealer-locator and https://www.lectrixev.com/lectrix-for-businesses [4] Ceeon India official website and catalogue: https://ceeonindia.co.in/ [5] CBIC, GST Invoice Rules: https://cbic-gst.gov.in/gst-invoice-rules.html [6] NIC, GST E-Way Bill System: https://docs.ewaybillgst.gov.in/

Research note: Websites and published pages can change. Treat all catalogue claims as leads to verify against current manufacturer/dealer documents before production data entry.

# 10. Client-confirmed requirements update — 12 October 2026

The client has answered the six high-level discovery items. The decisions below supersede earlier wording that treated the launch brand roster and these workstreams as unanswered; detailed source data and approval details remain open where noted.

A-01 — Launch roster: Lectrix EV, Ceeon India and Ujagar Electric are the initial intended operational brands. The system must allow additional brands later. Create all three brand records; keep purchase/sales activation controlled by verified catalogue, supplier and authorisation evidence. Verify whether Ujagar Electric is legally connected to Saffron EV Group rather than assuming equivalence.

A-02 — Legal and billing: the client confirms its legal business name, GST details and showroom address exist and that it files GST and income tax. Exact values and the accountant-approved invoice configuration have not yet been provided. Filing returns is outside MVP scope.

A-03 — Catalogue: the client agrees to collect and approve model, variant, colour, battery/motor, warranty, price and vehicle-identifier data per brand. The actual verified source files remain pending.

A-04 — Opening stock: the client wants an import template. Actual opening-stock data, import owner and physical reconciliation sign-off remain pending.

A-05 — Roles: the intended roles are Owner, Inventory Manager, Sales Executive, Accountant and Viewer. The granular permission matrix and approval thresholds still require owner sign-off.

A-06 — Delivery checklist: confirmed categories are keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional rules and exception handling still require confirmation.


---

<!-- Source file: 01_PRD.md -->

> Source: Version 1.2 approved project document; converted from DOCX for searchability in this prompt pack.

# Product Requirements Document (PRD)

Product: Maa Ambey Enterprises — Inventory Management System | Version: 1.2 draft | Updated: 12 October 2026 | Status: Client high-level decisions received; catalogue, legal-document values and granular approvals remain pending

## 1. Product vision

Provide Maa Ambey Enterprises with one reliable, searchable system for knowing exactly which electric scooters and related parts the showroom owns, where each item is, whether it is saleable, and what happened to it. The system should reduce manual spreadsheets, prevent duplicate or untraceable vehicle sales, and give the owner clear operational and stock reports.

The client has confirmed that multi-brand support is a mandatory P0 architecture requirement. Build one shared application and configurable catalogue that supports Lectrix EV, Ceeon India and Ujagar Electric as the initial intended operational brands, with additional brands addable through configuration. The client roster is confirmed, but the exact supplier/manufacturer identity, authorisation evidence, verified model/variant catalogue, approved prices, warranty and serial-field rules must be recorded per brand before operational purchasing/sales are enabled. Publicly available catalogue content is reference-only and must not be treated as verified showroom stock or approved pricing. Verify the relationship, if any, between Ujagar Electric and Saffron EV Group rather than assuming that the names represent the same legal supplier.

## 2. Goals and measurable outcomes

100% of saleable scooters in the system have a unique internal unit ID and at least one validated frame/chassis identifier before becoming available for sale.

No completed sale can be committed when the selected vehicle is unavailable, reserved by someone else, or already sold.

Every receipt, transfer, reservation, sale, cancellation, return and adjustment is represented in an audit trail and stock-movement ledger.

Staff can answer “what is available right now?” by model, variant, colour, location and status in under 30 seconds during pilot acceptance.

Opening stock can be imported with duplicate detection and reconciled to a signed physical count before go-live.

Management can review daily sales, stock value, ageing stock and low-stock spares without manually consolidating multiple sheets.

## 3. Users and permissions

|Role|Primary needs|Access baseline|
|---|---|---|
|Owner|Full visibility, setup, approvals, reports and user administration|All modules subject to explicit permissions; sensitive settings and final approvals|
|Inventory Manager|Receive stock, manage locations, PDI, serials, counts and adjustments|Inventory and suppliers; adjustments above configured limit require owner approval|
|Sales Executive|Customer, quotation, booking/reservation and sale initiation|Sales workflow; cannot alter cost price or stock ledger directly|
|Accountant|Payment reconciliation, invoice/receipt data and billing reports|Billing and authorised financial settings; no product or role administration by default|
|Future role: Service Advisor|Future service history and parts consumption if the later service module is approved|Out of MVP launch role set; do not enable unless separately approved|
|Viewer|Read-only stock, sales and assigned reports|Read-only access to explicitly assigned modules/reports; no edits|

Access should be granted by role and, where applicable, location. Any staff member may only perform actions allowed by explicit permissions; hiding a button is not a substitute for server-side enforcement.

## 4. Scope and release plan

### MVP — P0

1. Authentication, user management, role/permission administration and audit log.

2. Brand/model/variant/colour master and product images/documents.

3. Serialised vehicle registry (VIN/chassis/frame, motor, battery, charger where applicable).

4. Supplier management, purchase order, goods receipt note, PDI, reject/hold flow.

5. Stock ledger, unit lifecycle and stock search with filters.

6. Customer master, quotation, booking/reservation, sales order and delivery checklist.

7. Payment receipt tracking and configurable printable invoice/receipt/delivery note templates; accountant sign-off required before production invoice use.

8. Dashboard and exports/reports.

9. Opening-stock import template, duplicate checks, backup and deployment runbook.

### Phase 2 — P1, separate approval

Service/workshop job cards, warranty case tracking, supplier purchase invoice matching, advanced finance/RTO workflow, multi-location transfer approvals, alerts via external messaging, public website lead synchronisation, accounting/GST/e-invoice/e-way bill integrations and deeper BaaS contract management.

### Out of scope for MVP

Native iOS/Android apps, full accounting/ERP, payroll, financing decisioning, insurer/RTO government filing, payment gateway settlement automation, manufacturer API scraping or integrations without available authorised APIs.

## 5. Core workflows

### 5.1 Purchase and receiving

1. Inventory Manager creates/selects supplier and adds a purchase order with brand/model/variant, quantity, agreed cost and expected date.

2. On arrival, the user creates a goods receipt against the PO and captures each scooter’s serial identifiers individually.

3. System validates required identifiers, duplicates and model/variant against the order.

4. PDI records pass/fail/hold and photos/notes. Failed items go to QUALITY_HOLD or REJECTED; they are not saleable.

5. Accepted units enter AVAILABLE or DISPLAY based on configured policy, and a stock movement is recorded.

### 5.2 Reservation and sale

1. Sales Executive selects/creates customer and prepares quotation with model/variant, price, approved discount and applicable configured tax calculation.

2. If customer books, system assigns a specific vehicle unit or a model-level allocation policy approved by the owner; MVP recommendation is reserve a specific unit only when the actual unit is selected.

3. Reservation has expiry date/time and payment/advance status. Expiration releases the unit only through an explicit system action or scheduled job with audit log.

4. Sales order converts to delivery workflow after all required approvals and payment conditions are satisfied.

5. Finalisation atomically checks the unit is still reserved/available, creates the sale record and movement, and marks the vehicle SOLD/DELIVERED.

6. Delivery checklist tracks the client-confirmed categories: keys, charger, manuals, warranty documents, payment, insurance and registration, alongside PDI and customer acknowledgement where applicable. Which items are mandatory or conditional by model/transaction remains configurable and requires owner sign-off.

### 5.3 Stock adjustment / count

1. User submits count or adjustment with item, expected quantity/status, observed value and mandatory reason.

2. System creates a pending adjustment; the user who initiated it cannot self-approve if approval is required.

3. Approver confirms, rejects or asks for recount. Approval posts a movement record and audit entry.

4. Deleting or silently overwriting stock history is prohibited.

## 6. Functional requirements

|ID|Requirement|Priority|Acceptance criteria|
|---|---|---|---|
|FR-001|Secure login/logout and account status|P0|Disabled users cannot create new sessions; failed attempts are rate-limited; session revocation is supported.|
|FR-002|Role/permission control|P0|API enforces permissions; read-only users cannot mutate data even when calling endpoints directly.|
|FR-003|Multi-brand catalogue and product governance|P0|Multiple brands can coexist in one deployment. Brand, model, variant, supplier mapping, pricing, warranty and identifier rules remain correctly associated with their brand; inventory/sales/reports can be filtered by brand; a brand cannot be used for purchase/sales until its required data and approval are complete.|
|FR-004|Serialised vehicle registry|P0|Chassis/frame number is unique; duplicate motor/battery serial is flagged based on configured uniqueness rules; identifiers can be searched.|
|FR-005|Purchase order and supplier master|P0|Purchase order includes supplier, model/variant, quantity, costs, expected date and status history.|
|FR-006|Goods receipt and PDI|P0|Partial receipts supported; each scooter is individually captured; rejected/held units cannot be sold.|
|FR-007|Stock movement ledger|P0|All stock state changes create immutable ledger events with actor, timestamp, from/to status, reference and reason.|
|FR-008|Reservation lifecycle|P0|Same unit cannot be actively reserved by two bookings; expiry/cancellation releases stock with audit history.|
|FR-009|Sales order and delivery|P0|Sale finalisation is transactional; blocked when unit unavailable, payment/approval incomplete or mandatory checklist items fail.|
|FR-010|Payment tracking|P0|Receipts store amount, method, reference, status and allocation; refunds/voids require permission and reference. This is not a payment processor.|
|FR-011|Printable documents|P0|Invoice/receipt/delivery note can be previewed/printed/exported; numbering is unique per configured financial year/series; accountant approves tax layout.|
|FR-012|Reporting and export|P0|Date/model/brand/status filters produce consistent data; exported data respects permissions and includes generation timestamp.|
|FR-013|Search and barcode/QR|P0|Search supports chassis/frame, motor, battery, model, PO, customer phone and sales reference; unit QR opens a permission-checked unit detail.|
|FR-014|Import / opening balance|P0|Import preview shows row errors and duplicates; no partial commit on fatal errors; reconciliation report is produced.|
|FR-015|Spare/accessory inventory|P0-basic|Quantity stock supports SKU, supplier, reorder level and ledger movement; serial-tracked mode can be enabled for higher-risk items.|
|FR-016|Attachments|P0|Private files such as PDI photos and supplier bills require authenticated, authorised access and have size/type limits.|
|FR-017|Audit log|P0|Sensitive create/update/approve/cancel/export actions are captured; audit records are not editable through normal UI/API.|
|FR-018|Website lead import|P1|Only implemented after public website/API ownership and data mapping are confirmed; CSV fallback available first.|

## 7. Required data fields

### Vehicle model / variant

Brand, manufacturer model name, internal model code, vehicle type (two-wheeler EV for MVP), variant, colour, battery chemistry/type, battery capacity, claimed range, top speed, charger configuration, warranty terms (source and effective date), supported ownership scheme (owned/BaaS/other), active state, supplier/manufacturer SKU, photo/document links and versioned price references. Specifications must be marked verified/unverified and have a source/date.

### Individual vehicle unit

Internal unit ID, brand/model/variant/colour, chassis/VIN/frame number, motor serial, battery serial(s), charger serial (if captured), supplier, PO/GRN, receipt date, cost and cost currency, location, lifecycle status, PDI outcome, defects/notes, warranty start/reference as applicable, ownership scheme, reservation/sale reference, attachments and timestamps.

### Spare/accessory item

SKU, name, category, brand compatibility, unit of measure, quantity on hand (ledger-derived), reorder level, cost, selling price (if applicable), supplier, lot/batch or serial tracking flag, location and active status.

### Customer and sale

Name, phone, optional email, address/state/PIN as required for transaction, GSTIN if applicable, consent/purpose where marketing contact is stored, quote/sale references, vehicle unit, price lines, discount authorisation, tax fields approved by accountant, payments/receipts, delivery checklist and documents. Avoid collecting unnecessary personal data.

## 8. Non-functional requirements

Security: HTTPS, strong password hashing, secure HttpOnly SameSite session cookies, CSRF protection as appropriate, input validation, rate limiting, secrets in a managed secret store, private file access and periodic dependency updates.

Reliability: database backups at least daily with a defined retention period; restore drill before launch; transaction-safe sales and stock changes; alert on failed jobs/backups.

Performance: 95th percentile under 2 seconds for common paginated lists at pilot scale (target, not a guarantee); indexes on common identifiers and filters.

Auditability: append-only stock ledger and audit events; all date/time stored in UTC and rendered in Asia/Kolkata.

Usability: responsive desktop/tablet workflows, legible dense tables, keyboard-friendly forms, clear statuses and field-level errors.

Maintainability: documented API, migrations, error conventions, tests, environment configuration and seed/demo dataset.

Privacy: minimise customer data, restrict exports and agree retention/deletion with the client/legal advisor.

## 9. Success metrics and acceptance gate

Opening-stock reconciliation difference is zero or every discrepancy is explicitly approved and documented.

Duplicate chassis/frame identifiers are blocked.

All stock lifecycle transitions have test coverage, including cancellation, PDI fail, reservation expiry and sale concurrency.

Owner and inventory manager can complete one receipt-to-sale journey without developer intervention.

Invoice preview is signed off by the client's accountant before live use.

All critical and high security defects are resolved; backup restoration is demonstrated.

Staff are trained, credentials provisioned, production data verified, and rollback plan documented.

## 10. Assumptions and open decisions

Initial deployment is for one showroom, but the schema supports locations.

The platform must not be hardcoded to Lectrix or any single brand. Seed brand master records for the client-approved launch roster; model/variant catalogues and operational activation must follow the client/supplier evidence for each brand. The three-brand roster is confirmed; individual brand records can remain PENDING_VERIFICATION until required supplier, authorisation and catalogue evidence is approved, and cannot then be selected for live purchase/sales.

All prices/specifications are supplied or approved by client; public website scraping is not used as source of truth.

Invoice compliance depends on the client's legal/tax status and requires professional review.

Internet connection is available; offline-first stock mutations are not in MVP because they risk conflicting inventory.

An opening-stock import template is required. Actual vehicle/part stock data, the person preparing/importing it and the person signing off physical reconciliation remain to be confirmed.

## 11. Research references

See Research Brief & Decision Register, or: Maa Ambey website https://maa-ambey-enterprisesev.vercel.app/ ; Lectrix EV https://www.lectrixev.com/ ; Ceeon India https://ceeonindia.co.in/ ; CBIC invoice rules https://cbic-gst.gov.in/gst-invoice-rules.html ; NIC e-way bill portal https://docs.ewaybillgst.gov.in/ .

# 12. Client-confirmed baseline and remaining inputs

These updates record the client answers received on 12 October 2026. They are requirements context for implementation; they do not manufacture missing catalogue or legal values.

Brand roster: configure Lectrix EV, Ceeon India and Ujagar Electric as the intended initial brand roster; allow additional brands to be added later. Brand records may exist before commercial activation, but purchase/sales use requires verified supplier identity, authorisation, catalogue, approved price and warranty/identifier rules.

Legal and billing: the client confirms that legal business name, GST details and showroom address exist, and that GST and income-tax filings are performed. Exact values and accountant-approved invoice/numbering/tax settings must be collected from source documents. Tax-return filing, full accounting and government portal filing remain out of scope.

Catalogue: the client agrees to collect verified models, variants, colours, battery/motor details, warranty, approved prices and vehicle-identification rules per brand. Until provided and reviewed, records must be flagged as unverified.

Opening stock: provide an import template covering each vehicle and applicable serial fields. Actual opening data must be imported, duplicate-checked and physically reconciled before go-live.

Roles: initial roles are Owner, Inventory Manager, Sales Executive, Accountant and Viewer. Exact permission grants, self-approval restrictions and approval thresholds require owner sign-off.

Delivery: include keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory versus conditional fields and authorised override policy require sign-off.


---

<!-- Source file: 02_System_Architecture.md -->

> Source: Version 1.2 approved project document; converted from DOCX for searchability in this prompt pack.

# System Architecture & Technical Design

Product: Maa Ambey Enterprises — Inventory Management System | Version: 1.2 draft | Updated: 12 October 2026 | Implementation stack confirmed: React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository

## 1. Architecture goals

Keep stock, serial identities, financial references and audit history consistent under concurrent use.

Separate the web UI from business rules and persistent storage.

Enforce access control on the server and maintain an auditable history of sensitive operations.

Confirmed P0: support multiple EV brands from day one in one shared deployment and catalogue. Keep brand-specific model/variant data, supplier relationships, pricing, warranty, serial-field policies, purchasing, sales and reporting correctly scoped; do not hardcode a single brand. Start with one showroom while preserving location IDs for future expansion.

Avoid direct vendor integrations until documented APIs, authorisation and maintenance ownership are confirmed.

## 2. Recommended stack

|Layer|Recommendation|Rationale / notes|
|---|---|---|
|Web frontend|React + TypeScript + Vite, React Router, Tailwind CSS, accessible component primitives and TanStack Query|Single-repository SPA frontend with responsive staff screens, strict TypeScript, accessible components and a straightforward Vite development/build workflow.|
|API|Go (Golang) + Gin REST API (OpenAPI/Swagger)|Modular monolith with explicit handlers, request validation, middleware, services and testable business logic. Keep all stock/sales invariants in the Go API layer.|
|Database|PostgreSQL (managed)|Transactions, constraints, relational integrity, reporting and stable stock/sales records.|
|Database access / migrations|pgx connection pool with reviewed versioned SQL migrations (golang-migrate or equivalent)|Explicit SQL access and repeatable deployments; critical constraints, transactions and indexes remain visible and testable.|
|Authentication|Server-side session or managed identity; secure HttpOnly cookies|Avoid browser localStorage for session tokens. Enforce CSRF/session protections.|
|File storage|Private S3-compatible object storage|PDI images, supplier bills and documents; use short-lived signed URLs after authorisation.|
|Background jobs|Queue/worker (add only if needed for alerts, imports or scheduled expiry)|Avoid making core sales depend on an async queue; scheduled tasks must be idempotent.|
|Hosting|Managed app hosting + managed PostgreSQL + private object storage|Simple operations, backups, environment separation and scalable path. Provider choice is a deployment decision.|
|Observability|Structured logs, error tracking, uptime checks, DB metrics and backup alerts|Diagnose failed imports, unexpected stock movements and release issues.|

A modular monolith is preferred for the MVP. Do not introduce microservices, event streaming or a service mesh until usage and integration demands justify the operational overhead.

## 3. Logical architecture

Staff Browser (desktop/tablet)
        |
        | HTTPS / secure session cookie
        v
React + Vite Web App
  - Dashboard and reports
  - Product / vehicle registry
  - Purchasing / receiving / PDI
  - Reservations / sales / delivery
  - Admin / users / audit
        |
        | REST / JSON + validation
        v
Go + Gin API (modular monolith)
  - Auth & RBAC guards
  - Catalogue module
  - Suppliers & purchasing module
  - Inventory / stock ledger module
  - Customers & reservations module
  - Sales / billing module
  - Reporting / export module
  - Audit / attachment module
        |                   \
        | ACID transactions   \ signed URL / metadata
        v                     v
Managed PostgreSQL      Private Object Storage
        |
        +--> Backup / point-in-time recovery / monitoring

Optional later: email/SMS provider, accounting software, manufacturer APIs,
website lead integration, GST/e-invoice/e-way bill service (after approval).

## 4. Domain boundaries / modules

1. Identity & access: users, roles, permission keys, session invalidation, location scope.

2. Catalogue: brands, models, variants, colours, spec provenance, scheme/ownership options, approved prices and validity windows.

3. Supplier & procurement: supplier profile, purchase orders, PO lines, receipt notes, received quantities, bill attachment and PDI disposition.

4. Inventory: individual vehicle units, parts SKUs, location, statuses, stock movement ledger, cycle counts and adjustment approvals.

5. CRM-lite: customer records, enquiries/test rides only if needed, quotation and reservations.

6. Sales & delivery: order, unit allocation, payment receipts, discount approval, delivery checklist, invoice/receipt print record.

7. Reporting: derived dashboards, filtered reports and export jobs.

8. Audit & attachments: append-only audit events, document metadata, authorised access and retention policy.

Each module should own its business operations. Cross-module changes (for example finalising a sale and marking a vehicle sold) must execute in one database transaction or use a carefully designed idempotent workflow if an external system is involved.

## 5. Data model (recommended core tables)

|Entity|Important attributes / relationship|
|---|---|
|users|id, name, email/username, password/identity subject, active, last_login_at|
|roles, permissions, user_roles, role_permissions|RBAC grants; location scope may be separately modelled|
|locations|id, code, name, type, address, active; initial seed is one showroom|
|brands|id, name, lifecycle_status (PENDING_VERIFICATION / ACTIVE / ARCHIVED), authorisation_status, authorisation_reference, verified_at, approval owner|
|models|id, brand_id (required FK), name, internal_code, vehicle_type, active; enforce brand-scoped code uniqueness|
|variants|id, model_id (and thus brand), variant_code, battery_type/capacity, colour, specs JSON or typed columns, spec_source, spec_verified_at; brand-specific serial/warranty/scheme settings|
|price_lists / price_list_items|variant/brand context, effective_from/to, price_type, amount, tax treatment reference, approval actor; keep price and discount history by brand/variant/scheme, never overwrite approved history|
|suppliers|legal name, contact, GSTIN if applicable, address, status, documents; supplier may supply one or more brands|
|purchase_orders, purchase_order_lines|supplier, dates, status, qty, agreed cost, currency, terms/reference|
|goods_receipts, goods_receipt_lines|PO link, receipt date, received quantity, accepted/rejected quantity, attachments|
|vehicle_units|internal_unit_id, variant_id, chassis_number (unique), motor_serial, battery_serials, charger_serial, location_id, status, supplier_id, PO/GRN refs, cost, PDI fields|
|stock_items|SKU, name, category, tracking_mode, UOM, reorder_point, active|
|stock_item_balances (optional cache)|item/location and derived quantity; reconciled against ledger; do not use as sole truth|
|stock_movements|immutable event, unit_id or stock_item_id, qty delta, from/to location/status, reason, source reference, actor, time, idempotency key|
|stock_counts, stock_count_lines, stock_adjustments|observed count, variance, reason, approval, posted movement reference|
|customers|name, phone, email, address, state, GSTIN only when applicable, marketing consent/purpose fields|
|quotations, quotation_lines|customer, model/unit, effective price, discount approval, valid_until, status|
|reservations|unit_id, customer_id, expiry_at, advance amount/status, cancellation reason, state|
|sales_orders, sales_order_lines|customer, exact unit, approved commercial snapshot, status, totals|
|payment_receipts|order_id, amount, method, external reference, received_at, status, reversal reference|
|sales_documents|type, number, financial_year, series, immutable commercial/tax snapshot, file reference/status|
|delivery_checklists|required item keys, checked_by, checked_at, evidence/document reference|
|audit_events|actor, action, entity, entity_id, before/after diff or structured change summary, request ID, IP policy, timestamp|
|attachments|owner entity, private object key, content type, size, checksum, uploader and access policy|

Multi-brand data integrity: Add a brand_supplier_links junction table when suppliers can supply multiple brands. Store each relationship’s verification/authorisation status, effective dates, supporting document and verifier. Every query, export and report that includes catalogue or vehicle records must retain brand_id context and enforce the user’s permissions server-side. Use brand-scoped uniqueness for manufacturer model/variant codes where codes can overlap between brands; chassis/VIN remains globally unique across the showroom inventory.

Serial data note: If one model can have multiple battery packs or replacement packs, store batteries in a separate battery_units table and link installation/removal history; do not pack multiple battery serials into an unvalidated text field. Use the simplest model that fits supplier documentation, but preserve traceability.

## 6. Key state machines

### Vehicle lifecycle

ORDERED -> IN_TRANSIT -> RECEIVED_PENDING_PDI
                              |             |
                              v             v
                        QUALITY_HOLD     AVAILABLE <-> DISPLAY
                              |             |
                              v             v
                         REJECTED       RESERVED -> AVAILABLE (cancel/expire)
                                            |
                                            v
                                       SOLD -> DELIVERED

Any physical unit needing inspection/repair may move to SERVICE_HOLD only via
an authorised transition. Sold/delivered units are not returned to available
without an explicit return/credit/inspection workflow and approvals.

### Purchase order lifecycle

DRAFT -> SUBMITTED -> APPROVED -> PARTIALLY_RECEIVED -> RECEIVED/CLOSED with explicit CANCELLED state where allowed. Receipt quantities cannot exceed accepted PO remaining quantity without an authorised override and reason.

### Reservation / sales lifecycle

DRAFT_QUOTE -> QUOTED -> RESERVED -> SALE_PENDING_APPROVAL -> CONFIRMED -> DELIVERED. Cancellation, expiry, refund and reversal must be explicit states/events. Avoid deleting completed records.

## 7. API design conventions

Base path: /api/v1.

JSON request/response, consistent error envelope (code, message, field errors, requestId).

Pagination: cursor or page/limit with server maximum; stable sort required.

Filtering: query params for brand, model, variant, colour, status, location, receipt date and ageing.

Use idempotency keys for sale finalisation, receipt posting, payment capture/reversal and bulk imports.

Use optimistic concurrency (version or updatedAt) for edits to reservations, units and POs.

OpenAPI contract committed with code; validation schemas are server-side source of truth.

### Endpoint sketch

|Method|Endpoint|Purpose|
|---|---|---|
|GET/POST|/api/v1/brands|List/create brands (write permission required)|
|GET/POST|/api/v1/models|Catalogue models and variants|
|GET|/api/v1/vehicle-units|Search/filter individual vehicle records|
|POST|/api/v1/purchase-orders|Create purchase order|
|POST|/api/v1/goods-receipts|Create draft receipt|
|POST|/api/v1/goods-receipts/:id/post|Validate and post receipt / stock movements|
|POST|/api/v1/vehicle-units/:id/pdi|Record PDI outcome|
|POST|/api/v1/reservations|Create reservation with unit lock/validation|
|POST|/api/v1/reservations/:id/cancel|Cancel reservation with reason|
|POST|/api/v1/sales-orders|Create sale draft|
|POST|/api/v1/sales-orders/:id/finalise|Atomic stock/payment/approval checks, sale commit|
|POST|/api/v1/stock-adjustments|Request adjustment|
|POST|/api/v1/stock-adjustments/:id/approve|Approval and ledger posting|
|GET|/api/v1/reports/stock-summary|Current stock derived by model/status/location|
|GET|/api/v1/reports/sales-summary|Sales totals by time/product/owner|
|GET|/api/v1/audit-events|Permission-limited audit search|

Brand-aware list/report endpoints accept validated brandId filters; all-brand totals are available only within the caller’s authorised scope. Provide a permissioned supplier-brand relationship workflow/API as part of catalogue governance. This is a contract sketch, not a final locked API. Refine after discovery and write OpenAPI definitions before frontend integration.

## 8. Transaction and inventory consistency rules

A vehicle unit has at most one active reservation at a time. Enforce via transaction and/or database constraint.

Sale finalisation must lock/check the unit row, verify saleable status, create sale/document snapshot, post movement and set unit status in the same transaction.

Never calculate current stock solely from editable form fields. Use the vehicle registry plus posted movement ledger; parts balances may be cached but ledger reconciliation is mandatory.

Stock movement rows are append-only for normal app roles. Correct an error using reversal/correction events, not destructive edits.

Negative quantity is blocked for tracked stock unless an explicitly approved exceptional workflow exists (MVP recommendation: block it).

Imported rows are first staged and validated; commit only after preview/confirmation. Idempotency prevents repeated imports from duplicating stock.

## 9. Security and privacy

RBAC enforcement in API guards, not just UI. Deny by default.

Secure cookies, short session lifetime appropriate to staff use, CSRF protection, rate limiting and session revocation.

Do not store authentication tokens in localStorage. Do not include sensitive customer data in logs or analytics.

Private file bucket, MIME/size validation, malware scanning if available, short-lived signed links and authorised download checks.

Apply least privilege to exports; record who exported sensitive reports.

Secrets managed by deployment platform, distinct dev/staging/production environments, encrypted connections and managed DB encryption at rest.

Define backup retention, RPO/RTO with client. Suggested starting target: daily backups, 30-day retention, RPO <= 24 hours and RTO <= 8 business hours, subject to hosting capabilities and owner approval.

## 10. Testing strategy

Unit tests for price snapshots, status transitions, permissions and validation rules.

Database integration tests for duplicate chassis, reservation collisions, transaction rollback and idempotency.

API contract tests for role and location restrictions.

End-to-end tests: PO -> receipt -> PDI pass -> available -> reserve -> sale -> delivery; PDI fail; cancelled booking/refund; stock adjustment approval; duplicate import.

Security tests: IDOR, privilege escalation, unauthorised attachment downloads, CSRF/session behaviour, rate limiting and export permissions.

Reconciliation tests: movements vs current vehicle statuses and parts-ledger balances.

Restore test: restore a backup into non-production and verify record counts and audit continuity.

## 11. Deployment and operations

Environments: local, staging and production with isolated databases/storage.

CI: lint, type-check, unit/integration tests, build, migration validation and dependency/security scan.

Production release: backup -> migration dry-run where supported -> deployment -> smoke checks -> monitor; rollback steps documented.

Do not run destructive migrations against production without a verified backup and explicit approval.

Seed client-confirmed role names (Owner, Inventory Manager, Sales Executive, Accountant, Viewer) and the three client-confirmed brand master records idempotently. Keep fine-grained permission grants pending owner sign-off. Use only sample data in non-production; production stock comes from a verified, signed-off import.

Monitor failed login spikes, import errors, sale-finalisation failures, background expiry jobs and backup status.

## 12. Decisions requiring sign-off

1. Confirm deployment provider, API host and database region.

2. Decide whether billing documents are legally issued by this app or by external accounting software.

3. Approve whether motor/battery/charger serials are mandatory and whether battery swaps need historical component links.

4. Approve location hierarchy and who can view/cross-transfer inventory.

5. Confirm BaaS modelling depth and whether only a label is needed in MVP.

6. Confirm backup/RPO/RTO target and document retention policy.

## 13. Reference inputs

Catalogue and business assumptions are based on Maa Ambey Enterprises https://maa-ambey-enterprisesev.vercel.app/, Lectrix EV https://www.lectrixev.com/, Ceeon India https://ceeonindia.co.in/, CBIC invoice rules https://cbic-gst.gov.in/gst-invoice-rules.html and NIC E-Way Bill portal https://docs.ewaybillgst.gov.in/. See Research Brief for specific findings and caveats.

# 14. Confirmed implementation stack and client configuration

The business architecture stays a modular monolith with one shared multi-brand application. The implementation stack is now explicitly React + TypeScript + Vite for the frontend, Go + Gin for the REST API, PostgreSQL accessed with pgx, and versioned SQL migrations. Frontend and backend must remain in one Git repository.

## Repository and runtime boundaries

Use /frontend and /backend directories in one repository, with common root documentation, Docker Compose, Makefile and validation commands. Do not create separate frontend/backend repositories.

Frontend calls the actual Go API using an environment-based API URL. Never place private secrets or the PostgreSQL connection string in Vite-exposed variables.

Use explicit SQL through pgx; do not introduce an ORM or switch away from the confirmed Go + React stack. Use versioned migrations for schema changes and tests for critical uniqueness/transaction constraints.

## Client-confirmed configuration

Seed the initial brand records for Lectrix EV, Ceeon India and Ujagar Electric, and keep the brand model extensible for future brands. Brand lifecycle status and evidence-based purchase/sales authorisation must remain separate.

The five initial role names are Owner, Inventory Manager, Sales Executive, Accountant and Viewer. Fine-grained permission assignments and approval thresholds require owner sign-off.

The client confirms legal name, GST details and showroom address exist, and that GST and income-tax filings are performed. Exact values and accountant-approved invoice configuration still need to be collected. Do not implement return filing in this MVP.

Opening-stock template and delivery checklist are confirmed needs. Actual stock data, reconciliation sign-off, and mandatory-versus-conditional delivery fields remain pending.


---

<!-- Source file: 03_Business_Rules.md -->

> Source: Version 1.2 approved project document; converted from DOCX for searchability in this prompt pack.

# Business Rules & Validation Specification

Product: Maa Ambey Enterprises — Inventory Management System | Version: 1.2 draft | Updated: 12 October 2026 | Client high-level decisions received; data evidence and approval details remain to be completed

## 1. Rule principles

1. Inventory is an auditable business record, not a manually editable counter.

2. A complete scooter is tracked per physical unit. Parts/accessories can be quantity-tracked or serial-tracked by configuration.

3. Every stock-changing action requires an authenticated actor, permission, timestamp, source reference and reason where appropriate.

4. Completed records are reversed/corrected, not silently deleted.

5. Brand specifications, prices and warranty details require source provenance and client approval; public marketing pages are not the system of record.

6. Tax/invoice behaviour must be configured and approved for the client's legal entity by a qualified accountant/tax professional.

## 2. Catalogue rules

BR-001: A brand is PENDING_VERIFICATION, ACTIVE or ARCHIVED. Only active, client-authorised brands may be used for purchase/sales operations.

BR-002: Multi-brand support is a confirmed P0 requirement. The initial intended operational brands are Lectrix EV, Ceeon India and Ujagar Electric; additional brands must be addable without schema or application redesign. Create master records for all three. Before any brand is enabled for purchase/sales, record the relevant supplier/manufacturer legal identity, current authorisation status, approval owner, reference document and verification date. The client roster is not itself proof of authorisation. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity without evidence.

BR-003: A model belongs to exactly one brand; a variant belongs to one model. Store brand ID context through the model/variant relationships. Enforce internal/manufacturer model and variant code uniqueness within the appropriate brand scope; internal database IDs remain globally unique.

BR-004: Specifications include source URL/document, source date, verifier and confidence (UNVERIFIED, SUPPLIER_CONFIRMED, CLIENT_APPROVED). Unverified specs must not be used as definitive customer-facing claims.

BR-005: Approved costs, selling prices, offers, warranties, specifications and ownership schemes are versioned by relevant brand/model/variant and effective dates. Store the source and approver; do not overwrite prior approved commercial snapshots. Public prices/ranges/warranty terms may vary by variant, location, scheme and offer.

BR-006: Archive rather than delete a catalogue item that has received, reserved or sold units.

BR-007: Inventory, purchase, sales, reservation and reporting screens must support filtering by brand. An “All brands” view must only aggregate brands the signed-in user is permitted to access.

BR-008: Pending-verification and archived brands/models/variants may remain visible to authorised catalogue administrators for setup/history, but cannot be used to receive or sell new operational stock. Existing historical transactions must retain their original brand even after a brand is archived.

## 3. Vehicle identity and lifecycle

INV-001: Every received vehicle gets a unique internal unit ID.

INV-002: Chassis/VIN/frame number is mandatory before a unit can leave RECEIVED_PENDING_PDI and is unique across all units in the tenant, after normalisation (trim, case and permitted separators).

INV-003: Motor serial is required when supplier paperwork/vehicle labelling provides one; define whether it is globally unique after client validates brands. Duplicate serials should be blocked or routed to an authorised exception with documented reason.

INV-004: Battery serial and charger serial requirements are configurable by brand/model/variant. If batteries can be swapped, maintain component history rather than overwriting the previous serial.

INV-005: Allowed statuses: ORDERED, IN_TRANSIT, RECEIVED_PENDING_PDI, QUALITY_HOLD, AVAILABLE, DISPLAY, RESERVED, SERVICE_HOLD, SOLD, DELIVERED, REJECTED, RETURN_PENDING_INSPECTION, RETURNED_TO_SUPPLIER. Not all transitions are available to all roles.

INV-006: Only AVAILABLE units can be newly reserved/sold. DISPLAY units require an authorised saleable transition or explicit test-display sale approval.

INV-007: QUALITY_HOLD, SERVICE_HOLD, REJECTED, SOLD and DELIVERED units cannot be reserved as available stock.

INV-008: A PDI pass requires an actor, timestamp and checklist outcome. PDI fail places the unit on quality hold or rejected state with issue notes and evidence.

INV-009: A location/status change creates a stock movement event. Changing the unit's location field without the corresponding movement is prohibited.

INV-010: Sale/delivery is not undone by editing status. Returns require a return case, inspection, financial credit/refund process and authorised disposition.

## 4. Procurement and receiving

PO-001: Every PO has a unique number, supplier, create date, creator, at least one line, quantity and agreed cost; approval is required above a configurable amount or for first-time supplier where enabled.

PO-002: PO states: DRAFT, SUBMITTED, APPROVED, PARTIALLY_RECEIVED, RECEIVED/CLOSED, CANCELLED.

PO-003: Receipt quantities cannot exceed the remaining approved PO quantity without a permissioned override, reason and audit event.

PO-004: A goods receipt captures actual received units, serial identifiers, condition, documents and discrepancy notes; partial receipts are allowed.

PO-005: Posted receipt is immutable. Correct receipt errors by a reversal/correction workflow and a new receipt where appropriate.

PO-006: A unit in RECEIVED_PENDING_PDI or on hold does not count as saleable stock. Reports distinguish physical received stock from saleable stock.

PO-007: Supplier bill amount and PO cost variance over a configurable tolerance raises a warning and, if configured, an approval requirement.

## 5. Stock ledger and counts

LED-001: Each movement records item/unit, quantity or unit status, from/to location, from/to status, movement type, source reference, actor and timestamp.

LED-002: Posted movements are append-only for normal roles. Any correction creates a reversal/correction movement linked to the original.

LED-003: Current vehicle stock is derived from one unit record per vehicle plus posted movements; the unit record and movement ledger must reconcile.

LED-004: Negative quantity is blocked for quantity-based inventory in MVP.

LED-005: Stock count variance requires reason. Variance above a configurable value/quantity threshold requires owner/inventory manager approval.

LED-006: A user who submits a stock adjustment may not approve their own adjustment if approval is required.

LED-007: Barcode/QR is a lookup convenience, not permission; scanned records still enforce server-side access checks.

LED-008: Stock age is calculated from accepted GRN date (or agreed opening-stock date), and ageing bands are configurable (recommended defaults: 0–30, 31–60, 61–90, >90 days).

## 6. Reservation, quotation and sales

SAL-001: An active reservation refers to at most one specific unit in MVP; the same unit cannot have two active reservations.

SAL-002: Reservation expiry requires an expiry timestamp and an explicit event when released. An expired booking does not silently disappear from history.

SAL-003: Quoted price is a snapshot. Subsequent price-list changes do not silently modify an existing quote/reservation.

SAL-004: Discount above the configured role limit requires approval before quote confirmation/sale finalisation.

SAL-005: Sale finalisation re-checks in one transaction that the unit is still reserved by this order or otherwise available, required approvals exist and no conflicting sale has committed.

SAL-006: Each sale line identifies the exact vehicle unit. A chassis/frame number cannot appear in two completed active sales.

SAL-007: Finalised sales are not hard-deleted. Cancellation/credit note/return actions require an approved reason and appropriate financial reversal reference.

SAL-008: Customer identity fields are validated. Collect only data required for the sale/registration/accounting workflow; marketing use requires a separately recorded lawful basis/consent where applicable.

SAL-009: Delivery checklist requirements are configurable by model/transaction. Client-confirmed checklist categories are keys, charger, manuals, warranty documents, payment, insurance and registration; PDI, customer acknowledgement, finance and RTO status may also apply where approved. Mark each item mandatory or conditional only after owner sign-off; support an authorised override with reason where policy permits.

SAL-010: Unit moves to DELIVERED only after required checklist fields are satisfied or an authorised override with reason is recorded.

## 7. Payments and documents

PAY-001: Payment records include amount, currency, method, date/time, reference and status. Methods may include cash, UPI, bank transfer, card or other configured methods.

PAY-002: Payment status must not be inferred from a user-entered note alone; store receipt/reference and reconciliation status separately.

PAY-003: A payment reversal/refund requires permission, amount, reason, original receipt reference and external refund/bank reference where available.

PAY-004: This system is not a payment gateway. Do not store card PAN, CVV, UPI PIN or banking credentials.

DOC-001: Invoice/receipt numbering is unique within the configured financial year and series; it must not be reused after issue, even if later cancelled.

DOC-002: The document stores a snapshot of supplier/legal entity, customer, item, price, discount, HSN/SAC where applicable, tax values, place of supply and other required fields approved by the accountant.

DOC-003: Do not hardcode GST rates, e-invoice thresholds or e-way bill conditions into UI constants. Maintain effective-dated configuration and have statutory changes reviewed.

DOC-004: Printed document templates need written sign-off from the client's accountant/tax advisor before production use. Legal requirements can change.

DOC-005: External tax system references/status are stored separately from the internal document number if/when an integration is implemented.

## 8. Security and permissions

SEC-001: All APIs deny access by default; route guards and object-level checks enforce permissions.

SEC-002: Users may only access locations, reports and documents within their assigned scope unless explicitly granted multi-location access.

SEC-003: Editing cost price, tax configuration, discount approvals, stock adjustments, user roles, invoice series, refunds and data exports requires dedicated permissions.

SEC-004: Sensitive exports and role/permission changes are audited.

SEC-005: Attachments are private and served only after server-side authorisation; public object URLs are prohibited for customer/supplier documents.

SEC-006: Sessions use secure cookies; session credentials are not stored in browser localStorage.

SEC-007: Production data should not be copied into developer tools/test fixtures without masking and approval.

## 9. Import/export rules

IMP-001: Import files are staged for preview; validate required fields, formats, duplicates, brand/variant mapping and serial identifiers before committing.

IMP-002: Duplicate chassis/frame numbers are fatal import errors unless a reviewed migration exception process is approved.

IMP-003: Each import has a batch ID, uploader, source filename, hash, validation report and commit status. Replaying the same committed batch must not create duplicates.

IMP-004: All exports include generation time and active filters and respect the user's permissions. Exports of customer data are audited.

IMP-005: Importing opening stock does not fabricate supplier/PO history; rows must clearly identify as opening balance where original provenance is unavailable.

## 10. Reporting rules

REP-001: Available saleable stock excludes reserved, service-hold, quality-hold, rejected, sold and delivered units.

REP-002: Physical stock, saleable stock, reserved stock and in-transit stock are reported as distinct metrics.

REP-003: Stock valuation uses the selected, disclosed cost basis (recommended: recorded unit acquisition cost); never mix selling price with inventory cost.

REP-004: Gross sales, discounts, tax and receipts are shown separately; sales revenue is not equated with cash collected.

REP-005: Report filters and definitions are documented and consistent between dashboard and exports.

## 11. Validation/error behaviour

Errors should explain which field or business condition failed and how to fix it, without revealing secrets or internal stack traces.

Save draft where safe; never partially commit a sale/receipt if critical validation fails.

Display stale-data/conflict errors when another user changed a record; ask the user to reload/review rather than silently overwriting.

Destructive/irreversible actions require confirmation and a clear object identifier.

Server and client validations should be consistent, but server validation is authoritative.

## 12. Seed configuration recommendations

Seed the application with the client-confirmed role names (Owner, Inventory Manager, Sales Executive, Accountant and Viewer) and the brand master records Lectrix EV, Ceeon India and Ujagar Electric. Additional brands must be addable without schema changes. Detailed role-to-permission grants and approval thresholds remain pending owner sign-off. Publicly observed Lectrix/Ceeon catalogues may inform a draft import, but every model/variant/specification/price must be UNVERIFIED until confirmed by the client or supplier. All three launch brand records may be configured now, but a brand without the required supplier identity, catalogue, pricing, warranty/identifier rules and authorisation evidence must remain blocked from operational purchase/sales. Verify the relationship, if any, between Ujagar Electric and Saffron EV Group.

## 13. Compliance references

CBIC GST invoice rules: https://cbic-gst.gov.in/gst-invoice-rules.html NIC GST e-Way Bill portal: https://docs.ewaybillgst.gov.in/

These rules are system design guidance, not legal or tax advice. The owner's accountant/tax advisor must validate the actual business-specific invoice, tax and document obligations.

# 14. Client-confirmed configuration rules and unresolved approvals

These rules are based on the client answers received on 12 October 2026. They define the confirmed requirements without pretending that missing source data or fine-grained approval decisions have already been supplied.

## Launch brand roster

Initial intended operational brands: Lectrix EV, Ceeon India and Ujagar Electric. New brands must be supported through data/configuration, not hardcoded code branches.

The brand master record may exist before evidence is complete. Purchase/sales activation requires supplier/manufacturer identity, current authorisation evidence, verified catalogue, approved price and warranty/identifier rules.

Do not assume Ujagar Electric is the same legal supplier or brand as Saffron EV Group; record the verified identity/relationship.

## Roles and approval matrix

Initial role names confirmed by the client: Owner, Inventory Manager, Sales Executive, Accountant and Viewer.

Deny access by default and enforce permission checks in the Go API. The exact role-to-permission matrix, discount limits, stock adjustment thresholds and self-approval exceptions require owner approval before production.

## Legal, import and delivery inputs

The client confirms legal business name, GST details and showroom address exist, and that GST and income-tax filings are performed. Actual values and invoice settings must be captured from verified documents and approved by the accountant/tax adviser. Tax-return filing is out of scope.

An opening-stock template is required. Actual opening stock must come from client data, pass duplicate/required-field validation and be reconciled to physical stock before go-live.

Confirmed delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Mandatory/conditional status and override policy require owner sign-off.


---

<!-- Source file: 04_Design_Specification.md -->

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


---

<!-- Source file: 05_Implementation_Tasks.md -->

> Source: Version 1.2 approved project document; converted from DOCX for searchability in this prompt pack.

# Implementation Plan & Engineering Task Backlog

Product: Maa Ambey Enterprises — Inventory Management System | Version: 1.2 draft | Updated: 12 October 2026 | Planning estimates are indicative; implementation stack confirmed as React + TypeScript + Vite, Go + Gin, PostgreSQL + pgx in one Git repository.

## 1. Delivery approach

Ship a robust single-location MVP before integrations. Work in vertical slices so the UI, API, validation, audit trail and tests for each workflow move together. Do not build only static screens and postpone core API/data constraints: duplicate serial prevention and transactional sale finalisation are foundational requirements.

### Proposed milestones

|Milestone|Outcome|Indicative duration|
|---|---|---|
|M0 Discovery & specification sign-off|Legal/entity data, actual brands/models, invoice policy, opening stock template, role matrix|3–5 working days|
|M1 Foundation|Repo/CI, app shell, auth/RBAC, DB migrations, audit, environments|1 week|
|M2 Catalogue & procurement|Brand/model masters, suppliers, POs, GRN, individual unit records, PDI|1–2 weeks|
|M3 Ledger & operational screens|Movement history, stock search, counts/adjustments, parts inventory|1 week|
|M4 Sales & delivery|Customers, quotations, reservations, payments, documents, delivery checklist|1–2 weeks|
|M5 Reports, hardening & pilot|Reports/import/export, security tests, backup restore, training, launch|1–2 weeks|

Expected MVP planning range: approximately 6–8 calendar weeks for a focused small team, assuming timely client answers and no major third-party integration. Add contingency for tax/invoice integration or migration complexity.

## 2. Backlog by epic

### EPIC A — Discovery, product decisions and source data

|ID|Task|Acceptance / deliverable|Priority|Estimate|
|---|---|---|---|---|
|A-01|Record confirmed launch brands and activation evidence|Create master records for Lectrix EV, Ceeon India and Ujagar Electric; allow future brands. Store evidence/status and block purchase/sales until supplier, authorisation and catalogue approvals are complete. Verify any Ujagar/Saffron relationship.|P0|0.5d|
|A-02|Capture verified legal and billing data|Capture exact legal name, GST details and showroom address from client documents; record financial year, invoice series and accountant approval. Client confirms GST/income-tax filing; return filing is out of scope.|P0|0.5d|
|A-03|Capture verified brand-specific model catalogues|Collect verified models, variants, colours, battery/motor details, identifiers, warranty and approved prices for all launch brands; track source and verification status. Approved catalogue files remain pending.|P0|1.5d|
|A-04|Prepare opening-stock import template|Provide opening-stock template with brand/model, identifiers, status and provenance; validate duplicates. Actual stock file, import owner and physical-count sign-off are pending.|P0|0.5d|
|A-05|Finalise role and permission matrix|Use the five confirmed role names. Obtain Owner sign-off for detailed permissions, approval limits, self-approval restrictions and export rules.|P0|0.5d|
|A-06|Finalise delivery checklist rules|Include keys, charger, manuals, warranty documents, payment, insurance and registration. Define mandatory/conditional items by model/deal and override policy after Owner approval.|P0|0.5d|

### EPIC B — Project foundation and security

|ID|Task|Acceptance / deliverable|Priority|Estimate|
|---|---|---|---|---|
|B-01|Create single-repository Go + React environment strategy|One Git repository with /frontend (React + TypeScript + Vite) and /backend (Go + Gin); document environment variables, Docker Compose, root Makefile and no-secret policy.|P0|0.5d|
|B-02|Configure lint, type-check, tests and CI|Pull request fails on lint/type/test/build failure|P0|0.5d|
|B-03|Bootstrap React + Vite app shell|Responsive React navigation, header, breadcrumbs and notification slot, route structure and placeholders for permission-aware navigation; no fake authentication.|P0|1d|
|B-04|Bootstrap Go + Gin API / OpenAPI|Versioned REST base path, request-struct validation, standard error envelope, request ID, health/readiness endpoints and OpenAPI documentation.|P0|1d|
|B-05|Database and migration baseline|PostgreSQL + pgx, reviewed SQL migrations and documented recovery. Seed five confirmed role names and three brand records idempotently; leave granular permissions for Owner sign-off.|P0|1d|
|B-06|Authentication and session handling|Login/logout, disabled account denial, secure cookies, CSRF/rate-limit controls|P0|1.5d|
|B-07|RBAC and location scope|API guards + object-level permissions; automated negative permission tests|P0|1.5d|
|B-08|Audit event framework|Captures actor/action/entity/reference/time for sensitive mutations; normal API cannot edit events|P0|1d|
|B-09|Attachment service|Private object storage, MIME/size validation and signed access after authorisation|P0|1d|

### EPIC C — Catalogue and price governance

|ID|Task|Acceptance / deliverable|Priority|Estimate|
|---|---|---|---|---|
|C-01|Implement brand/model/variant schema|Brand-model-variant relationships and unique codes enforced|P0|1d|
|C-02|Catalogue API and validation|CRUD with archive semantics, approval/provenance fields and RBAC|P0|1d|
|C-03|Catalogue UI|Search/table/editor, source status, brand activation warning and variant fields|P0|1.5d|
|C-04|Price list history|Effective dates, approval actor, quote snapshot; no silent retroactive changes|P0|1d|
|C-05|Seed multi-brand catalogue|Seed master records for Lectrix EV, Ceeon India and Ujagar Electric. Only activate verified/approved models for purchase/sales; keep future brands data-driven.|P0|0.75d|
|C-06|Brand-specific rules and isolation tests|Verify prices, warranties, serial-field rules and supplier mapping stay brand-specific; verify brand filters/exports and prevent cross-brand model selection or unauthorised activation|P0|1d|

### EPIC D — Procurement, vehicle units and PDI

|ID|Task|Acceptance / deliverable|Priority|Estimate|
|---|---|---|---|---|
|D-01|Supplier master|Supplier list/details, GSTIN/contact/documents and active/archive status|P0|0.75d|
|D-02|Purchase order API and UI|Draft/submit/approve lifecycle, line quantity/cost and PO history|P0|1.5d|
|D-03|Goods receipt draft workflow|Receipt against PO, partial receipt, serial capture, attachments and discrepancies|P0|1.5d|
|D-04|Vehicle unit registry|Unique internal ID, unique normalised chassis/frame, motor/battery/charger fields, status/location|P0|1.5d|
|D-05|Duplicate identifier controls|DB unique constraint + friendly server/client messages + integration test|P0|0.75d|
|D-06|PDI checklist and disposition|Pass/hold/reject, defect notes/photos; held units excluded from saleable results|P0|1d|
|D-07|Receipt posting transaction|Accepted units and stock movement posted atomically; idempotency and rollback test|P0|1.5d|
|D-08|Vehicle list/detail UI|Search/filter/chassis lookup, timeline, documents, permission-aware fields and actions|P0|1.5d|
|D-09|QR/barcode print/scan support|QR resolves to internal unit detail; scanner/manual entry fallback|P1/P0 if scanner needed|1d|

### EPIC E — Stock ledger, parts and reconciliation

|ID|Task|Acceptance / deliverable|Priority|Estimate|
|---|---|---|---|---|
|E-01|Append-only stock movement API|Movement includes source, actor, timestamp, state/location delta and idempotency key|P0|1d|
|E-02|Stock movement timeline UI|Filters by unit, event, location, actor, date and source reference|P0|0.75d|
|E-03|Spare/accessory SKU master|SKU, UOM, tracking mode, supplier, reorder level and compatibility metadata|P0-basic|1d|
|E-04|Parts receipts/issues|Quantity ledger; prevent negative stock; source references mandatory|P0-basic|1.5d|
|E-05|Stock count and adjustment requests|Count, variance, reason, approval and ledger posting workflow|P0|1.5d|
|E-06|Ledger reconciliation job/report|Detect difference between unit records and movement history; no silent correction|P0|1d|
|E-07|Stock ageing logic|Age from GRN/opening date with configurable bands and tests|P0|0.5d|

### EPIC F — Customer, reservation and sale

|ID|Task|Acceptance / deliverable|Priority|Estimate|
|---|---|---|---|---|
|F-01|Customer master|Search by phone/name, minimal data fields, duplicate warning and privacy-aware access|P0|1d|
|F-02|Quotation builder|Product/unit, approved price snapshot, validity, discount limits and printable preview|P0|1.5d|
|F-03|Discount approvals|Configurable role threshold, audit trail, cannot finalise while approval pending|P0|0.75d|
|F-04|Reservation lifecycle|Reserve a specific unit, expiry, cancellation reason and atomic collision protection|P0|1.5d|
|F-05|Reservation expiry handler|Idempotent scheduled/job action, audit event, safe release rules and tests|P0|0.75d|
|F-06|Sales order workflow|Draft, approval, payment checklist and exact unit allocation|P0|1d|
|F-07|Atomic sale finalisation|Transaction locks/validates unit, saves sale snapshot, posts movement and sets sold status|P0|1.5d|
|F-08|Payment receipt tracking|Multiple receipts/methods, reference/status, reversals and export|P0|1d|
|F-09|Delivery checklist|Include keys, charger, manuals, warranty documents, payment, insurance and registration. Configure required/conditional rules by model/deal; block missing requirements unless an audited override is permitted.|P0|1d|
|F-10|Invoice/receipt/delivery templates|Configurable series and tax fields, unique numbering, print/PDF; accountant approval tracked|P0|1.5d|
|F-11|Return/cancel/reversal workflow|No hard delete; return inspection and linked financial/stock reversal|P1 (basic cancellation P0)|1.5d|

### EPIC G — Reports and imports/exports

|ID|Task|Acceptance / deliverable|Priority|Estimate|
|---|---|---|---|---|
|G-01|Dashboard metrics|Physical vs saleable vs reserved vs held/in-transit metrics are clearly separated|P0|1d|
|G-02|Inventory reports|By brand/model/variant/status/location, ageing and acquisition-cost valuation|P0|1d|
|G-03|Sales and collections reports|Sales value, discount, tax, payment collected and outstanding shown separately|P0|1d|
|G-04|Pending PO/PDI/reservation reports|Ageing queue and drill-through filters|P0|0.75d|
|G-05|Export framework|CSV/XLSX, generated timestamp/filters, permission checks and export audit|P0|1d|
|G-06|Opening stock import preview|Validate schema, duplicate chassis, unresolved catalogue mapping and errors by row|P0|1.5d|
|G-07|Import commit/idempotency|Staging and all-or-nothing commit for fatal errors; batch IDs and audit log|P0|1d|

### EPIC H — QA, security, operations and launch

|ID|Task|Acceptance / deliverable|Priority|Estimate|
|---|---|---|---|---|
|H-01|Unit + integration test suite|Price snapshots, status transitions, permissions, duplicate identifiers, stock ledger|P0|2d|
|H-02|End-to-end critical workflows|PO -> GRN -> PDI -> available -> reserve -> sale -> delivery and failure paths|P0|2d|
|H-03|Concurrency/idempotency tests|Two users reserve/sell same unit; duplicate post/import request is safe|P0|1d|
|H-04|Security/access tests|IDOR, privilege escalation, attachment permissions, export restrictions, session checks|P0|1.5d|
|H-05|Performance test / indexes|Common list/search reports within target at agreed pilot data volume|P0|0.75d|
|H-06|Backup and restore drill|Restore backup in non-production and verify row counts/key reports|P0|1d|
|H-07|Staging/UAT and defect triage|Client signs critical journeys and invoice preview; all P0 acceptance defects closed|P0|1.5d|
|H-08|Data import and physical stock reconciliation|Opening stock import approved and discrepancies signed off|P0|1d|
|H-09|Training and runbooks|Admin guide, receiving/sales quick guide, backup/release and support runbook|P0|1d|
|H-10|Production rollout and smoke test|Production configuration, secure admin account, logs/backups/alerts, rollback option|P0|0.75d|

## 3. Suggested sprint sequencing

Sprint 0 / Confirmation and source-data sign-off: A-01 to A-06 are now answered at high level. Capture the three-brand roster, verify per-brand catalogue/authorisation, enter exact legal/GST/address details, prepare the opening-stock template, obtain owner sign-off on granular permissions and define mandatory/conditional delivery items. Sprint 1 / Foundation: B-01 to B-09 and skeleton dashboard/navigation. Sprint 2 / Catalogue & receiving: C-01 to C-06, D-01 to D-07. Sprint 3 / Inventory operations: D-08/D-09, E-01 to E-07. Sprint 4 / Sales: F-01 to F-11. Sprint 5 / Reporting and launch: G-01 to G-07, H-01 to H-10. This sequence is indicative; re-plan based on dependencies, team capacity and remaining client sign-offs.

## 4. Definition of Ready

A task is ready when user value and behaviour are clear, relevant screen states are described, API/data constraints are known, dependencies and permissions are listed, acceptance criteria are testable, and any required supplier/client decision is resolved or explicitly accepted as a blocker.

## 5. Definition of Done

Code reviewed; lint/type checks/build pass.

Unit/integration/E2E tests added for business-critical paths.

Server-side validation and permission enforcement implemented.

Audit trail and stock ledger behaviour verified for mutations.

Loading, empty, error, conflict and permission states present in UI.

API contract/docs updated; database migrations reviewed.

No secrets or real customer data in source, logs or tests.

Staging workflow verified; screenshot/QA evidence attached for UI work.

## 6. Release acceptance checklist

[x] Client confirms initial intended operational brands: Lectrix EV, Ceeon India and Ujagar Electric; collect verified supplier/authorisation evidence and current brand-specific catalogues before enabling purchase/sales.

[ ] Client confirms legal business name, GST details and showroom address exist, and GST/income-tax filings are performed; capture exact values and verify the accountant-approved invoice/numbering/tax configuration before production.

[ ] Opening stock import validated; physical count signed off.

[ ] No duplicate active chassis/frame numbers.

[ ] Receipt/PDI/reservation/sale/cancel/adjustment flows tested.

[ ] Two-user concurrency test cannot sell/reserve the same unit twice.

[ ] Roles tested using both UI and direct API requests.

[ ] Sensitive files are private; export permissions and audit checked.

[ ] Backup restore completed and runbook stored.

[ ] Staff trained; support owner and incident path communicated.

[ ] Production smoke test, rollback plan and monitoring are ready.

## 7. Deferred/P1 backlog

Public website lead/test-ride sync after data ownership and API feasibility review.

Service job cards, warranty claim records and part consumption.

Full BaaS contract/recurring subscription details after supplier requirements are confirmed.

Accounting system integration and statutory API integrations after accountant/provider assessment.

Additional locations, inter-store transfers and mobile scanner optimisation if business need warrants.

WhatsApp/SMS notifications subject to consent, provider cost and opt-out handling.

## 8. Dependencies and risks

Client must supply current catalogue, supplier relationship, warranty, approved pricing and relevant authorisation evidence for each brand/model to be enabled. The current public website substantiates Lectrix positioning only; it is not a substitute for per-brand sign-off.

Accountant must sign off invoice layout/numbering/tax configuration.

Opening stock quality determines migration effort; dirty serial fields may need a manual remediation sprint.

Any manufacturer/accounting/GST integration depends on accessible, documented APIs and credentials; it is not included in MVP estimates.

Security, backup and restore work cannot be cut from launch scope.

# 9. Client-answer reconciliation and updated implementation assumptions

Use this section to distinguish completed high-level decisions from data and approvals still needed. Do not reopen the confirmed choices unless the client changes them.

## Confirmed

Launch roster intent: Lectrix EV, Ceeon India and Ujagar Electric. Additional brands must be addable later.

Legal business name, GST details and showroom address exist; client files GST and income tax. Actual exact values and accountant-approved invoice settings still need to be collected.

The client agrees to collect verified model/variant/colour/battery-motor/warranty/approved-price/identifier data for each brand.

An opening-stock import template is required; actual inventory data is still pending.

Initial roles: Owner, Inventory Manager, Sales Executive, Accountant and Viewer. Fine-grained role/permission matrix remains pending approval.

Delivery checklist categories: keys, charger, manuals, warranty documents, payment, insurance and registration. Whether each is mandatory or conditional remains to be signed off.

## Technology and data guardrails

Implementation stack: one Git repository, React + TypeScript + Vite frontend, Go + Gin backend, PostgreSQL via pgx and versioned SQL migrations. Do not switch to a different framework or database-access approach during setup without an approved architecture decision.

Create all three initial brand master records, but keep purchasing/sales blocked until per-brand supplier identity, authorisation and approved catalogue evidence is present. Do not invent Ujagar/Saffron affiliation or product values.

The MVP is inventory/showroom operations, not tax-return filing or a full accounting system. Invoice output and applicable tax configuration require accountant review.
