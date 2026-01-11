export type FetchLike = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;
export interface CredentialServiceOptions {
    issuerUrl?: string;
    verifierUrl?: string;
    gatewayUrl?: string;
    fetchImpl?: FetchLike;
}
export interface IssueRequest {
    subject_did: string;
    ttl_seconds?: number;
    claims?: Record<string, unknown>;
    format?: string;
}
export interface IssueResponse {
    credential: string;
    disclosures?: string[];
    format?: string;
    api_version?: string;
}
export interface VerifyRequest {
    credential?: string;
    credentials?: string[];
    expected_audience?: string;
    disclosures?: string[];
    format?: string;
}
export interface VerifyResponse {
    valid: boolean;
    subject: string;
    issuer?: string;
    expires_at: string;
    acting_on_behalf_of?: string;
    delegation_depth?: number;
    claims?: Record<string, unknown>;
    synthetic_jwt?: string;
    api_version?: string;
}
export interface AuthorizeRequest {
    credential?: string;
    credentials?: string[];
    expected_audience?: string;
    want_synthetic_jwt?: boolean;
}
export interface AuthorizeResponse {
    allowed: boolean;
    subject?: string;
    acting_on_behalf_of?: string;
    delegation_depth?: number;
    claims?: Record<string, unknown>;
    reason?: string;
    synthetic_jwt?: string;
    agent?: GatewayAgentContext;
    api_version?: string;
}
export interface GatewayAgentContext {
    acting_on_behalf_of: string;
    delegation_depth: number;
    scope?: string[];
}
export interface DelegateRequest {
    parent_credential: string;
    delegate_did: string;
    scope: string[];
    ttl_seconds: number;
    claims?: Record<string, unknown>;
}
export interface DelegateResponse {
    credential: string;
    api_version?: string;
}
