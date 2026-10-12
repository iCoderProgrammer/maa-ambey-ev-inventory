-- B-08 fix: failed or refused sign-ins are attempts by an unauthenticated
-- requester, not actions of the targeted account. Record them with actor_type
-- ANONYMOUS (no actor_user_id); the account stays the entity.
ALTER TABLE audit_events DROP CONSTRAINT audit_events_actor_type_valid;
ALTER TABLE audit_events ADD CONSTRAINT audit_events_actor_type_valid
    CHECK (actor_type IN ('USER', 'CLI', 'SYSTEM', 'ANONYMOUS'));
ALTER TABLE audit_events ADD CONSTRAINT audit_events_anonymous_has_no_user
    CHECK (actor_type <> 'ANONYMOUS' OR actor_user_id IS NULL);
