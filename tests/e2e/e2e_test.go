//go:build e2e

package e2e

import (
	"bytes"
	"crypto"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bradtumy/credential-service/internal/config"
	"github.com/bradtumy/credential-service/internal/domain"
	"github.com/bradtumy/credential-service/internal/httpserver"
	"github.com/bradtumy/credential-service/internal/keystore"
	"github.com/bradtumy/credential-service/internal/metrics"
)

type e2eEnvironment struct {
	issuerServer   *httptest.Server
	verifierServer *httptest.Server
	apiServer      *httptest.Server

	issuerDID string
	signer    ed25519.PrivateKey
	client    *http.Client

	issuerURL   string
	verifierURL string
	apiURL      string
}

func newE2EEnvironment(t *testing.T) *e2eEnvironment {
	t.Helper()

	issuerCfg := config.IssuerConfig{HTTPPort: "8080", DefaultTenantID: "tenant-e2e"}
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

	registry := domain.NewMemoryTrustRegistry()
	resolver := func(issuer string) (crypto.PublicKey, error) {
		if issuer != issuerDID {
			return nil, domain.ErrUntrustedIssuer
		}
		return signer.Public(), nil
	}

	mux := http.NewServeMux()
	httpserver.RegisterVerifierRoutes(mux, resolver, registry, issuerCfg.DefaultTenantID, &metrics.NoopVerifierMetrics{}, time.Now)
	httpserver.RegisterGatewayRoutes(mux, &httpserver.GatewayConfig{
		Resolver:        resolver,
		Registry:        registry,
		DefaultTenantID: issuerCfg.DefaultTenantID,
		SigningKey:      signer,
		JWTIssuer:       issuerDID,
		Now:             time.Now,
	})
	httpserver.RegisterTrustRegistryRoutes(mux, registry, issuerCfg.DefaultTenantID)
	verifierServer := httptest.NewServer(mux)
	t.Cleanup(verifierServer.Close)

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		if authz == "" || !strings.HasPrefix(authz, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"missing_token"}`))
			return
		}

		token := strings.TrimPrefix(authz, "Bearer ")
		payload, err := domain.DecodeSyntheticJWT(token)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"invalid_token"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"subject": payload["sub"],
			"claims":  payload,
		})
	})
	apiServer := httptest.NewServer(apiMux)
	t.Cleanup(apiServer.Close)

	return &e2eEnvironment{
		issuerServer:   issuerServer,
		verifierServer: verifierServer,
		apiServer:      apiServer,
		issuerDID:      issuerDID,
		signer:         signer.(ed25519.PrivateKey),
		client:         &http.Client{Timeout: 10 * time.Second},
		issuerURL:      envOrDefault("E2E_ISSUER_URL", issuerServer.URL),
		verifierURL:    envOrDefault("E2E_VERIFIER_URL", verifierServer.URL),
		apiURL:         envOrDefault("E2E_API_URL", apiServer.URL),
	}
}

func (env *e2eEnvironment) issueCredential(t *testing.T, subjectDID string, ttl time.Duration, claims map[string]any) string {
	t.Helper()
	reqBody := httpserver.IssueRequest{SubjectDID: subjectDID, TTLSeconds: int64(ttl.Seconds()), Claims: claims}
	payload, _ := json.Marshal(reqBody)

	resp, err := env.client.Post(env.issuerURL+"/v1/credentials/issue", "application/json", bytes.NewReader(payload))
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

func (env *e2eEnvironment) registerIssuer(t *testing.T) {
	t.Helper()
	payload, _ := json.Marshal(httpserver.AddTrustedIssuerRequest{IssuerDID: env.issuerDID})
	resp, err := env.client.Post(env.verifierURL+"/v1/trust/issuers", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("register issuer: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("trust registry status: %d", resp.StatusCode)
	}
}

func (env *e2eEnvironment) authorize(t *testing.T, credentials []string, expectedAudience string, wantSynthetic bool) (httpserver.GatewayAuthorizeResponse, int) {
	t.Helper()
	payload := httpserver.GatewayAuthorizeRequest{Credentials: credentials, ExpectedAudience: expectedAudience, WantSyntheticJWT: wantSynthetic, Resource: "orders", Action: "read"}
	body, _ := json.Marshal(payload)

	resp, err := env.client.Post(env.verifierURL+"/v1/gateway/authorize", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("gateway authorize request: %v", err)
	}
	defer resp.Body.Close()

	var authzResp httpserver.GatewayAuthorizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&authzResp); err != nil {
		t.Fatalf("decode gateway response: %v", err)
	}
	return authzResp, resp.StatusCode
}

func (env *e2eEnvironment) callSampleAPI(t *testing.T, token string) int {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, env.apiURL+"/orders", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := env.client.Do(req)
	if err != nil {
		t.Fatalf("sample api request: %v", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func envOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func TestHappyPathEndToEnd(t *testing.T) {
	env := newE2EEnvironment(t)

	_, subjectKey, _ := ed25519.GenerateKey(nil)
	subjectDID, err := domain.DIDFromPublicKey(subjectKey.Public())
	if err != nil {
		t.Fatalf("subject did: %v", err)
	}

	credential := env.issueCredential(t, subjectDID, 5*time.Minute, map[string]any{
		"aud":   "sample-api",
		"scope": []string{"orders:read"},
	})

	env.registerIssuer(t)

	authzResp, status := env.authorize(t, []string{credential}, "sample-api", true)
	if status != http.StatusOK {
		t.Fatalf("gateway status: %d", status)
	}
	if !authzResp.Allowed || authzResp.SyntheticJWT == "" {
		t.Fatalf("unexpected gateway response: %+v", authzResp)
	}

	apiStatus := env.callSampleAPI(t, authzResp.SyntheticJWT)
	if apiStatus != http.StatusOK {
		t.Fatalf("sample api status: %d", apiStatus)
	}
}

func TestExpiredCredentialDenied(t *testing.T) {
	env := newE2EEnvironment(t)

	_, subjectKey, _ := ed25519.GenerateKey(nil)
	subjectDID, err := domain.DIDFromPublicKey(subjectKey.Public())
	if err != nil {
		t.Fatalf("subject did: %v", err)
	}

	credential := env.issueCredential(t, subjectDID, time.Second, map[string]any{"aud": "sample-api"})
	env.registerIssuer(t)

	time.Sleep(2 * time.Second)

	authzResp, status := env.authorize(t, []string{credential}, "sample-api", false)
	if status != http.StatusForbidden {
		t.Fatalf("gateway status: %d", status)
	}
	if authzResp.Reason != "expired_credential" {
		t.Fatalf("expected expired_credential, got %+v", authzResp)
	}
}

func TestTamperedCredentialSignatureFails(t *testing.T) {
	env := newE2EEnvironment(t)

	_, subjectKey, _ := ed25519.GenerateKey(nil)
	subjectDID, err := domain.DIDFromPublicKey(subjectKey.Public())
	if err != nil {
		t.Fatalf("subject did: %v", err)
	}

	credential := env.issueCredential(t, subjectDID, 5*time.Minute, map[string]any{"aud": "sample-api"})
	env.registerIssuer(t)

	tampered := tamperJWTPayload(t, credential)

	authzResp, status := env.authorize(t, []string{tampered}, "sample-api", false)
	if status != http.StatusForbidden {
		t.Fatalf("gateway status: %d", status)
	}
	if authzResp.Reason != "invalid_signature" {
		t.Fatalf("expected invalid_signature, got %+v", authzResp)
	}
}

func tamperJWTPayload(t *testing.T, token string) string {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) < 3 {
		t.Fatalf("invalid token format")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}

	var claims map[string]any
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	claims["aud"] = "tampered-audience"

	newPayload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal tampered payload: %v", err)
	}

	parts[1] = base64.RawURLEncoding.EncodeToString(newPayload)
	return strings.Join(parts, ".")
}
