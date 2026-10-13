-- Rolls back C-02 permissions. Fails if a role still holds a grant for one of
-- them (role_permissions FK), so approved access is never removed silently.
DELETE FROM permissions WHERE key IN ('catalogue.view', 'catalogue.manage', 'catalogue.verify', 'catalogue.activate',
    'catalogue.archive', 'catalogue.brand.authorise', 'catalogue.brand.activate');
