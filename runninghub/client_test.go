package runninghub

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAccountStatus_UsesApikeyField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/uc/openapi/accountStatus" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer testkey" {
			t.Fatalf("Authorization=%q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if _, ok := body["apikey"]; !ok {
			t.Fatalf("expected body.apikey, got: %#v", body)
		}
		if _, ok := body["apiKey"]; ok {
			t.Fatalf("did not expect body.apiKey, got: %#v", body)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"msg":  "success",
			"data": map[string]any{"remainCoins": "1", "currentTaskCounts": "0", "apiType": "V2"},
		})
	}))
	defer srv.Close()

	c, err := New("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}
	data, err := c.AccountStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if data.RemainCoins != "1" {
		t.Fatalf("RemainCoins=%q", data.RemainCoins)
	}
}

func TestCanonicalGetterAliases(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/uc/openapi/accountStatus":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["apikey"] != "testkey" {
				t.Fatalf("apikey=%v", body["apikey"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"msg":  "success",
				"data": map[string]any{"remainCoins": "1", "currentTaskCounts": "0", "apiType": "V2"},
			})
		case "/openapi/v2/api-key/list":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"msg":  "success",
				"data": []map[string]any{{"key": "k1", "status": 1, "quotaUsed": 0.0, "visible": true, "createdAt": "2026-01-01"}},
			})
		case "/openapi/v2/queue/status":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"msg":  "success",
				"data": map[string]any{"apiKeyType": "corp", "concurrentLimit": 2, "runningCount": "1", "queuedCount": "3", "totalCurrentTasks": "4"},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c, err := New("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	account, err := c.GetAccountStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if account.APIType != "V2" {
		t.Fatalf("APIType=%q", account.APIType)
	}

	keys, err := c.ListAPIKeys(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0].Key != "k1" {
		t.Fatalf("keys=%#v", keys)
	}

	queue, err := c.GetQueueStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if queue.QueuedCount != "3" {
		t.Fatalf("QueuedCount=%q", queue.QueuedCount)
	}
}

func TestUploadBinaryReader_Multipart(t *testing.T) {
	const fileContent = "hello"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/openapi/v2/media/upload/binary" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Fatal(err)
		}
		if mediaType != "multipart/form-data" {
			t.Fatalf("Content-Type=%q", mediaType)
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		part, err := mr.NextPart()
		if err != nil {
			t.Fatal(err)
		}
		if part.FormName() != "file" {
			t.Fatalf("form name=%q", part.FormName())
		}
		if part.FileName() != "a.txt" {
			t.Fatalf("filename=%q", part.FileName())
		}
		b, _ := io.ReadAll(part)
		if string(b) != fileContent {
			t.Fatalf("content=%q", string(b))
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    0,
			"message": "success",
			"data": map[string]any{
				"type":         "image",
				"download_url": "https://example.com/x",
				"fileName":     "a.txt",
				"size":         "5",
			},
		})
	}))
	defer srv.Close()

	c, err := New("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}
	out, err := c.UploadBinaryReader(context.Background(), strings.NewReader(fileContent), "a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if out.FileName != "a.txt" {
		t.Fatalf("FileName=%q", out.FileName)
	}
}

func TestCreateClientAliasAndRunWithModifier(t *testing.T) {
	usePersonalQueue := false
	addMetadata := true

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/openapi/v2/run/workflow/wf-123" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		nodeInfoList, ok := body["nodeInfoList"].([]any)
		if !ok || len(nodeInfoList) != 3 {
			t.Fatalf("nodeInfoList=%#v", body["nodeInfoList"])
		}
		if body["addMetadata"] != true {
			t.Fatalf("addMetadata=%#v", body["addMetadata"])
		}
		if body["usePersonalQueue"] != false {
			t.Fatalf("usePersonalQueue=%#v", body["usePersonalQueue"])
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"taskId":       "task-run",
			"status":       "RUNNING",
			"errorCode":    "",
			"errorMessage": "",
			"clientId":     "c1",
			"promptTips":   "",
		})
	}))
	defer srv.Close()

	c, err := CreateClient("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	modifier := ModifyNodes().Text("6", "hello").Seed("3", 123).Steps("3", 25)
	resp, err := c.RunWithModifier(context.Background(), "wf-123", modifier, &RunOptions{
		AddMetadata:      &addMetadata,
		UsePersonalQueue: &usePersonalQueue,
		InstanceType:     "default",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.TaskID != "task-run" {
		t.Fatalf("TaskID=%q", resp.TaskID)
	}
}

func TestRunAIAppWithModifier(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/openapi/v2/run/ai-app/app-123" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		nodeInfoList, ok := body["nodeInfoList"].([]any)
		if !ok || len(nodeInfoList) != 2 {
			t.Fatalf("nodeInfoList=%#v", body["nodeInfoList"])
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"taskId":       "task-ai-app",
			"status":       "RUNNING",
			"errorCode":    "",
			"errorMessage": "",
			"clientId":     "c2",
			"promptTips":   "",
		})
	}))
	defer srv.Close()

	c, err := NewClient("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	modifier := ModifyNodes().Set("41", "select", "7").Text("50", "润色这段话")
	resp, err := c.RunAIAppWithModifier(context.Background(), "app-123", modifier, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.TaskID != "task-ai-app" {
		t.Fatalf("TaskID=%q", resp.TaskID)
	}
}

func TestWaitForCompletionCallsStatusHook(t *testing.T) {
	statuses := []string{"RUNNING", "SUCCESS"}
	callCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/openapi/v2/query" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		status := statuses[callCount]
		callCount++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"taskId":       "task-wait",
			"status":       status,
			"errorCode":    "",
			"errorMessage": "",
			"results":      []map[string]any{{"url": "https://example.com/1", "outputType": "png"}},
			"clientId":     "c3",
			"promptTips":   "",
		})
	}))
	defer srv.Close()

	c, err := New("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	var seen []TaskStatus
	out, err := c.WaitForCompletion(context.Background(), "task-wait", &WaitForCompletionOptions{
		PollInterval: 10 * time.Millisecond,
		OnStatusChange: func(status TaskStatus) {
			seen = append(seen, status)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "SUCCESS" {
		t.Fatalf("Status=%q", out.Status)
	}
	if len(seen) != 2 || seen[0] != TaskStatusRunning || seen[1] != TaskStatusSuccess {
		t.Fatalf("seen=%v", seen)
	}
}

func TestCreateComfyTaskSimple(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/task/openapi/create" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["apiKey"] != "testkey" {
			t.Fatalf("apiKey=%v", body["apiKey"])
		}
		if body["workflowId"] != "wf" {
			t.Fatalf("workflowId=%v", body["workflowId"])
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"msg":  "success",
			"data": map[string]any{
				"netWssUrl":  "wss://example",
				"taskId":     123,
				"clientId":   "c1",
				"taskStatus": "RUNNING",
				"promptTips": "",
			},
		})
	}))
	defer srv.Close()

	c, err := New("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.CreateComfyTaskSimple(context.Background(), "wf", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.TaskID != 123 {
		t.Fatalf("TaskID=%d", resp.TaskID)
	}
}

func TestCreateComfyTaskAdvanced(t *testing.T) {
	retainSeconds := 60
	usePersonalQueue := false
	addMetadata := true

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/task/openapi/create" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["apiKey"] != "testkey" {
			t.Fatalf("apiKey=%v", body["apiKey"])
		}
		if body["workflowId"] != "wf-adv" {
			t.Fatalf("workflowId=%v", body["workflowId"])
		}
		if body["accessPassword"] != "secret" {
			t.Fatalf("accessPassword=%v", body["accessPassword"])
		}
		if body["retainSeconds"] != float64(retainSeconds) {
			t.Fatalf("retainSeconds=%v", body["retainSeconds"])
		}
		nodeInfoList, ok := body["nodeInfoList"].([]any)
		if !ok || len(nodeInfoList) != 1 {
			t.Fatalf("nodeInfoList=%#v", body["nodeInfoList"])
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"msg":  "success",
			"data": map[string]any{
				"taskId":     "1910246754753896450",
				"clientId":   "client-1",
				"taskStatus": "QUEUED",
				"promptTips": "{}",
			},
		})
	}))
	defer srv.Close()

	c, err := New("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	out, err := c.CreateComfyTaskAdvanced(context.Background(), CreateComfyTaskAdvancedRequest{
		WorkflowID:       "wf-adv",
		NodeInfoList:     []WorkflowNodeInfo{{NodeID: "6", FieldName: "text", FieldValue: "hello"}},
		AddMetadata:      &addMetadata,
		WebhookURL:       "https://example.com/hook",
		InstanceType:     "plus",
		UsePersonalQueue: &usePersonalQueue,
		RetainSeconds:    &retainSeconds,
		AccessPassword:   "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.TaskID != 1910246754753896450 {
		t.Fatalf("TaskID=%d", out.TaskID)
	}
}

func TestCreateAIAppTask(t *testing.T) {
	usePersonalQueue := false
	retainSeconds := 30

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/task/openapi/ai-app/run" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["apiKey"] != "testkey" {
			t.Fatalf("apiKey=%v", body["apiKey"])
		}
		if body["webappId"] != "app-legacy" {
			t.Fatalf("webappId=%v", body["webappId"])
		}
		if body["accessPassword"] != "pw" {
			t.Fatalf("accessPassword=%v", body["accessPassword"])
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"msg":  "success",
			"data": map[string]any{
				"taskId":     1907035719658053634,
				"clientId":   "legacy-client",
				"taskStatus": "RUNNING",
				"promptTips": "{}",
			},
		})
	}))
	defer srv.Close()

	c, err := New("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	out, err := c.CreateAIAppTask(context.Background(), CreateAIAppTaskRequest{
		WebappID:         "app-legacy",
		NodeInfoList:     []AIAppNodeInfo{{NodeID: "122", FieldName: "prompt", FieldValue: "一个在教室里的金发女孩"}},
		WebhookURL:       "https://example.com/webhook",
		InstanceType:     "default",
		AccessPassword:   "pw",
		UsePersonalQueue: &usePersonalQueue,
		RetainSeconds:    &retainSeconds,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.TaskID != 1907035719658053634 {
		t.Fatalf("TaskID=%d", out.TaskID)
	}
}

func TestGetAIAppAPICallDemo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/webapp/apiCallDemo" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("apiKey") != "testkey" {
			t.Fatalf("apiKey=%q", r.URL.Query().Get("apiKey"))
		}
		if r.URL.Query().Get("webappId") != "app-demo" {
			t.Fatalf("webappId=%q", r.URL.Query().Get("webappId"))
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"msg":  "success",
			"data": map[string]any{
				"curl":            "curl --request POST ...",
				"accessEncrypted": false,
				"webappName":      "Flux Kontext单图模式",
				"statisticsInfo": map[string]any{
					"likeCount":     "138",
					"downloadCount": "0",
					"useCount":      "34545",
					"pv":            "0",
					"collectCount":  "498",
				},
				"nodeInfoList": []map[string]any{{
					"nodeId":      "39",
					"fieldName":   "image",
					"fieldValue":  "a.png",
					"fieldType":   "IMAGE",
					"description": "上传图像",
				}},
			},
		})
	}))
	defer srv.Close()

	c, err := New("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	out, err := c.GetAIAppAPICallDemo(context.Background(), "app-demo")
	if err != nil {
		t.Fatal(err)
	}
	if out.WebappName != "Flux Kontext单图模式" {
		t.Fatalf("WebappName=%q", out.WebappName)
	}
	if len(out.NodeInfoList) != 1 || out.NodeInfoList[0].FieldName != "image" {
		t.Fatalf("NodeInfoList=%#v", out.NodeInfoList)
	}
}

func TestGetWebhookDetailAndRetryWebhook(t *testing.T) {
	var retried bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/task/openapi/getWebhookDetail":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["taskId"] != "task-1" {
				t.Fatalf("taskId=%v", body["taskId"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code": 0,
				"msg":  "success",
				"data": map[string]any{
					"id":               "wh-1",
					"userApiKey":       "********",
					"taskId":           "task-1",
					"webhookUrl":       "https://example.com/webhook",
					"event":            "TASK_END",
					"eventData":        "{}",
					"callbackStatus":   "FAILED",
					"callbackResponse": "timeout",
					"retryCount":       3,
					"createTime":       "2025-03-25T16:05:07",
					"updateTime":       "2025-03-25T16:08:10",
				},
			})
		case "/task/openapi/retryWebhook":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["webhookId"] != "wh-1" {
				t.Fatalf("webhookId=%v", body["webhookId"])
			}
			if body["webhookUrl"] != "https://example.com/retry" {
				t.Fatalf("webhookUrl=%v", body["webhookUrl"])
			}
			retried = true
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "", "data": nil})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c, err := New("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	detail, err := c.GetWebhookDetail(context.Background(), "task-1")
	if err != nil {
		t.Fatal(err)
	}
	if detail.ID != "wh-1" || detail.CallbackStatus != "FAILED" {
		t.Fatalf("detail=%#v", detail)
	}
	if err := c.RetryWebhook(context.Background(), detail.ID, "https://example.com/retry"); err != nil {
		t.Fatal(err)
	}
	if !retried {
		t.Fatal("expected retry call")
	}
}

func TestGetLoraUploadURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/openapi/getLoraUploadUrl" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["loraName"] != "my-lora-name" {
			t.Fatalf("loraName=%v", body["loraName"])
		}
		if body["md5Hex"] != "f8d958506e6c8044f79ccd7c814c6179" {
			t.Fatalf("md5Hex=%v", body["md5Hex"])
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"msg":  "success",
			"data": map[string]any{
				"fileName": "api-lora-cn/f8d958506e6c8044f79ccd7c814c6179.safetensors",
				"url":      "https://rh-models.example.com/upload",
			},
		})
	}))
	defer srv.Close()

	c, err := New("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	out, err := c.GetLoraUploadURL(context.Background(), "my-lora-name", "f8d958506e6c8044f79ccd7c814c6179")
	if err != nil {
		t.Fatal(err)
	}
	if out.FileName == "" || out.URL == "" {
		t.Fatalf("out=%#v", out)
	}
}

func TestQueryTaskV2(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/openapi/v2/query" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["taskId"] != "t1" {
			t.Fatalf("taskId=%v", body["taskId"])
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"taskId":       "t1",
			"status":       "SUCCESS",
			"errorCode":    "",
			"errorMessage": "",
			"results":      []map[string]any{{"url": "https://example.com/x", "outputType": "png"}},
			"clientId":     "c",
			"promptTips":   "",
		})
	}))
	defer srv.Close()

	c, err := New("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}
	q, err := c.QueryTaskV2(context.Background(), "t1")
	if err != nil {
		t.Fatal(err)
	}
	if q.Status != "SUCCESS" {
		t.Fatalf("Status=%q", q.Status)
	}
	if len(q.Results) != 1 {
		t.Fatalf("Results=%d", len(q.Results))
	}
}

func TestRunAIApp(t *testing.T) {
	usePersonalQueue := false

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/openapi/v2/run/ai-app/2016796569449795585" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer testkey" {
			t.Fatalf("Authorization=%q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		nodeInfoList, ok := body["nodeInfoList"].([]any)
		if !ok || len(nodeInfoList) != 2 {
			t.Fatalf("nodeInfoList=%#v", body["nodeInfoList"])
		}
		if body["instanceType"] != "default" {
			t.Fatalf("instanceType=%v", body["instanceType"])
		}
		if body["usePersonalQueue"] != false {
			t.Fatalf("usePersonalQueue=%#v", body["usePersonalQueue"])
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"taskId":       "2013508786110730241",
			"status":       "RUNNING",
			"errorCode":    "",
			"errorMessage": "",
			"results":      nil,
			"clientId":     "f828b9af25161bc066ef152db7b29ccc",
			"promptTips":   "{\"result\":true}",
		})
	}))
	defer srv.Close()

	c, err := New("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.RunAIApp(context.Background(), "2016796569449795585", RunAIAppRequest{
		NodeInfoList: []AIAppNodeInfo{
			{NodeID: "41", FieldName: "select", FieldValue: "7", Description: "设置比例"},
			{NodeID: "50", FieldName: "text", FieldValue: "润色这段话", Description: "输入文本"},
		},
		InstanceType:     "default",
		UsePersonalQueue: &usePersonalQueue,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.TaskID != "2013508786110730241" {
		t.Fatalf("TaskID=%q", resp.TaskID)
	}
	if resp.Status != "RUNNING" {
		t.Fatalf("Status=%q", resp.Status)
	}
	if resp.ClientID == "" {
		t.Fatal("expected client ID")
	}
	if resp.PromptTips == "" {
		t.Fatal("expected prompt tips")
	}
}

func TestRunAIApp_FullPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/openapi/v2/run/ai-app/2016796569449795585" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"taskId":       "t-aiapp",
			"status":       "RUNNING",
			"errorCode":    "",
			"errorMessage": "",
			"results":      nil,
			"clientId":     "c",
			"promptTips":   "",
		})
	}))
	defer srv.Close()

	c, err := New("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.RunAIApp(context.Background(), "/openapi/v2/run/ai-app/2016796569449795585", RunAIAppRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.TaskID != "t-aiapp" {
		t.Fatalf("TaskID=%q", resp.TaskID)
	}
}

func TestRunWorkflow(t *testing.T) {
	addMetadata := true
	usePersonalQueue := false

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/openapi/v2/run/workflow/2037060865681264641" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer testkey" {
			t.Fatalf("Authorization=%q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["addMetadata"] != true {
			t.Fatalf("addMetadata=%#v", body["addMetadata"])
		}
		if body["instanceType"] != "default" {
			t.Fatalf("instanceType=%v", body["instanceType"])
		}
		if body["usePersonalQueue"] != false {
			t.Fatalf("usePersonalQueue=%#v", body["usePersonalQueue"])
		}
		nodeInfoList, ok := body["nodeInfoList"].([]any)
		if !ok || len(nodeInfoList) != 0 {
			t.Fatalf("nodeInfoList=%#v", body["nodeInfoList"])
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"taskId":       "2013508786110730241",
			"status":       "RUNNING",
			"errorCode":    "",
			"errorMessage": "",
			"results":      nil,
			"clientId":     "f828b9af25161bc066ef152db7b29ccc",
			"promptTips":   "{\"result\":true}",
		})
	}))
	defer srv.Close()

	c, err := New("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.RunWorkflow(context.Background(), "2037060865681264641", RunWorkflowRequest{
		AddMetadata:      &addMetadata,
		NodeInfoList:     []WorkflowNodeInfo{},
		InstanceType:     "default",
		UsePersonalQueue: &usePersonalQueue,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.TaskID != "2013508786110730241" {
		t.Fatalf("TaskID=%q", resp.TaskID)
	}
	if resp.Status != "RUNNING" {
		t.Fatalf("Status=%q", resp.Status)
	}
}

func TestRunWorkflow_FullPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/openapi/v2/run/workflow/2037060865681264641" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"taskId":       "t-workflow",
			"status":       "RUNNING",
			"errorCode":    "",
			"errorMessage": "",
			"results":      nil,
			"clientId":     "c",
			"promptTips":   "",
		})
	}))
	defer srv.Close()

	c, err := New("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.RunWorkflow(context.Background(), "/openapi/v2/run/workflow/2037060865681264641", RunWorkflowRequest{NodeInfoList: []WorkflowNodeInfo{}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.TaskID != "t-workflow" {
		t.Fatalf("TaskID=%q", resp.TaskID)
	}
}

func TestWaitForTask(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/openapi/v2/query" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		calls++
		status := "RUNNING"
		results := any(nil)
		if calls >= 2 {
			status = "SUCCESS"
			results = []map[string]any{{"url": "https://example.com/x", "nodeId": "2", "outputType": "png"}}
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"taskId":       "t-wait",
			"status":       status,
			"errorCode":    "",
			"errorMessage": "",
			"results":      results,
			"clientId":     "c",
			"promptTips":   "",
		})
	}))
	defer srv.Close()

	c, err := New("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	out, err := c.WaitForTask(context.Background(), "t-wait", 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "SUCCESS" {
		t.Fatalf("Status=%q", out.Status)
	}
	if calls < 2 {
		t.Fatalf("calls=%d", calls)
	}
	if len(out.Results) != 1 || out.Results[0].NodeID != "2" {
		t.Fatalf("Results=%#v", out.Results)
	}
}

func TestWaitForTask_ContextTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/openapi/v2/query" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"taskId":       "t-timeout",
			"status":       "RUNNING",
			"errorCode":    "",
			"errorMessage": "",
			"results":      nil,
			"clientId":     "c",
			"promptTips":   "",
		})
	}))
	defer srv.Close()

	c, err := New("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	_, err = c.WaitForTask(ctx, "t-timeout", 50*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err=%v", err)
	}
}

func TestUploadBinaryFile_FromDisk(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "x.txt")
	if err := os.WriteFile(p, []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/openapi/v2/media/upload/binary" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    0,
			"message": "success",
			"data": map[string]any{
				"type":         "file",
				"download_url": "https://example.com/x",
				"fileName":     "x.txt",
				"size":         "3",
			},
		})
	}))
	defer srv.Close()

	c, err := New("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.UploadBinaryFile(context.Background(), p); err != nil {
		t.Fatal(err)
	}
}

func TestRunStandardModel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/openapi/v2/vidu/image-to-video-q3-pro-fast" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer testkey" {
			t.Fatalf("Authorization=%q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["prompt"] != "hi" {
			t.Fatalf("prompt=%v", body["prompt"])
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"taskId":       "t1",
			"status":       "RUNNING",
			"errorCode":    "",
			"errorMessage": "",
			"results":      nil,
			"clientId":     "c",
			"promptTips":   "",
			"usage": map[string]any{
				"consumeCoins": "1",
			},
		})
	}))
	defer srv.Close()

	c, err := New("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.RunStandardModel(context.Background(), "vidu/image-to-video-q3-pro-fast", map[string]any{"prompt": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.TaskID != "t1" {
		t.Fatalf("TaskID=%q", resp.TaskID)
	}
	if resp.Usage == nil || resp.Usage.ConsumeCoins == nil || *resp.Usage.ConsumeCoins != "1" {
		t.Fatalf("Usage=%#v", resp.Usage)
	}
}

func TestRunStandardModel_BusinessError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/openapi/v2/vidu/image-to-video-q3-pro-fast" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"taskId":       "",
			"status":       "",
			"errorCode":    "812",
			"errorMessage": "CORPAPIKEY_INSUFFICIENT_FUNDS",
			"results":      nil,
			"clientId":     "",
			"promptTips":   "",
		})
	}))
	defer srv.Close()

	c, err := New("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.RunStandardModel(context.Background(), "vidu/image-to-video-q3-pro-fast", map[string]any{"prompt": "hi"})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T %v", err, err)
	}
	if apiErr.Code != 812 {
		t.Fatalf("Code=%d", apiErr.Code)
	}
	if apiErr.Message != "CORPAPIKEY_INSUFFICIENT_FUNDS" {
		t.Fatalf("Message=%q", apiErr.Message)
	}
}

func TestRunStandardModel_FullURLPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/openapi/v2/seedance-v1.5-pro/text-to-video" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"taskId":       "t2",
			"status":       "RUNNING",
			"errorCode":    "",
			"errorMessage": "",
			"results":      nil,
			"clientId":     "c",
			"promptTips":   "",
		})
	}))
	defer srv.Close()

	c, err := New("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	out, err := c.RunStandardModel(context.Background(), "https://www.runninghub.cn/openapi/v2/seedance-v1.5-pro/text-to-video", map[string]any{"prompt": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if out.TaskID != "t2" {
		t.Fatalf("TaskID=%q", out.TaskID)
	}
}

func TestPricePreview(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/openapi/v2/price-preview/vidu/image-to-video-q3-pro-fast" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["prompt"] != "hi" {
			t.Fatalf("prompt=%v", body["prompt"])
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"estimatedPrice":          4.8,
			"currency":                "CNY",
			"freeLimit":               false,
			"isFreeThisCall":          false,
			"priceText":               "¥0.50/次",
			"priceTextEn":             "CNY 0.50/call",
			"remainingFreeLimitCount": "0",
			"freeLimitCount":          "0",
		})
	}))
	defer srv.Close()

	c, err := New("testkey", WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	out, err := c.PricePreview(context.Background(), "/openapi/v2/vidu/image-to-video-q3-pro-fast", map[string]any{"prompt": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Currency != "CNY" {
		t.Fatalf("Currency=%q", out.Currency)
	}
}

func TestDownloadFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/x.jpg" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte("image-bytes"))
	}))
	defer srv.Close()

	c, err := New("testkey", WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(t.TempDir(), "out", "x.jpg")
	if err := c.DownloadFile(context.Background(), srv.URL+"/x.jpg", dest); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "image-bytes" {
		t.Fatalf("content=%q", string(b))
	}
}

func TestDownloadTaskResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/a.jpg":
			_, _ = w.Write([]byte("a"))
		case "/b.mp4":
			_, _ = w.Write([]byte("b"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c, err := New("testkey", WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}

	task := &QueryV2Response{
		TaskID: "task123",
		Results: []QueryV2ResultItem{
			{URL: srv.URL + "/a.jpg", OutputType: "jpg"},
			{URL: srv.URL + "/b.mp4", OutputType: "mp4"},
		},
	}
	dir := t.TempDir()
	out, err := c.DownloadTaskResults(context.Background(), task, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("downloads=%d", len(out))
	}
	for _, item := range out {
		if _, err := os.Stat(item.Path); err != nil {
			t.Fatalf("missing file %q: %v", item.Path, err)
		}
	}
}
