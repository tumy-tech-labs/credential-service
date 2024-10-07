package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInitializeRoutes(t *testing.T) {
	router := InitializeRoutes()

	// Test a route, e.g., the /v1/dids endpoint
	req, _ := http.NewRequest("GET", "/v1/dids", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", rr.Code)
	}
}
