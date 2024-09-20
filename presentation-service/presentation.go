// presentation-service/presentation.go

package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

// CredentialPresentation represents the structure of a credential presentation
type CredentialPresentation struct {
	CredentialID     string                 `json:"credentialId"`
	HolderDID        string                 `json:"holderDID"`
	PresentationData map[string]interface{} `json:"presentationData"`
}

// handlePresentation handles incoming credential presentations
func handlePresentation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var presentation CredentialPresentation
	if err := json.NewDecoder(r.Body).Decode(&presentation); err != nil {
		http.Error(w, "Invalid presentation format", http.StatusBadRequest)
		return
	}

	// Process the presentation (e.g., verify the credentials)
	err := processPresentation(presentation)
	if err != nil {
		log.Printf("Processing Error %s", err)
		http.Error(w, "Failed to process presentation", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// processPresentation verifies the provided credential presentation
func processPresentation(presentation CredentialPresentation) error {
	// Logic to verify the credential presentation
	// You may need to interact with the database or other services to verify the credential
	log.Printf("Processing presentation for credential ID %s", presentation.CredentialID)

	// Example verification (replace with real logic)
	if presentation.CredentialID == "" || presentation.HolderDID == "" {
		return errors.New("invalid presentation data")
	}

	// Verification logic goes here...

	return nil
}

func main() {

	http.HandleFunc("/presentations", handlePresentation)

	port := "8080" // Or any port you choose for the presentation service
	log.Printf("Presentation Service running on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
