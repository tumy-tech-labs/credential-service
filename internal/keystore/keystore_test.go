package keystore

import (
	"crypto"
	"crypto/ed25519"
	"testing"
)

func TestNewMemoryKeyStore(t *testing.T) {
	store := NewMemoryKeyStore()
	if store == nil {
		t.Fatalf("expected keystore instance")
	}
}

func TestMemoryKeyStoreReturnsSameKeyForTenant(t *testing.T) {
	store := NewMemoryKeyStore()

	signer1, err := store.GetSigningKey("tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	signer2, err := store.GetSigningKey("tenant-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	message := []byte("hello world")
	signature, err := signer1.Sign(nil, message, ed25519Options())
	if err != nil {
		t.Fatalf("signing failed: %v", err)
	}

	pubKey, ok := signer2.Public().(ed25519.PublicKey)
	if !ok {
		t.Fatalf("unexpected public key type")
	}

	if !ed25519.Verify(pubKey, message, signature) {
		t.Fatalf("signature verification failed")
	}
}

func TestMemoryKeyStoreProvidesUniqueKeysPerTenant(t *testing.T) {
	store := NewMemoryKeyStore()

	signer1, _ := store.GetSigningKey("tenant-1")
	signer2, _ := store.GetSigningKey("tenant-2")

	pub1 := signer1.Public().(ed25519.PublicKey)
	pub2 := signer2.Public().(ed25519.PublicKey)

	if string(pub1) == string(pub2) {
		t.Fatalf("expected different keys for different tenants")
	}
}

// ed25519Options returns nil to satisfy the crypto.Signer opts for Ed25519.
func ed25519Options() crypto.SignerOpts {
	return crypto.Hash(0)
}
