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
	// Create a new credential ID
	credentialID := uuid.New().String()

	// Generate issuance date and expiration date
	issuanceDate := time.Now().UTC().Format(time.RFC3339)
	expirationDate := time.Now().AddDate(1, 0, 0).UTC().Format(time.RFC3339)

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

	// Generate a unique ID for the credential subject
	subjectID := uuid.New().String()

	// Create the credential
	credential := map[string]interface{}{
		"@context":       "https://www.w3.org/2018/credentials/v1",
		"id":             credentialID,
		"type":           []string{"VerifiableCredential", "EmploymentCredential"},
		"issuer":         issuerDid, // Use the issuer DID from the payload
		"issuanceDate":   issuanceDate,
		"expirationDate": expirationDate,
		"credentialSubject": map[string]interface{}{
			"id":    "did:key:" + subjectID, // Generate a new unique ID for the subject
			"name":  subject["name"],
			"email": subject["email"],
			"phone": subject["phone"],
		},
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

func main() {
	_ = godotenv.Load()
	initDB()

	http.HandleFunc("/credentials", createCredential)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Credential Service running on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
