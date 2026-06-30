-- Per-user upstream-model daily token quotas and usage.
-- Rows are removed with their user, matching other user-owned quota data.
CREATE TABLE IF NOT EXISTS user_model_token_daily_usages (
    id                 BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id            BIGINT NOT NULL,
    model              VARCHAR(255) NOT NULL,
    usage_date         DATE NOT NULL,
    used_tokens        BIGINT NOT NULL DEFAULT 0,
    daily_limit_tokens BIGINT,
    created_at         DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at         DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT user_model_token_daily_usages_user_fk FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT user_model_token_daily_usages_used_nonnegative CHECK (used_tokens >= 0),
    CONSTRAINT user_model_token_daily_usages_limit_nonnegative CHECK (daily_limit_tokens IS NULL OR daily_limit_tokens >= 0),
    CONSTRAINT user_model_token_daily_usages_user_model_date_uq UNIQUE (user_id, model, usage_date),
    INDEX user_model_token_daily_usages_user_idx (user_id)
);
