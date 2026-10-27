-- Rolls back E-03. DESTRUCTIVE once SKUs exist: drops item categories, SKUs,
-- their supplier links and compatibility (refused in staging/production without
-- --confirm). Fails while a role holds a stock_items.* grant.
DROP TABLE IF EXISTS stock_item_compatibility;
DROP TABLE IF EXISTS stock_item_suppliers;
DROP TABLE IF EXISTS stock_items;
DROP TABLE IF EXISTS stock_item_categories;
DROP FUNCTION IF EXISTS stock_item_reference_check();
DROP FUNCTION IF EXISTS stock_item_history_guard();
DELETE FROM permissions WHERE key IN ('stock_items.view', 'stock_items.manage', 'stock_items.archive');
