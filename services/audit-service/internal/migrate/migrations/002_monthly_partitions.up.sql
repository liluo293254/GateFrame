-- Convert audit_events to monthly range partitions (upgrade from 001 flat table).

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_class c
        JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE c.relname = 'audit_events'
          AND c.relkind = 'r'
          AND NOT EXISTS (SELECT 1 FROM pg_inherits WHERE inhparent = c.oid)
    ) THEN
        ALTER TABLE audit_events RENAME TO audit_events_legacy;
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS audit_events (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID,
    action VARCHAR(16) NOT NULL,
    path VARCHAR(512) NOT NULL,
    status_code INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

CREATE OR REPLACE FUNCTION ensure_audit_partition(month_start DATE) RETURNS void AS $$
DECLARE
    part_name TEXT := 'audit_events_' || to_char(month_start, 'YYYY_MM');
    month_end DATE := (month_start + INTERVAL '1 month')::DATE;
BEGIN
    IF to_regclass(part_name) IS NULL THEN
        EXECUTE format(
            'CREATE TABLE %I PARTITION OF audit_events FOR VALUES FROM (%L) TO (%L)',
            part_name, month_start, month_end
        );
    END IF;
END;
$$ LANGUAGE plpgsql;

SELECT ensure_audit_partition(date_trunc('month', NOW())::DATE);
SELECT ensure_audit_partition((date_trunc('month', NOW()) + INTERVAL '1 month')::DATE);

DO $$
DECLARE
    m DATE;
BEGIN
    IF to_regclass('audit_events_legacy') IS NOT NULL THEN
        FOR m IN
            SELECT DISTINCT date_trunc('month', created_at)::DATE FROM audit_events_legacy
        LOOP
            PERFORM ensure_audit_partition(m);
        END LOOP;
        INSERT INTO audit_events SELECT * FROM audit_events_legacy;
        DROP TABLE audit_events_legacy;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_audit_events_tenant_created
    ON audit_events (tenant_id, created_at DESC);
