package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
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

func createDID(w http.ResponseWriter, r *http.Request) {
	// Sample stubbed DID creation logic
	did := "did:example:123456789abcdefghi"
	organizationID := "default-org" // Replace with actual organization ID as needed

	log.Printf("Creating DID: %s for organization: %s", did, organizationID)

	// Insert DID into the database
	query := "INSERT INTO dids (did, organization_id) VALUES ($1, $2)"
	log.Printf("Executing query: %s with DID: %s and organization_id: %s", query, did, organizationID)
	_, err := db.Exec(context.Background(), query, did, organizationID)
	if err != nil {
		log.Printf("Failed to execute query: %v", err)
		http.Error(w, "Failed to create DID", http.StatusInternalServerError)
		return
	}

	// Respond with the created DID
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]string{"did": did})
	if err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to create DID", http.StatusInternalServerError)
		return
	}
	log.Printf("DID created successfully: %s", did)
}

func main() {
	_ = godotenv.Load()
	initDB()

	http.HandleFunc("/dids", createDID)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("DID Creation Service running on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
