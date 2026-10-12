-- Append-only audit trail (B-08, FR-017, SEC-004). Rows are written by the
-- application inside the same transaction as the change they describe and can
-- never be updated, deleted or truncated through normal paths.

CREATE TABLE audit_events (
    id            bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    occurred_at   timestamptz NOT NULL DEFAULT now(),
    -- USER = signed-in staff member; CLI = operator command (cmd/*); SYSTEM = background job.
    actor_type    text        NOT NULL,
    actor_user_id uuid        REFERENCES users (id) ON DELETE RESTRICT,
    -- Readable actor at the time (username or command); kept even if the account changes later.
    actor_label   text        NOT NULL,
    -- Dotted, past-tense action key, e.g. auth.signed_in, user.disabled.
    action        text        NOT NULL,
    outcome       text        NOT NULL DEFAULT 'SUCCESS',
    entity_type   text        NOT NULL,
    entity_id     text,
    -- Human-facing business reference (document number etc.), when one exists.
    reference     text,
    request_id    text,
    -- Safe structured summary (before/after or changed fields). Secrets are rejected by the API.
    summary       jsonb       NOT NULL DEFAULT '{}'::jsonb,

    CONSTRAINT audit_events_actor_type_valid CHECK (actor_type IN ('USER', 'CLI', 'SYSTEM')),
    CONSTRAINT audit_events_user_actor_has_user CHECK (actor_type <> 'USER' OR actor_user_id IS NOT NULL),
    CONSTRAINT audit_events_actor_label_length CHECK (length(btrim(actor_label)) > 0 AND length(actor_label) <= 120),
    CONSTRAINT audit_events_action_format CHECK (action ~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$' AND length(action) <= 100),
    CONSTRAINT audit_events_outcome_valid CHECK (outcome IN ('SUCCESS', 'DENIED', 'FAILURE')),
    CONSTRAINT audit_events_entity_type_format CHECK (entity_type ~ '^[a-z][a-z0-9_]*$' AND length(entity_type) <= 60),
    CONSTRAINT audit_events_entity_id_length CHECK (entity_id IS NULL OR length(entity_id) <= 100),
    CONSTRAINT audit_events_reference_length CHECK (reference IS NULL OR length(reference) <= 120),
    CONSTRAINT audit_events_request_id_length CHECK (request_id IS NULL OR length(request_id) <= 100),
    CONSTRAINT audit_events_summary_object CHECK (jsonb_typeof(summary) = 'object'),
    CONSTRAINT audit_events_summary_size CHECK (pg_column_size(summary) <= 8192)
);

CREATE INDEX audit_events_occurred_at_idx ON audit_events (occurred_at DESC);
CREATE INDEX audit_events_entity_idx ON audit_events (entity_type, entity_id, occurred_at DESC);
CREATE INDEX audit_events_actor_idx ON audit_events (actor_user_id, occurred_at DESC);
CREATE INDEX audit_events_action_idx ON audit_events (action, occurred_at DESC);

CREATE FUNCTION audit_events_reject_change() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'audit_events is append-only: % is not allowed', TG_OP
        USING ERRCODE = 'insufficient_privilege';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_events_no_update_delete
    BEFORE UPDATE OR DELETE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION audit_events_reject_change();

CREATE TRIGGER audit_events_no_truncate
    BEFORE TRUNCATE ON audit_events
    FOR EACH STATEMENT EXECUTE FUNCTION audit_events_reject_change();
