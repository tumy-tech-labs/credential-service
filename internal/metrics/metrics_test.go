package metrics

import "testing"

type counterMetrics struct {
	success map[string]int
	failure map[string]int
}

func (c *counterMetrics) IncVerificationSuccess(reason string) { c.success[reason]++ }
func (c *counterMetrics) IncVerificationFailure(reason string) { c.failure[reason]++ }

func TestNoopMetricsDoesNotPanic(t *testing.T) {
	var m VerifierMetrics = NoopVerifierMetrics{}
	m.IncVerificationSuccess("ok")
	m.IncVerificationFailure("bad")
}

func TestCounterMetricsIncrements(t *testing.T) {
	m := &counterMetrics{success: make(map[string]int), failure: make(map[string]int)}
	m.IncVerificationSuccess("ok")
	m.IncVerificationFailure("expired")

	if m.success["ok"] != 1 {
		t.Fatalf("expected success count to be 1, got %d", m.success["ok"])
	}
	if m.failure["expired"] != 1 {
		t.Fatalf("expected failure count to be 1, got %d", m.failure["expired"])
	}
}
