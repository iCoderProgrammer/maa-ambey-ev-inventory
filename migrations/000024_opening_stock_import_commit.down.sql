-- Reverts 000024 (G-07). Refused once anything was approved or committed: committed
-- opening stock, its movements and provenance are never erased by a rollback.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM opening_stock_import_approvals) OR EXISTS (SELECT 1 FROM opening_stock_import_commits)
        OR EXISTS (SELECT 1 FROM vehicle_units WHERE source_type = 'OPENING_STOCK')
        OR EXISTS (SELECT 1 FROM stock_movements WHERE movement_type = 'OPENING_BALANCE' OR source_type = 'OPENING_STOCK') THEN
        RAISE EXCEPTION 'opening-stock approvals, commits or units exist; 000024 cannot be rolled back';
    END IF;
END $$;

-- Functions as before 000024 (000017 apply, 000015 ledger check).
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

CREATE OR REPLACE FUNCTION vehicle_unit_ledger_check() RETURNS trigger AS $$
DECLARE
    unit_status   text;
    unit_location uuid;
    last          stock_movements%ROWTYPE;
BEGIN
    SELECT status, location_id INTO unit_status, unit_location FROM vehicle_units WHERE id = NEW.id;
    SELECT * INTO last FROM stock_movements WHERE vehicle_unit_id = NEW.id ORDER BY seq DESC LIMIT 1;
    IF NOT FOUND OR NOT EXISTS (SELECT 1 FROM stock_movements WHERE vehicle_unit_id = NEW.id AND movement_type = 'RECEIPT')
        OR last.to_status IS DISTINCT FROM unit_status OR last.to_location_id IS DISTINCT FROM unit_location
        OR last.quantity_after <> 1 THEN
        RAISE EXCEPTION 'vehicle unit % does not match its stock movements', NEW.id
            USING ERRCODE = 'check_violation', CONSTRAINT = 'vehicle_units_ledger_reconciles';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP INDEX stock_movements_opening_balance_once;

ALTER TABLE stock_movements
    DROP CONSTRAINT stock_movements_type_valid,
    DROP CONSTRAINT stock_movements_transition_valid,
    DROP CONSTRAINT stock_movements_quantity_valid,
    DROP CONSTRAINT stock_movements_source_valid,
    DROP CONSTRAINT stock_movements_source_complete;

-- As 000018 (type, transition, quantity) and 000019 (source).
ALTER TABLE stock_movements
    ADD CONSTRAINT stock_movements_type_valid CHECK (movement_type IN
        ('RECEIPT', 'PDI_PASS', 'QUALITY_HOLD', 'REJECT', 'DISPLAY_ON', 'DISPLAY_OFF', 'PARTS_RECEIPT', 'PARTS_ISSUE',
         'COUNT_ADJUSTMENT')),
    ADD CONSTRAINT stock_movements_transition_valid CHECK (
        (item_type = 'VEHICLE_UNIT' AND (
            (movement_type = 'RECEIPT'      AND from_status IS NULL                   AND to_status = 'RECEIVED_PENDING_PDI')
         OR (movement_type = 'PDI_PASS'     AND from_status = 'RECEIVED_PENDING_PDI'  AND to_status = 'AVAILABLE')
         OR (movement_type = 'QUALITY_HOLD' AND from_status = 'RECEIVED_PENDING_PDI'  AND to_status = 'QUALITY_HOLD')
         OR (movement_type = 'REJECT'       AND from_status = 'QUALITY_HOLD'          AND to_status = 'REJECTED')
         OR (movement_type = 'DISPLAY_ON'   AND from_status = 'AVAILABLE'             AND to_status = 'DISPLAY')
         OR (movement_type = 'DISPLAY_OFF'  AND from_status = 'DISPLAY'               AND to_status = 'AVAILABLE')))
     OR (item_type = 'STOCK_ITEM' AND movement_type IN ('PARTS_RECEIPT', 'PARTS_ISSUE', 'COUNT_ADJUSTMENT'))),
    ADD CONSTRAINT stock_movements_quantity_valid CHECK (
        (item_type = 'VEHICLE_UNIT' AND (
            (movement_type = 'RECEIPT' AND quantity_before = 0 AND quantity_after = 1 AND from_location_id IS NULL)
         OR (movement_type <> 'RECEIPT' AND quantity_before = 1 AND quantity_after = 1
             AND from_location_id IS NOT DISTINCT FROM to_location_id)))
     OR (item_type = 'STOCK_ITEM' AND quantity_before >= 0 AND quantity_after >= 0 AND (
            (movement_type = 'PARTS_RECEIPT' AND quantity_after > quantity_before)
         OR (movement_type = 'PARTS_ISSUE' AND quantity_after < quantity_before)
         OR (movement_type = 'COUNT_ADJUSTMENT' AND quantity_after <> quantity_before)))),
    ADD CONSTRAINT stock_movements_source_valid CHECK (source_type IN
        ('GOODS_RECEIPT', 'STAFF_ACTION', 'SUPPLIER_DELIVERY', 'STOCK_COUNT', 'PDI_INSPECTION')),
    ADD CONSTRAINT stock_movements_source_complete CHECK (
        (source_type = 'GOODS_RECEIPT' AND movement_type = 'RECEIPT' AND source_id IS NOT NULL AND source_reference IS NOT NULL)
     OR (source_type = 'STAFF_ACTION' AND item_type = 'VEHICLE_UNIT' AND movement_type <> 'RECEIPT' AND source_id IS NULL
         AND source_reference IS NULL)
     OR (source_type = 'SUPPLIER_DELIVERY' AND movement_type = 'PARTS_RECEIPT' AND supplier_id IS NOT NULL
         AND source_id IS NULL AND source_reference IS NOT NULL)
     OR (source_type = 'STAFF_ACTION' AND movement_type = 'PARTS_ISSUE' AND source_id IS NULL AND supplier_id IS NULL
         AND reason IS NOT NULL)
     OR (source_type = 'STOCK_COUNT' AND movement_type = 'COUNT_ADJUSTMENT' AND source_id IS NOT NULL
         AND source_reference IS NOT NULL AND supplier_id IS NULL AND reason IS NOT NULL)
     OR (source_type = 'PDI_INSPECTION' AND item_type = 'VEHICLE_UNIT'
         AND movement_type IN ('PDI_PASS', 'QUALITY_HOLD', 'REJECT')
         AND source_id IS NOT NULL AND source_reference IS NOT NULL AND supplier_id IS NULL));

ALTER TABLE vehicle_unit_events DROP CONSTRAINT vehicle_unit_events_source_valid;
ALTER TABLE vehicle_unit_events ADD CONSTRAINT vehicle_unit_events_source_valid CHECK (source_type IN ('GOODS_RECEIPT'));

DROP TRIGGER vehicle_units_opening_stock_check ON vehicle_units;
DROP FUNCTION vehicle_unit_opening_stock_check();
DROP INDEX vehicle_units_opening_stock_batch_idx;
ALTER TABLE vehicle_units
    DROP CONSTRAINT vehicle_units_opening_stock_source_complete,
    DROP CONSTRAINT vehicle_units_opening_stock_row_key,
    DROP CONSTRAINT vehicle_units_opening_stock_row_fk,
    DROP CONSTRAINT vehicle_units_source_valid;
ALTER TABLE vehicle_units ADD CONSTRAINT vehicle_units_source_valid CHECK (source_type IN ('GOODS_RECEIPT'));
ALTER TABLE vehicle_units DROP COLUMN opening_stock_row_number, DROP COLUMN opening_stock_batch_id;

DROP TABLE opening_stock_import_commits;
DROP FUNCTION opening_stock_import_commit_complete();
DROP SEQUENCE opening_stock_commit_number_seq;
DROP TABLE opening_stock_import_approvals;
DROP FUNCTION opening_stock_import_previewed_check();

ALTER TABLE opening_stock_import_batches DROP CONSTRAINT opening_stock_import_batches_id_sha256_key;

DELETE FROM permissions WHERE key IN ('stock.import.approve', 'stock.import.commit');
