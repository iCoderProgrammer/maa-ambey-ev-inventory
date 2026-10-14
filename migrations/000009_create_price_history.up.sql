-- Effective-dated price history (C-04; BR-005, SAL-003, DOC-003, SEC-003).
-- Schema only: no price list or price is created. Prices come from approved
-- client/supplier sources entered by authorised users.
--
-- A price list belongs to one brand; each price item belongs to the list's
-- brand and names the model and variant it prices (composite foreign keys keep
-- brand/model/variant consistent). Items are drafted, then approved by a
-- different user. Approved rows are immutable: the only change allowed later is
-- controlled supersession, which ends an approved period at the start of its
-- approved successor and links the two. Approved periods for the same list,
-- variant and price type never overlap (exclusion constraint), so the price at
-- any instant is unambiguous and concurrent approvals cannot both succeed.

-- btree_gist lets the exclusion constraint combine equality on IDs with range
-- overlap. It is a trusted extension: the database owner may create it.
CREATE EXTENSION IF NOT EXISTS btree_gist;

-- Catalogue keys only: no role is granted them (RB-5, OQ-2). Cost prices keep
-- using catalogue.cost_price.view / catalogue.cost_price.edit (migration 000004).
INSERT INTO permissions (key, description, sensitive) VALUES
    ('catalogue.price.view',    'View selling price lists and price history',          false),
    ('catalogue.price.manage',  'Create price lists and draft selling prices',          true),
    ('catalogue.price.approve', 'Approve drafted prices (never one''s own draft)',      true);

-- Variants: allow price items to reference (variant, model) together.
ALTER TABLE variants ADD CONSTRAINT variants_id_model_key UNIQUE (id, model_id);

CREATE TABLE price_lists (
    id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id         uuid        NOT NULL REFERENCES brands (id) ON DELETE RESTRICT,
    name             text        NOT NULL,
    -- Amounts are Indian rupees; another currency would need its own decision.
    currency         text        NOT NULL DEFAULT 'INR',
    lifecycle_status text        NOT NULL DEFAULT 'ACTIVE',
    created_by       uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    version          integer     NOT NULL DEFAULT 1,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT price_lists_id_brand_key UNIQUE (id, brand_id),
    CONSTRAINT price_lists_name_length CHECK (length(btrim(name)) > 0 AND length(name) <= 120),
    CONSTRAINT price_lists_currency_valid CHECK (currency = 'INR'),
    CONSTRAINT price_lists_lifecycle_valid CHECK (lifecycle_status IN ('ACTIVE', 'ARCHIVED')),
    CONSTRAINT price_lists_version_positive CHECK (version > 0)
);
CREATE UNIQUE INDEX price_lists_brand_name_key ON price_lists (brand_id, lower(name));
CREATE TRIGGER price_lists_set_updated_at BEFORE UPDATE ON price_lists FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE price_items (
    id                   uuid          PRIMARY KEY DEFAULT gen_random_uuid(),
    price_list_id        uuid          NOT NULL,
    brand_id             uuid          NOT NULL,
    model_id             uuid          NOT NULL,
    variant_id           uuid          NOT NULL,
    -- Proposed value set pending owner/accountant sign-off (OQ-15).
    price_type           text          NOT NULL,
    amount               numeric(12,2) NOT NULL,
    currency             text          NOT NULL DEFAULT 'INR',
    effective_from       timestamptz   NOT NULL,
    -- Exclusive end; NULL = open-ended.
    effective_to         timestamptz,
    status               text          NOT NULL DEFAULT 'DRAFT',
    -- Approved tax configuration (DOC-003). No tax configuration exists yet
    -- (F-10, accountant sign-off), so it must stay NULL; a later migration adds
    -- the foreign key and drops this restriction.
    tax_configuration_id uuid,
    source_note          text,
    created_by           uuid          NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    approved_by          uuid          REFERENCES users (id) ON DELETE RESTRICT,
    approved_at          timestamptz,
    approval_reference   text,
    -- Controlled supersession: the draft names the approved item it replaces;
    -- on approval the predecessor gets superseded_by_id and its end date.
    supersedes_id        uuid          REFERENCES price_items (id) ON DELETE RESTRICT,
    superseded_by_id     uuid          REFERENCES price_items (id) ON DELETE RESTRICT,
    version              integer       NOT NULL DEFAULT 1,
    created_at           timestamptz   NOT NULL DEFAULT now(),
    updated_at           timestamptz   NOT NULL DEFAULT now(),

    CONSTRAINT price_items_list_fk FOREIGN KEY (price_list_id, brand_id) REFERENCES price_lists (id, brand_id) ON DELETE RESTRICT,
    CONSTRAINT price_items_model_fk FOREIGN KEY (model_id, brand_id) REFERENCES models (id, brand_id) ON DELETE RESTRICT,
    CONSTRAINT price_items_variant_brand_fk FOREIGN KEY (variant_id, brand_id) REFERENCES variants (id, brand_id) ON DELETE RESTRICT,
    CONSTRAINT price_items_variant_model_fk FOREIGN KEY (variant_id, model_id) REFERENCES variants (id, model_id) ON DELETE RESTRICT,
    CONSTRAINT price_items_type_valid CHECK (price_type IN ('SELLING', 'COST')),
    CONSTRAINT price_items_amount_positive CHECK (amount > 0),
    CONSTRAINT price_items_currency_valid CHECK (currency = 'INR'),
    CONSTRAINT price_items_period_valid CHECK (effective_to IS NULL OR effective_to > effective_from),
    CONSTRAINT price_items_status_valid CHECK (status IN ('DRAFT', 'APPROVED', 'CANCELLED')),
    CONSTRAINT price_items_tax_configuration_pending CHECK (tax_configuration_id IS NULL),
    CONSTRAINT price_items_source_note_length CHECK (source_note IS NULL OR (length(btrim(source_note)) > 0 AND length(source_note) <= 500)),
    CONSTRAINT price_items_approval_reference_length CHECK (approval_reference IS NULL OR (length(btrim(approval_reference)) > 0 AND length(approval_reference) <= 200)),
    -- Approved rows record who approved them and when; never their own drafter.
    CONSTRAINT price_items_approved_has_approver CHECK ((status = 'APPROVED') = (approved_by IS NOT NULL AND approved_at IS NOT NULL)),
    CONSTRAINT price_items_not_self_approved CHECK (approved_by IS NULL OR approved_by <> created_by),
    CONSTRAINT price_items_supersedes_other CHECK (supersedes_id IS NULL OR supersedes_id <> id),
    CONSTRAINT price_items_version_positive CHECK (version > 0),
    -- One approved price per list/variant/type at any instant.
    CONSTRAINT price_items_no_overlap EXCLUDE USING gist (
        price_list_id WITH =, variant_id WITH =, price_type WITH =,
        tstzrange(effective_from, effective_to, '[)') WITH &&
    ) WHERE (status = 'APPROVED')
);
CREATE INDEX price_items_lookup_idx ON price_items (variant_id, price_type, effective_from) WHERE status = 'APPROVED';
CREATE INDEX price_items_list_idx ON price_items (price_list_id, status);
CREATE INDEX price_items_brand_idx ON price_items (brand_id);
-- A draft may supersede an approved item; only one approved successor per item.
CREATE UNIQUE INDEX price_items_superseded_by_key ON price_items (superseded_by_id) WHERE superseded_by_id IS NOT NULL;
CREATE TRIGGER price_items_set_updated_at BEFORE UPDATE ON price_items FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Approved and cancelled rows are history. An approved row may change only by
-- supersession: superseded_by_id set once and effective_to moved earlier (never
-- later, never before effective_from). Nothing is ever deleted.
CREATE FUNCTION price_items_protect_history() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'price items are never deleted; cancel a draft or supersede an approved price'
            USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF OLD.status = 'CANCELLED' THEN
        RAISE EXCEPTION 'cancelled price items cannot change' USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF OLD.status = 'APPROVED' THEN
        IF NEW.status <> 'APPROVED'
           OR NEW.price_list_id IS DISTINCT FROM OLD.price_list_id
           OR NEW.brand_id IS DISTINCT FROM OLD.brand_id
           OR NEW.model_id IS DISTINCT FROM OLD.model_id
           OR NEW.variant_id IS DISTINCT FROM OLD.variant_id
           OR NEW.price_type IS DISTINCT FROM OLD.price_type
           OR NEW.amount IS DISTINCT FROM OLD.amount
           OR NEW.currency IS DISTINCT FROM OLD.currency
           OR NEW.effective_from IS DISTINCT FROM OLD.effective_from
           OR NEW.tax_configuration_id IS DISTINCT FROM OLD.tax_configuration_id
           OR NEW.source_note IS DISTINCT FROM OLD.source_note
           OR NEW.created_by IS DISTINCT FROM OLD.created_by
           OR NEW.approved_by IS DISTINCT FROM OLD.approved_by
           OR NEW.approved_at IS DISTINCT FROM OLD.approved_at
           OR NEW.approval_reference IS DISTINCT FROM OLD.approval_reference
           OR NEW.supersedes_id IS DISTINCT FROM OLD.supersedes_id
           OR OLD.superseded_by_id IS NOT NULL
           OR NEW.superseded_by_id IS NULL
           OR (NEW.effective_to IS NULL AND OLD.effective_to IS NOT NULL)
           OR (OLD.effective_to IS NOT NULL AND NEW.effective_to > OLD.effective_to)
        THEN
            RAISE EXCEPTION 'approved prices are immutable; record a superseding price instead (% refused)', TG_OP
                USING ERRCODE = 'insufficient_privilege';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER price_items_protect_history
    BEFORE UPDATE OR DELETE ON price_items
    FOR EACH ROW EXECUTE FUNCTION price_items_protect_history();
