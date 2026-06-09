-- Tenant lifecycle: status column, platform_admin role and seed user.

ALTER TABLE tenants ADD COLUMN IF NOT EXISTS status VARCHAR(32) NOT NULL DEFAULT 'active';

INSERT INTO roles (id, tenant_id, name, description)
VALUES (
    '00000000-0000-0000-0000-000000000030',
    '00000000-0000-0000-0000-000000000001',
    'platform_admin',
    'Platform tenant administrator (tenant lifecycle only)'
)
ON CONFLICT (tenant_id, name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-0000-0000-000000000030', p.id
FROM permissions p
WHERE p.code IN ('tenant.read', 'tenant.manage', 'user.read')
ON CONFLICT DO NOTHING;

-- platform / Platform@123456
INSERT INTO users (id, tenant_id, username, password_hash, display_name)
VALUES (
    '00000000-0000-0000-0000-000000000300',
    '00000000-0000-0000-0000-000000000001',
    'platform',
    '$2a$10$BEN.r3orJI.OlUNZGdkdR.EQ2iyxGA5nSFU/vSD96y6fxpCx.gQNG',
    'Platform Admin'
)
ON CONFLICT (tenant_id, username) DO NOTHING;

INSERT INTO user_roles (user_id, role_id)
VALUES (
    '00000000-0000-0000-0000-000000000300',
    '00000000-0000-0000-0000-000000000030'
)
ON CONFLICT DO NOTHING;
