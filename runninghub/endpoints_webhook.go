package runninghub

import (
	"context"
	"net/http"
)

// GetWebhookDetail calls POST /task/openapi/getWebhookDetail.
func (c *Client) GetWebhookDetail(ctx context.Context, taskID string) (*WebhookDetailData, error) {
	in := struct {
		APIKey string `json:"apiKey"`
		TaskID string `json:"taskId"`
	}{
		APIKey: c.apiKey,
		TaskID: taskID,
	}

	var resp Envelope[WebhookDetailData]
	if err := c.doJSON(ctx, http.MethodPost, "/task/openapi/getWebhookDetail", nil, nil, in, &resp); err != nil {
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

// RetryWebhook calls POST /task/openapi/retryWebhook.
func (c *Client) RetryWebhook(ctx context.Context, webhookID, webhookURL string) error {
	in := RetryWebhookRequest{
		APIKey:     c.apiKey,
		WebhookID:  webhookID,
		WebhookURL: webhookURL,
	}

	var resp Envelope[any]
	if err := c.doJSON(ctx, http.MethodPost, "/task/openapi/retryWebhook", nil, nil, in, &resp); err != nil {
		return err
	}
	if resp.Code != 0 {
		return &APIError{Code: resp.Code, Message: resp.Msg, Details: resp.ErrorMessages}
	}
	return nil
}