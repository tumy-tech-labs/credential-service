package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client is the entry point for interacting with the credential-service APIs.
//
// Configure the service URLs explicitly or use the helpers NewLocalDevClient and
// NewClientFromEnv to populate sensible defaults.
type Client struct {
	IssuerURL   string
	VerifierURL string
	GatewayURL  string
	HTTPClient  *http.Client
}

// NewClientFromEnv builds a client using environment variables commonly used by
// local development and deployment scripts.
func NewClientFromEnv() *Client {
	return &Client{
		IssuerURL:   os.Getenv("ISSUER_URL"),
		VerifierURL: os.Getenv("VERIFIER_URL"),
		GatewayURL:  os.Getenv("GATEWAY_URL"),
	}
}

// NewLocalDevClient returns a client preconfigured to talk to services started
// via `make dev` or `docker-compose up`.
func NewLocalDevClient() *Client {
	return &Client{
		IssuerURL:   "http://localhost:8080",
		VerifierURL: "http://localhost:8081",
		GatewayURL:  "http://localhost:8081",
	}
}

// IssueVC mints a standard VC-JWT credential using the issuer service.
func (c *Client) IssueVC(ctx context.Context, req IssueRequest) (IssueResponse, error) {
	req.Format = "jwt-vc"
	return c.issue(ctx, req)
}

// IssueSDJWT mints an SD-JWT credential with disclosures using the issuer service.
func (c *Client) IssueSDJWT(ctx context.Context, req IssueRequest) (IssueResponse, error) {
	req.Format = "sd-jwt"
	return c.issue(ctx, req)
}

func (c *Client) issue(ctx context.Context, req IssueRequest) (IssueResponse, error) {
	var resp IssueResponse
	if err := c.post(ctx, c.IssuerURL, "/v1/credentials/issue", req, &resp); err != nil {
		return IssueResponse{}, err
	}
	return resp, nil
}

// Verify checks a credential or chain using the verifier service.
func (c *Client) Verify(ctx context.Context, credential string) (VerifyResponse, error) {
	return c.VerifyWithOptions(ctx, VerifyRequest{Credential: credential})
}

// VerifyWithOptions allows passing audience expectations and chains.
func (c *Client) VerifyWithOptions(ctx context.Context, req VerifyRequest) (VerifyResponse, error) {
	var resp VerifyResponse
	if err := c.post(ctx, c.VerifierURL, "/v1/credentials/verify", req, &resp); err != nil {
		return VerifyResponse{}, err
	}
	return resp, nil
}

// Authorize validates credentials and returns an allow/deny decision plus an
// optional synthetic JWT for downstream services.
func (c *Client) Authorize(ctx context.Context, req AuthorizeRequest) (AuthorizeResponse, error) {
	base := c.GatewayURL
	if base == "" {
		base = c.VerifierURL
	}

	var resp AuthorizeResponse
	if err := c.post(ctx, base, "/v1/gateway/authorize", req, &resp); err != nil {
		return AuthorizeResponse{}, err
	}
	return resp, nil
}

// DelegateCredential calls the delegation endpoint to mint a delegated credential.
func (c *Client) DelegateCredential(ctx context.Context, req DelegateRequest) (DelegateResponse, error) {
	var resp DelegateResponse
	if err := c.post(ctx, c.IssuerURL, "/v1/credentials/delegate", req, &resp); err != nil {
		return DelegateResponse{}, err
	}
	return resp, nil
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func buildURL(base, path string) (string, error) {
	if base == "" {
		return "", errors.New("base URL is required")
	}
	trimmed := strings.TrimSuffix(base, "/")
	if strings.HasPrefix(path, "/") {
		return trimmed + path, nil
	}
	return trimmed + "/" + path, nil
}

func (c *Client) post(ctx context.Context, baseURL, path string, payload interface{}, out interface{}) error {
	fullURL, err := buildURL(baseURL, path)
	if err != nil {
		return err
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrNetwork, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return decodeAPIError(resp)
	}

	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func decodeAPIError(resp *http.Response) error {
	var apiErr APIError
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Error != "" {
		return fmt.Errorf("%w: %s", mapStatusToError(resp.StatusCode), apiErr.Error)
	}
	return mapStatusToError(resp.StatusCode)
}
