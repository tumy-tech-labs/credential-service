package crypto

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestVaultSigner_NewFromEnv(t *testing.T) {
	tests := []struct {
		name        string
		env         map[string]string
		expectError bool
	}{
		{
			name: "valid environment",
			env: map[string]string{
				"VAULT_ADDR":     "http://localhost:8200",
				"VAULT_TOKEN":    "test-token",
				"VAULT_KEY_NAME": "test-key",
			},
			expectError: false,
		},
		{
			name: "missing vault addr",
			env: map[string]string{
				"VAULT_TOKEN":    "test-token",
				"VAULT_KEY_NAME": "test-key",
			},
			expectError: true,
		},
		{
			name: "missing vault token",
			env: map[string]string{
				"VAULT_ADDR":     "http://localhost:8200",
				"VAULT_KEY_NAME": "test-key",
			},
			expectError: true,
		},
		{
			name: "missing key name",
			env: map[string]string{
				"VAULT_ADDR":  "http://localhost:8200",
				"VAULT_TOKEN": "test-token",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			clearVaultEnv()

			// Set test environment
			for k, v := range tt.env {
				os.Setenv(k, v)
			}
			defer func() {
				for k := range tt.env {
					os.Unsetenv(k)
				}
			}()

			// Mock server for successful tests
			var server *httptest.Server
			if !tt.expectError {
				server = newMockVaultServer()
				defer server.Close()
				os.Setenv("VAULT_ADDR", server.URL)
			}

			signer, err := NewVaultSignerFromEnv()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectError && signer == nil {
				t.Error("Expected non-nil signer")
			}
		})
	}
}

func TestVaultSigner_UninitializedAccess(t *testing.T) {
	signer := &VaultSigner{
		initialized: false,
	}

	// Test signing without initialization
	_, err := signer.Sign(rand.Reader, []byte("test"), nil)
	if err == nil {
		t.Error("Expected error when signing with uninitialized signer")
	}

	// Test public key access without initialization
	pubKey := signer.Public()
	if pubKey != nil {
		t.Error("Expected nil public key for uninitialized signer")
	}
}

// Mock Vault server for testing
func newMockVaultServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/keys/"):
			if r.Method == "POST" {
				// Create key response
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{"data": {"latest_version": 1}}`)
			} else if r.Method == "GET" {
				// Get key info response
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{
					"data": {
						"latest_version": 1,
						"keys": {
							"1": {
								"public_key": "Gb9ECWmEzf6FQbrBZ9w7lshQhqowtrbLDFw4rXAxZuE="
							}
						}
					}
				}`)
			}
		case strings.Contains(r.URL.Path, "/sign/"):
			// Sign operation response
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{
				"data": {
					"signature": "vault:v1:MEUCIQDTuYQ8o8V5Gk8W+Nh2hTxG5bJ7Nk9X/8RtZ4E3qK5r6wIgP9QbH5mN2cK8v3L7F1E9S6T4B8M3J2Q1R5C6A7D8G9K0="
				}
			}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func clearVaultEnv() {
	vars := []string{
		"VAULT_ADDR", "VAULT_TOKEN", "VAULT_KEY_NAME", "VAULT_MOUNT_PATH",
	}
	for _, v := range vars {
		os.Unsetenv(v)
	}
}