-- Rolls back C-01. DESTRUCTIVE for catalogue data: drops models, variants,
-- colours and specification history, and the brand authorisation-evidence
-- columns. cmd/migrate refuses a down migration in staging/production without
-- --confirm; take a backup first. Brands themselves are kept.
DROP TABLE IF EXISTS variant_specs;
DROP FUNCTION IF EXISTS variant_specs_protect_verified();
DROP TABLE IF EXISTS variant_colours;
DROP TABLE IF EXISTS variants;
DROP TABLE IF EXISTS models;
ALTER TABLE brands
    DROP CONSTRAINT IF EXISTS brands_verified_pair,
    DROP CONSTRAINT IF EXISTS brands_authorised_has_evidence,
    DROP CONSTRAINT IF EXISTS brands_authorisation_reference_length,
    DROP CONSTRAINT IF EXISTS brands_manufacturer_legal_name_length,
    DROP COLUMN IF EXISTS verified_by,
    DROP COLUMN IF EXISTS authorisation_evidence_id,
    DROP COLUMN IF EXISTS manufacturer_legal_name;
