package main

import (
	"github.com/gorilla/mux"
)

func initializeRoutes() *mux.Router {
	r := mux.NewRouter()

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
