// handlers.go

package main

import (
	"net/http"
)

// Handler for anchoring a DID
func AnchorDIDHandler(w http.ResponseWriter, r *http.Request) {
	// Extract DID from request (placeholder logic, adapt as needed)
	did := r.URL.Query().Get("did")

	adapter := NewBlockchainAdapter() // Use the factory to create the adapter
	txID, err := adapter.AnchorDID(did)
	if err != nil {
		http.Error(w, "Failed to anchor DID: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("DID anchored successfully. Transaction ID: " + txID))
}

// Handler for anchoring a credential
func AnchorCredentialHandler(w http.ResponseWriter, r *http.Request) {
	// Extract VC from request (placeholder logic, adapt as needed)
	vc := r.URL.Query().Get("vc")

	adapter := NewBlockchainAdapter() // Use factory to create the adapter
	txID, err := adapter.AnchorCredential(vc)
	if err != nil {
		http.Error(w, "Failed to anchor credential: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Credential anchored successfully. Transaction ID: " + txID))
}
