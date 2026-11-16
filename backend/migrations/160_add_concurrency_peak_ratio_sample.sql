-- Preserve the exact sample that produced the highest concurrency-pressure ratio.

SET @sub2api_add_obs_peak_ratio_actual = (
  SELECT IF(COUNT(*) = 0,
    'ALTER TABLE `observability_concurrency_bucket` ADD COLUMN `peak_ratio_actual` INT NOT NULL DEFAULT 0',
    'DO 0')
  FROM information_schema.columns
  WHERE table_schema = DATABASE() AND table_name = 'observability_concurrency_bucket' AND column_name = 'peak_ratio_actual'
);
PREPARE sub2api_add_obs_peak_ratio_actual_stmt FROM @sub2api_add_obs_peak_ratio_actual;
EXECUTE sub2api_add_obs_peak_ratio_actual_stmt;
DEALLOCATE PREPARE sub2api_add_obs_peak_ratio_actual_stmt;

SET @sub2api_add_obs_peak_ratio_limit = (
  SELECT IF(COUNT(*) = 0,
    'ALTER TABLE `observability_concurrency_bucket` ADD COLUMN `peak_ratio_limit` INT NOT NULL DEFAULT 0',
    'DO 0')
  FROM information_schema.columns
  WHERE table_schema = DATABASE() AND table_name = 'observability_concurrency_bucket' AND column_name = 'peak_ratio_limit'
);
PREPARE sub2api_add_obs_peak_ratio_limit_stmt FROM @sub2api_add_obs_peak_ratio_limit;
EXECUTE sub2api_add_obs_peak_ratio_limit_stmt;
DEALLOCATE PREPARE sub2api_add_obs_peak_ratio_limit_stmt;

SET @sub2api_add_obs_peak_ratio_sample_at = (
  SELECT IF(COUNT(*) = 0,
    'ALTER TABLE `observability_concurrency_bucket` ADD COLUMN `peak_ratio_sample_at` DATETIME(6) NULL',
    'DO 0')
  FROM information_schema.columns
  WHERE table_schema = DATABASE() AND table_name = 'observability_concurrency_bucket' AND column_name = 'peak_ratio_sample_at'
);
PREPARE sub2api_add_obs_peak_ratio_sample_at_stmt FROM @sub2api_add_obs_peak_ratio_sample_at;
EXECUTE sub2api_add_obs_peak_ratio_sample_at_stmt;
DEALLOCATE PREPARE sub2api_add_obs_peak_ratio_sample_at_stmt;
