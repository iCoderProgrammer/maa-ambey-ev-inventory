-- Rolls back D-04. DESTRUCTIVE once units exist: drops vehicle units and their
-- history (refused in staging/production without --confirm). Fails while a role
-- holds a vehicle_units.* grant.
DROP TABLE IF EXISTS vehicle_unit_events;
DROP TABLE IF EXISTS vehicle_units;
DROP FUNCTION IF EXISTS vehicle_unit_source_check();
DROP FUNCTION IF EXISTS vehicle_unit_history_guard();
DROP SEQUENCE IF EXISTS vehicle_unit_number_seq;
ALTER TABLE goods_receipt_units DROP CONSTRAINT IF EXISTS goods_receipt_units_id_line_key;
ALTER TABLE goods_receipt_lines DROP CONSTRAINT IF EXISTS goods_receipt_lines_id_po_line_key;
ALTER TABLE purchase_order_lines DROP CONSTRAINT IF EXISTS purchase_order_lines_id_variant_key;
ALTER TABLE purchase_orders DROP CONSTRAINT IF EXISTS purchase_orders_id_supplier_key;
DELETE FROM permissions WHERE key IN ('vehicle_units.view', 'vehicle_units.cost.view');
