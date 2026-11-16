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

type observabilityConcurrencyRepository struct {
	db *sql.DB
}

func NewObservabilityConcurrencyRepository(db *sql.DB) service.ConcurrencyHistoryRepository {
	if db == nil {
		return nil
	}
	return &observabilityConcurrencyRepository{db: db}
}

func (r *observabilityConcurrencyRepository) FlushConcurrencyHistory(ctx context.Context, minuteStart time.Time, rows []service.ConcurrencyHistoryRow) error {
	if r == nil || r.db == nil {
		return nil
	}
	minuteStart = minuteStart.UTC().Truncate(time.Minute)
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	loc := timezone.Location()
	minuteForDB := utcBucketForDB(minuteStart, loc)
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM observability_concurrency_bucket WHERE window_seconds = 60 AND bucket_start = ?",
		minuteForDB,
	); err != nil {
		return err
	}
	if len(rows) > 0 {
		const insert = `INSERT INTO observability_concurrency_bucket
(window_seconds, bucket_start, sample_kind, dimension_hash, account_id, group_id, route_alias,
 current_peak, current_sum, sample_count, effective_limit, waiting_peak, wait_sum_ms, wait_sample_count,
 peak_ratio_actual, peak_ratio_limit, peak_ratio_sample_at, first_sampled_at, last_sampled_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON DUPLICATE KEY UPDATE
 current_peak = VALUES(current_peak),
 current_sum = VALUES(current_sum),
 sample_count = VALUES(sample_count),
 effective_limit = VALUES(effective_limit),
 waiting_peak = VALUES(waiting_peak),
 wait_sum_ms = VALUES(wait_sum_ms),
 wait_sample_count = VALUES(wait_sample_count),
 peak_ratio_actual = VALUES(peak_ratio_actual),
 peak_ratio_limit = VALUES(peak_ratio_limit),
 peak_ratio_sample_at = VALUES(peak_ratio_sample_at),
 first_sampled_at = VALUES(first_sampled_at),
 last_sampled_at = VALUES(last_sampled_at),
 updated_at = CURRENT_TIMESTAMP(6)`
		stmt, err := tx.PrepareContext(ctx, insert)
		if err != nil {
			return err
		}
		for _, row := range rows {
			var alias any
			var peakSampleAt any
			if strings.TrimSpace(row.RouteAlias) != "" {
				alias = strings.TrimSpace(row.RouteAlias)
			}
			if !row.PeakRatioSampleAt.IsZero() {
				peakSampleAt = row.PeakRatioSampleAt.UTC()
			}
			if _, err := stmt.ExecContext(ctx,
				service.ObservabilityWindowMinute, minuteForDB, row.SampleKind, row.DimensionHash,
				row.AccountID, row.GroupID, alias, row.CurrentPeak, row.CurrentSum, row.SampleCount,
				row.EffectiveLimit, row.WaitingPeak, row.WaitSumMS, row.WaitSampleCount,
				row.PeakRatioActual, row.PeakRatioLimit, peakSampleAt,
				row.FirstSampledAt.UTC(), row.LastSampledAt.UTC(),
			); err != nil {
				_ = stmt.Close()
				return err
			}
		}
		if err := stmt.Close(); err != nil {
			return err
		}
	}
	for _, window := range []int{service.ObservabilityWindowFiveMinute, service.ObservabilityWindowHour} {
		if err := rollupConcurrencyWindow(ctx, tx, minuteStart, window); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO observability_concurrency_sampler_state
 (id, last_flush_at, last_minute_start, write_count, quality_status, updated_at)
VALUES (1, NOW(), ?, 1, 'healthy', NOW())
ON DUPLICATE KEY UPDATE
 last_flush_at = VALUES(last_flush_at),
 last_minute_start = VALUES(last_minute_start),
 write_count = write_count + 1,
 quality_status = VALUES(quality_status),
 updated_at = VALUES(updated_at)
`, minuteForDB)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func rollupConcurrencyWindow(ctx context.Context, tx *sql.Tx, minuteStart time.Time, window int) error {
	loc := timezone.Location()
	duration := time.Duration(window) * time.Second
	bucketStart := floorUTCTime(minuteStart, duration)
	bucketEnd := bucketStart.Add(duration)
	startForDB := utcBucketForDB(bucketStart, loc)
	endForDB := utcBucketForDB(bucketEnd, loc)
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM observability_concurrency_bucket WHERE window_seconds = ? AND bucket_start = ?",
		window, startForDB,
	); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO observability_concurrency_bucket
  (window_seconds, bucket_start, sample_kind, dimension_hash, account_id, group_id, route_alias,
   current_peak, current_sum, sample_count, effective_limit, waiting_peak, wait_sum_ms, wait_sample_count,
   peak_ratio_actual, peak_ratio_limit, peak_ratio_sample_at, first_sampled_at, last_sampled_at)
SELECT
  ?, ?, sample_kind, dimension_hash, account_id, group_id, route_alias,
  MAX(current_peak), SUM(current_sum), SUM(sample_count),
  MAX(effective_limit), MAX(waiting_peak), SUM(wait_sum_ms), SUM(wait_sample_count),
  CAST(SUBSTRING_INDEX(GROUP_CONCAT(peak_ratio_actual ORDER BY (peak_ratio_actual / NULLIF(peak_ratio_limit,0)) DESC, peak_ratio_sample_at DESC), ',', 1) AS SIGNED),
  CAST(SUBSTRING_INDEX(GROUP_CONCAT(peak_ratio_limit ORDER BY (peak_ratio_actual / NULLIF(peak_ratio_limit,0)) DESC, peak_ratio_sample_at DESC), ',', 1) AS SIGNED),
  STR_TO_DATE(SUBSTRING_INDEX(GROUP_CONCAT(DATE_FORMAT(peak_ratio_sample_at, '%Y-%m-%d %H:%i:%s.%f') ORDER BY (peak_ratio_actual / NULLIF(peak_ratio_limit,0)) DESC, peak_ratio_sample_at DESC), ',', 1), '%Y-%m-%d %H:%i:%s.%f'),
  MIN(first_sampled_at), MAX(last_sampled_at)
FROM observability_concurrency_bucket
WHERE window_seconds = 60 AND bucket_start >= ? AND bucket_start < ?
GROUP BY sample_kind, dimension_hash, account_id, group_id, route_alias
ON DUPLICATE KEY UPDATE
  current_peak = VALUES(current_peak),
  current_sum = VALUES(current_sum),
  sample_count = VALUES(sample_count),
  effective_limit = VALUES(effective_limit),
  waiting_peak = VALUES(waiting_peak),
  wait_sum_ms = VALUES(wait_sum_ms),
  wait_sample_count = VALUES(wait_sample_count),
  peak_ratio_actual = VALUES(peak_ratio_actual),
  peak_ratio_limit = VALUES(peak_ratio_limit),
  peak_ratio_sample_at = VALUES(peak_ratio_sample_at),
  first_sampled_at = VALUES(first_sampled_at),
  last_sampled_at = VALUES(last_sampled_at),
  updated_at = CURRENT_TIMESTAMP(6)
`, window, startForDB, startForDB, endForDB)
	return err
}

func (r *observabilityConcurrencyRepository) GetConcurrencySamplerState(ctx context.Context) (service.ConcurrencySamplerState, error) {
	var state service.ConcurrencySamplerState
	var lastSampleAt, lastFlushAt, lastMinuteStart sql.NullTime
	err := r.db.QueryRowContext(ctx, `
SELECT last_sample_at, last_flush_at, last_minute_start, samples_accumulated, write_count, failure_count, quality_status
FROM observability_concurrency_sampler_state WHERE id = 1`).Scan(
		&lastSampleAt, &lastFlushAt, &lastMinuteStart,
		&state.SamplesAccumulated, &state.WriteCount, &state.FailureCount, &state.QualityStatus,
	)
	if err == sql.ErrNoRows {
		return service.ConcurrencySamplerState{QualityStatus: "unavailable"}, nil
	}
	if err != nil {
		return state, err
	}
	state.LastSampleAt = nullTimePtr(lastSampleAt)
	state.LastFlushAt = nullTimePtr(lastFlushAt)
	state.LastMinuteStart = nullBucketTimePtr(lastMinuteStart)
	return state, nil
}

func (r *observabilityConcurrencyRepository) RecordConcurrencySample(ctx context.Context, sampledAt time.Time, accumulated int64, failed bool) error {
	status := "healthy"
	delta := int64(0)
	if failed {
		status = "failed"
		delta = 1
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO observability_concurrency_sampler_state
 (id, last_sample_at, samples_accumulated, failure_count, quality_status, updated_at)
VALUES (1, ?, ?, ?, ?, NOW())
ON DUPLICATE KEY UPDATE
 last_sample_at = VALUES(last_sample_at),
 samples_accumulated = ?,
 failure_count = failure_count + ?,
 quality_status = VALUES(quality_status),
 updated_at = VALUES(updated_at)
`, sampledAt.UTC(), accumulated, delta, status, accumulated, delta)
	return err
}

func (r *observabilityConcurrencyRepository) QueryConcurrencyHistory(ctx context.Context, filter service.ConcurrencyHistoryFilter) ([]service.ConcurrencyHistoryRow, error) {
	query, args := concurrencyHistoryQuery(filter, "")
	return r.queryConcurrencyHistoryRows(ctx, query, args...)
}

// QueryConcurrencyHeatmapHistory bounds an unfiltered heatmap to the busiest
// dimensions before reading their full time series. Without this fan-in, a
// 1000-account day would transfer hundreds of thousands of unused cells.
func (r *observabilityConcurrencyRepository) QueryConcurrencyHeatmapHistory(
	ctx context.Context,
	filter service.ConcurrencyHistoryFilter,
	groupBy string,
	limit int,
) ([]service.ConcurrencyHistoryRow, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	dimensionColumn := "account_id"
	sampleKind := service.ConcurrencySampleKindAccount
	switch groupBy {
	case "group_id":
		dimensionColumn = "group_id"
		sampleKind = service.ConcurrencySampleKindRoute
	case "route":
		dimensionColumn = "route_alias"
		sampleKind = service.ConcurrencySampleKindRoute
	}
	query, args := concurrencyHistoryQuery(filter, dimensionColumn)
	whereIndex := strings.Index(query, "WHERE ")
	if whereIndex < 0 {
		return nil, fmt.Errorf("build concurrency heatmap query: WHERE clause missing")
	}
	outerWhere := query[whereIndex+len("WHERE "):]
	if orderIndex := strings.LastIndex(outerWhere, "\nORDER BY"); orderIndex >= 0 {
		outerWhere = outerWhere[:orderIndex]
	}
	topQuery := `
SELECT ` + dimensionColumn + `
FROM observability_concurrency_bucket
WHERE ` + outerWhere + `
  AND sample_kind = ?
  AND ` + dimensionColumn + ` IS NOT NULL
  AND ` + dimensionColumn + ` <> ''
GROUP BY ` + dimensionColumn + `
ORDER BY MAX(COALESCE(
  peak_ratio_actual / NULLIF(peak_ratio_limit, 0),
  current_peak / NULLIF(effective_limit, 0),
  0
)) DESC, ` + dimensionColumn + `
LIMIT ?`
	topArgs := append(append([]any{}, args...), sampleKind, limit)
	rows, err := r.db.QueryContext(ctx, topQuery, topArgs...)
	if err != nil {
		return nil, err
	}
	dimensionValues := make([]string, 0, limit)
	for rows.Next() {
		var value sql.NullString
		if err := rows.Scan(&value); err != nil {
			_ = rows.Close()
			return nil, err
		}
		if value.Valid && value.String != "" {
			dimensionValues = append(dimensionValues, value.String)
		}
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(dimensionValues) == 0 {
		return []service.ConcurrencyHistoryRow{}, nil
	}
	placeholders := make([]string, len(dimensionValues))
	inArgs := make([]any, 0, len(dimensionValues))
	for index, value := range dimensionValues {
		placeholders[index] = "?"
		inArgs = append(inArgs, value)
	}
	query = `SELECT window_seconds, bucket_start, sample_kind, dimension_hash, account_id, group_id, COALESCE(route_alias, ''),
       current_peak, current_sum, sample_count, effective_limit, waiting_peak,
       peak_ratio_actual, peak_ratio_limit, peak_ratio_sample_at,
       wait_sum_ms, wait_sample_count, first_sampled_at, last_sampled_at
FROM observability_concurrency_bucket
WHERE ` + outerWhere + `
  AND sample_kind = ?
  AND ` + dimensionColumn + ` IN (` + strings.Join(placeholders, ",") + `)
ORDER BY bucket_start, sample_kind, account_id, group_id, route_alias`
	args = append(args, sampleKind)
	args = append(args, inArgs...)
	return r.queryConcurrencyHistoryRows(ctx, query, args...)
}

func concurrencyHistoryQuery(filter service.ConcurrencyHistoryFilter, sampledDimension string) (string, []any) {
	duration := time.Duration(filter.WindowSeconds) * time.Second
	start := utcBucketForDB(floorUTCTime(filter.Start, duration), timezone.Location())
	end := utcBucketForDB(ceilUTCTime(filter.End, duration), timezone.Location())
	conditions := []string{"window_seconds = ?", "bucket_start >= ?", "bucket_start < ?"}
	args := []any{filter.WindowSeconds, start, end}
	if filter.AccountID > 0 {
		conditions = append(conditions, "account_id = ?")
		args = append(args, filter.AccountID)
	}
	if filter.GroupID > 0 {
		conditions = append(conditions, "group_id = ?")
		args = append(args, filter.GroupID)
	}
	if strings.TrimSpace(filter.RouteAlias) != "" {
		conditions = append(conditions, "route_alias = ?")
		args = append(args, strings.TrimSpace(filter.RouteAlias))
	}
	query := `
SELECT window_seconds, bucket_start, sample_kind, dimension_hash, account_id, group_id, COALESCE(route_alias, ''),
       current_peak, current_sum, sample_count, effective_limit, waiting_peak,
       peak_ratio_actual, peak_ratio_limit, peak_ratio_sample_at,
       wait_sum_ms, wait_sample_count, first_sampled_at, last_sampled_at
FROM observability_concurrency_bucket
WHERE ` + strings.Join(conditions, " AND ") + `
ORDER BY bucket_start, sample_kind, account_id, group_id, route_alias`
	return query, args
}

func (r *observabilityConcurrencyRepository) queryConcurrencyHistoryRows(
	ctx context.Context,
	query string,
	args ...any,
) ([]service.ConcurrencyHistoryRow, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.ConcurrencyHistoryRow, 0)
	for rows.Next() {
		var item service.ConcurrencyHistoryRow
		var peakSampleAt sql.NullTime
		if err := rows.Scan(
			&item.WindowSeconds, &item.BucketStart, &item.SampleKind, &item.DimensionHash,
			&item.AccountID, &item.GroupID, &item.RouteAlias,
			&item.CurrentPeak, &item.CurrentSum, &item.SampleCount, &item.EffectiveLimit, &item.WaitingPeak,
			&item.PeakRatioActual, &item.PeakRatioLimit, &peakSampleAt,
			&item.WaitSumMS, &item.WaitSampleCount,
			&item.FirstSampledAt, &item.LastSampledAt,
		); err != nil {
			return nil, err
		}
		item.BucketStart = utcBucketFromDB(item.BucketStart)
		item.FirstSampledAt = item.FirstSampledAt.UTC()
		item.LastSampledAt = item.LastSampledAt.UTC()
		if peakSampleAt.Valid {
			item.PeakRatioSampleAt = peakSampleAt.Time.UTC()
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *observabilityConcurrencyRepository) CleanupConcurrencyHistory(ctx context.Context, now time.Time) error {
	cutoffs := []struct {
		window int
		cutoff time.Time
	}{
		{service.ObservabilityWindowMinute, now.UTC().Add(-24 * time.Hour)},
		{service.ObservabilityWindowFiveMinute, now.UTC().Add(-7 * 24 * time.Hour)},
		{service.ObservabilityWindowHour, now.UTC().Add(-7 * 24 * time.Hour)},
	}
	for _, item := range cutoffs {
		if _, err := r.db.ExecContext(ctx,
			"DELETE FROM observability_concurrency_bucket WHERE window_seconds = ? AND bucket_start < ?",
			item.window, utcBucketForDB(item.cutoff, timezone.Location()),
		); err != nil {
			return err
		}
	}
	return nil
}

func validateConcurrencyWindow(window int) error {
	switch window {
	case service.ObservabilityWindowMinute, service.ObservabilityWindowFiveMinute, service.ObservabilityWindowHour:
		return nil
	default:
		return fmt.Errorf("unsupported concurrency window %d", window)
	}
}
