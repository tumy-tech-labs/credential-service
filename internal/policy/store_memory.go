package policy

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

// MemoryStore provides an in-memory policy repository for tests and dev.
type MemoryStore struct {
	mu       sync.RWMutex
	policies map[string]map[int64]Policy
	nextID   int64
}

// NewMemoryStore constructs a MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{policies: make(map[string]map[int64]Policy), nextID: 1}
}

// ListPoliciesForTenant returns all policies scoped to a tenant, sorted by priority then ID.
func (m *MemoryStore) ListPoliciesForTenant(_ context.Context, tenantID string) ([]Policy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tenantPolicies := m.policies[tenantID]
	if tenantPolicies == nil {
		return []Policy{}, nil
	}

	result := make([]Policy, 0, len(tenantPolicies))
	for _, p := range tenantPolicies {
		result = append(result, p)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Priority == result[j].Priority {
			return result[i].ID < result[j].ID
		}
		return result[i].Priority < result[j].Priority
	})

	return result, nil
}

// CreatePolicy inserts a new policy and assigns an ID.
func (m *MemoryStore) CreatePolicy(_ context.Context, p *Policy) error {
	if p == nil {
		return errors.New("policy is nil")
	}
	if p.TenantID == "" {
		return errors.New("tenant_id is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.policies[p.TenantID] == nil {
		m.policies[p.TenantID] = make(map[int64]Policy)
	}

	p.ID = m.nextID
	m.nextID++
	now := time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	m.policies[p.TenantID][p.ID] = *p
	return nil
}

// UpdatePolicy updates an existing policy.
func (m *MemoryStore) UpdatePolicy(_ context.Context, p *Policy) error {
	if p == nil {
		return errors.New("policy is nil")
	}
	if p.TenantID == "" {
		return errors.New("tenant_id is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	tenantPolicies := m.policies[p.TenantID]
	existing, ok := tenantPolicies[p.ID]
	if !ok {
		return ErrPolicyNotFound
	}

	if p.CreatedAt.IsZero() {
		p.CreatedAt = existing.CreatedAt
	}
	p.UpdatedAt = time.Now().UTC()
	tenantPolicies[p.ID] = *p
	return nil
}

// GetPolicy retrieves a policy by tenant and ID.
func (m *MemoryStore) GetPolicy(_ context.Context, tenantID string, id int64) (Policy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if tenantPolicies := m.policies[tenantID]; tenantPolicies != nil {
		if p, ok := tenantPolicies[id]; ok {
			return p, nil
		}
	}
	return Policy{}, ErrPolicyNotFound
}

// DeletePolicy removes a policy.
func (m *MemoryStore) DeletePolicy(_ context.Context, tenantID string, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tenantPolicies := m.policies[tenantID]
	if tenantPolicies == nil {
		return ErrPolicyNotFound
	}

	if _, ok := tenantPolicies[id]; !ok {
		return ErrPolicyNotFound
	}

	delete(tenantPolicies, id)
	return nil
}
