//go:build integration

package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/service/observability"
	"github.com/stretchr/testify/require"
)

func TestObservabilityPathAggregationStableOrderAndTotal(t *testing.T) {
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
(window_seconds,bucket_start,dimension_hash,user_id,api_key_id,group_id,route_alias,account_id,request_count,success_count,error_count,total_tokens,last_fact_at)
VALUES
(300,'2026-11-11 02:10:00','it-path-1',1,11,7,'gpt-5',101,5,5,0,500,'2026-11-11 02:10:30'),
(300,'2026-11-11 02:10:00','it-path-2',2,22,8,'coding',102,2,1,1,900,'2026-11-11 02:10:40');`)
	require.NoError(t, err)
	repo := NewObservabilityPathRepository(integrationDB)
	rows, total, err := repo.LoadPathAggregates(context.Background(), service.ObservabilityPathQuery{
		Start:  time.Date(2026, 11, 11, 1, 0, 0, 0, time.UTC),
		End:    time.Date(2026, 11, 11, 3, 0, 0, 0, time.UTC),
		Window: observability.Window1h, Bucket: observability.Bucket5m,
		Page: 1, PageSize: 20, Sort: "requests",
	})
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, rows, 2)
	require.Equal(t, int64(11), rows[0].APIKeyID)
	require.Equal(t, "gpt-5", rows[0].RouteAlias)
	require.Equal(t, int64(5), rows[0].RequestCount)
}
