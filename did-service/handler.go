package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v4"
)

type DIDDocument struct {
	Context        string `json:"@context"`
	ID             string `json:"id"`
	PublicKey      string `json:"publicKey"`
	CreatedAt      string `json:"createdAt"`
	OrganizationID string `json:"organization_id"`
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

	// Convert ed25519.PrivateKey to base64 string
	encodedPrivateKey := base64.StdEncoding.EncodeToString(privateKey)

	// Extract organization_id from the request payload
	var payload map[string]interface{}
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	organizationID, ok := payload["organization_id"].(string)
	if !ok {
		organizationID = "default-org" // Fallback value if not provided
	}

	// Construct the DID
	did := fmt.Sprintf("did:key:z6M%s", encodedPublicKey)
	createdAt := time.Now().UTC()

	// Create the DID Document
	didDocument := DIDDocument{
		Context:        "https://www.w3.org/ns/did/v1",
		ID:             did,
		PublicKey:      encodedPublicKey,
		CreatedAt:      createdAt.Format(time.RFC3339),
		OrganizationID: organizationID,
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

	// Securely store the private key
	log.Printf("Private key for DID %s: %x", did, encodedPrivateKey)
	err = savePrivateKeyToVault(did, encodedPrivateKey)
	if err != nil {
		log.Printf("Error storing private key: %s", err)
	}

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

func getDID(w http.ResponseWriter, r *http.Request) {
	did := r.URL.Query().Get("did")
	if did == "" {
		http.Error(w, "Missing DID", http.StatusBadRequest)
		return
	}

	// Query to retrieve a specific DID document from the database
	var document string
	err := db.QueryRow(context.Background(), "SELECT document FROM dids WHERE did = $1", did).Scan(&document)
	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "DID not found", http.StatusNotFound)
		} else {
			log.Printf("Failed to execute query: %v", err)
			http.Error(w, "Failed to retrieve DID", http.StatusInternalServerError)
		}
		return
	}

	// Respond with the DID document
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(document))
	log.Printf("Retrieved DID document: %s", did)
}

func savePrivateKeyToVault(did string, privateKey string) error {
	client, err := getVaultClient()
	if err != nil {
		return fmt.Errorf("failed to initialize Vault client: %w", err)
	}

	data := map[string]interface{}{
		"data": map[string]interface{}{
			"private_key": privateKey,
		},
	}

	// Write the private key to Vault at the path "secret/data/dids/<did>"
	secretPath := fmt.Sprintf("secret/data/dids/%s", did)
	_, err = client.Logical().Write(secretPath, data)
	if err != nil {
		return fmt.Errorf("failed to write private key to Vault: %w", err)
	}

	return nil
}
