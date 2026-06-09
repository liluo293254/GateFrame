CREATE TABLE IF NOT EXISTS search_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_search_documents_tenant ON search_documents (tenant_id);
CREATE INDEX IF NOT EXISTS idx_search_documents_fts ON search_documents
    USING GIN (to_tsvector('english', coalesce(title, '') || ' ' || coalesce(content, '')));

INSERT INTO search_documents (id, tenant_id, title, content)
VALUES (
    '00000000-0000-0000-0000-000000000110',
    '00000000-0000-0000-0000-000000000001',
    'Welcome guide',
    'Getting started with GateFrame platform search'
)
ON CONFLICT (id) DO NOTHING;
