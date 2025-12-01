CREATE TABLE IF NOT EXISTS policies (
    id SERIAL PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    effect TEXT NOT NULL,
    actions TEXT[] NOT NULL,
    resources TEXT[] NOT NULL,
    subjects TEXT[] NOT NULL,
    conditions JSONB,
    priority INT DEFAULT 100,
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_policies_tenant ON policies(tenant_id);
CREATE INDEX IF NOT EXISTS idx_policies_tenant_priority ON policies(tenant_id, priority);
