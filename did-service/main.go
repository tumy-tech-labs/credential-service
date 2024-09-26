package main

import (
	"log"
	"os"

	"fmt"
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

	// Set up routes
	InitializeRoutes()

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("DID Service running on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
