package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

// InitializeRoutes sets up the routes for the verifier service
func InitializeRoutes() *mux.Router {
	router := mux.NewRouter()

	// Route to verify credentials
	router.HandleFunc("/verifier/verify", VerifyCredentialHandler).Methods("POST")

	// Health check route (optional)
	router.HandleFunc("/verifier/health", HealthCheckHandler).Methods("GET")

	return router
}

// HealthCheckHandler simply returns a success status
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Verifier service is running"))
}
