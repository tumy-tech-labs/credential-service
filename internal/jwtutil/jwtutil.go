package jwtutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// ValidationConfig controls how synthetic JWTs are validated.
type ValidationConfig struct {
	Issuer      string
	Audience    string
	AllowedAlgs []string
	JWKSURL     string
}

// Claims captures a parsed synthetic JWT in a safe structure.
type Claims struct {
	Subject         string
	OnBehalfOf      string
	Scope           []string
	DelegationDepth int
	Raw             map[string]any
}

// ValidateSyntheticJWT verifies a synthetic JWT against the provided configuration.
func ValidateSyntheticJWT(token string, cfg ValidationConfig) (*Claims, error) {
	if token == "" {
		return nil, errors.New("token is required")
	}
	if cfg.JWKSURL == "" {
		return nil, errors.New("jwks url is required")
	}
	algs := parseAllowedAlgorithms(cfg.AllowedAlgs)
	parsed, err := jwt.ParseSigned(token, algs)
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	if len(parsed.Headers) == 0 {
		return nil, errors.New("missing token header")
	}

	alg := string(parsed.Headers[0].Algorithm)
	if !isAlgAllowed(alg, cfg.AllowedAlgs) {
		return nil, fmt.Errorf("algorithm not allowed: %s", alg)
	}

	jwks, err := fetchJWKS(cfg.JWKSURL)
	if err != nil {
		return nil, fmt.Errorf("load jwks: %w", err)
	}

	key, err := selectVerificationKey(jwks, parsed.Headers[0].KeyID)
	if err != nil {
		return nil, err
	}

	var stdClaims jwt.Claims
	var custom struct {
		OnBehalfOf      string `json:"obo"`
		Scope           any    `json:"scope"`
		DelegationDepth int    `json:"delegation_depth"`
	}
	raw := map[string]any{}

	if err := parsed.Claims(key, &stdClaims, &custom, &raw); err != nil {
		return nil, fmt.Errorf("verify token: %w", err)
	}

	expected := jwt.Expected{Issuer: cfg.Issuer}
	if cfg.Audience != "" {
		expected.AnyAudience = jwt.Audience{cfg.Audience}
	}

	if err := stdClaims.ValidateWithLeeway(expected, time.Minute); err != nil {
		return nil, fmt.Errorf("claim validation: %w", err)
	}

	return &Claims{
		Subject:         stdClaims.Subject,
		OnBehalfOf:      custom.OnBehalfOf,
		Scope:           normalizeScope(custom.Scope),
		DelegationDepth: custom.DelegationDepth,
		Raw:             raw,
	}, nil
}

func isAlgAllowed(alg string, allowed []string) bool {
	if alg == "" {
		return false
	}
	if len(allowed) == 0 {
		return true
	}
	for _, a := range allowed {
		if a == alg {
			return true
		}
	}
	return false
}

func fetchJWKS(url string) (*jose.JSONWebKeySet, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var jwks jose.JSONWebKeySet
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("decode jwks: %w", err)
	}

	return &jwks, nil
}

func selectVerificationKey(set *jose.JSONWebKeySet, kid string) (any, error) {
	if set == nil {
		return nil, errors.New("jwks not provided")
	}

	keys := set.Key(kid)
	if len(keys) == 0 && kid == "" {
		keys = set.Keys
	}
	for _, key := range keys {
		if key.Valid() {
			return key.Key, nil
		}
	}

	return nil, fmt.Errorf("no valid key found for kid %s", kid)
}

func normalizeScope(val any) []string {
	switch v := val.(type) {
	case nil:
		return nil
	case string:
		if v == "" {
			return nil
		}
		return []string{v}
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func parseAllowedAlgorithms(algs []string) []jose.SignatureAlgorithm {
	if len(algs) == 0 {
		return nil
	}
	res := make([]jose.SignatureAlgorithm, 0, len(algs))
	for _, alg := range algs {
		res = append(res, jose.SignatureAlgorithm(alg))
	}
	return res
}
