-- Speed up active platform-key lookup by user, calling platform, and group.
CREATE INDEX IF NOT EXISTS idx_api_keys_user_platform_group_deleted_at
    ON api_keys (user_id, platform, group_id, deleted_at);
