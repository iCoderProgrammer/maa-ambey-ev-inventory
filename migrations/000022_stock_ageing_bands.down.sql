-- Rolls back E-07: removes the ageing band configuration and its permission.
-- Refuses while any configuration other than the migration's own proposal
-- exists (a proposal or approval made by a person is history and is never
-- destroyed by a rollback). Fails while a role holds stock.ageing.configure.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM stock_ageing_band_sets WHERE version <> 1 OR status <> 'PROPOSED' OR created_by IS NOT NULL) THEN
        RAISE EXCEPTION 'ageing band configurations were proposed or approved; rolling back E-07 would destroy that history';
    END IF;
END $$;

DROP TABLE IF EXISTS stock_ageing_bands;
DROP TABLE IF EXISTS stock_ageing_band_sets;
DROP FUNCTION IF EXISTS stock_ageing_band_set_has_bands();
DROP FUNCTION IF EXISTS stock_ageing_band_set_check();
DROP FUNCTION IF EXISTS stock_ageing_history_guard();

DELETE FROM permissions WHERE key = 'stock.ageing.configure';
