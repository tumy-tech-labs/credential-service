package domain

import "time"

// Tenant represents a logical tenant within the platform.
type Tenant struct {
	ID        string
	Name      string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TenantContext captures basic tenant-scoped request metadata.
type TenantContext struct {
	TenantID  string
	RequestID string
}
