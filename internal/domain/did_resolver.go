package domain

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DID resolution errors - following Go error handling best practices
var (
	ErrInvalidDIDFormat      = errors.New("invalid DID format")
	ErrUnsupportedDIDMethod  = errors.New("unsupported DID method")
	ErrInvalidJWK           = errors.New("invalid JWK structure")
	ErrUnsupportedKeyType    = errors.New("unsupported key type")
	ErrEmptyDID             = errors.New("empty DID")
	// did:web specific errors
	ErrWebResolutionFailed  = errors.New("did:web resolution failed")
	ErrInvalidDIDDocument   = errors.New("invalid DID document")
	ErrDocumentTooLarge     = errors.New("DID document too large")
	ErrUnsecureConnection   = errors.New("did:web requires HTTPS")
	ErrInvalidWebDID        = errors.New("invalid did:web format")
)

// DID method constants - avoid magic strings
const (
	DIDMethodJWK = "did:jwk"
	DIDMethodWeb = "did:web"
	DIDPrefix    = "did:"
)

// JWK constants - cryptographic parameters
const (
	jwkTypeOKP     = "OKP"
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
	MaxDIDLength       int
	MaxJWKSize         int
	Debug              bool
	// did:web specific configuration
	WebTimeoutSeconds  int
	MaxDIDDocSize      int64
	AllowInsecureWeb   bool // For testing only - never use in production
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
	
	return &CompositeResolver{
		resolvers: methodMap,
		config:    config,
	}, nil
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
	
	return resolver.ResolvePublicKey(ctx, did)
}

// SupportedMethods returns all methods supported by component resolvers.
func (r *CompositeResolver) SupportedMethods() []string {
	methods := make([]string, 0, len(r.resolvers))
	for method := range r.resolvers {
		methods = append(methods, method)
	}
	return methods
}

// JWKResolver resolves did:jwk DIDs by extracting embedded public keys.
// It is safe for concurrent use and performs comprehensive input validation.
type JWKResolver struct{
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
// This method is safe for concurrent use and validates all inputs.
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

// WebResolver resolves did:web DIDs by fetching DID documents from HTTPS endpoints.
// It follows the DID Web specification and enforces security best practices.
type WebResolver struct {
	client           *http.Client
	maxDocSize       int64
	allowInsecureWeb bool
}

// NewWebResolver creates a resolver for the did:web method with default config.
func NewWebResolver() *WebResolver {
	config := DefaultResolverConfig()
	return NewWebResolverWithConfig(config)
}

// NewWebResolverWithConfig creates a resolver with custom configuration.
func NewWebResolverWithConfig(config ResolverConfig) *WebResolver {
	return &WebResolver{
		client: &http.Client{
			Timeout: time.Duration(config.WebTimeoutSeconds) * time.Second,
		},
		maxDocSize:       config.MaxDIDDocSize,
		allowInsecureWeb: config.AllowInsecureWeb,
	}
}

// ResolvePublicKey resolves a did:web DID by fetching and parsing the DID document.
func (r *WebResolver) ResolvePublicKey(ctx context.Context, did string) (crypto.PublicKey, error) {
	// Input validation
	if did == "" {
		return nil, ErrEmptyDID
	}
	
	if !strings.HasPrefix(did, "did:web:") {
		return nil, fmt.Errorf("%w: not a did:web DID", ErrInvalidDIDFormat)
	}
	
	// Convert DID to URL
	didURL, err := r.didToURL(did)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidWebDID, err)
	}
	
	// Enforce HTTPS unless explicitly allowed
	if !r.allowInsecureWeb && didURL.Scheme != "https" {
		return nil, fmt.Errorf("%w: did:web requires HTTPS", ErrUnsecureConnection)
	}
	
	// Fetch DID document
	doc, err := r.fetchDIDDocument(ctx, didURL.String())
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWebResolutionFailed, err)
	}
	
	// Extract public key from document
	pubKey, err := r.extractPublicKeyFromDocument(doc)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidDIDDocument, err)
	}
	
	return pubKey, nil
}

// SupportedMethods returns the methods supported by this resolver.
func (r *WebResolver) SupportedMethods() []string {
	return []string{DIDMethodWeb}
}

// didToURL converts a did:web DID to an HTTPS URL according to the specification.
func (r *WebResolver) didToURL(did string) (*url.URL, error) {
	// Remove did:web: prefix
	identifier := strings.TrimPrefix(did, "did:web:")
	if identifier == "" {
		return nil, errors.New("empty identifier after did:web:")
	}
	
	// URL decode the identifier
	decoded, err := url.QueryUnescape(identifier)
	if err != nil {
		return nil, fmt.Errorf("invalid URL encoding: %v", err)
	}
	
	// Handle URL-encoded port numbers (e.g., %3A for :)
	// This allows did:web:example.com%3A8080 -> https://example.com:8080/.well-known/did.json
	if strings.Contains(decoded, "%3A") {
		decoded = strings.ReplaceAll(decoded, "%3A", ":")
	}
	
	// Split by : to separate domain and path
	parts := strings.Split(decoded, ":")
	if len(parts) == 0 {
		return nil, errors.New("empty domain in did:web")
	}
	
	domain := parts[0]
	if domain == "" {
		return nil, errors.New("empty domain in did:web")
	}
	
	// Construct base URL - default to HTTPS, unless explicitly allowing insecure
	scheme := "https"
	// For testing purposes, when AllowInsecureWeb is true, check if domain looks like localhost/127.0.0.1
	if r.allowInsecureWeb && (strings.HasPrefix(domain, "127.0.0.1") || strings.HasPrefix(domain, "localhost") || strings.Contains(domain, ":")) {
		scheme = "http"
	}
	
	var urlStr string
	if len(parts) == 1 {
		// No path specified, use /.well-known/did.json
		urlStr = fmt.Sprintf("%s://%s/.well-known/did.json", scheme, domain)
	} else {
		// Check if we have a port number (second part is numeric)
		if len(parts) >= 2 {
			// Try to detect if it's domain:port vs domain:path
			// Simple heuristic: if the second part is all digits, it's probably a port
			secondPart := parts[1]
			isPort := true
			for _, r := range secondPart {
				if r < '0' || r > '9' {
					isPort = false
					break
				}
			}
			
			if isPort && len(parts) == 2 {
				// It's domain:port, use /.well-known/did.json
				urlStr = fmt.Sprintf("%s://%s:%s/.well-known/did.json", scheme, domain, secondPart)
			} else {
				// It's domain:path or domain:port:path, construct full path
				if isPort && len(parts) > 2 {
					// domain:port:path...
					domainWithPort := fmt.Sprintf("%s:%s", domain, secondPart)
					path := strings.Join(parts[2:], "/")
					urlStr = fmt.Sprintf("%s://%s/%s/did.json", scheme, domainWithPort, path)
				} else {
					// domain:path...
					path := strings.Join(parts[1:], "/")
					urlStr = fmt.Sprintf("%s://%s/%s/did.json", scheme, domain, path)
				}
			}
		}
	}
	
	// Parse and validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL construction: %v", err)
	}
	
	// Additional security validations
	if parsedURL.Host == "" {
		return nil, errors.New("empty host in constructed URL")
	}
	
	return parsedURL, nil
}

// fetchDIDDocument fetches a DID document from the given URL with security controls.
func (r *WebResolver) fetchDIDDocument(ctx context.Context, urlStr string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	
	// Set appropriate headers
	req.Header.Set("Accept", "application/did+json, application/json")
	req.Header.Set("User-Agent", "credential-service/1.0")
	
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}
	
	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !strings.Contains(contentType, "application/json") && 
		!strings.Contains(contentType, "application/did+json") {
		return nil, fmt.Errorf("unexpected content type: %s", contentType)
	}
	
	// Read body with size limit
	body := http.MaxBytesReader(nil, resp.Body, r.maxDocSize)
	defer body.Close()
	
	data, err := io.ReadAll(body)
	if err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			return nil, fmt.Errorf("%w: document exceeds %d bytes", ErrDocumentTooLarge, r.maxDocSize)
		}
		return nil, fmt.Errorf("failed to read response: %v", err)
	}
	
	// Parse JSON
	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}
	
	// Basic DID document validation
	if doc["id"] == nil {
		return nil, errors.New("DID document missing 'id' field")
	}
	
	return doc, nil
}

// extractPublicKeyFromDocument extracts a public key from a DID document.
func (r *WebResolver) extractPublicKeyFromDocument(doc map[string]interface{}) (crypto.PublicKey, error) {
	// Look for verification methods
	verificationMethods, ok := doc["verificationMethod"]
	if !ok {
		return nil, errors.New("DID document missing 'verificationMethod' field")
	}
	
	methods, ok := verificationMethods.([]interface{})
	if !ok || len(methods) == 0 {
		return nil, errors.New("no verification methods found in DID document")
	}
	
	// Try to extract key from first verification method
	for _, method := range methods {
		methodMap, ok := method.(map[string]interface{})
		if !ok {
			continue
		}
		
		// Check for JWK format
		if jwkData, exists := methodMap["publicKeyJwk"]; exists {
			jwkMap, ok := jwkData.(map[string]interface{})
			if !ok {
				continue
			}
			
			// Convert to our JWK struct
			jwkBytes, err := json.Marshal(jwkMap)
			if err != nil {
				continue
			}
			
			var jwk jwkKey
			if err := json.Unmarshal(jwkBytes, &jwk); err != nil {
				continue
			}
			
			// Convert to public key
			pubKey, err := jwk.toPublicKey()
			if err == nil {
				return pubKey, nil
			}
		}
		
		// Could add support for other key formats here (publicKeyBase58, etc.)
	}
	
	return nil, errors.New("no supported public key format found in DID document")
}