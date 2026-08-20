-- Store daily time-window concurrency overrides for one model-route candidate.
-- The legacy group_model_route_accounts table remains the default-value source.
CREATE TABLE IF NOT EXISTS group_model_route_account_concurrency_schedules (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    group_id BIGINT NOT NULL,
    route_alias VARCHAR(255) NOT NULL,
    account_id BIGINT NOT NULL,
    start_minute SMALLINT UNSIGNED NOT NULL,
    end_minute SMALLINT UNSIGNED NOT NULL,
    max_concurrency INT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    UNIQUE KEY uq_route_account_schedule_window (
        group_id, route_alias, account_id, start_minute, end_minute
    ),
    KEY idx_route_account_schedule_candidate (
        group_id, route_alias, account_id, start_minute
    ),
    KEY idx_route_account_schedule_account (account_id),
    CONSTRAINT fk_route_account_schedule_group
        FOREIGN KEY (group_id) REFERENCES `groups` (id),
    CONSTRAINT fk_route_account_schedule_account
        FOREIGN KEY (account_id) REFERENCES accounts (id),
    CONSTRAINT chk_route_account_schedule_window
        CHECK (start_minute < end_minute AND end_minute <= 1440),
    CONSTRAINT chk_route_account_schedule_concurrency
        CHECK (max_concurrency IS NULL OR max_concurrency > 0)
);
