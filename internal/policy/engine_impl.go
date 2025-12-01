package policy

import (
	"context"
	"errors"
	"sort"
	"strings"
)

// EngineV1 evaluates policies stored in a Store.
type EngineV1 struct {
	store Store
}

// NewEngine constructs a V1 engine using the provided store.
func NewEngine(store Store) *EngineV1 {
	return &EngineV1{store: store}
}

// Evaluate resolves policies for the tenant and determines an authorization decision.
func (e *EngineV1) Evaluate(input EvaluationInput) (EvaluationResult, error) {
	if e == nil || e.store == nil {
		return EvaluationResult{}, errors.New("policy store not configured")
	}

	policies, err := e.store.ListPoliciesForTenant(context.Background(), input.TenantID)
	if err != nil {
		return EvaluationResult{}, err
	}

	matching := filterMatchingPolicies(policies, input)
	if len(matching) == 0 {
		return EvaluationResult{Allow: false, Reason: "no_matching_policy"}, nil
	}

	sort.SliceStable(matching, func(i, j int) bool {
		if matching[i].Priority == matching[j].Priority {
			return matching[i].ID < matching[j].ID
		}
		return matching[i].Priority < matching[j].Priority
	})

	var allowPolicy *Policy
	for i := range matching {
		p := matching[i]
		if strings.EqualFold(string(p.Effect), string(EffectDeny)) {
			return EvaluationResult{Allow: false, Reason: "policy_deny", PolicyID: &p.ID}, nil
		}
		if strings.EqualFold(string(p.Effect), string(EffectAllow)) && allowPolicy == nil {
			allowPolicy = &p
		}
	}

	if allowPolicy != nil {
		return EvaluationResult{Allow: true, Reason: "policy_allow", PolicyID: &allowPolicy.ID}, nil
	}

	return EvaluationResult{Allow: false, Reason: "no_allowing_policy"}, nil
}

func filterMatchingPolicies(policies []Policy, input EvaluationInput) []Policy {
	matches := []Policy{}
	for i := range policies {
		p := policies[i]
		if !p.Enabled {
			continue
		}
		if !matchesAction(p, input.Action) {
			continue
		}
		if !matchesResource(p, input.Resource) {
			continue
		}
		if !matchesSubject(p, input.Subject, input.Claims) {
			continue
		}
		if !matchesConditions(p, input) {
			continue
		}
		matches = append(matches, p)
	}
	return matches
}

func matchesAction(p Policy, action string) bool {
	for _, a := range p.Actions {
		if strings.EqualFold(a, action) {
			return true
		}
	}
	return false
}

func matchesResource(p Policy, resource string) bool {
	for _, r := range p.Resources {
		if r == resource {
			return true
		}
		if strings.HasSuffix(r, "*") {
			prefix := strings.TrimSuffix(r, "*")
			if strings.HasPrefix(resource, prefix) {
				return true
			}
		}
	}
	return false
}

func matchesSubject(p Policy, subject string, claims map[string]any) bool {
	for _, s := range p.Subjects {
		if s == "any" {
			return true
		}
		if s == subject {
			return true
		}
		if strings.HasPrefix(s, "role:") {
			role := strings.TrimPrefix(s, "role:")
			if role == "" {
				continue
			}
			if hasRole(claims, role) {
				return true
			}
		}
	}
	return false
}

func hasRole(claims map[string]any, role string) bool {
	if claims == nil {
		return false
	}
	rolesVal, ok := claims["roles"]
	if !ok {
		return false
	}

	switch v := rolesVal.(type) {
	case []string:
		for _, r := range v {
			if r == role {
				return true
			}
		}
	case []any:
		for _, item := range v {
			if str, ok := item.(string); ok && str == role {
				return true
			}
		}
	}
	return false
}

func matchesConditions(p Policy, input EvaluationInput) bool {
	if len(p.Conditions) == 0 {
		return true
	}

	for key, val := range p.Conditions {
		switch key {
		case "scope_contains":
			if !scopeContains(input.Scope, val) {
				return false
			}
		case "claim_equals":
			condMap, ok := val.(map[string]any)
			if !ok {
				return false
			}
			for claimKey, claimVal := range condMap {
				if !claimEquals(input.Claims, claimKey, claimVal) {
					return false
				}
			}
		default:
			return false
		}
	}
	return true
}

func scopeContains(scope []string, value any) bool {
	switch v := value.(type) {
	case string:
		for _, s := range scope {
			if s == v {
				return true
			}
		}
		return false
	case []any:
		for _, item := range v {
			if str, ok := item.(string); ok {
				found := false
				for _, s := range scope {
					if s == str {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
		}
		return true
	case []string:
		for _, str := range v {
			found := false
			for _, s := range scope {
				if s == str {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func claimEquals(claims map[string]any, key string, expected any) bool {
	if claims == nil {
		return false
	}
	val, ok := claims[key]
	if !ok {
		return false
	}
	switch exp := expected.(type) {
	case string:
		str, ok := val.(string)
		return ok && str == exp
	case float64:
		switch actual := val.(type) {
		case float64:
			return actual == exp
		case int:
			return float64(actual) == exp
		}
	case int:
		switch actual := val.(type) {
		case int:
			return actual == exp
		case float64:
			return int(actual) == exp
		}
	default:
		return val == expected
	}
	return false
}
