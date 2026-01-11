package did

import (
	"os"
	"testing"
)

func TestFileStoreRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/dids.json"

	store := NewFileStore(path)
	if err := store.Load(); err != nil {
		t.Fatalf("load empty store: %v", err)
	}

	doc, err := NewDIDJWK()
	if err != nil {
		t.Fatalf("NewDIDJWK: %v", err)
	}

	if err := store.Save("issuer", doc); err != nil {
		t.Fatalf("save doc: %v", err)
	}

	loaded := NewFileStore(path)
	if err := loaded.Load(); err != nil {
		t.Fatalf("reload store: %v", err)
	}

	got, ok := loaded.Get("issuer")
	if !ok {
		t.Fatalf("expected issuer label to exist")
	}

	if got.DID != doc.DID {
		t.Fatalf("expected DID %s, got %s", doc.DID, got.DID)
	}

	rotated, err := loaded.Rotate("issuer")
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}

	if rotated.DID == doc.DID {
		t.Fatalf("rotation did not update DID")
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected store file to exist: %v", err)
	}
}
