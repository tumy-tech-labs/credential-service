package did

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// FileStore provides a simple JSON-backed DID store for local development.
type FileStore struct {
	path string

	mu   sync.Mutex
	docs map[string]DIDDocument
}

// NewFileStore builds a FileStore for the given path.
func NewFileStore(path string) *FileStore {
	return &FileStore{path: path, docs: make(map[string]DIDDocument)}
}

// Path returns the underlying store location.
func (s *FileStore) Path() string {
	return s.path
}

// DefaultStorePath returns the default path for persisted DIDs.
func DefaultStorePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "dids.json"
	}
	return filepath.Join(home, ".credential-service", "dids.json")
}

// Load initializes the in-memory cache from disk.
func (s *FileStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("ensure store directory: %w", err)
	}

	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		s.docs = make(map[string]DIDDocument)
		return nil
	}
	if err != nil {
		return fmt.Errorf("read store: %w", err)
	}

	var docs map[string]DIDDocument
	if err := jsonUnmarshal(data, &docs); err != nil {
		return fmt.Errorf("decode store: %w", err)
	}

	s.docs = docs
	return nil
}

// Save writes the provided DIDDocument under the supplied label.
func (s *FileStore) Save(label string, doc DIDDocument) error {
	if label == "" {
		label = doc.DID
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.docs[label] = doc
	return s.persist()
}

// Get retrieves a DIDDocument by label.
func (s *FileStore) Get(label string) (DIDDocument, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, ok := s.docs[label]
	return doc, ok
}

// List returns a copy of the in-memory DID documents keyed by label.
func (s *FileStore) List() map[string]DIDDocument {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make(map[string]DIDDocument, len(s.docs))
	for k, v := range s.docs {
		out[k] = v
	}
	return out
}

// Rotate performs a key rotation for the label and persists the change.
func (s *FileStore) Rotate(label string) (DIDDocument, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc, ok := s.docs[label]
	if !ok {
		return DIDDocument{}, fmt.Errorf("did not find entry for label %s", label)
	}

	rotated, err := RotateKeys(doc)
	if err != nil {
		return DIDDocument{}, err
	}

	s.docs[label] = rotated
	if err := s.persist(); err != nil {
		return DIDDocument{}, err
	}

	return rotated, nil
}

func (s *FileStore) persist() error {
	data, err := jsonMarshal(s.docs)
	if err != nil {
		return fmt.Errorf("encode store: %w", err)
	}

	return os.WriteFile(s.path, data, 0o600)
}
