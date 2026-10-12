# Client Decisions & Remaining Inputs — Version 1.2

Use these prompts before activating production operations. Client answers have already been received for the high-level decisions below. Do not ask the client to re-decide confirmed items; collect the missing source data/evidence and obtain any remaining approvals. Store outputs in `docs/client-inputs/`. Never invent answers.

## A-01 — Initial brand roster (roster confirmed; evidence pending)
**Confirmed by client:** The initial intended operational brands are Lectrix EV, Ceeon India, and Ujagar Electric. The system must support adding more brands later.

**Claude Code prompt:**

Create `docs/client-inputs/launch-brand-roster.md` that records the roster above as client-confirmed. For each brand collect legal supplier/manufacturer entity, supplier contact, dealer/stocking authorisation reference/document, catalogue source, verification status, approval owner, verification date, and allowed purchase/sales status. Do not treat roster selection as proof of authorisation. Keep brand existence/lifecycle separate from operational eligibility and block purchase/sales until required evidence is reviewed and approved. Do not assume Ujagar Electric and Saffron EV Group are the same legal entity. Do not invent missing details.

## A-02 — Legal and billing data (existence confirmed; exact values and sign-off pending)
**Confirmed by client:** The legal business name, GST details, and showroom address exist. The client files GST and income tax.

**Claude Code prompt:**

Create `docs/client-inputs/billing-and-legal-signoff.md` listing the actual source documents/values still to be entered: legal entity name, GSTIN and registration status as applicable, registered/showroom address and state, financial year, invoice numbering series, required invoice fields, cancellation/credit-note process, accountant/tax-adviser reviewer, and sign-off date. Clearly mark values not yet provided as pending; do not fabricate them or hardcode tax rates/thresholds. This inventory system is not a GST-return or income-tax-return filing system. Do not add government portal integrations without separate approval.

## A-03 — Brand-specific model catalogues (scope confirmed; actual data pending)
**Confirmed by client:** For each brand, collect verified models, variants, colours, battery/motor details, warranties, approved prices, and vehicle-identification rules.

**Claude Code prompt:**

Create `docs/client-inputs/brand-catalogue-template.md` and a fillable template covering brand, model, variant, colour, applicable battery/motor details, chassis/frame/motor/battery identifier rules, warranty terms, approved price and effective date, supplier/manufacturer source document, supplier confirmation, client approval, verification date and approver. Distinguish blank/unverified from zero/not applicable. Do not scrape or fabricate production data. Include the initial brands confirmed by the client but leave unknown catalogue values blank and unverified.

## A-04 — Opening stock (template approved in principle; actual data pending)
**Confirmed by client:** An opening-stock import template is needed for actual scooter quantities, models, chassis numbers, applicable battery serials, and stock statuses.

**Claude Code prompt:**

Create `docs/client-inputs/opening-stock-template.csv` plus a data dictionary and validation rules. Include brand/model/variant, chassis/frame, motor/battery/charger serials where applicable, colour, location, received/opening-stock date, condition/status, source/provenance, and supplier/PO/GRN/cost references only where available and approved. Define identifier normalization and duplicate checks. Provide a clearly fake example only in separate documentation, never as a production import row. Actual stock and physical reconciliation sign-off remain pending.

## A-05 — Role permission matrix (role names confirmed; exact permissions pending)
**Confirmed by client:** Owner, Inventory Manager, Sales Executive, Accountant, and Viewer roles are required.

**Claude Code prompt:**

Create `docs/client-inputs/role-permission-matrix.md` using exactly these five initial roles. For each sensitive action, propose view/create/edit/approve/export grants, location scope, approval threshold and self-approval rules, but mark the granular matrix as **PROPOSED — OWNER SIGN-OFF REQUIRED** until approved. Include dedicated considerations for stock adjustment, discount approval, cost-price visibility, refunds, user/role administration, invoice settings and exports. Do not add Service Advisor as an initial role unless the client approves it later. Backend authorization must deny by default.

## A-06 — Delivery checklist (categories confirmed; required/conditional policy pending)
**Confirmed by client:** Checklist coverage includes keys, charger, manuals, warranty documents, payment, insurance, and registration.

**Claude Code prompt:**

Create `docs/client-inputs/delivery-checklist-decisions.md` with these categories and any existing approved PRD items such as PDI/customer acknowledgement. For each item ask the client to mark Required, Conditional, Optional, or Not Applicable and state the evidence to record. Confirm override roles, reason requirements, payment conditions, finance/insurance/registration states, and whether policy varies by model/transaction. Do not hardcode unresolved mandatory/conditional rules or allow production delivery when the checklist policy has not been approved.

## Exit gate
Before enabling real purchasing/sales for each brand, confirm the corresponding supplier/authorisation and catalogue evidence. Before issuing production invoices, obtain actual legal/GST values and accountant approval. Before production stock import, validate the client's actual opening stock and arrange physical reconciliation. Before enabling mutation grants in production, obtain owner sign-off for the granular permission matrix. Before delivery workflow goes live, approve mandatory/conditional checklist policy. Never substitute invented values for pending inputs.
