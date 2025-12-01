-- Create extension for UUID in the presentations table
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create DID management table
CREATE TABLE dids (
    id SERIAL PRIMARY KEY,
    did TEXT NOT NULL UNIQUE,
    organization_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    public_key JSONB, -- Store the public keys as a JSON array
    document JSONB     -- Store the DID document as JSON
);

-- Create DID document storage table
CREATE TABLE IF NOT EXISTS did_documents (
    id SERIAL PRIMARY KEY,
    did VARCHAR(255) UNIQUE NOT NULL,          -- Corresponding DID
    document JSONB NOT NULL,                   -- The full DID document (JSON format)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP -- Timestamp of document creation
);

-- Create verifiable credentials table with subject properties and revocation functionality
CREATE TABLE IF NOT EXISTS verifiable_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),  -- Unique identifier for the credential
    did VARCHAR(255) NOT NULL,                       -- DID of the subject (credential holder)
    issuer VARCHAR(255) NOT NULL,                    -- DID of the issuer
    credential JSONB NOT NULL,                       -- The verifiable credential (JSON format)
    subject JSONB NOT NULL,                          -- Dynamic subject properties as JSON
    issuance_date TIMESTAMP NOT NULL,                -- When the credential was issued
    expiration_date TIMESTAMP NOT NULL,              -- Expiration date of the credential
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,  -- Timestamp of credential issuance
    revoked BOOLEAN DEFAULT FALSE,                    -- Whether the credential is revoked
    revocation_reason TEXT,                           -- Reason for revocation (optional)
    revoked_at TIMESTAMP,                             -- Timestamp of when the credential was revoked (optional)
    proof JSONB                                       -- Proof of the credential
);


-- Create revocation table (optional, for more detailed tracking)
CREATE TABLE IF NOT EXISTS revocation_registry (
    id SERIAL PRIMARY KEY,
    credential_id UUID REFERENCES verifiable_credentials(id), -- Credential ID reference
    revocation_reason TEXT,                                   -- Reason for revocation
    revoked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP            -- Timestamp of revocation
);

-- Create presentations table 
CREATE TABLE IF NOT EXISTS presentations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    credential_id VARCHAR NOT NULL,
    holder_did VARCHAR NOT NULL,
    presentation_data JSONB NOT NULL,
    processing_id VARCHAR NOT NULL UNIQUE
);

-- Create the schemas table with schema_id from the payload
CREATE TABLE schemas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),                    -- Schema ID provided in the payload
    organization_did VARCHAR(255) NOT NULL,         -- Organization DID
    schema_name VARCHAR(255) NOT NULL,              -- Human-readable name for the schema
    schema_json JSONB NOT NULL,                     -- JSON representation of the schema's properties
    created_at TIMESTAMP DEFAULT NOW(),             -- Timestamp for when the schema was created
    updated_at TIMESTAMP DEFAULT NOW()              -- Timestamp for last update
);

-- Create an index on organization_did for faster lookups
CREATE INDEX idx_organization_did ON schemas (organization_did);

-- Add tenants table (from migration 0002_add_tenants.sql)
CREATE TABLE IF NOT EXISTS tenants (
    id TEXT PRIMARY KEY,
    name TEXT,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tenant_trusted_issuers (
    tenant_id TEXT NOT NULL,
    issuer_did TEXT NOT NULL,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, issuer_did)
);

-- Add trusted_issuers table (from migration 0001_create_trusted_issuers.sql)
CREATE TABLE IF NOT EXISTS trusted_issuers (
    id SERIAL PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    issuer_did TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, issuer_did)
);

-- Add policies table (from migration 0003_create_policies.sql)
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

