-- Catalogue API permissions (C-02). Catalogue keys only: no role is granted
-- them (permission matrix RB-5 and approval owner OQ-2 pending), so every
-- catalogue endpoint is denied until the owner approves grants.
INSERT INTO permissions (key, description, sensitive) VALUES
    ('catalogue.view',             'View brands, models, variants and colours',                              false),
    ('catalogue.manage',           'Create and edit draft brands, models, variants and colours',             true),
    ('catalogue.verify',           'Record source and verification of models, variants and colours',         true),
    ('catalogue.activate',         'Activate verified models, variants and colours',                         true),
    ('catalogue.archive',          'Archive brands, models, variants and colours',                           true),
    ('catalogue.brand.authorise',  'Record brand supplier/legal identity and authorisation evidence',        true),
    ('catalogue.brand.activate',   'Enable a brand for purchase and sales',                                  true);
