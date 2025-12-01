package sdk

// IssueRequest represents a request to issue a verifiable credential.
type IssueRequest struct {
	SubjectDID string                 `json:"subject_did"`
	TTLSeconds int64                  `json:"ttl_seconds"`
	Claims     map[string]interface{} `json:"claims,omitempty"`
}

// IssueResponse contains the issued credential token and associated metadata.
type IssueResponse struct {
	Credential       string                 `json:"credential"`
	Subject          string                 `json:"subject,omitempty"`
	ActingOnBehalfOf string                 `json:"acting_on_behalf_of,omitempty"`
	DelegationDepth  int                    `json:"delegation_depth,omitempty"`
	Claims           map[string]interface{} `json:"claims,omitempty"`
	SyntheticJWT     string                 `json:"synthetic_jwt,omitempty"`
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
	Credential       string                 `json:"credential"`
	Subject          string                 `json:"subject,omitempty"`
	ActingOnBehalfOf string                 `json:"acting_on_behalf_of,omitempty"`
	DelegationDepth  int                    `json:"delegation_depth,omitempty"`
	Claims           map[string]interface{} `json:"claims,omitempty"`
	SyntheticJWT     string                 `json:"synthetic_jwt,omitempty"`
}

// VerifyResponse represents verification details for a credential chain.
type VerifyResponse struct {
	Valid            bool                   `json:"valid"`
	Subject          string                 `json:"subject"`
	Issuer           string                 `json:"issuer,omitempty"`
	ExpiresAt        string                 `json:"expires_at,omitempty"`
	ActingOnBehalfOf string                 `json:"acting_on_behalf_of,omitempty"`
	DelegationDepth  int                    `json:"delegation_depth,omitempty"`
	Claims           map[string]interface{} `json:"claims,omitempty"`
	SyntheticJWT     string                 `json:"synthetic_jwt,omitempty"`
}

// GatewayAuthorizeRequest is the payload expected by the gateway authorize endpoint.
type GatewayAuthorizeRequest struct {
	Credential       string   `json:"credential"`
	Credentials      []string `json:"credentials"`
	ExpectedAudience string   `json:"expected_audience"`
	WantSyntheticJWT bool     `json:"want_synthetic_jwt"`
}

// GatewayAuthorizeResponse is returned when authorizing a request at the gateway.
type GatewayAuthorizeResponse struct {
        Allowed          bool                   `json:"allowed"`
        Subject          string                 `json:"subject,omitempty"`
        ActingOnBehalfOf string                 `json:"acting_on_behalf_of,omitempty"`
        DelegationDepth  int                    `json:"delegation_depth,omitempty"`
        Claims           map[string]interface{} `json:"claims,omitempty"`
        Reason           string                 `json:"reason,omitempty"`
        SyntheticJWT     string                 `json:"synthetic_jwt,omitempty"`
        Agent            *GatewayAgentContext   `json:"agent,omitempty"`
}

// GatewayAgentContext exposes agent metadata when delegation is detected.
type GatewayAgentContext struct {
        ActingOnBehalfOf string   `json:"acting_on_behalf_of"`
        DelegationDepth  int      `json:"delegation_depth"`
        Scope            []string `json:"scope,omitempty"`
}
