package domain

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Authorization bridge errors
var (
	ErrDecisionDenied = errors.New("cannot mint token for denied decision")
	ErrSigningKeyRequired = errors.New("signing key is required")
	ErrIssuerRequired = errors.New("issuer is required")
	ErrInvalidTTL = errors.New("ttl must be positive")
	ErrInvalidTokenFormat = errors.New("invalid token format")
	ErrUnsupportedPublicKeyType = errors.New("unsupported public key type")
	ErrInvalidTokenSignature = errors.New("invalid signature")
)

// AuthzDecision captures an authorization decision derived from a verified credential chain.
type AuthzDecision struct {
	Allowed          bool                   `json:"allowed"`
	SubjectDID       string                 `json:"subject"`
	ActingOnBehalfOf string                 `json:"acting_on_behalf_of,omitempty"`
	DelegationDepth  int                    `json:"delegation_depth"`
	Claims           map[string]interface{} `json:"claims,omitempty"`
	Reason           string                 `json:"reason,omitempty"`
}

// SyntheticJWT represents a minimal JWT-like structure for legacy services.
type SyntheticJWT struct {
	Token string `json:"token"`
}

// BuildAuthzDecisionFromVerification reshapes a DelegationChainResult into an AuthzDecision.
func BuildAuthzDecisionFromVerification(result *DelegationChainResult) AuthzDecision {
	decision := AuthzDecision{}
	if result == nil || len(result.Credentials) == 0 {
		decision.Reason = "missing_verification_result"
		return decision
	}

	leaf := result.Credentials[len(result.Credentials)-1]
	decision.Allowed = true
	decision.SubjectDID = result.CurrentSubject
	decision.ActingOnBehalfOf = result.RootDelegator
	decision.DelegationDepth = result.DelegationDepth
	decision.Claims = leaf.Claims

	return decision
}

// BuildSyntheticJWT issues a compact JWT-like token from an authorization decision.
func BuildSyntheticJWT(decision AuthzDecision, signingKey crypto.Signer, issuer string, ttl time.Duration) (SyntheticJWT, error) {
	if !decision.Allowed {
		return SyntheticJWT{}, ErrDecisionDenied
	}
	if signingKey == nil {
		return SyntheticJWT{}, ErrSigningKeyRequired
	}
	if issuer == "" {
		return SyntheticJWT{}, ErrIssuerRequired
	}
	if ttl <= 0 {
		return SyntheticJWT{}, ErrInvalidTTL
	}

	now := time.Now().UTC()
	payload := map[string]interface{}{
		"iss":              issuer,
		"sub":              decision.SubjectDID,
		"iat":              now.Unix(),
		"exp":              now.Add(ttl).Unix(),
		"delegation_depth": decision.DelegationDepth,
	}

	if decision.ActingOnBehalfOf != "" {
		payload["obo"] = decision.ActingOnBehalfOf
	}

	if scopeVal, ok := decision.Claims["scope"]; ok {
		payload["scope"] = scopeVal
	}

	header := map[string]string{
		"alg": "EdDSA",
		"typ": "JWT",
	}

	headerSegment, err := encodeSegment(header)
	if err != nil {
		return SyntheticJWT{}, err
	}

	payloadSegment, err := encodeSegment(payload)
	if err != nil {
		return SyntheticJWT{}, err
	}

	signingInput := headerSegment + "." + payloadSegment

	signature, err := signingKey.Sign(rand.Reader, []byte(signingInput), crypto.Hash(0))
	if err != nil {
		return SyntheticJWT{}, fmt.Errorf("sign payload: %w", err)
	}

	sigSegment := base64.RawURLEncoding.EncodeToString(signature)

	return SyntheticJWT{Token: signingInput + "." + sigSegment}, nil
}

// DecodeSyntheticJWT is a helper used in tests to decode a synthetic JWT payload.
func DecodeSyntheticJWT(token string) (map[string]interface{}, error) {
	parts := splitToken(token)
	if len(parts) != 3 {
		return nil, ErrInvalidTokenFormat
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}
	return payload, nil
}

// VerifySyntheticJWTSignature verifies the EdDSA signature on a synthetic JWT.
func VerifySyntheticJWTSignature(token string, publicKey crypto.PublicKey) error {
	parts := splitToken(token)
	if len(parts) != 3 {
		return ErrInvalidTokenFormat
	}

	signingInput := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	edKey, ok := publicKey.(ed25519.PublicKey)
	if !ok {
		return ErrUnsupportedPublicKeyType
	}

	if !ed25519.Verify(edKey, []byte(signingInput), signature) {
		return ErrInvalidTokenSignature
	}
	return nil
}

func splitToken(token string) []string {
	res := make([]string, 0, 3)
	start := 0
	for i := 0; i < len(token); i++ {
		if token[i] == '.' {
			res = append(res, token[start:i])
			start = i + 1
		}
	}
	if start <= len(token) {
		res = append(res, token[start:])
	}
	return res
}
