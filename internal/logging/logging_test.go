package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestInitSetsLogger(t *testing.T) {
	Init("debug")
	if Logger == nil {
		t.Fatalf("expected logger to be initialized")
	}
	if AuditLogger == nil {
		t.Fatalf("expected audit logger to be initialized")
	}
}

func TestParseLevel(t *testing.T) {
	if lvl := parseLevel("debug"); lvl != -4 {
		t.Fatalf("expected debug level, got %v", lvl)
	}
	if lvl := parseLevel("warn"); lvl != 4 {
		t.Fatalf("expected warn level, got %v", lvl)
	}
	if lvl := parseLevel("error"); lvl != 8 {
		t.Fatalf("expected error level, got %v", lvl)
	}
	if lvl := parseLevel("unknown"); lvl != 0 {
		t.Fatalf("expected info level fallback, got %v", lvl)
	}
}

func TestGenerateCorrelationID(t *testing.T) {
	id1 := GenerateCorrelationID()
	id2 := GenerateCorrelationID()

	if id1 == "" {
		t.Error("Expected non-empty correlation ID")
	}

	if id1 == id2 {
		t.Error("Expected different correlation IDs")
	}

	// Should be 32 characters (16 bytes in hex)
	if len(id1) != 32 {
		t.Errorf("Expected correlation ID length 32, got %d", len(id1))
	}
}

func TestContextValues(t *testing.T) {
	ctx := context.Background()
	
	correlationID := "test-correlation-id"
	tenantID := "test-tenant"
	userID := "test-user"

	// Test correlation ID
	ctx = WithCorrelationID(ctx, correlationID)
	if got := GetCorrelationID(ctx); got != correlationID {
		t.Errorf("Expected correlation ID %s, got %s", correlationID, got)
	}

	// Test tenant ID
	ctx = WithTenantID(ctx, tenantID)
	if got := GetTenantID(ctx); got != tenantID {
		t.Errorf("Expected tenant ID %s, got %s", tenantID, got)
	}

	// Test user ID
	ctx = WithUserID(ctx, userID)
	if got := GetUserID(ctx); got != userID {
		t.Errorf("Expected user ID %s, got %s", userID, got)
	}

	// Test empty context
	emptyCtx := context.Background()
	if got := GetCorrelationID(emptyCtx); got != "" {
		t.Errorf("Expected empty correlation ID, got %s", got)
	}
}

func TestContextLogger(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	Logger = slog.New(handler)

	ctx := context.Background()
	ctx = WithCorrelationID(ctx, "test-correlation")
	ctx = WithTenantID(ctx, "test-tenant")
	ctx = WithUserID(ctx, "test-user")

	logger := ContextLogger(ctx)
	logger.InfoContext(ctx, "test message")

	output := buf.String()
	if !strings.Contains(output, "test-correlation") {
		t.Error("Expected correlation ID in log output")
	}
	if !strings.Contains(output, "test-tenant") {
		t.Error("Expected tenant ID in log output")
	}
	if !strings.Contains(output, "test-user") {
		t.Error("Expected user ID in log output")
	}
}

func TestLogAuditEvent(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	AuditLogger = slog.New(handler)

	ctx := context.Background()
	ctx = WithCorrelationID(ctx, "audit-correlation")
	ctx = WithTenantID(ctx, "audit-tenant")

	event := AuditEvent{
		EventType:  AuditEventCredentialIssued,
		SubjectDID: "did:example:subject",
		IssuerDID:  "did:example:issuer",
		Outcome:    "success",
		Metadata: map[string]interface{}{
			"credential_type": "VerifiableCredential",
		},
	}

	LogAuditEvent(ctx, event)

	output := buf.String()
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output as JSON: %v", err)
	}

	// Verify audit event fields
	if logEntry["event_type"] != string(AuditEventCredentialIssued) {
		t.Errorf("Expected event_type %s, got %v", AuditEventCredentialIssued, logEntry["event_type"])
	}

	if logEntry["correlation_id"] != "audit-correlation" {
		t.Errorf("Expected correlation_id audit-correlation, got %v", logEntry["correlation_id"])
	}

	if logEntry["tenant_id"] != "audit-tenant" {
		t.Errorf("Expected tenant_id audit-tenant, got %v", logEntry["tenant_id"])
	}

	if logEntry["subject_did"] != "did:example:subject" {
		t.Errorf("Expected subject_did did:example:subject, got %v", logEntry["subject_did"])
	}

	if logEntry["outcome"] != "success" {
		t.Errorf("Expected outcome success, got %v", logEntry["outcome"])
	}

	// Verify timestamp is present and valid
	if _, ok := logEntry["timestamp"]; !ok {
		t.Error("Expected timestamp field in audit log")
	}
}

func TestConvenienceLogFunctions(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	AuditLogger = slog.New(handler)

	ctx := WithCorrelationID(context.Background(), "test-correlation")

	// Test LogSecurityEvent
	buf.Reset()
	LogSecurityEvent(ctx, AuditEventRateLimitExceeded, "blocked", "Too many requests", map[string]interface{}{
		"ip": "192.168.1.1",
	})

	output := buf.String()
	if !strings.Contains(output, string(AuditEventRateLimitExceeded)) {
		t.Error("Expected security event type in log output")
	}
	if !strings.Contains(output, "blocked") {
		t.Error("Expected outcome in log output")
	}

	// Test LogCredentialEvent
	buf.Reset()
	LogCredentialEvent(ctx, AuditEventCredentialVerified, "did:example:subject", "did:example:issuer", "success")

	output = buf.String()
	if !strings.Contains(output, string(AuditEventCredentialVerified)) {
		t.Error("Expected credential event type in log output")
	}
	if !strings.Contains(output, "did:example:subject") {
		t.Error("Expected subject DID in log output")
	}

	// Test LogAdminEvent
	buf.Reset()
	LogAdminEvent(ctx, AuditEventPolicyCreated, "/policies/123", "CREATE", "success", map[string]interface{}{
		"policy_type": "access_control",
	})

	output = buf.String()
	if !strings.Contains(output, string(AuditEventPolicyCreated)) {
		t.Error("Expected admin event type in log output")
	}
	if !strings.Contains(output, "/policies/123") {
		t.Error("Expected resource in log output")
	}
}

func TestAuditEventTypes(t *testing.T) {
	// Test that all event types are properly defined
	eventTypes := []AuditEventType{
		AuditEventLogin,
		AuditEventLogout,
		AuditEventAccessDenied,
		AuditEventPermissionGrant,
		AuditEventCredentialIssued,
		AuditEventCredentialVerified,
		AuditEventCredentialRevoked,
		AuditEventDelegationCreated,
		AuditEventTrustRegistryUpdate,
		AuditEventPolicyCreated,
		AuditEventPolicyUpdated,
		AuditEventPolicyDeleted,
		AuditEventKeyRotation,
		AuditEventConfigChange,
		AuditEventSecurityIncident,
		AuditEventRateLimitExceeded,
		AuditEventSuspiciousActivity,
		AuditEventDataAccess,
	}

	for _, eventType := range eventTypes {
		if string(eventType) == "" {
			t.Errorf("Event type %v should not be empty", eventType)
		}
	}
}

func BenchmarkGenerateCorrelationID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateCorrelationID()
	}
}
