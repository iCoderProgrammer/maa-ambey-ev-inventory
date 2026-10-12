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
