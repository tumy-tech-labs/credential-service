package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
)

// VerifiableCredential structure aligned with W3C
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

type PresentationRequest struct {
	HolderDID string   `json:"holderDid"`
	VCIDs     []string `json:"vcIds"`
}

// VerifiablePresentation represents a verifiable presentation
type VerifiablePresentation struct {
	Context              []string               `json:"@context"`
	Type                 []string               `json:"type"`
	Holder               string                 `json:"holder"`
	VerifiableCredential []VerifiableCredential `json:"verifiableCredential"`
	Proof                Proof                  `json:"proof,omitempty"`
}

func ReceiveCredential(w http.ResponseWriter, r *http.Request) {
	log.Println("Entered the Receive Credentials Handler")
	// Handle receiving a verifiable credential
	var vc VerifiableCredential

	err := json.NewDecoder(r.Body).Decode(&vc)
	if err != nil {
		http.Error(w, "Invalid credential format", http.StatusBadRequest)
		return
	}
	// Debug:
	log.Println("Debug: VC: ", vc)

	// Store the credential in memory (for now)
	StoreCredential(vc)

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Credential received successfully"))
}

func PresentCredential(w http.ResponseWriter, r *http.Request) {
	// Logic to present stored credentials
	var req PresentationRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	credentials := GetStoredCredentials()

	// Create the Verifiable Presentation
	presentation := VerifiablePresentation{
		Context:              []string{"https://www.w3.org/2018/credentials/v1"},
		Type:                 []string{"VerifiablePresentation"},
		Holder:               req.HolderDID,
		VerifiableCredential: credentials,
	}

	// Sign the presentation
	if err := SignPresentation(&presentation, req.HolderDID); err != nil {
		http.Error(w, "Failed to sign presentation", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(presentation)
}

// CredentialsHandler returns all verifiable credentials currently held by the holder
func CredentialsHandler(w http.ResponseWriter, r *http.Request) {
	credentials := GetStoredCredentials()
	jsonResponse, err := json.Marshal(credentials)
	if err != nil {
		log.Printf("Error marshalling credentials: %v", err)
		http.Error(w, "Failed to retrieve credentials", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

func CreateAndSignPresentation(w http.ResponseWriter, r *http.Request) {
	var req PresentationRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Retrieve credentials using the VC IDs
	credentials := GetStoredCredentialsByIDs(req.VCIDs)

	// Create the Verifiable Presentation
	presentation := VerifiablePresentation{
		Context:              []string{"https://www.w3.org/2018/credentials/v1"},
		Type:                 []string{"VerifiablePresentation"},
		Holder:               req.HolderDID,
		VerifiableCredential: credentials,
	}

	// Sign the presentation using the holder's private key from HashiCorp Vault
	err := SignPresentation(&presentation, req.HolderDID)
	if err != nil {
		http.Error(w, "Failed to sign presentation", http.StatusInternalServerError)
		return
	}

	// Send the signed presentation to the presentation service
	err = SendPresentationToService(presentation)
	if err != nil {
		http.Error(w, "Failed to send presentation", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Presentation sent successfully"))
}

// Assuming storedCredentials is a global slice or a suitable storage mechanism
var storedCredentials []VerifiableCredential

// GetStoredCredentialsByIDs retrieves credentials based on the provided IDs.
func GetStoredCredentialsByIDs(vcIDs []string) []VerifiableCredential {
	var credentials []VerifiableCredential

	// Iterate over the provided vcIDs to find matching credentials
	for _, vcID := range vcIDs {
		for _, credential := range storedCredentials {
			if credential.ID == vcID {
				credentials = append(credentials, credential)
				break // Found the credential, move to the next ID
			}
		}
	}

	return credentials
}

// SignPresentation signs a Verifiable Presentation and returns the signed presentation
func SignPresentation(presentation *VerifiablePresentation, holderDID string) error {

	// Fetch the private key from HashiCorp Vault (pseudo-code, implement actual retrieval)
	privateKey, err := fetchPrivateKeyFromVault(holderDID)
	if err != nil {
		log.Println("failed to fetch private key: ", err)
		return errors.New("failed to fetch private key")
	}

	// Prepare the data to be signed
	dataToSign, err := json.Marshal(presentation)
	if err != nil {
		return errors.New("failed to marshal presentation for signing")
	}

	// Sign the data using Ed25519
	signature := ed25519.Sign(privateKey, dataToSign)

	// Construct the proof
	proof := Proof{
		Type:               "EcdsaSignature2019",                         // Adjust type based on your requirements
		Created:            "2024-09-29T20:41:23Z",                       // Use current time in production
		ProofValue:         base64.StdEncoding.EncodeToString(signature), // Base64 encoding of signature
		VerificationMethod: holderDID + "#keys-1",                        // Example of using a key ID, adjust as needed
		ProofPurpose:       "assertionMethod",
	}

	// Attach the proof to the presentation
	presentation.Proof = proof

	return nil
}

// fetchPrivateKeyFromVault retrieves the private key for the specified holder DID from HashiCorp Vault
func fetchPrivateKeyFromVault(holderDID string) (ed25519.PrivateKey, error) {

	client, err := getVaultClient()
	if err != nil {
		return nil, err
	}

	// Define the path to your private key in Vault
	log.Println("Holder DID: ", holderDID)
	secretPath := fmt.Sprintf("secret/data/dids/%s", holderDID) // Adjust this path as needed
	log.Println("Secret Path --> We stored the secrets here: ", secretPath)

	// Read the private key from Vault
	secret, err := client.Logical().Read(secretPath)
	if err != nil {
		log.Printf("Error reading secret from Vault: %v", err)
		return nil, err
	}

	// Check if the secret is found
	if secret == nil {
		return nil, errors.New("private key not found in Vault")
	}

	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid secret format: missing 'data'")
	}

	privateKeyBase64, ok := data["private_key"].(string)
	if !ok {
		return nil, errors.New("private key not found in secret data")
	}

	// Decode the base64-encoded private key
	privateKeyBytes, err := base64.StdEncoding.DecodeString(privateKeyBase64)
	if err != nil {
		return nil, errors.New("failed to decode private key")
	}

	return ed25519.PrivateKey(privateKeyBytes), nil // Return the parsed ECDSA private key
}

// SendPresentationToService sends the verifiable presentation to the presentation service
func SendPresentationToService(presentation VerifiablePresentation) error {
	// Convert the presentation to JSON
	presentationJSON, err := json.Marshal(presentation)
	if err != nil {
		log.Printf("Error marshalling presentation: %v", err)
		return err
	}

	// Define the URL of the presentation service (adjust as needed)
	presentationServiceURL := "http://presentation-service:8080/presentation" // Update this URL

	// Create a new POST request
	req, err := http.NewRequest(http.MethodPost, presentationServiceURL, bytes.NewBuffer(presentationJSON))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return err
	}

	// Set the appropriate headers
	req.Header.Set("Content-Type", "application/json")

	// Perform the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending request: %v", err)
		return err
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		log.Printf("Received non-200 response: %s", resp.Status)
		return err
	}

	log.Println("Presentation sent successfully")
	return nil
}

// Handle the presentation request from the client
func handlePresentationRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PresentationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Retrieve credentials using the VC IDs
	credentials := GetStoredCredentialsByIDs(req.VCIDs)

	// Create the Verifiable Presentation
	presentation := VerifiablePresentation{
		Context:              []string{"https://www.w3.org/2018/credentials/v1"},
		Type:                 []string{"VerifiablePresentation"},
		Holder:               req.HolderDID,
		VerifiableCredential: credentials,
	}

	// Sign the presentation
	err := SignPresentation(&presentation, req.HolderDID)
	if err != nil {
		log.Printf("Failed to sign presentation: %s", err)
		http.Error(w, "Failed to sign presentation", http.StatusInternalServerError)
		return
	}

	// Send the signed presentation to the presentation service
	err = SendPresentationToService(presentation)
	if err != nil {
		http.Error(w, "Failed to send presentation", http.StatusInternalServerError)
		return
	}

	// Respond to the client
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Presentation sent successfully"))
}
