-- MySQL manual migration: persist the matched model-routing alias.
-- Historical rows remain NULL; no backfill is performed.
-- If an object already exists, skip only that duplicate statement.

ALTER TABLE usage_logs
  ADD COLUMN route_alias VARCHAR(255) NULL;

ALTER TABLE ops_error_logs
  ADD COLUMN route_alias VARCHAR(255) NULL;

CREATE INDEX idx_usage_logs_group_route_time
  ON usage_logs (group_id, route_alias, created_at);

CREATE INDEX idx_ops_error_logs_group_route_time
  ON ops_error_logs (group_id, route_alias, created_at);

CREATE INDEX idx_usage_logs_route_time
  ON usage_logs (route_alias, created_at);

CREATE INDEX idx_ops_error_logs_route_time
  ON ops_error_logs (route_alias, created_at);

-- Verification
SHOW COLUMNS FROM usage_logs LIKE 'route_alias';
SHOW COLUMNS FROM ops_error_logs LIKE 'route_alias';
SHOW INDEX FROM usage_logs WHERE Key_name LIKE 'idx_usage_logs_%route_time';
SHOW INDEX FROM ops_error_logs WHERE Key_name LIKE 'idx_ops_error_logs_%route_time';
