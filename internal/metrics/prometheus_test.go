package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestPrometheusGatewayMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewPrometheusGatewayMetricsWithRegistry("test", registry)

	// Test authorization request counting
	metrics.IncAuthzRequest("tenant1")
	metrics.IncAuthzRequest("tenant1")
	metrics.IncAuthzRequest("tenant2")

	// Test authorization allow/deny
	metrics.IncAuthzAllow("tenant1")
	metrics.IncAuthzDeny("tenant1", "invalid_token")
	metrics.IncAuthzDeny("tenant2", "expired")

	// Test cache metrics
	metrics.IncAuthzCacheHit("tenant1")
	metrics.IncAuthzCacheMiss("tenant1")

	// Test rate limiting
	metrics.IncRateLimitHit("tenant1")

	// Test duration recording
	metrics.RecordAuthzDuration("tenant1", "allowed", 0.05)
	metrics.RecordAuthzDuration("tenant1", "denied", 0.02)

	// Test delegation depth
	metrics.RecordDelegationDepth("tenant1", 2)
	metrics.RecordDelegationDepth("tenant2", 1)

	// Verify counter values
	if count := testutil.ToFloat64(metrics.authzRequests.WithLabelValues("tenant1")); count != 2 {
		t.Errorf("Expected 2 authz requests for tenant1, got %f", count)
	}

	if count := testutil.ToFloat64(metrics.authzRequests.WithLabelValues("tenant2")); count != 1 {
		t.Errorf("Expected 1 authz request for tenant2, got %f", count)
	}

	if count := testutil.ToFloat64(metrics.authzAllowed.WithLabelValues("tenant1")); count != 1 {
		t.Errorf("Expected 1 allowed for tenant1, got %f", count)
	}

	if count := testutil.ToFloat64(metrics.authzDenied.WithLabelValues("tenant1", "invalid_token")); count != 1 {
		t.Errorf("Expected 1 denied for tenant1/invalid_token, got %f", count)
	}

	if count := testutil.ToFloat64(metrics.authzCacheHits.WithLabelValues("tenant1")); count != 1 {
		t.Errorf("Expected 1 cache hit for tenant1, got %f", count)
	}

	if count := testutil.ToFloat64(metrics.rateLimitHits.WithLabelValues("tenant1")); count != 1 {
		t.Errorf("Expected 1 rate limit hit for tenant1, got %f", count)
	}
}

func TestPrometheusVerifierMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewPrometheusVerifierMetricsWithRegistry("test", registry)

	// Test verification success/failure
	metrics.IncVerificationSuccess("valid_signature")
	metrics.IncVerificationSuccess("valid_signature")
	metrics.IncVerificationFailure("invalid_signature")
	metrics.IncVerificationFailure("expired")

	// Test DID resolution timing
	metrics.RecordDIDResolution("did:key", 0.001)
	metrics.RecordDIDResolution("did:web", 0.150)

	// Test credential age
	metrics.RecordCredentialAge("tenant1", 300)  // 5 minutes
	metrics.RecordCredentialAge("tenant1", 3600) // 1 hour

	// Verify counter values
	if count := testutil.ToFloat64(metrics.verificationSuccess.WithLabelValues("valid_signature")); count != 2 {
		t.Errorf("Expected 2 verification successes for valid_signature, got %f", count)
	}

	if count := testutil.ToFloat64(metrics.verificationFailure.WithLabelValues("invalid_signature")); count != 1 {
		t.Errorf("Expected 1 verification failure for invalid_signature, got %f", count)
	}

	if count := testutil.ToFloat64(metrics.verificationFailure.WithLabelValues("expired")); count != 1 {
		t.Errorf("Expected 1 verification failure for expired, got %f", count)
	}
}

func TestPrometheusMetricsWithCustomNamespace(t *testing.T) {
	gatewayMetrics := NewPrometheusGatewayMetrics("custom_service")
	verifierMetrics := NewPrometheusVerifierMetrics("custom_service")

	// Test that metrics are created (won't panic)
	gatewayMetrics.IncAuthzRequest("tenant1")
	verifierMetrics.IncVerificationSuccess("test")

	// If we reach here without panicking, the metrics were created successfully
}

func TestPrometheusMetricsWithEmptyNamespace(t *testing.T) {
	gatewayMetrics := NewPrometheusGatewayMetrics("")
	verifierMetrics := NewPrometheusVerifierMetrics("")

	// Test that default namespace is used (won't panic)
	gatewayMetrics.IncAuthzRequest("tenant1")
	verifierMetrics.IncVerificationSuccess("test")

	// If we reach here without panicking, the default namespace was used
}

// BenchmarkPrometheusGatewayMetrics tests the performance of metric operations
func BenchmarkPrometheusGatewayMetrics(b *testing.B) {
	registry := prometheus.NewRegistry()
	metrics := NewPrometheusGatewayMetricsWithRegistry("bench", registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.IncAuthzRequest("tenant1")
		metrics.IncAuthzAllow("tenant1")
		metrics.RecordAuthzDuration("tenant1", "allowed", 0.05)
	}
}

// BenchmarkPrometheusVerifierMetrics tests the performance of verifier metrics
func BenchmarkPrometheusVerifierMetrics(b *testing.B) {
	registry := prometheus.NewRegistry()
	metrics := NewPrometheusVerifierMetricsWithRegistry("bench", registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.IncVerificationSuccess("valid")
		metrics.RecordDIDResolution("did:key", 0.001)
	}
}