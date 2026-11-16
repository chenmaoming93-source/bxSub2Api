package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service/observability"
)

type ObservabilityQueryRequest struct {
	Window    observability.Window
	Bucket    observability.Bucket
	Start     time.Time
	End       time.Time
	Filter    observability.Filter
	GroupBy   string
	ValueKind string
	Scope     string
}

type ObservabilityQueryService struct {
	loader      SnapshotRollupLoader
	concurrency ConcurrencyHistoryQuerier

	cacheMu sync.Mutex
	cache   map[string]queryCacheEntry
}

type ConcurrencyHistoryQuerier interface {
	QueryHistory(ctx context.Context, filter ConcurrencyHistoryFilter) (ConcurrencyHistoryResult, error)
}

func NewObservabilityQueryService(loader SnapshotRollupLoader, concurrency ConcurrencyHistoryQuerier) *ObservabilityQueryService {
	return &ObservabilityQueryService{loader: loader, concurrency: concurrency, cache: make(map[string]queryCacheEntry)}
}

type queryCacheEntry struct {
	value     any
	expiresAt time.Time
}

func (s *ObservabilityQueryService) Timeseries(ctx context.Context, request ObservabilityQueryRequest) (observability.TimeSeriesResponse, error) {
	cacheKey := "timeseries:" + observabilityQueryCacheKey(request)
	if cached, ok := s.getCached(cacheKey); ok {
		if response, ok := cached.(observability.TimeSeriesResponse); ok {
			return response, nil
		}
	}
	prepared, err := s.prepare(ctx, request)
	if err != nil {
		return observability.TimeSeriesResponse{}, err
	}
	groupBy := normalizeGroupBy(request.GroupBy)
	if groupBy != "" && !validGroupBy(groupBy) {
		return observability.TimeSeriesResponse{}, fmt.Errorf("unsupported group_by %q", request.GroupBy)
	}
	series := []observability.Series{
		{Key: "total_requests", Name: "请求数", Points: bucketValues(prepared.plan, prepared.overall, func(v SnapshotRollupTotals) float64 { return float64(v.RequestCount) })},
		{Key: "total_tokens", Name: "Token", Points: bucketValues(prepared.plan, prepared.overall, func(v SnapshotRollupTotals) float64 { return float64(v.TotalTokens) })},
		{Key: "input_tokens", Name: "输入 Token", Points: bucketValues(prepared.plan, prepared.overall, func(v SnapshotRollupTotals) float64 { return float64(v.InputTokens) })},
		{Key: "output_tokens", Name: "输出 Token", Points: bucketValues(prepared.plan, prepared.overall, func(v SnapshotRollupTotals) float64 { return float64(v.OutputTokens) })},
		{Key: "cache_tokens", Name: "缓存 Token", Points: bucketValues(prepared.plan, prepared.overall, func(v SnapshotRollupTotals) float64 { return float64(v.CacheCreationTokens + v.CacheReadTokens) })},
		{Key: "client_failure_rate", Name: "最终失败率", Points: bucketValues(prepared.plan, prepared.overall, func(v SnapshotRollupTotals) float64 { return ratio(v.ErrorCount, v.RequestCount) })},
		{Key: "business_limited_rate", Name: "业务限制率", Points: bucketValues(prepared.plan, prepared.overall, func(v SnapshotRollupTotals) float64 { return ratio(v.BusinessLimitedCount, v.RequestCount) })},
		{Key: "sla_error_rate", Name: "SLA 错误率", Points: bucketValues(prepared.plan, prepared.overall, func(v SnapshotRollupTotals) float64 { return ratio(v.SLAErrorCount, v.RequestCount) })},
	}
	if groupBy != "" {
		selected, selectedIDs, selectedAliases, err := s.selectedDimensions(ctx, request, groupBy, prepared.start, prepared.end)
		if err != nil {
			return observability.TimeSeriesResponse{}, err
		}
		bucketRows, err := s.loader.LoadSnapshotSeries(ctx, prepared.start, prepared.end, request.Bucket, request.Filter, groupBy, selectedIDs, selectedAliases)
		if err != nil {
			return observability.TimeSeriesResponse{}, err
		}
		grouped := groupSeriesByBucket(bucketRows, prepared.plan)
		series = append(series, grouped...)
		other := otherSeries(prepared.plan, prepared.overall, grouped)
		if len(other) > 0 {
			series = append(series, other...)
		}
		_ = selected
	}
	summary := summarizeSeries(series)
	response := observability.TimeSeriesResponse{
		Meta: prepared.meta, Labels: planLabels(prepared.plan),
		Series: series, Summary: summary,
	}
	s.putCached(cacheKey, response)
	return response, nil
}

func (s *ObservabilityQueryService) LatencyTimeseries(ctx context.Context, request ObservabilityQueryRequest) (observability.TimeSeriesResponse, error) {
	cacheKey := "latency:" + observabilityQueryCacheKey(request)
	if cached, ok := s.getCached(cacheKey); ok {
		if response, ok := cached.(observability.TimeSeriesResponse); ok {
			return response, nil
		}
	}
	prepared, err := s.prepare(ctx, request)
	if err != nil {
		return observability.TimeSeriesResponse{}, err
	}
	durationP50 := make([]float64, len(prepared.plan))
	durationP95 := make([]float64, len(prepared.plan))
	ttftP95 := make([]float64, len(prepared.plan))
	avgDuration := make([]float64, len(prepared.plan))
	avgTTFT := make([]float64, len(prepared.plan))
	failureRate := make([]float64, len(prepared.plan))
	businessRate := make([]float64, len(prepared.plan))
	slaRate := make([]float64, len(prepared.plan))
	for index, point := range prepared.plan {
		totals := prepared.overall[point.Start]
		durationP50[index] = float64(totals.Duration.Quantiles(0.5)[0.5])
		durationP95[index] = float64(totals.Duration.Quantiles(0.95)[0.95])
		ttftP95[index] = float64(totals.TTFT.Quantiles(0.95)[0.95])
		if totals.Duration.SampleCount > 0 {
			avgDuration[index] = float64(totals.Duration.SumMS) / float64(totals.Duration.SampleCount)
		}
		if totals.TTFT.SampleCount > 0 {
			avgTTFT[index] = float64(totals.TTFT.SumMS) / float64(totals.TTFT.SampleCount)
		}
		failureRate[index] = ratio(totals.ErrorCount, totals.RequestCount)
		businessRate[index] = ratio(totals.BusinessLimitedCount, totals.RequestCount)
		slaRate[index] = ratio(totals.SLAErrorCount, totals.RequestCount)
		if totals.Duration.SampleCount == 0 || totals.TTFT.SampleCount == 0 {
			prepared.meta.Warnings = uniqueStrings(append(prepared.meta.Warnings, "partial_latency_histogram"))
			prepared.meta.Complete = false
		}
	}
	series := []observability.Series{
		{Key: "p50_duration_ms", Name: "P50 耗时", Points: durationP50},
		{Key: "p95_duration_ms", Name: "P95 耗时", Points: durationP95},
		{Key: "p95_first_token_ms", Name: "P95 首字", Points: ttftP95},
		{Key: "avg_duration_ms", Name: "平均耗时", Points: avgDuration},
		{Key: "avg_first_token_ms", Name: "平均首字", Points: avgTTFT},
		{Key: "client_failure_rate", Name: "最终失败率", Points: failureRate},
		{Key: "business_limited_rate", Name: "业务限制率", Points: businessRate},
		{Key: "sla_error_rate", Name: "SLA 错误率", Points: slaRate},
	}
	response := observability.TimeSeriesResponse{
		Meta: prepared.meta, Labels: planLabels(prepared.plan),
		Series: series, Summary: summarizeSeries(series),
	}
	s.putCached(cacheKey, response)
	return response, nil
}

func (s *ObservabilityQueryService) Heatmap(ctx context.Context, request ObservabilityQueryRequest) (observability.HeatmapResponse, error) {
	cacheKey := "heatmap:" + observabilityQueryCacheKey(request)
	if cached, ok := s.getCached(cacheKey); ok {
		if response, ok := cached.(observability.HeatmapResponse); ok {
			return response, nil
		}
	}
	valueKind := strings.TrimSpace(request.ValueKind)
	if valueKind == "" {
		valueKind = "sampled_concurrency"
	}
	if valueKind == "sampled_concurrency" {
		prepared, err := prepareConcurrencyHeatmap(request)
		if err != nil {
			return observability.HeatmapResponse{}, err
		}
		response, err := s.concurrencyHeatmap(ctx, request, prepared)
		if err != nil {
			return observability.HeatmapResponse{}, err
		}
		s.putCached(cacheKey, response)
		return response, nil
	}
	prepared, err := s.prepare(ctx, request)
	if err != nil {
		return observability.HeatmapResponse{}, err
	}
	var response observability.HeatmapResponse
	switch valueKind {
	case "token_rate", "error_rate":
		response, err = s.requestHeatmap(ctx, request, prepared, valueKind)
	default:
		return observability.HeatmapResponse{}, fmt.Errorf("unsupported heatmap value_kind %q", valueKind)
	}
	if err != nil {
		return observability.HeatmapResponse{}, err
	}
	s.putCached(cacheKey, response)
	return response, nil
}

func prepareConcurrencyHeatmap(request ObservabilityQueryRequest) (preparedQuery, error) {
	now := time.Now().UTC().Truncate(time.Second)
	start, end := request.Start, request.End
	if start.IsZero() || end.IsZero() {
		resolved, err := observability.ResolveRange(request.Window, request.Bucket, "", "", now, timezone.Location())
		if err != nil {
			return preparedQuery{}, err
		}
		request.Window, request.Bucket, start, end = resolved.Window, resolved.Bucket, resolved.Start, resolved.End
	} else {
		if request.Bucket == "" {
			request.Bucket = observability.DefaultBucketForDuration(end.Sub(start))
		}
		if !start.Before(end) || end.Sub(start) > observability.MaxRange {
			return preparedQuery{}, fmt.Errorf("invalid observability range")
		}
	}
	plan, err := observability.BuildRangeBucketPlan(start, end, request.Bucket, timezone.Location())
	if err != nil {
		return preparedQuery{}, err
	}
	if len(plan) == 0 {
		return preparedQuery{}, fmt.Errorf("empty observability range")
	}
	start = plan[0].Start
	end = plan[len(plan)-1].End
	return preparedQuery{
		plan: plan, start: start, end: end,
		meta: observability.Meta{
			AsOf: now, StartAt: start.UTC(), EndAt: end.UTC(), Window: request.Window, Bucket: request.Bucket,
			Timezone:    timezone.Location().String(),
			Sources:     []string{"observability_concurrency_bucket"},
			Consistency: "eventual", Complete: true, Coverage: "full",
		},
	}, nil
}

type preparedQuery struct {
	plan       []observability.BucketPoint
	start, end time.Time
	overall    map[time.Time]SnapshotRollupTotals
	meta       observability.Meta
}

func (s *ObservabilityQueryService) prepare(ctx context.Context, request ObservabilityQueryRequest) (preparedQuery, error) {
	if s == nil || s.loader == nil {
		return preparedQuery{}, fmt.Errorf("observability query service is unavailable")
	}
	now := time.Now().UTC().Truncate(time.Second)
	start, end := request.Start, request.End
	if start.IsZero() || end.IsZero() {
		resolved, resolveErr := observability.ResolveRange(request.Window, request.Bucket, "", "", now, timezone.Location())
		if resolveErr != nil {
			return preparedQuery{}, resolveErr
		}
		request.Window, request.Bucket, start, end = resolved.Window, resolved.Bucket, resolved.Start, resolved.End
	} else if request.Bucket == "" {
		request.Bucket = observability.DefaultBucketForDuration(end.Sub(start))
	}
	resolvedStart, resolvedEnd := start, end
	plan, err := observability.BuildRangeBucketPlan(start, end, request.Bucket, timezone.Location())
	if err != nil {
		return preparedQuery{}, err
	}
	if len(plan) > 0 {
		start = plan[0].Start
		end = plan[len(plan)-1].End
	}
	rows, err := s.loader.LoadSnapshotSeries(ctx, start, end, request.Bucket, request.Filter, "", nil, nil)
	if err != nil {
		return preparedQuery{}, err
	}
	overall := make(map[time.Time]SnapshotRollupTotals)
	for _, row := range rows {
		apiStart := alignAPIBucket(row.BucketStart, request.Bucket)
		value := overall[apiStart]
		mergeSnapshotTotals(&value, row.Totals)
		overall[apiStart] = value
	}
	global, globalErr := s.loader.LoadSnapshotRollup(ctx, start, end, request.Bucket, observability.Filter{})
	warnings := make([]string, 0)
	complete := true
	lag := int64(0)
	if globalErr != nil || global.SourceMaxAt == nil {
		warnings = append(warnings, "request_rollup_unavailable")
		complete = false
	} else if time.Since(*global.SourceMaxAt) > 2*time.Minute {
		warnings = append(warnings, "request_rollup_lagging")
		complete = false
		lag = int64(time.Since(*global.SourceMaxAt).Seconds())
	}
	if missingSeriesBuckets(plan, overall) {
		warnings = append(warnings, "missing_request_bucket")
		complete = false
	}
	if resolvedEnd.Sub(resolvedStart) >= observability.MaxRange && request.Bucket != observability.Bucket6h {
		warnings = append(warnings, "bucket_window_invalid")
		complete = false
	}
	meta := observability.Meta{
		AsOf: now, StartAt: resolvedStart.UTC(), EndAt: resolvedEnd.UTC(), Window: request.Window, Bucket: request.Bucket,
		Timezone:    timezone.Location().String(),
		Sources:     []string{"observability_request_bucket", "observability_concurrency_bucket"},
		Consistency: "eventual", LagSeconds: lag, Complete: complete,
		Warnings: uniqueStrings(warnings), Coverage: "full",
	}
	return preparedQuery{plan: plan, start: start, end: end, overall: overall, meta: meta}, nil
}

func (s *ObservabilityQueryService) selectedDimensions(ctx context.Context, request ObservabilityQueryRequest, groupBy string, start, end time.Time) ([]SnapshotDimensionRow, []int64, []string, error) {
	rollup, err := s.loader.LoadSnapshotRollup(ctx, start, end, request.Bucket, request.Filter)
	if err != nil {
		return nil, nil, nil, err
	}
	var selected []SnapshotDimensionRow
	switch groupBy {
	case "user_id":
		selected = rollup.Users
	case "group_id":
		selected = rollup.Groups
	case "route_alias":
		selected = rollup.Aliases
	case "account_id":
		selected = rollup.Accounts
	}
	ids := make([]int64, 0, len(selected))
	aliases := make([]string, 0, len(selected))
	for _, row := range selected {
		if groupBy == "route_alias" {
			if row.RouteAlias != "" {
				aliases = append(aliases, row.RouteAlias)
			}
		} else if row.ID > 0 {
			ids = append(ids, row.ID)
		}
	}
	return selected, ids, aliases, nil
}

func (s *ObservabilityQueryService) requestHeatmap(ctx context.Context, request ObservabilityQueryRequest, prepared preparedQuery, valueKind string) (observability.HeatmapResponse, error) {
	groupBy := normalizeGroupBy(request.GroupBy)
	if groupBy == "" {
		groupBy = "group_id"
	}
	if !validGroupBy(groupBy) {
		return observability.HeatmapResponse{}, fmt.Errorf("unsupported heatmap group_by %q", groupBy)
	}
	selected, ids, aliases, err := s.selectedDimensions(ctx, request, groupBy, prepared.start, prepared.end)
	if err != nil {
		return observability.HeatmapResponse{}, err
	}
	rows, err := s.loader.LoadSnapshotSeries(ctx, prepared.start, prepared.end, request.Bucket, request.Filter, groupBy, ids, aliases)
	if err != nil {
		return observability.HeatmapResponse{}, err
	}
	grouped := groupSeriesByBucket(rows, prepared.plan)
	response := observability.HeatmapResponse{Meta: prepared.meta, ValueKind: valueKind, Available: true, Columns: planLabels(prepared.plan)}
	seen := make(map[string]struct{})
	for index := range selected {
		key := dimensionSeriesKey(selected[index])
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		response.Rows = append(response.Rows, key)
	}
	sort.Strings(response.Rows)
	for _, rowKey := range response.Rows {
		for _, point := range prepared.plan {
			bucketIndex := planIndex(prepared.plan, point.Start)
			value := 0.0
			if bucketIndex >= 0 {
				requests := seriesValueForDimension(grouped, "requests", rowKey, bucketIndex)
				switch valueKind {
				case "token_rate":
					value = seriesValueForDimension(grouped, "tokens", rowKey, bucketIndex) / point.End.Sub(point.Start).Seconds()
				case "error_rate":
					value = ratio(int64(seriesValueForDimension(grouped, "error_count", rowKey, bucketIndex)), int64(requests))
				}
			}
			response.Cells = append(response.Cells, observability.HeatmapCell{Row: rowKey, Column: point.Label, Value: value, Available: true})
		}
	}
	return response, nil
}

func (s *ObservabilityQueryService) concurrencyHeatmap(ctx context.Context, request ObservabilityQueryRequest, prepared preparedQuery) (observability.HeatmapResponse, error) {
	if s.concurrency == nil {
		prepared.meta.Warnings = uniqueStrings(append(prepared.meta.Warnings, "concurrency_samples_unavailable"))
		prepared.meta.Complete = false
		return observability.HeatmapResponse{Meta: prepared.meta, ValueKind: "sampled_concurrency", Available: false}, nil
	}
	groupBy := normalizeGroupBy(request.GroupBy)
	if groupBy == "" {
		groupBy = "group_id"
	}
	if groupBy == "user_id" {
		prepared.meta.Warnings = uniqueStrings(append(prepared.meta.Warnings, "user_concurrency_samples_unavailable"))
		prepared.meta.Complete = false
		return observability.HeatmapResponse{Meta: prepared.meta, ValueKind: "sampled_concurrency", Available: false}, nil
	}
	if groupBy == "route_alias" {
		groupBy = "route"
	}
	if groupBy != "group_id" && groupBy != "account_id" && groupBy != "route" {
		return observability.HeatmapResponse{}, fmt.Errorf("unsupported concurrency heatmap group_by %q", request.GroupBy)
	}
	storageWindow, _ := observability.RollupWindowSeconds(request.Bucket)
	historyFilter := ConcurrencyHistoryFilter{
		Start: prepared.start, End: prepared.end, WindowSeconds: storageWindow,
		AccountID: firstID(request.Filter.AccountIDs), GroupID: firstID(request.Filter.GroupIDs),
		RouteAlias: firstString(request.Filter.RouteAliases),
	}
	var history ConcurrencyHistoryResult
	var err error
	if bounded, ok := s.concurrency.(interface {
		QueryConcurrencyHeatmapHistory(context.Context, ConcurrencyHistoryFilter, string, int) ([]ConcurrencyHistoryRow, error)
	}); ok {
		var rows []ConcurrencyHistoryRow
		rows, err = bounded.QueryConcurrencyHeatmapHistory(ctx, historyFilter, groupBy, 20)
		history = ConcurrencyHistoryResult{Rows: rows, Available: err == nil, QualityStatus: "healthy"}
	} else {
		history, err = s.concurrency.QueryHistory(ctx, historyFilter)
	}
	if err != nil {
		prepared.meta.Warnings = uniqueStrings(append(prepared.meta.Warnings, "concurrency_samples_unavailable"))
		prepared.meta.Complete = false
		return observability.HeatmapResponse{Meta: prepared.meta, ValueKind: "sampled_concurrency", Available: false}, nil
	}
	type cellKey struct {
		row    string
		column time.Time
	}
	type heatValue struct {
		actual, limit int
		sampleAt      time.Time
		ratio         float64
	}
	values := make(map[cellKey]heatValue)
	rowSeen := make(map[string]struct{})
	for _, row := range history.Rows {
		if groupBy == "account_id" && row.SampleKind != ConcurrencySampleKindAccount {
			continue
		}
		if groupBy != "account_id" && row.SampleKind != ConcurrencySampleKindRoute {
			continue
		}
		key := heatmapRowKey(row, groupBy)
		if key == "" {
			continue
		}
		rowSeen[key] = struct{}{}
		column := alignAPIBucket(row.BucketStart, request.Bucket)
		keyCell := cellKey{row: key, column: column}
		actual, limit, sampledAt := row.PeakRatioActual, row.PeakRatioLimit, row.PeakRatioSampleAt
		if limit <= 0 {
			actual, limit = row.CurrentPeak, row.EffectiveLimit
			sampledAt = row.LastSampledAt
		}
		if limit > 0 {
			current := heatValue{actual: actual, limit: limit, sampleAt: sampledAt, ratio: float64(actual) / float64(limit)}
			existing, exists := values[keyCell]
			if !exists || current.ratio > existing.ratio {
				values[keyCell] = current
			}
		}
	}
	response := observability.HeatmapResponse{
		Meta: prepared.meta, ValueKind: "sampled_concurrency",
		Columns: planLabels(prepared.plan), Available: history.Available,
	}
	for row := range rowSeen {
		response.Rows = append(response.Rows, row)
	}
	sort.Strings(response.Rows)
	if !history.Available {
		response.Meta.Complete = false
		response.Meta.Warnings = uniqueStrings(append(response.Meta.Warnings, "concurrency_samples_unavailable"))
		return response, nil
	}
	for _, row := range response.Rows {
		for _, point := range prepared.plan {
			value := values[cellKey{row: row, column: point.Start}]
			var sampleAt *time.Time
			if !value.sampleAt.IsZero() {
				at := value.sampleAt.UTC()
				sampleAt = &at
			}
			response.Cells = append(response.Cells, observability.HeatmapCell{
				Row: row, Column: point.Label, Value: value.ratio, Available: value.limit > 0,
				ActualConcurrency: value.actual, ConcurrencyLimit: value.limit, SampleAt: sampleAt,
			})
		}
	}
	return response, nil
}

func groupSeriesByBucket(rows []SnapshotSeriesBucket, plan []observability.BucketPoint) []observability.Series {
	byKey := make(map[string]map[time.Time]SnapshotRollupTotals)
	for _, row := range rows {
		key := strconv.FormatInt(row.DimensionID, 10)
		if row.RouteAlias != "" {
			key = row.RouteAlias
		}
		if byKey[key] == nil {
			byKey[key] = make(map[time.Time]SnapshotRollupTotals)
		}
		apiStart := alignAPIBucket(row.BucketStart, planBucketFromRows(plan))
		value := byKey[key][apiStart]
		mergeSnapshotTotals(&value, row.Totals)
		byKey[key][apiStart] = value
	}
	out := make([]observability.Series, 0)
	for key, buckets := range byKey {
		requests := bucketValues(plan, buckets, func(v SnapshotRollupTotals) float64 { return float64(v.RequestCount) })
		tokens := bucketValues(plan, buckets, func(v SnapshotRollupTotals) float64 { return float64(v.TotalTokens) })
		inputTokens := bucketValues(plan, buckets, func(v SnapshotRollupTotals) float64 { return float64(v.InputTokens) })
		outputTokens := bucketValues(plan, buckets, func(v SnapshotRollupTotals) float64 { return float64(v.OutputTokens) })
		cacheTokens := bucketValues(plan, buckets, func(v SnapshotRollupTotals) float64 {
			return float64(v.CacheCreationTokens + v.CacheReadTokens)
		})
		failures := bucketValues(plan, buckets, func(v SnapshotRollupTotals) float64 { return float64(v.ErrorCount) })
		business := bucketValues(plan, buckets, func(v SnapshotRollupTotals) float64 { return float64(v.BusinessLimitedCount) })
		sla := bucketValues(plan, buckets, func(v SnapshotRollupTotals) float64 { return float64(v.SLAErrorCount) })
		out = append(out,
			observability.Series{Key: "requests:" + key, Name: key, Points: requests},
			observability.Series{Key: "tokens:" + key, Name: key, Points: tokens},
			observability.Series{Key: "input_tokens:" + key, Name: key, Points: inputTokens},
			observability.Series{Key: "output_tokens:" + key, Name: key, Points: outputTokens},
			observability.Series{Key: "cache_tokens:" + key, Name: key, Points: cacheTokens},
			observability.Series{Key: "error_count:" + key, Name: key, Points: failures},
			observability.Series{Key: "business_limited_count:" + key, Name: key, Points: business},
			observability.Series{Key: "sla_error_count:" + key, Name: key, Points: sla},
		)
	}
	return out
}

func otherSeries(plan []observability.BucketPoint, overall map[time.Time]SnapshotRollupTotals, grouped []observability.Series) []observability.Series {
	out := make([]observability.Series, 0)
	metrics := []string{
		"requests", "tokens", "input_tokens", "output_tokens", "cache_tokens",
		"error_count", "business_limited_count", "sla_error_count",
	}
	for _, metric := range metrics {
		points := make([]float64, len(plan))
		for index, point := range plan {
			total := overall[point.Start]
			var value float64
			switch metric {
			case "requests":
				value = float64(total.RequestCount)
			case "tokens":
				value = float64(total.TotalTokens)
			case "input_tokens":
				value = float64(total.InputTokens)
			case "output_tokens":
				value = float64(total.OutputTokens)
			case "cache_tokens":
				value = float64(total.CacheCreationTokens + total.CacheReadTokens)
			case "error_count":
				value = float64(total.ErrorCount)
			case "business_limited_count":
				value = float64(total.BusinessLimitedCount)
			case "sla_error_count":
				value = float64(total.SLAErrorCount)
			}
			for _, series := range grouped {
				if strings.HasPrefix(series.Key, metric+":") {
					value -= series.Points[index]
				}
			}
			if value < 0 {
				value = 0
			}
			points[index] = value
		}
		out = append(out, observability.Series{Key: metric + ":other", Name: "其他", Points: points})
	}
	return out
}

func summarizeSeries(series []observability.Series) []observability.Series {
	out := make([]observability.Series, 0)
	for _, item := range series {
		if strings.Contains(item.Key, ":") && !strings.HasSuffix(item.Key, ":other") && !strings.HasPrefix(item.Key, "p50_") && !strings.HasPrefix(item.Key, "p95_") && !strings.HasPrefix(item.Key, "avg_") && !strings.HasPrefix(item.Key, "total_") && !strings.HasPrefix(item.Key, "client_") && !strings.HasPrefix(item.Key, "business_") && !strings.HasPrefix(item.Key, "sla_") {
			continue
		}
		var sum float64
		for _, point := range item.Points {
			sum += point
		}
		out = append(out, observability.Series{Key: "summary_" + item.Key, Name: item.Name, Points: []float64{sum}})
	}
	return out
}

func bucketValues(plan []observability.BucketPoint, buckets map[time.Time]SnapshotRollupTotals, fn func(SnapshotRollupTotals) float64) []float64 {
	points := make([]float64, len(plan))
	for i, point := range plan {
		points[i] = fn(buckets[point.Start])
	}
	return points
}

func mergeSnapshotTotals(dst *SnapshotRollupTotals, src SnapshotRollupTotals) {
	dst.RequestCount += src.RequestCount
	dst.SuccessCount += src.SuccessCount
	dst.ErrorCount += src.ErrorCount
	dst.SLAErrorCount += src.SLAErrorCount
	dst.BusinessLimitedCount += src.BusinessLimitedCount
	dst.RateLimitedCount += src.RateLimitedCount
	dst.LegacyInferredCount += src.LegacyInferredCount
	dst.InputTokens += src.InputTokens
	dst.OutputTokens += src.OutputTokens
	dst.CacheCreationTokens += src.CacheCreationTokens
	dst.CacheReadTokens += src.CacheReadTokens
	dst.TotalTokens += src.TotalTokens
	mergeHistogram(&dst.Duration, src.Duration)
	mergeHistogram(&dst.TTFT, src.TTFT)
}

func mergeHistogram(dst *observability.Histogram, src observability.Histogram) {
	dst.SampleCount += src.SampleCount
	dst.SumMS += src.SumMS
	if src.MinMS > 0 && (dst.MinMS == 0 || src.MinMS < dst.MinMS) {
		dst.MinMS = src.MinMS
	}
	if src.MaxMS > dst.MaxMS {
		dst.MaxMS = src.MaxMS
	}
	for i := range dst.Buckets {
		dst.Buckets[i] += src.Buckets[i]
	}
}

func missingSeriesBuckets(plan []observability.BucketPoint, buckets map[time.Time]SnapshotRollupTotals) bool {
	seen := false
	missing := false
	for _, point := range plan {
		total, ok := buckets[point.Start]
		if ok && total.RequestCount > 0 {
			seen = true
		} else if seen {
			missing = true
		}
	}
	return seen && missing
}

func alignAPIBucket(value time.Time, bucket observability.Bucket) time.Time {
	duration := 5 * time.Minute
	switch bucket {
	case observability.Bucket15m:
		duration = 15 * time.Minute
	case observability.Bucket1h:
		duration = time.Hour
	case observability.Bucket6h:
		duration = 6 * time.Hour
	}
	return value.UTC().Truncate(duration)
}

func planBucketFromRows(plan []observability.BucketPoint) observability.Bucket {
	if len(plan) < 2 {
		return observability.Bucket5m
	}
	switch plan[1].Start.Sub(plan[0].Start) {
	case 15 * time.Minute:
		return observability.Bucket15m
	case time.Hour:
		return observability.Bucket1h
	case 6 * time.Hour:
		return observability.Bucket6h
	default:
		return observability.Bucket5m
	}
}

func planLabels(plan []observability.BucketPoint) []string {
	labels := make([]string, len(plan))
	for i := range plan {
		labels[i] = plan[i].Label
	}
	return labels
}

func planIndex(plan []observability.BucketPoint, start time.Time) int {
	for i := range plan {
		if plan[i].Start.Equal(start) {
			return i
		}
	}
	return -1
}

func normalizeGroupBy(value string) string {
	return strings.TrimSpace(value)
}

func validGroupBy(value string) bool {
	switch value {
	case "user_id", "group_id", "route_alias", "account_id":
		return true
	default:
		return false
	}
}

func dimensionSeriesKey(row SnapshotDimensionRow) string {
	if row.RouteAlias != "" {
		return row.RouteAlias
	}
	return strconv.FormatInt(row.ID, 10)
}

func firstID(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	return values[0]
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func seriesValueForDimension(series []observability.Series, metric, key string, bucketIndex int) float64 {
	target := metric + ":" + key
	for _, item := range series {
		if item.Key == target && bucketIndex >= 0 && bucketIndex < len(item.Points) {
			return item.Points[bucketIndex]
		}
	}
	return 0
}

func heatmapRowKey(row ConcurrencyHistoryRow, groupBy string) string {
	switch groupBy {
	case "account_id":
		return strconv.FormatInt(row.AccountID, 10)
	case "route":
		return row.RouteAlias
	default:
		return strconv.FormatInt(row.GroupID, 10)
	}
}

func observabilityQueryCacheKey(request ObservabilityQueryRequest) string {
	return snapshotRequestCacheKey(SnapshotRequest{
		Window: request.Window, Bucket: request.Bucket, Start: request.Start, End: request.End, Filter: request.Filter,
		Scope: request.Scope, Search: request.GroupBy + "|" + request.ValueKind,
	})
}

func (s *ObservabilityQueryService) getCached(key string) (any, bool) {
	if s == nil {
		return nil, false
	}
	now := time.Now()
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	entry, ok := s.cache[key]
	if !ok || !now.Before(entry.expiresAt) {
		if ok {
			delete(s.cache, key)
		}
		return nil, false
	}
	return entry.value, true
}

func (s *ObservabilityQueryService) putCached(key string, value any) {
	if s == nil {
		return
	}
	now := time.Now()
	s.cacheMu.Lock()
	for cacheKey, entry := range s.cache {
		if !now.Before(entry.expiresAt) {
			delete(s.cache, cacheKey)
		}
	}
	s.cache[key] = queryCacheEntry{value: value, expiresAt: now.Add(5 * time.Second)}
	s.cacheMu.Unlock()
}
