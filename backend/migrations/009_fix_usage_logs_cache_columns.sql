-- Ensure usage_logs cache token columns use the underscored names expected by code.

SET @sub2api_add_usage_logs_cache_5m_sql = (
    SELECT IF(
        EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = DATABASE()
              AND table_name = 'usage_logs'
              AND column_name = 'cache_creation_5m_tokens'
        ),
        'DO 0',
        'ALTER TABLE usage_logs ADD COLUMN cache_creation_5m_tokens INT NOT NULL DEFAULT 0'
    )
);

PREPARE sub2api_add_usage_logs_cache_5m_stmt FROM @sub2api_add_usage_logs_cache_5m_sql;
EXECUTE sub2api_add_usage_logs_cache_5m_stmt;
DEALLOCATE PREPARE sub2api_add_usage_logs_cache_5m_stmt;

SET @sub2api_add_usage_logs_cache_1h_sql = (
    SELECT IF(
        EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = DATABASE()
              AND table_name = 'usage_logs'
              AND column_name = 'cache_creation_1h_tokens'
        ),
        'DO 0',
        'ALTER TABLE usage_logs ADD COLUMN cache_creation_1h_tokens INT NOT NULL DEFAULT 0'
    )
);

PREPARE sub2api_add_usage_logs_cache_1h_stmt FROM @sub2api_add_usage_logs_cache_1h_sql;
EXECUTE sub2api_add_usage_logs_cache_1h_stmt;
DEALLOCATE PREPARE sub2api_add_usage_logs_cache_1h_stmt;

-- Legacy PostgreSQL column-name backfill is intentionally skipped for GoldenDB.
-- Existing production data should be transformed by the PostgreSQL-to-GoldenDB
-- data migration job before these application migrations run.
