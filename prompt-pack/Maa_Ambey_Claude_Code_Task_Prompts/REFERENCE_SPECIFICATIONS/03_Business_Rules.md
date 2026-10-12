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
