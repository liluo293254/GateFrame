DELETE FROM role_permissions
WHERE role_id = '00000000-0000-0000-0000-000000000020'
  AND permission_id IN (SELECT id FROM permissions WHERE code = 'file.read');
