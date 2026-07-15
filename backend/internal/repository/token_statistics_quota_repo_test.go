package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type quotaReadStub struct {
	snapshot service.DailyTokenQuotaSnapshot
	err      error
}

func (s *quotaReadStub) GetModelDailyTokenQuota(context.Context, service.ModelDailyTokenQuotaKey) (service.DailyTokenQuotaSnapshot, error) {
	return s.snapshot, s.err
}
func (s *quotaReadStub) GetUserModelDailyTokenQuota(context.Context, service.UserModelDailyTokenQuotaKey) (service.DailyTokenQuotaSnapshot, error) {
	return s.snapshot, s.err
}
func (s *quotaReadStub) GetGroupCandidateDailyTokenQuota(context.Context, service.GroupCandidateDailyTokenQuotaKey) (service.DailyTokenQuotaSnapshot, error) {
	return s.snapshot, s.err
}
func (s *quotaReadStub) IncrementDailyTokenQuotas(context.Context, service.DailyTokenQuotaIncrement) error {
	return nil
}

func TestTokenQuotaRedisFirst(t *testing.T) {
	server, client := newTokenStatisticsAccumulatorTestClient(t)
	limit := int64(100)
	date := time.Now()
	base := &quotaReadStub{snapshot: service.DailyTokenQuotaSnapshot{Exists: true, UsageDate: date, UsedTokens: 20, DailyLimitTokens: &limit}}
	repo := NewRedisFirstDailyTokenQuotaRepository(base, client)
	key, _ := TokenStatisticsKey(TokenStatisticsModel, date)
	field, _ := EncodeModelTokenStatisticsField("gpt-5")
	server.HSet(key, field, "90")
	snapshot, err := repo.GetModelDailyTokenQuota(context.Background(), service.ModelDailyTokenQuotaKey{Model: "gpt-5", At: date})
	require.NoError(t, err)
	require.Equal(t, int64(90), snapshot.UsedTokens)
	require.Equal(t, int64(100), *snapshot.DailyLimitTokens)
	err = service.CheckModelDailyTokenQuota(service.ModelDailyTokenQuotaKey{Model: "gpt-5", At: date}, snapshot)
	require.NoError(t, err)
	server.HSet(key, field, "100")
	snapshot, err = repo.GetModelDailyTokenQuota(context.Background(), service.ModelDailyTokenQuotaKey{Model: "gpt-5", At: date})
	require.NoError(t, err)
	require.ErrorIs(t, service.CheckModelDailyTokenQuota(service.ModelDailyTokenQuotaKey{Model: "gpt-5", At: date}, snapshot), service.ErrModelDailyTokenQuotaExhausted)
}

func TestTokenQuotaRedisFallback(t *testing.T) {
	server, client := newTokenStatisticsAccumulatorTestClient(t)
	date := time.Now()
	base := &quotaReadStub{snapshot: service.DailyTokenQuotaSnapshot{Exists: true, UsageDate: date, UsedTokens: 33}}
	repo := NewRedisFirstDailyTokenQuotaRepository(base, client)
	snapshot, err := repo.GetUserModelDailyTokenQuota(context.Background(), service.UserModelDailyTokenQuotaKey{UserID: 1, Model: "gpt", At: date})
	require.NoError(t, err)
	require.Equal(t, int64(33), snapshot.UsedTokens)
	server.Close()
	snapshot, err = repo.GetGroupCandidateDailyTokenQuota(context.Background(), service.GroupCandidateDailyTokenQuotaKey{GroupID: 1, RouteAlias: "r", UpstreamModel: "gpt", At: date})
	require.NoError(t, err)
	require.Equal(t, int64(33), snapshot.UsedTokens)
	base.err = errors.New("mysql unavailable")
	_, err = repo.GetModelDailyTokenQuota(context.Background(), service.ModelDailyTokenQuotaKey{Model: "gpt", At: date})
	require.ErrorContains(t, err, "stage=mysql_limit_read")
	require.ErrorContains(t, err, "mysql unavailable")
}

func TestTokenQuotaRedisMissingRepairsAllDimensionsWithoutOverwriting(t *testing.T) {
	server, client := newTokenStatisticsAccumulatorTestClient(t)
	date := time.Now()
	base := &quotaReadStub{snapshot: service.DailyTokenQuotaSnapshot{Exists: true, UsageDate: date, UsedTokens: 33}}
	repairer := NewRedisTokenUsageReadRepairer(client, 2, 10)
	repo := NewRedisFirstDailyTokenQuotaRepositoryWithRepair(base, client, repairer)
	require.NoError(t, func() error {
		_, err := repo.GetModelDailyTokenQuota(context.Background(), service.ModelDailyTokenQuotaKey{Model: "gpt", At: date})
		return err
	}())
	require.NoError(t, func() error {
		_, err := repo.GetUserModelDailyTokenQuota(context.Background(), service.UserModelDailyTokenQuotaKey{UserID: 1, Model: "gpt", At: date})
		return err
	}())
	require.NoError(t, func() error {
		_, err := repo.GetGroupCandidateDailyTokenQuota(context.Background(), service.GroupCandidateDailyTokenQuotaKey{GroupID: 1, RouteAlias: "r", UpstreamModel: "gpt", At: date})
		return err
	}())
	modelKey, _ := TokenStatisticsKey(TokenStatisticsModel, date)
	modelField, _ := EncodeModelTokenStatisticsField("gpt")
	require.Equal(t, "33", server.HGet(modelKey, modelField))
	userKey, _ := TokenStatisticsKey(TokenStatisticsUserModel, date)
	userField, _ := EncodeUserModelTokenStatisticsField(1, "gpt")
	require.Equal(t, "33", server.HGet(userKey, userField))
	routeKey, _ := TokenStatisticsKey(TokenStatisticsGroupCandidate, date)
	routeField, _ := EncodeGroupCandidateTokenStatisticsField(1, "r", "gpt")
	require.Equal(t, "33", server.HGet(routeKey, routeField))
	server.HSet(modelKey, modelField, "99")
	snapshot, err := repo.GetModelDailyTokenQuota(context.Background(), service.ModelDailyTokenQuotaKey{Model: "gpt", At: date})
	require.NoError(t, err)
	require.Equal(t, int64(99), snapshot.UsedTokens)
	require.Equal(t, "99", server.HGet(modelKey, modelField))
}

func TestTokenQuotaRedisConnectionFailureDoesNotRepair(t *testing.T) {
	server, client := newTokenStatisticsAccumulatorTestClient(t)
	date := time.Now()
	base := &quotaReadStub{snapshot: service.DailyTokenQuotaSnapshot{Exists: true, UsageDate: date, UsedTokens: 44}}
	repo := NewRedisFirstDailyTokenQuotaRepositoryWithRepair(base, client, NewRedisTokenUsageReadRepairer(client, 2, 10))
	server.Close()
	snapshot, err := repo.GetModelDailyTokenQuota(context.Background(), service.ModelDailyTokenQuotaKey{Model: "gpt", At: date})
	require.NoError(t, err)
	require.Equal(t, int64(44), snapshot.UsedTokens)
}
