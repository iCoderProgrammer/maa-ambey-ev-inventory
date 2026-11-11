-- PDI checklist and disposition (D-06; ADR-024; INV-008, INV-009, FR-006).
--
-- pdi_checklist_items: the configurable checklist. Data only: no item is seeded
-- (the client has not supplied an approved checklist). An item applies to every
-- brand (brand_id NULL) or to one brand. Items are never edited or deleted: an
-- item is retired and a new one created, so recorded outcomes keep their meaning.
--
-- pdi_inspections: one completed inspection of one vehicle unit, with its
-- disposition PASS / HOLD / REJECT, the actor and database time, notes, and the
-- stock movement that applied it (PDI_PASS, QUALITY_HOLD or REJECT, source
-- PDI_INSPECTION). Append-only: an inspection, its outcomes and its evidence
-- links are never updated or deleted.
--
-- Lifecycle (System Architecture §6, E-01 ADR-020): PASS = RECEIVED_PENDING_PDI ->
-- AVAILABLE, HOLD = RECEIVED_PENDING_PDI -> QUALITY_HOLD, REJECT = QUALITY_HOLD ->
-- REJECTED. RECEIVED_PENDING_PDI -> REJECTED and QUALITY_HOLD -> AVAILABLE are not
-- approved (OQ-31) and stay refused by the ledger CHECKs (unchanged here).
--
-- No grant is created for the two new permissions (RB-5).

INSERT INTO permissions (key, description, sensitive) VALUES
    ('vehicle_units.pdi.record',           'Record a PDI inspection and its disposition (PASS / HOLD / REJECT)', true),
    ('vehicle_units.pdi.checklist.manage', 'Create and retire PDI checklist items',                              true);

-- ── Checklist configuration ─────────────────────────────────────────────────

CREATE TABLE pdi_checklist_items (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    code          text        NOT NULL,
    label         text        NOT NULL,
    description   text,
    -- NULL = applies to units of every brand.
    brand_id      uuid        REFERENCES brands (id) ON DELETE RESTRICT,
    sort_order    integer     NOT NULL DEFAULT 0,
    status        text        NOT NULL DEFAULT 'ACTIVE',
    version       integer     NOT NULL DEFAULT 1,
    created_by    uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_at    timestamptz NOT NULL DEFAULT now(),
    retired_by    uuid        REFERENCES users (id) ON DELETE RESTRICT,
    retired_at    timestamptz,
    retire_reason text,

    CONSTRAINT pdi_checklist_items_code_format CHECK (code ~ '^[A-Z][A-Z0-9_]{1,39}$'),
    CONSTRAINT pdi_checklist_items_label_length CHECK (length(btrim(label)) BETWEEN 1 AND 200 AND label = btrim(label)),
    CONSTRAINT pdi_checklist_items_description_length CHECK (description IS NULL
        OR (length(btrim(description)) BETWEEN 1 AND 1000 AND description = btrim(description))),
    CONSTRAINT pdi_checklist_items_sort_order_range CHECK (sort_order BETWEEN 0 AND 9999),
    CONSTRAINT pdi_checklist_items_status_valid CHECK (status IN ('ACTIVE', 'RETIRED')),
    CONSTRAINT pdi_checklist_items_version_positive CHECK (version >= 1),
    CONSTRAINT pdi_checklist_items_retired_complete CHECK (
        (status = 'ACTIVE' AND retired_by IS NULL AND retired_at IS NULL AND retire_reason IS NULL)
     OR (status = 'RETIRED' AND retired_by IS NOT NULL AND retired_at IS NOT NULL
         AND retire_reason IS NOT NULL AND length(btrim(retire_reason)) BETWEEN 1 AND 500))
);
-- One active item per code and brand scope (all-brands items share one scope).
CREATE UNIQUE INDEX pdi_checklist_items_active_code_key ON pdi_checklist_items (code, brand_id) NULLS NOT DISTINCT
    WHERE status = 'ACTIVE';
CREATE INDEX pdi_checklist_items_brand_idx ON pdi_checklist_items (brand_id, status, sort_order);

-- Never deleted; the only change is ACTIVE -> RETIRED with who, when and why.
CREATE FUNCTION pdi_checklist_item_guard() RETURNS trigger AS $$
BEGIN
    IF TG_OP IN ('DELETE', 'TRUNCATE') THEN
        RAISE EXCEPTION 'PDI checklist items cannot be deleted (% refused); retire them', TG_OP
            USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF OLD.status <> 'ACTIVE' OR NEW.status <> 'RETIRED'
       OR NEW.id IS DISTINCT FROM OLD.id OR NEW.code IS DISTINCT FROM OLD.code
       OR NEW.label IS DISTINCT FROM OLD.label OR NEW.description IS DISTINCT FROM OLD.description
       OR NEW.brand_id IS DISTINCT FROM OLD.brand_id OR NEW.sort_order IS DISTINCT FROM OLD.sort_order
       OR NEW.created_by IS DISTINCT FROM OLD.created_by OR NEW.created_at IS DISTINCT FROM OLD.created_at
       OR NEW.version <> OLD.version + 1 THEN
        RAISE EXCEPTION 'a PDI checklist item can only be retired, once'
            USING ERRCODE = 'insufficient_privilege';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER pdi_checklist_items_guard BEFORE UPDATE OR DELETE ON pdi_checklist_items
    FOR EACH ROW EXECUTE FUNCTION pdi_checklist_item_guard();
CREATE TRIGGER pdi_checklist_items_no_truncate BEFORE TRUNCATE ON pdi_checklist_items
    FOR EACH STATEMENT EXECUTE FUNCTION pdi_checklist_item_guard();

-- ── Inspections ─────────────────────────────────────────────────────────────

CREATE SEQUENCE pdi_inspection_number_seq;

CREATE TABLE pdi_inspections (
    id                uuid        PRIMARY KEY,
    inspection_number text        NOT NULL,
    vehicle_unit_id   uuid        NOT NULL REFERENCES vehicle_units (id) ON DELETE RESTRICT,
    disposition       text        NOT NULL,
    -- Summary of the recorded checklist outcomes (server-computed).
    result            text        NOT NULL,
    from_status       text        NOT NULL,
    to_status         text        NOT NULL,
    -- The unit's location when inspected (NULL = the unit has none).
    location_id       uuid        REFERENCES locations (id) ON DELETE RESTRICT,
    notes             text,
    movement_id       uuid        NOT NULL REFERENCES stock_movements (id) ON DELETE RESTRICT,
    inspected_by      uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    inspected_at      timestamptz NOT NULL DEFAULT now(),
    request_id        text,
    idempotency_key   text        NOT NULL,
    request_hash      text        NOT NULL,

    CONSTRAINT pdi_inspections_number_format CHECK (inspection_number ~ '^PDI-[0-9]{6,}$'),
    CONSTRAINT pdi_inspections_disposition_valid CHECK (disposition IN ('PASS', 'HOLD', 'REJECT')),
    CONSTRAINT pdi_inspections_result_valid CHECK (result IN ('ALL_PASSED', 'HAS_FAILURES', 'NOT_CHECKED')),
    -- The approved lifecycle edges only (OQ-31: no direct reject, no release from hold).
    CONSTRAINT pdi_inspections_transition_valid CHECK (
        (disposition = 'PASS'   AND from_status = 'RECEIVED_PENDING_PDI' AND to_status = 'AVAILABLE')
     OR (disposition = 'HOLD'   AND from_status = 'RECEIVED_PENDING_PDI' AND to_status = 'QUALITY_HOLD')
     OR (disposition = 'REJECT' AND from_status = 'QUALITY_HOLD'         AND to_status = 'REJECTED')),
    -- INV-008: a pass needs a complete, all-passed checklist; a hold or reject needs issue notes.
    CONSTRAINT pdi_inspections_pass_needs_checklist CHECK (disposition <> 'PASS' OR result = 'ALL_PASSED'),
    CONSTRAINT pdi_inspections_notes_valid CHECK (
        (notes IS NULL OR (length(btrim(notes)) BETWEEN 1 AND 2000 AND notes = btrim(notes)))
        AND (disposition = 'PASS' OR notes IS NOT NULL)),
    CONSTRAINT pdi_inspections_idempotency_key_format CHECK (idempotency_key ~ '^[A-Za-z0-9_-]{8,100}$')
);
CREATE UNIQUE INDEX pdi_inspections_number_key ON pdi_inspections (inspection_number);
CREATE UNIQUE INDEX pdi_inspections_movement_key ON pdi_inspections (movement_id);
CREATE UNIQUE INDEX pdi_inspections_idempotency_key ON pdi_inspections (inspected_by, idempotency_key);
CREATE INDEX pdi_inspections_unit_idx ON pdi_inspections (vehicle_unit_id, inspected_at DESC);
CREATE INDEX pdi_inspections_holds_idx ON pdi_inspections (disposition, inspected_at DESC);

CREATE TABLE pdi_inspection_items (
    inspection_id     uuid    NOT NULL REFERENCES pdi_inspections (id) ON DELETE RESTRICT,
    checklist_item_id uuid    NOT NULL REFERENCES pdi_checklist_items (id) ON DELETE RESTRICT,
    line_no           integer NOT NULL,
    outcome           text    NOT NULL,
    note              text,

    PRIMARY KEY (inspection_id, checklist_item_id),
    CONSTRAINT pdi_inspection_items_line_no_key UNIQUE (inspection_id, line_no),
    CONSTRAINT pdi_inspection_items_line_no_positive CHECK (line_no >= 1),
    CONSTRAINT pdi_inspection_items_outcome_valid CHECK (outcome IN ('PASS', 'FAIL')),
    -- A failed item needs a note; a passed item may have one.
    CONSTRAINT pdi_inspection_items_note_valid CHECK (
        (note IS NULL OR (length(btrim(note)) BETWEEN 1 AND 500 AND note = btrim(note)))
        AND (outcome = 'PASS' OR note IS NOT NULL))
);
CREATE INDEX pdi_inspection_items_checklist_idx ON pdi_inspection_items (checklist_item_id);

-- Evidence: private attachments (B-09) linked to the inspection (owner pdi_inspection).
CREATE TABLE pdi_inspection_attachments (
    inspection_id uuid NOT NULL REFERENCES pdi_inspections (id) ON DELETE RESTRICT,
    attachment_id uuid NOT NULL REFERENCES attachments (id) ON DELETE RESTRICT,
    PRIMARY KEY (inspection_id, attachment_id),
    CONSTRAINT pdi_inspection_attachments_attachment_key UNIQUE (attachment_id)
);

-- Inspections, outcomes and evidence links are append-only.
CREATE FUNCTION pdi_append_only() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'PDI inspection records are append-only (% on % refused)', TG_OP, TG_TABLE_NAME
        USING ERRCODE = 'insufficient_privilege';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER pdi_inspections_append_only BEFORE UPDATE OR DELETE ON pdi_inspections
    FOR EACH ROW EXECUTE FUNCTION pdi_append_only();
CREATE TRIGGER pdi_inspections_no_truncate BEFORE TRUNCATE ON pdi_inspections
    FOR EACH STATEMENT EXECUTE FUNCTION pdi_append_only();
CREATE TRIGGER pdi_inspection_items_append_only BEFORE UPDATE OR DELETE ON pdi_inspection_items
    FOR EACH ROW EXECUTE FUNCTION pdi_append_only();
CREATE TRIGGER pdi_inspection_items_no_truncate BEFORE TRUNCATE ON pdi_inspection_items
    FOR EACH STATEMENT EXECUTE FUNCTION pdi_append_only();
CREATE TRIGGER pdi_inspection_attachments_append_only BEFORE UPDATE OR DELETE ON pdi_inspection_attachments
    FOR EACH ROW EXECUTE FUNCTION pdi_append_only();
CREATE TRIGGER pdi_inspection_attachments_no_truncate BEFORE TRUNCATE ON pdi_inspection_attachments
    FOR EACH STATEMENT EXECUTE FUNCTION pdi_append_only();

-- An inspection must match the movement that applied it (checked at commit, when
-- both rows exist): same unit, the disposition's movement type and edge, source
-- PDI_INSPECTION naming this inspection; outcomes may only be recorded for
-- checklist items of the unit's brand scope; a linked attachment must be owned
-- by this inspection and belong to the unit's location.
CREATE FUNCTION pdi_inspection_check() RETURNS trigger AS $$
DECLARE
    expected_type text := CASE NEW.disposition WHEN 'PASS' THEN 'PDI_PASS' WHEN 'HOLD' THEN 'QUALITY_HOLD' ELSE 'REJECT' END;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM stock_movements m
            WHERE m.id = NEW.movement_id AND m.vehicle_unit_id = NEW.vehicle_unit_id
              AND m.movement_type = expected_type AND m.from_status = NEW.from_status AND m.to_status = NEW.to_status
              AND m.source_type = 'PDI_INSPECTION' AND m.source_id = NEW.id
              AND m.source_reference = NEW.inspection_number
              AND m.to_location_id IS NOT DISTINCT FROM NEW.location_id) THEN
        RAISE EXCEPTION 'a PDI inspection needs its matching stock movement'
            USING ERRCODE = 'check_violation', CONSTRAINT = 'pdi_inspections_movement_matches';
    END IF;
    IF EXISTS (SELECT 1 FROM pdi_inspection_items i
            JOIN pdi_checklist_items c ON c.id = i.checklist_item_id
            JOIN vehicle_units v ON v.id = NEW.vehicle_unit_id
            WHERE i.inspection_id = NEW.id AND c.brand_id IS NOT NULL AND c.brand_id <> v.brand_id) THEN
        RAISE EXCEPTION 'a PDI outcome was recorded for another brand''s checklist item'
            USING ERRCODE = 'check_violation', CONSTRAINT = 'pdi_inspection_items_brand_matches';
    END IF;
    IF EXISTS (SELECT 1 FROM pdi_inspection_attachments l JOIN attachments a ON a.id = l.attachment_id
            WHERE l.inspection_id = NEW.id AND (a.owner_type IS DISTINCT FROM 'pdi_inspection'
                OR a.owner_id IS DISTINCT FROM NEW.id::text OR a.location_id IS DISTINCT FROM NEW.location_id)) THEN
        RAISE EXCEPTION 'PDI evidence must be owned by the inspection and belong to the unit''s location'
            USING ERRCODE = 'check_violation', CONSTRAINT = 'pdi_inspection_attachments_owner_matches';
    END IF;
    IF NEW.disposition = 'PASS' AND (
           NOT EXISTS (SELECT 1 FROM pdi_inspection_items WHERE inspection_id = NEW.id)
        OR EXISTS (SELECT 1 FROM pdi_inspection_items WHERE inspection_id = NEW.id AND outcome <> 'PASS')) THEN
        RAISE EXCEPTION 'a PDI pass needs recorded checklist outcomes, all passed'
            USING ERRCODE = 'check_violation', CONSTRAINT = 'pdi_inspections_pass_outcomes';
    END IF;
    IF NEW.disposition IN ('HOLD', 'REJECT')
       AND NOT EXISTS (SELECT 1 FROM pdi_inspection_attachments WHERE inspection_id = NEW.id) THEN
        RAISE EXCEPTION 'a PDI hold or rejection needs evidence'
            USING ERRCODE = 'check_violation', CONSTRAINT = 'pdi_inspections_evidence_required';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER pdi_inspections_check AFTER INSERT ON pdi_inspections
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION pdi_inspection_check();

-- ── Ledger: PDI movements may name their inspection ─────────────────────────

-- New source PDI_INSPECTION for the three PDI movement types (source_id = the
-- inspection, source_reference = its number). STAFF_ACTION stays valid for them
-- at the database level so the E-01 contract is unchanged; the only route that
-- makes PDI movements (POST /vehicle-units/{id}/pdi) always names the inspection.
ALTER TABLE stock_movements
    DROP CONSTRAINT stock_movements_source_valid,
    DROP CONSTRAINT stock_movements_source_complete;

ALTER TABLE stock_movements
    ADD CONSTRAINT stock_movements_source_valid CHECK (source_type IN
        ('GOODS_RECEIPT', 'STAFF_ACTION', 'SUPPLIER_DELIVERY', 'STOCK_COUNT', 'PDI_INSPECTION')),
    ADD CONSTRAINT stock_movements_source_complete CHECK (
        (source_type = 'GOODS_RECEIPT' AND movement_type = 'RECEIPT' AND source_id IS NOT NULL AND source_reference IS NOT NULL)
     OR (source_type = 'STAFF_ACTION' AND item_type = 'VEHICLE_UNIT' AND movement_type <> 'RECEIPT' AND source_id IS NULL
         AND source_reference IS NULL)
     OR (source_type = 'SUPPLIER_DELIVERY' AND movement_type = 'PARTS_RECEIPT' AND supplier_id IS NOT NULL
         AND source_id IS NULL AND source_reference IS NOT NULL)
     OR (source_type = 'STAFF_ACTION' AND movement_type = 'PARTS_ISSUE' AND source_id IS NULL AND supplier_id IS NULL
         AND reason IS NOT NULL)
     OR (source_type = 'STOCK_COUNT' AND movement_type = 'COUNT_ADJUSTMENT' AND source_id IS NOT NULL
         AND source_reference IS NOT NULL AND supplier_id IS NULL AND reason IS NOT NULL)
     OR (source_type = 'PDI_INSPECTION' AND item_type = 'VEHICLE_UNIT'
         AND movement_type IN ('PDI_PASS', 'QUALITY_HOLD', 'REJECT')
         AND source_id IS NOT NULL AND source_reference IS NOT NULL AND supplier_id IS NULL));

-- One movement per inspection, ever.
CREATE UNIQUE INDEX stock_movements_pdi_once ON stock_movements (source_id) WHERE source_type = 'PDI_INSPECTION';

-- A PDI_INSPECTION movement must belong to its inspection (checked at commit).
CREATE FUNCTION stock_movement_pdi_check() RETURNS trigger AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pdi_inspections i WHERE i.id = NEW.source_id AND i.movement_id = NEW.id) THEN
        RAISE EXCEPTION 'a PDI movement needs its inspection'
            USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_movements_pdi_inspection';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER stock_movements_pdi_check AFTER INSERT ON stock_movements
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW WHEN (NEW.source_type = 'PDI_INSPECTION')
    EXECUTE FUNCTION stock_movement_pdi_check();
