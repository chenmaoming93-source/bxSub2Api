-- 013: 记录 users.allowed_groups 中的孤立 group_id
-- 任务：fix-medium-data-hygiene 3.1
--
-- 目的：在删除 legacy allowed_groups 列前，记录所有引用了不存在 group 的孤立记录。
-- 这些记录可用于审计或后续数据修复。

-- 创建审计表存储孤立的 allowed_groups 记录
CREATE TABLE IF NOT EXISTS orphan_allowed_groups_audit (
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id     BIGINT NOT NULL,
    group_id    BIGINT NOT NULL,
    recorded_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    UNIQUE (user_id, group_id)
);

-- 记录孤立的 group_id（存在于 users.allowed_groups 但不存在于 groups 表）
INSERT IGNORE INTO orphan_allowed_groups_audit (user_id, group_id)
SELECT u.id, x.group_id
FROM users u
JOIN JSON_TABLE(u.allowed_groups, '$[*]' COLUMNS (group_id BIGINT PATH '$')) AS x
LEFT JOIN groups g ON g.id = x.group_id
WHERE u.allowed_groups IS NOT NULL
  AND g.id IS NULL;

-- 添加索引便于查询
CREATE INDEX IF NOT EXISTS idx_orphan_allowed_groups_audit_user_id
ON orphan_allowed_groups_audit(user_id);
