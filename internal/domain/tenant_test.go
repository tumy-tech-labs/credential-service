package domain

import "testing"

func TestTenantContext(t *testing.T) {
	ctx := TenantContext{TenantID: "tenant-123", RequestID: "req-abc"}

	if ctx.TenantID != "tenant-123" {
		t.Fatalf("expected tenant ID to be preserved")
	}

	if ctx.RequestID != "req-abc" {
		t.Fatalf("expected request ID to be preserved")
	}
}
