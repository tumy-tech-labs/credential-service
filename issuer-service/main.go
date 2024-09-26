package main

import (
	"context"
	"log"
	"net/http"

	"os"

	"github.com/jackc/pgx/v4/pgxpool"
)

var db *pgxpool.Pool

func initDB() {
	var err error
	db, err = pgxpool.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	} else {
		log.Println("Connected to database successfully")
	}
}

func main() {
	// Connect to PostgreSQL database
	initDB()

	// Initialize routes
	InitializeRoutes()

	// Start HTTP server
	log.Println("Credential service running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
