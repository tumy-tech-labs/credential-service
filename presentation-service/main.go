package main

import (
	"log"
	"net/http"
)

func main() {
	// Initialize routes
	InitializeRoutes()

	// Start HTTP server
	log.Println("Presentation service running on port 8081...")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
