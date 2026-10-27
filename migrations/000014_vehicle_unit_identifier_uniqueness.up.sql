-- D-05: duplicate vehicle identifier controls (ADR-019).
--
-- A normalised chassis/frame number belongs to at most one vehicle unit, across
-- every brand (INV-002; architecture: "chassis/VIN remains globally unique").
-- The normalised form is the generated column from migration 000013
-- (upper(btrim(chassis_number))), so no caller can store a value that
-- disagrees with it; separators are kept until OQ-26 is decided. Units without
-- a chassis number (allowed while RECEIVED_PENDING_PDI) never conflict: a
-- unique index ignores NULLs, and a blank number is refused by
-- vehicle_units_identifier_length.
--
-- Motor, battery and charger numbers get no uniqueness rule: no approved
-- policy exists yet (INV-003, INV-004, OQ-4, OQ-30). Their lookup indexes stay.

-- Refuse to proceed over existing duplicates instead of changing any unit.
DO $$
DECLARE dup text;
BEGIN
    SELECT string_agg(chassis_normalised, ', ' ORDER BY chassis_normalised) INTO dup FROM (
        SELECT chassis_normalised FROM vehicle_units WHERE chassis_normalised IS NOT NULL
        GROUP BY chassis_normalised HAVING count(*) > 1) d;
    IF dup IS NOT NULL THEN
        RAISE EXCEPTION 'vehicle units share a chassis/frame number (%); resolve them before applying D-05', dup
            USING ERRCODE = 'unique_violation';
    END IF;
END $$;

DROP INDEX vehicle_units_chassis_idx;
-- Stable name: the application maps a violation of exactly this index to DUPLICATE_IDENTIFIER (chassisNumber).
CREATE UNIQUE INDEX vehicle_units_chassis_key ON vehicle_units (chassis_normalised);
