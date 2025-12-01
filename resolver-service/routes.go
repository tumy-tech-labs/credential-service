package main

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

func initializeRoutes() *mux.Router {
	r := mux.NewRouter()

	// Basic health and readiness endpoints
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}).Methods("GET")
	r.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		// DB readiness check if pool initialized
		if db != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
			defer cancel()
			if err := db.Ping(ctx); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte("not ready"))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	}).Methods("GET")

	// Define the resolver route
	// Version 1 routes
	v1 := r.PathPrefix("/v1").Subrouter()
	v1.HandleFunc("/dids/resolver", resolveDIDHandler).Methods("GET")

	return r
}
