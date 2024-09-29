package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// PresentationRequest structure for the incoming presentation request payload
type PresentationRequest struct {
	HolderDID string   `json:"holderDid"`
	VCIDs     []string `json:"vcIds"`
}

// VerifiablePresentation structure for W3C Verifiable Presentation
type VerifiablePresentation struct {
	Context              []string               `json:"@context"`
	Type                 []string               `json:"type"`
	Holder               string                 `json:"holder"`
	VerifiableCredential []VerifiableCredential `json:"verifiableCredential"`
}

// Handler function for handling verifiable presentation requests
func handlePresentationRequest(w http.ResponseWriter, r *http.Request) {
	log.Println("Got here.")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PresentationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Retrieve the verifiable credentials by their IDs (placeholder)
	credentials := retrieveVerifiableCredentials(req.VCIDs)

	// Create verifiable presentation
	presentation := VerifiablePresentation{
		Context:              []string{"https://www.w3.org/2018/credentials/v1"},
		Type:                 []string{"VerifiablePresentation"},
		Holder:               req.HolderDID,
		VerifiableCredential: credentials,
	}

	// Serialize the presentation to JSON and send response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(presentation); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Failed to create presentation", http.StatusInternalServerError)
		return
	}

	log.Printf("Presentation created successfully for holder DID: %v", req.HolderDID)
}

// Placeholder for retrieving verifiable credentials
func retrieveVerifiableCredentials(vcIDs []string) []VerifiableCredential {
	// Retrieve the credentials from the database or another service
	// Placeholder logic: returning empty credentials for now
	var credentials []VerifiableCredential
	for _, id := range vcIDs {
		credentials = append(credentials, VerifiableCredential{
			ID:   id,
			Type: []string{"VerifiableCredential"},
		})
	}
	return credentials
}
