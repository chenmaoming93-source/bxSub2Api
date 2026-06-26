-- Ops Monitoring (vNext): squashed migration (030)
--
-- This repository originally planned Ops vNext as migrations 030-036:
--   030 drop legacy ops tables
--   031 core schema
--   032 pre-aggregation tables
--   033 indexes + optional extensions
--   034 add avg/max to preagg
--   035 add notify_email to alert rules
--   036 seed default alert rules
--
-- Since these migrations have NOT been applied to any environment yet, we squash them
-- into a single 030 migration for easier review and a cleaner migration history.
--
-- Notes:
-- - This is intentionally destructive for ops_* data (error logs / metrics / alerts).
-- - It is idempotent (DROP/CREATE/ALTER IF EXISTS/IF NOT EXISTS), but will wipe ops_* data if re-run.

-- =====================================================================
-- 030_ops_drop_legacy_ops_tables.sql
-- =====================================================================


-- Legacy pre-aggregation tables (from 026 and/or previous branches)
DROP TABLE IF EXISTS ops_metrics_daily CASCADE;
DROP TABLE IF EXISTS ops_metrics_hourly CASCADE;

-- Core ops tables that may exist in some deployments / branches
DROP TABLE IF EXISTS ops_system_metrics CASCADE;
DROP TABLE IF EXISTS ops_error_logs CASCADE;
DROP TABLE IF EXISTS ops_alert_events CASCADE;
DROP TABLE IF EXISTS ops_alert_rules CASCADE;
DROP TABLE IF EXISTS ops_job_heartbeats CASCADE;
DROP TABLE IF EXISTS ops_retry_attempts CASCADE;

-- Optional legacy tables (best-effort cleanup)
DROP TABLE IF EXISTS ops_scheduled_reports CASCADE;
DROP TABLE IF EXISTS ops_group_availability_configs CASCADE;
DROP TABLE IF EXISTS ops_group_availability_events CASCADE;

-- Optional legacy views/indexes
DROP VIEW IF EXISTS ops_latest_metrics CASCADE;

-- =====================================================================
-- 031_ops_core_schema.sql
-- =====================================================================

-- Ops Monitoring (vNext): core schema (errors / retries / metrics / jobs / alerts)
--
-- Design goals:
-- - Support global filtering (time/platform/group) across all ops modules.
-- - Persist enough context for two retry modes (client retry / pinned upstream retry).
-- - Make ops background jobs observable via job heartbeats.
-- - Keep schema stable and indexes targeted (high-write tables).
--
-- Notes:
-- - This migration is idempotent.
-- - ops_* tables intentionally avoid strict foreign keys to reduce write amplification/locks.


-- ============================================
-- 1) ops_error_logs: error log details (high-write)
-- ============================================

CREATE TABLE IF NOT EXISTS ops_error_logs (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,

    -- Correlation / identities
    request_id VARCHAR(64),
    client_request_id VARCHAR(64),
    user_id BIGINT,
    api_key_id BIGINT,
    account_id BIGINT,
    group_id BIGINT,
    client_ip inet,

    -- Dimensions for global filtering
    platform VARCHAR(32),

    -- Request metadata
    model VARCHAR(100),
    request_path VARCHAR(256),
    stream BOOLEAN NOT NULL DEFAULT false,
    user_agent TEXT,

    -- Core error classification
    error_phase VARCHAR(32) NOT NULL,
    error_type VARCHAR(64) NOT NULL,
    severity VARCHAR(8) NOT NULL DEFAULT 'P2',
    status_code INT,

    -- vNext metric semantics
    is_business_limited BOOLEAN NOT NULL DEFAULT false,

    -- Error details (sanitized/truncated at ingest time)
    error_message TEXT,
    error_body TEXT,

    -- Provider/upstream details (optional; useful for trends & account health)
    error_source VARCHAR(64),
    error_owner VARCHAR(32),
    account_status VARCHAR(50),
    upstream_status_code INT,
    upstream_error_message TEXT,
    upstream_error_detail TEXT,
    provider_error_code VARCHAR(64),
    provider_error_type VARCHAR(64),
    network_error_type VARCHAR(50),
    retry_after_seconds INT,

    -- Timings (ms) - optional
    duration_ms INT,
    time_to_first_token_ms BIGINT,
    auth_latency_ms BIGINT,
    routing_latency_ms BIGINT,
    upstream_latency_ms BIGINT,
    response_latency_ms BIGINT,

    -- Retry context (only stored for error requests)
    request_body JSON,
    request_headers JSON,
    request_body_truncated BOOLEAN NOT NULL DEFAULT false,
    request_body_bytes INT,

    -- Retryability flags (best-effort classification)
    is_retryable BOOLEAN NOT NULL DEFAULT false,
    retry_count INT NOT NULL DEFAULT 0,

    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
);

-- ============================================
-- 2) ops_retry_attempts: audit log for retries
-- ============================================

CREATE TABLE IF NOT EXISTS ops_retry_attempts (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,

    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),

    requested_by_user_id BIGINT,
    source_error_id BIGINT,

    -- client|upstream
    mode VARCHAR(16) NOT NULL,
    pinned_account_id BIGINT,

    -- queued|running|succeeded|failed
    status VARCHAR(16) NOT NULL DEFAULT 'queued',
    started_at DATETIME(6),
    finished_at DATETIME(6),
    duration_ms BIGINT,

    -- Optional result correlation
    result_request_id VARCHAR(64),
    result_error_id BIGINT,
    result_usage_request_id VARCHAR(64),

    error_message TEXT
);

-- ============================================
-- 3) ops_system_metrics: system + request window snapshots
-- ============================================

CREATE TABLE IF NOT EXISTS ops_system_metrics (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,

    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    window_minutes INT NOT NULL DEFAULT 1,

    -- Optional dimensions (only if collector chooses to write per-dimension snapshots)
    platform VARCHAR(32),
    group_id BIGINT,

    -- Core counts
    success_count BIGINT NOT NULL DEFAULT 0,
    error_count_total BIGINT NOT NULL DEFAULT 0,
    business_limited_count BIGINT NOT NULL DEFAULT 0,
    error_count_sla BIGINT NOT NULL DEFAULT 0,

    upstream_error_count_excl_429_529 BIGINT NOT NULL DEFAULT 0,
    upstream_429_count BIGINT NOT NULL DEFAULT 0,
    upstream_529_count BIGINT NOT NULL DEFAULT 0,

    token_consumed BIGINT NOT NULL DEFAULT 0,

    -- Rates
    qps DOUBLE PRECISION,
    tps DOUBLE PRECISION,

    -- Duration percentiles (ms) - success requests
    duration_p50_ms INT,
    duration_p90_ms INT,
    duration_p95_ms INT,
    duration_p99_ms INT,
    duration_avg_ms DOUBLE PRECISION,
    duration_max_ms INT,

    -- TTFT percentiles (ms) - success requests (streaming)
    ttft_p50_ms INT,
    ttft_p90_ms INT,
    ttft_p95_ms INT,
    ttft_p99_ms INT,
    ttft_avg_ms DOUBLE PRECISION,
    ttft_max_ms INT,

    -- System resources
    cpu_usage_percent DOUBLE PRECISION,
    memory_used_mb BIGINT,
    memory_total_mb BIGINT,
    memory_usage_percent DOUBLE PRECISION,

    -- Dependency health (best-effort)
    db_ok BOOLEAN,
    redis_ok BOOLEAN,

    -- DB pool & runtime
    db_conn_active INT,
    db_conn_idle INT,
    db_conn_waiting INT,
    goroutine_count INT,

    -- Queue / concurrency
    concurrency_queue_depth INT
);

-- ============================================
-- 4) ops_job_heartbeats: background jobs health
-- ============================================

CREATE TABLE IF NOT EXISTS ops_job_heartbeats (
    job_name VARCHAR(64) PRIMARY KEY,

    last_run_at DATETIME(6),
    last_success_at DATETIME(6),
    last_error_at DATETIME(6),
    last_error TEXT,
    last_duration_ms BIGINT,

    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
);

-- ============================================
-- 5) ops_alert_rules / ops_alert_events
-- ============================================

CREATE TABLE IF NOT EXISTS ops_alert_rules (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,

    name VARCHAR(128) NOT NULL,
    description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT true,

    severity VARCHAR(16) NOT NULL DEFAULT 'warning',

    -- Metric definition
    -- Metric definition
    metric_type VARCHAR(64) NOT NULL,
    operator VARCHAR(8) NOT NULL,
    threshold DOUBLE PRECISION NOT NULL,

    window_minutes INT NOT NULL DEFAULT 5,
    sustained_minutes INT NOT NULL DEFAULT 5,
    cooldown_minutes INT NOT NULL DEFAULT 10,

    -- Optional scoping: platform/group filters etc.
    filters JSON,

    last_triggered_at DATETIME(6),
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ops_alert_rules_name_unique
    ON ops_alert_rules (name);

CREATE INDEX IF NOT EXISTS idx_ops_alert_rules_enabled
    ON ops_alert_rules (enabled);

CREATE TABLE IF NOT EXISTS ops_alert_events (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,

    rule_id BIGINT,
    severity VARCHAR(16) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'firing',

    title VARCHAR(200),
    description TEXT,

    metric_value DOUBLE PRECISION,
    threshold_value DOUBLE PRECISION,
    dimensions JSON,

    fired_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    resolved_at DATETIME(6),

    email_sent BOOLEAN NOT NULL DEFAULT false,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
);

CREATE INDEX IF NOT EXISTS idx_ops_alert_events_rule_status
    ON ops_alert_events (rule_id, status);

CREATE INDEX IF NOT EXISTS idx_ops_alert_events_fired_at
    ON ops_alert_events (fired_at DESC);

-- =====================================================================
-- 032_ops_preaggregation_tables.sql
-- =====================================================================

-- Ops Monitoring (vNext): pre-aggregation tables
--
-- Purpose:
-- - Provide stable query performance for 1鈥?4h windows (and beyond), avoiding expensive
--   percentile_cont scans on raw logs for every dashboard refresh.
-- - Support global filter dimensions: overall / platform / group.
--
-- Design note:
-- - We keep a single table with nullable platform/group_id, and enforce uniqueness via a
--   COALESCE-based unique index (because UNIQUE with NULLs allows duplicates in Postgres).


-- ============================================
-- 1) ops_metrics_hourly
-- ============================================

CREATE TABLE IF NOT EXISTS ops_metrics_hourly (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,

    bucket_start DATETIME(6) NOT NULL,
    platform VARCHAR(32),
    group_id BIGINT,

    success_count BIGINT NOT NULL DEFAULT 0,
    error_count_total BIGINT NOT NULL DEFAULT 0,
    business_limited_count BIGINT NOT NULL DEFAULT 0,
    error_count_sla BIGINT NOT NULL DEFAULT 0,

    upstream_error_count_excl_429_529 BIGINT NOT NULL DEFAULT 0,
    upstream_429_count BIGINT NOT NULL DEFAULT 0,
    upstream_529_count BIGINT NOT NULL DEFAULT 0,

    token_consumed BIGINT NOT NULL DEFAULT 0,

    -- Duration percentiles (ms)
    duration_p50_ms INT,
    duration_p90_ms INT,
    duration_p95_ms INT,
    duration_p99_ms INT,

    -- TTFT percentiles (ms)
    ttft_p50_ms INT,
    ttft_p90_ms INT,
    ttft_p95_ms INT,
    ttft_p99_ms INT,

    computed_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
);

-- Uniqueness across three 鈥渄imension modes鈥?(overall / platform / group).
-- Postgres UNIQUE treats NULLs as distinct, so we enforce uniqueness via COALESCE.
CREATE UNIQUE INDEX IF NOT EXISTS idx_ops_metrics_hourly_unique_dim
    ON ops_metrics_hourly (
        bucket_start,
        COALESCE(platform, ''),
        COALESCE(group_id, 0)
    );

CREATE INDEX IF NOT EXISTS idx_ops_metrics_hourly_bucket
    ON ops_metrics_hourly (bucket_start DESC);

CREATE INDEX IF NOT EXISTS idx_ops_metrics_hourly_platform_bucket
    ON ops_metrics_hourly (platform, bucket_start DESC);

CREATE INDEX IF NOT EXISTS idx_ops_metrics_hourly_group_bucket
    ON ops_metrics_hourly (group_id, bucket_start DESC);

-- ============================================
-- 2) ops_metrics_daily (optional; for longer windows)
-- ============================================

CREATE TABLE IF NOT EXISTS ops_metrics_daily (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,

    bucket_date DATE NOT NULL,
    platform VARCHAR(32),
    group_id BIGINT,

    success_count BIGINT NOT NULL DEFAULT 0,
    error_count_total BIGINT NOT NULL DEFAULT 0,
    business_limited_count BIGINT NOT NULL DEFAULT 0,
    error_count_sla BIGINT NOT NULL DEFAULT 0,

    upstream_error_count_excl_429_529 BIGINT NOT NULL DEFAULT 0,
    upstream_429_count BIGINT NOT NULL DEFAULT 0,
    upstream_529_count BIGINT NOT NULL DEFAULT 0,

    token_consumed BIGINT NOT NULL DEFAULT 0,

    duration_p50_ms INT,
    duration_p90_ms INT,
    duration_p95_ms INT,
    duration_p99_ms INT,

    ttft_p50_ms INT,
    ttft_p90_ms INT,
    ttft_p95_ms INT,
    ttft_p99_ms INT,

    computed_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ops_metrics_daily_unique_dim
    ON ops_metrics_daily (
        bucket_date,
        COALESCE(platform, ''),
        COALESCE(group_id, 0)
    );

CREATE INDEX IF NOT EXISTS idx_ops_metrics_daily_bucket
    ON ops_metrics_daily (bucket_date DESC);

CREATE INDEX IF NOT EXISTS idx_ops_metrics_daily_platform_bucket
    ON ops_metrics_daily (platform, bucket_date DESC);

CREATE INDEX IF NOT EXISTS idx_ops_metrics_daily_group_bucket
    ON ops_metrics_daily (group_id, bucket_date DESC);

-- =====================================================================
-- 033_ops_indexes_and_extensions.sql
-- =====================================================================

-- Ops Monitoring (vNext): indexes and optional extensions
--
-- This migration intentionally keeps "optional" objects (like PostgreSQL trigram) best-effort,
-- so environments without extension privileges won't fail the whole migration chain.


-- ============================================
-- 1) Core btree indexes (always safe)
-- ============================================

-- ops_error_logs
CREATE INDEX IF NOT EXISTS idx_ops_error_logs_created_at
    ON ops_error_logs (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ops_error_logs_platform_time
    ON ops_error_logs (platform, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ops_error_logs_group_time
    ON ops_error_logs (group_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ops_error_logs_account_time
    ON ops_error_logs (account_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ops_error_logs_status_time
    ON ops_error_logs (status_code, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ops_error_logs_phase_time
    ON ops_error_logs (error_phase, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ops_error_logs_type_time
    ON ops_error_logs (error_type, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ops_error_logs_request_id
    ON ops_error_logs (request_id);

CREATE INDEX IF NOT EXISTS idx_ops_error_logs_client_request_id
    ON ops_error_logs (client_request_id);

-- ops_system_metrics
CREATE INDEX IF NOT EXISTS idx_ops_system_metrics_created_at
    ON ops_system_metrics (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ops_system_metrics_window_time
    ON ops_system_metrics (window_minutes, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ops_system_metrics_platform_time
    ON ops_system_metrics (platform, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ops_system_metrics_group_time
    ON ops_system_metrics (group_id, created_at DESC);

-- ops_retry_attempts
CREATE INDEX IF NOT EXISTS idx_ops_retry_attempts_created_at
    ON ops_retry_attempts (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ops_retry_attempts_source_error
    ON ops_retry_attempts (source_error_id, created_at DESC);

-- Prevent concurrent retries for the same ops_error_logs row (race-free, multi-instance safe).
CREATE UNIQUE INDEX IF NOT EXISTS idx_ops_retry_attempts_unique_active
    ON ops_retry_attempts (source_error_id);

-- ============================================
-- 2) Optional fuzzy-search indexes
-- ============================================
-- PostgreSQL PostgreSQL trigram/GIN indexes are not available in GoldenDB/MySQL mode.

-- =====================================================================
-- 034_ops_preaggregation_add_avg_max.sql
-- =====================================================================

-- Ops Monitoring (vNext): extend pre-aggregation tables with avg/max latency fields
--
-- Why:
-- - The dashboard overview returns avg/max for duration/TTFT.
-- - Hourly/daily pre-aggregation tables originally stored only p50/p90/p95/p99, which makes
--   it impossible to answer avg/max in preagg mode without falling back to raw scans.
--
-- This migration is idempotent and safe to run multiple times.
--
-- NOTE: We keep the existing p50/p90/p95/p99 columns as-is; these are still used for
--       approximate long-window summaries.


-- Hourly table
ALTER TABLE ops_metrics_hourly
    ADD COLUMN IF NOT EXISTS duration_avg_ms DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS duration_max_ms INT,
    ADD COLUMN IF NOT EXISTS ttft_avg_ms DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS ttft_max_ms INT;

-- Daily table
ALTER TABLE ops_metrics_daily
    ADD COLUMN IF NOT EXISTS duration_avg_ms DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS duration_max_ms INT,
    ADD COLUMN IF NOT EXISTS ttft_avg_ms DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS ttft_max_ms INT;

-- =====================================================================
-- 035_ops_alert_rules_notify_email.sql
-- =====================================================================

-- Ops Monitoring (vNext): alert rule notify settings
--
-- Adds notify_email flag to ops_alert_rules to keep UI parity with the backup Ops dashboard.
-- Migration is idempotent.


ALTER TABLE ops_alert_rules
    ADD COLUMN IF NOT EXISTS notify_email BOOLEAN NOT NULL DEFAULT true;

-- =====================================================================
-- 036_ops_seed_default_alert_rules.sql
-- =====================================================================

-- Ops Monitoring (vNext): seed default alert rules (idempotent)

INSERT IGNORE INTO ops_alert_rules (
    name, description, enabled, metric_type, operator, threshold,
    window_minutes, sustained_minutes, severity, notify_email, cooldown_minutes,
    created_at, updated_at
) VALUES
    ('high_error_rate', 'Error rate exceeds 5 percent for 5 minutes.', true, 'error_rate', '>', 5.0, 5, 5, 'P1', true, 20, NOW(), NOW()),
    ('low_success_rate', 'Success rate is below 95 percent for 5 minutes.', true, 'success_rate', '<', 95.0, 5, 5, 'P0', true, 15, NOW(), NOW()),
    ('p99_latency_high', 'P99 latency exceeds 3000 ms for 10 minutes.', true, 'p99_latency_ms', '>', 3000.0, 5, 10, 'P2', true, 30, NOW(), NOW()),
    ('p95_latency_high', 'P95 latency exceeds 2000 ms for 10 minutes.', true, 'p95_latency_ms', '>', 2000.0, 5, 10, 'P2', true, 30, NOW(), NOW()),
    ('cpu_usage_high', 'CPU usage exceeds 85 percent for 10 minutes.', true, 'cpu_usage_percent', '>', 85.0, 5, 10, 'P2', true, 30, NOW(), NOW()),
    ('memory_usage_high', 'Memory usage exceeds 90 percent for 10 minutes.', true, 'memory_usage_percent', '>', 90.0, 5, 10, 'P1', true, 20, NOW(), NOW()),
    ('concurrency_queue_buildup', 'Concurrency queue depth exceeds 100 for 5 minutes.', true, 'concurrency_queue_depth', '>', 100.0, 5, 5, 'P1', true, 20, NOW(), NOW()),
    ('error_rate_critical', 'Error rate exceeds 20 percent for 1 minute.', true, 'error_rate', '>', 20.0, 1, 1, 'P0', true, 15, NOW(), NOW());

-- Ops Monitoring vNext: add Redis pool stats fields to system metrics snapshots.
-- This migration is intentionally idempotent.

ALTER TABLE ops_system_metrics
  ADD COLUMN IF NOT EXISTS redis_conn_total INT,
  ADD COLUMN IF NOT EXISTS redis_conn_idle INT;
