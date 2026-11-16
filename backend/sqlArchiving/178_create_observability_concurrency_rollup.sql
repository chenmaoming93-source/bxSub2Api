-- Minute-level concurrency history. High-frequency Redis snapshots are
-- accumulated in memory and never persisted individually.

CREATE TABLE IF NOT EXISTS `observability_concurrency_bucket` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `window_seconds` INT NOT NULL,
  `bucket_start` DATETIME(6) NOT NULL,
  `sample_kind` VARCHAR(16) NOT NULL,
  `dimension_hash` CHAR(64) NOT NULL,
  `account_id` BIGINT NOT NULL,
  `group_id` BIGINT NOT NULL DEFAULT 0,
  `route_alias` VARCHAR(255) NULL,
  `current_peak` INT NOT NULL DEFAULT 0,
  `current_sum` BIGINT NOT NULL DEFAULT 0,
  `sample_count` BIGINT NOT NULL DEFAULT 0,
  `effective_limit` INT NOT NULL DEFAULT 0,
  `waiting_peak` INT NOT NULL DEFAULT 0,
  `first_sampled_at` DATETIME(6) NOT NULL,
  `last_sampled_at` DATETIME(6) NOT NULL,
  `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_obs_concurrency_bucket` (`window_seconds`, `bucket_start`, `dimension_hash`),
  KEY `idx_obs_concurrency_bucket_window_time` (`window_seconds`, `bucket_start`),
  KEY `idx_obs_concurrency_bucket_group_route_time` (`group_id`, `window_seconds`, `route_alias`, `bucket_start`),
  KEY `idx_obs_concurrency_bucket_account_time` (`account_id`, `window_seconds`, `bucket_start`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `observability_concurrency_sampler_state` (
  `id` TINYINT NOT NULL,
  `last_sample_at` DATETIME(6) NULL,
  `last_flush_at` DATETIME(6) NULL,
  `last_minute_start` DATETIME(6) NULL,
  `samples_accumulated` BIGINT NOT NULL DEFAULT 0,
  `write_count` BIGINT NOT NULL DEFAULT 0,
  `failure_count` BIGINT NOT NULL DEFAULT 0,
  `quality_status` VARCHAR(32) NOT NULL DEFAULT 'unknown',
  `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO `observability_concurrency_sampler_state`
  (`id`, `quality_status`)
VALUES
  (1, 'unknown')
ON DUPLICATE KEY UPDATE `id` = VALUES(`id`);
