package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestSchedulerCacheBatchAccountsUsesExistingAccountKeys(t *testing.T) {
	mini := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	cache := NewSchedulerCache(rdb)
	batch, ok := cache.(service.SchedulerAccountBatchCache)
	require.True(t, ok)

	err := batch.SetAccounts(context.Background(), []*service.Account{
		{ID: 11, Name: "eleven", Credentials: map[string]any{"model_mapping": map[string]any{"upstream-a": "alias"}}},
		{ID: 12, Name: "twelve"},
	})
	require.NoError(t, err)
	require.True(t, mini.Exists("sched:acc:11"))
	require.True(t, mini.Exists("sched:acc:12"))

	accounts, err := batch.GetAccounts(context.Background(), []int64{12, 13, 11, 12})
	require.NoError(t, err)
	require.Len(t, accounts, 2)
	require.Equal(t, "eleven", accounts[11].Name)
	require.Equal(t, "twelve", accounts[12].Name)
	require.Nil(t, accounts[13])
}
