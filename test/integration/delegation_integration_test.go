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

	"github.com/bradtumy/credential-service/internal/config"
	"github.com/bradtumy/credential-service/internal/domain"
	"github.com/bradtumy/credential-service/internal/httpserver"
	"github.com/bradtumy/credential-service/internal/keystore"
	"github.com/bradtumy/credential-service/internal/metrics"
)

func TestDelegationIntegration(t *testing.T) {
	store := keystore.NewMemoryKeyStore()
	issuerCfg := config.LoadIssuerConfigFromEnv()
	verifierCfg := config.LoadVerifierConfigFromEnv()

	signer, err := store.GetSigningKey(issuerCfg.DefaultTenantID)
	if err != nil {
		t.Fatalf("get signing key: %v", err)
	}
	issuerDID, err := domain.DIDFromPublicKey(signer.Public())
	if err != nil {
		t.Fatalf("derive issuer did: %v", err)
	}

	// Issuer server
	issuerMux := http.NewServeMux()
	httpserver.RegisterIssuerRoutes(issuerMux, store, issuerCfg, nil)
	issuerServer := httptest.NewServer(issuerMux)
	t.Cleanup(issuerServer.Close)

	// Verifier server
	registry := domain.NewMemoryTrustRegistry()
	registry.AddTrustedIssuer(context.Background(), verifierCfg.DefaultTenantID, issuerDID)
	resolver := func(issuer string) (crypto.PublicKey, error) {
		if issuer != issuerDID {
			return nil, domain.ErrUntrustedIssuer
		}
		return signer.Public(), nil
	}

	verifierMux := http.NewServeMux()
	httpserver.RegisterVerifierRoutes(verifierMux, resolver, registry, verifierCfg.DefaultTenantID, &metrics.NoopVerifierMetrics{}, time.Now)
	verifierServer := httptest.NewServer(verifierMux)
	t.Cleanup(verifierServer.Close)

	issueBody := httpserver.IssueRequest{
		SubjectDID: "did:jwk:owner",
		TTLSeconds: int64((10 * time.Minute).Seconds()),
		Claims:     map[string]interface{}{"scope": []string{"read", "write"}},
	}
	issuePayload, _ := json.Marshal(issueBody)
	issueResp, err := http.Post(issuerServer.URL+"/v1/credentials/issue", "application/json", bytes.NewReader(issuePayload))
	if err != nil {
		t.Fatalf("issue credential: %v", err)
	}
	t.Cleanup(func() {
		_ = issueResp.Body.Close()
	})
	if issueResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from issuer, got %d", issueResp.StatusCode)
	}
	var issued httpserver.IssueResponse
	if err := json.NewDecoder(issueResp.Body).Decode(&issued); err != nil {
		t.Fatalf("decode issue response: %v", err)
	}

	delegateReq := httpserver.DelegateRequest{
		ParentCredential: issued.Credential,
		DelegateDID:      "did:jwk:agent",
		Scope:            []string{"read"},
		TTLSeconds:       int64((5 * time.Minute).Seconds()),
	}
	delegatePayload, _ := json.Marshal(delegateReq)
	delegateResp, err := http.Post(issuerServer.URL+"/v1/credentials/delegate", "application/json", bytes.NewReader(delegatePayload))
	if err != nil {
		t.Fatalf("delegate credential: %v", err)
	}
	t.Cleanup(func() {
		_ = delegateResp.Body.Close()
	})
	if delegateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from delegate, got %d", delegateResp.StatusCode)
	}
	var delegated httpserver.IssueResponse
	if err := json.NewDecoder(delegateResp.Body).Decode(&delegated); err != nil {
		t.Fatalf("decode delegate response: %v", err)
	}

	verifyReq := httpserver.VerifyRequest{Credentials: []string{issued.Credential, delegated.Credential}}
	verifyPayload, _ := json.Marshal(verifyReq)
	verifyResp, err := http.Post(verifierServer.URL+"/v1/credentials/verify", "application/json", bytes.NewReader(verifyPayload))
	if err != nil {
		t.Fatalf("verify credential: %v", err)
	}
	t.Cleanup(func() {
		_ = verifyResp.Body.Close()
	})
	if verifyResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from verifier, got %d", verifyResp.StatusCode)
	}

	var verifyBody httpserver.VerifyResponse
	if err := json.NewDecoder(verifyResp.Body).Decode(&verifyBody); err != nil {
		t.Fatalf("decode verify response: %v", err)
	}

	if !verifyBody.Valid {
		t.Fatalf("expected valid verification")
	}
	if verifyBody.Subject != "did:jwk:agent" || verifyBody.ActingOnBehalfOf != "did:jwk:owner" || verifyBody.DelegationDepth != 1 {
		t.Fatalf("unexpected verification response: %+v", verifyBody)
	}
}
