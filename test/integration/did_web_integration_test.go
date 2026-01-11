package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bradtumy/credential-service/internal/domain"
)

// TestDidWebIntegration demonstrates the full did:web resolution workflow
func TestDidWebIntegration(t *testing.T) {
	// Step 1: Create a mock server that hosts a DID document
	didDocument := map[string]interface{}{
		"id": "did:web:example.com",
		"verificationMethod": []interface{}{
			map[string]interface{}{
				"id":         "did:web:example.com#key1",
				"type":       "JsonWebKey2020",
				"controller": "did:web:example.com",
				"publicKeyJwk": map[string]interface{}{
					"kty": "OKP",
					"crv": "Ed25519",
					"x":   "11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the correct path is requested
		expectedPath := "/.well-known/did.json"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// Set appropriate headers
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(didDocument)
	}))
	defer server.Close()

	// Step 2: Create a WebResolver with test configuration
	config := domain.DefaultResolverConfig()
	config.AllowInsecureWeb = true // Allow HTTP for testing
	resolver := domain.NewWebResolverWithConfig(config)

	// Step 3: Create test DID using the mock server
	serverHost := strings.TrimPrefix(server.URL, "http://")
	testDID := "did:web:" + serverHost

	// Step 4: Resolve the public key
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	publicKey, err := resolver.ResolvePublicKey(ctx, testDID)
	if err != nil {
		t.Fatalf("Failed to resolve public key: %v", err)
	}

	if publicKey == nil {
		t.Fatal("Expected public key but got nil")
	}

	// Step 5: Verify the resolver supports did:web
	methods := resolver.SupportedMethods()
	if len(methods) != 1 || methods[0] != domain.DIDMethodWeb {
		t.Errorf("Expected supported methods [%s], got %v", domain.DIDMethodWeb, methods)
	}

	t.Logf("✅ Successfully resolved did:web public key")
	t.Logf("   DID: %s", testDID)
	t.Logf("   Public Key Type: %T", publicKey)
}

// TestDidWebWithCompositeResolver tests did:web integration with the composite resolver
func TestDidWebWithCompositeResolver(t *testing.T) {
	// Create mock DID document server
	didDoc := map[string]interface{}{
		"id": "did:web:enterprise.com",
		"verificationMethod": []interface{}{
			map[string]interface{}{
				"id":         "did:web:enterprise.com#main",
				"type":       "JsonWebKey2020",
				"controller": "did:web:enterprise.com",
				"publicKeyJwk": map[string]interface{}{
					"kty": "OKP",
					"crv": "Ed25519",
					"x":   "11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo",
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(didDoc)
	}))
	defer server.Close()

	// Create composite resolver with both jwk and web resolvers
	jwkResolver := domain.NewJWKResolver()

	webConfig := domain.DefaultResolverConfig()
	webConfig.AllowInsecureWeb = true
	webResolver := domain.NewWebResolverWithConfig(webConfig)

	compositeResolver, err := domain.NewCompositeResolver(jwkResolver, webResolver)
	if err != nil {
		t.Fatalf("Failed to create composite resolver: %v", err)
	}

	// Test did:jwk resolution still works
	jwkDID := "did:jwk:eyJjcnYiOiJFZDI1NTE5Iiwia3R5IjoiT0tQIiwieCI6Ik9QRGhLb3hQeXJqbmJZUWc1cFNHS2FoOXFMQ3g2eE5vVEdJdHZ1MDhiZ1UifQ"
	ctx := context.Background()

	_, err = compositeResolver.ResolvePublicKey(ctx, jwkDID)
	if err != nil {
		t.Errorf("Failed to resolve did:jwk: %v", err)
	}

	// Test did:web resolution works
	serverHost := strings.TrimPrefix(server.URL, "http://")
	webDID := "did:web:" + serverHost

	_, err = compositeResolver.ResolvePublicKey(ctx, webDID)
	if err != nil {
		t.Errorf("Failed to resolve did:web: %v", err)
	}

	// Verify supported methods include both
	methods := compositeResolver.SupportedMethods()
	if len(methods) != 2 {
		t.Errorf("Expected 2 methods, got %d", len(methods))
	}

	hasJWK := false
	hasWeb := false
	for _, method := range methods {
		switch method {
		case domain.DIDMethodJWK:
			hasJWK = true
		case domain.DIDMethodWeb:
			hasWeb = true
		}
	}

	if !hasJWK {
		t.Error("Expected composite resolver to support did:jwk")
	}
	if !hasWeb {
		t.Error("Expected composite resolver to support did:web")
	}

	t.Logf("✅ Composite resolver supports both did:jwk and did:web")
}

// TestDidWebSecurityFeatures tests the security controls
func TestDidWebSecurityFeatures(t *testing.T) {
	t.Run("document size limits", func(t *testing.T) {
		// Create server that returns oversized document
		largeDoc := strings.Repeat("x", 20000) // 20KB
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write([]byte(`{"id":"did:web:test","data":"` + largeDoc + `"}`))
		}))
		defer server.Close()

		config := domain.DefaultResolverConfig()
		config.AllowInsecureWeb = true
		config.MaxDIDDocSize = 1024 // 1KB limit
		resolver := domain.NewWebResolverWithConfig(config)

		serverHost := strings.TrimPrefix(server.URL, "http://")
		testDID := "did:web:" + serverHost

		ctx := context.Background()
		_, err := resolver.ResolvePublicKey(ctx, testDID)

		if err == nil {
			t.Error("Expected error for oversized document")
		}

		if !strings.Contains(err.Error(), domain.ErrDocumentTooLarge.Error()) {
			t.Errorf("Expected document too large error, got: %v", err)
		}
	})

	t.Run("request timeout", func(t *testing.T) {
		// Create server with delay
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(3 * time.Second)
			w.WriteHeader(200)
		}))
		defer server.Close()

		config := domain.DefaultResolverConfig()
		config.AllowInsecureWeb = true
		config.WebTimeoutSeconds = 1 // 1 second timeout
		resolver := domain.NewWebResolverWithConfig(config)

		serverHost := strings.TrimPrefix(server.URL, "http://")
		testDID := "did:web:" + serverHost

		ctx := context.Background()
		_, err := resolver.ResolvePublicKey(ctx, testDID)

		if err == nil {
			t.Error("Expected timeout error")
		}

		// Should be a timeout-related error
		if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "context deadline") {
			t.Errorf("Expected timeout error, got: %v", err)
		}
	})
}
