//go:build integration

package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service/observability"
	"github.com/stretchr/testify/require"
)

func TestObservabilitySnapshotRollupJoinsCanonicalDimensions(t *testing.T) {
	require.NoError(t, timezone.Init("Asia/Shanghai"))
	t.Cleanup(func() { require.NoError(t, timezone.Init("UTC")) })
	ddl, err := os.ReadFile("../../sqlArchiving/177_create_observability_request_rollup.sql")
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(context.Background(), string(ddl))
	require.NoError(t, err)
	capacityDDL, err := os.ReadFile("../../sqlArchiving/179_add_observability_capacity_metrics.sql")
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(context.Background(), string(capacityDDL))
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(context.Background(), `
INSERT INTO observability_request_bucket
(window_seconds,bucket_start,dimension_hash,user_id,api_key_id,group_id,route_alias,account_id,
 request_count,success_count,error_count,legacy_inferred_count,total_tokens,
 duration_sample_count,sum_duration_ms,min_duration_ms,max_duration_ms,
 duration_le_100,duration_le_250,duration_le_500,duration_le_1000,duration_le_2000,duration_le_5000,duration_le_10000,duration_gt_10000,
 last_fact_at)
VALUES
(300,'2026-11-11 02:00:00','it-snapshot-1',1,11,7,'coding',101,2,2,0,0,100,1,400,400,400,0,0,1,0,0,0,0,0,'2026-11-11 02:00:30'),
(300,'2026-11-11 02:00:00','it-snapshot-2',2,22,7,'coding',102,3,3,0,0,300,1,900,900,900,0,0,0,1,0,0,0,0,'2026-11-11 02:00:40');`)
	require.NoError(t, err)
	repo := NewObservabilitySnapshotRepository(integrationDB)
	got, err := repo.LoadSnapshotRollup(context.Background(),
		time.Date(2026, 11, 11, 1, 0, 0, 0, time.UTC),
		time.Date(2026, 11, 11, 3, 0, 0, 0, time.UTC),
		observability.Bucket5m,
		observability.Filter{},
	)
	require.NoError(t, err)
	require.Equal(t, int64(5), got.Totals.RequestCount)
	require.Equal(t, int64(400), got.Totals.TotalTokens)
	require.Len(t, got.Users, 2)
	require.Len(t, got.Groups, 1)
	require.Len(t, got.Aliases, 1)
	require.Len(t, got.Accounts, 2)

	series, err := repo.LoadSnapshotSeries(context.Background(),
		time.Date(2026, 11, 11, 1, 0, 0, 0, time.UTC),
		time.Date(2026, 11, 11, 3, 0, 0, 0, time.UTC),
		observability.Bucket5m,
		observability.Filter{},
		"group_id", []int64{7}, nil,
	)
	require.NoError(t, err)
	require.Len(t, series, 1)
	require.Equal(t, int64(5), series[0].Totals.RequestCount)
}
