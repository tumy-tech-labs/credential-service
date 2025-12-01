package httpx

import (
	"context"
	"crypto"
	"net/http"
	"strings"
	"time"

	"github.com/bradtumy/credential-service/internal/domain"
)

// AdminAuthMiddleware creates middleware that requires valid admin VCs
func AdminAuthMiddleware(resolver func(string) (crypto.PublicKey, error), trustRegistry domain.TrustRegistry, defaultTenantID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract Bearer token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				WriteAPIError(w, http.StatusUnauthorized, "missing_authorization", "Authorization header required")
				return
			}

			if !strings.HasPrefix(authHeader, "Bearer ") {
				WriteAPIError(w, http.StatusUnauthorized, "invalid_authorization", "Authorization must be Bearer token")
				return
			}

			credential := strings.TrimPrefix(authHeader, "Bearer ")
			if credential == "" {
				WriteAPIError(w, http.StatusUnauthorized, "missing_credential", "Bearer token is empty")
				return
			}

			// Get tenant ID for verification
			tenantID := TenantIDFromContext(r.Context())
			if tenantID == "" {
				tenantID = defaultTenantID
			}

			// Create verifier dependencies
			deps := domain.VerifierDependencies{
				ResolveIssuerPublicKey: func(issuer string) (crypto.PublicKey, error) {
					if trustRegistry != nil {
						trusted, err := trustRegistry.IsTrustedIssuer(r.Context(), tenantID, issuer)
						if err != nil {
							return nil, err
						}
						if !trusted {
							return nil, domain.ErrUntrustedIssuer
						}
					}
					return resolver(issuer)
				},
			}

			// Verify the credential
			chainResult, err := domain.VerifyCredentialChain([]string{credential}, deps, domain.VerificationOptions{MaxDelegationDepth: 1}, time.Now())
			if err != nil {
				WriteAPIError(w, http.StatusUnauthorized, "invalid_credential", err.Error())
				return
			}

			if len(chainResult.Credentials) == 0 {
				WriteAPIError(w, http.StatusUnauthorized, "no_credentials", "No valid credentials found")
				return
			}

			// Get the leaf credential claims
			leafCredential := chainResult.Credentials[len(chainResult.Credentials)-1]
			
			// Validate admin claims
			if err := domain.ValidateAdminVC(leafCredential.Claims, domain.SuperAdminRole); err != nil {
				if err == domain.ErrNotAdminVC {
					WriteAPIError(w, http.StatusForbidden, "admin_credential_required", "Admin credential required")
				} else {
					WriteAPIError(w, http.StatusForbidden, "insufficient_privileges", err.Error())
				}
				return
			}

			// Add admin context to request
			ctx := context.WithValue(r.Context(), "admin_subject", leafCredential.Subject)
			ctx = context.WithValue(ctx, "admin_claims", leafCredential.Claims)
			
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdminSubjectFromContext extracts the admin subject from request context
func AdminSubjectFromContext(ctx context.Context) string {
	if subject, ok := ctx.Value("admin_subject").(string); ok {
		return subject
	}
	return ""
}

// AdminClaimsFromContext extracts admin claims from request context
func AdminClaimsFromContext(ctx context.Context) map[string]interface{} {
	if claims, ok := ctx.Value("admin_claims").(map[string]interface{}); ok {
		return claims
	}
	return nil
}