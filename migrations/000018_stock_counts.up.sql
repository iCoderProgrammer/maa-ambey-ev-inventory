-- E-05: stock counts and adjustment approval (ADR-023; PRD §5.3, LED-005, LED-006,
-- SEC-003). A stock count is an adjustment request: staff record what they
-- found at one location (one line per SKU or vehicle unit); the server records
-- what the ledger expected and the variance. Nothing changes stock until a
-- different person approves the submitted count; approval posts one
-- COUNT_ADJUSTMENT ledger movement per parts line with a variance, in the same
-- transaction. A rejected or pending count never moves stock.
--
-- No approval threshold exists yet (LED-005 "configurable threshold", A-05/RB-5
-- pending): every count needs an approver (approval_policy APPROVER_REQUIRED),
-- nothing is approved automatically by size. Separation of duties (LED-006): the
-- creator and the submitter never decide; the service also refuses anyone who
-- recorded a line.
--
-- Vehicle units: a discrepancy (unit not found) is recorded and approved as a
-- finding only. No approved movement exists for a missing or found vehicle
-- (OQ-33, OQ-39), so approval never changes a unit.

INSERT INTO permissions (key, description, sensitive) VALUES
    ('stock.adjustment.view', 'View stock counts, their expected and observed quantities, variances and decisions at the caller''s locations', false);

-- ── Counts ──────────────────────────────────────────────────────────────────

CREATE SEQUENCE stock_count_number_seq;

CREATE TABLE stock_counts (
    id                       uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    count_number             text        NOT NULL DEFAULT 'SC-' || lpad(nextval('stock_count_number_seq')::text, 6, '0'),
    location_id              uuid        NOT NULL REFERENCES locations (id) ON DELETE RESTRICT,
    status                   text        NOT NULL DEFAULT 'DRAFT',
    -- How the decision is required. Only APPROVER_REQUIRED exists until a
    -- threshold policy is approved (LED-005, A-05).
    approval_policy          text        NOT NULL DEFAULT 'APPROVER_REQUIRED',
    note                     text,
    created_by               uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_at               timestamptz NOT NULL DEFAULT now(),
    -- The create request's key (per creator): a repeat returns this count.
    create_idempotency_key   text        NOT NULL,
    create_request_hash      text        NOT NULL,
    submitted_by             uuid        REFERENCES users (id) ON DELETE RESTRICT,
    submitted_at             timestamptz,
    decided_by               uuid        REFERENCES users (id) ON DELETE RESTRICT,
    decided_at               timestamptz,
    -- Approval note (optional) or rejection reason (required).
    decision_note            text,
    decision_idempotency_key text,
    decision_request_hash    text,
    version                  integer     NOT NULL DEFAULT 1,
    updated_at               timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT stock_counts_status_valid CHECK (status IN ('DRAFT', 'SUBMITTED', 'APPROVED', 'REJECTED')),
    CONSTRAINT stock_counts_policy_valid CHECK (approval_policy IN ('APPROVER_REQUIRED')),
    CONSTRAINT stock_counts_number_format CHECK (count_number ~ '^SC-[0-9]{6,}$'),
    CONSTRAINT stock_counts_note_length CHECK (note IS NULL OR (length(btrim(note)) > 0 AND length(note) <= 500)),
    CONSTRAINT stock_counts_decision_note_length CHECK (decision_note IS NULL
        OR (length(btrim(decision_note)) > 0 AND length(decision_note) <= 500)),
    CONSTRAINT stock_counts_create_key_valid CHECK (create_idempotency_key ~ '^[A-Za-z0-9_-]{8,100}$'
        AND create_request_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT stock_counts_version_positive CHECK (version > 0),
    -- Who and when, by status.
    CONSTRAINT stock_counts_submission_complete CHECK (
        (status = 'DRAFT' AND submitted_by IS NULL AND submitted_at IS NULL)
     OR (status <> 'DRAFT' AND submitted_by IS NOT NULL AND submitted_at IS NOT NULL)),
    CONSTRAINT stock_counts_decision_complete CHECK (
        (status IN ('DRAFT', 'SUBMITTED') AND decided_by IS NULL AND decided_at IS NULL AND decision_note IS NULL
            AND decision_idempotency_key IS NULL AND decision_request_hash IS NULL)
     OR (status IN ('APPROVED', 'REJECTED') AND decided_by IS NOT NULL AND decided_at IS NOT NULL
            AND decision_idempotency_key ~ '^[A-Za-z0-9_-]{8,100}$' AND decision_request_hash ~ '^[0-9a-f]{64}$')),
    CONSTRAINT stock_counts_rejection_reason CHECK (status <> 'REJECTED' OR decision_note IS NOT NULL),
    -- LED-006: the person who created or submitted the count never decides it.
    CONSTRAINT stock_counts_no_self_decision CHECK (decided_by IS NULL
        OR (decided_by <> created_by AND decided_by <> submitted_by))
);
CREATE UNIQUE INDEX stock_counts_number_key ON stock_counts (count_number);
CREATE UNIQUE INDEX stock_counts_create_key ON stock_counts (created_by, create_idempotency_key);
CREATE INDEX stock_counts_status_idx ON stock_counts (status, location_id, created_at DESC);

-- ── Lines ───────────────────────────────────────────────────────────────────

-- One counted item: a quantity-tracked SKU (quantities in its unit) or a
-- vehicle unit (expected 1, observed 1 = found or 0 = not found). expected_* is
-- what the ledger held when the line was recorded (server-read, never sent by
-- the client); expected_version is the balance's (0: no balance row yet) or the
-- unit's version, so a later stock change is detected instead of overwritten.
CREATE TABLE stock_count_lines (
    id                 uuid          PRIMARY KEY DEFAULT gen_random_uuid(),
    stock_count_id     uuid          NOT NULL REFERENCES stock_counts (id) ON DELETE RESTRICT,
    item_type          text          NOT NULL,
    stock_item_id      uuid          REFERENCES stock_items (id) ON DELETE RESTRICT,
    vehicle_unit_id    uuid          REFERENCES vehicle_units (id) ON DELETE RESTRICT,
    uom                text,
    expected_quantity  numeric(12,3) NOT NULL,
    expected_status    text,
    expected_version   integer       NOT NULL,
    observed_quantity  numeric(12,3) NOT NULL,
    variance           numeric(13,3) GENERATED ALWAYS AS (observed_quantity - expected_quantity) STORED,
    reason             text,
    recorded_by        uuid          NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    recorded_at        timestamptz   NOT NULL DEFAULT now(),
    -- The ledger movement approval posted (parts lines with a variance only).
    posted_movement_id uuid          REFERENCES stock_movements (id) ON DELETE RESTRICT,

    CONSTRAINT stock_count_lines_item_valid CHECK (
        (item_type = 'STOCK_ITEM' AND stock_item_id IS NOT NULL AND vehicle_unit_id IS NULL AND uom IS NOT NULL
            AND expected_status IS NULL AND expected_quantity >= 0 AND observed_quantity >= 0)
     OR (item_type = 'VEHICLE_UNIT' AND vehicle_unit_id IS NOT NULL AND stock_item_id IS NULL AND uom IS NULL
            AND expected_status IS NOT NULL AND expected_quantity = 1 AND observed_quantity IN (0, 1)
            AND posted_movement_id IS NULL)),
    CONSTRAINT stock_count_lines_expected_version_valid CHECK (expected_version >= 0),
    -- LED-005: a variance needs a reason.
    CONSTRAINT stock_count_lines_reason_required CHECK (variance = 0 OR reason IS NOT NULL),
    CONSTRAINT stock_count_lines_reason_length CHECK (reason IS NULL OR (length(btrim(reason)) > 0 AND length(reason) <= 300)),
    CONSTRAINT stock_count_lines_posted_has_variance CHECK (posted_movement_id IS NULL OR variance <> 0)
);
CREATE UNIQUE INDEX stock_count_lines_item_key ON stock_count_lines (stock_count_id, stock_item_id) WHERE stock_item_id IS NOT NULL;
CREATE UNIQUE INDEX stock_count_lines_unit_key ON stock_count_lines (stock_count_id, vehicle_unit_id) WHERE vehicle_unit_id IS NOT NULL;
CREATE UNIQUE INDEX stock_count_lines_movement_key ON stock_count_lines (posted_movement_id) WHERE posted_movement_id IS NOT NULL;

-- ── Guards: lifecycle and no deletion ───────────────────────────────────────

-- Counts are never deleted. Status moves DRAFT -> SUBMITTED -> APPROVED | REJECTED
-- only; a decided count never changes again.
CREATE FUNCTION stock_count_guard() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'TRUNCATE' OR TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'stock counts are never deleted' USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF TG_OP = 'INSERT' THEN
        IF NEW.status <> 'DRAFT' OR NEW.version <> 1 THEN
            RAISE EXCEPTION 'a stock count starts as a draft' USING ERRCODE = 'check_violation',
                CONSTRAINT = 'stock_counts_lifecycle_valid';
        END IF;
        RETURN NEW;
    END IF;
    IF OLD.status IN ('APPROVED', 'REJECTED')
        OR NOT ((OLD.status = 'DRAFT' AND NEW.status IN ('DRAFT', 'SUBMITTED'))
             OR (OLD.status = 'SUBMITTED' AND NEW.status IN ('APPROVED', 'REJECTED')))
        OR NEW.id <> OLD.id OR NEW.count_number <> OLD.count_number OR NEW.location_id <> OLD.location_id
        OR NEW.created_by <> OLD.created_by OR NEW.created_at <> OLD.created_at
        OR NEW.create_idempotency_key <> OLD.create_idempotency_key OR NEW.create_request_hash <> OLD.create_request_hash
        OR NEW.approval_policy <> OLD.approval_policy
        OR (OLD.status = 'SUBMITTED' AND (NEW.submitted_by <> OLD.submitted_by OR NEW.submitted_at <> OLD.submitted_at
            OR NEW.note IS DISTINCT FROM OLD.note))
        OR NEW.version <> OLD.version + 1 THEN
        RAISE EXCEPTION 'stock count % cannot change from % to %', OLD.count_number, OLD.status, NEW.status
            USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_counts_lifecycle_valid';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER stock_counts_guard BEFORE INSERT OR UPDATE OR DELETE ON stock_counts
    FOR EACH ROW EXECUTE FUNCTION stock_count_guard();
CREATE TRIGGER stock_counts_no_truncate BEFORE TRUNCATE ON stock_counts
    FOR EACH STATEMENT EXECUTE FUNCTION stock_count_guard();

-- Lines are never deleted. They are recorded and re-recorded while the count is
-- a DRAFT (the count row is locked by the caller); the only later change is
-- linking the posted movement, once, while the count is SUBMITTED (approval).
CREATE FUNCTION stock_count_line_guard() RETURNS trigger AS $$
DECLARE
    count_status text;
BEGIN
    IF TG_OP = 'TRUNCATE' OR TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'stock count lines are never deleted' USING ERRCODE = 'insufficient_privilege';
    END IF;
    SELECT status INTO count_status FROM stock_counts WHERE id = NEW.stock_count_id;
    IF TG_OP = 'INSERT' THEN
        IF count_status = 'DRAFT' AND NEW.posted_movement_id IS NULL THEN
            RETURN NEW;
        END IF;
    ELSIF NEW.id = OLD.id AND NEW.stock_count_id = OLD.stock_count_id AND NEW.item_type = OLD.item_type
        AND NEW.stock_item_id IS NOT DISTINCT FROM OLD.stock_item_id
        AND NEW.vehicle_unit_id IS NOT DISTINCT FROM OLD.vehicle_unit_id THEN
        IF count_status = 'DRAFT' AND OLD.posted_movement_id IS NULL AND NEW.posted_movement_id IS NULL THEN
            RETURN NEW;
        END IF;
        IF count_status = 'SUBMITTED' AND OLD.posted_movement_id IS NULL AND NEW.posted_movement_id IS NOT NULL
            AND NEW.uom IS NOT DISTINCT FROM OLD.uom AND NEW.expected_quantity = OLD.expected_quantity
            AND NEW.expected_status IS NOT DISTINCT FROM OLD.expected_status AND NEW.expected_version = OLD.expected_version
            AND NEW.observed_quantity = OLD.observed_quantity AND NEW.reason IS NOT DISTINCT FROM OLD.reason
            AND NEW.recorded_by = OLD.recorded_by AND NEW.recorded_at = OLD.recorded_at THEN
            RETURN NEW;
        END IF;
    END IF;
    RAISE EXCEPTION 'stock count lines change only while the count is a draft'
        USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_count_lines_editable';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER stock_count_lines_guard BEFORE INSERT OR UPDATE OR DELETE ON stock_count_lines
    FOR EACH ROW EXECUTE FUNCTION stock_count_line_guard();
CREATE TRIGGER stock_count_lines_no_truncate BEFORE TRUNCATE ON stock_count_lines
    FOR EACH STATEMENT EXECUTE FUNCTION stock_count_line_guard();

-- At commit: an APPROVED count has exactly the movements of its parts lines with
-- a variance (each one from expected to observed); a count in any other status
-- has none.
CREATE FUNCTION stock_count_posting_check() RETURNS trigger AS $$
DECLARE
    st text;
BEGIN
    SELECT status INTO st FROM stock_counts WHERE id = NEW.id;
    IF st = 'APPROVED' THEN
        IF EXISTS (SELECT 1 FROM stock_count_lines l
                LEFT JOIN stock_movements m ON m.id = l.posted_movement_id
                WHERE l.stock_count_id = NEW.id AND l.item_type = 'STOCK_ITEM' AND l.variance <> 0
                  AND (m.id IS NULL OR m.movement_type <> 'COUNT_ADJUSTMENT' OR m.source_id <> l.stock_count_id
                       OR m.stock_item_id <> l.stock_item_id OR m.quantity_before <> l.expected_quantity
                       OR m.quantity_after <> l.observed_quantity)) THEN
            RAISE EXCEPTION 'approved stock count % is not posted exactly as counted', NEW.count_number
                USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_counts_posted_as_counted';
        END IF;
    END IF;
    IF EXISTS (SELECT 1 FROM stock_movements m WHERE m.movement_type = 'COUNT_ADJUSTMENT' AND m.source_id = NEW.id
            AND (st <> 'APPROVED' OR NOT EXISTS (SELECT 1 FROM stock_count_lines l
                WHERE l.posted_movement_id = m.id AND l.stock_count_id = NEW.id))) THEN
        RAISE EXCEPTION 'stock count % has a movement it did not approve', NEW.count_number
            USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_counts_posted_as_counted';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER stock_counts_posting_check AFTER INSERT OR UPDATE ON stock_counts
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION stock_count_posting_check();

-- ── Ledger: COUNT_ADJUSTMENT rows ───────────────────────────────────────────

-- A count adjustment sets a SKU's balance at the count's location from the
-- expected to the observed quantity (never below zero), names its count
-- (source STOCK_COUNT, source_id = count, source_reference = count number) and
-- the line's reason. The 000017 rules are otherwise unchanged.
ALTER TABLE stock_movements
    DROP CONSTRAINT stock_movements_type_valid,
    DROP CONSTRAINT stock_movements_transition_valid,
    DROP CONSTRAINT stock_movements_quantity_valid,
    DROP CONSTRAINT stock_movements_source_valid,
    DROP CONSTRAINT stock_movements_source_complete;

ALTER TABLE stock_movements
    ADD CONSTRAINT stock_movements_type_valid CHECK (movement_type IN
        ('RECEIPT', 'PDI_PASS', 'QUALITY_HOLD', 'REJECT', 'DISPLAY_ON', 'DISPLAY_OFF', 'PARTS_RECEIPT', 'PARTS_ISSUE',
         'COUNT_ADJUSTMENT')),
    ADD CONSTRAINT stock_movements_transition_valid CHECK (
        (item_type = 'VEHICLE_UNIT' AND (
            (movement_type = 'RECEIPT'      AND from_status IS NULL                   AND to_status = 'RECEIVED_PENDING_PDI')
         OR (movement_type = 'PDI_PASS'     AND from_status = 'RECEIVED_PENDING_PDI'  AND to_status = 'AVAILABLE')
         OR (movement_type = 'QUALITY_HOLD' AND from_status = 'RECEIVED_PENDING_PDI'  AND to_status = 'QUALITY_HOLD')
         OR (movement_type = 'REJECT'       AND from_status = 'QUALITY_HOLD'          AND to_status = 'REJECTED')
         OR (movement_type = 'DISPLAY_ON'   AND from_status = 'AVAILABLE'             AND to_status = 'DISPLAY')
         OR (movement_type = 'DISPLAY_OFF'  AND from_status = 'DISPLAY'               AND to_status = 'AVAILABLE')))
     OR (item_type = 'STOCK_ITEM' AND movement_type IN ('PARTS_RECEIPT', 'PARTS_ISSUE', 'COUNT_ADJUSTMENT'))),
    ADD CONSTRAINT stock_movements_quantity_valid CHECK (
        (item_type = 'VEHICLE_UNIT' AND (
            (movement_type = 'RECEIPT' AND quantity_before = 0 AND quantity_after = 1 AND from_location_id IS NULL)
         OR (movement_type <> 'RECEIPT' AND quantity_before = 1 AND quantity_after = 1
             AND from_location_id IS NOT DISTINCT FROM to_location_id)))
     OR (item_type = 'STOCK_ITEM' AND quantity_before >= 0 AND quantity_after >= 0 AND (
            (movement_type = 'PARTS_RECEIPT' AND quantity_after > quantity_before)
         OR (movement_type = 'PARTS_ISSUE' AND quantity_after < quantity_before)
         OR (movement_type = 'COUNT_ADJUSTMENT' AND quantity_after <> quantity_before)))),
    ADD CONSTRAINT stock_movements_source_valid CHECK (source_type IN
        ('GOODS_RECEIPT', 'STAFF_ACTION', 'SUPPLIER_DELIVERY', 'STOCK_COUNT')),
    ADD CONSTRAINT stock_movements_source_complete CHECK (
        (source_type = 'GOODS_RECEIPT' AND movement_type = 'RECEIPT' AND source_id IS NOT NULL AND source_reference IS NOT NULL)
     OR (source_type = 'STAFF_ACTION' AND item_type = 'VEHICLE_UNIT' AND movement_type <> 'RECEIPT' AND source_id IS NULL
         AND source_reference IS NULL)
     OR (source_type = 'SUPPLIER_DELIVERY' AND movement_type = 'PARTS_RECEIPT' AND supplier_id IS NOT NULL
         AND source_id IS NULL AND source_reference IS NOT NULL)
     OR (source_type = 'STAFF_ACTION' AND movement_type = 'PARTS_ISSUE' AND source_id IS NULL AND supplier_id IS NULL
         AND reason IS NOT NULL)
        -- A count adjustment names its approved count and the counted reason.
     OR (source_type = 'STOCK_COUNT' AND movement_type = 'COUNT_ADJUSTMENT' AND source_id IS NOT NULL
         AND source_reference IS NOT NULL AND supplier_id IS NULL AND reason IS NOT NULL));

-- One adjustment per SKU per count, ever.
CREATE UNIQUE INDEX stock_movements_count_adjustment_once ON stock_movements (source_id, stock_item_id)
    WHERE movement_type = 'COUNT_ADJUSTMENT';

-- A COUNT_ADJUSTMENT movement must belong to its count's approval: checked at
-- commit by stock_counts_posting_check (the count row is updated to APPROVED
-- in the same transaction). This trigger makes sure that check runs even if a
-- movement were inserted without touching the count.
CREATE FUNCTION stock_movement_count_check() RETURNS trigger AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM stock_counts c JOIN stock_count_lines l ON l.stock_count_id = c.id
            WHERE c.id = NEW.source_id AND c.status = 'APPROVED' AND l.posted_movement_id = NEW.id
              AND l.stock_item_id = NEW.stock_item_id AND c.location_id = NEW.to_location_id) THEN
        RAISE EXCEPTION 'a count adjustment needs its approved stock count'
            USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_movements_count_approved';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER stock_movements_count_check AFTER INSERT ON stock_movements
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW WHEN (NEW.movement_type = 'COUNT_ADJUSTMENT')
    EXECUTE FUNCTION stock_movement_count_check();
