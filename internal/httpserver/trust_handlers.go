package httpserver

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/bradtumy/credential-service/internal/domain"
)

// AddTrustedIssuerRequest represents the request to add a trusted issuer
type AddTrustedIssuerRequest struct {
	IssuerDID string `json:"issuer_did"`
}

// AddTrustedIssuerResponse represents the response after adding a trusted issuer
type AddTrustedIssuerResponse struct {
	Success   bool   `json:"success"`
	IssuerDID string `json:"issuer_did"`
	Message   string `json:"message"`
}

// RegisterTrustRegistryRoutes registers HTTP routes for trust registry management
// WARNING: These endpoints have no authentication in development mode
func RegisterTrustRegistryRoutes(mux *http.ServeMux, trustRegistry domain.TrustRegistry, defaultTenantID string) {
	// Add trusted issuer endpoint
	mux.HandleFunc("/v1/trust/issuers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		var req AddTrustedIssuerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteAPIError(w, http.StatusBadRequest, "bad_request", "invalid request payload")
			return
		}

		if req.IssuerDID == "" {
			WriteAPIError(w, http.StatusBadRequest, "invalid_request", "issuer_did is required")
			return
		}

		// Add issuer to trust registry
		if err := trustRegistry.AddTrustedIssuer(r.Context(), defaultTenantID, req.IssuerDID); err != nil {
			log.Printf("Failed to add trusted issuer %s: %v", req.IssuerDID, err)
			WriteAPIError(w, http.StatusInternalServerError, "trust_registry_error", err.Error())
			return
		}

		log.Printf("Added trusted issuer: %s", req.IssuerDID)

		response := AddTrustedIssuerResponse{
			Success:   true,
			IssuerDID: req.IssuerDID,
			Message:   "Issuer added to trust registry",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	})

	// List trusted issuers endpoint
	mux.HandleFunc("/v1/trust/issuers/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		// Note: This requires adding a ListTrustedIssuers method to the TrustRegistry interface
		// For now, return a placeholder response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "List endpoint not yet implemented - check IsTrustedIssuer for specific DIDs",
		})
	})
}
