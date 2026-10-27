# Launch Brand Roster and Activation Evidence (A-01)

Discovery item A-01 (`prompt-pack/Maa_Ambey_Claude_Code_Task_Prompts/00_START_HERE/00_Client_Discovery_Prompts.md`).
Created 2026-10-14. This file records what the client has confirmed and what evidence is still needed before any
brand may be used for purchasing or sales. It is the place to enter that evidence when the client supplies it.

**Status of this document:** roster confirmed by the client; **all per-brand activation evidence pending**.

## 1. Confirmed by the client

| Item | Status | Source |
| --- | --- | --- |
| Initial intended operational brands: **Lectrix EV**, **Ceeon India**, **Ujagar Electric** | Confirmed | Client answers of 12 Oct 2026 (Discovery A-01; `docs/REQUIREMENTS_BASELINE.md` §1) |
| More brands can be added later without code changes | Confirmed | Same |

Nothing else about these brands has been confirmed by the client.

## 2. Important limits

- **The roster is not proof of authorisation.** The client naming a brand confirms the intention to deal in it. It
  does not prove that Maa Ambey Enterprises is an authorised dealer, distributor or stockist of that brand, and it
  does not establish who the legal supplier or manufacturer is (BR-002).
- **Ujagar Electric and Saffron EV Group must not be assumed to be the same legal entity.** Any relationship between
  them (same company, brand owner, manufacturer, distributor, or none) must be shown by a document and recorded below.
  Until then they are treated as unrelated, and no alias is recorded.
- **Brand existence is separate from operational eligibility.** A brand record may exist (it does, pending
  verification) while purchasing and sales stay blocked until every activation requirement below is evidenced,
  reviewed and approved.
- Nothing in this file is invented. Every field without client evidence reads `Pending — client evidence not supplied`.

## 3. Per-brand evidence

Field meanings:

- **Legal manufacturer/supplier entity** — registered legal name of the company that supplies the showroom (and of the
  manufacturer, if different), as written on the supply/dealership document.
- **Supplier contact** — the named contact person/office for orders and documents.
- **Dealer/stocking authorisation** — the agreement, letter or certificate authorising the showroom to buy/stock/sell
  the brand: its reference number, date, validity and the private document copy (uploaded via the app's Documents).
- **Catalogue source and verification status** — where the model/variant list comes from (supplier price list,
  official catalogue document) and whether it has been confirmed by the supplier and approved by the client.
- **Approval owner** — the person at Maa Ambey Enterprises who approves the evidence (approval owner role is open
  question OQ-2).
- **Verification date** — the date the evidence was checked.
- **Allowed purchase/sales status** — whether the brand may be used operationally. It becomes "Allowed" only when the
  application's activation checklist is fully met and an authorised person enables the brand (audited).

### 3.1 Lectrix EV

| Field | Value |
| --- | --- |
| Roster status | Confirmed (client, 12 Oct 2026) |
| Legal manufacturer/supplier entity | Pending — client evidence not supplied |
| Supplier contact | Pending — client evidence not supplied |
| Dealer/stocking authorisation reference | Pending — client evidence not supplied |
| Authorisation evidence document | Pending — client evidence not supplied |
| Catalogue source | Pending — client evidence not supplied |
| Catalogue verification status | Unverified — no verified catalogue supplied (A-03) |
| Approval owner | Pending — client evidence not supplied (approval role: OQ-2) |
| Verification date | Pending — client evidence not supplied |
| Allowed purchase/sales status | **Not allowed** — blocked until the evidence above is recorded, reviewed and approved |

### 3.2 Ceeon India

| Field | Value |
| --- | --- |
| Roster status | Confirmed (client, 12 Oct 2026) |
| Legal manufacturer/supplier entity | Pending — client evidence not supplied |
| Supplier contact | Pending — client evidence not supplied |
| Dealer/stocking authorisation reference | Pending — client evidence not supplied |
| Authorisation evidence document | Pending — client evidence not supplied |
| Catalogue source | Pending — client evidence not supplied |
| Catalogue verification status | Unverified — no verified catalogue supplied (A-03) |
| Approval owner | Pending — client evidence not supplied (approval role: OQ-2) |
| Verification date | Pending — client evidence not supplied |
| Allowed purchase/sales status | **Not allowed** — blocked until the evidence above is recorded, reviewed and approved |

### 3.3 Ujagar Electric

| Field | Value |
| --- | --- |
| Roster status | Confirmed (client, 12 Oct 2026) |
| Legal manufacturer/supplier entity | Pending — client evidence not supplied |
| Relationship to Saffron EV Group | Unknown — **do not assume they are the same legal entity**; document required |
| Supplier contact | Pending — client evidence not supplied |
| Dealer/stocking authorisation reference | Pending — client evidence not supplied |
| Authorisation evidence document | Pending — client evidence not supplied |
| Catalogue source | Pending — client evidence not supplied |
| Catalogue verification status | Unverified — no verified catalogue supplied (A-03) |
| Approval owner | Pending — client evidence not supplied (approval role: OQ-2) |
| Verification date | Pending — client evidence not supplied |
| Allowed purchase/sales status | **Not allowed** — blocked until the evidence above is recorded, reviewed and approved |

## 4. What the application requires before a brand can be enabled

The brand's **Activation readiness** checklist (Catalogue & Brands → brand page; ADR-012, ADR-013) must show every
item met. Today, for all three brands:

| Requirement | Recorded where | Current state (all three brands) |
| --- | --- | --- |
| Authorisation status AUTHORISED | Brand → Record authorisation | Not met (UNVERIFIED) |
| Supplier/manufacturer legal name | Brand → Record authorisation | Not met |
| Authorisation reference, evidence document, verifier, date | Brand → Record authorisation | Not met |
| At least one verified, active model and variant | Catalogue models/variants (A-03, C-05) | Not met (no models) |
| Approved selling price valid now | Variant → Price history (C-04) | Not met (no prices) |
| Verified supplier relationship | Supplier master (D-01) | Not available yet |
| Approved warranty terms | Pending client data (RB-2) | Not available yet |
| Approved vehicle identifier rules | D-04, OQ-4 | Not available yet |

Even with all evidence supplied, activation also needs an owner-approved permission grant (RB-5); none exists.

## 5. Public references (not evidence)

The approved Research Brief (`REFERENCE_SPECIFICATIONS/00_Research_Brief_and_Decisions.md`, reviewed 10 Oct 2026)
notes public web content about these brands. It is **reference material only**: it does not establish authorisation,
legal identity, catalogue or prices, and must not be entered as evidence.

- The showroom's public website describes the business as an authorised Lectrix EV dealership — a public claim; the
  dealership document is still required.
- Lectrix EV and Ceeon India have official websites listing models; these are catalogue references, not a verified
  showroom catalogue or price list.
- Public research did not establish a reliable current Ujagar Electric catalogue or its relationship to Saffron EV Group.

## 6. How to complete A-01

For each brand, the client (or owner) supplies the documents above. Then:

1. Upload each document privately (Documents page) and record the brand's authorisation in the app (status, legal
   name, reference, evidence) — the app records the verifier and date and audits the change.
2. Fill in the tables in §3 here with the reference numbers, dates, approval owner and the document's location
   (never paste personal data beyond a business contact).
3. Record the Ujagar Electric / Saffron EV Group relationship from a document, or state that none exists.

A-01 is complete only when §3 has real evidence for every brand the showroom will operate at launch (or the client
removes a brand from the launch roster in writing).
