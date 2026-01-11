package main

import (
	"time"
)

// VerifiableCredential represents a W3C-compliant VC model
// This matches the canonical structure from the main service
type VerifiableCredential struct {
	// W3C Standard Fields
	Context           []string               `json:"@context"`
	Type              []string               `json:"type"`
	ID                string                 `json:"id"`
	Issuer            string                 `json:"issuer"`
	IssuanceDate      string                 `json:"issuanceDate"`
	ExpirationDate    string                 `json:"expirationDate,omitempty"`
	CredentialSubject map[string]interface{} `json:"credentialSubject"`
	Proof             *Proof                 `json:"proof,omitempty"`

	// JWT Standard Claims (when used as JWT)
	JTI string `json:"jti,omitempty"` // JWT ID
	ISS string `json:"iss,omitempty"` // Issuer
	SUB string `json:"sub,omitempty"` // Subject
	IAT int64  `json:"iat,omitempty"` // Issued At
	EXP int64  `json:"exp,omitempty"` // Expires At
	NBF int64  `json:"nbf,omitempty"` // Not Before

	// Extension Fields
	Scope           []string `json:"scope,omitempty"`
	DelegationChain []string `json:"delegation_chain,omitempty"`

	// Backward compatibility fields
	Claims    map[string]interface{} `json:"claims,omitempty"`
	Subject   string                 `json:"subject,omitempty"`
	IssuedAt  time.Time              `json:"issued_at,omitempty"`
	ExpiresAt time.Time              `json:"expires_at,omitempty"`
}

// Proof structure for digital signature
type Proof struct {
	Type               string `json:"type"`
	Created            string `json:"created"`
	ProofValue         string `json:"proofValue"`
	ProofPurpose       string `json:"proofPurpose"`
	VerificationMethod string `json:"verificationMethod"`
}

// Placeholder for additional service-related logic
