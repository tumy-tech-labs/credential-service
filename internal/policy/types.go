package policy

import "time"

// Effect defines whether a policy allows or denies.
type Effect string

const (
	// EffectAllow permits the action when matched.
	EffectAllow Effect = "allow"
	// EffectDeny forbids the action when matched.
	EffectDeny Effect = "deny"
)

// Policy represents a single authorization rule.
type Policy struct {
	ID          int64          `json:"id"`
	TenantID    string         `json:"tenant_id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Effect      Effect         `json:"effect"`
	Actions     []string       `json:"actions"`
	Resources   []string       `json:"resources"`
	Subjects    []string       `json:"subjects"`
	Conditions  map[string]any `json:"conditions,omitempty"`
	Priority    int            `json:"priority"`
	Enabled     bool           `json:"enabled"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// Condition captures arbitrary rule constraints.
type Condition map[string]any
