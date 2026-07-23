package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type tokenStatisticsSyncerStub struct {
	mu      sync.Mutex
	dates   []time.Time
	entered chan struct{}
	release chan struct{}
}

func (s *tokenStatisticsSyncerStub) SyncDate(ctx context.Context, date time.Time) error {
	s.mu.Lock()
	s.dates = append(s.dates, date)
	first := len(s.dates) == 1
	s.mu.Unlock()
	if first && s.entered != nil {
		close(s.entered)
		select {
		case <-s.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func newSchedulerRedis(t *testing.T) *redis.Client {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestTokenStatisticsSchedulerDayBoundary(t *testing.T) {
	client := newSchedulerRedis(t)
	syncer := &tokenStatisticsSyncerStub{}
	scheduler := NewTokenStatisticsScheduler(client, syncer, time.Minute)
	now := time.Date(2026, 7, 14, 0, 0, 1, 0, time.FixedZone("Asia/Shanghai", 8*3600))
	scheduler.now = func() time.Time { return now }
	require.NoError(t, scheduler.SyncOnce(context.Background()))
	require.Len(t, syncer.dates, 2)
	require.Equal(t, "2026-07-14", syncer.dates[0].Format(time.DateOnly))
	require.Equal(t, "2026-07-13", syncer.dates[1].Format(time.DateOnly))
}

func TestTokenStatisticsSchedulerLock(t *testing.T) {
	client := newSchedulerRedis(t)
	firstSyncer := &tokenStatisticsSyncerStub{entered: make(chan struct{}), release: make(chan struct{})}
	first := NewTokenStatisticsScheduler(client, firstSyncer, time.Minute)
	secondSyncer := &tokenStatisticsSyncerStub{}
	second := NewTokenStatisticsScheduler(client, secondSyncer, time.Minute)
	done := make(chan error, 1)
	go func() { done <- first.SyncOnce(context.Background()) }()
	<-firstSyncer.entered
	require.NoError(t, second.SyncOnce(context.Background()))
	require.Empty(t, secondSyncer.dates)
	close(firstSyncer.release)
	require.NoError(t, <-done)
}

func TestTokenStatisticsSchedulerLifecycle(t *testing.T) {
	client := newSchedulerRedis(t)
	syncer := &tokenStatisticsSyncerStub{}
	scheduler := NewTokenStatisticsScheduler(client, syncer, 5*time.Millisecond)
	require.NoError(t, scheduler.Start(context.Background()))
	time.Sleep(20 * time.Millisecond)
	scheduler.Stop()
	scheduler.Stop()
	syncer.mu.Lock()
	calls := len(syncer.dates)
	syncer.mu.Unlock()
	require.GreaterOrEqual(t, calls, 2)
}
