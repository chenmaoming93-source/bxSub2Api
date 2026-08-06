package tokenstat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	domain "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
	"github.com/redis/go-redis/v9"
)

const (
	PeriodStateOpen      = "OPEN"
	PeriodStateClosing   = "CLOSING"
	PeriodStateFinalSync = "FINAL_SYNC"
	PeriodStatePersisted = "PERSISTED"
	PeriodStateDeleted   = "DELETED"
)

type PeriodStateWriter interface {
	SetPeriodState(ctx context.Context, period domain.Period, state, lastError string) error
}

type PeriodVersionVerifier interface {
	VerifyPeriodVersions(ctx context.Context, period domain.Period, redisVersions map[string]int64) error
}

type PendingPeriodEvents interface {
	HasPendingBefore(end time.Time) bool
}

type PeriodSyncer interface {
	Sync(ctx context.Context) error
}

type PeriodFinalizer struct {
	redis    *redis.Client
	states   PeriodStateWriter
	verifier PeriodVersionVerifier
	pending  PendingPeriodEvents
	syncer   PeriodSyncer
}

func NewPeriodFinalizer(client *redis.Client, states PeriodStateWriter, verifier PeriodVersionVerifier, pending PendingPeriodEvents, syncer PeriodSyncer) *PeriodFinalizer {
	return &PeriodFinalizer{redis: client, states: states, verifier: verifier, pending: pending, syncer: syncer}
}

func (f *PeriodFinalizer) Start(ctx context.Context, interval time.Duration, location *time.Location) context.CancelFunc {
	runCtx, cancel := context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-runCtx.Done():
				return
			case now := <-ticker.C:
				current := domain.NaturalPeriods(now, location)
				previous := []domain.Period{
					{Type: domain.PeriodDay, Start: current[0].Start.AddDate(0, 0, -1), End: current[0].Start},
					{Type: domain.PeriodWeek, Start: current[1].Start.AddDate(0, 0, -7), End: current[1].Start},
					{Type: domain.PeriodMonth, Start: current[2].Start.AddDate(0, -1, 0), End: current[2].Start},
				}
				for _, period := range previous {
					_ = f.Finalize(runCtx, period, now)
				}
			}
		}
	}()
	return cancel
}

func (f *PeriodFinalizer) Finalize(ctx context.Context, period domain.Period, now time.Time) (resultErr error) {
	defer func() { domain.RecordFinalization(resultErr != nil) }()
	if !now.After(period.End) {
		return fmt.Errorf("period has not ended")
	}
	exists, err := f.hasPeriodKeys(ctx, period)
	if err != nil {
		return err
	}
	if !exists {
		// An empty natural period has nothing to persist or clean up. In
		// particular, do not create misleading historical period-state rows
		// when the feature is first enabled.
		return nil
	}
	if err := f.states.SetPeriodState(ctx, period, PeriodStateClosing, ""); err != nil {
		return err
	}
	if f.pending != nil && f.pending.HasPendingBefore(period.End) {
		return fmt.Errorf("period still has pending usage events")
	}
	if err := f.states.SetPeriodState(ctx, period, PeriodStateFinalSync, ""); err != nil {
		return err
	}
	if err := f.syncer.Sync(ctx); err != nil {
		_ = f.states.SetPeriodState(ctx, period, PeriodStateFinalSync, err.Error())
		return err
	}
	dirty, err := f.hasDirtyPeriod(ctx, period)
	if err != nil {
		return err
	}
	if dirty {
		return fmt.Errorf("period remains dirty after final sync")
	}
	keys, versions, err := f.periodVersions(ctx, period)
	if err != nil {
		return err
	}
	if err := f.verifier.VerifyPeriodVersions(ctx, period, versions); err != nil {
		return err
	}
	if err := f.states.SetPeriodState(ctx, period, PeriodStatePersisted, ""); err != nil {
		return err
	}
	if len(keys) > 0 {
		countKeys := make([]string, 0, len(keys))
		for _, versionKey := range keys {
			countKeys = append(countKeys, strings.Replace(versionKey, dynamicVersionPrefix, dynamicCountPrefix, 1))
		}
		keys = append(keys, countKeys...)
		if err := f.redis.Unlink(ctx, keys...).Err(); err != nil {
			return err
		}
	}
	return f.states.SetPeriodState(ctx, period, PeriodStateDeleted, "")
}

func (f *PeriodFinalizer) hasPeriodKeys(ctx context.Context, period domain.Period) (bool, error) {
	patterns := []string{
		fmt.Sprintf("%s%s:%s:*", dynamicCountPrefix, period.Type, RedisPeriodStart(period)),
		fmt.Sprintf("%s%s:%s:*", dynamicVersionPrefix, period.Type, RedisPeriodStart(period)),
	}
	for _, pattern := range patterns {
		keys, err := scanKeys(ctx, f.redis, pattern)
		if err != nil {
			return false, err
		}
		if len(keys) > 0 {
			return true, nil
		}
	}
	return false, nil
}

func (f *PeriodFinalizer) periodVersions(ctx context.Context, period domain.Period) ([]string, map[string]int64, error) {
	pattern := fmt.Sprintf("%s%s:%s:*", dynamicVersionPrefix, period.Type, RedisPeriodStart(period))
	keys, err := scanKeys(ctx, f.redis, pattern)
	if err != nil {
		return nil, nil, err
	}
	versions := make(map[string]int64)
	for _, key := range keys {
		fields, err := f.redis.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, nil, err
		}
		for field, raw := range fields {
			var version int64
			if _, err := fmt.Sscan(raw, &version); err != nil {
				return nil, nil, err
			}
			versions[key+"|"+field] = version
		}
	}
	return keys, versions, nil
}

func (f *PeriodFinalizer) hasDirtyPeriod(ctx context.Context, period domain.Period) (bool, error) {
	members, err := f.redis.SMembers(ctx, dynamicDirtyKey).Result()
	if err != nil {
		return false, err
	}
	for _, member := range members {
		var identity dirtyIdentity
		if json.Unmarshal([]byte(member), &identity) == nil &&
			identity.PeriodType == period.Type && identity.PeriodStart.Equal(period.Start) {
			return true, nil
		}
	}
	return false, nil
}

func scanKeys(ctx context.Context, client *redis.Client, pattern string) ([]string, error) {
	var cursor uint64
	var result []string
	for {
		keys, next, err := client.Scan(ctx, cursor, pattern, 200).Result()
		if err != nil {
			return nil, err
		}
		result = append(result, keys...)
		cursor = next
		if cursor == 0 {
			return result, nil
		}
	}
}
