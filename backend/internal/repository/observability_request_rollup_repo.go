package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type observabilityRequestRollupRepository struct {
	db *sql.DB
}

func NewObservabilityRequestRollupRepository(db *sql.DB) service.ObservabilityRequestRollupRepository {
	if db == nil {
		return nil
	}
	return &observabilityRequestRollupRepository{db: db}
}

func (r *observabilityRequestRollupRepository) AggregateRequestRange(ctx context.Context, start, end time.Time) error {
	if r == nil || r.db == nil {
		return nil
	}
	start = start.UTC()
	end = end.UTC()
	if !end.After(start) {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	maxFactAt, err := loadObservabilityMaxFactAt(ctx, tx, start, end)
	if err != nil {
		return err
	}
	for _, window := range []int{
		service.ObservabilityWindowMinute,
		service.ObservabilityWindowFiveMinute,
		service.ObservabilityWindowHour,
	} {
		if err := replaceObservabilityWindow(ctx, tx, start, end, window); err != nil {
			return err
		}
	}
	if err := updateObservabilityRollupSuccess(ctx, tx, end, maxFactAt); err != nil {
		return err
	}
	return tx.Commit()
}

func loadObservabilityMaxFactAt(ctx context.Context, tx *sql.Tx, start, end time.Time) (*time.Time, error) {
	loc := timezone.Location()
	query := `
SELECT MAX(f.created_at)
FROM (
	SELECT created_at FROM usage_logs WHERE created_at >= ? AND created_at < ?
	UNION ALL
	SELECT created_at FROM ops_error_logs
	WHERE created_at >= ? AND created_at < ?
	  AND (
	    COALESCE(status_code, 0) >= 400
	    OR (status_code IS NULL AND error_phase IN ('request','auth','routing','upstream','network','internal'))
	  )
) f`
	var value sql.NullTime
	if err := tx.QueryRowContext(ctx, query, start.In(loc), end.In(loc), start.In(loc), end.In(loc)).Scan(&value); err != nil {
		return nil, err
	}
	if !value.Valid {
		return nil, nil
	}
	value.Time = value.Time.UTC()
	return &value.Time, nil
}

func replaceObservabilityWindow(ctx context.Context, tx *sql.Tx, start, end time.Time, window int) error {
	bucketStart := floorUTCTime(start, time.Duration(window)*time.Second)
	bucketEnd := ceilUTCTime(end, time.Duration(window)*time.Second)
	aggregateStart, aggregateEnd := alignObservabilityRollupRange(start, end, window)
	loc := timezone.Location()
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM observability_request_bucket WHERE window_seconds = ? AND bucket_start >= ? AND bucket_start < ?",
		window, utcBucketForDB(bucketStart, loc), utcBucketForDB(bucketEnd, loc),
	); err != nil {
		return err
	}

	query, args := buildObservabilityRollupInsertSQL(window, aggregateStart, aggregateEnd)
	_, err := tx.ExecContext(ctx, query, args...)
	return err
}

// alignObservabilityRollupRange expands an incremental range to the full
// buckets being replaced. Without this, recomputing a trailing 2-minute
// lookback would delete a whole 5-minute/hour bucket and silently drop facts
// that occurred earlier in that same bucket.
func alignObservabilityRollupRange(start, end time.Time, window int) (time.Time, time.Time) {
	width := time.Duration(window) * time.Second
	return floorUTCTime(start, width), ceilUTCTime(end, width)
}

func buildObservabilityRollupInsertSQL(window int, start, end time.Time) (string, []any) {
	loc := timezone.Location()
	startLocal := start.In(loc)
	endLocal := end.In(loc)
	_, offsetSeconds := start.In(loc).Zone()
	offset := formatTimeZoneOffset(offsetSeconds)

	factAggregate := fmt.Sprintf(`
SELECT
  TIMESTAMPADD(
    SECOND,
    (TIMESTAMPDIFF(SECOND, '1970-01-01 00:00:00', CONVERT_TZ(f.created_at, ?, '+00:00')) DIV %d) * %d,
    '1970-01-01 00:00:00'
  ) AS bucket_start,
  SHA2(CONCAT_WS('#', f.user_id, f.api_key_id, f.group_id, f.route_alias, f.account_id), 256) AS dimension_hash,
  f.user_id,
  f.api_key_id,
  f.group_id,
  NULLIF(f.route_alias, '') AS route_alias,
  f.account_id,
  COUNT(*) AS request_count,
  SUM(CASE WHEN f.is_error = 0 THEN 1 ELSE 0 END) AS success_count,
  SUM(CASE WHEN f.is_error = 1 THEN 1 ELSE 0 END) AS error_count,
  SUM(CASE WHEN f.is_error = 1 AND f.sla_error = 1 THEN 1 ELSE 0 END) AS sla_error_count,
  SUM(CASE WHEN f.is_error = 1 AND f.business_limited = 1 THEN 1 ELSE 0 END) AS business_limited_count,
  SUM(CASE WHEN f.is_error = 1 AND f.rate_limited = 1 THEN 1 ELSE 0 END) AS rate_limited_count,
  SUM(f.legacy_inferred) AS legacy_inferred_count,
  SUM(f.input_tokens) AS input_tokens,
  SUM(f.output_tokens) AS output_tokens,
  SUM(f.cache_creation_tokens) AS cache_creation_tokens,
  SUM(f.cache_read_tokens) AS cache_read_tokens,
  SUM(f.total_tokens) AS total_tokens,
  COUNT(NULLIF(f.duration_ms, 0)) AS duration_sample_count,
  COALESCE(SUM(NULLIF(f.duration_ms, 0)), 0) AS sum_duration_ms,
  COALESCE(MIN(NULLIF(f.duration_ms, 0)), 0) AS min_duration_ms,
  COALESCE(MAX(NULLIF(f.duration_ms, 0)), 0) AS max_duration_ms,
  SUM(CASE WHEN f.duration_ms > 0 AND f.duration_ms <= 100 THEN 1 ELSE 0 END) AS duration_le_100,
  SUM(CASE WHEN f.duration_ms > 100 AND f.duration_ms <= 250 THEN 1 ELSE 0 END) AS duration_le_250,
  SUM(CASE WHEN f.duration_ms > 250 AND f.duration_ms <= 500 THEN 1 ELSE 0 END) AS duration_le_500,
  SUM(CASE WHEN f.duration_ms > 500 AND f.duration_ms <= 1000 THEN 1 ELSE 0 END) AS duration_le_1000,
  SUM(CASE WHEN f.duration_ms > 1000 AND f.duration_ms <= 2000 THEN 1 ELSE 0 END) AS duration_le_2000,
  SUM(CASE WHEN f.duration_ms > 2000 AND f.duration_ms <= 5000 THEN 1 ELSE 0 END) AS duration_le_5000,
  SUM(CASE WHEN f.duration_ms > 5000 AND f.duration_ms <= 10000 THEN 1 ELSE 0 END) AS duration_le_10000,
  SUM(CASE WHEN f.duration_ms > 10000 THEN 1 ELSE 0 END) AS duration_gt_10000,
  COUNT(NULLIF(f.ttft_ms, 0)) AS ttft_sample_count,
  COALESCE(SUM(NULLIF(f.ttft_ms, 0)), 0) AS sum_ttft_ms,
  COALESCE(MIN(NULLIF(f.ttft_ms, 0)), 0) AS min_ttft_ms,
  COALESCE(MAX(NULLIF(f.ttft_ms, 0)), 0) AS max_ttft_ms,
  SUM(CASE WHEN f.ttft_ms > 0 AND f.ttft_ms <= 100 THEN 1 ELSE 0 END) AS ttft_le_100,
  SUM(CASE WHEN f.ttft_ms > 100 AND f.ttft_ms <= 250 THEN 1 ELSE 0 END) AS ttft_le_250,
  SUM(CASE WHEN f.ttft_ms > 250 AND f.ttft_ms <= 500 THEN 1 ELSE 0 END) AS ttft_le_500,
  SUM(CASE WHEN f.ttft_ms > 500 AND f.ttft_ms <= 1000 THEN 1 ELSE 0 END) AS ttft_le_1000,
  SUM(CASE WHEN f.ttft_ms > 1000 AND f.ttft_ms <= 2000 THEN 1 ELSE 0 END) AS ttft_le_2000,
  SUM(CASE WHEN f.ttft_ms > 2000 AND f.ttft_ms <= 5000 THEN 1 ELSE 0 END) AS ttft_le_5000,
  SUM(CASE WHEN f.ttft_ms > 5000 AND f.ttft_ms <= 10000 THEN 1 ELSE 0 END) AS ttft_le_10000,
  SUM(CASE WHEN f.ttft_ms > 10000 THEN 1 ELSE 0 END) AS ttft_gt_10000,
  MAX(f.created_at) AS last_fact_at
FROM (
  SELECT
    request_key,
    MAX(is_error) AS is_error,
    MAX(sla_error) AS sla_error,
    MAX(business_limited) AS business_limited,
    MAX(rate_limited) AS rate_limited,
    MAX(legacy_inferred) AS legacy_inferred,
    MAX(user_id) AS user_id,
    MAX(api_key_id) AS api_key_id,
    MAX(group_id) AS group_id,
    MAX(route_alias) AS route_alias,
    MAX(account_id) AS account_id,
    SUM(input_tokens) AS input_tokens,
    SUM(output_tokens) AS output_tokens,
    SUM(cache_creation_tokens) AS cache_creation_tokens,
    SUM(cache_read_tokens) AS cache_read_tokens,
    SUM(total_tokens) AS total_tokens,
    MAX(duration_ms) AS duration_ms,
    MAX(ttft_ms) AS ttft_ms,
    MAX(created_at) AS created_at
  FROM (
    SELECT
      COALESCE(NULLIF(TRIM(u.request_id), ''), CONCAT('usage:', u.id)) AS request_key,
      CASE WHEN u.request_type = 4 THEN 1 ELSE 0 END AS is_error,
      0 AS sla_error,
      0 AS business_limited,
      0 AS rate_limited,
      CASE
        WHEN u.route_alias IS NULL
          AND NULLIF(TRIM(u.requested_model), '') IS NOT NULL
          AND u.requested_model <> u.model
        THEN 1 ELSE 0
      END AS legacy_inferred,
      u.user_id,
      u.api_key_id,
      COALESCE(u.group_id, 0) AS group_id,
      COALESCE(
        NULLIF(TRIM(u.route_alias), ''),
        CASE
          WHEN u.requested_model IS NOT NULL AND u.requested_model <> u.model THEN u.requested_model
          ELSE ''
        END
      ) AS route_alias,
      u.account_id,
      u.input_tokens,
      u.output_tokens,
      u.cache_creation_tokens,
      u.cache_read_tokens,
      (u.input_tokens + u.output_tokens + u.cache_creation_tokens + u.cache_read_tokens) AS total_tokens,
      COALESCE(u.duration_ms, 0) AS duration_ms,
      COALESCE(u.first_token_ms, 0) AS ttft_ms,
      u.created_at
    FROM usage_logs u
    WHERE u.created_at >= ? AND u.created_at < ?
    UNION ALL
    SELECT
      COALESCE(
        NULLIF(TRIM(e.request_id), ''),
        NULLIF(TRIM(e.client_request_id), ''),
        CONCAT('ops:', e.id)
      ) AS request_key,
      1 AS is_error,
      CASE WHEN COALESCE(e.status_code, 0) = 529 THEN 1 ELSE 0 END AS sla_error,
      CASE WHEN e.is_business_limited THEN 1 ELSE 0 END AS business_limited,
      CASE WHEN COALESCE(e.status_code, 0) = 429 OR e.is_business_limited THEN 1 ELSE 0 END AS rate_limited,
      CASE
        WHEN e.route_alias IS NULL
          AND NULLIF(TRIM(e.requested_model), '') IS NOT NULL
          AND e.requested_model <> e.model
        THEN 1 ELSE 0
      END AS legacy_inferred,
      COALESCE(e.user_id, 0) AS user_id,
      COALESCE(e.api_key_id, 0) AS api_key_id,
      COALESCE(e.group_id, 0) AS group_id,
      COALESCE(
        NULLIF(TRIM(e.route_alias), ''),
        CASE
          WHEN e.requested_model IS NOT NULL AND e.requested_model <> e.model THEN e.requested_model
          ELSE ''
        END
      ) AS route_alias,
      COALESCE(e.account_id, 0) AS account_id,
      0, 0, 0, 0, 0,
      COALESCE(e.response_latency_ms, e.upstream_latency_ms, 0) AS duration_ms,
      COALESCE(e.time_to_first_token_ms, 0) AS ttft_ms,
      e.created_at
    FROM ops_error_logs e
    WHERE e.created_at >= ? AND e.created_at < ?
      AND (
        COALESCE(e.status_code, 0) >= 400
        OR (e.status_code IS NULL AND e.error_phase IN ('request','auth','routing','upstream','network','internal'))
      )
  ) raw
  GROUP BY request_key
) f
GROUP BY
  bucket_start, dimension_hash, f.user_id, f.api_key_id, f.group_id, f.route_alias, f.account_id`,
		window, window)

	insert := `INSERT INTO observability_request_bucket (
  window_seconds, bucket_start, dimension_hash, user_id, api_key_id, group_id, route_alias, account_id,
  request_count, success_count, error_count, sla_error_count, business_limited_count, legacy_inferred_count,
  rate_limited_count,
  input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens, total_tokens,
  duration_sample_count, sum_duration_ms, min_duration_ms, max_duration_ms,
  duration_le_100, duration_le_250, duration_le_500, duration_le_1000,
  duration_le_2000, duration_le_5000, duration_le_10000, duration_gt_10000,
  ttft_sample_count, sum_ttft_ms, min_ttft_ms, max_ttft_ms,
  ttft_le_100, ttft_le_250, ttft_le_500, ttft_le_1000,
  ttft_le_2000, ttft_le_5000, ttft_le_10000, ttft_gt_10000,
  last_fact_at
) `
	update := `
ON DUPLICATE KEY UPDATE
  user_id = VALUES(user_id),
  api_key_id = VALUES(api_key_id),
  group_id = VALUES(group_id),
  route_alias = VALUES(route_alias),
  account_id = VALUES(account_id),
  request_count = VALUES(request_count),
  success_count = VALUES(success_count),
  error_count = VALUES(error_count),
  sla_error_count = VALUES(sla_error_count),
  business_limited_count = VALUES(business_limited_count),
  rate_limited_count = VALUES(rate_limited_count),
  legacy_inferred_count = VALUES(legacy_inferred_count),
  input_tokens = VALUES(input_tokens),
  output_tokens = VALUES(output_tokens),
  cache_creation_tokens = VALUES(cache_creation_tokens),
  cache_read_tokens = VALUES(cache_read_tokens),
  total_tokens = VALUES(total_tokens),
  duration_sample_count = VALUES(duration_sample_count),
  sum_duration_ms = VALUES(sum_duration_ms),
  min_duration_ms = VALUES(min_duration_ms),
  max_duration_ms = VALUES(max_duration_ms),
  duration_le_100 = VALUES(duration_le_100),
  duration_le_250 = VALUES(duration_le_250),
  duration_le_500 = VALUES(duration_le_500),
  duration_le_1000 = VALUES(duration_le_1000),
  duration_le_2000 = VALUES(duration_le_2000),
  duration_le_5000 = VALUES(duration_le_5000),
  duration_le_10000 = VALUES(duration_le_10000),
  duration_gt_10000 = VALUES(duration_gt_10000),
  ttft_sample_count = VALUES(ttft_sample_count),
  sum_ttft_ms = VALUES(sum_ttft_ms),
  min_ttft_ms = VALUES(min_ttft_ms),
  max_ttft_ms = VALUES(max_ttft_ms),
  ttft_le_100 = VALUES(ttft_le_100),
  ttft_le_250 = VALUES(ttft_le_250),
  ttft_le_500 = VALUES(ttft_le_500),
  ttft_le_1000 = VALUES(ttft_le_1000),
  ttft_le_2000 = VALUES(ttft_le_2000),
  ttft_le_5000 = VALUES(ttft_le_5000),
  ttft_le_10000 = VALUES(ttft_le_10000),
  ttft_gt_10000 = VALUES(ttft_gt_10000),
  last_fact_at = VALUES(last_fact_at),
  updated_at = CURRENT_TIMESTAMP(6)`

	selectSQL := strings.Replace(factAggregate, "SELECT\n", "SELECT\n  ? AS window_seconds,\n", 1)
	args := []any{
		window,
		offset,
		startLocal,
		endLocal,
		startLocal,
		endLocal,
	}
	return insert + selectSQL + update, args
}

func updateObservabilityRollupSuccess(ctx context.Context, tx *sql.Tx, bucket time.Time, maxFactAt *time.Time) error {
	bucketForDB := utcBucketForDB(bucket, timezone.Location())
	_, err := tx.ExecContext(ctx, `
INSERT INTO observability_request_rollup_state
  (id, last_source_at, last_success_at, last_success_bucket, skipped_count, delayed_count, quality_status, updated_at)
VALUES (1, ?, NOW(), ?, 0, 0, 'healthy', NOW())
ON DUPLICATE KEY UPDATE
  last_source_at = COALESCE(VALUES(last_source_at), last_source_at),
  last_success_at = VALUES(last_success_at),
  last_success_bucket = VALUES(last_success_bucket),
  skipped_count = VALUES(skipped_count),
  delayed_count = VALUES(delayed_count),
  quality_status = VALUES(quality_status),
  updated_at = VALUES(updated_at)
`, maxFactAt, bucketForDB)
	return err
}

func (r *observabilityRequestRollupRepository) GetRequestRollupState(ctx context.Context) (service.ObservabilityRollupState, error) {
	var state service.ObservabilityRollupState
	var lastSourceAt, lastSuccessAt, lastSuccessBucket sql.NullTime
	err := r.db.QueryRowContext(ctx, `
SELECT last_source_at, last_success_at, last_success_bucket, skipped_count, delayed_count, quality_status
FROM observability_request_rollup_state WHERE id = 1`).Scan(
		&lastSourceAt, &lastSuccessAt, &lastSuccessBucket,
		&state.SkippedCount, &state.DelayedCount, &state.QualityStatus,
	)
	if err == sql.ErrNoRows {
		return service.ObservabilityRollupState{QualityStatus: "unknown"}, nil
	}
	if err != nil {
		return state, err
	}
	state.LastSourceAt = nullTimePtr(lastSourceAt)
	state.LastSuccessAt = nullTimePtr(lastSuccessAt)
	state.LastSuccessBucket = nullBucketTimePtr(lastSuccessBucket)
	return state, nil
}

func (r *observabilityRequestRollupRepository) RecordRequestRollupFailure(ctx context.Context, delayed bool) error {
	increment := 0
	if delayed {
		increment = 1
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO observability_request_rollup_state
  (id, skipped_count, delayed_count, quality_status, updated_at)
VALUES (1, 0, ?, 'failed', NOW())
ON DUPLICATE KEY UPDATE
  delayed_count = delayed_count + ?,
  quality_status = 'failed',
  updated_at = NOW()
`, increment, increment)
	return err
}

func (r *observabilityRequestRollupRepository) QueryRequestBucketTotals(ctx context.Context, start, end time.Time, windowSeconds int) ([]service.ObservabilityBucketTotal, error) {
	switch windowSeconds {
	case service.ObservabilityWindowMinute, service.ObservabilityWindowFiveMinute, service.ObservabilityWindowHour:
	default:
		return nil, fmt.Errorf("unsupported observability window %d", windowSeconds)
	}
	window := time.Duration(windowSeconds) * time.Second
	queryStart := utcBucketForDB(floorUTCTime(start, window), timezone.Location())
	queryEnd := utcBucketForDB(ceilUTCTime(end, window), timezone.Location())
	rows, err := r.db.QueryContext(ctx, `
SELECT
  bucket_start,
  SUM(request_count), SUM(success_count), SUM(error_count), SUM(sla_error_count), SUM(business_limited_count), SUM(rate_limited_count), SUM(legacy_inferred_count),
  SUM(input_tokens), SUM(output_tokens), SUM(cache_creation_tokens), SUM(cache_read_tokens), SUM(total_tokens),
  SUM(duration_sample_count), SUM(sum_duration_ms), MIN(min_duration_ms), MAX(max_duration_ms),
  SUM(duration_le_100), SUM(duration_le_250), SUM(duration_le_500), SUM(duration_le_1000),
  SUM(duration_le_2000), SUM(duration_le_5000), SUM(duration_le_10000), SUM(duration_gt_10000),
  SUM(ttft_sample_count), SUM(sum_ttft_ms), MIN(min_ttft_ms), MAX(max_ttft_ms),
  SUM(ttft_le_100), SUM(ttft_le_250), SUM(ttft_le_500), SUM(ttft_le_1000),
  SUM(ttft_le_2000), SUM(ttft_le_5000), SUM(ttft_le_10000), SUM(ttft_gt_10000)
FROM observability_request_bucket
WHERE window_seconds = ? AND bucket_start >= ? AND bucket_start < ?
GROUP BY bucket_start
ORDER BY bucket_start`, windowSeconds, queryStart, queryEnd)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.ObservabilityBucketTotal, 0)
	for rows.Next() {
		var item service.ObservabilityBucketTotal
		if err := rows.Scan(
			&item.BucketStart,
			&item.RequestCount, &item.SuccessCount, &item.ErrorCount, &item.SLAErrorCount, &item.BusinessLimitedCount, &item.RateLimitedCount, &item.LegacyInferredCount,
			&item.InputTokens, &item.OutputTokens, &item.CacheCreationTokens, &item.CacheReadTokens, &item.TotalTokens,
			&item.Duration.SampleCount, &item.Duration.SumMS, &item.Duration.MinMS, &item.Duration.MaxMS,
			&item.Duration.Buckets[0], &item.Duration.Buckets[1], &item.Duration.Buckets[2], &item.Duration.Buckets[3],
			&item.Duration.Buckets[4], &item.Duration.Buckets[5], &item.Duration.Buckets[6], &item.Duration.Buckets[7],
			&item.TTFT.SampleCount, &item.TTFT.SumMS, &item.TTFT.MinMS, &item.TTFT.MaxMS,
			&item.TTFT.Buckets[0], &item.TTFT.Buckets[1], &item.TTFT.Buckets[2], &item.TTFT.Buckets[3],
			&item.TTFT.Buckets[4], &item.TTFT.Buckets[5], &item.TTFT.Buckets[6], &item.TTFT.Buckets[7],
		); err != nil {
			return nil, err
		}
		item.BucketStart = utcBucketFromDB(item.BucketStart)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *observabilityRequestRollupRepository) CleanupRequestRollups(ctx context.Context, now time.Time) error {
	cutoffs := []struct {
		window int
		cutoff time.Time
	}{
		{service.ObservabilityWindowMinute, now.UTC().Add(-48 * time.Hour)},
		{service.ObservabilityWindowFiveMinute, now.UTC().Add(-7 * 24 * time.Hour)},
		{service.ObservabilityWindowHour, now.UTC().Add(-90 * 24 * time.Hour)},
	}
	for _, item := range cutoffs {
		if _, err := r.db.ExecContext(ctx,
			"DELETE FROM observability_request_bucket WHERE window_seconds = ? AND bucket_start < ?",
			item.window, utcBucketForDB(item.cutoff, timezone.Location()),
		); err != nil {
			return err
		}
	}
	return nil
}

func floorUTCTime(value time.Time, window time.Duration) time.Time {
	return value.UTC().Truncate(window)
}

func ceilUTCTime(value time.Time, window time.Duration) time.Time {
	value = value.UTC()
	floored := value.Truncate(window)
	if value.After(floored) {
		return floored.Add(window)
	}
	return floored
}

func formatTimeZoneOffset(seconds int) string {
	sign := "+"
	if seconds < 0 {
		sign = "-"
		seconds = -seconds
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	return fmt.Sprintf("%s%02d:%02d", sign, hours, minutes)
}

// Bucket timestamps are UTC wall-clock DATETIME values. The MySQL driver is
// configured with the display timezone, so convert explicitly at the SQL
// boundary instead of relying on time.Time's attached Location.
func utcBucketForDB(value time.Time, loc *time.Location) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), value.Hour(), value.Minute(), value.Second(), value.Nanosecond(), loc)
}

func utcBucketFromDB(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), value.Hour(), value.Minute(), value.Second(), value.Nanosecond(), time.UTC)
}

func nullTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time.UTC()
	return &t
}

func nullBucketTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	t := utcBucketFromDB(value.Time)
	return &t
}
