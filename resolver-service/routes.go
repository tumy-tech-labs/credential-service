package main

import (
	"github.com/gorilla/mux"
)

func initializeRoutes() *mux.Router {
	r := mux.NewRouter()

	// Define the resolver route
	// Version 1 routes
	v1 := r.PathPrefix("/v1").Subrouter()
	v1.HandleFunc("/dids/resolver", resolveDIDHandler).Methods("GET")

	return r
}
