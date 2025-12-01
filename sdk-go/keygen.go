package sdk

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
)

// DIDKeyPair represents a generated DID with its associated key material.
type DIDKeyPair struct {
	DID           string        // The did:jwk identifier
	Algorithm     string        // EdDSA or ES256
	PublicJWK     string        // Public key in JWK format (JSON)
	PrivateKeyPEM string        // Private key in PEM format (PKCS8)
	PrivateKey    crypto.Signer // The actual private key for signing
}

// GenerateDIDJWK generates a new key pair and returns a did:jwk identifier
// along with the key material. Supported algorithms: "EdDSA" (Ed25519) and "ES256" (ECDSA P-256).
//
// Example:
//
//	keypair, err := sdk.GenerateDIDJWK("EdDSA")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("DID:", keypair.DID)
//	// Save private key to file if needed
//	err = os.WriteFile("key.pem", []byte(keypair.PrivateKeyPEM), 0600)
func GenerateDIDJWK(algorithm string) (*DIDKeyPair, error) {
	switch algorithm {
	case "EdDSA":
		return generateEd25519DID()
	case "ES256":
		return generateES256DID()
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s (supported: EdDSA, ES256)", algorithm)
	}
}

// generateEd25519DID generates an Ed25519 key pair and formats it as did:jwk.
func generateEd25519DID() (*DIDKeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519 key: %w", err)
	}

	// Create public JWK
	publicJWK := map[string]string{
		"kty": "OKP",
		"crv": "Ed25519",
		"x":   base64.RawURLEncoding.EncodeToString(pub),
	}

	publicJWKBytes, err := json.Marshal(publicJWK)
	if err != nil {
		return nil, fmt.Errorf("marshal public jwk: %w", err)
	}

	// Create DID by base64url-encoding the public JWK
	did := "did:jwk:" + base64.RawURLEncoding.EncodeToString(publicJWKBytes)

	// Convert private key to PKCS8 PEM format
	privPKCS8, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("marshal private key: %w", err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privPKCS8,
	})

	return &DIDKeyPair{
		DID:           did,
		Algorithm:     "EdDSA",
		PublicJWK:     string(publicJWKBytes),
		PrivateKeyPEM: string(privPEM),
		PrivateKey:    priv,
	}, nil
}

// generateES256DID generates an ECDSA P-256 key pair and formats it as did:jwk.
func generateES256DID() (*DIDKeyPair, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ecdsa key: %w", err)
	}

	// Create public JWK
	publicJWK := map[string]string{
		"kty": "EC",
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(priv.X.Bytes()),
		"y":   base64.RawURLEncoding.EncodeToString(priv.Y.Bytes()),
	}

	publicJWKBytes, err := json.Marshal(publicJWK)
	if err != nil {
		return nil, fmt.Errorf("marshal public jwk: %w", err)
	}

	// Create DID by base64url-encoding the public JWK
	did := "did:jwk:" + base64.RawURLEncoding.EncodeToString(publicJWKBytes)

	// Convert private key to PKCS8 PEM format
	privPKCS8, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("marshal private key: %w", err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privPKCS8,
	})

	return &DIDKeyPair{
		DID:           did,
		Algorithm:     "ES256",
		PublicJWK:     string(publicJWKBytes),
		PrivateKeyPEM: string(privPEM),
		PrivateKey:    priv,
	}, nil
}
