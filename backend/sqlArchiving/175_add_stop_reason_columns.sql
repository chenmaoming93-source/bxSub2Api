-- MySQL manual migration: persist request completion reasons.
-- Run once before deploying the new server.
-- If an object already exists, skip only that duplicate statement.

ALTER TABLE usage_logs
  ADD COLUMN stop_reason VARCHAR(64) NULL;

ALTER TABLE ops_error_logs
  ADD COLUMN stop_reason VARCHAR(64) NULL;

-- Verification
SHOW COLUMNS FROM usage_logs LIKE 'stop_reason';
SHOW COLUMNS FROM ops_error_logs LIKE 'stop_reason';
