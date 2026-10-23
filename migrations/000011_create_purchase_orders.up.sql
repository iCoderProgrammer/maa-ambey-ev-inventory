-- Purchase order lifecycle (D-02; PO-001, PO-002, PO-003, PO-007, Architecture
-- "Purchase order lifecycle"). Schema only: no purchase order, line or approval
-- threshold is seeded.
--
-- A PO has a unique number, a supplier, a creator and lines of one variant each
-- (brand/model/variant kept consistent by composite foreign keys). Each line
-- stores the agreed unit cost and, when a cost price list is named, a snapshot
-- of the approved catalogue cost valid at that moment: later price approvals
-- never change an existing PO. Totals and remaining quantities are computed by
-- the server from the lines, never taken from the client.
--
-- States: DRAFT -> SUBMITTED -> APPROVED -> PARTIALLY_RECEIVED -> RECEIVED ->
-- CLOSED, CANCELLED where allowed. Goods receipts (D-03) move APPROVED onwards
-- and maintain received_quantity. Lines change only while the PO is a draft.
-- Nothing is ever deleted except a line of a draft.

INSERT INTO permissions (key, description, sensitive) VALUES
    ('purchase_orders.view',            'View purchase orders, their lines, agreed costs and history', true),
    ('purchase_orders.manage',          'Create and edit draft purchase orders and submit them', true),
    ('purchase_orders.approve',         'Approve submitted purchase orders or return them to draft (never one''s own)', true),
    ('purchase_orders.cancel',          'Cancel purchase orders that have nothing received', true),
    ('purchase_orders.settings.manage', 'Change the purchase order approval threshold settings', true);

-- PO numbers: PO-000001, PO-000002, ... Gaps are possible (a rolled-back
-- create uses a number); the number is an internal reference, not a tax document.
CREATE SEQUENCE purchase_order_number_seq AS bigint START 1 NO CYCLE;

-- Approval settings (PO-001): one row. approval_threshold NULL means "not
-- configured": every PO then needs an approver. A threshold applies only with
-- the owner's sign-off reference recorded.
CREATE TABLE purchase_order_settings (
    id                                 boolean       PRIMARY KEY DEFAULT true,
    approval_threshold                 numeric(14,2),
    first_time_supplier_needs_approval boolean       NOT NULL DEFAULT true,
    approval_reference                 text,
    updated_by                         uuid          REFERENCES users (id) ON DELETE RESTRICT,
    version                            integer       NOT NULL DEFAULT 1,
    created_at                         timestamptz   NOT NULL DEFAULT now(),
    updated_at                         timestamptz   NOT NULL DEFAULT now(),

    CONSTRAINT purchase_order_settings_single_row CHECK (id),
    CONSTRAINT purchase_order_settings_threshold_positive CHECK (approval_threshold IS NULL OR approval_threshold > 0),
    CONSTRAINT purchase_order_settings_threshold_signed_off CHECK (approval_threshold IS NULL OR approval_reference IS NOT NULL),
    CONSTRAINT purchase_order_settings_reference_length CHECK (approval_reference IS NULL OR (length(btrim(approval_reference)) > 0 AND length(approval_reference) <= 200)),
    CONSTRAINT purchase_order_settings_version_positive CHECK (version > 0)
);
-- The row says "nothing configured"; it is not a business value.
INSERT INTO purchase_order_settings DEFAULT VALUES;
CREATE TRIGGER purchase_order_settings_set_updated_at BEFORE UPDATE ON purchase_order_settings FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE purchase_orders (
    id                 uuid          PRIMARY KEY DEFAULT gen_random_uuid(),
    po_number          text          NOT NULL DEFAULT ('PO-' || lpad(nextval('purchase_order_number_seq')::text, 6, '0')),
    supplier_id        uuid          NOT NULL REFERENCES suppliers (id) ON DELETE RESTRICT,
    status             text          NOT NULL DEFAULT 'DRAFT',
    expected_date      date,
    notes              text,
    -- Amounts are Indian rupees, excluding tax: tax treatment needs the accountant's configuration (F-10).
    currency           text          NOT NULL DEFAULT 'INR',
    created_by         uuid          NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    -- Repeating a create with the same key returns the first PO (double-submit protection).
    idempotency_key    text,
    submitted_by       uuid          REFERENCES users (id) ON DELETE RESTRICT,
    submitted_at       timestamptz,
    -- APPROVER: approved by a person; BELOW_THRESHOLD: no approval needed under the signed-off threshold.
    approval_basis     text,
    approved_by        uuid          REFERENCES users (id) ON DELETE RESTRICT,
    approved_at        timestamptz,
    approval_note      text,
    -- The threshold applied at submission (NULL: not configured).
    threshold_applied  numeric(14,2),
    cancelled_by       uuid          REFERENCES users (id) ON DELETE RESTRICT,
    cancelled_at       timestamptz,
    cancel_reason      text,
    version            integer       NOT NULL DEFAULT 1,
    created_at         timestamptz   NOT NULL DEFAULT now(),
    updated_at         timestamptz   NOT NULL DEFAULT now(),

    CONSTRAINT purchase_orders_number_key UNIQUE (po_number),
    CONSTRAINT purchase_orders_status_valid CHECK (status IN
        ('DRAFT', 'SUBMITTED', 'APPROVED', 'PARTIALLY_RECEIVED', 'RECEIVED', 'CLOSED', 'CANCELLED')),
    CONSTRAINT purchase_orders_currency_valid CHECK (currency = 'INR'),
    CONSTRAINT purchase_orders_notes_length CHECK (notes IS NULL OR (length(btrim(notes)) > 0 AND length(notes) <= 1000)),
    CONSTRAINT purchase_orders_idempotency_key_length CHECK (idempotency_key IS NULL OR length(idempotency_key) BETWEEN 8 AND 100),
    CONSTRAINT purchase_orders_submitted_pair CHECK ((submitted_by IS NULL) = (submitted_at IS NULL)),
    CONSTRAINT purchase_orders_submitted_when_past_draft CHECK (status IN ('DRAFT', 'CANCELLED') OR submitted_at IS NOT NULL),
    CONSTRAINT purchase_orders_approval_basis_valid CHECK (approval_basis IS NULL OR approval_basis IN ('APPROVER', 'BELOW_THRESHOLD')),
    -- Approved (and later) states record how they were approved; a person approval names the person.
    CONSTRAINT purchase_orders_approved_has_basis CHECK (
        status NOT IN ('APPROVED', 'PARTIALLY_RECEIVED', 'RECEIVED', 'CLOSED') OR (approval_basis IS NOT NULL AND approved_at IS NOT NULL)),
    CONSTRAINT purchase_orders_approver_matches_basis CHECK ((approval_basis IS NOT DISTINCT FROM 'APPROVER') = (approved_by IS NOT NULL)),
    -- Maker-checker: the creator and the submitter never approve their own PO (OQ-21).
    CONSTRAINT purchase_orders_not_self_approved CHECK (approved_by IS NULL OR (approved_by <> created_by AND approved_by <> submitted_by)),
    CONSTRAINT purchase_orders_approval_note_length CHECK (approval_note IS NULL OR (length(btrim(approval_note)) > 0 AND length(approval_note) <= 500)),
    CONSTRAINT purchase_orders_cancelled_complete CHECK ((status = 'CANCELLED') = (cancelled_by IS NOT NULL AND cancelled_at IS NOT NULL AND cancel_reason IS NOT NULL)),
    CONSTRAINT purchase_orders_cancel_reason_length CHECK (cancel_reason IS NULL OR (length(btrim(cancel_reason)) > 0 AND length(cancel_reason) <= 500)),
    CONSTRAINT purchase_orders_version_positive CHECK (version > 0)
);
CREATE UNIQUE INDEX purchase_orders_idempotency_key ON purchase_orders (created_by, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX purchase_orders_supplier_idx ON purchase_orders (supplier_id, status);
CREATE INDEX purchase_orders_status_idx ON purchase_orders (status, created_at DESC);
CREATE TRIGGER purchase_orders_set_updated_at BEFORE UPDATE ON purchase_orders FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE purchase_order_lines (
    id                       uuid          PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_order_id        uuid          NOT NULL REFERENCES purchase_orders (id) ON DELETE RESTRICT,
    line_no                  integer       NOT NULL,
    brand_id                 uuid          NOT NULL,
    model_id                 uuid          NOT NULL,
    variant_id               uuid          NOT NULL,
    quantity                 integer       NOT NULL,
    -- Agreed cost per unit with the supplier (PO-001), excluding tax.
    unit_cost                numeric(12,2) NOT NULL,
    -- Maintained by goods receipts (D-03); remaining = quantity - received_quantity.
    received_quantity        integer       NOT NULL DEFAULT 0,
    -- Snapshot of the approved catalogue COST price valid when the line was saved (C-04).
    reference_price_list_id  uuid,
    reference_price_item_id  uuid          REFERENCES price_items (id) ON DELETE RESTRICT,
    reference_unit_cost      numeric(12,2),
    reference_captured_at    timestamptz,
    note                     text,
    created_at               timestamptz   NOT NULL DEFAULT now(),
    updated_at               timestamptz   NOT NULL DEFAULT now(),

    CONSTRAINT purchase_order_lines_model_fk FOREIGN KEY (model_id, brand_id) REFERENCES models (id, brand_id) ON DELETE RESTRICT,
    CONSTRAINT purchase_order_lines_variant_brand_fk FOREIGN KEY (variant_id, brand_id) REFERENCES variants (id, brand_id) ON DELETE RESTRICT,
    CONSTRAINT purchase_order_lines_variant_model_fk FOREIGN KEY (variant_id, model_id) REFERENCES variants (id, model_id) ON DELETE RESTRICT,
    CONSTRAINT purchase_order_lines_price_list_fk FOREIGN KEY (reference_price_list_id, brand_id) REFERENCES price_lists (id, brand_id) ON DELETE RESTRICT,
    CONSTRAINT purchase_order_lines_no_key UNIQUE (purchase_order_id, line_no),
    -- One line per variant on a PO.
    CONSTRAINT purchase_order_lines_variant_key UNIQUE (purchase_order_id, variant_id),
    CONSTRAINT purchase_order_lines_line_no_positive CHECK (line_no > 0),
    CONSTRAINT purchase_order_lines_quantity_range CHECK (quantity BETWEEN 1 AND 10000),
    CONSTRAINT purchase_order_lines_unit_cost_positive CHECK (unit_cost > 0),
    CONSTRAINT purchase_order_lines_received_non_negative CHECK (received_quantity >= 0),
    CONSTRAINT purchase_order_lines_reference_complete CHECK (
        (reference_price_item_id IS NULL) = (reference_unit_cost IS NULL)
        AND (reference_price_item_id IS NULL OR (reference_price_list_id IS NOT NULL AND reference_captured_at IS NOT NULL))),
    CONSTRAINT purchase_order_lines_note_length CHECK (note IS NULL OR (length(btrim(note)) > 0 AND length(note) <= 300))
);
CREATE INDEX purchase_order_lines_variant_idx ON purchase_order_lines (variant_id);
CREATE INDEX purchase_order_lines_brand_idx ON purchase_order_lines (brand_id);
CREATE TRIGGER purchase_order_lines_set_updated_at BEFORE UPDATE ON purchase_order_lines FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Status history: one row per transition, append-only.
CREATE TABLE purchase_order_events (
    id                uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Order of entries (two transitions in one transaction share created_at).
    seq               bigint      GENERATED ALWAYS AS IDENTITY,
    purchase_order_id uuid        NOT NULL REFERENCES purchase_orders (id) ON DELETE RESTRICT,
    action            text        NOT NULL,
    from_status       text,
    to_status         text        NOT NULL,
    actor_id          uuid        REFERENCES users (id) ON DELETE RESTRICT,
    note              text,
    created_at        timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT purchase_order_events_action_valid CHECK (action IN
        ('CREATED', 'SUBMITTED', 'APPROVED', 'APPROVAL_NOT_REQUIRED', 'RETURNED', 'CANCELLED')),
    CONSTRAINT purchase_order_events_note_length CHECK (note IS NULL OR length(note) <= 500)
);
CREATE INDEX purchase_order_events_po_idx ON purchase_order_events (purchase_order_id, seq);

-- History guard: POs and events are never deleted; events never change; lines
-- change (or are removed) only while their PO is a draft — afterwards only
-- received_quantity may move (goods receipts, D-03).
CREATE FUNCTION purchase_order_history_guard() RETURNS trigger AS $$
DECLARE
    po_status text;
BEGIN
    IF TG_TABLE_NAME = 'purchase_orders' THEN
        RAISE EXCEPTION 'purchase orders are never deleted; cancel them instead' USING ERRCODE = 'insufficient_privilege';
    END IF;
    IF TG_TABLE_NAME = 'purchase_order_events' THEN
        RAISE EXCEPTION 'purchase order history cannot change' USING ERRCODE = 'insufficient_privilege';
    END IF;
    -- purchase_order_lines
    SELECT status INTO po_status FROM purchase_orders
        WHERE id = (CASE WHEN TG_OP = 'INSERT' THEN NEW.purchase_order_id ELSE OLD.purchase_order_id END);
    IF po_status = 'DRAFT' THEN
        IF TG_OP = 'UPDATE' AND NEW.purchase_order_id IS DISTINCT FROM OLD.purchase_order_id THEN
            RAISE EXCEPTION 'a line cannot move to another purchase order' USING ERRCODE = 'insufficient_privilege';
        END IF;
        IF TG_OP = 'DELETE' THEN
            RETURN OLD;
        END IF;
        RETURN NEW;
    END IF;
    IF TG_OP = 'UPDATE' AND (to_jsonb(NEW) - 'received_quantity' - 'updated_at') = (to_jsonb(OLD) - 'received_quantity' - 'updated_at') THEN
        RETURN NEW;
    END IF;
    RAISE EXCEPTION 'lines of a % purchase order cannot change', po_status USING ERRCODE = 'insufficient_privilege';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER purchase_orders_no_delete BEFORE DELETE ON purchase_orders FOR EACH ROW EXECUTE FUNCTION purchase_order_history_guard();
CREATE TRIGGER purchase_order_events_immutable BEFORE UPDATE OR DELETE ON purchase_order_events FOR EACH ROW EXECUTE FUNCTION purchase_order_history_guard();
CREATE TRIGGER purchase_order_lines_guard BEFORE INSERT OR UPDATE OR DELETE ON purchase_order_lines FOR EACH ROW EXECUTE FUNCTION purchase_order_history_guard();
