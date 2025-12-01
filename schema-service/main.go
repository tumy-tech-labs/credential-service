package main

import (
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
	router := initializeRoutes()

	// Apply standard middleware chain with comprehensive audit logging
	handler := httpx.StandardMiddlewareChain()(router)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Schema Service is running on port %s...", port)
	err := http.ListenAndServe(":"+port, handler)
	if err != nil {
		log.Fatal("Server failed to start: ", err)
	}
}
