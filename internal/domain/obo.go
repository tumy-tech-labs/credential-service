package domain

// OnBehalfOfContext captures delegation context for agents acting on behalf of humans.
type OnBehalfOfContext struct {
	Subject         string   `json:"subject"`
	ActingFor       string   `json:"acting_for"`
	DelegationDepth int      `json:"delegation_depth"`
	Scope           []string `json:"scope,omitempty"`
}

// BuildOnBehalfOfContext composes acting-on-behalf-of information from a parent/child credential pair.
func BuildOnBehalfOfContext(parent *VerifiableCredential, child *VerifiableCredential) OnBehalfOfContext {
	ctx := OnBehalfOfContext{}
	if child != nil {
		ctx.Subject = child.Subject
		ctx.Scope = ScopeFromClaims(child.Claims)
	}
	if parent != nil {
		ctx.ActingFor = parent.Subject
		depth := 1
		if v, ok := parent.Claims["delegation_depth"].(float64); ok {
			depth = int(v) + 1
		}
		ctx.DelegationDepth = depth
		if len(ctx.Scope) == 0 {
			ctx.Scope = ScopeFromClaims(parent.Claims)
		}
	}
	return ctx
}
