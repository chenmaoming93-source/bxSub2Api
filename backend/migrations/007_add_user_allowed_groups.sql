-- Add user_allowed_groups join table to replace users.allowed_groups (JSON array).
-- Phase 1: create table + backfill from the legacy array column.

CREATE TABLE IF NOT EXISTS user_allowed_groups (
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id    BIGINT NOT NULL REFERENCES `groups`(id) ON DELETE CASCADE,
    created_at  DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (user_id, group_id)
);

CREATE INDEX IF NOT EXISTS idx_user_allowed_groups_group_id ON user_allowed_groups(group_id);

-- Backfill from the legacy users.allowed_groups JSON array.
INSERT IGNORE INTO user_allowed_groups (user_id, group_id)
SELECT u.id, x.group_id
FROM users u
JOIN JSON_TABLE(u.allowed_groups, '$[*]' COLUMNS (group_id BIGINT PATH '$')) AS x
JOIN `groups` g ON g.id = x.group_id
WHERE u.allowed_groups IS NOT NULL;
