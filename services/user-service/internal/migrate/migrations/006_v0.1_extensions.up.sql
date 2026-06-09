-- 006_v0.1_extensions.up.sql
INSERT INTO permissions (code, description) VALUES
    ('tenant.read', 'Read tenants'),
    ('tenant.manage', 'Manage tenants'),
    ('file.read', 'Read files'),
    ('file.manage', 'Upload and manage files')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-0000-0000-000000000010', p.id
FROM permissions p
WHERE p.code IN ('tenant.read', 'tenant.manage', 'file.read', 'file.manage')
ON CONFLICT DO NOTHING;
