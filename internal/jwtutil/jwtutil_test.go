package jwtutil

import (
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

func TestValidateSyntheticJWTHappyPath(t *testing.T) {
	token, jwksURL, _ := buildTestToken(t, "did:example:issuer", "api", "kid-1")

	claims, err := ValidateSyntheticJWT(token, ValidationConfig{
		Issuer:      "did:example:issuer",
		Audience:    "api",
		AllowedAlgs: []string{"EdDSA"},
		JWKSURL:     jwksURL,
	})
	if err != nil {
		t.Fatalf("expected token to validate: %v", err)
	}

	if claims.Subject != "did:example:subject" {
		t.Fatalf("unexpected subject: %s", claims.Subject)
	}
	if claims.OnBehalfOf != "did:example:obo" {
		t.Fatalf("unexpected obo: %s", claims.OnBehalfOf)
	}
	if claims.DelegationDepth != 2 {
		t.Fatalf("unexpected delegation depth: %d", claims.DelegationDepth)
	}
	if len(claims.Scope) != 2 || claims.Scope[0] != "read" {
		t.Fatalf("unexpected scope: %+v", claims.Scope)
	}
}

func TestValidateSyntheticJWTSignatureFailure(t *testing.T) {
	token, _, _ := buildTestToken(t, "did:example:issuer", "api", "kid-1")

	// JWKS without the correct key should cause verification failure.
	_, otherPriv, _ := ed25519.GenerateKey(nil)
	otherPub := otherPriv.Public()
	otherSet := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{KeyID: "kid-2", Key: otherPub, Algorithm: string(jose.EdDSA)}}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(otherSet)
	}))
	defer srv.Close()

	if _, err := ValidateSyntheticJWT(token, ValidationConfig{Issuer: "did:example:issuer", Audience: "api", AllowedAlgs: []string{"EdDSA"}, JWKSURL: srv.URL}); err == nil {
		t.Fatalf("expected signature verification error")
	}
}

func TestValidateSyntheticJWTWrongIssuer(t *testing.T) {
	token, jwksURL, _ := buildTestToken(t, "did:example:issuer", "api", "kid-1")
	if _, err := ValidateSyntheticJWT(token, ValidationConfig{Issuer: "did:example:other", Audience: "api", AllowedAlgs: []string{"EdDSA"}, JWKSURL: jwksURL}); err == nil {
		t.Fatalf("expected issuer validation error")
	}
}

func TestValidateSyntheticJWTDisallowedAlg(t *testing.T) {
	token, jwksURL, _ := buildTestToken(t, "did:example:issuer", "api", "kid-1")
	if _, err := ValidateSyntheticJWT(token, ValidationConfig{Issuer: "did:example:issuer", Audience: "api", AllowedAlgs: []string{"RS256"}, JWKSURL: jwksURL}); err == nil {
		t.Fatalf("expected algorithm restriction error")
	}
}

func buildTestToken(t *testing.T, issuer, aud, kid string) (string, string, ed25519.PrivateKey) {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	set := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{KeyID: kid, Key: pub, Algorithm: string(jose.EdDSA)}}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(set)
	}))

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.EdDSA, Key: jose.JSONWebKey{KeyID: kid, Key: priv}}, nil)
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}

	claims := map[string]any{
		"iss":              issuer,
		"sub":              "did:example:subject",
		"aud":              aud,
		"exp":              time.Now().Add(time.Hour).Unix(),
		"iat":              time.Now().Add(-time.Minute).Unix(),
		"obo":              "did:example:obo",
		"scope":            []string{"read", "write"},
		"delegation_depth": 2,
	}

	token, err := jwt.Signed(signer).Claims(claims).Serialize()
	if err != nil {
		t.Fatalf("serialize token: %v", err)
	}

	return token, srv.URL, priv
}
