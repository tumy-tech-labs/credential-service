package sdk

import (
	"encoding/json"
)

// Credential represents the structure of a verifiable credential.
type Credential struct {
	ID             string `json:"id"`
	Subject        string `json:"subject"`
	SubjectEmail   string `json:"subject_email"`
	SubjectPhone   string `json:"subject_phone"`
	IssueDate      string `json:"issue_date"`      // Adjust type if needed
	ExpirationDate string `json:"expiration_date"` // Adjust type if needed
}

// CredentialResponse represents the response structure from the credential issuance service.
type CredentialResponse struct {
	CredentialID string `json:"credential_id"`
	Issuer       string `json:"issuer"` // DID of the issuer
}

// IssueCredential issues a verifiable credential
func (c *Client) IssueCredential(credential Credential) (string, error) {
	// Serialize the credential to JSON
	credentialJSON, err := json.Marshal(credential)
	if err != nil {
		return "", err
	}

	// Send the request with the serialized JSON
	respBody, err := c.sendRequest("POST", "credentials", credentialJSON)
	if err != nil {
		return "", err
	}

	// Assuming the response body contains a string
	return string(respBody), nil
}
