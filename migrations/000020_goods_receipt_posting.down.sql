-- Rolls back D-07 posting columns. Refuses while any receipt was posted through
-- D-07 (location or posting key recorded) or any PO event records a receipt:
-- posting history is never destroyed by a rollback. Fails while a role holds
-- goods_receipts.post.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM goods_receipts WHERE location_id IS NOT NULL OR post_idempotency_key IS NOT NULL)
        OR EXISTS (SELECT 1 FROM purchase_order_events WHERE action = 'GOODS_RECEIVED') THEN
        RAISE EXCEPTION 'posted goods receipts exist; rolling back D-07 would destroy posting history';
    END IF;
END $$;

ALTER TABLE purchase_order_events DROP CONSTRAINT purchase_order_events_action_valid;
ALTER TABLE purchase_order_events ADD CONSTRAINT purchase_order_events_action_valid CHECK (action IN
    ('CREATED', 'SUBMITTED', 'APPROVED', 'APPROVAL_NOT_REQUIRED', 'RETURNED', 'CANCELLED'));

DROP TRIGGER IF EXISTS goods_receipts_status_guard ON goods_receipts;
DROP FUNCTION IF EXISTS goods_receipt_status_guard();
DROP INDEX IF EXISTS goods_receipts_location_idx;
DROP INDEX IF EXISTS goods_receipts_post_idempotency_key;
ALTER TABLE goods_receipts
    DROP CONSTRAINT goods_receipts_post_key_complete,
    DROP CONSTRAINT goods_receipts_post_key_format,
    DROP COLUMN post_request_hash,
    DROP COLUMN post_idempotency_key,
    DROP COLUMN location_id;

DELETE FROM permissions WHERE key = 'goods_receipts.post';
