-- Migration: 045_add_accounts_extra_index
-- 为 accounts.extra 字段添加 GIN 索引，优化 FindByExtraField 查询性能
-- 用于支持通过 extra 字段中的 linked_openai_account_id 快速查找关联的 Sora 账号

-- GoldenDB/MySQL mode cannot create a PostgreSQL GIN index on JSON directly.
-- Repository queries should use JSON_EXTRACT predicates; add generated-column
-- indexes in a dedicated migration if a specific JSON path becomes hot.

-- 查询示例（使用 @> 操作符）
-- EXPLAIN ANALYZE
-- SELECT * FROM accounts
-- WHERE platform = 'sora'
--   AND extra @> '{"linked_openai_account_id": 123}'
--   AND deleted_at IS NULL;
