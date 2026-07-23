package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/rbac"
)

var (
	ErrInvalidRBACRoleCode       = errors.New("role code must match ^[a-z][a-z0-9_]{1,63}$")
	ErrInvalidRBACPermissionCode = errors.New("permission code must use lowercase dotted segments")
)

type RBACRole = rbac.Role
type RBACPermission struct {
	ID          int64          `json:"id"`
	Code        string         `json:"code"`
	Name        string         `json:"name"`
	Module      string         `json:"module"`
	Description string         `json:"description"`
	Risk        rbac.RiskLevel `json:"risk_level"`
	IsSystem    bool           `json:"is_system"`
	Status      string         `json:"status"`
}

type RBACRoleRepository interface {
	ListRoles(context.Context, int, int, string, string) ([]RBACRole, int64, error)
	CreateRole(context.Context, rbac.AuditActor, string, string, string) (*RBACRole, error)
	UpdateRole(context.Context, rbac.AuditActor, int64, string, string, string) (*RBACRole, error)
	DeleteRole(context.Context, rbac.AuditActor, int64) error
	GetRolePermissions(context.Context, int64) ([]string, error)
	ReplaceRolePermissions(context.Context, rbac.AuditActor, int64, []string) error
	GetUserRoles(context.Context, int64) ([]string, error)
	ReplaceUserRoles(context.Context, rbac.AuditActor, int64, []string) error
	ListPermissions(context.Context) ([]RBACPermission, error)
	CreatePermission(context.Context, rbac.AuditActor, string, string, string, string, rbac.RiskLevel) (*RBACPermission, error)
	UpdatePermission(context.Context, rbac.AuditActor, int64, string, string, string, rbac.RiskLevel, string) (*RBACPermission, error)
	DeletePermission(context.Context, rbac.AuditActor, int64) error
	PermissionsExist(context.Context, []string) (bool, error)
}

type RBACRoleService struct {
	repo        RBACRoleRepository
	permissions *rbac.PermissionService
}

func NewRBACRoleService(repo RBACRoleRepository, permissions *rbac.PermissionService) *RBACRoleService {
	return &RBACRoleService{repo: repo, permissions: permissions}
}

func (s *RBACRoleService) List(ctx context.Context, page, pageSize int, status, search string) ([]RBACRole, int64, error) {
	return s.repo.ListRoles(ctx, page, pageSize, status, search)
}
func (s *RBACRoleService) Create(ctx context.Context, actor rbac.AuditActor, code, name, description string) (*RBACRole, error) {
	code = strings.TrimSpace(code)
	if !regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`).MatchString(code) || code == "admin" || code == "user" {
		return nil, ErrInvalidRBACRoleCode
	}
	return s.repo.CreateRole(ctx, actor, code, strings.TrimSpace(name), strings.TrimSpace(description))
}
func (s *RBACRoleService) Update(ctx context.Context, actor rbac.AuditActor, id int64, name, description, status string) (*RBACRole, error) {
	if status != "" && status != "active" && status != "disabled" {
		return nil, errors.New("invalid role status")
	}
	return s.repo.UpdateRole(ctx, actor, id, strings.TrimSpace(name), strings.TrimSpace(description), status)
}
func (s *RBACRoleService) Delete(ctx context.Context, actor rbac.AuditActor, id int64) error {
	return s.repo.DeleteRole(ctx, actor, id)
}
func (s *RBACRoleService) Permissions(ctx context.Context, roleID int64) ([]string, error) {
	return s.repo.GetRolePermissions(ctx, roleID)
}
func (s *RBACRoleService) ReplacePermissions(ctx context.Context, actor rbac.AuditActor, roleID int64, codes []string) error {
	exists, err := s.repo.PermissionsExist(ctx, rbac.SortedUnique(codes))
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("one or more permissions are unknown or disabled")
	}
	return s.repo.ReplaceRolePermissions(ctx, actor, roleID, codes)
}
func (s *RBACRoleService) PermissionCatalog(ctx context.Context) ([]RBACPermission, error) {
	return s.repo.ListPermissions(ctx)
}
func (s *RBACRoleService) CreatePermission(ctx context.Context, actor rbac.AuditActor, code, name, module, description string, risk rbac.RiskLevel) (*RBACPermission, error) {
	code, name, module = strings.TrimSpace(code), strings.TrimSpace(name), strings.TrimSpace(module)
	if !regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`).MatchString(code) || code == rbac.PermissionAll {
		return nil, ErrInvalidRBACPermissionCode
	}
	if name == "" || module == "" || !validRisk(risk) {
		return nil, errors.New("name, module and a valid risk_level are required")
	}
	return s.repo.CreatePermission(ctx, actor, code, name, module, strings.TrimSpace(description), risk)
}
func (s *RBACRoleService) UpdatePermission(ctx context.Context, actor rbac.AuditActor, id int64, name, module, description string, risk rbac.RiskLevel, status string) (*RBACPermission, error) {
	if status != "active" && status != "disabled" {
		return nil, errors.New("invalid permission status")
	}
	if strings.TrimSpace(name) == "" || strings.TrimSpace(module) == "" || !validRisk(risk) {
		return nil, errors.New("name, module and a valid risk_level are required")
	}
	return s.repo.UpdatePermission(ctx, actor, id, strings.TrimSpace(name), strings.TrimSpace(module), strings.TrimSpace(description), risk, status)
}
func (s *RBACRoleService) DeletePermission(ctx context.Context, actor rbac.AuditActor, id int64) error {
	return s.repo.DeletePermission(ctx, actor, id)
}
func validRisk(risk rbac.RiskLevel) bool {
	return risk == rbac.RiskLow || risk == rbac.RiskMedium || risk == rbac.RiskHigh || risk == rbac.RiskCritical
}
func (s *RBACRoleService) UserRoles(ctx context.Context, userID int64) ([]string, error) {
	return s.repo.GetUserRoles(ctx, userID)
}
func (s *RBACRoleService) ReplaceUserRoles(ctx context.Context, actor rbac.AuditActor, actorIsSuper bool, userID int64, codes []string) error {
	before, err := s.repo.GetUserRoles(ctx, userID)
	if err != nil {
		return err
	}
	after := rbac.SortedUnique(codes)
	if containsRBACCode(after, "admin") && !containsRBACCode(before, "admin") {
		if !actorIsSuper || actor.UserID == nil || *actor.UserID == userID {
			return errors.New("only another super administrator can grant admin")
		}
	}
	if err := s.repo.ReplaceUserRoles(ctx, actor, userID, after); err != nil {
		return err
	}
	if s.permissions != nil {
		s.permissions.DeleteUserCache(ctx, userID)
	}
	return nil
}
func containsRBACCode(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
