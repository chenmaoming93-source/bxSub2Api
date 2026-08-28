package service

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

type securityConfigStoreStub struct {
	config  domain.SecurityCheckConfig
	err     error
	gets    int
	updates int
}

func (s *securityConfigStoreStub) GetSecurityCheckConfig(context.Context, int64) (domain.SecurityCheckConfig, error) {
	s.gets++
	return s.config, s.err
}

func (s *securityConfigStoreStub) UpdateSecurityCheckConfig(_ context.Context, _ int64, config domain.SecurityCheckConfig) error {
	s.updates++
	if s.err != nil {
		return s.err
	}
	s.config = config
	return nil
}

func TestSecurityConfigProviderUsesLocalRedisAndDatabaseFallback(t *testing.T) {
	mini := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	store := &securityConfigStoreStub{config: domain.DefaultSecurityCheckConfig()}
	provider := NewSecurityConfigProvider(rdb, store, time.Minute)

	first, err := provider.Get(context.Background(), 7)
	if err != nil {
		t.Fatalf("first get: %v", err)
	}
	if store.gets != 1 || !reflect.DeepEqual(first, store.config) {
		t.Fatalf("expected database fallback, gets=%d first=%#v", store.gets, first)
	}
	second, err := provider.Get(context.Background(), 7)
	if err != nil {
		t.Fatalf("second get: %v", err)
	}
	if store.gets != 1 || !reflect.DeepEqual(second, first) {
		t.Fatalf("expected local cache hit, gets=%d second=%#v", store.gets, second)
	}

	provider.Invalidate(7)
	third, err := provider.Get(context.Background(), 7)
	if err != nil || !reflect.DeepEqual(third, first) {
		t.Fatalf("redis fallback failed: config=%#v err=%v", third, err)
	}
	if store.gets != 1 {
		t.Fatalf("expected Redis to avoid database after invalidation, gets=%d", store.gets)
	}
}

func TestSecurityConfigProviderUsesStaleConfigWhenCachesFail(t *testing.T) {
	store := &securityConfigStoreStub{config: domain.DefaultSecurityCheckConfig()}
	provider := NewSecurityConfigProvider(nil, store, time.Millisecond)
	provider.now = func() time.Time { return time.Unix(10, 0) }
	if _, err := provider.Get(context.Background(), 3); err != nil {
		t.Fatalf("initial get: %v", err)
	}
	provider.now = func() time.Time { return time.Unix(10, 0).Add(time.Second) }
	store.err = errors.New("database unavailable")
	got, err := provider.Get(context.Background(), 3)
	if err != nil {
		t.Fatalf("stale fallback: %v", err)
	}
	if !reflect.DeepEqual(got, domain.DefaultSecurityCheckConfig()) {
		t.Fatalf("unexpected stale config: %#v", got)
	}
}

func TestSecurityConfigProviderInvalidatesOnlyOlderVersions(t *testing.T) {
	provider := NewSecurityConfigProvider(nil, nil, time.Minute)
	config := domain.DefaultSecurityCheckConfig()
	config.Version = 3
	provider.saveLocal(9, config, time.Now())
	provider.InvalidateIfOlder(9, 2)
	if _, ok := provider.local[9]; !ok {
		t.Fatal("older notification must not invalidate newer local config")
	}
	provider.InvalidateIfOlder(9, 3)
	if _, ok := provider.local[9]; ok {
		t.Fatal("same-version notification should invalidate local config")
	}
}

func TestSecurityConfigProviderPublishesVersionedUpdate(t *testing.T) {
	mini := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	store := &securityConfigStoreStub{config: domain.DefaultSecurityCheckConfig()}
	provider := NewSecurityConfigProvider(rdb, store, time.Minute)
	config := domain.DefaultSecurityCheckConfig()
	config.Enabled = true
	config.Version = 2
	if err := provider.Update(context.Background(), 5, config); err != nil {
		t.Fatalf("update: %v", err)
	}
	data, err := rdb.Get(context.Background(), securityConfigKey(5)).Bytes()
	if err != nil {
		t.Fatalf("redis get: %v", err)
	}
	var got domain.SecurityCheckConfig
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("decode redis config: %v", err)
	}
	if got.Version != 2 || store.updates != 1 {
		t.Fatalf("unexpected update result: config=%#v updates=%d", got, store.updates)
	}
}
