package integration

import (
    "bytes"
    "context"
    "crypto"
    "crypto/ed25519"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/bradtumy/credential-service/internal/domain"
    "github.com/bradtumy/credential-service/internal/httpx"
)

func TestGatewayAuthorize_UntrustedIssuer(t *testing.T) {
    // Setup issuer keys and DID
    _, priv, _ := ed25519.GenerateKey(nil)
    issuerDID, _ := domain.DIDFromPublicKey(priv.Public())

    // Trust registry empty (untrusted)
    registry := domain.NewMemoryTrustRegistry()

    // Resolver that can resolve keys but trust check should block
    resolver := func(issuer string) (crypto.PublicKey, error) {
        return priv.Public(), nil
    }

    mux := http.NewServeMux()
    // Minimal dependencies for gateway routes
    httpx.RegisterGatewayRoutes(mux, &httpx.GatewayConfig{
        Resolver:        resolver,
        Registry:        registry,
        DefaultTenantID: "tenant",
        JWTIssuer:       "did:jwk:gateway",
        Now:             time.Now,
    })

    // Issue a simple credential
    token, err := domain.IssueBasicCredential(issuerDID, "did:jwk:subject", priv, 5*time.Minute, map[string]any{"aud": "example-api"})
    if err != nil {
        t.Fatalf("issue credential: %v", err)
    }

    body, _ := json.Marshal(map[string]any{
        "credential": token,
        "expected_audience": "example-api",
        "resource": "orders",
        "action": "read",
    })

    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodPost, "/v1/gateway/authorize", bytes.NewReader(body))
    req = req.WithContext(context.Background())

    mux.ServeHTTP(rr, req)

    if rr.Code != http.StatusForbidden {
        t.Fatalf("expected 403 for untrusted issuer, got %d", rr.Code)
    }
}

func TestGatewayAuthorize_TrustedIssuerAllows(t *testing.T) {
    _, priv, _ := ed25519.GenerateKey(nil)
    issuerDID, _ := domain.DIDFromPublicKey(priv.Public())

    registry := domain.NewMemoryTrustRegistry()
    registry.AddTrustedIssuer(context.Background(), "tenant", issuerDID)

    resolver := func(issuer string) (crypto.PublicKey, error) {
        return priv.Public(), nil
    }

    mux := http.NewServeMux()
    httpx.RegisterGatewayRoutes(mux, &httpx.GatewayConfig{
        Resolver:        resolver,
        Registry:        registry,
        DefaultTenantID: "tenant",
        JWTIssuer:       issuerDID,
        Now:             time.Now,
    })

    token, err := domain.IssueBasicCredential(issuerDID, "did:jwk:subject", priv, 5*time.Minute, map[string]any{"aud": "example-api", "scope": []string{"read"}})
    if err != nil {
        t.Fatalf("issue credential: %v", err)
    }

    body, _ := json.Marshal(map[string]any{
        "credential": token,
        "expected_audience": "example-api",
        "resource": "orders",
        "action": "read",
    })

    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodPost, "/v1/gateway/authorize", bytes.NewReader(body))

    mux.ServeHTTP(rr, req)

    if rr.Code != http.StatusOK {
        t.Fatalf("expected 200 for trusted issuer, got %d", rr.Code)
    }
}
