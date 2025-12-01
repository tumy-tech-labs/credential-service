package domain

import (
	"crypto"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestVerifyCredentialSuccess(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	token, err := IssueBasicCredential("did:jwk:issuer", "did:jwk:subject", priv, 5*time.Minute, map[string]interface{}{"aud": "example"})
	if err != nil {
		t.Fatalf("issue credential: %v", err)
	}

	deps := VerifierDependencies{ResolveIssuerPublicKey: func(issuer string) (crypto.PublicKey, error) {
		if issuer != "did:jwk:issuer" {
			return nil, errors.New("unknown issuer")
		}
		return pub, nil
	}}

	result, err := VerifyCredential(token, deps, "example", time.Now())
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if result.Credential.Subject != "did:jwk:subject" {
		t.Fatalf("unexpected subject: %s", result.Credential.Subject)
	}
}

func TestVerifyCredentialChainNoDelegation(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	token, err := IssueBasicCredential("did:jwk:issuer", "did:jwk:subject", priv, 5*time.Minute, map[string]interface{}{"aud": "example"})
	if err != nil {
		t.Fatalf("issue credential: %v", err)
	}

	deps := VerifierDependencies{ResolveIssuerPublicKey: func(issuer string) (crypto.PublicKey, error) {
		if issuer != "did:jwk:issuer" {
			return nil, errors.New("unknown issuer")
		}
		return pub, nil
	}}

	result, err := VerifyCredentialChain([]string{token}, deps, VerificationOptions{ExpectedAudience: "example", MaxDelegationDepth: 2}, time.Now())
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if result.DelegationDepth != 0 {
		t.Fatalf("expected depth 0, got %d", result.DelegationDepth)
	}

	if result.CurrentSubject != "did:jwk:subject" || result.RootDelegator != "did:jwk:subject" {
		t.Fatalf("unexpected delegation parties: %+v", result)
	}
}

func TestVerifyCredentialChainDelegated(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	parentToken, err := IssueBasicCredential("did:jwk:issuer", "did:jwk:parent", priv, 10*time.Minute, map[string]interface{}{"scope": []string{"read", "write"}})
	if err != nil {
		t.Fatalf("issue parent: %v", err)
	}

	childToken, err := IssueBasicCredential("did:jwk:issuer", "did:jwk:agent", priv, 5*time.Minute, map[string]interface{}{"scope": []string{"read"}, "aud": "api"})
	if err != nil {
		t.Fatalf("issue child: %v", err)
	}

	deps := VerifierDependencies{ResolveIssuerPublicKey: func(issuer string) (crypto.PublicKey, error) {
		if issuer != "did:jwk:issuer" {
			return nil, errors.New("unknown issuer")
		}
		return pub, nil
	}}

	result, err := VerifyCredentialChain([]string{parentToken, childToken}, deps, VerificationOptions{ExpectedAudience: "api", MaxDelegationDepth: 3}, time.Now())
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if result.DelegationDepth != 1 {
		t.Fatalf("expected delegation depth 1, got %d", result.DelegationDepth)
	}

	if result.RootDelegator != "did:jwk:parent" || result.CurrentSubject != "did:jwk:agent" {
		t.Fatalf("unexpected parties: %+v", result)
	}
}

func TestVerifyCredentialChainScopeExpansionFails(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	parentToken, _ := IssueBasicCredential("did:jwk:issuer", "did:jwk:parent", priv, 5*time.Minute, map[string]interface{}{"scope": []string{"read"}})
	childToken, _ := IssueBasicCredential("did:jwk:issuer", "did:jwk:agent", priv, 3*time.Minute, map[string]interface{}{"scope": []string{"read", "write"}})

	deps := VerifierDependencies{ResolveIssuerPublicKey: func(string) (crypto.PublicKey, error) { return pub, nil }}

	if _, err := VerifyCredentialChain([]string{parentToken, childToken}, deps, VerificationOptions{MaxDelegationDepth: 2}, time.Now()); !errors.Is(err, ErrDelegationScope) {
		t.Fatalf("expected ErrDelegationScope, got %v", err)
	}
}

func TestVerifyCredentialChainTTLExpansionFails(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	parentToken, _ := IssueBasicCredential("did:jwk:issuer", "did:jwk:parent", priv, 5*time.Minute, map[string]interface{}{"scope": []string{"read"}})
	childToken, _ := IssueBasicCredential("did:jwk:issuer", "did:jwk:agent", priv, 10*time.Minute, map[string]interface{}{"scope": []string{"read"}})

	deps := VerifierDependencies{ResolveIssuerPublicKey: func(string) (crypto.PublicKey, error) { return pub, nil }}

	if _, err := VerifyCredentialChain([]string{parentToken, childToken}, deps, VerificationOptions{MaxDelegationDepth: 2}, time.Now()); !errors.Is(err, ErrDelegationTTL) {
		t.Fatalf("expected ErrDelegationTTL, got %v", err)
	}
}

func TestVerifyCredentialChainDepthLimit(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	parentToken, _ := IssueBasicCredential("did:jwk:issuer", "did:jwk:parent", priv, 5*time.Minute, map[string]interface{}{"scope": []string{"read"}})
	childToken, _ := IssueBasicCredential("did:jwk:issuer", "did:jwk:agent", priv, 4*time.Minute, map[string]interface{}{"scope": []string{"read"}})
	grandChildToken, _ := IssueBasicCredential("did:jwk:issuer", "did:jwk:agent-2", priv, 3*time.Minute, map[string]interface{}{"scope": []string{"read"}})

	deps := VerifierDependencies{ResolveIssuerPublicKey: func(string) (crypto.PublicKey, error) { return pub, nil }}

	if _, err := VerifyCredentialChain([]string{parentToken, childToken, grandChildToken}, deps, VerificationOptions{MaxDelegationDepth: 1}, time.Now()); !errors.Is(err, ErrDelegationDepth) {
		t.Fatalf("expected ErrDelegationDepth, got %v", err)
	}
}

func TestVerifyCredentialExpired(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	token, err := IssueBasicCredential("did:jwk:issuer", "did:jwk:subject", priv, time.Minute, nil)
	if err != nil {
		t.Fatalf("issue credential: %v", err)
	}

	vc := decodeCredentialPayload(t, token)
	deps := VerifierDependencies{ResolveIssuerPublicKey: func(string) (crypto.PublicKey, error) {
		return priv.Public(), nil
	}}

	if _, err := VerifyCredential(token, deps, "", vc.ExpiresAt.Add(time.Second)); !errors.Is(err, ErrExpiredCredential) {
		t.Fatalf("expected ErrExpiredCredential, got %v", err)
	}
}

func TestVerifyCredentialInvalidSignature(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	token, err := IssueBasicCredential("did:jwk:issuer", "did:jwk:subject", priv, 5*time.Minute, nil)
	if err != nil {
		t.Fatalf("issue credential: %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected token segments")
	}
	parts[2] = base64.RawURLEncoding.EncodeToString([]byte("tampered"))
	tampered := strings.Join(parts, ".")

	deps := VerifierDependencies{ResolveIssuerPublicKey: func(string) (crypto.PublicKey, error) {
		return priv.Public(), nil
	}}

	if _, err := VerifyCredential(tampered, deps, "", time.Now()); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestVerifyCredentialUntrustedIssuer(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	token, err := IssueBasicCredential("did:jwk:issuer", "did:jwk:subject", priv, 5*time.Minute, nil)
	if err != nil {
		t.Fatalf("issue credential: %v", err)
	}

	deps := VerifierDependencies{ResolveIssuerPublicKey: func(string) (crypto.PublicKey, error) {
		return nil, errors.New("issuer not trusted")
	}}

	if _, err := VerifyCredential(token, deps, "", time.Now()); !errors.Is(err, ErrUntrustedIssuer) {
		t.Fatalf("expected ErrUntrustedIssuer, got %v", err)
	}
}

func decodeCredentialPayload(t *testing.T, token string) VerifiableCredential {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("invalid token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}

	// Try W3C VC-JWT format first
	var vcjwt VCJWTPayload
	if err := json.Unmarshal(payload, &vcjwt); err == nil && vcjwt.VC.Context != nil {
		// Extract VC and populate time fields from JWT claims
		vc := vcjwt.VC
		if vcjwt.IAT > 0 {
			vc.IssuedAt = time.Unix(vcjwt.IAT, 0).UTC()
		}
		if vcjwt.EXP > 0 {
			vc.ExpiresAt = time.Unix(vcjwt.EXP, 0).UTC()
		}
		vc.Subject = vcjwt.SUB
		return vc
	}

	// Fallback to legacy flat format
	var vc VerifiableCredential
	if err := json.Unmarshal(payload, &vc); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return vc
}
