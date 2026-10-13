-- Brand → model → variant catalogue schema (C-01; BR-001…BR-008, Architecture §5).
-- Schema only: no catalogue rows are created. Models, variants, colours and
-- specifications come from client/supplier-verified sources later (C-05).
--
-- Brand context: every child row carries brand_id, kept consistent with its
-- parent by composite foreign keys, so a variant can never point at a model of
-- another brand and codes can be unique within a brand. Internal IDs are
-- globally unique UUIDs. Adding a brand is an INSERT; nothing here names one.
--
-- Provenance (BR-004): models, variants, colours and spec values each record
-- confidence UNVERIFIED / SUPPLIER_CONFIRMED / CLIENT_APPROVED, the source
-- (URL and/or private attachment), source date, verifier and verification
-- time. Anything above UNVERIFIED must say where it came from and who checked it.
--
-- Rules that need application context (only ACTIVE + authorised brands may be
-- used for purchase/sales, a variant cannot be ACTIVE under an inactive model,
-- who may approve) are enforced by the catalogue service (C-02), so pending and
-- archived history stays representable (ADR-006, ADR-011).

-- ── Brands: authorisation evidence (BR-002) ─────────────────────────────────
ALTER TABLE brands
    -- Supplier/manufacturer legal identity as verified (not assumed from the brand name).
    ADD COLUMN manufacturer_legal_name text,
    -- Private document proving the authorisation (B-09 attachment).
    ADD COLUMN authorisation_evidence_id uuid REFERENCES attachments (id) ON DELETE RESTRICT,
    -- Who verified the authorisation (approval owner role is pending, OQ-2).
    ADD COLUMN verified_by uuid REFERENCES users (id) ON DELETE RESTRICT,
    ADD CONSTRAINT brands_manufacturer_legal_name_length
        CHECK (manufacturer_legal_name IS NULL OR (length(btrim(manufacturer_legal_name)) > 0 AND length(manufacturer_legal_name) <= 200)),
    ADD CONSTRAINT brands_authorisation_reference_length
        CHECK (authorisation_reference IS NULL OR (length(btrim(authorisation_reference)) > 0 AND length(authorisation_reference) <= 200)),
    -- AUTHORISED needs the evidence BR-002 lists: legal identity, reference, verifier and date.
    ADD CONSTRAINT brands_authorised_has_evidence
        CHECK (authorisation_status <> 'AUTHORISED' OR (
            manufacturer_legal_name IS NOT NULL AND authorisation_reference IS NOT NULL
            AND verified_by IS NOT NULL AND verified_at IS NOT NULL)),
    ADD CONSTRAINT brands_verified_pair CHECK ((verified_by IS NULL) = (verified_at IS NULL));

-- ── Models (BR-003) ──────────────────────────────────────────────────────────
CREATE TABLE models (
    id                 uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id           uuid        NOT NULL REFERENCES brands (id) ON DELETE RESTRICT,
    name               text        NOT NULL,
    -- Internal code, unique within the brand (case-insensitive, stored upper-case).
    internal_code      text        NOT NULL,
    -- Manufacturer's own code, when the supplier documents one; unique within the brand.
    manufacturer_code  text,
    -- Free classification until the client confirms a value set (OQ-11).
    vehicle_type       text,
    lifecycle_status   text        NOT NULL DEFAULT 'PENDING_VERIFICATION',
    confidence         text        NOT NULL DEFAULT 'UNVERIFIED',
    source_url         text,
    source_document_id uuid        REFERENCES attachments (id) ON DELETE RESTRICT,
    source_date        date,
    verified_by        uuid        REFERENCES users (id) ON DELETE RESTRICT,
    verified_at        timestamptz,
    version            integer     NOT NULL DEFAULT 1,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT models_id_brand_key UNIQUE (id, brand_id),
    CONSTRAINT models_name_length CHECK (length(btrim(name)) > 0 AND length(name) <= 120),
    CONSTRAINT models_internal_code_format CHECK (internal_code ~ '^[A-Z0-9][A-Z0-9._-]{0,39}$'),
    CONSTRAINT models_manufacturer_code_length CHECK (manufacturer_code IS NULL OR (length(btrim(manufacturer_code)) > 0 AND length(manufacturer_code) <= 60)),
    CONSTRAINT models_vehicle_type_format CHECK (vehicle_type IS NULL OR vehicle_type ~ '^[A-Z][A-Z0-9_]{0,39}$'),
    CONSTRAINT models_lifecycle_status_valid CHECK (lifecycle_status IN ('PENDING_VERIFICATION', 'ACTIVE', 'ARCHIVED')),
    CONSTRAINT models_confidence_valid CHECK (confidence IN ('UNVERIFIED', 'SUPPLIER_CONFIRMED', 'CLIENT_APPROVED')),
    CONSTRAINT models_source_url_format CHECK (source_url IS NULL OR (source_url ~ '^https?://' AND length(source_url) <= 2000)),
    CONSTRAINT models_verified_pair CHECK ((verified_by IS NULL) = (verified_at IS NULL)),
    CONSTRAINT models_verified_has_provenance CHECK (confidence = 'UNVERIFIED' OR (
        (source_url IS NOT NULL OR source_document_id IS NOT NULL) AND source_date IS NOT NULL AND verified_by IS NOT NULL)),
    CONSTRAINT models_version_positive CHECK (version > 0)
);
CREATE UNIQUE INDEX models_brand_internal_code_key ON models (brand_id, internal_code);
CREATE UNIQUE INDEX models_brand_manufacturer_code_key ON models (brand_id, upper(manufacturer_code)) WHERE manufacturer_code IS NOT NULL;
CREATE UNIQUE INDEX models_brand_name_key ON models (brand_id, lower(name));
CREATE INDEX models_brand_status_idx ON models (brand_id, lifecycle_status);
CREATE TRIGGER models_set_updated_at BEFORE UPDATE ON models FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── Variants (BR-003) ────────────────────────────────────────────────────────
CREATE TABLE variants (
    id                 uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id           uuid        NOT NULL,
    model_id           uuid        NOT NULL,
    name               text        NOT NULL,
    -- Unique within the model (variant codes such as "STD" repeat across models).
    variant_code       text        NOT NULL,
    -- Manufacturer SKU/variant code, unique within the brand when documented.
    manufacturer_code  text,
    lifecycle_status   text        NOT NULL DEFAULT 'PENDING_VERIFICATION',
    confidence         text        NOT NULL DEFAULT 'UNVERIFIED',
    source_url         text,
    source_document_id uuid        REFERENCES attachments (id) ON DELETE RESTRICT,
    source_date        date,
    verified_by        uuid        REFERENCES users (id) ON DELETE RESTRICT,
    verified_at        timestamptz,
    version            integer     NOT NULL DEFAULT 1,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT variants_model_fk FOREIGN KEY (model_id, brand_id) REFERENCES models (id, brand_id) ON DELETE RESTRICT,
    CONSTRAINT variants_id_brand_key UNIQUE (id, brand_id),
    CONSTRAINT variants_name_length CHECK (length(btrim(name)) > 0 AND length(name) <= 120),
    CONSTRAINT variants_code_format CHECK (variant_code ~ '^[A-Z0-9][A-Z0-9._-]{0,39}$'),
    CONSTRAINT variants_manufacturer_code_length CHECK (manufacturer_code IS NULL OR (length(btrim(manufacturer_code)) > 0 AND length(manufacturer_code) <= 60)),
    CONSTRAINT variants_lifecycle_status_valid CHECK (lifecycle_status IN ('PENDING_VERIFICATION', 'ACTIVE', 'ARCHIVED')),
    CONSTRAINT variants_confidence_valid CHECK (confidence IN ('UNVERIFIED', 'SUPPLIER_CONFIRMED', 'CLIENT_APPROVED')),
    CONSTRAINT variants_source_url_format CHECK (source_url IS NULL OR (source_url ~ '^https?://' AND length(source_url) <= 2000)),
    CONSTRAINT variants_verified_pair CHECK ((verified_by IS NULL) = (verified_at IS NULL)),
    CONSTRAINT variants_verified_has_provenance CHECK (confidence = 'UNVERIFIED' OR (
        (source_url IS NOT NULL OR source_document_id IS NOT NULL) AND source_date IS NOT NULL AND verified_by IS NOT NULL)),
    CONSTRAINT variants_version_positive CHECK (version > 0)
);
CREATE UNIQUE INDEX variants_model_code_key ON variants (model_id, variant_code);
CREATE UNIQUE INDEX variants_brand_manufacturer_code_key ON variants (brand_id, upper(manufacturer_code)) WHERE manufacturer_code IS NOT NULL;
CREATE UNIQUE INDEX variants_model_name_key ON variants (model_id, lower(name));
CREATE INDEX variants_brand_status_idx ON variants (brand_id, lifecycle_status);
CREATE TRIGGER variants_set_updated_at BEFORE UPDATE ON variants FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── Colours per variant ──────────────────────────────────────────────────────
CREATE TABLE variant_colours (
    id                 uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id           uuid        NOT NULL,
    variant_id         uuid        NOT NULL,
    name               text        NOT NULL,
    -- Manufacturer colour code, when documented.
    colour_code        text,
    lifecycle_status   text        NOT NULL DEFAULT 'PENDING_VERIFICATION',
    confidence         text        NOT NULL DEFAULT 'UNVERIFIED',
    source_url         text,
    source_document_id uuid        REFERENCES attachments (id) ON DELETE RESTRICT,
    source_date        date,
    verified_by        uuid        REFERENCES users (id) ON DELETE RESTRICT,
    verified_at        timestamptz,
    version            integer     NOT NULL DEFAULT 1,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT variant_colours_variant_fk FOREIGN KEY (variant_id, brand_id) REFERENCES variants (id, brand_id) ON DELETE RESTRICT,
    CONSTRAINT variant_colours_name_length CHECK (length(btrim(name)) > 0 AND length(name) <= 80),
    CONSTRAINT variant_colours_code_length CHECK (colour_code IS NULL OR (length(btrim(colour_code)) > 0 AND length(colour_code) <= 40)),
    CONSTRAINT variant_colours_lifecycle_status_valid CHECK (lifecycle_status IN ('PENDING_VERIFICATION', 'ACTIVE', 'ARCHIVED')),
    CONSTRAINT variant_colours_confidence_valid CHECK (confidence IN ('UNVERIFIED', 'SUPPLIER_CONFIRMED', 'CLIENT_APPROVED')),
    CONSTRAINT variant_colours_source_url_format CHECK (source_url IS NULL OR (source_url ~ '^https?://' AND length(source_url) <= 2000)),
    CONSTRAINT variant_colours_verified_pair CHECK ((verified_by IS NULL) = (verified_at IS NULL)),
    CONSTRAINT variant_colours_verified_has_provenance CHECK (confidence = 'UNVERIFIED' OR (
        (source_url IS NOT NULL OR source_document_id IS NOT NULL) AND source_date IS NOT NULL AND verified_by IS NOT NULL)),
    CONSTRAINT variant_colours_version_positive CHECK (version > 0)
);
CREATE UNIQUE INDEX variant_colours_variant_name_key ON variant_colours (variant_id, lower(name));
CREATE UNIQUE INDEX variant_colours_variant_code_key ON variant_colours (variant_id, upper(colour_code)) WHERE colour_code IS NOT NULL;
CREATE INDEX variant_colours_brand_idx ON variant_colours (brand_id);
CREATE TRIGGER variant_colours_set_updated_at BEFORE UPDATE ON variant_colours FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ── Specification values, versioned (BR-004, BR-005) ─────────────────────────
-- One row per (variant, attribute, effective date). Keys are data, e.g.
-- battery_type, battery_capacity_kwh, motor_power_w — no key or value is
-- seeded. A row that reached SUPPLIER_CONFIRMED or CLIENT_APPROVED is a
-- verified snapshot and is never edited or deleted: changes are new rows with a
-- later effective_from (BR-005).
CREATE TABLE variant_specs (
    id                 uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id           uuid        NOT NULL,
    variant_id         uuid        NOT NULL,
    spec_key           text        NOT NULL,
    value_text         text,
    value_numeric      numeric(14, 4),
    unit               text,
    effective_from     date        NOT NULL DEFAULT current_date,
    confidence         text        NOT NULL DEFAULT 'UNVERIFIED',
    source_url         text,
    source_document_id uuid        REFERENCES attachments (id) ON DELETE RESTRICT,
    source_date        date,
    verified_by        uuid        REFERENCES users (id) ON DELETE RESTRICT,
    verified_at        timestamptz,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT variant_specs_variant_fk FOREIGN KEY (variant_id, brand_id) REFERENCES variants (id, brand_id) ON DELETE RESTRICT,
    CONSTRAINT variant_specs_key_format CHECK (spec_key ~ '^[a-z][a-z0-9_]{0,59}$'),
    CONSTRAINT variant_specs_one_value CHECK ((value_text IS NULL) <> (value_numeric IS NULL)),
    CONSTRAINT variant_specs_value_text_length CHECK (value_text IS NULL OR (length(btrim(value_text)) > 0 AND length(value_text) <= 500)),
    CONSTRAINT variant_specs_unit_length CHECK (unit IS NULL OR (length(btrim(unit)) > 0 AND length(unit) <= 20)),
    CONSTRAINT variant_specs_confidence_valid CHECK (confidence IN ('UNVERIFIED', 'SUPPLIER_CONFIRMED', 'CLIENT_APPROVED')),
    CONSTRAINT variant_specs_source_url_format CHECK (source_url IS NULL OR (source_url ~ '^https?://' AND length(source_url) <= 2000)),
    CONSTRAINT variant_specs_verified_pair CHECK ((verified_by IS NULL) = (verified_at IS NULL)),
    CONSTRAINT variant_specs_verified_has_provenance CHECK (confidence = 'UNVERIFIED' OR (
        (source_url IS NOT NULL OR source_document_id IS NOT NULL) AND source_date IS NOT NULL AND verified_by IS NOT NULL))
);
CREATE UNIQUE INDEX variant_specs_variant_key_effective_key ON variant_specs (variant_id, spec_key, effective_from);
CREATE INDEX variant_specs_brand_idx ON variant_specs (brand_id);
CREATE TRIGGER variant_specs_set_updated_at BEFORE UPDATE ON variant_specs FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE FUNCTION variant_specs_protect_verified() RETURNS trigger AS $$
BEGIN
    IF OLD.confidence <> 'UNVERIFIED' THEN
        RAISE EXCEPTION 'verified specification rows are immutable; record a new row with a later effective_from (% refused)', TG_OP
            USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER variant_specs_protect_verified
    BEFORE UPDATE OR DELETE ON variant_specs
    FOR EACH ROW EXECUTE FUNCTION variant_specs_protect_verified();
