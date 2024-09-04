package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

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
	// Generate a new Ed25519 key pair
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		log.Printf("Failed to generate key pair: %v", err)
		http.Error(w, "Failed to generate DID", http.StatusInternalServerError)
		return
	}

	// Encode the public key in base64 to use in the DID
	encodedPublicKey := base64.RawURLEncoding.EncodeToString(publicKey)

	// Construct the DID according to the did:key method
	did := fmt.Sprintf("did:key:z6M%s", encodedPublicKey)
	organizationID := "default-org" // Replace with actual organization ID as needed
	createdAt := time.Now().UTC()   // Current timestamp in UTC

	log.Printf("Creating DID: %s for organization: %s", did, organizationID)

	// Insert DID and public key into the database
	query := "INSERT INTO dids (did, organization_id, created_at, public_key) VALUES ($1, $2, $3, $4)"
	log.Printf("Executing query: %s with DID: %s, organization_id: %s, created_at: %s, public_key: %s", query, did, organizationID, createdAt, encodedPublicKey)
	_, err = db.Exec(context.Background(), query, did, organizationID, createdAt, encodedPublicKey)
	if err != nil {
		log.Printf("Failed to execute query: %v", err)
		http.Error(w, "Failed to create DID", http.StatusInternalServerError)
		return
	}

	// Securely store the private key, possibly in a file vault (stubbed out here)
	// For example, the private key could be saved to a secure vault like HashiCorp Vault
	// Here we just log the private key (DO NOT DO THIS IN PRODUCTION)
	log.Printf("Private key for DID %s: %x", did, privateKey)

	// Respond with the created DID, public key, and additional details
	response := map[string]interface{}{
		"did":             did,
		"organization_id": organizationID,
		"created_at":      createdAt.Format(time.RFC3339),
		"public_key":      encodedPublicKey,
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to create DID", http.StatusInternalServerError)
		return
	}

	log.Printf("DID created successfully: %s", did)
}

func getDIDs(w http.ResponseWriter, r *http.Request) {
	// Query to retrieve all DIDs from the database
	rows, err := db.Query(context.Background(), "SELECT did, organization_id, created_at, public_key FROM dids")
	if err != nil {
		log.Printf("Failed to execute query: %v", err)
		http.Error(w, "Failed to retrieve DIDs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Collect all DIDs into a slice of maps
	var dids []map[string]interface{}
	for rows.Next() {
		var did, organizationID, publicKey string
		var createdAt time.Time // Use time.Time to match the database type

		if err := rows.Scan(&did, &organizationID, &createdAt, &publicKey); err != nil {
			log.Printf("Failed to scan row: %v", err)
			http.Error(w, "Failed to retrieve DIDs", http.StatusInternalServerError)
			return
		}

		dids = append(dids, map[string]interface{}{
			"did":             did,
			"organization_id": organizationID,
			"created_at":      createdAt.Format(time.RFC3339),
			"public_key":      publicKey,
		})
	}

	// Respond with the list of DIDs
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(dids); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to retrieve DIDs", http.StatusInternalServerError)
		return
	}

	log.Printf("Retrieved %d DIDs", len(dids))
}

func main() {
	_ = godotenv.Load()
	initDB()

	http.HandleFunc("/dids", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			createDID(w, r)
		} else if r.Method == http.MethodGet {
			getDIDs(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("DID Creation Service running on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
