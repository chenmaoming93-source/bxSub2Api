package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type absoluteUsageStub struct {
	models   []service.ModelDailyTokenUsageAbsolute
	users    []service.UserModelDailyTokenUsageAbsolute
	groups   []service.GroupCandidateDailyTokenUsageAbsolute
	failures int
	calls    int
}

func (s *absoluteUsageStub) UpsertModelDailyTokenUsageAbsolute(_ context.Context, v []service.ModelDailyTokenUsageAbsolute) error {
	s.calls++
	if s.failures > 0 {
		s.failures--
		return errors.New("mysql 1205")
	}
	s.models = append(s.models, v...)
	return nil
}
func (s *absoluteUsageStub) UpsertUserModelDailyTokenUsageAbsolute(_ context.Context, v []service.UserModelDailyTokenUsageAbsolute) error {
	s.calls++
	s.users = append(s.users, v...)
	return nil
}
func (s *absoluteUsageStub) UpsertGroupCandidateDailyTokenUsageAbsolute(_ context.Context, v []service.GroupCandidateDailyTokenUsageAbsolute) error {
	s.calls++
	s.groups = append(s.groups, v...)
	return nil
}

func TestTokenStatisticsHScanSyncEngine(t *testing.T) {
	server, client := newTokenStatisticsAccumulatorTestClient(t)
	date := time.Date(2026, 7, 14, 12, 0, 0, 0, tokenStatisticsLocation)
	modelKey, _ := TokenStatisticsKey(TokenStatisticsModel, date)
	userKey, _ := TokenStatisticsKey(TokenStatisticsUserModel, date)
	groupKey, _ := TokenStatisticsKey(TokenStatisticsGroupCandidate, date)
	for i := 1; i <= 10000; i++ {
		model := fmt.Sprintf("model-%04d", i)
		mf, _ := EncodeModelTokenStatisticsField(model)
		server.HSet(modelKey, mf, fmt.Sprint(i))
		uf, _ := EncodeUserModelTokenStatisticsField(int64(i), model)
		server.HSet(userKey, uf, fmt.Sprint(i))
		gf, _ := EncodeGroupCandidateTokenStatisticsField(int64(i), "route", model)
		server.HSet(groupKey, gf, fmt.Sprint(i))
	}
	server.HSet(modelKey, "malformed", "1")
	target := &absoluteUsageStub{}
	engine := NewTokenStatisticsSyncEngine(client, target, 97, 113, 2)
	require.NoError(t, engine.SyncDate(context.Background(), date))
	require.Len(t, target.models, 10000)
	require.Len(t, target.users, 10000)
	require.Len(t, target.groups, 10000)
	require.Greater(t, target.calls, 3)
}

func TestTokenStatisticsSyncEngineRetry(t *testing.T) {
	server, client := newTokenStatisticsAccumulatorTestClient(t)
	date := time.Now()
	key, _ := TokenStatisticsKey(TokenStatisticsModel, date)
	field, _ := EncodeModelTokenStatisticsField("gpt-5")
	server.HSet(key, field, "10")
	target := &absoluteUsageStub{failures: 2}
	engine := NewTokenStatisticsSyncEngine(client, target, 10, 10, 2)
	require.NoError(t, engine.SyncDate(context.Background(), date))
	require.Equal(t, 3, target.calls)

	target = &absoluteUsageStub{failures: 3}
	engine = NewTokenStatisticsSyncEngine(client, target, 10, 10, 1)
	err := engine.SyncDate(context.Background(), date)
	require.ErrorContains(t, err, "stage=mysql_batch_upsert")
	require.ErrorContains(t, err, "mysql 1205")
}
