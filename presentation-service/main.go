package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	
	"github.com/bradtumy/credential-service/internal/httpx"
	"github.com/bradtumy/credential-service/internal/logging"
)

func main() {
	// Initialize logging
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	logging.Init(logLevel)

	// Initialize routes
	routes := InitializeRoutes()

	// Initialize Port from the env
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Apply standard middleware chain with comprehensive audit logging
	handler := httpx.StandardMiddlewareChain()(routes)

	// Start HTTP server
	log.Printf("Presentation Service running on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), handler))
}
