package metrics

import (
	"os"
	"testing"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory(PrometheusMetrics, "test_namespace")

	if factory.metricsType != PrometheusMetrics {
		t.Errorf("Expected metrics type %s, got %s", PrometheusMetrics, factory.metricsType)
	}

	if factory.namespace != "test_namespace" {
		t.Errorf("Expected namespace test_namespace, got %s", factory.namespace)
	}
}

func TestNewFactoryFromEnv(t *testing.T) {
	tests := []struct {
		name              string
		metricsTypeEnv    string
		namespaceEnv      string
		expectedType      MetricsType
		expectedNamespace string
	}{
		{
			name:              "default values",
			metricsTypeEnv:    "",
			namespaceEnv:      "",
			expectedType:      NoopMetrics,
			expectedNamespace: "credential_service",
		},
		{
			name:              "prometheus with custom namespace",
			metricsTypeEnv:    "prometheus",
			namespaceEnv:      "custom_service",
			expectedType:      PrometheusMetrics,
			expectedNamespace: "custom_service",
		},
		{
			name:              "noop with custom namespace",
			metricsTypeEnv:    "noop",
			namespaceEnv:      "test_service",
			expectedType:      NoopMetrics,
			expectedNamespace: "test_service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			if tt.metricsTypeEnv != "" {
				os.Setenv("METRICS_TYPE", tt.metricsTypeEnv)
			} else {
				os.Unsetenv("METRICS_TYPE")
			}

			if tt.namespaceEnv != "" {
				os.Setenv("METRICS_NAMESPACE", tt.namespaceEnv)
			} else {
				os.Unsetenv("METRICS_NAMESPACE")
			}

			// Clean up after test
			defer func() {
				os.Unsetenv("METRICS_TYPE")
				os.Unsetenv("METRICS_NAMESPACE")
			}()

			factory := NewFactoryFromEnv()

			if factory.metricsType != tt.expectedType {
				t.Errorf("Expected metrics type %s, got %s", tt.expectedType, factory.metricsType)
			}

			if factory.namespace != tt.expectedNamespace {
				t.Errorf("Expected namespace %s, got %s", tt.expectedNamespace, factory.namespace)
			}
		})
	}
}

func TestFactoryCreateGatewayMetrics(t *testing.T) {
	tests := []struct {
		name        string
		metricsType MetricsType
		expectError bool
	}{
		{
			name:        "noop metrics",
			metricsType: NoopMetrics,
			expectError: false,
		},
		{
			name:        "prometheus metrics",
			metricsType: PrometheusMetrics,
			expectError: false,
		},
		{
			name:        "invalid metrics type",
			metricsType: MetricsType("invalid"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewFactory(tt.metricsType, "test")
			metrics, err := factory.CreateGatewayMetrics()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if metrics == nil {
				t.Error("Expected non-nil metrics")
				return
			}

			// Test that the metrics can be used
			metrics.IncAuthzRequest("test_tenant")

			// Type assertions to verify correct implementation
			switch tt.metricsType {
			case NoopMetrics:
				if _, ok := metrics.(*NoopGatewayMetrics); !ok {
					t.Error("Expected NoopGatewayMetrics")
				}
			case PrometheusMetrics:
				if _, ok := metrics.(*PrometheusGatewayMetrics); !ok {
					t.Error("Expected PrometheusGatewayMetrics")
				}
			}
		})
	}
}

func TestFactoryCreateVerifierMetrics(t *testing.T) {
	tests := []struct {
		name        string
		metricsType MetricsType
		expectError bool
	}{
		{
			name:        "noop metrics",
			metricsType: NoopMetrics,
			expectError: false,
		},
		{
			name:        "prometheus metrics",
			metricsType: PrometheusMetrics,
			expectError: false,
		},
		{
			name:        "invalid metrics type",
			metricsType: MetricsType("invalid"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewFactory(tt.metricsType, "test")
			metrics, err := factory.CreateVerifierMetrics()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if metrics == nil {
				t.Error("Expected non-nil metrics")
				return
			}

			// Test that the metrics can be used
			metrics.IncVerificationSuccess("test_reason")

			// Type assertions to verify correct implementation
			switch tt.metricsType {
			case NoopMetrics:
				if _, ok := metrics.(*NoopVerifierMetrics); !ok {
					t.Error("Expected NoopVerifierMetrics")
				}
			case PrometheusMetrics:
				if _, ok := metrics.(*PrometheusVerifierMetrics); !ok {
					t.Error("Expected PrometheusVerifierMetrics")
				}
			}
		})
	}
}

func TestFactoryGetters(t *testing.T) {
	factory := NewFactory(PrometheusMetrics, "test_namespace")

	if factory.GetMetricsType() != PrometheusMetrics {
		t.Errorf("Expected metrics type %s, got %s", PrometheusMetrics, factory.GetMetricsType())
	}

	if factory.GetNamespace() != "test_namespace" {
		t.Errorf("Expected namespace test_namespace, got %s", factory.GetNamespace())
	}
}