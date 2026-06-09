-- 001_init.up.sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(64) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    username VARCHAR(64) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, username)
);

CREATE INDEX IF NOT EXISTS idx_users_tenant ON users (tenant_id);

CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(128) NOT NULL UNIQUE,
    description VARCHAR(255) NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(128) NOT NULL,
    description VARCHAR(255) NOT NULL DEFAULT '',
    UNIQUE (tenant_id, name)
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

-- Seed default tenant
INSERT INTO tenants (id, name, slug)
VALUES ('00000000-0000-0000-0000-000000000001', 'Default Tenant', 'default')
ON CONFLICT (slug) DO NOTHING;

-- Seed permissions
INSERT INTO permissions (code, description) VALUES
    ('user.read', 'Read users'),
    ('user.create', 'Create users'),
    ('user.update', 'Update users'),
    ('user.delete', 'Delete users'),
    ('rbac.read', 'Read roles and permissions'),
    ('rbac.manage', 'Manage roles and permissions')
ON CONFLICT (code) DO NOTHING;

-- Seed admin role
INSERT INTO roles (id, tenant_id, name, description)
VALUES (
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    'admin',
    'Tenant administrator'
)
ON CONFLICT (tenant_id, name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-0000-0000-000000000010', p.id
FROM permissions p
ON CONFLICT DO NOTHING;

-- Default admin: username admin / password Admin@123456 (bcrypt hash below)
INSERT INTO users (id, tenant_id, username, password_hash, display_name)
VALUES (
    '00000000-0000-0000-0000-000000000100',
    '00000000-0000-0000-0000-000000000001',
    'admin',
    '$2a$10$rdLZIa3luMiNClizZTbZX.S3W3AW3Pk9I084s8iIQ70SkLsqfiExa',
    'Administrator'
)
ON CONFLICT (tenant_id, username) DO NOTHING;

INSERT INTO user_roles (user_id, role_id)
VALUES (
    '00000000-0000-0000-0000-000000000100',
    '00000000-0000-0000-0000-000000000010'
)
ON CONFLICT DO NOTHING;
