package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusGatewayMetrics implements GatewayMetrics with Prometheus collectors.
type PrometheusGatewayMetrics struct {
	authzRequests   *prometheus.CounterVec
	authzAllowed    *prometheus.CounterVec
	authzDenied     *prometheus.CounterVec
	authzCacheHits  *prometheus.CounterVec
	authzCacheMiss  *prometheus.CounterVec
	authzDuration   *prometheus.HistogramVec
	rateLimitHits   *prometheus.CounterVec
	delegationDepth *prometheus.HistogramVec
}

// NewPrometheusGatewayMetrics creates Prometheus-backed gateway metrics.
func NewPrometheusGatewayMetrics(namespace string) *PrometheusGatewayMetrics {
	return NewPrometheusGatewayMetricsWithRegistry(namespace, prometheus.DefaultRegisterer)
}

// NewPrometheusGatewayMetricsWithRegistry creates Prometheus-backed gateway metrics with custom registry.
func NewPrometheusGatewayMetricsWithRegistry(namespace string, registerer prometheus.Registerer) *PrometheusGatewayMetrics {
	if namespace == "" {
		namespace = "credential_service"
	}

	factory := promauto.With(registerer)

	return &PrometheusGatewayMetrics{
		authzRequests: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "gateway_authz_requests_total",
				Help:      "Total authorization requests by tenant",
			},
			[]string{"tenant_id"},
		),
		authzAllowed: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "gateway_authz_allowed_total",
				Help:      "Total allowed authorization requests by tenant",
			},
			[]string{"tenant_id"},
		),
		authzDenied: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "gateway_authz_denied_total",
				Help:      "Total denied authorization requests by tenant and reason",
			},
			[]string{"tenant_id", "reason"},
		),
		authzCacheHits: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "gateway_authz_cache_hits_total",
				Help:      "Total authorization cache hits by tenant",
			},
			[]string{"tenant_id"},
		),
		authzCacheMiss: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "gateway_authz_cache_miss_total",
				Help:      "Total authorization cache misses by tenant",
			},
			[]string{"tenant_id"},
		),
		authzDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "gateway_authz_duration_seconds",
				Help:      "Authorization request duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"tenant_id", "result"},
		),
		rateLimitHits: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "gateway_rate_limit_hits_total",
				Help:      "Total rate limit hits by tenant",
			},
			[]string{"tenant_id"},
		),
		delegationDepth: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "gateway_delegation_depth",
				Help:      "Distribution of delegation chain depths",
				Buckets:   []float64{0, 1, 2, 3, 4, 5, 10},
			},
			[]string{"tenant_id"},
		),
	}
}

// IncAuthzRequest implements GatewayMetrics.
func (p *PrometheusGatewayMetrics) IncAuthzRequest(tenantID string) {
	p.authzRequests.WithLabelValues(tenantID).Inc()
}

// IncAuthzAllow implements GatewayMetrics.
func (p *PrometheusGatewayMetrics) IncAuthzAllow(tenantID string) {
	p.authzAllowed.WithLabelValues(tenantID).Inc()
}

// IncAuthzDeny implements GatewayMetrics.
func (p *PrometheusGatewayMetrics) IncAuthzDeny(tenantID, reason string) {
	p.authzDenied.WithLabelValues(tenantID, reason).Inc()
}

// IncAuthzCacheHit implements GatewayMetrics.
func (p *PrometheusGatewayMetrics) IncAuthzCacheHit(tenantID string) {
	p.authzCacheHits.WithLabelValues(tenantID).Inc()
}

// IncAuthzCacheMiss implements GatewayMetrics.
func (p *PrometheusGatewayMetrics) IncAuthzCacheMiss(tenantID string) {
	p.authzCacheMiss.WithLabelValues(tenantID).Inc()
}

// RecordAuthzDuration records the duration of an authorization request.
func (p *PrometheusGatewayMetrics) RecordAuthzDuration(tenantID, result string, seconds float64) {
	p.authzDuration.WithLabelValues(tenantID, result).Observe(seconds)
}

// IncRateLimitHit records a rate limit hit.
func (p *PrometheusGatewayMetrics) IncRateLimitHit(tenantID string) {
	p.rateLimitHits.WithLabelValues(tenantID).Inc()
}

// RecordDelegationDepth records the depth of a delegation chain.
func (p *PrometheusGatewayMetrics) RecordDelegationDepth(tenantID string, depth int) {
	p.delegationDepth.WithLabelValues(tenantID).Observe(float64(depth))
}

// PrometheusVerifierMetrics implements VerifierMetrics with Prometheus collectors.
type PrometheusVerifierMetrics struct {
	verificationSuccess *prometheus.CounterVec
	verificationFailure *prometheus.CounterVec
	didResolutionTime   *prometheus.HistogramVec
	credentialAge       *prometheus.HistogramVec
}

// NewPrometheusVerifierMetrics creates Prometheus-backed verifier metrics.
func NewPrometheusVerifierMetrics(namespace string) *PrometheusVerifierMetrics {
	return NewPrometheusVerifierMetricsWithRegistry(namespace, prometheus.DefaultRegisterer)
}

// NewPrometheusVerifierMetricsWithRegistry creates Prometheus-backed verifier metrics with custom registry.
func NewPrometheusVerifierMetricsWithRegistry(namespace string, registerer prometheus.Registerer) *PrometheusVerifierMetrics {
	if namespace == "" {
		namespace = "credential_service"
	}

	factory := promauto.With(registerer)

	return &PrometheusVerifierMetrics{
		verificationSuccess: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "verifier_verification_success_total",
				Help:      "Total successful credential verifications by reason",
			},
			[]string{"reason"},
		),
		verificationFailure: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "verifier_verification_failure_total",
				Help:      "Total failed credential verifications by reason",
			},
			[]string{"reason"},
		),
		didResolutionTime: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "verifier_did_resolution_duration_seconds",
				Help:      "Time taken to resolve DIDs by method",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"did_method"},
		),
		credentialAge: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "verifier_credential_age_seconds",
				Help:      "Age of verified credentials in seconds",
				Buckets:   []float64{60, 300, 900, 1800, 3600, 7200, 14400, 28800, 86400},
			},
			[]string{"tenant_id"},
		),
	}
}

// IncVerificationSuccess implements VerifierMetrics.
func (p *PrometheusVerifierMetrics) IncVerificationSuccess(reason string) {
	p.verificationSuccess.WithLabelValues(reason).Inc()
}

// IncVerificationFailure implements VerifierMetrics.
func (p *PrometheusVerifierMetrics) IncVerificationFailure(reason string) {
	p.verificationFailure.WithLabelValues(reason).Inc()
}

// RecordDIDResolution records the time taken to resolve a DID.
func (p *PrometheusVerifierMetrics) RecordDIDResolution(method string, seconds float64) {
	p.didResolutionTime.WithLabelValues(method).Observe(seconds)
}

// RecordCredentialAge records the age of a credential being verified.
func (p *PrometheusVerifierMetrics) RecordCredentialAge(tenantID string, ageSeconds float64) {
	p.credentialAge.WithLabelValues(tenantID).Observe(ageSeconds)
}