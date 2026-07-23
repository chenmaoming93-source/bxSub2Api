package admin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/rbac"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type RBACHandler struct{ service *service.RBACRoleService }

func NewRBACHandler(service *service.RBACRoleService) *RBACHandler {
	return &RBACHandler{service: service}
}

func (h *RBACHandler) ListRoles(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	roles, total, err := h.service.List(c.Request.Context(), page, pageSize, c.Query("status"), c.Query("search"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, roles, total, page, pageSize)
}

type createRBACRoleRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *RBACHandler) CreateRole(c *gin.Context) {
	var req createRBACRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Code == "" || req.Name == "" {
		response.BadRequest(c, "code and name are required")
		return
	}
	role, err := h.service.Create(c.Request.Context(), rbacActor(c), req.Code, req.Name, req.Description)
	if err != nil {
		rbacMutationError(c, err)
		return
	}
	response.Created(c, role)
}

type updateRBACRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

func (h *RBACHandler) UpdateRole(c *gin.Context) {
	id, ok := rbacRoleID(c)
	if !ok {
		return
	}
	var req updateRBACRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	role, err := h.service.Update(c.Request.Context(), rbacActor(c), id, req.Name, req.Description, req.Status)
	if err != nil {
		rbacMutationError(c, err)
		return
	}
	response.Success(c, role)
}
func (h *RBACHandler) DeleteRole(c *gin.Context) {
	id, ok := rbacRoleID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), rbacActor(c), id); err != nil {
		rbacMutationError(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
func (h *RBACHandler) ListPermissions(c *gin.Context) {
	items, err := h.service.PermissionCatalog(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

type permissionRequest struct {
	Code        string         `json:"code"`
	Name        string         `json:"name"`
	Module      string         `json:"module"`
	Description string         `json:"description"`
	RiskLevel   rbac.RiskLevel `json:"risk_level"`
	Status      string         `json:"status"`
}

func (h *RBACHandler) CreatePermission(c *gin.Context) {
	var req permissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	item, err := h.service.CreatePermission(c.Request.Context(), rbacActor(c), req.Code, req.Name, req.Module, req.Description, req.RiskLevel)
	if err != nil {
		rbacMutationError(c, err)
		return
	}
	response.Created(c, item)
}
func (h *RBACHandler) UpdatePermission(c *gin.Context) {
	id, ok := rbacPermissionID(c)
	if !ok {
		return
	}
	var req permissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}
	item, err := h.service.UpdatePermission(c.Request.Context(), rbacActor(c), id, req.Name, req.Module, req.Description, req.RiskLevel, req.Status)
	if err != nil {
		rbacMutationError(c, err)
		return
	}
	response.Success(c, item)
}
func (h *RBACHandler) DeletePermission(c *gin.Context) {
	id, ok := rbacPermissionID(c)
	if !ok {
		return
	}
	if err := h.service.DeletePermission(c.Request.Context(), rbacActor(c), id); err != nil {
		rbacMutationError(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
func (h *RBACHandler) GetRolePermissions(c *gin.Context) {
	id, ok := rbacRoleID(c)
	if !ok {
		return
	}
	values, err := h.service.Permissions(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"permissions": values})
}
func (h *RBACHandler) ReplaceRolePermissions(c *gin.Context) {
	id, ok := rbacRoleID(c)
	if !ok {
		return
	}
	var req struct {
		Permissions []string `json:"permissions" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "permissions is required")
		return
	}
	if err := h.service.ReplacePermissions(c.Request.Context(), rbacActor(c), id, req.Permissions); err != nil {
		rbacMutationError(c, err)
		return
	}
	response.Success(c, gin.H{"permissions": rbac.SortedUnique(req.Permissions)})
}
func (h *RBACHandler) GetUserRoles(c *gin.Context) {
	userID, ok := rbacUserID(c)
	if !ok {
		return
	}
	values, err := h.service.UserRoles(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"roles": values})
}
func (h *RBACHandler) ReplaceUserRoles(c *gin.Context) {
	userID, ok := rbacUserID(c)
	if !ok {
		return
	}
	var req struct {
		Roles []string `json:"roles" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Roles) == 0 {
		response.BadRequest(c, "roles is required")
		return
	}
	principal, _ := rbac.PrincipalFromContext(c)
	if err := h.service.ReplaceUserRoles(c.Request.Context(), rbacActor(c), principal.SuperAdmin, userID, req.Roles); err != nil {
		rbacMutationError(c, err)
		return
	}
	response.Success(c, gin.H{"roles": rbac.SortedUnique(req.Roles)})
}
func rbacUserID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		response.BadRequest(c, "invalid user id")
		return 0, false
	}
	return id, true
}
func rbacRoleID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		response.BadRequest(c, "invalid role id")
		return 0, false
	}
	return id, true
}
func rbacPermissionID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		response.BadRequest(c, "invalid permission id")
		return 0, false
	}
	return id, true
}
func rbacActor(c *gin.Context) rbac.AuditActor {
	id := getAdminIDFromContext(c)
	var ptr *int64
	if id > 0 {
		ptr = &id
	}
	return rbac.AuditActor{UserID: ptr, RequestID: c.GetString("request_id"), IPAddress: c.ClientIP()}
}
func rbacMutationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, rbac.ErrSystemRoleProtected), errors.Is(err, rbac.ErrSystemPermissionProtected), errors.Is(err, rbac.ErrWildcardReserved), errors.Is(err, service.ErrInvalidRBACRoleCode), errors.Is(err, service.ErrInvalidRBACPermissionCode):
		response.Error(c, http.StatusBadRequest, err.Error())
	default:
		response.ErrorFrom(c, err)
	}
}
