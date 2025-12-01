package crypto

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// LocalSigner implements Signer by loading keys from disk with strict permissions.
type LocalSigner struct {
	key       crypto.Signer
	alg       string
	jwk       []byte
	kid       string
	notBefore string
	notAfter  string
}

// NewLocalSignerFromFile loads a private key from disk and prepares a signing wrapper.
func NewLocalSignerFromFile(path string) (*LocalSigner, error) {
	if err := ensure0600(path); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}

	signer, jwk, alg, err := parseKeyMaterial(data)
	if err != nil {
		return nil, fmt.Errorf("parse key material: %w", err)
	}

	kid, err := KeyID(&LocalSigner{jwk: jwk})
	if err != nil {
		return nil, err
	}

	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat key file: %w", err)
	}

	nb := fi.ModTime().UTC().Format(time.RFC3339)
	return &LocalSigner{
		key:       signer,
		alg:       alg,
		jwk:       jwk,
		kid:       kid,
		notBefore: nb,
	}, nil
}

// NewEphemeralEd25519Signer returns a memory-only signer useful for tests.
func NewEphemeralEd25519Signer() (*LocalSigner, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519 key: %w", err)
	}
	jwk := map[string]string{
		"kty": "OKP",
		"crv": "Ed25519",
		"x":   base64.RawURLEncoding.EncodeToString(priv.Public().(ed25519.PublicKey)),
	}
	jwkBytes, _ := json.Marshal(jwk)
	kid, err := KeyID(&LocalSigner{jwk: jwkBytes})
	if err != nil {
		return nil, err
	}
	return &LocalSigner{key: priv, alg: "EdDSA", jwk: jwkBytes, kid: kid}, nil
}

func ensure0600(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("insecure permissions on %s; require 0600", path)
	}
	return nil
}

func parseKeyMaterial(data []byte) (crypto.Signer, []byte, string, error) {
	if pemBlock, _ := pem.Decode(data); pemBlock != nil {
		return parsePEMBlock(pemBlock)
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err == nil {
		if len(decoded) == ed25519.PrivateKeySize {
			priv := ed25519.PrivateKey(decoded)
			return finalizeKey(priv)
		}
		if key, err := x509.ParsePKCS8PrivateKey(decoded); err == nil {
			if signer, ok := key.(crypto.Signer); ok {
				return finalizeKey(signer)
			}
		}
	}

	return nil, nil, "", fmt.Errorf("unsupported key material")
}

func parsePEMBlock(block *pem.Block) (crypto.Signer, []byte, string, error) {
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		key, err = x509.ParseECPrivateKey(block.Bytes)
	}
	if err != nil {
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	}
	if err != nil {
		return nil, nil, "", fmt.Errorf("parse pem: %w", err)
	}

	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, nil, "", fmt.Errorf("key is not a signer")
	}
	return finalizeKey(signer)
}

func finalizeKey(signer crypto.Signer) (crypto.Signer, []byte, string, error) {
	switch k := signer.(type) {
	case ed25519.PrivateKey:
		jwk := map[string]string{
			"kty": "OKP",
			"crv": "Ed25519",
			"x":   base64.RawURLEncoding.EncodeToString(k.Public().(ed25519.PublicKey)),
		}
		jwkBytes, _ := json.Marshal(jwk)
		return signer, jwkBytes, "EdDSA", nil
	case *ecdsa.PrivateKey:
		if k.Curve != elliptic.P256() {
			return nil, nil, "", fmt.Errorf("unsupported curve %s", k.Curve.Params().Name)
		}
		jwk := map[string]string{
			"kty": "EC",
			"crv": "P-256",
			"x":   base64.RawURLEncoding.EncodeToString(k.X.Bytes()),
			"y":   base64.RawURLEncoding.EncodeToString(k.Y.Bytes()),
		}
		jwkBytes, _ := json.Marshal(jwk)
		return signer, jwkBytes, "ES256", nil
	case *rsa.PrivateKey:
		if k.Size()*8 < 2048 {
			return nil, nil, "", fmt.Errorf("rsa key too small")
		}
		return nil, nil, "", fmt.Errorf("rsa keys are not permitted for signing")
	default:
		return nil, nil, "", fmt.Errorf("unsupported key type %T", signer)
	}
}

// Sign signs payload bytes. For ECDSA keys SHA-256 is applied before signing.
func (l *LocalSigner) Sign(_ io.Reader, payload []byte, opts crypto.SignerOpts) ([]byte, error) {
	switch key := l.key.(type) {
	case ed25519.PrivateKey:
		return key.Sign(rand.Reader, payload, crypto.Hash(0))
	case *ecdsa.PrivateKey:
		if opts != nil && opts.HashFunc() == crypto.SHA256 && len(payload) == sha256.Size {
			return key.Sign(rand.Reader, payload, crypto.SHA256)
		}
		digest := sha256.Sum256(payload)
		return key.Sign(rand.Reader, digest[:], crypto.SHA256)
	default:
		return nil, fmt.Errorf("unsupported key type %T", l.key)
	}
}

// PublicJWK returns the cached JWK representation.
func (l *LocalSigner) PublicJWK() ([]byte, error) { return append([]byte{}, l.jwk...), nil }

// Algorithm returns the JOSE alg label.
func (l *LocalSigner) Algorithm() string { return l.alg }

// KeyID exposes the derived key identifier.
func (l *LocalSigner) KeyID() string { return l.kid }

// PublicKey returns the underlying public key.
func (l *LocalSigner) PublicKey() crypto.PublicKey { return l.key.Public() }

// Public satisfies the crypto.Signer interface.
func (l *LocalSigner) Public() crypto.PublicKey { return l.PublicKey() }

// ValidityWindow returns metadata derived from file timestamps.
func (l *LocalSigner) ValidityWindow() (string, string) {
	return l.notBefore, l.notAfter
}
