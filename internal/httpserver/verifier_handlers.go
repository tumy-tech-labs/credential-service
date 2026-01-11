package httpserver

import (
	"crypto"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bradtumy/credential-service/internal/domain"
	"github.com/bradtumy/credential-service/internal/logging"
	"github.com/bradtumy/credential-service/internal/metrics"
	"github.com/bradtumy/credential-service/internal/version"
)

// VerifyRequest represents a verifier request payload.
type VerifyRequest struct {
	Credential       string   `json:"credential"`
	Credentials      []string `json:"credentials"`
	ExpectedAudience string   `json:"expected_audience"`
	Disclosures      []string `json:"disclosures"`
	Format           string   `json:"format"`
}

// VerifyResponse represents the verifier response payload.
type VerifyResponse struct {
	Valid            bool      `json:"valid"`
	Subject          string    `json:"subject"`
	Issuer           string    `json:"issuer"`
	ExpiresAt        time.Time `json:"expires_at"`
	ActingOnBehalfOf string    `json:"acting_on_behalf_of,omitempty"`
	DelegationDepth  int       `json:"delegation_depth"`
	APIVersion       string    `json:"api_version"`
}

// RegisterVerifierRoutes wires verifier HTTP routes into the provided mux.
func RegisterVerifierRoutes(mux *http.ServeMux, resolver func(string) (crypto.PublicKey, error), registry domain.TrustRegistry, defaultTenantID string, verifierMetrics metrics.VerifierMetrics, now func() time.Time) {
	if now == nil {
		now = time.Now
	}

	if verifierMetrics == nil {
		verifierMetrics = metrics.DefaultVerifierMetrics
	}

	mux.HandleFunc("/v1/credentials/verify", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		var req VerifyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteAPIError(w, http.StatusBadRequest, "bad_request", "invalid request payload")
			return
		}

		tokens := req.Credentials
		if len(tokens) == 0 && req.Credential != "" {
			tokens = []string{req.Credential}
		}

		if req.Format == "sd-jwt" || len(req.Disclosures) > 0 {
			if req.Credential == "" {
				WriteAPIError(w, http.StatusBadRequest, "invalid_request", "credential is required for sd-jwt verification")
				return
			}
			combined := req.Credential
			if len(req.Disclosures) > 0 {
				combined = combined + "~" + strings.Join(req.Disclosures, "~")
			}
			tokens = []string{combined}
		}

		if len(tokens) == 0 {
			WriteAPIError(w, http.StatusBadRequest, "invalid_request", "credential is required")
			return
		}

		tenantID := TenantIDFromContext(r.Context())
		if tenantID == "" {
			tenantID = defaultTenantID
		}

		deps := domain.VerifierDependencies{ResolveIssuerPublicKey: func(issuer string) (crypto.PublicKey, error) {
			if registry != nil {
				trusted, err := registry.IsTrustedIssuer(r.Context(), tenantID, issuer)
				if err != nil {
					return nil, fmt.Errorf("trust lookup: %w", err)
				}
				if !trusted {
					return nil, domain.ErrUntrustedIssuer
				}
			}
			if resolver == nil {
				return nil, fmt.Errorf("resolver not configured")
			}
			return resolver(issuer)
		}}

		chainResult, err := domain.VerifyCredentialChain(tokens, deps, domain.VerificationOptions{ExpectedAudience: req.ExpectedAudience, MaxDelegationDepth: 3}, now())
		if err != nil {
			reason := "verification_failed"
			switch {
			case errors.Is(err, domain.ErrUntrustedIssuer):
				reason = "untrusted_issuer"
				WriteAPIError(w, http.StatusForbidden, reason, err.Error())
			case errors.Is(err, domain.ErrExpiredCredential), errors.Is(err, domain.ErrInvalidSignature), errors.Is(err, domain.ErrUnexpectedAudience), errors.Is(err, domain.ErrIssuedInFuture):
				reason = "invalid_credential"
				WriteAPIError(w, http.StatusUnauthorized, reason, err.Error())
			case errors.Is(err, domain.ErrInvalidDisclosure), errors.Is(err, domain.ErrMissingDisclosure):
				reason = "invalid_disclosure"
				WriteAPIError(w, http.StatusBadRequest, reason, err.Error())
			default:
				WriteAPIError(w, http.StatusBadRequest, reason, err.Error())
			}

			// Log failed verification attempt
			if len(tokens) > 0 {
				logging.LogAuditEvent(r.Context(), logging.AuditEvent{
					EventType: logging.AuditEventCredentialVerified,
					Outcome:   "failure",
					Reason:    reason,
					Metadata: map[string]interface{}{
						"expected_audience": req.ExpectedAudience,
						"credential_count":  len(tokens),
						"error":             err.Error(),
					},
				})
			}

			verifierMetrics.IncVerificationFailure(reason)
			return
		}

		leaf := chainResult.Credentials[len(chainResult.Credentials)-1]
		actingOnBehalfOf := leaf.Subject
		if chainResult.DelegationDepth > 0 {
			actingOnBehalfOf = chainResult.RootDelegator
		}

		// Log successful verification
		logging.LogAuditEvent(r.Context(), logging.AuditEvent{
			EventType:  logging.AuditEventCredentialVerified,
			SubjectDID: leaf.Subject,
			IssuerDID:  leaf.Issuer,
			Outcome:    "success",
			Metadata: map[string]interface{}{
				"expected_audience":   req.ExpectedAudience,
				"credential_count":    len(tokens),
				"delegation_depth":    chainResult.DelegationDepth,
				"acting_on_behalf_of": actingOnBehalfOf,
				"expires_at":          leaf.ExpiresAt.Format(time.RFC3339),
				"format":              leaf.Format,
			},
		})

		verifierMetrics.IncVerificationSuccess("ok")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(VerifyResponse{
			Valid:            true,
			Subject:          leaf.Subject,
			Issuer:           leaf.Issuer,
			ExpiresAt:        leaf.ExpiresAt,
			ActingOnBehalfOf: actingOnBehalfOf,
			DelegationDepth:  chainResult.DelegationDepth,
			APIVersion:       version.APIVersion,
		})
	})
}
