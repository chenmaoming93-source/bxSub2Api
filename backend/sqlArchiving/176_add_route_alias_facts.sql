-- Persist the final matched model-routing alias on success and error facts.
-- Historical rows stay NULL; this file deliberately performs no JSON or data
-- backfill. It is safe to execute again because every DDL statement is guarded
-- by information_schema checks.

-- usage_logs.route_alias
SET @sub2api_add_usage_route_alias = (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `usage_logs` ADD COLUMN `route_alias` VARCHAR(255) NULL',
    'DO 0'
  )
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'usage_logs'
    AND column_name = 'route_alias'
);
PREPARE sub2api_add_usage_route_alias_stmt FROM @sub2api_add_usage_route_alias;
EXECUTE sub2api_add_usage_route_alias_stmt;
DEALLOCATE PREPARE sub2api_add_usage_route_alias_stmt;

-- ops_error_logs.route_alias
SET @sub2api_add_ops_route_alias = (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `ops_error_logs` ADD COLUMN `route_alias` VARCHAR(255) NULL',
    'DO 0'
  )
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'ops_error_logs'
    AND column_name = 'route_alias'
);
PREPARE sub2api_add_ops_route_alias_stmt FROM @sub2api_add_ops_route_alias;
EXECUTE sub2api_add_ops_route_alias_stmt;
DEALLOCATE PREPARE sub2api_add_ops_route_alias_stmt;

-- Group-scoped observability queries.
SET @sub2api_add_usage_group_route_time_index = (
  SELECT IF(
    COUNT(*) = 0,
    'CREATE INDEX `idx_usage_logs_group_route_time` ON `usage_logs` (`group_id`, `route_alias`, `created_at`)',
    'DO 0'
  )
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'usage_logs'
    AND index_name = 'idx_usage_logs_group_route_time'
);
PREPARE sub2api_add_usage_group_route_time_index_stmt FROM @sub2api_add_usage_group_route_time_index;
EXECUTE sub2api_add_usage_group_route_time_index_stmt;
DEALLOCATE PREPARE sub2api_add_usage_group_route_time_index_stmt;

SET @sub2api_add_ops_group_route_time_index = (
  SELECT IF(
    COUNT(*) = 0,
    'CREATE INDEX `idx_ops_error_logs_group_route_time` ON `ops_error_logs` (`group_id`, `route_alias`, `created_at`)',
    'DO 0'
  )
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'ops_error_logs'
    AND index_name = 'idx_ops_error_logs_group_route_time'
);
PREPARE sub2api_add_ops_group_route_time_index_stmt FROM @sub2api_add_ops_group_route_time_index;
EXECUTE sub2api_add_ops_group_route_time_index_stmt;
DEALLOCATE PREPARE sub2api_add_ops_group_route_time_index_stmt;

-- Route-only trend/filter queries that do not constrain group_id.
SET @sub2api_add_usage_route_time_index = (
  SELECT IF(
    COUNT(*) = 0,
    'CREATE INDEX `idx_usage_logs_route_time` ON `usage_logs` (`route_alias`, `created_at`)',
    'DO 0'
  )
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'usage_logs'
    AND index_name = 'idx_usage_logs_route_time'
);
PREPARE sub2api_add_usage_route_time_index_stmt FROM @sub2api_add_usage_route_time_index;
EXECUTE sub2api_add_usage_route_time_index_stmt;
DEALLOCATE PREPARE sub2api_add_usage_route_time_index_stmt;

SET @sub2api_add_ops_route_time_index = (
  SELECT IF(
    COUNT(*) = 0,
    'CREATE INDEX `idx_ops_error_logs_route_time` ON `ops_error_logs` (`route_alias`, `created_at`)',
    'DO 0'
  )
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'ops_error_logs'
    AND index_name = 'idx_ops_error_logs_route_time'
);
PREPARE sub2api_add_ops_route_time_index_stmt FROM @sub2api_add_ops_route_time_index;
EXECUTE sub2api_add_ops_route_time_index_stmt;
DEALLOCATE PREPARE sub2api_add_ops_route_time_index_stmt;
