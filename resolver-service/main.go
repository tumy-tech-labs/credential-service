package main

import (
	"log"
	"net/http"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
)

var db *pgxpool.Pool

func main() {
	// Load environment variables
	_ = godotenv.Load()

	// Initialize the database connection
	initDB()

	// Initialize routes
	router := initializeRoutes()

	// Start the server
	log.Println("Starting Resolver Service on port 8080...")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
