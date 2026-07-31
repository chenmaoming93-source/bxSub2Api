-- 人工移除旧固定维度 Token 统计与限额表。
-- 适用：MySQL 8 / GoldenDB（MySQL 兼容模式）。
-- 注意：本文件不得接入应用启动、Ent 自动迁移或部署脚本；由生产管理员审核后手工执行。
-- MySQL DDL 会隐式提交，请先完成备份并确认应用版本已不再访问这些表。

-- 执行前：六张待删除表应按实际存量返回；新体系五张表与 usage_logs 必须存在。
SELECT table_name
FROM information_schema.tables
WHERE table_schema = DATABASE()
  AND table_name IN (
    'model_token_daily_usages',
    'user_model_token_daily_usages',
    'group_candidate_token_daily_usages',
    'model_token_daily_limit_configs',
    'user_model_token_daily_limit_configs',
    'group_candidate_token_daily_limit_configs'
  )
ORDER BY table_name;

SELECT table_name
FROM information_schema.tables
WHERE table_schema = DATABASE()
  AND table_name IN (
    'token_stat_projections',
    'token_stat_projection_metrics',
    'token_stat_aggregates',
    'token_stat_quota_rules',
    'token_stat_period_states',
    'usage_logs'
  )
ORDER BY table_name;

-- 先删除使用量表，再删除限额配置表；只处理六张旧表。
DROP TABLE IF EXISTS group_candidate_token_daily_usages;
DROP TABLE IF EXISTS user_model_token_daily_usages;
DROP TABLE IF EXISTS model_token_daily_usages;
DROP TABLE IF EXISTS group_candidate_token_daily_limit_configs;
DROP TABLE IF EXISTS user_model_token_daily_limit_configs;
DROP TABLE IF EXISTS model_token_daily_limit_configs;

-- 执行后：第一条查询应返回 0 行；第二条查询仍应返回六张受保护表。
SELECT table_name
FROM information_schema.tables
WHERE table_schema = DATABASE()
  AND table_name IN (
    'model_token_daily_usages',
    'user_model_token_daily_usages',
    'group_candidate_token_daily_usages',
    'model_token_daily_limit_configs',
    'user_model_token_daily_limit_configs',
    'group_candidate_token_daily_limit_configs'
  )
ORDER BY table_name;

SELECT table_name
FROM information_schema.tables
WHERE table_schema = DATABASE()
  AND table_name IN (
    'token_stat_projections',
    'token_stat_projection_metrics',
    'token_stat_aggregates',
    'token_stat_quota_rules',
    'token_stat_period_states',
    'usage_logs'
  )
ORDER BY table_name;
