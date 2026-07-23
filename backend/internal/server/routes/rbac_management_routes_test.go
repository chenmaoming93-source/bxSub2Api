package routes

import (
	"os"
	"strings"
	"testing"
)

func TestRBACManagementRoutesDeclareTheirOwnPermissions(t *testing.T) {
	data, err := os.ReadFile("admin.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	for _, declaration := range []string{
		`"/roles", rbac.PermissionRolesRead`,
		`"/roles", rbac.PermissionRolesCreate`,
		`"/roles/:id", rbac.PermissionRolesUpdate`,
		`"/roles/:id", rbac.PermissionRolesDelete`,
		`"/permissions", rbac.PermissionPermissionsRead`,
		`"/permissions", rbac.PermissionPermissionsCreate`,
		`"/permissions/:id", rbac.PermissionPermissionsUpdate`,
		`"/permissions/:id", rbac.PermissionPermissionsDelete`,
		`"/roles/:id/permissions", rbac.PermissionRolesPermissionsAssign`,
	} {
		if !strings.Contains(source, declaration) {
			t.Errorf("missing RBAC management declaration %s", declaration)
		}
	}
}
