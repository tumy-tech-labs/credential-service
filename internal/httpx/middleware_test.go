package httpx

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bradtumy/credential-service/internal/logging"
)

func TestRequestContextAddsIDs(t *testing.T) {
	handler := RequestContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id := RequestIDFromContext(r.Context()); id == "" {
			t.Fatalf("expected request ID to be set")
		}
		if tenant := TenantIDFromContext(r.Context()); tenant != "tenant-1" {
			t.Fatalf("expected tenant ID to be propagated")
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", "tenant-1")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
}

func TestRequestContextUsesProvidedRequestID(t *testing.T) {
	const expected = "req-custom"
	handler := RequestContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id := RequestIDFromContext(r.Context()); id != expected {
			t.Fatalf("expected %s, got %s", expected, id)
		}
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", expected)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
}

func TestLoggingMiddlewareInvokesNext(t *testing.T) {
	called := false
	logging.Logger = slog.New(slog.NewJSONHandler(io.Discard, nil))

	handler := LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	}))

	req := httptest.NewRequest(http.MethodGet, "/log", nil)
	req = req.WithContext(context.WithValue(req.Context(), requestIDKey, "req-1"))
	req = req.WithContext(context.WithValue(req.Context(), tenantIDKey, "tenant-1"))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatalf("expected handler to be called")
	}
	if rr.Code != http.StatusTeapot {
		t.Fatalf("expected status to propagate, got %d", rr.Code)
	}
}

func TestCorrelationIDMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := logging.GetCorrelationID(r.Context())
		if correlationID == "" {
			t.Error("Expected correlation ID in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := CorrelationIDMiddleware(handler)

	tests := []struct {
		name           string
		headerValue    string
		expectGenerated bool
	}{
		{
			name:           "with correlation ID header",
			headerValue:    "existing-correlation-id",
			expectGenerated: false,
		},
		{
			name:           "without correlation ID header",
			headerValue:    "",
			expectGenerated: true,
		},
		{
			name:           "with legacy X-Request-ID header",
			headerValue:    "legacy-request-id",
			expectGenerated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.headerValue != "" {
				if tt.name == "with legacy X-Request-ID header" {
					req.Header.Set("X-Request-ID", tt.headerValue)
				} else {
					req.Header.Set("X-Correlation-ID", tt.headerValue)
				}
			}

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			// Check response header
			responseCorrelationID := rr.Header().Get("X-Correlation-ID")
			if responseCorrelationID == "" {
				t.Error("Expected correlation ID in response header")
			}

			// Check backward compatibility
			legacyID := rr.Header().Get("X-Request-ID")
			if legacyID == "" {
				t.Error("Expected X-Request-ID for backward compatibility")
			}

			if !tt.expectGenerated && responseCorrelationID != tt.headerValue {
				t.Errorf("Expected correlation ID %s, got %s", tt.headerValue, responseCorrelationID)
			}

			if tt.expectGenerated && len(responseCorrelationID) != 32 {
				t.Errorf("Expected generated correlation ID length 32, got %d", len(responseCorrelationID))
			}
		})
	}
}

func TestAuditMiddleware(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logging.Logger = slog.New(handler)
	logging.AuditLogger = slog.New(handler)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	middleware := CorrelationIDMiddleware(AuditMiddleware(testHandler))

	tests := []struct {
		name           string
		path           string
		method         string
		expectedAudit  bool
	}{
		{
			name:          "sensitive endpoint - credentials",
			path:          "/credentials/issue",
			method:        "POST",
			expectedAudit: true,
		},
		{
			name:          "sensitive endpoint - verify",
			path:          "/verify",
			method:        "POST",
			expectedAudit: true,
		},
		{
			name:          "non-sensitive endpoint",
			path:          "/health",
			method:        "GET",
			expectedAudit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()

			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("User-Agent", "test-agent")
			req.RemoteAddr = "192.168.1.1:12345"

			rr := httptest.NewRecorder()
			middleware.ServeHTTP(rr, req)

			output := buf.String()
			lines := strings.Split(strings.TrimSpace(output), "\n")

			// Should have at least request start and complete logs
			if len(lines) < 2 {
				t.Errorf("Expected at least 2 log lines, got %d", len(lines))
			}

			// Check for audit event if expected
			auditFound := false
			for _, line := range lines {
				if strings.Contains(line, "audit_event") {
					auditFound = true
					break
				}
			}

			if tt.expectedAudit && !auditFound {
				t.Error("Expected audit event for sensitive endpoint")
			}

			if !tt.expectedAudit && auditFound {
				t.Error("Unexpected audit event for non-sensitive endpoint")
			}
		})
	}
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := SecurityHeadersMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	expectedHeaders := map[string]string{
		"X-Content-Type-Options":           "nosniff",
		"X-Frame-Options":                  "DENY",
		"X-XSS-Protection":                 "1; mode=block",
		"Strict-Transport-Security":        "max-age=31536000; includeSubDomains",
		"Content-Security-Policy":          "default-src 'self'",
		"Referrer-Policy":                  "strict-origin-when-cross-origin",
	}

	for header, expectedValue := range expectedHeaders {
		if got := rr.Header().Get(header); got != expectedValue {
			t.Errorf("Expected header %s: %s, got: %s", header, expectedValue, got)
		}
	}
}

func TestRateLimitAuditMiddleware(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logging.AuditLogger = slog.New(handler)

	tests := []struct {
		name           string
		statusCode     int
		expectAudit    bool
	}{
		{
			name:        "rate limit exceeded",
			statusCode:  http.StatusTooManyRequests,
			expectAudit: true,
		},
		{
			name:        "normal request",
			statusCode:  http.StatusOK,
			expectAudit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			})

			middleware := CorrelationIDMiddleware(RateLimitAuditMiddleware(handler))

			req := httptest.NewRequest("GET", "/test", nil)
			rr := httptest.NewRecorder()

			middleware.ServeHTTP(rr, req)

			output := buf.String()
			auditFound := strings.Contains(output, string(logging.AuditEventRateLimitExceeded))

			if tt.expectAudit && !auditFound {
				t.Error("Expected rate limit audit event")
			}

			if !tt.expectAudit && auditFound {
				t.Error("Unexpected rate limit audit event")
			}
		})
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name        string
		remoteAddr  string
		headers     map[string]string
		expectedIP  string
	}{
		{
			name:       "X-Forwarded-For header",
			remoteAddr: "127.0.0.1:12345",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1, 192.168.1.1",
			},
			expectedIP: "203.0.113.1",
		},
		{
			name:       "X-Real-IP header",
			remoteAddr: "127.0.0.1:12345",
			headers: map[string]string{
				"X-Real-IP": "203.0.113.2",
			},
			expectedIP: "203.0.113.2",
		},
		{
			name:       "RemoteAddr fallback",
			remoteAddr: "192.168.1.100:54321",
			headers:    map[string]string{},
			expectedIP: "192.168.1.100",
		},
		{
			name:       "IPv6 RemoteAddr",
			remoteAddr: "[::1]:12345",
			headers:    map[string]string{},
			expectedIP: "[::1]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr

			for header, value := range tt.headers {
				req.Header.Set(header, value)
			}

			ip := getClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}

func TestExtractTenantID(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		headers    map[string]string
		expectedID string
	}{
		{
			name: "X-Tenant-ID header",
			path: "/test",
			headers: map[string]string{
				"X-Tenant-ID": "tenant-123",
			},
			expectedID: "tenant-123",
		},
		{
			name:       "URL path with tenant",
			path:       "/tenants/tenant-456/credentials",
			headers:    map[string]string{},
			expectedID: "tenant-456",
		},
		{
			name:       "no tenant ID",
			path:       "/health",
			headers:    map[string]string{},
			expectedID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)

			for header, value := range tt.headers {
				req.Header.Set(header, value)
			}

			tenantID := extractTenantID(req)
			if tenantID != tt.expectedID {
				t.Errorf("Expected tenant ID %s, got %s", tt.expectedID, tenantID)
			}
		})
	}
}

func TestIsSensitiveEndpoint(t *testing.T) {
	tests := []struct {
		path      string
		sensitive bool
	}{
		{"/credentials/issue", true},
		{"/verify", true},
		{"/revoke", true},
		{"/policies/create", true},
		{"/admin/users", true},
		{"/keys/rotate", true},
		{"/presentations/verify", true},
		{"/schemas/create", true},
		{"/health", false},
		{"/metrics", false},
		{"/version", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isSensitiveEndpoint(tt.path)
			if result != tt.sensitive {
				t.Errorf("Expected %s sensitive=%t, got %t", tt.path, tt.sensitive, result)
			}
		})
	}
}

func TestStandardMiddlewareChain(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that correlation ID was added to context
		if correlationID := logging.GetCorrelationID(r.Context()); correlationID == "" {
			t.Error("Expected correlation ID in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := StandardMiddlewareChain()(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	middleware.ServeHTTP(rr, req)

	// Check that security headers are present
	if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Error("Expected security headers to be set")
	}

	// Check that correlation ID is in response
	if got := rr.Header().Get("X-Correlation-ID"); got == "" {
		t.Error("Expected correlation ID in response header")
	}
}
