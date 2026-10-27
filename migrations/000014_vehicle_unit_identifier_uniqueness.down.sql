-- Rolls back D-05: chassis/frame numbers are no longer unique (lookup index kept). No data changes.
DROP INDEX IF EXISTS vehicle_units_chassis_key;
CREATE INDEX IF NOT EXISTS vehicle_units_chassis_idx ON vehicle_units (chassis_normalised);
