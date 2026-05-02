package runninghub

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

func normalizeAIAppPath(appIDOrPath string) (string, error) {
	p := strings.TrimSpace(appIDOrPath)
	if p == "" {
		return "", errors.New("empty ai app id")
	}
	if u, err := url.Parse(p); err == nil && u.Scheme != "" && u.Host != "" {
		if u.Path == "" {
			return "", errors.New("empty ai app path")
		}
		p = u.Path
	}
	if strings.Contains(p, "/") {
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		if strings.HasPrefix(p, "/openapi/") {
			return p, nil
		}
		return "/openapi/v2" + p, nil
	}
	return "/openapi/v2/run/ai-app/" + p, nil
}

func normalizeWorkflowPath(workflowIDOrPath string) (string, error) {
	p := strings.TrimSpace(workflowIDOrPath)
	if p == "" {
		return "", errors.New("empty workflow id")
	}
	if u, err := url.Parse(p); err == nil && u.Scheme != "" && u.Host != "" {
		if u.Path == "" {
			return "", errors.New("empty workflow path")
		}
		p = u.Path
	}
	if strings.Contains(p, "/") {
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		if strings.HasPrefix(p, "/openapi/") {
			return p, nil
		}
		return "/openapi/v2" + p, nil
	}
	return "/openapi/v2/run/workflow/" + p, nil
}

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

// RunAIApp submits an AI App task to /openapi/v2/run/ai-app/{appId}.
//
// Pass either:
// - an app ID like "2016796569449795585", or
// - a full path like "/openapi/v2/run/ai-app/2016796569449795585".
func (c *Client) RunAIApp(ctx context.Context, appIDOrPath string, req RunAIAppRequest) (*RunAIAppResponse, error) {
	p, err := normalizeAIAppPath(appIDOrPath)
	if err != nil {
		return nil, err
	}

	var out RunAIAppResponse
	if err := c.doJSON(ctx, http.MethodPost, p, nil, nil, req, &out); err != nil {
		return nil, err
	}
	if err := standardModelResponseError(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RunWorkflow submits a workflow task to /openapi/v2/run/workflow/{workflowId}.
//
// Pass either:
// - a workflow ID like "2037060865681264641", or
// - a full path like "/openapi/v2/run/workflow/2037060865681264641".
func (c *Client) RunWorkflow(ctx context.Context, workflowIDOrPath string, req RunWorkflowRequest) (*RunWorkflowResponse, error) {
	p, err := normalizeWorkflowPath(workflowIDOrPath)
	if err != nil {
		return nil, err
	}

	var out RunWorkflowResponse
	if err := c.doJSON(ctx, http.MethodPost, p, nil, nil, req, &out); err != nil {
		return nil, err
	}
	if err := standardModelResponseError(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
