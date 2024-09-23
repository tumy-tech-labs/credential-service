package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Initialize routes for verifier service
	routes := InitializeRoutes()

	fmt.Println("Starting Verifier service on port 8083...")
	log.Fatal(http.ListenAndServe(":8080", routes))
}
