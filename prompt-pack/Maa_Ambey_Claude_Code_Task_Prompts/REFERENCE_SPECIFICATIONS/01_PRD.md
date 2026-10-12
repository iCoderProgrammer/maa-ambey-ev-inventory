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
