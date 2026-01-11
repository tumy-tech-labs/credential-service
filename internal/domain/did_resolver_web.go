package domain

import (
	"context"
	"crypto"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// WebResolver resolves did:web DIDs by fetching DID documents from HTTPS endpoints.
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
	if r.allowInsecureWeb && (strings.HasPrefix(domain, "127.0.0.1") || strings.HasPrefix(domain, "localhost") || strings.Contains(domain, ":")) {
		scheme = "http"
	}

	var urlStr string
	if len(parts) == 1 {
		urlStr = fmt.Sprintf("%s://%s/.well-known/did.json", scheme, domain)
	} else {
		if len(parts) >= 2 {
			secondPart := parts[1]
			isPort := true
			for _, rch := range secondPart {
				if rch < '0' || rch > '9' {
					isPort = false
					break
				}
			}

			if isPort && len(parts) == 2 {
				urlStr = fmt.Sprintf("%s://%s:%s/.well-known/did.json", scheme, domain, secondPart)
			} else {
				if isPort && len(parts) > 2 {
					domainWithPort := fmt.Sprintf("%s:%s", domain, secondPart)
					path := strings.Join(parts[2:], "/")
					urlStr = fmt.Sprintf("%s://%s/%s/did.json", scheme, domainWithPort, path)
				} else {
					path := strings.Join(parts[1:], "/")
					urlStr = fmt.Sprintf("%s://%s/%s/did.json", scheme, domain, path)
				}
			}
		}
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL construction: %v", err)
	}

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

	req.Header.Set("Accept", "application/did+json, application/json")
	req.Header.Set("User-Agent", "credential-service/1.0")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !strings.Contains(contentType, "application/json") && !strings.Contains(contentType, "application/did+json") {
		return nil, fmt.Errorf("unexpected content type: %s", contentType)
	}

	body := http.MaxBytesReader(nil, resp.Body, r.maxDocSize)
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			return nil, fmt.Errorf("%w: document exceeds %d bytes", ErrDocumentTooLarge, r.maxDocSize)
		}
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	if doc["id"] == nil {
		return nil, errors.New("DID document missing 'id' field")
	}

	return doc, nil
}

// extractPublicKeyFromDocument extracts a public key from a DID document.
func (r *WebResolver) extractPublicKeyFromDocument(doc map[string]interface{}) (crypto.PublicKey, error) {
	verificationMethods, ok := doc["verificationMethod"]
	if !ok {
		return nil, errors.New("DID document missing 'verificationMethod' field")
	}

	methods, ok := verificationMethods.([]interface{})
	if !ok || len(methods) == 0 {
		return nil, errors.New("no verification methods found in DID document")
	}

	for _, method := range methods {
		methodMap, ok := method.(map[string]interface{})
		if !ok {
			continue
		}

		if jwkData, exists := methodMap["publicKeyJwk"]; exists {
			jwkMap, ok := jwkData.(map[string]interface{})
			if !ok {
				continue
			}

			jwkBytes, err := json.Marshal(jwkMap)
			if err != nil {
				continue
			}

			var jwk jwkKey
			if err := json.Unmarshal(jwkBytes, &jwk); err != nil {
				continue
			}

			pubKey, err := jwk.toPublicKey()
			if err == nil {
				return pubKey, nil
			}
		}
	}

	return nil, errors.New("no supported public key format found in DID document")
}
