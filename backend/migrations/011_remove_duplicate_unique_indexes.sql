-- 011_remove_duplicate_unique_indexes.sql
-- 移除重复的唯一索引。
-- 这些字段已通过 schema 字段级 Unique 声明唯一约束；历史迁移或索引声明可能又创建了重复索引。
-- 本迁移只清理已知冗余索引；索引不存在时跳过，保证全新安装和旧库升级都可重复执行。
--
-- 常见命名约定：
-- - 字段级 Unique 创建的索引名: <table>_<field>_key
-- - Indexes 中 Unique 创建的索引名: <table>_<field>
-- - 初始化迁移中的普通索引: idx_<table>_<field>

-- api_keys 的 key 字段
DROP INDEX IF EXISTS apikey_key ON api_keys;
DROP INDEX IF EXISTS api_keys_key ON api_keys;
DROP INDEX IF EXISTS idx_api_keys_key ON api_keys;

-- users 的 email 字段
DROP INDEX IF EXISTS user_email ON users;
DROP INDEX IF EXISTS users_email ON users;
DROP INDEX IF EXISTS idx_users_email ON users;

-- settings 的 key 字段
DROP INDEX IF EXISTS settings_key ON settings;
DROP INDEX IF EXISTS idx_settings_key ON settings;

-- redeem_codes 的 code 字段
DROP INDEX IF EXISTS redeemcode_code ON redeem_codes;
DROP INDEX IF EXISTS redeem_codes_code ON redeem_codes;
DROP INDEX IF EXISTS idx_redeem_codes_code ON redeem_codes;

-- groups 的 name 字段
DROP INDEX IF EXISTS group_name ON `groups`;
DROP INDEX IF EXISTS groups_name ON `groups`;
DROP INDEX IF EXISTS idx_groups_name ON `groups`;

-- 注意：字段级唯一约束对应的索引会保留，例如 api_keys_key_key、users_email_key 等。