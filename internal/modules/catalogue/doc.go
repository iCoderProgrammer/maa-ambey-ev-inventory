// Package catalogue will hold the brand → model → variant catalogue service
// (C-02). C-01 delivers only the schema (migration 000007) and its tests:
//
//   - brands: lifecycle (PENDING_VERIFICATION / ACTIVE / ARCHIVED) separate from
//     authorisation (UNVERIFIED / AUTHORISED / NOT_AUTHORISED / REVOKED);
//     AUTHORISED requires manufacturer legal name, reference, verifier and date,
//     and may point at a private evidence attachment (BR-001, BR-002).
//   - models → brand, variants → model, variant_colours and variant_specs →
//     variant. Each child carries brand_id, kept consistent by composite
//     foreign keys, so a variant cannot sit under another brand's model and
//     codes are unique within their brand scope (BR-003).
//   - Every catalogue row records provenance: confidence UNVERIFIED /
//     SUPPLIER_CONFIRMED / CLIENT_APPROVED, source URL or attachment, source
//     date, verifier (BR-004). Verified spec rows are immutable; changes are new
//     rows with a later effective_from (BR-005).
//   - Foreign keys are ON DELETE RESTRICT: a brand with models, or a variant
//     with specs, cannot be deleted; archive instead (BR-006).
//
// Brands are data: no brand, model or specification appears in code, and no
// catalogue data is seeded (C-05 loads verified data only).
package catalogue
