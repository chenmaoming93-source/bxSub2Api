//go:build integration

package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestObservabilityAlertMetricReader(t *testing.T) {
	for _, path := range []string{
		"../../sqlArchiving/177_create_observability_request_rollup.sql",
		"../../sqlArchiving/178_create_observability_concurrency_rollup.sql",
		"../../sqlArchiving/179_add_observability_capacity_metrics.sql",
		"../../sqlArchiving/180_add_concurrency_peak_ratio_sample.sql",
	} {
		ddl, err := os.ReadFile(path)
		require.NoError(t, err)
		_, err = integrationDB.ExecContext(context.Background(), string(ddl))
		require.NoError(t, err)
	}
	now := time.Now().UTC().Truncate(time.Minute)
	bucket := utcBucketForDB(now, timezone.Location())
	_, err := integrationDB.ExecContext(context.Background(), `
INSERT INTO observability_request_bucket
(window_seconds,bucket_start,dimension_hash,user_id,api_key_id,group_id,route_alias,account_id,
 request_count,success_count,error_count,business_limited_count,sla_error_count,total_tokens,
 ttft_sample_count,sum_ttft_ms,min_ttft_ms,max_ttft_ms,
 ttft_le_100,ttft_le_250,ttft_le_500,ttft_le_1000,ttft_le_2000,ttft_le_5000,ttft_le_10000,ttft_gt_10000,last_fact_at)
VALUES (60,?,'it-alert-metric',1,11,7,'gpt-5',101,100,97,3,1,2,1000,100,150000,100,4000,
       0,0,10,40,30,15,5,0,?)`, bucket, now)
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(context.Background(), `
INSERT INTO observability_concurrency_bucket
(window_seconds,bucket_start,sample_kind,dimension_hash,account_id,group_id,route_alias,current_peak,current_sum,sample_count,effective_limit,waiting_peak,first_sampled_at,last_sampled_at)
VALUES (60,?,'route','it-alert-concurrency',101,7,'gpt-5',18,45,3,20,4,?,?)`,
		bucket, now.Add(-30*time.Second), now)
	require.NoError(t, err)

	reader, ok := NewOpsRepository(integrationDB).(service.OpsObservabilityAlertMetricReader)
	require.True(t, ok)
	groupID, accountID := int64(7), int64(101)
	scope := service.OpsAlertRuleScope{GroupID: &groupID, RouteAlias: "gpt-5", AccountID: &accountID}
	failure, ok := reader.QueryObservabilityAlertMetric(context.Background(), "observability_client_failure_rate", scope, now.Add(-time.Minute), now.Add(time.Minute))
	require.True(t, ok)
	require.InDelta(t, 3, failure.Value, 0.001)
	require.Equal(t, int64(100), failure.SampleCount)

	ttft, ok := reader.QueryObservabilityAlertMetric(context.Background(), "observability_ttft_p95_ms", scope, now.Add(-time.Minute), now.Add(time.Minute))
	require.True(t, ok)
	require.InDelta(t, 5000, ttft.Value, 2500)

	saturation, ok := reader.QueryObservabilityAlertMetric(context.Background(), "observability_concurrency_saturation", scope, now.Add(-time.Minute), now.Add(time.Minute))
	require.True(t, ok)
	require.InDelta(t, 90, saturation.Value, 0.001)
}
