package domain

import (
	"testing"
	"time"
)

func TestIsScopeSubset(t *testing.T) {
	parent := []string{"read", "write"}

	tests := []struct {
		name   string
		child  []string
		expect bool
	}{
		{"child subset", []string{"read"}, true},
		{"child equal", []string{"read", "write"}, true},
		{"child empty", nil, true},
		{"child expands", []string{"read", "delete"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsScopeSubset(parent, tt.child); got != tt.expect {
				t.Fatalf("expected %v, got %v", tt.expect, got)
			}
		})
	}
}

func TestIsTTLWithinParent(t *testing.T) {
	now := time.Now()
	parent := now.Add(5 * time.Minute)
	childSooner := now.Add(3 * time.Minute)
	childLater := now.Add(10 * time.Minute)

	if !IsTTLWithinParent(parent, childSooner) {
		t.Fatalf("expected child expiring sooner to be allowed")
	}

	if IsTTLWithinParent(parent, childLater) {
		t.Fatalf("expected child expiring later to be rejected")
	}
}

func TestIsDepthAllowed(t *testing.T) {
	maxDepth := 3

	if !IsDepthAllowed(2, maxDepth) {
		t.Fatalf("depth within max should be allowed")
	}

	if IsDepthAllowed(4, maxDepth) {
		t.Fatalf("depth beyond max should not be allowed")
	}
}
