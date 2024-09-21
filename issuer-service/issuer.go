package main

import (
	"context"

	//"crypto/ecdsa"
	"crypto/ed25519"

	//"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid" // Import the UUID package
	"github.com/hashicorp/vault/api"
	vault "github.com/hashicorp/vault/api"
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

// Create a Vault client
func initVaultClient() (*vault.Client, error) {
	client, err := vault.NewClient(vault.DefaultConfig())
	if err != nil {
		log.Printf("Error: %v", err)
		return nil, fmt.Errorf("failed to initialize Vault client: %w", err)
	}

	client.SetAddress(os.Getenv("VAULT_ADDR"))
	client.SetToken(os.Getenv("VAULT_TOKEN"))

	log.Printf("Vault Addy: %v", client.Address())
	log.Printf("Vault Token: %v", client.Token())
	log.Printf("Vault Client: %v", client)

	return client, nil
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
	Signature      string    `json:"signature"`
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

	// Initialize Vault client
	vaultClient, err := initVaultClient()
	if err != nil {
		log.Printf("Failed to initialize Vault client: %v", err)
		http.Error(w, "Failed to connect to Vault", http.StatusInternalServerError)
		return
	}

	// Retrieve the private key from Vault
	privateKeyPEM, err := getPrivateKeyFromVault(req.IssuerDid, vaultClient)
	if err != nil {
		log.Printf("Failed to retrieve private key from Vault: %v", err)
		http.Error(w, "Failed to issue credential", http.StatusInternalServerError)
		return
	}

	// Decode the private key (assuming it's in PEM format)
	// privateKey, err := parsePrivateKeyFromPEM(privateKeyPEM)
	privateKey, err := parseEd25519PrivateKeyFromBase64(privateKeyPEM)
	if err != nil {
		log.Printf("Failed to parse private key: %v", err)
		http.Error(w, "Failed to issue credential", http.StatusInternalServerError)
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

	// Create a digital signature
	signature, err := signCredential(privateKey, credentialJSON)
	if err != nil {
		log.Printf("Failed to sign credential: %v", err)
		http.Error(w, "Failed to issue credential", http.StatusInternalServerError)
		return
	}

	// Attach signature to the credential
	credential.Signature = base64.StdEncoding.EncodeToString(signature)

	// Store the credential in the database
	_, err = db.Exec(context.Background(),
		"INSERT INTO verifiable_credentials (id, did, issuer, credential, issuance_date, expiration_date, signature) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		credential.ID,
		"", // Assuming the DID of the subject is not in the payload; adjust if needed
		req.IssuerDid,
		credentialJSON,
		credential.IssuanceDate,
		credential.ExpirationDate,
		credential.Signature,
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

	// Query the database and extract the subject details from the JSON field
	rows, err := db.Query(context.Background(),
		`SELECT id, 
			credential->'subject'->>'name' AS subject_name, 
			credential->'subject'->>'email' AS subject_email, 
			credential->'subject'->>'phone' AS subject_phone, 
			issuance_date, 
			expiration_date, 
			issuer, 
			revoked, 
			revoked_at 
		FROM verifiable_credentials`)
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

		// Scan the result into the variables
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

// Retrieves the private key from Vault
func getPrivateKeyFromVault(did string, vaultClient *api.Client) (string, error) {

	log.Printf("Function: getPrivateKeyFromVault")
	log.Printf("Passed DID: %s", did)
	log.Println("Passed Client:", vaultClient.Address())
	log.Println("Passed Token:", vaultClient.Token())
	log.Println("Vault Logical:", vaultClient.Logical())

	// URL encode the DID to safely include it in the path
	// encodedDID := url.QueryEscape(did)
	vaultPath := fmt.Sprintf("secret/data/dids/%s", did)
	log.Printf("Vault path: %s", vaultPath)

	secret, err := vaultClient.Logical().Read(vaultPath)

	log.Println("Secret: ", secret)
	log.Println("Secret Data: ", secret.Data)

	if err != nil {
		return "", fmt.Errorf("failed to read private key from Vault: %w", err)
	}

	if secret == nil || secret.Data["data"] == nil {
		return "", fmt.Errorf("no private key found for DID: %s", did)
	}

	// Check if "data" exists in the response and is of correct type
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected data format in Vault response for DID: %s", did)
	}

	// Check if "privateKey" exists in the "data" map and is a string
	privateKey, ok := data["private_key"].(string)
	if !ok {
		// Use fmt.Printf to log the actual type of the privateKey
		if val, exists := data["private_key"]; exists {
			fmt.Printf("privateKey exists but is of type: %T, value: %#v\n", val, val)
		} else {
			fmt.Printf("privateKey not found for DID: %s\n", did)
		}
		return "", fmt.Errorf("private key not found or invalid format for DID: %s", did)
	}

	return privateKey, nil
}

/*
func parsePrivateKeyFromPEM(privateKeyPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("failed to decode PEM block containing the key")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}
*/

// parseEd25519PrivateKeyFromBase64 decodes a Base64-encoded Ed25519 private key.
func parseEd25519PrivateKeyFromBase64(encodedPrivateKey string) (ed25519.PrivateKey, error) {
	// Decode the Base64 string
	privateKeyBytes, err := base64.StdEncoding.DecodeString(encodedPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	// Ensure the private key is the correct length for Ed25519
	if len(privateKeyBytes) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size: expected %d, got %d", ed25519.PrivateKeySize, len(privateKeyBytes))
	}

	// Return the parsed Ed25519 private key
	privateKey := ed25519.PrivateKey(privateKeyBytes)
	return privateKey, nil
}

// signCredential creates a digital signature for the given credential JSON using the Ed25519 private key.
func signCredential(privateKey ed25519.PrivateKey, credentialJSON []byte) ([]byte, error) {
	// Create a digital signature
	signature := ed25519.Sign(privateKey, credentialJSON)

	// Optionally, you can return the signature as JSON
	return signature, nil
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
