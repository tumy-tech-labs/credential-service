package keystore

import (
	"crypto"
	"fmt"
	"sync"

	"github.com/bradtumy/credential-service/internal/did"
)

// KeyStore abstracts retrieval of signing keys for tenants.
type KeyStore interface {
	GetSigningKey(tenantID string) (crypto.Signer, error)
}

// MemoryKeyStore is an in-memory implementation suitable for development.
type MemoryKeyStore struct {
	mu   sync.Mutex
	keys map[string]crypto.Signer
	dids map[string]did.DIDDocument
}

// NewMemoryKeyStore constructs a MemoryKeyStore instance.
func NewMemoryKeyStore() *MemoryKeyStore {
	return &MemoryKeyStore{
		keys: make(map[string]crypto.Signer),
		dids: make(map[string]did.DIDDocument),
	}
}

// GetSigningKey returns or lazily generates a signing key for the given tenant.
func (m *MemoryKeyStore) GetSigningKey(tenantID string) (crypto.Signer, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenantID is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if signer, ok := m.keys[tenantID]; ok {
		return signer, nil
	}

	doc, err := did.NewDIDJWK()
	if err != nil {
		return nil, fmt.Errorf("generate did document: %w", err)
	}

	signer, err := doc.CurrentSigner()
	if err != nil {
		return nil, fmt.Errorf("load signer from did: %w", err)
	}

	m.keys[tenantID] = signer
	m.dids[tenantID] = doc
	return signer, nil
}
