package policy

import (
	"context"
	"testing"
)

func TestMemoryStoreCRUD(t *testing.T) {
	store := NewMemoryStore()

	policy := &Policy{TenantID: "tenant-1", Name: "read orders", Effect: EffectAllow, Actions: []string{"read"}, Resources: []string{"orders/*"}, Subjects: []string{"any"}, Priority: 10, Enabled: true}
	if err := store.CreatePolicy(context.Background(), policy); err != nil {
		t.Fatalf("create policy: %v", err)
	}

	got, err := store.GetPolicy(context.Background(), "tenant-1", policy.ID)
	if err != nil {
		t.Fatalf("get policy: %v", err)
	}
	if got.Name != policy.Name {
		t.Fatalf("unexpected policy: %+v", got)
	}

	policy.Name = "read orders updated"
	if err := store.UpdatePolicy(context.Background(), policy); err != nil {
		t.Fatalf("update policy: %v", err)
	}

	list, err := store.ListPoliciesForTenant(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("list policies: %v", err)
	}
	if len(list) != 1 || list[0].Name != "read orders updated" {
		t.Fatalf("unexpected list: %+v", list)
	}

	if err := store.DeletePolicy(context.Background(), "tenant-1", policy.ID); err != nil {
		t.Fatalf("delete policy: %v", err)
	}

	if _, err := store.GetPolicy(context.Background(), "tenant-1", policy.ID); err == nil {
		t.Fatalf("expected error after delete")
	}
}

func TestMemoryStoreIsolation(t *testing.T) {
	store := NewMemoryStore()
	p1 := &Policy{TenantID: "a", Name: "p1", Effect: EffectAllow, Actions: []string{"read"}, Resources: []string{"a"}, Subjects: []string{"any"}, Priority: 1, Enabled: true}
	p2 := &Policy{TenantID: "b", Name: "p2", Effect: EffectAllow, Actions: []string{"read"}, Resources: []string{"b"}, Subjects: []string{"any"}, Priority: 1, Enabled: true}

	_ = store.CreatePolicy(context.Background(), p1)
	_ = store.CreatePolicy(context.Background(), p2)

	listA, _ := store.ListPoliciesForTenant(context.Background(), "a")
	listB, _ := store.ListPoliciesForTenant(context.Background(), "b")

	if len(listA) != 1 || len(listB) != 1 {
		t.Fatalf("unexpected isolation: %d %d", len(listA), len(listB))
	}
	if listA[0].TenantID != "a" || listB[0].TenantID != "b" {
		t.Fatalf("tenant mismatch")
	}
}
