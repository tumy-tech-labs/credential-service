package crypto

import (
	"crypto"
	"crypto/rand"
	"testing"
)

func TestGenerateSecureKeyName(t *testing.T) {
	tests := []struct {
		name      string
		tenant    string
		purpose   string
		expectLen int
	}{
		{
			name:      "basic generation",
			tenant:    "test-tenant",
			purpose:   "signing",
			expectLen: len("test-tenant-signing-") + 16, // 16 hex chars for 8 random bytes
		},
		{
			name:      "empty tenant",
			tenant:    "",
			purpose:   "signing",
			expectLen: len("-signing-") + 16,
		},
		{
			name:      "empty purpose",
			tenant:    "tenant",
			purpose:   "",
			expectLen: len("tenant--") + 16,
		},
		{
			name:      "long names",
			tenant:    "very-long-tenant-name-for-testing",
			purpose:   "credential-signing-operations",
			expectLen: len("very-long-tenant-name-for-testing-credential-signing-operations-") + 16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyName := GenerateSecureKeyName(tt.tenant, tt.purpose)
			
			if len(keyName) != tt.expectLen {
				t.Errorf("Expected key name length %d, got %d (key: %s)", tt.expectLen, len(keyName), keyName)
			}

			// Verify format
			expectedPrefix := tt.tenant + "-" + tt.purpose + "-"
			if !startsWith(keyName, expectedPrefix) {
				t.Errorf("Expected key name to start with '%s', got '%s'", expectedPrefix, keyName)
			}

			// Verify suffix is hex
			suffix := keyName[len(expectedPrefix):]
			if len(suffix) != 16 {
				t.Errorf("Expected 16 character hex suffix, got %d characters", len(suffix))
			}
			for _, c := range suffix {
				if !isHex(c) {
					t.Errorf("Non-hex character in suffix: %c", c)
				}
			}
		})
	}
}

func TestGenerateSecureKeyName_Uniqueness(t *testing.T) {
	keys := make(map[string]bool)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		keyName := GenerateSecureKeyName("test", "purpose")
		if keys[keyName] {
			t.Fatalf("Duplicate key name generated: %s", keyName)
		}
		keys[keyName] = true
	}

	if len(keys) != iterations {
		t.Errorf("Expected %d unique keys, got %d", iterations, len(keys))
	}
}

func TestHashSHA256(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string // First few chars of expected hash
	}{
		{
			name:     "empty input",
			input:    []byte{},
			expected: "e3b0c44298fc1c14", // SHA256 of empty string starts with this
		},
		{
			name:     "simple string",
			input:    []byte("hello"),
			expected: "2cf24dba5fb0a30e", // SHA256 of "hello" starts with this
		},
		{
			name:     "longer content",
			input:    []byte("The quick brown fox jumps over the lazy dog"),
			expected: "d7a8fbb307d78094", // SHA256 of this phrase starts with this
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := HashSHA256(tt.input)
			
			if len(hash) != 64 { // SHA256 hex string is 64 chars
				t.Errorf("Expected hash length 64, got %d", len(hash))
			}

			if !startsWith(hash, tt.expected) {
				t.Errorf("Expected hash to start with %s, got %s", tt.expected, hash[:16])
			}

			// Verify it's valid hex
			for _, c := range hash {
				if !isHex(c) {
					t.Errorf("Non-hex character in hash: %c", c)
				}
			}
		})
	}
}

func TestHashSHA256_Deterministic(t *testing.T) {
	input := []byte("test input for deterministic hash")
	
	hash1 := HashSHA256(input)
	hash2 := HashSHA256(input)
	
	if hash1 != hash2 {
		t.Error("SHA256 hash should be deterministic")
	}
}

func TestValidateSignerCompatibility(t *testing.T) {
	// Create a test signer (using local signer for simplicity)
	signer, err := NewLocalSigner()
	if err != nil {
		t.Fatalf("Failed to create test signer: %v", err)
	}

	tests := []struct {
		name        string
		signer      crypto.Signer
		expectError bool
	}{
		{
			name:        "valid signer",
			signer:      signer,
			expectError: false,
		},
		{
			name:        "nil signer",
			signer:      nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSignerCompatibility(tt.signer)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestValidateSignerCompatibility_RealSigning(t *testing.T) {
	signer, err := NewLocalSigner()
	if err != nil {
		t.Fatalf("Failed to create test signer: %v", err)
	}

	err = ValidateSignerCompatibility(signer)
	if err != nil {
		t.Fatalf("Signer validation failed: %v", err)
	}

	// Test that the signer actually works
	payload := []byte("test payload")
	signature, err := signer.Sign(rand.Reader, payload, nil)
	if err != nil {
		t.Errorf("Validated signer failed to sign: %v", err)
	}

	if len(signature) == 0 {
		t.Error("Expected non-empty signature")
	}
}

func TestSanitizeKeyName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean input",
			input:    "test-key-123",
			expected: "test-key-123",
		},
		{
			name:     "spaces and special chars",
			input:    "test key!@#$%^&*()",
			expected: "test-key", // Spaces converted to hyphens, special chars removed
		},
		{
			name:     "underscores preserved",
			input:    "test_key_name",
			expected: "test_key_name",
		},
		{
			name:     "dots preserved",
			input:    "test.key.name",
			expected: "test.key.name",
		},
		{
			name:     "mixed case preserved",
			input:    "TestKeyName",
			expected: "TestKeyName",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only special chars",
			input:    "!@#$%^&*()",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeKeyName(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// Helper functions
func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func isHex(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}