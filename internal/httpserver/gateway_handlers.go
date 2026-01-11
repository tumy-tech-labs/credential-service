package httpserver

import (
	"crypto"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/bradtumy/credential-service/internal/cache"
	"github.com/bradtumy/credential-service/internal/domain"
	"github.com/bradtumy/credential-service/internal/logging"
	"github.com/bradtumy/credential-service/internal/metrics"
	"github.com/bradtumy/credential-service/internal/policy"
	"github.com/bradtumy/credential-service/internal/ratelimit"
	"github.com/bradtumy/credential-service/internal/version"
)

// GatewayAuthorizeRequest is the payload expected by the gateway authorize endpoint.
type GatewayAuthorizeRequest struct {
	Credential       string   `json:"credential"`
	Credentials      []string `json:"credentials"`
	ExpectedAudience string   `json:"expected_audience"`
	WantSyntheticJWT bool     `json:"want_synthetic_jwt"`
	Resource         string   `json:"resource"`
	Action           string   `json:"action"`
}

// GatewayAuthorizeResponse is returned to gateways or reverse proxies.
type GatewayAuthorizeResponse struct {
	Allowed          bool                   `json:"allowed"`
	Subject          string                 `json:"subject,omitempty"`
	ActingOnBehalfOf string                 `json:"acting_on_behalf_of,omitempty"`
	DelegationDepth  int                    `json:"delegation_depth,omitempty"`
	Claims           map[string]interface{} `json:"claims,omitempty"`
	SyntheticJWT     string                 `json:"synthetic_jwt,omitempty"`
	Agent            *AgentContext          `json:"agent,omitempty"`
	TenantID         string                 `json:"tenant_id,omitempty"`
	PolicyID         *int64                 `json:"policy_id,omitempty"`
	ErrorCode        string                 `json:"error_code,omitempty"`
	Message          string                 `json:"message,omitempty"`
	Details          map[string]any         `json:"details,omitempty"`
	Reason           string                 `json:"reason,omitempty"`
	APIVersion       string                 `json:"api_version"`
}

// AgentContext surfaces on-behalf-of metadata for agent invocations.
type AgentContext struct {
	ActingOnBehalfOf string   `json:"acting_on_behalf_of"`
	DelegationDepth  int      `json:"delegation_depth"`
	Scope            []string `json:"scope,omitempty"`
}

const (
	maxCredentialLength = 8192
	maxFieldLength      = 512
)

// GatewayConfig encapsulates all dependencies for gateway route handlers.
type GatewayConfig struct {
	Resolver        func(string) (crypto.PublicKey, error)
	Registry        domain.TrustRegistry
	PolicyEngine    policy.Engine
	DefaultTenantID string
	SigningKey      crypto.Signer
	JWTIssuer       string
	DecisionCache   cache.DecisionCache
	Limiter         ratelimit.Limiter
	Metrics         metrics.GatewayMetrics
	Now             func() time.Time
}

// RegisterGatewayRoutes wires gateway-specific routes into the provided mux.
func RegisterGatewayRoutes(mux *http.ServeMux, cfg *GatewayConfig) {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}

	if cfg.PolicyEngine == nil {
		cfg.PolicyEngine = policy.NoOpEngine{}
	}

	if cfg.DecisionCache == nil {
		cfg.DecisionCache = cache.NoopDecisionCache{}
	}

	if cfg.Limiter == nil {
		cfg.Limiter = ratelimit.NoopLimiter{}
	}

	if cfg.Metrics == nil {
		cfg.Metrics = metrics.DefaultGatewayMetrics
	}

	mux.HandleFunc("/v1/gateway/authorize", func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		tenantID := TenantIDFromContext(r.Context())
		if tenantID == "" {
			tenantID = cfg.DefaultTenantID
		}
		if r.Method != http.MethodPost {
			writeGatewayError(w, tenantID, gatewayError{status: http.StatusMethodNotAllowed, code: "method_not_allowed", message: "method not allowed"})
			return
		}

		var req GatewayAuthorizeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeGatewayError(w, tenantID, gatewayError{status: http.StatusBadRequest, code: "invalid_request", message: "invalid request payload"})
			return
		}

		cfg.Metrics.IncAuthzRequest(tenantID)

		if err := validateGatewayRequest(req); err != nil {
			cfg.Metrics.IncAuthzDeny(tenantID, "invalid_request")
			writeGatewayError(w, tenantID, gatewayError{status: http.StatusBadRequest, code: "invalid_request", message: err.Error()})
			return
		}

		tokens := req.Credentials
		if len(tokens) == 0 && req.Credential != "" {
			tokens = []string{req.Credential}
		}

		rateKey := buildRateLimitKey(r.RemoteAddr, tenantID)
		allowed, err := cfg.Limiter.Allow(rateKey)
		if err != nil {
			logging.Logger.With("error", err.Error(), "tenant_id", tenantID).Warn("rate limit check error")
		}
		if !allowed {
			cfg.Metrics.IncAuthzDeny(tenantID, "rate_limited")
			writeGatewayError(w, tenantID, gatewayError{status: http.StatusTooManyRequests, code: "rate_limited", message: "rate limit exceeded"})
			return
		}

		cacheKey := buildCacheKey(tokens, tenantID, req.Resource, req.Action, req.ExpectedAudience, req.WantSyntheticJWT)
		if cached, ok := readCachedDecision(cfg.DecisionCache, cacheKey); ok {
			cfg.Metrics.IncAuthzCacheHit(tenantID)
			logGatewayDecision(started, tenantID, cached, true)
			writeGatewayDecision(w, cached)
			if cached.Allowed {
				cfg.Metrics.IncAuthzAllow(tenantID)
			} else {
				denyReason := cached.ErrorCode
				if denyReason == "" {
					denyReason = cached.Reason
				}
				cfg.Metrics.IncAuthzDeny(tenantID, denyReason)
			}
			return
		}
		cfg.Metrics.IncAuthzCacheMiss(tenantID)

		deps := domain.VerifierDependencies{ResolveIssuerPublicKey: func(issuer string) (crypto.PublicKey, error) {
			if cfg.Registry != nil {
				trusted, err := cfg.Registry.IsTrustedIssuer(r.Context(), tenantID, issuer)
				if err != nil {
					return nil, fmt.Errorf("trust lookup: %w", err)
				}
				if !trusted {
					return nil, domain.ErrUntrustedIssuer
				}
			}
			if cfg.Resolver == nil {
				return nil, fmt.Errorf("resolver not configured")
			}
			return cfg.Resolver(issuer)
		}}

		chainResult, err := domain.VerifyCredentialChain(tokens, deps, domain.VerificationOptions{ExpectedAudience: req.ExpectedAudience, MaxDelegationDepth: 3}, cfg.Now())
		if err != nil {
			gwErr := mapVerificationError(err)
			metrics.DefaultVerifierMetrics.IncVerificationFailure(gwErr.code)
			cfg.Metrics.IncAuthzDeny(tenantID, gwErr.code)
			resp := writeGatewayError(w, tenantID, gwErr)
			logGatewayDecision(started, tenantID, resp, false)
			return
		}

		decision := domain.BuildAuthzDecisionFromVerification(chainResult)
		evalInput := policy.EvaluationInput{
			TenantID:         tenantID,
			Subject:          decision.SubjectDID,
			ActingOnBehalfOf: decision.ActingOnBehalfOf,
			Scope:            domain.ScopeFromClaims(decision.Claims),
			Claims:           decision.Claims,
			Resource:         req.Resource,
			Action:           req.Action,
			Context: map[string]any{
				"delegation_depth": chainResult.DelegationDepth,
				"path":             r.URL.Path,
			},
		}

		policyResult, err := cfg.PolicyEngine.Evaluate(evalInput)
		if err != nil {
			gwErr := gatewayError{status: http.StatusInternalServerError, code: "policy_error", message: err.Error()}
			writeGatewayError(w, tenantID, gwErr)
			return
		}
		if !policyResult.Allow {
			cfg.Metrics.IncAuthzDeny(tenantID, "policy_denied")
			denyResp := GatewayAuthorizeResponse{Allowed: false, ErrorCode: "policy_denied", Message: policyResult.Reason, Details: map[string]any{"policy_id": policyResult.PolicyID}, Reason: "policy_denied", TenantID: tenantID, PolicyID: policyResult.PolicyID, APIVersion: version.APIVersion}
			logGatewayDecision(started, tenantID, denyResp, false)
			writeGatewayDecision(w, denyResp)
			return
		}
		var agentContext *AgentContext
		if len(chainResult.Credentials) > 1 {
			parent := chainResult.Credentials[len(chainResult.Credentials)-2]
			child := chainResult.Credentials[len(chainResult.Credentials)-1]
			ctx := domain.BuildOnBehalfOfContext(&parent, &child)
			agentContext = &AgentContext{
				ActingOnBehalfOf: ctx.ActingFor,
				DelegationDepth:  ctx.DelegationDepth,
				Scope:            ctx.Scope,
			}
		}

		response := GatewayAuthorizeResponse{
			Allowed:          decision.Allowed,
			Subject:          decision.SubjectDID,
			ActingOnBehalfOf: decision.ActingOnBehalfOf,
			DelegationDepth:  decision.DelegationDepth,
			Claims:           filterSafeClaims(decision.Claims),
			Reason:           policyResult.Reason,
			PolicyID:         policyResult.PolicyID,
			Agent:            agentContext,
			TenantID:         tenantID,
			APIVersion:       version.APIVersion,
		}

		if req.WantSyntheticJWT {
			jwt, err := domain.BuildSyntheticJWT(decision, cfg.SigningKey, cfg.JWTIssuer, 15*time.Minute)
			if err != nil {
				resp := GatewayAuthorizeResponse{Allowed: false, ErrorCode: "internal_error", Message: "failed to mint synthetic jwt", Details: map[string]any{"error": err.Error()}, Reason: "jwt_error", APIVersion: version.APIVersion, TenantID: tenantID}
				cfg.Metrics.IncAuthzDeny(tenantID, "jwt_error")
				writeGatewayDecision(w, resp)
				return
			}
			response.SyntheticJWT = jwt.Token
		}

		metrics.DefaultVerifierMetrics.IncVerificationSuccess("ok")
		cfg.Metrics.IncAuthzAllow(tenantID)
		ttl := computeDecisionTTL(chainResult, cfg.Now())
		persistDecision(cfg.DecisionCache, cacheKey, response, ttl)
		logGatewayDecision(started, tenantID, response, false)
		writeGatewayDecision(w, response)
	})
}

type gatewayError struct {
	status  int
	code    string
	message string
	details map[string]any
}

func writeGatewayDecision(w http.ResponseWriter, resp GatewayAuthorizeResponse) {
	writeGatewayResponse(w, statusForGatewayResponse(resp), resp)
}

func writeGatewayError(w http.ResponseWriter, tenantID string, err gatewayError) GatewayAuthorizeResponse {
	resp := GatewayAuthorizeResponse{
		Allowed:    false,
		ErrorCode:  err.code,
		Message:    err.message,
		Details:    err.details,
		Reason:     err.code,
		TenantID:   tenantID,
		APIVersion: version.APIVersion,
	}
	writeGatewayResponse(w, err.status, resp)
	return resp
}

func writeGatewayResponse(w http.ResponseWriter, status int, resp GatewayAuthorizeResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

func statusForGatewayResponse(resp GatewayAuthorizeResponse) int {
	if resp.Allowed {
		return http.StatusOK
	}

	switch resp.ErrorCode {
	case "invalid_request":
		return http.StatusBadRequest
	case "rate_limited":
		return http.StatusTooManyRequests
	case "policy_denied":
		return http.StatusForbidden
	case "credential_expired", "invalid_credential", "delegation_invalid", "issuer_not_trusted", "credential_revoked", "tenant_mismatch":
		return http.StatusUnauthorized
	case "policy_error", "internal_error":
		return http.StatusInternalServerError
	}

	if resp.Reason == "jwt_error" {
		return http.StatusInternalServerError
	}

	return http.StatusForbidden
}

func mapVerificationError(err error) gatewayError {
	switch {
	case errors.Is(err, domain.ErrExpiredCredential):
		return gatewayError{status: http.StatusUnauthorized, code: "credential_expired", message: "credential has expired"}
	case errors.Is(err, domain.ErrUntrustedIssuer):
		return gatewayError{status: http.StatusUnauthorized, code: "issuer_not_trusted", message: "issuer is not trusted"}
	case errors.Is(err, domain.ErrInvalidSignature), errors.Is(err, domain.ErrInvalidToken), errors.Is(err, domain.ErrIssuedInFuture), errors.Is(err, domain.ErrUnexpectedAudience), errors.Is(err, domain.ErrMissingDisclosure), errors.Is(err, domain.ErrInvalidDisclosure):
		return gatewayError{status: http.StatusUnauthorized, code: "invalid_credential", message: err.Error()}
	case errors.Is(err, domain.ErrDelegationDepth), errors.Is(err, domain.ErrDelegationScope), errors.Is(err, domain.ErrDelegationTTL), errors.Is(err, domain.ErrInvalidDelegation):
		return gatewayError{status: http.StatusUnauthorized, code: "delegation_invalid", message: err.Error()}
	default:
		return gatewayError{status: http.StatusInternalServerError, code: "internal_error", message: "credential verification failed", details: map[string]any{"error": err.Error()}}
	}
}

func validateGatewayRequest(req GatewayAuthorizeRequest) error {
	tokens := req.Credentials
	if len(tokens) == 0 && req.Credential != "" {
		tokens = []string{req.Credential}
	}

	if len(tokens) == 0 {
		return fmt.Errorf("credential is required")
	}

	if len(req.Resource) == 0 {
		return fmt.Errorf("resource is required")
	}
	if len(req.Action) == 0 {
		return fmt.Errorf("action is required")
	}

	if len(req.ExpectedAudience) > maxFieldLength || len(req.Resource) > maxFieldLength || len(req.Action) > maxFieldLength {
		return fmt.Errorf("input fields too long")
	}

	for _, token := range tokens {
		if token == "" {
			return fmt.Errorf("credential cannot be empty")
		}
		if len(token) > maxCredentialLength {
			return fmt.Errorf("credential too large")
		}
	}

	return nil
}

func filterSafeClaims(claims map[string]interface{}) map[string]interface{} {
	if claims == nil {
		return nil
	}
	safe := map[string]interface{}{}
	if scope, ok := claims["scope"]; ok {
		safe["scope"] = scope
	}
	if roles, ok := claims["roles"]; ok {
		safe["roles"] = roles
	}
	if len(safe) == 0 {
		return nil
	}
	return safe
}

func buildCacheKey(tokens []string, tenantID, resource, action, expectedAudience string, wantSynthetic bool) string {
	h := sha256.New()
	for _, token := range tokens {
		_, _ = h.Write([]byte(token))
	}
	_, _ = h.Write([]byte(tenantID))
	_, _ = h.Write([]byte(resource))
	_, _ = h.Write([]byte(action))
	_, _ = h.Write([]byte(expectedAudience))
	if wantSynthetic {
		h.Write([]byte("synthetic"))
	}
	return fmt.Sprintf("gw:%x", h.Sum(nil))
}

func readCachedDecision(decisionCache cache.DecisionCache, key string) (GatewayAuthorizeResponse, bool) {
	if decisionCache == nil {
		return GatewayAuthorizeResponse{}, false
	}

	data, ok, err := decisionCache.Get(key)
	if err != nil || !ok {
		return GatewayAuthorizeResponse{}, false
	}

	var resp GatewayAuthorizeResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return GatewayAuthorizeResponse{}, false
	}
	return resp, true
}

func persistDecision(decisionCache cache.DecisionCache, key string, resp GatewayAuthorizeResponse, ttl time.Duration) {
	if decisionCache == nil || ttl <= 0 {
		return
	}
	payload, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_ = decisionCache.Set(key, payload, ttl)
}

func computeDecisionTTL(chainResult *domain.DelegationChainResult, now time.Time) time.Duration {
	defaultTTL := 60 * time.Second
	if chainResult == nil || len(chainResult.Credentials) == 0 {
		return defaultTTL
	}
	leaf := chainResult.Credentials[len(chainResult.Credentials)-1]
	if leaf.ExpiresAt.IsZero() {
		return defaultTTL
	}
	remaining := leaf.ExpiresAt.Sub(now)
	if remaining <= 0 {
		return 0
	}
	if remaining < defaultTTL {
		return remaining
	}
	return defaultTTL
}

func buildRateLimitKey(remoteAddr, tenantID string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	return fmt.Sprintf("%s:%s", tenantID, host)
}

func logGatewayDecision(started time.Time, tenantID string, resp GatewayAuthorizeResponse, cacheHit bool) {
	latency := time.Since(started)
	logging.Logger.With(
		"tenant_id", tenantID,
		"subject", resp.Subject,
		"acting_on_behalf_of", resp.ActingOnBehalfOf,
		"delegation_depth", resp.DelegationDepth,
		"allowed", resp.Allowed,
		"cache_hit", cacheHit,
		"latency_ms", latency.Milliseconds(),
	).Info("gateway_authorize")
}
