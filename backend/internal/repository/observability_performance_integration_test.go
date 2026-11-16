//go:build integration

package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/service/observability"
	"github.com/stretchr/testify/require"
)

const (
	performanceAccountCount = 1000
	performancePathCount    = 10000
)

// TestObservabilityPerformanceGate is an opt-in release gate. It runs against
// a dedicated, disposable schema on the configured MySQL server so the normal
// integration suite can still skip cleanly when Docker is unavailable.
func TestObservabilityPerformanceGate(t *testing.T) {
	if os.Getenv("OBSERVABILITY_PERF_EXTERNAL") != "1" && os.Getenv("OBSERVABILITY_PERF_SHADOW_TABLES") != "1" {
		t.Skip("set OBSERVABILITY_PERF_EXTERNAL=1 to run the fixed-size observability performance gate")
	}
	ctx := context.Background()
	if os.Getenv("OBSERVABILITY_PERF_SHADOW_TABLES") != "1" {
		applyPerformanceDDL(t, ctx)
	}
	seedPerformanceFixture(t, ctx)

	accountRepo := &countingAccountRepository{AccountRepository: NewAccountRepository(integrationEntClient, integrationDB, nil)}
	userRepo := &countingUserRepository{UserRepository: NewUserRepository(integrationEntClient, integrationDB)}
	groupRepo := &countingGroupRepository{GroupRepository: NewGroupRepository(integrationEntClient, integrationDB)}
	loader := NewObservabilitySnapshotRepository(integrationDB)
	concurrencyRepo := NewObservabilityConcurrencyRepository(integrationDB)
	concurrency := &performanceConcurrencyReader{history: concurrencyRepo}

	end := time.Now().UTC().Truncate(time.Hour)
	start := end.Add(-6 * time.Hour)
	_, err := (&observabilityConcurrencyRepository{db: integrationDB}).QueryConcurrencyHeatmapHistory(ctx, service.ConcurrencyHistoryFilter{
		Start: start, End: end, WindowSeconds: 300,
	}, "account_id", 20)
	require.NoError(t, err, "bounded concurrency heatmap query")
	snapshotRequest := service.SnapshotRequest{
		Window: observability.Window6h, Bucket: observability.Bucket1h,
		Start: start, End: end, Scope: "performance", OptionLimit: 20,
	}

	coldSnapshotSamples := make([]time.Duration, 0, 20)
	var snapshotBody []byte
	for i := 0; i < cap(coldSnapshotSamples); i++ {
		snapshotService := service.NewObservabilitySnapshotService(loader, accountRepo, userRepo, groupRepo, concurrency, nil)
		started := time.Now()
		result, err := snapshotService.Snapshot(ctx, snapshotRequest)
		require.NoError(t, err)
		coldSnapshotSamples = append(coldSnapshotSamples, time.Since(started))
		if i == 0 {
			snapshotBody, err = json.Marshal(result.Response)
			require.NoError(t, err)
			require.True(t, result.Response.Meta.Coverage == "full")
		}
	}

	require.GreaterOrEqual(t, accountRepo.batchCalls.Load(), int64(len(coldSnapshotSamples)), "snapshot account labels must use batch loads")
	require.LessOrEqual(t, accountRepo.batchCalls.Load(), int64(len(coldSnapshotSamples)*2), "snapshot account labels must not fan out")
	require.Equal(t, int64(0), accountRepo.itemCalls.Load(), "snapshot must not issue one account query per row")
	require.GreaterOrEqual(t, userRepo.batchCalls.Load(), int64(len(coldSnapshotSamples)), "snapshot user labels must use batch loads")
	require.LessOrEqual(t, userRepo.batchCalls.Load(), int64(len(coldSnapshotSamples)*2), "snapshot user labels must not fan out")
	require.Equal(t, int64(0), userRepo.itemCalls.Load(), "snapshot must not issue one user query per row")
	require.GreaterOrEqual(t, groupRepo.batchCalls.Load(), int64(len(coldSnapshotSamples)), "snapshot group labels must use batch loads")
	require.LessOrEqual(t, groupRepo.batchCalls.Load(), int64(len(coldSnapshotSamples)*2), "snapshot group labels must not fan out")
	require.Equal(t, int64(0), groupRepo.itemCalls.Load(), "snapshot must not issue one group query per row")

	cacheService := service.NewObservabilitySnapshotService(loader, accountRepo, userRepo, groupRepo, concurrency, nil)
	_, err = cacheService.Snapshot(ctx, snapshotRequest)
	require.NoError(t, err)
	cacheHitSamples := make([]time.Duration, 0, 30)
	for i := 0; i < cap(cacheHitSamples); i++ {
		started := time.Now()
		result, hitErr := cacheService.Snapshot(ctx, snapshotRequest)
		require.NoError(t, hitErr)
		require.True(t, result.CacheHit)
		cacheHitSamples = append(cacheHitSamples, time.Since(started))
	}

	queryRequest := service.ObservabilityQueryRequest{
		Window: observability.Window6h, Bucket: observability.Bucket1h,
		Start: start, End: end, Scope: "performance",
	}
	trendSamples := measureDuration(20, func() {
		queryService := service.NewObservabilityQueryService(loader, concurrency)
		response, queryErr := queryService.Timeseries(ctx, queryRequest)
		require.NoError(t, queryErr)
		require.NotEmpty(t, response.Series)
	})
	latencySamples := measureDuration(20, func() {
		queryService := service.NewObservabilityQueryService(loader, concurrency)
		response, queryErr := queryService.LatencyTimeseries(ctx, queryRequest)
		require.NoError(t, queryErr)
		require.NotEmpty(t, response.Series)
	})
	heatmapSamples := measureDuration(20, func() {
		queryService := service.NewObservabilityQueryService(loader, concurrency)
		request := queryRequest
		request.Bucket = observability.Bucket5m
		request.GroupBy = "account_id"
		request.ValueKind = "sampled_concurrency"
		response, queryErr := queryService.Heatmap(ctx, request)
		require.NoError(t, queryErr)
		require.True(t, response.Available)
		require.NotEmpty(t, response.Cells)
	})

	pathRepo := NewObservabilityPathRepository(integrationDB)
	errorStageLoader, ok := pathRepo.(interface {
		LoadPathErrorStages(context.Context, time.Time, time.Time, []service.PathRef) (map[string]string, error)
	})
	require.True(t, ok, "path repository must expose error-stage aggregation")
	pathQuery := service.ObservabilityPathQuery{
		Start: start, End: end, Window: observability.Window6h, Bucket: observability.Bucket1h,
		Page: 1, PageSize: 50, Sort: "requests",
	}
	var pathRows []service.ObservabilityPathAggregate
	pathSamples := measureDuration(20, func() {
		var queryErr error
		pathRows, _, queryErr = pathRepo.LoadPathAggregates(ctx, pathQuery)
		require.NoError(t, queryErr)
		require.Len(t, pathRows, 50)
	})
	pathRefs := make([]service.PathRef, 0, len(pathRows))
	for _, row := range pathRows {
		if row.ErrorCount <= 0 {
			continue
		}
		pathRefs = append(pathRefs, service.PathRef{
			UserID: row.UserID, APIKeyID: row.APIKeyID, GroupID: row.GroupID,
			RouteAlias: row.RouteAlias, AccountID: row.AccountID,
		})
	}
	require.NotEmpty(t, pathRefs, "performance fixture must exercise the error-stage query")
	errorStageSamples := measureDuration(20, func() {
		stages, queryErr := errorStageLoader.LoadPathErrorStages(ctx, start, end, pathRefs)
		require.NoError(t, queryErr)
		require.Len(t, stages, len(pathRefs))
	})
	pathBody, err := json.Marshal(pathRows)
	require.NoError(t, err)
	trendBody := marshalQueryResponse(t, service.NewObservabilityQueryService(loader, concurrency), queryRequest, "trend")
	latencyBody := marshalQueryResponse(t, service.NewObservabilityQueryService(loader, concurrency), queryRequest, "latency")
	heatmapRequest := queryRequest
	heatmapRequest.Bucket = observability.Bucket5m
	heatmapRequest.GroupBy = "account_id"
	heatmapRequest.ValueKind = "sampled_concurrency"
	heatmapBody := marshalQueryResponse(t, service.NewObservabilityQueryService(loader, concurrency), heatmapRequest, "heatmap")

	coldSnapshotP95 := percentile95(coldSnapshotSamples)
	cacheHitP95 := percentile95(cacheHitSamples)
	trendP95 := percentile95(trendSamples)
	latencyP95 := percentile95(latencySamples)
	heatmapP95 := percentile95(heatmapSamples)
	pathP95 := percentile95(pathSamples)
	errorStageP95 := percentile95(errorStageSamples)
	maxBodyBytes := int64(0)
	for _, body := range [][]byte{snapshotBody, pathBody, trendBody, latencyBody, heatmapBody} {
		if int64(len(body)) > maxBodyBytes {
			maxBodyBytes = int64(len(body))
		}
	}

	t.Logf("fixture accounts=%d paths=%d", performanceAccountCount, performancePathCount)
	t.Logf("snapshot_cold_p95=%s snapshot_304_p95=%s", coldSnapshotP95, cacheHitP95)
	t.Logf("trend_p95=%s latency_p95=%s heatmap_p95=%s path_first_page_p95=%s path_error_stage_p95=%s",
		trendP95, latencyP95, heatmapP95, pathP95, errorStageP95)
	t.Logf("max_serialized_body_bytes=%d", maxBodyBytes)
	logPerformancePlans(t, ctx, start, end)

	require.LessOrEqual(t, coldSnapshotP95, 2*time.Second)
	require.LessOrEqual(t, cacheHitP95, 200*time.Millisecond)
	require.LessOrEqual(t, trendP95, 1500*time.Millisecond)
	require.LessOrEqual(t, latencyP95, 1500*time.Millisecond)
	require.LessOrEqual(t, heatmapP95, 1500*time.Millisecond)
	require.LessOrEqual(t, pathP95, time.Second)
	require.LessOrEqual(t, errorStageP95, time.Second)
	require.LessOrEqual(t, maxBodyBytes, int64(2*1024*1024))
}

func applyPerformanceDDL(t *testing.T, ctx context.Context) {
	t.Helper()
	for _, name := range []string{
		"169_create_group_model_route_accounts.sql",
		"177_create_observability_request_rollup.sql",
		"178_create_observability_concurrency_rollup.sql",
		"179_add_observability_capacity_metrics.sql",
		"180_add_concurrency_peak_ratio_sample.sql",
	} {
		body, err := os.ReadFile("../../sqlArchiving/" + name)
		require.NoError(t, err)
		_, err = integrationDB.ExecContext(ctx, string(body))
		require.NoErrorf(t, err, "apply %s", name)
	}
}

func seedPerformanceFixture(t *testing.T, ctx context.Context) {
	t.Helper()
	tx, err := integrationDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	insertPerformanceUsers(t, ctx, tx)
	insertPerformanceGroups(t, ctx, tx)
	insertPerformanceAccounts(t, ctx, tx)
	insertPerformanceAPIKeys(t, ctx, tx)
	insertPerformanceRoutes(t, ctx, tx)
	insertPerformanceRequestBuckets(t, ctx, tx)
	insertPerformanceErrorLogs(t, ctx, tx)
	insertPerformanceConcurrencyBuckets(t, ctx, tx)
	require.NoError(t, tx.Commit())
}

func insertPerformanceUsers(t *testing.T, ctx context.Context, tx *sql.Tx) {
	t.Helper()
	for start := 1; start <= 100; start += 25 {
		var values []string
		for i := start; i < start+25; i++ {
			values = append(values, fmt.Sprintf(
				"('perf-user-%03d@example.com','hash','user',0,10,'active','performance',NOW(6),NOW(6))", i,
			))
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO users
(email,password_hash,role,balance,concurrency,status,department,created_at,updated_at)
VALUES `+strings.Join(values, ","))
		require.NoError(t, err)
	}
}

func insertPerformanceGroups(t *testing.T, ctx context.Context, tx *sql.Tx) {
	t.Helper()
	for start := 1; start <= 100; start += 25 {
		var values []string
		for i := start; i < start+25; i++ {
			values = append(values, fmt.Sprintf(
				"('perf-group-%03d','anthropic','performance',1.0,0,'active','standard','{}',NOW(6),NOW(6))", i,
			))
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO `+"`groups`"+`
(name,platform,description,rate_multiplier,is_exclusive,status,subscription_type,security_check_config,created_at,updated_at)
VALUES `+strings.Join(values, ","))
		require.NoError(t, err)
	}
}

func insertPerformanceAccounts(t *testing.T, ctx context.Context, tx *sql.Tx) {
	t.Helper()
	for start := 1; start <= performanceAccountCount; start += 100 {
		var values []string
		for i := start; i < start+100; i++ {
			values = append(values, fmt.Sprintf(
				"('perf-account-%04d','anthropic','oauth','{}','{}',20,50,'active','',1,NOW(6),NOW(6))", i,
			))
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO accounts
(name,platform,type,credentials,extra,concurrency,priority,status,error_message,schedulable,created_at,updated_at)
VALUES `+strings.Join(values, ","))
		require.NoError(t, err)
	}
}

func insertPerformanceAPIKeys(t *testing.T, ctx context.Context, tx *sql.Tx) {
	t.Helper()
	for start := 1; start <= performanceAccountCount; start += 100 {
		var values []string
		for i := start; i < start+100; i++ {
			values = append(values, fmt.Sprintf(
				"(%d,'perf-key-%04d','perf-key-%04d',%d,'active',NOW(6),NOW(6))",
				((i-1)%100)+1, i, i, ((i-1)%100)+1,
			))
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO api_keys
(user_id,`+"`key`"+`,name,group_id,status,created_at,updated_at)
VALUES `+strings.Join(values, ","))
		require.NoError(t, err)
	}
}

func insertPerformanceRoutes(t *testing.T, ctx context.Context, tx *sql.Tx) {
	t.Helper()
	for start := 0; start < performancePathCount; start += 250 {
		var values []string
		for i := start; i < start+250; i++ {
			groupID := (i % 100) + 1
			accountID := (i % performanceAccountCount) + 1
			values = append(values, fmt.Sprintf(
				"(%d,'perf-route-%05d',%d,20,NOW(6),NOW(6))", groupID, i, accountID,
			))
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO group_model_route_accounts
(group_id,route_alias,account_id,max_concurrency,created_at,updated_at)
VALUES `+strings.Join(values, ","))
		require.NoError(t, err)
	}
}

func insertPerformanceRequestBuckets(t *testing.T, ctx context.Context, tx *sql.Tx) {
	t.Helper()
	end := time.Now().UTC().Truncate(time.Hour)
	start := end.Add(-6 * time.Hour)
	for batchStart := 0; batchStart < performancePathCount; batchStart += 250 {
		var values []string
		for i := batchStart; i < batchStart+250; i++ {
			userID := (i % 100) + 1
			groupID := (i % 100) + 1
			accountID := (i % performanceAccountCount) + 1
			keyID := (i % performanceAccountCount) + 1
			routeAlias := fmt.Sprintf("perf-route-%05d", i)
			bucket := start.Add(time.Duration(i%6) * time.Hour)
			hash := sha256.Sum256([]byte(fmt.Sprintf("%d:%d:%d:%s:%d", userID, keyID, groupID, routeAlias, accountID)))
			values = append(values, fmt.Sprintf(
				"(3600,'%s','%s',%d,%d,%d,'%s',%d,2,1,1,1200,1,1200,1200,1200,1,900,900,900,'%s')",
				bucket.Format("2006-01-02 15:04:05"), hex.EncodeToString(hash[:]),
				userID, keyID, groupID, routeAlias, accountID,
				time.Now().UTC().Format("2006-01-02 15:04:05.000000"),
			))
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO observability_request_bucket
(window_seconds,bucket_start,dimension_hash,user_id,api_key_id,group_id,route_alias,account_id,
  request_count,success_count,error_count,total_tokens,
 duration_sample_count,sum_duration_ms,min_duration_ms,max_duration_ms,
 ttft_sample_count,sum_ttft_ms,min_ttft_ms,max_ttft_ms,last_fact_at)
VALUES `+strings.Join(values, ","))
		require.NoError(t, err)
	}
}

func insertPerformanceErrorLogs(t *testing.T, ctx context.Context, tx *sql.Tx) {
	t.Helper()
	end := time.Now().UTC().Truncate(time.Hour)
	start := end.Add(-6 * time.Hour)
	for batchStart := 0; batchStart < performancePathCount; batchStart += 250 {
		var values []string
		for i := batchStart; i < batchStart+250; i++ {
			userID := (i % 100) + 1
			groupID := (i % 100) + 1
			accountID := (i % performanceAccountCount) + 1
			keyID := (i % performanceAccountCount) + 1
			routeAlias := fmt.Sprintf("perf-route-%05d", i)
			createdAt := start.Add(time.Duration(i%6) * time.Hour)
			phase := []string{"request", "auth", "routing", "upstream", "network", "internal"}[i%6]
			values = append(values, fmt.Sprintf(
				"('perf-request-%05d',%d,%d,%d,500,'%s',%d,'%s','perf_error','P2',0,0,0,0,'perf error','%s')",
				i, userID, keyID, groupID, routeAlias, accountID, phase,
				createdAt.Format("2006-01-02 15:04:05"),
			))
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO ops_error_logs
(request_id,user_id,api_key_id,group_id,status_code,route_alias,account_id,error_phase,
 error_type,severity,stream,is_business_limited,is_count_tokens,resolved,error_message,created_at)
VALUES `+strings.Join(values, ","))
		require.NoError(t, err)
	}
}

func insertPerformanceConcurrencyBuckets(t *testing.T, ctx context.Context, tx *sql.Tx) {
	t.Helper()
	end := time.Now().UTC().Truncate(5 * time.Minute)
	start := end.Add(-6 * time.Hour)
	for batchStart := 0; batchStart < 7200; batchStart += 250 {
		var values []string
		for i := batchStart; i < batchStart+250; i++ {
			accountID := (i % 100) + 1
			groupID := (i % 100) + 1
			bucket := start.Add(time.Duration(i%72) * 5 * time.Minute)
			hash := sha256.Sum256([]byte(fmt.Sprintf("account:%d:%d", accountID, bucket.Unix())))
			actual := 5 + (i % 15)
			values = append(values, fmt.Sprintf(
				"(300,'%s','account','%s',%d,%d,NULL,%d,%d,5,20,2,100,20,%d,20,'%s','%s','%s')",
				bucket.Format("2006-01-02 15:04:05"), hex.EncodeToString(hash[:]), accountID, groupID,
				actual, actual*5, actual, bucket.Format("2006-01-02 15:04:05"),
				bucket.Format("2006-01-02 15:04:05"), bucket.Format("2006-01-02 15:04:05"),
			))
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO observability_concurrency_bucket
(window_seconds,bucket_start,sample_kind,dimension_hash,account_id,group_id,route_alias,
 current_peak,current_sum,sample_count,effective_limit,waiting_peak,wait_sum_ms,wait_sample_count,
 peak_ratio_actual,peak_ratio_limit,peak_ratio_sample_at,first_sampled_at,last_sampled_at)
VALUES `+strings.Join(values, ","))
		require.NoError(t, err)
	}
}

type countingAccountRepository struct {
	service.AccountRepository
	batchCalls atomic.Int64
	itemCalls  atomic.Int64
}

func (r *countingAccountRepository) GetByIDs(ctx context.Context, ids []int64) ([]*service.Account, error) {
	r.batchCalls.Add(1)
	return r.AccountRepository.GetByIDs(ctx, ids)
}

func (r *countingAccountRepository) GetByID(ctx context.Context, id int64) (*service.Account, error) {
	r.itemCalls.Add(1)
	return r.AccountRepository.GetByID(ctx, id)
}

type countingUserRepository struct {
	service.UserRepository
	batchCalls atomic.Int64
	itemCalls  atomic.Int64
}

func (r *countingUserRepository) GetByIDs(ctx context.Context, ids []int64) ([]service.User, error) {
	r.batchCalls.Add(1)
	return r.UserRepository.(interface {
		GetByIDs(context.Context, []int64) ([]service.User, error)
	}).GetByIDs(ctx, ids)
}

func (r *countingUserRepository) GetByID(ctx context.Context, id int64) (*service.User, error) {
	r.itemCalls.Add(1)
	return r.UserRepository.GetByID(ctx, id)
}

type countingGroupRepository struct {
	service.GroupRepository
	batchCalls atomic.Int64
	itemCalls  atomic.Int64
}

func (r *countingGroupRepository) GetByIDsLite(ctx context.Context, ids []int64) ([]service.Group, error) {
	r.batchCalls.Add(1)
	return r.GroupRepository.(interface {
		GetByIDsLite(context.Context, []int64) ([]service.Group, error)
	}).GetByIDsLite(ctx, ids)
}

func (r *countingGroupRepository) GetByID(ctx context.Context, id int64) (*service.Group, error) {
	r.itemCalls.Add(1)
	return r.GroupRepository.GetByID(ctx, id)
}

type performanceConcurrencyReader struct {
	history service.ConcurrencyHistoryRepository
}

func (r *performanceConcurrencyReader) CollectCurrent(_ context.Context, sampledAt time.Time) ([]service.ConcurrencyCurrentValue, error) {
	out := make([]service.ConcurrencyCurrentValue, 0, performanceAccountCount+20)
	for i := 1; i <= performanceAccountCount; i++ {
		out = append(out, service.ConcurrencyCurrentValue{
			Kind: service.ConcurrencySampleKindAccount, AccountID: int64(i), GroupID: int64((i % 100) + 1),
			Current: (i % 20), Limit: 20, Waiting: i % 3, SampledAt: sampledAt, Available: true,
		})
	}
	for i := 0; i < 20; i++ {
		out = append(out, service.ConcurrencyCurrentValue{
			Kind: service.ConcurrencySampleKindRoute, AccountID: int64(i + 1), GroupID: int64(i + 1),
			RouteAlias: fmt.Sprintf("perf-route-%03d", i), Current: i, Limit: 20, Waiting: i % 2,
			SampledAt: sampledAt, Available: true,
		})
	}
	return out, nil
}

func (r *performanceConcurrencyReader) QueryHistory(ctx context.Context, filter service.ConcurrencyHistoryFilter) (service.ConcurrencyHistoryResult, error) {
	rows, err := r.history.QueryConcurrencyHistory(ctx, filter)
	if err != nil {
		return service.ConcurrencyHistoryResult{}, err
	}
	return service.ConcurrencyHistoryResult{Rows: rows, Available: true, QualityStatus: "healthy"}, nil
}

func (r *performanceConcurrencyReader) QueryConcurrencyHeatmapHistory(
	ctx context.Context,
	filter service.ConcurrencyHistoryFilter,
	groupBy string,
	limit int,
) ([]service.ConcurrencyHistoryRow, error) {
	bounded, ok := r.history.(interface {
		QueryConcurrencyHeatmapHistory(context.Context, service.ConcurrencyHistoryFilter, string, int) ([]service.ConcurrencyHistoryRow, error)
	})
	if !ok {
		return r.history.QueryConcurrencyHistory(ctx, filter)
	}
	return bounded.QueryConcurrencyHeatmapHistory(ctx, filter, groupBy, limit)
}

func measureDuration(iterations int, fn func()) []time.Duration {
	out := make([]time.Duration, 0, iterations)
	for i := 0; i < iterations; i++ {
		started := time.Now()
		fn()
		out = append(out, time.Since(started))
	}
	return out
}

func marshalQueryResponse(
	t *testing.T,
	queryService *service.ObservabilityQueryService,
	request service.ObservabilityQueryRequest,
	kind string,
) []byte {
	t.Helper()
	ctx := context.Background()
	var response any
	var err error
	switch kind {
	case "trend":
		response, err = queryService.Timeseries(ctx, request)
	case "latency":
		response, err = queryService.LatencyTimeseries(ctx, request)
	case "heatmap":
		response, err = queryService.Heatmap(ctx, request)
	default:
		t.Fatalf("unsupported response kind %q", kind)
	}
	require.NoError(t, err)
	body, err := json.Marshal(response)
	require.NoError(t, err)
	return body
}

func percentile95(values []time.Duration) time.Duration {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]time.Duration(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	index := (95*len(sorted) + 99) / 100
	if index < 1 {
		index = 1
	}
	return sorted[index-1]
}

func logPerformancePlans(t *testing.T, ctx context.Context, start, end time.Time) {
	t.Helper()
	queries := []string{
		`EXPLAIN FORMAT=JSON SELECT SUM(request_count), SUM(total_tokens)
FROM observability_request_bucket
WHERE window_seconds=3600 AND bucket_start>=? AND bucket_start<?`,
		`EXPLAIN FORMAT=JSON SELECT user_id,api_key_id,group_id,route_alias,account_id,SUM(request_count)
FROM observability_request_bucket
WHERE window_seconds=3600 AND bucket_start>=? AND bucket_start<?
GROUP BY user_id,api_key_id,group_id,route_alias,account_id
ORDER BY SUM(request_count) DESC LIMIT 50`,
		`EXPLAIN FORMAT=JSON SELECT account_id,group_id,route_alias,current_peak,effective_limit
FROM observability_concurrency_bucket
WHERE window_seconds=300 AND bucket_start>=? AND bucket_start<?
ORDER BY bucket_start LIMIT 5000`,
	}
	for _, query := range queries {
		var plan string
		require.NoError(t, integrationDB.QueryRowContext(ctx, query, start, end).Scan(&plan))
		require.Contains(t, plan, "observability_")
		t.Logf("query_plan=%s", plan)
	}
}
