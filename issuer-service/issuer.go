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

func closeDB() {
	if db != nil {
		db.Close()
		log.Println("Database connection closed")
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

// getCredentials retrieves all credentials from the database, including their revocation status
func getCredentials(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := db.Query(context.Background(),
		"SELECT id, subject_name, subject_email, subject_phone, issue_date, expiration_date, issuer, revoked, revoked_at FROM verifiable_credentials")
	if err != nil {
		log.Printf("Failed to retrieve credentials: %v", err)
		http.Error(w, "Failed to retrieve credentials", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var credentials []map[string]interface{}
	for rows.Next() {
		var (
			id             string
			subjectName    string
			subjectEmail   string
			subjectPhone   string
			issueDate      time.Time
			expirationDate time.Time
			issuer         string
			revoked        bool
			revokedAt      *time.Time // Nullable field
		)

		err := rows.Scan(&id, &subjectName, &subjectEmail, &subjectPhone, &issueDate, &expirationDate, &issuer, &revoked, &revokedAt)
		if err != nil {
			log.Printf("Failed to scan credential: %v", err)
			http.Error(w, "Failed to retrieve credentials", http.StatusInternalServerError)
			return
		}

		credential := map[string]interface{}{
			"id":             id,
			"subjectName":    subjectName,
			"subjectEmail":   subjectEmail,
			"subjectPhone":   subjectPhone,
			"issueDate":      issueDate.Format(time.RFC3339),
			"expirationDate": expirationDate.Format(time.RFC3339),
			"issuer":         issuer,
			"revoked":        revoked,
		}

		// Add revokedAt if the credential is revoked
		if revokedAt != nil {
			credential["revokedAt"] = revokedAt.Format(time.RFC3339)
		}

		credentials = append(credentials, credential)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Failed to retrieve credentials: %v", err)
		http.Error(w, "Failed to retrieve credentials", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(credentials)
}

// revokeCredential handles revoking a credential based on the credential ID and issuer DID
func revokeCredential(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the incoming revocation request
	var request struct {
		CredentialID string `json:"credentialId"`
		IssuerDid    string `json:"issuerDid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate the existence of the credential and ensure it belongs to the issuer
	var issuerDid string
	err := db.QueryRow(context.Background(),
		"SELECT issuer FROM verifiable_credentials WHERE id = $1", request.CredentialID).Scan(&issuerDid)
	if err != nil {
		log.Printf("Credential not found: %v", err)
		http.Error(w, "Credential not found", http.StatusNotFound)
		return
	}

	if issuerDid != request.IssuerDid {
		log.Printf("Issuer mismatch: expected %v, got %v", issuerDid, request.IssuerDid)
		http.Error(w, "Issuer mismatch", http.StatusUnauthorized)
		return
	}

	// Mark the credential as revoked
	_, err = db.Exec(context.Background(),
		"UPDATE verifiable_credentials SET revoked = TRUE, revoked_at = $1 WHERE id = $2", time.Now().UTC(), request.CredentialID)
	if err != nil {
		log.Printf("Failed to revoke credential: %v", err)
		http.Error(w, "Failed to revoke credential", http.StatusInternalServerError)
		return
	}

	// Send response
	response := map[string]interface{}{
		"message":   "Credential revoked successfully",
		"revoked":   true,
		"revokedAt": time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	_ = godotenv.Load()
	initDB()

	// Ensure the database connection is closed when the program exits
	defer closeDB()

	http.HandleFunc("/credentials", issueCredential)
	http.HandleFunc("/credentials/all", getCredentials)
	http.HandleFunc("/credentials/revoke", revokeCredential) // New endpoint for revocation

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Issuer Service running on port %s\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
