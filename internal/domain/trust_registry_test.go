package domain

import (
	"context"
	"testing"
)

func TestMemoryTrustRegistry(t *testing.T) {
	registry := NewMemoryTrustRegistry()

	tenantA := "tenant-a"
	tenantB := "tenant-b"
	issuer := "did:jwk:issuer"

	if err := registry.AddTrustedIssuer(context.Background(), tenantA, issuer); err != nil {
		t.Fatalf("unexpected error adding issuer: %v", err)
	}

	trusted, err := registry.IsTrustedIssuer(context.Background(), tenantA, issuer)
	if err != nil {
		t.Fatalf("unexpected error checking trust: %v", err)
	}
	if !trusted {
		t.Fatalf("expected issuer to be trusted for tenantA")
	}

	if err := registry.RemoveTrustedIssuer(context.Background(), tenantB, issuer); err != nil {
		t.Fatalf("unexpected error removing issuer for tenantB: %v", err)
	}

	if trusted, _ := registry.IsTrustedIssuer(context.Background(), tenantB, issuer); trusted {
		t.Fatalf("issuer should not be trusted for tenantB")
	}

	if trusted, _ := registry.IsTrustedIssuer(context.Background(), tenantA, "did:jwk:unknown"); trusted {
		t.Fatalf("unexpected trust for unknown issuer")
	}
}
