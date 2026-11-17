-- E-07: stock ageing bands (ADR-029; LED-008, REP-005).
--
-- Stock age itself is never stored: it is calculated on read from the unit's
-- approved origin date (the posted goods receipt's received date) and the
-- business date. What is stored here is only the band configuration, as
-- immutable versions: a change adds a new version, so earlier configurations
-- remain as history and no stock record is ever rewritten.
--
-- Boundary convention: a band covers min_days <= age <= max_days (whole days,
-- both ends inclusive); the last band has no max_days (open-ended). A set starts
-- at 0, has no gap and no overlap (each band starts the day after the previous
-- one ends) and is checked again at commit.
--
-- Version 1 is the Business Rules LED-008 "recommended defaults" (0–30, 31–60,
-- 61–90, >90 days). It is recorded as PROPOSED: the owner has not approved
-- thresholds yet. Approval (by someone other than the proposer) is recorded on
-- the version; nothing else about a version can change.

INSERT INTO permissions (key, description, sensitive) VALUES
    ('stock.ageing.configure', 'Propose and approve stock ageing band configurations (approval needs a different person than the proposer)', true);

CREATE TABLE stock_ageing_band_sets (
    version            integer     PRIMARY KEY,
    status             text        NOT NULL DEFAULT 'PROPOSED',
    -- Where the thresholds come from (document, decision reference).
    source             text        NOT NULL,
    -- NULL: recorded by this migration (no person proposed it).
    created_by         uuid        REFERENCES users (id) ON DELETE RESTRICT,
    created_at         timestamptz NOT NULL DEFAULT now(),
    approved_by        uuid        REFERENCES users (id) ON DELETE RESTRICT,
    approved_at        timestamptz,
    approval_reference text,

    CONSTRAINT stock_ageing_band_sets_version_positive CHECK (version > 0),
    CONSTRAINT stock_ageing_band_sets_status_valid CHECK (status IN ('PROPOSED', 'APPROVED')),
    CONSTRAINT stock_ageing_band_sets_source_length CHECK (length(btrim(source)) > 0 AND length(source) <= 300),
    CONSTRAINT stock_ageing_band_sets_approval_complete CHECK (
        (status = 'PROPOSED' AND approved_by IS NULL AND approved_at IS NULL AND approval_reference IS NULL)
     OR (status = 'APPROVED' AND approved_by IS NOT NULL AND approved_at IS NOT NULL)),
    CONSTRAINT stock_ageing_band_sets_no_self_approval CHECK (approved_by IS NULL OR created_by IS NULL OR approved_by <> created_by),
    CONSTRAINT stock_ageing_band_sets_reference_length CHECK (approval_reference IS NULL
        OR (length(btrim(approval_reference)) > 0 AND length(approval_reference) <= 200))
);

CREATE TABLE stock_ageing_bands (
    set_version integer  NOT NULL REFERENCES stock_ageing_band_sets (version) ON DELETE RESTRICT,
    position    smallint NOT NULL,
    key         text     NOT NULL,
    label       text     NOT NULL,
    min_days    integer  NOT NULL,
    max_days    integer,

    PRIMARY KEY (set_version, position),
    CONSTRAINT stock_ageing_bands_key_unique UNIQUE (set_version, key),
    CONSTRAINT stock_ageing_bands_position_positive CHECK (position >= 1 AND position <= 20),
    CONSTRAINT stock_ageing_bands_key_format CHECK (key ~ '^[a-z0-9][a-z0-9_-]{0,31}$'),
    CONSTRAINT stock_ageing_bands_label_length CHECK (length(btrim(label)) > 0 AND length(label) <= 60),
    CONSTRAINT stock_ageing_bands_days_valid CHECK (min_days >= 0 AND min_days <= 36500
        AND (max_days IS NULL OR (max_days >= min_days AND max_days <= 36500)))
);

-- Versions never disappear and bands never change. The one allowed change to a
-- version is its approval (PROPOSED -> APPROVED, approval fields only).
CREATE FUNCTION stock_ageing_history_guard() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'UPDATE' AND TG_TABLE_NAME = 'stock_ageing_band_sets'
        AND OLD.status = 'PROPOSED' AND NEW.status = 'APPROVED'
        AND NEW.version = OLD.version AND NEW.source = OLD.source
        AND NEW.created_by IS NOT DISTINCT FROM OLD.created_by AND NEW.created_at = OLD.created_at THEN
        RETURN NEW;
    END IF;
    RAISE EXCEPTION 'stock ageing band configurations are append-only: % is not allowed', TG_OP
        USING ERRCODE = 'insufficient_privilege';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER stock_ageing_band_sets_guard BEFORE UPDATE OR DELETE ON stock_ageing_band_sets
    FOR EACH ROW EXECUTE FUNCTION stock_ageing_history_guard();
CREATE TRIGGER stock_ageing_band_sets_no_truncate BEFORE TRUNCATE ON stock_ageing_band_sets
    FOR EACH STATEMENT EXECUTE FUNCTION stock_ageing_history_guard();
CREATE TRIGGER stock_ageing_bands_guard BEFORE UPDATE OR DELETE ON stock_ageing_bands
    FOR EACH ROW EXECUTE FUNCTION stock_ageing_history_guard();
CREATE TRIGGER stock_ageing_bands_no_truncate BEFORE TRUNCATE ON stock_ageing_bands
    FOR EACH STATEMENT EXECUTE FUNCTION stock_ageing_history_guard();

-- At commit, a version's bands start at 0, follow each other without gap or
-- overlap, and only the last one is open-ended.
CREATE FUNCTION stock_ageing_band_set_check() RETURNS trigger AS $$
DECLARE
    b        record;
    expected integer := 0;
    n        integer := 0;
    open_end boolean := false;
BEGIN
    FOR b IN SELECT min_days, max_days FROM stock_ageing_bands WHERE set_version = NEW.set_version ORDER BY position LOOP
        n := n + 1;
        IF open_end OR b.min_days <> expected THEN
            RAISE EXCEPTION 'ageing bands of version % must start at 0 and follow each other without gap or overlap', NEW.set_version
                USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_ageing_bands_contiguous';
        END IF;
        IF b.max_days IS NULL THEN
            open_end := true;
        ELSE
            expected := b.max_days + 1;
        END IF;
    END LOOP;
    IF n = 0 OR NOT open_end THEN
        RAISE EXCEPTION 'the last ageing band of version % must be open-ended', NEW.set_version
            USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_ageing_bands_contiguous';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER stock_ageing_bands_contiguous AFTER INSERT ON stock_ageing_bands
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION stock_ageing_band_set_check();

-- A version without bands is refused at commit too.
CREATE FUNCTION stock_ageing_band_set_has_bands() RETURNS trigger AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM stock_ageing_bands WHERE set_version = NEW.version) THEN
        RAISE EXCEPTION 'ageing band version % has no bands', NEW.version
            USING ERRCODE = 'check_violation', CONSTRAINT = 'stock_ageing_bands_contiguous';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER stock_ageing_band_sets_has_bands AFTER INSERT ON stock_ageing_band_sets
    DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION stock_ageing_band_set_has_bands();

INSERT INTO stock_ageing_band_sets (version, status, source) VALUES
    (1, 'PROPOSED', 'Business Rules v1.2 LED-008 recommended defaults (0–30, 31–60, 61–90, >90 days); owner approval pending');
INSERT INTO stock_ageing_bands (set_version, position, key, label, min_days, max_days) VALUES
    (1, 1, '0-30',    '0–30 days',    0,  30),
    (1, 2, '31-60',   '31–60 days',   31, 60),
    (1, 3, '61-90',   '61–90 days',   61, 90),
    (1, 4, 'over-90', 'Over 90 days', 91, NULL);
