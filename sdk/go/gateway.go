package sdk

import "context"

// GatewayAuthorize validates credentials and returns an authorization decision.
func (c *Client) GatewayAuthorize(ctx context.Context, req GatewayAuthorizeRequest) (*GatewayAuthorizeResponse, error) {
	var resp GatewayAuthorizeResponse
	if err := c.post(ctx, "/v1/gateway/authorize", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
