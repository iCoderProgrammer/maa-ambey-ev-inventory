-- Rolls back G-06. Refuses while any import batch exists: previews and their
-- uploaded files are history and are never destroyed by a rollback. Fails while
-- a role holds stock.import.preview.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM opening_stock_import_batches) OR EXISTS (SELECT 1 FROM attachments WHERE content_type = 'text/csv') THEN
        RAISE EXCEPTION 'opening stock import previews exist; rolling back G-06 would destroy them';
    END IF;
END $$;

DROP TABLE IF EXISTS opening_stock_import_rows;
DROP TABLE IF EXISTS opening_stock_import_batches;
DROP FUNCTION IF EXISTS opening_stock_import_guard();
DROP SEQUENCE IF EXISTS opening_stock_import_number_seq;

ALTER TABLE attachments DROP CONSTRAINT attachments_content_type_valid;
ALTER TABLE attachments ADD CONSTRAINT attachments_content_type_valid
    CHECK (content_type IN ('image/jpeg', 'image/png', 'image/webp', 'application/pdf'));

DELETE FROM permissions WHERE key = 'stock.import.preview';
