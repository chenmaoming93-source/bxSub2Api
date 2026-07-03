-- 158_token_quota_config_and_usage_split.sql
-- Separate daily limit configuration from daily usage recording
-- for all three quota scopes: model (global), user+model, group candidate.

-- ============================================================
-- 1. New config tables
-- ============================================================

CREATE TABLE IF NOT EXISTS model_token_daily_limit_configs (
    id              BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    model           VARCHAR(255) NOT NULL,
    daily_limit_tokens BIGINT DEFAULT NULL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY unique_model (model)
);

CREATE TABLE IF NOT EXISTS user_model_token_daily_limit_configs (
    id              BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id         BIGINT NOT NULL,
    model           VARCHAR(255) NOT NULL,
    daily_limit_tokens BIGINT DEFAULT NULL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY unique_user_model (user_id, model),
    INDEX idx_user_id (user_id),
    CONSTRAINT fk_user_model_config_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS group_candidate_token_daily_limit_configs (
    id              BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    group_id        BIGINT NOT NULL,
    route_alias     VARCHAR(255) NOT NULL,
    upstream_model  VARCHAR(255) NOT NULL,
    daily_limit_tokens BIGINT DEFAULT NULL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY unique_group_candidate (group_id, route_alias, upstream_model),
    INDEX idx_group_id (group_id),
    CONSTRAINT fk_group_candidate_config_group FOREIGN KEY (group_id) REFERENCES `groups`(id) ON DELETE CASCADE
);

-- ============================================================
-- 2. Drop daily_limit_tokens from existing usage tables
-- ============================================================

ALTER TABLE model_token_daily_usages
    DROP COLUMN daily_limit_tokens;

ALTER TABLE user_model_token_daily_usages
    DROP COLUMN daily_limit_tokens;

ALTER TABLE group_candidate_token_daily_usages
    DROP COLUMN daily_limit_tokens;
