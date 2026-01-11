package keystore

import (
	"crypto"
	"fmt"
	"os"

	"github.com/bradtumy/credential-service/internal/did"
)

// FileBackedKeyStore loads signing keys from the DID FileStore.
type FileBackedKeyStore struct {
	store *did.FileStore
}

// Path returns the underlying store path.
func (f *FileBackedKeyStore) Path() string {
	return f.store.Path()
}

// NewFileBackedKeyStore constructs a FileBackedKeyStore for the given path.
func NewFileBackedKeyStore(path string) (*FileBackedKeyStore, error) {
	store := did.NewFileStore(path)
	if err := store.Load(); err != nil {
		return nil, err
	}

	return &FileBackedKeyStore{store: store}, nil
}

// NewFileBackedKeyStoreFromEnv builds a FileBackedKeyStore using DID_STORE_PATH or the default path.
func NewFileBackedKeyStoreFromEnv() (*FileBackedKeyStore, error) {
	path := os.Getenv("DID_STORE_PATH")
	if path == "" {
		path = did.DefaultStorePath()
	}
	return NewFileBackedKeyStore(path)
}

// GetSigningKey returns the current signer for the tenant, creating one if needed.
func (f *FileBackedKeyStore) GetSigningKey(tenantID string) (crypto.Signer, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenantID is required")
	}

	doc, ok := f.store.Get(tenantID)
	if !ok {
		var err error
		doc, err = did.NewDIDJWK()
		if err != nil {
			return nil, fmt.Errorf("generate did document: %w", err)
		}
		if err := f.store.Save(tenantID, doc); err != nil {
			return nil, fmt.Errorf("persist did document: %w", err)
		}
	}

	signer, err := doc.CurrentSigner()
	if err != nil {
		return nil, fmt.Errorf("load signer: %w", err)
	}

	return signer, nil
}
