package storage

import (
	"context"
	"database/sql"
)

// PGTrustRegistry persists trusted issuers in Postgres.
type PGTrustRegistry struct {
	db *sql.DB
}

// NewPGTrustRegistry constructs a Postgres-backed trust registry.
func NewPGTrustRegistry(db *sql.DB) *PGTrustRegistry {
	return &PGTrustRegistry{db: db}
}

// IsTrustedIssuer returns whether the issuer is trusted for the given tenant.
func (r *PGTrustRegistry) IsTrustedIssuer(ctx context.Context, tenantID, issuerDID string) (bool, error) {
	const query = `SELECT 1 FROM tenant_trusted_issuers WHERE tenant_id = $1 AND issuer_did = $2`
	var exists int
	if err := r.db.QueryRowContext(ctx, query, tenantID, issuerDID).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// AddTrustedIssuer inserts a trusted issuer row. Duplicate inserts are ignored.
func (r *PGTrustRegistry) AddTrustedIssuer(ctx context.Context, tenantID, issuerDID string) error {
	const stmt = `INSERT INTO tenant_trusted_issuers (tenant_id, issuer_did, metadata) VALUES ($1, $2, '{}') ON CONFLICT (tenant_id, issuer_did) DO UPDATE SET metadata = EXCLUDED.metadata`
	_, err := r.db.ExecContext(ctx, stmt, tenantID, issuerDID)
	return err
}

// RemoveTrustedIssuer deletes a trusted issuer entry.
func (r *PGTrustRegistry) RemoveTrustedIssuer(ctx context.Context, tenantID, issuerDID string) error {
	const stmt = `DELETE FROM tenant_trusted_issuers WHERE tenant_id = $1 AND issuer_did = $2`
	_, err := r.db.ExecContext(ctx, stmt, tenantID, issuerDID)
	return err
}
