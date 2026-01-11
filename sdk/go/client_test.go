package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIssueVCFormatsRequest(t *testing.T) {
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
		_ = json.NewEncoder(w).Encode(IssueResponse{Credential: "vc.jwt", Format: captured.Format})
	}))
	t.Cleanup(srv.Close)

	client := &Client{IssuerURL: srv.URL}
	resp, err := client.IssueVC(context.Background(), IssueRequest{SubjectDID: "did:example:1", TTLSeconds: 60})
	if err != nil {
		t.Fatalf("IssueVC returned error: %v", err)
	}
	if captured.Format != "jwt-vc" {
		t.Fatalf("expected format jwt-vc, got %s", captured.Format)
	}
	if resp.Credential != "vc.jwt" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestIssueSDJWTIncludesDisclosures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(IssueResponse{Credential: "sd.jwt", Disclosures: []string{"disc"}, Format: "sd-jwt"})
	}))
	t.Cleanup(srv.Close)

	client := &Client{IssuerURL: srv.URL}
	resp, err := client.IssueSDJWT(context.Background(), IssueRequest{SubjectDID: "did:example:1", TTLSeconds: 60})
	if err != nil {
		t.Fatalf("IssueSDJWT returned error: %v", err)
	}
	if resp.Format != "sd-jwt" || len(resp.Disclosures) != 1 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestVerifyUsesVerifierURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/credentials/verify" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(VerifyResponse{Valid: true, Subject: "did:sub", DelegationDepth: 0, ExpiresAt: time.Now()})
	}))
	t.Cleanup(srv.Close)

	client := &Client{VerifierURL: srv.URL}
	resp, err := client.Verify(context.Background(), "token")
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if !resp.Valid || resp.Subject != "did:sub" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestAuthorizeFallsBackToVerifierURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/gateway/authorize" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var req AuthorizeRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Resource != "orders" || req.Action != "read" {
			t.Fatalf("unexpected request payload: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(AuthorizeResponse{Allowed: true, SyntheticJWT: "synthetic"})
	}))
	t.Cleanup(srv.Close)

	client := &Client{VerifierURL: srv.URL}
	resp, err := client.Authorize(context.Background(), AuthorizeRequest{Credential: "cred", Resource: "orders", Action: "read", WantSyntheticJWT: true})
	if err != nil {
		t.Fatalf("Authorize returned error: %v", err)
	}
	if !resp.Allowed || resp.SyntheticJWT != "synthetic" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestDecodeAPIErrorMapsStatus(t *testing.T) {
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
			_ = json.NewEncoder(w).Encode(APIError{Error: "boom"})
		}))
		client := &Client{IssuerURL: srv.URL}
		_, err := client.IssueVC(context.Background(), IssueRequest{SubjectDID: "did:ex", TTLSeconds: 1})
		srv.Close()

		if !errors.Is(err, tc.want) {
			t.Fatalf("status %d: expected %v got %v", tc.status, tc.want, err)
		}
	}
}

func TestMissingBaseURL(t *testing.T) {
	client := &Client{}
	_, err := client.IssueVC(context.Background(), IssueRequest{SubjectDID: "did:ex", TTLSeconds: 1})
	if err == nil {
		t.Fatalf("expected error for missing issuer url")
	}
}
