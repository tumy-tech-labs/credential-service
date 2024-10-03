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

// initializeRoutes sets up the application routes.
func initializeRoutes() *mux.Router {

	r := mux.NewRouter()

	// Version 1 routes
	v1 := r.PathPrefix("/v1").Subrouter()
	v1.Handle("/", LoggingMiddleware(http.HandlerFunc(helloWorld))).Methods("POST", "GET") // for debug
	v1.Handle("/anchor/did", LoggingMiddleware(http.HandlerFunc(AnchorDIDHandler))).Methods("POST", "GET")
	v1.Handle("/anchor/credential", LoggingMiddleware(http.HandlerFunc(AnchorCredentialHandler))).Methods("POST", "GET")

	return r
}

// for debug
func helloWorld(w http.ResponseWriter, r *http.Request) {
	log.Println("Hello, World")
	w.Write([]byte("Hello, World"))
}
