-- Rolls back E-01. DESTRUCTIVE once movements exist: drops the stock movement
-- ledger (refused in staging/production without --confirm), and units already
-- moved past RECEIVED_PENDING_PDI keep their status with no history behind it.
-- Fails while a role holds the vehicle_units.display.manage grant. The unit
-- guard goes back to refusing every update (D-04).
DROP TRIGGER IF EXISTS vehicle_units_ledger_check ON vehicle_units;
DROP FUNCTION IF EXISTS vehicle_unit_ledger_check();
DROP TABLE IF EXISTS stock_movements;
DROP FUNCTION IF EXISTS stock_movement_apply();
DROP FUNCTION IF EXISTS stock_movements_reject_change();

CREATE OR REPLACE FUNCTION vehicle_unit_history_guard() RETURNS trigger AS $$
BEGIN
    IF TG_TABLE_NAME = 'vehicle_unit_events' THEN
        RAISE EXCEPTION 'vehicle unit history cannot change' USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'vehicle units are never deleted' USING ERRCODE = 'insufficient_privilege';
    END IF;
    RAISE EXCEPTION 'vehicle units change only through the stock movement service' USING ERRCODE = 'insufficient_privilege';
END;
$$ LANGUAGE plpgsql;

DELETE FROM permissions WHERE key = 'vehicle_units.display.manage';
