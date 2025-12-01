package domain

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// VCJWTPayload represents a W3C VC-JWT compliant payload structure.
// Per the W3C spec, JWT claims are at the top level with the VC nested under 'vc' claim.
type VCJWTPayload struct {
	// JWT Standard Claims
	ISS string `json:"iss"` // Issuer DID
	SUB string `json:"sub"` // Subject DID
	JTI string `json:"jti"` // JWT ID
	IAT int64  `json:"iat"` // Issued At (Unix timestamp)
	EXP int64  `json:"exp"` // Expires At (Unix timestamp)
	NBF int64  `json:"nbf"` // Not Before (Unix timestamp)

	// W3C VC-JWT Required: Credential nested under 'vc' claim
	VC VerifiableCredential `json:"vc"`
}

// VerifiableCredential represents a W3C-compliant VC model used across all services.
// This is the canonical model - all other services must use this exact structure.
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
	JTI string `json:"jti,omitempty"` // JWT ID (maps to ID)
	ISS string `json:"iss,omitempty"` // Issuer (maps to issuer)
	SUB string `json:"sub,omitempty"` // Subject
	IAT int64  `json:"iat,omitempty"` // Issued At (Unix timestamp)
	EXP int64  `json:"exp,omitempty"` // Expires At (Unix timestamp)
	NBF int64  `json:"nbf,omitempty"` // Not Before (Unix timestamp)

	// Extension Fields for delegation and authorization
	Scope           []string `json:"scope,omitempty"`            // Authorized scopes
	DelegationChain []string `json:"delegation_chain,omitempty"` // Chain of delegating entities
	// Selective disclosure fields (SD-JWT)
	Format      string   `json:"format,omitempty"`
	SDDigests   []string `json:"_sd,omitempty"`
	SDDigestAlg string   `json:"_sd_alg,omitempty"`

	// Backward compatibility fields (deprecated but maintained for existing code)
	Claims    map[string]interface{} `json:"claims,omitempty"`     // Legacy claims field
	Subject   string                 `json:"subject,omitempty"`    // Legacy subject field (use SUB instead)
	IssuedAt  time.Time              `json:"issued_at,omitempty"`  // Legacy field (use IAT instead)
	ExpiresAt time.Time              `json:"expires_at,omitempty"` // Legacy field (use EXP instead)
}

// Proof structure for digital signature.
type Proof struct {
	Type               string `json:"type"`
	Created            string `json:"created"`
	ProofValue         string `json:"proofValue"`
	ProofPurpose       string `json:"proofPurpose"`
	VerificationMethod string `json:"verificationMethod"`
}

// CredentialRequest represents the issuance request payload.
type CredentialRequest struct {
	IssuerDid string                   `json:"issuerDid"`
	Subjects  []map[string]interface{} `json:"subject"`
}

// BaseSchema represents the structure of the base schema.
type BaseSchema struct {
	CredentialID   Property `json:"credentialID"`
	CredentialType Property `json:"credentialType"`
	IssueDate      Property `json:"issueDate"`
	ExpirationDate Property `json:"expirationDate"`
	Issuer         Property `json:"issuer"`
}

// Property represents a property of the schema.
type Property struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

// Schema represents a JSON Schema structure.
type Schema struct {
	Properties []Property `json:"properties"`
}

// LoadBaseSchema loads the base schema from a JSON file.
func LoadBaseSchema(filePath string) (BaseSchema, error) {
	var baseSchema BaseSchema
	file, err := osReadFile(filePath)
	if err != nil {
		return baseSchema, fmt.Errorf("failed to read base schema file: %w", err)
	}

	if err := json.Unmarshal(file, &baseSchema); err != nil {
		return baseSchema, fmt.Errorf("failed to unmarshal base schema JSON: %w", err)
	}

	return baseSchema, nil
}

// BuildCredential constructs a W3C-compliant credential with validation and consistent defaults.
func BuildCredential(id, issuer string, issuanceDate, expirationDate string, subject map[string]interface{}) (VerifiableCredential, error) {
	if err := ValidateDID(issuer); err != nil {
		return VerifiableCredential{}, fmt.Errorf("invalid issuer: %w", err)
	}
	if id == "" {
		return VerifiableCredential{}, NewFieldValidationError("id", "cannot be empty")
	}
	if err := ValidateCredentialSubject(subject); err != nil {
		return VerifiableCredential{}, err
	}

	// Parse and validate dates
	if _, err := time.Parse(time.RFC3339, issuanceDate); err != nil {
		return VerifiableCredential{}, NewFieldValidationError("issuanceDate", "must be RFC3339 format")
	}
	if expirationDate != "" {
		if _, err := time.Parse(time.RFC3339, expirationDate); err != nil {
			return VerifiableCredential{}, NewFieldValidationError("expirationDate", "must be RFC3339 format")
		}
	}

	return VerifiableCredential{
		Context:           []string{"https://www.w3.org/2018/credentials/v1"},
		Type:              []string{"VerifiableCredential"},
		ID:                id,
		Issuer:            issuer,
		IssuanceDate:      issuanceDate,
		ExpirationDate:    expirationDate,
		CredentialSubject: subject,
	}, nil
}

// AttachProof adds the proof information to a credential.
func AttachProof(vc VerifiableCredential, signature []byte) VerifiableCredential {
	vc.Proof = &Proof{
		Type:               "Ed25519Signature2018",
		Created:            time.Now().UTC().Format(time.RFC3339),
		ProofValue:         base64.StdEncoding.EncodeToString(signature),
		ProofPurpose:       "assertionMethod",
		VerificationMethod: vc.Issuer + "#keys-1",
	}
	return vc
}

// ParseEd25519PrivateKeyFromBase64 parses the base64-encoded Ed25519 private key.
func ParseEd25519PrivateKeyFromBase64(base64Key string) ([]byte, error) {
	privateKeyBytes, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 private key: %w", err)
	}
	return privateKeyBytes, nil
}

// SignCredential signs the credential payload using a proper crypto.Signer.
func SignCredential(ctx context.Context, signer crypto.Signer, credentialJSON []byte) ([]byte, error) {
	if signer == nil {
		return nil, errors.New("signer is required")
	}
	if len(credentialJSON) == 0 {
		return nil, errors.New("credential payload is empty")
	}

	// Sign the credential JSON payload
	signature, err := signer.Sign(rand.Reader, credentialJSON, crypto.Hash(0))
	if err != nil {
		return nil, fmt.Errorf("failed to sign credential: %w", err)
	}

	return signature, nil
}

// IssueBasicCredential creates and signs a W3C-compliant VC as JWT.
func IssueBasicCredential(issuerDID, subjectDID string, signer crypto.Signer, ttl time.Duration, claims map[string]interface{}) (string, error) {
	if err := ValidateDID(issuerDID); err != nil {
		return "", fmt.Errorf("invalid issuer DID: %w", err)
	}
	if err := ValidateDID(subjectDID); err != nil {
		return "", fmt.Errorf("invalid subject DID: %w", err)
	}
	if signer == nil {
		return "", NewValidationError("signer is required")
	}
	if ttl <= 0 {
		return "", NewValidationError("ttl must be positive")
	}

	issuedAt := time.Now().UTC()
	expiresAt := issuedAt.Add(ttl)
	credentialID := uuid.NewString()

	// Create W3C-compliant VC (without JWT claims - those go in wrapper)
	vc := VerifiableCredential{
		Context:           []string{"https://www.w3.org/2018/credentials/v1"},
		Type:              []string{"VerifiableCredential"},
		ID:                credentialID,
		Issuer:            issuerDID,
		IssuanceDate:      issuedAt.Format(time.RFC3339),
		ExpirationDate:    expiresAt.Format(time.RFC3339),
		CredentialSubject: claims,

		// Backward compatibility (populate legacy fields for internal use)
		Subject:   subjectDID,
		Claims:    claims,
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
	}

	// Wrap VC in JWT payload per W3C VC-JWT specification
	payload := VCJWTPayload{
		ISS: issuerDID,
		SUB: subjectDID,
		JTI: credentialID,
		IAT: issuedAt.Unix(),
		EXP: expiresAt.Unix(),
		NBF: issuedAt.Unix(),
		VC:  vc,
	}

	header := map[string]string{
		"alg": "EdDSA",
		"typ": "vc+jwt", // W3C VC-JWT specification requires vc+jwt type
	}

	headerSegment, err := encodeSegment(header)
	if err != nil {
		return "", fmt.Errorf("encode header: %w", err)
	}

	payloadSegment, err := encodeSegment(payload)
	if err != nil {
		return "", fmt.Errorf("encode payload: %w", err)
	}

	signingInput := headerSegment + "." + payloadSegment
	signature, err := signer.Sign(rand.Reader, []byte(signingInput), crypto.Hash(0))
	if err != nil {
		return "", fmt.Errorf("sign payload: %w", err)
	}

	signatureSegment := base64.RawURLEncoding.EncodeToString(signature)

	return signingInput + "." + signatureSegment, nil
}

// SDJWT represents an issued selective-disclosure credential.
type SDJWT struct {
	Token       string   `json:"token"`
	Disclosures []string `json:"disclosures"`
}

// IssueSDJWTCredential issues an SD-JWT with salted digests for each claim.
func IssueSDJWTCredential(issuerDID, subjectDID string, signer crypto.Signer, ttl time.Duration, claims map[string]interface{}) (*SDJWT, error) {
	if claims == nil {
		claims = map[string]interface{}{}
	}

	// Build digests
	disclosures := make([]string, 0, len(claims))
	digests := make([]string, 0, len(claims))
	for key, value := range claims {
		disclosure, digest, err := buildDisclosure(key, value)
		if err != nil {
			return nil, fmt.Errorf("build disclosure: %w", err)
		}
		disclosures = append(disclosures, disclosure)
		digests = append(digests, digest)
	}

	sdClaims := map[string]interface{}{
		"_sd":     digests,
		"_sd_alg": "sha256",
	}

	token, err := IssueBasicCredentialWithFormat(issuerDID, subjectDID, signer, ttl, sdClaims, "sd-jwt")
	if err != nil {
		return nil, err
	}

	return &SDJWT{Token: token, Disclosures: disclosures}, nil
}

// IssueBasicCredentialWithFormat mirrors IssueBasicCredential but annotates the VC format.
func IssueBasicCredentialWithFormat(issuerDID, subjectDID string, signer crypto.Signer, ttl time.Duration, claims map[string]interface{}, format string) (string, error) {
	if claims == nil {
		claims = map[string]interface{}{}
	}

	token, err := IssueBasicCredential(issuerDID, subjectDID, signer, ttl, claims)
	if err != nil {
		return "", err
	}

	// Rebuild payload with format annotation
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return token, nil
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return token, nil
	}

	// Unmarshal as VCJWTPayload to preserve JWT wrapper structure
	var vcjwtPayload VCJWTPayload
	if err := json.Unmarshal(payloadBytes, &vcjwtPayload); err != nil {
		return token, nil
	}

	// Set format on the nested VC
	vcjwtPayload.VC.Format = format

	// Re-encode the full VCJWTPayload structure
	newPayload, err := encodeSegment(vcjwtPayload)
	if err != nil {
		return token, nil
	}

	signingInput := parts[0] + "." + newPayload
	signature, err := signer.Sign(rand.Reader, []byte(signingInput), crypto.Hash(0))
	if err != nil {
		return "", fmt.Errorf("sign payload: %w", err)
	}

	parts[1] = newPayload
	parts[2] = base64.RawURLEncoding.EncodeToString(signature)
	return strings.Join(parts, "."), nil
}

func buildDisclosure(key string, value interface{}) (string, string, error) {
	saltBytes := make([]byte, 12)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", "", fmt.Errorf("generate salt: %w", err)
	}
	salt := base64.RawURLEncoding.EncodeToString(saltBytes)

	disclosureArray := []interface{}{salt, key, value}
	disclosureBytes, err := json.Marshal(disclosureArray)
	if err != nil {
		return "", "", fmt.Errorf("marshal disclosure: %w", err)
	}

	disclosure := base64.RawURLEncoding.EncodeToString(disclosureBytes)
	digest := computeDisclosureDigest(disclosureBytes)

	return disclosure, digest, nil
}

func computeDisclosureDigest(disclosure []byte) string {
	h := sha256.Sum256(disclosure)
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func encodeSegment(data interface{}) (string, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal segment: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(jsonBytes), nil
}

// osReadFile is abstracted for testing.
var osReadFile = func(path string) ([]byte, error) {
	return osReadFileFn(path)
}

// osReadFileFn allows substitution during tests.
var osReadFileFn = defaultReadFile

func defaultReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// ValidationError represents input validation errors with better developer experience.
type ValidationError struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

func (e ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

func NewValidationError(message string) ValidationError {
	return ValidationError{Message: message, Code: "validation_error"}
}

func NewFieldValidationError(field, message string) ValidationError {
	return ValidationError{Field: field, Message: message, Code: "field_validation_error"}
}

// ValidateDID validates DID format according to W3C DID specification.
func ValidateDID(did string) error {
	if did == "" {
		return NewValidationError("DID cannot be empty")
	}
	if len(did) > 1000 {
		return NewValidationError("DID too long (max 1000 characters)")
	}
	if !strings.HasPrefix(did, "did:") {
		return NewValidationError("DID must start with 'did:'")
	}

	// Basic format: did:method:method-specific-id
	parts := strings.SplitN(did, ":", 3)
	if len(parts) < 3 {
		return NewValidationError("DID must have format 'did:method:method-specific-id'")
	}

	method := parts[1]
	if method == "" {
		return NewValidationError("DID method cannot be empty")
	}

	methodSpecificID := parts[2]
	if methodSpecificID == "" {
		return NewValidationError("DID method-specific-id cannot be empty")
	}

	return nil
}

// ValidateCredentialSubject validates the credential subject field.
func ValidateCredentialSubject(subject map[string]interface{}) error {
	if subject == nil {
		return NewFieldValidationError("credentialSubject", "cannot be null")
	}
	if len(subject) == 0 {
		return NewFieldValidationError("credentialSubject", "cannot be empty")
	}
	return nil
}

// ValidateCredentialRequest validates a complete credential request.
func ValidateCredentialRequest(req CredentialRequest) error {
	if err := ValidateDID(req.IssuerDid); err != nil {
		return fmt.Errorf("invalid issuer DID: %w", err)
	}
	if len(req.Subjects) == 0 {
		return NewFieldValidationError("subjects", "at least one subject is required")
	}
	for i, subject := range req.Subjects {
		if err := ValidateCredentialSubject(subject); err != nil {
			return fmt.Errorf("invalid subject[%d]: %w", i, err)
		}
	}
	return nil
}
