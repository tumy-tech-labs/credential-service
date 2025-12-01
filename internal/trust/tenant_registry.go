package trust

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
)

// TenantTrustRegistry defines tenant-scoped trust registry operations.
type TenantTrustRegistry interface {
	IsTrusted(ctx context.Context, tenantID string, issuerDID string) (bool, error)
	AddTrustedIssuer(ctx context.Context, tenantID, issuerDID string, md map[string]any) error
	ListTrustedIssuers(ctx context.Context, tenantID string) ([]string, error)
}

// InMemoryTenantRegistry implements tenant trust registry with in-memory storage.
type InMemoryTenantRegistry struct {
	mu      sync.RWMutex
	trusted map[string]map[string]map[string]any
}

// NewInMemoryTenantRegistry constructs an empty registry.
func NewInMemoryTenantRegistry() *InMemoryTenantRegistry {
	return &InMemoryTenantRegistry{trusted: make(map[string]map[string]map[string]any)}
}

// IsTrusted returns whether issuer is trusted for tenant.
func (m *InMemoryTenantRegistry) IsTrusted(_ context.Context, tenantID, issuerDID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	issuers := m.trusted[tenantID]
	if issuers == nil {
		return false, nil
	}
	_, ok := issuers[issuerDID]
	return ok, nil
}

// AddTrustedIssuer records issuer trust for tenant.
func (m *InMemoryTenantRegistry) AddTrustedIssuer(_ context.Context, tenantID, issuerDID string, md map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.trusted[tenantID] == nil {
		m.trusted[tenantID] = make(map[string]map[string]any)
	}
	m.trusted[tenantID][issuerDID] = md
	return nil
}

// ListTrustedIssuers enumerates trusted issuers for tenant.
func (m *InMemoryTenantRegistry) ListTrustedIssuers(_ context.Context, tenantID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	issuers := m.trusted[tenantID]
	res := make([]string, 0, len(issuers))
	for id := range issuers {
		res = append(res, id)
	}
	return res, nil
}

// PGTenantTrustRegistry stores trusted issuers per tenant in Postgres.
type PGTenantTrustRegistry struct {
	db *sql.DB
}

// NewPGTenantTrustRegistry constructs a Postgres-backed tenant trust registry.
func NewPGTenantTrustRegistry(db *sql.DB) *PGTenantTrustRegistry {
	return &PGTenantTrustRegistry{db: db}
}

// IsTrusted checks trust membership.
func (p *PGTenantTrustRegistry) IsTrusted(ctx context.Context, tenantID string, issuerDID string) (bool, error) {
	const query = `SELECT 1 FROM tenant_trusted_issuers WHERE tenant_id = $1 AND issuer_did = $2`
	var exists int
	if err := p.db.QueryRowContext(ctx, query, tenantID, issuerDID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// AddTrustedIssuer inserts trust entry with optional metadata.
func (p *PGTenantTrustRegistry) AddTrustedIssuer(ctx context.Context, tenantID, issuerDID string, md map[string]any) error {
	const stmt = `INSERT INTO tenant_trusted_issuers (tenant_id, issuer_did, metadata) VALUES ($1, $2, $3)
ON CONFLICT (tenant_id, issuer_did) DO UPDATE SET metadata = EXCLUDED.metadata`
	metaBytes, err := json.Marshal(md)
	if err != nil {
		return err
	}
	_, err = p.db.ExecContext(ctx, stmt, tenantID, issuerDID, metaBytes)
	return err
}

// ListTrustedIssuers lists issuers trusted for a tenant.
func (p *PGTenantTrustRegistry) ListTrustedIssuers(ctx context.Context, tenantID string) ([]string, error) {
	const query = `SELECT issuer_did FROM tenant_trusted_issuers WHERE tenant_id = $1`
	rows, err := p.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	issuers := []string{}
	for rows.Next() {
		var did string
		if err := rows.Scan(&did); err != nil {
			return nil, err
		}
		issuers = append(issuers, did)
	}
	return issuers, rows.Err()
}
