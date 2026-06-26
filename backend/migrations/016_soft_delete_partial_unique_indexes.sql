-- 016_soft_delete_partial_unique_indexes.sql
-- 修复软删除 + 唯一约束冲突问题。
-- GoldenDB/MySQL 模式不支持 PostgreSQL partial unique index，这里保留普通唯一索引语义。

-- 1. users 表 email 字段
DROP INDEX IF EXISTS users_email_key;
DROP INDEX IF EXISTS user_email_key;

CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique_active
    ON users(email);

-- 2. groups 表 name 字段
DROP INDEX IF EXISTS groups_name_key;
DROP INDEX IF EXISTS group_name_key;

CREATE UNIQUE INDEX IF NOT EXISTS groups_name_unique_active
    ON groups(name);

-- 3. user_subscriptions 表 (user_id, group_id) 组合字段
DROP INDEX IF EXISTS user_subscriptions_user_id_group_id_key;
DROP INDEX IF EXISTS usersubscription_user_id_group_id;

CREATE UNIQUE INDEX IF NOT EXISTS user_subscriptions_user_group_unique_active
    ON user_subscriptions(user_id, group_id);

-- 注意：api_keys 表的 key 字段保留普通唯一约束。
-- API Key 即使软删除后也不应该重复使用（安全考虑）。
