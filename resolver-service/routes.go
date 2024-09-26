package main

import (
	"github.com/gorilla/mux"
)

func initializeRoutes() *mux.Router {
	router := mux.NewRouter()

	// Define the resolver route
	router.HandleFunc("/dids/resolver", resolveDIDHandler).Methods("GET")

	return router
}
