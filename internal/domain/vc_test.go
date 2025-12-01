package domain

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestIssueBasicCredential(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	claims := map[string]interface{}{"role": "tester"}
	token, err := IssueBasicCredential("did:jwk:issuer", "did:jwk:subject", priv, 5*time.Minute, claims)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token == "" {
		t.Fatalf("expected token to be non-empty")
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 token segments, got %d", len(parts))
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}

	// Decode W3C VC-JWT payload structure
	var payload VCJWTPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	// Verify JWT claims
	if payload.ISS != "did:jwk:issuer" || payload.SUB != "did:jwk:subject" {
		t.Fatalf("unexpected issuer or subject in JWT claims: iss=%s sub=%s", payload.ISS, payload.SUB)
	}

	if payload.EXP <= payload.IAT {
		t.Fatalf("expected exp after iat")
	}

	// Verify nested VC
	if payload.VC.Issuer != "did:jwk:issuer" {
		t.Fatalf("unexpected issuer in nested VC: %s", payload.VC.Issuer)
	}

	if payload.VC.CredentialSubject == nil {
		t.Fatalf("expected credentialSubject to be populated")
	}
}

func TestIssueBasicCredentialRequiresTTL(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	if _, err := IssueBasicCredential("issuer", "subject", priv, 0, nil); err == nil {
		t.Fatalf("expected error for non-positive ttl")
	}
}
