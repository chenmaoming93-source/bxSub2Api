package rbac

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSystemRoleGuards(t *testing.T) {
	require.ErrorIs(t, ValidateRolePermissionReplacement("admin", true, []string{PermissionUsersRead}), ErrSystemRoleProtected)
	require.ErrorIs(t, ValidateRolePermissionReplacement("custom", false, []string{PermissionAll}), ErrWildcardReserved)
	require.NoError(t, ValidateRolePermissionReplacement("admin", true, []string{PermissionAll}))
	require.NoError(t, ValidateRolePermissionReplacement("user", true, []string{PermissionProfileSelfRead}))
}
