package metrics

// VerifierMetrics describes counters for verification outcomes.
type VerifierMetrics interface {
	IncVerificationSuccess(reason string)
	IncVerificationFailure(reason string)
}

// NoopVerifierMetrics is a placeholder implementation used by default.
type NoopVerifierMetrics struct{}

// IncVerificationSuccess satisfies VerifierMetrics without recording metrics.
func (NoopVerifierMetrics) IncVerificationSuccess(reason string) {}

// IncVerificationFailure satisfies VerifierMetrics without recording metrics.
func (NoopVerifierMetrics) IncVerificationFailure(reason string) {}

// DefaultVerifierMetrics is the global metrics collector for the verifier.
var DefaultVerifierMetrics VerifierMetrics = NoopVerifierMetrics{}

// GatewayMetrics describes counters for gateway authorization flow.
type GatewayMetrics interface {
	IncAuthzRequest(tenantID string)
	IncAuthzAllow(tenantID string)
	IncAuthzDeny(tenantID string, reason string)
	IncAuthzCacheHit(tenantID string)
	IncAuthzCacheMiss(tenantID string)
}

// NoopGatewayMetrics is a placeholder implementation.
type NoopGatewayMetrics struct{}

// IncAuthzRequest implements GatewayMetrics.
func (NoopGatewayMetrics) IncAuthzRequest(string) {}

// IncAuthzAllow implements GatewayMetrics.
func (NoopGatewayMetrics) IncAuthzAllow(string) {}

// IncAuthzDeny implements GatewayMetrics.
func (NoopGatewayMetrics) IncAuthzDeny(string, string) {}

// IncAuthzCacheHit implements GatewayMetrics.
func (NoopGatewayMetrics) IncAuthzCacheHit(string) {}

// IncAuthzCacheMiss implements GatewayMetrics.
func (NoopGatewayMetrics) IncAuthzCacheMiss(string) {}

// DefaultGatewayMetrics is the global metrics collector for gateway-specific stats.
var DefaultGatewayMetrics GatewayMetrics = NoopGatewayMetrics{}
