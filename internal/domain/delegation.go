package domain

import "time"

// DelegationRequest captures the input needed to mint a delegated credential.
type DelegationRequest struct {
	ParentCredentialToken string
	DelegateDID           string
	Scope                 []string
	TTL                   time.Duration
}

// DelegationInfo represents a resolved delegation relationship.
type DelegationInfo struct {
	ParentID        string
	DelegatorDID    string
	DelegateDID     string
	Scope           []string
	ExpiresAt       time.Time
	DelegationDepth int
}

// IsScopeSubset checks whether the child scope is a subset of the parent scope.
func IsScopeSubset(parentScope, childScope []string) bool {
	if len(childScope) == 0 {
		return true
	}

	parentSet := make(map[string]struct{}, len(parentScope))
	for _, s := range parentScope {
		parentSet[s] = struct{}{}
	}

	for _, s := range childScope {
		if _, ok := parentSet[s]; !ok {
			return false
		}
	}
	return true
}

// IsTTLWithinParent ensures a child credential expires no later than the parent.
func IsTTLWithinParent(parentExpiresAt, childExpiresAt time.Time) bool {
	if parentExpiresAt.IsZero() {
		return true
	}
	if childExpiresAt.IsZero() {
		return false
	}
	return !childExpiresAt.After(parentExpiresAt)
}

// IsDepthAllowed verifies the delegation depth does not exceed the configured maximum.
func IsDepthAllowed(depth, maxDepth int) bool {
	if maxDepth <= 0 {
		return true
	}
	return depth <= maxDepth
}
