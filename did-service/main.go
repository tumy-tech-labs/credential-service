package main

import (
	"log"
	"os"

	"fmt"
	"net/http"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
	
	"github.com/bradtumy/credential-service/internal/httpx"
	"github.com/bradtumy/credential-service/internal/logging"
)

var db *pgxpool.Pool

func main() {
	// Load environment variables
	_ = godotenv.Load()

	// Initialize logging
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	logging.Init(logLevel)

	// Initialize the database connection
	initDB()

	// Set up routes
	routes := InitializeRoutes()

	// Apply standard middleware chain with audit logging
	handler := httpx.StandardMiddlewareChain()(routes)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("DID Service running on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), handler))
}
