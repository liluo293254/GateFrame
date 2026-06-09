CREATE TABLE IF NOT EXISTS file_objects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    object_key VARCHAR(512) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    content_type VARCHAR(128) NOT NULL DEFAULT 'application/octet-stream',
    size_bytes BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_file_objects_tenant ON file_objects (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_file_objects_tenant_object_key ON file_objects (tenant_id, object_key);
