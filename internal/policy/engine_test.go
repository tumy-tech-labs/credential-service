package policy

import (
	"context"
	"testing"
)

func TestEngineAllowWithMatchingPolicy(t *testing.T) {
	store := NewMemoryStore()
	p := &Policy{TenantID: "t1", Name: "allow-read", Effect: EffectAllow, Actions: []string{"read"}, Resources: []string{"orders/"}, Subjects: []string{"any"}, Priority: 5, Enabled: true}
	_ = store.CreatePolicy(context.Background(), p)

	engine := NewEngine(store)
	res, err := engine.Evaluate(EvaluationInput{TenantID: "t1", Subject: "did:example:alice", Resource: "orders/", Action: "read", Claims: map[string]any{"roles": []string{"user"}}, Scope: []string{"read"}})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if !res.Allow || res.PolicyID == nil || *res.PolicyID != p.ID {
		t.Fatalf("expected allow with policy id, got %+v", res)
	}
}

func TestEngineResourcePrefix(t *testing.T) {
	store := NewMemoryStore()
	p := &Policy{TenantID: "t1", Name: "prefix", Effect: EffectAllow, Actions: []string{"read"}, Resources: []string{"orders/*"}, Subjects: []string{"any"}, Priority: 1, Enabled: true}
	_ = store.CreatePolicy(context.Background(), p)

	engine := NewEngine(store)
	res, _ := engine.Evaluate(EvaluationInput{TenantID: "t1", Subject: "did:x", Resource: "orders/123", Action: "read"})
	if !res.Allow {
		t.Fatalf("expected allow on prefix match")
	}
}

func TestEngineRoleSubjectMatch(t *testing.T) {
	store := NewMemoryStore()
	p := &Policy{TenantID: "t1", Name: "role", Effect: EffectAllow, Actions: []string{"write"}, Resources: []string{"docs"}, Subjects: []string{"role:admin"}, Priority: 1, Enabled: true}
	_ = store.CreatePolicy(context.Background(), p)

	engine := NewEngine(store)
	res, _ := engine.Evaluate(EvaluationInput{TenantID: "t1", Subject: "did:alice", Resource: "docs", Action: "write", Claims: map[string]any{"roles": []string{"admin"}}})
	if !res.Allow {
		t.Fatalf("expected allow for matching role")
	}
}

func TestEngineConditions(t *testing.T) {
	store := NewMemoryStore()
	p := &Policy{TenantID: "t1", Name: "scoped", Effect: EffectAllow, Actions: []string{"read"}, Resources: []string{"orders"}, Subjects: []string{"any"}, Priority: 1, Enabled: true, Conditions: map[string]any{"scope_contains": "orders:read", "claim_equals": map[string]any{"tier": "gold"}}}
	_ = store.CreatePolicy(context.Background(), p)

	engine := NewEngine(store)
	res, _ := engine.Evaluate(EvaluationInput{TenantID: "t1", Subject: "did:x", Resource: "orders", Action: "read", Scope: []string{"orders:read", "payments"}, Claims: map[string]any{"tier": "gold"}})
	if !res.Allow {
		t.Fatalf("expected allow when conditions satisfied")
	}

	res2, _ := engine.Evaluate(EvaluationInput{TenantID: "t1", Subject: "did:x", Resource: "orders", Action: "read", Scope: []string{"other"}, Claims: map[string]any{"tier": "gold"}})
	if res2.Allow {
		t.Fatalf("expected deny when scope missing")
	}
}

func TestEnginePriorityDenyOverrides(t *testing.T) {
	store := NewMemoryStore()
	allow := &Policy{TenantID: "t1", Name: "allow", Effect: EffectAllow, Actions: []string{"read"}, Resources: []string{"orders"}, Subjects: []string{"any"}, Priority: 100, Enabled: true}
	deny := &Policy{TenantID: "t1", Name: "deny", Effect: EffectDeny, Actions: []string{"read"}, Resources: []string{"orders"}, Subjects: []string{"any"}, Priority: 1, Enabled: true}
	_ = store.CreatePolicy(context.Background(), allow)
	_ = store.CreatePolicy(context.Background(), deny)

	engine := NewEngine(store)
	res, _ := engine.Evaluate(EvaluationInput{TenantID: "t1", Subject: "did:x", Resource: "orders", Action: "read"})
	if res.Allow {
		t.Fatalf("expected deny due to higher priority deny policy")
	}
	if res.PolicyID == nil || *res.PolicyID != deny.ID {
		t.Fatalf("expected deny policy id")
	}
}

func TestEngineDefaultDeny(t *testing.T) {
	store := NewMemoryStore()
	engine := NewEngine(store)
	res, _ := engine.Evaluate(EvaluationInput{TenantID: "t1", Subject: "did:x", Resource: "orders", Action: "read"})
	if res.Allow || res.Reason == "" {
		t.Fatalf("expected default deny")
	}
}
