-- Remove global tenant lifecycle permissions from tenant-scoped admin role.
-- Platform operators use the platform_admin role (migration 007).

DELETE FROM role_permissions
WHERE role_id = '00000000-0000-0000-0000-000000000010'
  AND permission_id IN (
    SELECT id FROM permissions WHERE code IN ('tenant.read', 'tenant.manage')
  );
