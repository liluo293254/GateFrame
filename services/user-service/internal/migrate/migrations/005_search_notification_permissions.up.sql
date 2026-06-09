-- 005_search_notification_permissions.up.sql
INSERT INTO permissions (code, description) VALUES
    ('search.read', 'Read search results'),
    ('search.manage', 'Manage search index'),
    ('notification.read', 'Read notifications'),
    ('notification.manage', 'Manage notifications')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT '00000000-0000-0000-0000-000000000010', p.id
FROM permissions p
WHERE p.code IN ('search.read', 'search.manage', 'notification.read', 'notification.manage')
ON CONFLICT DO NOTHING;
