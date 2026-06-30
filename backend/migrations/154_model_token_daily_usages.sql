-- Global upstream-model daily token quotas and usage.
-- NULL/0 daily_limit_tokens means unlimited; positive values are enforced limits.
CREATE TABLE IF NOT EXISTS model_token_daily_usages (
    id                 BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    model              VARCHAR(255) NOT NULL,
    usage_date         DATE NOT NULL,
    used_tokens        BIGINT NOT NULL DEFAULT 0,
    daily_limit_tokens BIGINT,
    created_at         DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at         DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT model_token_daily_usages_used_nonnegative CHECK (used_tokens >= 0),
    CONSTRAINT model_token_daily_usages_limit_nonnegative CHECK (daily_limit_tokens IS NULL OR daily_limit_tokens >= 0),
    CONSTRAINT model_token_daily_usages_model_date_uq UNIQUE (model, usage_date)
);
