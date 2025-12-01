package domain

import (
	"context"
	"crypto/ed25519"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWKResolver_ResolvePublicKey(t *testing.T) {
	resolver := NewJWKResolver()
	
	// Generate a test key pair
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}
	
	// Create a did:jwk DID from the public key
	expectedDID, err := DIDFromPublicKey(pub)
	if err != nil {
		t.Fatalf("failed to create DID from public key: %v", err)
	}
	
	// Resolve the public key back from the DID
	resolvedKey, err := resolver.ResolvePublicKey(context.Background(), expectedDID)
	if err != nil {
		t.Fatalf("failed to resolve public key: %v", err)
	}
	
	// Verify the resolved key matches the original
	resolvedEd25519, ok := resolvedKey.(ed25519.PublicKey)
	if !ok {
		t.Fatalf("resolved key is not Ed25519: %T", resolvedKey)
	}
	
	if !resolvedEd25519.Equal(pub) {
		t.Fatalf("resolved key does not match original")
	}
}

func TestJWKResolver_InvalidDID(t *testing.T) {
	resolver := NewJWKResolver()
	
	testCases := []struct {
		name        string
		did         string
		expectedErr error
	}{
		{"not a DID", "not-a-did", ErrInvalidDIDFormat},
		{"wrong method", "did:web:example.com", ErrInvalidDIDFormat},
		{"empty JWK", "did:jwk:", ErrInvalidJWK},
		{"invalid base64", "did:jwk:invalid-base64!", ErrInvalidJWK},
		{"invalid JSON", "did:jwk:aW52YWxpZC1qc29u", ErrInvalidJWK}, // "invalid-json" in base64
		{"empty DID", "", ErrInvalidDIDFormat},
		{"too large", "did:jwk:" + strings.Repeat("a", 2000), ErrInvalidJWK},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolver.ResolvePublicKey(context.Background(), tc.did)
			require.Error(t, err)
			assert.ErrorIs(t, err, tc.expectedErr)
		})
	}
}

func TestJWKResolver_SupportedMethods(t *testing.T) {
	resolver := NewJWKResolver()
	methods := resolver.SupportedMethods()
	
	expected := []string{"did:jwk"}
	assert.Equal(t, expected, methods)
}

// TestCompositeResolver_EdgeCases tests edge cases for the composite resolver.
func TestCompositeResolver_EdgeCases(t *testing.T) {
	// Test empty resolver list
	_, err := NewCompositeResolver()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one resolver required")
	
	// Test nil resolver
	_, err = NewCompositeResolver(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil resolver provided")
	
	// Test valid resolver
	jwkResolver := NewJWKResolver()
	composite, err := NewCompositeResolver(jwkResolver)
	require.NoError(t, err)
	assert.NotNil(t, composite)
	
	// Test supported methods
	methods := composite.SupportedMethods()
	assert.Equal(t, []string{"did:jwk"}, methods)
	
	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = composite.ResolvePublicKey(ctx, "did:jwk:test")
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	
	// Test empty DID
	_, err = composite.ResolvePublicKey(context.Background(), "")
	assert.ErrorIs(t, err, ErrEmptyDID)
	
	// Test DID too long
	longDID := "did:jwk:" + strings.Repeat("a", 3000)
	_, err = composite.ResolvePublicKey(context.Background(), longDID)
	assert.ErrorIs(t, err, ErrInvalidDIDFormat)
	assert.Contains(t, err.Error(), "DID too long")
}

func TestCompositeResolver(t *testing.T) {
	jwkResolver := NewJWKResolver()
	composite, err := NewCompositeResolver(jwkResolver)
	require.NoError(t, err)
	
	// Test that composite resolver delegates to JWK resolver
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}
	
	did, err := DIDFromPublicKey(pub)
	if err != nil {
		t.Fatalf("failed to create DID: %v", err)
	}
	
	resolvedKey, err := composite.ResolvePublicKey(context.Background(), did)
	if err != nil {
		t.Fatalf("composite resolver failed: %v", err)
	}
	
	resolvedEd25519, ok := resolvedKey.(ed25519.PublicKey)
	if !ok {
		t.Fatalf("resolved key is not Ed25519: %T", resolvedKey)
	}
	
	if !resolvedEd25519.Equal(pub) {
		t.Fatalf("composite resolver returned wrong key")
	}
}

func TestCompositeResolver_UnsupportedMethod(t *testing.T) {
	jwkResolver := NewJWKResolver()
	composite, err := NewCompositeResolver(jwkResolver)
	require.NoError(t, err)
	
	_, err = composite.ResolvePublicKey(context.Background(), "did:web:example.com")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedDIDMethod)
	assert.Contains(t, err.Error(), "did:web")
}

func TestExtractDIDMethod(t *testing.T) {
	testCases := []struct {
		name        string
		did         string
		expected    string
		expectError bool
	}{
		{"valid jwk", "did:jwk:eyJrdHkiOi...", "did:jwk", false},
		{"valid web", "did:web:example.com", "did:web", false},
		{"valid key", "did:key:z6Mk...", "did:key", false},
		{"invalid format", "not-a-did", "", true},
		{"incomplete did", "did", "", true},
		{"only method", "did:onlymethod", "", true},
		{"empty", "", "", true},
		{"empty method", "did::something", "", true},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := extractDIDMethod(tc.did)
			if tc.expectError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

// TestConfigurableResolver tests custom resolver configuration
func TestConfigurableResolver(t *testing.T) {
	// Test custom configuration
	config := ResolverConfig{
		MaxDIDLength: 100, // Very small limit for testing
		MaxJWKSize:   64,  // Very small limit for testing
		Debug:        true,
	}
	
	// Test composite resolver with custom config catches DID length limit
	resolver := NewJWKResolverWithConfig(config)
	composite, err := NewCompositeResolverWithConfig(config, resolver)
	require.NoError(t, err)
	assert.NotNil(t, composite)
	
	// Test that DID length limit is enforced at composite level
	longDID := "did:jwk:" + strings.Repeat("a", 200) // Exceeds our 100 char limit
	_, err = composite.ResolvePublicKey(context.Background(), longDID)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidDIDFormat) // DID length check happens first
	assert.Contains(t, err.Error(), "DID too long")
	
	// Test JWK size limit with a smaller DID
	smallDID := "did:jwk:" + strings.Repeat("a", 80) // Within DID limit, but JWK will be too big after decode
	_, err = resolver.ResolvePublicKey(context.Background(), smallDID)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidJWK) // Should hit JWK size limit
}

// TestDefaultConfiguration tests that defaults work correctly
func TestDefaultConfiguration(t *testing.T) {
	defaultConfig := DefaultResolverConfig()
	
	assert.Equal(t, DefaultMaxDIDLength, defaultConfig.MaxDIDLength)
	assert.Equal(t, DefaultMaxJWKSize, defaultConfig.MaxJWKSize)
	assert.False(t, defaultConfig.Debug)
	
	// Test that default resolver uses default config
	resolver := NewJWKResolver()
	assert.NotNil(t, resolver)
	
	// Should work with reasonable sized DIDs (this will fail at JWK parsing, not length)
	reasonableDID := "did:jwk:" + strings.Repeat("a", 1000) // Within default limits
	_, err := resolver.ResolvePublicKey(context.Background(), reasonableDID)
	// This will fail at base64 or JSON decode, but should pass length checks
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidJWK)
	// The exact error message depends on whether it fails at base64 or JSON parsing
	assert.True(t, strings.Contains(err.Error(), "invalid base64") || strings.Contains(err.Error(), "invalid JSON"))
}