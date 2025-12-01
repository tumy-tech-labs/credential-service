package domain

import (
	"crypto"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Verification errors.
var (
	ErrInvalidToken       = errors.New("invalid credential token")
	ErrInvalidSignature   = errors.New("invalid credential signature")
	ErrExpiredCredential  = errors.New("credential has expired")
	ErrIssuedInFuture     = errors.New("credential is issued in the future")
	ErrUntrustedIssuer    = errors.New("issuer is not trusted")
	ErrUnexpectedAudience = errors.New("unexpected audience")
	ErrInvalidDelegation  = errors.New("invalid delegation chain")
	ErrDelegationScope    = errors.New("delegated scope must be subset of parent")
	ErrDelegationTTL      = errors.New("delegated credential expires after parent")
	ErrDelegationDepth    = errors.New("delegation depth exceeded")
	ErrMissingDisclosure  = errors.New("disclosure missing for SD-JWT digest")
	ErrInvalidDisclosure  = errors.New("invalid SD-JWT disclosure")
)

// VerificationResult captures the verified credential payload.
type VerificationResult struct {
	Credential VerifiableCredential
}

// DelegationChainResult captures details of a verified delegation chain.
type DelegationChainResult struct {
	Credentials     []VerifiableCredential
	CurrentSubject  string
	RootDelegator   string
	DelegationDepth int
}

// VerifierDependencies holds collaborators needed for verification.
type VerifierDependencies struct {
	ResolveIssuerPublicKey func(issuerDID string) (crypto.PublicKey, error)
}

// VerificationOptions captures optional verification constraints.
type VerificationOptions struct {
	ExpectedAudience   string
	MaxDelegationDepth int
}

// VerifyCredentialChain parses and verifies a chain of credential tokens.
func VerifyCredentialChain(tokens []string, deps VerifierDependencies, opts VerificationOptions, now time.Time) (*DelegationChainResult, error) {
	if deps.ResolveIssuerPublicKey == nil {
		return nil, fmt.Errorf("resolve issuer public key is required: %w", ErrInvalidToken)
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("no credentials supplied: %w", ErrInvalidDelegation)
	}

	maxDepth := opts.MaxDelegationDepth
	if maxDepth == 0 {
		maxDepth = 3
	}

	credentials := make([]VerifiableCredential, 0, len(tokens))
	for idx, token := range tokens {
		baseToken, disclosures := parseSDJWTPayload(token)
		expectedAudience := ""
		if idx == len(tokens)-1 {
			expectedAudience = opts.ExpectedAudience
		}

		credential, err := verifySingleCredential(baseToken, disclosures, deps, expectedAudience, now)
		if err != nil {
			return nil, err
		}

		credentials = append(credentials, credential)

		if idx == 0 {
			continue
		}

		parent := credentials[idx-1]
		child := credential

		if !IsDepthAllowed(idx, maxDepth) {
			return nil, ErrDelegationDepth
		}

		if !IsScopeSubset(ScopeFromClaims(parent.Claims), ScopeFromClaims(child.Claims)) {
			return nil, ErrDelegationScope
		}

		if !IsTTLWithinParent(parent.ExpiresAt, child.ExpiresAt) {
			return nil, ErrDelegationTTL
		}
	}

	result := &DelegationChainResult{
		Credentials:     credentials,
		CurrentSubject:  credentials[len(credentials)-1].Subject,
		RootDelegator:   credentials[0].Subject,
		DelegationDepth: len(credentials) - 1,
	}

	if !IsDepthAllowed(result.DelegationDepth, maxDepth) {
		return nil, ErrDelegationDepth
	}

	return result, nil
}

// VerifyCredential parses and verifies a compact JWS/JWT credential token.
func VerifyCredential(token string, deps VerifierDependencies, expectedAudience string, now time.Time) (*VerificationResult, error) {
	res, err := VerifyCredentialChain([]string{token}, deps, VerificationOptions{ExpectedAudience: expectedAudience, MaxDelegationDepth: 1}, now)
	if err != nil {
		return nil, err
	}
	return &VerificationResult{Credential: res.Credentials[0]}, nil
}

func verifySingleCredential(token string, disclosures []string, deps VerifierDependencies, expectedAudience string, now time.Time) (VerifiableCredential, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return VerifiableCredential{}, ErrInvalidToken
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return VerifiableCredential{}, ErrInvalidToken
	}

	// Try W3C VC-JWT format first (vc claim wrapper)
	var vcjwtPayload VCJWTPayload
	if err := json.Unmarshal(payloadBytes, &vcjwtPayload); err == nil && vcjwtPayload.VC.Context != nil {
		// W3C VC-JWT format: extract credential and populate JWT fields
		credential := vcjwtPayload.VC
		credential.ISS = vcjwtPayload.ISS
		credential.SUB = vcjwtPayload.SUB
		credential.JTI = vcjwtPayload.JTI
		credential.IAT = vcjwtPayload.IAT
		credential.EXP = vcjwtPayload.EXP
		credential.NBF = vcjwtPayload.NBF

		// Populate legacy fields from JWT claims and VC data
		if credential.IAT > 0 {
			credential.IssuedAt = time.Unix(credential.IAT, 0).UTC()
		}
		if credential.EXP > 0 {
			credential.ExpiresAt = time.Unix(credential.EXP, 0).UTC()
		}
		credential.Subject = vcjwtPayload.SUB
		// Ensure Claims field mirrors CredentialSubject for backward compatibility
		if credential.Claims == nil && credential.CredentialSubject != nil {
			credential.Claims = credential.CredentialSubject
		}

		// Verify signature with issuer from vc.issuer
		publicKey, err := deps.ResolveIssuerPublicKey(credential.Issuer)
		if err != nil {
			return VerifiableCredential{}, fmt.Errorf("resolve issuer: %w", ErrUntrustedIssuer)
		}

		edKey, ok := publicKey.(ed25519.PublicKey)
		if !ok {
			return VerifiableCredential{}, ErrInvalidToken
		}

		signingInput := parts[0] + "." + parts[1]
		signatureBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
		if err != nil {
			return VerifiableCredential{}, ErrInvalidToken
		}

		if !ed25519.Verify(edKey, []byte(signingInput), signatureBytes) {
			return VerifiableCredential{}, ErrInvalidSignature
		}

		// Validate expiration using EXP claim
		if credential.EXP > 0 && now.Unix() >= credential.EXP {
			return VerifiableCredential{}, ErrExpiredCredential
		}

		// Validate issued time with clock skew
		if credential.IAT > 0 && credential.IAT > now.Add(1*time.Minute).Unix() {
			return VerifiableCredential{}, ErrIssuedInFuture
		}

		if len(credential.SDDigests) > 0 {
			if err := applyDisclosures(&credential, disclosures); err != nil {
				return VerifiableCredential{}, err
			}
		}

		if expectedAudience != "" {
			if aud, ok := credential.Claims["aud"]; !ok || aud != expectedAudience {
				return VerifiableCredential{}, ErrUnexpectedAudience
			}
		}

		return credential, nil
	}

	// Fallback: try legacy flat format for backward compatibility
	var credential VerifiableCredential
	if err := json.Unmarshal(payloadBytes, &credential); err != nil {
		return VerifiableCredential{}, ErrInvalidToken
	}

	// For legacy flat format, use ISS/SUB/EXP fields if Issuer/Subject/ExpiresAt are empty
	issuerDID := credential.Issuer
	if issuerDID == "" && credential.ISS != "" {
		issuerDID = credential.ISS
		credential.Issuer = credential.ISS
	}
	if credential.Subject == "" && credential.SUB != "" {
		credential.Subject = credential.SUB
	}
	if credential.ExpiresAt.IsZero() && credential.EXP > 0 {
		credential.ExpiresAt = time.Unix(credential.EXP, 0).UTC()
	}
	if credential.IssuedAt.IsZero() && credential.IAT > 0 {
		credential.IssuedAt = time.Unix(credential.IAT, 0).UTC()
	}

	publicKey, err := deps.ResolveIssuerPublicKey(issuerDID)
	if err != nil {
		return VerifiableCredential{}, fmt.Errorf("resolve issuer: %w", ErrUntrustedIssuer)
	}

	edKey, ok := publicKey.(ed25519.PublicKey)
	if !ok {
		return VerifiableCredential{}, ErrInvalidToken
	}

	signingInput := parts[0] + "." + parts[1]
	signatureBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return VerifiableCredential{}, ErrInvalidToken
	}

	if !ed25519.Verify(edKey, []byte(signingInput), signatureBytes) {
		return VerifiableCredential{}, ErrInvalidSignature
	}

	if !credential.ExpiresAt.IsZero() && !now.Before(credential.ExpiresAt) {
		return VerifiableCredential{}, ErrExpiredCredential
	}

	// Allow small clock skew; reject tokens issued far in the future.
	if !credential.IssuedAt.IsZero() && credential.IssuedAt.After(now.Add(1*time.Minute)) {
		return VerifiableCredential{}, ErrIssuedInFuture
	}

	if len(credential.SDDigests) > 0 {
		if err := applyDisclosures(&credential, disclosures); err != nil {
			return VerifiableCredential{}, err
		}
	}

	if expectedAudience != "" {
		aud, _ := credential.Claims["aud"].(string)
		if aud != expectedAudience {
			return VerifiableCredential{}, ErrUnexpectedAudience
		}
	}

	return credential, nil
}

func parseSDJWTPayload(token string) (string, []string) {
	parts := strings.Split(token, "~")
	if len(parts) == 0 {
		return token, nil
	}
	base := parts[0]
	if len(parts) == 1 {
		return base, nil
	}
	return base, parts[1:]
}

func applyDisclosures(vc *VerifiableCredential, disclosures []string) error {
	if len(vc.SDDigests) == 0 {
		return nil
	}
	digestSet := map[string]struct{}{}
	for _, d := range vc.SDDigests {
		digestSet[d] = struct{}{}
	}

	resolvedClaims := map[string]interface{}{}
	for _, disclosure := range disclosures {
		raw, err := base64.RawURLEncoding.DecodeString(disclosure)
		if err != nil {
			return ErrInvalidToken
		}
		var arr []interface{}
		if err := json.Unmarshal(raw, &arr); err != nil {
			return ErrInvalidToken
		}
		if len(arr) != 3 {
			return ErrInvalidToken
		}
		salt, _ := arr[0].(string)
		key, _ := arr[1].(string)
		if salt == "" || key == "" {
			return ErrInvalidToken
		}
		reconstructed, err := json.Marshal(arr)
		if err != nil {
			return ErrInvalidToken
		}

		digest := computeDisclosureDigest(reconstructed)
		if _, ok := digestSet[digest]; !ok {
			return ErrInvalidDisclosure
		}
		resolvedClaims[key] = arr[2]
	}

	for digest := range digestSet {
		match := false
		for _, disclosure := range disclosures {
			raw, err := base64.RawURLEncoding.DecodeString(disclosure)
			if err != nil {
				return ErrInvalidDisclosure
			}
			if computeDisclosureDigest(raw) == digest {
				match = true
				break
			}
		}
		if !match {
			return ErrMissingDisclosure
		}
	}

	if vc.Claims == nil {
		vc.Claims = map[string]interface{}{}
	}
	for k, v := range resolvedClaims {
		vc.Claims[k] = v
	}
	vc.SDDigests = nil
	return nil
}

func ScopeFromClaims(claims map[string]interface{}) []string {
	if claims == nil {
		return nil
	}
	scopeVal, ok := claims["scope"]
	if !ok {
		return nil
	}

	switch v := scopeVal.(type) {
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
	default:
		return nil
	}
}
