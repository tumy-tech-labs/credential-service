package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bradtumy/credential-service/internal/domain"
	"github.com/bradtumy/credential-service/internal/tenant"
)

func TestTenantMiddlewareMissingHeaderMulti(t *testing.T) {
	store := tenant.NewMemoryStore()
	_ = store.UpsertTenant(context.Background(), domain.Tenant{ID: "tenant-a", Enabled: true})

	resolver := tenant.Resolver{Mode: tenant.ModeMulti, DefaultTenantID: "tenant-a", Store: store}

	handler := TenantMiddleware(resolver, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestTenantMiddlewareDisabledTenant(t *testing.T) {
	store := tenant.NewMemoryStore()
	_ = store.UpsertTenant(context.Background(), domain.Tenant{ID: "tenant-b", Enabled: false})

	resolver := tenant.Resolver{Mode: tenant.ModeMulti, DefaultTenantID: "tenant-b", Store: store}

	handler := TenantMiddleware(resolver, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant-ID", "tenant-b")

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rr.Code)
	}
}

func TestTenantMiddlewareSingleModeDefault(t *testing.T) {
	store := tenant.NewMemoryStore()
	_ = store.UpsertTenant(context.Background(), domain.Tenant{ID: "default", Enabled: true})

	resolver := tenant.Resolver{Mode: tenant.ModeSingle, DefaultTenantID: "default", Store: store}

	handler := TenantMiddleware(resolver, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if TenantIDFromContext(r.Context()) != "default" {
			t.Fatalf("expected tenant to be injected")
		}
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
}
