-- MySQL manual migration: persist the matched model-routing alias.
-- Historical rows remain NULL; no backfill is performed.
-- Do not add new indexes here: this deployment database may already be at
-- MySQL's 64-key limit. Observability reads aggregate tables and does not
-- require these raw-fact indexes for the dashboard to function.
-- If an object already exists, skip only that duplicate statement.

ALTER TABLE usage_logs
  ADD COLUMN route_alias VARCHAR(255) NULL;

ALTER TABLE ops_error_logs
  ADD COLUMN route_alias VARCHAR(255) NULL;

-- Verification
SHOW COLUMNS FROM usage_logs LIKE 'route_alias';
SHOW COLUMNS FROM ops_error_logs LIKE 'route_alias';
