-- Audit rows cannot be changed, so existing ANONYMOUS rows stay. The narrower
-- check is restored NOT VALID: it applies to new rows only.
ALTER TABLE audit_events DROP CONSTRAINT IF EXISTS audit_events_anonymous_has_no_user;
ALTER TABLE audit_events DROP CONSTRAINT audit_events_actor_type_valid;
ALTER TABLE audit_events ADD CONSTRAINT audit_events_actor_type_valid
    CHECK (actor_type IN ('USER', 'CLI', 'SYSTEM')) NOT VALID;
