package rbac

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type RouteClassification string

const (
	RouteControlled RouteClassification = "rbac"
	RouteExcluded   RouteClassification = "excluded"
)

type RouteDeclaration struct {
	Method          string
	Path            string
	Classification  RouteClassification
	Permission      string
	ExclusionReason string
}

type Registry struct {
	mu     sync.RWMutex
	routes map[string]RouteDeclaration
}

func NewRegistry() *Registry {
	return &Registry{routes: make(map[string]RouteDeclaration)}
}

func (r *Registry) RegisterControlled(method, path, permission string) error {
	if strings.TrimSpace(permission) == "" {
		return fmt.Errorf("RBAC permission is empty for %s %s", method, path)
	}
	if !PermissionExists(permission) {
		return fmt.Errorf("unknown RBAC permission %q for %s %s", permission, method, path)
	}
	return r.register(RouteDeclaration{
		Method: strings.ToUpper(method), Path: path,
		Classification: RouteControlled, Permission: permission,
	})
}

func (r *Registry) RegisterExcluded(method, path, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("RBAC exclusion reason is empty for %s %s", method, path)
	}
	return r.register(RouteDeclaration{
		Method: strings.ToUpper(method), Path: path,
		Classification: RouteExcluded, ExclusionReason: reason,
	})
}

func (r *Registry) register(declaration RouteDeclaration) error {
	key := declaration.Method + " " + declaration.Path
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.routes[key]; ok {
		return fmt.Errorf("RBAC route %s already declared as %s", key, existing.Classification)
	}
	r.routes[key] = declaration
	return nil
}

func (r *Registry) Routes() []RouteDeclaration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]RouteDeclaration, 0, len(r.routes))
	for _, route := range r.routes {
		result = append(result, route)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Path == result[j].Path {
			return result[i].Method < result[j].Method
		}
		return result[i].Path < result[j].Path
	})
	return result
}

func (r *Registry) Lookup(method, path string) (RouteDeclaration, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	declaration, ok := r.routes[strings.ToUpper(method)+" "+path]
	return declaration, ok
}
