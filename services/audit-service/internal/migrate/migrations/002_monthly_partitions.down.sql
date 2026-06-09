DROP INDEX IF EXISTS idx_audit_events_tenant_created;
DROP TABLE IF EXISTS audit_events CASCADE;
DROP FUNCTION IF EXISTS ensure_audit_partition(DATE);

CREATE TABLE IF NOT EXISTS audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID,
    action VARCHAR(16) NOT NULL,
    path VARCHAR(512) NOT NULL,
    status_code INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_events_tenant_created
    ON audit_events (tenant_id, created_at DESC);
