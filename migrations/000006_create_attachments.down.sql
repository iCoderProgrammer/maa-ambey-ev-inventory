-- Rolls back B-09. Drops attachment metadata (files in storage are left in
-- place for the operator). Fails if a role still holds an attachments.* grant.
DROP TABLE IF EXISTS attachments;
DROP FUNCTION IF EXISTS attachments_guard_change();
DELETE FROM permissions WHERE key IN ('attachments.upload', 'attachments.view');
