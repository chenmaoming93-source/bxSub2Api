package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const tokenStatisticsSyncLockKey = "sub2api:token_stats:sync_lock"

type TokenStatisticsScheduler struct {
	rdb        *redis.Client
	syncer     TokenStatisticsDateSyncer
	interval   time.Duration
	lockTTL    time.Duration
	instanceID string
	now        func() time.Time
	cancel     context.CancelFunc
	done       chan struct{}
	mu         sync.Mutex
}

func NewTokenStatisticsScheduler(rdb *redis.Client, syncer TokenStatisticsDateSyncer, interval time.Duration) *TokenStatisticsScheduler {
	lockTTL := interval * 3
	if lockTTL < 30*time.Second {
		lockTTL = 30 * time.Second
	}
	return &TokenStatisticsScheduler{rdb: rdb, syncer: syncer, interval: interval, lockTTL: lockTTL, instanceID: uuid.NewString(), now: time.Now}
}

func (s *TokenStatisticsScheduler) Start(ctx context.Context) error {
	if s == nil || s.rdb == nil || s.syncer == nil || s.interval <= 0 {
		return fmt.Errorf("token statistics scheduler: invalid configuration")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.done = make(chan struct{})
	go s.run(runCtx)
	return nil
}

func (s *TokenStatisticsScheduler) run(ctx context.Context) {
	defer close(s.done)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.SyncOnce(ctx)
		}
	}
}

func (s *TokenStatisticsScheduler) Stop() {
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.cancel = nil
	s.done = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
		<-done
	}
}

func (s *TokenStatisticsScheduler) SyncOnce(ctx context.Context) error {
	acquired, err := s.rdb.SetNX(ctx, tokenStatisticsSyncLockKey, s.instanceID, s.lockTTL).Result()
	if err != nil {
		return fmt.Errorf("token_statistics.sync_lock_failed stage=redis_set_nx key=%s ttl=%s: %w", tokenStatisticsSyncLockKey, s.lockTTL, err)
	}
	if !acquired {
		return nil
	}
	defer func() {
		_, _ = s.rdb.Eval(context.Background(), `if redis.call("GET", KEYS[1]) == ARGV[1] then return redis.call("DEL", KEYS[1]) end return 0`, []string{tokenStatisticsSyncLockKey}, s.instanceID).Result()
	}()
	now := s.now()
	if err := s.syncer.SyncDate(ctx, now); err != nil {
		return fmt.Errorf("token statistics scheduler stage=sync_current_day date=%s: %w", now.Format(time.DateOnly), err)
	}
	previous := now.AddDate(0, 0, -1)
	if err := s.syncer.SyncDate(ctx, previous); err != nil {
		return fmt.Errorf("token statistics scheduler stage=sync_previous_day date=%s: %w", previous.Format(time.DateOnly), err)
	}
	return nil
}
