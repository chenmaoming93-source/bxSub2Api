-- MySQL manual migration: capacity-card metrics.
-- Use this file instead of auto-migration 159 when the runtime user has no DDL permission.
-- If a column already exists, skip only that duplicate statement.

ALTER TABLE observability_request_bucket
  ADD COLUMN rate_limited_count BIGINT NOT NULL DEFAULT 0;

ALTER TABLE observability_concurrency_bucket
  ADD COLUMN wait_sum_ms BIGINT NOT NULL DEFAULT 0;

ALTER TABLE observability_concurrency_bucket
  ADD COLUMN wait_sample_count BIGINT NOT NULL DEFAULT 0;

-- Verification
SHOW COLUMNS FROM observability_request_bucket LIKE 'rate_limited_count';
SHOW COLUMNS FROM observability_concurrency_bucket LIKE 'wait_sum_ms';
SHOW COLUMNS FROM observability_concurrency_bucket LIKE 'wait_sample_count';
