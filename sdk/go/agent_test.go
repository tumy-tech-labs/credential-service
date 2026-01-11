package sdk

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type delegateCall struct {
	Scope      []string `json:"scope"`
	TTLSeconds int64    `json:"ttl_seconds"`
}

func TestAgentSessionLifecycle(t *testing.T) {
	now := time.Now().UTC()
	parentToken := buildToken(t, credentialPayload{
		Subject:   "did:example:parent",
		ExpiresAt: now.Add(1 * time.Hour),
		Claims:    map[string]interface{}{"scope": []string{"read", "write"}},
	})

	var captured delegateCall
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/credentials/delegate":
			if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
				t.Fatalf("decode delegate: %v", err)
			}
			resp := DelegateResponse{Credential: buildToken(t, credentialPayload{Subject: "did:agent", ExpiresAt: now.Add(30 * time.Minute), Claims: map[string]interface{}{"scope": []string{"read"}}})}
			_ = json.NewEncoder(w).Encode(resp)
		case "/v1/gateway/authorize":
			_ = json.NewEncoder(w).Encode(AuthorizeResponse{Allowed: true, SyntheticJWT: "synthetic.jwt.token"})
		case "/resource":
			if got := r.Header.Get("Authorization"); got != "Bearer synthetic.jwt.token" {
				t.Fatalf("unexpected auth header: %s", got)
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	client := &Client{IssuerURL: server.URL, VerifierURL: server.URL, GatewayURL: server.URL, HTTPClient: server.Client()}
	agent := &AgentClient{SDK: client}

	session, err := agent.StartAgentSession(context.Background(), parentToken, "did:agent", []string{"read"}, 45*time.Minute)
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if captured.TTLSeconds <= 0 || captured.TTLSeconds > int64(time.Until(now.Add(1*time.Hour)).Seconds()) {
		t.Fatalf("ttl not clamped: %d", captured.TTLSeconds)
	}

	resp, err := agent.CallAuthorized(context.Background(), session, http.MethodGet, server.URL+"/resource", nil)
	if err != nil {
		t.Fatalf("call authorized: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}

func TestAgentRefreshOnExpiry(t *testing.T) {
	now := time.Now().UTC()
	parentToken := buildToken(t, credentialPayload{Subject: "did:parent", ExpiresAt: now.Add(30 * time.Minute), Claims: map[string]interface{}{"scope": []string{"read"}}})

	refreshCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/credentials/delegate":
			refreshCount++
			resp := DelegateResponse{Credential: buildToken(t, credentialPayload{Subject: "did:agent", ExpiresAt: now.Add(time.Duration(refreshCount) * 10 * time.Minute), Claims: map[string]interface{}{"scope": []string{"read"}}})}
			_ = json.NewEncoder(w).Encode(resp)
		case "/v1/gateway/authorize":
			_ = json.NewEncoder(w).Encode(AuthorizeResponse{Allowed: true, SyntheticJWT: "synthetic.jwt.token"})
		case "/resource":
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	client := &Client{IssuerURL: server.URL, VerifierURL: server.URL, GatewayURL: server.URL, HTTPClient: server.Client()}
	agent := &AgentClient{SDK: client}

	session, err := agent.StartAgentSession(context.Background(), parentToken, "did:agent", []string{"read"}, 5*time.Minute)
	if err != nil {
		t.Fatalf("start session: %v", err)
	}

	session.ExpiresAt = time.Now().Add(-1 * time.Minute)
	if _, err := agent.CallAuthorized(context.Background(), session, http.MethodGet, server.URL+"/resource", nil); err != nil {
		t.Fatalf("call authorized: %v", err)
	}
	if refreshCount < 2 {
		t.Fatalf("expected refresh to run at least twice, got %d", refreshCount)
	}
}

func buildToken(t *testing.T, payload credentialPayload) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	payloadBytes, _ := json.Marshal(payload)
	middle := base64.RawURLEncoding.EncodeToString(payloadBytes)
	return header + "." + middle + ".sig"
}
