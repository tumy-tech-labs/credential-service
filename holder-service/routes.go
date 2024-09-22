package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// LoggingMiddleware logs incoming requests
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Received request: %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("Handled request: %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

/*
func InitializeRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/holder/receive", LoggingMiddleware(http.HandlerFunc(ReceiveCredential)).Methods("POST")))
	mux.Handle("/holder/present", LoggingMiddleware(http.HandlerFunc(PresentCredential).Methods("GET")))
	return mux
}
*/

// InitializeRoutes sets up the HTTP routes for the holder service.

func InitializeRoutes() *mux.Router {
	//mux := http.NewServeMux()
	r := mux.NewRouter()

	r.HandleFunc("/holder/receive", ReceiveCredential).Methods("POST")
	r.HandleFunc("/holder/present", PresentCredential).Methods("GET")

	return r
}

func ReceiveCredential(w http.ResponseWriter, r *http.Request) {
	// Handle receiving a verifiable credential
	var vc VerifiableCredential
	err := json.NewDecoder(r.Body).Decode(&vc)
	if err != nil {
		http.Error(w, "Invalid credential format", http.StatusBadRequest)
		return
	}

	// Store the credential in memory (for now)
	StoreCredential(vc)

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Credential received successfully"))
}

func PresentCredential(w http.ResponseWriter, r *http.Request) {
	// Logic to present stored credentials
	credentials := GetStoredCredentials()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(credentials)
}
