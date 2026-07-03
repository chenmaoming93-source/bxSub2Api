-- Per-group route-candidate daily token quotas and usage.
CREATE TABLE IF NOT EXISTS group_candidate_token_daily_usages (
    id                 BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    group_id           BIGINT NOT NULL,
    route_alias        VARCHAR(255) NOT NULL,
    upstream_model     VARCHAR(255) NOT NULL,
    usage_date         DATE NOT NULL,
    used_tokens        BIGINT NOT NULL DEFAULT 0,
    daily_limit_tokens BIGINT,
    created_at         DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at         DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    CONSTRAINT group_candidate_token_daily_usages_group_fk FOREIGN KEY (group_id) REFERENCES `groups`(id) ON DELETE CASCADE,
    CONSTRAINT group_candidate_token_daily_usages_used_nonnegative CHECK (used_tokens >= 0),
    CONSTRAINT group_candidate_token_daily_usages_limit_nonnegative CHECK (daily_limit_tokens IS NULL OR daily_limit_tokens >= 0),
    CONSTRAINT group_candidate_token_daily_usages_key_uq UNIQUE (group_id, route_alias, upstream_model, usage_date),
    INDEX group_candidate_token_daily_usages_group_idx (group_id)
);
