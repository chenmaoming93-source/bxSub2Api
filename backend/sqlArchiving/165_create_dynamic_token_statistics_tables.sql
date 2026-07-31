-- 可配置多维 Token 统计与限额的独立表。
-- MySQL 8 / GoldenDB（MySQL 兼容模式）；可重复执行建表语句。

CREATE TABLE IF NOT EXISTS token_stat_projections (
    id BIGINT NOT NULL AUTO_INCREMENT,
    name VARCHAR(128) NOT NULL,
    dimension_codes JSON NOT NULL,
    dimension_signature VARCHAR(512) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    config_version BIGINT UNSIGNED NOT NULL DEFAULT 0,
    published_at DATETIME(6) NULL,
    enabled_at DATETIME(6) NULL,
    disabled_at DATETIME(6) NULL,
    created_by BIGINT UNSIGNED NOT NULL,
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_token_stat_projection_signature (dimension_signature),
    KEY idx_token_stat_projection_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS token_stat_projection_metrics (
    id BIGINT NOT NULL AUTO_INCREMENT,
    projection_id BIGINT NOT NULL,
    metric_code VARCHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    enabled_at DATETIME(6) NULL,
    disabled_at DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_token_stat_projection_metric (projection_id, metric_code),
    KEY idx_token_stat_metric_status (metric_code, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS token_stat_aggregates (
    id BIGINT NOT NULL AUTO_INCREMENT,
    period_type VARCHAR(1) NOT NULL,
    period_start DATETIME(6) NOT NULL,
    period_end DATETIME(6) NOT NULL,
    projection_id BIGINT NOT NULL,
    dimension_hash BINARY(16) NOT NULL,
    dimension_values JSON NOT NULL,
    metric_code VARCHAR(64) NOT NULL,
    metric_value BIGINT NOT NULL DEFAULT 0,
    source_version BIGINT NOT NULL DEFAULT 0,
    user_id BIGINT NULL,
    api_key_id BIGINT NULL,
    group_id BIGINT NULL,
    route_alias VARCHAR(255) NULL,
    account_id BIGINT NULL,
    upstream_model VARCHAR(255) NULL,
    last_synced_at DATETIME(6) NOT NULL,
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_token_stat_aggregate_identity
        (period_type, period_start, projection_id, dimension_hash, metric_code),
    KEY idx_token_stat_aggregate_query (projection_id, metric_code, period_type, period_start),
    KEY idx_token_stat_aggregate_user (user_id),
    KEY idx_token_stat_aggregate_api_key (api_key_id),
    KEY idx_token_stat_aggregate_group (group_id),
    KEY idx_token_stat_aggregate_account (account_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS token_stat_quota_rules (
    id BIGINT NOT NULL AUTO_INCREMENT,
    name VARCHAR(128) NOT NULL,
    projection_id BIGINT NOT NULL,
    dimension_hash BINARY(16) NOT NULL,
    dimension_values JSON NOT NULL,
    metric_code VARCHAR(64) NOT NULL,
    period_type VARCHAR(1) NOT NULL,
    limit_value BIGINT NOT NULL,
    enforcement_mode VARCHAR(20) NOT NULL DEFAULT 'REJECT',
    status VARCHAR(20) NOT NULL DEFAULT 'DISABLED',
    effective_from DATETIME(6) NULL,
    effective_until DATETIME(6) NULL,
    created_by BIGINT UNSIGNED NOT NULL,
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_token_stat_quota_lookup (projection_id, metric_code, period_type, status),
    KEY idx_token_stat_quota_dimension (dimension_hash)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS token_stat_period_states (
    id BIGINT NOT NULL AUTO_INCREMENT,
    period_type VARCHAR(1) NOT NULL,
    period_start DATETIME(6) NOT NULL,
    period_end DATETIME(6) NOT NULL,
    state VARCHAR(20) NOT NULL DEFAULT 'OPEN',
    final_sync_version BIGINT NOT NULL DEFAULT 0,
    closed_at DATETIME(6) NULL,
    persisted_at DATETIME(6) NULL,
    deleted_at DATETIME(6) NULL,
    last_error TEXT NULL,
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_token_stat_period (period_type, period_start),
    KEY idx_token_stat_period_state_end (state, period_end)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
