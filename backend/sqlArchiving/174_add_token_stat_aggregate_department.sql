-- 174_add_token_stat_aggregate_department.sql
-- Preserve the department snapshot used by configurable Token statistics.
-- MySQL 8 / GoldenDB compatible.
-- Projection 和统计指标由管理员后续在界面中配置。

ALTER TABLE token_stat_aggregates
    ADD COLUMN department VARCHAR(255) NULL AFTER upstream_model;

CREATE INDEX idx_token_stat_aggregate_department
    ON token_stat_aggregates (department);
