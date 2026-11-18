package routes

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGrowthExchangeAdjustAuditReconciliationAndOtherAdminGrowthRoutesDeclareRBACPermissions(t *testing.T) {
	source, err := os.ReadFile("admin.go")
	require.NoError(t, err)
	text := string(source)
	for _, declaration := range []string{
		`"/stats/overview", rbac.PermissionGrowthRead`,
		`"/budget", rbac.PermissionGrowthRead`,
		`"/budget/entries", rbac.PermissionGrowthRead`,
		`"/budget/entries", rbac.PermissionGrowthManage`,
		`"/budget/allocations", rbac.PermissionGrowthRead`,
		`"/budget/allocations", rbac.PermissionGrowthManage`,
		`"/tasks/:id/budget/revoke", rbac.PermissionGrowthManage`,
		`"/tasks", rbac.PermissionGrowthRead`,
		`"/tasks", rbac.PermissionGrowthManage`,
		`"/tasks/:id", rbac.PermissionGrowthRead`,
		`"/tasks/:id", rbac.PermissionGrowthManage`,
		`"/tasks/:id/publish", rbac.PermissionGrowthManage`,
		`"/tasks/:id/offline", rbac.PermissionGrowthManage`,
		`"/tasks/:id/online", rbac.PermissionGrowthManage`,
		`"/tasks/:id/rule-versions", rbac.PermissionGrowthRead`,
		`"/products", rbac.PermissionGrowthRead`,
		`"/products", rbac.PermissionGrowthManage`,
		`"/products/:id", rbac.PermissionGrowthRead`,
		`"/products/:id", rbac.PermissionGrowthManage`,
		`"/products/:id/publish", rbac.PermissionGrowthManage`,
		`"/products/:id/stock", rbac.PermissionGrowthManage`,
		`"/products/:id/offline", rbac.PermissionGrowthManage`,
		`"/products/:id/versions", rbac.PermissionGrowthRead`,
		`"/exchanges", rbac.PermissionGrowthRead`,
		`"/exchanges/:id", rbac.PermissionGrowthRead`,
		`"/exchanges/:id/retry", rbac.PermissionGrowthManage`,
		`"/exchanges/:id/manual-complete", rbac.PermissionGrowthManage`,
		`"/exchanges/:id/refund", rbac.PermissionGrowthManage`,
		`"/users", rbac.PermissionGrowthRead`,
		`"/users/:id/ledger", rbac.PermissionGrowthRead`,
		`"/users/:id/exchanges", rbac.PermissionGrowthRead`,
		`"/users/:id/adjustments", rbac.PermissionGrowthAdjust`,
		`"/audit-logs", rbac.PermissionGrowthRead`,
		`"/reconciliation", rbac.PermissionGrowthRead`,
	} {
		require.Contains(t, text, declaration)
	}
	require.NotContains(t, text, `adminDELETE(routes, growth, "/tasks`)
	require.NotContains(t, text, `adminDELETE(routes, growth, "/products`)
}

func TestUserGrowthRoutesUseSelfReadPermission(t *testing.T) {
	source, err := os.ReadFile("user_growth.go")
	require.NoError(t, err)
	text := string(source)
	for _, endpoint := range []string{
		`"/overview"`,
		`"/tasks"`,
		`"/products"`,
		`"/products/:id"`,
		`"/exchanges"`,
		`"/ledger"`,
	} {
		require.Contains(t, text, endpoint+", rbac.PermissionGrowthSelfRead")
	}
	require.NotContains(t, text, "checkins")
	require.NotContains(t, text, "completion-requests")
	require.Contains(t, text, `routes.POST(growth, "/exchanges", rbac.PermissionGrowthSelfExchange`)
	require.Contains(t, text, `routes.GET(growth, "/exchanges/:id", rbac.PermissionGrowthSelfRead`)
}
