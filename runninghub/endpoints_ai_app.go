package runninghub

import (
	"context"
	"net/http"
	"net/url"
)

// RunLegacyAIApp calls POST /task/openapi/ai-app/run.
//
// This is the legacy AI App entry from the older task/openapi API surface.
func (c *Client) RunLegacyAIApp(ctx context.Context, req CreateAIAppTaskRequest) (*TaskCreateResponse, error) {
	req.APIKey = c.apiKey
	var resp Envelope[TaskCreateResponse]
	if err := c.doJSON(ctx, http.MethodPost, "/task/openapi/ai-app/run", nil, nil, req, &resp); err != nil {
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

// Deprecated: use RunLegacyAIApp.
func (c *Client) CreateAIAppTask(ctx context.Context, req CreateAIAppTaskRequest) (*TaskCreateResponse, error) {
	return c.RunLegacyAIApp(ctx, req)
}

// GetAIAppAPICallDemo calls GET /api/webapp/apiCallDemo.
func (c *Client) GetAIAppAPICallDemo(ctx context.Context, webappID string) (*AIAppAPICallDemoData, error) {
	query := url.Values{}
	query.Set("apiKey", c.apiKey)
	query.Set("webappId", webappID)

	var resp Envelope[AIAppAPICallDemoData]
	if err := c.doJSON(ctx, http.MethodGet, "/api/webapp/apiCallDemo", query, nil, nil, &resp); err != nil {
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