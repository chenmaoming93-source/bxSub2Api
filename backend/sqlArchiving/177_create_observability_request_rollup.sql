-- Idempotent request observability rollups. Raw facts stay in usage_logs and
-- ops_error_logs; this table stores only recomputable aggregates.

CREATE TABLE IF NOT EXISTS `observability_request_bucket` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `window_seconds` INT NOT NULL,
  `bucket_start` DATETIME(6) NOT NULL,
  `dimension_hash` CHAR(64) NOT NULL,
  `user_id` BIGINT NOT NULL DEFAULT 0,
  `api_key_id` BIGINT NOT NULL DEFAULT 0,
  `group_id` BIGINT NOT NULL DEFAULT 0,
  `route_alias` VARCHAR(255) NULL,
  `account_id` BIGINT NOT NULL DEFAULT 0,
  `request_count` BIGINT NOT NULL DEFAULT 0,
  `success_count` BIGINT NOT NULL DEFAULT 0,
  `error_count` BIGINT NOT NULL DEFAULT 0,
  `sla_error_count` BIGINT NOT NULL DEFAULT 0,
  `business_limited_count` BIGINT NOT NULL DEFAULT 0,
  `legacy_inferred_count` BIGINT NOT NULL DEFAULT 0,
  `input_tokens` BIGINT NOT NULL DEFAULT 0,
  `output_tokens` BIGINT NOT NULL DEFAULT 0,
  `cache_creation_tokens` BIGINT NOT NULL DEFAULT 0,
  `cache_read_tokens` BIGINT NOT NULL DEFAULT 0,
  `total_tokens` BIGINT NOT NULL DEFAULT 0,
  `duration_sample_count` BIGINT NOT NULL DEFAULT 0,
  `sum_duration_ms` BIGINT NOT NULL DEFAULT 0,
  `min_duration_ms` BIGINT NOT NULL DEFAULT 0,
  `max_duration_ms` BIGINT NOT NULL DEFAULT 0,
  `duration_le_100` BIGINT NOT NULL DEFAULT 0,
  `duration_le_250` BIGINT NOT NULL DEFAULT 0,
  `duration_le_500` BIGINT NOT NULL DEFAULT 0,
  `duration_le_1000` BIGINT NOT NULL DEFAULT 0,
  `duration_le_2000` BIGINT NOT NULL DEFAULT 0,
  `duration_le_5000` BIGINT NOT NULL DEFAULT 0,
  `duration_le_10000` BIGINT NOT NULL DEFAULT 0,
  `duration_gt_10000` BIGINT NOT NULL DEFAULT 0,
  `ttft_sample_count` BIGINT NOT NULL DEFAULT 0,
  `sum_ttft_ms` BIGINT NOT NULL DEFAULT 0,
  `min_ttft_ms` BIGINT NOT NULL DEFAULT 0,
  `max_ttft_ms` BIGINT NOT NULL DEFAULT 0,
  `ttft_le_100` BIGINT NOT NULL DEFAULT 0,
  `ttft_le_250` BIGINT NOT NULL DEFAULT 0,
  `ttft_le_500` BIGINT NOT NULL DEFAULT 0,
  `ttft_le_1000` BIGINT NOT NULL DEFAULT 0,
  `ttft_le_2000` BIGINT NOT NULL DEFAULT 0,
  `ttft_le_5000` BIGINT NOT NULL DEFAULT 0,
  `ttft_le_10000` BIGINT NOT NULL DEFAULT 0,
  `ttft_gt_10000` BIGINT NOT NULL DEFAULT 0,
  `last_fact_at` DATETIME(6) NOT NULL,
  `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_obs_request_bucket` (`window_seconds`, `bucket_start`, `dimension_hash`),
  KEY `idx_obs_request_bucket_window_time` (`window_seconds`, `bucket_start`),
  KEY `idx_obs_request_bucket_group_route_time` (`group_id`, `window_seconds`, `route_alias`, `bucket_start`),
  KEY `idx_obs_request_bucket_user_time` (`user_id`, `window_seconds`, `bucket_start`),
  KEY `idx_obs_request_bucket_api_key_time` (`api_key_id`, `window_seconds`, `bucket_start`),
  KEY `idx_obs_request_bucket_account_time` (`account_id`, `window_seconds`, `bucket_start`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `observability_request_rollup_state` (
  `id` TINYINT NOT NULL,
  `last_source_at` DATETIME(6) NULL,
  `last_success_at` DATETIME(6) NULL,
  `last_success_bucket` DATETIME(6) NULL,
  `skipped_count` BIGINT NOT NULL DEFAULT 0,
  `delayed_count` BIGINT NOT NULL DEFAULT 0,
  `quality_status` VARCHAR(32) NOT NULL DEFAULT 'unknown',
  `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO `observability_request_rollup_state`
  (`id`, `quality_status`)
VALUES
  (1, 'unknown');
