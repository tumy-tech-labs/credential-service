package sdk

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// AgentClient wraps the core SDK client with agent-specific helpers.
type AgentClient struct {
	SDK *Client
}

// AgentSession captures a delegated credential usable by an agent.
type AgentSession struct {
	AgentDID    string
	AccessToken string
	ExpiresAt   time.Time
	ParentToken string
	Scope       []string
}

// StartAgentSession mints a delegated credential for an agent using a parent token.
func (a *AgentClient) StartAgentSession(ctx context.Context, parentToken string, agentDID string, scope []string, ttl time.Duration) (*AgentSession, error) {
	if a == nil || a.SDK == nil {
		return nil, errors.New("sdk client is required")
	}
	if parentToken == "" {
		return nil, errors.New("parent token is required")
	}
	if agentDID == "" {
		return nil, errors.New("agent did is required")
	}

	parent, err := decodeCredentialPayload(parentToken)
	if err != nil {
		return nil, fmt.Errorf("parse parent token: %w", err)
	}

	if !isScopeSubset(parent.Scope, scope) {
		return nil, fmt.Errorf("scope must be subset of parent")
	}

	ttl = clampTTL(ttl, parent.ExpiresAt)
	if ttl <= 0 {
		return nil, fmt.Errorf("ttl must be positive and within parent")
	}

	resp, err := a.SDK.DelegateCredential(ctx, DelegateRequest{
		ParentCredential: parentToken,
		DelegateDID:      agentDID,
		Scope:            scope,
		TTLSeconds:       int64(ttl.Seconds()),
	})
	if err != nil {
		return nil, err
	}

	child, err := decodeCredentialPayload(resp.Credential)
	if err != nil {
		return nil, fmt.Errorf("parse delegated credential: %w", err)
	}

	return &AgentSession{
		AgentDID:    agentDID,
		AccessToken: resp.Credential,
		ExpiresAt:   child.ExpiresAt,
		ParentToken: parentToken,
		Scope:       scope,
	}, nil
}

// RefreshAgentSession refreshes an existing delegated credential.
func (a *AgentClient) RefreshAgentSession(ctx context.Context, session *AgentSession) error {
	if session == nil {
		return errors.New("session is required")
	}
	newSession, err := a.StartAgentSession(ctx, session.ParentToken, session.AgentDID, session.Scope, session.ExpiresAt.Sub(time.Now()))
	if err != nil {
		return err
	}
	session.AccessToken = newSession.AccessToken
	session.ExpiresAt = newSession.ExpiresAt
	return nil
}

// CallAuthorized validates the delegated credential, refreshes if needed, and calls the target service.
func (a *AgentClient) CallAuthorized(ctx context.Context, session *AgentSession, method string, url string, body any) (*http.Response, error) {
	if session == nil {
		return nil, errors.New("session is required")
	}

	if time.Now().After(session.ExpiresAt) {
		if err := a.RefreshAgentSession(ctx, session); err != nil {
			return nil, err
		}
	}

	req := GatewayAuthorizeRequest{Credentials: []string{session.ParentToken, session.AccessToken}, WantSyntheticJWT: true}
	decision, err := a.SDK.GatewayAuthorize(ctx, req)
	if err != nil {
		return nil, err
	}
	if !decision.Allowed {
		return nil, fmt.Errorf("authorization denied: %s", decision.Reason)
	}

	var bodyReader *bytes.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(encoded)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+decision.SyntheticJWT)
	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	return a.SDK.httpClient().Do(httpReq)
}

type credentialPayload struct {
	Subject   string                 `json:"subject"`
	ExpiresAt time.Time              `json:"expires_at"`
	Claims    map[string]interface{} `json:"claims"`
	Scope     []string               `json:"-"`
}

func decodeCredentialPayload(token string) (credentialPayload, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return credentialPayload{}, fmt.Errorf("invalid token format")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return credentialPayload{}, fmt.Errorf("decode payload: %w", err)
	}
	var payload credentialPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return credentialPayload{}, fmt.Errorf("unmarshal payload: %w", err)
	}

	payload.Scope = scopeFromClaims(payload.Claims)
	return payload, nil
}

func scopeFromClaims(claims map[string]interface{}) []string {
	if claims == nil {
		return nil
	}
	if raw, ok := claims["scope"]; ok {
		switch v := raw.(type) {
		case []string:
			return append([]string{}, v...)
		case []interface{}:
			res := make([]string, 0, len(v))
			for _, item := range v {
				if s, ok := item.(string); ok {
					res = append(res, s)
				}
			}
			return res
		case string:
			return []string{v}
		}
	}
	return nil
}

func isScopeSubset(parentScope, childScope []string) bool {
	if len(childScope) == 0 {
		return true
	}
	parentSet := make(map[string]struct{}, len(parentScope))
	for _, s := range parentScope {
		parentSet[s] = struct{}{}
	}
	for _, s := range childScope {
		if _, ok := parentSet[s]; !ok {
			return false
		}
	}
	return true
}

func clampTTL(requested time.Duration, parentExpiry time.Time) time.Duration {
	if parentExpiry.IsZero() {
		return requested
	}
	remaining := time.Until(parentExpiry)
	if remaining <= 0 {
		return 0
	}
	if requested <= 0 || requested > remaining {
		return remaining
	}
	return requested
}
