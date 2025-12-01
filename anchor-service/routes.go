package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// LoggingMiddleware provides basic request logging
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

// initializeRoutes sets up the application routes.
func initializeRoutes() *mux.Router {

	r := mux.NewRouter()

	// Version 1 routes
	v1 := r.PathPrefix("/v1").Subrouter()
	v1.Handle("/", LoggingMiddleware(http.HandlerFunc(helloWorld))).Methods("POST", "GET")
	v1.Handle("/anchor/did", LoggingMiddleware(http.HandlerFunc(AnchorDIDHandler))).Methods("POST", "GET")
	v1.Handle("/anchor/credential", LoggingMiddleware(http.HandlerFunc(AnchorCredentialHandler))).Methods("POST", "GET")

	return r
}

func helloWorld(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello, World"))
}
