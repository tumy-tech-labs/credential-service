package integration

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bradtumy/credential-service/internal/config"
	"github.com/bradtumy/credential-service/internal/domain"
	"github.com/bradtumy/credential-service/internal/httpx"
	"github.com/bradtumy/credential-service/internal/keystore"
	"github.com/bradtumy/credential-service/internal/metrics"
)

func TestVerifierIntegration(t *testing.T) {
	store := keystore.NewMemoryKeyStore()
	cfg := config.LoadVerifierConfigFromEnv()

	signer, err := store.GetSigningKey(cfg.DefaultTenantID)
	if err != nil {
		t.Fatalf("get signing key: %v", err)
	}
	issuerDID, err := domain.DIDFromPublicKey(signer.Public())
	if err != nil {
		t.Fatalf("derive issuer did: %v", err)
	}

	registry := domain.NewMemoryTrustRegistry()
	registry.AddTrustedIssuer(context.Background(), cfg.DefaultTenantID, issuerDID)

	keyMap := map[string]crypto.PublicKey{issuerDID: signer.Public()}
	resolver := func(issuer string) (crypto.PublicKey, error) {
		key, ok := keyMap[issuer]
		if !ok {
			return nil, domain.ErrUntrustedIssuer
		}
		return key, nil
	}

	mux := http.NewServeMux()
	httpx.RegisterVerifierRoutes(mux, resolver, registry, cfg.DefaultTenantID, &metrics.NoopVerifierMetrics{}, time.Now)

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	token, err := domain.IssueBasicCredential(issuerDID, "did:jwk:subject", signer, 5*time.Minute, map[string]interface{}{"aud": "example-api"})
	if err != nil {
		t.Fatalf("issue credential: %v", err)
	}

	reqBody := httpx.VerifyRequest{Credential: token, ExpectedAudience: "example-api"}
	payload, _ := json.Marshal(reqBody)

	resp, err := http.Post(ts.URL+"/v1/credentials/verify", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("post verify: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var verifyResp httpx.VerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !verifyResp.Valid || verifyResp.Subject != "did:jwk:subject" {
		t.Fatalf("unexpected response: %+v", verifyResp)
	}
}

func TestVerifierIntegrationRejectsTamperedToken(t *testing.T) {
	cfg := config.LoadVerifierConfigFromEnv()

	_, priv, _ := ed25519.GenerateKey(nil)
	issuerDID, _ := domain.DIDFromPublicKey(priv.Public())

	registry := domain.NewMemoryTrustRegistry()
	registry.AddTrustedIssuer(context.Background(), cfg.DefaultTenantID, issuerDID)

	keyMap := map[string]crypto.PublicKey{issuerDID: priv.Public()}
	resolver := func(issuer string) (crypto.PublicKey, error) {
		key, ok := keyMap[issuer]
		if !ok {
			return nil, domain.ErrUntrustedIssuer
		}
		return key, nil
	}

	mux := http.NewServeMux()
	httpx.RegisterVerifierRoutes(mux, resolver, registry, cfg.DefaultTenantID, &metrics.NoopVerifierMetrics{}, time.Now)

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	token, err := domain.IssueBasicCredential(issuerDID, "did:jwk:subject", priv, 5*time.Minute, nil)
	if err != nil {
		t.Fatalf("issue credential: %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("invalid token format")
	}
	parts[2] = base64.RawURLEncoding.EncodeToString([]byte("tamper"))
	tampered := strings.Join(parts, ".")

	reqBody := httpx.VerifyRequest{Credential: tampered}
	payload, _ := json.Marshal(reqBody)

	resp, err := http.Post(ts.URL+"/v1/credentials/verify", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("post verify: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}
