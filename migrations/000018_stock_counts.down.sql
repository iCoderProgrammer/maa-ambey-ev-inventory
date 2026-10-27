-- Rolls back E-05: removes stock counts and the COUNT_ADJUSTMENT extension of the
-- ledger, restoring the 000017 rules. Refuses to run while any stock count or
-- count adjustment exists: counts, decisions and ledger history are never
-- destroyed by a rollback. Fails while a role holds stock.adjustment.view.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM stock_movements WHERE movement_type = 'COUNT_ADJUSTMENT')
        OR EXISTS (SELECT 1 FROM stock_counts) THEN
        RAISE EXCEPTION 'stock counts exist; rolling back E-05 would destroy count and ledger history';
    END IF;
END $$;

DROP TRIGGER IF EXISTS stock_movements_count_check ON stock_movements;
DROP FUNCTION IF EXISTS stock_movement_count_check();
DROP INDEX IF EXISTS stock_movements_count_adjustment_once;

ALTER TABLE stock_movements
    DROP CONSTRAINT stock_movements_type_valid,
    DROP CONSTRAINT stock_movements_transition_valid,
    DROP CONSTRAINT stock_movements_quantity_valid,
    DROP CONSTRAINT stock_movements_source_valid,
    DROP CONSTRAINT stock_movements_source_complete;

ALTER TABLE stock_movements
    ADD CONSTRAINT stock_movements_type_valid CHECK (movement_type IN
        ('RECEIPT', 'PDI_PASS', 'QUALITY_HOLD', 'REJECT', 'DISPLAY_ON', 'DISPLAY_OFF', 'PARTS_RECEIPT', 'PARTS_ISSUE')),
    ADD CONSTRAINT stock_movements_transition_valid CHECK (
        (item_type = 'VEHICLE_UNIT' AND (
            (movement_type = 'RECEIPT'      AND from_status IS NULL                   AND to_status = 'RECEIVED_PENDING_PDI')
         OR (movement_type = 'PDI_PASS'     AND from_status = 'RECEIVED_PENDING_PDI'  AND to_status = 'AVAILABLE')
         OR (movement_type = 'QUALITY_HOLD' AND from_status = 'RECEIVED_PENDING_PDI'  AND to_status = 'QUALITY_HOLD')
         OR (movement_type = 'REJECT'       AND from_status = 'QUALITY_HOLD'          AND to_status = 'REJECTED')
         OR (movement_type = 'DISPLAY_ON'   AND from_status = 'AVAILABLE'             AND to_status = 'DISPLAY')
         OR (movement_type = 'DISPLAY_OFF'  AND from_status = 'DISPLAY'               AND to_status = 'AVAILABLE')))
     OR (item_type = 'STOCK_ITEM' AND movement_type IN ('PARTS_RECEIPT', 'PARTS_ISSUE'))),
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
     OR (source_type = 'SUPPLIER_DELIVERY' AND movement_type = 'PARTS_RECEIPT' AND supplier_id IS NOT NULL
         AND source_id IS NULL AND source_reference IS NOT NULL)
     OR (source_type = 'STAFF_ACTION' AND movement_type = 'PARTS_ISSUE' AND source_id IS NULL AND supplier_id IS NULL
         AND reason IS NOT NULL));

DROP TRIGGER IF EXISTS stock_counts_posting_check ON stock_counts;
DROP FUNCTION IF EXISTS stock_count_posting_check();
DROP TABLE IF EXISTS stock_count_lines;
DROP FUNCTION IF EXISTS stock_count_line_guard();
DROP TABLE IF EXISTS stock_counts;
DROP FUNCTION IF EXISTS stock_count_guard();
DROP SEQUENCE IF EXISTS stock_count_number_seq;

DELETE FROM permissions WHERE key = 'stock.adjustment.view';
