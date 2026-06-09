package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gateframe/user-service/internal/config"
	"github.com/gateframe/user-service/internal/model"
	"github.com/gateframe/user-service/internal/permversion"
	"github.com/gateframe/user-service/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserDisabled       = errors.New("user disabled")
	ErrTenantDisabled     = errors.New("tenant disabled")
)

type AuthService struct {
	repo   *repository.Repository
	secret []byte
	expiry time.Duration
	perm   permversion.Store
}

func NewAuthService(repo *repository.Repository, cfg config.Config, perm permversion.Store) *AuthService {
	if perm == nil {
		perm = permversion.Noop{}
	}
	return &AuthService{
		repo:   repo,
		secret: []byte(cfg.JWTSecret),
		expiry: time.Duration(cfg.JWTExpirySecs) * time.Second,
		perm:   perm,
	}
}

type jwtClaims struct {
	Sub         string   `json:"sub"`
	TenantID    string   `json:"tenant_id"`
	Permissions []string `json:"permissions"`
	PermVer     uint64   `json:"perm_ver"`
	jwt.RegisteredClaims
}

func (s *AuthService) Login(ctx context.Context, req model.LoginRequest) (*model.LoginResponse, error) {
	tenantID, err := s.repo.GetTenantIDBySlug(ctx, req.TenantSlug)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	user, err := s.repo.GetUserByUsername(ctx, tenantID, req.Username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if user.Status != "active" {
		return nil, ErrUserDisabled
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	perms, err := s.repo.ListPermissionsForUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	token, err := s.issueToken(ctx, user, perms)
	if err != nil {
		return nil, err
	}
	return s.loginResponse(user, perms, token), nil
}

func (s *AuthService) loginResponse(user *model.User, perms []string, token string) *model.LoginResponse {
	return &model.LoginResponse{
		Token:       token,
		UserID:      user.ID.String(),
		TenantID:    user.TenantID.String(),
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Permissions: perms,
	}
}

func (s *AuthService) issueToken(ctx context.Context, user *model.User, perms []string) (string, error) {
	ver, err := s.perm.Current(ctx, user.ID)
	if err != nil {
		return "", err
	}
	now := time.Now()
	claims := jwtClaims{
		Sub:         user.ID.String(),
		TenantID:    user.TenantID.String(),
		Permissions: perms,
		PermVer:     ver,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expiry)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(s.secret)
}

func (s *AuthService) PermissionsForUser(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return s.repo.ListPermissionsForUser(ctx, userID)
}

// RefreshSession re-issues a JWT with the user's current permissions from the database.
func (s *AuthService) RefreshSession(ctx context.Context, tenantID, userID uuid.UUID) (*model.LoginResponse, error) {
	status, err := s.repo.GetTenantStatus(ctx, tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if status != "active" {
		return nil, ErrTenantDisabled
	}
	user, err := s.repo.GetUserByID(ctx, tenantID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if user.Status != "active" {
		return nil, ErrUserDisabled
	}
	perms, err := s.repo.ListPermissionsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	token, err := s.issueToken(ctx, user, perms)
	if err != nil {
		return nil, err
	}
	return s.loginResponse(user, perms, token), nil
}

// ResolveOidc links or looks up a user by OIDC subject and issues a JWT.
func (s *AuthService) ResolveOidc(ctx context.Context, req model.OidcResolveRequest) (*model.LoginResponse, error) {
	tenantID, err := s.repo.GetTenantIDBySlug(ctx, req.TenantSlug)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if req.Subject == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.GetUserByOidcSub(ctx, tenantID, req.Subject)
	if errors.Is(err, repository.ErrNotFound) {
		username := req.Username
		if username == "" && req.Email != "" {
			if local, _, ok := strings.Cut(req.Email, "@"); ok && local != "" {
				username = local
			}
		}
		if username == "" {
			return nil, ErrInvalidCredentials
		}
		user, err = s.repo.GetUserByUsername(ctx, tenantID, username)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, ErrInvalidCredentials
			}
			return nil, err
		}
		if err := s.repo.LinkUserOidcSub(ctx, tenantID, user.ID, req.Subject); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	if user.Status != "active" {
		return nil, ErrUserDisabled
	}
	perms, err := s.repo.ListPermissionsForUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	token, err := s.issueToken(ctx, user, perms)
	if err != nil {
		return nil, err
	}
	return s.loginResponse(user, perms, token), nil
}

type UserService struct {
	repo *repository.Repository
}

func NewUserService(repo *repository.Repository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) List(ctx context.Context, tenantID uuid.UUID) ([]model.User, error) {
	return s.repo.ListUsers(ctx, tenantID)
}

func (s *UserService) Get(ctx context.Context, tenantID, userID uuid.UUID) (*model.User, error) {
	return s.repo.GetUserByID(ctx, tenantID, userID)
}

func (s *UserService) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateUserRequest) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.Username
	}
	return s.repo.CreateUser(ctx, tenantID, req.Username, string(hash), displayName)
}

func (s *UserService) Update(ctx context.Context, tenantID, userID uuid.UUID, req model.UpdateUserRequest) (*model.User, error) {
	var hash *string
	if req.Password != nil {
		h, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		s := string(h)
		hash = &s
	}
	return s.repo.UpdateUser(ctx, tenantID, userID, req.DisplayName, req.Status, hash)
}

func (s *UserService) Delete(ctx context.Context, tenantID, userID uuid.UUID) error {
	return s.repo.DeleteUser(ctx, tenantID, userID)
}

type RBACService struct {
	repo *repository.Repository
	perm permversion.Store
}

func NewRBACService(repo *repository.Repository, perm permversion.Store) *RBACService {
	if perm == nil {
		perm = permversion.Noop{}
	}
	return &RBACService{repo: repo, perm: perm}
}

func (s *RBACService) ListRoles(ctx context.Context, tenantID uuid.UUID) ([]model.Role, error) {
	return s.repo.ListRoles(ctx, tenantID)
}

func (s *RBACService) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	return s.repo.ListPermissions(ctx)
}

func (s *RBACService) ListPermissionsForRole(ctx context.Context, tenantID, roleID uuid.UUID) ([]model.Permission, error) {
	return s.repo.ListPermissionsForRole(ctx, tenantID, roleID)
}

func (s *RBACService) ListRolePermissions(ctx context.Context, tenantID uuid.UUID) ([]model.RolePermissions, error) {
	return s.repo.ListRolePermissionsByTenant(ctx, tenantID)
}

func (s *RBACService) SetRolePermissions(ctx context.Context, tenantID, roleID uuid.UUID, codes []string) error {
	if err := s.repo.SetRolePermissions(ctx, tenantID, roleID, codes); err != nil {
		return err
	}
	userIDs, err := s.repo.ListUserIDsByRole(ctx, roleID)
	if err != nil {
		return err
	}
	return s.perm.BumpUsers(ctx, userIDs)
}

type TenantService struct {
	repo *repository.Repository
}

func NewTenantService(repo *repository.Repository) *TenantService {
	return &TenantService{repo: repo}
}

func (s *TenantService) List(ctx context.Context) ([]model.Tenant, error) {
	return s.repo.ListTenants(ctx)
}

func (s *TenantService) Create(ctx context.Context, req model.CreateTenantRequest) (*model.Tenant, error) {
	return s.repo.CreateTenant(ctx, req.Name, req.Slug)
}

func (s *TenantService) Update(ctx context.Context, tenantID uuid.UUID, req model.UpdateTenantRequest) (*model.Tenant, error) {
	return s.repo.UpdateTenant(ctx, tenantID, req.Name, req.Status)
}

func (s *TenantService) GetByID(ctx context.Context, tenantID uuid.UUID) (*model.Tenant, error) {
	return s.repo.GetTenantByID(ctx, tenantID)
}

type DashboardService struct {
	repo *repository.Repository
}

func NewDashboardService(repo *repository.Repository) *DashboardService {
	return &DashboardService{repo: repo}
}

func (s *DashboardService) Stats(ctx context.Context, auth model.AuthContext) (*model.DashboardStats, error) {
	stats := &model.DashboardStats{}
	if auth.HasPermission("user.read") {
		n, err := s.repo.CountByTable(ctx, "users", &auth.TenantID, nil)
		if err != nil {
			return nil, err
		}
		stats.Users = &n
	}
	if auth.HasPermission("rbac.read") {
		n, err := s.repo.CountByTable(ctx, "roles", &auth.TenantID, nil)
		if err != nil {
			return nil, err
		}
		stats.Roles = &n
	}
	if auth.HasPermission("workflow.read") {
		n, err := s.repo.CountByTable(ctx, "workflows", &auth.TenantID, nil)
		if err != nil {
			return nil, err
		}
		stats.Workflows = &n
	}
	if auth.HasPermission("search.read") {
		n, err := s.repo.CountByTable(ctx, "search_documents", &auth.TenantID, nil)
		if err != nil {
			return nil, err
		}
		stats.SearchDocs = &n
	}
	if auth.HasPermission("notification.read") {
		n, err := s.repo.CountByTable(ctx, "notifications", &auth.TenantID, &auth.UserID)
		if err != nil {
			return nil, err
		}
		stats.Notifications = &n
	}
	if auth.HasPermission("file.read") {
		n, err := s.repo.CountByTable(ctx, "file_objects", &auth.TenantID, nil)
		if err != nil {
			return nil, err
		}
		stats.Files = &n
	}
	if auth.HasPermission("audit.read") {
		n, err := s.repo.CountByTable(ctx, "audit_events", &auth.TenantID, nil)
		if err != nil {
			return nil, err
		}
		stats.AuditEvents = &n
	}
	if auth.HasPermission("tenant.read") {
		n, err := s.repo.CountByTable(ctx, "tenants", nil, nil)
		if err != nil {
			return nil, err
		}
		stats.Tenants = &n
	}
	return stats, nil
}

func HashPassword(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(h), err
}
