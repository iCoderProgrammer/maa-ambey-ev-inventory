-- Placeholder fixture for the D-06 browser test (throwaway DB only). Stands in
-- for receipt posting (D-07), mirroring vehicles.RegisterFromReceiptUnit.
DO $$
DECLARE usr uuid; appr uuid; loc uuid; b uuid; m uuid; v uuid; s uuid; po uuid; pl uuid; g uuid; gl uuid; gu uuid; vu uuid; grn text; i int;
BEGIN
  SELECT id INTO usr FROM users WHERE username = 'e2e.catalogue';
  SELECT id INTO loc FROM locations WHERE code = 'E06';
  SELECT id INTO appr FROM users WHERE username = 'e2e.catviewer';
  INSERT INTO brands (name, slug, lifecycle_status, authorisation_status, manufacturer_legal_name, authorisation_reference, verified_by, verified_at)
    VALUES ('Test Brand', 'test-brand', 'ACTIVE', 'AUTHORISED', 'Example Manufacturer', 'TEST-AUTH', usr, now()) RETURNING id INTO b;
  INSERT INTO models (brand_id, name, internal_code, lifecycle_status, confidence, source_url, source_date, verified_by, verified_at)
    VALUES (b, 'Test Model', 'TM', 'ACTIVE', 'SUPPLIER_CONFIRMED', 'https://example.com/m', current_date, usr, now()) RETURNING id INTO m;
  INSERT INTO variants (brand_id, model_id, name, variant_code, lifecycle_status, confidence, source_url, source_date, verified_by, verified_at)
    VALUES (b, m, 'Standard', 'STD', 'ACTIVE', 'SUPPLIER_CONFIRMED', 'https://example.com/v', current_date, usr, now()) RETURNING id INTO v;
  INSERT INTO suppliers (legal_name, created_by) VALUES ('Example Supplier', usr) RETURNING id INTO s;
  INSERT INTO purchase_orders (supplier_id, created_by, expected_date) VALUES (s, usr, current_date + 7) RETURNING id INTO po;
  INSERT INTO purchase_order_lines (purchase_order_id, line_no, brand_id, model_id, variant_id, quantity, unit_cost)
    VALUES (po, 1, b, m, v, 50, 95000.00) RETURNING id INTO pl;
  UPDATE purchase_orders SET status = 'APPROVED', submitted_by = usr, submitted_at = now(), approval_basis = 'APPROVER',
    approved_by = appr, approved_at = now() WHERE id = po;
  INSERT INTO goods_receipts (purchase_order_id, received_date, supplier_reference, created_by)
    VALUES (po, current_date, 'DC-TEST', usr) RETURNING id, grn_number INTO g, grn;
  INSERT INTO goods_receipt_lines (goods_receipt_id, purchase_order_id, purchase_order_line_id, received_quantity, accepted_quantity, rejected_quantity)
    VALUES (g, po, pl, 8, 8, 0) RETURNING id INTO gl;
  FOR i IN 1..8 LOOP
    INSERT INTO goods_receipt_units (goods_receipt_id, goods_receipt_line_id, unit_no, condition, chassis_number, chassis_normalised)
      VALUES (g, gl, i, 'ACCEPTED', 'TEST-CH-' || i, 'TEST-CH-' || i) RETURNING id INTO gu;
    INSERT INTO vehicle_units (brand_id, model_id, variant_id, status, chassis_number, location_id, source_type, supplier_id,
        purchase_order_id, purchase_order_line_id, goods_receipt_id, goods_receipt_line_id, goods_receipt_unit_id, received_date, unit_cost, registered_by)
      VALUES (b, m, v, 'RECEIVED_PENDING_PDI', 'TEST-CH-' || i, loc, 'GOODS_RECEIPT', s, po, pl, g, gl, gu, current_date, 95000.00, usr)
      RETURNING id INTO vu;
    INSERT INTO vehicle_unit_events (vehicle_unit_id, action, to_status, source_type, source_id, source_reference, actor_id)
      VALUES (vu, 'REGISTERED', 'RECEIVED_PENDING_PDI', 'GOODS_RECEIPT', g, grn, usr);
    INSERT INTO stock_movements (vehicle_unit_id, movement_type, quantity_before, quantity_after, from_status, to_status, to_location_id,
        source_type, source_id, source_reference, actor_id, idempotency_key, request_hash)
      VALUES (vu, 'RECEIPT', 0, 1, NULL, 'RECEIVED_PENDING_PDI', loc, 'GOODS_RECEIPT', g, grn, usr, 'receipt-unit-' || gu, encode(sha256(('fixture' || gu)::bytea), 'hex'));
  END LOOP;
  UPDATE goods_receipts SET status = 'POSTED', posted_by = usr, posted_at = now() WHERE id = g;
END $$;
SELECT status, count(*) FROM vehicle_units GROUP BY status;
