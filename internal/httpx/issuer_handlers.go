package httpx

import (
	"crypto"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"time"

	"github.com/bradtumy/credential-service/internal/config"
	"github.com/bradtumy/credential-service/internal/domain"
	"github.com/bradtumy/credential-service/internal/keystore"
	"github.com/bradtumy/credential-service/internal/logging"
	"github.com/bradtumy/credential-service/internal/storage"
	"github.com/bradtumy/credential-service/internal/version"
)

// IssueRequest represents the request payload for issuing credentials.
type IssueRequest struct {
	SubjectDID string                 `json:"subject_did"`
	TTLSeconds int64                  `json:"ttl_seconds"`
	Claims     map[string]interface{} `json:"claims"`
	Format     string                 `json:"format"`
}

// IssueResponse represents the response payload after issuance.
type IssueResponse struct {
	Credential  string   `json:"credential"`
	Disclosures []string `json:"disclosures,omitempty"`
	Format      string   `json:"format,omitempty"`
	APIVersion  string   `json:"api_version"`
}

// DelegateRequest represents the payload for issuing delegated credentials.
type DelegateRequest struct {
	ParentCredential string   `json:"parent_credential"`
	DelegateDID      string   `json:"delegate_did"`
	Scope            []string `json:"scope"`
	TTLSeconds       int64    `json:"ttl_seconds"`
}

// RegisterIssuerRoutes wires issuer HTTP routes into the provided mux.
func RegisterIssuerRoutes(mux *http.ServeMux, store keystore.KeyStore, cfg config.IssuerConfig, auditStore storage.AuditStore) {
	// Key generation endpoint
	mux.HandleFunc("/v1/keys/generate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
			return
		}
		HandleKeygeneration(w, r)
	})

	mux.HandleFunc("/v1/credentials/issue", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
			return
		}

		var req IssueRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteAPIError(w, http.StatusBadRequest, "bad_request", "Invalid JSON payload: "+err.Error())
			return
		}

		// Validate DID format
		if err := domain.ValidateDID(req.SubjectDID); err != nil {
			WriteDIDError(w, err)
			return
		}

		// Validate TTL with policy-aware maximum
		ttl := time.Duration(req.TTLSeconds) * time.Second
		if ttl <= 0 {
			WriteValidationError(w, domain.NewFieldValidationError("ttl_seconds", "must be positive"))
			return
		}
		maxTTL := 24 * time.Hour
		if cfg.Policy.MaxTTLSeconds > 0 {
			policyLimit := time.Duration(cfg.Policy.MaxTTLSeconds) * time.Second
			if policyLimit < maxTTL {
				maxTTL = policyLimit
			}
		}
		if ttl > maxTTL {
			WriteValidationError(w, domain.NewFieldValidationError("ttl_seconds", "exceeds policy limit"))
			return
		}

		// Validate claims
		if req.Claims != nil {
			if err := domain.ValidateCredentialSubject(req.Claims); err != nil {
				WriteValidationError(w, err)
				return
			}
		} else {
			req.Claims = map[string]interface{}{}
		}
		if err := cfg.Policy.ValidateRequest(req.TTLSeconds, req.Claims); err != nil {
			WriteAPIError(w, http.StatusBadRequest, "policy_violation", err.Error())
			return
		}

		format := req.Format
		if format == "" {
			format = "jwt-vc"
		}

		tenantID := TenantIDFromContext(r.Context())
		if tenantID == "" {
			tenantID = cfg.DefaultTenantID
		}

		signer, err := store.GetSigningKey(tenantID)
		if err != nil {
			WriteAPIError(w, http.StatusInternalServerError, "keystore_error", err.Error())
			return
		}

		issuerDID, err := domain.DIDFromPublicKey(signer.Public())
		if err != nil {
			WriteAPIError(w, http.StatusInternalServerError, "did_error", err.Error())
			return
		}

		var (
			token       string
			disclosures []string
		)

		if format == "sd-jwt" {
			sdjwt, err := domain.IssueSDJWTCredential(issuerDID, req.SubjectDID, signer, ttl, req.Claims)
			if err != nil {
				logging.LogCredentialEvent(r.Context(), logging.AuditEventCredentialIssued, req.SubjectDID, issuerDID, "failure")
				WriteInternalError(w, "Failed to issue SD-JWT credential")
				return
			}
			token = sdjwt.Token
			disclosures = sdjwt.Disclosures
		} else {
			issuedToken, err := domain.IssueBasicCredentialWithFormat(issuerDID, req.SubjectDID, signer, ttl, req.Claims, format)
			if err != nil {
				// Log failed credential issuance
				logging.LogCredentialEvent(r.Context(), logging.AuditEventCredentialIssued, req.SubjectDID, issuerDID, "failure")

				var validationErr domain.ValidationError
				if errors.As(err, &validationErr) {
					WriteValidationError(w, err)
				} else {
					WriteInternalError(w, "Failed to issue credential")
				}
				return
			}
			token = issuedToken
		}

		auditEvent := logging.AuditEvent{
			EventType:  logging.AuditEventCredentialIssued,
			SubjectDID: req.SubjectDID,
			IssuerDID:  issuerDID,
			Outcome:    "success",
			Metadata: map[string]interface{}{
				"format":                 format,
				"ttl_seconds":            req.TTLSeconds,
				"claims_keys":            claimKeys(req.Claims),
				"policy_max_ttl_seconds": cfg.Policy.MaxTTLSeconds,
			},
		}
		auditEvent.Timestamp = time.Now().UTC()
		if auditStore != nil {
			_ = auditStore.InsertAuditEvent(r.Context(), auditEvent)
		}
		logging.LogAuditEvent(r.Context(), auditEvent)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(IssueResponse{Credential: token, Disclosures: disclosures, Format: format, APIVersion: version.APIVersion})
	})

	mux.HandleFunc("/v1/credentials/delegate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
			return
		}

		var req DelegateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteAPIError(w, http.StatusBadRequest, "bad_request", "Invalid JSON payload: "+err.Error())
			return
		}

		// Validate parent credential
		if req.ParentCredential == "" {
			WriteValidationError(w, domain.NewFieldValidationError("parent_credential", "is required"))
			return
		}

		// Validate delegate DID
		if err := domain.ValidateDID(req.DelegateDID); err != nil {
			WriteDIDError(w, err)
			return
		}

		// Validate scope
		if len(req.Scope) == 0 {
			WriteValidationError(w, domain.NewFieldValidationError("scope", "at least one scope is required"))
			return
		}

		// Validate TTL
		ttl := time.Duration(req.TTLSeconds) * time.Second
		if ttl <= 0 {
			WriteValidationError(w, domain.NewFieldValidationError("ttl_seconds", "must be positive"))
			return
		}

		tenantID := TenantIDFromContext(r.Context())
		if tenantID == "" {
			tenantID = cfg.DefaultTenantID
		}

		signer, err := store.GetSigningKey(tenantID)
		if err != nil {
			WriteAPIError(w, http.StatusInternalServerError, "keystore_error", err.Error())
			return
		}

		issuerDID, err := domain.DIDFromPublicKey(signer.Public())
		if err != nil {
			WriteAPIError(w, http.StatusInternalServerError, "did_error", err.Error())
			return
		}

		deps := domain.VerifierDependencies{ResolveIssuerPublicKey: func(issuer string) (crypto.PublicKey, error) {
			if issuer != issuerDID {
				return nil, domain.ErrUntrustedIssuer
			}
			return signer.Public(), nil
		}}

		now := time.Now()
		parentResult, err := domain.VerifyCredentialChain([]string{req.ParentCredential}, deps, domain.VerificationOptions{MaxDelegationDepth: 1}, now)
		if err != nil {
			status := http.StatusBadRequest
			switch {
			case errors.Is(err, domain.ErrUntrustedIssuer):
				status = http.StatusForbidden
			case errors.Is(err, domain.ErrExpiredCredential), errors.Is(err, domain.ErrInvalidSignature):
				status = http.StatusUnauthorized
			}
			WriteAPIError(w, status, "invalid_parent", err.Error())
			return
		}

		parentCred := parentResult.Credentials[0]
		parentScope := domain.ScopeFromClaims(parentCred.Claims)
		if !domain.IsScopeSubset(parentScope, req.Scope) {
			WriteAPIError(w, http.StatusBadRequest, "invalid_scope", "delegated scope must be within parent scope")
			return
		}

		expiresAt := now.Add(ttl)
		if !domain.IsTTLWithinParent(parentCred.ExpiresAt, expiresAt) {
			WriteAPIError(w, http.StatusBadRequest, "invalid_ttl", "delegated credential must expire before parent")
			return
		}

		claims := map[string]interface{}{
			"scope":     req.Scope,
			"parent_id": parentCred.ID,
		}
		if aud, ok := parentCred.Claims["aud"]; ok {
			claims["aud"] = aud
		}

		if err := cfg.Policy.ValidateRequest(req.TTLSeconds, claims); err != nil {
			WriteAPIError(w, http.StatusBadRequest, "policy_violation", err.Error())
			return
		}

		token, err := domain.IssueBasicCredentialWithFormat(issuerDID, req.DelegateDID, signer, ttl, claims, "jwt-vc")
		if err != nil {
			// Log failed delegation
			logging.LogAuditEvent(r.Context(), logging.AuditEvent{
				EventType:  logging.AuditEventDelegationCreated,
				SubjectDID: req.DelegateDID,
				IssuerDID:  issuerDID,
				Outcome:    "failure",
				Metadata: map[string]interface{}{
					"scope":       req.Scope,
					"parent_id":   parentCred.ID,
					"ttl_seconds": req.TTLSeconds,
				},
			})
			WriteAPIError(w, http.StatusInternalServerError, "issuance_error", err.Error())
			return
		}

		// Log successful delegation
		event := logging.AuditEvent{
			EventType:  logging.AuditEventDelegationCreated,
			SubjectDID: req.DelegateDID,
			IssuerDID:  issuerDID,
			Outcome:    "success",
			Metadata: map[string]interface{}{
				"scope":       req.Scope,
				"parent_id":   parentCred.ID,
				"ttl_seconds": req.TTLSeconds,
				"format":      "jwt-vc",
			},
		}
		event.Timestamp = time.Now().UTC()
		if auditStore != nil {
			_ = auditStore.InsertAuditEvent(r.Context(), event)
		}
		logging.LogAuditEvent(r.Context(), event)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(IssueResponse{Credential: token, APIVersion: version.APIVersion})
	})
}

func claimKeys(claims map[string]interface{}) []string {
	keys := make([]string, 0, len(claims))
	for k := range claims {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
