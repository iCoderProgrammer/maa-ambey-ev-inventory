-- Brand master. Multi-brand support is a P0 requirement: brands are data,
-- never code constants. A brand may exist in the catalogue without being
-- authorised for purchase/sales (BR-001, BR-002, BR-008).

CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE brands (
    id                      uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    name                    text        NOT NULL,
    slug                    text        NOT NULL,
    -- Catalogue lifecycle (BR-001). New brands always start unverified.
    lifecycle_status        text        NOT NULL DEFAULT 'PENDING_VERIFICATION',
    -- Dealership authorisation is tracked separately from lifecycle (BR-002).
    -- Value set is a proposal pending client sign-off; see docs/ARCHITECTURE_DECISIONS.md.
    authorisation_status    text        NOT NULL DEFAULT 'UNVERIFIED',
    authorisation_reference text,
    verified_at             timestamptz,
    -- Optimistic concurrency token for future edits.
    version                 integer     NOT NULL DEFAULT 1,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT brands_name_not_blank CHECK (length(btrim(name)) > 0),
    CONSTRAINT brands_name_length CHECK (length(name) <= 120),
    CONSTRAINT brands_slug_format CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND length(slug) <= 80),
    CONSTRAINT brands_lifecycle_status_valid
        CHECK (lifecycle_status IN ('PENDING_VERIFICATION', 'ACTIVE', 'ARCHIVED')),
    CONSTRAINT brands_authorisation_status_valid
        CHECK (authorisation_status IN ('UNVERIFIED', 'AUTHORISED', 'NOT_AUTHORISED', 'REVOKED')),
    CONSTRAINT brands_version_positive CHECK (version > 0)
);

CREATE UNIQUE INDEX brands_slug_key ON brands (slug);
CREATE UNIQUE INDEX brands_name_lower_key ON brands (lower(name));
CREATE INDEX brands_lifecycle_status_idx ON brands (lifecycle_status);

CREATE TRIGGER brands_set_updated_at
    BEFORE UPDATE ON brands
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
