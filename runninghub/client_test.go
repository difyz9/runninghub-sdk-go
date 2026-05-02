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
