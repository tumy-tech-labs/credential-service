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

func InitializeRoutes() *mux.Router {

	r := mux.NewRouter()

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
