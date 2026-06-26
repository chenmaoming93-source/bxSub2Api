-- 086_channel_platform_pricing.sql
-- 渠道按平台维度：model_pricing 加 platform 列，model_mapping 改为嵌套格式。

-- 1. channel_model_pricing 加 platform 列
ALTER TABLE channel_model_pricing
    ADD COLUMN platform VARCHAR(50) NOT NULL DEFAULT 'anthropic';

CREATE INDEX idx_channel_model_pricing_platform
    ON channel_model_pricing (platform);

-- 2. model_mapping JSON 数据回填
-- GoldenDB/MySQL 模式：历史扁平 model_mapping JSON 的迁移由离线 PostgreSQL-to-GoldenDB
-- 数据迁移脚本处理。
