-- Supplier master (D-01; BR-002, Architecture §5 suppliers / brand_supplier_links).
-- Schema only: no supplier, contact, document or brand relationship is seeded.
--
-- A supplier record is not proof of dealership authorisation. A supplier starts
-- PENDING_VERIFICATION and becomes VERIFIED only with a reference, a supplier
-- document (private attachment) and a verifier. A supplier may serve several
-- brands; each supplier-brand relationship has its own evidence and verifier and
-- is ended (with a date), never deleted. Archive keeps every record as history.

INSERT INTO permissions (key, description, sensitive) VALUES
    ('suppliers.view',    'View suppliers, contacts, documents and brand relationships', true),
    ('suppliers.manage',  'Create and edit suppliers, contacts, documents and proposed brand relationships', true),
    ('suppliers.verify',  'Verify suppliers and supplier-brand relationships with evidence', true),
    ('suppliers.archive', 'Archive suppliers and end supplier-brand relationships', true);

CREATE TABLE suppliers (
    id                    uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    legal_name            text        NOT NULL,
    trade_name            text,
    -- GSTIN when applicable (format checked; the value comes from the supplier's registration).
    gstin                 text,
    address_line1         text,
    address_line2         text,
    city                  text,
    state                 text,
    postal_code           text,
    status                text        NOT NULL DEFAULT 'PENDING_VERIFICATION',
    verification_reference text,
    verification_document_id uuid     REFERENCES attachments (id) ON DELETE RESTRICT,
    verified_by           uuid        REFERENCES users (id) ON DELETE RESTRICT,
    verified_at           timestamptz,
    notes                 text,
    created_by            uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    version               integer     NOT NULL DEFAULT 1,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT suppliers_legal_name_length CHECK (length(btrim(legal_name)) > 0 AND length(legal_name) <= 200),
    CONSTRAINT suppliers_trade_name_length CHECK (trade_name IS NULL OR (length(btrim(trade_name)) > 0 AND length(trade_name) <= 200)),
    -- 15 characters: state code, PAN, entity number, 'Z', check character.
    CONSTRAINT suppliers_gstin_format CHECK (gstin IS NULL OR gstin ~ '^[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z][1-9A-Z]Z[0-9A-Z]$'),
    CONSTRAINT suppliers_postal_code_format CHECK (postal_code IS NULL OR postal_code ~ '^[1-9][0-9]{5}$'),
    CONSTRAINT suppliers_address_lengths CHECK (
        (address_line1 IS NULL OR length(address_line1) <= 200) AND (address_line2 IS NULL OR length(address_line2) <= 200)
        AND (city IS NULL OR length(city) <= 100) AND (state IS NULL OR length(state) <= 100)),
    CONSTRAINT suppliers_notes_length CHECK (notes IS NULL OR length(notes) <= 1000),
    CONSTRAINT suppliers_status_valid CHECK (status IN ('PENDING_VERIFICATION', 'VERIFIED', 'ARCHIVED')),
    CONSTRAINT suppliers_verified_pair CHECK ((verified_by IS NULL) = (verified_at IS NULL)),
    -- Verified needs the evidence: reference, document and verifier.
    CONSTRAINT suppliers_verified_has_evidence CHECK (status <> 'VERIFIED' OR (
        verification_reference IS NOT NULL AND verification_document_id IS NOT NULL AND verified_by IS NOT NULL)),
    CONSTRAINT suppliers_reference_length CHECK (verification_reference IS NULL OR (length(btrim(verification_reference)) > 0 AND length(verification_reference) <= 200)),
    CONSTRAINT suppliers_version_positive CHECK (version > 0)
);
CREATE UNIQUE INDEX suppliers_legal_name_key ON suppliers (lower(legal_name));
CREATE UNIQUE INDEX suppliers_gstin_key ON suppliers (gstin) WHERE gstin IS NOT NULL;
CREATE INDEX suppliers_status_idx ON suppliers (status);
CREATE TRIGGER suppliers_set_updated_at BEFORE UPDATE ON suppliers FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Business contacts. Removed contacts stay as history (active = false).
CREATE TABLE supplier_contacts (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    supplier_id uuid        NOT NULL REFERENCES suppliers (id) ON DELETE RESTRICT,
    name        text        NOT NULL,
    role        text,
    phone       text,
    email       text,
    is_primary  boolean     NOT NULL DEFAULT false,
    active      boolean     NOT NULL DEFAULT true,
    version     integer     NOT NULL DEFAULT 1,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT supplier_contacts_name_length CHECK (length(btrim(name)) > 0 AND length(name) <= 120),
    CONSTRAINT supplier_contacts_role_length CHECK (role IS NULL OR length(role) <= 80),
    CONSTRAINT supplier_contacts_phone_format CHECK (phone IS NULL OR phone ~ '^\+?[0-9][0-9 -]{5,18}[0-9]$'),
    CONSTRAINT supplier_contacts_email_format CHECK (email IS NULL OR (email ~ '^[^@\s]+@[^@\s]+\.[^@\s]+$' AND length(email) <= 254)),
    CONSTRAINT supplier_contacts_reachable CHECK (phone IS NOT NULL OR email IS NOT NULL),
    CONSTRAINT supplier_contacts_version_positive CHECK (version > 0)
);
CREATE INDEX supplier_contacts_supplier_idx ON supplier_contacts (supplier_id);
-- At most one active primary contact per supplier.
CREATE UNIQUE INDEX supplier_contacts_one_primary ON supplier_contacts (supplier_id) WHERE is_primary AND active;
CREATE TRIGGER supplier_contacts_set_updated_at BEFORE UPDATE ON supplier_contacts FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Supplier documents: private attachments linked to the supplier (B-09).
-- document_type is a proposed value set (OQ-19).
CREATE TABLE supplier_documents (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    supplier_id   uuid        NOT NULL REFERENCES suppliers (id) ON DELETE RESTRICT,
    attachment_id uuid        NOT NULL REFERENCES attachments (id) ON DELETE RESTRICT,
    document_type text        NOT NULL,
    note          text,
    added_by      uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_at    timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT supplier_documents_attachment_key UNIQUE (attachment_id),
    -- Target for "document of this supplier" foreign keys.
    CONSTRAINT supplier_documents_attachment_supplier_key UNIQUE (attachment_id, supplier_id),
    CONSTRAINT supplier_documents_type_valid CHECK (document_type IN ('GST_REGISTRATION', 'AUTHORISATION', 'AGREEMENT', 'OTHER')),
    CONSTRAINT supplier_documents_note_length CHECK (note IS NULL OR length(note) <= 300)
);
CREATE INDEX supplier_documents_supplier_idx ON supplier_documents (supplier_id);

-- The supplier's verification document must be one of its own documents.
ALTER TABLE suppliers ADD CONSTRAINT suppliers_verification_document_fk
    FOREIGN KEY (verification_document_id, id) REFERENCES supplier_documents (attachment_id, supplier_id) ON DELETE RESTRICT;

-- Supplier ↔ brand relationships (Architecture: brand_supplier_links).
CREATE TABLE supplier_brand_links (
    id                   uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    supplier_id          uuid        NOT NULL REFERENCES suppliers (id) ON DELETE RESTRICT,
    brand_id             uuid        NOT NULL REFERENCES brands (id) ON DELETE RESTRICT,
    status               text        NOT NULL DEFAULT 'UNVERIFIED',
    authorisation_reference text,
    evidence_document_id uuid,
    effective_from       date,
    effective_to         date,
    verified_by          uuid        REFERENCES users (id) ON DELETE RESTRICT,
    verified_at          timestamptz,
    note                 text,
    ended_reason         text,
    created_by           uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    version              integer     NOT NULL DEFAULT 1,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),

    -- Evidence must be a document of the same supplier.
    CONSTRAINT supplier_brand_links_evidence_fk FOREIGN KEY (evidence_document_id, supplier_id)
        REFERENCES supplier_documents (attachment_id, supplier_id) ON DELETE RESTRICT,
    CONSTRAINT supplier_brand_links_status_valid CHECK (status IN ('UNVERIFIED', 'VERIFIED', 'ENDED')),
    CONSTRAINT supplier_brand_links_verified_pair CHECK ((verified_by IS NULL) = (verified_at IS NULL)),
    CONSTRAINT supplier_brand_links_verified_has_evidence CHECK (status <> 'VERIFIED' OR (
        authorisation_reference IS NOT NULL AND evidence_document_id IS NOT NULL AND effective_from IS NOT NULL AND verified_by IS NOT NULL)),
    CONSTRAINT supplier_brand_links_period_valid CHECK (effective_to IS NULL OR effective_from IS NULL OR effective_to >= effective_from),
    CONSTRAINT supplier_brand_links_reference_length CHECK (authorisation_reference IS NULL OR (length(btrim(authorisation_reference)) > 0 AND length(authorisation_reference) <= 200)),
    CONSTRAINT supplier_brand_links_note_length CHECK ((note IS NULL OR length(note) <= 300) AND (ended_reason IS NULL OR length(ended_reason) <= 300)),
    CONSTRAINT supplier_brand_links_version_positive CHECK (version > 0)
);
-- One open (not ended) relationship per supplier and brand; ended ones remain as history.
CREATE UNIQUE INDEX supplier_brand_links_open_key ON supplier_brand_links (supplier_id, brand_id) WHERE status <> 'ENDED';
CREATE INDEX supplier_brand_links_brand_idx ON supplier_brand_links (brand_id, status);
CREATE TRIGGER supplier_brand_links_set_updated_at BEFORE UPDATE ON supplier_brand_links FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Nothing here is ever deleted: suppliers are archived, contacts deactivated,
-- relationships ended. Ended relationships and verified evidence are history.
CREATE FUNCTION supplier_history_guard() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION '% rows are never deleted; archive, deactivate or end them instead', TG_TABLE_NAME
            USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF TG_TABLE_NAME = 'supplier_brand_links' AND OLD.status = 'ENDED' THEN
        RAISE EXCEPTION 'ended supplier-brand relationships cannot change' USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF TG_TABLE_NAME = 'supplier_documents' THEN
        RAISE EXCEPTION 'supplier documents cannot change' USING ERRCODE = 'insufficient_privilege';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER suppliers_no_delete BEFORE DELETE ON suppliers FOR EACH ROW EXECUTE FUNCTION supplier_history_guard();
CREATE TRIGGER supplier_contacts_no_delete BEFORE DELETE ON supplier_contacts FOR EACH ROW EXECUTE FUNCTION supplier_history_guard();
CREATE TRIGGER supplier_documents_immutable BEFORE UPDATE OR DELETE ON supplier_documents FOR EACH ROW EXECUTE FUNCTION supplier_history_guard();
CREATE TRIGGER supplier_brand_links_guard BEFORE UPDATE OR DELETE ON supplier_brand_links FOR EACH ROW EXECUTE FUNCTION supplier_history_guard();
