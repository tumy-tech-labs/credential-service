package policy

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// IssuancePolicy defines constraints enforced during credential issuance.
type IssuancePolicy struct {
	MaxTTLSeconds  int64    // maximum lifetime allowed for issued credentials
	AllowedScopes  []string // optional allowlist of scopes
	RequiredClaims []string // claims that must be present in the credential subject
}

// LoadIssuancePolicyFromEnv constructs a policy from environment variables.
//   - ISSUER_POLICY_MAX_TTL_SECONDS: integer max TTL (default: 86400)
//   - ISSUER_POLICY_ALLOWED_SCOPES: comma-separated list of scopes (optional)
//   - ISSUER_POLICY_REQUIRED_CLAIMS: comma-separated list of claim keys that must be provided
func LoadIssuancePolicyFromEnv() IssuancePolicy {
	policy := IssuancePolicy{
		MaxTTLSeconds: 86400,
	}

	if raw := os.Getenv("ISSUER_POLICY_MAX_TTL_SECONDS"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil && v > 0 {
			policy.MaxTTLSeconds = v
		}
	}

	if raw := os.Getenv("ISSUER_POLICY_ALLOWED_SCOPES"); raw != "" {
		policy.AllowedScopes = splitAndTrim(raw)
	}

	if raw := os.Getenv("ISSUER_POLICY_REQUIRED_CLAIMS"); raw != "" {
		policy.RequiredClaims = splitAndTrim(raw)
	}

	return policy
}

// ValidateRequest ensures the TTL and claims satisfy policy requirements.
func (p IssuancePolicy) ValidateRequest(ttlSeconds int64, claims map[string]interface{}) error {
	if p.MaxTTLSeconds > 0 && ttlSeconds > p.MaxTTLSeconds {
		return fmt.Errorf("%w: ttl_seconds exceeds policy limit of %d", ErrPolicyViolation, p.MaxTTLSeconds)
	}

	if len(p.RequiredClaims) > 0 {
		for _, key := range p.RequiredClaims {
			if _, ok := claims[key]; !ok {
				return fmt.Errorf("%w: missing required claim '%s'", ErrPolicyViolation, key)
			}
		}
	}

	if len(p.AllowedScopes) > 0 {
		if scopeVal, ok := claims["scope"]; ok {
			scopes := normalizeScope(scopeVal)
			for _, requested := range scopes {
				if !contains(p.AllowedScopes, requested) {
					return fmt.Errorf("%w: scope '%s' not permitted by issuance policy", ErrPolicyViolation, requested)
				}
			}
		}
	}

	return nil
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func splitAndTrim(raw string) []string {
	parts := strings.Split(raw, ",")
	res := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			res = append(res, trimmed)
		}
	}
	return res
}

func normalizeScope(scope any) []string {
	switch v := scope.(type) {
	case string:
		return []string{v}
	case []string:
		return v
	case []interface{}:
		res := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				res = append(res, s)
			}
		}
		return res
	default:
		return nil
	}
}

// ValidateClaimsOnly checks claims without TTL; useful for SD-JWT disclosure validation.
func (p IssuancePolicy) ValidateClaimsOnly(claims map[string]interface{}) error {
	return p.ValidateRequest(p.MaxTTLSeconds, claims)
}

// Combine allows merging another IssuancePolicy, preferring stricter limits.
func (p IssuancePolicy) Combine(other IssuancePolicy) IssuancePolicy {
	combined := p
	if other.MaxTTLSeconds > 0 && (combined.MaxTTLSeconds == 0 || other.MaxTTLSeconds < combined.MaxTTLSeconds) {
		combined.MaxTTLSeconds = other.MaxTTLSeconds
	}
	if len(other.AllowedScopes) > 0 {
		combined.AllowedScopes = other.AllowedScopes
	}
	if len(other.RequiredClaims) > 0 {
		combined.RequiredClaims = other.RequiredClaims
	}
	return combined
}

// ErrPolicyViolation is returned when issuance constraints are not met.
var ErrPolicyViolation = errors.New("issuance policy violation")
