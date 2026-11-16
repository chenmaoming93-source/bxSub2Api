-- Add capacity-card metrics to recomputable observability rollups.
-- Existing rows default to zero; no raw request or credential payload is copied.

SET @sub2api_add_obs_rate_limited_count = (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `observability_request_bucket` ADD COLUMN `rate_limited_count` BIGINT NOT NULL DEFAULT 0',
    'DO 0'
  )
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'observability_request_bucket'
    AND column_name = 'rate_limited_count'
);
PREPARE sub2api_add_obs_rate_limited_count_stmt FROM @sub2api_add_obs_rate_limited_count;
EXECUTE sub2api_add_obs_rate_limited_count_stmt;
DEALLOCATE PREPARE sub2api_add_obs_rate_limited_count_stmt;

SET @sub2api_add_obs_wait_sum_ms = (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `observability_concurrency_bucket` ADD COLUMN `wait_sum_ms` BIGINT NOT NULL DEFAULT 0',
    'DO 0'
  )
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'observability_concurrency_bucket'
    AND column_name = 'wait_sum_ms'
);
PREPARE sub2api_add_obs_wait_sum_ms_stmt FROM @sub2api_add_obs_wait_sum_ms;
EXECUTE sub2api_add_obs_wait_sum_ms_stmt;
DEALLOCATE PREPARE sub2api_add_obs_wait_sum_ms_stmt;

SET @sub2api_add_obs_wait_sample_count = (
  SELECT IF(
    COUNT(*) = 0,
    'ALTER TABLE `observability_concurrency_bucket` ADD COLUMN `wait_sample_count` BIGINT NOT NULL DEFAULT 0',
    'DO 0'
  )
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'observability_concurrency_bucket'
    AND column_name = 'wait_sample_count'
);
PREPARE sub2api_add_obs_wait_sample_count_stmt FROM @sub2api_add_obs_wait_sample_count;
EXECUTE sub2api_add_obs_wait_sample_count_stmt;
DEALLOCATE PREPARE sub2api_add_obs_wait_sample_count_stmt;
