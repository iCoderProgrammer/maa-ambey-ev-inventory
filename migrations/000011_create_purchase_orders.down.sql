-- Rolls back D-02. DESTRUCTIVE: drops purchase orders, their lines, status
-- history and the approval settings (refused in staging/production without
-- --confirm). Fails while a role holds a purchase_orders.* grant.
DROP TABLE IF EXISTS purchase_order_events;
DROP TABLE IF EXISTS purchase_order_lines;
DROP TABLE IF EXISTS purchase_orders;
DROP TABLE IF EXISTS purchase_order_settings;
DROP SEQUENCE IF EXISTS purchase_order_number_seq;
DROP FUNCTION IF EXISTS purchase_order_history_guard();
DELETE FROM permissions WHERE key IN ('purchase_orders.view', 'purchase_orders.manage', 'purchase_orders.approve',
    'purchase_orders.cancel', 'purchase_orders.settings.manage');
