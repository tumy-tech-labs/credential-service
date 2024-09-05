package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid" // Import the UUID package
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

// Credential structure for incoming POST requests
type CredentialRequest struct {
	IssuerDid string `json:"issuerDid"`
	Subject   struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	} `json:"subject"`
}

// Credential structure for responses
type Credential struct {
	ID      string `json:"id"`
	Issuer  string `json:"issuer"`
	Subject struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	} `json:"subject"`
	IssuanceDate   time.Time `json:"issuanceDate"`
	ExpirationDate time.Time `json:"expirationDate"`
}

// IssueCredential issues a new credential
func issueCredential(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode the request body
	var req CredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Resolve the issuer DID
	resolverURL := fmt.Sprintf("http://did-service:8080/dids/resolver?did=%s", url.QueryEscape(req.IssuerDid))
	resp, err := http.Get(resolverURL)
	if err != nil {
		log.Printf("Failed to fetch DID document from resolver: %v", err)
		http.Error(w, "Failed to resolve DID", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Received non-OK response from resolver: %s", resp.Status)
		http.Error(w, "Failed to resolve DID", http.StatusInternalServerError)
		return
	}

	// Process the DID document
	var didDocument map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&didDocument); err != nil {
		log.Printf("Failed to decode DID document: %v", err)
		http.Error(w, "Failed to process DID document", http.StatusInternalServerError)
		return
	}

	// Generate a UUID for the credential ID
	credentialID := uuid.New().String()
	credential := Credential{
		ID:             credentialID,
		Issuer:         req.IssuerDid,
		Subject:        req.Subject,
		IssuanceDate:   time.Now().UTC(),
		ExpirationDate: time.Now().UTC().Add(365 * 24 * time.Hour),
	}

	// Serialize the credential to JSON
	credentialJSON, err := json.Marshal(credential)
	if err != nil {
		log.Printf("Failed to marshal credential to JSON: %v", err)
		http.Error(w, "Failed to process credential", http.StatusInternalServerError)
		return
	}

	// Store the credential in the database
	_, err = db.Exec(context.Background(),
		"INSERT INTO verifiable_credentials (id, did, issuer, credential, issuance_date, expiration_date) VALUES ($1, $2, $3, $4, $5, $6)",
		credential.ID,
		"", // Assuming the DID of the subject is not in the payload; adjust if needed
		req.IssuerDid,
		credentialJSON,
		credential.IssuanceDate,
		credential.ExpirationDate,
	)
	if err != nil {
		log.Printf("Failed to insert credential into database: %v", err)
		http.Error(w, "Failed to issue credential", http.StatusInternalServerError)
		return
	}

	// Respond with the generated credential
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(credential); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to issue credential", http.StatusInternalServerError)
		return
	}

	log.Printf("Credential issued successfully: %v", credential)
}

// getCredentials retrieves all credentials from the database
func getCredentials(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Query to retrieve all credentials from the database
	rows, err := db.Query(context.Background(), "SELECT id, issuer, credential, issuance_date, expiration_date FROM verifiable_credentials")
	if err != nil {
		log.Printf("Failed to execute query: %v", err)
		http.Error(w, "Failed to retrieve credentials", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Collect credentials into a list
	var credentials []map[string]interface{}
	for rows.Next() {
		var id, issuer string
		var credential json.RawMessage
		var issuanceDate, expirationDate time.Time

		if err := rows.Scan(&id, &issuer, &credential, &issuanceDate, &expirationDate); err != nil {
			log.Printf("Failed to scan row: %v", err)
			http.Error(w, "Failed to retrieve credentials", http.StatusInternalServerError)
			return
		}

		// Unmarshal the credential JSONB field
		var credentialData map[string]interface{}
		if err := json.Unmarshal(credential, &credentialData); err != nil {
			log.Printf("Failed to unmarshal credential data: %v", err)
			http.Error(w, "Failed to process credential data", http.StatusInternalServerError)
			return
		}

		// Create a combined map for the response
		cred := map[string]interface{}{
			"id":             id,
			"issuer":         issuer,
			"credential":     credentialData,
			"issuanceDate":   issuanceDate.Format(time.RFC3339),
			"expirationDate": expirationDate.Format(time.RFC3339),
		}
		credentials = append(credentials, cred)
	}

	// Respond with the credentials
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(credentials); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to retrieve credentials", http.StatusInternalServerError)
		return
	}

	log.Printf("Retrieved %d credentials", len(credentials))
}

func main() {
	_ = godotenv.Load()
	initDB()

	http.HandleFunc("/credentials", issueCredential)
	http.HandleFunc("/credentials/all", getCredentials)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Issuer Service running on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
