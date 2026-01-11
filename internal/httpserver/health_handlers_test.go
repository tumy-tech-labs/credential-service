package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthRoutesHealthy(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHealthRoutes(mux, nil)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	mux.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	mux.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}

func TestReadyRoutesInvokeCheck(t *testing.T) {
	mux := http.NewServeMux()
	called := false
	RegisterHealthRoutes(mux, func(ctx context.Context) error {
		called = true
		return nil
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	mux.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	if !called {
		t.Fatalf("expected readiness check to be called")
	}
}

func TestReadyRouteWithError(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHealthRoutes(mux, func(ctx context.Context) error {
		return errors.New("db down")
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	mux.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", recorder.Code)
	}
}
