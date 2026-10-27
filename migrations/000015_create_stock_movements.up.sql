-- E-01: append-only stock movement ledger (ADR-020; INV-009, LED-001…LED-003).
-- One row per change of a vehicle unit's stock state: its registration from a
-- posted goods receipt (RECEIPT) and every later status change. Each row keeps
-- quantity, status and location before and after, the source, the actor, the
-- server time, and the idempotency key of the request that made it.
--
-- The ledger drives the snapshot: inserting a movement is what changes
-- vehicle_units.status / location_id (trigger below), so a unit cannot change
-- without its movement, and a movement cannot be recorded against a state the
-- unit is not in. At commit every unit must agree with its latest movement.
--
-- Allowed movement types are exactly the edges of the approved vehicle
-- lifecycle (System Architecture §6) that have an owner so far. Not here:
-- ORDERED/IN_TRANSIT (units are registered at RECEIVED_PENDING_PDI), reservation
-- and sale edges (F-04, F-07 add them), SERVICE_HOLD and returns (no approved
-- exit/approval rule), RECEIVED_PENDING_PDI -> REJECTED and QUALITY_HOLD ->
-- AVAILABLE (not in the diagram; PDI policy pending), location transfers
-- (one location; transfer approvals pending), reversals (no approved
-- correction flow: the column and its link exist, the CHECK keeps it empty).
-- Spare-part quantities are E-03/E-04: item_type is VEHICLE_UNIT only.

INSERT INTO permissions (key, description, sensitive) VALUES
    ('vehicle_units.display.manage', 'Put an AVAILABLE vehicle unit on display and return a DISPLAY unit to AVAILABLE (stock movement)', true);

CREATE TABLE stock_movements (
    id                   uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    seq                  bigint      GENERATED ALWAYS AS IDENTITY,
    item_type            text        NOT NULL DEFAULT 'VEHICLE_UNIT',
    vehicle_unit_id      uuid        NOT NULL REFERENCES vehicle_units (id) ON DELETE RESTRICT,
    movement_type        text        NOT NULL,
    -- A vehicle unit is one item: 0 before its receipt, 1 while it is held.
    quantity_before      smallint    NOT NULL,
    quantity_after       smallint    NOT NULL,
    from_status          text,
    to_status            text        NOT NULL,
    from_location_id     uuid        REFERENCES locations (id) ON DELETE RESTRICT,
    to_location_id       uuid        REFERENCES locations (id) ON DELETE RESTRICT,
    -- What caused it: a goods receipt (RECEIPT), or a staff action through the API.
    source_type          text        NOT NULL,
    source_id            uuid,
    source_reference     text,
    reverses_movement_id uuid,
    actor_id             uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    request_id           text,
    -- Per actor (API convention): a repeat returns the first result instead of
    -- moving the unit again. request_hash is the SHA-256 (hex) of the operation;
    -- the same key with a different operation is refused.
    idempotency_key      text        NOT NULL,
    request_hash         text        NOT NULL,
    note                 text,
    created_at           timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT stock_movements_unit_key UNIQUE (id, vehicle_unit_id),
    CONSTRAINT stock_movements_item_type_valid CHECK (item_type = 'VEHICLE_UNIT'),
    CONSTRAINT stock_movements_type_valid CHECK (movement_type IN
        ('RECEIPT', 'PDI_PASS', 'QUALITY_HOLD', 'REJECT', 'DISPLAY_ON', 'DISPLAY_OFF')),
    -- The approved lifecycle edges, one movement type each.
    CONSTRAINT stock_movements_transition_valid CHECK (
        (movement_type = 'RECEIPT'      AND from_status IS NULL                   AND to_status = 'RECEIVED_PENDING_PDI')
     OR (movement_type = 'PDI_PASS'     AND from_status = 'RECEIVED_PENDING_PDI'  AND to_status = 'AVAILABLE')
     OR (movement_type = 'QUALITY_HOLD' AND from_status = 'RECEIVED_PENDING_PDI'  AND to_status = 'QUALITY_HOLD')
     OR (movement_type = 'REJECT'       AND from_status = 'QUALITY_HOLD'          AND to_status = 'REJECTED')
     OR (movement_type = 'DISPLAY_ON'   AND from_status = 'AVAILABLE'             AND to_status = 'DISPLAY')
     OR (movement_type = 'DISPLAY_OFF'  AND from_status = 'DISPLAY'               AND to_status = 'AVAILABLE')),
    -- A receipt brings the unit in (0 -> 1) from nowhere; every other movement keeps it (1 -> 1) where it is.
    CONSTRAINT stock_movements_quantity_valid CHECK (
        (movement_type = 'RECEIPT' AND quantity_before = 0 AND quantity_after = 1 AND from_location_id IS NULL)
     OR (movement_type <> 'RECEIPT' AND quantity_before = 1 AND quantity_after = 1
         AND from_location_id IS NOT DISTINCT FROM to_location_id)),
    CONSTRAINT stock_movements_source_valid CHECK (source_type IN ('GOODS_RECEIPT', 'STAFF_ACTION')),
    CONSTRAINT stock_movements_source_complete CHECK (
        (source_type = 'GOODS_RECEIPT' AND movement_type = 'RECEIPT' AND source_id IS NOT NULL AND source_reference IS NOT NULL)
     OR (source_type = 'STAFF_ACTION' AND movement_type <> 'RECEIPT' AND source_id IS NULL AND source_reference IS NULL)),
    -- Corrections are new movements linked to the original (LED-002). No correction
    -- flow is approved yet, so the link stays empty until one adds its movement type.
    CONSTRAINT stock_movements_reversal_link_fk FOREIGN KEY (reverses_movement_id, vehicle_unit_id)
        REFERENCES stock_movements (id, vehicle_unit_id) ON DELETE RESTRICT,
    CONSTRAINT stock_movements_reversal_once UNIQUE (reverses_movement_id),
    CONSTRAINT stock_movements_reversal_not_approved CHECK (reverses_movement_id IS NULL),
    CONSTRAINT stock_movements_idempotency_key_valid CHECK (idempotency_key ~ '^[A-Za-z0-9_-]{8,100}$'),
    CONSTRAINT stock_movements_request_hash_valid CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT stock_movements_request_id_length CHECK (request_id IS NULL OR length(request_id) <= 64),
    CONSTRAINT stock_movements_note_length CHECK (note IS NULL OR (length(btrim(note)) > 0 AND length(note) <= 500))
);

CREATE UNIQUE INDEX stock_movements_idempotency_key ON stock_movements (actor_id, idempotency_key);
-- A unit is received once.
CREATE UNIQUE INDEX stock_movements_receipt_once ON stock_movements (vehicle_unit_id) WHERE movement_type = 'RECEIPT';
CREATE INDEX stock_movements_unit_idx ON stock_movements (vehicle_unit_id, seq);
CREATE INDEX stock_movements_source_idx ON stock_movements (source_type, source_id);
CREATE INDEX stock_movements_actor_idx ON stock_movements (actor_id, created_at DESC);
CREATE INDEX stock_movements_created_at_idx ON stock_movements (created_at DESC);

-- Append-only: no UPDATE, DELETE or TRUNCATE (LED-002).
CREATE FUNCTION stock_movements_reject_change() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'stock_movements is append-only: % is not allowed', TG_OP USING ERRCODE = 'insufficient_privilege';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER stock_movements_no_update_delete BEFORE UPDATE OR DELETE ON stock_movements
    FOR EACH ROW EXECUTE FUNCTION stock_movements_reject_change();
CREATE TRIGGER stock_movements_no_truncate BEFORE TRUNCATE ON stock_movements
    FOR EACH STATEMENT EXECUTE FUNCTION stock_movements_reject_change();

-- Applying a movement. The unit row is locked; its current state must be the
-- movement's "before" (a stale or concurrent change fails instead of
-- overwriting). A RECEIPT must be the unit's first movement, from its own goods
-- receipt, at the state it was registered in; it changes nothing. Any other
-- movement sets the unit's status and location (and bumps its version) — the
-- only UPDATE vehicle_units accepts (see vehicle_unit_history_guard).
CREATE FUNCTION stock_movement_apply() RETURNS trigger AS $$
DECLARE
    cur_status   text;
    cur_location uuid;
    cur_receipt  uuid;
BEGIN
    SELECT status, location_id, goods_receipt_id INTO cur_status, cur_location, cur_receipt
        FROM vehicle_units WHERE id = NEW.vehicle_unit_id FOR UPDATE;
    IF NEW.movement_type = 'RECEIPT' THEN
        IF cur_status IS DISTINCT FROM NEW.to_status OR cur_location IS DISTINCT FROM NEW.to_location_id
            OR cur_receipt IS DISTINCT FROM NEW.source_id
            OR EXISTS (SELECT 1 FROM stock_movements WHERE vehicle_unit_id = NEW.vehicle_unit_id AND id <> NEW.id) THEN
            RAISE EXCEPTION 'a receipt movement must be the unit''s first, from its own goods receipt'
                USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_movements_receipt_matches_unit';
        END IF;
        RETURN NULL;
    END IF;
    IF cur_status IS DISTINCT FROM NEW.from_status OR cur_location IS DISTINCT FROM NEW.from_location_id THEN
        RAISE EXCEPTION 'the vehicle unit is no longer in the movement''s starting state'
            USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_movements_from_matches_unit';
    END IF;
    PERFORM set_config('maa.stock_movement_unit', NEW.vehicle_unit_id::text, true);
    UPDATE vehicle_units SET status = NEW.to_status, location_id = NEW.to_location_id, version = version + 1, updated_at = now()
        WHERE id = NEW.vehicle_unit_id;
    PERFORM set_config('maa.stock_movement_unit', '', true);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER stock_movements_apply AFTER INSERT ON stock_movements
    FOR EACH ROW EXECUTE FUNCTION stock_movement_apply();

-- Units are still never deleted and their history never changes. The one
-- allowed UPDATE is the one stock_movement_apply makes: issued from inside that
-- trigger (trigger depth > 1, so no statement sent by a client qualifies even if
-- it sets the marker), for the unit it names, changing only status, location,
-- version (+1) and updated_at.
CREATE OR REPLACE FUNCTION vehicle_unit_history_guard() RETURNS trigger AS $$
BEGIN
    IF TG_TABLE_NAME = 'vehicle_unit_events' THEN
        RAISE EXCEPTION 'vehicle unit history cannot change' USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'vehicle units are never deleted' USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF pg_trigger_depth() > 1
        AND current_setting('maa.stock_movement_unit', true) = OLD.id::text
        AND NEW.version = OLD.version + 1
        AND (to_jsonb(NEW) - ARRAY['status', 'location_id', 'version', 'updated_at', 'chassis_normalised',
                'motor_normalised', 'battery_normalised', 'charger_normalised'])
          = (to_jsonb(OLD) - ARRAY['status', 'location_id', 'version', 'updated_at', 'chassis_normalised',
                'motor_normalised', 'battery_normalised', 'charger_normalised']) THEN
        RETURN NEW;
    END IF;
    RAISE EXCEPTION 'vehicle units change only through the stock movement service' USING ERRCODE = 'insufficient_privilege';
END;
$$ LANGUAGE plpgsql;

-- Units registered before this migration get their RECEIPT movement from the
-- registration they already have (same actor, time and goods receipt). They are
-- all still RECEIVED_PENDING_PDI: nothing could change them before E-01.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM vehicle_units WHERE status <> 'RECEIVED_PENDING_PDI' OR source_type <> 'GOODS_RECEIPT') THEN
        RAISE EXCEPTION 'vehicle units exist outside RECEIVED_PENDING_PDI or without a goods receipt; resolve them before applying E-01';
    END IF;
END $$;

INSERT INTO stock_movements (vehicle_unit_id, movement_type, quantity_before, quantity_after, from_status, to_status,
        to_location_id, source_type, source_id, source_reference, actor_id, idempotency_key, request_hash, created_at)
SELECT v.id, 'RECEIPT', 0, 1, NULL, v.status, v.location_id, 'GOODS_RECEIPT', v.goods_receipt_id, g.grn_number,
       v.registered_by, 'receipt-unit-' || v.goods_receipt_unit_id,
       encode(sha256(convert_to('RECEIPT|' || v.goods_receipt_unit_id, 'UTF8')), 'hex'), v.created_at
FROM vehicle_units v JOIN goods_receipts g ON g.id = v.goods_receipt_id
ORDER BY v.created_at, v.id;

-- LED-003, checked at commit: every unit has its RECEIPT movement and agrees
-- with its latest movement (status, location, still held).
CREATE FUNCTION vehicle_unit_ledger_check() RETURNS trigger AS $$
DECLARE
    unit_status   text;
    unit_location uuid;
    last          stock_movements%ROWTYPE;
BEGIN
    SELECT status, location_id INTO unit_status, unit_location FROM vehicle_units WHERE id = NEW.id;
    SELECT * INTO last FROM stock_movements WHERE vehicle_unit_id = NEW.id ORDER BY seq DESC LIMIT 1;
    IF NOT FOUND OR NOT EXISTS (SELECT 1 FROM stock_movements WHERE vehicle_unit_id = NEW.id AND movement_type = 'RECEIPT')
        OR last.to_status IS DISTINCT FROM unit_status OR last.to_location_id IS DISTINCT FROM unit_location
        OR last.quantity_after <> 1 THEN
        RAISE EXCEPTION 'vehicle unit % does not match its stock movements', NEW.id
            USING ERRCODE = 'check_violation', CONSTRAINT = 'vehicle_units_ledger_reconciles';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER vehicle_units_ledger_check AFTER INSERT OR UPDATE ON vehicle_units
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION vehicle_unit_ledger_check();
