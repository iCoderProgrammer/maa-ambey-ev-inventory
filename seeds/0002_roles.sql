-- Client-confirmed role identities (A-05, docs/REQUIREMENTS_BASELINE.md §5).
-- Idempotent. Roles only: no permission is granted to any role, because the
-- permission matrix is pending owner sign-off (RB-5). Grants are added later,
-- with approval evidence, through cmd/access.
INSERT INTO roles (key, name, description)
VALUES
    ('owner',             'Owner',             'Client-confirmed role. Permissions pending owner sign-off.'),
    ('inventory_manager', 'Inventory Manager', 'Client-confirmed role. Permissions pending owner sign-off.'),
    ('sales_executive',   'Sales Executive',   'Client-confirmed role. Permissions pending owner sign-off.'),
    ('accountant',        'Accountant',        'Client-confirmed role. Permissions pending owner sign-off.'),
    ('viewer',            'Viewer',            'Client-confirmed role. Permissions pending owner sign-off.')
ON CONFLICT DO NOTHING;
