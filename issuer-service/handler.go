package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/vault/api"
)

// VerifiableCredential structure following W3C schema
type VerifiableCredential struct {
	Context           []string               `json:"@context"`
	Type              []string               `json:"type"`
	ID                string                 `json:"id"`
	Issuer            string                 `json:"issuer"`
	IssuanceDate      string                 `json:"issuanceDate"`
	ExpirationDate    string                 `json:"expirationDate"`
	CredentialSubject map[string]interface{} `json:"credentialSubject"`
	Proof             Proof                  `json:"proof,omitempty"`
}

// Proof structure for digital signature
type Proof struct {
	Type               string `json:"type"`
	Created            string `json:"created"`
	ProofValue         string `json:"proofValue"`
	ProofPurpose       string `json:"proofPurpose"`
	VerificationMethod string `json:"verificationMethod"`
}

// Updated Request payload for issuing a credential - using a map enables us to support different schema combinations.
type CredentialRequest struct {
	IssuerDid string                   `json:"issuerDid"`
	Subjects  []map[string]interface{} `json:"subject"` // Change to a dynamic structure
}

// BaseSchema represents the structure of the base schema
// TODO: use a map for the base schema too so that we can change the base schema json file and dynamically update the type
type BaseSchema struct {
	CredentialID   Property `json:"credentialID"`
	CredentialType Property `json:"credentialType"`
	IssueDate      Property `json:"issueDate"`
	ExpirationDate Property `json:"expirationDate"`
	Issuer         Property `json:"issuer"`
}

// Property represents a property of the schema
type Property struct {
	Name     string `json:"name"`     // Name of the property
	Type     string `json:"type"`     // Type of the property (e.g., string, integer)
	Required bool   `json:"required"` // Indicate if the property is required
}

// Schema represents a JSON Schema structure.
type Schema struct {
	Properties []Property `json:"properties"` // Array of properties
}

// loadBaseSchema loads the base schema from a JSON file.
func loadBaseSchema(filePath string) (BaseSchema, error) {
	var baseSchema BaseSchema
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		return baseSchema, fmt.Errorf("failed to read base schema file: %v", err)
	}

	if err := json.Unmarshal(file, &baseSchema); err != nil {
		return baseSchema, fmt.Errorf("failed to unmarshal base schema JSON: %v", err)
	}

	return baseSchema, nil
}

/*
func mergeSchemas(base BaseSchema, customerSchema Schema) Schema {
	// Create a new schema to hold the merged properties
	mergedSchema := Schema{
		Properties: []Property{
			base.CredentialID,
			base.CredentialType,
			base.IssueDate,
			base.ExpirationDate,
			base.Issuer,
		},
	}

	// Append customer schema properties
	mergedSchema.Properties = append(mergedSchema.Properties, customerSchema.Properties...)

	return mergedSchema
}
*/

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

	if len(req.Subjects) == 0 {
		http.Error(w, "No subjects provided", http.StatusBadRequest)
		return
	}

	// Enqueue the bulk request to RabbitMQ for processing.
	if err := enqueueBulkIssuance(req); err != nil {
		log.Printf("Failed to enqueue bulk issuance: %v", err)
		http.Error(w, "Failed to process request", http.StatusInternalServerError)
		return
	}

	// Log the request for debugging purposes
	log.Printf("Received credential request: %+v", req)

	// Resolve the issuer DID
	log.Println("Here is the issuer DID: ", req.IssuerDid)
	resolverURL := fmt.Sprintf("http://resolver-service:8080/v1/dids/resolver?did=%s", url.QueryEscape(req.IssuerDid))
	log.Println("Resolver Endpoint:: ", resolverURL)
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

	log.Println("Here's the Private Key to parse: ", privateKeyPEM)

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
	//credentialID := uuid.New().String()
	issuanceDate := time.Now().UTC().Format(time.RFC3339)
	expirationDate := time.Now().AddDate(1, 0, 0).UTC().Format(time.RFC3339)

	log.Printf("*** I have: %d subjects in the request", len(req.Subjects))

	// Loop through each subject and create a verifiable credential
	for _, subject := range req.Subjects {
		// Generate a unique credential ID for each subject
		credentialID := uuid.New().String()

		log.Printf("*** Here is the Unique Credential ID: %s", credentialID)

		// Create the verifiable credential for the current subject
		credential := VerifiableCredential{
			Context:        []string{"https://www.w3.org/2018/credentials/v1"},
			Type:           []string{"VerifiableCredential"},
			ID:             "",
			Issuer:         req.IssuerDid,
			IssuanceDate:   issuanceDate,
			ExpirationDate: expirationDate,
			CredentialSubject: map[string]interface{}{
				"subject": subject, // Use the current subject
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

		log.Printf("Inserting credential with DID: %s", credential.Issuer) // Assuming you use Issuer as DID

		// Loop through each subject in the request
		for _, subject := range req.Subjects {
			// Extract subjectID from the current subject
			subjectID, ok := subject["id"].(string) // Assuming the subject has an "id" field
			if !ok {
				log.Printf("Subject ID is missing or not a string for subject: %+v", subject)
				http.Error(w, "Invalid subject data", http.StatusBadRequest)
				return
			}

			// Store the credential in the database for each subject
			_, err = db.Exec(context.Background(),
				"INSERT INTO verifiable_credentials (did, issuer, credential, subject, issuance_date, expiration_date, proof) VALUES ($1, $2, $3, $4, $5, $6, $7)",
				subjectID,      // Use the extracted subject ID for each subject
				req.IssuerDid,  // Issuer DID
				credentialJSON, // Credential JSON
				subject,        // Insert the current subject's properties
				credential.IssuanceDate,
				credential.ExpirationDate,
				proofJSON, // Insert the proof JSON here
			)

			if err != nil {
				if err.Error() == "ERROR: duplicate key value violates unique constraint \"verifiable_credentials_pkey\"" {
					log.Printf("Duplicate key for subject %v. Attempting to regenerate credential ID.", subject)
					// Optionally, you can generate a new credential ID and retry the insertion
					continue // Skip or handle the error as needed
				} else {
					log.Printf("Error inserting credential for subject %v: %v", subject, err)
				}
			}
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
}

/*
func loadCustomerSchema(s string) {
	panic("unimplemented")
}
*/

// Additional functions for Vault, parsing, signing, etc.

// Function to retrieve the private key from HashiCorp Vault
func getPrivateKeyFromVault(issuerDid string, client *api.Client) (string, error) {
	// Assuming the path to the secret in Vault is structured as "secret/data/dids/<issuerDid>"
	secretPath := fmt.Sprintf("secret/data/dids/%s", issuerDid)
	log.Println("Secrets path --> We are looking for secrets here: ", secretPath)

	// Fetch the secret from Vault
	secret, err := client.Logical().Read(secretPath)
	if err != nil {
		return "", fmt.Errorf("failed to read secret from Vault: %w", err)
	}
	if secret == nil {
		return "", fmt.Errorf("no secret found at path: %s", secretPath)
	}

	// Retrieve the base64-encoded private key from the secret data
	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("secret data is not in the expected format")
	}

	privateKeyBase64, ok := data["private_key"].(string)
	if !ok {
		return "", fmt.Errorf("private key not found in secret data")
	}

	return privateKeyBase64, nil
}

// Function to parse the base64-encoded Ed25519 private key
func parseEd25519PrivateKeyFromBase64(base64Key string) ([]byte, error) {
	// Decode the base64 string
	privateKeyBytes, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 private key: %w", err)
	}
	return privateKeyBytes, nil
}

// Function to sign the credential using the Ed25519 private key
func signCredential(privateKey []byte, credentialJSON []byte) ([]byte, error) {
	// Use your signing logic here.
	// Placeholder: In practice, use a library to sign with the Ed25519 key
	signature := []byte("SIGNATURE") // Replace with actual signing logic

	return signature, nil
}
