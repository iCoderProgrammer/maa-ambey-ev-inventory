-- E-04: parts receipts and issues (ADR-022; FR-015, LED-001…LED-003). Quantity
-- stock of spare parts and accessories, per SKU and location.
--
-- One ledger: the approved data model has a single stock_movements table for
-- "unit_id or stock_item_id" (System Architecture §5), so parts movements are
-- rows of the E-01 ledger with item_type STOCK_ITEM — not a second ledger.
-- Every vehicle rule of migration 000015 still applies, unchanged, to
-- item_type VEHICLE_UNIT rows.
--
-- stock_item_balances is the current quantity per SKU and location. As for
-- vehicle units, inserting a movement is what changes it (trigger below), the
-- table accepts no other change, and at commit every balance must equal its
-- latest movement. A balance never goes below zero (CHECK).
--
-- Only QUANTITY-tracked SKUs move here: no serial register or serial policy
-- exists yet (OQ-36, ADR-021), so SERIAL SKUs are refused rather than counted
-- as bulk quantities. No unit conversion (OQ-34): the movement's unit must be
-- the SKU's. No cost, price, batch/lot, adjustment or reversal (E-05, OQ-33, OQ-36).

INSERT INTO permissions (key, description, sensitive) VALUES
    ('stock.parts.view',    'View spare part and accessory stock balances and receipt/issue history at the caller''s locations', false),
    ('stock.parts.receive', 'Record spare part and accessory receipts from suppliers (increases stock)', true),
    ('stock.parts.issue',   'Record spare part and accessory issues/consumption (decreases stock)', true);

-- ── Ledger: STOCK_ITEM rows ─────────────────────────────────────────────────

ALTER TABLE stock_movements
    ALTER COLUMN vehicle_unit_id DROP NOT NULL,
    ALTER COLUMN to_status DROP NOT NULL,
    ALTER COLUMN quantity_before TYPE numeric(12,3),
    ALTER COLUMN quantity_after TYPE numeric(12,3),
    ADD COLUMN stock_item_id uuid REFERENCES stock_items (id) ON DELETE RESTRICT,
    -- The SKU's unit at the time (the movement's quantities are in it).
    ADD COLUMN uom text,
    -- Receipts: who delivered.
    ADD COLUMN supplier_id uuid REFERENCES suppliers (id) ON DELETE RESTRICT,
    -- Issues: why the stock left (required); free text until reasons are approved.
    ADD COLUMN reason text,
    DROP CONSTRAINT stock_movements_item_type_valid,
    DROP CONSTRAINT stock_movements_type_valid,
    DROP CONSTRAINT stock_movements_transition_valid,
    DROP CONSTRAINT stock_movements_quantity_valid,
    DROP CONSTRAINT stock_movements_source_valid,
    DROP CONSTRAINT stock_movements_source_complete;

ALTER TABLE stock_movements
    ADD CONSTRAINT stock_movements_item_type_valid CHECK (item_type IN ('VEHICLE_UNIT', 'STOCK_ITEM')),
    -- Each row is about exactly one thing; a parts movement happens at one location.
    ADD CONSTRAINT stock_movements_item_ref_valid CHECK (
        (item_type = 'VEHICLE_UNIT' AND vehicle_unit_id IS NOT NULL AND stock_item_id IS NULL AND to_status IS NOT NULL
         AND uom IS NULL AND supplier_id IS NULL AND reason IS NULL)
     OR (item_type = 'STOCK_ITEM' AND stock_item_id IS NOT NULL AND vehicle_unit_id IS NULL AND from_status IS NULL
         AND to_status IS NULL AND to_location_id IS NOT NULL AND from_location_id = to_location_id AND uom IS NOT NULL)),
    ADD CONSTRAINT stock_movements_type_valid CHECK (movement_type IN
        ('RECEIPT', 'PDI_PASS', 'QUALITY_HOLD', 'REJECT', 'DISPLAY_ON', 'DISPLAY_OFF', 'PARTS_RECEIPT', 'PARTS_ISSUE')),
    -- Vehicle edges exactly as in 000015, for vehicle rows only; parts types for parts rows only.
    ADD CONSTRAINT stock_movements_transition_valid CHECK (
        (item_type = 'VEHICLE_UNIT' AND (
            (movement_type = 'RECEIPT'      AND from_status IS NULL                   AND to_status = 'RECEIVED_PENDING_PDI')
         OR (movement_type = 'PDI_PASS'     AND from_status = 'RECEIVED_PENDING_PDI'  AND to_status = 'AVAILABLE')
         OR (movement_type = 'QUALITY_HOLD' AND from_status = 'RECEIVED_PENDING_PDI'  AND to_status = 'QUALITY_HOLD')
         OR (movement_type = 'REJECT'       AND from_status = 'QUALITY_HOLD'          AND to_status = 'REJECTED')
         OR (movement_type = 'DISPLAY_ON'   AND from_status = 'AVAILABLE'             AND to_status = 'DISPLAY')
         OR (movement_type = 'DISPLAY_OFF'  AND from_status = 'DISPLAY'               AND to_status = 'AVAILABLE')))
     OR (item_type = 'STOCK_ITEM' AND movement_type IN ('PARTS_RECEIPT', 'PARTS_ISSUE'))),
    -- Vehicles: one item, 0 -> 1 on receipt, 1 -> 1 otherwise (000015). Parts: a receipt
    -- adds, an issue takes away, and nothing goes below zero.
    ADD CONSTRAINT stock_movements_quantity_valid CHECK (
        (item_type = 'VEHICLE_UNIT' AND (
            (movement_type = 'RECEIPT' AND quantity_before = 0 AND quantity_after = 1 AND from_location_id IS NULL)
         OR (movement_type <> 'RECEIPT' AND quantity_before = 1 AND quantity_after = 1
             AND from_location_id IS NOT DISTINCT FROM to_location_id)))
     OR (item_type = 'STOCK_ITEM' AND quantity_before >= 0 AND quantity_after >= 0 AND (
            (movement_type = 'PARTS_RECEIPT' AND quantity_after > quantity_before)
         OR (movement_type = 'PARTS_ISSUE' AND quantity_after < quantity_before)))),
    ADD CONSTRAINT stock_movements_source_valid CHECK (source_type IN ('GOODS_RECEIPT', 'STAFF_ACTION', 'SUPPLIER_DELIVERY')),
    ADD CONSTRAINT stock_movements_source_complete CHECK (
        (source_type = 'GOODS_RECEIPT' AND movement_type = 'RECEIPT' AND source_id IS NOT NULL AND source_reference IS NOT NULL)
     OR (source_type = 'STAFF_ACTION' AND item_type = 'VEHICLE_UNIT' AND movement_type <> 'RECEIPT' AND source_id IS NULL
         AND source_reference IS NULL)
        -- A parts receipt names the supplier and its paperwork (delivery challan / invoice number).
     OR (source_type = 'SUPPLIER_DELIVERY' AND movement_type = 'PARTS_RECEIPT' AND supplier_id IS NOT NULL
         AND source_id IS NULL AND source_reference IS NOT NULL)
        -- A parts issue gives its reason; a reference (e.g. job card number) is optional.
     OR (source_type = 'STAFF_ACTION' AND movement_type = 'PARTS_ISSUE' AND source_id IS NULL AND supplier_id IS NULL
         AND reason IS NOT NULL)),
    ADD CONSTRAINT stock_movements_source_reference_length CHECK (source_reference IS NULL
        OR (length(btrim(source_reference)) > 0 AND length(source_reference) <= 100)),
    ADD CONSTRAINT stock_movements_reason_length CHECK (reason IS NULL OR (length(btrim(reason)) > 0 AND length(reason) <= 300)),
    ADD CONSTRAINT stock_movements_uom_format CHECK (uom IS NULL OR uom ~ '^[A-Z][A-Z0-9]{0,9}$');

CREATE INDEX stock_movements_stock_item_idx ON stock_movements (stock_item_id, to_location_id, seq) WHERE stock_item_id IS NOT NULL;

-- ── Balances ────────────────────────────────────────────────────────────────

CREATE TABLE stock_item_balances (
    stock_item_id uuid          NOT NULL REFERENCES stock_items (id) ON DELETE RESTRICT,
    location_id   uuid          NOT NULL REFERENCES locations (id) ON DELETE RESTRICT,
    quantity      numeric(12,3) NOT NULL DEFAULT 0,
    version       integer       NOT NULL DEFAULT 1,
    updated_at    timestamptz   NOT NULL DEFAULT now(),

    PRIMARY KEY (stock_item_id, location_id),
    -- The final guard against negative stock.
    CONSTRAINT stock_item_balances_not_negative CHECK (quantity >= 0),
    CONSTRAINT stock_item_balances_version_positive CHECK (version > 0)
);
CREATE INDEX stock_item_balances_location_idx ON stock_item_balances (location_id);

-- A balance row may be created empty (0, version 1) so a first movement can lock
-- it; after that it changes only from inside stock_movement_apply (trigger depth
-- > 1, for the row the marker names, quantity/version/updated_at only), is never
-- deleted or truncated.
CREATE FUNCTION stock_item_balance_guard() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'TRUNCATE' OR TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'stock balances are never deleted' USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF TG_OP = 'INSERT' THEN
        IF NEW.quantity = 0 AND NEW.version = 1 THEN
            RETURN NEW;
        END IF;
        RAISE EXCEPTION 'stock balances start empty and change only through stock movements' USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF pg_trigger_depth() > 1
        AND current_setting('maa.stock_movement_balance', true) = OLD.stock_item_id::text || '/' || OLD.location_id::text
        AND NEW.stock_item_id = OLD.stock_item_id AND NEW.location_id = OLD.location_id
        AND NEW.version = OLD.version + 1 THEN
        RETURN NEW;
    END IF;
    RAISE EXCEPTION 'stock balances change only through stock movements' USING ERRCODE = 'insufficient_privilege';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER stock_item_balances_guard BEFORE INSERT OR UPDATE OR DELETE ON stock_item_balances
    FOR EACH ROW EXECUTE FUNCTION stock_item_balance_guard();
CREATE TRIGGER stock_item_balances_no_truncate BEFORE TRUNCATE ON stock_item_balances
    FOR EACH STATEMENT EXECUTE FUNCTION stock_item_balance_guard();

-- ── Applying a movement ─────────────────────────────────────────────────────

-- Vehicle rows: exactly as in 000015. Parts rows: the SKU must be ACTIVE and
-- QUANTITY-tracked with the movement's unit, the location active, a receipt's
-- supplier not archived (checked under share locks so a concurrent archive
-- waits); the balance row is created empty if missing, locked, must hold the
-- movement's "before" (a stale or concurrent change fails instead of
-- overwriting) and takes its "after".
CREATE OR REPLACE FUNCTION stock_movement_apply() RETURNS trigger AS $$
DECLARE
    cur_status   text;
    cur_location uuid;
    cur_receipt  uuid;
    item_status  text;
    item_mode    text;
    item_uom     text;
    loc_active   boolean;
    sup_status   text;
    rows         integer;
BEGIN
    IF NEW.item_type = 'STOCK_ITEM' THEN
        SELECT status, tracking_mode, uom INTO item_status, item_mode, item_uom
            FROM stock_items WHERE id = NEW.stock_item_id FOR SHARE;
        IF item_status IS DISTINCT FROM 'ACTIVE' THEN
            RAISE EXCEPTION 'the SKU is archived' USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_movements_item_active';
        END IF;
        IF item_mode IS DISTINCT FROM 'QUANTITY' THEN
            RAISE EXCEPTION 'only quantity-tracked SKUs move by quantity'
                USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_movements_item_quantity_tracked';
        END IF;
        IF item_uom IS DISTINCT FROM NEW.uom THEN
            RAISE EXCEPTION 'the movement''s unit is not the SKU''s'
                USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_movements_uom_matches_item';
        END IF;
        SELECT active INTO loc_active FROM locations WHERE id = NEW.to_location_id FOR SHARE;
        IF loc_active IS DISTINCT FROM true THEN
            RAISE EXCEPTION 'the location is not active' USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_movements_location_active';
        END IF;
        IF NEW.supplier_id IS NOT NULL THEN
            SELECT status INTO sup_status FROM suppliers WHERE id = NEW.supplier_id FOR SHARE;
            IF sup_status = 'ARCHIVED' THEN
                RAISE EXCEPTION 'the supplier is archived'
                    USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_movements_supplier_active';
            END IF;
        END IF;
        INSERT INTO stock_item_balances (stock_item_id, location_id) VALUES (NEW.stock_item_id, NEW.to_location_id)
            ON CONFLICT DO NOTHING;
        PERFORM set_config('maa.stock_movement_balance', NEW.stock_item_id::text || '/' || NEW.to_location_id::text, true);
        UPDATE stock_item_balances SET quantity = NEW.quantity_after, version = version + 1, updated_at = now()
            WHERE stock_item_id = NEW.stock_item_id AND location_id = NEW.to_location_id AND quantity = NEW.quantity_before;
        GET DIAGNOSTICS rows = ROW_COUNT;
        PERFORM set_config('maa.stock_movement_balance', '', true);
        IF rows <> 1 THEN
            RAISE EXCEPTION 'the stock balance is no longer the movement''s starting quantity'
                USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_movements_from_matches_balance';
        END IF;
        RETURN NULL;
    END IF;

    SELECT status, location_id, goods_receipt_id INTO cur_status, cur_location, cur_receipt
        FROM vehicle_units WHERE id = NEW.vehicle_unit_id FOR UPDATE;
    IF NEW.movement_type = 'RECEIPT' THEN
        IF cur_status IS DISTINCT FROM NEW.to_status OR cur_location IS DISTINCT FROM NEW.to_location_id
            OR cur_receipt IS DISTINCT FROM NEW.source_id
            OR EXISTS (SELECT 1 FROM stock_movements WHERE vehicle_unit_id = NEW.vehicle_unit_id AND id <> NEW.id) THEN
            RAISE EXCEPTION 'a receipt movement must be the unit''s first, from its own goods receipt'
                USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_movements_receipt_matches_unit';
        END IF;
        RETURN NULL;
    END IF;
    IF cur_status IS DISTINCT FROM NEW.from_status OR cur_location IS DISTINCT FROM NEW.from_location_id THEN
        RAISE EXCEPTION 'the vehicle unit is no longer in the movement''s starting state'
            USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_movements_from_matches_unit';
    END IF;
    PERFORM set_config('maa.stock_movement_unit', NEW.vehicle_unit_id::text, true);
    UPDATE vehicle_units SET status = NEW.to_status, location_id = NEW.to_location_id, version = version + 1, updated_at = now()
        WHERE id = NEW.vehicle_unit_id;
    PERFORM set_config('maa.stock_movement_unit', '', true);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- LED-003 for parts, checked at commit: a balance equals the quantity after its
-- latest movement, or 0 when it has none.
CREATE FUNCTION stock_item_balance_ledger_check() RETURNS trigger AS $$
DECLARE
    qty  numeric;
    last numeric;
BEGIN
    SELECT quantity INTO qty FROM stock_item_balances WHERE stock_item_id = NEW.stock_item_id AND location_id = NEW.location_id;
    SELECT quantity_after INTO last FROM stock_movements
        WHERE stock_item_id = NEW.stock_item_id AND to_location_id = NEW.location_id ORDER BY seq DESC LIMIT 1;
    IF qty IS DISTINCT FROM coalesce(last, 0) THEN
        RAISE EXCEPTION 'stock balance of % at % does not match its stock movements', NEW.stock_item_id, NEW.location_id
            USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_item_balances_ledger_reconciles';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER stock_item_balances_ledger_check AFTER INSERT OR UPDATE ON stock_item_balances
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION stock_item_balance_ledger_check();
