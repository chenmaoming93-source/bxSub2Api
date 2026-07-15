package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisFirstDailyTokenQuotaRepository struct {
	base     service.DailyTokenQuotaRepository
	rdb      *redis.Client
	repairer *RedisTokenUsageReadRepairer
}

func NewRedisFirstDailyTokenQuotaRepositoryWithRepair(base service.DailyTokenQuotaRepository, rdb *redis.Client, repairer *RedisTokenUsageReadRepairer) service.DailyTokenQuotaRepository {
	if base == nil || rdb == nil {
		return base
	}
	return &RedisFirstDailyTokenQuotaRepository{base: base, rdb: rdb, repairer: repairer}
}

func NewRedisFirstDailyTokenQuotaRepository(base service.DailyTokenQuotaRepository, rdb *redis.Client) service.DailyTokenQuotaRepository {
	if base == nil || rdb == nil {
		return base
	}
	return &RedisFirstDailyTokenQuotaRepository{base: base, rdb: rdb}
}

func (r *RedisFirstDailyTokenQuotaRepository) GetModelDailyTokenQuota(ctx context.Context, key service.ModelDailyTokenQuotaKey) (service.DailyTokenQuotaSnapshot, error) {
	snapshot, err := r.base.GetModelDailyTokenQuota(ctx, key)
	if err != nil {
		return snapshot, fmt.Errorf("token statistics quota stage=mysql_limit_read type=model date=%s fallback_attempted=true: %w", key.At.Format("2006-01-02"), err)
	}
	redisKey, _ := TokenStatisticsKey(TokenStatisticsModel, key.At)
	field, err := EncodeModelTokenStatisticsField(key.Model)
	if err != nil {
		return snapshot, err
	}
	return r.overlay(ctx, snapshot, TokenStatisticsModel, redisKey, field, func() {
		if r.repairer != nil {
			_, _ = r.repairer.RepairModelUsage(ctx, key.At, []service.ModelTokenUsageRow{{UsageDate: key.At, Model: key.Model, UsedTokens: snapshot.UsedTokens}})
		}
	})
}
func (r *RedisFirstDailyTokenQuotaRepository) GetUserModelDailyTokenQuota(ctx context.Context, key service.UserModelDailyTokenQuotaKey) (service.DailyTokenQuotaSnapshot, error) {
	snapshot, err := r.base.GetUserModelDailyTokenQuota(ctx, key)
	if err != nil {
		return snapshot, fmt.Errorf("token statistics quota stage=mysql_limit_read type=user_model date=%s fallback_attempted=true: %w", key.At.Format("2006-01-02"), err)
	}
	redisKey, _ := TokenStatisticsKey(TokenStatisticsUserModel, key.At)
	field, err := EncodeUserModelTokenStatisticsField(key.UserID, key.Model)
	if err != nil {
		return snapshot, err
	}
	return r.overlay(ctx, snapshot, TokenStatisticsUserModel, redisKey, field, func() {
		if r.repairer != nil {
			_, _ = r.repairer.RepairUserModelUsage(ctx, key.At, []service.UserTokenUsageRow{{UsageDate: key.At, UserID: key.UserID, Model: key.Model, UsedTokens: snapshot.UsedTokens}})
		}
	})
}
func (r *RedisFirstDailyTokenQuotaRepository) GetGroupCandidateDailyTokenQuota(ctx context.Context, key service.GroupCandidateDailyTokenQuotaKey) (service.DailyTokenQuotaSnapshot, error) {
	snapshot, err := r.base.GetGroupCandidateDailyTokenQuota(ctx, key)
	if err != nil {
		return snapshot, fmt.Errorf("token statistics quota stage=mysql_limit_read type=group_candidate date=%s fallback_attempted=true: %w", key.At.Format("2006-01-02"), err)
	}
	redisKey, _ := TokenStatisticsKey(TokenStatisticsGroupCandidate, key.At)
	field, err := EncodeGroupCandidateTokenStatisticsField(key.GroupID, key.RouteAlias, key.UpstreamModel)
	if err != nil {
		return snapshot, err
	}
	return r.overlay(ctx, snapshot, TokenStatisticsGroupCandidate, redisKey, field, func() {
		if r.repairer != nil {
			_, _ = r.repairer.RepairRouteUsage(ctx, key.At, []service.RouteTokenUsageRow{{UsageDate: key.At, GroupID: key.GroupID, RouteAlias: key.RouteAlias, UpstreamModel: key.UpstreamModel, UsedTokens: snapshot.UsedTokens}})
		}
	})
}
func (r *RedisFirstDailyTokenQuotaRepository) overlay(ctx context.Context, snapshot service.DailyTokenQuotaSnapshot, statisticsType TokenStatisticsType, key, field string, repair func()) (service.DailyTokenQuotaSnapshot, error) {
	raw, err := r.rdb.HGet(ctx, key, field).Result()
	if errors.Is(err, redis.Nil) {
		if snapshot.Exists && snapshot.UsedTokens > 0 && repair != nil {
			repair()
		}
		return snapshot, nil
	}
	if err != nil {
		logger.L().Warn("token_statistics.quota_redis_read_failed", zap.String("stage", "redis_hget"), zap.String("statistics_type", string(statisticsType)), zap.String("redis_key", key), zap.Bool("fallback_attempted", true), zap.Error(err))
		return snapshot, nil
	}
	used, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return snapshot, fmt.Errorf("token statistics quota stage=redis_value_parse type=%s key=%s value=%q fallback_attempted=false: %w", statisticsType, key, raw, err)
	}
	if used < 0 {
		return snapshot, fmt.Errorf("token statistics quota stage=redis_value_parse type=%s key=%s value=%q fallback_attempted=false: expected non-negative integer", statisticsType, key, raw)
	}
	snapshot.Exists = true
	snapshot.UsedTokens = used
	return snapshot, nil
}
func (r *RedisFirstDailyTokenQuotaRepository) IncrementDailyTokenQuotas(ctx context.Context, increment service.DailyTokenQuotaIncrement) error {
	return r.base.IncrementDailyTokenQuotas(ctx, increment)
}
