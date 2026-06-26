-- 邀请返利流水补充订单关联和转余额快照。
-- 这些字段只用于审计展示；历史旧流水无法可靠反推的字段保持 NULL，避免把当前状态误展示为历史状态。
ALTER TABLE user_affiliate_ledger
    ADD COLUMN IF NOT EXISTS source_order_id BIGINT NULL REFERENCES payment_orders(id) ON DELETE SET NULL;

ALTER TABLE user_affiliate_ledger
    ADD COLUMN IF NOT EXISTS balance_after DECIMAL(20,8) NULL;

ALTER TABLE user_affiliate_ledger
    ADD COLUMN IF NOT EXISTS aff_quota_after DECIMAL(20,8) NULL;

ALTER TABLE user_affiliate_ledger
    ADD COLUMN IF NOT EXISTS aff_frozen_quota_after DECIMAL(20,8) NULL;

ALTER TABLE user_affiliate_ledger
    ADD COLUMN IF NOT EXISTS aff_history_quota_after DECIMAL(20,8) NULL;

CREATE INDEX IF NOT EXISTS idx_user_affiliate_ledger_source_order_id
    ON user_affiliate_ledger(source_order_id);

CREATE INDEX IF NOT EXISTS idx_user_affiliate_ledger_rebate_lookup
    ON user_affiliate_ledger(action, source_order_id, user_id, source_user_id, created_at);

-- 尽力回填 PR #2169 合并后、该迁移前已产生的返利流水。
-- GoldenDB/MySQL 模式：历史审计日志文本中的返利回填由离线 PostgreSQL-to-GoldenDB
-- 数据迁移脚本处理。offline PostgreSQL-to-GoldenDB data migration script.
