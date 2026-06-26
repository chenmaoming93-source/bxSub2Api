-- Sub2API subscription migration
-- Add subscription groups and user subscriptions

-- 1. Extend groups table with subscription columns
ALTER TABLE `groups` ADD COLUMN platform VARCHAR(50) NOT NULL DEFAULT 'anthropic';
ALTER TABLE `groups` ADD COLUMN subscription_type VARCHAR(20) NOT NULL DEFAULT 'standard';
ALTER TABLE `groups` ADD COLUMN daily_limit_usd DECIMAL(20, 8) DEFAULT NULL;
ALTER TABLE `groups` ADD COLUMN weekly_limit_usd DECIMAL(20, 8) DEFAULT NULL;
ALTER TABLE `groups` ADD COLUMN monthly_limit_usd DECIMAL(20, 8) DEFAULT NULL;
ALTER TABLE `groups` ADD COLUMN default_validity_days INT NOT NULL DEFAULT 30;

CREATE INDEX idx_groups_platform ON `groups`(platform);
CREATE INDEX idx_groups_subscription_type ON `groups`(subscription_type);

-- 2. Create user_subscriptions table
CREATE TABLE IF NOT EXISTS user_subscriptions (
    id                      BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id                 BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id                BIGINT NOT NULL REFERENCES `groups`(id) ON DELETE CASCADE,

    starts_at               DATETIME(6) NOT NULL,
    expires_at              DATETIME(6) NOT NULL,
    status                  VARCHAR(20) NOT NULL DEFAULT 'active',

    daily_window_start      DATETIME(6),
    weekly_window_start     DATETIME(6),
    monthly_window_start    DATETIME(6),

    daily_usage_usd         DECIMAL(20, 10) NOT NULL DEFAULT 0,
    weekly_usage_usd        DECIMAL(20, 10) NOT NULL DEFAULT 0,
    monthly_usage_usd       DECIMAL(20, 10) NOT NULL DEFAULT 0,

    assigned_by             BIGINT REFERENCES users(id) ON DELETE SET NULL,
    assigned_at             DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    notes                   TEXT,

    created_at              DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at              DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),

    UNIQUE(user_id, group_id)
);

CREATE INDEX idx_user_subscriptions_user_id ON user_subscriptions(user_id);
CREATE INDEX idx_user_subscriptions_group_id ON user_subscriptions(group_id);
CREATE INDEX idx_user_subscriptions_status ON user_subscriptions(status);
CREATE INDEX idx_user_subscriptions_expires_at ON user_subscriptions(expires_at);
CREATE INDEX idx_user_subscriptions_assigned_by ON user_subscriptions(assigned_by);

-- 3. Extend usage_logs table
ALTER TABLE usage_logs ADD COLUMN group_id BIGINT REFERENCES `groups`(id) ON DELETE SET NULL;
ALTER TABLE usage_logs ADD COLUMN subscription_id BIGINT REFERENCES user_subscriptions(id) ON DELETE SET NULL;
ALTER TABLE usage_logs ADD COLUMN rate_multiplier DECIMAL(10, 4) NOT NULL DEFAULT 1;
ALTER TABLE usage_logs ADD COLUMN first_token_ms INT;

CREATE INDEX idx_usage_logs_group_id ON usage_logs(group_id);
CREATE INDEX idx_usage_logs_subscription_id ON usage_logs(subscription_id);
CREATE INDEX idx_usage_logs_sub_created ON usage_logs(subscription_id, created_at);
