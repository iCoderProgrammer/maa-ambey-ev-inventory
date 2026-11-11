-- Rolls back D-06: removes PDI checklist items, inspections and the
-- PDI_INSPECTION ledger source, restoring the 000018 rules. Refuses to run while
-- any checklist item, inspection or PDI_INSPECTION movement exists: inspection,
-- evidence and ledger history are never destroyed by a rollback. Fails while a
-- role holds one of the two PDI permissions.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM stock_movements WHERE source_type = 'PDI_INSPECTION')
        OR EXISTS (SELECT 1 FROM pdi_inspections)
        OR EXISTS (SELECT 1 FROM pdi_checklist_items) THEN
        RAISE EXCEPTION 'PDI records exist; rolling back D-06 would destroy inspection and ledger history';
    END IF;
END $$;

DROP TRIGGER IF EXISTS stock_movements_pdi_check ON stock_movements;
DROP FUNCTION IF EXISTS stock_movement_pdi_check();
DROP INDEX IF EXISTS stock_movements_pdi_once;

ALTER TABLE stock_movements
    DROP CONSTRAINT stock_movements_source_valid,
    DROP CONSTRAINT stock_movements_source_complete;

ALTER TABLE stock_movements
    ADD CONSTRAINT stock_movements_source_valid CHECK (source_type IN
        ('GOODS_RECEIPT', 'STAFF_ACTION', 'SUPPLIER_DELIVERY', 'STOCK_COUNT')),
    ADD CONSTRAINT stock_movements_source_complete CHECK (
        (source_type = 'GOODS_RECEIPT' AND movement_type = 'RECEIPT' AND source_id IS NOT NULL AND source_reference IS NOT NULL)
     OR (source_type = 'STAFF_ACTION' AND item_type = 'VEHICLE_UNIT' AND movement_type <> 'RECEIPT' AND source_id IS NULL
         AND source_reference IS NULL)
     OR (source_type = 'SUPPLIER_DELIVERY' AND movement_type = 'PARTS_RECEIPT' AND supplier_id IS NOT NULL
         AND source_id IS NULL AND source_reference IS NOT NULL)
     OR (source_type = 'STAFF_ACTION' AND movement_type = 'PARTS_ISSUE' AND source_id IS NULL AND supplier_id IS NULL
         AND reason IS NOT NULL)
     OR (source_type = 'STOCK_COUNT' AND movement_type = 'COUNT_ADJUSTMENT' AND source_id IS NOT NULL
         AND source_reference IS NOT NULL AND supplier_id IS NULL AND reason IS NOT NULL));

DROP TRIGGER IF EXISTS pdi_inspections_check ON pdi_inspections;
DROP FUNCTION IF EXISTS pdi_inspection_check();
DROP TABLE IF EXISTS pdi_inspection_attachments;
DROP TABLE IF EXISTS pdi_inspection_items;
DROP TABLE IF EXISTS pdi_inspections;
DROP FUNCTION IF EXISTS pdi_append_only();
DROP SEQUENCE IF EXISTS pdi_inspection_number_seq;
DROP TABLE IF EXISTS pdi_checklist_items;
DROP FUNCTION IF EXISTS pdi_checklist_item_guard();

DELETE FROM permissions WHERE key IN ('vehicle_units.pdi.record', 'vehicle_units.pdi.checklist.manage');
