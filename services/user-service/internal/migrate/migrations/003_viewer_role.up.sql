-- Seed read-only viewer role and user for RBAC gate testing (no rbac.manage).

INSERT INTO roles (id, tenant_id, name, description)
VALUES (
    '00000000-0000-0000-0000-000000000020',
    '00000000-0000-0000-0000-000000000001',
    'viewer',
    'Read-only tenant viewer'
)
ON CONFLICT (tenant_id, name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-0000-0000-000000000020', p.id
FROM permissions p
WHERE p.code IN ('user.read', 'rbac.read', 'audit.read')
ON CONFLICT DO NOTHING;

-- viewer / Viewer@123456
INSERT INTO users (id, tenant_id, username, password_hash, display_name)
VALUES (
    '00000000-0000-0000-0000-000000000200',
    '00000000-0000-0000-0000-000000000001',
    'viewer',
    '$2a$10$xnzQjkpjs/zdcEe/vKHYyee.7ApxOYnHOWHQjfysxh3Jze5YQwlki',
    'Viewer'
)
ON CONFLICT (tenant_id, username) DO NOTHING;

INSERT INTO user_roles (user_id, role_id)
VALUES (
    '00000000-0000-0000-0000-000000000200',
    '00000000-0000-0000-0000-000000000020'
)
ON CONFLICT DO NOTHING;
