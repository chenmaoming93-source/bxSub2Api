package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"sync/atomic"

	"github.com/Wei-Shaw/sub2api/internal/rbac"
	"github.com/gin-gonic/gin"
)

type RBACMode string

const (
	RBACModeShadow  RBACMode = "shadow"
	RBACModeEnforce RBACMode = "enforce"
)

type PermissionProvider interface {
	GetEffectivePermissions(context.Context, int64) (rbac.EffectivePermissions, error)
}

type AuthorizationDecision struct {
	Principal  rbac.Principal
	Permission string
	Allowed    bool
	Shadow     bool
	Reason     string
	Path       string
	Method     string
}

type AuthorizationAuditHook func(context.Context, AuthorizationDecision)

var rbacShadowDenials atomic.Uint64
var rbacEnforceDenials atomic.Uint64

func ParseRBACMode(value string) RBACMode {
	if value == string(RBACModeShadow) {
		return RBACModeShadow
	}
	return RBACModeEnforce
}

// RBACDecisionCounts exposes process-local counters for metrics collectors.
func RBACDecisionCounts() (shadowDenials, enforceDenials uint64) {
	return rbacShadowDenials.Load(), rbacEnforceDenials.Load()
}

func NewRBACAuthorizationAuditHook(enabled bool) AuthorizationAuditHook {
	if !enabled {
		return nil
	}
	return func(_ context.Context, decision AuthorizationDecision) {
		slog.Warn("rbac authorization denied",
			"user_id", decision.Principal.UserID,
			"principal_type", decision.Principal.Type,
			"permission", decision.Permission,
			"method", decision.Method,
			"path", decision.Path,
			"shadow", decision.Shadow,
			"reason", decision.Reason,
		)
	}
}

func PrincipalFromAuthenticatedSubject() gin.HandlerFunc {
	return func(c *gin.Context) {
		subject, exists := GetAuthSubjectFromContext(c)
		if !exists {
			c.Next()
			return
		}
		authMethod, _ := c.Get("auth_method")
		method, _ := authMethod.(string)
		principal := rbac.Principal{
			Type: rbac.PrincipalUser, UserID: subject.UserID,
			Status: "active", AuthMethod: method,
		}
		if method == "admin_api_key" {
			principal.Type = rbac.PrincipalAdminAPIKey
			principal.SuperAdmin = true
		}
		rbac.SetPrincipal(c, principal)
		c.Next()
	}
}

func RequireLegacyAdminForUnregistered(registry *rbac.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		if declaration, ok := registry.Lookup(c.Request.Method, c.FullPath()); ok &&
			declaration.Classification == rbac.RouteControlled {
			c.Next()
			return
		}
		principal, ok := rbac.PrincipalFromContext(c)
		if principal.SuperAdmin {
			c.Next()
			return
		}
		role, _ := GetUserRoleFromContext(c)
		if !ok || role != "admin" {
			AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "Admin access required")
			return
		}
		c.Next()
	}
}

func RequirePermission(
	provider PermissionProvider,
	mode RBACMode,
	audit AuthorizationAuditHook,
) func(string) gin.HandlerFunc {
	return func(permission string) gin.HandlerFunc {
		if permission == "" || !rbac.PermissionExists(permission) {
			panic("RequirePermission received an empty or unknown permission: " + permission)
		}
		return func(c *gin.Context) {
			principal, ok := rbac.PrincipalFromContext(c)
			if !ok {
				AbortWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization required")
				return
			}
			if principal.Status != "active" {
				denyRBAC(c, principal, permission, mode, audit, "principal_inactive")
				return
			}
			if principal.SuperAdmin {
				c.Next()
				return
			}
			effective, err := provider.GetEffectivePermissions(c.Request.Context(), principal.UserID)
			if err != nil {
				AbortWithError(c, http.StatusServiceUnavailable, "AUTHORIZATION_UNAVAILABLE", "Authorization temporarily unavailable")
				return
			}
			allowed := effective.IsSuperAdmin || containsPermission(effective.Permissions, permission)
			if !allowed {
				denyRBAC(c, principal, permission, mode, audit, "permission_missing")
				return
			}
			c.Next()
		}
	}
}

func containsPermission(permissions []string, permission string) bool {
	for _, candidate := range permissions {
		if candidate == rbac.PermissionAll || candidate == permission {
			return true
		}
	}
	return false
}

func denyRBAC(
	c *gin.Context,
	principal rbac.Principal,
	permission string,
	mode RBACMode,
	audit AuthorizationAuditHook,
	reason string,
) {
	shadow := mode == RBACModeShadow
	if shadow {
		rbacShadowDenials.Add(1)
	} else {
		rbacEnforceDenials.Add(1)
	}
	if audit != nil {
		audit(c.Request.Context(), AuthorizationDecision{
			Principal: principal, Permission: permission, Allowed: false,
			Shadow: shadow, Reason: reason, Path: c.FullPath(), Method: c.Request.Method,
		})
	}
	if shadow {
		c.Next()
		return
	}
	AbortWithError(c, http.StatusForbidden, "FORBIDDEN", "Permission denied")
}
