package runninghub

import (
	"context"
	"time"
)

const (
	DefaultPollInterval = 5 * time.Second
	DefaultWaitTimeout  = 30 * time.Minute
)

type RunningHubClient = Client

type NodeInput = WorkflowNodeInfo

type CreateTaskResponse = RunWorkflowResponse

type V2QueryResult = QueryV2Response

type ModelPricePreview = PricePreviewResponse

type TaskStatus string

const (
	TaskStatusQueued    TaskStatus = "QUEUED"
	TaskStatusRunning   TaskStatus = "RUNNING"
	TaskStatusSuccess   TaskStatus = "SUCCESS"
	TaskStatusFailed    TaskStatus = "FAILED"
	TaskStatusCancelled TaskStatus = "CANCELLED"
)

type RunOptions struct {
	NodeInfoList     []WorkflowNodeInfo
	AddMetadata      *bool
	WebhookURL       string
	InstanceType     string
	UsePersonalQueue *bool
	RetainSeconds    *int
}

type AIAppRunOptions struct {
	NodeInfoList     []AIAppNodeInfo
	WebhookURL       string
	InstanceType     string
	UsePersonalQueue *bool
	RetainSeconds    *int
}

type WaitForCompletionOptions struct {
	PollInterval   time.Duration
	Timeout        time.Duration
	OnStatusChange func(TaskStatus)
}

func NewClient(apiKey string, opts ...Option) (*RunningHubClient, error) {
	return New(apiKey, opts...)
}

func CreateClient(apiKey string, opts ...Option) (*RunningHubClient, error) {
	return New(apiKey, opts...)
}

func (c *Client) Run(ctx context.Context, workflowID string, options *RunOptions) (*CreateTaskResponse, error) {
	if options == nil {
		options = &RunOptions{}
	}

	return c.RunWorkflow(ctx, workflowID, RunWorkflowRequest{
		AddMetadata:      options.AddMetadata,
		NodeInfoList:     options.NodeInfoList,
		InstanceType:     options.InstanceType,
		UsePersonalQueue: options.UsePersonalQueue,
		RetainSeconds:    options.RetainSeconds,
		WebhookURL:       options.WebhookURL,
	})
}

func (c *Client) RunWithModifier(ctx context.Context, workflowID string, modifier *NodeModifier, options *RunOptions) (*CreateTaskResponse, error) {
	if options == nil {
		options = &RunOptions{}
	}
	if modifier != nil {
		options.NodeInfoList = modifier.ToWorkflowNodeInfoList()
	}
	return c.Run(ctx, workflowID, options)
}

func (c *Client) RunAIAppWithOptions(ctx context.Context, appID string, options *AIAppRunOptions) (*RunAIAppResponse, error) {
	if options == nil {
		options = &AIAppRunOptions{}
	}

	return c.RunAIApp(ctx, appID, RunAIAppRequest{
		NodeInfoList:     options.NodeInfoList,
		InstanceType:     options.InstanceType,
		UsePersonalQueue: options.UsePersonalQueue,
		RetainSeconds:    options.RetainSeconds,
		WebhookURL:       options.WebhookURL,
	})
}

func (c *Client) RunAIAppWithModifier(ctx context.Context, appID string, modifier *NodeModifier, options *AIAppRunOptions) (*RunAIAppResponse, error) {
	if options == nil {
		options = &AIAppRunOptions{}
	}
	if modifier != nil {
		options.NodeInfoList = modifier.ToAIAppNodeInfoList()
	}
	return c.RunAIAppWithOptions(ctx, appID, options)
}

func (c *Client) WaitForCompletion(ctx context.Context, taskID string, options *WaitForCompletionOptions) (*V2QueryResult, error) {
	if options == nil {
		options = &WaitForCompletionOptions{}
	}
	if ctx == nil {
		ctx = context.Background()
	}

	pollInterval := options.PollInterval
	if pollInterval <= 0 {
		pollInterval = DefaultPollInterval
	}

	if options.Timeout <= 0 {
		options.Timeout = DefaultWaitTimeout
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, options.Timeout)
	defer cancel()

	for {
		out, err := c.QueryTaskV2(ctx, taskID)
		if err != nil {
			return nil, err
		}

		if options.OnStatusChange != nil && out != nil {
			options.OnStatusChange(TaskStatus(out.Status))
		}

		if out != nil && isTerminalTaskStatus(out.Status) {
			return out, nil
		}

		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func (c *Client) PreviewModelPrice(ctx context.Context, endpoint string, payload any) (*ModelPricePreview, error) {
	return c.PricePreview(ctx, endpoint, payload)
}

func (c *Client) WaitForQueryV2Completion(ctx context.Context, taskID string, options *WaitForCompletionOptions) (*V2QueryResult, error) {
	return c.WaitForCompletion(ctx, taskID, options)
}