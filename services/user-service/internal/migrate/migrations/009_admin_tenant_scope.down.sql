INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-0000-0000-000000000010', p.id
FROM permissions p
WHERE p.code IN ('tenant.read', 'tenant.manage')
ON CONFLICT DO NOTHING;
