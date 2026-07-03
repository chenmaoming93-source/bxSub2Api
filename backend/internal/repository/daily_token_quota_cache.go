package repository

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const (
	dailyTokenQuotaCachePrefix = "quota:daily_token:"
	dailyTokenQuotaCacheBuffer = 5 * time.Minute

	dailyTokenQuotaFieldExists = "exists"
	dailyTokenQuotaFieldUsed   = "used_tokens"
	dailyTokenQuotaFieldLimit  = "daily_limit_tokens"
)

type CachedDailyTokenQuotaRepository struct {
	base  service.DailyTokenQuotaRepository
	cache *dailyTokenQuotaRedisCache
}

func NewCachedDailyTokenQuotaRepository(base service.DailyTokenQuotaRepository, rdb *redis.Client) service.DailyTokenQuotaRepository {
	if base == nil || rdb == nil {
		return base
	}
	return &CachedDailyTokenQuotaRepository{base: base, cache: &dailyTokenQuotaRedisCache{rdb: rdb}}
}

type DailyTokenQuotaCacheInvalidator struct {
	cache *dailyTokenQuotaRedisCache
}

func NewDailyTokenQuotaCacheInvalidator(rdb *redis.Client) *DailyTokenQuotaCacheInvalidator {
	if rdb == nil {
		return nil
	}
	return &DailyTokenQuotaCacheInvalidator{cache: &dailyTokenQuotaRedisCache{rdb: rdb}}
}

func (i *DailyTokenQuotaCacheInvalidator) InvalidateModelDailyTokenQuota(ctx context.Context, key service.ModelDailyTokenQuotaKey) error {
	if i == nil || i.cache == nil {
		return nil
	}
	return i.cache.delete(ctx, modelDailyTokenQuotaCacheKey(key))
}

func (i *DailyTokenQuotaCacheInvalidator) InvalidateUserModelDailyTokenQuota(ctx context.Context, key service.UserModelDailyTokenQuotaKey) error {
	if i == nil || i.cache == nil {
		return nil
	}
	return i.cache.delete(ctx, userModelDailyTokenQuotaCacheKey(key))
}

func (r *CachedDailyTokenQuotaRepository) GetModelDailyTokenQuota(ctx context.Context, key service.ModelDailyTokenQuotaKey) (service.DailyTokenQuotaSnapshot, error) {
	cacheKey := modelDailyTokenQuotaCacheKey(key)
	if snapshot, ok, err := r.cache.get(ctx, cacheKey); err != nil {
		return service.DailyTokenQuotaSnapshot{}, err
	} else if ok {
		snapshot.UsageDate = timezone.StartOfDay(key.At)
		return snapshot, nil
	}
	snapshot, err := r.base.GetModelDailyTokenQuota(ctx, key)
	if err != nil {
		return service.DailyTokenQuotaSnapshot{}, err
	}
	return snapshot, r.cache.set(ctx, cacheKey, snapshot, dailyTokenQuotaCacheTTL(key.At, time.Now()))
}

func (r *CachedDailyTokenQuotaRepository) GetUserModelDailyTokenQuota(ctx context.Context, key service.UserModelDailyTokenQuotaKey) (service.DailyTokenQuotaSnapshot, error) {
	cacheKey := userModelDailyTokenQuotaCacheKey(key)
	if snapshot, ok, err := r.cache.get(ctx, cacheKey); err != nil {
		return service.DailyTokenQuotaSnapshot{}, err
	} else if ok {
		snapshot.UsageDate = timezone.StartOfDay(key.At)
		return snapshot, nil
	}
	snapshot, err := r.base.GetUserModelDailyTokenQuota(ctx, key)
	if err != nil {
		return service.DailyTokenQuotaSnapshot{}, err
	}
	return snapshot, r.cache.set(ctx, cacheKey, snapshot, dailyTokenQuotaCacheTTL(key.At, time.Now()))
}

func (r *CachedDailyTokenQuotaRepository) GetGroupCandidateDailyTokenQuota(ctx context.Context, key service.GroupCandidateDailyTokenQuotaKey) (service.DailyTokenQuotaSnapshot, error) {
	// Candidate limits are edited together with model routing. Until that admin
	// path owns a cache invalidator, read the authoritative config table to avoid
	// serving a stale limit for the rest of the day.
	return r.base.GetGroupCandidateDailyTokenQuota(ctx, key)
}

func (r *CachedDailyTokenQuotaRepository) IncrementDailyTokenQuotas(ctx context.Context, increment service.DailyTokenQuotaIncrement) error {
	if err := r.base.IncrementDailyTokenQuotas(ctx, increment); err != nil {
		return err
	}
	return errors.Join(
		r.cache.incrementIfPresent(ctx, modelDailyTokenQuotaCacheKey(increment.ModelKey), increment.Tokens),
		r.cache.incrementIfPresent(ctx, userModelDailyTokenQuotaCacheKey(increment.UserModelKey), increment.Tokens),
		r.cache.incrementIfPresent(ctx, groupCandidateDailyTokenQuotaCacheKey(increment.GroupCandidateKey), increment.Tokens),
	)
}

func (r *CachedDailyTokenQuotaRepository) InvalidateModelDailyTokenQuota(ctx context.Context, key service.ModelDailyTokenQuotaKey) error {
	return r.cache.delete(ctx, modelDailyTokenQuotaCacheKey(key))
}

func (r *CachedDailyTokenQuotaRepository) InvalidateUserModelDailyTokenQuota(ctx context.Context, key service.UserModelDailyTokenQuotaKey) error {
	return r.cache.delete(ctx, userModelDailyTokenQuotaCacheKey(key))
}

func (r *CachedDailyTokenQuotaRepository) InvalidateGroupCandidateDailyTokenQuota(ctx context.Context, key service.GroupCandidateDailyTokenQuotaKey) error {
	return r.cache.delete(ctx, groupCandidateDailyTokenQuotaCacheKey(key))
}

func modelDailyTokenQuotaCacheKey(key service.ModelDailyTokenQuotaKey) string {
	return dailyTokenQuotaCachePrefix + "model:" + cacheDay(key.At) + ":" + cachePart(key.Model)
}

func userModelDailyTokenQuotaCacheKey(key service.UserModelDailyTokenQuotaKey) string {
	return dailyTokenQuotaCachePrefix + "user_model:" + cacheDay(key.At) + ":" + strconv.FormatInt(key.UserID, 10) + ":" + cachePart(key.Model)
}

func groupCandidateDailyTokenQuotaCacheKey(key service.GroupCandidateDailyTokenQuotaKey) string {
	return dailyTokenQuotaCachePrefix + "group_candidate:" + cacheDay(key.At) + ":" + strconv.FormatInt(key.GroupID, 10) + ":" + cachePart(key.RouteAlias) + ":" + cachePart(key.UpstreamModel)
}

func cacheDay(at time.Time) string {
	return timezone.StartOfDay(at).Format("2006-01-02")
}

func cachePart(value string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}

func dailyTokenQuotaCacheTTL(at, now time.Time) time.Duration {
	expiresAt := timezone.StartOfDay(at).AddDate(0, 0, 1).Add(dailyTokenQuotaCacheBuffer)
	ttl := expiresAt.Sub(now)
	if ttl <= 0 {
		return time.Minute
	}
	return ttl
}

type dailyTokenQuotaRedisCache struct {
	rdb *redis.Client
}

func (c *dailyTokenQuotaRedisCache) get(ctx context.Context, key string) (service.DailyTokenQuotaSnapshot, bool, error) {
	data, err := c.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return service.DailyTokenQuotaSnapshot{}, false, err
	}
	if len(data) == 0 {
		return service.DailyTokenQuotaSnapshot{}, false, nil
	}
	used, err := strconv.ParseInt(data[dailyTokenQuotaFieldUsed], 10, 64)
	if err != nil {
		return service.DailyTokenQuotaSnapshot{}, false, fmt.Errorf("parse daily token quota cache used tokens: %w", err)
	}
	snapshot := service.DailyTokenQuotaSnapshot{
		Exists:     data[dailyTokenQuotaFieldExists] == "1",
		UsedTokens: used,
	}
	if rawLimit := data[dailyTokenQuotaFieldLimit]; rawLimit != "" {
		limit, err := strconv.ParseInt(rawLimit, 10, 64)
		if err != nil {
			return service.DailyTokenQuotaSnapshot{}, false, fmt.Errorf("parse daily token quota cache limit: %w", err)
		}
		snapshot.DailyLimitTokens = &limit
	}
	return snapshot, true, nil
}

func (c *dailyTokenQuotaRedisCache) set(ctx context.Context, key string, snapshot service.DailyTokenQuotaSnapshot, ttl time.Duration) error {
	exists := "0"
	if snapshot.Exists {
		exists = "1"
	}
	limit := ""
	if snapshot.DailyLimitTokens != nil {
		limit = strconv.FormatInt(*snapshot.DailyLimitTokens, 10)
	}
	pipe := c.rdb.TxPipeline()
	pipe.HSet(ctx, key,
		dailyTokenQuotaFieldExists, exists,
		dailyTokenQuotaFieldUsed, strconv.FormatInt(snapshot.UsedTokens, 10),
		dailyTokenQuotaFieldLimit, limit,
	)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (c *dailyTokenQuotaRedisCache) incrementIfPresent(ctx context.Context, key string, tokens int64) error {
	_, err := c.rdb.Eval(ctx, `
if redis.call("EXISTS", KEYS[1]) == 0 then
	return 0
end
redis.call("HSET", KEYS[1], "exists", "1")
redis.call("HINCRBY", KEYS[1], "used_tokens", ARGV[1])
return 1
`, []string{key}, tokens).Result()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	return err
}

func (c *dailyTokenQuotaRedisCache) delete(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, key).Err()
}
