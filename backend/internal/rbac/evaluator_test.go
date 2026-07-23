package rbac

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvaluateMergesRolesAndPermissions(t *testing.T) {
	result := Evaluate([]Grant{
		{RoleCode: "operator", RoleActive: true, PermissionCode: PermissionUsersRead, PermissionActive: true},
		{RoleCode: "finance", RoleActive: true, PermissionCode: PermissionBillingRead, PermissionActive: true},
		{RoleCode: "finance", RoleActive: true, PermissionCode: PermissionUsersRead, PermissionActive: true},
	}, 7, 12)
	require.Equal(t, []string{"finance", "operator"}, result.Roles)
	require.Equal(t, []string{PermissionBillingRead, PermissionUsersRead}, result.Permissions)
	require.False(t, result.IsSuperAdmin)
	require.EqualValues(t, 7, result.UserVersion)
	require.EqualValues(t, 12, result.PolicyVersion)
}

func TestEvaluateFiltersInactiveAndRecognizesWildcard(t *testing.T) {
	result := Evaluate([]Grant{
		{RoleCode: "disabled", RoleActive: false, PermissionCode: PermissionAll, PermissionActive: true},
		{RoleCode: "admin", RoleActive: true, PermissionCode: PermissionAll, PermissionActive: true},
		{RoleCode: "operator", RoleActive: true, PermissionCode: PermissionUsersRead, PermissionActive: false},
	}, 1, 1)
	require.Equal(t, []string{"admin", "operator"}, result.Roles)
	require.Equal(t, []string{PermissionAll}, result.Permissions)
	require.True(t, result.IsSuperAdmin)
}

func TestEvaluateEmptyRolesDeniesEverything(t *testing.T) {
	result := Evaluate(nil, 1, 1)
	require.Empty(t, result.Roles)
	require.Empty(t, result.Permissions)
	require.False(t, result.IsSuperAdmin)
}
