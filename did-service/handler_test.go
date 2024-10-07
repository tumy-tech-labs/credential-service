package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateDID(t *testing.T) {
	// Example: Test the createDID handler
	// Log the state of the database connection
	// Check if the database is closed before proceeding
	if dbClosed {
		t.Fatal("Database connection pool is closed")
	}

	requestBody := `{"type": "organization", "organization_id": "org123"}`
	req, _ := http.NewRequest("POST", "/v1/dids", bytes.NewBuffer([]byte(requestBody)))
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(createDID)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", rr.Code)
	}
}
