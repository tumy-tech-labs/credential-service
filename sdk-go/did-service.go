package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// DIDRequest represents the request payload for creating a DID
type DIDRequest struct {
	Type           string `json:"type"`
	OrganizationID string `json:"organization_id"`
}

// Define the structure for the response when creating a DID
type DIDResponse struct {
	ID        string `json:"id"`
	PublicKey []struct {
		ID              string `json:"id"`
		Type            string `json:"type"`
		Controller      string `json:"controller"`
		PublicKeyBase58 string `json:"publicKeyBase58"`
	} `json:"publicKey"`
	CreatedAt      string `json:"createdAt"`
	OrganizationID string `json:"organization_id"`
}

// CreateDID creates a new DID and returns the DIDResponse.
func (c *Client) CreateDID() (DIDResponse, error) {
	url := fmt.Sprintf("%s/v1/dids", c.DIDServiceURL)

	// Construct the request payload
	didRequest := DIDRequest{
		Type:           "organization", // This is the type expected by the service
		OrganizationID: "orgABC",       // Replace with actual organization ID if necessary
	}

	// Serialize the request payload to JSON
	requestBody, err := json.Marshal(didRequest)
	if err != nil {
		return DIDResponse{}, err
	}

	// Send the request
	respBody, err := c.sendRequest(http.MethodPost, url, requestBody)
	if err != nil {
		return DIDResponse{}, err
	}

	var didResp DIDResponse
	// Unmarshal the response body into the DIDResponse struct
	if err := json.Unmarshal(respBody, &didResp); err != nil {
		return DIDResponse{}, err
	}

	return didResp, nil
}

// ResolveDID resolves a given DID and returns the resolved data.
func (c *Client) ResolveDID(did string) (string, error) {
	url := fmt.Sprintf("%s/v1/dids/resolver?did=%s", c.ResolverServiceURL, did)

	respBody, err := c.sendRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	return string(respBody), nil // Return the response as a string or modify based on the expected format
}
