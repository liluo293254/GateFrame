-- 002_audit_read_permission.up.sql
INSERT INTO permissions (code, description) VALUES
    ('audit.read', 'Read audit logs')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-0000-0000-000000000010', p.id
FROM permissions p
WHERE p.code = 'audit.read'
ON CONFLICT DO NOTHING;
