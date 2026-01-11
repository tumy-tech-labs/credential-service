package smoke

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

func TestSmokeFlow(t *testing.T) {
	issuerCfg := config.IssuerConfig{HTTPPort: "8080", DefaultTenantID: "tenant-1"}
	issuerStore := keystore.NewMemoryKeyStore()
	issuerMux := http.NewServeMux()
	httpserver.RegisterIssuerRoutes(issuerMux, issuerStore, issuerCfg, nil)
	issuerServer := httptest.NewServer(issuerMux)
	t.Cleanup(issuerServer.Close)

	signer, err := issuerStore.GetSigningKey(issuerCfg.DefaultTenantID)
	if err != nil {
		t.Fatalf("get signing key: %v", err)
	}
	issuerDID, err := domain.DIDFromPublicKey(signer.Public())
	if err != nil {
		t.Fatalf("issuer did: %v", err)
	}

	parentCred := issueCredential(t, issuerServer.URL+"/v1/credentials/issue", httpserver.IssueRequest{
		SubjectDID: "did:example:parent",
		TTLSeconds: int64((5 * time.Minute).Seconds()),
		Claims: map[string]interface{}{
			"scope": []string{"read", "write"},
			"aud":   "sample-api",
		},
	})

	delegated := delegateCredential(t, issuerServer.URL+"/v1/credentials/delegate", httpserver.DelegateRequest{
		ParentCredential: parentCred,
		DelegateDID:      "did:example:agent",
		Scope:            []string{"read"},
		TTLSeconds:       int64((3 * time.Minute).Seconds()),
	})

	registry := domain.NewMemoryTrustRegistry()
	if err := registry.AddTrustedIssuer(context.Background(), issuerCfg.DefaultTenantID, issuerDID); err != nil {
		t.Fatalf("seed registry: %v", err)
	}

	resolver := func(issuer string) (crypto.PublicKey, error) {
		if issuer != issuerDID {
			return nil, domain.ErrUntrustedIssuer
		}
		return signer.Public(), nil
	}

	verifierMux := http.NewServeMux()
	httpserver.RegisterVerifierRoutes(verifierMux, resolver, registry, issuerCfg.DefaultTenantID, &metrics.NoopVerifierMetrics{}, time.Now)
	httpserver.RegisterGatewayRoutes(verifierMux, &httpserver.GatewayConfig{
		Resolver:        resolver,
		Registry:        registry,
		DefaultTenantID: issuerCfg.DefaultTenantID,
		SigningKey:      signer,
		JWTIssuer:       issuerDID,
		Now:             time.Now,
	})
	verifierServer := httptest.NewServer(verifierMux)
	t.Cleanup(verifierServer.Close)

	authzPayload := httpserver.GatewayAuthorizeRequest{
		Credentials:      []string{parentCred, delegated},
		ExpectedAudience: "sample-api",
		WantSyntheticJWT: true,
		Resource:         "sample-api",
		Action:           "invoke",
	}
	body, _ := json.Marshal(authzPayload)

	resp, err := verifierServer.Client().Post(verifierServer.URL+"/v1/gateway/authorize", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("gateway authorize request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gateway, got %d", resp.StatusCode)
	}

	var authzResp httpserver.GatewayAuthorizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&authzResp); err != nil {
		t.Fatalf("decode gateway response: %v", err)
	}

	if !authzResp.Allowed || authzResp.Subject != "did:example:agent" {
		t.Fatalf("unexpected gateway decision: %+v", authzResp)
	}

	if authzResp.SyntheticJWT == "" {
		t.Fatalf("expected synthetic jwt to be minted")
	}
}

func issueCredential(t *testing.T, url string, req httpserver.IssueRequest) string {
	t.Helper()
	payload, _ := json.Marshal(req)
	resp, err := http.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("issue credential request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("issue credential status: %d", resp.StatusCode)
	}

	var issueResp httpserver.IssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&issueResp); err != nil {
		t.Fatalf("decode issue response: %v", err)
	}
	return issueResp.Credential
}

func delegateCredential(t *testing.T, url string, req httpserver.DelegateRequest) string {
	t.Helper()
	payload, _ := json.Marshal(req)
	resp, err := http.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("delegate request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delegate status: %d", resp.StatusCode)
	}

	var issueResp httpserver.IssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&issueResp); err != nil {
		t.Fatalf("decode delegate response: %v", err)
	}
	return issueResp.Credential
}
