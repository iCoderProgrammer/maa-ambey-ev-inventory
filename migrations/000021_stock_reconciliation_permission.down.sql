-- Rolls back E-06: removes the reconciliation permission. Reconciliation stored
-- no data, so nothing else is lost. Fails while a role holds the permission
-- (role_permissions refers to it): revoke it first.
DELETE FROM permissions WHERE key = 'stock.reconciliation.view';
