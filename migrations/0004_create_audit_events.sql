CREATE TABLE IF NOT EXISTS audit_events (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
    correlation_id TEXT,
    event_type TEXT NOT NULL,
    tenant_id TEXT,
    user_id TEXT,
    subject_did TEXT,
    issuer_did TEXT,
    resource TEXT,
    action TEXT,
    outcome TEXT NOT NULL,
    reason TEXT,
    ip_address TEXT,
    user_agent TEXT,
    metadata JSONB
);
