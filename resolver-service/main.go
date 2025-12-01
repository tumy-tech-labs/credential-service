package main

import (
	"log"
	"net/http"
	"os"

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

	// Initialize routes
	router := initializeRoutes()

	// Apply standard middleware chain with comprehensive audit logging
	handler := httpx.StandardMiddlewareChain()(router)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting Resolver Service on port %s...", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
