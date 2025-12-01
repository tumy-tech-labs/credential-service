package policy

import (
	"context"
	"errors"
)

// ErrPolicyNotFound indicates no policy matches the query.
var ErrPolicyNotFound = errors.New("policy not found")

// Store defines persistence behavior for policies.
type Store interface {
	ListPoliciesForTenant(ctx context.Context, tenantID string) ([]Policy, error)
	CreatePolicy(ctx context.Context, p *Policy) error
	UpdatePolicy(ctx context.Context, p *Policy) error
	GetPolicy(ctx context.Context, tenantID string, id int64) (Policy, error)
	DeletePolicy(ctx context.Context, tenantID string, id int64) error
}
