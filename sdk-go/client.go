package sdk

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type Client struct {
	DIDServiceURL      string
	ResolverServiceURL string
	HTTPClient         *http.Client
}

// NewClient creates a new instance of Client with an initialized HTTP client.
func NewClient(didServiceURL, resolverServiceURL string) *Client {
	return &Client{
		DIDServiceURL:      didServiceURL,
		ResolverServiceURL: resolverServiceURL,
		HTTPClient:         &http.Client{},
	}
}

// sendRequest is a helper method to send HTTP requests.
func (c *Client) sendRequest(method, url string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json") // Set the content type to JSON

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Log raw response for debugging purposes
	log.Printf("Raw response body: %s", respBody)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, respBody)
	}

	return respBody, nil
}
