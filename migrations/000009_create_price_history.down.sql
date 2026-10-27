-- Rolls back C-04. DESTRUCTIVE: drops all price lists and price history
-- (refused in staging/production without --confirm). Fails if a role still
-- holds a catalogue.price.* grant, so approved access is never removed silently.
-- btree_gist is left installed (harmless, may be used elsewhere).
DROP TABLE IF EXISTS price_items;
DROP FUNCTION IF EXISTS price_items_protect_history();
DROP TABLE IF EXISTS price_lists;
ALTER TABLE variants DROP CONSTRAINT IF EXISTS variants_id_model_key;
DELETE FROM permissions WHERE key IN ('catalogue.price.view', 'catalogue.price.manage', 'catalogue.price.approve');
