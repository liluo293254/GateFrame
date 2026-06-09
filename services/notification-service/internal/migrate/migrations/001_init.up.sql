CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID,
    title VARCHAR(255) NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_tenant ON notifications (tenant_id);
CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications (tenant_id, user_id);

INSERT INTO notifications (id, tenant_id, user_id, title, body)
VALUES (
    '00000000-0000-0000-0000-000000000120',
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000100',
    'Welcome',
    'Your GateFrame tenant workspace is ready.'
)
ON CONFLICT (id) DO NOTHING;
