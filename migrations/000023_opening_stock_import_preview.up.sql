-- G-06: opening stock import preview (ADR-030; IMP-001…IMP-005, FR-014).
--
-- A CSV file in the opening-stock template is uploaded, stored privately (the
-- B-09 object store and an attachments row owned by the batch), parsed and
-- validated row by row against the catalogue, locations, suppliers and existing
-- vehicle identifiers. The result is staged here as an immutable batch with one
-- row per source line. NOTHING ELSE CHANGES: no vehicle unit, receipt, stock
-- movement, balance or catalogue record is created or modified. Committing a
-- batch is G-07, which will add its own columns/transitions.

INSERT INTO permissions (key, description, sensitive) VALUES
    ('stock.import.preview', 'Upload opening-stock files and view their validation previews (creates no stock)', true);

-- The original file is a private attachment of the batch. CSV is accepted only
-- through the import route (the generic attachment upload keeps its allow-list).
ALTER TABLE attachments DROP CONSTRAINT attachments_content_type_valid;
ALTER TABLE attachments ADD CONSTRAINT attachments_content_type_valid
    CHECK (content_type IN ('image/jpeg', 'image/png', 'image/webp', 'application/pdf', 'text/csv'));

CREATE SEQUENCE opening_stock_import_number_seq;

CREATE TABLE opening_stock_import_batches (
    id                 uuid        PRIMARY KEY,
    reference          text        NOT NULL DEFAULT 'OSI-' || lpad(nextval('opening_stock_import_number_seq')::text, 6, '0'),
    template_version   text,
    file_attachment_id uuid        NOT NULL REFERENCES attachments (id) ON DELETE RESTRICT,
    file_name          text        NOT NULL,
    file_sha256        text        NOT NULL,
    file_size_bytes    bigint      NOT NULL,
    uploaded_by        uuid        NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    uploaded_at        timestamptz NOT NULL DEFAULT now(),
    -- Per uploader: a repeat with the same key returns this batch (same file) or is refused.
    idempotency_key    text        NOT NULL,
    -- PREVIEWED: rows were validated. REJECTED: the file itself is unusable (no rows staged).
    status             text        NOT NULL,
    -- The business date dates were checked against (E-07 calendar).
    business_date      date        NOT NULL,
    -- After this a preview is history only: G-07 must refuse to commit it.
    expires_at         timestamptz NOT NULL,
    total_rows         integer     NOT NULL,
    valid_rows         integer     NOT NULL,
    warning_rows       integer     NOT NULL,
    error_rows         integer     NOT NULL,
    blocked_rows       integer     NOT NULL,
    duplicate_rows     integer     NOT NULL,
    unresolved_rows    integer     NOT NULL,
    skipped_blank_rows integer     NOT NULL,
    file_issues        jsonb       NOT NULL DEFAULT '[]',

    CONSTRAINT opening_stock_import_batches_reference_key UNIQUE (reference),
    CONSTRAINT opening_stock_import_batches_attachment_key UNIQUE (file_attachment_id),
    CONSTRAINT opening_stock_import_batches_status_valid CHECK (status IN ('PREVIEWED', 'REJECTED')),
    CONSTRAINT opening_stock_import_batches_sha256_format CHECK (file_sha256 ~ '^[0-9a-f]{64}$'),
    CONSTRAINT opening_stock_import_batches_size_positive CHECK (file_size_bytes > 0),
    CONSTRAINT opening_stock_import_batches_key_format CHECK (idempotency_key ~ '^[A-Za-z0-9_-]{8,100}$'),
    CONSTRAINT opening_stock_import_batches_name_length CHECK (length(btrim(file_name)) > 0 AND length(file_name) <= 200),
    CONSTRAINT opening_stock_import_batches_expiry_after_upload CHECK (expires_at > uploaded_at),
    CONSTRAINT opening_stock_import_batches_counts_valid CHECK (
        total_rows >= 0 AND valid_rows >= 0 AND warning_rows >= 0 AND error_rows >= 0 AND blocked_rows >= 0
        AND duplicate_rows >= 0 AND unresolved_rows >= 0 AND skipped_blank_rows >= 0
        AND total_rows = valid_rows + warning_rows + error_rows + blocked_rows
        AND duplicate_rows <= total_rows AND unresolved_rows <= total_rows),
    CONSTRAINT opening_stock_import_batches_rejected_has_no_rows CHECK (status <> 'REJECTED' OR total_rows = 0),
    CONSTRAINT opening_stock_import_batches_issues_array CHECK (jsonb_typeof(file_issues) = 'array')
);
CREATE UNIQUE INDEX opening_stock_import_batches_idempotency_key ON opening_stock_import_batches (uploaded_by, idempotency_key);
CREATE INDEX opening_stock_import_batches_uploader_idx ON opening_stock_import_batches (uploaded_by, uploaded_at DESC);
CREATE INDEX opening_stock_import_batches_sha256_idx ON opening_stock_import_batches (file_sha256);

CREATE TABLE opening_stock_import_rows (
    batch_id           uuid    NOT NULL REFERENCES opening_stock_import_batches (id) ON DELETE RESTRICT,
    -- The line number in the uploaded file (1 = the template marker line).
    row_number         integer NOT NULL,
    status             text    NOT NULL,
    can_proceed        boolean NOT NULL,
    duplicate          boolean NOT NULL,
    unresolved         boolean NOT NULL,
    -- Every template column with the value exactly as uploaded (strings).
    source             jsonb   NOT NULL,
    -- Resolved references for G-07 (NULL when not resolved).
    brand_id           uuid    REFERENCES brands (id) ON DELETE RESTRICT,
    model_id           uuid    REFERENCES models (id) ON DELETE RESTRICT,
    variant_id         uuid    REFERENCES variants (id) ON DELETE RESTRICT,
    colour_id          uuid    REFERENCES variant_colours (id) ON DELETE RESTRICT,
    location_id        uuid    REFERENCES locations (id) ON DELETE RESTRICT,
    supplier_id        uuid    REFERENCES suppliers (id) ON DELETE RESTRICT,
    -- Normalised values (trimmed; identifiers also upper-cased, as D-04/D-05).
    chassis_number     text,
    chassis_normalised text,
    motor_normalised   text,
    battery_normalised text,
    charger_normalised text,
    stock_status       text,
    received_date      date,
    acquisition_cost   numeric(12,2),
    -- Display snapshot of the resolution (names and statuses at preview time).
    resolved           jsonb   NOT NULL DEFAULT '{}',
    -- [{field, code, severity: ERROR|BLOCKED|WARNING, message, suggestion}]
    issues             jsonb   NOT NULL DEFAULT '[]',

    PRIMARY KEY (batch_id, row_number),
    CONSTRAINT opening_stock_import_rows_row_number_positive CHECK (row_number >= 3),
    CONSTRAINT opening_stock_import_rows_status_valid CHECK (status IN ('VALID', 'WARNING', 'ERROR', 'BLOCKED_UNVERIFIED_MAPPING')),
    CONSTRAINT opening_stock_import_rows_can_proceed CHECK (can_proceed = (status IN ('VALID', 'WARNING'))),
    CONSTRAINT opening_stock_import_rows_json_shapes CHECK (jsonb_typeof(source) = 'object' AND jsonb_typeof(resolved) = 'object'
        AND jsonb_typeof(issues) = 'array'),
    CONSTRAINT opening_stock_import_rows_cost_valid CHECK (acquisition_cost IS NULL OR acquisition_cost >= 0)
);
CREATE INDEX opening_stock_import_rows_status_idx ON opening_stock_import_rows (batch_id, status);
CREATE INDEX opening_stock_import_rows_chassis_idx ON opening_stock_import_rows (chassis_normalised) WHERE chassis_normalised IS NOT NULL;

-- A preview is evidence: batches and rows never change or disappear.
CREATE FUNCTION opening_stock_import_guard() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'opening stock import previews are immutable: % is not allowed', TG_OP
        USING ERRCODE = 'insufficient_privilege';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER opening_stock_import_batches_guard BEFORE UPDATE OR DELETE ON opening_stock_import_batches
    FOR EACH ROW EXECUTE FUNCTION opening_stock_import_guard();
CREATE TRIGGER opening_stock_import_batches_no_truncate BEFORE TRUNCATE ON opening_stock_import_batches
    FOR EACH STATEMENT EXECUTE FUNCTION opening_stock_import_guard();
CREATE TRIGGER opening_stock_import_rows_guard BEFORE UPDATE OR DELETE ON opening_stock_import_rows
    FOR EACH ROW EXECUTE FUNCTION opening_stock_import_guard();
CREATE TRIGGER opening_stock_import_rows_no_truncate BEFORE TRUNCATE ON opening_stock_import_rows
    FOR EACH STATEMENT EXECUTE FUNCTION opening_stock_import_guard();
