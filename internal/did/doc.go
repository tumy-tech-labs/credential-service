package did

import (
	"crypto"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	cryptokit "github.com/bradtumy/credential-service/internal/crypto"
)

const (
	methodJWK    = "did:jwk"
	maxKeyRecord = 5
)

// DIDDocument captures a DID and its key rotation history.
type DIDDocument struct {
	DID          string     `json:"did"`
	Method       string     `json:"method"`
	Keys         []KeyEntry `json:"keys"`
	CurrentKeyID string     `json:"current_key_id"`
}

// KeyEntry tracks an individual key version for a DID.
type KeyEntry struct {
	ID            string                 `json:"id"`
	DID           string                 `json:"did"`
	Algorithm     string                 `json:"algorithm"`
	PublicJWK     map[string]interface{} `json:"public_jwk"`
	PrivateKeyPEM string                 `json:"private_key_pem,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
}

// NewDIDJWK creates a new did:jwk DIDDocument using the EdDSA algorithm.
func NewDIDJWK() (DIDDocument, error) {
	return NewDIDJWKWithAlg("EdDSA")
}

// NewDIDJWKWithAlg creates a new did:jwk DIDDocument using the provided algorithm.
func NewDIDJWKWithAlg(algorithm string) (DIDDocument, error) {
	keyEntry, err := generateKey(algorithm)
	if err != nil {
		return DIDDocument{}, err
	}

	doc := DIDDocument{
		DID:          keyEntry.DID,
		Method:       methodJWK,
		Keys:         []KeyEntry{keyEntry},
		CurrentKeyID: keyEntry.ID,
	}

	return doc, nil
}

// ExportDID serializes a DIDDocument to JSON.
func ExportDID(doc DIDDocument) ([]byte, error) {
	return jsonMarshal(doc)
}

// ImportDID deserializes a DIDDocument from JSON.
func ImportDID(data []byte) (DIDDocument, error) {
	var doc DIDDocument
	if err := jsonUnmarshal(data, &doc); err != nil {
		return DIDDocument{}, err
	}

	if doc.CurrentKeyID == "" || len(doc.Keys) == 0 {
		return DIDDocument{}, errors.New("invalid DID document: missing keys")
	}

	return doc, nil
}

// RotateKeys issues a new key for the DID and keeps a bounded history.
func RotateKeys(doc DIDDocument) (DIDDocument, error) {
	current, err := doc.CurrentKey()
	if err != nil {
		return DIDDocument{}, err
	}

	next, err := generateKey(current.Algorithm)
	if err != nil {
		return DIDDocument{}, err
	}

	doc.DID = next.DID
	doc.CurrentKeyID = next.ID
	doc.Keys = append(doc.Keys, next)

	if len(doc.Keys) > maxKeyRecord {
		doc.Keys = doc.Keys[len(doc.Keys)-maxKeyRecord:]
	}

	return doc, nil
}

// CurrentKey returns the active key entry for the DIDDocument.
func (d DIDDocument) CurrentKey() (KeyEntry, error) {
	for _, k := range d.Keys {
		if k.ID == d.CurrentKeyID {
			return k, nil
		}
	}
	return KeyEntry{}, fmt.Errorf("current key %s not found", d.CurrentKeyID)
}

// CurrentSigner returns the crypto.Signer for the active key.
func (d DIDDocument) CurrentSigner() (crypto.Signer, error) {
	current, err := d.CurrentKey()
	if err != nil {
		return nil, err
	}

	if current.PrivateKeyPEM == "" {
		return nil, errors.New("current key is missing private key material")
	}

	block, _ := pem.Decode([]byte(current.PrivateKeyPEM))
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	signer, ok := parsed.(crypto.Signer)
	if !ok {
		return nil, errors.New("parsed private key does not implement crypto.Signer")
	}

	return signer, nil
}

func generateKey(algorithm string) (KeyEntry, error) {
	kp, err := cryptokit.GenerateDIDJWK(algorithm)
	if err != nil {
		return KeyEntry{}, err
	}

	var public map[string]interface{}
	if err := jsonUnmarshal(kp.PublicJWK, &public); err != nil {
		return KeyEntry{}, fmt.Errorf("decode public jwk: %w", err)
	}

	return KeyEntry{
		ID:            uuid.NewString(),
		DID:           kp.DID,
		Algorithm:     kp.Algorithm,
		PublicJWK:     public,
		PrivateKeyPEM: string(kp.PrivateKeyPEM),
		CreatedAt:     time.Now().UTC(),
	}, nil
}

// jsonMarshal and jsonUnmarshal are small wrappers to aid testing/mocking.
var (
	jsonMarshal   = func(v interface{}) ([]byte, error) { return json.MarshalIndent(v, "", "  ") }
	jsonUnmarshal = json.Unmarshal
)
