package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Handler for verifying the credential presentation
func VerifyCredentialHandler(w http.ResponseWriter, r *http.Request) {
	var presentation VerifiableCredential

	// Decode the incoming JSON credential presentation
	err := json.NewDecoder(r.Body).Decode(&presentation)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Verify the credential
	isValid, err := VerifyCredential(presentation)
	if err != nil || !isValid {
		http.Error(w, "Credential verification failed", http.StatusBadRequest)
		return
	}

	// Respond with success if verification passes
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Credential verified successfully")
}
