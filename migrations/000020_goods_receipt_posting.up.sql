-- Goods receipt posting (D-07; ADR-025; PO-003, PO-004, PO-005, INV-001, INV-009).
--
-- POST /goods-receipts/{id}/post turns a DRAFT receipt into POSTED in one
-- transaction: every ACCEPTED captured unit becomes a vehicle unit at
-- RECEIVED_PENDING_PDI (vehicles.RegisterFromReceiptUnit: unit, history, RECEIPT
-- movement, audit), the PO lines' received_quantity grows by the received
-- quantity (the D-03 basis, OQ-27) and the PO moves to PARTIALLY_RECEIVED or
-- RECEIVED. Nothing becomes saleable: PDI (D-06) follows.
--
-- New here:
--   * permission goods_receipts.post (no grant, RB-5);
--   * goods_receipts.location_id: where the goods were received, chosen at
--     posting (PO delivery location is still open, OQ-24); the vehicle units are
--     registered there;
--   * the posting's idempotency key and request hash (per poster);
--   * purchase order event action GOODS_RECEIVED (status change by a posting).

INSERT INTO permissions (key, description, sensitive) VALUES
    ('goods_receipts.post', 'Post a draft goods receipt: registers the accepted vehicle units and updates the purchase order', true);

ALTER TABLE goods_receipts
    ADD COLUMN location_id           uuid REFERENCES locations (id) ON DELETE RESTRICT,
    ADD COLUMN post_idempotency_key  text,
    ADD COLUMN post_request_hash     text,
    ADD CONSTRAINT goods_receipts_post_key_format CHECK (post_idempotency_key IS NULL
        OR post_idempotency_key ~ '^[A-Za-z0-9_-]{8,100}$'),
    -- A key is recorded only by a posting, with its hash and the location.
    ADD CONSTRAINT goods_receipts_post_key_complete CHECK (
        (post_idempotency_key IS NULL AND post_request_hash IS NULL)
     OR (post_idempotency_key IS NOT NULL AND post_request_hash IS NOT NULL AND status = 'POSTED'
         AND location_id IS NOT NULL));

CREATE UNIQUE INDEX goods_receipts_post_idempotency_key ON goods_receipts (posted_by, post_idempotency_key)
    WHERE post_idempotency_key IS NOT NULL;
CREATE INDEX goods_receipts_location_idx ON goods_receipts (location_id) WHERE location_id IS NOT NULL;

-- A posted receipt is immutable (PO-005): once POSTED (or CANCELLED) nothing on
-- the receipt row changes any more, and a DRAFT may only become POSTED with its
-- posting columns set together. (Lines, units, documents and events were
-- already guarded by 000012.)
CREATE FUNCTION goods_receipt_status_guard() RETURNS trigger AS $$
BEGIN
    IF OLD.status <> 'DRAFT' AND to_jsonb(NEW) IS DISTINCT FROM to_jsonb(OLD) THEN
        RAISE EXCEPTION 'a % goods receipt cannot change', OLD.status USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF OLD.status = 'DRAFT' AND NEW.status = 'DRAFT'
       AND (NEW.location_id IS NOT NULL OR NEW.post_idempotency_key IS NOT NULL) THEN
        RAISE EXCEPTION 'posting columns are set only by posting' USING ERRCODE = 'insufficient_privilege';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER goods_receipts_status_guard BEFORE UPDATE ON goods_receipts
    FOR EACH ROW EXECUTE FUNCTION goods_receipt_status_guard();

ALTER TABLE purchase_order_events DROP CONSTRAINT purchase_order_events_action_valid;
ALTER TABLE purchase_order_events ADD CONSTRAINT purchase_order_events_action_valid CHECK (action IN
    ('CREATED', 'SUBMITTED', 'APPROVED', 'APPROVAL_NOT_REQUIRED', 'RETURNED', 'CANCELLED', 'GOODS_RECEIVED'));
