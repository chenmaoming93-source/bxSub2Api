package rbac

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

const (
	userPermissionKeyPrefix = "rbac:user:"
	userPermissionKeySuffix = ":permissions"
)

type CacheMetrics struct {
	Hits           atomic.Uint64
	Misses         atomic.Uint64
	RedisFallbacks atomic.Uint64
	DatabaseLoads  atomic.Uint64
	LoadErrors     atomic.Uint64
}

type PermissionService struct {
	repository AuthorizationRepository
	redis      *redis.Client
	ttl        time.Duration
	loads      singleflight.Group
	metrics    CacheMetrics
}

func NewPermissionService(repository AuthorizationRepository, redisClient *redis.Client, ttl time.Duration) *PermissionService {
	if ttl <= 0 {
		ttl = 20 * time.Minute
	}
	return &PermissionService{repository: repository, redis: redisClient, ttl: ttl}
}

func UserPermissionCacheKey(userID int64) string {
	return userPermissionKeyPrefix + strconv.FormatInt(userID, 10) + userPermissionKeySuffix
}

func (s *PermissionService) GetEffectivePermissions(ctx context.Context, userID int64) (EffectivePermissions, error) {
	userVersion, policyVersion, err := s.currentVersions(ctx, userID)
	if err != nil {
		s.metrics.LoadErrors.Add(1)
		return EffectivePermissions{}, err
	}
	if cached, ok := s.readMatchingCache(ctx, userID, userVersion, policyVersion); ok {
		s.metrics.Hits.Add(1)
		return cached, nil
	}
	s.metrics.Misses.Add(1)

	value, err, _ := s.loads.Do(strconv.FormatInt(userID, 10), func() (any, error) {
		// Re-read both authoritative versions after waiting for an in-flight load.
		currentUserVersion, currentPolicyVersion, versionErr := s.currentVersions(ctx, userID)
		if versionErr != nil {
			return EffectivePermissions{}, versionErr
		}
		if cached, ok := s.readMatchingCache(ctx, userID, currentUserVersion, currentPolicyVersion); ok {
			return cached, nil
		}
		grants, loadErr := s.repository.LoadActiveGrants(ctx, userID)
		if loadErr != nil {
			return EffectivePermissions{}, fmt.Errorf("load RBAC grants: %w", loadErr)
		}
		s.metrics.DatabaseLoads.Add(1)
		effective := Evaluate(grants, currentUserVersion, currentPolicyVersion)
		s.writeCacheBestEffort(ctx, userID, effective)
		return effective, nil
	})
	if err != nil {
		s.metrics.LoadErrors.Add(1)
		return EffectivePermissions{}, err
	}
	return value.(EffectivePermissions), nil
}

func (s *PermissionService) currentVersions(ctx context.Context, userID int64) (int64, int64, error) {
	userVersion, err := s.repository.GetUserVersion(ctx, userID)
	if err != nil {
		return 0, 0, err
	}
	policyVersion, err := s.repository.GetPolicyVersion(ctx)
	if err != nil {
		return 0, 0, err
	}
	return userVersion, policyVersion, nil
}

func (s *PermissionService) readMatchingCache(
	ctx context.Context,
	userID, userVersion, policyVersion int64,
) (EffectivePermissions, bool) {
	if s.redis == nil {
		return EffectivePermissions{}, false
	}
	raw, err := s.redis.Get(ctx, UserPermissionCacheKey(userID)).Bytes()
	if err != nil {
		if err != redis.Nil {
			s.metrics.RedisFallbacks.Add(1)
		}
		return EffectivePermissions{}, false
	}
	var cached EffectivePermissions
	if err := json.Unmarshal(raw, &cached); err != nil {
		s.metrics.RedisFallbacks.Add(1)
		return EffectivePermissions{}, false
	}
	if cached.UserVersion != userVersion || cached.PolicyVersion != policyVersion {
		return EffectivePermissions{}, false
	}
	return cached, true
}

func (s *PermissionService) writeCacheBestEffort(ctx context.Context, userID int64, value EffectivePermissions) {
	if s.redis == nil {
		return
	}
	raw, err := json.Marshal(value)
	if err != nil {
		s.metrics.RedisFallbacks.Add(1)
		return
	}
	if err := s.redis.Set(ctx, UserPermissionCacheKey(userID), raw, s.ttl).Err(); err != nil {
		s.metrics.RedisFallbacks.Add(1)
	}
}

func (s *PermissionService) DeleteUserCache(ctx context.Context, userID int64) {
	if s.redis != nil {
		_ = s.redis.Del(ctx, UserPermissionCacheKey(userID)).Err()
	}
}

func (s *PermissionService) Metrics() *CacheMetrics {
	return &s.metrics
}
