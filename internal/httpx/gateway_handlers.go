package httpx

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
	Reason           string                 `json:"reason,omitempty"`
	SyntheticJWT     string                 `json:"synthetic_jwt,omitempty"`
	Agent            *AgentContext          `json:"agent,omitempty"`
	TenantID         string                 `json:"tenant_id,omitempty"`
	PolicyID         *int64                 `json:"policy_id,omitempty"`
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
		if r.Method != http.MethodPost {
			WriteAPIError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		var req GatewayAuthorizeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteAPIError(w, http.StatusBadRequest, "bad_request", "invalid request payload")
			return
		}

		tenantID := TenantIDFromContext(r.Context())
		if tenantID == "" {
			tenantID = cfg.DefaultTenantID
		}

		cfg.Metrics.IncAuthzRequest(tenantID)

		if err := validateGatewayRequest(req); err != nil {
			cfg.Metrics.IncAuthzDeny(tenantID, "invalid_request")
			WriteAPIError(w, http.StatusBadRequest, "invalid_request", err.Error())
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
			WriteAPIError(w, http.StatusTooManyRequests, "rate_limited", "rate limit exceeded")
			return
		}

		cacheKey := buildCacheKey(tokens, tenantID, req.Resource, req.Action, req.ExpectedAudience, req.WantSyntheticJWT)
		if cached, ok := readCachedDecision(cfg.DecisionCache, cacheKey); ok {
			cfg.Metrics.IncAuthzCacheHit(tenantID)
			logGatewayDecision(started, tenantID, cached, true)
			writeDecision(w, cached)
			if cached.Allowed {
				cfg.Metrics.IncAuthzAllow(tenantID)
			} else {
				cfg.Metrics.IncAuthzDeny(tenantID, cached.Reason)
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
			reason := mapVerificationErrorToReason(err)
			metrics.DefaultVerifierMetrics.IncVerificationFailure(reason)
			resp := GatewayAuthorizeResponse{Allowed: false, Reason: reason, APIVersion: version.APIVersion, TenantID: tenantID}
			cfg.Metrics.IncAuthzDeny(tenantID, reason)
			logGatewayDecision(started, tenantID, resp, false)
			writeDecision(w, resp)
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
			WriteAPIError(w, http.StatusInternalServerError, "policy_error", err.Error())
			return
		}
		if !policyResult.Allow {
			cfg.Metrics.IncAuthzDeny(tenantID, "policy_denied")
			denyResp := GatewayAuthorizeResponse{Allowed: false, Reason: "policy_denied", TenantID: tenantID, PolicyID: policyResult.PolicyID, APIVersion: version.APIVersion}
			logGatewayDecision(started, tenantID, denyResp, false)
			writeDecision(w, denyResp)
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
				resp := GatewayAuthorizeResponse{Allowed: false, Reason: "jwt_error", APIVersion: version.APIVersion, TenantID: tenantID}
				cfg.Metrics.IncAuthzDeny(tenantID, "jwt_error")
				writeDecision(w, resp)
				return
			}
			response.SyntheticJWT = jwt.Token
		}

		metrics.DefaultVerifierMetrics.IncVerificationSuccess("ok")
		cfg.Metrics.IncAuthzAllow(tenantID)
		ttl := computeDecisionTTL(chainResult, cfg.Now())
		persistDecision(cfg.DecisionCache, cacheKey, response, ttl)
		logGatewayDecision(started, tenantID, response, false)
		writeDecision(w, response)
	})
}

func writeDecision(w http.ResponseWriter, resp GatewayAuthorizeResponse) {
	w.Header().Set("Content-Type", "application/json")
	status := http.StatusOK
	if !resp.Allowed {
		status = http.StatusForbidden
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

func mapVerificationErrorToReason(err error) string {
	switch {
	case errors.Is(err, domain.ErrExpiredCredential):
		return "expired_credential"
	case errors.Is(err, domain.ErrUntrustedIssuer):
		return "untrusted_issuer"
	case errors.Is(err, domain.ErrInvalidSignature):
		return "invalid_signature"
	case errors.Is(err, domain.ErrUnexpectedAudience):
		return "unexpected_audience"
	case errors.Is(err, domain.ErrDelegationDepth):
		return "delegation_depth_exceeded"
	case errors.Is(err, domain.ErrDelegationScope):
		return "invalid_scope"
	case errors.Is(err, domain.ErrDelegationTTL):
		return "invalid_ttl"
	default:
		return "verification_failed"
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
