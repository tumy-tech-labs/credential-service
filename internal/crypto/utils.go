package crypto

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
)

// generateKeyID creates a consistent key identifier from a public key.
func generateKeyID(publicKey crypto.PublicKey) (string, error) {
	switch key := publicKey.(type) {
	case ed25519.PublicKey:
		hash := sha256.Sum256(key)
		return base64.RawURLEncoding.EncodeToString(hash[:8]), nil
	default:
		return "", fmt.Errorf("unsupported key type: %T", publicKey)
	}
}

// buildEd25519JWK creates a JWK from an Ed25519 public key.
func buildEd25519JWK(publicKey ed25519.PublicKey) ([]byte, error) {
	jwk := map[string]string{
		"kty": "OKP",
		"crv": "Ed25519",
		"x":   base64.RawURLEncoding.EncodeToString(publicKey),
	}
	return json.Marshal(jwk)
}

// GenerateSecureKeyName generates a secure, unique key name for a tenant and purpose.
func GenerateSecureKeyName(tenant, purpose string) string {
	// Create random suffix for uniqueness
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	suffix := fmt.Sprintf("%x", randomBytes)
	
	return fmt.Sprintf("%s-%s-%s", tenant, purpose, suffix)
}

// HashSHA256 computes the SHA256 hash of input and returns it as a hex string.
func HashSHA256(input []byte) string {
	hash := sha256.Sum256(input)
	return fmt.Sprintf("%x", hash)
}

// ValidateSignerCompatibility validates that a signer implements the crypto.Signer interface properly.
func ValidateSignerCompatibility(signer crypto.Signer) error {
	if signer == nil {
		return fmt.Errorf("signer cannot be nil")
	}

	// Test that we can get the public key
	pubKey := signer.Public()
	if pubKey == nil {
		return fmt.Errorf("signer public key cannot be nil")
	}

	// Test that we can sign something
	testPayload := []byte("compatibility test")
	_, err := signer.Sign(rand.Reader, testPayload, nil)
	if err != nil {
		return fmt.Errorf("signer failed compatibility test: %w", err)
	}

	return nil
}

// SanitizeKeyName sanitizes a key name by removing invalid characters and converting spaces to hyphens.
func SanitizeKeyName(input string) string {
	result := ""
	for _, r := range input {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			result += string(r)
		} else if r == ' ' {
			result += "-"
		}
		// Otherwise skip the character
	}
	return result
}

// NewLocalSigner creates a new local Ed25519 signer for testing and development.
func NewLocalSigner() (crypto.Signer, error) {
	// Generate a new Ed25519 key pair directly
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519 key: %w", err)
	}
	
	// Return a simple wrapper that implements crypto.Signer
	return &testSigner{
		privateKey: priv,
		publicKey:  pub,
	}, nil
}

// testSigner is a simple wrapper that implements crypto.Signer for testing.
type testSigner struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
}

func (t *testSigner) Public() crypto.PublicKey {
	return t.publicKey
}

func (t *testSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	return ed25519.Sign(t.privateKey, digest), nil
}

// SecureKeyGenerator provides secure key generation for development and fallback scenarios.
type SecureKeyGenerator struct{}

// GenerateEd25519Key generates a new Ed25519 key pair using secure random generation.
func (g *SecureKeyGenerator) GenerateEd25519Key() (crypto.PublicKey, crypto.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate ed25519 key: %w", err)
	}
	return pub, priv, nil
}