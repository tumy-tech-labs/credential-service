package tenant

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/bradtumy/credential-service/internal/domain"
)

// Mode configures tenancy behavior.
type Mode string

const (
	ModeSingle Mode = "single"
	ModeMulti  Mode = "multi"
)

// ErrTenantNotFound indicates the tenant does not exist.
var ErrTenantNotFound = errors.New("tenant not found")

// Store persists or retrieves tenant metadata.
type Store interface {
	GetTenant(ctx context.Context, id string) (domain.Tenant, error)
	UpsertTenant(ctx context.Context, tenant domain.Tenant) error
	ListTenants(ctx context.Context) ([]domain.Tenant, error)
}

// MemoryStore is an in-memory implementation useful for tests and dev.
type MemoryStore struct {
	tenants map[string]domain.Tenant
}

// NewMemoryStore constructs an empty in-memory tenant store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{tenants: make(map[string]domain.Tenant)}
}

// GetTenant returns tenant metadata.
func (m *MemoryStore) GetTenant(_ context.Context, id string) (domain.Tenant, error) {
	tenant, ok := m.tenants[id]
	if !ok {
		return domain.Tenant{}, ErrTenantNotFound
	}
	return tenant, nil
}

// UpsertTenant creates or updates a tenant entry.
func (m *MemoryStore) UpsertTenant(_ context.Context, t domain.Tenant) error {
	if t.ID == "" {
		return errors.New("tenant id is required")
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now().UTC()
	}
	t.UpdatedAt = time.Now().UTC()
	m.tenants[t.ID] = t
	return nil
}

// ListTenants returns all tenants in memory.
func (m *MemoryStore) ListTenants(_ context.Context) ([]domain.Tenant, error) {
	out := make([]domain.Tenant, 0, len(m.tenants))
	for _, t := range m.tenants {
		out = append(out, t)
	}
	return out, nil
}

// PGTenantStore persists tenants in Postgres.
type PGTenantStore struct {
	db *sql.DB
}

// NewPGTenantStore builds a Postgres-backed tenant store.
func NewPGTenantStore(db *sql.DB) *PGTenantStore {
	return &PGTenantStore{db: db}
}

// GetTenant fetches a tenant row by ID.
func (p *PGTenantStore) GetTenant(ctx context.Context, id string) (domain.Tenant, error) {
	const query = `SELECT id, name, enabled, created_at, updated_at FROM tenants WHERE id = $1`
	var t domain.Tenant
	if err := p.db.QueryRowContext(ctx, query, id).Scan(&t.ID, &t.Name, &t.Enabled, &t.CreatedAt, &t.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Tenant{}, ErrTenantNotFound
		}
		return domain.Tenant{}, err
	}
	return t, nil
}

// UpsertTenant inserts or updates tenant metadata.
func (p *PGTenantStore) UpsertTenant(ctx context.Context, t domain.Tenant) error {
	if t.ID == "" {
		return errors.New("tenant id is required")
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now().UTC()
	}
	t.UpdatedAt = time.Now().UTC()

	const stmt = `INSERT INTO tenants (id, name, enabled, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, enabled = EXCLUDED.enabled, updated_at = EXCLUDED.updated_at`
	_, err := p.db.ExecContext(ctx, stmt, t.ID, t.Name, t.Enabled, t.CreatedAt, t.UpdatedAt)
	return err
}

// ListTenants returns tenants persisted in the database.
func (p *PGTenantStore) ListTenants(ctx context.Context) ([]domain.Tenant, error) {
	const query = `SELECT id, name, enabled, created_at, updated_at FROM tenants`
	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tenants := []domain.Tenant{}
	for rows.Next() {
		var t domain.Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Enabled, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}
	return tenants, rows.Err()
}
