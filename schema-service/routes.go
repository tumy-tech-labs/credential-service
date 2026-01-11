package main

import (
	"github.com/gorilla/mux"
	"net/http"
)

func initializeRoutes() *mux.Router {
	r := mux.NewRouter()

	// Basic health and readiness endpoints
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}).Methods("GET")
	r.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	}).Methods("GET")

	// Version 1 routes
	v1 := r.PathPrefix("/v1").Subrouter()
	// Schema routes
	v1.HandleFunc("/schemas", createSchema).Methods("POST")
	v1.HandleFunc("/schemas", getAllSchemas).Methods("GET")
	v1.HandleFunc("/schemas/{id}", getSchemaByID).Methods("GET")
	v1.HandleFunc("/schemas/{id}", updateSchema).Methods("PUT")
	v1.HandleFunc("/schemas/{id}", deleteSchema).Methods("DELETE")

	return r
}
