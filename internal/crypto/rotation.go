package crypto

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// KeySet groups the current and previous signing keys.
type KeySet struct {
	Current  Signer
	Previous []Signer

	metadata []KeyMetadata
	byKID    map[string]KeyMetadata
}

// LoadKeySet loads keys from disk or KMS depending on environment flags.
func LoadKeySet(dir string) (*KeySet, error) {
	useKMS := os.Getenv("ENABLE_KMS") == "true" || os.Getenv("KMS_ENABLE") == "true"
	if useKMS {
		signer, err := NewKMSSignerFromEnv(context.Background())
		if err != nil {
			return nil, err
		}
		meta, err := BuildKeyMetadata(signer)
		if err != nil {
			return nil, err
		}
		return &KeySet{Current: signer, metadata: []KeyMetadata{meta}, byKID: map[string]KeyMetadata{meta.KID: meta}}, nil
	}

	if dir == "" {
		dir = "./keys"
	}

	currentDir := filepath.Join(dir, "current")
	previousDir := filepath.Join(dir, "previous")

	currentSigners, err := loadSignersFromDir(currentDir)
	if err != nil {
		return nil, err
	}
	if len(currentSigners) == 0 {
		return nil, fmt.Errorf("no current keys found in %s", currentDir)
	}

	previousSigners, err := loadSignersFromDir(previousDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	ks := &KeySet{Current: currentSigners[0], Previous: previousSigners}
	ks.metadata = []KeyMetadata{}
	ks.byKID = make(map[string]KeyMetadata)

	for _, signer := range append([]Signer{ks.Current}, ks.Previous...) {
		meta, err := BuildKeyMetadata(signer)
		if err != nil {
			return nil, err
		}
		ks.metadata = append(ks.metadata, meta)
		ks.byKID[meta.KID] = meta
	}

	return ks, nil
}

func loadSignersFromDir(dir string) ([]Signer, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	var signers []Signer
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		signer, err := NewLocalSignerFromFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		signers = append(signers, signer)
	}
	return signers, nil
}

// Resolve returns metadata for the requested key identifier.
func (k *KeySet) Resolve(kid string) (KeyMetadata, bool) {
	if k == nil {
		return KeyMetadata{}, false
	}
	meta, ok := k.byKID[kid]
	return meta, ok
}

// Metadata returns a copy of all key metadata entries.
func (k *KeySet) Metadata() []KeyMetadata {
	if k == nil {
		return nil
	}
	out := make([]KeyMetadata, len(k.metadata))
	copy(out, k.metadata)
	return out
}

// JWKS renders a public JWKS document including metadata.
func (k *KeySet) JWKS() ([]byte, error) {
	if k == nil {
		return nil, fmt.Errorf("nil keyset")
	}
	keys := make([]map[string]interface{}, 0, len(k.metadata))
	for _, meta := range k.metadata {
		var m map[string]interface{}
		if err := json.Unmarshal(meta.JWK, &m); err != nil {
			return nil, err
		}
		m["kid"] = meta.KID
		m["alg"] = meta.Alg
		m["use"] = "sig"
		if meta.NotBefore != "" {
			m["not_before"] = meta.NotBefore
		}
		if meta.NotAfter != "" {
			m["not_after"] = meta.NotAfter
		}
		keys = append(keys, m)
	}

	payload := map[string]interface{}{"keys": keys}
	return json.Marshal(payload)
}
