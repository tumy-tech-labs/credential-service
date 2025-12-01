package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	
	"github.com/bradtumy/credential-service/internal/httpx"
	"github.com/bradtumy/credential-service/internal/logging"
)

func main() {

	// Load environment variables
	_ = godotenv.Load()

	// Initialize logging
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	logging.Init(logLevel)

	// Initialize routes
	routes := InitializeRoutes()

	// Apply standard middleware chain with audit logging
	handler := httpx.StandardMiddlewareChain()(routes)

	// Initialize Port from the env
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Holder Service running on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), handler))
}
