package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTokenStatisticsFaultEventNames(t *testing.T) {
	server, client := newTokenStatisticsAccumulatorTestClient(t)
	increment := tokenStatisticsTestIncrement()
	server.Close()
	err := NewRedisTokenStatisticsAccumulator(client, 2).Accumulate(context.Background(), increment)
	require.ErrorContains(t, err, "token_statistics.redis_increment_failed")
	require.ErrorContains(t, err, "stage=redis_pipeline_exec")

	server, client = newTokenStatisticsAccumulatorTestClient(t)
	date := time.Now()
	key, _ := TokenStatisticsKey(TokenStatisticsModel, date)
	field, _ := EncodeModelTokenStatisticsField("gpt-5")
	server.HSet(key, field, "10")
	target := &absoluteUsageStub{failures: 3}
	err = NewTokenStatisticsSyncEngine(client, target, 10, 10, 1).SyncDate(context.Background(), date)
	require.ErrorContains(t, err, "token_statistics.mysql_sync_failed")
	require.ErrorContains(t, err, "max_retries=1")
	require.ErrorContains(t, err, "mysql 1205")
}
