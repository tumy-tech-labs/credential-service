package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bradtumy/credential-service/internal/config"
	"github.com/bradtumy/credential-service/internal/domain"
	"github.com/bradtumy/credential-service/internal/httpserver"
	"github.com/bradtumy/credential-service/internal/keystore"
	"github.com/bradtumy/credential-service/internal/logging"
	"github.com/bradtumy/credential-service/internal/tenant"
)

func main() {
	fmt.Println("🔍 Credential Service - Audit Logging Demo")
	fmt.Println("==========================================")

	// Initialize structured logging with audit capabilities
	logging.Init("info")
	fmt.Println("✅ Initialized structured logging with audit trail capabilities")

	// Create in-memory keystore and demo tenant
	store := keystore.NewMemoryKeyStore()
	cfg := config.IssuerConfig{
		HTTPPort:        "8086",
		DefaultTenantID: "demo-tenant",
		LogLevel:        "info",
	}
	tenantStore := tenant.NewMemoryStore()
	_ = tenantStore.UpsertTenant(context.Background(), domain.Tenant{ID: cfg.DefaultTenantID, Name: "Demo Tenant", Enabled: true})

	// Wire routes
	mux := http.NewServeMux()
	httpserver.RegisterIssuerRoutes(mux, store, cfg, nil)
	httpserver.RegisterHealthRoutes(mux, nil)

	// Apply comprehensive middleware chain (correlation IDs, audit, security headers)
	handler := httpserver.StandardMiddlewareChain()(mux)

	// Start demo server
	srv := &http.Server{Addr: ":" + cfg.HTTPPort, Handler: handler}
	go func() {
		fmt.Printf("🚀 Server running on http://localhost:%s\n", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	// Give the server a moment to start
	time.Sleep(200 * time.Millisecond)

	// Run demo requests
	demoAuditLogging(cfg.HTTPPort)

	// Shutdown cleanly
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	fmt.Println("\n✨ Demo completed! Check logs above for audit events.")
}

func demoAuditLogging(port string) {
	base := fmt.Sprintf("http://localhost:%s", port)
	client := &http.Client{Timeout: 5 * time.Second}

	// 1) Health (non-sensitive)
	fmt.Println("\n1) Health endpoint (non-sensitive)")
	req, _ := http.NewRequest(http.MethodGet, base+"/health", nil)
	req.Header.Set("X-Correlation-ID", "demo-health-check")
	req.Header.Set("X-Tenant-ID", "demo-tenant")
	if resp, err := client.Do(req); err != nil {
		fmt.Printf("   ❌ Health check error: %v\n", err)
	} else {
		fmt.Printf("   ✅ Health status: %d\n", resp.StatusCode)
		_ = resp.Body.Close()
	}

	// 2) Issue credential (sensitive) → audited
	fmt.Println("\n2) Credential issuance (audited)")
	issueReq := httpserver.IssueRequest{
		SubjectDID: "did:jwk:demo-subject",
		TTLSeconds: 1800,
		Claims: map[string]interface{}{
			"role":        "demo-user",
			"permissions": []string{"read", "write"},
		},
	}
	payload, _ := json.Marshal(issueReq)
	req, _ = http.NewRequest(http.MethodPost, base+"/v1/credentials/issue", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Correlation-ID", "demo-credential-issuance")
	req.Header.Set("X-Tenant-ID", "demo-tenant")
	req.Header.Set("User-Agent", "AuditLoggingDemo/1.0")
	if resp, err := client.Do(req); err != nil {
		fmt.Printf("   ❌ Issuance error: %v\n", err)
	} else {
		fmt.Printf("   ✅ Issuance status: %d\n", resp.StatusCode)
		if resp.StatusCode == http.StatusOK {
			var out httpserver.IssueResponse
			_ = json.NewDecoder(resp.Body).Decode(&out)
			fmt.Println("   📄 Credential issued")
		}
		_ = resp.Body.Close()
	}

	// 3) Invalid request (audited failure)
	fmt.Println("\n3) Invalid issuance (audited failure)")
	bad := map[string]interface{}{"subject_did": "not-a-did", "ttl_seconds": -1}
	payload, _ = json.Marshal(bad)
	req, _ = http.NewRequest(http.MethodPost, base+"/v1/credentials/issue", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Correlation-ID", "demo-invalid-request")
	req.Header.Set("X-Tenant-ID", "demo-tenant")
	if resp, err := client.Do(req); err != nil {
		fmt.Printf("   ❌ Invalid request error: %v\n", err)
	} else {
		fmt.Printf("   ✅ Invalid handled: %d (expected error)\n", resp.StatusCode)
		_ = resp.Body.Close()
	}

	// 4) Manual audit events
	fmt.Println("\n4) Manual audit events")
	ctx := logging.WithCorrelationID(context.Background(), "demo-manual-audit")
	ctx = logging.WithTenantID(ctx, "demo-tenant")
	ctx = logging.WithUserID(ctx, "demo-admin")

	logging.LogSecurityEvent(ctx, logging.AuditEventAccessDenied, "blocked", "insufficient_privileges",
		map[string]interface{}{"resource": "/admin/sensitive-data"})
	logging.LogAdminEvent(ctx, logging.AuditEventPolicyCreated, "/policies/demo-policy", "CREATE", "success",
		map[string]interface{}{"policy_type": "access_control", "effect": "allow"})
	logging.LogCredentialEvent(ctx, logging.AuditEventCredentialRevoked, "did:jwk:revoked-subject", "did:jwk:demo-issuer", "success")
	fmt.Println("   ✅ Manual events emitted")
}
