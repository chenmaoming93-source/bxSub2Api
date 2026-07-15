package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

type RedisTokenStatisticsAccumulator struct {
	rdb           *redis.Client
	retentionDays int
}

func NewRedisTokenStatisticsAccumulator(rdb *redis.Client, retentionDays int) service.TokenStatisticsAccumulator {
	return &RedisTokenStatisticsAccumulator{rdb: rdb, retentionDays: retentionDays}
}

func (a *RedisTokenStatisticsAccumulator) Accumulate(ctx context.Context, increment service.TokenStatisticsIncrement) error {
	if increment.TotalTokens <= 0 {
		return nil
	}
	if a == nil || a.rdb == nil {
		return fmt.Errorf("token statistics accumulate stage=validate: redis client is nil")
	}

	modelKey, err := TokenStatisticsKey(TokenStatisticsModel, increment.UsageDate)
	if err != nil {
		return fmt.Errorf("token statistics accumulate stage=build_key type=%s: %w", TokenStatisticsModel, err)
	}
	userModelKey, err := TokenStatisticsKey(TokenStatisticsUserModel, increment.UsageDate)
	if err != nil {
		return fmt.Errorf("token statistics accumulate stage=build_key type=%s: %w", TokenStatisticsUserModel, err)
	}
	groupCandidateKey, err := TokenStatisticsKey(TokenStatisticsGroupCandidate, increment.UsageDate)
	if err != nil {
		return fmt.Errorf("token statistics accumulate stage=build_key type=%s: %w", TokenStatisticsGroupCandidate, err)
	}
	modelField, err := EncodeModelTokenStatisticsField(increment.Model)
	if err != nil {
		return fmt.Errorf("token statistics accumulate stage=encode_field type=%s key=%s: %w", TokenStatisticsModel, modelKey, err)
	}
	userModelField, err := EncodeUserModelTokenStatisticsField(increment.UserID, increment.Model)
	if err != nil {
		return fmt.Errorf("token statistics accumulate stage=encode_field type=%s key=%s: %w", TokenStatisticsUserModel, userModelKey, err)
	}
	groupCandidateField, err := EncodeGroupCandidateTokenStatisticsField(increment.GroupID, increment.RouteAlias, increment.UpstreamModel)
	if err != nil {
		return fmt.Errorf("token statistics accumulate stage=encode_field type=%s key=%s: %w", TokenStatisticsGroupCandidate, groupCandidateKey, err)
	}
	expiresAt, err := TokenStatisticsExpireAt(increment.UsageDate, a.retentionDays)
	if err != nil {
		return fmt.Errorf("token statistics accumulate stage=calculate_expiry keys=%s: %w", strings.Join([]string{modelKey, userModelKey, groupCandidateKey}, ","), err)
	}

	pipe := a.rdb.Pipeline()
	pipe.HIncrBy(ctx, modelKey, modelField, increment.TotalTokens)
	pipe.ExpireAt(ctx, modelKey, expiresAt)
	pipe.HIncrBy(ctx, userModelKey, userModelField, increment.TotalTokens)
	pipe.ExpireAt(ctx, userModelKey, expiresAt)
	pipe.HIncrBy(ctx, groupCandidateKey, groupCandidateField, increment.TotalTokens)
	pipe.ExpireAt(ctx, groupCandidateKey, expiresAt)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("token_statistics.redis_increment_failed stage=redis_pipeline_exec keys=%s: %w", strings.Join([]string{modelKey, userModelKey, groupCandidateKey}, ","), err)
	}
	return nil
}
