-- E-03: spare part and accessory SKU master (ADR-021; FR-015, PRD "Spare/accessory
-- item"). Item categories and SKUs with unit of measure, tracking mode
-- (QUANTITY or SERIAL), supplier links, reorder level and optional brand/model
-- compatibility. This is master data only: no quantity, balance, serial
-- instance or stock movement is created here (stock in/out is E-04).
--
-- Nothing is deleted: categories and SKUs are archived (and an archived row no
-- longer changes), supplier links are ended. No category, SKU, unit of measure,
-- supplier link or compatibility is seeded or invented.
--
-- Pending (ADR-021): an approved unit-of-measure list (OQ-34), batch/lot tracking,
-- cost and selling price of parts (approval like C-04), per-location stock
-- (E-04), reactivating archived SKUs, correcting a SKU code (OQ-35).

INSERT INTO permissions (key, description, sensitive) VALUES
    ('stock_items.view',    'View spare part and accessory SKUs and item categories', false),
    ('stock_items.manage',  'Create and edit spare part and accessory SKUs and item categories, their supplier links and compatibility', true),
    ('stock_items.archive', 'Archive spare part and accessory SKUs and item categories', true);

-- Categories are data: the business adds its own; none is predefined.
CREATE TABLE stock_item_categories (
    id             uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    name           text        NOT NULL,
    description    text,
    status         text        NOT NULL DEFAULT 'ACTIVE',
    archived_by    uuid        REFERENCES users (id) ON DELETE RESTRICT,
    archived_at    timestamptz,
    archive_reason text,
    created_by     uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    version        integer     NOT NULL DEFAULT 1,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT stock_item_categories_name_length CHECK (length(btrim(name)) > 0 AND length(name) <= 100 AND name = btrim(name)),
    CONSTRAINT stock_item_categories_description_length CHECK (description IS NULL OR (length(btrim(description)) > 0 AND length(description) <= 500)),
    CONSTRAINT stock_item_categories_status_valid CHECK (status IN ('ACTIVE', 'ARCHIVED')),
    CONSTRAINT stock_item_categories_archived_complete CHECK ((status = 'ARCHIVED') = (archived_by IS NOT NULL AND archived_at IS NOT NULL AND archive_reason IS NOT NULL)),
    CONSTRAINT stock_item_categories_reason_length CHECK (archive_reason IS NULL OR (length(btrim(archive_reason)) > 0 AND length(archive_reason) <= 300)),
    CONSTRAINT stock_item_categories_version_positive CHECK (version > 0)
);
-- One category per name, ignoring case (archived ones keep their name).
CREATE UNIQUE INDEX stock_item_categories_name_key ON stock_item_categories (lower(name));
CREATE INDEX stock_item_categories_status_idx ON stock_item_categories (status);

CREATE TABLE stock_items (
    id              uuid          PRIMARY KEY DEFAULT gen_random_uuid(),
    -- The SKU code as entered (trimmed) and its comparison form: upper-cased.
    -- Codes are letters, digits and . _ / - only (no spaces), so "abc-1" and
    -- " ABC-1 " are the same SKU and the database decides when two requests race.
    sku             text          NOT NULL,
    sku_normalised  text          GENERATED ALWAYS AS (upper(sku)) STORED,
    name            text          NOT NULL,
    description     text,
    category_id     uuid          NOT NULL REFERENCES stock_item_categories (id) ON DELETE RESTRICT,
    -- Unit of measure code (e.g. what the supplier's paperwork uses). No approved
    -- list exists yet (OQ-34): stored upper-cased, letters and digits only.
    uom             text          NOT NULL,
    -- QUANTITY: stock is counted in the unit of measure. SERIAL: each piece is a
    -- separately identified instance (registered by stock operations, E-04).
    -- Fixed at creation: future stock records depend on it.
    tracking_mode   text          NOT NULL,
    -- Reorder threshold in the unit of measure (configuration only; nothing
    -- compares it with stock here). SERIAL items count whole pieces.
    reorder_level   numeric(12,3),
    status          text          NOT NULL DEFAULT 'ACTIVE',
    archived_by     uuid          REFERENCES users (id) ON DELETE RESTRICT,
    archived_at     timestamptz,
    archive_reason  text,
    created_by      uuid          NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    version         integer       NOT NULL DEFAULT 1,
    created_at      timestamptz   NOT NULL DEFAULT now(),
    updated_at      timestamptz   NOT NULL DEFAULT now(),

    CONSTRAINT stock_items_sku_format CHECK (sku ~ '^[A-Za-z0-9][A-Za-z0-9._/-]{0,39}$'),
    CONSTRAINT stock_items_name_length CHECK (length(btrim(name)) > 0 AND length(name) <= 200 AND name = btrim(name)),
    CONSTRAINT stock_items_description_length CHECK (description IS NULL OR (length(btrim(description)) > 0 AND length(description) <= 1000)),
    CONSTRAINT stock_items_uom_format CHECK (uom ~ '^[A-Z][A-Z0-9]{0,9}$'),
    CONSTRAINT stock_items_tracking_mode_valid CHECK (tracking_mode IN ('QUANTITY', 'SERIAL')),
    CONSTRAINT stock_items_reorder_level_valid CHECK (reorder_level IS NULL OR reorder_level >= 0),
    CONSTRAINT stock_items_serial_reorder_whole CHECK (tracking_mode <> 'SERIAL' OR reorder_level IS NULL OR reorder_level = trunc(reorder_level)),
    CONSTRAINT stock_items_status_valid CHECK (status IN ('ACTIVE', 'ARCHIVED')),
    CONSTRAINT stock_items_archived_complete CHECK ((status = 'ARCHIVED') = (archived_by IS NOT NULL AND archived_at IS NOT NULL AND archive_reason IS NOT NULL)),
    CONSTRAINT stock_items_reason_length CHECK (archive_reason IS NULL OR (length(btrim(archive_reason)) > 0 AND length(archive_reason) <= 300)),
    CONSTRAINT stock_items_version_positive CHECK (version > 0)
);
-- One SKU per code across every brand and category, archived ones included
-- (an archived code stays attached to its history).
CREATE UNIQUE INDEX stock_items_sku_key ON stock_items (sku_normalised);
CREATE INDEX stock_items_category_idx ON stock_items (category_id);
CREATE INDEX stock_items_status_idx ON stock_items (status, lower(name));
CREATE INDEX stock_items_tracking_mode_idx ON stock_items (tracking_mode);

-- Who supplies the item. A link names an existing supplier; it is ended, not
-- deleted, when the item is no longer bought there.
CREATE TABLE stock_item_suppliers (
    id                   uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    stock_item_id        uuid        NOT NULL REFERENCES stock_items (id) ON DELETE RESTRICT,
    supplier_id          uuid        NOT NULL REFERENCES suppliers (id) ON DELETE RESTRICT,
    -- The supplier's own part number, when its paperwork shows one.
    supplier_part_number text,
    created_by           uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_at           timestamptz NOT NULL DEFAULT now(),
    ended_by             uuid        REFERENCES users (id) ON DELETE RESTRICT,
    ended_at             timestamptz,

    CONSTRAINT stock_item_suppliers_part_number_length CHECK (supplier_part_number IS NULL
        OR (length(btrim(supplier_part_number)) > 0 AND length(supplier_part_number) <= 60)),
    CONSTRAINT stock_item_suppliers_ended_pair CHECK ((ended_by IS NULL) = (ended_at IS NULL))
);
CREATE UNIQUE INDEX stock_item_suppliers_open_key ON stock_item_suppliers (stock_item_id, supplier_id) WHERE ended_at IS NULL;
CREATE INDEX stock_item_suppliers_supplier_idx ON stock_item_suppliers (supplier_id) WHERE ended_at IS NULL;

-- Optional compatibility: a whole brand (model_id NULL) or one model of it. The
-- model must belong to the brand (composite key). Metadata entered by staff,
-- not proof of an approved technical fitment.
CREATE TABLE stock_item_compatibility (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    stock_item_id uuid        NOT NULL REFERENCES stock_items (id) ON DELETE RESTRICT,
    brand_id      uuid        NOT NULL REFERENCES brands (id) ON DELETE RESTRICT,
    model_id      uuid,
    created_by    uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_at    timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT stock_item_compatibility_model_brand_fk FOREIGN KEY (model_id, brand_id) REFERENCES models (id, brand_id) ON DELETE RESTRICT
);
CREATE UNIQUE INDEX stock_item_compatibility_key ON stock_item_compatibility (stock_item_id, brand_id, model_id) NULLS NOT DISTINCT;
CREATE INDEX stock_item_compatibility_brand_idx ON stock_item_compatibility (brand_id, model_id);

CREATE TRIGGER stock_item_categories_set_updated_at BEFORE UPDATE ON stock_item_categories FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER stock_items_set_updated_at BEFORE UPDATE ON stock_items FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- History stays: no row is deleted; archived categories and SKUs and ended
-- supplier links never change; a link changes only by being ended; the SKU
-- code and tracking mode never change.
CREATE FUNCTION stock_item_history_guard() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'TRUNCATE' THEN
        RAISE EXCEPTION '% is never truncated', TG_TABLE_NAME USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF TG_TABLE_NAME = 'stock_item_compatibility' AND TG_OP = 'UPDATE' THEN
        RAISE EXCEPTION 'compatibility entries are replaced, not edited' USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF TG_TABLE_NAME = 'stock_item_compatibility' AND TG_OP = 'DELETE' THEN
        -- Compatibility is replaceable metadata of an ACTIVE item (changes are audited).
        IF EXISTS (SELECT 1 FROM stock_items WHERE id = OLD.stock_item_id AND status = 'ACTIVE') THEN
            RETURN OLD;
        END IF;
        RAISE EXCEPTION 'the compatibility of an archived SKU cannot change' USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION '% rows are never deleted; archive or end them instead', TG_TABLE_NAME
            USING ERRCODE = 'insufficient_privilege';
    END IF;
    -- Each table's columns are read only inside its own branch (PL/pgSQL
    -- resolves every field of a condition, whatever the table).
    IF TG_TABLE_NAME = 'stock_item_suppliers' THEN
        IF OLD.ended_at IS NOT NULL
            OR (to_jsonb(NEW) - ARRAY['ended_by', 'ended_at']) <> (to_jsonb(OLD) - ARRAY['ended_by', 'ended_at']) THEN
            RAISE EXCEPTION 'a supplier link only changes by being ended' USING ERRCODE = 'insufficient_privilege';
        END IF;
        RETURN NEW;
    END IF;
    IF OLD.status = 'ARCHIVED' THEN
        RAISE EXCEPTION 'archived % rows cannot change', TG_TABLE_NAME USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF TG_TABLE_NAME = 'stock_items' THEN
        IF NEW.sku IS DISTINCT FROM OLD.sku OR NEW.tracking_mode IS DISTINCT FROM OLD.tracking_mode THEN
            RAISE EXCEPTION 'the SKU code and tracking mode of an item never change' USING ERRCODE = 'insufficient_privilege';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER stock_item_categories_guard BEFORE UPDATE OR DELETE ON stock_item_categories FOR EACH ROW EXECUTE FUNCTION stock_item_history_guard();
CREATE TRIGGER stock_items_guard BEFORE UPDATE OR DELETE ON stock_items FOR EACH ROW EXECUTE FUNCTION stock_item_history_guard();
CREATE TRIGGER stock_item_suppliers_guard BEFORE UPDATE OR DELETE ON stock_item_suppliers FOR EACH ROW EXECUTE FUNCTION stock_item_history_guard();
CREATE TRIGGER stock_item_compatibility_guard BEFORE UPDATE OR DELETE ON stock_item_compatibility FOR EACH ROW EXECUTE FUNCTION stock_item_history_guard();
CREATE TRIGGER stock_items_no_truncate BEFORE TRUNCATE ON stock_items FOR EACH STATEMENT EXECUTE FUNCTION stock_item_history_guard();
CREATE TRIGGER stock_item_categories_no_truncate BEFORE TRUNCATE ON stock_item_categories FOR EACH STATEMENT EXECUTE FUNCTION stock_item_history_guard();
CREATE TRIGGER stock_item_suppliers_no_truncate BEFORE TRUNCATE ON stock_item_suppliers FOR EACH STATEMENT EXECUTE FUNCTION stock_item_history_guard();

-- New references must be usable, checked under a share lock so a concurrent
-- archive waits (or is seen): an item goes into an ACTIVE category; supplier
-- links and compatibility are added to ACTIVE items only, for a supplier that
-- is not archived and a brand/model that is not archived; a category with
-- ACTIVE items cannot be archived.
CREATE FUNCTION stock_item_reference_check() RETURNS trigger AS $$
DECLARE
    st text;
BEGIN
    IF TG_TABLE_NAME = 'stock_item_categories' THEN
        IF NEW.status = 'ARCHIVED' AND OLD.status <> 'ARCHIVED'
            AND EXISTS (SELECT 1 FROM stock_items WHERE category_id = NEW.id AND status = 'ACTIVE') THEN
            RAISE EXCEPTION 'a category with active SKUs cannot be archived'
                USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_item_categories_no_active_items';
        END IF;
        RETURN NEW;
    END IF;
    IF TG_TABLE_NAME = 'stock_items' THEN
        IF TG_OP = 'INSERT' OR NEW.category_id IS DISTINCT FROM OLD.category_id THEN
            SELECT status INTO st FROM stock_item_categories WHERE id = NEW.category_id FOR SHARE;
            IF st IS DISTINCT FROM 'ACTIVE' THEN
                RAISE EXCEPTION 'the category is archived' USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_items_category_active';
            END IF;
        END IF;
        RETURN NEW;
    END IF;
    SELECT status INTO st FROM stock_items WHERE id = NEW.stock_item_id FOR SHARE;
    IF st IS DISTINCT FROM 'ACTIVE' THEN
        RAISE EXCEPTION 'the SKU is archived' USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_items_active';
    END IF;
    IF TG_TABLE_NAME = 'stock_item_suppliers' THEN
        SELECT status INTO st FROM suppliers WHERE id = NEW.supplier_id FOR SHARE;
        IF st = 'ARCHIVED' THEN
            RAISE EXCEPTION 'the supplier is archived' USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_item_suppliers_supplier_active';
        END IF;
    ELSE
        SELECT lifecycle_status INTO st FROM brands WHERE id = NEW.brand_id FOR SHARE;
        IF st = 'ARCHIVED' THEN
            RAISE EXCEPTION 'the brand is archived' USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_item_compatibility_brand_active';
        END IF;
        IF NEW.model_id IS NOT NULL THEN
            SELECT lifecycle_status INTO st FROM models WHERE id = NEW.model_id FOR SHARE;
            IF st = 'ARCHIVED' THEN
                RAISE EXCEPTION 'the model is archived' USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_item_compatibility_model_active';
            END IF;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER stock_item_categories_reference_check BEFORE UPDATE ON stock_item_categories FOR EACH ROW EXECUTE FUNCTION stock_item_reference_check();
CREATE TRIGGER stock_items_reference_check BEFORE INSERT OR UPDATE ON stock_items FOR EACH ROW EXECUTE FUNCTION stock_item_reference_check();
CREATE TRIGGER stock_item_suppliers_reference_check BEFORE INSERT ON stock_item_suppliers FOR EACH ROW EXECUTE FUNCTION stock_item_reference_check();
CREATE TRIGGER stock_item_compatibility_reference_check BEFORE INSERT ON stock_item_compatibility FOR EACH ROW EXECUTE FUNCTION stock_item_reference_check();
