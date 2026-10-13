-- Rolling back removes the audit history. Only run after a verified backup
-- (cmd/migrate refuses down in staging/production without --confirm).
DROP TABLE IF EXISTS audit_events;
DROP FUNCTION IF EXISTS audit_events_reject_change();
