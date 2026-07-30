package rbac

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegistryAndRouteRegistrar(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	group := engine.Group("/api/v1/users")
	registry := NewRegistry()
	registrar := NewRouteRegistrar(registry, func(string) gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	})
	registrar.GET(group, "", PermissionUsersRead, func(c *gin.Context) { c.Status(http.StatusNoContent) })
	require.Equal(t, []RouteDeclaration{{
		Method: http.MethodGet, Path: "/api/v1/users",
		Classification: RouteControlled, Permission: PermissionUsersRead,
	}}, registry.Routes())
}

func TestRegistryRejectsEmptyUnknownAndDuplicateDeclarations(t *testing.T) {
	registry := NewRegistry()
	require.Error(t, registry.RegisterControlled("GET", "/empty", ""))
	require.Error(t, registry.RegisterControlled("GET", "/unknown", "does.not.exist"))
	require.NoError(t, registry.RegisterControlled("GET", "/users", PermissionUsersRead))
	require.Error(t, registry.RegisterControlled("GET", "/users", PermissionUsersRead))
	require.Error(t, registry.RegisterExcluded("POST", "/webhook", ""))
}
