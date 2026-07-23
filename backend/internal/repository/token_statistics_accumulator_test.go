package repository

import (
	"context"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type pipelineCountingHook struct{ calls atomic.Int64 }

func (h *pipelineCountingHook) DialHook(next redis.DialHook) redis.DialHook          { return next }
func (h *pipelineCountingHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook { return next }
func (h *pipelineCountingHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		for _, cmd := range cmds {
			if cmd.Name() == "hincrby" {
				h.calls.Add(1)
				break
			}
		}
		return next(ctx, cmds)
	}
}

func newTokenStatisticsAccumulatorTestClient(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	return server, client
}

func tokenStatisticsTestIncrement() service.TokenStatisticsIncrement {
	now := time.Now().In(tokenStatisticsLocation)
	return service.TokenStatisticsIncrement{
		UserID: 11, GroupID: 22, RouteAlias: "route|alias", Model: "client|model",
		UpstreamModel: "upstream|model", UsageDate: time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, tokenStatisticsLocation), TotalTokens: 7,
	}
}

func TestTokenStatisticsAccumulatePipeline(t *testing.T) {
	server, client := newTokenStatisticsAccumulatorTestClient(t)
	hook := &pipelineCountingHook{}
	client.AddHook(hook)
	accumulator := NewRedisTokenStatisticsAccumulator(client, 2)
	increment := tokenStatisticsTestIncrement()

	require.NoError(t, accumulator.Accumulate(context.Background(), increment))
	require.Equal(t, int64(1), hook.calls.Load())

	modelKey, _ := TokenStatisticsKey(TokenStatisticsModel, increment.UsageDate)
	userKey, _ := TokenStatisticsKey(TokenStatisticsUserModel, increment.UsageDate)
	groupKey, _ := TokenStatisticsKey(TokenStatisticsGroupCandidate, increment.UsageDate)
	modelField, _ := EncodeModelTokenStatisticsField(increment.Model)
	userField, _ := EncodeUserModelTokenStatisticsField(increment.UserID, increment.Model)
	groupField, _ := EncodeGroupCandidateTokenStatisticsField(increment.GroupID, increment.RouteAlias, increment.UpstreamModel)
	require.Equal(t, "7", server.HGet(modelKey, modelField))
	require.Equal(t, "7", server.HGet(userKey, userField))
	require.Equal(t, "7", server.HGet(groupKey, groupField))
	expiresAt, err := TokenStatisticsExpireAt(increment.UsageDate, 2)
	require.NoError(t, err)
	wantTTL := time.Until(expiresAt)
	for _, key := range []string{modelKey, userKey, groupKey} {
		require.InDelta(t, wantTTL.Seconds(), server.TTL(key).Seconds(), 2)
	}
}

func TestTokenStatisticsAccumulateConcurrent(t *testing.T) {
	server, client := newTokenStatisticsAccumulatorTestClient(t)
	accumulator := NewRedisTokenStatisticsAccumulator(client, 2)
	increment := tokenStatisticsTestIncrement()
	increment.TotalTokens = 1

	const workers = 64
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- accumulator.Accumulate(context.Background(), increment)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	modelKey, _ := TokenStatisticsKey(TokenStatisticsModel, increment.UsageDate)
	modelField, _ := EncodeModelTokenStatisticsField(increment.Model)
	require.Equal(t, "64", server.HGet(modelKey, modelField))
}

func TestTokenStatisticsAccumulateSkipsNonPositive(t *testing.T) {
	_, client := newTokenStatisticsAccumulatorTestClient(t)
	hook := &pipelineCountingHook{}
	client.AddHook(hook)
	accumulator := NewRedisTokenStatisticsAccumulator(client, 2)
	increment := tokenStatisticsTestIncrement()
	increment.TotalTokens = 0
	require.NoError(t, accumulator.Accumulate(context.Background(), increment))
	require.Zero(t, hook.calls.Load())
}

func TestTokenStatisticsAccumulatePipelineError(t *testing.T) {
	server, client := newTokenStatisticsAccumulatorTestClient(t)
	accumulator := NewRedisTokenStatisticsAccumulator(client, 2)
	increment := tokenStatisticsTestIncrement()
	modelKey, _ := TokenStatisticsKey(TokenStatisticsModel, increment.UsageDate)
	server.Close()

	err := accumulator.Accumulate(context.Background(), increment)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "stage=redis_pipeline_exec") && strings.Contains(err.Error(), modelKey), err.Error())
	var netErr net.Error
	require.ErrorAs(t, err, &netErr)
}
