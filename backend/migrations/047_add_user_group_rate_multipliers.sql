-- User-specific group rate multipliers.
CREATE TABLE IF NOT EXISTS user_group_rate_multipliers (
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id        BIGINT NOT NULL REFERENCES `groups`(id) ON DELETE CASCADE,
    rate_multiplier DECIMAL(10,4) NOT NULL,
    created_at      DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at      DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (user_id, group_id)
);

CREATE INDEX idx_user_group_rate_multipliers_group_id
    ON user_group_rate_multipliers(group_id);