package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/service/observability"
)

type snapshotRollupTotals = service.SnapshotRollupTotals
type snapshotDimensionRow = service.SnapshotDimensionRow
type ObservabilitySnapshotRollup = service.ObservabilitySnapshotRollup

type observabilitySnapshotRepository struct {
	db *sql.DB
}

func NewObservabilitySnapshotRepository(db *sql.DB) service.SnapshotRollupLoader {
	if db == nil {
		return nil
	}
	return &observabilitySnapshotRepository{db: db}
}

func (r *observabilitySnapshotRepository) LoadSnapshotRollup(
	ctx context.Context,
	start, end time.Time,
	bucket observability.Bucket,
	filter observability.Filter,
) (ObservabilitySnapshotRollup, error) {
	var out ObservabilitySnapshotRollup
	if r == nil || r.db == nil {
		return out, fmt.Errorf("observability snapshot repository is unavailable")
	}
	windowSeconds, err := observability.RollupWindowSeconds(bucket)
	if err != nil {
		return out, err
	}
	baseWhere, baseArgs := snapshotRollupWhere(start, end, windowSeconds, filter)
	totalsQuery := `
SELECT
  COALESCE(SUM(request_count), 0), COALESCE(SUM(success_count), 0),
  COALESCE(SUM(error_count), 0), COALESCE(SUM(sla_error_count), 0), COALESCE(SUM(business_limited_count), 0),
  COALESCE(SUM(rate_limited_count), 0), COALESCE(SUM(legacy_inferred_count), 0),
  COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0),
  COALESCE(SUM(cache_creation_tokens), 0), COALESCE(SUM(cache_read_tokens), 0),
  COALESCE(SUM(total_tokens), 0),
  COALESCE(SUM(duration_sample_count), 0), COALESCE(SUM(sum_duration_ms), 0),
  COALESCE(MIN(NULLIF(min_duration_ms,0)), 0), COALESCE(MAX(max_duration_ms), 0),
  COALESCE(SUM(duration_le_100),0), COALESCE(SUM(duration_le_250),0),
  COALESCE(SUM(duration_le_500),0), COALESCE(SUM(duration_le_1000),0),
  COALESCE(SUM(duration_le_2000),0), COALESCE(SUM(duration_le_5000),0),
  COALESCE(SUM(duration_le_10000),0), COALESCE(SUM(duration_gt_10000),0),
  COALESCE(SUM(ttft_sample_count), 0), COALESCE(SUM(sum_ttft_ms), 0),
  COALESCE(MIN(NULLIF(min_ttft_ms,0)), 0), COALESCE(MAX(max_ttft_ms), 0),
  COALESCE(SUM(ttft_le_100),0), COALESCE(SUM(ttft_le_250),0),
  COALESCE(SUM(ttft_le_500),0), COALESCE(SUM(ttft_le_1000),0),
  COALESCE(SUM(ttft_le_2000),0), COALESCE(SUM(ttft_le_5000),0),
  COALESCE(SUM(ttft_le_10000),0), COALESCE(SUM(ttft_gt_10000),0),
  MAX(last_fact_at)
FROM observability_request_bucket ` + baseWhere
	var sourceMax sql.NullTime
	if err := r.db.QueryRowContext(ctx, totalsQuery, baseArgs...).Scan(
		&out.Totals.RequestCount, &out.Totals.SuccessCount, &out.Totals.ErrorCount, &out.Totals.SLAErrorCount,
		&out.Totals.BusinessLimitedCount, &out.Totals.RateLimitedCount, &out.Totals.LegacyInferredCount,
		&out.Totals.InputTokens, &out.Totals.OutputTokens, &out.Totals.CacheCreationTokens,
		&out.Totals.CacheReadTokens, &out.Totals.TotalTokens,
		&out.Totals.Duration.SampleCount, &out.Totals.Duration.SumMS,
		&out.Totals.Duration.MinMS, &out.Totals.Duration.MaxMS,
		&out.Totals.Duration.Buckets[0], &out.Totals.Duration.Buckets[1],
		&out.Totals.Duration.Buckets[2], &out.Totals.Duration.Buckets[3],
		&out.Totals.Duration.Buckets[4], &out.Totals.Duration.Buckets[5],
		&out.Totals.Duration.Buckets[6], &out.Totals.Duration.Buckets[7],
		&out.Totals.TTFT.SampleCount, &out.Totals.TTFT.SumMS,
		&out.Totals.TTFT.MinMS, &out.Totals.TTFT.MaxMS,
		&out.Totals.TTFT.Buckets[0], &out.Totals.TTFT.Buckets[1],
		&out.Totals.TTFT.Buckets[2], &out.Totals.TTFT.Buckets[3],
		&out.Totals.TTFT.Buckets[4], &out.Totals.TTFT.Buckets[5],
		&out.Totals.TTFT.Buckets[6], &out.Totals.TTFT.Buckets[7],
		&sourceMax,
	); err != nil {
		return out, err
	}
	if sourceMax.Valid {
		value := sourceMax.Time.UTC()
		out.SourceMaxAt = &value
	}
	out.Users, err = r.loadSnapshotDimension(ctx, start, end, windowSeconds, filter, "user_id", 20)
	if err != nil {
		return out, err
	}
	out.Groups, err = r.loadSnapshotDimension(ctx, start, end, windowSeconds, filter, "group_id", 20)
	if err != nil {
		return out, err
	}
	out.Aliases, err = r.loadSnapshotDimension(ctx, start, end, windowSeconds, filter, "route_alias", 20)
	if err != nil {
		return out, err
	}
	out.Accounts, err = r.loadSnapshotDimension(ctx, start, end, windowSeconds, filter, "account_id", 50)
	return out, err
}

func (r *observabilitySnapshotRepository) LoadRateLimitedCount(
	ctx context.Context,
	start, end time.Time,
	bucket observability.Bucket,
	filter observability.Filter,
) (int64, error) {
	windowSeconds, err := observability.RollupWindowSeconds(bucket)
	if err != nil {
		return 0, err
	}
	where, args := snapshotRollupWhere(start, end, windowSeconds, filter)
	var value int64
	err = r.db.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(rate_limited_count), 0) FROM observability_request_bucket "+where,
		args...,
	).Scan(&value)
	return value, err
}

func (r *observabilitySnapshotRepository) LoadSnapshotSeries(
	ctx context.Context,
	start, end time.Time,
	bucket observability.Bucket,
	filter observability.Filter,
	dimension string,
	selectedIDs []int64,
	selectedAliases []string,
) ([]service.SnapshotSeriesBucket, error) {
	windowSeconds, err := observability.RollupWindowSeconds(bucket)
	if err != nil {
		return nil, err
	}
	where, args := snapshotRollupWhere(start, end, windowSeconds, filter)
	column := map[string]string{
		"": "0", "user_id": "user_id", "group_id": "group_id",
		"route_alias": "route_alias", "account_id": "account_id",
	}[dimension]
	if column == "" {
		return nil, fmt.Errorf("unsupported snapshot series dimension %q", dimension)
	}
	groupBy := "bucket_start"
	if dimension != "" {
		conditions := make([]string, 0)
		if dimension == "route_alias" && len(selectedAliases) > 0 {
			placeholders := make([]string, len(selectedAliases))
			for i, alias := range selectedAliases {
				placeholders[i] = "?"
				args = append(args, alias)
			}
			conditions = append(conditions, "route_alias IN ("+strings.Join(placeholders, ",")+")")
		} else if dimension != "route_alias" && len(selectedIDs) > 0 {
			placeholders := make([]string, len(selectedIDs))
			for i, id := range selectedIDs {
				placeholders[i] = "?"
				args = append(args, id)
			}
			conditions = append(conditions, column+" IN ("+strings.Join(placeholders, ",")+")")
		}
		if len(conditions) > 0 {
			where += " AND " + strings.Join(conditions, " AND ")
		}
		groupBy += ", " + column
	}
	query := fmt.Sprintf(`
SELECT bucket_start, %s AS dimension_id,
       SUM(request_count), SUM(success_count), SUM(error_count), SUM(sla_error_count),
       SUM(business_limited_count), SUM(rate_limited_count), SUM(legacy_inferred_count),
       SUM(input_tokens), SUM(output_tokens), SUM(cache_creation_tokens), SUM(cache_read_tokens), SUM(total_tokens),
       SUM(duration_sample_count), SUM(sum_duration_ms), COALESCE(MIN(NULLIF(min_duration_ms,0)),0), COALESCE(MAX(max_duration_ms),0),
       SUM(duration_le_100), SUM(duration_le_250), SUM(duration_le_500), SUM(duration_le_1000),
       SUM(duration_le_2000), SUM(duration_le_5000), SUM(duration_le_10000), SUM(duration_gt_10000),
       SUM(ttft_sample_count), SUM(sum_ttft_ms), COALESCE(MIN(NULLIF(min_ttft_ms,0)),0), COALESCE(MAX(max_ttft_ms),0),
       SUM(ttft_le_100), SUM(ttft_le_250), SUM(ttft_le_500), SUM(ttft_le_1000),
       SUM(ttft_le_2000), SUM(ttft_le_5000), SUM(ttft_le_10000), SUM(ttft_gt_10000)
FROM observability_request_bucket %s
GROUP BY %s
ORDER BY bucket_start, dimension_id`, column, where, groupBy)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.SnapshotSeriesBucket, 0)
	for rows.Next() {
		var item service.SnapshotSeriesBucket
		var dimensionID sql.NullString
		if err := rows.Scan(
			&item.BucketStart, &dimensionID,
			&item.Totals.RequestCount, &item.Totals.SuccessCount, &item.Totals.ErrorCount, &item.Totals.SLAErrorCount,
			&item.Totals.BusinessLimitedCount, &item.Totals.RateLimitedCount, &item.Totals.LegacyInferredCount,
			&item.Totals.InputTokens, &item.Totals.OutputTokens, &item.Totals.CacheCreationTokens,
			&item.Totals.CacheReadTokens, &item.Totals.TotalTokens,
			&item.Totals.Duration.SampleCount, &item.Totals.Duration.SumMS,
			&item.Totals.Duration.MinMS, &item.Totals.Duration.MaxMS,
			&item.Totals.Duration.Buckets[0], &item.Totals.Duration.Buckets[1],
			&item.Totals.Duration.Buckets[2], &item.Totals.Duration.Buckets[3],
			&item.Totals.Duration.Buckets[4], &item.Totals.Duration.Buckets[5],
			&item.Totals.Duration.Buckets[6], &item.Totals.Duration.Buckets[7],
			&item.Totals.TTFT.SampleCount, &item.Totals.TTFT.SumMS,
			&item.Totals.TTFT.MinMS, &item.Totals.TTFT.MaxMS,
			&item.Totals.TTFT.Buckets[0], &item.Totals.TTFT.Buckets[1],
			&item.Totals.TTFT.Buckets[2], &item.Totals.TTFT.Buckets[3],
			&item.Totals.TTFT.Buckets[4], &item.Totals.TTFT.Buckets[5],
			&item.Totals.TTFT.Buckets[6], &item.Totals.TTFT.Buckets[7],
		); err != nil {
			return nil, err
		}
		item.BucketStart = utcBucketFromDB(item.BucketStart)
		if dimension == "route_alias" {
			item.RouteAlias = dimensionID.String
		} else if dimensionID.Valid {
			item.DimensionID, _ = strconv.ParseInt(dimensionID.String, 10, 64)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *observabilitySnapshotRepository) LoadSnapshotDimensionNames(
	ctx context.Context,
	dimension string,
	ids []int64,
) (map[int64]string, error) {
	return loadObservabilityDimensionNames(ctx, r.db, dimension, ids)
}

func (r *observabilitySnapshotRepository) loadSnapshotDimension(
	ctx context.Context,
	start, end time.Time,
	windowSeconds int,
	filter observability.Filter,
	dimension string,
	limit int,
) ([]snapshotDimensionRow, error) {
	column := map[string]string{
		"user_id": "user_id", "group_id": "group_id",
		"route_alias": "route_alias", "account_id": "account_id",
	}[dimension]
	if column == "" {
		return nil, fmt.Errorf("unsupported snapshot dimension %q", dimension)
	}
	where, args := snapshotRollupWhere(start, end, windowSeconds, filter)
	query := fmt.Sprintf(`
SELECT %s AS dimension_id,
       SUM(request_count), SUM(error_count), SUM(sla_error_count), SUM(business_limited_count), SUM(total_tokens),
       SUM(duration_sample_count), SUM(sum_duration_ms), COALESCE(MIN(NULLIF(min_duration_ms,0)),0), COALESCE(MAX(max_duration_ms),0),
       SUM(duration_le_100), SUM(duration_le_250), SUM(duration_le_500), SUM(duration_le_1000),
       SUM(duration_le_2000), SUM(duration_le_5000), SUM(duration_le_10000), SUM(duration_gt_10000),
       SUM(ttft_sample_count), SUM(sum_ttft_ms), COALESCE(MIN(NULLIF(min_ttft_ms,0)),0), COALESCE(MAX(max_ttft_ms),0),
       SUM(ttft_le_100), SUM(ttft_le_250), SUM(ttft_le_500), SUM(ttft_le_1000),
       SUM(ttft_le_2000), SUM(ttft_le_5000), SUM(ttft_le_10000), SUM(ttft_gt_10000)
FROM observability_request_bucket %s
GROUP BY %s
ORDER BY SUM(total_tokens) DESC, SUM(request_count) DESC
LIMIT ?`, column, where, column)
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]snapshotDimensionRow, 0, limit)
	for rows.Next() {
		var item snapshotDimensionRow
		var dimensionID sql.NullString
		if err := rows.Scan(
			&dimensionID,
			&item.RequestCount, &item.ErrorCount, &item.SLAErrorCount, &item.BusinessLimited, &item.TotalTokens,
			&item.Duration.SampleCount, &item.Duration.SumMS, &item.Duration.MinMS, &item.Duration.MaxMS,
			&item.Duration.Buckets[0], &item.Duration.Buckets[1], &item.Duration.Buckets[2], &item.Duration.Buckets[3],
			&item.Duration.Buckets[4], &item.Duration.Buckets[5], &item.Duration.Buckets[6], &item.Duration.Buckets[7],
			&item.TTFT.SampleCount, &item.TTFT.SumMS, &item.TTFT.MinMS, &item.TTFT.MaxMS,
			&item.TTFT.Buckets[0], &item.TTFT.Buckets[1], &item.TTFT.Buckets[2], &item.TTFT.Buckets[3],
			&item.TTFT.Buckets[4], &item.TTFT.Buckets[5], &item.TTFT.Buckets[6], &item.TTFT.Buckets[7],
		); err != nil {
			return nil, err
		}
		if dimension == "route_alias" {
			item.RouteAlias = dimensionID.String
		} else {
			item.ID, _ = strconv.ParseInt(dimensionID.String, 10, 64)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func snapshotRollupWhere(start, end time.Time, windowSeconds int, filter observability.Filter) (string, []any) {
	conditions := []string{"window_seconds = ?", "bucket_start >= ?", "bucket_start < ?"}
	args := []any{
		windowSeconds,
		utcBucketForDB(start, timezone.Location()),
		utcBucketForDB(end, timezone.Location()),
	}
	appendIDs := func(column string, values []int64) {
		if len(values) == 0 {
			return
		}
		placeholders := make([]string, len(values))
		for i, value := range values {
			placeholders[i] = "?"
			args = append(args, value)
		}
		conditions = append(conditions, column+" IN ("+strings.Join(placeholders, ",")+")")
	}
	appendIDs("user_id", filter.UserIDs)
	appendIDs("api_key_id", filter.APIKeyIDs)
	appendIDs("group_id", filter.GroupIDs)
	appendIDs("account_id", filter.AccountIDs)
	if len(filter.RouteAliases) > 0 {
		placeholders := make([]string, len(filter.RouteAliases))
		for i, value := range filter.RouteAliases {
			placeholders[i] = "?"
			args = append(args, value)
		}
		conditions = append(conditions, "route_alias IN ("+strings.Join(placeholders, ",")+")")
	}
	if filter.Department != "" {
		conditions = append(conditions, "user_id IN (SELECT id FROM users WHERE department = ?)")
		args = append(args, filter.Department)
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}
