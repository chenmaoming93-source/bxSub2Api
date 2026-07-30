package rbac

import "github.com/gin-gonic/gin"

type PrincipalType string

const (
	PrincipalUser        PrincipalType = "user"
	PrincipalAdminAPIKey PrincipalType = "admin_api_key"
)

type Principal struct {
	Type       PrincipalType
	UserID     int64
	Status     string
	AuthMethod string
	SuperAdmin bool
}

const principalContextKey = "rbac.principal"

func SetPrincipal(c *gin.Context, principal Principal) {
	c.Set(principalContextKey, principal)
}

func PrincipalFromContext(c *gin.Context) (Principal, bool) {
	value, ok := c.Get(principalContextKey)
	if !ok {
		return Principal{}, false
	}
	principal, ok := value.(Principal)
	return principal, ok
}
