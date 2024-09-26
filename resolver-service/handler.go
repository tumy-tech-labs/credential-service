package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
)

func resolveDIDHandler(w http.ResponseWriter, r *http.Request) {
	did := r.URL.Query().Get("did")
	if did == "" {
		http.Error(w, "Missing DID in query parameter", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	// Resolve the DID
	didDocument, err := resolveDID(ctx, did)
	if err != nil {
		log.Printf("Error resolving DID: %v", err)
		http.Error(w, "Failed to resolve DID", http.StatusInternalServerError)
		return
	}

	// Respond with the DID Document
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(didDocument)
}
