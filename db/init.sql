-- Create DID management table
CREATE TABLE IF NOT EXISTS dids (
    id SERIAL PRIMARY KEY,
    did VARCHAR(255) UNIQUE NOT NULL,          -- Decentralized Identifier
    organization_id VARCHAR(255),              -- Organization-specific identifier
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP -- Timestamp of creation
);

-- Create DID document storage table
CREATE TABLE IF NOT EXISTS did_documents (
    id SERIAL PRIMARY KEY,
    did VARCHAR(255) UNIQUE NOT NULL,          -- Corresponding DID
    document JSONB NOT NULL,                   -- The full DID document (JSON format)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP -- Timestamp of document creation
);

-- Create verifiable credentials table
CREATE TABLE IF NOT EXISTS verifiable_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    did VARCHAR(255) NOT NULL,                 -- DID of the subject (credential holder)
    issuer VARCHAR(255) NOT NULL,              -- DID of the issuer
    credential JSONB NOT NULL,                 -- The verifiable credential (JSON format)
    issuance_date TIMESTAMP NOT NULL,          -- When the credential was issued
    expiration_date TIMESTAMP NOT NULL,        -- Expiration date of the credential
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, -- Timestamp of credential issuance
    revoked BOOLEAN DEFAULT FALSE              -- Whether the credential is revoked
);

-- Create revocation table (optional, for more detailed tracking)
CREATE TABLE IF NOT EXISTS revocation_registry (
    id SERIAL PRIMARY KEY,
    credential_id UUID REFERENCES verifiable_credentials(id), -- Credential ID reference
    revocation_reason TEXT,                                   -- Reason for revocation
    revoked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP            -- Timestamp of revocation
);


