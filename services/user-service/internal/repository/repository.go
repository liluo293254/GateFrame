package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/gateframe/user-service/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")
var ErrInvalidPermission = errors.New("invalid permission code")

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) GetTenantIDBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	if slug == "" {
		slug = "default"
	}
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT id FROM tenants WHERE slug = $1 AND status = 'active'`, slug).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrNotFound
		}
		return uuid.Nil, err
	}
	return id, nil
}

func (r *Repository) GetUserByUsername(ctx context.Context, tenantID uuid.UUID, username string) (*model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, username, password_hash, display_name, status, created_at
		FROM users WHERE tenant_id = $1 AND username = $2
	`, tenantID, username).Scan(
		&u.ID, &u.TenantID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Status, &u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *Repository) ListUserIDsByRole(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `SELECT user_id FROM user_roles WHERE role_id = $1`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *Repository) GetUserByOidcSub(ctx context.Context, tenantID uuid.UUID, oidcSub string) (*model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, username, password_hash, display_name, status, created_at
		FROM users WHERE tenant_id = $1 AND oidc_sub = $2
	`, tenantID, oidcSub).Scan(
		&u.ID, &u.TenantID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Status, &u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *Repository) LinkUserOidcSub(ctx context.Context, tenantID, userID uuid.UUID, oidcSub string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users SET oidc_sub = $3
		WHERE tenant_id = $1 AND id = $2 AND (oidc_sub IS NULL OR oidc_sub = $3)
	`, tenantID, userID, oidcSub)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) ListPermissionsForUser(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT p.code
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		JOIN user_roles ur ON ur.role_id = rp.role_id
		WHERE ur.user_id = $1
		ORDER BY p.code
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var codes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, rows.Err()
}

func (r *Repository) ListUsers(ctx context.Context, tenantID uuid.UUID) ([]model.User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, username, password_hash, display_name, status, created_at
		FROM users WHERE tenant_id = $1 ORDER BY username
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.TenantID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Status, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *Repository) GetUserByID(ctx context.Context, tenantID, userID uuid.UUID) (*model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, username, password_hash, display_name, status, created_at
		FROM users WHERE tenant_id = $1 AND id = $2
	`, tenantID, userID).Scan(
		&u.ID, &u.TenantID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Status, &u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *Repository) CreateUser(ctx context.Context, tenantID uuid.UUID, username, hash, displayName string) (*model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (tenant_id, username, password_hash, display_name)
		VALUES ($1, $2, $3, $4)
		RETURNING id, tenant_id, username, password_hash, display_name, status, created_at
	`, tenantID, username, hash, displayName).Scan(
		&u.ID, &u.TenantID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Status, &u.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &u, nil
}

func (r *Repository) UpdateUser(ctx context.Context, tenantID, userID uuid.UUID, displayName, status, passwordHash *string) (*model.User, error) {
	u, err := r.GetUserByID(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	if displayName != nil {
		u.DisplayName = *displayName
	}
	if status != nil {
		u.Status = *status
	}
	if passwordHash != nil {
		u.PasswordHash = *passwordHash
	}
	err = r.pool.QueryRow(ctx, `
		UPDATE users SET display_name = $3, status = $4, password_hash = $5
		WHERE tenant_id = $1 AND id = $2
		RETURNING id, tenant_id, username, password_hash, display_name, status, created_at
	`, tenantID, userID, u.DisplayName, u.Status, u.PasswordHash).Scan(
		&u.ID, &u.TenantID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Status, &u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *Repository) DeleteUser(ctx context.Context, tenantID, userID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE tenant_id = $1 AND id = $2`, tenantID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) ListRoles(ctx context.Context, tenantID uuid.UUID) ([]model.Role, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, name, description FROM roles WHERE tenant_id = $1 ORDER BY name
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roles []model.Role
	for rows.Next() {
		var role model.Role
		if err := rows.Scan(&role.ID, &role.TenantID, &role.Name, &role.Description); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (r *Repository) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, code, description FROM permissions ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var perms []model.Permission
	for rows.Next() {
		var p model.Permission
		if err := rows.Scan(&p.ID, &p.Code, &p.Description); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (r *Repository) ListPermissionsForRole(ctx context.Context, tenantID, roleID uuid.UUID) ([]model.Permission, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.code, p.description
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		JOIN roles r ON r.id = rp.role_id
		WHERE r.tenant_id = $1 AND r.id = $2
		ORDER BY p.code
	`, tenantID, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var perms []model.Permission
	for rows.Next() {
		var p model.Permission
		if err := rows.Scan(&p.ID, &p.Code, &p.Description); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return perms, nil
}

func (r *Repository) ListRolePermissionsByTenant(ctx context.Context, tenantID uuid.UUID) ([]model.RolePermissions, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT r.id, COALESCE(array_agg(p.code ORDER BY p.code) FILTER (WHERE p.code IS NOT NULL), '{}')
		FROM roles r
		LEFT JOIN role_permissions rp ON rp.role_id = r.id
		LEFT JOIN permissions p ON p.id = rp.permission_id
		WHERE r.tenant_id = $1
		GROUP BY r.id
		ORDER BY r.id
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bindings []model.RolePermissions
	for rows.Next() {
		var binding model.RolePermissions
		if err := rows.Scan(&binding.RoleID, &binding.Permissions); err != nil {
			return nil, err
		}
		bindings = append(bindings, binding)
	}
	return bindings, rows.Err()
}

func (r *Repository) SetRolePermissions(ctx context.Context, tenantID, roleID uuid.UUID, codes []string) error {
	var exists bool
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM roles WHERE tenant_id = $1 AND id = $2)
	`, tenantID, roleID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, roleID); err != nil {
		return err
	}
	if len(codes) == 0 {
		return tx.Commit(ctx)
	}

	rows, err := tx.Query(ctx, `
		SELECT id, code FROM permissions WHERE code = ANY($1)
	`, codes)
	if err != nil {
		return err
	}
	defer rows.Close()

	found := make(map[string]uuid.UUID, len(codes))
	for rows.Next() {
		var id uuid.UUID
		var code string
		if err := rows.Scan(&id, &code); err != nil {
			return err
		}
		found[code] = id
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, code := range codes {
		pid, ok := found[code]
		if !ok {
			return fmt.Errorf("%w: %s", ErrInvalidPermission, code)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, roleID, pid); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) ListTenants(ctx context.Context) ([]model.Tenant, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, slug, status, created_at FROM tenants ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []model.Tenant
	for rows.Next() {
		var t model.Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Status, &t.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, rows.Err()
}

func (r *Repository) CreateTenant(ctx context.Context, name, slug string) (*model.Tenant, error) {
	var t model.Tenant
	err := r.pool.QueryRow(ctx, `
		INSERT INTO tenants (name, slug, status) VALUES ($1, $2, 'active')
		RETURNING id, name, slug, status, created_at
	`, name, slug).Scan(&t.ID, &t.Name, &t.Slug, &t.Status, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) UpdateTenant(ctx context.Context, tenantID uuid.UUID, name *string, status *string) (*model.Tenant, error) {
	current, err := r.GetTenantByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	nextName := current.Name
	if name != nil {
		nextName = *name
	}
	nextStatus := current.Status
	if status != nil {
		nextStatus = *status
	}
	var t model.Tenant
	err = r.pool.QueryRow(ctx, `
		UPDATE tenants SET name = $2, status = $3
		WHERE id = $1
		RETURNING id, name, slug, status, created_at
	`, tenantID, nextName, nextStatus).Scan(&t.ID, &t.Name, &t.Slug, &t.Status, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *Repository) GetTenantByID(ctx context.Context, tenantID uuid.UUID) (*model.Tenant, error) {
	var t model.Tenant
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, slug, status, created_at FROM tenants WHERE id = $1
	`, tenantID).Scan(&t.ID, &t.Name, &t.Slug, &t.Status, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r *Repository) GetTenantStatus(ctx context.Context, tenantID uuid.UUID) (string, error) {
	var status string
	err := r.pool.QueryRow(ctx, `SELECT status FROM tenants WHERE id = $1`, tenantID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return status, nil
}

func (r *Repository) CountByTable(ctx context.Context, table string, tenantID *uuid.UUID, userID *uuid.UUID) (int64, error) {
	var count int64
	var err error
	switch table {
	case "users":
		err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE tenant_id = $1`, *tenantID).Scan(&count)
	case "roles":
		err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM roles WHERE tenant_id = $1`, *tenantID).Scan(&count)
	case "workflows":
		err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM workflows WHERE tenant_id = $1`, *tenantID).Scan(&count)
	case "search_documents":
		err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM search_documents WHERE tenant_id = $1`, *tenantID).Scan(&count)
	case "notifications":
		if userID != nil {
			err = r.pool.QueryRow(ctx, `
				SELECT COUNT(*) FROM notifications
				WHERE tenant_id = $1 AND (user_id IS NULL OR user_id = $2)
			`, *tenantID, *userID).Scan(&count)
		} else {
			err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE tenant_id = $1`, *tenantID).Scan(&count)
		}
	case "file_objects":
		err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM file_objects WHERE tenant_id = $1`, *tenantID).Scan(&count)
	case "audit_events":
		err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_events WHERE tenant_id = $1`, *tenantID).Scan(&count)
	case "tenants":
		err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tenants`).Scan(&count)
	default:
		return 0, fmt.Errorf("unsupported table %q", table)
	}
	if err != nil {
		return 0, err
	}
	return count, nil
}
