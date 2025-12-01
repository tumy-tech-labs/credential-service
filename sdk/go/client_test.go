package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestIssueCredential(t *testing.T) {
	var captured IssueRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/credentials/issue" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(IssueResponse{
			Credential:       "token123",
			Subject:          captured.SubjectDID,
			ActingOnBehalfOf: captured.SubjectDID,
			DelegationDepth:  0,
			Claims:           captured.Claims,
		})
	}))
	defer srv.Close()

	client := &Client{BaseURL: srv.URL}
	req := IssueRequest{SubjectDID: "did:example:alice", TTLSeconds: 600, Claims: map[string]interface{}{"aud": "example"}}
	resp, err := client.IssueCredential(context.Background(), req)
	if err != nil {
		t.Fatalf("IssueCredential returned error: %v", err)
	}

	if resp.Credential != "token123" || resp.Subject != "did:example:alice" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if !reflect.DeepEqual(resp.Claims, req.Claims) {
		t.Fatalf("claims mismatch: %+v", resp.Claims)
	}
	if captured.TTLSeconds != req.TTLSeconds {
		t.Fatalf("ttl mismatch: %d", captured.TTLSeconds)
	}
}

func TestDelegateCredential(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/credentials/delegate" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(DelegateResponse{Credential: "delegated", DelegationDepth: 1, ActingOnBehalfOf: "did:parent"})
	}))
	defer srv.Close()

	client := &Client{BaseURL: srv.URL}
	resp, err := client.DelegateCredential(context.Background(), DelegateRequest{ParentCredential: "parent", DelegateDID: "did:child", Scope: []string{"read"}, TTLSeconds: 300})
	if err != nil {
		t.Fatalf("DelegateCredential returned error: %v", err)
	}
	if resp.Credential != "delegated" || resp.DelegationDepth != 1 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestVerify(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(VerifyResponse{Valid: true, Subject: "did:subject", DelegationDepth: 0, Claims: map[string]interface{}{"role": "admin"}})
	}))
	defer srv.Close()

	client := &Client{BaseURL: srv.URL}
	resp, err := client.Verify(context.Background(), "token")
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if !resp.Valid || resp.Subject != "did:subject" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Claims["role"] != "admin" {
		t.Fatalf("missing claim role")
	}
}

func TestGatewayAuthorize(t *testing.T) {
	var captured GatewayAuthorizeRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(GatewayAuthorizeResponse{Allowed: true, Subject: "did:subject", ActingOnBehalfOf: "did:parent", DelegationDepth: 1, Claims: map[string]interface{}{"scope": []string{"read"}}, SyntheticJWT: "jwt"})
	}))
	defer srv.Close()

	client := &Client{BaseURL: srv.URL}
	req := GatewayAuthorizeRequest{Credential: "token", ExpectedAudience: "api", WantSyntheticJWT: true}
	resp, err := client.GatewayAuthorize(context.Background(), req)
	if err != nil {
		t.Fatalf("GatewayAuthorize returned error: %v", err)
	}
	if !resp.Allowed || resp.SyntheticJWT != "jwt" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if !captured.WantSyntheticJWT || captured.ExpectedAudience != "api" {
		t.Fatalf("unexpected request captured: %+v", captured)
	}
}

func TestErrorMapping(t *testing.T) {
	cases := []struct {
		status int
		want   error
	}{
		{400, ErrInvalidCredential},
		{401, ErrUnauthorized},
		{403, ErrUnauthorized},
		{500, ErrServerError},
	}

	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.status)
		}))
		client := &Client{BaseURL: srv.URL}
		_, err := client.IssueCredential(context.Background(), IssueRequest{SubjectDID: "did:ex", TTLSeconds: 1})
		srv.Close()

		if !errors.Is(err, tc.want) {
			t.Fatalf("status %d: expected %v got %v", tc.status, tc.want, err)
		}
	}
}
