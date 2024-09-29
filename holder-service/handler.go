package main

import (
	"encoding/json"
	"log"
	"net/http"
)

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
	credentials := GetStoredCredentials()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(credentials)
}
