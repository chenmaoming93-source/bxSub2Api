package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const opsLatencyModelExpr = "COALESCE(NULLIF(TRIM(ul.requested_model), ''), NULLIF(TRIM(ul.model), ''), 'unknown')"

func (r *opsRepository) GetModelLatencyPercentiles(ctx context.Context, filter *service.OpsDashboardFilter) (*service.OpsModelLatencyPercentilesResponse, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	if filter == nil || filter.StartTime.IsZero() || filter.EndTime.IsZero() {
		return nil, fmt.Errorf("start_time/end_time required")
	}

	start, end := filter.StartTime.UTC(), filter.EndTime.UTC()
	join, where, args, _ := buildUsageWhere(filter, start, end, 1)
	query := `
WITH filtered AS (
  SELECT ` + opsLatencyModelExpr + ` AS model, ul.duration_ms
  FROM usage_logs ul
  ` + join + `
  ` + where + `
    AND ul.duration_ms IS NOT NULL
    AND ul.duration_ms >= 0
), ranked AS (
  SELECT model, duration_ms,
         ROW_NUMBER() OVER (PARTITION BY model ORDER BY duration_ms) AS row_num,
         COUNT(*) OVER (PARTITION BY model) AS sample_count
  FROM filtered
)
SELECT model,
       MAX(sample_count) AS request_count,
       ROUND(AVG(duration_ms)) AS avg_ms,
       MAX(CASE WHEN row_num = GREATEST(1, CEIL(sample_count * 0.50)) THEN duration_ms END) AS p50_ms,
       MAX(CASE WHEN row_num = GREATEST(1, CEIL(sample_count * 0.90)) THEN duration_ms END) AS p90_ms,
       MAX(CASE WHEN row_num = GREATEST(1, CEIL(sample_count * 0.95)) THEN duration_ms END) AS p95_ms,
       MAX(CASE WHEN row_num = GREATEST(1, CEIL(sample_count * 0.99)) THEN duration_ms END) AS p99_ms
FROM ranked
GROUP BY model
ORDER BY request_count DESC, model ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	models := make([]*service.OpsModelLatencyPercentile, 0)
	for rows.Next() {
		item := &service.OpsModelLatencyPercentile{}
		var avg, p50, p90, p95, p99 sql.NullInt64
		if err := rows.Scan(&item.Model, &item.RequestCount, &avg, &p50, &p90, &p95, &p99); err != nil {
			return nil, err
		}
		item.AvgMS = opsNullableIntPtr(avg)
		item.P50MS = opsNullableIntPtr(p50)
		item.P90MS = opsNullableIntPtr(p90)
		item.P95MS = opsNullableIntPtr(p95)
		item.P99MS = opsNullableIntPtr(p99)
		models = append(models, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &service.OpsModelLatencyPercentilesResponse{
		StartTime: start,
		EndTime:   end,
		Platform:  strings.TrimSpace(filter.Platform),
		GroupID:   filter.GroupID,
		Model:     strings.TrimSpace(filter.Model),
		Models:    models,
	}, nil
}

func (r *opsRepository) GetModelLatencyTrend(ctx context.Context, filter *service.OpsDashboardFilter, bucketSeconds int) (*service.OpsModelLatencyTrendResponse, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil ops repository")
	}
	if filter == nil || filter.StartTime.IsZero() || filter.EndTime.IsZero() {
		return nil, fmt.Errorf("start_time/end_time required")
	}
	if bucketSeconds != 60 && bucketSeconds != 300 && bucketSeconds != 3600 {
		return nil, fmt.Errorf("unsupported bucket seconds: %d", bucketSeconds)
	}

	start, end := filter.StartTime.UTC(), filter.EndTime.UTC()
	join, where, args, _ := buildUsageWhere(filter, start, end, 1)
	bucketExpr := opsBucketExprForUsage(bucketSeconds)
	query := `
WITH filtered AS (
  SELECT ` + bucketExpr + ` AS bucket_start,
         ` + opsLatencyModelExpr + ` AS model,
         ul.duration_ms
  FROM usage_logs ul
  ` + join + `
  ` + where + `
    AND ul.duration_ms IS NOT NULL
    AND ul.duration_ms >= 0
), ranked AS (
  SELECT bucket_start, model, duration_ms,
         ROW_NUMBER() OVER (PARTITION BY bucket_start, model ORDER BY duration_ms) AS row_num,
         COUNT(*) OVER (PARTITION BY bucket_start, model) AS sample_count
  FROM filtered
)
SELECT bucket_start, model,
       MAX(sample_count) AS request_count,
       ROUND(AVG(duration_ms)) AS avg_ms,
       MAX(CASE WHEN row_num = GREATEST(1, CEIL(sample_count * 0.50)) THEN duration_ms END) AS p50_ms,
       MAX(CASE WHEN row_num = GREATEST(1, CEIL(sample_count * 0.90)) THEN duration_ms END) AS p90_ms,
       MAX(CASE WHEN row_num = GREATEST(1, CEIL(sample_count * 0.95)) THEN duration_ms END) AS p95_ms,
       MAX(CASE WHEN row_num = GREATEST(1, CEIL(sample_count * 0.99)) THEN duration_ms END) AS p99_ms
FROM ranked
GROUP BY bucket_start, model
ORDER BY bucket_start ASC, model ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	points := make([]*service.OpsModelLatencyTrendPoint, 0)
	for rows.Next() {
		point := &service.OpsModelLatencyTrendPoint{}
		var bucket time.Time
		var avg, p50, p90, p95, p99 sql.NullInt64
		if err := rows.Scan(&bucket, &point.Model, &point.RequestCount, &avg, &p50, &p90, &p95, &p99); err != nil {
			return nil, err
		}
		point.BucketStart = bucket.UTC()
		point.AvgMS = opsNullableIntPtr(avg)
		point.P50MS = opsNullableIntPtr(p50)
		point.P90MS = opsNullableIntPtr(p90)
		point.P95MS = opsNullableIntPtr(p95)
		point.P99MS = opsNullableIntPtr(p99)
		points = append(points, point)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &service.OpsModelLatencyTrendResponse{
		StartTime: start,
		EndTime:   end,
		Platform:  strings.TrimSpace(filter.Platform),
		GroupID:   filter.GroupID,
		Model:     strings.TrimSpace(filter.Model),
		Bucket:    opsBucketLabel(bucketSeconds),
		Points:    points,
	}, nil
}

func opsNullableIntPtr(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	converted := int(value.Int64)
	return &converted
}
