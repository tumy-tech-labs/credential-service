package domain

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWebResolver_ResolvePublicKey(t *testing.T) {
	tests := []struct {
		name           string
		did            string
		mockResponse   string
		mockStatusCode int
		expectError    bool
		errorType      error
	}{
		{
			name: "valid did:web resolution",
			did:  "did:web:example.com",
			mockResponse: `{
				"id": "did:web:example.com",
				"verificationMethod": [{
					"id": "did:web:example.com#key1",
					"type": "JsonWebKey2020",
					"controller": "did:web:example.com",
					"publicKeyJwk": {
						"kty": "OKP",
						"crv": "Ed25519",
						"x": "11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo"
					}
				}]
			}`,
			mockStatusCode: 200,
			expectError:    false,
		},
		{
			name:           "empty DID",
			did:            "",
			mockResponse:   "",
			mockStatusCode: 200,
			expectError:    true,
			errorType:      ErrEmptyDID,
		},
		{
			name:           "not a did:web DID",
			did:            "did:jwk:eyJrdHkiOiJPS1AiLCJjcnYiOiJFZDI1NTE5IiwieCI6IjExcVlBWUt4Q3JmVlNfN1R5V1FIT2c3aGN2UGFwaU1scndJYWFQY0hVUm8ifQ",
			mockResponse:   "",
			mockStatusCode: 200,
			expectError:    true,
			errorType:      ErrInvalidDIDFormat,
		},
		{
			name:           "HTTP 404 response",
			did:            "did:web:example.com",
			mockResponse:   "Not Found",
			mockStatusCode: 404,
			expectError:    true,
			errorType:      ErrWebResolutionFailed,
		},
		{
			name: "invalid JSON response",
			did:  "did:web:example.com",
			mockResponse: `{
				"id": "did:web:example.com",
				"verificationMethod": [
					invalid json
				]
			}`,
			mockStatusCode: 200,
			expectError:    true,
			errorType:      ErrWebResolutionFailed,
		},
		{
			name: "missing id in DID document",
			did:  "did:web:example.com",
			mockResponse: `{
				"verificationMethod": [{
					"id": "did:web:example.com#key1",
					"type": "JsonWebKey2020",
					"controller": "did:web:example.com",
					"publicKeyJwk": {
						"kty": "OKP",
						"crv": "Ed25519",
						"x": "11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo"
					}
				}]
			}`,
			mockStatusCode: 200,
			expectError:    true,
			errorType:      ErrWebResolutionFailed,
		},
		{
			name: "missing verification method",
			did:  "did:web:example.com",
			mockResponse: `{
				"id": "did:web:example.com"
			}`,
			mockStatusCode: 200,
			expectError:    true,
			errorType:      ErrInvalidDIDDocument,
		},
		{
			name: "invalid public key format",
			did:  "did:web:example.com",
			mockResponse: `{
				"id": "did:web:example.com",
				"verificationMethod": [{
					"id": "did:web:example.com#key1",
					"type": "JsonWebKey2020",
					"controller": "did:web:example.com",
					"publicKeyJwk": {
						"kty": "OKP",
						"crv": "Ed25519",
						"x": "invalid-base64!"
					}
				}]
			}`,
			mockStatusCode: 200,
			expectError:    true,
			errorType:      ErrInvalidDIDDocument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server - use HTTP server for testing
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			// Create resolver with custom config
			config := DefaultResolverConfig()
			config.AllowInsecureWeb = true // Allow HTTP for testing
			resolver := NewWebResolverWithConfig(config)

			// Replace the DID host with our mock server
			serverHost := strings.TrimPrefix(server.URL, "http://")
			testDID := strings.Replace(tt.did, "example.com", serverHost, 1)
			
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			pubKey, err := resolver.ResolvePublicKey(ctx, testDID)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorType != nil && !strings.Contains(err.Error(), tt.errorType.Error()) {
					t.Errorf("expected error containing %q, got %q", tt.errorType.Error(), err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if pubKey == nil {
					t.Error("expected public key but got nil")
				}
			}
		})
	}
}

func TestWebResolver_didToURL(t *testing.T) {
	tests := []struct {
		name         string
		did          string
		expectedURL  string
		expectError  bool
		allowHTTP    bool
	}{
		{
			name:        "simple domain",
			did:         "did:web:example.com",
			expectedURL: "https://example.com/.well-known/did.json",
			expectError: false,
		},
		{
			name:        "domain with path",
			did:         "did:web:example.com:user:alice",
			expectedURL: "https://example.com/user/alice/did.json",
			expectError: false,
		},
		{
			name:        "domain with URL encoded characters",
			did:         "did:web:example.com%3A8080",
			expectedURL: "https://example.com:8080/.well-known/did.json",
			expectError: false,
		},
		{
			name:        "HTTP when allowed",
			did:         "did:web:127.0.0.1:8080",
			expectedURL: "http://127.0.0.1:8080/.well-known/did.json",
			expectError: false,
			allowHTTP:   true,
		},
		{
			name:        "empty identifier",
			did:         "did:web:",
			expectedURL: "",
			expectError: true,
		},
		{
			name:        "empty domain",
			did:         "did:web::path",
			expectedURL: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultResolverConfig()
			config.AllowInsecureWeb = tt.allowHTTP
			resolver := NewWebResolverWithConfig(config)

			url, err := resolver.didToURL(tt.did)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if url.String() != tt.expectedURL {
					t.Errorf("expected URL %q, got %q", tt.expectedURL, url.String())
				}
			}
		})
	}
}

func TestWebResolver_fetchDIDDocument(t *testing.T) {
	tests := []struct {
		name           string
		response       string
		statusCode     int
		contentType    string
		expectError    bool
		maxDocSize     int64
	}{
		{
			name:        "valid JSON response",
			response:    `{"id": "did:web:example.com"}`,
			statusCode:  200,
			contentType: "application/json",
			expectError: false,
			maxDocSize:  1024,
		},
		{
			name:        "valid DID+JSON response",
			response:    `{"id": "did:web:example.com"}`,
			statusCode:  200,
			contentType: "application/did+json",
			expectError: false,
			maxDocSize:  1024,
		},
		{
			name:        "document too large",
			response:    strings.Repeat("x", 2000),
			statusCode:  200,
			contentType: "application/json",
			expectError: true,
			maxDocSize:  1024,
		},
		{
			name:        "invalid content type",
			response:    `{"id": "did:web:example.com"}`,
			statusCode:  200,
			contentType: "text/html",
			expectError: true,
			maxDocSize:  1024,
		},
		{
			name:        "HTTP error status",
			response:    "Internal Server Error",
			statusCode:  500,
			contentType: "text/plain",
			expectError: true,
			maxDocSize:  1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.contentType != "" {
					w.Header().Set("Content-Type", tt.contentType)
				}
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			config := DefaultResolverConfig()
			config.MaxDIDDocSize = tt.maxDocSize
			config.AllowInsecureWeb = true
			resolver := NewWebResolverWithConfig(config)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			doc, err := resolver.fetchDIDDocument(ctx, server.URL)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if doc == nil {
					t.Error("expected document but got nil")
				}
			}
		})
	}
}

func TestWebResolver_extractPublicKeyFromDocument(t *testing.T) {
	tests := []struct {
		name        string
		document    map[string]interface{}
		expectError bool
	}{
		{
			name: "valid Ed25519 key",
			document: map[string]interface{}{
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
			},
			expectError: false,
		},
		{
			name: "missing verification method",
			document: map[string]interface{}{
				"id": "did:web:example.com",
			},
			expectError: true,
		},
		{
			name: "empty verification method array",
			document: map[string]interface{}{
				"id":                 "did:web:example.com",
				"verificationMethod": []interface{}{},
			},
			expectError: true,
		},
		{
			name: "invalid JWK format",
			document: map[string]interface{}{
				"id": "did:web:example.com",
				"verificationMethod": []interface{}{
					map[string]interface{}{
						"id":         "did:web:example.com#key1",
						"type":       "JsonWebKey2020",
						"controller": "did:web:example.com",
						"publicKeyJwk": map[string]interface{}{
							"kty": "OKP",
							"crv": "Ed25519",
							"x":   "invalid-base64!",
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "no supported key format",
			document: map[string]interface{}{
				"id": "did:web:example.com",
				"verificationMethod": []interface{}{
					map[string]interface{}{
						"id":               "did:web:example.com#key1",
						"type":             "EcdsaSecp256k1VerificationKey2019",
						"controller":       "did:web:example.com",
						"publicKeyBase58": "some-base58-key",
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewWebResolver()

			pubKey, err := resolver.extractPublicKeyFromDocument(tt.document)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if pubKey == nil {
					t.Error("expected public key but got nil")
				}
			}
		})
	}
}

func TestWebResolver_SupportedMethods(t *testing.T) {
	resolver := NewWebResolver()
	methods := resolver.SupportedMethods()

	if len(methods) != 1 {
		t.Errorf("expected 1 method, got %d", len(methods))
	}

	if methods[0] != DIDMethodWeb {
		t.Errorf("expected %q, got %q", DIDMethodWeb, methods[0])
	}
}

func TestWebResolver_SecurityFeatures(t *testing.T) {
	t.Run("HTTPS enforcement", func(t *testing.T) {
		// Test that HTTPS is used by default when AllowInsecureWeb is false
		config := DefaultResolverConfig()
		config.AllowInsecureWeb = false
		resolver := NewWebResolverWithConfig(config)

		// Test URL construction - should always use HTTPS for regular domains
		url, err := resolver.didToURL("did:web:example.com")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		if url.Scheme != "https" {
			t.Errorf("expected HTTPS scheme, got %s", url.Scheme)
		}
	})

	t.Run("timeout handling", func(t *testing.T) {
		// Create a server that delays response
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second)
			w.WriteHeader(200)
			w.Write([]byte(`{"id": "did:web:example.com"}`))
		}))
		defer server.Close()

		config := DefaultResolverConfig()
		config.WebTimeoutSeconds = 1 // 1 second timeout
		config.AllowInsecureWeb = true
		resolver := NewWebResolverWithConfig(config)

		testDID := fmt.Sprintf("did:web:%s", strings.TrimPrefix(server.URL, "http://"))

		ctx := context.Background()
		_, err := resolver.ResolvePublicKey(ctx, testDID)

		if err == nil {
			t.Error("expected timeout error")
		}
	})

	t.Run("document size limit", func(t *testing.T) {
		// Create a server that returns a large document
		largeDoc := fmt.Sprintf(`{"id": "did:web:example.com", "data": "%s"}`, strings.Repeat("x", 2000))
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write([]byte(largeDoc))
		}))
		defer server.Close()

		config := DefaultResolverConfig()
		config.MaxDIDDocSize = 1024 // 1KB limit
		config.AllowInsecureWeb = true
		resolver := NewWebResolverWithConfig(config)

		serverHost := strings.TrimPrefix(server.URL, "http://")
		testDID := fmt.Sprintf("did:web:%s", serverHost)

		ctx := context.Background()
		_, err := resolver.ResolvePublicKey(ctx, testDID)

		if err == nil {
			t.Error("expected document too large error")
		}

		if !strings.Contains(err.Error(), ErrDocumentTooLarge.Error()) {
			t.Errorf("expected document too large error, got: %v", err)
		}
	})
}

// Benchmark tests for performance
func BenchmarkWebResolver_didToURL(b *testing.B) {
	resolver := NewWebResolver()
	did := "did:web:example.com:user:alice"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := resolver.didToURL(did)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWebResolver_extractPublicKeyFromDocument(b *testing.B) {
	resolver := NewWebResolver()
	doc := map[string]interface{}{
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := resolver.extractPublicKeyFromDocument(doc)
		if err != nil {
			b.Fatal(err)
		}
	}
}