package policy

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/lib/pq"
)

// PGStore persists policies in Postgres.
type PGStore struct {
	db *sql.DB
}

// NewPGStore constructs a Postgres policy store.
func NewPGStore(db *sql.DB) *PGStore {
	return &PGStore{db: db}
}

// ListPoliciesForTenant returns policies sorted by priority and id.
func (p *PGStore) ListPoliciesForTenant(ctx context.Context, tenantID string) ([]Policy, error) {
	const query = `SELECT id, tenant_id, name, description, effect, actions, resources, subjects, conditions, priority, enabled, created_at, updated_at FROM policies WHERE tenant_id = $1 ORDER BY priority ASC, id ASC`
	rows, err := p.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	policies := []Policy{}
	for rows.Next() {
		var (
			policy   Policy
			condJSON []byte
		)
		if err := rows.Scan(&policy.ID, &policy.TenantID, &policy.Name, &policy.Description, &policy.Effect, pq.Array(&policy.Actions), pq.Array(&policy.Resources), pq.Array(&policy.Subjects), &condJSON, &policy.Priority, &policy.Enabled, &policy.CreatedAt, &policy.UpdatedAt); err != nil {
			return nil, err
		}
		if len(condJSON) > 0 {
			if err := json.Unmarshal(condJSON, &policy.Conditions); err != nil {
				return nil, err
			}
		}
		policies = append(policies, policy)
	}
	return policies, rows.Err()
}

// CreatePolicy inserts a policy and populates ID timestamps.
func (p *PGStore) CreatePolicy(ctx context.Context, policy *Policy) error {
	if policy == nil {
		return errors.New("policy is nil")
	}
	condJSON, err := json.Marshal(policy.Conditions)
	if err != nil {
		return err
	}
	const stmt = `INSERT INTO policies (tenant_id, name, description, effect, actions, resources, subjects, conditions, priority, enabled) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id, created_at, updated_at`
	return p.db.QueryRowContext(ctx, stmt, policy.TenantID, policy.Name, policy.Description, policy.Effect, pq.Array(policy.Actions), pq.Array(policy.Resources), pq.Array(policy.Subjects), condJSON, policy.Priority, policy.Enabled).
		Scan(&policy.ID, &policy.CreatedAt, &policy.UpdatedAt)
}

// UpdatePolicy updates an existing policy.
func (p *PGStore) UpdatePolicy(ctx context.Context, policy *Policy) error {
	if policy == nil {
		return errors.New("policy is nil")
	}
	condJSON, err := json.Marshal(policy.Conditions)
	if err != nil {
		return err
	}

	const stmt = `UPDATE policies SET name = $1, description = $2, effect = $3, actions = $4, resources = $5, subjects = $6, conditions = $7, priority = $8, enabled = $9, updated_at = NOW() WHERE tenant_id = $10 AND id = $11 RETURNING updated_at`
	if err := p.db.QueryRowContext(ctx, stmt, policy.Name, policy.Description, policy.Effect, pq.Array(policy.Actions), pq.Array(policy.Resources), pq.Array(policy.Subjects), condJSON, policy.Priority, policy.Enabled, policy.TenantID, policy.ID).
		Scan(&policy.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrPolicyNotFound
		}
		return err
	}
	return nil
}

// GetPolicy fetches a single policy by tenant and id.
func (p *PGStore) GetPolicy(ctx context.Context, tenantID string, id int64) (Policy, error) {
	const query = `SELECT id, tenant_id, name, description, effect, actions, resources, subjects, conditions, priority, enabled, created_at, updated_at FROM policies WHERE tenant_id = $1 AND id = $2`
	var (
		policy   Policy
		condJSON []byte
	)
	if err := p.db.QueryRowContext(ctx, query, tenantID, id).Scan(&policy.ID, &policy.TenantID, &policy.Name, &policy.Description, &policy.Effect, pq.Array(&policy.Actions), pq.Array(&policy.Resources), pq.Array(&policy.Subjects), &condJSON, &policy.Priority, &policy.Enabled, &policy.CreatedAt, &policy.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Policy{}, ErrPolicyNotFound
		}
		return Policy{}, err
	}
	if len(condJSON) > 0 {
		if err := json.Unmarshal(condJSON, &policy.Conditions); err != nil {
			return Policy{}, err
		}
	}
	return policy, nil
}

// DeletePolicy removes a policy row.
func (p *PGStore) DeletePolicy(ctx context.Context, tenantID string, id int64) error {
	const stmt = `DELETE FROM policies WHERE tenant_id = $1 AND id = $2`
	res, err := p.db.ExecContext(ctx, stmt, tenantID, id)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrPolicyNotFound
	}
	return nil
}
