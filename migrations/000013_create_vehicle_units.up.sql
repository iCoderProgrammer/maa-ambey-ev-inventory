-- D-04: vehicle unit registry (ADR-018). One row per physical scooter
-- (INV-001), tied to its verified catalogue variant (and so its brand), the
-- supplier, purchase order and goods receipt it came from, with the acquisition
-- cost behind its own permission.
--
-- Nothing here creates a unit: the only source is a POSTED goods receipt's
-- accepted unit, written by receipt posting (D-07) in the posting transaction.
-- A draft goods receipt never has a vehicle unit (deferred check below).
-- No unit, identifier or grant is seeded.
--
-- Deliberately not here: chassis/frame uniqueness and motor/battery uniqueness
-- policies (D-05), status/location changes (only through the stock movement
-- service, E-01, INV-009), PDI fields (D-06), colour (not captured on receipt).

INSERT INTO permissions (key, description, sensitive) VALUES
    ('vehicle_units.view',      'View vehicle units: identifiers, status, location and source supplier, purchase order and goods receipt', true),
    ('vehicle_units.cost.view', 'See the recorded acquisition cost of vehicle units', true);

-- Internal unit number. MAE-VEH- follows the design specification example
-- (MAE-VEH-00042); the exact format is pending confirmation (OQ-28).
CREATE SEQUENCE vehicle_unit_number_seq;

-- Keys that let vehicle_units prove, by foreign key, that its references belong together.
ALTER TABLE purchase_orders ADD CONSTRAINT purchase_orders_id_supplier_key UNIQUE (id, supplier_id);
ALTER TABLE purchase_order_lines ADD CONSTRAINT purchase_order_lines_id_variant_key UNIQUE (id, variant_id);
ALTER TABLE goods_receipt_lines ADD CONSTRAINT goods_receipt_lines_id_po_line_key UNIQUE (id, purchase_order_line_id);
ALTER TABLE goods_receipt_units ADD CONSTRAINT goods_receipt_units_id_line_key UNIQUE (id, goods_receipt_line_id);

CREATE TABLE vehicle_units (
    id                     uuid          PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_number            text          NOT NULL DEFAULT ('MAE-VEH-' || lpad(nextval('vehicle_unit_number_seq')::text, 6, '0')),
    brand_id               uuid          NOT NULL,
    model_id               uuid          NOT NULL,
    variant_id             uuid          NOT NULL,
    -- Approved lifecycle statuses (INV-005). Units are registered at
    -- RECEIVED_PENDING_PDI; changes go through the stock movement service (E-01).
    status                 text          NOT NULL DEFAULT 'RECEIVED_PENDING_PDI',
    -- Identifiers as captured, and their normalised form (INV-002: trim and case;
    -- which separators may be ignored is pending, OQ-26). Same rule as
    -- goods_receipt_units.chassis_normalised. Uniqueness is D-05.
    chassis_number         text,
    chassis_normalised     text          GENERATED ALWAYS AS (upper(btrim(chassis_number))) STORED,
    motor_serial           text,
    motor_normalised       text          GENERATED ALWAYS AS (upper(btrim(motor_serial))) STORED,
    battery_serial         text,
    battery_normalised     text          GENERATED ALWAYS AS (upper(btrim(battery_serial))) STORED,
    charger_serial         text,
    charger_normalised     text          GENERATED ALWAYS AS (upper(btrim(charger_serial))) STORED,
    -- Optional until receipts carry a location (OQ-24). With a location, readers
    -- need it in their location scope (B-07).
    location_id            uuid          REFERENCES locations (id) ON DELETE RESTRICT,
    -- Where the unit came from. GOODS_RECEIPT is the only source so far; opening
    -- stock (G-07) will add its own.
    source_type            text          NOT NULL,
    supplier_id            uuid          REFERENCES suppliers (id) ON DELETE RESTRICT,
    purchase_order_id      uuid,
    purchase_order_line_id uuid,
    goods_receipt_id       uuid,
    goods_receipt_line_id  uuid,
    goods_receipt_unit_id  uuid,
    received_date          date          NOT NULL,
    -- Acquisition cost (REP-003). Recorded from the PO line's agreed unit cost
    -- until the cost basis is confirmed (OQ-29). Shown only with vehicle_units.cost.view.
    unit_cost              numeric(12,2) NOT NULL,
    cost_currency          text          NOT NULL DEFAULT 'INR',
    registered_by          uuid          NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    version                integer       NOT NULL DEFAULT 1,
    created_at             timestamptz   NOT NULL DEFAULT now(),
    updated_at             timestamptz   NOT NULL DEFAULT now(),

    CONSTRAINT vehicle_units_number_key UNIQUE (unit_number),
    -- Brand, model and variant belong together (C-06).
    CONSTRAINT vehicle_units_model_fk FOREIGN KEY (model_id, brand_id) REFERENCES models (id, brand_id) ON DELETE RESTRICT,
    CONSTRAINT vehicle_units_variant_brand_fk FOREIGN KEY (variant_id, brand_id) REFERENCES variants (id, brand_id) ON DELETE RESTRICT,
    CONSTRAINT vehicle_units_variant_model_fk FOREIGN KEY (variant_id, model_id) REFERENCES variants (id, model_id) ON DELETE RESTRICT,
    -- The PO is the supplier's, the PO line is the PO's and is for this variant,
    -- the receipt is against the PO, the receipt line is the receipt's and is for
    -- the PO line, and the captured unit is on the receipt line.
    CONSTRAINT vehicle_units_po_supplier_fk FOREIGN KEY (purchase_order_id, supplier_id)
        REFERENCES purchase_orders (id, supplier_id) ON DELETE RESTRICT,
    CONSTRAINT vehicle_units_po_line_fk FOREIGN KEY (purchase_order_line_id, purchase_order_id)
        REFERENCES purchase_order_lines (id, purchase_order_id) ON DELETE RESTRICT,
    CONSTRAINT vehicle_units_po_line_variant_fk FOREIGN KEY (purchase_order_line_id, variant_id)
        REFERENCES purchase_order_lines (id, variant_id) ON DELETE RESTRICT,
    CONSTRAINT vehicle_units_receipt_fk FOREIGN KEY (goods_receipt_id, purchase_order_id)
        REFERENCES goods_receipts (id, purchase_order_id) ON DELETE RESTRICT,
    CONSTRAINT vehicle_units_receipt_line_fk FOREIGN KEY (goods_receipt_line_id, goods_receipt_id)
        REFERENCES goods_receipt_lines (id, goods_receipt_id) ON DELETE RESTRICT,
    CONSTRAINT vehicle_units_receipt_line_po_line_fk FOREIGN KEY (goods_receipt_line_id, purchase_order_line_id)
        REFERENCES goods_receipt_lines (id, purchase_order_line_id) ON DELETE RESTRICT,
    CONSTRAINT vehicle_units_receipt_unit_fk FOREIGN KEY (goods_receipt_unit_id, goods_receipt_line_id)
        REFERENCES goods_receipt_units (id, goods_receipt_line_id) ON DELETE RESTRICT,
    -- A captured unit becomes at most one vehicle unit (a repeated post cannot duplicate it).
    CONSTRAINT vehicle_units_receipt_unit_key UNIQUE (goods_receipt_unit_id),
    CONSTRAINT vehicle_units_status_valid CHECK (status IN ('ORDERED', 'IN_TRANSIT', 'RECEIVED_PENDING_PDI', 'QUALITY_HOLD',
        'AVAILABLE', 'DISPLAY', 'RESERVED', 'SERVICE_HOLD', 'SOLD', 'DELIVERED', 'REJECTED', 'RETURN_PENDING_INSPECTION',
        'RETURNED_TO_SUPPLIER')),
    -- INV-002: a chassis/frame number is mandatory before a unit leaves RECEIVED_PENDING_PDI.
    CONSTRAINT vehicle_units_chassis_after_receipt CHECK (chassis_number IS NOT NULL
        OR status IN ('ORDERED', 'IN_TRANSIT', 'RECEIVED_PENDING_PDI')),
    -- Same limits as goods_receipt_units; blank is stored as NULL.
    CONSTRAINT vehicle_units_identifier_length CHECK (
        (chassis_number IS NULL OR length(btrim(chassis_number)) BETWEEN 1 AND 50)
        AND (motor_serial IS NULL OR length(btrim(motor_serial)) BETWEEN 1 AND 50)
        AND (battery_serial IS NULL OR length(btrim(battery_serial)) BETWEEN 1 AND 50)
        AND (charger_serial IS NULL OR length(btrim(charger_serial)) BETWEEN 1 AND 50)),
    CONSTRAINT vehicle_units_source_valid CHECK (source_type IN ('GOODS_RECEIPT')),
    CONSTRAINT vehicle_units_goods_receipt_source_complete CHECK (source_type <> 'GOODS_RECEIPT' OR (
        supplier_id IS NOT NULL AND purchase_order_id IS NOT NULL AND purchase_order_line_id IS NOT NULL
        AND goods_receipt_id IS NOT NULL AND goods_receipt_line_id IS NOT NULL AND goods_receipt_unit_id IS NOT NULL)),
    CONSTRAINT vehicle_units_cost_positive CHECK (unit_cost > 0),
    CONSTRAINT vehicle_units_cost_currency_valid CHECK (cost_currency = 'INR'),
    CONSTRAINT vehicle_units_version_positive CHECK (version > 0)
);

CREATE INDEX vehicle_units_brand_idx ON vehicle_units (brand_id);
CREATE INDEX vehicle_units_variant_idx ON vehicle_units (variant_id);
CREATE INDEX vehicle_units_status_idx ON vehicle_units (status);
CREATE INDEX vehicle_units_location_idx ON vehicle_units (location_id);
CREATE INDEX vehicle_units_supplier_idx ON vehicle_units (supplier_id);
CREATE INDEX vehicle_units_purchase_order_idx ON vehicle_units (purchase_order_id);
CREATE INDEX vehicle_units_goods_receipt_idx ON vehicle_units (goods_receipt_id);
CREATE INDEX vehicle_units_received_date_idx ON vehicle_units (received_date);
-- Lookup by identifier. Not unique here: D-05 owns the uniqueness policy.
CREATE INDEX vehicle_units_chassis_idx ON vehicle_units (chassis_normalised);
CREATE INDEX vehicle_units_motor_idx ON vehicle_units (motor_normalised);
CREATE INDEX vehicle_units_battery_idx ON vehicle_units (battery_normalised);
CREATE INDEX vehicle_units_charger_idx ON vehicle_units (charger_normalised);

-- Unit history: append-only, each entry pointing at its source document.
CREATE TABLE vehicle_unit_events (
    id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    seq              bigint      GENERATED ALWAYS AS IDENTITY,
    vehicle_unit_id  uuid        NOT NULL REFERENCES vehicle_units (id) ON DELETE RESTRICT,
    action           text        NOT NULL,
    to_status        text        NOT NULL,
    source_type      text        NOT NULL,
    source_id        uuid        NOT NULL,
    source_reference text        NOT NULL,
    actor_id         uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    note             text,
    created_at       timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT vehicle_unit_events_action_valid CHECK (action IN ('REGISTERED')),
    CONSTRAINT vehicle_unit_events_source_valid CHECK (source_type IN ('GOODS_RECEIPT')),
    CONSTRAINT vehicle_unit_events_note_length CHECK (note IS NULL OR (length(btrim(note)) > 0 AND length(note) <= 500))
);
CREATE INDEX vehicle_unit_events_unit_idx ON vehicle_unit_events (vehicle_unit_id, seq);

-- Units are never deleted, and nothing changes them yet: status and location
-- change only through the stock movement service (E-01, INV-009), identifier
-- corrections need an approved flow (D-05). Those tasks replace this guard.
CREATE FUNCTION vehicle_unit_history_guard() RETURNS trigger AS $$
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

CREATE TRIGGER vehicle_units_guard BEFORE UPDATE OR DELETE ON vehicle_units FOR EACH ROW EXECUTE FUNCTION vehicle_unit_history_guard();
CREATE TRIGGER vehicle_unit_events_immutable BEFORE UPDATE OR DELETE ON vehicle_unit_events FOR EACH ROW EXECUTE FUNCTION vehicle_unit_history_guard();

-- Checked at commit, so posting may register units and mark the receipt POSTED
-- in either order inside its transaction: a unit's goods receipt is POSTED, its
-- captured unit was ACCEPTED, and its received date is the receipt's.
CREATE FUNCTION vehicle_unit_source_check() RETURNS trigger AS $$
DECLARE
    receipt_status text;
    receipt_date   date;
    unit_condition text;
BEGIN
    IF NEW.source_type = 'GOODS_RECEIPT' THEN
        SELECT g.status, g.received_date INTO receipt_status, receipt_date FROM goods_receipts g WHERE g.id = NEW.goods_receipt_id;
        IF receipt_status IS DISTINCT FROM 'POSTED' THEN
            RAISE EXCEPTION 'a vehicle unit needs a posted goods receipt (receipt is %)', coalesce(receipt_status, 'missing')
                USING ERRCODE = 'check_violation', CONSTRAINT = 'vehicle_units_receipt_posted';
        END IF;
        IF receipt_date IS DISTINCT FROM NEW.received_date THEN
            RAISE EXCEPTION 'a vehicle unit''s received date is its goods receipt''s'
                USING ERRCODE = 'check_violation', CONSTRAINT = 'vehicle_units_received_date_matches';
        END IF;
        SELECT u.condition INTO unit_condition FROM goods_receipt_units u WHERE u.id = NEW.goods_receipt_unit_id;
        IF unit_condition IS DISTINCT FROM 'ACCEPTED' THEN
            RAISE EXCEPTION 'only an accepted unit becomes a vehicle unit'
                USING ERRCODE = 'check_violation', CONSTRAINT = 'vehicle_units_receipt_unit_accepted';
        END IF;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER vehicle_units_source_check AFTER INSERT ON vehicle_units
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION vehicle_unit_source_check();
