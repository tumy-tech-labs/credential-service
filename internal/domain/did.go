package domain

import (
	"crypto"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// DIDFromPublicKey returns a deterministic did:jwk representation for the provided public key.
func DIDFromPublicKey(pub crypto.PublicKey) (string, error) {
	switch k := pub.(type) {
	case ed25519.PublicKey:
		jwk := map[string]string{
			"kty": "OKP",
			"crv": "Ed25519",
			"x":   base64.RawURLEncoding.EncodeToString(k),
		}

		payload, err := json.Marshal(jwk)
		if err != nil {
			return "", fmt.Errorf("marshal jwk: %w", err)
		}

		return "did:jwk:" + base64.RawURLEncoding.EncodeToString(payload), nil
	default:
		return "", fmt.Errorf("unsupported public key type %T", pub)
	}
}
