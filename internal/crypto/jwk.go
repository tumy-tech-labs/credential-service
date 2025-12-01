package crypto

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
)

// KeyMetadata holds normalized information about a signer.
type KeyMetadata struct {
	Key       Signer
	PublicKey crypto.PublicKey
	KID       string
	Alg       string
	JWK       json.RawMessage
	NotBefore string
	NotAfter  string
}

// KeyID derives a stable kid for the provided signer using RFC 7638 JWK Thumbprint.
// RFC 7638 specifies a canonical JSON representation with only required fields
// in lexicographic order, then SHA-256 hash, then base64url encoding.
func KeyID(s Signer) (string, error) {
	type keyed interface{ KeyID() string }
	if k, ok := s.(keyed); ok && k.KeyID() != "" {
		return k.KeyID(), nil
	}
	jwk, err := s.PublicJWK()
	if err != nil {
		return "", err
	}
	
	// Compute RFC 7638 JWK Thumbprint
	thumbprint, err := computeJWKThumbprint(jwk)
	if err != nil {
		return "", fmt.Errorf("compute jwk thumbprint: %w", err)
	}
	return thumbprint, nil
}

// computeJWKThumbprint implements RFC 7638 JWK Thumbprint computation.
// It uses only the required fields for each key type in lexicographic order.
func computeJWKThumbprint(jwk []byte) (string, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(jwk, &payload); err != nil {
		return "", fmt.Errorf("unmarshal jwk: %w", err)
	}

	kty, ok := payload["kty"].(string)
	if !ok {
		return "", errors.New("missing or invalid kty")
	}

	// Build canonical representation with only required fields in lexicographic order
	var canonical map[string]interface{}
	
	switch kty {
	case "OKP":
		// Required fields for OKP: crv, kty, x (lexicographic order)
		crv, ok := payload["crv"].(string)
		if !ok {
			return "", errors.New("missing or invalid crv for OKP key")
		}
		x, ok := payload["x"].(string)
		if !ok {
			return "", errors.New("missing or invalid x for OKP key")
		}
		canonical = map[string]interface{}{
			"crv": crv,
			"kty": kty,
			"x":   x,
		}
	case "EC":
		// Required fields for EC: crv, kty, x, y (lexicographic order)
		crv, ok := payload["crv"].(string)
		if !ok {
			return "", errors.New("missing or invalid crv for EC key")
		}
		x, ok := payload["x"].(string)
		if !ok {
			return "", errors.New("missing or invalid x for EC key")
		}
		y, ok := payload["y"].(string)
		if !ok {
			return "", errors.New("missing or invalid y for EC key")
		}
		canonical = map[string]interface{}{
			"crv": crv,
			"kty": kty,
			"x":   x,
			"y":   y,
		}
	case "RSA":
		// Required fields for RSA: e, kty, n (lexicographic order)
		e, ok := payload["e"].(string)
		if !ok {
			return "", errors.New("missing or invalid e for RSA key")
		}
		n, ok := payload["n"].(string)
		if !ok {
			return "", errors.New("missing or invalid n for RSA key")
		}
		canonical = map[string]interface{}{
			"e":   e,
			"kty": kty,
			"n":   n,
		}
	default:
		return "", fmt.Errorf("unsupported kty: %s", kty)
	}

	// Marshal to JSON with no whitespace (canonical form)
	canonicalJSON, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("marshal canonical jwk: %w", err)
	}

	// SHA-256 hash
	hash := sha256.Sum256(canonicalJSON)
	
	// Base64url encode without padding
	return base64.RawURLEncoding.EncodeToString(hash[:]), nil
}

// ParsePublicJWK decodes a public JWK into a crypto.PublicKey and algorithm.
func ParsePublicJWK(jwk []byte) (crypto.PublicKey, string, error) {
	var payload map[string]string
	if err := json.Unmarshal(jwk, &payload); err != nil {
		return nil, "", fmt.Errorf("unmarshal jwk: %w", err)
	}

	kty := payload["kty"]
	switch kty {
	case "OKP":
		crv := payload["crv"]
		if crv != "Ed25519" {
			return nil, "", fmt.Errorf("unsupported curve %s", crv)
		}
		x, err := base64.RawURLEncoding.DecodeString(payload["x"])
		if err != nil {
			return nil, "", fmt.Errorf("decode x: %w", err)
		}
		return ed25519.PublicKey(x), "EdDSA", nil
	case "EC":
		crv := payload["crv"]
		if crv != "P-256" {
			return nil, "", fmt.Errorf("unsupported curve %s", crv)
		}
		xBytes, err := base64.RawURLEncoding.DecodeString(payload["x"])
		if err != nil {
			return nil, "", fmt.Errorf("decode x: %w", err)
		}
		yBytes, err := base64.RawURLEncoding.DecodeString(payload["y"])
		if err != nil {
			return nil, "", fmt.Errorf("decode y: %w", err)
		}
		x := new(big.Int).SetBytes(xBytes)
		y := new(big.Int).SetBytes(yBytes)
		curve := elliptic.P256()
		if !curve.IsOnCurve(x, y) {
			return nil, "", errors.New("point not on curve")
		}
		return &ecdsa.PublicKey{Curve: curve, X: x, Y: y}, "ES256", nil
	case "RSA":
		return nil, "", errors.New("RSA keys are not supported")
	default:
		return nil, "", fmt.Errorf("unsupported kty %s", kty)
	}
}

// BuildKeyMetadata constructs metadata for a signer.
func BuildKeyMetadata(s Signer) (KeyMetadata, error) {
	jwk, err := s.PublicJWK()
	if err != nil {
		return KeyMetadata{}, err
	}
	kid, err := KeyID(s)
	if err != nil {
		return KeyMetadata{}, err
	}
	pub, alg, err := ParsePublicJWK(jwk)
	if err != nil {
		return KeyMetadata{}, err
	}

	meta := KeyMetadata{
		Key:       s,
		PublicKey: pub,
		KID:       kid,
		Alg:       alg,
		JWK:       jwk,
	}

	type windowed interface{ ValidityWindow() (string, string) }
	if w, ok := s.(windowed); ok {
		nb, na := w.ValidityWindow()
		meta.NotBefore = nb
		meta.NotAfter = na
	}

	return meta, nil
}

// algorithmForPublicKey infers JWT alg from a public key.
func algorithmForPublicKey(pub crypto.PublicKey) (string, error) {
	switch k := pub.(type) {
	case ed25519.PublicKey:
		return "EdDSA", nil
	case *ecdsa.PublicKey:
		if k.Params().Name != "P-256" {
			return "", fmt.Errorf("unsupported curve %s", k.Params().Name)
		}
		return "ES256", nil
	case *rsa.PublicKey:
		if k.Size()*8 < 2048 {
			return "", fmt.Errorf("rsa key too small")
		}
		return "RS256", nil
	default:
		return "", fmt.Errorf("unsupported key type %T", pub)
	}
}
