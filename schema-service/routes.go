package main

import (
	"github.com/gorilla/mux"
)

func initializeRoutes() *mux.Router {
	router := mux.NewRouter()

	// Schema routes
	router.HandleFunc("/schemas", createSchema).Methods("POST")
	router.HandleFunc("/schemas", getAllSchemas).Methods("GET")
	router.HandleFunc("/schemas/{id}", getSchemaByID).Methods("GET")
	router.HandleFunc("/schemas/{id}", updateSchema).Methods("PUT")
	router.HandleFunc("/schemas/{id}", deleteSchema).Methods("DELETE")

	return router
}
