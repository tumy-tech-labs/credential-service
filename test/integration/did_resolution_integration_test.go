package integration

import (
	"bytes"
	"context"
	"crypto"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bradtumy/credential-service/internal/config"
	"github.com/bradtumy/credential-service/internal/domain"
	"github.com/bradtumy/credential-service/internal/httpx"
	"github.com/bradtumy/credential-service/internal/keystore"
	"github.com/bradtumy/credential-service/internal/metrics"
)

// TestDistributedDIDResolution tests that a verifier can verify credentials
// from a completely separate issuer using DID resolution without pre-configuration
func TestDistributedDIDResolution(t *testing.T) {
	// Simulate Organization A (Issuer)
	issuerStore := keystore.NewMemoryKeyStore()
	issuerCfg := config.IssuerConfig{HTTPPort: "8080", DefaultTenantID: "org-a-tenant"}

	issuerSigner, err := issuerStore.GetSigningKey(issuerCfg.DefaultTenantID)
	if err != nil {
		t.Fatalf("get issuer signing key: %v", err)
	}

	issuerDID, err := domain.DIDFromPublicKey(issuerSigner.Public())
	if err != nil {
		t.Fatalf("derive issuer DID: %v", err)
	}

	// Create issuer service
	issuerMux := http.NewServeMux()
	httpx.RegisterIssuerRoutes(issuerMux, issuerStore, issuerCfg, nil)
	issuerServer := httptest.NewServer(issuerMux)
	t.Cleanup(issuerServer.Close)

	// Simulate Organization B (Verifier) with separate tenant and no pre-configured keys
	verifierCfg := config.LoadVerifierConfigFromEnv()
	verifierCfg.DefaultTenantID = "org-b-tenant" // Different organization

	// Create separate verifier with DID resolution (as done in our updated main.go)
	trustRegistry := domain.NewMemoryTrustRegistry()
	didResolver, err := domain.NewCompositeResolver(
		domain.NewJWKResolver(), // This enables did:jwk resolution
	)
	require.NoError(t, err)

	resolver := func(issuer string) (crypto.PublicKey, error) {
		// Check trust registry first
		trusted, err := trustRegistry.IsTrustedIssuer(context.Background(), verifierCfg.DefaultTenantID, issuer)
		if err != nil {
			return nil, err
		}
		if !trusted {
			return nil, domain.ErrUntrustedIssuer
		}

		// Resolve public key from DID (this is the key part - no pre-configuration needed!)
		return didResolver.ResolvePublicKey(context.Background(), issuer)
	}

	verifierMux := http.NewServeMux()
	httpx.RegisterVerifierRoutes(verifierMux, resolver, trustRegistry, verifierCfg.DefaultTenantID, &metrics.NoopVerifierMetrics{}, time.Now)
	verifierServer := httptest.NewServer(verifierMux)
	t.Cleanup(verifierServer.Close)

	// Step 1: Organization A issues a credential
	issueReq := httpx.IssueRequest{
		SubjectDID: "did:jwk:subject123",
		TTLSeconds: int64((5 * time.Minute).Seconds()),
		Claims:     map[string]interface{}{"scope": []string{"read"}, "aud": "org-b-api"},
	}
	issuePayload, _ := json.Marshal(issueReq)

	issueResp, err := http.Post(issuerServer.URL+"/v1/credentials/issue", "application/json", bytes.NewReader(issuePayload))
	if err != nil {
		t.Fatalf("issue credential: %v", err)
	}
	defer issueResp.Body.Close()

	if issueResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", issueResp.StatusCode)
	}

	var issueResult httpx.IssueResponse
	if err := json.NewDecoder(issueResp.Body).Decode(&issueResult); err != nil {
		t.Fatalf("decode issue response: %v", err)
	}

	// Step 2: Organization B initially doesn't trust Organization A (should fail)
	verifyReq := httpx.VerifyRequest{
		Credential:       issueResult.Credential,
		ExpectedAudience: "org-b-api",
	}
	verifyPayload, _ := json.Marshal(verifyReq)

	verifyResp, err := http.Post(verifierServer.URL+"/v1/credentials/verify", "application/json", bytes.NewReader(verifyPayload))
	if err != nil {
		t.Fatalf("verify credential: %v", err)
	}
	defer verifyResp.Body.Close()

	// Should fail with untrusted issuer
	if verifyResp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 (untrusted), got %d", verifyResp.StatusCode)
	}

	// Step 3: Organization B decides to trust Organization A's DID
	// This is the key distributed trust decision - no key exchange needed!
	if err := trustRegistry.AddTrustedIssuer(context.Background(), verifierCfg.DefaultTenantID, issuerDID); err != nil {
		t.Fatalf("add trusted issuer: %v", err)
	}

	// Step 4: Now verification should succeed using DID resolution
	verifyResp2, err := http.Post(verifierServer.URL+"/v1/credentials/verify", "application/json", bytes.NewReader(verifyPayload))
	if err != nil {
		t.Fatalf("verify credential after trust: %v", err)
	}
	defer verifyResp2.Body.Close()

	if verifyResp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 after trust established, got %d", verifyResp2.StatusCode)
	}

	var verifyResult httpx.VerifyResponse
	if err := json.NewDecoder(verifyResp2.Body).Decode(&verifyResult); err != nil {
		t.Fatalf("decode verify response: %v", err)
	}

	// Verify the credential details
	if verifyResult.Valid != true {
		t.Fatalf("expected valid credential")
	}

	if verifyResult.Subject != "did:jwk:subject123" {
		t.Fatalf("unexpected subject: %s", verifyResult.Subject)
	}

	if verifyResult.Issuer != issuerDID {
		t.Fatalf("unexpected issuer: %s", verifyResult.Issuer)
	}

	t.Logf("✅ Distributed verification successful!")
	t.Logf("   Organization A (issuer): %s", issuerDID)
	t.Logf("   Organization B (verifier): %s", verifierCfg.DefaultTenantID)
	t.Logf("   No pre-shared keys needed - DID resolution worked!")
}
