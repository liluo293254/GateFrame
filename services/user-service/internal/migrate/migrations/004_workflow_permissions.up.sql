-- 004_workflow_permissions.up.sql
INSERT INTO permissions (code, description) VALUES
    ('workflow.read', 'Read workflows'),
    ('workflow.manage', 'Manage workflows')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-0000-0000-000000000010', p.id
FROM permissions p
WHERE p.code IN ('workflow.read', 'workflow.manage')
ON CONFLICT DO NOTHING;
