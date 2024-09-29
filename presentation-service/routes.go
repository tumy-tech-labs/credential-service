package main

import (
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

// InitializeRoutes sets up the HTTP routes
func InitializeRoutes() *mux.Router {

	r := mux.NewRouter()
	r.HandleFunc("/presentation", handlePresentationRequest).Methods("POST")

	return r
}
