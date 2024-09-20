package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Presentation struct {
	ID               string                 `json:"id"`
	CredentialID     string                 `json:"credentialId"`
	HolderDID        string                 `json:"holderDID"`
	PresentationData map[string]interface{} `json:"presentationData"`
	ProcessingID     string                 `json:"processingId"`
}

func handleVerification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var presentation Presentation
	if err := json.NewDecoder(r.Body).Decode(&presentation); err != nil {
		http.Error(w, "Invalid presentation format", http.StatusBadRequest)
		return
	}

	// Implement your verification logic here
	err := verifyPresentation(presentation)
	if err != nil {
		log.Printf("Verification Error: %s", err)
		http.Error(w, "Verification failed", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "verified", "presentationId": presentation.ID})
}

func verifyPresentation(presentation Presentation) error {
	// Example verification logic (replace with real logic)
	if presentation.CredentialID == "" || presentation.HolderDID == "" {
		return fmt.Errorf("invalid presentation data")
	}

	// Implement further checks (e.g., checking against a database, validating signatures)
	// For example, check if the presentation exists in a database or if the credentials are valid.

	return nil
}

func main() {
	http.HandleFunc("/verify", handleVerification)
	port := "8080"
	log.Printf("Verification Service running on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
