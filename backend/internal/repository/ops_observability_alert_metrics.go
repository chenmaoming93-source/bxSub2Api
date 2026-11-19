package repository

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/service/observability"
)

func (r *opsRepository) QueryObservabilityAlertMetric(
	ctx context.Context,
	metricType string,
	scope service.OpsAlertRuleScope,
	start, end time.Time,
) (service.OpsObservabilityAlertMetric, bool) {
	if r == nil || r.db == nil || !end.After(start) {
		return service.OpsObservabilityAlertMetric{}, false
	}
	if metricType == "observability_concurrency_saturation" {
		return r.queryObservabilityConcurrencySaturation(ctx, scope, start, end)
	}
	conditions := []string{"window_seconds = 60", "bucket_start >= ?", "bucket_start < ?"}
	args := []any{
		utcBucketForDB(start, timezone.Location()),
		utcBucketForDB(end, timezone.Location()),
	}
	if scope.GroupID != nil && *scope.GroupID > 0 {
		conditions = append(conditions, "group_id = ?")
		args = append(args, *scope.GroupID)
	}
	if strings.TrimSpace(scope.RouteAlias) != "" {
		conditions = append(conditions, "route_alias = ?")
		args = append(args, strings.TrimSpace(scope.RouteAlias))
	}
	if scope.AccountID != nil && *scope.AccountID > 0 {
		conditions = append(conditions, "account_id = ?")
		args = append(args, *scope.AccountID)
	}
	where := " WHERE " + strings.Join(conditions, " AND ")
	switch metricType {
	case "observability_client_failure_rate",
		"observability_business_limited_rate",
		"observability_sla_error_rate":
		query := `
SELECT COALESCE(SUM(request_count),0),
       COALESCE(SUM(error_count),0),
       COALESCE(SUM(business_limited_count),0),
       COALESCE(SUM(sla_error_count),0),
       MAX(last_fact_at)
FROM observability_request_bucket` + where
		var requests, errors, business, sla int64
		var last sql.NullTime
		if err := r.db.QueryRowContext(ctx, query, args...).Scan(&requests, &errors, &business, &sla, &last); err != nil || requests <= 0 {
			return service.OpsObservabilityAlertMetric{}, false
		}
		value := float64(errors) * 100 / float64(requests)
		if metricType == "observability_business_limited_rate" {
			value = float64(business) * 100 / float64(requests)
		} else if metricType == "observability_sla_error_rate" {
			value = float64(sla) * 100 / float64(requests)
		}
		return service.OpsObservabilityAlertMetric{
			Value: value, SampleCount: requests, DataAt: optionalAlertMetricTime(last),
		}, true
	case "observability_ttft_p95_ms":
		query := `
SELECT COALESCE(SUM(ttft_sample_count),0), COALESCE(SUM(sum_ttft_ms),0),
       COALESCE(MIN(NULLIF(min_ttft_ms,0)),0), COALESCE(MAX(max_ttft_ms),0),
       COALESCE(SUM(ttft_le_100),0), COALESCE(SUM(ttft_le_250),0),
       COALESCE(SUM(ttft_le_500),0), COALESCE(SUM(ttft_le_1000),0),
       COALESCE(SUM(ttft_le_2000),0), COALESCE(SUM(ttft_le_5000),0),
       COALESCE(SUM(ttft_le_10000),0), COALESCE(SUM(ttft_gt_10000),0),
       MAX(last_fact_at)
FROM observability_request_bucket` + where
		var histogram observability.Histogram
		var last sql.NullTime
		if err := r.db.QueryRowContext(ctx, query, args...).Scan(
			&histogram.SampleCount, &histogram.SumMS, &histogram.MinMS, &histogram.MaxMS,
			&histogram.Buckets[0], &histogram.Buckets[1], &histogram.Buckets[2], &histogram.Buckets[3],
			&histogram.Buckets[4], &histogram.Buckets[5], &histogram.Buckets[6], &histogram.Buckets[7],
			&last,
		); err != nil || histogram.SampleCount <= 0 {
			return service.OpsObservabilityAlertMetric{}, false
		}
		return service.OpsObservabilityAlertMetric{
			Value:       float64(histogram.Quantiles(0.95)[0.95]),
			SampleCount: histogram.SampleCount,
			DataAt:      optionalAlertMetricTime(last),
		}, true
	default:
		return service.OpsObservabilityAlertMetric{}, false
	}
}

func (r *opsRepository) queryObservabilityConcurrencySaturation(
	ctx context.Context,
	scope service.OpsAlertRuleScope,
	start, end time.Time,
) (service.OpsObservabilityAlertMetric, bool) {
	conditions := []string{"window_seconds = 60", "bucket_start >= ?", "bucket_start < ?", "effective_limit > 0"}
	args := []any{
		utcBucketForDB(start, timezone.Location()),
		utcBucketForDB(end, timezone.Location()),
	}
	if scope.GroupID != nil && *scope.GroupID > 0 {
		conditions = append(conditions, "group_id = ?")
		args = append(args, *scope.GroupID)
	}
	if strings.TrimSpace(scope.RouteAlias) != "" {
		conditions = append(conditions, "route_alias = ?")
		args = append(args, strings.TrimSpace(scope.RouteAlias))
	}
	if scope.AccountID != nil && *scope.AccountID > 0 {
		conditions = append(conditions, "account_id = ?")
		args = append(args, *scope.AccountID)
	}
	query := `
SELECT COALESCE(MAX(current_peak * 100 / NULLIF(effective_limit,0)),0),
       COALESCE(SUM(sample_count),0), MAX(last_sampled_at)
FROM observability_concurrency_bucket
WHERE ` + strings.Join(conditions, " AND ")
	var value float64
	var samples int64
	var last sql.NullTime
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&value, &samples, &last); err != nil || samples <= 0 {
		return service.OpsObservabilityAlertMetric{}, false
	}
	return service.OpsObservabilityAlertMetric{Value: value, SampleCount: samples, DataAt: optionalAlertMetricTime(last)}, true
}

func optionalAlertMetricTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time.UTC()
	return &t
}
