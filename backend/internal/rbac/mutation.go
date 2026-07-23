package rbac

import (
	"errors"
	"sort"
)

var (
	ErrSystemRoleProtected       = errors.New("RBAC system role is protected")
	ErrSystemPermissionProtected = errors.New("RBAC system permission is protected")
	ErrWildcardReserved          = errors.New("RBAC wildcard permission is reserved for admin")
	ErrLastSuperAdmin            = errors.New("cannot remove the last active super administrator")
)

type AuditActor struct {
	UserID               *int64
	RequestID, IPAddress string
}

type Role struct {
	ID          int64    `json:"id"`
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	IsSystem    bool     `json:"is_system"`
	Status      string   `json:"status"`
	Permissions []string `json:"permissions,omitempty"`
}

func ValidateRolePermissionReplacement(roleCode string, system bool, permissions []string) error {
	hasWildcard := false
	for _, code := range permissions {
		if code == PermissionAll {
			hasWildcard = true
			break
		}
	}
	if roleCode == "admin" {
		if !system || !hasWildcard {
			return ErrSystemRoleProtected
		}
		return nil
	}
	if hasWildcard {
		return ErrWildcardReserved
	}
	return nil
}

func SortedUnique(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value != "" {
			set[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
