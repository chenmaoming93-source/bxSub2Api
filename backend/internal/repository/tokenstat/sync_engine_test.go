package tokenstat

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	domain "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
)

type aggregateSinkStub struct {
	mu       sync.Mutex
	rows     []Aggregate
	err      error
	callback func()
}

func (s *aggregateSinkStub) UpsertAggregate(_ context.Context, aggregate Aggregate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.callback != nil {
		callback := s.callback
		s.callback = nil
		callback()
	}
	if s.err != nil {
		return s.err
	}
	s.rows = append(s.rows, aggregate)
	return nil
}

func TestDynamicTokenSyncPersistsAbsoluteSnapshotAndPreservesConcurrentDirty(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	accumulator := NewRedisAccumulator(client, 16, 7)
	location, _ := time.LoadLocation("Asia/Shanghai")
	period := domain.NaturalPeriods(time.Now(), location)[0]
	var hash [16]byte
	hash[0] = 2
	operation := domain.AccountingOperation{
		Period: period, ProjectionID: 8, DimensionHash: hash,
		DimensionValues: map[string]any{"user_id": int64(42), "department": "研发部"}, MetricCode: domain.MetricTotalTokens, Delta: 10,
	}
	require.NoError(t, accumulator.Add(context.Background(), []domain.AccountingOperation{operation}))

	sink := &aggregateSinkStub{}
	sink.callback = func() {
		operation.Delta = 5
		require.NoError(t, accumulator.Add(context.Background(), []domain.AccountingOperation{operation}))
	}
	engine := NewSyncEngine(client, sink, 10)
	require.NoError(t, engine.Sync(context.Background()))
	require.Len(t, sink.rows, 1)
	require.Equal(t, int64(10), sink.rows[0].MetricValue)
	require.NotNil(t, sink.rows[0].Department)
	require.Equal(t, "研发部", *sink.rows[0].Department)
	require.Equal(t, int64(1), sink.rows[0].SourceVersion)
	require.True(t, mini.Exists(dynamicDirtyKey), "write during sync must remain in current dirty set")

	require.NoError(t, engine.Sync(context.Background()))
	require.Len(t, sink.rows, 2)
	require.Equal(t, int64(15), sink.rows[1].MetricValue)
	require.Equal(t, int64(2), sink.rows[1].SourceVersion)
}

func TestDynamicTokenSyncFailureRequeuesForSafeRetry(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	accumulator := NewRedisAccumulator(client, 16, 7)
	location, _ := time.LoadLocation("Asia/Shanghai")
	period := domain.NaturalPeriods(time.Now(), location)[0]
	require.NoError(t, accumulator.Add(context.Background(), []domain.AccountingOperation{{
		Period: period, ProjectionID: 1, MetricCode: domain.MetricTotalTokens, Delta: 1,
		DimensionValues: map[string]any{"user_id": int64(1)},
	}}))
	sink := &aggregateSinkStub{err: errors.New("mysql unavailable")}
	engine := NewSyncEngine(client, sink, 10)
	require.Error(t, engine.Sync(context.Background()))
	require.True(t, mini.Exists(dynamicDirtyKey))
	sink.err = nil
	require.NoError(t, engine.Sync(context.Background()))
	require.Len(t, sink.rows, 1)
	require.Equal(t, uint64(1), engine.Stats().Failures)
}

func TestDynamicTokenSyncStartImmediatelyPersistsCurrentPeriods(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	accumulator := NewRedisAccumulator(client, 16, 7)
	location, _ := time.LoadLocation("Asia/Shanghai")
	periods := domain.NaturalPeriods(time.Now(), location)
	var hash [16]byte
	for _, period := range periods {
		require.NoError(t, accumulator.Add(context.Background(), []domain.AccountingOperation{{
			Period: period, ProjectionID: 1, MetricCode: domain.MetricTotalTokens, Delta: 1,
			DimensionHash: hash, DimensionValues: map[string]any{"user_id": int64(1)},
		}}))
	}

	sink := &aggregateSinkStub{}
	engine := NewSyncEngine(client, sink, 10)
	cancel := engine.Start(context.Background(), time.Hour)
	t.Cleanup(cancel)

	require.Eventually(t, func() bool {
		sink.mu.Lock()
		defer sink.mu.Unlock()
		return len(sink.rows) == 3
	}, time.Second, 10*time.Millisecond)
	require.False(t, mini.Exists(dynamicDirtyKey))
}
