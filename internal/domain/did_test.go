package domain

import (
	"crypto/ed25519"
	"testing"
)

func TestDIDFromPublicKey(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	did, err := DIDFromPublicKey(pub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if did == "" {
		t.Fatalf("expected non-empty did")
	}

	if want := "did:jwk:"; len(did) <= len(want) || did[:len(want)] != want {
		t.Fatalf("did does not start with expected prefix: %s", did)
	}

	did2, err := DIDFromPublicKey(pub)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}

	if did != did2 {
		t.Fatalf("expected deterministic did values")
	}
}
