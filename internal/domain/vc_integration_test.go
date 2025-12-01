package domain_test

import (
	"crypto"
	"io"
	"testing"
	"time"

	localcrypto "github.com/bradtumy/credential-service/internal/crypto"
	"github.com/bradtumy/credential-service/internal/domain"
)

func TestVerifiableCredentialIntegration(t *testing.T) {
	// Create an ephemeral signer for testing
	localSigner, err := localcrypto.NewEphemeralEd25519Signer()
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	// Wrap in adapter that implements crypto.Signer interface
	signer := &signerAdapter{localSigner}

	// Test DID validation
	t.Run("DID validation", func(t *testing.T) {
		tests := []struct {
			name    string
			did     string
			wantErr bool
		}{
			{"valid did:jwk", "did:jwk:eyJrdHkiOiJPS1AiLCJjcnYiOiJFZDI1NTE5IiwieCI6IjExcWFJLVN6Z0RQN1lrZmduWE51SDVzUkVOdHVmcExfSTdiNVpTbzJFUkkifQ", false},
			{"valid did:web", "did:web:example.com", false},
			{"empty DID", "", true},
			{"invalid prefix", "not:a:did", true},
			{"too short", "did:method", true},
			{"too long", "did:method:" + string(make([]byte, 1000)), true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := domain.ValidateDID(tt.did)
				if (err != nil) != tt.wantErr {
					t.Errorf("ValidateDID() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})

	// Test credential building with validation
	t.Run("BuildCredential validation", func(t *testing.T) {
		validTime := time.Now().UTC().Format(time.RFC3339)
		validSubject := map[string]interface{}{"name": "John Doe", "email": "john@example.com"}

		tests := []struct {
			name           string
			id             string
			issuer         string
			issuanceDate   string
			expirationDate string
			subject        map[string]interface{}
			wantErr        bool
		}{
			{
				"valid credential",
				"cred-123",
				"did:web:example.com",
				validTime,
				validTime,
				validSubject,
				false,
			},
			{
				"invalid issuer DID",
				"cred-123",
				"not-a-did",
				validTime,
				validTime,
				validSubject,
				true,
			},
			{
				"empty ID",
				"",
				"did:web:example.com",
				validTime,
				validTime,
				validSubject,
				true,
			},
			{
				"empty subject",
				"cred-123",
				"did:web:example.com",
				validTime,
				validTime,
				map[string]interface{}{},
				true,
			},
			{
				"invalid date format",
				"cred-123",
				"did:web:example.com",
				"not-a-date",
				validTime,
				validSubject,
				true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := domain.BuildCredential(tt.id, tt.issuer, tt.issuanceDate, tt.expirationDate, tt.subject)
				if (err != nil) != tt.wantErr {
					t.Errorf("BuildCredential() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})

	// Test credential issuance with proper JWT structure
	t.Run("IssueBasicCredential", func(t *testing.T) {
		issuerDID := "did:web:issuer.example.com"
		subjectDID := "did:web:subject.example.com"
		ttl := time.Hour
		claims := map[string]interface{}{
			"name":  "John Doe",
			"email": "john@example.com",
		}

		token, err := domain.IssueBasicCredential(issuerDID, subjectDID, signer, ttl, claims)
		if err != nil {
			t.Fatalf("IssueBasicCredential() error = %v", err)
		}

		if token == "" {
			t.Error("Expected non-empty JWT token")
		}

		// Verify JWT structure (should have 3 parts separated by dots)
		parts := splitJWT(token)
		if len(parts) != 3 {
			t.Errorf("Expected 3 JWT parts, got %d", len(parts))
		}

		t.Logf("Issued credential JWT: %s", token)
	})

	// Test error types
	t.Run("ValidationError types", func(t *testing.T) {
		err := domain.NewFieldValidationError("test_field", "test message")
		if err.Field != "test_field" {
			t.Errorf("Expected field 'test_field', got '%s'", err.Field)
		}
		if err.Code != "field_validation_error" {
			t.Errorf("Expected code 'field_validation_error', got '%s'", err.Code)
		}

		errorMsg := err.Error()
		expectedMsg := "test_field: test message"
		if errorMsg != expectedMsg {
			t.Errorf("Expected error message '%s', got '%s'", expectedMsg, errorMsg)
		}
	})
}

func splitJWT(token string) []string {
	parts := []string{}
	current := ""
	for _, char := range token {
		if char == '.' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// signerAdapter adapts internal/crypto.LocalSigner to crypto.Signer interface
type signerAdapter struct {
	localSigner *localcrypto.LocalSigner
}

func (s *signerAdapter) Public() crypto.PublicKey {
	return s.localSigner.PublicKey()
}

func (s *signerAdapter) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	return s.localSigner.Sign(rand, digest, opts)
}
