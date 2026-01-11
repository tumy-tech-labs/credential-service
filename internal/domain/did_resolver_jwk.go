package domain

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// JWKResolver resolves did:jwk DIDs by extracting embedded public keys.
type JWKResolver struct {
	config ResolverConfig
}

// NewJWKResolver creates a resolver for the did:jwk method with default config.
func NewJWKResolver() *JWKResolver {
	return NewJWKResolverWithConfig(DefaultResolverConfig())
}

// NewJWKResolverWithConfig creates a resolver with custom configuration.
func NewJWKResolverWithConfig(config ResolverConfig) *JWKResolver {
	return &JWKResolver{config: config}
}

// ResolvePublicKey extracts the public key from a did:jwk DID.
func (r *JWKResolver) ResolvePublicKey(ctx context.Context, did string) (crypto.PublicKey, error) {
	// Check for context cancellation early
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Validate DID format
	if !strings.HasPrefix(did, DIDMethodJWK+":") {
		return nil, fmt.Errorf("%w: expected did:jwk, got %s", ErrInvalidDIDFormat, extractMethodPrefix(did))
	}

	// Extract and validate the encoded JWK part
	encodedJWK := strings.TrimPrefix(did, DIDMethodJWK+":")
	if encodedJWK == "" {
		return nil, fmt.Errorf("%w: missing JWK data", ErrInvalidJWK)
	}

	// Security: limit JWK size to prevent DoS
	if len(encodedJWK) > r.config.MaxJWKSize*4/3 { // base64 expansion factor
		return nil, fmt.Errorf("%w: JWK data too large", ErrInvalidJWK)
	}

	// Decode the base64-encoded JWK
	jwkBytes, err := base64.RawURLEncoding.DecodeString(encodedJWK)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid base64 encoding: %v", ErrInvalidJWK, err)
	}

	// Security: validate decoded size
	if len(jwkBytes) > r.config.MaxJWKSize {
		return nil, fmt.Errorf("%w: JWK too large (%d > %d bytes)", ErrInvalidJWK, len(jwkBytes), r.config.MaxJWKSize)
	}

	// Parse and validate JWK JSON
	var jwk jwkKey
	if err := json.Unmarshal(jwkBytes, &jwk); err != nil {
		return nil, fmt.Errorf("%w: invalid JSON: %v", ErrInvalidJWK, err)
	}

	// Convert JWK to crypto.PublicKey with validation
	return jwk.toPublicKey()
}

// SupportedMethods returns the DID methods this resolver supports.
func (r *JWKResolver) SupportedMethods() []string {
	return []string{"did:jwk"}
}

// jwkKey represents a JSON Web Key with proper validation.
type jwkKey struct {
	Kty string `json:"kty"` // Key type (required)
	Crv string `json:"crv"` // Curve (for OKP keys)
	X   string `json:"x"`   // Public key value (required)
}

// toPublicKey converts a JWK to a crypto.PublicKey with comprehensive validation.
func (jwk *jwkKey) toPublicKey() (crypto.PublicKey, error) {
	// Validate required fields
	if jwk.Kty == "" {
		return nil, fmt.Errorf("%w: missing 'kty' field", ErrInvalidJWK)
	}
	if jwk.X == "" {
		return nil, fmt.Errorf("%w: missing 'x' field", ErrInvalidJWK)
	}

	switch jwk.Kty {
	case jwkTypeOKP:
		return jwk.parseOKPKey()
	default:
		return nil, fmt.Errorf("%w: unsupported key type '%s'", ErrUnsupportedKeyType, jwk.Kty)
	}
}

// parseOKPKey parses an Octet Key Pair (OKP) JWK.
func (jwk *jwkKey) parseOKPKey() (crypto.PublicKey, error) {
	if jwk.Crv == "" {
		return nil, fmt.Errorf("%w: missing 'crv' field for OKP key", ErrInvalidJWK)
	}

	switch jwk.Crv {
	case jwkCurveEd25519:
		return jwk.parseEd25519Key()
	default:
		return nil, fmt.Errorf("%w: unsupported OKP curve '%s'", ErrUnsupportedKeyType, jwk.Crv)
	}
}

// parseEd25519Key parses an Ed25519 public key from JWK.
func (jwk *jwkKey) parseEd25519Key() (crypto.PublicKey, error) {
	keyBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid base64 in 'x' parameter: %v", ErrInvalidJWK, err)
	}

	if len(keyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("%w: invalid Ed25519 key size: got %d, want %d", ErrInvalidJWK, len(keyBytes), ed25519.PublicKeySize)
	}

	// Additional validation: ensure key is valid
	pubKey := ed25519.PublicKey(keyBytes)
	if len(pubKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("%w: key validation failed", ErrInvalidJWK)
	}

	return pubKey, nil
}
