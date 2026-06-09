package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	Username     string    `json:"username"`
	DisplayName  string    `json:"display_name"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	PasswordHash string    `json:"-"`
}

type Role struct {
	ID          uuid.UUID `json:"id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

type RolePermissions struct {
	RoleID      uuid.UUID `json:"role_id"`
	Permissions []string  `json:"permissions"`
}

type UpdateRolePermissionsRequest struct {
	Permissions []string `json:"permissions" binding:"required"`
}

type Permission struct {
	ID          uuid.UUID `json:"id"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
}

type LoginRequest struct {
	Username   string `json:"username" binding:"required,min=2,max=64"`
	Password   string `json:"password" binding:"required,min=6,max=128"`
	TenantSlug string `json:"tenant_slug" binding:"omitempty,max=64"`
}

type LoginResponse struct {
	Token       string   `json:"token"`
	UserID      string   `json:"user_id"`
	TenantID    string   `json:"tenant_id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"display_name"`
	Permissions []string `json:"permissions"`
}

type OidcResolveRequest struct {
	Subject    string `json:"subject" binding:"required,max=255"`
	Username   string `json:"username" binding:"omitempty,max=64"`
	Email      string `json:"email" binding:"omitempty,max=255"`
	TenantSlug string `json:"tenant_slug" binding:"omitempty,max=64"`
}

type CreateUserRequest struct {
	Username    string `json:"username" binding:"required,min=2,max=64"`
	Password    string `json:"password" binding:"required,min=8,max=128"`
	DisplayName string `json:"display_name" binding:"max=255"`
}

type UpdateUserRequest struct {
	DisplayName *string `json:"display_name" binding:"omitempty,max=255"`
	Status      *string `json:"status" binding:"omitempty,oneof=active disabled"`
	Password    *string `json:"password" binding:"omitempty,min=8,max=128"`
}

type AuthContext struct {
	UserID   uuid.UUID
	TenantID uuid.UUID
	Perms    []string
}

type Tenant struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateTenantRequest struct {
	Name string `json:"name" binding:"required,min=1,max=255"`
	Slug string `json:"slug" binding:"required,min=2,max=64"`
}

type UpdateTenantRequest struct {
	Name   *string `json:"name" binding:"omitempty,min=1,max=255"`
	Status *string `json:"status" binding:"omitempty,oneof=active disabled"`
}

type DashboardStats struct {
	Users         *int64 `json:"users,omitempty"`
	Roles         *int64 `json:"roles,omitempty"`
	Workflows     *int64 `json:"workflows,omitempty"`
	SearchDocs    *int64 `json:"search_documents,omitempty"`
	Notifications *int64 `json:"notifications,omitempty"`
	Files         *int64 `json:"files,omitempty"`
	AuditEvents   *int64 `json:"audit_events,omitempty"`
	Tenants       *int64 `json:"tenants,omitempty"`
}

func (a AuthContext) HasPermission(code string) bool {
	for _, p := range a.Perms {
		if p == code || p == "*" {
			return true
		}
	}
	return false
}
