package main

import (
	"log"
	"net/http"
)

// Entry point of the application
func main() {
	// Initialize routes
	router := initializeRoutes()

	// Start the server
	log.Println("Starting server on :8080...")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
