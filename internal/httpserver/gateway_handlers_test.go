package httpserver

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
	"github.com/bradtumy/credential-service/internal/policy"
	"github.com/bradtumy/credential-service/internal/version"
)

func TestGatewayAuthorizeAllow(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	issuerDID, _ := domain.DIDFromPublicKey(priv.Public())

	registry := domain.NewMemoryTrustRegistry()
	registry.AddTrustedIssuer(context.Background(), "tenant", issuerDID)

	resolver := func(issuer string) (crypto.PublicKey, error) {
		if issuer != issuerDID {
			return nil, domain.ErrUntrustedIssuer
		}
		return priv.Public(), nil
	}

	token, err := domain.IssueBasicCredential(issuerDID, "did:example:agent", priv, 5*time.Minute, map[string]interface{}{"scope": "read:orders"})
	if err != nil {
		t.Fatalf("issue credential: %v", err)
	}

	mux := http.NewServeMux()
	RegisterGatewayRoutes(mux, &GatewayConfig{Resolver: resolver, Registry: registry, DefaultTenantID: "tenant", SigningKey: priv, JWTIssuer: issuerDID, Now: time.Now})

	payload := GatewayAuthorizeRequest{Credential: token, WantSyntheticJWT: true, Resource: "orders", Action: "read"}
	body, _ := json.Marshal(payload)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/gateway/authorize", bytes.NewReader(body))

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp GatewayAuthorizeResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !resp.Allowed || resp.Subject != "did:example:agent" {
		t.Fatalf("unexpected response: %+v", resp)
	}

	if resp.SyntheticJWT == "" {
		t.Fatalf("expected synthetic jwt to be present")
	}

	if resp.APIVersion != version.APIVersion {
		t.Fatalf("expected api version %s, got %s", version.APIVersion, resp.APIVersion)
	}
}

func TestGatewayAuthorizeAgentContext(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	issuerDID, _ := domain.DIDFromPublicKey(priv.Public())

	registry := domain.NewMemoryTrustRegistry()
	registry.AddTrustedIssuer(context.Background(), "tenant", issuerDID)

	resolver := func(issuer string) (crypto.PublicKey, error) {
		return priv.Public(), nil
	}

	parentToken, _ := domain.IssueBasicCredential(issuerDID, "did:example:parent", priv, 5*time.Minute, map[string]interface{}{"scope": []string{"read"}})
	childToken, _ := domain.IssueBasicCredential(issuerDID, "did:example:agent", priv, 2*time.Minute, map[string]interface{}{"scope": []string{"read"}})

	mux := http.NewServeMux()
	RegisterGatewayRoutes(mux, &GatewayConfig{
		Resolver:        resolver,
		Registry:        registry,
		PolicyEngine:    nil,
		DefaultTenantID: "tenant",
		SigningKey:      priv,
		JWTIssuer:       issuerDID,
		DecisionCache:   nil,
		Limiter:         nil,
		Metrics:         nil,
		Now:             time.Now,
	})

	payload := GatewayAuthorizeRequest{Credentials: []string{parentToken, childToken}, WantSyntheticJWT: true, Resource: "orders", Action: "read"}
	body, _ := json.Marshal(payload)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/gateway/authorize", bytes.NewReader(body))

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp GatewayAuthorizeResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Agent == nil || resp.Agent.ActingOnBehalfOf != "did:example:parent" || resp.Agent.DelegationDepth == 0 {
		t.Fatalf("expected agent context, got %+v", resp.Agent)
	}

	if resp.APIVersion != version.APIVersion {
		t.Fatalf("expected api version %s, got %s", version.APIVersion, resp.APIVersion)
	}
}

func TestGatewayAuthorizeDenyExpired(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	issuerDID, _ := domain.DIDFromPublicKey(priv.Public())

	registry := domain.NewMemoryTrustRegistry()
	registry.AddTrustedIssuer(context.Background(), "tenant", issuerDID)

	resolver := func(string) (crypto.PublicKey, error) {
		return priv.Public(), nil
	}

	token, err := domain.IssueBasicCredential(issuerDID, "did:example:bob", priv, time.Minute, nil)
	if err != nil {
		t.Fatalf("issue credential: %v", err)
	}

	cred := decodeCredentialForHandlerTest(t, token)

	mux := http.NewServeMux()
	RegisterGatewayRoutes(mux, &GatewayConfig{
		Resolver:        resolver,
		Registry:        registry,
		DefaultTenantID: "tenant",
		SigningKey:      priv,
		JWTIssuer:       issuerDID,
		Now: func() time.Time {
			return cred.ExpiresAt.Add(time.Second)
		},
	})

	payload := GatewayAuthorizeRequest{Credential: token, Resource: "orders", Action: "read"}
	body, _ := json.Marshal(payload)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/gateway/authorize", bytes.NewReader(body))

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}

	var resp GatewayAuthorizeResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)

	if resp.Allowed || resp.ErrorCode != "credential_expired" {
		t.Fatalf("unexpected deny response: %+v", resp)
	}
}

func TestGatewayAuthorizeUntrustedIssuer(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	issuerDID, _ := domain.DIDFromPublicKey(priv.Public())

	registry := domain.NewMemoryTrustRegistry()
	resolver := func(string) (crypto.PublicKey, error) {
		return priv.Public(), nil
	}

	token, err := domain.IssueBasicCredential(issuerDID, "did:example:bob", priv, 5*time.Minute, nil)
	if err != nil {
		t.Fatalf("issue credential: %v", err)
	}

	mux := http.NewServeMux()
	RegisterGatewayRoutes(mux, &GatewayConfig{Resolver: resolver, Registry: registry, DefaultTenantID: "tenant", SigningKey: priv, JWTIssuer: issuerDID, Now: time.Now})

	payload := GatewayAuthorizeRequest{Credential: token, Resource: "orders", Action: "read"}
	body, _ := json.Marshal(payload)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/gateway/authorize", bytes.NewReader(body))

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}

	var resp GatewayAuthorizeResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Allowed || resp.ErrorCode != "issuer_not_trusted" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestGatewayAuthorizeMissingCredential(t *testing.T) {
	mux := http.NewServeMux()
	RegisterGatewayRoutes(mux, &GatewayConfig{DefaultTenantID: "tenant", Now: time.Now})

	payload := GatewayAuthorizeRequest{Resource: "orders", Action: "read"}
	body, _ := json.Marshal(payload)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/gateway/authorize", bytes.NewReader(body))

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestGatewayAuthorizeMissingResource(t *testing.T) {
	mux := http.NewServeMux()
	RegisterGatewayRoutes(mux, &GatewayConfig{DefaultTenantID: "tenant", Now: time.Now})

	payload := GatewayAuthorizeRequest{Credential: "token", Action: "read"}
	body, _ := json.Marshal(payload)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/gateway/authorize", bytes.NewReader(body))

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestGatewayAuthorizeRateLimited(t *testing.T) {
	limiter := &stubLimiter{allow: false}
	mux := http.NewServeMux()
	RegisterGatewayRoutes(mux, &GatewayConfig{DefaultTenantID: "tenant", Limiter: limiter, Now: time.Now})

	payload := GatewayAuthorizeRequest{Credential: "token", Resource: "orders", Action: "read"}
	body, _ := json.Marshal(payload)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/gateway/authorize", bytes.NewReader(body))

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rr.Code)
	}
}

func TestGatewayAuthorizeCacheHitSkipsVerification(t *testing.T) {
	cache := &recordingCache{}
	mux := http.NewServeMux()
	RegisterGatewayRoutes(mux, &GatewayConfig{DefaultTenantID: "tenant", DecisionCache: cache, Now: time.Now})

	// Prime cache with allowed decision.
	cached := GatewayAuthorizeResponse{Allowed: true, Subject: "did:example:cached", TenantID: "tenant", APIVersion: version.APIVersion}
	key := buildCacheKey([]string{"cached-token"}, "tenant", "orders", "read", "", false)
	payload, _ := json.Marshal(cached)
	_ = cache.Set(key, payload, time.Minute)

	reqBody, _ := json.Marshal(GatewayAuthorizeRequest{Credential: "cached-token", Resource: "orders", Action: "read"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/gateway/authorize", bytes.NewReader(reqBody))

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	if cache.getCalls == 0 {
		t.Fatalf("expected cache get to be invoked")
	}
}

func TestGatewayAuthorizeCachesDecision(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	issuerDID, _ := domain.DIDFromPublicKey(priv.Public())

	registry := domain.NewMemoryTrustRegistry()
	registry.AddTrustedIssuer(context.Background(), "tenant", issuerDID)

	resolverCalls := 0
	resolver := func(issuer string) (crypto.PublicKey, error) {
		resolverCalls++
		return priv.Public(), nil
	}

	token, _ := domain.IssueBasicCredential(issuerDID, "did:example:agent", priv, 5*time.Minute, map[string]interface{}{"scope": "read:orders"})

	cache := &recordingCache{}
	mux := http.NewServeMux()
	RegisterGatewayRoutes(mux, &GatewayConfig{Resolver: resolver, Registry: registry, DefaultTenantID: "tenant", SigningKey: priv, JWTIssuer: issuerDID, DecisionCache: cache, Now: time.Now})

	body, _ := json.Marshal(GatewayAuthorizeRequest{Credential: token, Resource: "orders", Action: "read"})

	// First request populates cache.
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/gateway/authorize", bytes.NewReader(body))
	mux.ServeHTTP(rr, req)

	if cache.setCalls == 0 {
		t.Fatalf("expected cache set to be invoked")
	}

	resolverCallsAfterFirst := resolverCalls

	// Second request should hit cache and avoid resolver.
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/v1/gateway/authorize", bytes.NewReader(body))
	mux.ServeHTTP(rr2, req2)

	if resolverCalls != resolverCallsAfterFirst {
		t.Fatalf("expected resolver calls to remain constant on cache hit")
	}
}

func TestGatewayAuthorizePolicyDeny(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	issuerDID, _ := domain.DIDFromPublicKey(priv.Public())

	registry := domain.NewMemoryTrustRegistry()
	registry.AddTrustedIssuer(context.Background(), "tenant", issuerDID)

	resolver := func(issuer string) (crypto.PublicKey, error) { return priv.Public(), nil }
	token, _ := domain.IssueBasicCredential(issuerDID, "did:example:agent", priv, 5*time.Minute, map[string]any{"scope": []string{"read"}})

	store := policy.NewMemoryStore()
	_ = store.CreatePolicy(context.Background(), &policy.Policy{TenantID: "tenant", Name: "deny", Effect: policy.EffectDeny, Actions: []string{"read"}, Resources: []string{"orders"}, Subjects: []string{"any"}, Priority: 1, Enabled: true})
	engine := policy.NewEngine(store)

	mux := http.NewServeMux()
	RegisterGatewayRoutes(mux, &GatewayConfig{Resolver: resolver, Registry: registry, PolicyEngine: engine, DefaultTenantID: "tenant", SigningKey: priv, JWTIssuer: issuerDID, Now: time.Now})

	body, _ := json.Marshal(GatewayAuthorizeRequest{Credential: token, Resource: "orders", Action: "read"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/gateway/authorize", bytes.NewReader(body))
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
	var resp GatewayAuthorizeResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Allowed || resp.ErrorCode != "policy_denied" {
		t.Fatalf("expected policy deny, got %+v", resp)
	}
}

func TestGatewayAuthorizePolicyAllow(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	issuerDID, _ := domain.DIDFromPublicKey(priv.Public())

	registry := domain.NewMemoryTrustRegistry()
	registry.AddTrustedIssuer(context.Background(), "tenant", issuerDID)

	resolver := func(issuer string) (crypto.PublicKey, error) { return priv.Public(), nil }
	token, _ := domain.IssueBasicCredential(issuerDID, "did:example:agent", priv, 5*time.Minute, map[string]any{"scope": []string{"read"}, "roles": []string{"admin"}})

	store := policy.NewMemoryStore()
	_ = store.CreatePolicy(context.Background(), &policy.Policy{TenantID: "tenant", Name: "allow", Effect: policy.EffectAllow, Actions: []string{"read"}, Resources: []string{"orders/*"}, Subjects: []string{"role:admin"}, Priority: 1, Enabled: true})
	engine := policy.NewEngine(store)

	mux := http.NewServeMux()
	RegisterGatewayRoutes(mux, &GatewayConfig{Resolver: resolver, Registry: registry, PolicyEngine: engine, DefaultTenantID: "tenant", SigningKey: priv, JWTIssuer: issuerDID, Now: time.Now})

	body, _ := json.Marshal(GatewayAuthorizeRequest{Credential: token, Resource: "orders/123", Action: "read"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/gateway/authorize", bytes.NewReader(body))
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp GatewayAuthorizeResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if !resp.Allowed || resp.PolicyID == nil {
		t.Fatalf("expected allow with policy id, got %+v", resp)
	}
}

func TestGatewayAuthorizePolicyDefaultDeny(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	issuerDID, _ := domain.DIDFromPublicKey(priv.Public())

	registry := domain.NewMemoryTrustRegistry()
	registry.AddTrustedIssuer(context.Background(), "tenant", issuerDID)

	resolver := func(issuer string) (crypto.PublicKey, error) { return priv.Public(), nil }
	token, _ := domain.IssueBasicCredential(issuerDID, "did:example:agent", priv, 5*time.Minute, map[string]any{"scope": []string{"read"}})

	engine := policy.NewEngine(policy.NewMemoryStore())

	mux := http.NewServeMux()
	RegisterGatewayRoutes(mux, &GatewayConfig{Resolver: resolver, Registry: registry, PolicyEngine: engine, DefaultTenantID: "tenant", SigningKey: priv, JWTIssuer: issuerDID, Now: time.Now})

	body, _ := json.Marshal(GatewayAuthorizeRequest{Credential: token, Resource: "orders", Action: "read"})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/gateway/authorize", bytes.NewReader(body))
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
	var resp GatewayAuthorizeResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Allowed || resp.ErrorCode != "policy_denied" {
		t.Fatalf("expected default policy deny, got %+v", resp)
	}
}

type stubLimiter struct {
	allow bool
}

func (s *stubLimiter) Allow(string) (bool, error) {
	return s.allow, nil
}

type recordingCache struct {
	stored   map[string][]byte
	getCalls int
	setCalls int
}

func (c *recordingCache) Get(key string) ([]byte, bool, error) {
	c.getCalls++
	if c.stored == nil {
		return nil, false, nil
	}
	val, ok := c.stored[key]
	if !ok {
		return nil, false, nil
	}
	return val, true, nil
}

func (c *recordingCache) Set(key string, value []byte, ttl time.Duration) error {
	_ = ttl
	if c.stored == nil {
		c.stored = map[string][]byte{}
	}
	c.stored[key] = value
	c.setCalls++
	return nil
}
