//go:build unit

package limit_exemption_test

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/rbac"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type rbacRoleRepositoryStub struct {
	userIDs  []int64
	replaced bool
}

func (s *rbacRoleRepositoryStub) ListRoles(context.Context, int, int, string, string) ([]rbac.Role, int64, error) {
	return nil, 0, nil
}
func (s *rbacRoleRepositoryStub) CreateRole(context.Context, rbac.AuditActor, string, string, string) (*rbac.Role, error) {
	return nil, nil
}
func (s *rbacRoleRepositoryStub) UpdateRole(context.Context, rbac.AuditActor, int64, string, string, string) (*rbac.Role, error) {
	return nil, nil
}
func (s *rbacRoleRepositoryStub) DeleteRole(context.Context, rbac.AuditActor, int64) error {
	return nil
}
func (s *rbacRoleRepositoryStub) GetRolePermissions(context.Context, int64) ([]string, error) {
	return nil, nil
}
func (s *rbacRoleRepositoryStub) ReplaceRolePermissions(context.Context, rbac.AuditActor, int64, []string) error {
	s.replaced = true
	return nil
}
func (s *rbacRoleRepositoryStub) GetUserRoles(context.Context, int64) ([]string, error) {
	return []string{"user"}, nil
}
func (s *rbacRoleRepositoryStub) ReplaceUserRoles(context.Context, rbac.AuditActor, int64, []string) error {
	s.replaced = true
	return nil
}
func (s *rbacRoleRepositoryStub) ListUserIDsByRoleID(context.Context, int64) ([]int64, error) {
	return append([]int64(nil), s.userIDs...), nil
}
func (s *rbacRoleRepositoryStub) ListPermissions(context.Context) ([]service.RBACPermission, error) {
	return nil, nil
}
func (s *rbacRoleRepositoryStub) CreatePermission(context.Context, rbac.AuditActor, string, string, string, string, rbac.RiskLevel) (*service.RBACPermission, error) {
	return nil, nil
}
func (s *rbacRoleRepositoryStub) UpdatePermission(context.Context, rbac.AuditActor, int64, string, string, string, rbac.RiskLevel, string) (*service.RBACPermission, error) {
	return nil, nil
}
func (s *rbacRoleRepositoryStub) DeletePermission(context.Context, rbac.AuditActor, int64) error {
	return nil
}
func (s *rbacRoleRepositoryStub) PermissionsExist(context.Context, []string) (bool, error) {
	return true, nil
}

type authCacheInvalidatorRecorder struct {
	userIDs []int64
}

func (r *authCacheInvalidatorRecorder) InvalidateAuthCacheByKey(context.Context, string) {}
func (r *authCacheInvalidatorRecorder) InvalidateAuthCacheByUserID(_ context.Context, userID int64) {
	r.userIDs = append(r.userIDs, userID)
}
func (r *authCacheInvalidatorRecorder) InvalidateAuthCacheByGroupID(context.Context, int64) {}

func TestRBACRoleChangesInvalidateAPIKeyAuthSnapshots(t *testing.T) {
	repo := &rbacRoleRepositoryStub{userIDs: []int64{7, 8}}
	invalidator := &authCacheInvalidatorRecorder{}
	svc := service.NewRBACRoleService(repo, nil, invalidator)
	actorUserID := int64(99)
	actor := rbac.AuditActor{UserID: &actorUserID}

	err := svc.ReplaceUserRoles(context.Background(), actor, true, 7, []string{"user"})
	require.NoError(t, err)
	require.Equal(t, []int64{7}, invalidator.userIDs)

	err = svc.ReplacePermissions(context.Background(), actor, 3, []string{"users.read"})
	require.NoError(t, err)
	require.Equal(t, []int64{7, 7, 8}, invalidator.userIDs)
	require.True(t, repo.replaced)
}
