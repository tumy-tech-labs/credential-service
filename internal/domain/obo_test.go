package domain

import "testing"

func TestBuildOnBehalfOfContext(t *testing.T) {
	parent := &VerifiableCredential{Subject: "did:parent", Claims: map[string]interface{}{"scope": []string{"read"}, "delegation_depth": float64(1)}}
	child := &VerifiableCredential{Subject: "did:child", Claims: map[string]interface{}{"scope": []string{"read", "write"}}}

	ctx := BuildOnBehalfOfContext(parent, child)

	if ctx.Subject != "did:child" || ctx.ActingFor != "did:parent" {
		t.Fatalf("unexpected context: %+v", ctx)
	}
	if ctx.DelegationDepth != 2 {
		t.Fatalf("unexpected depth: %d", ctx.DelegationDepth)
	}
	if len(ctx.Scope) != 2 {
		t.Fatalf("unexpected scope: %v", ctx.Scope)
	}
}
