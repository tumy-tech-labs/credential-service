package domain

import (
	"context"
	"sync"
)

// TrustRegistry describes issuer trust lookups.
type TrustRegistry interface {
	IsTrustedIssuer(ctx context.Context, tenantID, issuerDID string) (bool, error)
	AddTrustedIssuer(ctx context.Context, tenantID, issuerDID string) error
	RemoveTrustedIssuer(ctx context.Context, tenantID, issuerDID string) error
}

// MemoryTrustRegistry provides an in-memory trust store.
type MemoryTrustRegistry struct {
	mu      sync.RWMutex
	trusted map[string]map[string]struct{}
}

// NewMemoryTrustRegistry constructs a MemoryTrustRegistry.
func NewMemoryTrustRegistry() *MemoryTrustRegistry {
	return &MemoryTrustRegistry{trusted: make(map[string]map[string]struct{})}
}

// IsTrustedIssuer checks whether an issuer is trusted for a tenant.
func (m *MemoryTrustRegistry) IsTrustedIssuer(_ context.Context, tenantID, issuerDID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	issuers, ok := m.trusted[tenantID]
	if !ok {
		return false, nil
	}

	_, trusted := issuers[issuerDID]
	return trusted, nil
}

// AddTrustedIssuer marks an issuer as trusted for a tenant.
func (m *MemoryTrustRegistry) AddTrustedIssuer(_ context.Context, tenantID, issuerDID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.trusted[tenantID] == nil {
		m.trusted[tenantID] = make(map[string]struct{})
	}

	m.trusted[tenantID][issuerDID] = struct{}{}
	return nil
}

// RemoveTrustedIssuer deletes an issuer from the trust registry.
func (m *MemoryTrustRegistry) RemoveTrustedIssuer(_ context.Context, tenantID, issuerDID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	issuers := m.trusted[tenantID]
	if issuers == nil {
		return nil
	}

	delete(issuers, issuerDID)
	return nil
}
