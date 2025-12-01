package multitenant

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
	"github.com/bradtumy/credential-service/internal/httpx"
	"github.com/bradtumy/credential-service/internal/keystore"
	"github.com/bradtumy/credential-service/internal/metrics"
	"github.com/bradtumy/credential-service/internal/policy"
	"github.com/bradtumy/credential-service/internal/tenant"
)

type recordingEngine struct {
	inputs []policy.EvaluationInput
	allow  bool
}

func (r *recordingEngine) Evaluate(input policy.EvaluationInput) (policy.EvaluationResult, error) {
	r.inputs = append(r.inputs, input)
	if r.allow {
		return policy.EvaluationResult{Allow: true, Reason: "record_allow"}, nil
	}
	return policy.EvaluationResult{Allow: false, Reason: "record_deny"}, nil
}

func TestMultiTenantEndToEnd(t *testing.T) {
	ctx := context.Background()
	tenantStore := tenant.NewMemoryStore()
	tenants := []string{"tenant-a", "tenant-b"}
	for _, id := range tenants {
		if err := tenantStore.UpsertTenant(ctx, domain.Tenant{ID: id, Enabled: true}); err != nil {
			t.Fatalf("seed tenant: %v", err)
		}
	}

	trustRegistry := domain.NewMemoryTrustRegistry()
	keyStore := keystore.NewMemoryKeyStore()
	issuerDIDs := make(map[string]string)
	issuerKeys := make(map[string]crypto.PublicKey)

	for _, tenantID := range tenants {
		signer, err := keyStore.GetSigningKey(tenantID)
		if err != nil {
			t.Fatalf("get signing key: %v", err)
		}
		did, err := domain.DIDFromPublicKey(signer.Public())
		if err != nil {
			t.Fatalf("derive did: %v", err)
		}
		issuerDIDs[tenantID] = did
		issuerKeys[did] = signer.Public()
	}

	if err := trustRegistry.AddTrustedIssuer(ctx, "tenant-a", issuerDIDs["tenant-a"]); err != nil {
		t.Fatalf("trust tenant-a issuer: %v", err)
	}

	signingKeyA, err := keyStore.GetSigningKey("tenant-a")
	if err != nil {
		t.Fatalf("fetch signing key: %v", err)
	}

	resolver := func(issuer string) (crypto.PublicKey, error) {
		if key, ok := issuerKeys[issuer]; ok {
			return key, nil
		}
		return nil, domain.ErrUntrustedIssuer
	}

	cfg := config.IssuerConfig{DefaultTenantID: "tenant-a"}
	policyEngine := &recordingEngine{allow: true}

	mux := http.NewServeMux()
	httpx.RegisterIssuerRoutes(mux, keyStore, cfg, nil)
	httpx.RegisterVerifierRoutes(mux, resolver, trustRegistry, cfg.DefaultTenantID, &metrics.NoopVerifierMetrics{}, time.Now)
	httpx.RegisterGatewayRoutes(mux, &httpx.GatewayConfig{
		Resolver:        resolver,
		Registry:        trustRegistry,
		PolicyEngine:    policyEngine,
		DefaultTenantID: cfg.DefaultTenantID,
		SigningKey:      signingKeyA,
		JWTIssuer:       issuerDIDs[cfg.DefaultTenantID],
		Now:             time.Now,
	})

	handler := httpx.RequestContext(httpx.TenantMiddleware(tenant.Resolver{Mode: tenant.ModeMulti, DefaultTenantID: cfg.DefaultTenantID, Store: tenantStore}, mux))

	issuePayload := httpx.IssueRequest{SubjectDID: "did:example:alice", TTLSeconds: 600}
	issueBody, _ := json.Marshal(issuePayload)
	issueReq := httptest.NewRequest(http.MethodPost, "/v1/credentials/issue", bytes.NewReader(issueBody))
	issueReq.Header.Set("X-Tenant-ID", "tenant-a")
	issueRec := httptest.NewRecorder()
	handler.ServeHTTP(issueRec, issueReq)
	if issueRec.Code != http.StatusOK {
		t.Fatalf("issue credential status: %d", issueRec.Code)
	}
	var issueResp httpx.IssueResponse
	_ = json.NewDecoder(issueRec.Body).Decode(&issueResp)
	if issueResp.Credential == "" {
		t.Fatalf("expected credential token")
	}

	verifyPayload := httpx.VerifyRequest{Credential: issueResp.Credential}
	verifyBody, _ := json.Marshal(verifyPayload)

	verifyReq := httptest.NewRequest(http.MethodPost, "/v1/credentials/verify", bytes.NewReader(verifyBody))
	verifyReq.Header.Set("X-Tenant-ID", "tenant-a")
	verifyRec := httptest.NewRecorder()
	handler.ServeHTTP(verifyRec, verifyReq)
	if verifyRec.Code != http.StatusOK {
		t.Fatalf("verify tenant-a status: %d", verifyRec.Code)
	}

	verifyReqB := httptest.NewRequest(http.MethodPost, "/v1/credentials/verify", bytes.NewReader(verifyBody))
	verifyReqB.Header.Set("X-Tenant-ID", "tenant-b")
	verifyRecB := httptest.NewRecorder()
	handler.ServeHTTP(verifyRecB, verifyReqB)
	if verifyRecB.Code != http.StatusForbidden {
		t.Fatalf("verify tenant-b expected forbidden, got %d", verifyRecB.Code)
	}

	gatewayPayload := httpx.GatewayAuthorizeRequest{Credential: issueResp.Credential, Resource: "sample-api", Action: "invoke"}
	gatewayBody, _ := json.Marshal(gatewayPayload)
	gatewayReq := httptest.NewRequest(http.MethodPost, "/v1/gateway/authorize", bytes.NewReader(gatewayBody))
	gatewayReq.Header.Set("X-Tenant-ID", "tenant-a")
	gatewayRec := httptest.NewRecorder()
	handler.ServeHTTP(gatewayRec, gatewayReq)
	if gatewayRec.Code != http.StatusOK {
		t.Fatalf("gateway status: %d", gatewayRec.Code)
	}
	var gatewayResp httpx.GatewayAuthorizeResponse
	_ = json.NewDecoder(gatewayRec.Body).Decode(&gatewayResp)
	if gatewayResp.TenantID != "tenant-a" || !gatewayResp.Allowed {
		t.Fatalf("unexpected gateway response: %+v", gatewayResp)
	}
	if len(policyEngine.inputs) == 0 || policyEngine.inputs[0].TenantID != "tenant-a" {
		t.Fatalf("policy engine did not receive tenant context")
	}
}
