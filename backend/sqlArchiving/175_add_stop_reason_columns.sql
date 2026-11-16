-- 175_add_stop_reason_columns.sql
-- Hotfix for deployed builds that persist the upstream completion reason.
-- Applies to both successful usage records and failed request telemetry.
-- Idempotent and safe to run while the service is online:
-- adding nullable columns does not rewrite existing rows.

-- usage_logs.stop_reason
SET @sub2api_add_usage_logs_stop_reason = (
    SELECT IF(
        COUNT(*) = 0,
        'ALTER TABLE `usage_logs` ADD COLUMN `stop_reason` VARCHAR(64) NULL AFTER `reasoning_effort`',
        'SELECT 1'
    )
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = 'usage_logs'
      AND column_name = 'stop_reason'
);
PREPARE sub2api_add_usage_logs_stop_reason_stmt FROM @sub2api_add_usage_logs_stop_reason;
EXECUTE sub2api_add_usage_logs_stop_reason_stmt;
DEALLOCATE PREPARE sub2api_add_usage_logs_stop_reason_stmt;

-- ops_error_logs.stop_reason
SET @sub2api_add_ops_error_logs_stop_reason = (
    SELECT IF(
        COUNT(*) = 0,
        'ALTER TABLE `ops_error_logs` ADD COLUMN `stop_reason` VARCHAR(64) NULL AFTER `upstream_model`',
        'SELECT 1'
    )
    FROM information_schema.columns
    WHERE table_schema = DATABASE()
      AND table_name = 'ops_error_logs'
      AND column_name = 'stop_reason'
);
PREPARE sub2api_add_ops_error_logs_stop_reason_stmt FROM @sub2api_add_ops_error_logs_stop_reason;
EXECUTE sub2api_add_ops_error_logs_stop_reason_stmt;
DEALLOCATE PREPARE sub2api_add_ops_error_logs_stop_reason_stmt;

-- Verification: both rows should be returned.
SELECT table_name, column_name, column_type, is_nullable
FROM information_schema.columns
WHERE table_schema = DATABASE()
  AND (
      (table_name = 'usage_logs' AND column_name = 'stop_reason')
      OR (table_name = 'ops_error_logs' AND column_name = 'stop_reason')
  )
ORDER BY table_name;
