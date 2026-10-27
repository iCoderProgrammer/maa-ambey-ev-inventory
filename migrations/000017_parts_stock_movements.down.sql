-- Rolls back E-04: removes parts balances and the STOCK_ITEM extension of the
-- ledger, restoring the 000015 vehicle-only rules. Refuses to run while any parts
-- movement exists: the ledger is append-only and its history is not destroyed by
-- a rollback. Fails while a role holds a stock.parts.* grant.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM stock_movements WHERE item_type <> 'VEHICLE_UNIT') THEN
        RAISE EXCEPTION 'parts stock movements exist; rolling back E-04 would destroy ledger history';
    END IF;
END $$;

DROP TRIGGER IF EXISTS stock_item_balances_ledger_check ON stock_item_balances;
DROP FUNCTION IF EXISTS stock_item_balance_ledger_check();
DROP TABLE IF EXISTS stock_item_balances;
DROP FUNCTION IF EXISTS stock_item_balance_guard();
DROP INDEX IF EXISTS stock_movements_stock_item_idx;

ALTER TABLE stock_movements
    DROP CONSTRAINT stock_movements_item_type_valid,
    DROP CONSTRAINT stock_movements_item_ref_valid,
    DROP CONSTRAINT stock_movements_type_valid,
    DROP CONSTRAINT stock_movements_transition_valid,
    DROP CONSTRAINT stock_movements_quantity_valid,
    DROP CONSTRAINT stock_movements_source_valid,
    DROP CONSTRAINT stock_movements_source_complete,
    DROP CONSTRAINT stock_movements_source_reference_length,
    DROP CONSTRAINT stock_movements_reason_length,
    DROP CONSTRAINT stock_movements_uom_format,
    DROP COLUMN reason,
    DROP COLUMN supplier_id,
    DROP COLUMN uom,
    DROP COLUMN stock_item_id,
    ALTER COLUMN quantity_before TYPE smallint,
    ALTER COLUMN quantity_after TYPE smallint,
    ALTER COLUMN to_status SET NOT NULL,
    ALTER COLUMN vehicle_unit_id SET NOT NULL;

ALTER TABLE stock_movements
    ADD CONSTRAINT stock_movements_item_type_valid CHECK (item_type = 'VEHICLE_UNIT'),
    ADD CONSTRAINT stock_movements_type_valid CHECK (movement_type IN
        ('RECEIPT', 'PDI_PASS', 'QUALITY_HOLD', 'REJECT', 'DISPLAY_ON', 'DISPLAY_OFF')),
    ADD CONSTRAINT stock_movements_transition_valid CHECK (
        (movement_type = 'RECEIPT'      AND from_status IS NULL                   AND to_status = 'RECEIVED_PENDING_PDI')
     OR (movement_type = 'PDI_PASS'     AND from_status = 'RECEIVED_PENDING_PDI'  AND to_status = 'AVAILABLE')
     OR (movement_type = 'QUALITY_HOLD' AND from_status = 'RECEIVED_PENDING_PDI'  AND to_status = 'QUALITY_HOLD')
     OR (movement_type = 'REJECT'       AND from_status = 'QUALITY_HOLD'          AND to_status = 'REJECTED')
     OR (movement_type = 'DISPLAY_ON'   AND from_status = 'AVAILABLE'             AND to_status = 'DISPLAY')
     OR (movement_type = 'DISPLAY_OFF'  AND from_status = 'DISPLAY'               AND to_status = 'AVAILABLE')),
    ADD CONSTRAINT stock_movements_quantity_valid CHECK (
        (movement_type = 'RECEIPT' AND quantity_before = 0 AND quantity_after = 1 AND from_location_id IS NULL)
     OR (movement_type <> 'RECEIPT' AND quantity_before = 1 AND quantity_after = 1
         AND from_location_id IS NOT DISTINCT FROM to_location_id)),
    ADD CONSTRAINT stock_movements_source_valid CHECK (source_type IN ('GOODS_RECEIPT', 'STAFF_ACTION')),
    ADD CONSTRAINT stock_movements_source_complete CHECK (
        (source_type = 'GOODS_RECEIPT' AND movement_type = 'RECEIPT' AND source_id IS NOT NULL AND source_reference IS NOT NULL)
     OR (source_type = 'STAFF_ACTION' AND movement_type <> 'RECEIPT' AND source_id IS NULL AND source_reference IS NULL));

CREATE OR REPLACE FUNCTION stock_movement_apply() RETURNS trigger AS $$
DECLARE
    cur_status   text;
    cur_location uuid;
    cur_receipt  uuid;
BEGIN
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

DELETE FROM permissions WHERE key IN ('stock.parts.view', 'stock.parts.receive', 'stock.parts.issue');
