package httpserver

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/bradtumy/credential-service/internal/logging"
)

type contextKey string

const (
	requestIDKey contextKey = "requestID"
	tenantIDKey  contextKey = "tenantID"
)

// CorrelationIDMiddleware adds correlation IDs to requests for tracing.
func CorrelationIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := r.Header.Get("X-Correlation-ID")
		if correlationID == "" {
			// Fallback to X-Request-ID for backward compatibility
			correlationID = r.Header.Get("X-Request-ID")
			if correlationID == "" {
				correlationID = logging.GenerateCorrelationID()
			}
		}

		// Add correlation ID to response headers
		w.Header().Set("X-Correlation-ID", correlationID)
		w.Header().Set("X-Request-ID", correlationID) // Backward compatibility

		// Add correlation ID to request context using new logging package
		ctx := logging.WithCorrelationID(r.Context(), correlationID)

		// Extract and add tenant ID if present
		tenantID := extractTenantID(r)
		if tenantID != "" {
			ctx = logging.WithTenantID(ctx, tenantID)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestContext is deprecated, use CorrelationIDMiddleware instead.
// Kept for backward compatibility.
func RequestContext(next http.Handler) http.Handler {
	return CorrelationIDMiddleware(next)
}

// RequestIDFromContext extracts the request ID from context.
// Deprecated: Use logging.GetCorrelationID instead.
func RequestIDFromContext(ctx context.Context) string {
	// First try new context key
	if id := logging.GetCorrelationID(ctx); id != "" {
		return id
	}
	// Fallback to old context key for backward compatibility
	if v := ctx.Value(requestIDKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

// TenantIDFromContext extracts the tenant ID from context.
// Deprecated: Use logging.GetTenantID instead.
func TenantIDFromContext(ctx context.Context) string {
	// First try new context key
	if id := logging.GetTenantID(ctx); id != "" {
		return id
	}
	// Fallback to old context key for backward compatibility
	if v := ctx.Value(tenantIDKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

// responseWriter wraps http.ResponseWriter to capture status code and bytes written.
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(data)
	rw.bytesWritten += int64(n)
	return n, err
}

// statusRecorder is deprecated, use responseWriter instead.
// Kept for backward compatibility.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// AuditMiddleware logs HTTP requests for security and compliance.
func AuditMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap ResponseWriter to capture status code and bytes
		wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Extract client information
		ipAddress := getClientIP(r)
		userAgent := r.UserAgent()

		// Log request start
		ctx := r.Context()
		logger := logging.ContextLogger(ctx)
		logger.InfoContext(ctx, "http_request_start",
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"ip_address", ipAddress,
			"user_agent", userAgent,
		)

		// Process request
		next.ServeHTTP(wrappedWriter, r)

		// Calculate duration
		duration := time.Since(start)

		// Determine outcome
		outcome := "success"
		if wrappedWriter.statusCode >= 400 {
			outcome = "failure"
		}

		// Create audit event for sensitive endpoints
		if isSensitiveEndpoint(r.URL.Path) {
			logging.LogAuditEvent(ctx, logging.AuditEvent{
				EventType: logging.AuditEventDataAccess,
				Resource:  r.URL.Path,
				Action:    r.Method,
				Outcome:   outcome,
				IPAddress: ipAddress,
				UserAgent: userAgent,
				Metadata: map[string]interface{}{
					"status_code":    wrappedWriter.statusCode,
					"duration_ms":    duration.Milliseconds(),
					"content_length": wrappedWriter.bytesWritten,
					"query_params":   r.URL.RawQuery,
				},
			})
		}

		// Log request completion
		logger.InfoContext(ctx, "http_request_complete",
			"method", r.Method,
			"path", r.URL.Path,
			"status_code", wrappedWriter.statusCode,
			"duration_ms", duration.Milliseconds(),
			"content_length", wrappedWriter.bytesWritten,
			"outcome", outcome,
		)
	})
}

// LoggingMiddleware is deprecated, use AuditMiddleware instead.
// Kept for backward compatibility.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()

		next.ServeHTTP(recorder, r)

		duration := time.Since(start)
		logging.Logger.Info("http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"duration_ms", duration.Milliseconds(),
			"request_id", RequestIDFromContext(r.Context()),
			"tenant_id", TenantIDFromContext(r.Context()),
		)
	})
}

// SecurityHeadersMiddleware adds security headers to responses.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next.ServeHTTP(w, r)
	})
}

// RateLimitAuditMiddleware logs rate limit events for security monitoring.
func RateLimitAuditMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrappedWriter, r)

		// Log rate limit exceeded events
		if wrappedWriter.statusCode == http.StatusTooManyRequests {
			ctx := r.Context()
			logging.LogSecurityEvent(ctx,
				logging.AuditEventRateLimitExceeded,
				"blocked",
				"Rate limit exceeded for client",
				map[string]interface{}{
					"path":       r.URL.Path,
					"method":     r.Method,
					"ip_address": getClientIP(r),
					"user_agent": r.UserAgent(),
				},
			)
		}
	})
}

// getClientIP extracts the real client IP address from various headers.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (most common)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		if ips := strings.Split(xff, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	if ip := r.RemoteAddr; ip != "" {
		// Remove port if present
		if colonPos := strings.LastIndex(ip, ":"); colonPos != -1 {
			return ip[:colonPos]
		}
		return ip
	}

	return "unknown"
}

// extractTenantID extracts tenant ID from request headers or path.
func extractTenantID(r *http.Request) string {
	// Check X-Tenant-ID header
	if tenantID := r.Header.Get("X-Tenant-ID"); tenantID != "" {
		return tenantID
	}

	// Check for tenant in URL path patterns like /tenants/{id}/...
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	for i, part := range pathParts {
		if part == "tenants" && i+1 < len(pathParts) {
			return pathParts[i+1]
		}
	}

	return ""
}

// isSensitiveEndpoint determines if an endpoint should be audited.
func isSensitiveEndpoint(path string) bool {
	sensitivePatterns := []string{
		"/credentials/",
		"/verify",
		"/issue",
		"/revoke",
		"/delegate",
		"/policies/",
		"/trust-registry/",
		"/admin/",
		"/keys/",
		"/presentations/",
		"/schemas/",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

// MiddlewareChain combines multiple middleware functions.
func MiddlewareChain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			handler = middlewares[i](handler)
		}
		return handler
	}
}

// StandardMiddlewareChain provides a standard set of middleware for services.
func StandardMiddlewareChain() func(http.Handler) http.Handler {
	return MiddlewareChain(
		SecurityHeadersMiddleware,
		CorrelationIDMiddleware,
		AuditMiddleware,
		RateLimitAuditMiddleware,
	)
}
