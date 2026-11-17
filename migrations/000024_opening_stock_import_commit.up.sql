-- G-07: opening stock import commit (ADR-031; IMP-001…IMP-005, FR-014, LED-001/003/008).
--
-- A G-06 preview batch can become stock exactly once, all rows or none:
--   1. an approval of the batch is recorded (physical count reference and date,
--      mappings and duplicates reviewed, whether the received dates are the
--      agreed opening-stock dates) by a holder of stock.import.approve;
--   2. a commit (one per batch, ever) by a holder of stock.import.commit
--      re-validates every row against the current database and, in one
--      transaction, creates one vehicle unit per row (source OPENING_STOCK,
--      pointing at its batch row: no supplier/PO/GRN history is fabricated,
--      IMP-005), its REGISTERED history entry and its OPENING_BALANCE movement
--      (E-01 ledger), and the audit events.
-- Both permissions are sensitive and granted to no role (RB-5). Approvals,
-- commits and the units they create are never changed or deleted.

INSERT INTO permissions (key, description, sensitive) VALUES
    ('stock.import.approve', 'Record the approval (physical count reconciliation) of an opening-stock import batch', true),
    ('stock.import.commit',  'Commit an approved opening-stock import batch into stock (creates vehicle units)', true);

-- An approval or commit names the exact file it is about.
ALTER TABLE opening_stock_import_batches
    ADD CONSTRAINT opening_stock_import_batches_id_sha256_key UNIQUE (id, file_sha256);

-- ── Approval ────────────────────────────────────────────────────────────────
CREATE TABLE opening_stock_import_approvals (
    id                       uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id                 uuid        NOT NULL,
    file_sha256              text        NOT NULL,
    approved_by              uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    approved_at              timestamptz NOT NULL DEFAULT now(),
    -- The signed physical count the batch was reconciled to (sheet/reference) and its date.
    physical_count_reference text        NOT NULL,
    physical_count_date      date        NOT NULL,
    -- Explicit attestations; the commit needs both true.
    mappings_reviewed        boolean     NOT NULL,
    duplicates_reviewed      boolean     NOT NULL,
    -- The file's received dates are the agreed opening-stock dates (LED-008). When
    -- false the units are still created, but E-07 keeps their age MISSING_ORIGIN.
    received_dates_approved  boolean     NOT NULL,
    note                     text,
    request_id               text,

    CONSTRAINT opening_stock_import_approvals_batch_key UNIQUE (batch_id),
    CONSTRAINT opening_stock_import_approvals_id_batch_key UNIQUE (id, batch_id),
    CONSTRAINT opening_stock_import_approvals_batch_fk FOREIGN KEY (batch_id, file_sha256)
        REFERENCES opening_stock_import_batches (id, file_sha256) ON DELETE RESTRICT,
    CONSTRAINT opening_stock_import_approvals_reviewed CHECK (mappings_reviewed AND duplicates_reviewed),
    CONSTRAINT opening_stock_import_approvals_reference_length CHECK (
        length(btrim(physical_count_reference)) BETWEEN 1 AND 100 AND length(physical_count_reference) <= 100),
    CONSTRAINT opening_stock_import_approvals_note_length CHECK (note IS NULL OR (length(btrim(note)) > 0 AND length(note) <= 500))
);

-- ── Commit ──────────────────────────────────────────────────────────────────
CREATE SEQUENCE opening_stock_commit_number_seq;

CREATE TABLE opening_stock_import_commits (
    id              uuid        PRIMARY KEY,
    reference       text        NOT NULL DEFAULT 'OSC-' || lpad(nextval('opening_stock_commit_number_seq')::text, 6, '0'),
    batch_id        uuid        NOT NULL,
    file_sha256     text        NOT NULL,
    approval_id     uuid        NOT NULL,
    committed_by    uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    committed_at    timestamptz NOT NULL DEFAULT now(),
    -- Per committer: a repeat with the same key returns this commit.
    idempotency_key text        NOT NULL,
    units_created   integer     NOT NULL,
    request_id      text,

    -- A batch commits once, ever (IMP-003).
    CONSTRAINT opening_stock_import_commits_batch_key UNIQUE (batch_id),
    CONSTRAINT opening_stock_import_commits_reference_key UNIQUE (reference),
    CONSTRAINT opening_stock_import_commits_batch_fk FOREIGN KEY (batch_id, file_sha256)
        REFERENCES opening_stock_import_batches (id, file_sha256) ON DELETE RESTRICT,
    CONSTRAINT opening_stock_import_commits_approval_fk FOREIGN KEY (approval_id, batch_id)
        REFERENCES opening_stock_import_approvals (id, batch_id) ON DELETE RESTRICT,
    CONSTRAINT opening_stock_import_commits_key_format CHECK (idempotency_key ~ '^[A-Za-z0-9_-]{8,100}$'),
    CONSTRAINT opening_stock_import_commits_units_positive CHECK (units_created > 0)
);
CREATE UNIQUE INDEX opening_stock_import_commits_idempotency_key ON opening_stock_import_commits (committed_by, idempotency_key);

-- Only a PREVIEWED batch can be approved or committed.
CREATE FUNCTION opening_stock_import_previewed_check() RETURNS trigger AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM opening_stock_import_batches WHERE id = NEW.batch_id AND status = 'PREVIEWED') THEN
        RAISE EXCEPTION 'only a previewed opening-stock batch can be approved or committed'
            USING ERRCODE = 'check_violation', CONSTRAINT = 'opening_stock_import_batch_previewed';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER opening_stock_import_approvals_previewed BEFORE INSERT ON opening_stock_import_approvals
    FOR EACH ROW EXECUTE FUNCTION opening_stock_import_previewed_check();
CREATE TRIGGER opening_stock_import_commits_previewed BEFORE INSERT ON opening_stock_import_commits
    FOR EACH ROW EXECUTE FUNCTION opening_stock_import_previewed_check();

-- All or nothing (FR-014), checked at transaction commit: a commit created exactly
-- one unit for every row of its batch, and says so.
CREATE FUNCTION opening_stock_import_commit_complete() RETURNS trigger AS $$
DECLARE
    batch_rows integer;
    units      integer;
BEGIN
    SELECT total_rows INTO batch_rows FROM opening_stock_import_batches WHERE id = NEW.batch_id;
    SELECT count(*) INTO units FROM vehicle_units WHERE opening_stock_batch_id = NEW.batch_id;
    IF units <> NEW.units_created OR units <> batch_rows THEN
        RAISE EXCEPTION 'an opening-stock commit must create one unit for every row of its batch (% rows, % units, % recorded)',
            batch_rows, units, NEW.units_created
            USING ERRCODE = 'check_violation', CONSTRAINT = 'opening_stock_import_commit_complete';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER opening_stock_import_commits_complete AFTER INSERT ON opening_stock_import_commits
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION opening_stock_import_commit_complete();

-- Approvals and commits are evidence: they never change or disappear.
CREATE TRIGGER opening_stock_import_approvals_guard BEFORE UPDATE OR DELETE ON opening_stock_import_approvals
    FOR EACH ROW EXECUTE FUNCTION opening_stock_import_guard();
CREATE TRIGGER opening_stock_import_approvals_no_truncate BEFORE TRUNCATE ON opening_stock_import_approvals
    FOR EACH STATEMENT EXECUTE FUNCTION opening_stock_import_guard();
CREATE TRIGGER opening_stock_import_commits_guard BEFORE UPDATE OR DELETE ON opening_stock_import_commits
    FOR EACH ROW EXECUTE FUNCTION opening_stock_import_guard();
CREATE TRIGGER opening_stock_import_commits_no_truncate BEFORE TRUNCATE ON opening_stock_import_commits
    FOR EACH STATEMENT EXECUTE FUNCTION opening_stock_import_guard();

-- ── Vehicle units from opening stock ────────────────────────────────────────
-- The unit points at the staged row it came from (source values, normalised
-- values, resolved IDs, file and uploader are on the row and its batch). No
-- supplier PO or goods receipt is invented: those columns stay NULL (IMP-005);
-- a supplier is linked only when the file named an existing one.
ALTER TABLE vehicle_units
    ADD COLUMN opening_stock_batch_id   uuid,
    ADD COLUMN opening_stock_row_number integer;

ALTER TABLE vehicle_units
    DROP CONSTRAINT vehicle_units_source_valid;

ALTER TABLE vehicle_units
    ADD CONSTRAINT vehicle_units_source_valid CHECK (source_type IN ('GOODS_RECEIPT', 'OPENING_STOCK')),
    ADD CONSTRAINT vehicle_units_opening_stock_row_fk FOREIGN KEY (opening_stock_batch_id, opening_stock_row_number)
        REFERENCES opening_stock_import_rows (batch_id, row_number) ON DELETE RESTRICT,
    -- A staged row becomes at most one vehicle unit, ever.
    ADD CONSTRAINT vehicle_units_opening_stock_row_key UNIQUE (opening_stock_batch_id, opening_stock_row_number),
    ADD CONSTRAINT vehicle_units_opening_stock_source_complete CHECK (
        (source_type = 'OPENING_STOCK') = (opening_stock_batch_id IS NOT NULL)
        AND (opening_stock_batch_id IS NULL) = (opening_stock_row_number IS NULL)
        AND (source_type <> 'OPENING_STOCK' OR (location_id IS NOT NULL AND chassis_number IS NOT NULL
            AND purchase_order_id IS NULL AND purchase_order_line_id IS NULL AND goods_receipt_id IS NULL
            AND goods_receipt_line_id IS NULL AND goods_receipt_unit_id IS NULL)));

CREATE INDEX vehicle_units_opening_stock_batch_idx ON vehicle_units (opening_stock_batch_id) WHERE opening_stock_batch_id IS NOT NULL;

-- An opening-stock unit is created only inside its batch's commit, from a row
-- that could proceed at preview, with the row's resolved catalogue, location
-- and chassis (the commit re-validates them first).
CREATE FUNCTION vehicle_unit_opening_stock_check() RETURNS trigger AS $$
DECLARE
    r opening_stock_import_rows%ROWTYPE;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM opening_stock_import_commits WHERE batch_id = NEW.opening_stock_batch_id) THEN
        RAISE EXCEPTION 'an opening-stock unit is created only by its batch''s commit'
            USING ERRCODE = 'check_violation', CONSTRAINT = 'vehicle_units_opening_stock_committed';
    END IF;
    SELECT * INTO r FROM opening_stock_import_rows
        WHERE batch_id = NEW.opening_stock_batch_id AND row_number = NEW.opening_stock_row_number;
    IF NOT FOUND OR NOT r.can_proceed OR r.brand_id IS DISTINCT FROM NEW.brand_id OR r.model_id IS DISTINCT FROM NEW.model_id
        OR r.variant_id IS DISTINCT FROM NEW.variant_id OR r.location_id IS DISTINCT FROM NEW.location_id
        OR r.chassis_normalised IS DISTINCT FROM upper(btrim(NEW.chassis_number))
        OR r.stock_status IS DISTINCT FROM NEW.status THEN
        RAISE EXCEPTION 'an opening-stock unit must match its staged row'
            USING ERRCODE = 'check_violation', CONSTRAINT = 'vehicle_units_opening_stock_matches_row';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER vehicle_units_opening_stock_check BEFORE INSERT ON vehicle_units
    FOR EACH ROW WHEN (NEW.source_type = 'OPENING_STOCK') EXECUTE FUNCTION vehicle_unit_opening_stock_check();

ALTER TABLE vehicle_unit_events DROP CONSTRAINT vehicle_unit_events_source_valid;
ALTER TABLE vehicle_unit_events
    ADD CONSTRAINT vehicle_unit_events_source_valid CHECK (source_type IN ('GOODS_RECEIPT', 'OPENING_STOCK'));

-- ── Ledger: the OPENING_BALANCE movement ────────────────────────────────────
-- An opening-stock unit enters the ledger with OPENING_BALANCE (0 -> 1, none ->
-- its opening status, at its location), sourced from its batch. It is the
-- unit's first movement, exactly like RECEIPT for goods-receipt units.
ALTER TABLE stock_movements
    DROP CONSTRAINT stock_movements_type_valid,
    DROP CONSTRAINT stock_movements_transition_valid,
    DROP CONSTRAINT stock_movements_quantity_valid,
    DROP CONSTRAINT stock_movements_source_valid,
    DROP CONSTRAINT stock_movements_source_complete;

ALTER TABLE stock_movements
    ADD CONSTRAINT stock_movements_type_valid CHECK (movement_type IN
        ('RECEIPT', 'PDI_PASS', 'QUALITY_HOLD', 'REJECT', 'DISPLAY_ON', 'DISPLAY_OFF', 'PARTS_RECEIPT', 'PARTS_ISSUE',
         'COUNT_ADJUSTMENT', 'OPENING_BALANCE')),
    ADD CONSTRAINT stock_movements_transition_valid CHECK (
        (item_type = 'VEHICLE_UNIT' AND (
            (movement_type = 'RECEIPT'      AND from_status IS NULL                   AND to_status = 'RECEIVED_PENDING_PDI')
         OR (movement_type = 'PDI_PASS'     AND from_status = 'RECEIVED_PENDING_PDI'  AND to_status = 'AVAILABLE')
         OR (movement_type = 'QUALITY_HOLD' AND from_status = 'RECEIVED_PENDING_PDI'  AND to_status = 'QUALITY_HOLD')
         OR (movement_type = 'REJECT'       AND from_status = 'QUALITY_HOLD'          AND to_status = 'REJECTED')
         OR (movement_type = 'DISPLAY_ON'   AND from_status = 'AVAILABLE'             AND to_status = 'DISPLAY')
         OR (movement_type = 'DISPLAY_OFF'  AND from_status = 'DISPLAY'               AND to_status = 'AVAILABLE')
            -- The in-stock statuses the ledger can hold; which of them opening stock may use is OQ-47 (configuration).
         OR (movement_type = 'OPENING_BALANCE' AND from_status IS NULL
             AND to_status IN ('RECEIVED_PENDING_PDI', 'QUALITY_HOLD', 'AVAILABLE', 'DISPLAY', 'REJECTED'))))
     OR (item_type = 'STOCK_ITEM' AND movement_type IN ('PARTS_RECEIPT', 'PARTS_ISSUE', 'COUNT_ADJUSTMENT'))),
    ADD CONSTRAINT stock_movements_quantity_valid CHECK (
        (item_type = 'VEHICLE_UNIT' AND (
            (movement_type IN ('RECEIPT', 'OPENING_BALANCE') AND quantity_before = 0 AND quantity_after = 1
             AND from_location_id IS NULL)
         OR (movement_type NOT IN ('RECEIPT', 'OPENING_BALANCE') AND quantity_before = 1 AND quantity_after = 1
             AND from_location_id IS NOT DISTINCT FROM to_location_id)))
     OR (item_type = 'STOCK_ITEM' AND quantity_before >= 0 AND quantity_after >= 0 AND (
            (movement_type = 'PARTS_RECEIPT' AND quantity_after > quantity_before)
         OR (movement_type = 'PARTS_ISSUE' AND quantity_after < quantity_before)
         OR (movement_type = 'COUNT_ADJUSTMENT' AND quantity_after <> quantity_before)))),
    ADD CONSTRAINT stock_movements_source_valid CHECK (source_type IN
        ('GOODS_RECEIPT', 'STAFF_ACTION', 'SUPPLIER_DELIVERY', 'STOCK_COUNT', 'PDI_INSPECTION', 'OPENING_STOCK')),
    ADD CONSTRAINT stock_movements_source_complete CHECK (
        (source_type = 'GOODS_RECEIPT' AND movement_type = 'RECEIPT' AND source_id IS NOT NULL AND source_reference IS NOT NULL)
     OR (source_type = 'STAFF_ACTION' AND item_type = 'VEHICLE_UNIT' AND movement_type NOT IN ('RECEIPT', 'OPENING_BALANCE')
         AND source_id IS NULL AND source_reference IS NULL)
     OR (source_type = 'SUPPLIER_DELIVERY' AND movement_type = 'PARTS_RECEIPT' AND supplier_id IS NOT NULL
         AND source_id IS NULL AND source_reference IS NOT NULL)
     OR (source_type = 'STAFF_ACTION' AND movement_type = 'PARTS_ISSUE' AND source_id IS NULL AND supplier_id IS NULL
         AND reason IS NOT NULL)
     OR (source_type = 'STOCK_COUNT' AND movement_type = 'COUNT_ADJUSTMENT' AND source_id IS NOT NULL
         AND source_reference IS NOT NULL AND supplier_id IS NULL AND reason IS NOT NULL)
     OR (source_type = 'PDI_INSPECTION' AND item_type = 'VEHICLE_UNIT'
         AND movement_type IN ('PDI_PASS', 'QUALITY_HOLD', 'REJECT')
         AND source_id IS NOT NULL AND source_reference IS NOT NULL AND supplier_id IS NULL)
        -- An opening balance names its import batch (id and OSI reference).
     OR (source_type = 'OPENING_STOCK' AND item_type = 'VEHICLE_UNIT' AND movement_type = 'OPENING_BALANCE'
         AND source_id IS NOT NULL AND source_reference IS NOT NULL AND supplier_id IS NULL));

CREATE UNIQUE INDEX stock_movements_opening_balance_once ON stock_movements (vehicle_unit_id)
    WHERE movement_type = 'OPENING_BALANCE';

-- As in 000017, plus: an OPENING_BALANCE must be the unit's first movement, from
-- its own import batch, at the state it was created in; it changes nothing.
CREATE OR REPLACE FUNCTION stock_movement_apply() RETURNS trigger AS $$
DECLARE
    cur_status   text;
    cur_location uuid;
    cur_receipt  uuid;
    cur_batch    uuid;
    item_status  text;
    item_mode    text;
    item_uom     text;
    loc_active   boolean;
    sup_status   text;
    rows         integer;
BEGIN
    IF NEW.item_type = 'STOCK_ITEM' THEN
        SELECT status, tracking_mode, uom INTO item_status, item_mode, item_uom
            FROM stock_items WHERE id = NEW.stock_item_id FOR SHARE;
        IF item_status IS DISTINCT FROM 'ACTIVE' THEN
            RAISE EXCEPTION 'the SKU is archived' USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_movements_item_active';
        END IF;
        IF item_mode IS DISTINCT FROM 'QUANTITY' THEN
            RAISE EXCEPTION 'only quantity-tracked SKUs move by quantity'
                USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_movements_item_quantity_tracked';
        END IF;
        IF item_uom IS DISTINCT FROM NEW.uom THEN
            RAISE EXCEPTION 'the movement''s unit is not the SKU''s'
                USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_movements_uom_matches_item';
        END IF;
        SELECT active INTO loc_active FROM locations WHERE id = NEW.to_location_id FOR SHARE;
        IF loc_active IS DISTINCT FROM true THEN
            RAISE EXCEPTION 'the location is not active' USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_movements_location_active';
        END IF;
        IF NEW.supplier_id IS NOT NULL THEN
            SELECT status INTO sup_status FROM suppliers WHERE id = NEW.supplier_id FOR SHARE;
            IF sup_status = 'ARCHIVED' THEN
                RAISE EXCEPTION 'the supplier is archived'
                    USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_movements_supplier_active';
            END IF;
        END IF;
        INSERT INTO stock_item_balances (stock_item_id, location_id) VALUES (NEW.stock_item_id, NEW.to_location_id)
            ON CONFLICT DO NOTHING;
        PERFORM set_config('maa.stock_movement_balance', NEW.stock_item_id::text || '/' || NEW.to_location_id::text, true);
        UPDATE stock_item_balances SET quantity = NEW.quantity_after, version = version + 1, updated_at = now()
            WHERE stock_item_id = NEW.stock_item_id AND location_id = NEW.to_location_id AND quantity = NEW.quantity_before;
        GET DIAGNOSTICS rows = ROW_COUNT;
        PERFORM set_config('maa.stock_movement_balance', '', true);
        IF rows <> 1 THEN
            RAISE EXCEPTION 'the stock balance is no longer the movement''s starting quantity'
                USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_movements_from_matches_balance';
        END IF;
        RETURN NULL;
    END IF;

    SELECT status, location_id, goods_receipt_id, opening_stock_batch_id INTO cur_status, cur_location, cur_receipt, cur_batch
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
    IF NEW.movement_type = 'OPENING_BALANCE' THEN
        IF cur_status IS DISTINCT FROM NEW.to_status OR cur_location IS DISTINCT FROM NEW.to_location_id
            OR cur_batch IS NULL OR cur_batch IS DISTINCT FROM NEW.source_id
            OR EXISTS (SELECT 1 FROM stock_movements WHERE vehicle_unit_id = NEW.vehicle_unit_id AND id <> NEW.id) THEN
            RAISE EXCEPTION 'an opening balance must be the unit''s first movement, from its own import batch'
                USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_movements_opening_balance_matches_unit';
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

-- LED-003 as in 000015, with the entry movement being RECEIPT (goods receipt)
-- or OPENING_BALANCE (opening stock), whichever matches the unit's source.
CREATE OR REPLACE FUNCTION vehicle_unit_ledger_check() RETURNS trigger AS $$
DECLARE
    unit_status   text;
    unit_location uuid;
    unit_source   text;
    last          stock_movements%ROWTYPE;
BEGIN
    SELECT status, location_id, source_type INTO unit_status, unit_location, unit_source FROM vehicle_units WHERE id = NEW.id;
    SELECT * INTO last FROM stock_movements WHERE vehicle_unit_id = NEW.id ORDER BY seq DESC LIMIT 1;
    IF NOT FOUND OR NOT EXISTS (SELECT 1 FROM stock_movements WHERE vehicle_unit_id = NEW.id
            AND movement_type = CASE unit_source WHEN 'OPENING_STOCK' THEN 'OPENING_BALANCE' ELSE 'RECEIPT' END)
        OR last.to_status IS DISTINCT FROM unit_status OR last.to_location_id IS DISTINCT FROM unit_location
        OR last.quantity_after <> 1 THEN
        RAISE EXCEPTION 'vehicle unit % does not match its stock movements', NEW.id
            USING ERRCODE = 'check_violation', CONSTRAINT = 'vehicle_units_ledger_reconciles';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
