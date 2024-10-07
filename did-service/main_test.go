package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestMain(m *testing.M) {

	// Set environment variables
	os.Setenv("DATABASE_URL", "postgres://cred-service:cred-service-1@postgres:5432/credential-service")

	// Initialize the database connection
	var err error
	db, err = initDB() // Ensure you return the db from initDB
	if err != nil {
		os.Exit(1) // Exit if the DB connection fails
	}
	// Run tests
	code := m.Run()

	// Cleanup
	closeDB()
	os.Exit(code)
}

// TestServerStart checks if the server responds correctly to the root endpoint.
func TestServerStart(t *testing.T) {
	// Create a new HTTP request to the root endpoint
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err) // Fail if creating the request fails
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()
	routes := InitializeRoutes() // Initialize your routes

	// Serve the HTTP request using your router
	routes.ServeHTTP(rr, req)

	// Check if the response code is as expected
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", rr.Code)
	}

	// Optionally, check the response body or headers if needed
	// expectedBody := `...`
	// if rr.Body.String() != expectedBody {
	//     t.Errorf("Expected response body %v, got %v", expectedBody, rr.Body.String())
	// }
}
