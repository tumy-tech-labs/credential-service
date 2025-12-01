CREATE TABLE IF NOT EXISTS trusted_issuers (
    id SERIAL PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    issuer_did TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, issuer_did)
);
