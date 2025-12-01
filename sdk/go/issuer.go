package sdk

import "context"

// IssueCredential calls the issuer endpoint to mint a new credential.
func (c *Client) IssueCredential(ctx context.Context, req IssueRequest) (*IssueResponse, error) {
	var resp IssueResponse
	if err := c.post(ctx, "/v1/credentials/issue", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DelegateCredential calls the delegation endpoint to mint a delegated credential.
func (c *Client) DelegateCredential(ctx context.Context, req DelegateRequest) (*DelegateResponse, error) {
	var resp DelegateResponse
	if err := c.post(ctx, "/v1/credentials/delegate", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
