package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"
)

type opsPreaggKey struct {
	bucket   time.Time
	platform string
	groupID  int64
}

type opsPreaggBucket struct {
	successCount                 int64
	ttftSampleCount              int64
	errorCountTotal              int64
	businessLimitedCount         int64
	errorCountSLA                int64
	upstreamErrorCountExcl429529 int64
	upstream429Count             int64
	upstream529Count             int64
	tokenConsumed                int64
	durationValues               []int
	durationSum                  int64
	ttftValues                   []int
	ttftSum                      int64
}

func (r *opsRepository) UpsertHourlyMetrics(ctx context.Context, startTime, endTime time.Time) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil ops repository")
	}
	if startTime.IsZero() || endTime.IsZero() || !endTime.After(startTime) {
		return nil
	}

	start := startTime.UTC()
	end := endTime.UTC()

	buckets := make(map[opsPreaggKey]*opsPreaggBucket)
	bucketFor := func(key opsPreaggKey) *opsPreaggBucket {
		item := buckets[key]
		if item == nil {
			item = &opsPreaggBucket{}
			buckets[key] = item
		}
		return item
	}
	keysFor := func(bucket time.Time, platform string, groupID sql.NullInt64) []opsPreaggKey {
		keys := []opsPreaggKey{{bucket: bucket}}
		if platform != "" {
			keys = append(keys, opsPreaggKey{bucket: bucket, platform: platform})
		}
		if groupID.Valid {
			keys = append(keys, opsPreaggKey{bucket: bucket, platform: platform, groupID: groupID.Int64})
		}
		return keys
	}

	usageRows, err := r.db.QueryContext(ctx, `
SELECT
  ul.created_at,
  COALESCE(g.platform, '') AS platform,
  ul.group_id,
  ul.duration_ms,
  ul.first_token_ms,
  COALESCE(ul.input_tokens, 0)
    + COALESCE(ul.output_tokens, 0)
    + COALESCE(ul.cache_creation_tokens, 0)
    + COALESCE(ul.cache_read_tokens, 0) AS tokens
FROM usage_logs ul
JOIN `+"`groups`"+` g ON g.id = ul.group_id
WHERE ul.created_at >= ? AND ul.created_at < ?`, start, end)
	if err != nil {
		return err
	}
	for usageRows.Next() {
		var createdAt time.Time
		var platform string
		var groupID sql.NullInt64
		var durationMS sql.NullInt64
		var firstTokenMS sql.NullInt64
		var tokens sql.NullInt64
		if err := usageRows.Scan(&createdAt, &platform, &groupID, &durationMS, &firstTokenMS, &tokens); err != nil {
			_ = usageRows.Close()
			return err
		}
		bucketStart := createdAt.UTC().Truncate(time.Hour)
		for _, key := range keysFor(bucketStart, platform, groupID) {
			item := bucketFor(key)
			item.successCount++
			if tokens.Valid {
				item.tokenConsumed += tokens.Int64
			}
			if durationMS.Valid {
				value := int(durationMS.Int64)
				item.durationValues = append(item.durationValues, value)
				item.durationSum += durationMS.Int64
			}
			if firstTokenMS.Valid {
				value := int(firstTokenMS.Int64)
				item.ttftValues = append(item.ttftValues, value)
				item.ttftSum += firstTokenMS.Int64
				item.ttftSampleCount++
			}
		}
	}
	if err := usageRows.Close(); err != nil {
		return err
	}
	if err := usageRows.Err(); err != nil {
		return err
	}

	errorRows, err := r.db.QueryContext(ctx, `
SELECT
  created_at,
  COALESCE(platform, 'unknown') AS platform,
  group_id,
  is_business_limited,
  error_owner,
  status_code,
  COALESCE(upstream_status_code, status_code, 0) AS effective_status_code
FROM ops_error_logs
WHERE created_at >= ? AND created_at < ?
  AND is_count_tokens = FALSE`, start, end)
	if err != nil {
		return err
	}
	for errorRows.Next() {
		var createdAt time.Time
		var platform string
		var groupID sql.NullInt64
		var isBusinessLimited bool
		var errorOwner sql.NullString
		var clientStatusCode sql.NullInt64
		var effectiveStatusCode sql.NullInt64
		if err := errorRows.Scan(&createdAt, &platform, &groupID, &isBusinessLimited, &errorOwner, &clientStatusCode, &effectiveStatusCode); err != nil {
			_ = errorRows.Close()
			return err
		}
		bucketStart := createdAt.UTC().Truncate(time.Hour)
		for _, key := range keysFor(bucketStart, platform, groupID) {
			item := bucketFor(key)
			clientStatus := int64(0)
			if clientStatusCode.Valid {
				clientStatus = clientStatusCode.Int64
			}
			effectiveStatus := int64(0)
			if effectiveStatusCode.Valid {
				effectiveStatus = effectiveStatusCode.Int64
			}
			if clientStatus >= 400 {
				item.errorCountTotal++
				if isBusinessLimited {
					item.businessLimitedCount++
				} else {
					item.errorCountSLA++
				}
			}
			if errorOwner.Valid && errorOwner.String == "provider" && !isBusinessLimited {
				switch effectiveStatus {
				case 429:
					item.upstream429Count++
				case 529:
					item.upstream529Count++
				default:
					item.upstreamErrorCountExcl429529++
				}
			}
		}
	}
	if err := errorRows.Close(); err != nil {
		return err
	}
	if err := errorRows.Err(); err != nil {
		return err
	}
	if len(buckets) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	upsert := `
INSERT INTO ops_metrics_hourly (
  bucket_start,
  platform,
  group_id,
  success_count,
  ttft_sample_count,
  error_count_total,
  business_limited_count,
  error_count_sla,
  upstream_error_count_excl_429_529,
  upstream_429_count,
  upstream_529_count,
  token_consumed,
  duration_p50_ms,
  duration_p90_ms,
  duration_p95_ms,
  duration_p99_ms,
  duration_avg_ms,
  duration_max_ms,
  ttft_p50_ms,
  ttft_p90_ms,
  ttft_p95_ms,
  ttft_p99_ms,
  ttft_avg_ms,
  ttft_max_ms,
  computed_at
) VALUES (
  ?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,NOW()
)
ON DUPLICATE KEY UPDATE
  success_count = VALUES(success_count),
  ttft_sample_count = VALUES(ttft_sample_count),
  error_count_total = VALUES(error_count_total),
  business_limited_count = VALUES(business_limited_count),
  error_count_sla = VALUES(error_count_sla),
  upstream_error_count_excl_429_529 = VALUES(upstream_error_count_excl_429_529),
  upstream_429_count = VALUES(upstream_429_count),
  upstream_529_count = VALUES(upstream_529_count),
  token_consumed = VALUES(token_consumed),
  duration_p50_ms = VALUES(duration_p50_ms),
  duration_p90_ms = VALUES(duration_p90_ms),
  duration_p95_ms = VALUES(duration_p95_ms),
  duration_p99_ms = VALUES(duration_p99_ms),
  duration_avg_ms = VALUES(duration_avg_ms),
  duration_max_ms = VALUES(duration_max_ms),
  ttft_p50_ms = VALUES(ttft_p50_ms),
  ttft_p90_ms = VALUES(ttft_p90_ms),
  ttft_p95_ms = VALUES(ttft_p95_ms),
  ttft_p99_ms = VALUES(ttft_p99_ms),
  ttft_avg_ms = VALUES(ttft_avg_ms),
  ttft_max_ms = VALUES(ttft_max_ms),
  computed_at = NOW()`

	keys := make([]opsPreaggKey, 0, len(buckets))
	for key := range buckets {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if !keys[i].bucket.Equal(keys[j].bucket) {
			return keys[i].bucket.Before(keys[j].bucket)
		}
		if keys[i].platform != keys[j].platform {
			return keys[i].platform < keys[j].platform
		}
		return keys[i].groupID < keys[j].groupID
	})

	for _, key := range keys {
		item := buckets[key]
		sort.Ints(item.durationValues)
		sort.Ints(item.ttftValues)
		durationPercentiles := opsPercentilesFromSortedInts(item.durationValues)
		ttftPercentiles := opsPercentilesFromSortedInts(item.ttftValues)
		var durationAvg *float64
		if len(item.durationValues) > 0 {
			v := float64(item.durationSum) / float64(len(item.durationValues))
			durationAvg = &v
		}
		var ttftAvg *float64
		if len(item.ttftValues) > 0 {
			v := float64(item.ttftSum) / float64(len(item.ttftValues))
			ttftAvg = &v
		}
		var platform *string
		if key.platform != "" {
			v := key.platform
			platform = &v
		}
		var groupID *int64
		if key.groupID != 0 {
			v := key.groupID
			groupID = &v
		}

		if _, err := tx.ExecContext(
			ctx,
			upsert,
			key.bucket,
			opsNullString(platform),
			opsNullInt64(groupID),
			item.successCount,
			item.ttftSampleCount,
			item.errorCountTotal,
			item.businessLimitedCount,
			item.errorCountSLA,
			item.upstreamErrorCountExcl429529,
			item.upstream429Count,
			item.upstream529Count,
			item.tokenConsumed,
			opsNullInt(durationPercentiles.P50),
			opsNullInt(durationPercentiles.P90),
			opsNullInt(durationPercentiles.P95),
			opsNullInt(durationPercentiles.P99),
			opsNullFloat64(durationAvg),
			opsNullInt(durationPercentiles.Max),
			opsNullInt(ttftPercentiles.P50),
			opsNullInt(ttftPercentiles.P90),
			opsNullInt(ttftPercentiles.P95),
			opsNullInt(ttftPercentiles.P99),
			opsNullFloat64(ttftAvg),
			opsNullInt(ttftPercentiles.Max),
		); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (r *opsRepository) UpsertDailyMetrics(ctx context.Context, startTime, endTime time.Time) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil ops repository")
	}
	if startTime.IsZero() || endTime.IsZero() || !endTime.After(startTime) {
		return nil
	}

	start := startTime.UTC()
	end := endTime.UTC()

	q := `
INSERT INTO ops_metrics_daily (
  bucket_date,
  platform,
  group_id,
  success_count,
  ttft_sample_count,
  error_count_total,
  business_limited_count,
  error_count_sla,
  upstream_error_count_excl_429_529,
  upstream_429_count,
  upstream_529_count,
  token_consumed,
  duration_p50_ms,
  duration_p90_ms,
  duration_p95_ms,
  duration_p99_ms,
  duration_avg_ms,
  duration_max_ms,
  ttft_p50_ms,
  ttft_p90_ms,
  ttft_p95_ms,
  ttft_p99_ms,
  ttft_avg_ms,
  ttft_max_ms,
  computed_at
)
SELECT
  DATE(bucket_start) AS bucket_date,
  platform,
  group_id,

  COALESCE(SUM(success_count), 0) AS success_count,
  COALESCE(SUM(ttft_sample_count), 0) AS ttft_sample_count,
  COALESCE(SUM(error_count_total), 0) AS error_count_total,
  COALESCE(SUM(business_limited_count), 0) AS business_limited_count,
  COALESCE(SUM(error_count_sla), 0) AS error_count_sla,
  COALESCE(SUM(upstream_error_count_excl_429_529), 0) AS upstream_error_count_excl_429_529,
  COALESCE(SUM(upstream_429_count), 0) AS upstream_429_count,
  COALESCE(SUM(upstream_529_count), 0) AS upstream_529_count,
  COALESCE(SUM(token_consumed), 0) AS token_consumed,

  -- Approximation: weighted average for p50/p90, max for p95/p99 (conservative tail).
  CAST(ROUND(SUM(CASE WHEN duration_p50_ms IS NOT NULL THEN duration_p50_ms * success_count ELSE 0 END)
    / NULLIF(SUM(CASE WHEN duration_p50_ms IS NOT NULL THEN success_count ELSE 0 END), 0)) AS SIGNED) AS duration_p50_ms,
  CAST(ROUND(SUM(CASE WHEN duration_p90_ms IS NOT NULL THEN duration_p90_ms * success_count ELSE 0 END)
    / NULLIF(SUM(CASE WHEN duration_p90_ms IS NOT NULL THEN success_count ELSE 0 END), 0)) AS SIGNED) AS duration_p90_ms,
  MAX(duration_p95_ms) AS duration_p95_ms,
  MAX(duration_p99_ms) AS duration_p99_ms,
  SUM(CASE WHEN duration_avg_ms IS NOT NULL THEN duration_avg_ms * success_count ELSE 0 END)
    / NULLIF(SUM(CASE WHEN duration_avg_ms IS NOT NULL THEN success_count ELSE 0 END), 0) AS duration_avg_ms,
  MAX(duration_max_ms) AS duration_max_ms,

  -- TTFT is weighted by ttft_sample_count (streaming rows only), NOT success_count,
  -- because first_token_ms is recorded only for streaming requests.
  CAST(ROUND(SUM(CASE WHEN ttft_p50_ms IS NOT NULL THEN ttft_p50_ms * ttft_sample_count ELSE 0 END)
    / NULLIF(SUM(CASE WHEN ttft_p50_ms IS NOT NULL THEN ttft_sample_count ELSE 0 END), 0)) AS SIGNED) AS ttft_p50_ms,
  CAST(ROUND(SUM(CASE WHEN ttft_p90_ms IS NOT NULL THEN ttft_p90_ms * ttft_sample_count ELSE 0 END)
    / NULLIF(SUM(CASE WHEN ttft_p90_ms IS NOT NULL THEN ttft_sample_count ELSE 0 END), 0)) AS SIGNED) AS ttft_p90_ms,
  MAX(ttft_p95_ms) AS ttft_p95_ms,
  MAX(ttft_p99_ms) AS ttft_p99_ms,
  SUM(CASE WHEN ttft_avg_ms IS NOT NULL THEN ttft_avg_ms * ttft_sample_count ELSE 0 END)
    / NULLIF(SUM(CASE WHEN ttft_avg_ms IS NOT NULL THEN ttft_sample_count ELSE 0 END), 0) AS ttft_avg_ms,
  MAX(ttft_max_ms) AS ttft_max_ms,

  NOW()
FROM ops_metrics_hourly
WHERE bucket_start >= ? AND bucket_start < ?
GROUP BY 1, 2, 3
ON DUPLICATE KEY UPDATE
  success_count = VALUES(success_count),
  ttft_sample_count = VALUES(ttft_sample_count),
  error_count_total = VALUES(error_count_total),
  business_limited_count = VALUES(business_limited_count),
  error_count_sla = VALUES(error_count_sla),
  upstream_error_count_excl_429_529 = VALUES(upstream_error_count_excl_429_529),
  upstream_429_count = VALUES(upstream_429_count),
  upstream_529_count = VALUES(upstream_529_count),
  token_consumed = VALUES(token_consumed),
  duration_p50_ms = VALUES(duration_p50_ms),
  duration_p90_ms = VALUES(duration_p90_ms),
  duration_p95_ms = VALUES(duration_p95_ms),
  duration_p99_ms = VALUES(duration_p99_ms),
  duration_avg_ms = VALUES(duration_avg_ms),
  duration_max_ms = VALUES(duration_max_ms),
  ttft_p50_ms = VALUES(ttft_p50_ms),
  ttft_p90_ms = VALUES(ttft_p90_ms),
  ttft_p95_ms = VALUES(ttft_p95_ms),
  ttft_p99_ms = VALUES(ttft_p99_ms),
  ttft_avg_ms = VALUES(ttft_avg_ms),
  ttft_max_ms = VALUES(ttft_max_ms),
  computed_at = NOW()
`

	_, err := r.db.ExecContext(ctx, q, start, end)
	return err
}

func (r *opsRepository) GetLatestHourlyBucketStart(ctx context.Context) (time.Time, bool, error) {
	if r == nil || r.db == nil {
		return time.Time{}, false, fmt.Errorf("nil ops repository")
	}

	var value sql.NullTime
	if err := r.db.QueryRowContext(ctx, `SELECT MAX(bucket_start) FROM ops_metrics_hourly`).Scan(&value); err != nil {
		return time.Time{}, false, err
	}
	if !value.Valid {
		return time.Time{}, false, nil
	}
	return value.Time.UTC(), true, nil
}

func (r *opsRepository) GetLatestDailyBucketDate(ctx context.Context) (time.Time, bool, error) {
	if r == nil || r.db == nil {
		return time.Time{}, false, fmt.Errorf("nil ops repository")
	}

	var value sql.NullTime
	if err := r.db.QueryRowContext(ctx, `SELECT MAX(bucket_date) FROM ops_metrics_daily`).Scan(&value); err != nil {
		return time.Time{}, false, err
	}
	if !value.Valid {
		return time.Time{}, false, nil
	}
	t := value.Time.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), true, nil
}
