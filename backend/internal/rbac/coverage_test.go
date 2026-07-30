package rbac

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRBACRouteCoverageRejectsUnregisteredRoute(t *testing.T) {
	registry := NewRegistry()
	require.NoError(t, registry.RegisterControlled("GET", "/controlled", PermissionProfileSelfRead))
	actual := gin.RoutesInfo{
		{Method: "GET", Path: "/controlled"},
		{Method: "POST", Path: "/forgotten"},
	}
	report := ValidateRouteCoverage(actual, registry)
	require.Equal(t, []string{"POST /forgotten"}, report.Missing)
	require.Error(t, report.Err())
}

func TestCatalogConsistencyWithCompatibilitySeed(t *testing.T) {
	seedPath := filepath.Join("..", "..", "sqlArchiving", "163_seed_rbac_compatibility.sql")
	data, err := os.ReadFile(seedPath)
	require.NoError(t, err)
	seed := string(data)
	for _, permission := range Catalog() {
		require.Truef(t, strings.Contains(seed, "'"+permission.Code+"'"),
			"permission %s is missing from compatibility seed", permission.Code)
	}
}

func TestRBACRouteCoverageKnownExclusionsHaveReasons(t *testing.T) {
	registry := NewRegistry()
	actual := gin.RoutesInfo{
		{Method: "POST", Path: "/api/v1/payment/webhook/stripe"},
		{Method: "POST", Path: "/api/v1/integrations/users"},
		{Method: "POST", Path: "/v1/messages"},
	}
	require.NoError(t, RegisterKnownExclusions(actual, registry))
	report := ValidateRouteCoverage(actual, registry)
	require.NoError(t, report.Err())
	require.Equal(t, 3, report.Excluded)
	for _, route := range registry.Routes() {
		require.NotEmpty(t, route.ExclusionReason)
	}
}

func TestRBACRouteCoverageClassifiesRuntimePublicAndGatewayAliases(t *testing.T) {
	registry := NewRegistry()
	actual := gin.RoutesInfo{
		{Method: "GET", Path: "/setup/status"},
		{Method: "POST", Path: "/api/event_logging/batch"},
		{Method: "GET", Path: "/api/v1/settings/public"},
		{Method: "GET", Path: "/api/v1/settings/email-unsubscribe"},
		{Method: "GET", Path: "/api/v1/pages/:slug/images/*filename"},
		{Method: "GET", Path: "/antigravity/v1/models"},
		{Method: "POST", Path: "/antigravity/v1/messages"},
		{Method: "POST", Path: "/responses"},
		{Method: "POST", Path: "/responses/*subpath"},
		{Method: "GET", Path: "/backend-api/codex/responses"},
		{Method: "POST", Path: "/chat/completions"},
		{Method: "POST", Path: "/embeddings"},
		{Method: "POST", Path: "/images/generations"},
	}
	require.NoError(t, RegisterKnownExclusions(actual, registry))
	report := ValidateRouteCoverage(actual, registry)
	require.NoError(t, report.Err())
	require.Equal(t, len(actual), report.Excluded)
}
