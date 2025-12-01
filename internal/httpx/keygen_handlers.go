package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/bradtumy/credential-service/internal/crypto"
	"github.com/bradtumy/credential-service/internal/logging"
)

// KeygenRequest represents the request to generate a new DID.
type KeygenRequest struct {
	Algorithm        string `json:"algorithm"`          // EdDSA or ES256
	ReturnPrivateKey bool   `json:"return_private_key"` // Whether to return the private key in response
}

// KeygenResponse represents the generated DID and optional key material.
type KeygenResponse struct {
	DID           string `json:"did"`
	Algorithm     string `json:"algorithm"`
	PublicJWK     string `json:"public_jwk"`
	PrivateKeyPEM string `json:"private_key_pem,omitempty"` // Only included if requested
}

// HandleKeygeneration handles POST /v1/keys/generate requests.
// It generates a new DID with embedded public key and optionally returns the private key.
//
// Security Note: Returning private keys over HTTP is intended for development/testing only.
// In production, consider:
// - Only returning the DID (not the private key)
// - Using TLS for all connections
// - Implementing proper key custody (vault, HSM, etc.)
// - Rate limiting to prevent abuse
func HandleKeygeneration(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var req KeygenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logging.Logger.Warn("Invalid keygen request", "error", err)
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Default to EdDSA if not specified
	if req.Algorithm == "" {
		req.Algorithm = "EdDSA"
	}

	// Validate algorithm
	if req.Algorithm != "EdDSA" && req.Algorithm != "ES256" {
		logging.Logger.Warn("Unsupported algorithm requested", "algorithm", req.Algorithm)
		http.Error(w, `{"error":"unsupported algorithm, use EdDSA or ES256"}`, http.StatusBadRequest)
		return
	}

	// Generate DID and key pair
	keypair, err := crypto.GenerateDIDJWK(req.Algorithm)
	if err != nil {
		logging.Logger.Error("Failed to generate DID", "error", err, "algorithm", req.Algorithm)
		http.Error(w, `{"error":"failed to generate DID"}`, http.StatusInternalServerError)
		return
	}

	// Build response
	resp := KeygenResponse{
		DID:       keypair.DID,
		Algorithm: keypair.Algorithm,
		PublicJWK: string(keypair.PublicJWK),
	}

	// Include private key only if explicitly requested
	if req.ReturnPrivateKey {
		resp.PrivateKeyPEM = string(keypair.PrivateKeyPEM)
		logging.Logger.Warn("Private key returned in API response - ensure TLS is enabled", "did", keypair.DID)
	}

	logging.Logger.Info("Generated new DID", 
		"did", keypair.DID, 
		"algorithm", keypair.Algorithm,
		"private_key_returned", req.ReturnPrivateKey)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logging.Logger.Error("Failed to encode response", "error", err)
	}
}
