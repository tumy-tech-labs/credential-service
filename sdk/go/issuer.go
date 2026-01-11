package sdk

import "context"

// IssueCredential calls the issuer endpoint to mint a new credential.
func (c *Client) IssueCredential(ctx context.Context, req IssueRequest) (*IssueResponse, error) {
	var resp IssueResponse
	if err := c.post(ctx, c.IssuerURL, "/v1/credentials/issue", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
