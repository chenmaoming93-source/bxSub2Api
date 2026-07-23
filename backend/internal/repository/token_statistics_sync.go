package repository

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type TokenStatisticsSyncEngine struct {
	rdb        *redis.Client
	target     service.DailyTokenUsageAbsoluteRepository
	hscanCount int64
	batchSize  int
	retries    int
}

func NewTokenStatisticsSyncEngine(rdb *redis.Client, target service.DailyTokenUsageAbsoluteRepository, hscanCount, batchSize, retries int) *TokenStatisticsSyncEngine {
	return &TokenStatisticsSyncEngine{rdb: rdb, target: target, hscanCount: int64(hscanCount), batchSize: batchSize, retries: retries}
}

func (e *TokenStatisticsSyncEngine) SyncDate(ctx context.Context, usageDate time.Time) error {
	if e == nil || e.rdb == nil || e.target == nil || e.hscanCount <= 0 || e.batchSize <= 0 || e.retries < 0 {
		return fmt.Errorf("token statistics sync stage=validate: invalid engine configuration")
	}
	for _, statisticsType := range []TokenStatisticsType{TokenStatisticsModel, TokenStatisticsUserModel, TokenStatisticsGroupCandidate} {
		if err := e.syncType(ctx, statisticsType, usageDate); err != nil {
			return err
		}
	}
	return nil
}

func (e *TokenStatisticsSyncEngine) syncType(ctx context.Context, statisticsType TokenStatisticsType, usageDate time.Time) error {
	key, err := TokenStatisticsKey(statisticsType, usageDate)
	if err != nil {
		return fmt.Errorf("token statistics sync stage=build_key type=%s: %w", statisticsType, err)
	}
	var cursor uint64
	for {
		entries, next, err := e.rdb.HScan(ctx, key, cursor, "*", e.hscanCount).Result()
		if err != nil {
			return fmt.Errorf("token_statistics.redis_scan_failed stage=redis_hscan type=%s key=%s cursor=%d count=%d retry_attempt=0: %w", statisticsType, key, cursor, e.hscanCount, err)
		}
		if len(entries)%2 != 0 {
			return fmt.Errorf("token statistics sync stage=redis_hscan type=%s key=%s cursor=%d: odd entry count=%d", statisticsType, key, cursor, len(entries))
		}
		if err := e.syncEntries(ctx, statisticsType, usageDate, key, entries); err != nil {
			return err
		}
		cursor = next
		if cursor == 0 {
			return nil
		}
	}
}

func (e *TokenStatisticsSyncEngine) syncEntries(ctx context.Context, statisticsType TokenStatisticsType, usageDate time.Time, key string, entries []string) error {
	switch statisticsType {
	case TokenStatisticsModel:
		records := make([]service.ModelDailyTokenUsageAbsolute, 0, len(entries)/2)
		for i := 0; i < len(entries); i += 2 {
			model, err := DecodeModelTokenStatisticsField(entries[i])
			if err != nil {
				logTokenStatisticsFieldDecodeFailure(statisticsType, key, entries[i], err)
				continue
			}
			tokens, err := parseTokenStatisticsValue(statisticsType, key, entries[i+1])
			if err != nil {
				return err
			}
			records = append(records, service.ModelDailyTokenUsageAbsolute{Model: model, UsageDate: usageDate, UsedTokens: tokens})
		}
		return forTokenStatisticsBatches(records, e.batchSize, func(batch []service.ModelDailyTokenUsageAbsolute) error {
			return e.retry(func() error { return e.target.UpsertModelDailyTokenUsageAbsolute(ctx, batch) }, statisticsType, key, len(batch))
		})
	case TokenStatisticsUserModel:
		records := make([]service.UserModelDailyTokenUsageAbsolute, 0, len(entries)/2)
		for i := 0; i < len(entries); i += 2 {
			userID, model, err := DecodeUserModelTokenStatisticsField(entries[i])
			if err != nil {
				logTokenStatisticsFieldDecodeFailure(statisticsType, key, entries[i], err)
				continue
			}
			tokens, err := parseTokenStatisticsValue(statisticsType, key, entries[i+1])
			if err != nil {
				return err
			}
			records = append(records, service.UserModelDailyTokenUsageAbsolute{UserID: userID, Model: model, UsageDate: usageDate, UsedTokens: tokens})
		}
		return forTokenStatisticsBatches(records, e.batchSize, func(batch []service.UserModelDailyTokenUsageAbsolute) error {
			return e.retry(func() error { return e.target.UpsertUserModelDailyTokenUsageAbsolute(ctx, batch) }, statisticsType, key, len(batch))
		})
	case TokenStatisticsGroupCandidate:
		records := make([]service.GroupCandidateDailyTokenUsageAbsolute, 0, len(entries)/2)
		for i := 0; i < len(entries); i += 2 {
			groupID, alias, model, err := DecodeGroupCandidateTokenStatisticsField(entries[i])
			if err != nil {
				logTokenStatisticsFieldDecodeFailure(statisticsType, key, entries[i], err)
				continue
			}
			tokens, err := parseTokenStatisticsValue(statisticsType, key, entries[i+1])
			if err != nil {
				return err
			}
			records = append(records, service.GroupCandidateDailyTokenUsageAbsolute{GroupID: groupID, RouteAlias: alias, UpstreamModel: model, UsageDate: usageDate, UsedTokens: tokens})
		}
		return forTokenStatisticsBatches(records, e.batchSize, func(batch []service.GroupCandidateDailyTokenUsageAbsolute) error {
			return e.retry(func() error { return e.target.UpsertGroupCandidateDailyTokenUsageAbsolute(ctx, batch) }, statisticsType, key, len(batch))
		})
	default:
		return fmt.Errorf("token statistics sync stage=dispatch type=%s key=%s: unsupported statistics type", statisticsType, key)
	}
}

func logTokenStatisticsFieldDecodeFailure(statisticsType TokenStatisticsType, key, field string, err error) {
	if len(field) > 128 {
		field = field[:128]
	}
	logger.L().Warn("token_statistics.field_decode_failed", zap.String("stage", "decode_field"), zap.String("statistics_type", string(statisticsType)), zap.String("redis_key", key), zap.String("encoded_field", field), zap.Error(err))
}

func parseTokenStatisticsValue(statisticsType TokenStatisticsType, key, value string) (int64, error) {
	tokens, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("token statistics sync stage=parse_value type=%s key=%s value=%q: %w", statisticsType, key, value, err)
	}
	if tokens < 0 {
		return 0, fmt.Errorf("token statistics sync stage=parse_value type=%s key=%s value=%q: expected non-negative integer", statisticsType, key, value)
	}
	return tokens, nil
}

func forTokenStatisticsBatches[T any](records []T, size int, write func([]T) error) error {
	for start := 0; start < len(records); start += size {
		end := min(start+size, len(records))
		if err := write(records[start:end]); err != nil {
			return err
		}
	}
	return nil
}

func (e *TokenStatisticsSyncEngine) retry(write func() error, statisticsType TokenStatisticsType, key string, rows int) error {
	var err error
	for attempt := 0; attempt <= e.retries; attempt++ {
		if err = write(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("token_statistics.mysql_sync_failed stage=mysql_batch_upsert type=%s key=%s rows=%d retry_attempt=%d max_retries=%d: %w", statisticsType, key, rows, e.retries, e.retries, err)
}
