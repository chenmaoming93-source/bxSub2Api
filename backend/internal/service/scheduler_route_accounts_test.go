package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type routeBatchCacheStub struct {
	SchedulerCache
	accounts   map[int64]*Account
	readIDs    []int64
	writtenIDs []int64
	writeErr   error
}

func (c *routeBatchCacheStub) GetAccounts(_ context.Context, ids []int64) (map[int64]*Account, error) {
	c.readIDs = append([]int64(nil), ids...)
	result := make(map[int64]*Account)
	for _, id := range ids {
		if account := c.accounts[id]; account != nil {
			result[id] = account
		}
	}
	return result, nil
}

func (c *routeBatchCacheStub) SetAccounts(_ context.Context, accounts []*Account) error {
	for _, account := range accounts {
		c.writtenIDs = append(c.writtenIDs, account.ID)
		if c.writeErr == nil {
			c.accounts[account.ID] = account
		}
	}
	return c.writeErr
}

type routeBatchRepoStub struct {
	AccountRepository
	accounts map[int64]*Account
	calls    int
	readIDs  []int64
}

type routeWarmupGroupRepoStub struct {
	GroupRepository
	groups []Group
	err    error
}

func (r *routeWarmupGroupRepoStub) ListActive(context.Context) ([]Group, error) {
	return r.groups, r.err
}

func (r *routeBatchRepoStub) GetByIDs(_ context.Context, ids []int64) ([]*Account, error) {
	r.calls++
	r.readIDs = append([]int64(nil), ids...)
	result := make([]*Account, 0, len(ids))
	for _, id := range ids {
		if account := r.accounts[id]; account != nil {
			result = append(result, account)
		}
	}
	return result, nil
}

func TestRouteAccountBatchCacheFullHitSkipsDatabase(t *testing.T) {
	cache := &routeBatchCacheStub{accounts: map[int64]*Account{1: {ID: 1}, 2: {ID: 2}}}
	repo := &routeBatchRepoStub{}
	svc := NewSchedulerSnapshotService(cache, nil, repo, nil, nil)
	accounts, err := svc.GetAccounts(context.Background(), []int64{1, 2, 1})
	require.NoError(t, err)
	require.Len(t, accounts, 2)
	require.Equal(t, 0, repo.calls)
	require.Equal(t, []int64{1, 2}, cache.readIDs)
}

func TestRouteAccountBatchCachePartialMissUsesOneFallbackAndWritesBack(t *testing.T) {
	cache := &routeBatchCacheStub{accounts: map[int64]*Account{1: {ID: 1}}}
	repo := &routeBatchRepoStub{accounts: map[int64]*Account{2: {ID: 2}, 3: {ID: 3}}}
	svc := NewSchedulerSnapshotService(cache, nil, repo, nil, nil)

	accounts, err := svc.GetAccounts(context.Background(), []int64{1, 2, 3})
	require.NoError(t, err)
	require.Len(t, accounts, 3)
	require.Equal(t, 1, repo.calls)
	require.Equal(t, []int64{2, 3}, repo.readIDs)
	require.Equal(t, []int64{2, 3}, cache.writtenIDs)

	_, err = svc.GetAccounts(context.Background(), []int64{1, 2, 3})
	require.NoError(t, err)
	require.Equal(t, 1, repo.calls, "warm read must not query the database again")
}

func TestRouteAccountBatchCacheWriteFailureDoesNotFailCurrentRead(t *testing.T) {
	cache := &routeBatchCacheStub{accounts: map[int64]*Account{}, writeErr: errors.New("redis unavailable")}
	repo := &routeBatchRepoStub{accounts: map[int64]*Account{7: {ID: 7}}}
	svc := NewSchedulerSnapshotService(cache, nil, repo, nil, nil)
	accounts, err := svc.GetAccounts(context.Background(), []int64{7})
	require.NoError(t, err)
	require.Equal(t, int64(7), accounts[7].ID)
}

func TestRouteAccountBatchFallbackLimiterRemainsEffective(t *testing.T) {
	svc := NewSchedulerSnapshotService(&routeBatchCacheStub{accounts: map[int64]*Account{}}, nil, &routeBatchRepoStub{}, nil, nil)
	svc.fallbackLimit = &fallbackLimiter{maxQPS: 1, window: time.Now(), count: 1}
	_, err := svc.GetAccounts(context.Background(), []int64{9})
	require.ErrorIs(t, err, ErrSchedulerFallbackLimited)
}

func TestRouteAccountWarmupLoadsPureRoutingAccountsInBoundedBatches(t *testing.T) {
	ids := make([]int64, 0, routeAccountWarmupBatchSize+2)
	accounts := make(map[int64]*Account)
	for id := int64(1); id <= routeAccountWarmupBatchSize+2; id++ {
		ids = append(ids, id)
		accounts[id] = &Account{ID: id, Status: StatusActive, Schedulable: true}
	}
	cache := &routeBatchCacheStub{accounts: map[int64]*Account{}}
	repo := &routeBatchRepoStub{accounts: accounts}
	groups := &routeWarmupGroupRepoStub{groups: []Group{
		{ID: 1, ModelRoutingEnabled: true, ModelRouting: map[string]any{"coding": []any{map[string]any{"account_ids": ids, "priority": 0}}}},
		{ID: 2, ModelRoutingEnabled: true, ModelRouting: map[string]any{"broken": nil}},
	}}
	svc := NewSchedulerSnapshotService(cache, nil, repo, groups, nil)
	svc.warmRouteAccounts(context.Background())

	require.Equal(t, 2, repo.calls)
	require.Len(t, cache.accounts, routeAccountWarmupBatchSize+2)
	require.Equal(t, int64(routeAccountWarmupBatchSize+2), cache.accounts[routeAccountWarmupBatchSize+2].ID)
}
