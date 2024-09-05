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

// DID Document structure
type DIDDocument struct {
	Context   string `json:"@context"`
	ID        string `json:"id"`
	PublicKey string `json:"publicKey"`
	CreatedAt string `json:"createdAt"`
}

// Create a new DID and store the DID document in the database
func createDID(w http.ResponseWriter, r *http.Request) {
	// Generate a new Ed25519 key pair
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		log.Printf("Failed to generate key pair: %v", err)
		http.Error(w, "Failed to generate DID", http.StatusInternalServerError)
		return
	}

	// Encode the public key in base64
	encodedPublicKey := base64.RawURLEncoding.EncodeToString(publicKey)

	// Construct the DID
	did := fmt.Sprintf("did:key:z6M%s", encodedPublicKey)
	organizationID := "default-org" // Replace with actual organization ID
	createdAt := time.Now().UTC()

	// Create the DID Document
	didDocument := DIDDocument{
		Context:   "https://www.w3.org/ns/did/v1",
		ID:        did,
		PublicKey: encodedPublicKey,
		CreatedAt: createdAt.Format(time.RFC3339),
	}

	// Convert the DID document to JSON for storage
	didDocJSON, err := json.Marshal(didDocument)
	if err != nil {
		log.Printf("Failed to marshal DID document: %v", err)
		http.Error(w, "Failed to generate DID", http.StatusInternalServerError)
		return
	}

	// Store the DID, public key, and DID document in the database
	query := "INSERT INTO dids (did, organization_id, created_at, public_key, document) VALUES ($1, $2, $3, $4, $5)"
	_, err = db.Exec(context.Background(), query, did, organizationID, createdAt, encodedPublicKey, didDocJSON)
	if err != nil {
		log.Printf("Failed to insert DID into database: %v", err)
		http.Error(w, "Failed to store DID", http.StatusInternalServerError)
		return
	}

	// Securely store the private key (stubbed out for now)
	log.Printf("Private key for DID %s: %x", did, privateKey)

	// Respond with the DID document
	w.Header().Set("Content-Type", "application/json")
	w.Write(didDocJSON)
	log.Printf("DID created successfully: %s", did)
}

func getDIDs(w http.ResponseWriter, r *http.Request) {
	// Query to retrieve DIDs from the database
	rows, err := db.Query(context.Background(), "SELECT did, document FROM dids")
	if err != nil {
		log.Printf("Failed to execute query: %v", err)
		http.Error(w, "Failed to retrieve DIDs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Collect DIDs into a list of documents
	var dids []json.RawMessage
	for rows.Next() {
		var did, document string
		if err := rows.Scan(&did, &document); err != nil {
			log.Printf("Failed to scan row: %v", err)
			http.Error(w, "Failed to retrieve DIDs", http.StatusInternalServerError)
			return
		}
		dids = append(dids, json.RawMessage(document))
	}

	// Respond with the DID documents
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
	log.Printf("DID Service running on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
