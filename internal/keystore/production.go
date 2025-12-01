package keystore

import (
	"context"
	"crypto"
	"fmt"
	"log"
	"os"
	"strings"

	cryptoImpl "github.com/bradtumy/credential-service/internal/crypto"
)

// ProductionKeyStore provides secure key management for production environments.
// It supports multiple backends: HashiCorp Vault, Google Cloud KMS, and secure in-memory fallback.
type ProductionKeyStore struct {
	backend      string
	cache        map[string]crypto.Signer
	fallbackKS   *MemoryKeyStore
	tenantPrefix string
}

// KeyBackend represents the available key storage backends.
type KeyBackend string

const (
	BackendVault     KeyBackend = "vault"
	BackendGoogleKMS KeyBackend = "gcp-kms"
	BackendMemory    KeyBackend = "memory"
	BackendAuto      KeyBackend = "auto" // Auto-detect based on environment
)

// ProductionConfig configures the production keystore.
type ProductionConfig struct {
	Backend      KeyBackend
	TenantPrefix string // Prefix for tenant-specific keys (e.g., "tenant-")
}

// NewProductionKeyStore creates a production-ready keystore.
func NewProductionKeyStore(config ProductionConfig) (*ProductionKeyStore, error) {
	backend := config.Backend
	if backend == BackendAuto {
		backend = detectBackend()
	}

	tenantPrefix := config.TenantPrefix
	if tenantPrefix == "" {
		tenantPrefix = "tenant-"
	}

	store := &ProductionKeyStore{
		backend:      string(backend),
		cache:        make(map[string]crypto.Signer),
		fallbackKS:   NewMemoryKeyStore(),
		tenantPrefix: tenantPrefix,
	}

	// Validate backend availability
	if err := store.validateBackend(); err != nil {
		log.Printf("Warning: Production backend %s not available: %v", backend, err)
		if backend != BackendMemory {
			log.Printf("Falling back to secure in-memory keystore for development")
			store.backend = string(BackendMemory)
		}
	}

	return store, nil
}

// NewProductionKeyStoreFromEnv creates a keystore using environment configuration.
func NewProductionKeyStoreFromEnv() (*ProductionKeyStore, error) {
	config := ProductionConfig{
		Backend:      BackendAuto,
		TenantPrefix: os.Getenv("KEY_TENANT_PREFIX"),
	}

	if backendEnv := os.Getenv("KEY_BACKEND"); backendEnv != "" {
		config.Backend = KeyBackend(strings.ToLower(backendEnv))
	}

	return NewProductionKeyStore(config)
}

// GetSigningKey retrieves or creates a signing key for the given tenant.
func (p *ProductionKeyStore) GetSigningKey(tenantID string) (crypto.Signer, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenantID is required")
	}

	// Check cache first
	cacheKey := p.tenantPrefix + tenantID
	if signer, exists := p.cache[cacheKey]; exists {
		return signer, nil
	}

	// Get or create key from backend
	signer, err := p.getOrCreateKey(cacheKey)
	if err != nil {
		return nil, fmt.Errorf("get signing key for tenant %s: %w", tenantID, err)
	}

	// Cache the signer
	p.cache[cacheKey] = signer

	return signer, nil
}

func (p *ProductionKeyStore) getOrCreateKey(keyName string) (crypto.Signer, error) {
	ctx := context.Background()

	switch p.backend {
	case string(BackendVault):
		return p.getVaultKey(ctx, keyName)
	case string(BackendGoogleKMS):
		return p.getGCPKMSKey(ctx, keyName)
	case string(BackendMemory):
		return p.fallbackKS.GetSigningKey(keyName)
	default:
		return nil, fmt.Errorf("unsupported backend: %s", p.backend)
	}
}

func (p *ProductionKeyStore) getVaultKey(ctx context.Context, keyName string) (crypto.Signer, error) {
	// Set key name for Vault
	os.Setenv("VAULT_KEY_NAME", keyName)
	defer os.Unsetenv("VAULT_KEY_NAME")

	signer, err := cryptoImpl.NewVaultSignerFromEnv()
	if err != nil {
		return nil, fmt.Errorf("vault signer: %w", err)
	}

	return signer, nil
}

func (p *ProductionKeyStore) getGCPKMSKey(ctx context.Context, keyName string) (crypto.Signer, error) {
	projectID := os.Getenv("GCP_PROJECT")
	if projectID == "" {
		return nil, fmt.Errorf("GCP_PROJECT is required for KMS backend")
	}
	location := os.Getenv("KMS_LOCATION")
	if location == "" {
		location = "global"
	}
	keyRing := os.Getenv("KMS_KEYRING")
	if keyRing == "" {
		return nil, fmt.Errorf("KMS_KEYRING is required for KMS backend")
	}

	baseKeyID := os.Getenv("KMS_KEY_ID")
	if baseKeyID == "" {
		return nil, fmt.Errorf("KMS_KEY_ID is required for KMS backend")
	}

	cryptoKey := fmt.Sprintf("%s-%s", baseKeyID, keyName)
	version := os.Getenv("KMS_KEY_VERSION")
	if version == "" {
		version = "1"
	}

	keyResource := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s/cryptoKeyVersions/%s", projectID, location, keyRing, cryptoKey, version)

	signer, err := cryptoImpl.NewKMSSignerWithKeyID(ctx, keyResource)
	if err != nil {
		return nil, fmt.Errorf("kms signer: %w", err)
	}

	return signer, nil
}

func (p *ProductionKeyStore) validateBackend() error {
	ctx := context.Background()

	switch p.backend {
	case string(BackendVault):
		return p.validateVault(ctx)
	case string(BackendGoogleKMS):
		return p.validateGCPKMS(ctx)
	case string(BackendMemory):
		return nil // Always available
	default:
		return fmt.Errorf("unknown backend: %s", p.backend)
	}
}

func (p *ProductionKeyStore) validateVault(ctx context.Context) error {
	// Check if Vault is enabled and configured
	if os.Getenv("ENABLE_VAULT") != "true" && os.Getenv("VAULT_ENABLE") != "true" {
		return fmt.Errorf("vault not enabled")
	}

	if os.Getenv("VAULT_ADDR") == "" {
		return fmt.Errorf("VAULT_ADDR not configured")
	}

	if os.Getenv("VAULT_TOKEN") == "" {
		return fmt.Errorf("VAULT_TOKEN not configured")
	}

	return nil
}

func (p *ProductionKeyStore) validateGCPKMS(ctx context.Context) error {
	if os.Getenv("ENABLE_KMS") != "true" && os.Getenv("KMS_ENABLE") != "true" {
		return fmt.Errorf("gcp kms not enabled")
	}

	return nil
}

// detectBackend automatically detects the best available backend.
func detectBackend() KeyBackend {
	// Check for Vault first (preferred for on-premise/hybrid)
	if (os.Getenv("ENABLE_VAULT") == "true" || os.Getenv("VAULT_ENABLE") == "true") &&
		os.Getenv("VAULT_ADDR") != "" && os.Getenv("VAULT_TOKEN") != "" {
		return BackendVault
	}

	// Check for Google Cloud KMS
	if os.Getenv("ENABLE_KMS") == "true" || os.Getenv("KMS_ENABLE") == "true" {
		return BackendGoogleKMS
	}

	// Fallback to secure in-memory
	return BackendMemory
}

// GetBackend returns the currently configured backend.
func (p *ProductionKeyStore) GetBackend() string {
	return p.backend
}

// ClearCache clears the signer cache (useful for key rotation).
func (p *ProductionKeyStore) ClearCache() {
	p.cache = make(map[string]crypto.Signer)
}
