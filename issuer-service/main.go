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

	/* troubleshooting code */

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current working directory: %v", err)
	}
	log.Printf("Current working directory: %s", cwd)

	if _, err := os.Stat("./configs/base-schema.json"); os.IsNotExist(err) {
		log.Fatalf("File does not exist: %v", err)
	} else {
		log.Println("the base schema file exists")
	}

	// Connect to PostgreSQL database
	initDB()

	// Initialize routes
	route := InitializeRoutes()

	// Load the base schema at startup
	baseSchema, err := loadBaseSchema("configs/base-schema.json")

	if err != nil {
		log.Fatalf("Error loading base schema: %v", err)
	}

	// Use the base schema as needed in your application
	log.Printf("Loaded Base Schema: %+v", baseSchema)

	// Start HTTP server
	log.Println("Credential service running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", route))
}
