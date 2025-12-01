package policy

// Engine evaluates authorization decisions.
type Engine interface {
	Evaluate(input EvaluationInput) (EvaluationResult, error)
}

// EvaluationInput captures the attributes sent to the policy engine.
type EvaluationInput struct {
	TenantID         string
	Subject          string
	ActingOnBehalfOf string
	Scope            []string
	Claims           map[string]any
	Resource         string
	Action           string
	Context          map[string]any
}

// EvaluationResult is returned by the policy engine.
type EvaluationResult struct {
	Allow    bool
	Reason   string
	PolicyID *int64
}
