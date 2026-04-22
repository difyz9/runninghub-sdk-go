package runninghub

import (
	"context"
	"encoding/json"
	"net/http"
)

// CreateComfyTaskSimple calls POST /task/openapi/create.
// This is the "简易" mode: it runs the workflow without changing any parameters.
func (c *Client) CreateComfyTaskSimple(ctx context.Context, workflowID string, addMetadata *bool) (*TaskCreateResponse, error) {
	in := CreateComfyTaskSimpleRequest{APIKey: c.apiKey, WorkflowID: workflowID, AddMetadata: addMetadata}
	var resp Envelope[TaskCreateResponse]
	if err := c.doJSON(ctx, http.MethodPost, "/task/openapi/create", nil, nil, in, &resp); err != nil {
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

// CancelComfyTask calls POST /task/openapi/cancel.
func (c *Client) CancelComfyTask(ctx context.Context, taskID string) error {
	in := CancelTaskRequest{APIKey: c.apiKey, TaskID: taskID}
	var resp Envelope[any]
	if err := c.doJSON(ctx, http.MethodPost, "/task/openapi/cancel", nil, nil, in, &resp); err != nil {
		return err
	}
	if resp.Code != 0 {
		return &APIError{Code: resp.Code, Message: resp.Msg, Details: resp.ErrorMessages}
	}
	return nil
}

// GetTaskStatusDeprecated calls POST /task/openapi/status (deprecated in docs).
// It returns data as one of: ["QUEUED","RUNNING","FAILED","SUCCESS"].
func (c *Client) GetTaskStatusDeprecated(ctx context.Context, taskID string) (string, error) {
	in := GetTaskStatusRequest{APIKey: c.apiKey, TaskID: taskID}
	var resp Envelope[string]
	if err := c.doJSON(ctx, http.MethodPost, "/task/openapi/status", nil, nil, in, &resp); err != nil {
		return "", err
	}
	if resp.Code != 0 {
		return "", &APIError{Code: resp.Code, Message: resp.Msg, Details: resp.ErrorMessages}
	}
	if resp.Data == nil {
		return "", nil
	}
	return *resp.Data, nil
}

// GetTaskOutputsDeprecated calls POST /task/openapi/outputs (deprecated in docs).
// This endpoint returns different `data` shapes depending on code:
// - code=0: data is []TaskOutputItem
// - code=805: data contains {failedReason: {...}}
// - code=804: data contains {netWssUrl: "..."}
func (c *Client) GetTaskOutputsDeprecated(ctx context.Context, taskID string) (items []TaskOutputItem, failed *TaskOutputsFailedReason, netWssURL string, err error) {
	in := GetTaskStatusRequest{APIKey: c.apiKey, TaskID: taskID}
	var raw struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/task/openapi/outputs", nil, nil, in, &raw); err != nil {
		return nil, nil, "", err
	}
	if raw.Code == 0 {
		if len(raw.Data) == 0 || string(raw.Data) == "null" {
			return nil, nil, "", nil
		}
		var out []TaskOutputItem
		if err := json.Unmarshal(raw.Data, &out); err != nil {
			return nil, nil, "", err
		}
		return out, nil, "", nil
	}

	// best-effort decode known shapes
	if len(raw.Data) > 0 && string(raw.Data) != "null" {
		var failData TaskOutputsFailureData
		if err := json.Unmarshal(raw.Data, &failData); err == nil && failData.FailedReason.ExceptionType != "" {
			return nil, &failData.FailedReason, "", &APIError{Code: raw.Code, Message: raw.Msg, Details: failData}
		}
		var running TaskOutputsRunningData
		if err := json.Unmarshal(raw.Data, &running); err == nil && running.NetWssURL != "" {
			return nil, nil, running.NetWssURL, &APIError{Code: raw.Code, Message: raw.Msg, Details: running}
		}
	}

	return nil, nil, "", &APIError{Code: raw.Code, Message: raw.Msg, Details: raw.Data}
}

// GetWorkflowJSONPrompt calls POST /api/openapi/getJsonApiFormat and returns the `prompt` string.
func (c *Client) GetWorkflowJSONPrompt(ctx context.Context, workflowID string) (string, error) {
	in := GetWorkflowJSONRequest{APIKey: c.apiKey, WorkflowID: workflowID}
	var resp Envelope[GetWorkflowJSONData]
	if err := c.doJSON(ctx, http.MethodPost, "/api/openapi/getJsonApiFormat", nil, nil, in, &resp); err != nil {
		return "", err
	}
	if resp.Code != 0 {
		return "", &APIError{Code: resp.Code, Message: resp.Msg, Details: resp.ErrorMessages}
	}
	if resp.Data == nil {
		return "", &APIError{Code: resp.Code, Message: "empty data"}
	}
	return resp.Data.Prompt, nil
}

// GetWorkflowJSON calls GetWorkflowJSONPrompt and parses the prompt JSON string.
func (c *Client) GetWorkflowJSON(ctx context.Context, workflowID string) (map[string]any, error) {
	prompt, err := c.GetWorkflowJSONPrompt(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(prompt), &m); err != nil {
		return nil, err
	}
	return m, nil
}
