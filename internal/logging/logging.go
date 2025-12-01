package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
	"time"
)

// Logger is the global structured logger used across services.
var Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

// AuditLogger is specialized for security and compliance events.
// Initialize to a sane default to avoid nil dereferences in tests/services
// that don't explicitly call Init(). Init() will override this.
var AuditLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
	Level:    slog.LevelInfo,
	AddSource: true,
}))

// ContextKey represents keys for context values.
type ContextKey string

const (
	// CorrelationIDKey is used to store correlation IDs in context.
	CorrelationIDKey ContextKey = "correlation_id"
	// TenantIDKey is used to store tenant IDs in context.
	TenantIDKey ContextKey = "tenant_id"
	// UserIDKey is used to store user IDs in context.
	UserIDKey ContextKey = "user_id"
)

// AuditEventType represents different types of audit events.
type AuditEventType string

const (
	// Authentication and authorization events
	AuditEventLogin          AuditEventType = "auth.login"
	AuditEventLogout         AuditEventType = "auth.logout"
	AuditEventAccessDenied   AuditEventType = "auth.access_denied"
	AuditEventPermissionGrant AuditEventType = "auth.permission_grant"

	// Credential lifecycle events
	AuditEventCredentialIssued   AuditEventType = "credential.issued"
	AuditEventCredentialVerified AuditEventType = "credential.verified"
	AuditEventCredentialRevoked  AuditEventType = "credential.revoked"
	AuditEventDelegationCreated  AuditEventType = "credential.delegation_created"

	// Administrative events
	AuditEventTrustRegistryUpdate AuditEventType = "admin.trust_registry_update"
	AuditEventPolicyCreated       AuditEventType = "admin.policy_created"
	AuditEventPolicyUpdated       AuditEventType = "admin.policy_updated"
	AuditEventPolicyDeleted       AuditEventType = "admin.policy_deleted"
	AuditEventKeyRotation         AuditEventType = "admin.key_rotation"
	AuditEventConfigChange        AuditEventType = "admin.config_change"

	// Security events
	AuditEventSecurityIncident    AuditEventType = "security.incident"
	AuditEventRateLimitExceeded   AuditEventType = "security.rate_limit_exceeded"
	AuditEventSuspiciousActivity  AuditEventType = "security.suspicious_activity"
	AuditEventDataAccess          AuditEventType = "security.data_access"
)

// AuditEvent represents a structured audit log entry.
type AuditEvent struct {
	Timestamp     time.Time              `json:"timestamp"`
	CorrelationID string                 `json:"correlation_id"`
	EventType     AuditEventType         `json:"event_type"`
	TenantID      string                 `json:"tenant_id,omitempty"`
	UserID        string                 `json:"user_id,omitempty"`
	SubjectDID    string                 `json:"subject_did,omitempty"`
	IssuerDID     string                 `json:"issuer_did,omitempty"`
	Resource      string                 `json:"resource,omitempty"`
	Action        string                 `json:"action,omitempty"`
	Outcome       string                 `json:"outcome"` // success, failure, error
	Reason        string                 `json:"reason,omitempty"`
	IPAddress     string                 `json:"ip_address,omitempty"`
	UserAgent     string                 `json:"user_agent,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Init configures the global logger with the provided level.
func Init(level string) {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLevel(level)})
	Logger = slog.New(handler)

	// Initialize audit logger with structured format
	auditHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		AddSource: true,
	})
	AuditLogger = slog.New(auditHandler)
}

// GenerateCorrelationID creates a new correlation ID for request tracing.
func GenerateCorrelationID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// WithCorrelationID adds a correlation ID to the context.
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, correlationID)
}

// GetCorrelationID retrieves the correlation ID from context.
func GetCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return id
	}
	return ""
}

// WithTenantID adds a tenant ID to the context.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, TenantIDKey, tenantID)
}

// GetTenantID retrieves the tenant ID from context.
func GetTenantID(ctx context.Context) string {
	if id, ok := ctx.Value(TenantIDKey).(string); ok {
		return id
	}
	return ""
}

// WithUserID adds a user ID to the context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// GetUserID retrieves the user ID from context.
func GetUserID(ctx context.Context) string {
	if id, ok := ctx.Value(UserIDKey).(string); ok {
		return id
	}
	return ""
}

// ContextLogger returns a logger with context values pre-populated.
func ContextLogger(ctx context.Context) *slog.Logger {
	logger := Logger
	
	if correlationID := GetCorrelationID(ctx); correlationID != "" {
		logger = logger.With("correlation_id", correlationID)
	}
	
	if tenantID := GetTenantID(ctx); tenantID != "" {
		logger = logger.With("tenant_id", tenantID)
	}
	
	if userID := GetUserID(ctx); userID != "" {
		logger = logger.With("user_id", userID)
	}
	
	return logger
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// LogAuditEvent logs a structured audit event with full context.
func LogAuditEvent(ctx context.Context, event AuditEvent) {
	// Populate correlation ID from context if not set
	if event.CorrelationID == "" {
		event.CorrelationID = GetCorrelationID(ctx)
	}
	
	// Populate tenant ID from context if not set
	if event.TenantID == "" {
		event.TenantID = GetTenantID(ctx)
	}
	
	// Populate user ID from context if not set
	if event.UserID == "" {
		event.UserID = GetUserID(ctx)
	}
	
	// Set timestamp
	event.Timestamp = time.Now().UTC()
	
	// Use AuditLogger if set; otherwise fall back to slog.Default()
	logger := AuditLogger
	if logger == nil {
		logger = slog.Default()
	}
	logger.InfoContext(ctx, "audit_event",
		"timestamp", event.Timestamp.Format(time.RFC3339),
		"correlation_id", event.CorrelationID,
		"event_type", string(event.EventType),
		"tenant_id", event.TenantID,
		"user_id", event.UserID,
		"subject_did", event.SubjectDID,
		"issuer_did", event.IssuerDID,
		"resource", event.Resource,
		"action", event.Action,
		"outcome", event.Outcome,
		"reason", event.Reason,
		"ip_address", event.IPAddress,
		"user_agent", event.UserAgent,
		"metadata", event.Metadata,
	)
}

// LogSecurityEvent is a convenience function for logging security events.
func LogSecurityEvent(ctx context.Context, eventType AuditEventType, outcome, reason string, metadata map[string]interface{}) {
	LogAuditEvent(ctx, AuditEvent{
		EventType: eventType,
		Outcome:   outcome,
		Reason:    reason,
		Metadata:  metadata,
	})
}

// LogCredentialEvent is a convenience function for logging credential lifecycle events.
func LogCredentialEvent(ctx context.Context, eventType AuditEventType, subjectDID, issuerDID, outcome string) {
	LogAuditEvent(ctx, AuditEvent{
		EventType:  eventType,
		SubjectDID: subjectDID,
		IssuerDID:  issuerDID,
		Outcome:    outcome,
	})
}

// LogAdminEvent is a convenience function for logging administrative events.
func LogAdminEvent(ctx context.Context, eventType AuditEventType, resource, action, outcome string, metadata map[string]interface{}) {
	LogAuditEvent(ctx, AuditEvent{
		EventType: eventType,
		Resource:  resource,
		Action:    action,
		Outcome:   outcome,
		Metadata:  metadata,
	})
}
