package rbac

import "sort"

type Grant struct {
	RoleCode         string
	RoleActive       bool
	PermissionCode   string
	PermissionActive bool
}

type EffectivePermissions struct {
	Roles         []string `json:"roles"`
	Permissions   []string `json:"permissions"`
	IsSuperAdmin  bool     `json:"is_super_admin"`
	UserVersion   int64    `json:"user_version"`
	PolicyVersion int64    `json:"policy_version"`
}

func Evaluate(grants []Grant, userVersion, policyVersion int64) EffectivePermissions {
	roleSet := make(map[string]struct{})
	permissionSet := make(map[string]struct{})
	superAdmin := false
	for _, grant := range grants {
		if !grant.RoleActive || grant.RoleCode == "" {
			continue
		}
		roleSet[grant.RoleCode] = struct{}{}
		if !grant.PermissionActive || grant.PermissionCode == "" {
			continue
		}
		permissionSet[grant.PermissionCode] = struct{}{}
		if grant.PermissionCode == PermissionAll {
			superAdmin = true
		}
	}
	roles := setKeys(roleSet)
	permissions := setKeys(permissionSet)
	return EffectivePermissions{
		Roles: roles, Permissions: permissions, IsSuperAdmin: superAdmin,
		UserVersion: userVersion, PolicyVersion: policyVersion,
	}
}

func setKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
