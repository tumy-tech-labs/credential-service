package httpx

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bradtumy/credential-service/internal/config"
	"github.com/bradtumy/credential-service/internal/domain"
	"github.com/bradtumy/credential-service/internal/keystore"
	"github.com/bradtumy/credential-service/internal/version"
)

// BootstrapRequest represents the payload for system bootstrap
type BootstrapRequest struct {
	RootAdminDID string `json:"root_admin_did"`
}

// BootstrapResponse represents the response from system bootstrap
type BootstrapResponse struct {
	IssuerDID           string    `json:"issuer_did"`
	RootAdminCredential string    `json:"root_admin_credential"`
	ExpiresAt          time.Time `json:"expires_at"`
	APIVersion         string    `json:"api_version"`
}

// RegisterBootstrapRoutes wires bootstrap HTTP routes into the provided mux
// WARNING: Bootstrap endpoint has no authentication - should be disabled in production
// or protected by network-level controls (VPN, private subnets, etc.)
func RegisterBootstrapRoutes(mux *http.ServeMux, store keystore.KeyStore, cfg config.IssuerConfig, trustRegistry domain.TrustRegistry) {
	mux.HandleFunc("/v1/setup/bootstrap", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		var req BootstrapRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteAPIError(w, http.StatusBadRequest, "bad_request", "invalid request payload")
			return
		}

		if req.RootAdminDID == "" {
			WriteAPIError(w, http.StatusBadRequest, "invalid_request", "root_admin_did is required")
			return
		}

		// Get or create issuer signing key
		tenantID := cfg.DefaultTenantID
		signer, err := store.GetSigningKey(tenantID)
		if err != nil {
			WriteAPIError(w, http.StatusInternalServerError, "keystore_error", err.Error())
			return
		}

		// Derive issuer DID from public key
		issuerDID, err := domain.DIDFromPublicKey(signer.Public())
		if err != nil {
			WriteAPIError(w, http.StatusInternalServerError, "did_error", err.Error())
			return
		}

		// Add issuer to trust registry if provided
		if trustRegistry != nil {
			if err := trustRegistry.AddTrustedIssuer(r.Context(), tenantID, issuerDID); err != nil {
				// Log error but don't fail - issuer might already exist
				fmt.Printf("Warning: failed to add issuer to trust registry: %v\n", err)
			}
		}

		// Create admin claims for root admin VC
		adminClaims := domain.CreateAdminClaims([]string{domain.SuperAdminRole}, map[string]interface{}{
			"bootstrap": true,
			"created_at": time.Now().UTC().Format(time.RFC3339),
		})

		// Issue the root admin credential with 6-month TTL
		expiresAt := time.Now().Add(domain.AdminVCTTL)
		rootAdminCredential, err := domain.IssueBasicCredential(
			issuerDID,
			req.RootAdminDID,
			signer,
			domain.AdminVCTTL,
			adminClaims,
		)
		if err != nil {
			WriteAPIError(w, http.StatusInternalServerError, "issuance_error", err.Error())
			return
		}

		// Return bootstrap information
		response := BootstrapResponse{
			IssuerDID:           issuerDID,
			RootAdminCredential: rootAdminCredential,
			ExpiresAt:          expiresAt,
			APIVersion:         version.APIVersion,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
}

