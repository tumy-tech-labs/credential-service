package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
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

func createCredential(w http.ResponseWriter, r *http.Request) {
	// Generate issuance date and expiration date
	issuanceDate := time.Now().UTC()
	expirationDate := issuanceDate.AddDate(1, 0, 0).UTC()

	// Read the request body for subject details and issuer DID
	var payload map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Extract issuer DID from the payload
	issuerDid, ok := payload["issuerDid"].(string)
	if !ok {
		http.Error(w, "Invalid or missing 'issuerDid' field", http.StatusBadRequest)
		return
	}

	// Extract subject details from the payload
	subject, ok := payload["subject"].(map[string]interface{})
	if !ok {
		http.Error(w, "Invalid or missing 'subject' field", http.StatusBadRequest)
		return
	}

	// Generate a unique ID for the credential
	credentialID := uuid.New()

	// Generate a unique ID for the credential subject
	subjectDid := fmt.Sprintf("did:key:%s", uuid.New().String())

	// Create the credential JSON
	credential := map[string]interface{}{
		"@context":       "https://www.w3.org/2018/credentials/v1",
		"id":             credentialID.String(),
		"type":           []string{"VerifiableCredential", "EmploymentCredential"},
		"issuer":         issuerDid,
		"issuanceDate":   issuanceDate.Format(time.RFC3339),
		"expirationDate": expirationDate.Format(time.RFC3339),
		"credentialSubject": map[string]interface{}{
			"id":    subjectDid,
			"name":  subject["name"],
			"email": subject["email"],
			"phone": subject["phone"],
		},
	}

	// Convert credential to JSON
	credentialJSON, err := json.Marshal(credential)
	if err != nil {
		log.Printf("Failed to marshal credential: %v", err)
		http.Error(w, "Failed to create credential", http.StatusInternalServerError)
		return
	}

	// Insert the credential into the database
	query := `INSERT INTO verifiable_credentials (id, did, issuer, credential, issuance_date, expiration_date) 
	          VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = db.Exec(context.Background(), query, credentialID, subjectDid, issuerDid, credentialJSON, issuanceDate, expirationDate)
	if err != nil {
		log.Printf("Failed to execute query: %v", err)
		http.Error(w, "Failed to create credential", http.StatusInternalServerError)
		return
	}

	// Respond with the created credential
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(credential)
	if err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to create credential", http.StatusInternalServerError)
		return
	}

	log.Printf("Credential created successfully: %s", credentialID)
}

func getCredentials(w http.ResponseWriter, r *http.Request) {
	// Query to retrieve all credentials from the database
	rows, err := db.Query(context.Background(), "SELECT id, did, issuer, credential, issuance_date, expiration_date FROM verifiable_credentials")
	if err != nil {
		log.Printf("Failed to execute query: %v", err)
		http.Error(w, "Failed to retrieve credentials", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Collect all credentials into a slice of maps
	var credentials []map[string]interface{}
	for rows.Next() {
		var id, did, issuer, credentialJSON string
		var issuanceDate, expirationDate time.Time

		if err := rows.Scan(&id, &did, &issuer, &credentialJSON, &issuanceDate, &expirationDate); err != nil {
			log.Printf("Failed to scan row: %v", err)
			http.Error(w, "Failed to retrieve credentials", http.StatusInternalServerError)
			return
		}

		var credential map[string]interface{}
		err = json.Unmarshal([]byte(credentialJSON), &credential)
		if err != nil {
			log.Printf("Failed to unmarshal credential: %v", err)
			http.Error(w, "Failed to retrieve credentials", http.StatusInternalServerError)
			return
		}

		credentials = append(credentials, map[string]interface{}{
			"id":             id,
			"did":            did,
			"issuer":         issuer,
			"credential":     credential,
			"issuanceDate":   issuanceDate.Format(time.RFC3339),
			"expirationDate": expirationDate.Format(time.RFC3339),
		})
	}

	// Respond with the list of credentials
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

	http.HandleFunc("/credentials", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			createCredential(w, r)
		} else if r.Method == http.MethodGet {
			getCredentials(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Credential Service running on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
