package rbac

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type RouteRegistrar struct {
	registry *Registry
	guard    func(string) gin.HandlerFunc
}

func NewRouteRegistrar(registry *Registry, guard func(string) gin.HandlerFunc) *RouteRegistrar {
	return &RouteRegistrar{registry: registry, guard: guard}
}

func (r *RouteRegistrar) GET(group *gin.RouterGroup, path, permission string, handlers ...gin.HandlerFunc) {
	r.handle(group, http.MethodGet, path, permission, handlers...)
}
func (r *RouteRegistrar) POST(group *gin.RouterGroup, path, permission string, handlers ...gin.HandlerFunc) {
	r.handle(group, http.MethodPost, path, permission, handlers...)
}
func (r *RouteRegistrar) PUT(group *gin.RouterGroup, path, permission string, handlers ...gin.HandlerFunc) {
	r.handle(group, http.MethodPut, path, permission, handlers...)
}
func (r *RouteRegistrar) PATCH(group *gin.RouterGroup, path, permission string, handlers ...gin.HandlerFunc) {
	r.handle(group, http.MethodPatch, path, permission, handlers...)
}
func (r *RouteRegistrar) DELETE(group *gin.RouterGroup, path, permission string, handlers ...gin.HandlerFunc) {
	r.handle(group, http.MethodDelete, path, permission, handlers...)
}

func (r *RouteRegistrar) handle(
	group *gin.RouterGroup,
	method, path, permission string,
	handlers ...gin.HandlerFunc,
) {
	fullPath := strings.TrimSuffix(group.BasePath(), "/") + "/" + strings.TrimPrefix(path, "/")
	if path == "" {
		fullPath = group.BasePath()
	}
	if err := r.registry.RegisterControlled(method, fullPath, permission); err != nil {
		panic(err)
	}
	chain := make([]gin.HandlerFunc, 0, len(handlers)+1)
	chain = append(chain, r.guard(permission))
	chain = append(chain, handlers...)
	group.Handle(method, path, chain...)
}
