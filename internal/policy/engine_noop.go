package policy

// NoOpEngine permits every request while policy rules are being developed.
type NoOpEngine struct{}

// Evaluate always allows the request.
func (NoOpEngine) Evaluate(input EvaluationInput) (EvaluationResult, error) {
	_ = input
	return EvaluationResult{Allow: true, Reason: "allow_all"}, nil
}
