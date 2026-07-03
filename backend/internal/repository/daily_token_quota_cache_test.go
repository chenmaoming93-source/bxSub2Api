package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

type tokenQuotaCacheBaseStub struct {
	modelCalls int
	snapshot   service.DailyTokenQuotaSnapshot
	increment  service.DailyTokenQuotaIncrement
}

func (s *tokenQuotaCacheBaseStub) GetModelDailyTokenQuota(context.Context, service.ModelDailyTokenQuotaKey) (service.DailyTokenQuotaSnapshot, error) {
	s.modelCalls++
	return s.snapshot, nil
}

func (s *tokenQuotaCacheBaseStub) GetUserModelDailyTokenQuota(context.Context, service.UserModelDailyTokenQuotaKey) (service.DailyTokenQuotaSnapshot, error) {
	return s.snapshot, nil
}

func (s *tokenQuotaCacheBaseStub) GetGroupCandidateDailyTokenQuota(context.Context, service.GroupCandidateDailyTokenQuotaKey) (service.DailyTokenQuotaSnapshot, error) {
	return s.snapshot, nil
}

func (s *tokenQuotaCacheBaseStub) IncrementDailyTokenQuotas(_ context.Context, increment service.DailyTokenQuotaIncrement) error {
	s.increment = increment
	return nil
}

func newTokenQuotaCacheTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return mr, client
}

func TestDailyTokenQuotaCacheHitSkipsDB(t *testing.T) {
	_, client := newTokenQuotaCacheTestRedis(t)
	ctx := context.Background()
	at := time.Now()
	limit := int64(100)
	base := &tokenQuotaCacheBaseStub{snapshot: service.DailyTokenQuotaSnapshot{Exists: true, UsageDate: timezone.StartOfDay(at), UsedTokens: 1}}
	cache := &dailyTokenQuotaRedisCache{rdb: client}
	key := service.ModelDailyTokenQuotaKey{Model: "gpt-5", At: at}
	if err := cache.set(ctx, modelDailyTokenQuotaCacheKey(key), service.DailyTokenQuotaSnapshot{Exists: true, UsedTokens: 42, DailyLimitTokens: &limit}, time.Hour); err != nil {
		t.Fatalf("set cache: %v", err)
	}

	repo := NewCachedDailyTokenQuotaRepository(base, client)
	snapshot, err := repo.GetModelDailyTokenQuota(ctx, key)
	if err != nil {
		t.Fatalf("GetModelDailyTokenQuota: %v", err)
	}
	if base.modelCalls != 0 {
		t.Fatalf("DB calls = %d, want 0", base.modelCalls)
	}
	if !snapshot.Exists || snapshot.UsedTokens != 42 || snapshot.DailyLimitTokens == nil || *snapshot.DailyLimitTokens != limit {
		t.Fatalf("snapshot = %+v", snapshot)
	}
}

func TestDailyTokenQuotaCacheMissLoadsDBAndSetsTTL(t *testing.T) {
	mr, client := newTokenQuotaCacheTestRedis(t)
	ctx := context.Background()
	at := time.Now()
	limit := int64(200)
	base := &tokenQuotaCacheBaseStub{snapshot: service.DailyTokenQuotaSnapshot{Exists: true, UsageDate: timezone.StartOfDay(at), UsedTokens: 9, DailyLimitTokens: &limit}}
	key := service.ModelDailyTokenQuotaKey{Model: "gpt-5", At: at}

	repo := NewCachedDailyTokenQuotaRepository(base, client)
	snapshot, err := repo.GetModelDailyTokenQuota(ctx, key)
	if err != nil {
		t.Fatalf("GetModelDailyTokenQuota: %v", err)
	}
	if base.modelCalls != 1 {
		t.Fatalf("DB calls = %d, want 1", base.modelCalls)
	}
	if snapshot.UsedTokens != 9 {
		t.Fatalf("snapshot used = %d, want 9", snapshot.UsedTokens)
	}
	ttl := mr.TTL(modelDailyTokenQuotaCacheKey(key))
	if ttl <= 0 || ttl > 25*time.Hour {
		t.Fatalf("cache TTL = %v, want positive until next day plus buffer", ttl)
	}
}

func TestDailyTokenQuotaCacheKeysAreIsolated(t *testing.T) {
	at := time.Now()
	modelKey := modelDailyTokenQuotaCacheKey(service.ModelDailyTokenQuotaKey{Model: "1:2", At: at})
	userKey := userModelDailyTokenQuotaCacheKey(service.UserModelDailyTokenQuotaKey{UserID: 1, Model: "2", At: at})
	groupKey := groupCandidateDailyTokenQuotaCacheKey(service.GroupCandidateDailyTokenQuotaKey{GroupID: 1, RouteAlias: "2", UpstreamModel: "3:4", At: at})
	if modelKey == userKey || modelKey == groupKey || userKey == groupKey {
		t.Fatalf("cache keys collided: model=%q user=%q group=%q", modelKey, userKey, groupKey)
	}
}

func TestDailyTokenQuotaCacheIncrementSyncsExistingEntries(t *testing.T) {
	_, client := newTokenQuotaCacheTestRedis(t)
	ctx := context.Background()
	at := time.Now()
	base := &tokenQuotaCacheBaseStub{}
	cache := &dailyTokenQuotaRedisCache{rdb: client}
	modelKey := service.ModelDailyTokenQuotaKey{Model: "gpt-5", At: at}
	userKey := service.UserModelDailyTokenQuotaKey{UserID: 7, Model: "gpt-5", At: at}
	groupKey := service.GroupCandidateDailyTokenQuotaKey{GroupID: 8, RouteAlias: "chat", UpstreamModel: "gpt-5", At: at}
	for _, key := range []string{
		modelDailyTokenQuotaCacheKey(modelKey),
		userModelDailyTokenQuotaCacheKey(userKey),
		groupCandidateDailyTokenQuotaCacheKey(groupKey),
	} {
		if err := cache.set(ctx, key, service.DailyTokenQuotaSnapshot{Exists: true, UsedTokens: 10}, time.Hour); err != nil {
			t.Fatalf("set cache: %v", err)
		}
	}

	repo := NewCachedDailyTokenQuotaRepository(base, client)
	if err := repo.IncrementDailyTokenQuotas(ctx, service.DailyTokenQuotaIncrement{
		ModelKey:          modelKey,
		UserModelKey:      userKey,
		GroupCandidateKey: groupKey,
		Tokens:            5,
	}); err != nil {
		t.Fatalf("IncrementDailyTokenQuotas: %v", err)
	}

	for _, key := range []string{
		modelDailyTokenQuotaCacheKey(modelKey),
		userModelDailyTokenQuotaCacheKey(userKey),
		groupCandidateDailyTokenQuotaCacheKey(groupKey),
	} {
		got, err := client.HGet(ctx, key, dailyTokenQuotaFieldUsed).Int64()
		if err != nil {
			t.Fatalf("HGet %s: %v", key, err)
		}
		if got != 15 {
			t.Fatalf("%s used_tokens = %d, want 15", key, got)
		}
	}
}

func TestDailyTokenQuotaCacheInvalidationDeletesKey(t *testing.T) {
	_, client := newTokenQuotaCacheTestRedis(t)
	ctx := context.Background()
	at := time.Now()
	key := service.ModelDailyTokenQuotaKey{Model: "gpt-5", At: at}
	cacheKey := modelDailyTokenQuotaCacheKey(key)
	cache := &dailyTokenQuotaRedisCache{rdb: client}
	if err := cache.set(ctx, cacheKey, service.DailyTokenQuotaSnapshot{Exists: true, UsedTokens: 10}, time.Hour); err != nil {
		t.Fatalf("set cache: %v", err)
	}

	repo := NewCachedDailyTokenQuotaRepository(&tokenQuotaCacheBaseStub{}, client).(*CachedDailyTokenQuotaRepository)
	if err := repo.InvalidateModelDailyTokenQuota(ctx, key); err != nil {
		t.Fatalf("InvalidateModelDailyTokenQuota: %v", err)
	}
	if exists, err := client.Exists(ctx, cacheKey).Result(); err != nil {
		t.Fatalf("Exists: %v", err)
	} else if exists != 0 {
		t.Fatalf("cache key still exists after invalidation")
	}
}

func TestModelTokenQuotaAdminInvalidationMakesCachedReadFresh(t *testing.T) {
	_, redisClient := newTokenQuotaCacheTestRedis(t)
	client := newDailyTokenQuotaRepoTestClient(t)
	ctx := context.Background()
	at := time.Now()
	oldLimit := int64(10)
	newLimit := int64(20)
	base := NewDailyTokenQuotaRepository(client)
	adminRepo := ProvideModelTokenQuotaAdminRepository(client)
	invalidator := NewDailyTokenQuotaCacheInvalidator(redisClient)
	cached := NewCachedDailyTokenQuotaRepository(base, redisClient)
	if _, err := adminRepo.SetModelDailyTokenQuota(ctx, "gpt-5", at, &oldLimit); err != nil {
		t.Fatalf("set old quota: %v", err)
	}
	first, err := cached.GetModelDailyTokenQuota(ctx, service.ModelDailyTokenQuotaKey{Model: "gpt-5", At: at})
	if err != nil {
		t.Fatalf("first cached read: %v", err)
	}
	if first.DailyLimitTokens == nil || *first.DailyLimitTokens != oldLimit {
		t.Fatalf("first limit = %v, want %d", first.DailyLimitTokens, oldLimit)
	}
	if _, err := adminRepo.SetModelDailyTokenQuota(ctx, "gpt-5", at, &newLimit); err != nil {
		t.Fatalf("set new quota: %v", err)
	}
	if err := invalidator.InvalidateModelDailyTokenQuota(ctx, service.ModelDailyTokenQuotaKey{Model: "gpt-5", At: at}); err != nil {
		t.Fatalf("invalidate quota: %v", err)
	}

	second, err := cached.GetModelDailyTokenQuota(ctx, service.ModelDailyTokenQuotaKey{Model: "gpt-5", At: at})
	if err != nil {
		t.Fatalf("second cached read: %v", err)
	}
	if second.DailyLimitTokens == nil || *second.DailyLimitTokens != newLimit {
		t.Fatalf("second limit = %v, want %d", second.DailyLimitTokens, newLimit)
	}
}
