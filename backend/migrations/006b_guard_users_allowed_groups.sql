-- Ensure the legacy compatibility column exists before 007 backfills it.
SET @sub2api_guard_users_allowed_groups_sql = (
    SELECT IF(
        EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = DATABASE()
              AND table_name = 'users'
              AND column_name = 'allowed_groups'
        ),
        'DO 0',
        'ALTER TABLE users ADD COLUMN allowed_groups JSON'
    )
);

PREPARE sub2api_guard_users_allowed_groups_stmt FROM @sub2api_guard_users_allowed_groups_sql;
EXECUTE sub2api_guard_users_allowed_groups_stmt;
DEALLOCATE PREPARE sub2api_guard_users_allowed_groups_stmt;
