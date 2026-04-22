package runninghub

import "encoding/json"

type Envelope[T any] struct {
	Code          int             `json:"code"`
	Msg           string          `json:"msg"`
	ErrorMessages json.RawMessage `json:"errorMessages,omitempty"`
	Data          *T              `json:"data"`
}

type MessageEnvelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    *T     `json:"data"`
}

// --- Upload v2 ---

type UploadBinaryData struct {
	Type        string `json:"type"`
	DownloadURL string `json:"download_url"`
	FileName    string `json:"fileName"`
	Size        string `json:"size"`
}

// --- Query v2 ---

type QueryV2Request struct {
	TaskID string `json:"taskId"`
}

type QueryV2ResultItem struct {
	URL        string `json:"url"`
	OutputType string `json:"outputType"`
	Text       string `json:"text,omitempty"`
}

type TaskUsage struct {
	ThirdPartyConsumeMoney *string `json:"thirdPartyConsumeMoney,omitempty"`
	ConsumeMoney           *string `json:"consumeMoney,omitempty"`
	ConsumeCoins           *string `json:"consumeCoins,omitempty"`
	TaskCostTime           *string `json:"taskCostTime,omitempty"`
}

type QueryV2Response struct {
	TaskID       string              `json:"taskId"`
	Status       string              `json:"status"`
	ErrorCode    string              `json:"errorCode"`
	ErrorMessage string              `json:"errorMessage"`
	Results      []QueryV2ResultItem `json:"results"`
	ClientID     string              `json:"clientId"`
	PromptTips   string              `json:"promptTips"`
	FailedReason map[string]any      `json:"failedReason,omitempty"`
	Usage        *TaskUsage          `json:"usage,omitempty"`
}

// --- Account ---

type AccountStatusData struct {
	RemainCoins       string `json:"remainCoins"`
	CurrentTaskCounts string `json:"currentTaskCounts"`
	RemainMoney       any    `json:"remainMoney"`
	Currency          any    `json:"currency"`
	APIType           string `json:"apiType"`
}

// --- APIKey list ---

type APIKeyItem struct {
	Key            string  `json:"key"`
	APIKeyName     any     `json:"apiKeyName"`
	Status         int     `json:"status"`
	QuotaLimit     any     `json:"quotaLimit"`
	QuotaUsed      float64 `json:"quotaUsed"`
	Visible        bool    `json:"visible"`
	ExpireAt       any     `json:"expireAt"`
	ExpireInMinute any     `json:"expireInMinute"`
	CreatedAt      string  `json:"createdAt"`
}

// --- Queue status ---

type QueueStatusData struct {
	APIKeyType       string `json:"apiKeyType"`
	ConcurrentLimit  int    `json:"concurrentLimit"`
	RunningCount     string `json:"runningCount"`
	QueuedCount      string `json:"queuedCount"`
	TotalCurrentTasks string `json:"totalCurrentTasks"`
}

// --- Comfy task ---

type CreateComfyTaskSimpleRequest struct {
	APIKey      string `json:"apiKey"`
	WorkflowID  string `json:"workflowId"`
	AddMetadata *bool  `json:"addMetadata,omitempty"`
}

type TaskCreateResponse struct {
	NetWssURL  string `json:"netWssUrl"`
	TaskID     int64  `json:"taskId"`
	ClientID   string `json:"clientId"`
	TaskStatus string `json:"taskStatus"`
	PromptTips string `json:"promptTips"`
}

type CancelTaskRequest struct {
	APIKey string `json:"apiKey"`
	TaskID string `json:"taskId"`
}

type GetTaskStatusRequest struct {
	APIKey string `json:"apiKey"`
	TaskID string `json:"taskId"`
}

type TaskOutputItem struct {
	FileURL              string `json:"fileUrl"`
	FileType             string `json:"fileType"`
	TaskCostTime          string `json:"taskCostTime"`
	NodeID               string `json:"nodeId"`
	ThirdPartyConsumeMoney any   `json:"thirdPartyConsumeMoney"`
	ConsumeMoney         any    `json:"consumeMoney"`
	ConsumeCoins         string `json:"consumeCoins"`
}

type TaskOutputsFailedReason struct {
	CurrentOutputs   any    `json:"current_outputs"`
	ExceptionType    string `json:"exception_type"`
	NodeName         string `json:"node_name"`
	CurrentInputs    any    `json:"current_inputs"`
	Traceback        any    `json:"traceback"`
	NodeID           string `json:"node_id"`
	ExceptionMessage string `json:"exception_message"`
}

type TaskOutputsFailureData struct {
	FailedReason TaskOutputsFailedReason `json:"failedReason"`
}

type TaskOutputsRunningData struct {
	NetWssURL string `json:"netWssUrl"`
}

// --- Workflow json ---

type GetWorkflowJSONRequest struct {
	APIKey     string `json:"apiKey"`
	WorkflowID string `json:"workflowId"`
}

type GetWorkflowJSONData struct {
	Prompt string `json:"prompt"`
}

// --- Public resources ---

type ResourceType string

const (
	ResourceTypeUNET       ResourceType = "UNET"
	ResourceTypeCHECKPOINT ResourceType = "CHECKPOINT"
	ResourceTypeLORA       ResourceType = "LORA"
	ResourceTypeGGUF       ResourceType = "GGUF"
)

type ListPublicResourcesRequest struct {
	ResourceType ResourceType `json:"resourceType,omitempty"`
	ResourceName string       `json:"resourceName,omitempty"`
	BaseModels   []string     `json:"baseModels,omitempty"`
	Tags         []int64      `json:"tags,omitempty"`
	Current      int64        `json:"current,omitempty"`
	Size         int64        `json:"size,omitempty"`
}

type ResourceOwner struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type ResourceTag struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type ResourcePosterInfo struct {
	PosterURL    string `json:"posterUrl"`
	ThumbnailURL string `json:"thumbnailUrl"`
	ImageWidth   int    `json:"imageWidth"`
	ImageHeight  int    `json:"imageHeight"`
}

type ResourceVersion struct {
	ID                 string              `json:"id"`
	Version            string              `json:"version"`
	VersionResourceName string             `json:"versionResourceName"`
	BaseModel          string              `json:"baseModel"`
	BaseModelSubtype   string              `json:"baseModelSubtype"`
	TriggerWords       string              `json:"triggerWords"`
	Desc              string              `json:"desc"`
	PosterInfos        []ResourcePosterInfo `json:"posterInfos"`
}

type ResourceRecord struct {
	ID            string          `json:"id"`
	ResourceName  string          `json:"resourceName"`
	ResourceType  string          `json:"resourceType"`
	CreateTime    string          `json:"createTime"`
	Desc          string          `json:"desc"`
	NodeModelName string          `json:"nodeModelName"`
	PosterURL     string          `json:"posterUrl"`
	ThumbnailURL  string          `json:"thumbnailUrl"`
	Owner         ResourceOwner   `json:"owner"`
	Tags          []ResourceTag   `json:"tags"`
	Versions      []ResourceVersion `json:"versions"`
}

type ListPublicResourcesPage struct {
	Records     []ResourceRecord `json:"records"`
	Size        int             `json:"size"`
	Current     int             `json:"current"`
	Total       int             `json:"total"`
	Pages       int             `json:"pages"`
	HasNext     bool            `json:"hasNext"`
	HasPrevious bool            `json:"hasPrevious"`
}
