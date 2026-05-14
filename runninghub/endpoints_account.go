package runninghub

import "context"

// GetAccountStatus calls POST /uc/openapi/accountStatus.
// Note: request body uses `apikey` (lowercase), not `apiKey`.
func (c *Client) GetAccountStatus(ctx context.Context) (*AccountStatusData, error) {
	in := struct {
		APIKey string `json:"apikey"`
	}{APIKey: c.apiKey}

	var resp Envelope[AccountStatusData]
	if err := c.doJSON(ctx, "POST", "/uc/openapi/accountStatus", nil, nil, in, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, &APIError{Code: resp.Code, Message: resp.Msg, Details: resp.ErrorMessages}
	}
	if resp.Data == nil {
		return nil, &APIError{Code: resp.Code, Message: "empty data"}
	}
	return resp.Data, nil
}

// Deprecated: use GetAccountStatus.
func (c *Client) AccountStatus(ctx context.Context) (*AccountStatusData, error) {
	return c.GetAccountStatus(ctx)
}
