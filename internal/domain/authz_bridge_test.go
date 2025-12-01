package domain

import (
	"crypto/ed25519"
	"testing"
	"time"
)

func TestBuildAuthzDecisionFromVerification(t *testing.T) {
	credential := VerifiableCredential{
		Subject: "did:example:alice",
		Claims: map[string]interface{}{
			"scope": []string{"read:orders"},
		},
	}

	result := &DelegationChainResult{
		Credentials:     []VerifiableCredential{credential},
		CurrentSubject:  credential.Subject,
		RootDelegator:   "did:example:alice",
		DelegationDepth: 0,
	}

	decision := BuildAuthzDecisionFromVerification(result)

	if !decision.Allowed || decision.SubjectDID != credential.Subject || decision.ActingOnBehalfOf != result.RootDelegator {
		t.Fatalf("unexpected decision: %+v", decision)
	}

	if scope := ScopeFromClaims(decision.Claims); len(scope) != 1 || scope[0] != "read:orders" {
		t.Fatalf("claims not propagated: %+v", decision.Claims)
	}
}

func TestBuildSyntheticJWT(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)

	decision := AuthzDecision{
		Allowed:          true,
		SubjectDID:       "did:example:carol",
		ActingOnBehalfOf: "did:example:root",
		DelegationDepth:  1,
		Claims: map[string]interface{}{
			"scope": "read:orders",
		},
	}

	token, err := BuildSyntheticJWT(decision, priv, "did:issuer:verifier", time.Minute)
	if err != nil {
		t.Fatalf("build synthetic jwt: %v", err)
	}

	if token.Token == "" {
		t.Fatalf("expected token to be populated")
	}

	if err := VerifySyntheticJWTSignature(token.Token, priv.Public()); err != nil {
		t.Fatalf("verify signature: %v", err)
	}

	payload, err := DecodeSyntheticJWT(token.Token)
	if err != nil {
		t.Fatalf("decode token: %v", err)
	}

	if payload["sub"] != decision.SubjectDID {
		t.Fatalf("expected sub %s, got %v", decision.SubjectDID, payload["sub"])
	}

	if payload["obo"] != decision.ActingOnBehalfOf {
		t.Fatalf("expected obo %s, got %v", decision.ActingOnBehalfOf, payload["obo"])
	}

	if payload["delegation_depth"] != float64(decision.DelegationDepth) {
		t.Fatalf("unexpected delegation depth: %v", payload["delegation_depth"])
	}

	if payload["scope"] != "read:orders" {
		t.Fatalf("unexpected scope: %v", payload["scope"])
	}
}
