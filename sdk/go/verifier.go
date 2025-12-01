package sdk

import "context"

// Verify submits a credential token or chain for verification.
func (c *Client) Verify(ctx context.Context, token string) (*VerifyResponse, error) {
	req := map[string]interface{}{
		"credential": token,
	}
	var resp VerifyResponse
	if err := c.post(ctx, "/v1/credentials/verify", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
