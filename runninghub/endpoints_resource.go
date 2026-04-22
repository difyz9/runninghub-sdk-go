package runninghub

import (
	"context"
	"net/http"
)

// ListPublicResources calls POST /openapi/v2/resource/list.
// All request fields are optional.
func (c *Client) ListPublicResources(ctx context.Context, req ListPublicResourcesRequest) (*ListPublicResourcesPage, error) {
	var resp Envelope[ListPublicResourcesPage]
	if err := c.doJSON(ctx, http.MethodPost, "/openapi/v2/resource/list", nil, nil, req, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, &APIError{Code: resp.Code, Message: resp.Msg, Details: resp.ErrorMessages}
	}
	return resp.Data, nil
}
