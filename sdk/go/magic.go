package sdk

import "context"

// MagicClient is a tiny façade over Client that wires the minimum fields needed
// for demos: issue a credential, spawn an agent, verify, and authorize with
// resource/action semantics.
type MagicClient struct {
	core *Client
}

// NewMagicClient builds a MagicClient with explicit service URLs.
func NewMagicClient(issuerURL, verifierURL, gatewayURL string) *MagicClient {
	return &MagicClient{core: &Client{IssuerURL: issuerURL, VerifierURL: verifierURL, GatewayURL: gatewayURL}}
}

// NewMagicLocalClient preconfigures the client for docker-compose defaults.
func NewMagicLocalClient() *MagicClient {
	return NewMagicClient("http://localhost:8080", "http://localhost:8081", "http://localhost:8081")
}

// NewMagicClientFromEnv mirrors NewClientFromEnv with magic defaults.
func NewMagicClientFromEnv() *MagicClient {
	base := NewClientFromEnv()
	if base.IssuerURL == "" {
		base.IssuerURL = "http://localhost:8080"
	}
	if base.VerifierURL == "" {
		base.VerifierURL = "http://localhost:8081"
	}
	if base.GatewayURL == "" {
		base.GatewayURL = base.VerifierURL
	}
	return &MagicClient{core: base}
}

// IssueVC mints a VC-JWT for the given subject with optional claims.
func (m *MagicClient) IssueVC(ctx context.Context, subjectDID string, ttlSeconds int64, claims map[string]any) (string, error) {
	resp, err := m.core.IssueVC(ctx, IssueRequest{SubjectDID: subjectDID, TTLSeconds: ttlSeconds, Claims: claims})
	if err != nil {
		return "", err
	}
	return resp.Credential, nil
}

// SpawnAgent delegates a parent credential to an agent DID with narrowed scope and TTL.
func (m *MagicClient) SpawnAgent(ctx context.Context, parentCredential, agentDID string, scope []string, ttlSeconds int64, claims map[string]any) (string, error) {
	resp, err := m.core.DelegateCredential(ctx, DelegateRequest{
		ParentCredential: parentCredential,
		DelegateDID:      agentDID,
		Scope:            scope,
		TTLSeconds:       ttlSeconds,
		Claims:           claims,
	})
	if err != nil {
		return "", err
	}
	return resp.Credential, nil
}

// Verify runs gateway verification on a single credential with an optional audience expectation.
func (m *MagicClient) Verify(ctx context.Context, credential, expectedAudience string) (VerifyResponse, error) {
	return m.core.VerifyWithOptions(ctx, VerifyRequest{Credential: credential, ExpectedAudience: expectedAudience})
}

// Authorize runs the gateway authorize endpoint with resource/action inputs and optional synthetic JWT minting.
func (m *MagicClient) Authorize(ctx context.Context, credentials []string, resource, action, expectedAudience string, wantSyntheticJWT bool) (AuthorizeResponse, error) {
	req := AuthorizeRequest{
		Credentials:      credentials,
		ExpectedAudience: expectedAudience,
		Resource:         resource,
		Action:           action,
		WantSyntheticJWT: wantSyntheticJWT,
	}
	return m.core.Authorize(ctx, req)
}
