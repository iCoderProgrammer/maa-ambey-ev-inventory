-- Rolls back D-03. DESTRUCTIVE: drops goods receipt drafts, their lines, unit
-- identifiers, document links and history (refused in staging/production without
-- --confirm). The attachments themselves stay (their owner type remains
-- "goods_receipt", so they stay hidden). Fails while a role holds a
-- goods_receipts.* grant.
DROP TABLE IF EXISTS goods_receipt_events;
DROP TABLE IF EXISTS goods_receipt_documents;
DROP TABLE IF EXISTS goods_receipt_units;
DROP TABLE IF EXISTS goods_receipt_lines;
DROP TABLE IF EXISTS goods_receipts;
DROP SEQUENCE IF EXISTS goods_receipt_number_seq;
DROP FUNCTION IF EXISTS goods_receipt_history_guard();
ALTER TABLE purchase_order_lines DROP CONSTRAINT IF EXISTS purchase_order_lines_id_po_key;
DELETE FROM permissions WHERE key IN ('goods_receipts.view', 'goods_receipts.manage', 'goods_receipts.over_receipt');
