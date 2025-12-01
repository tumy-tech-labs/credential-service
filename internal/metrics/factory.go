package metrics

import (
	"fmt"
	"os"
)

// MetricsType represents the type of metrics backend to use.
type MetricsType string

const (
	// NoopMetrics disables metrics collection.
	NoopMetrics MetricsType = "noop"
	// PrometheusMetrics enables Prometheus metrics collection.
	PrometheusMetrics MetricsType = "prometheus"
)

// Factory creates metrics instances based on configuration.
type Factory struct {
	metricsType MetricsType
	namespace   string
}

// NewFactory creates a new metrics factory with the specified configuration.
func NewFactory(metricsType MetricsType, namespace string) *Factory {
	return &Factory{
		metricsType: metricsType,
		namespace:   namespace,
	}
}

// NewFactoryFromEnv creates a metrics factory from environment variables.
// METRICS_TYPE: "noop" or "prometheus" (default: "noop")
// METRICS_NAMESPACE: custom namespace (default: "credential_service")
func NewFactoryFromEnv() *Factory {
	metricsType := MetricsType(os.Getenv("METRICS_TYPE"))
	if metricsType == "" {
		metricsType = NoopMetrics
	}

	namespace := os.Getenv("METRICS_NAMESPACE")
	if namespace == "" {
		namespace = "credential_service"
	}

	return NewFactory(metricsType, namespace)
}

// CreateGatewayMetrics creates a GatewayMetrics instance.
func (f *Factory) CreateGatewayMetrics() (GatewayMetrics, error) {
	switch f.metricsType {
	case NoopMetrics:
		return &NoopGatewayMetrics{}, nil
	case PrometheusMetrics:
		return NewPrometheusGatewayMetrics(f.namespace), nil
	default:
		return nil, fmt.Errorf("unsupported metrics type: %s", f.metricsType)
	}
}

// CreateVerifierMetrics creates a VerifierMetrics instance.
func (f *Factory) CreateVerifierMetrics() (VerifierMetrics, error) {
	switch f.metricsType {
	case NoopMetrics:
		return &NoopVerifierMetrics{}, nil
	case PrometheusMetrics:
		return NewPrometheusVerifierMetrics(f.namespace), nil
	default:
		return nil, fmt.Errorf("unsupported metrics type: %s", f.metricsType)
	}
}

// GetMetricsType returns the configured metrics type.
func (f *Factory) GetMetricsType() MetricsType {
	return f.metricsType
}

// GetNamespace returns the configured namespace.
func (f *Factory) GetNamespace() string {
	return f.namespace
}