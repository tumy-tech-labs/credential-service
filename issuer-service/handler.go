package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/vault/api"
)

// VerifiableCredential structure following W3C schema
type VerifiableCredential struct {
	Context           []string          `json:"@context"`
	Type              []string          `json:"type"`
	ID                string            `json:"id"`
	Issuer            string            `json:"issuer"`
	IssuanceDate      string            `json:"issuanceDate"`
	ExpirationDate    string            `json:"expirationDate"`
	CredentialSubject map[string]string `json:"credentialSubject"`
	Proof             Proof             `json:"proof,omitempty"`
}

// Proof structure for digital signature
type Proof struct {
	Type               string `json:"type"`
	Created            string `json:"created"`
	ProofValue         string `json:"proofValue"`
	ProofPurpose       string `json:"proofPurpose"`
	VerificationMethod string `json:"verificationMethod"`
}

// Request payload for issuing a credential
type CredentialRequest struct {
	IssuerDid string `json:"issuerDid"`
	Subject   struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	} `json:"subject"`
}

// IssueCredential issues a verifiable credential
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
	resolverURL := fmt.Sprintf("http://resolver-service:8080/dids/resolver?did=%s", url.QueryEscape(req.IssuerDid))
	//resolverURL := fmt.Sprintf("http://resolver-service:8087/dids/resolver?did=%s", req.IssuerDid)
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
	vaultClient, err := getVaultClient()
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

	privateKey, err := parseEd25519PrivateKeyFromBase64(privateKeyPEM)
	if err != nil {
		log.Printf("Failed to parse private key: %v", err)
		http.Error(w, "Failed to issue credential", http.StatusInternalServerError)
		return
	}

	// Process DID document
	var didDocument map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&didDocument); err != nil {
		log.Printf("Failed to decode DID document: %v", err)
		http.Error(w, "Failed to process DID document", http.StatusInternalServerError)
		return
	}

	// Generate credential ID and set issuance/expiration dates
	credentialID := uuid.New().String()
	issuanceDate := time.Now().UTC().Format(time.RFC3339)
	expirationDate := time.Now().AddDate(1, 0, 0).UTC().Format(time.RFC3339)

	// Create the verifiable credential
	credential := VerifiableCredential{
		Context:        []string{"https://www.w3.org/2018/credentials/v1"},
		Type:           []string{"VerifiableCredential"},
		ID:             credentialID,
		Issuer:         req.IssuerDid,
		IssuanceDate:   issuanceDate,
		ExpirationDate: expirationDate,
		CredentialSubject: map[string]string{
			"id":    "did:example:" + uuid.New().String(),
			"name":  req.Subject.Name,
			"email": req.Subject.Email,
			"phone": req.Subject.Phone,
		},
	}

	// Serialize the credential to JSON
	credentialJSON, err := json.Marshal(credential)
	if err != nil {
		log.Printf("Failed to marshal credential to JSON: %v", err)
		http.Error(w, "Failed to process credential", http.StatusInternalServerError)
		return
	}

	// Sign the credential
	signature, err := signCredential(privateKey, credentialJSON)
	if err != nil {
		log.Printf("Failed to sign credential: %v", err)
		http.Error(w, "Failed to issue credential", http.StatusInternalServerError)
		return
	}

	// Attach proof to the credential
	credential.Proof = Proof{
		Type:               "Ed25519Signature2018",
		Created:            time.Now().UTC().Format(time.RFC3339),
		ProofValue:         base64.StdEncoding.EncodeToString(signature),
		ProofPurpose:       "assertionMethod",
		VerificationMethod: req.IssuerDid + "#keys-1",
	}

	// Serialize the proof to JSON
	proofJSON, err := json.Marshal(credential.Proof)
	if err != nil {
		log.Printf("Failed to marshal proof to JSON: %v", err)
		http.Error(w, "Failed to process credential", http.StatusInternalServerError)
		return
	}

	// Store the credential in the database
	_, err = db.Exec(context.Background(),
		"INSERT INTO verifiable_credentials (id, did, issuer, credential, issuance_date, expiration_date, proof) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		credential.ID,
		credential.CredentialSubject["id"], // Use the subject ID here
		req.IssuerDid,
		credentialJSON,
		credential.IssuanceDate,
		credential.ExpirationDate,
		proofJSON, // Insert the proof JSON here
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

// Additional functions for Vault, parsing, signing, etc.

func getPrivateKeyFromVault(issuerDid string, client *api.Client) (string, error) {
	// Placeholder: add logic to retrieve the key from HashiCorp Vault
	return "PRIVATE_KEY_BASE64_STRING", nil
}

func parseEd25519PrivateKeyFromBase64(key string) ([]byte, error) {
	// Placeholder: add logic to parse the private key
	return []byte(key), nil
}

func signCredential(privateKey []byte, credentialJSON []byte) ([]byte, error) {
	// Placeholder: add logic to sign the credential using Ed25519
	return []byte("SIGNATURE"), nil
}
