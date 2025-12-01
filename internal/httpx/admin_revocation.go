package httpx

import (
    "encoding/json"
    "net/http"

    "github.com/bradtumy/credential-service/internal/domain"
)

type RevocationRequest struct {
    CredentialID string `json:"credential_id"`
    Reason       string `json:"reason,omitempty"`
    Remove       bool   `json:"remove,omitempty"`
}

// RegisterAdminRevocationRoutes adds simple revocation management endpoints for demos.
func RegisterAdminRevocationRoutes(mux *http.ServeMux) {
    mux.HandleFunc("/admin/revocations", func(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case http.MethodPost:
            var req RevocationRequest
            if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CredentialID == "" {
                WriteAPIError(w, http.StatusBadRequest, "bad_request", "credential_id required")
                return
            }
            if req.Remove {
                domain.RemoveRevocation(req.CredentialID)
            } else {
                domain.AddRevocation(req.CredentialID, req.Reason)
            }
            w.Header().Set("Content-Type", "application/json")
            _ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
        default:
            WriteAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
        }
    })
}
