package sdk

import "time"

// IssueRequest represents a request to issue a verifiable credential.
type IssueRequest struct {
	SubjectDID string                 `json:"subject_did"`
	TTLSeconds int64                  `json:"ttl_seconds"`
	Claims     map[string]interface{} `json:"claims,omitempty"`
	Format     string                 `json:"format,omitempty"`
}

// IssueResponse contains the issued credential token and associated metadata.
type IssueResponse struct {
	Credential  string   `json:"credential"`
	Disclosures []string `json:"disclosures,omitempty"`
	Format      string   `json:"format,omitempty"`
	APIVersion  string   `json:"api_version,omitempty"`
}

// DelegateRequest represents a delegation issuance request.
type DelegateRequest struct {
	ParentCredential string                 `json:"parent_credential"`
	DelegateDID      string                 `json:"delegate_did"`
	Scope            []string               `json:"scope"`
	TTLSeconds       int64                  `json:"ttl_seconds"`
	Claims           map[string]interface{} `json:"claims,omitempty"`
}

// DelegateResponse contains the delegated credential token and metadata.
type DelegateResponse struct {
	Credential string `json:"credential"`
	APIVersion string `json:"api_version,omitempty"`
}

// VerifyRequest represents a verification request payload.
type VerifyRequest struct {
	Credential       string   `json:"credential,omitempty"`
	Credentials      []string `json:"credentials,omitempty"`
	ExpectedAudience string   `json:"expected_audience,omitempty"`
	Disclosures      []string `json:"disclosures,omitempty"`
	Format           string   `json:"format,omitempty"`
}

// VerifyResponse represents verification details for a credential chain.
type VerifyResponse struct {
	Valid            bool                   `json:"valid"`
	Subject          string                 `json:"subject"`
	Issuer           string                 `json:"issuer,omitempty"`
	ExpiresAt        time.Time              `json:"expires_at"`
	ActingOnBehalfOf string                 `json:"acting_on_behalf_of,omitempty"`
	DelegationDepth  int                    `json:"delegation_depth"`
	Claims           map[string]interface{} `json:"claims,omitempty"`
	SyntheticJWT     string                 `json:"synthetic_jwt,omitempty"`
	APIVersion       string                 `json:"api_version,omitempty"`
}

// AuthorizeRequest is the payload expected by the gateway authorize endpoint.
type AuthorizeRequest struct {
	Credential       string   `json:"credential,omitempty"`
	Credentials      []string `json:"credentials,omitempty"`
	ExpectedAudience string   `json:"expected_audience,omitempty"`
	Resource         string   `json:"resource,omitempty"`
	Action           string   `json:"action,omitempty"`
	WantSyntheticJWT bool     `json:"want_synthetic_jwt,omitempty"`
}

// AuthorizeResponse is returned when authorizing a request at the gateway.
type AuthorizeResponse struct {
	Allowed          bool                   `json:"allowed"`
	Subject          string                 `json:"subject,omitempty"`
	ActingOnBehalfOf string                 `json:"acting_on_behalf_of,omitempty"`
	DelegationDepth  int                    `json:"delegation_depth"`
	Claims           map[string]interface{} `json:"claims,omitempty"`
	Reason           string                 `json:"reason,omitempty"`
	SyntheticJWT     string                 `json:"synthetic_jwt,omitempty"`
	Agent            *GatewayAgentContext   `json:"agent,omitempty"`
	APIVersion       string                 `json:"api_version,omitempty"`
}

// GatewayAgentContext exposes agent metadata when delegation is detected.
type GatewayAgentContext struct {
	ActingOnBehalfOf string   `json:"acting_on_behalf_of"`
	DelegationDepth  int      `json:"delegation_depth"`
	Scope            []string `json:"scope,omitempty"`
}

// APIError models the common error envelope returned by services.
type APIError struct {
	Error       string `json:"error"`
	Description string `json:"description"`
	Code        string `json:"code"`
	APIVersion  string `json:"api_version"`
}
