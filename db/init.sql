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
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    did VARCHAR(255) NOT NULL,                 -- DID of the subject (credential holder)
    issuer VARCHAR(255) NOT NULL,              -- DID of the issuer
    credential JSONB NOT NULL,                 -- The verifiable credential (JSON format)
    subject_name VARCHAR(255),                  -- Subject's name
    subject_email VARCHAR(255),                 -- Subject's email
    subject_phone VARCHAR(50),                  -- Subject's phone number
    issuance_date TIMESTAMP NOT NULL,          -- When the credential was issued
    expiration_date TIMESTAMP NOT NULL,        -- Expiration date of the credential
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, -- Timestamp of credential issuance
    revoked BOOLEAN DEFAULT FALSE,              -- Whether the credential is revoked
    revocation_reason TEXT,                     -- Reason for revocation (optional)
    revoked_at TIMESTAMP,                       -- Timestamp of when the credential was revoked (optional)
    proof JSONB                                -- Proof of the credential
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

