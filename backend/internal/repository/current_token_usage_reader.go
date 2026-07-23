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

type RedisCurrentTokenUsageReader struct {
	rdb        *redis.Client
	hscanCount int64
}

func NewRedisCurrentTokenUsageReader(rdb *redis.Client, hscanCount int) *RedisCurrentTokenUsageReader {
	return &RedisCurrentTokenUsageReader{rdb: rdb, hscanCount: int64(hscanCount)}
}

func (r *RedisCurrentTokenUsageReader) ReadModelUsage(ctx context.Context, day time.Time, models []string) (service.CurrentTokenUsageReadResult[service.ModelTokenUsageRow], error) {
	fields := make([]string, 0, len(models))
	for _, model := range models {
		field, err := EncodeModelTokenStatisticsField(model)
		if err != nil {
			return service.CurrentTokenUsageReadResult[service.ModelTokenUsageRow]{}, err
		}
		fields = append(fields, field)
	}
	entries, err := r.read(ctx, TokenStatisticsModel, day, fields)
	if err != nil {
		logCurrentTokenUsageReadFailure(TokenStatisticsModel, day, err)
		return service.CurrentTokenUsageReadResult[service.ModelTokenUsageRow]{}, err
	}
	result := service.CurrentTokenUsageReadResult[service.ModelTokenUsageRow]{Rows: make([]service.ModelTokenUsageRow, 0, len(entries))}
	for _, entry := range entries {
		model, decodeErr := DecodeModelTokenStatisticsField(entry.field)
		tokens, valueErr := parseCurrentTokenUsageValue(entry.value)
		if decodeErr != nil || valueErr != nil {
			result.InvalidEntries++
			continue
		}
		result.Rows = append(result.Rows, service.ModelTokenUsageRow{UsageDate: day, Model: model, UsedTokens: tokens})
	}
	logInvalidCurrentTokenUsageEntries(TokenStatisticsModel, day, result.InvalidEntries)
	return result, nil
}

func (r *RedisCurrentTokenUsageReader) ReadRouteUsage(ctx context.Context, day time.Time, filters []service.RouteTokenUsageRow) (service.CurrentTokenUsageReadResult[service.RouteTokenUsageRow], error) {
	fields := make([]string, 0, len(filters))
	for _, filter := range filters {
		field, err := EncodeGroupCandidateTokenStatisticsField(filter.GroupID, filter.RouteAlias, filter.UpstreamModel)
		if err != nil {
			return service.CurrentTokenUsageReadResult[service.RouteTokenUsageRow]{}, err
		}
		fields = append(fields, field)
	}
	entries, err := r.read(ctx, TokenStatisticsGroupCandidate, day, fields)
	if err != nil {
		logCurrentTokenUsageReadFailure(TokenStatisticsGroupCandidate, day, err)
		return service.CurrentTokenUsageReadResult[service.RouteTokenUsageRow]{}, err
	}
	result := service.CurrentTokenUsageReadResult[service.RouteTokenUsageRow]{Rows: make([]service.RouteTokenUsageRow, 0, len(entries))}
	for _, entry := range entries {
		groupID, alias, model, decodeErr := DecodeGroupCandidateTokenStatisticsField(entry.field)
		tokens, valueErr := parseCurrentTokenUsageValue(entry.value)
		if decodeErr != nil || valueErr != nil {
			result.InvalidEntries++
			continue
		}
		result.Rows = append(result.Rows, service.RouteTokenUsageRow{UsageDate: day, GroupID: groupID, RouteAlias: alias, UpstreamModel: model, UsedTokens: tokens})
	}
	logInvalidCurrentTokenUsageEntries(TokenStatisticsGroupCandidate, day, result.InvalidEntries)
	return result, nil
}

func (r *RedisCurrentTokenUsageReader) ReadUserModelUsage(ctx context.Context, day time.Time, filters []service.UserTokenUsageRow) (service.CurrentTokenUsageReadResult[service.UserTokenUsageRow], error) {
	fields := make([]string, 0, len(filters))
	for _, filter := range filters {
		field, err := EncodeUserModelTokenStatisticsField(filter.UserID, filter.Model)
		if err != nil {
			return service.CurrentTokenUsageReadResult[service.UserTokenUsageRow]{}, err
		}
		fields = append(fields, field)
	}
	entries, err := r.read(ctx, TokenStatisticsUserModel, day, fields)
	if err != nil {
		logCurrentTokenUsageReadFailure(TokenStatisticsUserModel, day, err)
		return service.CurrentTokenUsageReadResult[service.UserTokenUsageRow]{}, err
	}
	result := service.CurrentTokenUsageReadResult[service.UserTokenUsageRow]{Rows: make([]service.UserTokenUsageRow, 0, len(entries))}
	for _, entry := range entries {
		userID, model, decodeErr := DecodeUserModelTokenStatisticsField(entry.field)
		tokens, valueErr := parseCurrentTokenUsageValue(entry.value)
		if decodeErr != nil || valueErr != nil {
			result.InvalidEntries++
			continue
		}
		result.Rows = append(result.Rows, service.UserTokenUsageRow{UsageDate: day, UserID: userID, Model: model, UsedTokens: tokens})
	}
	logInvalidCurrentTokenUsageEntries(TokenStatisticsUserModel, day, result.InvalidEntries)
	return result, nil
}

func logCurrentTokenUsageReadFailure(kind TokenStatisticsType, day time.Time, err error) {
	logger.L().Warn("token_statistics.current_usage_read_failed", zap.String("statistics_type", string(kind)), zap.String("usage_date", day.Format(time.DateOnly)), zap.String("stage", "redis_bulk_read"), zap.Bool("fallback_attempted", true), zap.Error(err))
}
func logInvalidCurrentTokenUsageEntries(kind TokenStatisticsType, day time.Time, count int) {
	if count == 0 {
		return
	}
	logger.L().Warn("token_statistics.current_usage_invalid_entries", zap.String("statistics_type", string(kind)), zap.String("usage_date", day.Format(time.DateOnly)), zap.String("stage", "decode_entries"), zap.Int("invalid_entries", count), zap.Bool("fallback_attempted", true))
}

type currentTokenUsageEntry struct{ field, value string }

func (r *RedisCurrentTokenUsageReader) read(ctx context.Context, statisticsType TokenStatisticsType, day time.Time, fields []string) ([]currentTokenUsageEntry, error) {
	if r == nil || r.rdb == nil || r.hscanCount <= 0 {
		return nil, fmt.Errorf("current token usage reader: invalid configuration")
	}
	key, err := TokenStatisticsKey(statisticsType, day)
	if err != nil {
		return nil, err
	}
	if len(fields) > 0 {
		values, err := r.rdb.HMGet(ctx, key, fields...).Result()
		if err != nil {
			return nil, fmt.Errorf("current token usage read type=%s stage=redis_hmget: %w", statisticsType, err)
		}
		entries := make([]currentTokenUsageEntry, 0, len(values))
		for i, value := range values {
			if value != nil {
				entries = append(entries, currentTokenUsageEntry{field: fields[i], value: fmt.Sprint(value)})
			}
		}
		return entries, nil
	}
	var cursor uint64
	entries := make([]currentTokenUsageEntry, 0)
	for {
		values, next, err := r.rdb.HScan(ctx, key, cursor, "*", r.hscanCount).Result()
		if err != nil {
			return nil, fmt.Errorf("current token usage read type=%s stage=redis_hscan: %w", statisticsType, err)
		}
		if len(values)%2 != 0 {
			return nil, fmt.Errorf("current token usage read type=%s stage=redis_hscan: odd entry count=%d", statisticsType, len(values))
		}
		for i := 0; i < len(values); i += 2 {
			entries = append(entries, currentTokenUsageEntry{field: values[i], value: values[i+1]})
		}
		cursor = next
		if cursor == 0 {
			return entries, nil
		}
	}
}

func parseCurrentTokenUsageValue(value string) (int64, error) {
	tokens, err := strconv.ParseInt(value, 10, 64)
	if err != nil || tokens < 0 {
		return 0, fmt.Errorf("invalid used_tokens=%q", value)
	}
	return tokens, nil
}
