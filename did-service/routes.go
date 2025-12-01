package main

import (
	"context"
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

func InitializeRoutes() *mux.Router {

	r := mux.NewRouter()

	// Basic health endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}).Methods("GET")

	// Readiness with DB ping if available
	r.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if db != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
			defer cancel()
			// pgxpool doesn't have Ping directly; simple query to validate connectivity
			if err := db.Ping(ctx); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte("not ready"))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	}).Methods("GET")

	// Version 1 routes
	v1 := r.PathPrefix("/v1").Subrouter()
	v1.Handle("/dids", LoggingMiddleware(http.HandlerFunc(createDID))).Methods("POST")
	v1.Handle("/dids", LoggingMiddleware(http.HandlerFunc(getDIDs))).Methods("GET")

	// Version 2 routes
	// whent he time comes put the v2 routes here.
	// e.g.
	// v2 := r.PathPrefix("/v2").Subrouter()
	// v2.Handle("/dids", LoggingMiddleware(http.HandlerFunc(getDIDsV2))).Methods("GET")

	return r
}
