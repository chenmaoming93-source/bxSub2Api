package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/service/observability"
)

type observabilityPathRepository struct {
	db *sql.DB
}

func NewObservabilityPathRepository(db *sql.DB) service.ObservabilityPathLoader {
	if db == nil {
		return nil
	}
	return &observabilityPathRepository{db: db}
}

func (r *observabilityPathRepository) LoadPathAggregates(ctx context.Context, query service.ObservabilityPathQuery) ([]service.ObservabilityPathAggregate, int, error) {
	windowSeconds, err := observability.RollupWindowSeconds(query.Bucket)
	if err != nil {
		return nil, 0, err
	}
	where, args := pathWhere(query.Start, query.End, windowSeconds, query.Filter)
	searchConditions := make([]string, 0)
	if search := strings.TrimSpace(query.Search); search != "" {
		like := "%" + search + "%"
		searchConditions = append(searchConditions,
			"user_id IN (SELECT id FROM users WHERE email LIKE ? OR department LIKE ? OR username LIKE ?)",
			"api_key_id IN (SELECT id FROM api_keys WHERE name LIKE ?)",
			"group_id IN (SELECT id FROM `groups` WHERE name LIKE ?)",
			"account_id IN (SELECT id FROM accounts WHERE name LIKE ?)",
		)
		args = append(args, like, like, like, like, like, like)
		where += " AND (" + strings.Join(searchConditions, " OR ") + ")"
	}
	having := ""
	switch strings.TrimSpace(query.Status) {
	case "":
	case "healthy":
		having = " HAVING SUM(error_count) / NULLIF(SUM(request_count),0) < 0.2"
	case "degraded":
		having = " HAVING SUM(error_count) / NULLIF(SUM(request_count),0) >= 0.2 AND SUM(error_count) / NULLIF(SUM(request_count),0) < 0.5"
	case "error":
		having = " HAVING SUM(error_count) / NULLIF(SUM(request_count),0) >= 0.5"
	default:
		return nil, 0, fmt.Errorf("unsupported path status %q", query.Status)
	}
	groupBy := "user_id, api_key_id, group_id, route_alias, account_id"
	countQuery := "SELECT COUNT(*) FROM (SELECT 1 FROM observability_request_bucket " + where + " GROUP BY " + groupBy + having + ") path_count"
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	orderBy := pathOrderBy(query.Sort)
	offset := (query.Page - 1) * query.PageSize
	selectArgs := append(append([]any{}, args...), query.PageSize, offset)
	selectQuery := `
SELECT user_id, api_key_id, group_id, COALESCE(route_alias,''), account_id,
       SUM(request_count), SUM(success_count), SUM(error_count), SUM(sla_error_count),
       SUM(business_limited_count), SUM(legacy_inferred_count), SUM(total_tokens),
       SUM(duration_sample_count), SUM(sum_duration_ms), COALESCE(MIN(NULLIF(min_duration_ms,0)),0), COALESCE(MAX(max_duration_ms),0),
       SUM(duration_le_100), SUM(duration_le_250), SUM(duration_le_500), SUM(duration_le_1000),
       SUM(duration_le_2000), SUM(duration_le_5000), SUM(duration_le_10000), SUM(duration_gt_10000),
       SUM(ttft_sample_count), SUM(sum_ttft_ms), COALESCE(MIN(NULLIF(min_ttft_ms,0)),0), COALESCE(MAX(max_ttft_ms),0),
       SUM(ttft_le_100), SUM(ttft_le_250), SUM(ttft_le_500), SUM(ttft_le_1000),
       SUM(ttft_le_2000), SUM(ttft_le_5000), SUM(ttft_le_10000), SUM(ttft_gt_10000),
       MAX(last_fact_at)
FROM observability_request_bucket ` + where + `
GROUP BY ` + groupBy + having + `
ORDER BY ` + orderBy + `
LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, selectQuery, selectArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.ObservabilityPathAggregate, 0, query.PageSize)
	for rows.Next() {
		var item service.ObservabilityPathAggregate
		if err := rows.Scan(
			&item.UserID, &item.APIKeyID, &item.GroupID, &item.RouteAlias, &item.AccountID,
			&item.RequestCount, &item.SuccessCount, &item.ErrorCount, &item.SLAErrorCount,
			&item.BusinessLimited, &item.LegacyInferred, &item.TotalTokens,
			&item.Duration.SampleCount, &item.Duration.SumMS, &item.Duration.MinMS, &item.Duration.MaxMS,
			&item.Duration.Buckets[0], &item.Duration.Buckets[1], &item.Duration.Buckets[2], &item.Duration.Buckets[3],
			&item.Duration.Buckets[4], &item.Duration.Buckets[5], &item.Duration.Buckets[6], &item.Duration.Buckets[7],
			&item.TTFT.SampleCount, &item.TTFT.SumMS, &item.TTFT.MinMS, &item.TTFT.MaxMS,
			&item.TTFT.Buckets[0], &item.TTFT.Buckets[1], &item.TTFT.Buckets[2], &item.TTFT.Buckets[3],
			&item.TTFT.Buckets[4], &item.TTFT.Buckets[5], &item.TTFT.Buckets[6], &item.TTFT.Buckets[7],
			&item.DataAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, rows.Err()
}

func (r *observabilityPathRepository) LoadPathWindowMetrics(
	ctx context.Context,
	start, end time.Time,
	bucket observability.Bucket,
	refs []service.PathRef,
) (map[string]service.PathWindowMetrics, error) {
	out := make(map[string]service.PathWindowMetrics, len(refs))
	if len(refs) == 0 {
		return out, nil
	}
	windowSeconds, err := observability.RollupWindowSeconds(bucket)
	if err != nil {
		return nil, err
	}
	conditions := make([]string, 0, len(refs))
	args := []any{
		windowSeconds,
		utcBucketForDB(start, timezone.Location()),
		utcBucketForDB(end, timezone.Location()),
	}
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		key := servicePathRefKey(ref)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		conditions = append(conditions,
			"(user_id = ? AND api_key_id = ? AND group_id = ? AND COALESCE(route_alias, '') = ? AND account_id = ?)")
		args = append(args, ref.UserID, ref.APIKeyID, ref.GroupID, strings.TrimSpace(ref.RouteAlias), ref.AccountID)
	}
	if len(conditions) == 0 {
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT user_id, api_key_id, group_id, COALESCE(route_alias, ''), account_id,
       SUM(duration_sample_count), SUM(sum_duration_ms),
       SUM(ttft_sample_count), SUM(sum_ttft_ms)
FROM observability_request_bucket
WHERE window_seconds = ? AND bucket_start >= ? AND bucket_start < ?
  AND (`+strings.Join(conditions, " OR ")+`)
GROUP BY user_id, api_key_id, group_id, route_alias, account_id`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var ref service.PathRef
		var durationSamples, durationSum, ttftSamples, ttftSum int64
		if err := rows.Scan(
			&ref.UserID, &ref.APIKeyID, &ref.GroupID, &ref.RouteAlias, &ref.AccountID,
			&durationSamples, &durationSum, &ttftSamples, &ttftSum,
		); err != nil {
			return nil, err
		}
		value := service.PathWindowMetrics{}
		if durationSamples > 0 {
			value.AverageDurationMS = float64(durationSum) / float64(durationSamples)
		}
		if ttftSamples > 0 {
			value.AverageFirstTokenMS = float64(ttftSum) / float64(ttftSamples)
		}
		out[servicePathRefKey(ref)] = value
	}
	return out, rows.Err()
}

func (r *observabilityPathRepository) LoadPathErrorStages(
	ctx context.Context,
	start, end time.Time,
	refs []service.PathRef,
) (map[string]string, error) {
	out := make(map[string]string, len(refs))
	if len(refs) == 0 {
		return out, nil
	}
	conditions := make([]string, 0, len(refs))
	args := []any{start.In(timezone.Location()), end.In(timezone.Location())}
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		key := servicePathRefKey(ref)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		conditions = append(conditions,
			"(user_id = ? AND api_key_id = ? AND group_id = ? AND COALESCE(route_alias, '') = ? AND account_id = ?)")
		args = append(args, ref.UserID, ref.APIKeyID, ref.GroupID, strings.TrimSpace(ref.RouteAlias), ref.AccountID)
	}
	if len(conditions) == 0 {
		return out, nil
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT user_id, api_key_id, group_id, route_alias, account_id, error_phase
FROM (
  SELECT
    user_id,
    api_key_id,
    group_id,
    COALESCE(route_alias, '') AS route_alias,
    account_id,
    COALESCE(NULLIF(TRIM(error_phase), ''), 'unknown') AS error_phase,
    ROW_NUMBER() OVER (
      PARTITION BY user_id, api_key_id, group_id, COALESCE(route_alias, ''), account_id
      ORDER BY COUNT(*) DESC, COALESCE(NULLIF(TRIM(error_phase), ''), 'unknown')
    ) AS phase_rank
  FROM ops_error_logs
  WHERE created_at >= ? AND created_at < ?
    AND (
      COALESCE(status_code, 0) >= 400
      OR (status_code IS NULL AND error_phase IN ('request','auth','routing','upstream','network','internal'))
    )
    AND (`+strings.Join(conditions, " OR ")+`)
  GROUP BY user_id, api_key_id, group_id, route_alias, account_id, error_phase
) ranked
WHERE phase_rank = 1`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var ref service.PathRef
		var phase string
		if err := rows.Scan(
			&ref.UserID, &ref.APIKeyID, &ref.GroupID, &ref.RouteAlias, &ref.AccountID, &phase,
		); err != nil {
			return nil, err
		}
		out[servicePathRefKey(ref)] = phase
	}
	return out, rows.Err()
}

func servicePathRefKey(ref service.PathRef) string {
	return fmt.Sprintf("%d|%d|%d|%s|%d", ref.UserID, ref.APIKeyID, ref.GroupID, ref.RouteAlias, ref.AccountID)
}

func pathWhere(start, end time.Time, windowSeconds int, filter observability.Filter) (string, []any) {
	conditions := []string{"window_seconds = ?", "bucket_start >= ?", "bucket_start < ?"}
	args := []any{windowSeconds, utcBucketForDB(start, timezone.Location()), utcBucketForDB(end, timezone.Location())}
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

func pathOrderBy(sortBy string) string {
	switch strings.TrimSpace(sortBy) {
	case "requests":
		return "SUM(request_count) DESC, SUM(total_tokens) DESC, user_id, api_key_id, group_id, route_alias, account_id"
	case "error_rate":
		return "SUM(error_count) / NULLIF(SUM(request_count),0) DESC, SUM(request_count) DESC, user_id, api_key_id, group_id, route_alias, account_id"
	case "avg_duration":
		return "SUM(sum_duration_ms) / NULLIF(SUM(duration_sample_count),0) DESC, user_id, api_key_id, group_id, route_alias, account_id"
	default:
		return "SUM(total_tokens) DESC, SUM(request_count) DESC, user_id, api_key_id, group_id, route_alias, account_id"
	}
}

func (r *observabilityPathRepository) LoadAPIKeyRelations(ctx context.Context, filter observability.Filter, search string) ([]service.APIKeyRelation, error) {
	conditions := []string{"k.deleted_at IS NULL", "k.group_id IS NOT NULL"}
	args := make([]any, 0)
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
	appendIDs("k.user_id", filter.UserIDs)
	appendIDs("k.id", filter.APIKeyIDs)
	appendIDs("k.group_id", filter.GroupIDs)
	if filter.Department != "" {
		conditions = append(conditions, "u.department = ?")
		args = append(args, filter.Department)
	}
	if strings.TrimSpace(search) != "" {
		like := "%" + strings.TrimSpace(search) + "%"
		conditions = append(conditions, "(u.email LIKE ? OR u.department LIKE ? OR k.name LIKE ?)")
		args = append(args, like, like, like)
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT k.user_id, k.id, COALESCE(k.name,''), k.group_id
FROM api_keys k JOIN users u ON u.id = k.user_id
WHERE `+strings.Join(conditions, " AND ")+`
ORDER BY k.user_id, k.id
LIMIT 10000`, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.APIKeyRelation, 0)
	for rows.Next() {
		var item service.APIKeyRelation
		if err := rows.Scan(&item.UserID, &item.KeyID, &item.KeyName, &item.GroupID); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *observabilityPathRepository) LoadRouteRelations(ctx context.Context, filter observability.Filter) ([]service.RouteRelation, error) {
	conditions := []string{"g.deleted_at IS NULL"}
	args := make([]any, 0)
	if len(filter.GroupIDs) > 0 {
		placeholders := make([]string, len(filter.GroupIDs))
		for i, value := range filter.GroupIDs {
			placeholders[i] = "?"
			args = append(args, value)
		}
		conditions = append(conditions, "r.group_id IN ("+strings.Join(placeholders, ",")+")")
	}
	if len(filter.AccountIDs) > 0 {
		placeholders := make([]string, len(filter.AccountIDs))
		for i, value := range filter.AccountIDs {
			placeholders[i] = "?"
			args = append(args, value)
		}
		conditions = append(conditions, "r.account_id IN ("+strings.Join(placeholders, ",")+")")
	}
	if len(filter.RouteAliases) > 0 {
		placeholders := make([]string, len(filter.RouteAliases))
		for i, value := range filter.RouteAliases {
			placeholders[i] = "?"
			args = append(args, value)
		}
		conditions = append(conditions, "r.route_alias IN ("+strings.Join(placeholders, ",")+")")
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT r.group_id, r.route_alias, r.account_id, r.max_concurrency
FROM group_model_route_accounts r JOIN `+"`groups`"+` g ON g.id = r.group_id
WHERE `+strings.Join(conditions, " AND ")+`
ORDER BY r.group_id, r.route_alias, r.account_id
LIMIT 10000`, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]service.RouteRelation, 0)
	for rows.Next() {
		var item service.RouteRelation
		var limit sql.NullInt64
		if err := rows.Scan(&item.GroupID, &item.RouteAlias, &item.AccountID, &limit); err != nil {
			return nil, err
		}
		if limit.Valid {
			value := int(limit.Int64)
			item.Limit = &value
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
