package tokenstat

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	domain "github.com/Wei-Shaw/sub2api/internal/service/tokenstat"
)

type periodStateStub struct{ states []string }

func (s *periodStateStub) SetPeriodState(_ context.Context, _ domain.Period, state, _ string) error {
	s.states = append(s.states, state)
	return nil
}

type versionVerifierStub struct{ err error }

func (s versionVerifierStub) VerifyPeriodVersions(context.Context, domain.Period, map[string]int64) error {
	return s.err
}

type pendingStub bool

func (p pendingStub) HasPendingBefore(time.Time) bool { return bool(p) }

type syncerStub struct {
	client *redis.Client
	err    error
}

func (s syncerStub) Sync(ctx context.Context) error {
	if s.err == nil {
		return s.client.Del(ctx, dynamicDirtyKey).Err()
	}
	return s.err
}

func TestDynamicTokenPeriodFinalizerUnlinksOnlyEndedPeriod(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	location, _ := time.LoadLocation("Asia/Shanghai")
	periods := domain.NaturalPeriods(time.Now().AddDate(0, 0, -2), location)
	ended := periods[0]
	current := domain.NaturalPeriods(time.Now(), location)[0]
	accumulator := NewRedisAccumulator(client, 4, 7)
	var hash [16]byte
	require.NoError(t, accumulator.Add(context.Background(), []domain.AccountingOperation{
		{Period: ended, ProjectionID: 1, DimensionHash: hash, DimensionValues: map[string]any{"user_id": int64(1)}, MetricCode: domain.MetricTotalTokens, Delta: 3},
		{Period: current, ProjectionID: 1, DimensionHash: hash, DimensionValues: map[string]any{"user_id": int64(1)}, MetricCode: domain.MetricTotalTokens, Delta: 4},
	}))
	states := &periodStateStub{}
	finalizer := NewPeriodFinalizer(client, states, versionVerifierStub{}, pendingStub(false), syncerStub{client: client})
	require.NoError(t, finalizer.Finalize(context.Background(), ended, time.Now()))
	require.Equal(t, []string{PeriodStateClosing, PeriodStateFinalSync, PeriodStatePersisted, PeriodStateDeleted}, states.states)
	endedPattern := dynamicCountPrefix + string(ended.Type) + ":" + RedisPeriodStart(ended) + ":*"
	endedKeys, _ := scanKeys(context.Background(), client, endedPattern)
	require.Empty(t, endedKeys)
	currentKeys, _ := scanKeys(context.Background(), client, dynamicCountPrefix+string(current.Type)+":"+RedisPeriodStart(current)+":*")
	require.NotEmpty(t, currentKeys)
}

func TestDynamicTokenPeriodFinalizerSkipsPeriodWithoutRedisKeys(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	period := domain.Period{
		Type:  domain.PeriodDay,
		Start: time.Now().Add(-48 * time.Hour),
		End:   time.Now().Add(-24 * time.Hour),
	}
	states := &periodStateStub{}
	finalizer := NewPeriodFinalizer(client, states, versionVerifierStub{}, pendingStub(false), syncerStub{client: client})

	require.NoError(t, finalizer.Finalize(context.Background(), period, time.Now()))
	require.Empty(t, states.states)
}

func TestDynamicTokenPeriodFinalizerGuardsPendingSyncAndVersionFailures(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	period := domain.Period{Type: domain.PeriodDay, Start: time.Now().Add(-48 * time.Hour), End: time.Now().Add(-24 * time.Hour)}
	for name, tc := range map[string]struct {
		pending   bool
		syncErr   error
		verifyErr error
	}{
		"pending": {pending: true},
		"sync":    {syncErr: errors.New("mysql failed")},
		"version": {verifyErr: errors.New("version mismatch")},
	} {
		t.Run(name, func(t *testing.T) {
			require.NoError(t, client.Set(
				context.Background(),
				dynamicCountPrefix+string(period.Type)+":"+RedisPeriodStart(period)+":1:0",
				"exists",
				0,
			).Err())
			states := &periodStateStub{}
			finalizer := NewPeriodFinalizer(client, states, versionVerifierStub{err: tc.verifyErr}, pendingStub(tc.pending), syncerStub{client: client, err: tc.syncErr})
			require.Error(t, finalizer.Finalize(context.Background(), period, time.Now()))
			require.NotContains(t, states.states, PeriodStateDeleted)
			require.NoError(t, client.FlushDB(context.Background()).Err())
		})
	}
}
