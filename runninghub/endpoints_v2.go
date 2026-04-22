package runninghub

import (
	"context"
	"net/http"
)

// APIKeyList calls GET /openapi/v2/api-key/list.
func (c *Client) APIKeyList(ctx context.Context) ([]APIKeyItem, error) {
	var resp Envelope[[]APIKeyItem]
	if err := c.doJSON(ctx, http.MethodGet, "/openapi/v2/api-key/list", nil, nil, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, &APIError{Code: resp.Code, Message: resp.Msg, Details: resp.ErrorMessages}
	}
	if resp.Data == nil {
		return nil, nil
	}
	return *resp.Data, nil
}

// QueueStatus calls GET /openapi/v2/queue/status.
func (c *Client) QueueStatus(ctx context.Context) (*QueueStatusData, error) {
	var resp Envelope[QueueStatusData]
	if err := c.doJSON(ctx, http.MethodGet, "/openapi/v2/queue/status", nil, nil, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, &APIError{Code: resp.Code, Message: resp.Msg, Details: resp.ErrorMessages}
	}
	return resp.Data, nil
}

// QueryTaskV2 calls POST /openapi/v2/query.
// This endpoint returns the task object directly (not an envelope).
func (c *Client) QueryTaskV2(ctx context.Context, taskID string) (*QueryV2Response, error) {
	in := QueryV2Request{TaskID: taskID}
	var out QueryV2Response
	if err := c.doJSON(ctx, http.MethodPost, "/openapi/v2/query", nil, nil, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
