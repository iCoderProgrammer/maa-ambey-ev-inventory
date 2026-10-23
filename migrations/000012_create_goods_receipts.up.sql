-- Goods receipt drafts (D-03; PO-003, PO-004, PO-005, PO-006, Architecture
-- "goods_receipts, goods_receipt_lines"). Schema only: no receipt, unit or grant
-- is seeded.
--
-- A goods receipt note (GRN) records what a supplier delivered against one
-- approved purchase order: per PO line the delivered, accepted and rejected
-- quantities and a discrepancy note, per physical unit the identifiers as read
-- from the vehicle and paperwork, and the supplier's documents (private
-- attachments, B-09).
--
-- A DRAFT changes no stock: it creates no vehicle unit or movement and leaves
-- purchase_order_lines.received_quantity and the PO status alone. Posting
-- (D-07) does that in one transaction; POSTED exists here for it, but nothing in
-- D-03 sets it. Receiving more than the PO line has left (ordered - posted -
-- other open drafts) needs goods_receipts.over_receipt and a reason (PO-003).
-- Nothing is deleted except lines and units of a draft.

INSERT INTO permissions (key, description, sensitive) VALUES
    ('goods_receipts.view',         'View goods receipts, delivered quantities, unit identifiers and documents', true),
    ('goods_receipts.manage',       'Create, edit and cancel draft goods receipts against approved purchase orders', true),
    ('goods_receipts.over_receipt', 'Authorise receiving more than a purchase order line has left, with a reason', true);

-- GRN numbers: GRN-000001, ... Gaps are possible; an internal reference only.
CREATE SEQUENCE goods_receipt_number_seq AS bigint START 1 NO CYCLE;

-- Lets receipt lines reference a PO line of the receipt's own PO (composite FK).
ALTER TABLE purchase_order_lines ADD CONSTRAINT purchase_order_lines_id_po_key UNIQUE (id, purchase_order_id);

CREATE TABLE goods_receipts (
    id                 uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    grn_number         text        NOT NULL DEFAULT ('GRN-' || lpad(nextval('goods_receipt_number_seq')::text, 6, '0')),
    purchase_order_id  uuid        NOT NULL REFERENCES purchase_orders (id) ON DELETE RESTRICT,
    status             text        NOT NULL DEFAULT 'DRAFT',
    -- Day the goods arrived (not in the future; checked by the API against the database date).
    received_date      date        NOT NULL,
    -- The supplier's delivery challan / invoice number as printed.
    supplier_reference text,
    notes              text,
    created_by         uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    idempotency_key    text,
    cancelled_by       uuid        REFERENCES users (id) ON DELETE RESTRICT,
    cancelled_at       timestamptz,
    cancel_reason      text,
    -- Set by posting (D-07).
    posted_by          uuid        REFERENCES users (id) ON DELETE RESTRICT,
    posted_at          timestamptz,
    version            integer     NOT NULL DEFAULT 1,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT goods_receipts_number_key UNIQUE (grn_number),
    CONSTRAINT goods_receipts_id_po_key UNIQUE (id, purchase_order_id),
    CONSTRAINT goods_receipts_status_valid CHECK (status IN ('DRAFT', 'POSTED', 'CANCELLED')),
    CONSTRAINT goods_receipts_supplier_reference_length CHECK (supplier_reference IS NULL OR (length(btrim(supplier_reference)) > 0 AND length(supplier_reference) <= 100)),
    CONSTRAINT goods_receipts_notes_length CHECK (notes IS NULL OR (length(btrim(notes)) > 0 AND length(notes) <= 1000)),
    CONSTRAINT goods_receipts_idempotency_key_length CHECK (idempotency_key IS NULL OR length(idempotency_key) BETWEEN 8 AND 100),
    CONSTRAINT goods_receipts_cancelled_complete CHECK ((status = 'CANCELLED') = (cancelled_by IS NOT NULL AND cancelled_at IS NOT NULL AND cancel_reason IS NOT NULL)),
    CONSTRAINT goods_receipts_cancel_reason_length CHECK (cancel_reason IS NULL OR (length(btrim(cancel_reason)) > 0 AND length(cancel_reason) <= 500)),
    CONSTRAINT goods_receipts_posted_complete CHECK ((status = 'POSTED') = (posted_by IS NOT NULL AND posted_at IS NOT NULL)),
    CONSTRAINT goods_receipts_version_positive CHECK (version > 0)
);
CREATE UNIQUE INDEX goods_receipts_idempotency_key ON goods_receipts (created_by, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX goods_receipts_po_idx ON goods_receipts (purchase_order_id, status);
CREATE INDEX goods_receipts_status_idx ON goods_receipts (status, created_at DESC);
CREATE TRIGGER goods_receipts_set_updated_at BEFORE UPDATE ON goods_receipts FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- What arrived for one PO line. received = accepted + rejected.
CREATE TABLE goods_receipt_lines (
    id                       uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    goods_receipt_id         uuid        NOT NULL,
    purchase_order_id        uuid        NOT NULL,
    purchase_order_line_id   uuid        NOT NULL,
    received_quantity        integer     NOT NULL,
    accepted_quantity        integer     NOT NULL,
    rejected_quantity        integer     NOT NULL,
    discrepancy_note         text,
    -- Over-receipt authorisation (PO-003): who allowed receiving up to
    -- over_receipt_quantity on this line although the PO line had less left, and why.
    over_receipt_quantity    integer,
    over_receipt_reason      text,
    over_receipt_by          uuid        REFERENCES users (id) ON DELETE RESTRICT,
    over_receipt_at          timestamptz,
    created_at               timestamptz NOT NULL DEFAULT now(),
    updated_at               timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT goods_receipt_lines_receipt_fk FOREIGN KEY (goods_receipt_id, purchase_order_id)
        REFERENCES goods_receipts (id, purchase_order_id) ON DELETE RESTRICT,
    CONSTRAINT goods_receipt_lines_po_line_fk FOREIGN KEY (purchase_order_line_id, purchase_order_id)
        REFERENCES purchase_order_lines (id, purchase_order_id) ON DELETE RESTRICT,
    CONSTRAINT goods_receipt_lines_id_receipt_key UNIQUE (id, goods_receipt_id),
    -- One line per PO line on a receipt.
    CONSTRAINT goods_receipt_lines_po_line_key UNIQUE (goods_receipt_id, purchase_order_line_id),
    CONSTRAINT goods_receipt_lines_received_range CHECK (received_quantity BETWEEN 1 AND 10000),
    CONSTRAINT goods_receipt_lines_split CHECK (accepted_quantity >= 0 AND rejected_quantity >= 0
        AND accepted_quantity + rejected_quantity = received_quantity),
    CONSTRAINT goods_receipt_lines_discrepancy_length CHECK (discrepancy_note IS NULL OR (length(btrim(discrepancy_note)) > 0 AND length(discrepancy_note) <= 500)),
    CONSTRAINT goods_receipt_lines_over_receipt_complete CHECK (
        (over_receipt_quantity IS NULL) = (over_receipt_reason IS NULL)
        AND (over_receipt_quantity IS NULL) = (over_receipt_by IS NULL)
        AND (over_receipt_quantity IS NULL) = (over_receipt_at IS NULL)),
    CONSTRAINT goods_receipt_lines_over_receipt_covers CHECK (over_receipt_quantity IS NULL OR over_receipt_quantity >= received_quantity),
    CONSTRAINT goods_receipt_lines_over_receipt_reason_length CHECK (over_receipt_reason IS NULL OR (length(btrim(over_receipt_reason)) > 0 AND length(over_receipt_reason) <= 500))
);
CREATE INDEX goods_receipt_lines_po_line_idx ON goods_receipt_lines (purchase_order_line_id);
CREATE TRIGGER goods_receipt_lines_set_updated_at BEFORE UPDATE ON goods_receipt_lines FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- One physical unit as received, with its identifiers as captured. Identifier
-- rules per brand (format, which serials are mandatory, motor/battery
-- uniqueness) are pending (OQ-4, D-04/D-05): every identifier is optional in a
-- draft. chassis_normalised is the chassis/frame number trimmed and upper-cased
-- (INV-002; which separators may be ignored is not yet approved). A chassis
-- appears once per receipt (index) and once across open drafts (checked by the
-- API under an advisory lock); uniqueness against registered vehicles comes
-- with the vehicle registry (D-04/D-05) and is enforced at posting (D-07).
CREATE TABLE goods_receipt_units (
    id                    uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    goods_receipt_id      uuid        NOT NULL,
    goods_receipt_line_id uuid        NOT NULL,
    unit_no               integer     NOT NULL,
    condition             text        NOT NULL,
    chassis_number        text,
    chassis_normalised    text,
    motor_serial          text,
    battery_serial        text,
    charger_serial        text,
    note                  text,
    created_at            timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT goods_receipt_units_line_fk FOREIGN KEY (goods_receipt_line_id, goods_receipt_id)
        REFERENCES goods_receipt_lines (id, goods_receipt_id) ON DELETE RESTRICT,
    CONSTRAINT goods_receipt_units_no_key UNIQUE (goods_receipt_line_id, unit_no),
    CONSTRAINT goods_receipt_units_unit_no_positive CHECK (unit_no > 0),
    CONSTRAINT goods_receipt_units_condition_valid CHECK (condition IN ('ACCEPTED', 'REJECTED')),
    CONSTRAINT goods_receipt_units_chassis_pair CHECK ((chassis_number IS NULL) = (chassis_normalised IS NULL)),
    CONSTRAINT goods_receipt_units_chassis_normalised CHECK (chassis_normalised IS NULL OR chassis_normalised = upper(btrim(chassis_number))),
    CONSTRAINT goods_receipt_units_identifier_length CHECK (
        (chassis_number IS NULL OR length(btrim(chassis_number)) BETWEEN 1 AND 50)
        AND (motor_serial IS NULL OR length(btrim(motor_serial)) BETWEEN 1 AND 50)
        AND (battery_serial IS NULL OR length(btrim(battery_serial)) BETWEEN 1 AND 50)
        AND (charger_serial IS NULL OR length(btrim(charger_serial)) BETWEEN 1 AND 50)),
    CONSTRAINT goods_receipt_units_note_length CHECK (note IS NULL OR (length(btrim(note)) > 0 AND length(note) <= 300))
);
CREATE UNIQUE INDEX goods_receipt_units_chassis_key ON goods_receipt_units (goods_receipt_id, chassis_normalised) WHERE chassis_normalised IS NOT NULL;
CREATE INDEX goods_receipt_units_chassis_idx ON goods_receipt_units (chassis_normalised) WHERE chassis_normalised IS NOT NULL;

-- Supplier documents for the delivery: private attachments linked to the receipt (B-09).
CREATE TABLE goods_receipt_documents (
    id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    goods_receipt_id uuid        NOT NULL REFERENCES goods_receipts (id) ON DELETE RESTRICT,
    attachment_id    uuid        NOT NULL REFERENCES attachments (id) ON DELETE RESTRICT,
    document_type    text        NOT NULL,
    note             text,
    added_by         uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_at       timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT goods_receipt_documents_attachment_key UNIQUE (attachment_id),
    -- Proposal pending confirmation (OQ-25).
    CONSTRAINT goods_receipt_documents_type_valid CHECK (document_type IN ('DELIVERY_CHALLAN', 'SUPPLIER_INVOICE', 'PHOTO', 'OTHER')),
    CONSTRAINT goods_receipt_documents_note_length CHECK (note IS NULL OR (length(btrim(note)) > 0 AND length(note) <= 300))
);
CREATE INDEX goods_receipt_documents_receipt_idx ON goods_receipt_documents (goods_receipt_id, created_at);

-- Status history: one row per transition, append-only.
CREATE TABLE goods_receipt_events (
    id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    seq              bigint      GENERATED ALWAYS AS IDENTITY,
    goods_receipt_id uuid        NOT NULL REFERENCES goods_receipts (id) ON DELETE RESTRICT,
    action           text        NOT NULL,
    from_status      text,
    to_status        text        NOT NULL,
    actor_id         uuid        REFERENCES users (id) ON DELETE RESTRICT,
    note             text,
    created_at       timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT goods_receipt_events_action_valid CHECK (action IN ('CREATED', 'CANCELLED', 'POSTED')),
    CONSTRAINT goods_receipt_events_note_length CHECK (note IS NULL OR length(note) <= 500)
);
CREATE INDEX goods_receipt_events_receipt_idx ON goods_receipt_events (goods_receipt_id, seq);

-- History guard: receipts, documents and events are never deleted; documents
-- and events never change; lines and units change (or are removed) only while
-- their receipt is a draft. A posted receipt is immutable (PO-005).
CREATE FUNCTION goods_receipt_history_guard() RETURNS trigger AS $$
DECLARE
    receipt_id     uuid;
    receipt_status text;
BEGIN
    IF TG_TABLE_NAME = 'goods_receipts' THEN
        RAISE EXCEPTION 'goods receipts are never deleted; cancel the draft instead' USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF TG_TABLE_NAME IN ('goods_receipt_events', 'goods_receipt_documents') THEN
        RAISE EXCEPTION 'goods receipt history cannot change' USING ERRCODE = 'insufficient_privilege';
    END IF;
    -- goods_receipt_lines, goods_receipt_units
    IF TG_OP = 'UPDATE' AND NEW.goods_receipt_id IS DISTINCT FROM OLD.goods_receipt_id THEN
        RAISE EXCEPTION 'a receipt line or unit cannot move to another receipt' USING ERRCODE = 'insufficient_privilege';
    END IF;
    receipt_id := CASE WHEN TG_OP = 'INSERT' THEN NEW.goods_receipt_id ELSE OLD.goods_receipt_id END;
    SELECT status INTO receipt_status FROM goods_receipts WHERE id = receipt_id;
    IF receipt_status IS DISTINCT FROM 'DRAFT' THEN
        RAISE EXCEPTION 'lines of a % goods receipt cannot change', receipt_status USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER goods_receipts_no_delete BEFORE DELETE ON goods_receipts FOR EACH ROW EXECUTE FUNCTION goods_receipt_history_guard();
CREATE TRIGGER goods_receipt_events_immutable BEFORE UPDATE OR DELETE ON goods_receipt_events FOR EACH ROW EXECUTE FUNCTION goods_receipt_history_guard();
CREATE TRIGGER goods_receipt_documents_immutable BEFORE UPDATE OR DELETE ON goods_receipt_documents FOR EACH ROW EXECUTE FUNCTION goods_receipt_history_guard();
CREATE TRIGGER goods_receipt_lines_guard BEFORE INSERT OR UPDATE OR DELETE ON goods_receipt_lines FOR EACH ROW EXECUTE FUNCTION goods_receipt_history_guard();
CREATE TRIGGER goods_receipt_units_guard BEFORE INSERT OR UPDATE OR DELETE ON goods_receipt_units FOR EACH ROW EXECUTE FUNCTION goods_receipt_history_guard();
