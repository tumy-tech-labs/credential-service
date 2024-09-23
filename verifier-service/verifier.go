package main

import (
	"errors"
	"time"
)

// VerifiableCredential represents the structure of the verifiable credential
type VerifiableCredential struct {
	Context           []string          `json:"@context"`
	Type              []string          `json:"type"`
	ID                string            `json:"id"`
	Issuer            string            `json:"issuer"`
	IssuanceDate      string            `json:"issuanceDate"`
	ExpirationDate    string            `json:"expirationDate"`
	CredentialSubject map[string]string `json:"credentialSubject"`
	Proof             Proof             `json:"proof,omitempty"`
}

// Proof represents the proof structure for digital signature
type Proof struct {
	Type               string `json:"type"`
	Created            string `json:"created"`
	ProofValue         string `json:"proofValue"`
	ProofPurpose       string `json:"proofPurpose"`
	VerificationMethod string `json:"verificationMethod"`
}

// VerifyCredential validates a Verifiable Credential against basic checks
func VerifyCredential(vc VerifiableCredential) (bool, error) {
	// Check if credential is expired
	issuanceDate, err := time.Parse(time.RFC3339, vc.IssuanceDate)
	if err != nil {
		return false, errors.New("invalid issuance date")
	}

	expirationDate, err := time.Parse(time.RFC3339, vc.ExpirationDate)
	if err != nil {
		return false, errors.New("invalid expiration date")
	}

	if time.Now().After(expirationDate) {
		return false, errors.New("credential has expired")
	}

	if issuanceDate.After(expirationDate) {
		return false, errors.New("issuance date is after expiration date")
	}

	// Signature verification logic (simplified for now)
	if vc.Proof.ProofValue == "" {
		return false, errors.New("missing proof or invalid signature")
	}

	// Further checks can be added here (e.g., DID verification, schema validation)

	return true, nil
}
