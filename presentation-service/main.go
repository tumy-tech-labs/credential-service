package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	// Initialize routes
	routes := InitializeRoutes()

	// Initialize Port from the env
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Apply the logger middleware to all routes
	loggedRoutes := LoggingMiddleware(routes)

	// Start HTTP server
	log.Printf("Presentation Service running on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), loggedRoutes))
}
