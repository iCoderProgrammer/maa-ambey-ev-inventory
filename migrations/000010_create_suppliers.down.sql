-- Rolls back D-01. DESTRUCTIVE: drops suppliers, contacts, document links and
-- supplier-brand relationships (refused in staging/production without --confirm).
-- Linked attachments keep their owner marking (attachments are immutable).
-- Fails while a role holds a suppliers.* grant.
DROP TABLE IF EXISTS supplier_brand_links;
ALTER TABLE IF EXISTS suppliers DROP CONSTRAINT IF EXISTS suppliers_verification_document_fk;
DROP TABLE IF EXISTS supplier_documents;
DROP TABLE IF EXISTS supplier_contacts;
DROP TABLE IF EXISTS suppliers;
DROP FUNCTION IF EXISTS supplier_history_guard();
DELETE FROM permissions WHERE key IN ('suppliers.view', 'suppliers.manage', 'suppliers.verify', 'suppliers.archive');
