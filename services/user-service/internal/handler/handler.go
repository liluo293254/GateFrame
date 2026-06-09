package handler

import (
	"errors"

	"github.com/gateframe/user-service/internal/middleware"
	"github.com/gateframe/user-service/internal/model"
	"github.com/gateframe/user-service/internal/repository"
	"github.com/gateframe/user-service/internal/response"
	"github.com/gateframe/user-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	res, err := h.auth.Login(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) || errors.Is(err, service.ErrUserDisabled) {
			response.Unauthorized(c)
			return
		}
		response.Internal(c)
		return
	}
	response.OK(c, res)
}

func (h *AuthHandler) Permissions(c *gin.Context) {
	userID, tenantID, err := parseIdentity(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	perms, err := h.auth.PermissionsForUser(c.Request.Context(), userID)
	if err != nil {
		response.Internal(c)
		return
	}
	response.OK(c, gin.H{
		"user_id":     userID.String(),
		"tenant_id":   tenantID.String(),
		"permissions": perms,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	// Stateless JWT: client drops token; audit handled at Gateway.
	response.OK(c, gin.H{"message": "logged out"})
}

func (h *AuthHandler) InternalOidcResolve(c *gin.Context) {
	var req model.OidcResolveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	res, err := h.auth.ResolveOidc(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) || errors.Is(err, service.ErrUserDisabled) {
			response.Unauthorized(c)
			return
		}
		response.Internal(c)
		return
	}
	response.OK(c, res)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	userID, tenantID, err := parseIdentity(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	res, err := h.auth.RefreshSession(c.Request.Context(), tenantID, userID)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) ||
			errors.Is(err, service.ErrUserDisabled) ||
			errors.Is(err, service.ErrTenantDisabled) {
			response.Unauthorized(c)
			return
		}
		response.Internal(c)
		return
	}
	response.OK(c, res)
}

type UserHandler struct {
	users *service.UserService
}

func NewUserHandler(users *service.UserService) *UserHandler {
	return &UserHandler{users: users}
}

func (h *UserHandler) List(c *gin.Context) {
	tenantID, err := tenantFromContext(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	users, err := h.users.List(c.Request.Context(), tenantID)
	if err != nil {
		response.Internal(c)
		return
	}
	response.OK(c, users)
}

func (h *UserHandler) Get(c *gin.Context) {
	tenantID, err := tenantFromContext(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	user, err := h.users.Get(c.Request.Context(), tenantID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.Internal(c)
		return
	}
	response.OK(c, user)
}

func (h *UserHandler) Create(c *gin.Context) {
	tenantID, err := tenantFromContext(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	var req model.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	user, err := h.users.Create(c.Request.Context(), tenantID, req)
	if err != nil {
		response.Internal(c)
		return
	}
	response.Created(c, user)
}

func (h *UserHandler) Update(c *gin.Context) {
	tenantID, err := tenantFromContext(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	var req model.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	user, err := h.users.Update(c.Request.Context(), tenantID, userID, req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.Internal(c)
		return
	}
	response.OK(c, user)
}

func (h *UserHandler) Delete(c *gin.Context) {
	tenantID, err := tenantFromContext(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	if err := h.users.Delete(c.Request.Context(), tenantID, userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.Internal(c)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

type RBACHandler struct {
	rbac *service.RBACService
}

func NewRBACHandler(rbac *service.RBACService) *RBACHandler {
	return &RBACHandler{rbac: rbac}
}

func (h *RBACHandler) ListRoles(c *gin.Context) {
	tenantID, err := tenantFromContext(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	roles, err := h.rbac.ListRoles(c.Request.Context(), tenantID)
	if err != nil {
		response.Internal(c)
		return
	}
	response.OK(c, roles)
}

func (h *RBACHandler) ListPermissions(c *gin.Context) {
	perms, err := h.rbac.ListPermissions(c.Request.Context())
	if err != nil {
		response.Internal(c)
		return
	}
	response.OK(c, perms)
}

func (h *RBACHandler) ListRolePermissions(c *gin.Context) {
	tenantID, err := tenantFromContext(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	bindings, err := h.rbac.ListRolePermissions(c.Request.Context(), tenantID)
	if err != nil {
		response.Internal(c)
		return
	}
	response.OK(c, bindings)
}

func (h *RBACHandler) GetRolePermissions(c *gin.Context) {
	tenantID, err := tenantFromContext(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid role id")
		return
	}
	perms, err := h.rbac.ListPermissionsForRole(c.Request.Context(), tenantID, roleID)
	if err != nil {
		response.Internal(c)
		return
	}
	response.OK(c, perms)
}

func (h *RBACHandler) UpdateRolePermissions(c *gin.Context) {
	tenantID, err := tenantFromContext(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid role id")
		return
	}
	var req model.UpdateRolePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	err = h.rbac.SetRolePermissions(c.Request.Context(), tenantID, roleID, req.Permissions)
	if errors.Is(err, repository.ErrNotFound) {
		response.Error(c, 404, "not_found", "role not found")
		return
	}
	if errors.Is(err, repository.ErrInvalidPermission) {
		response.BadRequest(c, err.Error())
		return
	}
	if err != nil {
		response.Internal(c)
		return
	}
	perms, err := h.rbac.ListPermissionsForRole(c.Request.Context(), tenantID, roleID)
	if err != nil {
		response.Internal(c)
		return
	}
	response.OK(c, perms)
}

type TenantHandler struct {
	tenants *service.TenantService
}

func NewTenantHandler(tenants *service.TenantService) *TenantHandler {
	return &TenantHandler{tenants: tenants}
}

func (h *TenantHandler) List(c *gin.Context) {
	items, err := h.tenants.List(c.Request.Context())
	if err != nil {
		response.Internal(c)
		return
	}
	if items == nil {
		items = []model.Tenant{}
	}
	response.OK(c, items)
}

func (h *TenantHandler) Create(c *gin.Context) {
	var req model.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	tenant, err := h.tenants.Create(c.Request.Context(), req)
	if err != nil {
		response.Internal(c)
		return
	}
	response.Created(c, tenant)
}

func (h *TenantHandler) Update(c *gin.Context) {
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid tenant id")
		return
	}
	callerTenantID, err := tenantFromContext(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	// Platform scope (tenant.manage) may update any tenant; others may only touch own tenant.
	if targetID != callerTenantID && !hasPermission(c, "tenant.manage") {
		response.Forbidden(c)
		return
	}
	var req model.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if req.Name == nil && req.Status == nil {
		response.BadRequest(c, "name or status required")
		return
	}
	tenant, err := h.tenants.Update(c.Request.Context(), targetID, req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.Internal(c)
		return
	}
	response.OK(c, tenant)
}

func (h *TenantHandler) InternalStatus(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid tenant id")
		return
	}
	tenant, err := h.tenants.GetByID(c.Request.Context(), tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.Internal(c)
		return
	}
	response.OK(c, gin.H{"status": tenant.Status})
}

type DashboardHandler struct {
	dashboard *service.DashboardService
}

func NewDashboardHandler(dashboard *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboard: dashboard}
}

func (h *DashboardHandler) Stats(c *gin.Context) {
	userID, tenantID, err := parseIdentity(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	perms, _ := c.Get("permissions")
	permList, _ := perms.([]string)
	auth := model.AuthContext{
		UserID:   userID,
		TenantID: tenantID,
		Perms:    permList,
	}
	stats, err := h.dashboard.Stats(c.Request.Context(), auth)
	if err != nil {
		response.Internal(c)
		return
	}
	response.OK(c, stats)
}

func Health(c *gin.Context) {
	response.OK(c, gin.H{"status": "ok"})
}

func parseIdentity(c *gin.Context) (uuid.UUID, uuid.UUID, error) {
	userStr, _ := c.Get("user_id")
	tenantStr, _ := c.Get("tenant_id")
	userID, err := uuid.Parse(userStr.(string))
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	tenantID, err := uuid.Parse(tenantStr.(string))
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return userID, tenantID, nil
}

func tenantFromContext(c *gin.Context) (uuid.UUID, error) {
	tenantStr, ok := c.Get("tenant_id")
	if !ok {
		return uuid.Nil, errors.New("missing tenant")
	}
	return uuid.Parse(tenantStr.(string))
}

func hasPermission(c *gin.Context, code string) bool {
	perms, ok := c.Get("permissions")
	if !ok {
		return false
	}
	list, ok := perms.([]string)
	if !ok {
		return false
	}
	for _, p := range list {
		if p == code || p == "*" {
			return true
		}
	}
	return false
}

// LoadPermissionsMiddleware loads permissions for authenticated routes.
func LoadPermissionsMiddleware(auth *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, _ := c.Get("user_id")
		userID, err := uuid.Parse(userIDStr.(string))
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}
		perms, err := auth.PermissionsForUser(c.Request.Context(), userID)
		if err != nil {
			response.Internal(c)
			c.Abort()
			return
		}
		c.Set("permissions", perms)
		c.Next()
	}
}

// Ensure middleware import used in routes file
var _ = middleware.RequireInternalToken
