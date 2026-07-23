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

const repairMissingTokenUsageScript = `
if redis.call('HEXISTS', KEYS[1], ARGV[1]) == 0 then
  redis.call('HSET', KEYS[1], ARGV[1], ARGV[2])
  redis.call('EXPIREAT', KEYS[1], ARGV[3])
  return 1
end
redis.call('EXPIREAT', KEYS[1], ARGV[3])
return 0
`

type TokenUsageRepairResult struct {
	Repaired int
	Skipped  int
	Failed   int
}

type RedisTokenUsageReadRepairer struct {
	rdb           *redis.Client
	retentionDays int
	batchSize     int
}

func NewRedisTokenUsageReadRepairer(rdb *redis.Client, retentionDays, batchSize int) *RedisTokenUsageReadRepairer {
	return &RedisTokenUsageReadRepairer{rdb: rdb, retentionDays: retentionDays, batchSize: batchSize}
}

func (r *RedisTokenUsageReadRepairer) RepairModelUsage(ctx context.Context, day time.Time, rows []service.ModelTokenUsageRow) (TokenUsageRepairResult, error) {
	return r.repair(ctx, TokenStatisticsModel, day, len(rows), func(i int) (string, int64, error) {
		field, err := EncodeModelTokenStatisticsField(rows[i].Model)
		return field, rows[i].UsedTokens, err
	})
}

type currentTokenUsageRepairerAdapter struct{ repairer *RedisTokenUsageReadRepairer }

func (a currentTokenUsageRepairerAdapter) RepairModelUsage(ctx context.Context, day time.Time, rows []service.ModelTokenUsageRow) error {
	_, err := a.repairer.RepairModelUsage(ctx, day, rows)
	return err
}
func (a currentTokenUsageRepairerAdapter) RepairRouteUsage(ctx context.Context, day time.Time, rows []service.RouteTokenUsageRow) error {
	_, err := a.repairer.RepairRouteUsage(ctx, day, rows)
	return err
}
func (a currentTokenUsageRepairerAdapter) RepairUserModelUsage(ctx context.Context, day time.Time, rows []service.UserTokenUsageRow) error {
	_, err := a.repairer.RepairUserModelUsage(ctx, day, rows)
	return err
}

func NewCurrentTokenUsageRepairer(rdb *redis.Client, retentionDays, batchSize int) service.CurrentTokenUsageRepairer {
	return currentTokenUsageRepairerAdapter{repairer: NewRedisTokenUsageReadRepairer(rdb, retentionDays, batchSize)}
}

func (r *RedisTokenUsageReadRepairer) RepairRouteUsage(ctx context.Context, day time.Time, rows []service.RouteTokenUsageRow) (TokenUsageRepairResult, error) {
	return r.repair(ctx, TokenStatisticsGroupCandidate, day, len(rows), func(i int) (string, int64, error) {
		field, err := EncodeGroupCandidateTokenStatisticsField(rows[i].GroupID, rows[i].RouteAlias, rows[i].UpstreamModel)
		return field, rows[i].UsedTokens, err
	})
}

func (r *RedisTokenUsageReadRepairer) RepairUserModelUsage(ctx context.Context, day time.Time, rows []service.UserTokenUsageRow) (TokenUsageRepairResult, error) {
	return r.repair(ctx, TokenStatisticsUserModel, day, len(rows), func(i int) (string, int64, error) {
		field, err := EncodeUserModelTokenStatisticsField(rows[i].UserID, rows[i].Model)
		return field, rows[i].UsedTokens, err
	})
}

func (r *RedisTokenUsageReadRepairer) repair(ctx context.Context, kind TokenStatisticsType, day time.Time, count int, item func(int) (string, int64, error)) (TokenUsageRepairResult, error) {
	var result TokenUsageRepairResult
	if r == nil || r.rdb == nil || r.retentionDays <= 0 || r.batchSize <= 0 {
		return result, fmt.Errorf("token usage read repair: invalid configuration")
	}
	key, err := TokenStatisticsKey(kind, day)
	if err != nil {
		return result, err
	}
	expiresAt, err := TokenStatisticsExpireAt(day, r.retentionDays)
	if err != nil {
		return result, err
	}
	for start := 0; start < count; start += r.batchSize {
		end := min(start+r.batchSize, count)
		cmds := make([]*redis.Cmd, 0, end-start)
		pipe := r.rdb.Pipeline()
		for i := start; i < end; i++ {
			field, tokens, itemErr := item(i)
			if itemErr != nil || tokens < 0 {
				result.Failed++
				continue
			}
			cmds = append(cmds, pipe.Eval(ctx, repairMissingTokenUsageScript, []string{key}, field, strconv.FormatInt(tokens, 10), expiresAt.Unix()))
		}
		_, execErr := pipe.Exec(ctx)
		for _, cmd := range cmds {
			value, cmdErr := cmd.Int64()
			if cmdErr != nil {
				result.Failed++
				continue
			}
			if value == 1 {
				result.Repaired++
			} else {
				result.Skipped++
			}
		}
		if execErr != nil && result.Failed == 0 {
			result.Failed += len(cmds)
		}
	}
	if result.Repaired+result.Skipped+result.Failed > 0 {
		logger.L().Info("token_statistics.read_repair_completed", zap.String("statistics_type", string(kind)), zap.String("usage_date", day.Format(time.DateOnly)), zap.String("stage", "redis_atomic_repair"), zap.Int("repaired", result.Repaired), zap.Int("concurrent_skipped", result.Skipped), zap.Int("failed", result.Failed), zap.Bool("repair_succeeded", result.Failed == 0))
	}
	if result.Failed > 0 {
		return result, fmt.Errorf("token_statistics.read_repair_failed type=%s usage_date=%s stage=redis_atomic_repair repair_succeeded=false failed=%d", kind, day.Format(time.DateOnly), result.Failed)
	}
	return result, nil
}
