CREATE TABLE IF NOT EXISTS audit_log (
    id BIGSERIAL PRIMARY KEY,
    organization_id TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    actor_name TEXT,
    action TEXT NOT NULL,
    resource_type TEXT,
    resource_id TEXT,
    detail JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_log_org ON audit_log(organization_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_created ON audit_log(created_at DESC);
