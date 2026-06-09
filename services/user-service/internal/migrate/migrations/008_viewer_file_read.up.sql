-- Grant viewer file.read for read-only file list (no file.manage).

INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-0000-0000-000000000020', p.id
FROM permissions p
WHERE p.code = 'file.read'
ON CONFLICT DO NOTHING;
