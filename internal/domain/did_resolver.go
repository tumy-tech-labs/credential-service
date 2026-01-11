package domain

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// DID resolution errors - following Go error handling best practices
var (
	ErrInvalidDIDFormat     = errors.New("invalid DID format")
	ErrUnsupportedDIDMethod = errors.New("unsupported DID method")
	ErrInvalidJWK           = errors.New("invalid JWK structure")
	ErrUnsupportedKeyType   = errors.New("unsupported key type")
	ErrEmptyDID             = errors.New("empty DID")
	// did:web specific errors
	ErrWebResolutionFailed = errors.New("did:web resolution failed")
	ErrInvalidDIDDocument  = errors.New("invalid DID document")
	ErrDocumentTooLarge    = errors.New("DID document too large")
	ErrUnsecureConnection  = errors.New("did:web requires HTTPS")
	ErrInvalidWebDID       = errors.New("invalid did:web format")
)

// DID method constants - avoid magic strings
const (
	DIDMethodJWK = "did:jwk"
	DIDMethodWeb = "did:web"
	DIDPrefix    = "did:"
)

// JWK constants - cryptographic parameters
const (
	jwkTypeOKP      = "OKP"
	jwkCurveEd25519 = "Ed25519"
)

// Default limits - can be overridden via configuration
const (
	// DefaultMaxDIDLength - maximum DID length for security (prevent DoS)
	DefaultMaxDIDLength = 2048
	// DefaultMaxJWKSize - maximum JWK size for security
	DefaultMaxJWKSize = 1024
	// DefaultWebTimeout - default timeout for did:web resolution
	DefaultWebTimeoutSeconds = 10
	// DefaultMaxDIDDocSize - maximum DID document size for did:web
	DefaultMaxDIDDocSize = 10240 // 10KB
)

// ResolverConfig holds configuration options for DID resolvers.
type ResolverConfig struct {
	MaxDIDLength int
	MaxJWKSize   int
	Debug        bool
	// did:web specific configuration
	WebTimeoutSeconds int
	MaxDIDDocSize     int64
	AllowInsecureWeb  bool // For testing only - never use in production
	// CacheTTLSeconds configures an in-memory TTL (seconds) for resolved DID public keys.
	// A value <= 0 disables caching.
	CacheTTLSeconds int
}

// DefaultResolverConfig returns sensible defaults for production use.
func DefaultResolverConfig() ResolverConfig {
	return ResolverConfig{
		MaxDIDLength:      DefaultMaxDIDLength,
		MaxJWKSize:        DefaultMaxJWKSize,
		Debug:             false,
		WebTimeoutSeconds: DefaultWebTimeoutSeconds,
		MaxDIDDocSize:     int64(DefaultMaxDIDDocSize),
		AllowInsecureWeb:  false, // Security: HTTPS only in production
		CacheTTLSeconds:   60,    // default short-lived cache
	}
}

// DIDResolver resolves Decentralized Identifiers (DIDs) to cryptographic public keys
// for verifying verifiable credentials in a distributed environment.
//
// Implementations must be safe for concurrent use and should validate all inputs
// to prevent security vulnerabilities.
type DIDResolver interface {
	// ResolvePublicKey extracts the verification public key from a DID.
	// The context should be used for cancellation and timeouts.
	//
	// Returns ErrInvalidDIDFormat for malformed DIDs,
	// ErrUnsupportedDIDMethod for unknown methods, or
	// method-specific errors for resolution failures.
	ResolvePublicKey(ctx context.Context, did string) (crypto.PublicKey, error)

	// SupportedMethods returns the DID methods this resolver can handle.
	// The returned slice should not be modified by callers.
	SupportedMethods() []string
}

// CompositeResolver combines multiple resolvers for different DID methods.
// It is safe for concurrent use.
type CompositeResolver struct {
	// resolvers maps DID methods to their corresponding resolvers
	resolvers map[string]DIDResolver
	config    ResolverConfig
	// in-memory cache for resolved public keys
	cache    map[string]cacheEntry
	cacheTTL time.Duration
	mu       sync.RWMutex
}

// NewCompositeResolver creates a resolver with default configuration.
func NewCompositeResolver(resolvers ...DIDResolver) (*CompositeResolver, error) {
	return NewCompositeResolverWithConfig(DefaultResolverConfig(), resolvers...)
}

// NewCompositeResolverWithConfig creates a resolver that delegates to method-specific resolvers.
// If multiple resolvers support the same method, the last one wins.
// Returns an error if no resolvers are provided.
func NewCompositeResolverWithConfig(config ResolverConfig, resolvers ...DIDResolver) (*CompositeResolver, error) {
	if len(resolvers) == 0 {
		return nil, errors.New("at least one resolver required")
	}

	methodMap := make(map[string]DIDResolver)

	for _, resolver := range resolvers {
		if resolver == nil {
			return nil, errors.New("nil resolver provided")
		}

		for _, method := range resolver.SupportedMethods() {
			if method == "" {
				return nil, errors.New("empty method name not allowed")
			}
			methodMap[method] = resolver
		}
	}

	if len(methodMap) == 0 {
		return nil, errors.New("no valid DID methods found")
	}

	cr := &CompositeResolver{
		resolvers: methodMap,
		config:    config,
	}

	if config.CacheTTLSeconds > 0 {
		cr.cache = make(map[string]cacheEntry)
		cr.cacheTTL = time.Duration(config.CacheTTLSeconds) * time.Second
	}

	return cr, nil
}

// ResolvePublicKey resolves a DID using the appropriate method-specific resolver.
func (r *CompositeResolver) ResolvePublicKey(ctx context.Context, did string) (crypto.PublicKey, error) {
	// Input validation for security
	if did == "" {
		return nil, ErrEmptyDID
	}

	if len(did) > r.config.MaxDIDLength {
		return nil, fmt.Errorf("%w: DID too long (%d > %d)", ErrInvalidDIDFormat, len(did), r.config.MaxDIDLength)
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	method, err := extractDIDMethod(did)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidDIDFormat, err)
	}

	resolver, exists := r.resolvers[method]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedDIDMethod, method)
	}

	// Check cache first (if enabled)
	if r.cache != nil {
		r.mu.RLock()
		if e, ok := r.cache[did]; ok {
			if time.Now().Before(e.expires) {
				pub := e.key
				r.mu.RUnlock()
				return pub, nil
			}
			// expired
		}
		r.mu.RUnlock()
	}

	pub, err := resolver.ResolvePublicKey(ctx, did)
	if err != nil {
		return nil, err
	}

	// store in cache
	if r.cache != nil && pub != nil {
		r.mu.Lock()
		r.cache[did] = cacheEntry{key: pub, expires: time.Now().Add(r.cacheTTL)}
		r.mu.Unlock()
	}

	return pub, nil
}

// SupportedMethods returns all methods supported by component resolvers.
func (r *CompositeResolver) SupportedMethods() []string {
	methods := make([]string, 0, len(r.resolvers))
	for method := range r.resolvers {
		methods = append(methods, method)
	}
	return methods
}

// extractDIDMethod extracts the method from a DID string with proper validation.
func extractDIDMethod(did string) (string, error) {
	if did == "" {
		return "", errors.New("empty DID")
	}

	parts := strings.SplitN(did, ":", 3)
	if len(parts) < 3 {
		return "", fmt.Errorf("DID must have at least 3 parts separated by ':'")
	}

	if parts[0] != "did" {
		return "", fmt.Errorf("DID must start with 'did:'")
	}

	if parts[1] == "" {
		return "", fmt.Errorf("DID method cannot be empty")
	}

	return DIDPrefix + parts[1], nil
}

// extractMethodPrefix extracts just the "did:method" part for error messages.
func extractMethodPrefix(did string) string {
	parts := strings.SplitN(did, ":", 3)
	if len(parts) >= 2 {
		return DIDPrefix + parts[1]
	}
	return "invalid"
}

// cacheEntry stores a resolved public key and its expiry time.
type cacheEntry struct {
	key     crypto.PublicKey
	expires time.Time
}
