package rbac

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type cacheRepositoryStub struct {
	userVersion   atomic.Int64
	policyVersion atomic.Int64
	loads         atomic.Int64
	grants        []Grant
	loadErr       error
}

func (s *cacheRepositoryStub) LoadActiveGrants(context.Context, int64) ([]Grant, error) {
	s.loads.Add(1)
	if s.loadErr != nil {
		return nil, s.loadErr
	}
	time.Sleep(5 * time.Millisecond)
	return s.grants, nil
}
func (s *cacheRepositoryStub) GetUserVersion(context.Context, int64) (int64, error) {
	return s.userVersion.Load(), nil
}
func (s *cacheRepositoryStub) GetPolicyVersion(context.Context) (int64, error) {
	return s.policyVersion.Load(), nil
}

func newPermissionCacheTest(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return server, client
}

func TestPermissionCacheHitsAndVersionMismatchOverwritesFixedKey(t *testing.T) {
	server, redisClient := newPermissionCacheTest(t)
	repo := &cacheRepositoryStub{grants: []Grant{{
		RoleCode: "operator", RoleActive: true,
		PermissionCode: PermissionUsersRead, PermissionActive: true,
	}}}
	repo.userVersion.Store(1)
	repo.policyVersion.Store(1)
	service := NewPermissionService(repo, redisClient, time.Minute)

	first, err := service.GetEffectivePermissions(context.Background(), 42)
	require.NoError(t, err)
	second, err := service.GetEffectivePermissions(context.Background(), 42)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.EqualValues(t, 1, repo.loads.Load())
	require.Len(t, server.DB(0).Keys(), 1)

	repo.policyVersion.Store(2)
	third, err := service.GetEffectivePermissions(context.Background(), 42)
	require.NoError(t, err)
	require.EqualValues(t, 2, third.PolicyVersion)
	require.EqualValues(t, 2, repo.loads.Load())
	require.Len(t, server.DB(0).Keys(), 1)
}

func TestPermissionCacheRedisFailureFallsBackToDatabase(t *testing.T) {
	repo := &cacheRepositoryStub{grants: []Grant{{
		RoleCode: "admin", RoleActive: true,
		PermissionCode: PermissionAll, PermissionActive: true,
	}}}
	repo.userVersion.Store(1)
	repo.policyVersion.Store(1)
	brokenRedis := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:1", DialTimeout: 10 * time.Millisecond,
		ReadTimeout: 10 * time.Millisecond, WriteTimeout: 10 * time.Millisecond,
		MaxRetries: -1,
	})
	t.Cleanup(func() { _ = brokenRedis.Close() })

	result, err := NewPermissionService(repo, brokenRedis, time.Minute).
		GetEffectivePermissions(context.Background(), 1)
	require.NoError(t, err)
	require.True(t, result.IsSuperAdmin)
	require.Equal(t, []string{PermissionAll}, result.Permissions)
}

func TestPermissionCacheCoalescesConcurrentLoads(t *testing.T) {
	_, redisClient := newPermissionCacheTest(t)
	repo := &cacheRepositoryStub{grants: []Grant{{
		RoleCode: "user", RoleActive: true,
		PermissionCode: PermissionProfileSelfRead, PermissionActive: true,
	}}}
	repo.userVersion.Store(1)
	repo.policyVersion.Store(1)
	service := NewPermissionService(repo, redisClient, time.Minute)

	var wait sync.WaitGroup
	errs := make(chan error, 20)
	for range 20 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, err := service.GetEffectivePermissions(context.Background(), 9)
			errs <- err
		}()
	}
	wait.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.EqualValues(t, 1, repo.loads.Load())
}

func TestPermissionCacheNeverUsesDataWhenVersionCannotBeConfirmed(t *testing.T) {
	_, redisClient := newPermissionCacheTest(t)
	repo := &cacheRepositoryStub{loadErr: errors.New("database unavailable")}
	repo.userVersion.Store(1)
	repo.policyVersion.Store(1)
	service := NewPermissionService(repo, redisClient, time.Minute)
	_, err := service.GetEffectivePermissions(context.Background(), 3)
	require.Error(t, err)
}
