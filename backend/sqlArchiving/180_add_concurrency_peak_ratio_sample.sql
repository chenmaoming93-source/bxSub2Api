-- MySQL manual migration: retain the exact maximum-pressure sample.
-- Use this file instead of auto-migration 160 when the runtime user has no DDL permission.
-- If a column already exists, skip only that duplicate statement.

ALTER TABLE observability_concurrency_bucket
  ADD COLUMN peak_ratio_actual INT NOT NULL DEFAULT 0;

ALTER TABLE observability_concurrency_bucket
  ADD COLUMN peak_ratio_limit INT NOT NULL DEFAULT 0;

ALTER TABLE observability_concurrency_bucket
  ADD COLUMN peak_ratio_sample_at DATETIME(6) NULL;

-- Verification
SHOW COLUMNS FROM observability_concurrency_bucket LIKE 'peak_ratio_actual';
SHOW COLUMNS FROM observability_concurrency_bucket LIKE 'peak_ratio_limit';
SHOW COLUMNS FROM observability_concurrency_bucket LIKE 'peak_ratio_sample_at';
