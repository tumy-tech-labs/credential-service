package did

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewDIDJWK(t *testing.T) {
	doc, err := NewDIDJWK()
	if err != nil {
		t.Fatalf("NewDIDJWK returned error: %v", err)
	}

	if !strings.HasPrefix(doc.DID, "did:jwk:") {
		t.Fatalf("expected did:jwk prefix, got %s", doc.DID)
	}

	if doc.CurrentKeyID == "" {
		t.Fatalf("CurrentKeyID should be set")
	}

	if len(doc.Keys) != 1 {
		t.Fatalf("expected a single key entry, got %d", len(doc.Keys))
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	doc, err := NewDIDJWK()
	if err != nil {
		t.Fatalf("NewDIDJWK returned error: %v", err)
	}

	exported, err := ExportDID(doc)
	if err != nil {
		t.Fatalf("ExportDID returned error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(exported, &raw); err != nil {
		t.Fatalf("exported JSON invalid: %v", err)
	}

	imported, err := ImportDID(exported)
	if err != nil {
		t.Fatalf("ImportDID returned error: %v", err)
	}

	if imported.DID != doc.DID {
		t.Fatalf("imported DID mismatch: %s vs %s", imported.DID, doc.DID)
	}

	if imported.CurrentKeyID != doc.CurrentKeyID {
		t.Fatalf("current key mismatch after import")
	}
}

func TestRotateKeys(t *testing.T) {
	doc, err := NewDIDJWK()
	if err != nil {
		t.Fatalf("NewDIDJWK returned error: %v", err)
	}

	rotated, err := RotateKeys(doc)
	if err != nil {
		t.Fatalf("RotateKeys returned error: %v", err)
	}

	if rotated.DID == doc.DID {
		t.Fatalf("expected rotated DID to change")
	}

	if len(rotated.Keys) != 2 {
		t.Fatalf("expected two key entries after rotation, got %d", len(rotated.Keys))
	}

	if rotated.CurrentKeyID == doc.CurrentKeyID {
		t.Fatalf("current key id should update after rotation")
	}
}
