package crypto

import "crypto"

// Signer abstracts signing operations so implementations can back onto local keys
// or remote KMS providers.
type Signer interface {
	crypto.Signer
	// PublicJWK returns the public key in JWK form.
	PublicJWK() ([]byte, error)
	// Algorithm returns the JOSE/JWT alg header value for this signer.
	Algorithm() string
}
