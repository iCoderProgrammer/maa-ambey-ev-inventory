-- Role-based access control and location scope (B-07, SEC-001..SEC-004, ADR-007).
--
-- Deny by default: a user can do only what an APPROVED grant on one of their
-- roles allows. The permission matrix is pending owner sign-off (RB-5), so this
-- migration grants nothing. Grants, role assignments, locations and location
-- scope are entered later as data through the audited cmd/access tool.

-- Permission catalogue: the actions the code can protect. Keys are referenced by
-- code (internal/access/permissions.go); a test keeps both in step.
CREATE TABLE permissions (
    key         text        PRIMARY KEY,
    description text        NOT NULL,
    sensitive   boolean     NOT NULL DEFAULT false,
    created_at  timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT permissions_key_format CHECK (key ~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$' AND length(key) <= 80),
    CONSTRAINT permissions_description_not_blank CHECK (length(btrim(description)) > 0)
);

INSERT INTO permissions (key, description, sensitive) VALUES
    ('access.roles.view',          'View roles, permission grants and their approval status', false),
    ('access.roles.manage',        'Change role grants and assign roles to users',              true),
    ('audit.events.view',          'View the audit log',                                        true),
    ('locations.all',              'Access every location (multi-location scope)',              true),
    ('catalogue.cost_price.view',  'View cost prices and stock valuation',                      true),
    ('catalogue.cost_price.edit',  'Edit cost prices',                                          true),
    ('sales.discount.approve',     'Approve discounts above the role limit',                    true),
    ('sales.refund.create',        'Create refunds and reversals',                              true),
    ('sales.refund.approve',       'Approve refunds and reversals',                             true),
    ('stock.adjustment.create',    'Request stock adjustments',                                 true),
    ('stock.adjustment.approve',   'Approve stock adjustments (never one''s own)',              true),
    ('settings.invoice.manage',    'Manage business profile, invoice series and templates',     true),
    ('settings.tax.manage',        'Manage tax configuration',                                  true),
    ('reports.export',             'Export reports',                                            true),
    ('customers.export',           'Export customer data',                                      true);

CREATE TABLE roles (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    key         text        NOT NULL,
    name        text        NOT NULL,
    description text        NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT roles_key_format CHECK (key ~ '^[a-z][a-z0-9_]{1,49}$'),
    CONSTRAINT roles_name_not_blank CHECK (length(btrim(name)) > 0 AND length(name) <= 80)
);
CREATE UNIQUE INDEX roles_key_key ON roles (key);
CREATE UNIQUE INDEX roles_name_lower_key ON roles (lower(name));

-- A grant counts only when APPROVED with the approver and evidence recorded.
-- PROPOSED grants are ignored unless RBAC_ALLOW_PROPOSED_GRANTS is on, which
-- the API refuses in staging and production.
CREATE TABLE role_permissions (
    role_id            uuid        NOT NULL REFERENCES roles (id) ON DELETE RESTRICT,
    permission_key     text        NOT NULL REFERENCES permissions (key) ON DELETE RESTRICT,
    status             text        NOT NULL DEFAULT 'PROPOSED',
    approved_by        text,
    approval_reference text,
    approved_at        timestamptz,
    created_at         timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (role_id, permission_key),

    CONSTRAINT role_permissions_status_valid CHECK (status IN ('PROPOSED', 'APPROVED')),
    CONSTRAINT role_permissions_approval_evidence CHECK (
        (status = 'PROPOSED' AND approved_by IS NULL AND approval_reference IS NULL AND approved_at IS NULL)
        OR (status = 'APPROVED' AND length(btrim(approved_by)) > 0 AND length(btrim(approval_reference)) > 0
            AND approved_at IS NOT NULL)),
    CONSTRAINT role_permissions_evidence_length CHECK (
        coalesce(length(approved_by), 0) <= 120 AND coalesce(length(approval_reference), 0) <= 200)
);
CREATE INDEX role_permissions_permission_idx ON role_permissions (permission_key);

CREATE TABLE user_roles (
    user_id     uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    role_id     uuid        NOT NULL REFERENCES roles (id) ON DELETE RESTRICT,
    assigned_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role_id)
);
CREATE INDEX user_roles_role_idx ON user_roles (role_id);

-- Locations (showroom, store …). Only identity here; address/type and the
-- initial showroom's details are pending client input, so nothing is seeded.
CREATE TABLE locations (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    code       text        NOT NULL,
    name       text        NOT NULL,
    active     boolean     NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT locations_code_format CHECK (code ~ '^[A-Z0-9][A-Z0-9-]{1,19}$'),
    CONSTRAINT locations_name_not_blank CHECK (length(btrim(name)) > 0 AND length(name) <= 120)
);
CREATE UNIQUE INDEX locations_code_key ON locations (code);

CREATE TRIGGER locations_set_updated_at
    BEFORE UPDATE ON locations
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Locations a user may work in (SEC-002). The locations.all permission grants every location.
CREATE TABLE user_location_scopes (
    user_id     uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    location_id uuid        NOT NULL REFERENCES locations (id) ON DELETE RESTRICT,
    assigned_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, location_id)
);
CREATE INDEX user_location_scopes_location_idx ON user_location_scopes (location_id);
