package runninghub

import (
	"encoding/json"
	"fmt"
)

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

// --- AI App v2 ---

type AIAppNodeInfo struct {
	NodeID      string `json:"nodeId"`
	FieldName   string `json:"fieldName"`
	FieldData   any    `json:"fieldData,omitempty"`
	FieldValue  any    `json:"fieldValue"`
	Description string `json:"description,omitempty"`
}

type RunAIAppRequest struct {
	NodeInfoList     []AIAppNodeInfo `json:"nodeInfoList"`
	InstanceType     string          `json:"instanceType,omitempty"`
	UsePersonalQueue *bool           `json:"usePersonalQueue,omitempty"`
	RetainSeconds    *int            `json:"retainSeconds,omitempty"`
	WebhookURL       string          `json:"webhookUrl,omitempty"`
}

type RunAIAppResponse = QueryV2Response

type WorkflowNodeInfo = AIAppNodeInfo

type RunWorkflowRequest struct {
	AddMetadata      *bool              `json:"addMetadata,omitempty"`
	NodeInfoList     []WorkflowNodeInfo `json:"nodeInfoList"`
	InstanceType     string             `json:"instanceType,omitempty"`
	UsePersonalQueue *bool              `json:"usePersonalQueue,omitempty"`
	RetainSeconds    *int               `json:"retainSeconds,omitempty"`
	WebhookURL       string             `json:"webhookUrl,omitempty"`
}

type RunWorkflowResponse = QueryV2Response

// --- Query v2 ---

type QueryV2Request struct {
	TaskID string `json:"taskId"`
}

type QueryV2ResultItem struct {
	URL        string `json:"url"`
	NodeID     string `json:"nodeId,omitempty"`
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
	APIKeyType        string `json:"apiKeyType"`
	ConcurrentLimit   int    `json:"concurrentLimit"`
	RunningCount      string `json:"runningCount"`
	QueuedCount       string `json:"queuedCount"`
	TotalCurrentTasks string `json:"totalCurrentTasks"`
}

// --- Comfy task ---

type CreateComfyTaskSimpleRequest struct {
	APIKey      string `json:"apiKey"`
	WorkflowID  string `json:"workflowId"`
	AddMetadata *bool  `json:"addMetadata,omitempty"`
	AccessPassword string `json:"accessPassword,omitempty"`
}

type CreateComfyTaskAdvancedRequest struct {
	APIKey           string             `json:"apiKey"`
	WorkflowID       string             `json:"workflowId,omitempty"`
	NodeInfoList     []WorkflowNodeInfo `json:"nodeInfoList,omitempty"`
	AddMetadata      *bool              `json:"addMetadata,omitempty"`
	WebhookURL       string             `json:"webhookUrl,omitempty"`
	Workflow         string             `json:"workflow,omitempty"`
	InstanceType     string             `json:"instanceType,omitempty"`
	UsePersonalQueue *bool              `json:"usePersonalQueue,omitempty"`
	RetainSeconds    *int               `json:"retainSeconds,omitempty"`
	AccessPassword   string             `json:"accessPassword,omitempty"`
}

type TaskCreateResponse struct {
	NetWssURL  string `json:"netWssUrl"`
	TaskID     int64  `json:"taskId"`
	ClientID   string `json:"clientId"`
	TaskStatus string `json:"taskStatus"`
	PromptTips string `json:"promptTips"`
}

func (r *TaskCreateResponse) UnmarshalJSON(data []byte) error {
	type alias struct {
		NetWssURL  string          `json:"netWssUrl"`
		TaskID     json.RawMessage `json:"taskId"`
		ClientID   string          `json:"clientId"`
		TaskStatus string          `json:"taskStatus"`
		PromptTips string          `json:"promptTips"`
	}

	var raw alias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	r.NetWssURL = raw.NetWssURL
	r.ClientID = raw.ClientID
	r.TaskStatus = raw.TaskStatus
	r.PromptTips = raw.PromptTips

	if len(raw.TaskID) == 0 || string(raw.TaskID) == "null" {
		r.TaskID = 0
		return nil
	}
	if err := json.Unmarshal(raw.TaskID, &r.TaskID); err == nil {
		return nil
	}

	var taskIDString string
	if err := json.Unmarshal(raw.TaskID, &taskIDString); err != nil {
		return err
	}
	var parsed int64
	if _, err := fmt.Sscanf(taskIDString, "%d", &parsed); err != nil {
		return err
	}
	r.TaskID = parsed
	return nil
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
	FileURL                string `json:"fileUrl"`
	FileType               string `json:"fileType"`
	TaskCostTime           string `json:"taskCostTime"`
	NodeID                 string `json:"nodeId"`
	ThirdPartyConsumeMoney any    `json:"thirdPartyConsumeMoney"`
	ConsumeMoney           any    `json:"consumeMoney"`
	ConsumeCoins           string `json:"consumeCoins"`
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

// --- Legacy AI App ---

type CreateAIAppTaskRequest struct {
	APIKey           string          `json:"apiKey"`
	WebappID         string          `json:"webappId"`
	NodeInfoList     []AIAppNodeInfo `json:"nodeInfoList,omitempty"`
	WebhookURL       string          `json:"webhookUrl,omitempty"`
	InstanceType     string          `json:"instanceType,omitempty"`
	AccessPassword   string          `json:"accessPassword,omitempty"`
	UsePersonalQueue *bool           `json:"usePersonalQueue,omitempty"`
	RetainSeconds    *int            `json:"retainSeconds,omitempty"`
}

type AIAppStatisticsInfo struct {
	LikeCount    string `json:"likeCount"`
	DownloadCount string `json:"downloadCount"`
	UseCount     string `json:"useCount"`
	PV           string `json:"pv"`
	CollectCount string `json:"collectCount"`
}

type AIAppDemoNodeInfo struct {
	NodeID        string `json:"nodeId"`
	NodeName      string `json:"nodeName,omitempty"`
	FieldName     string `json:"fieldName"`
	FieldValue    any    `json:"fieldValue"`
	FieldData     any    `json:"fieldData,omitempty"`
	FieldType     string `json:"fieldType,omitempty"`
	Description   string `json:"description,omitempty"`
	DescriptionEn string `json:"descriptionEn,omitempty"`
}

type AIAppDemoCover struct {
	ID           string `json:"id"`
	ObjName      string `json:"objName"`
	URL          string `json:"url"`
	ThumbnailURI string `json:"thumbnailUri"`
	ImageWidth   string `json:"imageWidth"`
	ImageHeight  string `json:"imageHeight"`
}

type AIAppDemoTag struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	NameEn  string `json:"nameEn,omitempty"`
	Labels  any    `json:"labels,omitempty"`
}

type AIAppAPICallDemoData struct {
	Curl            string               `json:"curl"`
	AccessEncrypted bool                 `json:"accessEncrypted"`
	WebappName      string               `json:"webappName"`
	StatisticsInfo  *AIAppStatisticsInfo `json:"statisticsInfo,omitempty"`
	NodeInfoList    []AIAppDemoNodeInfo  `json:"nodeInfoList,omitempty"`
	Covers          []AIAppDemoCover     `json:"covers,omitempty"`
	Tags            []AIAppDemoTag       `json:"tags,omitempty"`
}

// --- Webhook ---

type WebhookDetailData struct {
	ID               string `json:"id"`
	UserAPIKey       string `json:"userApiKey"`
	TaskID           string `json:"taskId"`
	WebhookURL       string `json:"webhookUrl"`
	Event            string `json:"event"`
	EventData        string `json:"eventData"`
	CallbackStatus   string `json:"callbackStatus"`
	CallbackResponse string `json:"callbackResponse"`
	RetryCount       int    `json:"retryCount"`
	CreateTime       string `json:"createTime"`
	UpdateTime       string `json:"updateTime"`
}

type RetryWebhookRequest struct {
	APIKey     string `json:"apiKey"`
	WebhookID  string `json:"webhookId"`
	WebhookURL string `json:"webhookUrl,omitempty"`
}

// --- Lora upload ---

type GetLoraUploadURLRequest struct {
	APIKey   string `json:"apiKey"`
	LoraName string `json:"loraName"`
	MD5Hex   string `json:"md5Hex"`
}

type LoraUploadURLData struct {
	FileName string `json:"fileName"`
	URL      string `json:"url"`
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
	ID                  string               `json:"id"`
	Version             string               `json:"version"`
	VersionResourceName string               `json:"versionResourceName"`
	BaseModel           string               `json:"baseModel"`
	BaseModelSubtype    string               `json:"baseModelSubtype"`
	TriggerWords        string               `json:"triggerWords"`
	Desc                string               `json:"desc"`
	PosterInfos         []ResourcePosterInfo `json:"posterInfos"`
}

type ResourceRecord struct {
	ID            string            `json:"id"`
	ResourceName  string            `json:"resourceName"`
	ResourceType  string            `json:"resourceType"`
	CreateTime    string            `json:"createTime"`
	Desc          string            `json:"desc"`
	NodeModelName string            `json:"nodeModelName"`
	PosterURL     string            `json:"posterUrl"`
	ThumbnailURL  string            `json:"thumbnailUrl"`
	Owner         ResourceOwner     `json:"owner"`
	Tags          []ResourceTag     `json:"tags"`
	Versions      []ResourceVersion `json:"versions"`
}

type ListPublicResourcesPage struct {
	Records     []ResourceRecord `json:"records"`
	Size        int              `json:"size"`
	Current     int              `json:"current"`
	Total       int              `json:"total"`
	Pages       int              `json:"pages"`
	HasNext     bool             `json:"hasNext"`
	HasPrevious bool             `json:"hasPrevious"`
}
